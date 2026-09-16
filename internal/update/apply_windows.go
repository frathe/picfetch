//go:build windows

package update

import (
	"errors"
	"os"
	"os/exec"
	"strconv"

	"golang.org/x/sys/windows"
)

// applyWindows installs the staged binary over the running executable from
// inside this process. Controlled Folder Access never trusts cmd.exe, so the
// apply script this replaced was blocked outright whenever PicFetch lived in
// a protected folder; picfetch.exe is judged on its own reputation instead.
func applyWindows(stage Stage, dest string, options ApplyOptions) error {
	staged, err := openVerifiedStageBinary(stage)
	if err != nil {
		return &ApplyError{Op: "verify", Path: dest, Err: err}
	}
	defer func() { _ = staged.Close() }()
	return swapBinaryFrom(staged, stage.verification.BinaryDigest, dest, options, defaultBinaryOps(relaunchWindows))
}

// A read-only share holds the verified source stable through the final copy.
// Existing writers prevent admission, and new writes/deletes stay blocked until
// applyWindows closes this handle.
func openStageBinary(path string) (*os.File, error) {
	name, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return nil, err
	}
	h, err := windows.CreateFile(name, windows.GENERIC_READ, windows.FILE_SHARE_READ,
		nil, windows.OPEN_EXISTING, windows.FILE_ATTRIBUTE_NORMAL, 0)
	if err != nil {
		return nil, &os.PathError{Op: "open", Path: path, Err: err}
	}
	return os.NewFile(uintptr(h), path), nil
}

// relaunchWindows starts the freshly installed executable and tells it which
// process to wait for. The new instance must not touch preferences before the
// old one is gone: Apply runs inside Fyne's stopped callback, and Fyne saves
// preferences immediately after that callback returns.
func relaunchWindows(dest string) error {
	return windowsRelaunchCommand(dest, os.Getpid()).Start()
}

// windowsRelaunchCommand sets no CREATE_NO_WINDOW, unlike the console helpers
// in trash, wallpaper, filepicker and clipboard: those start console-subsystem
// tools, while this starts picfetch.exe, which release builds link as
// GUI-subsystem (fyne-cross without -console) and which therefore has no
// console for the flag to suppress.
func windowsRelaunchCommand(dest string, pid int) *exec.Cmd {
	cmd := exec.Command(dest)
	// The successor is started with no arguments because main.go treats every
	// bare argument as a file to open, so the PID travels in an inherited
	// environment instead of a replaced one: the user's own variables have to
	// survive an update.
	cmd.Env = append(os.Environ(), AwaitPIDEnv+"="+strconv.Itoa(pid))
	return cmd
}

// applyUnix's real implementation (apply_unix.go) only compiles off
// Windows. Apply's GOOS switch makes this stub unreachable; it exists so
// the package still compiles on Windows.
func applyUnix(_ Stage, _ string, _ ApplyOptions) error {
	return errors.New("unix apply only exists in non-windows builds")
}
