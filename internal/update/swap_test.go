package update

import (
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"
)

// The swap fakes never touch the filesystem, so these paths only have to be
// distinguishable from one another, not valid on the host OS.
const (
	fakeStaged = "/cache/picfetch-staged"
	fakeDest   = "/app/picfetch"
	fakeOld    = fakeDest + ".old"
)

func swapLabel(path string) string {
	switch path {
	case fakeStaged:
		return "staged"
	case fakeDest:
		return "dest"
	case fakeOld:
		return "old"
	default:
		return path
	}
}

// fakeOps records every file operation swapBinary performs, in order, so the
// tests can assert on the rollback sequence rather than only on the returned
// error: a swap that never restored the backup would still return the same
// error.
type fakeOps struct {
	calls     []string
	renameErr map[string]error // keyed by the recorded "src→dst" label pair
	// renameFailures caps how many attempts a renameErr entry actually
	// fails, so a test can drive restoreBinary's retry: an absent entry
	// means every attempt fails.
	renameFailures map[string]int
	renameCalls    map[string]int
	removeErr      map[string]error // keyed by the removed path's label
	copyErr        map[string]error // keyed the same way as renameErr
	same           bool
	sameErr        error
	relaunched     bool
	relaunchErr    error
}

func newFakeOps() *fakeOps {
	return &fakeOps{
		renameErr:      map[string]error{},
		renameFailures: map[string]int{},
		renameCalls:    map[string]int{},
		removeErr:      map[string]error{},
		copyErr:        map[string]error{},
		same:           true,
	}
}

func (f *fakeOps) ops() binaryOps {
	return binaryOps{
		Rename: func(oldPath, newPath string) error {
			pair := swapLabel(oldPath) + "→" + swapLabel(newPath)
			f.calls = append(f.calls, "rename "+pair)
			attempt := f.renameCalls[pair]
			f.renameCalls[pair]++
			if limit, capped := f.renameFailures[pair]; capped && attempt >= limit {
				return nil
			}
			return f.renameErr[pair]
		},
		Copy: func(src, dst string) error {
			pair := swapLabel(src) + "→" + swapLabel(dst)
			f.calls = append(f.calls, "copy "+pair)
			return f.copyErr[pair]
		},
		Remove: func(path string) error {
			f.calls = append(f.calls, "remove "+swapLabel(path))
			return f.removeErr[swapLabel(path)]
		},
		Same: func(a, b string) (bool, error) {
			f.calls = append(f.calls, "same "+swapLabel(a)+" "+swapLabel(b))
			return f.same, f.sameErr
		},
		Relaunch: func(dest string) error {
			f.calls = append(f.calls, "relaunch "+swapLabel(dest))
			f.relaunched = true
			return f.relaunchErr
		},
	}
}

func (f *fakeOps) assertCalls(t *testing.T, want ...string) {
	t.Helper()
	if !slices.Equal(f.calls, want) {
		t.Errorf("call order =\n\t%q\nwant\n\t%q", f.calls, want)
	}
}

func assertApplyError(t *testing.T, err error, wantOp string) {
	t.Helper()
	if err == nil {
		t.Fatalf("swapBinary returned nil, want an *ApplyError with Op %q", wantOp)
	}
	var applyErr *ApplyError
	if !errors.As(err, &applyErr) {
		t.Fatalf("swapBinary error = %T (%v), want *ApplyError", err, err)
	}
	if applyErr.Op != wantOp {
		t.Errorf("Op = %q, want %q", applyErr.Op, wantOp)
	}
	if applyErr.Path != fakeDest {
		t.Errorf("Path = %q, want %q", applyErr.Path, fakeDest)
	}
}

func TestSwapBinary_HappyPath(t *testing.T) {
	ops := newFakeOps()
	if err := swapBinary(fakeStaged, fakeDest, ApplyOptions{}, ops.ops()); err != nil {
		t.Fatalf("swapBinary = %v, want nil", err)
	}
	// The single "remove old" is the stale-leftover sweep before the rename;
	// no second one may follow, because on Windows the backup is the running
	// image and the next launch is what deletes it.
	ops.assertCalls(t,
		"remove old",
		"rename dest→old",
		"copy staged→dest",
		"same staged dest",
	)
	if ops.relaunched {
		t.Error("relaunched without ApplyOptions.Relaunch")
	}
}

