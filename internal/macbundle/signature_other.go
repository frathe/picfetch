//go:build !darwin || !cgo

package macbundle

import (
	"errors"
	"os"
)

func verifyNative(_, _ string) error {
	return errors.New("native code verification requires macOS and cgo")
}

func executablePath() (string, error) { return os.Executable() }
