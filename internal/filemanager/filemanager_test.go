package filemanager

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
)

// afterProgram is cmd.Args without the program name, so a failing
// assertion prints the arguments rather than panicking on a command that
// was never built.
func afterProgram(args []string) []string {
	if len(args) == 0 {
		return nil
	}

	return args[1:]
}

// existingFile gives the Windows path checks something real to stat: they
// refuse a path that isn't there before they ever reach explorer, so a
// hardcoded C:\ literal would fail for that reason on this machine rather
// than for the reason the test is about.
func existingFile(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "holiday one.jpg")
	if err := os.WriteFile(path, []byte("x"), 0o600); err != nil {
		t.Fatalf("writing the fixture file: %v", err)
	}

	return path
}

// TestReveal_DispatchesForThisPlatform pins the dispatcher var itself, not
// just the per-platform functions the rest of this file calls directly: a
// switch that forgot a case would leave every test below passing.
func TestReveal_DispatchesForThisPlatform(t *testing.T) {
	origDBus, origXDG, origRun := lookupDBusSend, lookupXDGOpen, runRevealCommand
	t.Cleanup(func() { lookupDBusSend, lookupXDGOpen, runRevealCommand = origDBus, origXDG, origRun })

	lookupDBusSend = func() (string, error) { return "/usr/bin/dbus-send", nil }
	lookupXDGOpen = func() (string, error) { return "/usr/bin/xdg-open", nil }

	var gotPath string
	runRevealCommand = func(cmd *exec.Cmd) ([]byte, error) {
		gotPath = cmd.Path
		return nil, nil
	}

	path := existingFile(t)
	if err := Reveal(path); err != nil {
		t.Fatalf("Reveal(%q) error = %v", path, err)
	}

	want := map[string]string{"darwin": "open", "windows": "explorer"}[runtime.GOOS]
	if want == "" {
		want = "dbus-send"
	}
	if !strings.Contains(gotPath, want) {
		t.Errorf("cmd.Path = %q, want it to run %s on %s", gotPath, want, runtime.GOOS)
	}
}

// TestRevealDarwin_RunsOpenWithTheRevealFlag: -R is what makes open select
// the file in Finder instead of opening it in whatever app owns its type,
// which would be the exact opposite of this command.
func TestRevealDarwin_RunsOpenWithTheRevealFlag(t *testing.T) {
	origRun := runRevealCommand
	t.Cleanup(func() { runRevealCommand = origRun })

	var gotPath string
	var gotArgs []string
	runRevealCommand = func(cmd *exec.Cmd) ([]byte, error) {
		gotPath, gotArgs = cmd.Path, cmd.Args
		return nil, nil
	}

	if err := revealDarwin("/pics/a.jpg"); err != nil {
		t.Fatalf("revealDarwin() error = %v", err)
	}

	if !strings.Contains(gotPath, "open") {
		t.Errorf("cmd.Path = %q, want it to run open", gotPath)
	}
	if want := []string{"-R", "/pics/a.jpg"}; !slices.Equal(afterProgram(gotArgs), want) {
		t.Errorf("open args = %v, want %v", afterProgram(gotArgs), want)
	}
}

func TestRevealDarwin_ReportsAFailure(t *testing.T) {
	origRun := runRevealCommand
	t.Cleanup(func() { runRevealCommand = origRun })

	boom := errors.New("no such file")
	runRevealCommand = func(*exec.Cmd) ([]byte, error) { return nil, boom }

	if err := revealDarwin("/pics/a.jpg"); !errors.Is(err, boom) {
		t.Errorf("revealDarwin() error = %v, want %v", err, boom)
	}
}

