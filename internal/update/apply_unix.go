//go:build !windows

package update

import (
	"errors"
	"io"
	"os"
	"path/filepath"
)

func applyUnix(stage Stage, dest string) error {
	dest, err := filepath.EvalSymlinks(dest)
	if err != nil {
		return err
	}
	old := dest + ".old"
	if err := os.Rename(dest, old); err != nil {
		return err
	}
	if err := copyFile(stage.BinaryPath, dest); err != nil {
		if rbErr := os.Rename(old, dest); rbErr != nil {
			return errors.Join(err, rbErr)
		}
		return err
	}
	if err := os.Chmod(dest, 0o755); err != nil {
		return err
	}
	_ = os.Remove(old)
	if stage.PlistPath != "" {
		plistDest := filepath.Join(filepath.Dir(dest), "..", "Info.plist")
		if err := copyFile(stage.PlistPath, plistDest); err != nil {
			return err
		}
	}
	return nil
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
func applyWindows(Stage, string) error {
	return errors.New("windows apply only exists in windows builds")
}