func TestSwapBinary_StaleBackupRemoveFailureIsIgnored(t *testing.T) {
	ops := newFakeOps()
	ops.removeErr["old"] = errors.New("still locked")
	if err := swapBinary(fakeStaged, fakeDest, ApplyOptions{}, ops.ops()); err != nil {
		t.Fatalf("swapBinary = %v, want nil", err)
	}
	ops.assertCalls(t,
		"remove old",
		"rename dest→old",
		"copy staged→dest",
		"same staged dest",
	)
}

func TestSwapBinary_RenameFailureIsNotRolledBack(t *testing.T) {
	ops := newFakeOps()
	cause := errors.New("access denied")
	ops.renameErr["dest→old"] = cause
	err := swapBinary(fakeStaged, fakeDest, ApplyOptions{Relaunch: true}, ops.ops())
	assertApplyError(t, err, "rename")
	if !errors.Is(err, cause) {
		t.Errorf("error does not unwrap to the rename cause: %v", err)
	}
	ops.assertCalls(t, "remove old", "rename dest→old")
	if slices.Contains(ops.calls, "copy staged→dest") {
		t.Error("copied after the backup rename failed")
	}
	if ops.relaunched {
		t.Error("relaunched after the backup rename failed")
	}
}

func TestSwapBinary_CopyFailureRestoresOriginal(t *testing.T) {
	ops := newFakeOps()
	cause := errors.New("virus scan blocked the write")
	ops.copyErr["staged→dest"] = cause
	err := swapBinary(fakeStaged, fakeDest, ApplyOptions{Relaunch: true}, ops.ops())
	assertApplyError(t, err, "copy")
	if !errors.Is(err, cause) {
		t.Errorf("error does not unwrap to the copy cause: %v", err)
	}
	ops.assertCalls(t,
		"remove old",
		"rename dest→old",
		"copy staged→dest",
		"rename old→dest",
	)
	if ops.relaunched {
		t.Error("relaunched after the copy failed")
	}
}

// TestSwapBinary_RestoreFailureReportsBoth is the worst case: neither the
// retried rename nor the copy fallback puts the backup back, so dest is left
// unusable and every reason has to reach the caller.
func TestSwapBinary_RestoreFailureReportsBoth(t *testing.T) {
	ops := newFakeOps()
	copyCause := errors.New("copy blocked")
	restoreCause := errors.New("restore blocked")
	fallbackCause := errors.New("backup unreadable")
	ops.copyErr["staged→dest"] = copyCause
	ops.copyErr["old→dest"] = fallbackCause
	ops.renameErr["old→dest"] = restoreCause
	err := swapBinary(fakeStaged, fakeDest, ApplyOptions{}, ops.ops())
	assertApplyError(t, err, "restore")
	for _, cause := range []error{copyCause, restoreCause, fallbackCause} {
		if !errors.Is(err, cause) {
			t.Errorf("error does not unwrap to %v: %v", cause, err)
		}
	}
	ops.assertCalls(t,
		"remove old",
		"rename dest→old",
		"copy staged→dest",
		"rename old→dest",
		"rename old→dest",
		"rename old→dest",
		"copy old→dest",
	)
}

// TestSwapBinary_RestoreRetriesTheRename covers the reason the retry exists:
// what refuses the rollback on Windows is usually a handle that is about to
// be released, not a standing denial.
func TestSwapBinary_RestoreRetriesTheRename(t *testing.T) {
	ops := newFakeOps()
	copyCause := errors.New("copy blocked")
	ops.copyErr["staged→dest"] = copyCause
	ops.renameErr["old→dest"] = errors.New("still locked")
	ops.renameFailures["old→dest"] = 1

	err := swapBinary(fakeStaged, fakeDest, ApplyOptions{}, ops.ops())

	// A rollback that landed reports the step that actually failed, so the
	// next launch says the old PicFetch still works and the sweep is free to
	// drop the backup.
	assertApplyError(t, err, "copy")
	if !errors.Is(err, copyCause) {
		t.Errorf("error does not unwrap to the copy cause: %v", err)
	}
	ops.assertCalls(t,
		"remove old",
		"rename dest→old",
		"copy staged→dest",
		"rename old→dest",
		"rename old→dest",
	)
}

