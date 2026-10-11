//go:build darwin && appleappstore

package heic

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"syscall"

	"github.com/frathe/picfetch/internal/macworker"
)

func workerCommand(ctx context.Context, executable string) *exec.Cmd {
	return macworker.Command(ctx, executable, macworker.HEIC)
}

func inheritedWorkerCommand(ctx context.Context, executable string) *exec.Cmd {
	return exec.CommandContext(ctx, executable)
}

// The XPC service owns the group. Nested decoders must stay in that group so
// service cancellation also retires analysis's native decoding descendants.
func prepareWorker(_ *exec.Cmd) {}

func retireWorker(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return os.ErrProcessDone
	}
	// A caller explicitly owning a group (including isolated test fixtures)
	// remains responsible for retiring it. Production commands inherit XPC's group.
	if cmd.SysProcAttr != nil && cmd.SysProcAttr.Setpgid {
		err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL)
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	return cmd.Process.Signal(syscall.SIGTERM)
}
