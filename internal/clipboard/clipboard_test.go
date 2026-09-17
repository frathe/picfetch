package clipboard

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestClipboardCommandOwnerLifetime(t *testing.T) {
	switch os.Getenv("PICFETCH_CLIPBOARD_OWNER_FIXTURE") {
	case "launcher":
		data, err := io.ReadAll(os.Stdin)
		if err != nil || string(data) != "copied image" {
			os.Exit(2)
		}
		child := exec.Command(os.Args[0], "-test.run=^TestClipboardCommandOwnerLifetime$")
		child.Env = append(os.Environ(), "PICFETCH_CLIPBOARD_OWNER_FIXTURE=owner")
		child.Stdout, child.Stderr = os.Stdout, os.Stderr
		child.ExtraFiles = []*os.File{os.NewFile(3, "release"), os.NewFile(4, "ready")}
		if err := child.Start(); err != nil {
			os.Exit(3)
		}
		_ = child.Process.Release()
		os.Exit(0)
	case "owner":
		ready, release := os.NewFile(4, "ready"), os.NewFile(3, "release")
		_, _ = ready.Write([]byte{1})
		_, _ = io.Copy(io.Discard, release)
		// The owner must still be able to serve the clipboard after return.
		_, outErr := os.Stdout.Write([]byte("owner output"))
		_, errErr := os.Stderr.Write([]byte("owner diagnostic"))
		if outErr == nil && errErr == nil {
			_, _ = ready.Write([]byte{2})
		}
		os.Exit(0)
	}
	if runtime.GOOS == "windows" {
		t.Skip("Linux clipboard-owner inheritance uses Unix extra descriptors")
	}
	release, unblock, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = release.Close(); _ = unblock.Close() }()
	ready, signal, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = ready.Close(); _ = signal.Close() }()
	cmd := exec.Command(os.Args[0], "-test.run=^TestClipboardCommandOwnerLifetime$")
	cmd.Env = append(os.Environ(), "PICFETCH_CLIPBOARD_OWNER_FIXTURE=launcher")
	cmd.Stdin = strings.NewReader("copied image")
	cmd.ExtraFiles = []*os.File{release, signal}
	completed := make(chan error, 1)
	go func() { _, err := runClipboardCommand(cmd); completed <- err }()
	if err := ready.SetReadDeadline(time.Now().Add(5 * time.Second)); err != nil {
		t.Fatal(err)
	}
	var status [1]byte
	if _, err := io.ReadFull(ready, status[:]); err != nil || status[0] != 1 {
		t.Fatalf("owner did not start: %v, %v", status, err)
	}
	select {
	case err := <-completed:
		if err != nil {
			t.Error(err)
		}
	case <-time.After(2 * time.Second):
		t.Error("copy remained pending after launcher exit while clipboard owner held its streams")
		_ = unblock.Close()
		if err := <-completed; err != nil {
			t.Error(err)
		}
	}
	_ = unblock.Close()
	if _, err := io.ReadFull(ready, status[:]); err != nil || status[0] != 2 {
		t.Fatalf("owner streams were closed prematurely: %v, %v", status, err)
	}
}

func TestClipboardCommandFailureDiagnostics(t *testing.T) {
	switch os.Getenv("PICFETCH_CLIPBOARD_ERROR_FIXTURE") {
	case "short":
		_, _ = os.Stderr.WriteString("clipboard unavailable")
		os.Exit(2)
	case "long":
		_, _ = os.Stderr.WriteString(strings.Repeat("diagnostic\n", 16384))
		os.Exit(3)
	}
	for _, fixture := range []string{"short", "long"} {
		t.Run(fixture, func(t *testing.T) {
			cmd := exec.Command(os.Args[0], "-test.run=^TestClipboardCommandFailureDiagnostics$")
			cmd.Env = append(os.Environ(), "PICFETCH_CLIPBOARD_ERROR_FIXTURE="+fixture)
			_, err := runClipboardCommand(cmd)
			exitErr, ok := errors.AsType[*exec.ExitError](err)
			if !ok || len(exitErr.Stderr) == 0 || len(exitErr.Stderr) > 64*1024 {
				t.Fatalf("missing or unbounded launch diagnostic: %v", err)
			}
			if fixture == "short" && (exitErr.ExitCode() != 2 || string(exitErr.Stderr) != "clipboard unavailable") {
				t.Fatalf("launch failure detail changed: %v, %q", exitErr, exitErr.Stderr)
			}
		})
	}
}

