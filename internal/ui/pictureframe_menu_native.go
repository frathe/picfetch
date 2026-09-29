//go:build darwin && !no_native_menus

package ui

// inWindowMenuBar is false where Fyne uses the system menu bar. That bar
// is outside the window, and fullscreen already shows and hides it.
func inWindowMenuBar() bool { return false }
