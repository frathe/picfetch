package similarity

import (
	"context"
	"os"
	"os/exec"
	"syscall"
)

// Windows runs a normal local subprocess. It does not configure AppContainer,
// BFS policies, firewall rules, or filesystem permissions.
func workerCommand(ctx context.Context, executable string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, executable)
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
	return cmd
}

func isolateWorker() error { return nil }

// Windows os.File.Close cancels pending pipe I/O. The private worker owns stdin;
// closing it releases the control reader even if the parent's pipe stays open.
func controlInput() (*os.File, error) { return os.Stdin, nil }
