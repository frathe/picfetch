//go:build windows

package consolehelp

import (
	"fmt"
	"io"
	"os"

	"golang.org/x/sys/windows"
)

func readTerminalByte(file *os.File) (byte, bool, error) {
	status, err := windows.WaitForSingleObject(windows.Handle(file.Fd()), 100)
	if err != nil {
		return 0, false, err
	}
	if status == uint32(windows.WAIT_TIMEOUT) {
		return 0, false, nil
	}
	if status != windows.WAIT_OBJECT_0 {
		return 0, false, fmt.Errorf("terminal wait status %d", status)
	}
	var data [1]byte
	n, err := file.Read(data[:])
	if n == 0 && err == nil {
		err = io.EOF
	}
	return data[0], n != 0, err
}
