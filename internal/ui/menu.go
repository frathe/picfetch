// The window's menu bar: File (open, close, settings), Favorites, Actions,
// Window, and Help.

package ui

import (
	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/ui/menus"
)

// buildMainMenu assembles the menu bar. Composed here rather than inside
// either feature package, per the "internal/ui decides how features
// compose" rule (see ARCHITECTURE.md) - help.Menu and settingswin.Window
// both stay ignorant of where they sit in the bar. The File, Actions and
// Window items themselves, and the whole Checked/Disabled matrix over
// them, live in internal/ui/menus; everything those items do stays here.
func buildMainMenu(view *viewer) *fyne.MainMenu {
	view.menus = menus.New(menus.Callbacks{
		OpenFiles:    func() { view.openFileDialog() },
		SaveRotation: func() { view.saveRotation() },
		PromptExport: func() { view.promptExport() },
		CloseFiles:   func() { view.closeFiles() },
		ShowSettings: func() {
			if !view.cancelRegionCopyBeforeAction() {
				return
			}
			view.settingsWin.Show()
		},

		ShowViewer:       view.showViewer,
		ShowExif:         view.showWindowExif,
		ShowGrid:         view.showWindowGrid,
		ShowPictureFrame: view.showWindowPictureFrame,
		ShowHelp:         view.showWindowHelp,

		SetSort:              view.setActionsSort,
		ToggleHideDuplicates: view.toggleActionsHideDuplicates,
		ShowVariant:          view.showActionsVariant,
		Rotate:               view.rotateActionsImage,
		ZoomIn:               view.zoomActionsIn,
		ZoomOut:              view.zoomActionsOut,
		ToggleMergeMode:      view.toggleActionsMergeMode,
		ToggleInfoOverlay:    view.toggleActionsInfoOverlay,
		CopyImage:            view.copyActionsImage,
		CopySelection:        view.copyActionsSelection,
		CopyPath:             view.copyActionsPath,
		SetWallpaper:         view.wallpaperActionsImage,
		Trash:                view.trashActionsImage,
	}, view.SortMode())

	view.help.SetOnManualClosed(view.syncMenus)
	view.help.SetOnManualOpened(view.syncMenus)
	view.exif.SetOnClosed(view.syncMenus)
	view.grid.SetOnVisibilityChanged(view.syncMenus)
	view.slides.SetOnActiveChanged(view.syncMenus)
	view.grid.SetOnDupeStateChanged(view.syncMenus)
	view.syncMenus()

	return fyne.NewMainMenu(view.menus.FileMenu(), view.favorites.Menu(), view.menus.ActionsMenu(), view.menus.WindowMenu(), view.help.Menu())
}

// menuState is the one place the snapshot internal/ui/menus reads is
// built: every condition its enablement matrix depends on, gathered from
// whichever feature owns it. One function rather than a Host interface
// the package calls back through - the dependency surface is then this
// struct literal, readable at a glance, instead of a dozen methods.
func (v *viewer) menuState() menus.State {
	_, displayed := v.DisplayedFile()

	return menus.State{
		SortMode:         v.SortMode(),
		VariantGroupSize: v.grid.SourceDuplicateGroupSize(),

		NoFiles:            v.FileCount() == 0,
		GridUp:             v.grid.Visible(),
		NoImage:            v.display.Count() == 0,
		SlidesActive:       v.slides.Active(),
		ExifOpen:           v.exif.Open(),
		ManualOpen:         v.help.ManualOpen(),
		Displayed:          displayed,
		MergeMode:          v.MergeMode(),
		HideDuplicates:     v.dupes.HideDuplicates(),
		BrowsingDuplicates: v.grid.BrowsingDuplicates(),
		VariantsSession:    v.variantsSession(),
		InfoVisible:        v.info.Visible(),
		CanSave:            v.canSaveRotation(),
		CanExport:          v.canExport(),
		CanWallpaper:       v.canSetWallpaper(),
		CanCopySelection:   v.regionCopyAvailable(),
	}
}

// syncMenus recomputes the whole menu matrix from the current state and
// rebuilds the native bar only when something in it actually moved. It is
// the single entry point every site that can change what is loaded,
// displayed or shown calls - directly, or through the feature observers
// registered in buildMainMenu, which cover the surface toggles (grid,
// picture-frame, manual, EXIF close) so those don't sync per call site;
// the nil guard covers the window of construction before buildMainMenu
// has run.
//
// Whether there are any files at all also drives the Favorites menu's
// "Add Current List" item, which belongs to that feature rather than to
// internal/ui/menus, so it is pushed here alongside.
//
// SetHasFiles is inside the changed branch, and before refreshMainMenu,
// for two reasons that are both load-bearing - do not lift it out:
//
//   - SetHasFiles itself no longer publishes anything - it only flips
//     addItem.Disabled - so refreshMainMenu on the next line is what
//     actually gets that new state onto the bar. Ordering it before
//     refreshMainMenu still matters: that is the one call that folds the
//     Darwin native bar back together (syncNativeMenuBar) after Apply's
//     own item changes, and it has to see the Favorites item's post-toggle
//     state to publish it, not the state from before this turn.
//   - Skipping it when nothing moved cannot skip a needed update: it reads
//     FileCount() > 0, the exact complement of the NoFiles that
//     menuState() reads in the same turn, and Apply assigns
//     closeFiles.Disabled = NoFiles outright. So the Favorites item can
//     only move on a turn where closeFiles moved too, which is a turn
//     Apply reports as changed.
//
// The startup sync is the one call where changed can be false with files
// already loaded; it costs nothing, because addItem is constructed
// Disabled and SetHasFiles(false) is what that sync would have set.
func (v *viewer) syncMenus() {
	if v.menus == nil {
		return
	}
	if v.menus.Apply(v.menuState()) {
		v.favorites.SetHasFiles(v.FileCount() > 0)
		v.refreshMainMenu()
	}
}
