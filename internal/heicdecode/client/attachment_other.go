//go:build !darwin && !linux && !windows

package client

import (
	"os"
	"os/exec"
)

func inheritBrokerPipes(_ *exec.Cmd, _, _ *os.File) (uint64, uint64, []*os.File, error) {
	return 0, 0, nil, ErrUnavailable
}
func openBrokerPipe(_ uint64, _ bool) (*os.File, error) { return nil, ErrUnavailable }
