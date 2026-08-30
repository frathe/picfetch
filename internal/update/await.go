package update

import (
	"errors"
	"io/fs"
	"os"
	"strconv"
	"time"
)

// AwaitPIDEnv names the environment variable an update relaunch sets on the
// executable it starts, holding the PID of the process that installed it. A
// relaunched PicFetch must let that process finish exiting before it touches
// preferences: Apply runs inside Fyne's stopped callback, and Fyne saves
// preferences immediately after that callback returns.
const AwaitPIDEnv = "PICFETCH_UPDATE_AWAIT_PID"

// awaitPredecessorTimeout bounds the wait because this runs before PicFetch
// has a window: a predecessor that never exits has to cost a slow start, not
// a launch that never finishes and cannot say why.
const awaitPredecessorTimeout = 15 * time.Second

// CleanupPredecessor waits for the process that installed this executable to
// exit, then deletes the leftovers an update from before 2026-08-30 could
// have left beside it. Both halves are no-ops on a normal launch: only a
// relaunch sets AwaitPIDEnv, and only an update leaves anything to sweep.
//
// The backup the current apply leaves behind is deliberately not swept here.
// Whether it is still the user's only working executable is a question only
// the recorded apply failure can answer, and that record lives in the Fyne
// app cache - which does not exist yet, because this has to run before
// preferences are read. internal/ui calls SweepBackup once it does.
//
// A bare PID is a weak handle - Windows recycles PIDs aggressively, so an
// unrelated process can carry the predecessor's number by the time this
// reads it, and the wait would then be spent on a stranger. That is accepted
// rather than solved: the bounded wait caps the cost at a slow start, and
// the sweep is safe either way.
func CleanupPredecessor() {
	if pid, err := strconv.Atoi(os.Getenv(AwaitPIDEnv)); err == nil && pid > 0 {
		awaitProcessExit(pid, awaitPredecessorTimeout)
	}
	// Everything PicFetch spawns from here on inherits this environment, and
	// a PID that has already been waited for would only make some unrelated
	// child wait again, on whatever now answers to that number.
	_ = os.Unsetenv(AwaitPIDEnv)

	dest, err := os.Executable()
	if err != nil {
		return
	}
	sweepLeftovers(dest)
}

// sweepLeftovers removes the two files an apply left beside the executable
// back when it ran through a cmd.exe script. Errors are ignored: a file the
// exiting predecessor still holds open is swept by the launch after this one.
func sweepLeftovers(dest string) {
	if dest == "" {
		// The suffixes below would otherwise resolve against the working
		// directory and delete files no update ever wrote.
		return
	}
	_ = os.Remove(dest + ".new")
	_ = os.Remove(dest + ".apply.cmd")
}

// sweepAttempts and sweepRetryDelay bound the second chance the sweep gets.
// awaitProcessExit returns when the predecessor's process object is
// signalled, but Windows only releases the executable image once the last
// handle to that process is closed, and a virus scanner examining the file
// the update just wrote holds one of its own. Both are over in well under a
// second, and a single silent attempt at exactly the wrong moment is how the
// backup ends up living on the user's disk forever - nothing else ever
// deletes it, because the next launch finds no update to clean up after.
//
// Retrying costs nothing on a normal launch: there is no backup to remove,
// os.Remove says so on the first attempt, and a missing file ends the loop
// rather than sleeping through it. The total is kept short because this runs
// on the Fyne main goroutine from the launch's OnStarted, where the window
// is already on screen: the launch after an update may pause, but only for
// as long as a released image handle takes to become deletable.
const (
	sweepAttempts   = 4
	sweepRetryDelay = 100 * time.Millisecond
)

// SweepBackup removes the executable swapBinary renamed aside before
// installing the new one. On Windows that file is the running image until
// the predecessor exits, which is why it survives the apply itself; by the
// time a successor gets here CleanupPredecessor has already waited for that
// exit.
//
// dest missing means nothing landed where the new executable should be, so
// the backup is the only PicFetch left and stays. That guard alone is not
// enough - a copy that failed partway through leaves a truncated dest that
// stats perfectly well - which is why callers consult the recorded apply
// failure first and skip this entirely after a failed restore.
//
// A backup that survives every attempt is left alone rather than reported:
// the one thing that still refuses the delete after this is Controlled
// Folder Access denying it, and the dialog that sent the user to fix that
// has already been shown by the launch that could not install the update.
func SweepBackup(dest string) {
	if dest == "" {
		// ".old" would otherwise resolve against the working directory and
		// delete a file no update ever wrote.
		return
	}
	if _, err := os.Stat(dest); err != nil {
		return
	}
	old := dest + ".old"
	for attempt := range sweepAttempts {
		if attempt > 0 {
			time.Sleep(sweepRetryDelay)
		}
		if err := os.Remove(old); err == nil || errors.Is(err, fs.ErrNotExist) {
			return
		}
	}
}
