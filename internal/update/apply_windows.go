//go:build windows

package update

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// createNoWindow is CREATE_NO_WINDOW (same constant as clipboard/wallpaper).
const createNoWindow = 0x08000000

func applyWindows(stage Stage, dest string, options ApplyOptions) error {
	scriptPath := dest + ".apply.cmd"
	script := windowsApplyScript(dest, stage.BinaryPath, os.Getpid(), options)
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		return err
	}
	cmd := exec.Command("cmd", "/C", scriptPath)
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: createNoWindow}
	return cmd.Start()
}

// applyUnix's real implementation (apply_unix.go) only compiles off
// Windows. Apply's GOOS switch makes this stub unreachable; it exists so
// the package still compiles on Windows.
func applyUnix(Stage, string, ApplyOptions) error {
	return errors.New("unix apply only exists in non-windows builds")
}
