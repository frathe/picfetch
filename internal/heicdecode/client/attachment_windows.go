package client

import (
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

func inheritBrokerPipes(cmd *exec.Cmd, input, output *os.File) (uint64, uint64, []*os.File, error) {
	if cmd.SysProcAttr == nil {
		cmd.SysProcAttr = &syscall.SysProcAttr{}
	}
	if cmd.SysProcAttr.NoInheritHandles || cmd.SysProcAttr.ParentProcess != 0 {
		return 0, 0, nil, ErrUnavailable
	}
	var copies []*os.File
	for _, file := range []*os.File{input, output} {
		var handle windows.Handle
		if err := windows.DuplicateHandle(windows.CurrentProcess(), windows.Handle(file.Fd()), windows.CurrentProcess(), &handle, 0, true, windows.DUPLICATE_SAME_ACCESS); err != nil {
			for _, copy := range copies {
				_ = copy.Close()
			}
			return 0, 0, nil, err
		}
		// Wrap before any I/O starts: NewFile's synchronous-mode query must
		// not race an outstanding operation on this handle.
		copies = append(copies, os.NewFile(uintptr(handle), "HEIC inherited pipe"))
	}
	readID, writeID := copies[0].Fd(), copies[1].Fd()
	cmd.SysProcAttr.AdditionalInheritedHandles = append(cmd.SysProcAttr.AdditionalInheritedHandles, syscall.Handle(readID), syscall.Handle(writeID))
	return uint64(readID), uint64(writeID), copies, nil
}

func openBrokerPipe(id uint64, _ bool) (*os.File, error) {
	if id > uint64(^uintptr(0)) {
		return nil, ErrUnavailable
	}
	handle := windows.Handle(id)
	typeID, err := windows.GetFileType(handle)
	if err != nil {
		return nil, err
	}
	if typeID != windows.FILE_TYPE_PIPE {
		return nil, ErrUnavailable
	}
	if err = windows.SetHandleInformation(handle, windows.HANDLE_FLAG_INHERIT, 0); err != nil {
		return nil, err
	}
	return os.NewFile(uintptr(handle), "HEIC broker pipe"), nil
}
