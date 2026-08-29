//go:build !windows

package update

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
)

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
	newBinary := dest + ".new"
	old := dest + ".old"
	defer func() { _ = os.Remove(newBinary) }()
	if err := copyFile(stage.BinaryPath, newBinary); err != nil {
		return err
	}
	if err := os.Chmod(newBinary, 0o755); err != nil {
		return err
	}

	plistDest := ""
	newPlist := ""
	if stage.PlistPath != "" {
		plistDest = filepath.Join(filepath.Dir(dest), "..", "Info.plist")
		newPlist = plistDest + ".new"
		defer func() { _ = os.Remove(newPlist) }()
		if err := copyFile(stage.PlistPath, newPlist); err != nil {
			return err
		}
		if err := os.Chmod(newPlist, 0o644); err != nil {
			return err
		}
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
		return launch(dest)
	}
	return nil
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

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() { _ = in.Close() }()
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

// applyWindows's real implementation (apply_windows.go) only compiles on
// Windows. Apply's GOOS switch makes this stub unreachable; it exists so
// the package still compiles everywhere else.
func applyWindows(_ Stage, _ string, _ ApplyOptions) error {
	return errors.New("windows apply only exists in windows builds")
}
