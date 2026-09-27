// The window's menu bar: File (open, close, settings), Favorites, Actions,
// Window, and Help.

package ui

import (
	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/ui/favorites"
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
		FindMoreLikeThis: view.findMoreLikeThis,
		OpenFiles:        func() { view.openFileDialog() },
		SaveRotation:     func() { view.saveRotation() },
		PromptExport:     func() { view.promptExport() },
		CloseFiles:       func() { view.closeFiles() },
		ShowSettings:     view.showSettings,

		ShowViewer:       view.showViewer,
		ShowExplorer:     view.showExplorer,
		ShowLocationMap:  view.showLocationMap,
		ShowExif:         view.showWindowExif,
		ShowGrid:         view.showWindowGrid,
		ShowPictureFrame: view.showWindowPictureFrame,
		ShowHelp:         view.showWindowHelp,

		SetSort:              view.setActionsSort,
		ToggleHideDuplicates: view.toggleActionsHideDuplicates,
		ShowVariant:          view.showActionsVariant,
		Compare:              view.compareSelected,
		Mosaic:               view.showMosaic,
		Rotate:               view.rotateActionsImage,
		ZoomIn:               view.zoomActionsIn,
		ZoomOut:              view.zoomActionsOut,
		ToggleMergeMode:      view.toggleActionsMergeMode,
		ToggleInfoOverlay:    view.toggleActionsInfoOverlay,
		CopyImage:            view.copyActionsImage,
		CopySelection:        view.copyActionsSelection,
		CopyPath:             view.copyActionsPath,
		Reveal:               view.revealActionsFile,
		SetWallpaper:         view.wallpaperActionsImage,
		Trash:                view.trashActionsImage,
	}, view.SortMode())

	view.help.SetOnManualClosed(view.syncMenus)
	view.help.SetOnManualOpened(view.syncMenus)
	view.exif.SetOnClosed(view.syncMenus)
	view.grid.SetOnVisibilityChanged(view.explorerGridChanged)
	view.grid.SetOnSelectionChanged(view.syncMenus)
	view.grid.SetOnResultChanged(view.syncMenus)
	view.slides.SetOnActiveChanged(view.syncMenus)
	view.grid.SetOnDupeStateChanged(view.syncDuplicateState)
	view.grid.SetOnDuplicateProgress(view.syncDuplicatePreparationProgress)
	view.syncMenus()

	return fyne.NewMainMenu(view.menus.FileMenu(), view.favorites.Menu(), view.menus.ActionsMenu(), view.menus.WindowMenu(), view.help.Menu())
}

// menuState is the sole adapter from UI-owned facts to the menu snapshot.
// Decisions are derived from one observation; no query yields or starts work.
func (v *viewer) menuState() menus.State {
	return v.menuStateFor(v.commandContext())
}

// Keep named menu decisions explicit; similar field assignments are not a
// second policy or a reason to introduce a runtime command registry.
//
//goland:noinspection DuplicatedCode
func (v *viewer) menuStateFor(context commandContext) menus.State {
	can := func(id commandID, intent commandIntent) bool {
		return decideCommand(commandRequest{command: id, intent: intent, route: routeMenu}, context).allowed
	}
	return menus.State{
		SortMode: v.SortMode(), MergeMode: v.MergeMode(),
		HideDuplicates: context.hideDuplicates, BrowsingDuplicates: context.browsingDuplicates,
		InfoVisible: v.info.Visible(), LocationMapActive: context.surface == surfaceLocationMap,
		Availability: menus.Availability{
			Open: can(commandOpenChooser, intentAction), Save: can(commandSave, intentAction),
			Export: can(commandExport, intentAction), CloseFiles: can(commandCloseFiles, intentAction),
			Settings: can(commandSettings, intentShow), Viewer: can(commandViewer, intentShow),
			Explorer: can(commandExplorer, intentShow), LocationMap: can(commandLocationMap, intentShow),
			Mosaic: can(commandMosaic, intentShow), Exif: can(commandExif, intentShow),
			Grid: can(commandGrid, intentShow), PictureFrame: can(commandPictureFrame, intentShow),
			Help: can(commandHelp, intentShow), Sort: can(commandSort, intentToggle),
			HideDuplicates: can(commandHideDuplicates, intentToggle), BrowseDuplicates: can(commandBrowseDuplicates, intentShow),
			Compare: can(commandCompare, intentShow), Search: can(commandSearch, intentAction),
			Rotate: can(commandRotate, intentAction), Zoom: can(commandZoom, intentAction),
			Merge: can(commandMerge, intentToggle), Info: can(commandInfo, intentToggle),
			Copy: can(commandCopy, intentAction), CopySelection: can(commandCopyRegion, intentAction),
			CopyPath: can(commandCopyPath, intentAction), Reveal: can(commandReveal, intentAction),
			Wallpaper: can(commandWallpaper, intentAction), Trash: can(commandTrash, intentAction),
		},
	}
}

// syncMenus publishes all feature menu decisions together, once, only when
// their rendered state changed. Native reconstruction sees the complete update.
func (v *viewer) syncMenus() {
	if v.stopping || v.menus == nil {
		return
	}
	context := v.commandContext()
	changed := v.menus.Apply(v.menuStateFor(context))
	can := func(id commandID) bool {
		return decideCommand(commandRequest{command: id, route: routeMenu}, context).allowed
	}
	if v.favorites.SetAvailability(favorites.Availability{
		Open: can(commandFavoriteOpen), Add: can(commandFavoriteAdd), Manage: can(commandFavoriteManage),
	}) {
		changed = true
	}
	if v.help.SetCommandsEnabled(can(commandHelp)) {
		changed = true
	}
	if changed {
		v.refreshMainMenu()
	}
}
