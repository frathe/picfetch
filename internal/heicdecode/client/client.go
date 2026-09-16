// Package client owns bounded HEIC helper admission, transport and process
// lifetime. The app-family service must share one Client across its consumers.
package client

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"

	"github.com/frathe/picfetch/internal/heicdecode"
)

var (
	ErrUnavailable     = errors.New("isolated HEIC helper unavailable")
	ErrBusy            = errors.New("HEIC request queue is full")
	ErrDiagnosticLimit = errors.New("HEIC helper exceeded diagnostic limit")
)

// Config is immutable after New. SHA256 pins the packaged, already signed
// helper executable. An absent helper or pin never selects an ambient program.
type Config struct {
	Executable string
	SHA256     [32]byte
	Limits     heicdecode.Limits
}

// Input reads a source only after admission and native sandbox readiness. It
// must honor ctx and maxBytes, and may not return a larger encoded allocation.
type Input func(ctx context.Context, maxBytes int64) ([]byte, error)

// Client admits one job and at most 64 pending callers. Stop is nonblocking;
// Wait joins active process/pipe work and cancelled waiting callers off the UI.
type Client struct {
	config      Config
	ctx         context.Context
	cancel      context.CancelFunc
	lane        admissionQueue
	mu          sync.Mutex
	closed      bool
	active      int
	services    int
	work        sync.WaitGroup
	unavailable bool
	release     func()
	releaseOnce sync.Once
}

func New(config Config) (*Client, error) {
	if err := config.Limits.Validate(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(config.Executable) || config.SHA256 == [32]byte{} {
		return nil, ErrUnavailable
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{config: config, ctx: ctx, cancel: cancel}, nil
}

func (c *Client) Stop() {
	c.mu.Lock()
	c.closed = true
	c.cancel()
	c.mu.Unlock()
}

func (c *Client) Wait() {
	c.work.Wait()
	c.mu.Lock()
	closed := c.closed
	c.mu.Unlock()
	if closed && c.release != nil {
		c.releaseOnce.Do(c.release)
	}
}

// Unavailable reports the latest helper identity/readiness failure, separately
// from immutable reader capability and the user's saved preference.
func (c *Client) Unavailable() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.unavailable
}

func (c *Client) recordAvailability(err error) {
	if err == nil || errors.Is(err, ErrUnavailable) {
		c.mu.Lock()
		c.unavailable = err != nil
		c.mu.Unlock()
	}
}

// Do retains admission until source work, the helper, all pipes, and output
// validation have finished. A cancelled queued call never invokes input.
func (c *Client) Do(ctx context.Context, op heicdecode.Operation, input Input) (heicdecode.Response, error) {
	return c.DoWithPriority(ctx, Foreground, op, input)
}

// DoWithPriority preserves FIFO within each class and gives waiting background
// work a turn after at most three foreground grants. The timeout begins after
// admission; queued callers retain their own cancellation/deadline.
func (c *Client) DoWithPriority(ctx context.Context, priority Priority, op heicdecode.Operation, input Input) (heicdecode.Response, error) {
	var result heicdecode.Response
	err := c.withAdmission(ctx, priority, func(ctx context.Context) error {
		if err := c.verifyExecutable(ctx); err != nil {
			return err
		}
		var err error
		result, err = c.run(ctx, op, input)
		return err
	})
	c.recordAvailability(err)
	if err != nil {
		return heicdecode.Response{}, err
	}
	return result, nil
}

// The pipe service also holds this grant through downstream response delivery,
// so a stalled analysis consumer cannot accumulate decoded outputs off-lane.
func (c *Client) withAdmission(ctx context.Context, priority Priority, operation func(context.Context) error) error {
	if priority > Background {
		return heicdecode.ErrInvalidRequest
	}
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return context.Canceled
	}
	if c.active >= 65 {
		c.mu.Unlock()
		return ErrBusy
	}
	c.active++
	c.work.Add(1)
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		c.active--
		c.mu.Unlock()
		c.work.Done()
	}()
	ctx, cancel := context.WithCancel(ctx)
	stop := context.AfterFunc(c.ctx, cancel)
	defer func() { stop(); cancel() }()
	admitted := c.lane.enter(ctx, priority)
	defer admitted.release()
	if err := admitted.wait(); err != nil {
		return err
	}
	ctx, deadline := context.WithTimeout(ctx, c.config.Limits.Timeout)
	defer deadline()
	if err := ctx.Err(); err != nil {
		return err
	}
	err := operation(ctx)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	return err
}

func (c *Client) verifyExecutable(ctx context.Context) error {
	file, err := os.Open(c.config.Executable)
	if err != nil {
		return fmt.Errorf("%w: missing executable", ErrUnavailable)
	}
	defer func() { _ = file.Close() }()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 64*1024*1024 {
		return fmt.Errorf("%w: executable size or type", ErrUnavailable)
	}
	hash := sha256.New()
	if _, err = io.Copy(hash, io.LimitReader(file, 64*1024*1024+1)); err != nil {
		return fmt.Errorf("%w: executable read", ErrUnavailable)
	}
	if string(hash.Sum(nil)) != string(c.config.SHA256[:]) {
		return fmt.Errorf("%w: executable identity", ErrUnavailable)
	}
	return ctx.Err()
}
