//go:build windows

package update

import (
	"errors"
	"os"
	"os/exec"
	"strconv"
)

// applyWindows installs the staged binary over the running executable from
// inside this process. Controlled Folder Access never trusts cmd.exe, so the
// apply script this replaced was blocked outright whenever PicFetch lived in
// a protected folder; picfetch.exe is judged on its own reputation instead.
func applyWindows(stage Stage, dest string, options ApplyOptions) error {
	return swapBinary(stage.BinaryPath, dest, options, defaultBinaryOps(relaunchWindows))
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
func applyUnix(Stage, string, ApplyOptions) error {
	return errors.New("unix apply only exists in non-windows builds")
}
