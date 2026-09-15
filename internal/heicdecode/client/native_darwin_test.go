//go:build heicnative && darwin && cgo

package client

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"errors"
	"image"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/heicdecode"
)

// Run on macOS outside a surrounding sandbox after building with heicnative.
// It creates an ad-hoc signed helper bundle and uses only owned probe files,
// owned loopback listeners and ordinary licensed image fixtures.
func TestNativeMacSandboxHelper(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	config := buildNativeMacHelper(t, root, "compiler")
	config.Limits.MaxInputBytes = 1024 * 1024
	config.Limits.WASMMemoryBytes = 128 * 1024 * 1024
	config.Limits.MaxPixels = 1024 * 1024
	config.Limits.MaxOutputBytes = 8 * 1024 * 1024
	client, err := New(config)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { client.Stop(); client.Wait() }()
	fixture := filepath.Join(root, "scripts", "heicbuild", "testdata", "tenbit.heic")
	t.Run("missing sandbox refuses source", func(t *testing.T) {
		unprotected := buildNativeMacHelper(t, root, "unprotected-control")
		other, newErr := New(unprotected)
		if newErr != nil {
			t.Fatal(newErr)
		}
		defer func() { other.Stop(); other.Wait() }()
		read := false
		_, decodeErr := other.Do(context.Background(), heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) {
			read = true
			return nil, nil
		})
		if !errors.Is(decodeErr, ErrUnavailable) || read {
			t.Fatalf("missing sandbox: error=%v, read=%v", decodeErr, read)
		}
	})
	t.Run("ordinary image after native readiness", func(t *testing.T) {
		result, decodeErr := client.Do(context.Background(), heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) {
			return os.ReadFile(fixture)
		})
		if decodeErr != nil {
			t.Fatal(decodeErr)
		}
		if img, ok := result.Image.(*image.NRGBA64); !ok || img.Bounds().Dx() != 16 || img.Bounds().Dy() != 16 {
			t.Fatalf("ordinary ten-bit result: %T", result.Image)
		}
	})
	t.Run("cancellation joins admitted and queued work", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		entered := make(chan struct{})
		done := make(chan error, 1)
		go func() {
			_, decodeErr := client.Do(ctx, heicdecode.Decode, func(ctx context.Context, _ int64) ([]byte, error) {
				close(entered)
				<-ctx.Done()
				return nil, ctx.Err()
			})
			done <- decodeErr
		}()
		select {
		case <-entered:
		case err = <-done:
			t.Fatalf("failed before source admission: %v", err)
		case <-time.After(30 * time.Second):
			t.Fatal("source was never admitted")
		}
		cancelled, cancelQueued := context.WithCancel(context.Background())
		cancelQueued()
		_, queuedErr := client.Do(cancelled, heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) {
			t.Error("cancelled queued source was read")
			return nil, nil
		})
		if !errors.Is(queuedErr, context.Canceled) {
			t.Fatalf("queued cancellation: %v", queuedErr)
		}
		cancel()
		if decodeErr := <-done; !errors.Is(decodeErr, context.Canceled) {
			t.Fatalf("admitted cancellation: %v", decodeErr)
		}
		client.Stop()
		client.Wait()
	})
}

func buildNativeMacHelper(t *testing.T, root, engine string) Config {
	t.Helper()
	var err error
	tags := "no_emoji,nodynamic"
	entitlements := filepath.Join(root, "packaging", "heic", "macos.entitlements.plist")
	if engine == "interpreter" {
		tags += ",heicinterpreter"
		entitlements = filepath.Join(t.TempDir(), "interpreter.plist")
		const minimal = `<?xml version="1.0"?><plist version="1.0"><dict><key>com.apple.security.app-sandbox</key><true/></dict></plist>`
		if err = os.WriteFile(entitlements, []byte(minimal), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if engine == "unprotected-control" {
		entitlements = filepath.Join(t.TempDir(), "no-sandbox.plist")
		const empty = `<?xml version="1.0"?><plist version="1.0"><dict/></plist>`
		if err = os.WriteFile(entitlements, []byte(empty), 0600); err != nil {
			t.Fatal(err)
		}
	}
	bundle := filepath.Join(t.TempDir(), "HEICWorker.app")
	executable := filepath.Join(bundle, "Contents", "MacOS", "picfetch-heic-worker")
	if err = os.MkdirAll(filepath.Dir(executable), 0700); err != nil {
		t.Fatal(err)
	}
	build := exec.Command("go", "build", "-tags", tags, "-o", executable, "./cmd/picfetch-heic-worker")
	build.Dir = root
	if out, buildErr := build.CombinedOutput(); buildErr != nil {
		t.Fatalf("build helper: %v: %s", buildErr, out)
	}
	plist, err := os.ReadFile(filepath.Join(root, "packaging", "heic", "macos.Info.plist"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(bundle, "Contents", "Info.plist"), plist, 0600); err != nil {
		t.Fatal(err)
	}
	sign := exec.Command("/usr/bin/codesign", "--force", "--sign", "-", "--options", "runtime", "--entitlements", entitlements, bundle)
	if out, signErr := sign.CombinedOutput(); signErr != nil {
		t.Fatalf("sign helper: %v: %s", signErr, out)
	}
	verify := exec.Command("/usr/bin/codesign", "--verify", "--strict", bundle)
	if out, verifyErr := verify.CombinedOutput(); verifyErr != nil {
		t.Fatalf("verify helper: %v: %s", verifyErr, out)
	}
	contents, err := os.ReadFile(executable)
	if err != nil {
		t.Fatal(err)
	}
	limits := heicdecode.DefaultLimits(0)
	config := Config{Executable: executable, SHA256: sha256.Sum256(contents), Limits: limits}
	return config
}

// This comparison records an ordinary owned image within the production
// deadline. Interpreter deadline refusal is an observed result, not a skipped
// sandbox gate or permission to extend resource limits.
func TestNativeMacRuntimeChoice(t *testing.T) {
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	for _, engine := range []string{"interpreter", "compiler"} {
		t.Run(engine, func(t *testing.T) {
			config := buildNativeMacHelper(t, root, engine)
			client, err := New(config)
			if err != nil {
				t.Fatal(err)
			}
			defer func() { client.Stop(); client.Wait() }()
			for _, name := range []string{"basic.heic", "photo-gradient.heic.gz"} {
				start := time.Now()
				result, err := client.Do(context.Background(), heicdecode.Decode, func(_ context.Context, maxBytes int64) ([]byte, error) {
					path := filepath.Join(root, "scripts", "heicbuild", "testdata", name)
					if name == "basic.heic" {
						return os.ReadFile(path)
					}
					file, err := os.Open(path)
					if err != nil {
						return nil, err
					}
					defer func() { _ = file.Close() }()
					reader, err := gzip.NewReader(file)
					if err != nil {
						return nil, err
					}
					defer func() { _ = reader.Close() }()
					return io.ReadAll(io.LimitReader(reader, maxBytes+1))
				})
				if err != nil {
					t.Logf("%s: elapsed=%s, result=%v", name, time.Since(start), err)
					if engine != "interpreter" || !errors.Is(err, context.DeadlineExceeded) {
						t.Fatal(err)
					}
					continue
				}
				t.Logf("%s: elapsed=%s, bounds=%v", name, time.Since(start), result.Image.Bounds())
				if name == "photo-gradient.heic.gz" && result.Image.Bounds() != image.Rect(0, 0, 4032, 3024) {
					t.Fatal("gradient dimensions differ")
				}
			}
		})
	}
}
