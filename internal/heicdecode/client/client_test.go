package client

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"io"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/heicdecode"
)

func TestRejectsChangedHelperBeforeReadingSource(t *testing.T) {
	path := filepath.Join(t.TempDir(), "helper")
	if err := os.WriteFile(path, []byte("changed helper"), 0700); err != nil {
		t.Fatal(err)
	}
	client, err := New(Config{Executable: path, SHA256: sha256.Sum256([]byte("expected helper")), Limits: heicdecode.DefaultLimits(0)})
	if err != nil {
		t.Fatal(err)
	}
	defer client.Stop()
	read := false
	_, err = client.Do(context.Background(), heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) {
		read = true
		return nil, nil
	})
	if !errors.Is(err, ErrUnavailable) || read {
		t.Fatalf("changed helper: err=%v, source read=%v", err, read)
	}
	client.Stop()
	client.Wait()
}

// TestMain also supplies an owned protocol peer. It is never a decoder or OS
// sandbox test: these modes exercise only the parent's process/pipe boundary.
func TestMain(m *testing.M) {
	if len(os.Args) == 3 && (os.Args[1] == "--owned-heic-remote" || os.Args[1] == "--owned-heic-remote-cancel") {
		var config PipeConfig
		if err := json.Unmarshal([]byte(os.Args[2]), &config); err != nil {
			os.Exit(2)
		}
		remote, err := OpenRemote(config)
		if err != nil {
			os.Exit(2)
		}
		ctx := context.Background()
		if os.Args[1] == "--owned-heic-remote-cancel" {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, 500*time.Millisecond)
			defer cancel()
		}
		result, err := remote.Do(ctx, heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) { return []byte("owned inherited input"), nil })
		remote.Stop()
		remote.Wait()
		if os.Args[1] == "--owned-heic-remote-cancel" {
			if !errors.Is(err, context.DeadlineExceeded) {
				os.Exit(2)
			}
			_, _ = fmt.Fprintln(os.Stdout, "owned remote cancellation complete")
			os.Exit(0)
		}
		if err != nil || result.Image == nil || result.Image.Bounds() != image.Rect(0, 0, 1, 1) {
			os.Exit(2)
		}
		_, _ = fmt.Fprintln(os.Stdout, "owned remote decode complete")
		os.Exit(0)
	}
	if len(os.Args) == 3 && os.Args[1] == "--owned-heic-descendant" {
		connection, err := net.DialTimeout("tcp4", os.Args[2], time.Second)
		if err != nil {
			os.Exit(2)
		}
		if err = binary.Write(connection, binary.LittleEndian, uint32(os.Getpid())); err != nil {
			os.Exit(2)
		}
		_, _ = io.Copy(io.Discard, connection)
		_ = connection.Close()
		os.Exit(0)
	}
	if len(os.Args) == 7 && os.Args[1] == "--heic-worker-v2" {
		var limits heicdecode.Limits
		if err := json.Unmarshal([]byte(os.Args[2]), &limits); err != nil {
			os.Exit(2)
		}
		mode := strings.TrimSuffix(filepath.Base(os.Args[0]), ".exe")
		if mode == "startup" {
			_, _ = fmt.Fprintln(os.Stderr, "owned startup refusal")
			os.Exit(3)
		}
		if mode == "startup-overflow" {
			_, _ = io.CopyN(os.Stderr, zeroReader{}, int64(limits.MaxDiagnosticBytes)+1)
			os.Exit(3)
		}
		if err := heicdecode.WriteReady(os.Stdout, heicdecode.Ready{WASMMemoryBytes: limits.WASMMemoryBytes, NativeMemoryBytes: limits.OSProcessBytes}); err != nil {
			os.Exit(2)
		}
		switch mode {
		case "success":
			request, err := heicdecode.ReadRequest(os.Stdin, limits)
			if err != nil {
				os.Exit(2)
			}
			result := heicdecode.Response{Image: image.NewNRGBA64(image.Rect(0, 0, 1, 1))}
			if err = heicdecode.WriteResponse(os.Stdout, request.Operation, result, limits); err != nil {
				os.Exit(2)
			}
			os.Exit(0)
		case "hang":
			// Remain alive with all pipes open until the parent terminates us.
			for {
				time.Sleep(time.Hour)
			}
		case "crash":
			_, _ = fmt.Fprintln(os.Stderr, "owned post-readiness diagnostic")
			os.Exit(7)
		case "diagnostics":
			_, _ = io.CopyN(os.Stderr, zeroReader{}, int64(limits.MaxDiagnosticBytes)+1)
			os.Exit(0)
		case "family":
			request, err := heicdecode.ReadRequest(os.Stdin, limits)
			if err != nil {
				os.Exit(2)
			}
			// This peer launches only another copy of its own test executable.
			// The source contains the test's owned loopback listener address.
			descendant := exec.Command(os.Args[0], "--owned-heic-descendant", string(request.Input))
			descendant.Stdout, descendant.Stderr = os.Stdout, os.Stderr
			if err = descendant.Run(); err != nil {
				os.Exit(2)
			}
			os.Exit(0)
		default:
			os.Exit(2)
		}
	}
	os.Exit(m.Run())
}

