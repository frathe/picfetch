package heic

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func prepareWorker(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: windows.CREATE_NO_WINDOW}
}

func retireWorker(cmd *exec.Cmd) error { return cmd.Process.Kill() }
