package worker

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"

	"github.com/frathe/picfetch/internal/heicdecode"
)

// This runtime seam tests WASI bounds only. Native sandbox qualification is a
// separate subprocess test; it cannot be inferred from an in-process runtime.
func TestRuntimeOrdinaryFixture(t *testing.T) {
	input, err := os.ReadFile(filepath.Join("..", "..", "..", "scripts", "heicbuild", "testdata", "tenbit.heic"))
	if err != nil {
		t.Fatal(err)
	}
	limits := heicdecode.DefaultLimits(1024 * 1024)
	limits.WASMMemoryBytes = 128 * 1024 * 1024
	limits.MaxPixels = 1024 * 1024
	limits.MaxOutputBytes = 8 * 1024 * 1024
	var in, out, diagnostic bytes.Buffer
	if err = heicdecode.WriteRequest(&in, heicdecode.Request{Operation: heicdecode.Decode, Input: input}, limits); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), limits.Timeout)
	defer cancel()
	if err = execute(ctx, decoder, &in, &out, &diagnostic, limits); err != nil {
		t.Fatalf("execute: %v (%s)", err, diagnostic.String())
	}
	result, err := heicdecode.ReadResponse(&out, heicdecode.Decode, limits)
	if err != nil {
		t.Fatal(err)
	}
	if result.Image == nil || result.Image.Bounds().Dx() != 16 || result.Image.Bounds().Dy() != 16 {
		t.Fatalf("unexpected ordinary fixture result: %+v", result.Config)
	}
}

func TestRuntimeEnforcesMemoryLimit(t *testing.T) {
	limits := heicdecode.DefaultLimits(0)
	if err := execute(context.Background(), ownedModule(2, []byte{0x0b}), bytes.NewReader(nil), io.Discard, io.Discard, limits); err != nil {
		t.Fatalf("owned positive control is invalid: %v", err)
	}
	limits.WASMMemoryBytes = 64 * 1024
	// This owned module declares two pages; no image or codec is involved.
	if err := execute(context.Background(), ownedModule(2, []byte{0x0b}), bytes.NewReader(nil), io.Discard, io.Discard, limits); err == nil {
		t.Fatal("module exceeding initial linear-memory limit was admitted")
	}
	// memory.grow(1) must return -1 at the one-page limit. Trap if it succeeds.
	grow := []byte{0x41, 1, 0x40, 0, 0x41, 0x7f, 0x47, 0x04, 0x40, 0, 0x0b, 0x0b}
	if err := execute(context.Background(), ownedModule(1, grow), bytes.NewReader(nil), io.Discard, io.Discard, limits); err != nil {
		t.Fatalf("growth was not refused at the limit: %v", err)
	}
}

func TestRuntimeCancelsOwnedLoop(t *testing.T) {
	limits := heicdecode.DefaultLimits(0)
	limits.Timeout = 50 * time.Millisecond
	// A PicFetch-owned loop exercises context termination, not a decoder defect.
	loop := []byte{0x03, 0x40, 0x0c, 0, 0x0b, 0x0b}
	err := execute(context.Background(), ownedModule(1, loop), bytes.NewReader(nil), io.Discard, io.Discard, limits)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("loop termination = %v", err)
	}
}

func TestRuntimeBoundsGuestDiagnostics(t *testing.T) {
	limits := heicdecode.DefaultLimits(0)
	var diagnostic bytes.Buffer
	if err := execute(context.Background(), diagnosticModule(), bytes.NewReader(nil), io.Discard, &diagnostic, limits); err != nil || diagnostic.String() != "ok" {
		t.Fatalf("owned diagnostic positive control: err=%v, bytes=%q", err, diagnostic.String())
	}
	diagnostic.Reset()
	limits.MaxDiagnosticBytes = 1
	// The owned module writes two bytes to stderr. Both the WASI errno and
	// process result must report refusal; the host must not forward those bytes.
	err := execute(context.Background(), diagnosticModule(), bytes.NewReader(nil), io.Discard, &diagnostic, limits)
	if err == nil || diagnostic.Len() != 0 {
		t.Fatalf("diagnostic bound: err=%v, forwarded=%d", err, diagnostic.Len())
	}
}

func TestRuntimeHasNoFilesystemPreopen(t *testing.T) {
	module := preopenModule()
	limits := heicdecode.DefaultLimits(0)
	if err := execute(context.Background(), module, bytes.NewReader(nil), io.Discard, io.Discard, limits); err != nil {
		t.Fatalf("filesystem descriptor available to guest: %v", err)
	}
	// Deliberately give the owned control a temporary directory. Its assertion
	// must now trap, demonstrating that it observes a real preopen capability.
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	runtime := wazero.NewRuntimeWithConfig(ctx, runtimeConfig().WithCloseOnContextDone(true))
	defer func() { _ = runtime.Close(context.Background()) }()
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		t.Fatal(err)
	}
	config := wazero.NewModuleConfig().WithFSConfig(wazero.NewFSConfig().WithDirMount(t.TempDir(), "/"))
	if _, err := runtime.InstantiateWithConfig(ctx, module, config); err == nil {
		t.Fatal("owned control did not detect the deliberately installed preopen")
	}
}

