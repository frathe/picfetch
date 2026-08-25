// The window's menu bar: File (open, close, settings), Favorites, Actions,
// Window, and Help.

package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/filesort"
)

// buildMainMenu assembles the menu bar. Composed here rather than inside
// either feature package, per the "internal/ui decides how features
// compose" rule (see ARCHITECTURE.md) - help.Menu and settingswin.Window
// both stay ignorant of where they sit in the bar.
func buildMainMenu(view *viewer) *fyne.MainMenu {
	open := fyne.NewMenuItem(lang.L("Open Files…"), func() { view.openFileDialog() })
	// Display-only: the Cmd/Ctrl+O binding itself is wireOpenShortcuts's
	// AddShortcut call in shortcuts.go. This just shows the same accelerator
	// as a hint next to the menu item.
	open.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyO,
		Modifier: fyne.KeyModifierShortcutDefault,
	}

	save := fyne.NewMenuItem(lang.L("Save Changes"), func() { view.saveRotation() })
	save.Disabled = true // updateFileMenuState (save.go) enables it once there's a pending rotation to save
	save.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyS,
		Modifier: fyne.KeyModifierShortcutDefault,
	}
	view.saveItem = save

	export := fyne.NewMenuItem(lang.L("Export image"), func() { view.promptExport() })
	export.Disabled = true // updateFileMenuState (save.go) enables it once an image is loaded
	// Display-only, like Open's above: the binding itself is
	// wireExportShortcuts's AddShortcut call in shortcuts.go.
	export.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyE,
		Modifier: fyne.KeyModifierShortcutDefault,
	}
	view.exportItem = export

	closeFiles := fyne.NewMenuItem(lang.L("Close Files"), func() { view.closeFiles() })
	closeFiles.Disabled = true // updateFileMenuState (save.go) enables it once a file is loaded
	view.closeFilesItem = closeFiles
	settings := fyne.NewMenuItem(lang.L("Settings…"), func() { view.settingsWin.Show() })

	fileMenu := fyne.NewMenu(lang.L("File"),
		open, save, export, closeFiles, fyne.NewMenuItemSeparator(), settings)

	viewerItem := fyne.NewMenuItem(lang.L("Viewer"), view.showViewer)
	viewerItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyV}
	viewerItem.Disabled = true
	view.windowViewerItem = viewerItem

	exifItem := fyne.NewMenuItem(lang.L("EXIF Data"), view.showWindowExif)
	exifItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyE}
	exifItem.Disabled = true
	view.windowExifItem = exifItem

	gridItem := fyne.NewMenuItem(lang.L("Grid View"), view.showWindowGrid)
	gridItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyG}
	gridItem.Disabled = true
	view.windowGridItem = gridItem

	pfItem := fyne.NewMenuItem(lang.L("Picture-frame mode"), view.showWindowPictureFrame)
	pfItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyP}
	pfItem.Disabled = true
	view.windowPictureFrameItem = pfItem

	helpItem := fyne.NewMenuItem(lang.L("Help"), view.showWindowHelp)
	helpItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyF1}
	view.windowHelpItem = helpItem

	windowMenu := fyne.NewMenu(lang.L("Window"),
		viewerItem, exifItem, gridItem, pfItem, helpItem)

	modes := filesort.Modes()
	sortItems := make([]*fyne.MenuItem, len(modes))
	for i, m := range modes {
		mode := m
		it := fyne.NewMenuItem(filesort.DisplayName(mode), func() { view.setActionsSort(mode) })
		if mode == view.SortMode() {
			it.Checked = true
		}
		sortItems[i] = it
	}
	view.actionsSortItems = sortItems

	sortParent := fyne.NewMenuItem(lang.L("Sort order"), nil)
	sortParent.ChildMenu = fyne.NewMenu("", sortItems...)
	sortParent.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyS}

	hideItem := fyne.NewMenuItem(lang.L("Show/Hide duplicates"), view.toggleActionsHideDuplicates)
	hideItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyD}
	hideItem.Disabled = true
	view.actionsHideItem = hideItem

	variantItem := fyne.NewMenuItem(lang.L("Show variants"), view.showActionsVariant)
	variantItem.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyD,
		Modifier: fyne.KeyModifierShift,
	}
	variantItem.Disabled = true
	view.actionsShowVariantItem = variantItem

	rotateItem := fyne.NewMenuItem(lang.L("Rotate image"), view.rotateActionsImage)
	rotateItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyR}
	rotateItem.Disabled = true
	view.actionsRotateItem = rotateItem

	zoomIn := fyne.NewMenuItem(lang.L("Zoom in"), view.zoomActionsIn)
	zoomIn.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyPlus}
	zoomIn.Disabled = true
	view.actionsZoomInItem = zoomIn

	zoomOut := fyne.NewMenuItem(lang.L("Zoom out"), view.zoomActionsOut)
	zoomOut.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyMinus}
	zoomOut.Disabled = true
	view.actionsZoomOutItem = zoomOut

	mergeItem := fyne.NewMenuItem(lang.L("Toggle merge mode"), view.toggleActionsMergeMode)
	// Unmodified M. Fyne's Darwin native menus leave a zero modifier mask
	// unset, so AppKit would default this to ⌘M (Minimize); refreshMainMenu
	// clears that via applyUnmodifiedNativeAccelerators.
	mergeItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyM}
	view.actionsMergeItem = mergeItem

	infoItem := fyne.NewMenuItem(lang.L("Show/Hide info overlay"), view.toggleActionsInfoOverlay)
	infoItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyI}
	view.actionsInfoItem = infoItem

	copyItem := fyne.NewMenuItem(lang.L("Copy image"), view.copyActionsImage)
	// Display-only: the Cmd/Ctrl+C binding is wireClipboardShortcuts's
	// AddShortcut of fyne.ShortcutCopy. A second CustomShortcut here would
	// double-fire copy.
	copyItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyC, Modifier: fyne.KeyModifierShortcutDefault}
	copyItem.Disabled = true
	view.actionsCopyItem = copyItem

	copyPath := fyne.NewMenuItem(lang.L("Copy image path"), view.copyActionsPath)
	// Display-only, like File -> Export: the Cmd/Ctrl+Shift+C binding is
	// wireClipboardShortcuts. This just shows the same accelerator.
	copyPath.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyC,
		Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift,
	}
	copyPath.Disabled = true
	view.actionsCopyPathItem = copyPath

	wallpaperAction := fyne.NewMenuItem(lang.L("Set as Wallpaper"), view.wallpaperActionsImage)
	// Display-only: the Cmd/Ctrl+Shift+E binding is wireExportShortcuts.
	wallpaperAction.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyE,
		Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift,
	}
	wallpaperAction.Disabled = true
	view.actionsWallpaperItem = wallpaperAction

	trashItem := fyne.NewMenuItem(lang.L("Move image to Trash"), view.trashActionsImage)
	// Display-only: the Shift+Delete binding is wireDeleteShortcut's
	// AddShortcut of fyne.ShortcutCut. A CustomShortcut{KeyDelete, Shift}
	// would never be reached by the driver anyway.
	trashItem.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyDelete, Modifier: fyne.KeyModifierShift}
	trashItem.Disabled = true
	view.actionsTrashItem = trashItem

	actionsMenu := fyne.NewMenu(lang.L("Actions"),
		sortParent, hideItem, variantItem,
		fyne.NewMenuItemSeparator(),
		rotateItem, zoomIn, zoomOut,
		fyne.NewMenuItemSeparator(),
		mergeItem, infoItem,
		fyne.NewMenuItemSeparator(),
		copyItem, copyPath, wallpaperAction, trashItem,
	)

	view.help.SetOnManualClosed(view.updateWindowMenuState)
	view.help.SetOnManualOpened(view.updateWindowMenuState)
	view.exif.SetOnClosed(view.updateWindowMenuState)
	view.grid.SetOnVisibilityChanged(view.updateWindowMenuState)
	view.slides.SetOnActiveChanged(view.updateWindowMenuState)
	view.grid.SetOnDupeStateChanged(view.updateActionsMenuState)
	view.applyActionsMenuState()
	view.updateWindowMenuState()

	return fyne.NewMainMenu(fileMenu, view.favorites.Menu(), actionsMenu, windowMenu, view.help.Menu())
}
