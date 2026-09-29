package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/theme"
)

// onPictureFrameActive runs after picture-frame mode enters or leaves.
// The sliding bar is a window-menu concern, so it stays next to menu
// refresh rather than inside the slideshow package.
func (v *viewer) onPictureFrameActive() {
	v.syncPictureFrameMenu()
	v.syncMenus()
}

// syncPictureFrameMenu detaches the in-window menu while picture-frame
// mode is up and puts the same menu back on the way out. macOS, and any
// run with the flag off, keeps the menu Fyne already installed.
func (v *viewer) syncPictureFrameMenu() {
	if v.frameChrome == nil {
		return
	}
	if v.pictureFrameSlidingMenu && v.slides != nil && v.slides.Active() {
		if v.win != nil && v.win.MainMenu() != nil {
			v.win.SetMainMenu(nil)
		}
		// Fyne's glfw driver clears the menu overlay without moving the
		// content back up, so the old menu strip stays empty and the
		// sliding bar starts below the screen edge. Pull the content
		// flush with the canvas, padding included, before the bar is shown.
		v.useFullBleedForPictureFrame()
		v.frameChrome.Activate(v.mainMenu)
		return
	}
	v.frameChrome.Deactivate()
	v.restorePictureFrameBleed()
	v.restoreMainMenu()
}

// useFullBleedForPictureFrame drops the window padding and sizes the
// content to the canvas. The hot strip then includes the top screen edge.
func (v *viewer) useFullBleedForPictureFrame() {
	if v.win == nil {
		return
	}
	if v.pictureFramePadSaved {
		v.refitContentToCanvas()
		return
	}
	v.pictureFrameWasPadded = v.win.Padded()
	v.pictureFramePadSaved = true
	if v.pictureFrameWasPadded {
		v.win.SetPadded(false)
	}
	v.refitContentToCanvas()
}

func (v *viewer) restorePictureFrameBleed() {
	if v.win == nil || !v.pictureFramePadSaved {
		return
	}
	v.pictureFramePadSaved = false
	if v.pictureFrameWasPadded {
		v.win.SetPadded(true)
	}
	// The in-window menu, once restored, insets the content again. This
	// covers the test driver, which does not relayout on SetMainMenu.
	v.refitContentToCanvas()
}

// refitContentToCanvas places the window content where Fyne would after a
// real resize. Clearing the in-window menu does not do that itself.
func (v *viewer) refitContentToCanvas() {
	if v.win == nil {
		return
	}
	content := v.win.Content()
	if content == nil {
		return
	}
	size := v.win.Canvas().Size()
	if size.Width < 1 || size.Height < 1 {
		return
	}
	pos := fyne.NewPos(0, 0)
	if v.win.Padded() {
		pad := theme.Size(theme.SizeNamePadding)
		pos = fyne.NewPos(pad, pad)
		size = size.Subtract(fyne.NewSquareSize(pad * 2))
		if size.Width < 1 {
			size.Width = 1
		}
		if size.Height < 1 {
			size.Height = 1
		}
	}
	content.Move(pos)
	content.Resize(size)
}

func (v *viewer) restoreMainMenu() {
	if v.mainMenu == nil || v.win == nil || v.win.MainMenu() == v.mainMenu {
		return
	}
	v.win.SetMainMenu(v.mainMenu)
}
