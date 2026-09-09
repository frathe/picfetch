//go:build darwin && arm64

package similarity

import (
	"os"
	"syscall"
)

// A blocking os.Stdin cannot be closed while its reader is in a system call.
// Give the control reader a pollable descriptor so Close releases it on worker
// completion or failure, even while the parent keeps its input pipe open.
func controlInput() (*os.File, error) {
	fd, err := syscall.Dup(int(os.Stdin.Fd()))
	if err != nil {
		return nil, err
	}
	if err := syscall.SetNonblock(fd, true); err != nil {
		_ = syscall.Close(fd)
		return nil, err
	}
	return os.NewFile(uintptr(fd), "similarity controls"), nil
}
