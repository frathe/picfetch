//go:build !windows

package wallpaper

import (
	"os/exec"
)

// hideConsoleWindow's real implementation (windows.go) sets a Windows-only
// SysProcAttr field, so it needs this no-op twin for every other platform:
// setWindows is compiled - and unit-tested - everywhere, even though Set
// only ever dispatches to it on Windows.
func hideConsoleWindow(_ *exec.Cmd) {}

func setWindowsTarget(_ Request) error {
	return &TargetUnsupportedError{Platform: "Windows", Reason: "the native adapter is unavailable in this build"}
}