// TestSwapBinary_RestoreFallsBackToCopyingTheBackup covers the other half:
// a lock that stops the backup from being moved need not stop it from being
// read, and a copied-back dest is just as launchable as a renamed one.
func TestSwapBinary_RestoreFallsBackToCopyingTheBackup(t *testing.T) {
	ops := newFakeOps()
	copyCause := errors.New("copy blocked")
	ops.copyErr["staged→dest"] = copyCause
	ops.renameErr["old→dest"] = errors.New("cannot move the backup")

	err := swapBinary(fakeStaged, fakeDest, ApplyOptions{}, ops.ops())

	assertApplyError(t, err, "copy")
	if !errors.Is(err, copyCause) {
		t.Errorf("error does not unwrap to the copy cause: %v", err)
	}
	if last := ops.calls[len(ops.calls)-1]; last != "copy old→dest" {
		t.Errorf("last call = %q, want the backup copied back over dest", last)
	}
}

func TestSwapBinary_VerifyMismatchRollsBack(t *testing.T) {
	ops := newFakeOps()
	ops.same = false
	err := swapBinary(fakeStaged, fakeDest, ApplyOptions{}, ops.ops())
	assertApplyError(t, err, "verify")
	if !errors.Is(err, errVerifyMismatch) {
		t.Errorf("error does not unwrap to errVerifyMismatch: %v", err)
	}
	ops.assertCalls(t,
		"remove old",
		"rename dest→old",
		"copy staged→dest",
		"same staged dest",
		"remove dest",
		"rename old→dest",
	)
}

func TestSwapBinary_VerifyErrorRollsBack(t *testing.T) {
	ops := newFakeOps()
	cause := errors.New("hashing the installed file failed")
	ops.sameErr = cause
	err := swapBinary(fakeStaged, fakeDest, ApplyOptions{}, ops.ops())
	assertApplyError(t, err, "verify")
	if !errors.Is(err, cause) {
		t.Errorf("error does not unwrap to the verify cause: %v", err)
	}
	ops.assertCalls(t,
		"remove old",
		"rename dest→old",
		"copy staged→dest",
		"same staged dest",
		"remove dest",
		"rename old→dest",
	)
}

func TestSwapBinary_VerifyRestoreFailureReportsBoth(t *testing.T) {
	ops := newFakeOps()
	ops.same = false
	restoreCause := errors.New("restore blocked")
	fallbackCause := errors.New("backup unreadable")
	ops.renameErr["old→dest"] = restoreCause
	ops.copyErr["old→dest"] = fallbackCause
	err := swapBinary(fakeStaged, fakeDest, ApplyOptions{}, ops.ops())
	assertApplyError(t, err, "restore")
	if !errors.Is(err, restoreCause) {
		t.Errorf("error does not unwrap to the restore cause: %v", err)
	}
	if !errors.Is(err, fallbackCause) {
		t.Errorf("error does not unwrap to the fallback cause: %v", err)
	}
	if !errors.Is(err, errVerifyMismatch) {
		t.Errorf("error does not unwrap to the verify cause: %v", err)
	}
}

func TestSwapBinary_RelaunchOnlyAfterSuccessfulVerify(t *testing.T) {
	ops := newFakeOps()
	ops.same = false
	err := swapBinary(fakeStaged, fakeDest, ApplyOptions{Relaunch: true}, ops.ops())
	assertApplyError(t, err, "verify")
	if ops.relaunched {
		t.Error("relaunched a binary that failed verification")
	}
	if slices.Contains(ops.calls, "relaunch dest") {
		t.Errorf("call order must not contain a relaunch: %q", ops.calls)
	}
}

func TestSwapBinary_RelaunchRunsAfterVerify(t *testing.T) {
	ops := newFakeOps()
	if err := swapBinary(fakeStaged, fakeDest, ApplyOptions{Relaunch: true}, ops.ops()); err != nil {
		t.Fatalf("swapBinary = %v, want nil", err)
	}
	ops.assertCalls(t,
		"remove old",
		"rename dest→old",
		"copy staged→dest",
		"same staged dest",
		"relaunch dest",
	)
}

func TestSwapBinary_RelaunchFailureIsReported(t *testing.T) {
	ops := newFakeOps()
	cause := errors.New("start failed")
	ops.relaunchErr = cause
	err := swapBinary(fakeStaged, fakeDest, ApplyOptions{Relaunch: true}, ops.ops())
	assertApplyError(t, err, "relaunch")
	if !errors.Is(err, cause) {
		t.Errorf("error does not unwrap to the relaunch cause: %v", err)
	}
	// The installed binary is good; a failed relaunch must not undo it.
	ops.assertCalls(t,
		"remove old",
		"rename dest→old",
		"copy staged→dest",
		"same staged dest",
		"relaunch dest",
	)
}