// preopenModule traps unless fd_prestat_get(3) returns WASI EBADF (8).
// Literal module layouts keep each capability control independently readable.
//
// noinspection DuplicatedCode
func preopenModule() []byte {
	module := []byte{0, 0x61, 0x73, 0x6d, 1, 0, 0, 0}
	section := func(id byte, content []byte) {
		module = append(module, id, byte(len(content)))
		module = append(module, content...)
	}
	section(1, []byte{2, 0x60, 0, 0, 0x60, 2, 0x7f, 0x7f, 1, 0x7f})
	imports := append([]byte{1, 22}, []byte("wasi_snapshot_preview1")...)
	imports = append(imports, 14)
	imports = append(imports, []byte("fd_prestat_get")...)
	section(2, append(imports, 0, 1))
	section(3, []byte{1, 0})
	section(5, []byte{1, 1, 1, 1})
	section(7, []byte{2, 6, '_', 's', 't', 'a', 'r', 't', 0, 1, 6, 'm', 'e', 'm', 'o', 'r', 'y', 2, 0})
	body := []byte{0, 0x41, 3, 0x41, 0, 0x10, 0, 0x41, 8, 0x47, 0x04, 0x40, 0, 0x0b, 0x0b}
	section(10, append([]byte{1, byte(len(body))}, body...))
	return module
}

// diagnosticModule retains its literal layout for an independent stream control.
//
// noinspection DuplicatedCode
func diagnosticModule() []byte {
	module := []byte{0, 0x61, 0x73, 0x6d, 1, 0, 0, 0}
	section := func(id byte, content []byte) {
		module = append(module, id, byte(len(content)))
		module = append(module, content...)
	}
	section(1, []byte{2, 0x60, 0, 0, 0x60, 4, 0x7f, 0x7f, 0x7f, 0x7f, 1, 0x7f})
	imports := append([]byte{1, 22}, []byte("wasi_snapshot_preview1")...)
	imports = append(imports, 8)
	imports = append(imports, []byte("fd_write")...)
	section(2, append(imports, 0, 1))
	section(3, []byte{1, 0})
	section(5, []byte{1, 1, 1, 1})
	section(7, []byte{2, 6, '_', 's', 't', 'a', 'r', 't', 0, 1, 6, 'm', 'e', 'm', 'o', 'r', 'y', 2, 0})
	// fd_write(stderr, iovec at 0, count 1, result at 12); discard errno.
	body := []byte{0, 0x41, 2, 0x41, 0, 0x41, 1, 0x41, 12, 0x10, 0, 0x1a, 0x0b}
	section(10, append([]byte{1, byte(len(body))}, body...))
	// One iovec points to the two literal bytes following it.
	section(11, []byte{1, 0, 0x41, 0, 0x0b, 10, 8, 0, 0, 0, 2, 0, 0, 0, 'o', 'k'})
	return module
}

// ownedModule builds one function and a one-or-two-page memory from literal
// instructions. It has no imports, image input, or third-party parser code.
func ownedModule(pages byte, body []byte) []byte {
	module := []byte{0, 0x61, 0x73, 0x6d, 1, 0, 0, 0,
		1, 4, 1, 0x60, 0, 0, // one () -> () type
		3, 2, 1, 0, // one function
		5, 4, 1, 1, pages, 2, // bounded memory
		7, 10, 1, 6, '_', 's', 't', 'a', 'r', 't', 0, 0,
		10, byte(len(body) + 3), 1, byte(len(body) + 1), 0}
	return append(module, body...)
}

func TestNativeDenialErrorsRequirePermission(t *testing.T) {
	for _, err := range []error{os.ErrPermission, &os.PathError{Op: "open", Path: "owned", Err: os.ErrPermission}} {
		if !permissionDenied(err) {
			t.Fatalf("permission refusal not recognized: %v", err)
		}
	}
	for _, err := range []error{nil, os.ErrNotExist, context.DeadlineExceeded, errors.New("connection refused")} {
		if permissionDenied(err) {
			t.Fatalf("non-permission failure counted as isolation: %v", err)
		}
	}
}

func TestNativeNetworkDenialRejectsUnisolatedProcess(t *testing.T) {
	// Even on Windows a timeout is insufficient in an ordinary process. The
	// native API must confirm the nonexempt AppContainer identity and policy.
	for _, probeErr := range []error{nil, os.ErrDeadlineExceeded, context.Canceled, errors.New("connection refused")} {
		if err := confirmNetworkDenial(context.Background(), "127.0.0.1", probeErr); err == nil {
			t.Fatalf("unisolated network failure accepted as denial: %v", probeErr)
		}
	}
}
