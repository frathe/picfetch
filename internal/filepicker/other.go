//go:build !darwin

package filepicker

import (
	"errors"

	"fyne.io/fyne/v2"
)

// chooseFilesDarwin's real implementation (darwin.go) is cgo/AppKit and
// only compiles on macOS. Choose's GOOS switch makes this stub unreachable;
// it exists so the package still compiles everywhere else.
func chooseFilesDarwin() ([]byte, error) {
	return nil, errors.New("the macOS file chooser only exists in darwin builds")
}

// chooseSaveDarwin is the same arrangement for the save panel, unreachable
// behind ChooseSave's own GOOS switch.
func chooseSaveDarwin(_ string) (fyne.URI, error) {
	return nil, errors.New("the macOS save panel only exists in darwin builds")
}
