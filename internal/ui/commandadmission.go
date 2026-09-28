package ui

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/ui/favorites"
)

func (v *viewer) AdmitFavorite(command favorites.Command) bool {
	var id commandID
	switch command {
	case favorites.OpenCommand:
		id = commandFavoriteOpen
	case favorites.AddCommand:
		id = commandFavoriteAdd
	case favorites.ManageCommand:
		id = commandFavoriteManage
	default:
		return false
	}
	_, ok := v.admitCommand(commandRequest{command: id})
	return ok
}

type deletionHost struct{ *viewer }

func (h deletionHost) ForceRepaint()   { h.promptChanged() }
func (h deletionHost) ShowImage(i int) { h.loadImage(i) }

func (v *viewer) promptChanged() {
	// In-tree cards do not have Fyne's overlay focus manager. Keep Tab and
	// prompt controls on one owner, then release focus on dismissal.
	switch {
	case v.deletion.Visible():
		v.deletion.Focus(v.win.Canvas())
	case v.exportPrompt.Visible():
		v.exportPrompt.Focus(v.win.Canvas())
	default:
		v.Unfocus()
	}
	v.syncMenus()
	v.ForceRepaint()
}

// commandContext observes UI-owned facts without capturing an action's payload
// or mutating the interaction. Availability queries use this same observation.
func (v *viewer) commandContext() commandContext {
	_, displayed := v.DisplayedFile()
	collection := v.state.Observe()
	context := commandContext{
		stopping:           v.stopping,
		canSave:            v.canSaveRotation(),
		cohortVisit:        v.browsing.has(browsingExplorer),
		locationVisit:      v.locationVisitActive(),
		searchVisit:        v.searchActive(),
		variantsVisit:      v.variantsSession(),
		regionActive:       v.regionCopy.State().Active,
		regionBusy:         v.regionCopy.State().Busy,
		clipboardBusy:      v.clipboardWork.pending.Load(),
		clipboardClosed:    v.clipboardWork.closed.Load(),
		hasFiles:           collection.Count() > 0,
		hasCollection:      collection.HasMembers(),
		hasImage:           v.display.Count() > 0 && v.img.Image != nil,
		hasPixels:          v.img.Image != nil,
		loading:            v.display.Snapshot().Loading,
		gridTargets:        v.grid.SelectionCount() > 0 || v.grid.Highlight() >= 0,
		fileWorkActive:     v.scanOp.active || v.sortOp.active,
		pendingLaunchFrame: v.pendingPictureFrame,
		chooserClosed:      v.openChooserClosed,
		hasSession:         len(v.savedSession) > 0,
		canExport:          v.canExport(),
		canWallpaper:       v.canSetWallpaper(),
		canCompare:         v.grid.Visible() && v.grid.SelectionCount() == 2,
		displayed:          displayed,
		exifOpen:           v.exif.Open(),
		manualOpen:         v.help.ManualOpen(),
		analysisBusy:       v.analysisMaintenanceBusy(),
		explorerCanRetry:   v.explorerCanRetry(),
		canMosaic:          v.canMosaic(),
		mosaicOpen:         v.mosaicWin.Opened(),
		hideDuplicates:     v.dupes.HideDuplicates(),
		browsingDuplicates: v.grid.BrowsingDuplicates(),
		hasSearchTarget:    v.searchTarget() != "",
		variantGroupSize:   v.grid.SourceDuplicateGroupSize(),
		sortMode:           v.SortMode(),
		canNavigate:        v.FileCount() >= 2 && !v.display.Snapshot().Loading,
	}
	switch {
	case v.comparisonActive():
		context.surface = surfaceComparison
	case v.explorerMapActive():
		context.surface = surfaceExplorer
	case v.locationMapVisible():
		context.surface = surfaceLocationMap
	case v.grid.Visible():
		context.surface = surfaceGrid
	case v.slides.Active():
		context.surface = surfacePictureFrame
	}
	if _, editor := v.win.Canvas().Focused().(interface {
		fyne.Shortcutable
		SelectedText() string
	}); editor {
		context.input = inputEditor
		context.editorFocused = true
	}
	if overlay := v.win.Canvas().Overlays().Top(); overlay != nil {
		// Fyne wraps PopUpMenu in an internal overlay container, but gives
		// the menu focus. A dialog below that menu still owns input.
		if _, menu := v.win.Canvas().Focused().(*widget.PopUpMenu); menu && len(v.win.Canvas().Overlays().List()) == 1 {
			context.input = inputMenu
		} else {
			context.input = inputModal
		}
	}
	if v.deletion.Visible() || v.exportPrompt.Visible() {
		context.input = inputModal
		context.editorFocused = false
	}
	return context
}

func (v *viewer) queryCommand(request commandRequest) commandDecision {
	return decideCommand(request, v.commandContext())
}

// admitCommand reevaluates live facts before yielding. It never captures or
// executes a command, and a refusal leaves the owning interaction intact.
func (v *viewer) admitCommand(request commandRequest) (commandDecision, bool) {
	decision := v.queryCommand(request)
	if !decision.allowed {
		if decision.refusal == refusalRegionBusy || decision.refusal == refusalClipboardBusy {
			v.ShowToast(lang.L("finishing the copy - try again in a moment"))
		}
		if decision.refusal == refusalComparisonOpen && request.route != routeDelivery {
			v.ShowToast(lang.L("Return to Grid View before opening files"))
		}
		if decision.refusal == refusalCompareTargets {
			v.ShowToast(lang.L("Select exactly 2 images to compare"))
		}
		return decision, false
	}
	if decision.yieldRegion {
		v.cancelRegionCopy()
	}
	return decision, true
}

func (v *viewer) showSettings() {
	if _, ok := v.admitCommand(commandRequest{command: commandSettings}); !ok {
		return
	}
	v.settingsWin.Show(v.settingsState(), v.storeManaged)
}