// TestRevealLinux_SendsShowItemsOverDBus pins the freedesktop call that
// actually selects the file. --print-reply is asserted deliberately: without
// it dbus-send does not wait for a reply, so a desktop with no
// FileManager1 implementation would exit 0 and the xdg-open fallback below
// would never run.
func TestRevealLinux_SendsShowItemsOverDBus(t *testing.T) {
	origDBus, origXDG, origRun := lookupDBusSend, lookupXDGOpen, runRevealCommand
	t.Cleanup(func() { lookupDBusSend, lookupXDGOpen, runRevealCommand = origDBus, origXDG, origRun })

	lookupDBusSend = func() (string, error) { return "/usr/bin/dbus-send", nil }
	lookupXDGOpen = func() (string, error) {
		t.Fatal("xdg-open should not be consulted when the FileManager1 call succeeds")
		return "", nil
	}

	var gotPath string
	var gotArgs []string
	runRevealCommand = func(cmd *exec.Cmd) ([]byte, error) {
		gotPath, gotArgs = cmd.Path, cmd.Args
		return nil, nil
	}

	if err := revealLinux("/pics/a.jpg"); err != nil {
		t.Fatalf("revealLinux() error = %v", err)
	}

	if !strings.Contains(gotPath, "dbus-send") {
		t.Errorf("cmd.Path = %q, want it to run dbus-send", gotPath)
	}
	for _, want := range []string{
		"--session",
		"--print-reply",
		"--dest=org.freedesktop.FileManager1",
		"--type=method_call",
		"/org/freedesktop/FileManager1",
		"org.freedesktop.FileManager1.ShowItems",
		"array:string:file:///pics/a.jpg",
		"string:",
	} {
		if !slices.Contains(gotArgs, want) {
			t.Errorf("dbus-send args = %v, want %q present", gotArgs, want)
		}
	}
}

// TestRevealLinux_PercentEncodesTheURI: ShowItems takes URIs, not paths, so
// a space or a '#' in a file name has to survive the D-Bus call the same way
// internal/clipboard's uri-list makes it survive a paste.
func TestRevealLinux_PercentEncodesTheURI(t *testing.T) {
	origDBus, origRun := lookupDBusSend, runRevealCommand
	t.Cleanup(func() { lookupDBusSend, runRevealCommand = origDBus, origRun })

	lookupDBusSend = func() (string, error) { return "/usr/bin/dbus-send", nil }

	var gotArgs []string
	runRevealCommand = func(cmd *exec.Cmd) ([]byte, error) {
		gotArgs = cmd.Args
		return nil, nil
	}

	if err := revealLinux("/pics/holiday #1.jpg"); err != nil {
		t.Fatalf("revealLinux() error = %v", err)
	}

	if want := "array:string:file:///pics/holiday%20%231.jpg"; !slices.Contains(gotArgs, want) {
		t.Errorf("dbus-send args = %v, want %q present", gotArgs, want)
	}
}

func TestRevealLinux_FallsBackToXDGOpenWhenDBusSendIsMissing(t *testing.T) {
	origDBus, origXDG, origRun := lookupDBusSend, lookupXDGOpen, runRevealCommand
	t.Cleanup(func() { lookupDBusSend, lookupXDGOpen, runRevealCommand = origDBus, origXDG, origRun })

	lookupDBusSend = func() (string, error) { return "", errors.New("not found") }
	lookupXDGOpen = func() (string, error) { return "/usr/bin/xdg-open", nil }

	var gotPath string
	var gotArgs []string
	runRevealCommand = func(cmd *exec.Cmd) ([]byte, error) {
		gotPath, gotArgs = cmd.Path, cmd.Args
		return nil, nil
	}

	if err := revealLinux("/pics/a.jpg"); err != nil {
		t.Fatalf("revealLinux() error = %v", err)
	}

	if !strings.Contains(gotPath, "xdg-open") {
		t.Errorf("cmd.Path = %q, want it to run xdg-open", gotPath)
	}
	// The parent directory, not the file: xdg-open on an image would launch
	// an image viewer, which is not what "reveal" means.
	if want := []string{"/pics"}; !slices.Equal(afterProgram(gotArgs), want) {
		t.Errorf("xdg-open args = %v, want %v", afterProgram(gotArgs), want)
	}
}

