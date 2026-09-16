package client

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/frathe/picfetch/internal/heicdecode"
)

func (c *Client) run(ctx context.Context, op heicdecode.Operation, input Input) (result heicdecode.Response, resultErr error) {
	var empty heicdecode.Response
	probe, err := os.MkdirTemp("", "picfetch-heic-probe-")
	if err != nil {
		return empty, err
	}
	defer func() { _ = os.RemoveAll(probe) }()
	readPath, writePath := filepath.Join(probe, "read"), filepath.Join(probe, "write")
	if err = os.WriteFile(readPath, []byte("PicFetch owned sandbox probe\n"), 0600); err != nil {
		return empty, err
	}
	// Both endpoints belong to this request. A failed sandbox can reach only
	// these disposable loopback sockets, never an unrelated network service.
	probes, err := openNativeProbes(ctx)
	if err != nil {
		return empty, err
	}
	defer probes.close()
	limitsJSON, err := json.Marshal(c.config.Limits)
	if err != nil {
		return empty, err
	}
	cmd := exec.Command(c.config.Executable, "--heic-worker-v2", string(limitsJSON), readPath, writePath, probes.tcp.Addr().String(), probes.udp.LocalAddr().String())
	cmd.Env = []string{"GOMAXPROCS=1"}
	cmd.Dir = string(filepath.Separator)
	var pipes []*os.File
	defer func() {
		for _, file := range pipes {
			_ = file.Close()
		}
	}()
	newPipe := func() (*os.File, *os.File, error) {
		reader, writer, pipeErr := os.Pipe()
		if pipeErr == nil {
			pipes = append(pipes, reader, writer)
		}
		return reader, writer, pipeErr
	}
	inRead, inWrite, err := newPipe()
	if err != nil {
		return empty, err
	}
	outRead, outWrite, err := newPipe()
	if err != nil {
		return empty, err
	}
	errRead, errWrite, err := newPipe()
	if err != nil {
		return empty, err
	}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = inRead, outWrite, errWrite
	process, err := startProcess(cmd, c.config.Limits)
	if err != nil {
		return empty, fmt.Errorf("%w: launch: %v", ErrUnavailable, err)
	}
	_ = inRead.Close()
	_ = outWrite.Close()
	_ = errWrite.Close()
	// Own every pipe rather than relying on exec.Cmd's background copy loops.
	// On cancellation, close both transport directions and terminate the family.
	ctx, cancel := context.WithCancel(ctx)
	var cancelWork sync.WaitGroup
	cancelWork.Add(1)
	context.AfterFunc(ctx, func() {
		defer cancelWork.Done()
		process.Kill()
		_ = inWrite.Close()
		_ = outRead.Close()
		_ = errRead.Close()
	})
	var pipeWork sync.WaitGroup
	var readinessFailed bool
	var diagnostic bytes.Buffer // Read only after pipeWork.Wait.
	var diagnosticErr error     // Read only after pipeWork.Wait.
	var stderrErr error         // Read only after stderrDone closes.
	pipeWork.Add(1)
	stderrDone := make(chan struct{})
	go func() {
		defer pipeWork.Done()
		defer close(stderrDone)
		count, readErr := io.Copy(&diagnostic, io.LimitReader(errRead, int64(c.config.Limits.MaxDiagnosticBytes)+1))
		if count > int64(c.config.Limits.MaxDiagnosticBytes) {
			readErr = ErrDiagnosticLimit
			diagnosticErr = readErr
			cancel()
		}
		stderrErr = readErr
	}()
	// Every return after Start waits for process exit and all transport workers.
	defer func() {
		// Stop the producer, then drain its bounded diagnostics before closing
		// our read end. Otherwise stdout EOF can race the overflow reader and
		// turn a diagnostic-limit failure into an ordinary protocol error.
		process.Kill()
		<-stderrDone
		cancel()
		cancelWork.Wait()
		// Kill the group before reaping its leader, so its PID cannot be
		// recycled between Wait and a late cancellation callback.
		processErr := process.Wait()
		pipeWork.Wait()
		if diagnosticErr != nil {
			result = empty
			resultErr = errors.Join(resultErr, diagnosticErr)
		} else if readinessFailed && diagnostic.Len() > 0 {
			// Readiness precedes all image input. Quote its bounded startup
			// diagnostic, but never publish parser diagnostics after readiness.
			resultErr = fmt.Errorf("%w: startup diagnostic %q", resultErr, diagnostic.String())
		}
		if resultErr == nil && processErr != nil {
			result = empty
			resultErr = fmt.Errorf("HEIC helper exit: %w", processErr)
		}
	}()
	ready, err := heicdecode.ReadReady(outRead)
	if err != nil || ready.WASMMemoryBytes != c.config.Limits.WASMMemoryBytes ||
		ready.NativeMemoryBytes > c.config.Limits.OSProcessBytes ||
		(runtime.GOOS != "darwin" && ready.NativeMemoryBytes == 0) {
		readinessFailed = true
		process.Kill()
		<-stderrDone
		return empty, fmt.Errorf("%w: native readiness", ErrUnavailable)
	}
	// No loopback listener remains reachable when image bytes are admitted.
	probes.close()
	if err = ctx.Err(); err != nil {
		return empty, err
	}
	data, err := input(ctx, c.config.Limits.MaxInputBytes)
	if err != nil {
		return empty, err
	}
	if err = ctx.Err(); err != nil {
		return empty, err
	}
	writeDone := make(chan error, 1)
	pipeWork.Add(1)
	go func() {
		defer pipeWork.Done()
		writeErr := heicdecode.WriteRequest(inWrite, heicdecode.Request{Operation: op, Input: data}, c.config.Limits)
		_ = inWrite.Close()
		writeDone <- writeErr
	}()
	decoded, readErr := heicdecode.ReadResponse(outRead, op, c.config.Limits)
	if readErr != nil {
		return empty, readErr
	}
	if err = <-writeDone; err != nil {
		return empty, err
	}
	<-stderrDone
	if stderrErr != nil {
		return empty, stderrErr
	}
	if err = ctx.Err(); err != nil {
		return empty, err
	}
	return decoded, nil
}

// Each platform owns family termination and a final join. Windows creates a
// suspended AppContainer process rather than asking exec.Cmd to start it.
type helperProcess interface {
	Kill()
	Wait() error
}
