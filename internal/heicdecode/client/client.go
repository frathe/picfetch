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
	config Config
	ctx    context.Context
	cancel context.CancelFunc
	slot   chan struct{}
	mu     sync.Mutex
	closed bool
	active int
	work   sync.WaitGroup
}

func New(config Config) (*Client, error) {
	if err := config.Limits.Validate(); err != nil {
		return nil, err
	}
	if !filepath.IsAbs(config.Executable) || config.SHA256 == [32]byte{} {
		return nil, ErrUnavailable
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &Client{config: config, ctx: ctx, cancel: cancel, slot: make(chan struct{}, 1)}, nil
}

func (c *Client) Stop() {
	c.mu.Lock()
	c.closed = true
	c.cancel()
	c.mu.Unlock()
}

func (c *Client) Wait() { c.work.Wait() }

// Do retains admission until source work, the helper, all pipes, and output
// validation have finished. A cancelled queued call never invokes input.
func (c *Client) Do(ctx context.Context, op heicdecode.Operation, input Input) (heicdecode.Response, error) {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return heicdecode.Response{}, context.Canceled
	}
	if c.active >= 65 {
		c.mu.Unlock()
		return heicdecode.Response{}, ErrBusy
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
	select {
	case c.slot <- struct{}{}:
		defer func() { <-c.slot }()
	case <-ctx.Done():
		return heicdecode.Response{}, ctx.Err()
	}
	ctx, deadline := context.WithTimeout(ctx, c.config.Limits.Timeout)
	defer deadline()
	if err := ctx.Err(); err != nil {
		return heicdecode.Response{}, err
	}
	if err := c.verifyExecutable(ctx); err != nil {
		return heicdecode.Response{}, err
	}
	result, err := c.run(ctx, op, input)
	if ctx.Err() != nil {
		return heicdecode.Response{}, ctx.Err()
	}
	return result, err
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
