//go:build darwin || linux

package client

import (
	"os/exec"
	"syscall"

	"github.com/frathe/picfetch/internal/heicdecode"
)

type unixProcess struct{ cmd *exec.Cmd }

func startProcess(cmd *exec.Cmd, _ heicdecode.Limits) (helperProcess, error) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	return &unixProcess{cmd: cmd}, nil
}

func (p *unixProcess) Kill() {
	if p.cmd.Process != nil {
		_ = syscall.Kill(-p.cmd.Process.Pid, syscall.SIGKILL)
	}
}

func (p *unixProcess) Wait() error { return p.cmd.Wait() }
