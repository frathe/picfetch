package main

import (
	"bytes"
	"context"
	"errors"
	"image"
	"os"
	"path/filepath"
	"testing"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/frathe/picfetch/internal/heicdecode"
)

// Only unchanged licensed small ordinary fixtures enter the development guest.
// This proves the guest ABI, not native helper resource or sandbox enforcement.
func TestWASIGuestOrdinaryFixtures(t *testing.T) {
	wasm, err := os.ReadFile(filepath.Join("..", "..", "internal", "heicdecode", "worker", "decoder.wasm"))
	if err != nil {
		t.Fatal(err)
	}
	// Compile the fixed artifact once for ABI checks. Native helper tests
	// separately cover the real cold-start deadline for every disposable job.
	ctx, cancel := context.WithTimeout(context.Background(), heicdecode.DefaultLimits(0).Timeout)
	defer cancel()
	runtime := wazero.NewRuntimeWithConfig(ctx, wazero.NewRuntimeConfigCompiler().WithMemoryLimitPages(2048).WithCloseOnContextDone(true))
	defer func() { _ = runtime.Close(context.Background()) }()
	if _, err = wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		t.Fatal(err)
	}
	compiled, err := runtime.CompileModule(ctx, wasm)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = compiled.Close(context.Background()) }()
	for _, tt := range []struct {
		name  string
		depth int
		alpha bool
	}{
		{"basic.heic", 8, false}, {"main10.heic", 8, false}, {"tenbit.heic", 16, false}, {"alpha.heic", 8, true},
		{"chroma422.heic", 0, false}, {"chroma444.heic", 0, false}, {"lossless.heic", 0, false}, {"thumb.heic", 0, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input, err := os.ReadFile(filepath.Join("testdata", tt.name))
			if err != nil {
				t.Fatal(err)
			}
			result := runGuest(t, runtime, compiled, input, heicdecode.Decode)
			if result.Image == nil || result.Image.Bounds().Empty() {
				t.Fatal("missing decoded image")
			}
			depth := 8
			if _, sixteen := result.Image.(*image.NRGBA64); sixteen {
				depth = 16
			}
			if tt.depth != 0 && depth != tt.depth {
				t.Fatalf("pixel type = %T", result.Image)
			}
			if tt.alpha {
				img, ok := result.Image.(*image.NRGBA)
				if !ok {
					t.Fatalf("alpha type = %T", result.Image)
				}
				left := img.NRGBAAt(0, 0).A
				middle := img.NRGBAAt(img.Bounds().Dx()/2, 0).A
				right := img.NRGBAAt(img.Bounds().Dx()-1, 0).A
				if left > 8 || right < 240 || middle <= left || middle >= right {
					t.Fatalf("alpha ramp was lost: left=%d middle=%d right=%d", left, middle, right)
				}
			}
			config := runGuest(t, runtime, compiled, input, heicdecode.DecodeConfig)
			if config.Image != nil || config.Config.Width != result.Image.Bounds().Dx() || config.Config.Height != result.Image.Bounds().Dy() {
				t.Fatal("config differs from displayed dimensions")
			}
			_ = runGuest(t, runtime, compiled, input, heicdecode.DecodeExif)
		})
	}
}

func runGuest(t *testing.T, runtime wazero.Runtime, compiled wazero.CompiledModule, input []byte, op heicdecode.Operation) heicdecode.Response {
	t.Helper()
	limits := heicdecode.DefaultLimits(1024 * 1024)
	ctx, cancel := context.WithTimeout(context.Background(), limits.Timeout)
	defer cancel()
	limits.MaxPixels = 1024 * 1024
	limits.MaxOutputBytes = 8 * 1024 * 1024
	var request bytes.Buffer
	if err := heicdecode.WriteRequest(&request, heicdecode.Request{Operation: op, Input: input}, limits); err != nil {
		t.Fatal(err)
	}
	// No preopens, environment, real clock, random source, or socket handles.
	output := &boundedOutput{limit: int(limits.MaxOutputBytes) + int(limits.MaxMetadataBytes) + 4096}
	config := wazero.NewModuleConfig().WithStdin(&request).WithStdout(output).WithStderr(&boundedOutput{limit: 4096})
	if _, err := runtime.InstantiateModule(ctx, compiled, config); err != nil {
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
