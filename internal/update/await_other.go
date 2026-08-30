//go:build !windows

package update

import (
	"syscall"
	"time"
)

// awaitPollInterval trades a little launch latency for not spinning: the
// predecessor is exiting rather than working, so it is gone within a poll or
// two.
const awaitPollInterval = 100 * time.Millisecond

// awaitProcessExit polls because the predecessor is this process's parent,
// not its child, and Unix offers no wait for a process it did not fork.
// Signal 0 runs the kernel's existence and permission checks without
// delivering anything. pid must be positive: 0 and negative values address a
// process group, which is never what a predecessor is.
//
// Unix relaunches never set AwaitPIDEnv - launchUnix hands the same wait to a
// shell helper that outlives the process being replaced (apply_unix.go) - so
// this exists to keep the sweep half building everywhere and to keep the wait
// itself testable off Windows.
func awaitProcessExit(pid int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for {
		if err := syscall.Kill(pid, 0); err != nil {
			return
		}
		if !time.Now().Before(deadline) {
			return
		}
		time.Sleep(awaitPollInterval)
	}
}
