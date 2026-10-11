//go:build !darwin || !appleappstore

package openwith

import "fyne.io/fyne/v2"

// InstallWindowDrop leaves the ordinary Fyne/platform drop callback unchanged.
func InstallWindowDrop(_ fyne.Window) bool { return true }