func TestCopyImageLinux_PrefersXClip(t *testing.T) {
	origXClip, origWlCopy, origRun := lookupXClip, lookupWlCopy, runClipboardCommand
	t.Cleanup(func() { lookupXClip, lookupWlCopy, runClipboardCommand = origXClip, origWlCopy, origRun })

	lookupXClip = func() (string, error) { return "/usr/bin/xclip", nil }
	lookupWlCopy = func() (string, error) {
		t.Fatal("wl-copy should not be consulted when xclip is present")
		return "", nil
	}

	var gotArgs []string
	var gotStdin []byte
	runClipboardCommand = func(cmd *exec.Cmd) ([]byte, error) {
		gotArgs = cmd.Args
		gotStdin, _ = io.ReadAll(cmd.Stdin)
		return nil, nil
	}

	if err := copyImageLinux([]byte("png-bytes")); err != nil {
		t.Fatalf("copyImageLinux() error = %v", err)
	}

	if !strings.Contains(strings.Join(gotArgs, " "), "image/png") {
		t.Errorf("xclip args = %v, want image/png mime type", gotArgs)
	}
	if string(gotStdin) != "png-bytes" {
		t.Errorf("stdin = %q, want the PNG bytes piped straight through", gotStdin)
	}
}

func TestCopyImageLinux_FallsBackToWlCopy(t *testing.T) {
	origXClip, origWlCopy, origRun := lookupXClip, lookupWlCopy, runClipboardCommand
	t.Cleanup(func() { lookupXClip, lookupWlCopy, runClipboardCommand = origXClip, origWlCopy, origRun })

	lookupXClip = func() (string, error) { return "", errors.New("not found") }
	lookupWlCopy = func() (string, error) { return "/usr/bin/wl-copy", nil }

	var gotPath string
	runClipboardCommand = func(cmd *exec.Cmd) ([]byte, error) {
		gotPath = cmd.Path
		return nil, nil
	}

	if err := copyImageLinux([]byte("png-bytes")); err != nil {
		t.Fatalf("copyImageLinux() error = %v", err)
	}
	if !strings.Contains(gotPath, "wl-copy") {
		t.Errorf("cmd.Path = %q, want it to run wl-copy", gotPath)
	}
}

func TestCopyImageLinux_ReturnsErrorWhenNeitherToolInstalled(t *testing.T) {
	origXClip, origWlCopy := lookupXClip, lookupWlCopy
	t.Cleanup(func() { lookupXClip, lookupWlCopy = origXClip, origWlCopy })

	lookupXClip = func() (string, error) { return "", errors.New("not found") }
	lookupWlCopy = func() (string, error) { return "", errors.New("not found") }

	if err := copyImageLinux([]byte("png-bytes")); err == nil {
		t.Error("expected an error when neither xclip nor wl-copy is installed")
	}
}

func TestCopyImageDarwin_RunsOsascriptAgainstATempPNGFile(t *testing.T) {
	origRun := runClipboardCommand
	t.Cleanup(func() { runClipboardCommand = origRun })

	var gotScript, gotPath string
	runClipboardCommand = func(cmd *exec.Cmd) ([]byte, error) {
		gotPath = cmd.Path
		for i, a := range cmd.Args {
			if a == "-e" && i+1 < len(cmd.Args) {
				gotScript = cmd.Args[i+1]
			}
		}
		return nil, nil
	}

	if err := copyImageDarwin([]byte("png-bytes")); err != nil {
		t.Fatalf("copyImageDarwin() error = %v", err)
	}

	if !strings.Contains(gotPath, "osascript") {
		t.Errorf("cmd.Path = %q, want it to run osascript", gotPath)
	}
	for _, want := range []string{"set the clipboard to", "PNGf", "POSIX file"} {
		if !strings.Contains(gotScript, want) {
			t.Errorf("script does not contain %q:\n%s", want, gotScript)
		}
	}
}

func TestCopyImageWindows_BuildsExpectedScript(t *testing.T) {
	origRun := runClipboardCommand
	t.Cleanup(func() { runClipboardCommand = origRun })

	var gotScript string
	runClipboardCommand = func(cmd *exec.Cmd) ([]byte, error) {
		for i, a := range cmd.Args {
			if a == "-Command" && i+1 < len(cmd.Args) {
				gotScript = cmd.Args[i+1]
			}
		}
		return nil, nil
	}

	if err := copyImageWindows([]byte("png-bytes")); err != nil {
		t.Fatalf("copyImageWindows() error = %v", err)
	}

	for _, want := range []string{
		"System.Windows.Forms",
		"System.Drawing",
		"[System.Drawing.Image]::FromFile",
		"[System.Windows.Forms.Clipboard]::SetDataObject",
		"catch",
		"exit 1",
		"$env:PICFETCH_CLIPBOARD_PNG",
	} {
		if !strings.Contains(gotScript, want) {
			t.Errorf("script does not contain %q:\n%s", want, gotScript)
		}
	}
	if strings.Contains(gotScript, os.TempDir()) {
		t.Errorf("script embeds the temporary path instead of reading it from the environment:\n%s", gotScript)
	}
}

