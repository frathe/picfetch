package update

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// openVerifiedStageBinary returns one handle whose identity is retained from
// verification through installation. Reopening Stage.BinaryPath after hashing
// would let another same-user process swap the cache entry between those two
// operations.
func openVerifiedStageBinary(stage Stage) (*os.File, error) {
	if err := ValidateStage(stage); err != nil {
		return nil, err
	}
	f, err := os.Open(stage.BinaryPath)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		_ = f.Close()
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("%q is not a regular file", stage.BinaryPath)
	}
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		_ = f.Close()
		return nil, err
	}
	if !strings.EqualFold(hex.EncodeToString(h.Sum(nil)), stage.verification.BinaryDigest) {
		_ = f.Close()
		return nil, errors.New("update: staged binary verification: SHA-256 mismatch")
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		_ = f.Close()
		return nil, err
	}
	return f, nil
}

// swapBinaryFrom installs bytes from an already authenticated open handle.
// It is the Windows trust-boundary path; unlike swapBinary it never reopens a
// cache-controlled source pathname.
func swapBinaryFrom(staged *os.File, expectedDigest, dest string, options ApplyOptions, ops binaryOps) error {
	old := dest + ".old"
	_ = ops.Remove(old)
	if err := ops.Rename(dest, old); err != nil {
		return &ApplyError{Op: "rename", Path: dest, Err: err}
	}
	if err := copyReader(staged, dest); err != nil {
		_ = ops.Remove(dest)
		return restoreBinary(ops, dest, old, "copy", err)
	}
	got, err := fileSHA256(dest)
	if err == nil && !strings.EqualFold(got, expectedDigest) {
		err = errVerifyMismatch
	}
	if err != nil {
		_ = ops.Remove(dest)
		return restoreBinary(ops, dest, old, "verify", err)
	}
	if options.Relaunch {
		if err := ops.Relaunch(dest); err != nil {
			return &ApplyError{Op: "relaunch", Path: dest, Err: err}
		}
	}
	return nil
}

// errVerifyMismatch reports that the bytes that landed at the destination are
// not the bytes that were staged. A filter driver can accept a write and
// still discard or alter it, so a successful copy is not proof of an
// installed update.
var errVerifyMismatch = errors.New("installed binary does not match the staged update")

// binaryOps are the file operations swapBinary performs, injected so the
// rollback ordering is testable without a real executable to overwrite.
type binaryOps struct {
	Rename   func(oldPath, newPath string) error
	Copy     func(src, dst string) error
	Remove   func(path string) error
	Same     func(a, b string) (bool, error)
	Relaunch func(dest string) error
}

// defaultBinaryOps wires the real filesystem into binaryOps. relaunch is
// platform-specific because starting the installed executable has to hand the
// successor whatever it needs to wait for this process.
func defaultBinaryOps(relaunch func(string) error) binaryOps {
	return binaryOps{
		Rename:   os.Rename,
		Copy:     copyFile,
		Remove:   os.Remove,
		Same:     sameContents,
		Relaunch: relaunch,
	}
}

// swapBinary installs stagedPath over dest by renaming dest aside first,
// which is the one form of replacement Windows allows on a running image.
// Every failure past that rename tries to restore the backup, so a denial by
// Controlled Folder Access or a virus scanner normally leaves the user's
// installed executable exactly as it was. Normally, not always: see
// restoreBinary for what is left if the rollback itself is refused.
//
// The backup at dest+".old" deliberately survives a successful swap: it is
// still this process's own running image and cannot be deleted from here.
// The next launch sweeps it.
func swapBinary(stagedPath, dest string, options ApplyOptions, ops binaryOps) error {
	old := dest + ".old"
	// A leftover backup from an interrupted earlier attempt would make the
	// rename below fail on platforms that refuse to clobber, so drop it
	// first; if it is still locked, the rename reports the real problem.
	_ = ops.Remove(old)

	if err := ops.Rename(dest, old); err != nil {
		return &ApplyError{Op: "rename", Path: dest, Err: err}
	}
	if err := ops.Copy(stagedPath, dest); err != nil {
		// A failed copy can still leave dest behind as a truncated file;
		// removing it first gives the restoring rename a clear destination.
		_ = ops.Remove(dest)
		return restoreBinary(ops, dest, old, "copy", err)
	}

	same, err := ops.Same(stagedPath, dest)
	if err == nil && !same {
		err = errVerifyMismatch
	}
	if err != nil {
		// The bytes at dest are unusable, and removing them first gives the
		// restoring rename a clear destination.
		_ = ops.Remove(dest)
		return restoreBinary(ops, dest, old, "verify", err)
	}

	if options.Relaunch {
		if err := ops.Relaunch(dest); err != nil {
			// The update itself succeeded, so this is reported without a
			// rollback: the user only has to start PicFetch again.
			return &ApplyError{Op: "relaunch", Path: dest, Err: err}
		}
	}
	return nil
}

// restoreAttempts and restoreRetryDelay bound the second chance the rollback
// gets. What refuses a rename here is a scanner still holding a handle on the
// file it has just inspected, or Controlled Folder Access arbitrating a
// write, and both are usually over in well under a second. The whole swap
// runs from Fyne's stopped callback, so nobody is waiting on this.
const (
	restoreAttempts   = 3
	restoreRetryDelay = 100 * time.Millisecond
)

// restoreBinary puts the backup back after step op failed with cause.
//
// This is the one failure the swap cannot report its way out of: if the
// backup does not go back, dest is truncated or missing, the user has no
// PicFetch to start, and the Op "restore" record that exists to protect the
// backup from the next launch's sweep is never read, because reading it
// needs the app that will not start. So a single refused rename is not taken
// as an answer - it is retried, and then the backup is copied back instead,
// which is a different syscall meeting a different obstacle: a lock that
// stops old from being moved need not stop it from being read.
//
// A rollback that lands either way reports the step that actually failed, so
// the next launch tells the user their old PicFetch still works and the
// sweep is free to drop the backup. Only when nothing worked does this
// report under its own op, with every error joined rather than one hidden
// behind the other.
func restoreBinary(ops binaryOps, dest, old, op string, cause error) error {
	var err error
	for attempt := range restoreAttempts {
		if attempt > 0 {
			time.Sleep(restoreRetryDelay)
		}
		if err = ops.Rename(old, dest); err == nil {
			return &ApplyError{Op: op, Path: dest, Err: cause}
		}
	}
	if copyErr := ops.Copy(old, dest); copyErr != nil {
		return &ApplyError{Op: "restore", Path: dest, Err: errors.Join(cause, err, copyErr)}
	}
	return &ApplyError{Op: op, Path: dest, Err: cause}
}

// sameContents compares two files by SHA-256 rather than by size or
// modification time, because the writes this guards are the ones an
// antivirus filter is most likely to have quietly rewritten.
func sameContents(a, b string) (bool, error) {
	sumA, err := fileSHA256(a)
	if err != nil {
		return false, err
	}
	sumB, err := fileSHA256(b)
	if err != nil {
		return false, err
	}
	return sumA == sumB, nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
	return copyReader(in, dst)
}

func copyReader(in io.Reader, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(out, in)
	closeErr := out.Close()
	if copyErr != nil {
		return copyErr
	}
	return closeErr
}
