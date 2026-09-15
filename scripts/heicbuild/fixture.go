package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// generateFixture runs only the fixed-input fixture generator. It accepts no
// image path or input data. It cannot be used as an unrestricted decoder helper.
func generateFixture(root string) error {
	dir, err := os.MkdirTemp("", "picfetch-heic-fixture-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	artifact := filepath.Join(dir, "fixture.wasm")
	if _, err = goCommand(root, true, "build", "-mod=readonly", "-tags=noasm", "-trimpath", "-buildvcs=false", "-ldflags=-s -w -buildid=", "-o", artifact, "./fixturegen"); err != nil {
		return err
	}
	wasm, err := os.ReadFile(artifact)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	runtime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter().WithMemoryLimitPages(2048).WithCloseOnContextDone(true))
	defer func() { _ = runtime.Close(context.Background()) }()
	if _, err = wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		return err
	}
	output := &fixtureOutput{}
	if _, err = runtime.InstantiateWithConfig(ctx, wasm, wazero.NewModuleConfig().WithStdout(output)); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, "scripts/heicbuild/testdata/tenbit.heic"), output.Bytes(), 0644)
}

type fixtureOutput struct{ bytes.Buffer }

func (w *fixtureOutput) Write(p []byte) (int, error) {
	if len(p) > 64*1024-w.Len() {
		return 0, errors.New("fixed fixture exceeded 64 KiB output")
	}
	return w.Buffer.Write(p)
}
