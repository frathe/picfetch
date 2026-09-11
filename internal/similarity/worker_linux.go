//go:build linux && (amd64 || arm64)

package similarity

import (
	"context"
	"fmt"
	"os/exec"
	"runtime"
	"unsafe"

	"golang.org/x/sys/unix"
)

func workerCommand(ctx context.Context, executable string) *exec.Cmd {
	return exec.CommandContext(ctx, executable)
}

// Install denial before reading requests or loading native code. TSYNC applies
// it to every existing thread; future threads and child processes inherit it.
// No namespace helper, root privileges, or system configuration is required.
func isolateWorker() error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := unix.Prctl(unix.PR_SET_NO_NEW_PRIVS, 1, 0, 0, 0); err != nil {
		return fmt.Errorf("offline worker no_new_privs: %w", err)
	}
	architecture := uint32(unix.AUDIT_ARCH_X86_64)
	if runtime.GOARCH == "arm64" {
		architecture = unix.AUDIT_ARCH_AARCH64
	}
	// The worker inherits only standard pipes, never network descriptors.
	// Denying io_uring also closes its alternative socket creation path.
	raw, err := linuxNetworkFilter(architecture, unix.SYS_SOCKET, unix.SYS_SOCKETPAIR, unix.SYS_IO_URING_SETUP)
	if err != nil {
		return fmt.Errorf("offline worker filter: %w", err)
	}
	filter := make([]unix.SockFilter, len(raw))
	for i, instruction := range raw {
		filter[i] = unix.SockFilter{Code: instruction.Op, Jt: instruction.Jt, Jf: instruction.Jf, K: instruction.K}
	}
	program := unix.SockFprog{Len: uint16(len(filter)), Filter: &filter[0]}
	result, _, errno := unix.Syscall(unix.SYS_SECCOMP, unix.SECCOMP_SET_MODE_FILTER, unix.SECCOMP_FILTER_FLAG_TSYNC, uintptr(unsafe.Pointer(&program)))
	runtime.KeepAlive(filter)
	if errno != 0 {
		return fmt.Errorf("offline worker seccomp: %w", errno)
	}
	if result != 0 {
		return fmt.Errorf("offline worker seccomp could not synchronize thread %d", result)
	}
	return nil
}
