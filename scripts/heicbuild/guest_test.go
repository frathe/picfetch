package main

import (
	"bytes"
	"context"
	"errors"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/frathe/picfetch/internal/heicdecode"
)

// Only unchanged licensed small ordinary fixtures enter the development guest.
// This proves the guest ABI, not native helper resource or sandbox enforcement.
func TestWASIGuestOrdinaryFixtures(t *testing.T) {
	wasm, err := os.ReadFile(filepath.Join("..", "heicguest", "decoder.wasm"))
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name           string
		sixteen, alpha bool
	}{
		{"basic.heic", false, false}, {"main10.heic", false, false}, {"tenbit.heic", true, false}, {"alpha.heic", false, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input, err := os.ReadFile(filepath.Join("testdata", tt.name))
			if err != nil {
				t.Fatal(err)
			}
			result := runGuest(t, wasm, input, heicdecode.Decode)
			if result.Image == nil || result.Image.Bounds().Empty() {
				t.Fatal("missing decoded image")
			}
			_, sixteen := result.Image.(*image.NRGBA64)
			if sixteen != tt.sixteen {
				t.Fatalf("pixel type = %T", result.Image)
			}
			if tt.alpha {
				img, ok := result.Image.(*image.NRGBA)
				if !ok {
					t.Fatalf("alpha type = %T", result.Image)
				}
				transparent := false
				for i := 3; i < len(img.Pix); i += 4 {
					if img.Pix[i] < 255 {
						transparent = true
						break
					}
				}
				if !transparent {
					t.Fatal("alpha was flattened")
				}
			}
			config := runGuest(t, wasm, input, heicdecode.DecodeConfig)
			if config.Image != nil || config.Config.Width != result.Image.Bounds().Dx() || config.Config.Height != result.Image.Bounds().Dy() {
				t.Fatal("config differs from displayed dimensions")
			}
			_ = runGuest(t, wasm, input, heicdecode.DecodeExif)
		})
	}
}

func runGuest(t *testing.T, wasm, input []byte, op heicdecode.Operation) heicdecode.Response {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	limits := heicdecode.DefaultLimits(1024 * 1024)
	limits.MaxPixels = 1024 * 1024
	limits.MaxOutputBytes = 8 * 1024 * 1024
	var request bytes.Buffer
	if err := heicdecode.WriteRequest(&request, heicdecode.Request{Operation: op, Input: input}, limits); err != nil {
		t.Fatal(err)
	}
	// Interpreter avoids executable JIT memory; WASI has no preopens, environment,
	// real clock, random source, or sockets. The 128 MiB cap is linear memory only.
	runtime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigInterpreter().WithMemoryLimitPages(2048).WithCloseOnContextDone(true))
	defer func() {
		if err := runtime.Close(context.Background()); err != nil {
			t.Error(err)
		}
	}()
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		t.Fatal(err)
	}
	output := &boundedOutput{limit: int(limits.MaxOutputBytes) + int(limits.MaxMetadataBytes) + 4096}
	config := wazero.NewModuleConfig().WithStdin(&request).WithStdout(output).WithStderr(&boundedOutput{limit: 4096})
	if _, err := runtime.InstantiateWithConfig(ctx, wasm, config); err != nil {
		t.Fatal(err)
	}
	result, err := heicdecode.ReadResponse(bytes.NewReader(output.Bytes()), op, limits)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

type boundedOutput struct {
	bytes.Buffer
	limit int
}

func (w *boundedOutput) Write(p []byte) (int, error) {
	if len(p) > w.limit-w.Len() {
		return 0, errors.New("test guest exceeded output bound")
	}
	return w.Buffer.Write(p)
}
