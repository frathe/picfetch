//go:build darwin || linux || freebsd || openbsd || netbsd || dragonfly

package consolehelp

import (
	"errors"
	"io"
	"os"

	"golang.org/x/sys/unix"
)

// Polling bounds signal response without leaving an unjoinable stdin worker.
func readTerminalByte(file *os.File) (byte, bool, error) {
	events := []unix.PollFd{{Fd: int32(file.Fd()), Events: unix.POLLIN}}
	count, err := unix.Poll(events, 100)
	if errors.Is(err, unix.EINTR) {
		return 0, false, nil
	}
	if err != nil || count == 0 {
		return 0, false, err
	}
	if events[0].Revents&unix.POLLNVAL != 0 {
		return 0, false, os.ErrClosed
	}
	var data [1]byte
	n, err := file.Read(data[:])
	if n == 0 && err == nil {
		err = io.EOF
	}
	return data[0], n != 0, err
}
