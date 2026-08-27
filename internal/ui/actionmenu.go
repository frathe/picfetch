// Actions menu: sort, duplicates, image transforms, merge/info toggles,
// clipboard, wallpaper, and trash. Composed in menu.go; this file keeps
// Checked/Disabled in sync and holds the actions those items run.

package ui

import (
	"github.com/frathe/picfetch/internal/filesort"
)

func (v *viewer) applyActionsMenuState() {
	if v.actionsHideItem == nil {
		return
	}
	modes := filesort.Modes()
	cur := v.SortMode()
	for i, item := range v.actionsSortItems {
		if item == nil || i >= len(modes) {
			continue
		}
		item.Checked = modes[i] == cur
		item.Disabled = false
	}
	noFiles := v.FileCount() == 0
	gridUp := v.grid.Visible()
	noImage := len(v.displayFrames) == 0

	v.actionsHideItem.Checked = v.dupes.HideDuplicates()
	v.actionsHideItem.Disabled = noFiles || v.variantsSession()
	v.actionsShowVariantItem.Checked = v.grid.BrowsingDuplicates()
	canShowVariants := v.dupes.HideDuplicates() && v.grid.SourceDuplicateGroupSize() >= 2
	v.actionsShowVariantItem.Disabled = noFiles || v.slides.Active() || !(canShowVariants || v.grid.BrowsingDuplicates())

	rotZoomOff := noImage || gridUp
	v.actionsRotateItem.Disabled = rotZoomOff
	v.actionsZoomInItem.Disabled = rotZoomOff
	v.actionsZoomOutItem.Disabled = rotZoomOff

	v.actionsMergeItem.Checked = v.MergeMode()
	v.actionsMergeItem.Disabled = false
	v.actionsInfoItem.Checked = v.infoVisible
	v.actionsInfoItem.Disabled = gridUp

	v.actionsCopyItem.Disabled = noFiles
	v.actionsCopyPathItem.Disabled = noFiles
	v.actionsWallpaperItem.Disabled = !v.canSetWallpaper()
	v.actionsTrashItem.Disabled = noFiles
}

func (v *viewer) updateActionsMenuState() {
	v.applyWindowMenuState()
	v.applyActionsMenuState()
	v.refreshMainMenu()
}

func (v *viewer) setActionsSort(m filesort.Mode) {
	if v.SortMode() == m {
		return
	}
	v.SetSortMode(m)
}

func (v *viewer) toggleHideDuplicates() {
	v.pushHideDuplicates(!v.dupes.HideDuplicates())
}

func (v *viewer) browseCurrentDuplicates() {
	if v.slides.Active() {
		return
	}
	v.grid.ToggleBrowseDuplicates()
	if v.grid.BrowsingDuplicates() && !v.grid.Visible() {
		v.grid.Toggle()
	}
}

func (v *viewer) reopenVariantGrid() {
	if v.slides.Active() {
		return
	}
	v.dupes.ClearInspect()
	v.grid.SetBrowsingDuplicates(true)
	if v.grid.BrowsingDuplicates() && !v.grid.Visible() {
		v.grid.Toggle()
	}
	v.updateWindowMenuState()
}

func (v *viewer) variantsSession() bool {
	return v.grid.BrowsingDuplicates() || v.dupes.Inspecting()
}

func (v *viewer) toggleActionsHideDuplicates() {
	if v.FileCount() == 0 || v.variantsSession() {
		return
	}
	v.toggleHideDuplicates()
}

func (v *viewer) showActionsVariant() {
	if v.FileCount() == 0 || v.slides.Active() {
		return
	}
	if v.grid.BrowsingDuplicates() {
		v.browseCurrentDuplicates() // leave browse even if hide is now off
		return
	}
	if !v.dupes.HideDuplicates() || v.grid.SourceDuplicateGroupSize() < 2 {
		return
	}
	v.browseCurrentDuplicates()
}

func (v *viewer) rotateActionsImage() {
	if len(v.displayFrames) == 0 || v.grid.Visible() {
		return
	}
	v.rotateBy(1)
}

func (v *viewer) zoomActionsIn() {
	if len(v.displayFrames) == 0 || v.grid.Visible() {
		return
	}
	v.zoom.In()
}

func (v *viewer) zoomActionsOut() {
	if len(v.displayFrames) == 0 || v.grid.Visible() {
		return
	}
	v.zoom.Out()
}

func (v *viewer) toggleActionsMergeMode() { v.toggleMergeMode() }

func (v *viewer) toggleActionsInfoOverlay() {
	if v.grid.Visible() {
		return
	}
	v.toggleInfoOverlay()
}

func (v *viewer) copyActionsImage() {
	if v.FileCount() == 0 {
		return
	}
	v.copySelection()
}

func (v *viewer) copyActionsPath() { v.copyPathToClipboard() }

func (v *viewer) wallpaperActionsImage() { v.setAsWallpaper() }

func (v *viewer) trashActionsImage() {
	if v.FileCount() == 0 {
		return
	}
	v.requestDelete()
}
