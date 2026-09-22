package heic

import (
	"os/exec"
	"syscall"
)

// The kernel retires a native child even if its analysis producer crashes before
// it can run Client.Stop. This is inherited through the child's exec.
func bindWorkerToParent(cmd *exec.Cmd) { cmd.SysProcAttr.Pdeathsig = syscall.SIGKILL }
