//go:build !windows

package update

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"
)

// A recent sibling may belong to another PicFetch process that is still
// applying an update. A later launch can sweep a file left by a crash.
const temporarySiblingGrace = 5 * time.Minute

func sweepTemporarySiblings(dest string) {
	if resolved, err := filepath.EvalSymlinks(dest); err == nil {
		dest = resolved
	}
	sweepSiblingDirectory(filepath.Dir(dest), temporarySiblingPrefix(dest))

	macOSDir := filepath.Dir(dest)
	contentsDir := filepath.Dir(macOSDir)
	appDir := filepath.Dir(contentsDir)
	if filepath.Base(macOSDir) != "MacOS" || filepath.Base(contentsDir) != "Contents" || !strings.HasSuffix(filepath.Base(appDir), ".app") {
		return
	}
	plistDest := filepath.Join(contentsDir, "Info.plist")
	sweepSiblingDirectory(contentsDir, temporarySiblingPrefix(plistDest))
}

func sweepSiblingDirectory(dir, prefix string) {
	f, err := os.Open(dir)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()
	cutoff := time.Now().Add(-temporarySiblingGrace)
	for {
		names, readErr := f.Readdirnames(64)
		for _, name := range names {
			suffix, matched := strings.CutPrefix(name, prefix)
			if !matched || !temporarySuffix(suffix) {
				continue
			}
			path := filepath.Join(dir, name)
			info, statErr := os.Lstat(path)
			if statErr != nil || !info.Mode().IsRegular() || info.ModTime().After(cutoff) {
				continue
			}
			_ = os.Remove(path)
		}
		if readErr != nil {
			return
		}
	}
}

func temporarySuffix(suffix string) bool {
	// os.CreateTemp appends a decimal uint32 without leading zeroes.
	n, err := strconv.ParseUint(suffix, 10, 32)
	return err == nil && strconv.FormatUint(n, 10) == suffix
}

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
