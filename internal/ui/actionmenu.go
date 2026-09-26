// Actions menu: sort, duplicates, image transforms, merge/info toggles,
// clipboard, reveal, wallpaper, and trash. The items and their Checked/Disabled
// matrix live in internal/ui/menus, composed in menu.go; this file holds
// the actions those items run.

package ui

import (
	"github.com/frathe/picfetch/internal/filesort"
)

func (v *viewer) setActionsSort(m filesort.Mode) {
	if _, ok := v.admitCommand(commandRequest{command: commandSort, sortMode: m}); !ok {
		return
	}
	v.SetSortMode(m)
}

func (v *viewer) toggleHideDuplicates() {
	if _, ok := v.admitCommand(commandRequest{command: commandHideDuplicates}); !ok {
		return
	}
	v.pushHideDuplicates(!v.dupes.HideDuplicates())
}

func (v *viewer) browseCurrentDuplicates() {
	if _, ok := v.admitCommand(commandRequest{command: commandBrowseDuplicates, intent: intentToggle}); !ok {
		return
	}
	v.grid.ToggleBrowseDuplicates()
}

func (v *viewer) reopenVariantGrid() {
	if v.slides.Active() {
		return
	}
	v.dupes.ClearInspect()
	v.grid.SetBrowsingDuplicates(true)
	// Not redundant with the grid observers: the model's ClearInspect
	// fires nothing, and SetBrowsingDuplicates can no-op without firing
	// when it finds no source file - this call is what resyncs the menus
	// on that path, for every door in (Escape, G, Window -> Grid View).
	v.syncMenus()
}

// Progress delivery is independent of accepted groups and native menu updates.
func (v *viewer) syncDuplicatePreparationProgress() {
	completed, total := v.grid.DuplicatePreparationProgress()
	if v.locationInput.prepare != nil {
		v.locationMap.SetPreparationProgress(completed, total)
	}
	if v.explorerInput.prepare != nil {
		v.explorer.SetPreparationProgress(completed, total)
	}
}

// Opening waits for the accepted group so a unique source remains a no-op.
func (v *viewer) syncDuplicateState() {
	if v.locationInput.prepare != nil && v.grid.DuplicateGroupsReady() {
		ready := v.locationInput.prepare
		v.locationInput.prepare = nil
		ready()
	}
	if v.explorerInput.prepare != nil && v.grid.DuplicateGroupsReady() {
		ready := v.explorerInput.prepare
		v.explorerInput.prepare = nil
		ready()
	}
	if v.grid.BrowseReady() && !v.grid.Visible() {
		if _, ok := v.admitCommand(commandRequest{command: commandBrowseDuplicates, intent: intentShow, route: routeDelivery}); ok {
			v.grid.Toggle()
		} else {
			// Preparation has finished, but its presentation is no longer
			// admitted. Retire it rather than replaying after the prompt.
			v.grid.SetBrowsingDuplicates(false)
		}
	}
	v.syncMenus()
}

func (v *viewer) variantsSession() bool {
	return v.grid.BrowsingDuplicates() || v.dupes.Inspecting()
}

func (v *viewer) toggleActionsHideDuplicates() {
	if _, ok := v.admitCommand(commandRequest{command: commandHideDuplicates, route: routeMenu}); !ok {
		return
	}
	v.toggleHideDuplicates()
}

func (v *viewer) showActionsVariant() {
	if _, ok := v.admitCommand(commandRequest{command: commandBrowseDuplicates, intent: intentShow}); !ok {
		return
	}
	v.browseCurrentDuplicates()
}

func (v *viewer) rotateActionsImage() {
	v.rotateBy(1)
}

func (v *viewer) zoomActionsIn() {
	if _, ok := v.admitCommand(commandRequest{command: commandZoom}); !ok {
		return
	}
	v.zoom.In()
}

func (v *viewer) zoomActionsOut() {
	if _, ok := v.admitCommand(commandRequest{command: commandZoom}); !ok {
		return
	}
	v.zoom.Out()
}

func (v *viewer) toggleActionsMergeMode() {
	v.toggleMergeMode()
}

func (v *viewer) toggleActionsInfoOverlay() {
	v.toggleInfoOverlay()
}

func (v *viewer) copyActionsImage() {
	v.copySelection()
}

func (v *viewer) copyActionsSelection() {
	v.startRegionCopy()
}

func (v *viewer) copyActionsPath() {
	v.copyPathToClipboard()
}

func (v *viewer) revealActionsFile() {
	v.revealCurrentFile()
}

func (v *viewer) wallpaperActionsImage() {
	v.setAsWallpaper()
}

func (v *viewer) trashActionsImage() {
	v.requestDelete()
}
