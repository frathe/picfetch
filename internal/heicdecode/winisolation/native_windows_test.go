//go:build windows && (amd64 || arm64)

package winisolation

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"

	"github.com/frathe/picfetch/internal/heicdecode"
)

// This process is PicFetch's owned native control, with no image or codec.
func TestMain(m *testing.M) {
	if len(os.Args) == 2 && os.Args[1] == "--owned-exit" {
		os.Exit(0)
	}
	if len(os.Args) == 3 && os.Args[1] == "--owned-windows-policy" {
		var limits heicdecode.Limits
		if json.Unmarshal([]byte(os.Args[2]), &limits) != nil {
			os.Exit(2)
		}
		if err := ownedPolicyControl(limits); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(3)
		}
		_, _ = fmt.Fprintln(os.Stdout, "verified")
		_, _ = io.Copy(io.Discard, os.Stdin)
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func ownedPolicyControl(limits heicdecode.Limits) error {
	if err := Verify(limits); err != nil {
		return err
	}
	if os.Getenv("PICFETCH_OWNED_PARENT_ENVIRONMENT") != "" {
		return errors.New("parent environment reached the helper")
	}
	windowsDirectory, err := windows.GetWindowsDirectory()
	if err != nil || os.Getenv("SystemRoot") != windowsDirectory {
		return fmt.Errorf("helper SystemRoot does not match Windows directory: %w", err)
	}
	// The request is capped at 65 MiB even if the tested restriction is broken.
	// No page is touched; an unexpected success is freed immediately.
	if limits.OSProcessBytes != 64*1024*1024 {
		return errors.New("owned memory control requires its small fixed budget")
	}
	address, err := windows.VirtualAlloc(0, 65*1024*1024, windows.MEM_RESERVE|windows.MEM_COMMIT, windows.PAGE_READWRITE)
	if address != 0 {
		_ = windows.VirtualFree(address, 0, windows.MEM_RELEASE)
		return errors.New("job accepted commitment beyond its memory ceiling")
	}
	if !errors.Is(err, windows.ERROR_COMMITMENT_LIMIT) && !errors.Is(err, windows.ERROR_NOT_ENOUGH_MEMORY) && !errors.Is(err, windows.ERROR_NOT_ENOUGH_QUOTA) {
		return fmt.Errorf("unexpected commitment refusal: %w", err)
	}
	// A second copy of this inert control must be refused by the kernel.
	child, err := os.StartProcess(os.Args[0], []string{os.Args[0], "--owned-exit"}, &os.ProcAttr{})
	if child != nil {
		_ = child.Kill()
		_, _ = child.Wait()
		return errors.New("job admitted a second process")
	}
	if !errors.Is(err, os.ErrPermission) && !errors.Is(err, windows.ERROR_NOT_ENOUGH_QUOTA) && !errors.Is(err, windows.ERROR_CHILD_PROCESS_BLOCKED) {
		return fmt.Errorf("unexpected child-process refusal: %w", err)
	}
	return nil
}

func TestNativeWindowsJobAndToken(t *testing.T) {
	t.Setenv("PICFETCH_OWNED_PARENT_ENVIRONMENT", "owned test sentinel")
	source, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	executable := filepath.Join(directory, "owned-policy.exe")
	data, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(executable, data, 0700); err != nil {
		t.Fatal(err)
	}
	if err = PrepareExecutable(executable); err != nil {
		t.Fatal(err)
	}
	input, feed, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = input.Close(); _ = feed.Close() }()
	output, response, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = output.Close(); _ = response.Close() }()
	diagnostic, warnings, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = diagnostic.Close(); _ = warnings.Close() }()
	limits := heicdecode.DefaultLimits(0)
	limits.Timeout = 10 * time.Second
	limits.OSProcessBytes = 64 * 1024 * 1024
	limits.WASMMemoryBytes = 16 * 1024 * 1024
	encoded, err := json.Marshal(limits)
	if err != nil {
		t.Fatal(err)
	}
	process, err := Start(executable, []string{"--owned-windows-policy", string(encoded)}, [3]*os.File{input, response, warnings}, limits)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { process.Kill(); _ = process.Wait() }()
	deadline := time.AfterFunc(limits.Timeout, process.Kill)
	defer deadline.Stop()
	_ = input.Close()
	_ = response.Close()
	_ = warnings.Close()
	_ = feed.Close()
	actual, readErr := io.ReadAll(output)
	detail, detailErr := io.ReadAll(diagnostic)
	waitErr := process.Wait()
	if readErr != nil || detailErr != nil || waitErr != nil || string(actual) != "verified\n" {
		t.Fatalf("native policy: output=%q diagnostics=%q read=%v/%v wait=%v", actual, detail, readErr, detailErr, waitErr)
	}
}

func TestNativeWindowsUnsandboxedRefused(t *testing.T) {
	if Verify(heicdecode.DefaultLimits(0)) == nil {
		t.Fatal("ordinary test process accepted as isolated helper")
	}
}
