// Window menu: the show/enter actions its items run, plus the native
// menu-bar refresh every menu rebuild needs. Which surface is already
// showing decides what is enabled, but that matrix lives in
// internal/ui/menus now; this file implements the actions. The actions
// don't resync that matrix themselves: the feature observers registered
// in buildMainMenu (grid visibility, slideshow active, manual
// opened/closed) fire syncMenus after each surface change settles, on
// these doors and every other one - the grid's own G/Escape, the About
// window's manual link - alike. The EXIF panel is the one exception: it
// reports closes but not opens, so its openers sync by hand (see
// showWindowExif).

package ui

import "fyne.io/fyne/v2"

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

func (v *viewer) showViewer() {
	// Close() ClearInspects even when the overlay is already hidden.
	// V / Window → Viewer is a no-op in the image view, so skip Close
	// there or inspect would end. Leave the grid or picture-frame as before.
	if v.grid.Visible() || v.slides.Active() {
		v.grid.Close()
	}
	if v.slides.Active() {
		v.slides.Exit()
		v.resetFade()
	}
	v.win.RequestFocus()
}

func (v *viewer) showWindowExif() {
	v.exif.Show()
	// By hand: the EXIF window fires an observer on close (SetOnClosed in
	// buildMainMenu) but none on open, so every opener - this item, the E
	// key, the info card's link - resyncs itself after Show.
	v.syncMenus()
}

func (v *viewer) showWindowGrid() {
	if v.grid.Visible() || v.slides.Active() || v.FileCount() == 0 {
		return
	}
	if v.dupes.Inspecting() {
		v.reopenVariantGrid()
		return
	}
	v.grid.Toggle()
}

func (v *viewer) showWindowPictureFrame() {
	if v.slides.Active() || v.FileCount() == 0 {
		return
	}
	if v.variantsSession() {
		return
	}
	v.togglePictureFrameMode()
}

func (v *viewer) showWindowHelp() {
	v.help.ShowManual()
}
