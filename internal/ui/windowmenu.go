// Window menu enablement: which surface is already showing, and the
// show/enter actions those items run. Composed in menu.go; this file
// keeps Disabled in sync and implements the actions.

package ui

import "fyne.io/fyne/v2"

func (v *viewer) applyWindowMenuState() {
	if v.windowViewerItem == nil {
		return
	}
	_, displayed := v.DisplayedFile()
	v.windowViewerItem.Disabled = !v.grid.Visible() && !v.slides.Active()
	v.windowExifItem.Disabled = v.exif.Open() || !displayed
	v.windowGridItem.Disabled = v.grid.Visible() || v.FileCount() == 0 || v.slides.Active()
	v.windowPictureFrameItem.Disabled = v.slides.Active() || v.FileCount() == 0
	v.windowHelpItem.Disabled = v.help.ManualOpen()
}

func (v *viewer) refreshMainMenu() {
	if v.win == nil || v.win.MainMenu() == nil {
		return
	}
	bar := v.win.MainMenu()
	bar.Refresh()
	// Refresh is SetMainMenu, which rebuilds the Darwin native bar
	// (clearNativeMenu + a new Fyne Window next to GLFW's). Merge now if
	// that rebuild already ran, and again on the next UI turn in case it
	// was only queued. fyne.Do before Run runs inline and is too early
	// (see Run: fold after Show).
	syncNativeMenuBar(bar)
	fyne.Do(func() {
		if v.win == nil {
			return
		}
		syncNativeMenuBar(v.win.MainMenu())
	})
}

// syncNativeMenuBar is the Darwin native-bar follow-up after every Fyne
// MainMenu rebuild: fold Window items into NSApp.windowsMenu, then clear
// AppKit's default Command mask on unmodified letter accelerators. Both
// steps are no-ops off Darwin.
func syncNativeMenuBar(bar *fyne.MainMenu) {
	mergeNativeWindowMenu()
	applyUnmodifiedNativeAccelerators(bar)
}

func (v *viewer) updateWindowMenuState() {
	v.applyWindowMenuState()
	v.applyActionsMenuState()
	v.refreshMainMenu()
}

func (v *viewer) showViewer() {
	v.grid.Close()
	if v.slides.Active() {
		v.slides.Exit()
		v.resetFade()
	}
	v.win.RequestFocus()
	v.updateWindowMenuState()
}

func (v *viewer) showWindowExif() {
	v.exif.Show()
	v.updateWindowMenuState()
}

func (v *viewer) showWindowGrid() {
	if v.grid.Visible() || v.slides.Active() || v.FileCount() == 0 {
		return
	}
	v.grid.Toggle()
	v.updateWindowMenuState()
}

func (v *viewer) showWindowPictureFrame() {
	if v.slides.Active() || v.FileCount() == 0 {
		return
	}
	v.togglePictureFrameMode()
	v.updateWindowMenuState()
}

func (v *viewer) showWindowHelp() {
	v.help.ShowManual()
	v.updateWindowMenuState()
}
