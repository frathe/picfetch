// Package filemanager opens the current OS's own file manager with one file
// selected, for the Actions > "Reveal in file manager" action - the step
// between knowing which image is on screen and doing anything with it
// outside PicFetch. Each platform needs its own approach: `open -R` on macOS,
// `explorer.exe /select,` on Windows, and the freedesktop
// org.freedesktop.FileManager1 D-Bus interface on Linux, falling back to
// `xdg-open` on the file's folder where nothing implements it.
//
// macOS goes through /usr/bin/open rather than cgo/AppKit, unlike this
// repository's other macOS integrations (internal/trash, internal/wallpaper,
// internal/filepicker). Those use frameworks because their only shell-out
// alternative was an AppleScript "tell application …", which triggers the
// one-time Automation permission prompt a direct framework call avoids. open
// is a LaunchServices binary, not an Apple Event addressed at Finder, so
// there is no prompt to avoid here and nothing for cgo to buy.
package filemanager

import (
	"errors"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
)

// Reveal dispatches to the current OS's own way of showing path in a file
// manager. A var so callers' tests can stub the whole platform dispatch,
// exactly as trash.Move and wallpaper.Set are.
var Reveal = func(path string) error {
	switch runtime.GOOS {
	case "darwin":
		return revealDarwin(path)
	case "windows":
		return revealWindows(path)
	default:
		return revealLinux(path)
	}
}

// lookupDBusSend/lookupXDGOpen find their respective binaries; vars so tests
// can force either outcome deterministically instead of depending on what's
// installed on the machine running the test.
var lookupDBusSend = func() (string, error) { return exec.LookPath("dbus-send") }
var lookupXDGOpen = func() (string, error) { return exec.LookPath("xdg-open") }

// runRevealCommand runs the already-built command; a var so tests can stub
// the process out entirely.
var runRevealCommand = func(cmd *exec.Cmd) ([]byte, error) { return cmd.Output() }

// The freedesktop.org "show these items, selected" interface. Nautilus,
// Dolphin, Nemo, Thunar and PCManFM all export it, which makes it the one
// portable way to ask for a *selected* file rather than just an open folder.
const (
	fileManagerDest      = "org.freedesktop.FileManager1"
	fileManagerObject    = "/org/freedesktop/FileManager1"
	fileManagerShowItems = "org.freedesktop.FileManager1.ShowItems"
)

// revealDarwin asks LaunchServices to select path in Finder. -R is the whole
// point of the call: without it, open would hand the file to whatever app
// owns its type - which for an image is another viewer, the opposite of what
// this command means.
func revealDarwin(path string) error {
	_, err := runRevealCommand(exec.Command("open", "-R", path))

	return err
}

// revealLinux prefers the FileManager1 D-Bus call, the only portable way to
// get the file *selected*, and falls back to opening its folder with
// xdg-open where no file manager answers.
//
// --print-reply is load-bearing rather than diagnostic: without it dbus-send
// does not wait for a reply at all and exits 0 even when the destination
// cannot be activated, so a desktop with no FileManager1 implementation
// would report success and never reach the fallback below. The reply timeout
// bounds the wait that flag introduces, since this runs on a goroutine the
// viewer waits out at shutdown.
//
// The fallback deliberately opens filepath.Dir(path), not path: xdg-open on
// an image launches an image viewer.
func revealLinux(path string) error {
	if bin, err := lookupDBusSend(); err == nil {
		cmd := exec.Command(bin,
			"--session",
			"--print-reply",
			"--reply-timeout=5000",
			"--dest="+fileManagerDest,
			"--type=method_call",
			fileManagerObject,
			fileManagerShowItems,
			"array:string:"+fileURI(path),
			// The interface's startup-id argument; empty is legal and is
			// what a caller with no launch context passes.
			"string:",
		)
		if _, err := runRevealCommand(cmd); err == nil {
			return nil
		}
	}

	bin, err := lookupXDGOpen()
	if err != nil {
		return errors.New("no org.freedesktop.FileManager1 file manager answered and xdg-open is not installed")
	}

	_, err = runRevealCommand(exec.Command(bin, filepath.Dir(path)))

	return err
}

// fileURI converts a filesystem path to the file:// URI ShowItems takes.
// Built through net/url rather than by concatenation so the characters that
// are legal in a file name but structural in a URI - '#', which would
// otherwise start a fragment and truncate the path, '?', and spaces - are
// percent-encoded, the same reason internal/wallpaper and
// internal/clipboard build their own URIs this way.
func fileURI(path string) string {
	u := url.URL{Scheme: "file", Path: path}

	return u.String()
}

// revealWindows opens Explorer with path selected.
//
// The stat is not defensive tidying: explorer.exe's exit status is discarded
// below, so without it a path that is no longer there would be reported as a
// successful reveal.
//
// explorer.exe returns a non-zero exit status even when it did open the
// folder and select the file, so an *exec.ExitError carries no information
// and must not become a user-visible failure. Any other error means the
// process never started, which is real and is returned.
func revealWindows(path string) error {
	if _, err := os.Stat(path); err != nil {
		return err
	}

	cmd := exec.Command("explorer.exe")
	applyExplorerCommandLine(cmd, path)

	if _, err := runRevealCommand(cmd); err != nil {
		if _, isExit := errors.AsType[*exec.ExitError](err); isExit {
			return nil
		}

		return err
	}

	return nil
}

// explorerCmdLine is the exact command line applyExplorerCommandLine hands
// to CreateProcess, kept portable and pure so the one genuinely tricky part
// of the Windows path is unit-tested from any platform.
//
// The quotes go around the path and nothing else. os/exec's own argument
// escaping would quote the whole "/select,<path>" argument as soon as the
// path contained a space, and explorer.exe does not parse its command line
// with CommandLineToArgvW, so that form is not safely interchangeable with
// the one Microsoft documents. No escaping is needed inside the quotes: '"'
// is an illegal character in a Windows file name, the same argument
// internal/trash's escapePowerShellPath makes for its own quoting.
func explorerCmdLine(path string) string {
	return `explorer.exe /select,"` + path + `"`
}