// --- the real filesystem --------------------------------------------------
//
// Everything above drives fakes, which pins the rollback ordering but not
// which real function each binaryOps field is bound to. defaultBinaryOps has
// no caller outside //go:build windows, so without the two tests below a
// Copy bound to Same, or a Remove aimed at dest instead of the backup, would
// compile, vet and pass every other test in this package.

func writeSwapFile(t *testing.T, path, content string) string {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func assertSwapFile(t *testing.T, path, want string) {
	t.Helper()
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading %s: %v", filepath.Base(path), err)
	}
	if string(got) != want {
		t.Errorf("%s holds %q, want %q", filepath.Base(path), got, want)
	}
}

// TestSwapBinary_DefaultOpsInstallTheStagedBytes runs the whole swap against
// real files, so the wiring the Windows build depends on is exercised
// somewhere that actually runs.
func TestSwapBinary_DefaultOpsInstallTheStagedBytes(t *testing.T) {
	dir := t.TempDir()
	staged := writeSwapFile(t, filepath.Join(dir, "picfetch-staged"), "new build")
	dest := writeSwapFile(t, filepath.Join(dir, "picfetch"), "running build")
	// A leftover backup from an interrupted earlier attempt, which the swap
	// clears before it renames anything aside.
	old := writeSwapFile(t, dest+".old", "stale backup")

	if err := swapBinary(staged, dest, ApplyOptions{}, defaultBinaryOps(nil)); err != nil {
		t.Fatalf("swapBinary = %v, want nil", err)
	}

	assertSwapFile(t, dest, "new build")
	assertSwapFile(t, old, "running build")
	// The stage survives on purpose: internal/ui/autoupdate removes it only
	// after Apply reports success, so a retry still has something to retry.
	assertSwapFile(t, staged, "new build")

	info, err := os.Stat(dest)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm()&0o100 == 0 {
		t.Errorf("installed binary mode = %v, want the owner-execute bit", info.Mode().Perm())
	}
}

// TestDefaultBinaryOps_BindsEveryFieldToItsOwnOperation checks the fields one
// at a time, because a successful swap cannot tell some of them apart from a
// stub: a Same that always answers true, or a Remove that does nothing, both
// still leave the staged bytes installed.
func TestDefaultBinaryOps_BindsEveryFieldToItsOwnOperation(t *testing.T) {
	dir := t.TempDir()
	ops := defaultBinaryOps(nil)

	a := writeSwapFile(t, filepath.Join(dir, "a"), "same bytes")
	b := writeSwapFile(t, filepath.Join(dir, "b"), "same bytes")
	c := writeSwapFile(t, filepath.Join(dir, "c"), "other bytes")

	if same, err := ops.Same(a, b); err != nil || !same {
		t.Errorf("Same(equal, equal) = %v, %v; want true, nil", same, err)
	}
	if same, err := ops.Same(a, c); err != nil || same {
		t.Errorf("Same(equal, other) = %v, %v; want false, nil", same, err)
	}

	if err := ops.Copy(c, b); err != nil {
		t.Fatalf("Copy = %v", err)
	}
	assertSwapFile(t, b, "other bytes")
	assertSwapFile(t, c, "other bytes") // src, not dst, is the one read

	moved := filepath.Join(dir, "moved")
	if err := ops.Rename(b, moved); err != nil {
		t.Fatalf("Rename = %v", err)
	}
	assertSwapFile(t, moved, "other bytes")
	if _, err := os.Stat(b); err == nil {
		t.Error("Rename left the source behind")
	}

	if err := ops.Remove(a); err != nil {
		t.Fatalf("Remove = %v", err)
	}
	if _, err := os.Stat(a); err == nil {
		t.Error("Remove did not delete the path it was given")
	}
}

// TestDefaultBinaryOps_RelaunchIsTheInjectedStarter pins the one field that
// is a parameter rather than a package function: the platform-specific
// starter has to arrive at swapBinary unwrapped and with dest intact.
func TestDefaultBinaryOps_RelaunchIsTheInjectedStarter(t *testing.T) {
	got := ""
	want := errors.New("could not start")

	err := defaultBinaryOps(func(dest string) error {
		got = dest
		return want
	}).Relaunch("/Applications/PicFetch.app")

	if got != "/Applications/PicFetch.app" {
		t.Errorf("Relaunch was called with %q, want the destination path", got)
	}
	if !errors.Is(err, want) {
		t.Errorf("Relaunch error = %v, want %v", err, want)
	}
}
