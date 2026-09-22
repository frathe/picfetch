//go:build !darwin

package heic

import (
	"context"
	"os/exec"
)

func workerCommand(ctx context.Context, executable string) *exec.Cmd {
	return exec.CommandContext(ctx, executable)
}

func inheritedWorkerCommand(ctx context.Context, executable string) *exec.Cmd {
	return workerCommand(ctx, executable)
}
