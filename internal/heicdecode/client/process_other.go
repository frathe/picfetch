//go:build !darwin && !linux && (!windows || (!amd64 && !arm64))

package client

import (
	"os/exec"

	"github.com/frathe/picfetch/internal/heicdecode"
)

func startProcess(_ *exec.Cmd, _ heicdecode.Limits) (helperProcess, error) {
	return nil, ErrUnavailable
}
