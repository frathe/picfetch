//go:build !darwin || no_native_menus

package ui

// inWindowMenuBar is true where Fyne draws the menu inside the window.
// Picture-frame mode replaces that bar with the sliding one.
func inWindowMenuBar() bool { return true }
