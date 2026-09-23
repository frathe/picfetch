package heic

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const workerEnvironment = "PICFETCH_HEIC_WORKER"

// Client owns bounded native children. Stop closes admission and cancels all
// admitted work without waiting; Wait joins it off the UI thread.
type Client struct {
	executable string
	command    func(context.Context, string) *exec.Cmd
	started    func()
	deadline   time.Duration
	slots      chan struct{}
	mu         sync.Mutex
	stopped    bool
	active     map[uint64]context.CancelFunc
	next       uint64
	workers    sync.WaitGroup
}

func NewClient(executable string) *Client {
	return &Client{executable: executable, command: workerCommand,
		slots: make(chan struct{}, 2), active: make(map[uint64]context.CancelFunc)}
}

// NewInheritedSandboxClient is for an analysis producer that verifies its own
// OS network denial before reading images. Its children inherit that sandbox;
// installing a second macOS sandbox can fail even with the same policy.
// Other platforms retain their ordinary child launch and resource limits.
func NewInheritedSandboxClient(executable string) *Client {
	client := NewClient(executable)
	client.command = inheritedWorkerCommand
	return client
}

func (c *Client) Check(ctx context.Context) error {
	_, err := c.run(ctx, nil, wireRequest{Check: true}, 10*time.Second)
	return err
}

func (c *Client) Read(ctx context.Context, data []byte, request Request) (Result, error) {
	request.MaxEncodedBytes = min(request.MaxEncodedBytes, encodedLimit)
	request.MaxPixels = min(request.MaxPixels, pixelLimit)
	if err := validateRequest(request, int64(len(data))); err != nil {
		return Result{}, err
	}
	return c.run(ctx, data, wireRequest{Request: request, Encoded: len(data)}, 30*time.Second)
}

func (c *Client) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopped = true
	for _, cancel := range c.active {
		cancel()
	}
}

func (c *Client) Wait() { c.workers.Wait() }

func (c *Client) run(ctx context.Context, data []byte, request wireRequest, limit time.Duration) (Result, error) {
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if c.deadline > 0 {
		limit = c.deadline
	}
	ctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	c.mu.Lock()
	if c.stopped {
		c.mu.Unlock()
		return Result{}, context.Canceled
	}
	c.next++
	id := c.next
	c.active[id] = cancel
	c.workers.Add(1)
	c.mu.Unlock()
	defer func() {
		c.mu.Lock()
		delete(c.active, id)
		c.mu.Unlock()
		c.workers.Done()
	}()
	select {
	case c.slots <- struct{}{}:
		defer func() { <-c.slots }()
	case <-ctx.Done():
		return Result{}, ctx.Err()
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	executable := c.executable
	if executable == "" {
		var err error
		executable, err = os.Executable()
		if err != nil {
			return Result{}, err
		}
	}
	cmd := c.command(ctx, executable)
	prepareWorker(cmd)
	cmd.Env = workerEnv(cmd.Env)
	cmd.WaitDelay = time.Second
	var header bytes.Buffer
	if err := writeHeader(&header, request); err != nil {
		return Result{}, err
	}
	cmd.Stdin = io.MultiReader(&header, bytes.NewReader(data))
	stderr := &limitedLog{remaining: 4096}
	cmd.Stderr = stderr
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Result{}, err
	}
	defer func() { _ = stdout.Close() }()
	cmd.Cancel = func() error {
		err := retireWorker(cmd)
		_ = stdout.Close()
		return err
	}
	if err := cmd.Start(); err != nil {
		return Result{}, fmt.Errorf("start HEIC worker: %w", err)
	}
	if c.started != nil {
		c.started()
	}
	result, decodeErr := readResponse(stdout, request)
	if decodeErr != nil {
		_ = retireWorker(cmd)
	}
	waitErr := cmd.Wait()
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if waitErr != nil && (decodeErr == nil || errors.Is(decodeErr, io.EOF) || errors.Is(decodeErr, io.ErrUnexpectedEOF)) {
		return Result{}, fmt.Errorf("HEIC worker exited: %w: %s", waitErr, stderr.String())
	}
	if decodeErr != nil {
		return Result{}, decodeErr
	}
	if waitErr != nil {
		return Result{}, fmt.Errorf("HEIC worker exited: %w", waitErr)
	}
	return result, nil
}

func workerEnv(base []string) []string {
	if base == nil {
		base = os.Environ()
	}
	result := make([]string, 0, len(base)+1)
	for _, entry := range base {
		key, _, _ := strings.Cut(entry, "=")
		if key == workerEnvironment || key == "LIBHEIF_PLUGIN_PATH" || key == "LIBHEIF_PLUGIN_PATHS" ||
			key == "LIBHEIF_SECURITY_LIMITS" || key == "LD_LIBRARY_PATH" || key == "LD_PRELOAD" {
			continue
		}
		result = append(result, entry)
	}
	return append(result, workerEnvironment+"=1")
}

type limitedLog struct {
	bytes.Buffer
	remaining int
}

func (b *limitedLog) Write(p []byte) (int, error) {
	n := len(p)
	if len(p) > b.remaining {
		p = p[:b.remaining]
	}
	b.remaining -= len(p)
	_, _ = b.Buffer.Write(p)
	return n, nil
}
