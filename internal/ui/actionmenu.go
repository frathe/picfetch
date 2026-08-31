// Actions menu: sort, duplicates, image transforms, merge/info toggles,
// clipboard, wallpaper, and trash. The items and their Checked/Disabled
// matrix live in internal/ui/menus, composed in menu.go; this file holds
// the actions those items run.

package ui

import (
	"github.com/frathe/picfetch/internal/filesort"
)

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
	// Not redundant with the grid observers: the model's ClearInspect
	// fires nothing, and SetBrowsingDuplicates can no-op without firing
	// when it finds no source file - this call is what resyncs the menus
	// on that path, for every door in (Escape, G, Window -> Grid View).
	v.syncMenus()
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
	if v.display.Count() == 0 || v.grid.Visible() {
		return
	}
	v.rotateBy(1)
}

func (v *viewer) zoomActionsIn() {
	if v.display.Count() == 0 || v.grid.Visible() {
		return
	}
	v.zoom.In()
}

func (v *viewer) zoomActionsOut() {
	if v.display.Count() == 0 || v.grid.Visible() {
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

func (v *viewer) copyActionsSelection() {
	if !v.regionCopyAvailable() {
		return
	}
	v.startRegionCopy()
}

func (v *viewer) copyActionsPath() { v.copyPathToClipboard() }

func (v *viewer) wallpaperActionsImage() { v.setAsWallpaper() }

func (v *viewer) trashActionsImage() {
	if v.FileCount() == 0 {
		return
	}
	v.requestDelete()
}
