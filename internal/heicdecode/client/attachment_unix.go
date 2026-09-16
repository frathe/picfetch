//go:build darwin || linux

package client

import (
	"os"
	"os/exec"

	"golang.org/x/sys/unix"
)

func inheritBrokerPipes(cmd *exec.Cmd, input, output *os.File) (uint64, uint64, []*os.File, error) {
	readID := uint64(3 + len(cmd.ExtraFiles))
	cmd.ExtraFiles = append(cmd.ExtraFiles, input, output)
	return readID, readID + 1, nil, nil
}

func openBrokerPipe(id uint64, read bool) (*os.File, error) {
	if id > uint64(^uint(0)>>1) {
		return nil, ErrUnavailable
	}
	fd := int(id)
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return nil, err
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFIFO {
		return nil, ErrUnavailable
	}
	flags, err := unix.FcntlInt(uintptr(fd), unix.F_GETFL, 0)
	if err != nil {
		return nil, err
	}
	want := unix.O_WRONLY
	if read {
		want = unix.O_RDONLY
	}
	if flags&unix.O_ACCMODE != want {
		return nil, ErrUnavailable
	}
	if err = unix.SetNonblock(fd, true); err != nil {
		return nil, err
	}
	unix.CloseOnExec(fd)
	return os.NewFile(uintptr(fd), "HEIC broker pipe"), nil
}
