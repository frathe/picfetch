package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
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
	tcp, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		return empty, err
	}
	defer func() { _ = tcp.Close() }()
	udp, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		return empty, err
	}
	defer func() { _ = udp.Close() }()
	limitsJSON, err := json.Marshal(c.config.Limits)
	if err != nil {
		return empty, err
	}
	cmd := exec.Command(c.config.Executable, "--heic-worker-v2", string(limitsJSON), readPath, writePath, tcp.Addr().String(), udp.LocalAddr().String())
	cmd.Env = []string{"GOMAXPROCS=1"}
	cmd.Dir = string(filepath.Separator)
	configureProcess(cmd)
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
	if err = cmd.Start(); err != nil {
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
		killProcess(cmd)
		_ = inWrite.Close()
		_ = outRead.Close()
		_ = errRead.Close()
	})
	var pipeWork sync.WaitGroup
	var diagnosticErr error // Read only after pipeWork.Wait.
	pipeWork.Add(1)
	stderrDone := make(chan error, 1)
	go func() {
		defer pipeWork.Done()
		count, readErr := io.Copy(io.Discard, io.LimitReader(errRead, int64(c.config.Limits.MaxDiagnosticBytes)+1))
		if count > int64(c.config.Limits.MaxDiagnosticBytes) {
			readErr = ErrDiagnosticLimit
			diagnosticErr = readErr
			cancel()
		}
		stderrDone <- readErr
	}()
	// Every return after Start waits for process exit and all transport workers.
	defer func() {
		cancel()
		cancelWork.Wait()
		// Kill the group before reaping its leader, so its PID cannot be
		// recycled between Wait and a late cancellation callback.
		processErr := cmd.Wait()
		pipeWork.Wait()
		if diagnosticErr != nil {
			result = empty
			resultErr = errors.Join(resultErr, diagnosticErr)
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
		return empty, fmt.Errorf("%w: native readiness", ErrUnavailable)
	}
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
	if err = <-stderrDone; err != nil {
		return empty, err
	}
	if err = ctx.Err(); err != nil {
		return empty, err
	}
	return decoded, nil
}
