package main

import (
	"bytes"
	"compress/gzip"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

// generateFixture runs only the fixed-input fixture generator. It accepts no
// image path or input data. It cannot be used as an unrestricted decoder helper.
func generateFixture(root string) error {
	return generateFixedFixture(root, "./fixturegen", "tenbit.heic", 128*1024*1024, 64*1024, 15*time.Second, wazero.NewRuntimeConfigInterpreter())
}

func generatePhotoFixture(root string) error {
	return generateFixedFixture(root, "./fixturephoto", "photo-gradient.heic.gz", 1024*1024*1024, 64*1024*1024, 30*time.Second, wazero.NewRuntimeConfigCompiler())
}

func generateFixedFixture(root, source, target string, memory, outputBytes int64, timeout time.Duration, config wazero.RuntimeConfig) error {
	dir, err := os.MkdirTemp("", "picfetch-heic-fixture-")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	artifact := filepath.Join(dir, "fixture.wasm")
	if _, err = goCommand(root, true, "build", "-mod=readonly", "-tags=noasm", "-trimpath", "-buildvcs=false", "-ldflags=-s -w -buildid=", "-o", artifact, source); err != nil {
		return err
	}
	wasm, err := os.ReadFile(artifact)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	runtime := wazero.NewRuntimeWithConfig(ctx, config.WithMemoryLimitPages(uint32(memory/(64*1024))).WithCloseOnContextDone(true))
	defer func() { _ = runtime.Close(context.Background()) }()
	if _, err = wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		return err
	}
	output := &fixtureOutput{limit: outputBytes}
	if _, err = runtime.InstantiateWithConfig(ctx, wasm, wazero.NewModuleConfig().WithStdout(output)); err != nil {
		return err
	}
	data := output.Bytes()
	if strings.HasSuffix(target, ".gz") {
		var compressed bytes.Buffer
		writer := gzip.NewWriter(&compressed)
		if _, err = writer.Write(data); err != nil {
			_ = writer.Close()
			return err
		}
		if err = writer.Close(); err != nil {
			return err
		}
		data = compressed.Bytes()
	}
	return os.WriteFile(filepath.Join(root, "scripts/heicbuild/testdata", target), data, 0644)
}

type fixtureOutput struct {
	bytes.Buffer
	limit int64
}

func (w *fixtureOutput) Write(p []byte) (int, error) {
	if int64(len(p)) > w.limit-int64(w.Len()) {
		return 0, errors.New("fixed fixture exceeded output limit")
	}
	return w.Buffer.Write(p)
}