// TestCopyImageWindows_RunsInSTA guards the fix for a Windows bug where
// Ctrl+C silently left the clipboard untouched: System.Windows.Forms
// clipboard access needs an STA thread, so the shell-out passes -STA
// explicitly rather than trusting powershell.exe's default apartment state.
func TestCopyImageWindows_RunsInSTA(t *testing.T) {
	origRun := runClipboardCommand
	t.Cleanup(func() { runClipboardCommand = origRun })

	var gotArgs []string
	runClipboardCommand = func(cmd *exec.Cmd) ([]byte, error) {
		gotArgs = cmd.Args
		return nil, nil
	}

	if err := copyImageWindows([]byte("png-bytes")); err != nil {
		t.Fatalf("copyImageWindows() error = %v", err)
	}

	found := false
	for _, a := range gotArgs {
		if a == "-STA" {
			found = true
		}
	}
	if !found {
		t.Errorf("cmd.Args = %v, want -STA present", gotArgs)
	}
}

func TestWriteTempPNG_WritesAndCleansUp(t *testing.T) {
	path, err := writeTempPNG([]byte("hello"))
	if err != nil {
		t.Fatalf("writeTempPNG() error = %v", err)
	}
	defer func() { _ = os.Remove(path) }()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v", path, err)
	}
	if string(data) != "hello" {
		t.Errorf("file content = %q, want %q", data, "hello")
	}
}

type faultyPNGFile struct {
	*os.File
	writeErr, closeErr error
	short              bool
	calls              *[]string
}

func (f *faultyPNGFile) Write(data []byte) (int, error) {
	*f.calls = append(*f.calls, "write")
	if f.writeErr != nil {
		return 0, f.writeErr
	}
	if f.short {
		return 0, nil
	}
	return f.File.Write(data)
}
func (f *faultyPNGFile) Close() error {
	*f.calls = append(*f.calls, "close")
	return errors.Join(f.File.Close(), f.closeErr)
}

func TestWriteTempPNGFile_PreservesPrimaryFailures(t *testing.T) {
	writeErr := errors.New("write failed")
	closeErr := errors.New("close failed")
	cleanupErr := errors.New("cleanup failed")
	for _, tc := range []struct {
		name                  string
		write, close, cleanup error
		short                 bool
		primary               error
	}{
		{"write", writeErr, nil, nil, false, writeErr},
		{"write and cleanup", writeErr, nil, cleanupErr, false, writeErr},
		{"close", nil, closeErr, nil, false, closeErr},
		{"close and cleanup", nil, closeErr, cleanupErr, false, closeErr},
		{"all failures", writeErr, closeErr, cleanupErr, false, writeErr},
		{"short write", nil, nil, nil, true, io.ErrShortWrite},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := os.CreateTemp(t.TempDir(), "image-*.png")
			if err != nil {
				t.Fatal(err)
			}
			var calls []string
			f := &faultyPNGFile{File: file, writeErr: tc.write, closeErr: tc.close, short: tc.short, calls: &calls}
			remove := func(path string) error {
				calls = append(calls, "remove")
				if path != file.Name() {
					t.Errorf("cleanup path = %q, want %q", path, file.Name())
				}
				if tc.cleanup != nil {
					return tc.cleanup
				}
				return os.Remove(path)
			}
			path, err := writeTempPNGFile(f, []byte("PNG fixture"), remove)
			if path != "" || !errors.Is(err, tc.primary) {
				t.Errorf("result = %q, %v; want primary cause %v", path, err, tc.primary)
			}
			if tc.cleanup != nil && !errors.Is(err, tc.cleanup) {
				t.Errorf("cleanup cause was lost: %v", err)
			}
			if got := strings.Join(calls, ","); got != "write,close,remove" {
				t.Errorf("operations = %s", got)
			}
			if tc.cleanup == nil {
				if _, err := os.Stat(file.Name()); !errors.Is(err, os.ErrNotExist) {
					t.Errorf("temporary file survived cleanup: %v", err)
				}
			}
		})
	}
}
