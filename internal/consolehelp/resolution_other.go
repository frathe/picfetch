//go:build !darwin && !linux && !freebsd && !openbsd && !netbsd && !dragonfly && !windows

package consolehelp

import (
	"errors"
	"os"
)

func readTerminalByte(_ *os.File) (byte, bool, error) {
	return 0, false, errors.New("interactive resolution selection is unsupported on this platform")
}
