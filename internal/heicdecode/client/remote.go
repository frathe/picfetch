package client

import (
	"context"
	"errors"
	"io"
	"sync"

	"github.com/frathe/picfetch/internal/heicdecode"
)

// Remote uses an explicitly inherited pair of interruptible pipes to reach the
// application's admission owner. It cannot launch a helper itself. Cancellation
// retires this connection; its owning analysis process must obtain a new pair
// before later work. Each Remote permits one active call and no hidden backlog.
type Remote struct {
	input          io.ReadCloser
	output         io.WriteCloser
	limits         heicdecode.Limits
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.Mutex
	closed, active bool
	work           sync.WaitGroup
}

func NewRemote(input io.ReadCloser, output io.WriteCloser, limits heicdecode.Limits) (*Remote, error) {
	if err := limits.Validate(); err != nil {
		return nil, err
	}
	if input == nil || output == nil {
		return nil, ErrUnavailable
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Remote{input: input, output: output, limits: limits, ctx: ctx, cancel: cancel}, nil
}

func (r *Remote) Stop() {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return
	}
	r.closed = true
	r.cancel()
	r.mu.Unlock()
	_ = r.input.Close()
	_ = r.output.Close()
}

func (r *Remote) Wait() { r.work.Wait() }

func (r *Remote) Do(ctx context.Context, op heicdecode.Operation, input Input) (result heicdecode.Response, resultErr error) {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return result, ErrUnavailable
	}
	if r.active {
		r.mu.Unlock()
		return result, ErrBusy
	}
	r.active = true
	r.work.Add(1)
	r.mu.Unlock()
	defer func() { r.mu.Lock(); r.active = false; r.mu.Unlock(); r.work.Done() }()
	ctx, cancel := context.WithCancel(ctx)
	stopParent := watchCancellation(r.ctx, cancel)
	stopIO := watchCancellation(ctx, r.Stop)
	defer func() {
		stopIO()
		stopParent()
		ctxErr := ctx.Err()
		cancel()
		if ctxErr != nil {
			result = heicdecode.Response{}
			resultErr = ctxErr
		}
		var failure *heicdecode.Failure
		if resultErr != nil && !errors.As(resultErr, &failure) {
			r.Stop()
		}
	}()
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if err := writeBrokerHeader(r.output, brokerHeader{kind: brokerIntent, operation: op}); err != nil {
		return result, err
	}
	header, err := readBrokerHeader(r.input)
	if err != nil {
		return result, err
	}
	if header.operation != op {
		return result, heicdecode.ErrInvalidResponse
	}
	if header.kind == brokerResult {
		// A helper refused before source admission. A successful pixel result
		// here would violate the handshake and must not be published.
		_, err = readBrokerResult(r.input, header.size, op, r.limits)
		var failure *heicdecode.Failure
		if errors.As(err, &failure) {
			return result, err
		}
		return result, heicdecode.ErrInvalidResponse
	}
	if header.kind != brokerGrant || header.size == 0 || header.size > uint64(r.limits.MaxInputBytes) {
		return result, heicdecode.ErrInvalidResponse
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	data, err := input(ctx, int64(header.size))
	if err != nil {
		return result, err
	}
	if err = ctx.Err(); err != nil {
		return result, err
	}
	if len(data) == 0 || uint64(len(data)) > header.size {
		return result, heicdecode.ErrInvalidRequest
	}
	if err = writeBrokerHeader(r.output, brokerHeader{kind: brokerInput, operation: op, size: uint64(len(data)) + 24}); err != nil {
		return result, err
	}
	if err = heicdecode.WriteRequest(r.output, heicdecode.Request{Operation: op, Input: data}, r.limits); err != nil {
		return result, err
	}
	header, err = readBrokerHeader(r.input)
	if err != nil {
		return result, err
	}
	if header.kind != brokerResult || header.operation != op {
		return result, heicdecode.ErrInvalidResponse
	}
	return readBrokerResult(r.input, header.size, op, r.limits)
}

// watchCancellation returns a join, including when a callback has already
// started. Close functions must interrupt pending pipe operations.
func watchCancellation(ctx context.Context, action func()) func() {
	done := make(chan struct{})
	stop := context.AfterFunc(ctx, func() { defer close(done); action() })
	return func() {
		if !stop() {
			<-done
		}
	}
}