type zeroReader struct{}

func (zeroReader) Read(p []byte) (int, error) { clear(p); return len(p), nil }

func ownedPeer(t *testing.T, mode string, timeout time.Duration) *Client {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	name := mode
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	path := filepath.Join(t.TempDir(), name)
	// Its basename selects an owned protocol behavior above. Unix uses a
	// hard link; Windows prepares an independent AppContainer-readable copy.
	if err = installOwnedPeer(executable, path); err != nil {
		t.Fatal(err)
	}
	file, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	hash := sha256.New()
	_, copyErr := io.Copy(hash, file)
	_ = file.Close()
	if copyErr != nil {
		t.Fatal(copyErr)
	}
	var digest [32]byte
	copy(digest[:], hash.Sum(nil))
	limits := heicdecode.DefaultLimits(1024 * 1024)
	limits.Timeout = timeout
	client, err := New(Config{Executable: path, SHA256: digest, Limits: limits})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { client.Stop(); client.Wait() })
	return client
}

func TestOwnedPeerTimeoutJoinsBlockedWriter(t *testing.T) {
	client := ownedPeer(t, "hang", 500*time.Millisecond)
	_, err := client.Do(context.Background(), heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) {
		// Larger than a pipe buffer; the peer deliberately never reads stdin.
		return make([]byte, 1024*1024), nil
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("blocked peer termination: %v", err)
	}
	client.Stop()
	client.Wait()
}

func TestOwnedPeerCrashAndDiagnosticsFailClosed(t *testing.T) {
	for _, mode := range []string{"crash", "diagnostics"} {
		t.Run(mode, func(t *testing.T) {
			client := ownedPeer(t, mode, 5*time.Second)
			response, err := client.Do(context.Background(), heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) {
				return []byte("owned protocol test input"), nil
			})
			if err == nil || response.Image != nil {
				t.Fatalf("invalid peer response accepted: %+v, %v", response, err)
			}
			want := heicdecode.ErrInvalidResponse
			if mode == "diagnostics" {
				want = ErrDiagnosticLimit
			}
			if !errors.Is(err, want) {
				t.Fatalf("peer did not exercise expected boundary: %v, want %v", err, want)
			}
			if strings.Contains(err.Error(), "owned post-readiness diagnostic") {
				t.Fatal("post-readiness helper diagnostics reached the caller")
			}
		})
	}
}

func TestOwnedPeerStartupDiagnostics(t *testing.T) {
	for _, mode := range []string{"startup", "startup-overflow"} {
		t.Run(mode, func(t *testing.T) {
			client := ownedPeer(t, mode, 5*time.Second)
			called := false
			_, err := client.Do(context.Background(), heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) {
				called = true
				return nil, nil
			})
			if called || !errors.Is(err, ErrUnavailable) {
				t.Fatalf("startup refusal: input called=%v, err=%v", called, err)
			}
			if mode == "startup" {
				if !strings.Contains(err.Error(), `"owned startup refusal\n"`) {
					t.Fatalf("bounded startup diagnostic missing: %v", err)
				}
			} else if !errors.Is(err, ErrDiagnosticLimit) || strings.Contains(err.Error(), "startup diagnostic") {
				t.Fatalf("oversized startup diagnostic not bounded: %v", err)
			}
		})
	}
}

func TestStopCancelsAdmittedAndWaitingSources(t *testing.T) {
	client := ownedPeer(t, "hang", 5*time.Second)
	entered := make(chan struct{})
	done := make(chan error, 2)
	go func() {
		_, err := client.Do(context.Background(), heicdecode.Decode, func(ctx context.Context, _ int64) ([]byte, error) {
			close(entered)
			<-ctx.Done()
			return nil, ctx.Err()
		})
		done <- err
	}()
	select {
	case <-entered:
	case err := <-done:
		t.Fatalf("owned peer readiness: %v", err)
	case <-time.After(5 * time.Second):
		t.Fatal("owned peer never reached source admission")
	}
	go func() {
		_, err := client.Do(context.Background(), heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) {
			return nil, fmt.Errorf("waiting source was admitted")
		})
		done <- err
	}()
	client.Stop()
	client.Wait()
	for range 2 {
		if err := <-done; !errors.Is(err, context.Canceled) {
			t.Fatalf("stop result: %v", err)
		}
	}
}
