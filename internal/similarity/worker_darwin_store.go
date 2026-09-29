//go:build darwin && appleappstore

package similarity

import (
	"context"
	"os/exec"
	"syscall"

	"github.com/frathe/picfetch/internal/macworker"
)

func workerCommand(ctx context.Context, executable string) *exec.Cmd {
	return macworker.Command(ctx, executable, macworker.Similarity)
}

func stopWorkerProcess(cmd *exec.Cmd) error { return cmd.Process.Signal(syscall.SIGTERM) }

func isolateWorker() error { return nil }
