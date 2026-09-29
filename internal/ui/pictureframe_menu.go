package ui

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
		v.frameChrome.Activate(v.mainMenu)
		return
	}
	v.frameChrome.Deactivate()
	v.restoreMainMenu()
}

func (v *viewer) restoreMainMenu() {
	if v.mainMenu == nil || v.win == nil || v.win.MainMenu() == v.mainMenu {
		return
	}
	v.win.SetMainMenu(v.mainMenu)
}
