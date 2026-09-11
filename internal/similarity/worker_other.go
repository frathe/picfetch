//go:build !windows && !(linux && (amd64 || arm64))

package similarity

import (
	"context"
	"os/exec"
)

func workerCommand(ctx context.Context, executable string) *exec.Cmd {
	return exec.CommandContext(ctx, "/usr/bin/sandbox-exec", "-p", "(version 1) (allow default) (deny network*)", executable)
}

func isolateWorker() error { return nil }
