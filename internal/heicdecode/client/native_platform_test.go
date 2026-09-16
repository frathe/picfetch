//go:build heicnative && (linux || windows) && (amd64 || arm64)

package client

import (
	"compress/gzip"
	"context"
	"image"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/heicdecode"
)

func TestNativeSandboxHelper(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}

	packageRoot := t.TempDir()
	build := exec.Command("go", "run", "./scripts/heicpackage", "-os", runtime.GOOS, "-arch", runtime.GOARCH, "-out", packageRoot)
	build.Dir = root
	if output, buildErr := build.CombinedOutput(); buildErr != nil {
		t.Fatalf("stage helper package: %v: %s", buildErr, output)
	}
	mainExecutable := filepath.Join(packageRoot, "picfetch")
	if runtime.GOOS == "windows" {
		mainExecutable += ".exe"
	}
	if err = os.WriteFile(mainExecutable, []byte("owned installation marker"), 0700); err != nil {
		t.Fatal(err)
	}
	client, err := OpenInstalled(context.Background(), mainExecutable, filepath.Join(t.TempDir(), "heic-helpers"), heicdecode.DefaultLimits(0))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { client.Stop(); client.Wait() }()
	for _, fixture := range []struct {
		name   string
		bounds image.Rectangle
	}{
		{"tenbit.heic", image.Rect(0, 0, 16, 16)},
		{"photo-gradient.heic.gz", image.Rect(0, 0, 4032, 3024)},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			start := time.Now()
			result, decodeErr := client.Do(context.Background(), heicdecode.Decode, func(_ context.Context, maxBytes int64) ([]byte, error) {
				file, readErr := os.Open(filepath.Join(root, "scripts", "heicbuild", "testdata", fixture.name))
				if readErr != nil {
					return nil, readErr
				}
				defer func() { _ = file.Close() }()
				var reader io.Reader = file
				if filepath.Ext(fixture.name) == ".gz" {
					compressed, zipErr := gzip.NewReader(file)
					if zipErr != nil {
						return nil, zipErr
					}
					defer func() { _ = compressed.Close() }()
					reader = compressed
				}
				return io.ReadAll(io.LimitReader(reader, maxBytes+1))
			})
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			if result.Image == nil || result.Image.Bounds() != fixture.bounds {
				t.Fatalf("unexpected result: %+v", result.Config)
			}
			if fixture.name == "tenbit.heic" {
				if _, ok := result.Image.(*image.NRGBA64); !ok {
					t.Fatalf("ten-bit precision lost: %T", result.Image)
				}
			}
			t.Logf("%s: elapsed=%s, bounds=%v", fixture.name, time.Since(start), result.Image.Bounds())
		})
	}
}
