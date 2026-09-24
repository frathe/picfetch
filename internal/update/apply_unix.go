//go:build !windows

package update

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

func openStageBinary(path string) (*os.File, error) {
	return os.Open(path)
}

func applyUnix(stage Stage, dest string, options ApplyOptions) error {
	return applyUnixWithLauncher(stage, dest, options, launchUnix)
}

// applyUnixWithLauncher keeps the copy-before-launch ordering directly
// testable without replacing package-level state or starting PicFetch in a
// test. launch is called only after the executable and optional plist have
// both been installed successfully.
func applyUnixWithLauncher(stage Stage, dest string, options ApplyOptions, launch func(string) error) error {
	dest, err := filepath.EvalSymlinks(dest)
	if err != nil {
		return err
	}
	newBinary, err := copyToTemporarySibling(stage.BinaryPath, dest, 0o755)
	if err != nil {
		return err
	}
	old := dest + ".old"
	defer func() { _ = os.Remove(newBinary) }()

	plistDest := ""
	newPlist := ""
	if stage.PlistPath != "" {
		plistDest = filepath.Join(filepath.Dir(dest), "..", "Info.plist")
		newPlist, err = copyToTemporarySibling(stage.PlistPath, plistDest, 0o644)
		if err != nil {
			return err
		}
		defer func() { _ = os.Remove(newPlist) }()
	}

	if err := os.Rename(dest, old); err != nil {
		return err
	}
	if err := os.Rename(newBinary, dest); err != nil {
		if rbErr := os.Rename(old, dest); rbErr != nil {
			return errors.Join(err, rbErr)
		}
		return err
	}

	plistOld := ""
	plistBackedUp := false
	if plistDest != "" {
		plistOld = plistDest + ".old"
		if _, statErr := os.Stat(plistDest); statErr == nil {
			if err := os.Rename(plistDest, plistOld); err != nil {
				return rollbackUnixBinary(dest, old, err)
			}
			plistBackedUp = true
		} else if !errors.Is(statErr, os.ErrNotExist) {
			return rollbackUnixBinary(dest, old, statErr)
		}
		if err := os.Rename(newPlist, plistDest); err != nil {
			if plistBackedUp {
				if rbErr := os.Rename(plistOld, plistDest); rbErr != nil {
					err = errors.Join(err, rbErr)
				}
			}
			return rollbackUnixBinary(dest, old, err)
		}
	}

	_ = os.Remove(old)
	if plistBackedUp {
		_ = os.Remove(plistOld)
	}
	if options.Relaunch {
		if err := launch(dest); err != nil {
			// The update itself succeeded, so this is reported without a
			// rollback: the user only has to start PicFetch again.
			return &ApplyError{Op: "relaunch", Path: dest, Err: err}
		}
	}
	return nil
}

func copyToTemporarySibling(src, dest string, mode os.FileMode) (name string, err error) {
	in, err := os.Open(src)
	if err != nil {
		return "", err
	}
	defer func() { _ = in.Close() }()

	dir := filepath.Dir(dest)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	out, err := os.CreateTemp(dir, "."+filepath.Base(dest)+".new-*")
	if err != nil {
		return "", err
	}
	name = out.Name()
	tempName := name
	defer func() {
		if out != nil {
			_ = out.Close()
		}
		if err != nil {
			_ = os.Remove(tempName)
		}
	}()

	info, err := out.Stat()
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("temporary update file %q is not regular", name)
	}
	if err := out.Chmod(mode); err != nil {
		return "", err
	}
	if _, err := io.Copy(out, in); err != nil {
		return "", err
	}
	if err := out.Sync(); err != nil {
		return "", err
	}
	if err := out.Close(); err != nil {
		return "", err
	}
	out = nil
	return name, nil
}

func rollbackUnixBinary(dest, old string, cause error) error {
	if err := os.Remove(dest); err != nil && !errors.Is(err, os.ErrNotExist) {
		cause = errors.Join(cause, err)
	}
	if err := os.Rename(old, dest); err != nil {
		return errors.Join(cause, err)
	}
	return cause
}

func launchUnix(dest string) error {
	// Apply runs from Fyne's stopped callback, before Fyne finishes flushing
	// preferences. Keep the relaunch helper alive while the old process exits
	// so the new instance cannot race that final shutdown work.
	return unixRelaunchCommand(dest, os.Getpid()).Start()
}

func unixRelaunchCommand(dest string, pid int) *exec.Cmd {
	const waitAndLaunch = `while kill -0 "$1" 2>/dev/null; do sleep 0.1; done; exec "$2"`
	cmd := exec.Command("/bin/sh", "-c", waitAndLaunch, "picfetch-relaunch", strconv.Itoa(pid), dest)
	// The old process is gone if the eventual exec fails, so preserve the
	// shell's diagnostic on PicFetch's stderr instead of discarding it.
	cmd.Stderr = os.Stderr
	return cmd
}

// applyWindows's real implementation (apply_windows.go) only compiles on
// Windows. Apply's GOOS switch makes this stub unreachable; it exists so
// the package still compiles everywhere else.
func applyWindows(_ Stage, _ string, _ ApplyOptions) error {
	return errors.New("windows apply only exists in windows builds")
}
