//go:build !darwin && !linux

package client

import "os/exec"

func configureProcess(_ *exec.Cmd) {}

func killProcess(cmd *exec.Cmd) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
