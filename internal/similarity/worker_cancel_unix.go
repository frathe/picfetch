//go:build !windows

package similarity

import (
	"os/exec"
	"syscall"
)

// Let WorkerMain cancel and join nested decoder work before producer exit.
// The command's existing WaitDelay still bounds a producer that ignores SIGTERM.
func gracefulWorker(cmd *exec.Cmd) *exec.Cmd {
	cmd.Cancel = func() error { return cmd.Process.Signal(syscall.SIGTERM) }
	return cmd
}
