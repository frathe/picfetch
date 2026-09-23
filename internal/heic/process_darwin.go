package heic

import (
	"context"
	"os/exec"
)

func workerCommand(ctx context.Context, executable string) *exec.Cmd {
	return exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1) (allow default) (deny network*)", executable)
}

func inheritedWorkerCommand(ctx context.Context, executable string) *exec.Cmd {
	return exec.CommandContext(ctx, executable)
}

func bindWorkerToParent(_ *exec.Cmd) {}
