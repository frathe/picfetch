// Package worker hosts the fixed WASI HEIC module in the dedicated helper.
// Only cmd/picfetch-heic-worker may import it in a production binary.
package worker

import (
	"context"
	_ "embed"
	"errors"
	"io"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/frathe/picfetch/internal/heicdecode"
)

// decoder is reproduced and verified by scripts/heicbuild. It is never loaded
// from a request, a user preference, or an environment variable.
//
//go:embed decoder.wasm
var decoder []byte

// execute requires the caller to establish native isolation first. Production
// reaches it only through Main; package tests also use small owned WASM modules
// to observe the runtime's memory and cancellation boundaries independently.
func execute(ctx context.Context, module []byte, stdin io.Reader, stdout, stderr io.Writer, limits heicdecode.Limits) error {
	if err := limits.Validate(); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, limits.Timeout)
	defer cancel()
	config := runtimeConfig().
		WithMemoryLimitPages(uint32(limits.WASMMemoryBytes / (64 * 1024))).
		WithCloseOnContextDone(true)
	runtime := wazero.NewRuntimeWithConfig(ctx, config)
	defer func() { _ = runtime.Close(context.Background()) }()
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		return err
	}
	// All stream limits include their fixed protocol header. No image-sized
	// host buffer is used: WASI copies directly between guest memory and pipes.
	output := &limitedWriter{writer: stdout, remaining: limits.MaxOutputBytes + int64(limits.MaxMetadataBytes) + int64(limits.MaxDiagnosticBytes) + 40}
	diagnostic := &limitedWriter{writer: stderr, remaining: int64(limits.MaxDiagnosticBytes)}
	moduleConfig := wazero.NewModuleConfig().
		WithStdin(io.LimitReader(stdin, limits.MaxInputBytes+24+1)).
		WithStdout(output).WithStderr(diagnostic)
	// No filesystem, environment, clock, random source, or socket capability.
	_, err := runtime.InstantiateWithConfig(ctx, module, moduleConfig)
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if output.exceeded || diagnostic.exceeded {
		return errors.New("HEIC guest exceeded stream limit")
	}
	return err
}

type limitedWriter struct {
	writer    io.Writer
	remaining int64
	exceeded  bool
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	if int64(len(p)) > w.remaining {
		w.exceeded = true
		return 0, errors.New("HEIC stream limit")
	}
	n, err := w.writer.Write(p)
	w.remaining -= int64(n)
	if n < len(p) && err == nil {
		err = io.ErrShortWrite
	}
	return n, err
}
