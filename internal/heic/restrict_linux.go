//go:build linux && (amd64 || arm64)

package heic

import (
	"fmt"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

func restrictWorker() error {
	for _, limit := range []struct {
		resource int
		value    uint64
	}{
		{unix.RLIMIT_AS, 4 * 1024 * 1024 * 1024},
		{unix.RLIMIT_CPU, 40},
		{unix.RLIMIT_CORE, 0},
		{unix.RLIMIT_FSIZE, 0},
	} {
		if err := unix.Setrlimit(limit.resource, &unix.Rlimit{Cur: limit.value, Max: limit.value}); err != nil {
			return err
		}
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return err
	}
	architecture := uint32(unix.AUDIT_ARCH_X86_64)
	if runtime.GOARCH == "arm64" {
		architecture = unix.AUDIT_ARCH_AARCH64
	}
	const deny = 0x00050001
	filter := []unix.SockFilter{
		{Code: 0x20, K: 4}, {Code: 0x15, Jt: 1, K: architecture}, {Code: 6, K: deny},
		{Code: 0x20, K: 0}, {Code: 0x35, Jf: 1, K: 0x40000000}, {Code: 6, K: deny},
	}
	for _, call := range []uint32{unix.SYS_SOCKET, unix.SYS_SOCKETPAIR, unix.SYS_IO_URING_SETUP} {
		filter = append(filter, unix.SockFilter{Code: 0x15, Jf: 1, K: call}, unix.SockFilter{Code: 6, K: deny})
	}
	filter = append(filter, unix.SockFilter{Code: 6, K: 0x7fff0000})
	program := unix.SockFprog{Len: uint16(len(filter)), Filter: &filter[0]}
	result, _, errno := unix.Syscall(unix.SYS_SECCOMP, unix.SECCOMP_SET_MODE_FILTER, unix.SECCOMP_FILTER_FLAG_TSYNC, uintptr(unsafe.Pointer(&program)))
	runtime.KeepAlive(filter)
	if errno != 0 {
		return errno
	}
	if result != 0 {
		return fmt.Errorf("HEIC worker seccomp could not synchronize thread %d", result)
	}
	return nil
}
