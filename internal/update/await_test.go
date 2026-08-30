package update

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestCleanupPredecessor_NoEnvReturnsImmediately(t *testing.T) {
	t.Setenv(AwaitPIDEnv, "")

	done := make(chan struct{})
	go func() {
		CleanupPredecessor()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("CleanupPredecessor blocked with no pid to await")
	}
}

func TestCleanupPredecessor_IgnoresAPIDItCannotUse(t *testing.T) {
	// Anything that is not a positive decimal integer reached the successor
	// by accident, not from a relaunch, so none of these may cost a wait.
	for _, value := range []string{"0", "-1", "seventeen", " 42", "42 "} {
		t.Run(value, func(t *testing.T) {
			t.Setenv(AwaitPIDEnv, value)

			done := make(chan struct{})
			go func() {
				CleanupPredecessor()
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatalf("CleanupPredecessor waited on %q", value)
			}
			if got, ok := os.LookupEnv(AwaitPIDEnv); ok {
				t.Errorf("%s survived as %q; every process PicFetch spawns would inherit it", AwaitPIDEnv, got)
			}
		})
	}
}

func TestSweepBackup_RemovesTheBackupBesideAnInstalledExecutable(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "picfetch")
	for _, suffix := range []string{"", ".old"} {
		if err := os.WriteFile(dest+suffix, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	SweepBackup(dest)

	if _, err := os.Stat(dest + ".old"); !os.IsNotExist(err) {
		t.Errorf(".old survived the sweep: %v", err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("sweep deleted the executable itself: %v", err)
	}
}

func TestSweepBackup_KeepsTheBackupWhenTheExecutableIsMissing(t *testing.T) {
	// A missing dest beside an intact backup is what a failed restore leaves
	// behind: the backup is the user's only working executable, so deleting
	// it would end the update in no PicFetch at all. This is the cheap half
	// of the guard - the failure record internal/ui consults before calling
	// this is what catches a dest that exists but is unusable.
	dir := t.TempDir()
	dest := filepath.Join(dir, "picfetch")
	if err := os.WriteFile(dest+".old", []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	SweepBackup(dest)

	if _, err := os.Stat(dest + ".old"); err != nil {
		t.Errorf("the only intact executable was swept: %v", err)
	}
}

func TestSweepBackup_LeavesTheScriptEraLeftoversToCleanupPredecessor(t *testing.T) {
	// The split is the point: this half needs a fyne.App to consult the
	// failure record, the other half must run before one exists.
	dir := t.TempDir()
	dest := filepath.Join(dir, "picfetch")
	for _, suffix := range []string{"", ".new", ".apply.cmd"} {
		if err := os.WriteFile(dest+suffix, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	SweepBackup(dest)

	for _, suffix := range []string{".new", ".apply.cmd"} {
		if _, err := os.Stat(dest + suffix); err != nil {
			t.Errorf("%s was swept by the backup half: %v", suffix, err)
		}
	}
}

func TestSweepBackup_ToleratesNothingToSweep(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "picfetch")
	if err := os.WriteFile(dest, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	SweepBackup(dest)

	if _, err := os.Stat(dest); err != nil {
		t.Errorf("sweep deleted the executable itself: %v", err)
	}
}

// TestSweepBackup_RetriesABackupItCannotRemoveYet pins the two halves of the
// retry budget: a launch with nothing to sweep must not pay for it, and a
// backup that never becomes deletable must not hold the launch open.
func TestSweepBackup_RetriesABackupItCannotRemoveYet(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "picfetch")
	if err := os.WriteFile(dest, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("nothing to sweep costs no delay", func(t *testing.T) {
		start := time.Now()
		SweepBackup(dest)
		if elapsed := time.Since(start); elapsed >= sweepRetryDelay {
			t.Errorf("sweep with no backup took %v, want it to return on the first attempt", elapsed)
		}
	})

	t.Run("an undeletable backup is bounded", func(t *testing.T) {
		// A directory with a file in it is the portable stand-in for the
		// backup Windows still has open: os.Remove refuses it every time.
		old := dest + ".old"
		if err := os.Mkdir(old, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(old, "held"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}

		start := time.Now()
		SweepBackup(dest)
		elapsed := time.Since(start)

		if want := time.Duration(sweepAttempts) * sweepRetryDelay; elapsed > want {
			t.Errorf("sweep of an undeletable backup took %v, want at most %v", elapsed, want)
		}
		if elapsed < sweepRetryDelay {
			t.Errorf("sweep of an undeletable backup took %v, want it to have retried", elapsed)
		}
		if _, err := os.Stat(dest); err != nil {
			t.Errorf("sweep deleted the executable itself: %v", err)
		}
	})
}

func TestSweepBackup_EmptyDestSweepsNothing(t *testing.T) {
	// An empty dest would resolve ".old" against the working directory and
	// delete a file this process never installed.
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.WriteFile(filepath.Join(dir, ".old"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	SweepBackup("")

	if _, err := os.Stat(filepath.Join(dir, ".old")); err != nil {
		t.Errorf(".old was swept out of the working directory: %v", err)
	}
}

func TestSweepLeftovers_RemovesTheScriptEraFilesAndKeepsTheBackup(t *testing.T) {
	// The backup is not this half's to remove even with dest in place: only
	// the failure record can say whether dest is a working executable, and
	// this runs before there is a fyne.App to read it from.
	dir := t.TempDir()
	dest := filepath.Join(dir, "picfetch")
	for _, suffix := range []string{"", ".old", ".new", ".apply.cmd"} {
		if err := os.WriteFile(dest+suffix, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	sweepLeftovers(dest)

	for _, suffix := range []string{".new", ".apply.cmd"} {
		if _, err := os.Stat(dest + suffix); !os.IsNotExist(err) {
			t.Errorf("%s survived the sweep: %v", suffix, err)
		}
	}
	if _, err := os.Stat(dest + ".old"); err != nil {
		t.Errorf("the backup was swept before anything could check the failure record: %v", err)
	}
	if _, err := os.Stat(dest); err != nil {
		t.Errorf("sweep deleted the executable itself: %v", err)
	}
}

func TestSweepLeftovers_ToleratesNothingToSweep(t *testing.T) {
	dir := t.TempDir()
	dest := filepath.Join(dir, "picfetch")
	if err := os.WriteFile(dest, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}

	sweepLeftovers(dest)

	if _, err := os.Stat(dest); err != nil {
		t.Errorf("sweep deleted the executable itself: %v", err)
	}
}

func TestSweepLeftovers_EmptyDestSweepsNothing(t *testing.T) {
	// An empty dest would resolve the two suffixes against the working
	// directory and delete files this process never installed.
	dir := t.TempDir()
	t.Chdir(dir)
	for _, name := range []string{".new", ".apply.cmd"} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	sweepLeftovers("")

	for _, name := range []string{".new", ".apply.cmd"} {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			t.Errorf("%s was swept out of the working directory: %v", name, err)
		}
	}
}

func TestAwaitProcessExit_DeadPIDReturnsBeforeTheTimeout(t *testing.T) {
	// Re-running the test binary with a filter that matches no test is the
	// portable way to get a pid that has certainly exited.
	cmd := exec.Command(os.Args[0], "-test.run=^$")
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	pid := cmd.Process.Pid
	if err := cmd.Wait(); err != nil {
		t.Fatal(err)
	}

	start := time.Now()
	awaitProcessExit(pid, 30*time.Second)
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("waited %v for a process that had already exited", elapsed)
	}
}

func TestAwaitProcessExit_LivePIDReturnsAtTheTimeout(t *testing.T) {
	// A predecessor that never exits must cost a slow launch, never a launch
	// that never finishes - this process is the one pid guaranteed to still
	// be running.
	const timeout = 300 * time.Millisecond

	start := time.Now()
	done := make(chan struct{})
	go func() {
		awaitProcessExit(os.Getpid(), timeout)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		t.Fatal("awaitProcessExit never returned for a process that is still running")
	}

	elapsed := time.Since(start)
	if elapsed < timeout/2 {
		t.Errorf("returned after %v, want at least %v: a running process was not waited for", elapsed, timeout/2)
	}
	if elapsed > 5*time.Second {
		t.Errorf("returned after %v, want a wait bounded by the %v timeout", elapsed, timeout)
	}
}

func TestAwaitProcessExit_ZeroTimeoutDoesNotWait(t *testing.T) {
	start := time.Now()
	done := make(chan struct{})
	go func() {
		awaitProcessExit(os.Getpid(), 0)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(10 * time.Second):
		// Guarded like the two tests above: an unbounded-wait regression has
		// to fail this test, not hang the whole package until go test's own
		// timeout kills it.
		t.Fatal("awaitProcessExit never returned for a zero timeout")
	}

	if elapsed := time.Since(start); elapsed > time.Second {
		t.Errorf("a zero timeout waited %v on a running process", elapsed)
	}
}
