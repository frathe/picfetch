//go:build !darwin

package clipboard

import "errors"

func publishPNGDarwin(_ []byte) error {
	return errors.New("copying images to the clipboard via AppKit is only available on macOS")
}

// copyFilesDarwin is the non-darwin half of the build-tag pair whose real
// implementation lives in darwin.go. CopyFiles never reaches it off macOS;
// it exists so the dispatcher there compiles everywhere, the same shape
// internal/trash's own other.go has.
func copyFilesDarwin(_ []string) error {
	return errors.New("copying files to the clipboard via AppKit is only available on macOS")
}