// TestRevealLinux_FallsBackWhenTheFileManagerCallFails covers the desktop
// that has dbus-send installed but nothing answering FileManager1 - the
// common case on a minimal window manager.
func TestRevealLinux_FallsBackWhenTheFileManagerCallFails(t *testing.T) {
	origDBus, origXDG, origRun := lookupDBusSend, lookupXDGOpen, runRevealCommand
	t.Cleanup(func() { lookupDBusSend, lookupXDGOpen, runRevealCommand = origDBus, origXDG, origRun })

	lookupDBusSend = func() (string, error) { return "/usr/bin/dbus-send", nil }
	lookupXDGOpen = func() (string, error) { return "/usr/bin/xdg-open", nil }

	var ran []string
	runRevealCommand = func(cmd *exec.Cmd) ([]byte, error) {
		ran = append(ran, cmd.Path)
		if strings.Contains(cmd.Path, "dbus-send") {
			return nil, errors.New("org.freedesktop.DBus.Error.ServiceUnknown")
		}
		return nil, nil
	}

	if err := revealLinux("/pics/a.jpg"); err != nil {
		t.Fatalf("revealLinux() error = %v", err)
	}

	if len(ran) != 2 || !strings.Contains(ran[1], "xdg-open") {
		t.Errorf("commands run = %v, want dbus-send then xdg-open", ran)
	}
}

func TestRevealLinux_ReturnsErrorWhenNeitherToolInstalled(t *testing.T) {
	origDBus, origXDG := lookupDBusSend, lookupXDGOpen
	t.Cleanup(func() { lookupDBusSend, lookupXDGOpen = origDBus, origXDG })

	lookupDBusSend = func() (string, error) { return "", errors.New("not found") }
	lookupXDGOpen = func() (string, error) { return "", errors.New("not found") }

	if err := revealLinux("/pics/a.jpg"); err == nil {
		t.Error("expected an error when neither dbus-send nor xdg-open is installed")
	}
}

// TestExplorerCmdLine_QuotesOnlyThePath is the whole reason this command
// line is built by hand instead of by os/exec's own argument escaping,
// which would quote the /select, prefix along with it.
func TestExplorerCmdLine_QuotesOnlyThePath(t *testing.T) {
	got := explorerCmdLine(`C:\Users\me\My Pictures\a.jpg`)

	if want := `explorer.exe /select,"C:\Users\me\My Pictures\a.jpg"`; got != want {
		t.Errorf("explorerCmdLine() = %q, want %q", got, want)
	}
}

// TestRevealWindows_TreatsExplorersNonZeroExitAsSuccess: explorer.exe exits
// non-zero even when it did open and select the file, so an ExitError here
// carries no information and must not become a user-visible failure.
func TestRevealWindows_TreatsExplorersNonZeroExitAsSuccess(t *testing.T) {
	origRun := runRevealCommand
	t.Cleanup(func() { runRevealCommand = origRun })

	var gotPath string
	runRevealCommand = func(cmd *exec.Cmd) ([]byte, error) {
		gotPath = cmd.Path
		return nil, &exec.ExitError{ProcessState: &os.ProcessState{}}
	}

	if err := revealWindows(existingFile(t)); err != nil {
		t.Errorf("revealWindows() error = %v, want nil for explorer's own exit status", err)
	}
	if !strings.Contains(gotPath, "explorer") {
		t.Errorf("cmd.Path = %q, want it to run explorer.exe", gotPath)
	}
}

// TestRevealWindows_ReportsAFailureToStart: an ExitError is explorer being
// explorer, but anything else means the process never ran, which is a real
// failure worth a toast.
func TestRevealWindows_ReportsAFailureToStart(t *testing.T) {
	origRun := runRevealCommand
	t.Cleanup(func() { runRevealCommand = origRun })

	boom := errors.New("executable file not found in %PATH%")
	runRevealCommand = func(*exec.Cmd) ([]byte, error) { return nil, boom }

	if err := revealWindows(existingFile(t)); !errors.Is(err, boom) {
		t.Errorf("revealWindows() error = %v, want %v", err, boom)
	}
}

// TestRevealWindows_ReportsAMissingFile is what the stat is for: since
// explorer's exit status is discarded, a path that isn't there would
// otherwise be reported as a successful reveal.
func TestRevealWindows_ReportsAMissingFile(t *testing.T) {
	origRun := runRevealCommand
	t.Cleanup(func() { runRevealCommand = origRun })

	runRevealCommand = func(*exec.Cmd) ([]byte, error) {
		t.Fatal("explorer should not run for a path that is not there")
		return nil, nil
	}

	if err := revealWindows(filepath.Join(t.TempDir(), "gone.jpg")); err == nil {
		t.Error("expected an error for a missing file")
	}
}
