//go:build windows

package update

import (
	"syscall"
	"time"
)

// awaitProcessExit waits on the process object instead of polling: Windows
// signals it the moment the predecessor exits, so the launch is held for
// exactly as long as it has to be. SYNCHRONIZE is the only access
// WaitForSingleObject needs, and asking for no more than that is what keeps
// the open itself from being denied.
func awaitProcessExit(pid int, timeout time.Duration) {
	handle, err := syscall.OpenProcess(syscall.SYNCHRONIZE, false, uint32(pid))
	if err != nil {
		// The predecessor has already exited, or belongs to a session this
		// process may not open. Neither is worth waiting for.
		return
	}
	defer func() { _ = syscall.CloseHandle(handle) }()
	// A wait that fails is reported by returning, same as one that times out:
	// the sweep that follows tolerates a predecessor still holding its files.
	_, _ = syscall.WaitForSingleObject(handle, waitMilliseconds(timeout))
}

// waitMilliseconds converts a timeout for WaitForSingleObject, which reads
// 0xFFFFFFFF as "wait forever": a duration long enough to convert to that
// value, or a negative one wrapped by the conversion, would silently turn the
// bounded wait into a hang.
func waitMilliseconds(timeout time.Duration) uint32 {
	ms := timeout.Milliseconds()
	if ms < 0 {
		return 0
	}
	if ms >= syscall.INFINITE {
		return syscall.INFINITE - 1
	}
	return uint32(ms)
}
