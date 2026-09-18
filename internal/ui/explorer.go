package ui

import (
	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/preferences"
	explorerui "github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/winpos"
)

// explorerInput contains only root-owned collection, launch and window policy.
type explorerInput struct {
	prepare                  func()
	prepareOp                requestLifecycle
	favoriteDir              string
	pendingLaunch, maximized bool
}

func (v *viewer) showExplorer() {
	// The previous collection stays installed until replacement sorting commits.
	if v.scanOp.active || v.sortOp.active || v.analysisMaintenanceBusy() {
		return
	}
	if v.stopping || v.comparisonActive() || v.explorerMapActive() && !v.explorerCanRetry() || v.FileCount() == 0 || !v.yieldCopySelection() {
		return
	}
	defer v.syncMenus()
	if !v.explorer.EnsureReady(func() { preferences.Save(v.app, v.currentPreferences()); v.showExplorer() }) {
		return
	}
	if v.slides.Active() {
		v.slides.Exit()
		v.resetFade()
	}
	v.explorer.WaitBefore(v.visualsearch.Suspend())
	v.closeVisualSearch()
	v.grid.Close()
	winpos.Maximize(v.win)
	v.explorerInput.maximized = true
	if v.dupes.HideDuplicates() {
		token := v.explorerInput.prepareOp.begin()
		v.explorerInput.prepare = func() {
			if token.current() {
				v.beginExplorerAnalysis()
			}
		}
		if !v.grid.PrepareDuplicateGroups() {
			v.explorer.Preparing()
			return
		}
		v.explorerInput.prepare = nil
	}
	v.beginExplorerAnalysis()
}
func (v *viewer) beginExplorerAnalysis() {
	paths := make([]string, 0, v.FileCount())
	visibility := v.dupes.Visibility()
	for i := range v.FileCount() {
		if visibility.Visible(i) {
			paths = append(paths, v.FileAt(i).Path())
		}
	}
	request := explorerui.OpenRequest{Sources: paths, FavoriteDir: v.explorerInput.favoriteDir, FavoritesDir: v.favorites.Dir()}
	cache := v.searchCachePolicy()
	if cache.LooseEnabled {
		request.GeneralAnalysisDir, request.GeneralAnalysisLimitBytes = cache.Roots.GeneralDir, cache.GeneralLimitBytes
	}
	v.explorer.Open(request)
}
func (v *viewer) UpdateSimilarityMap()                    { v.explorer.UpdateSimilarityMap() }
func (v *viewer) SetSimilarityAutoUpdate(on bool)         { v.explorer.SetSimilarityAutoUpdate(on) }
func (v *viewer) ShowSimilarityPresets()                  { v.explorer.ShowSimilarityPresets() }
func (v *viewer) OpenSimilarityCohort(paths []string)     { v.explorer.OpenSimilarityCohort(paths) }
func (v *viewer) OpenSimilarityUnassigned(paths []string) { v.explorer.OpenSimilarityUnassigned(paths) }
func (v *viewer) analyzeSimilaritySelection() {
	if v.comparisonActive() || !v.grid.Visible() || v.grid.SelectionCount() < 2 {
		return
	}
	var paths []string
	for _, i := range v.grid.Selection() {
		if i >= 0 && i < v.FileCount() {
			paths = append(paths, v.FileAt(i).Path())
		}
	}
	v.explorer.AnalyzeSelection(paths)
}
func (v *viewer) openExplorerGrid(paths []string, unassigned bool) {
	if unassigned {
		v.grid.OpenUnassigned(paths, v.backToSimilarityMap, v.analyzeSimilaritySelection)
	} else {
		v.grid.OpenSubset(paths, v.backToSimilarityMap)
	}
}

func (v *viewer) LeaveSimilarityMap() {
	v.closeExplorer()
	v.grid.Close()
	v.syncMenus()
	if v.FileCount() > 0 && v.img.Image == nil {
		v.ShowImage(v.state.index)
	}
}
func (v *viewer) cancelExplorerPreparation() {
	v.explorerInput.prepareOp.invalidate()
	if v.explorerInput.prepare != nil {
		v.explorerInput.prepare = nil
		v.grid.Close()
	}
}
func (v *viewer) closeExplorer() {
	v.cancelExplorerPreparation()
	v.explorer.Close()
}
func (v *viewer) settleExplorer() {
	for {
		v.grid.Settle()
		if !v.explorer.Settle() {
			return
		}
	}
}
func (v *viewer) explorerGridChanged() { v.syncMenus() }
func (v *viewer) explorerCanRetry() bool {
	return v.explorer.State().CanRetry && v.explorerInput.prepare == nil
}
func (v *viewer) explorerMapActive() bool {
	return v.explorer != nil && v.explorer.Surface().Visible() && !v.grid.Visible()
}
func (v *viewer) explorerImageOpened() {
	if v.explorer.HasCohort() {
		v.explorer.Surface().Hide()
	}
}
func (v *viewer) explorerKey(key fyne.KeyName) bool {
	if v.explorer.Surface().Visible() {
		switch key {
		case fyne.KeyEscape, fyne.KeyV:
			v.LeaveSimilarityMap()
		case fyne.KeyF1:
			v.help.ShowManual()
		default:
			v.explorer.Surface().HandleKey(key)
		}
		return true
	}
	if v.explorer.HasCohort() && (key == fyne.KeyEscape || key == fyne.KeyG) {
		v.openExplorerGrid(v.explorer.Cohort())
		v.explorer.Surface().Show()
		v.ForceRepaint()
		return true
	}
	return false
}

func (v *viewer) cohortIndexes() []int {
	if order := v.captureSearchOrder(); order.active {
		return order.indexes
	}
	paths, _ := v.explorer.Cohort()
	if len(paths) == 0 {
		return nil
	}
	members := make(map[string]bool, len(paths))
	for _, path := range paths {
		members[path] = true
	}
	var indexes []int
	for i := range v.FileCount() {
		if members[v.FileAt(i).Path()] {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func (v *viewer) backToSimilarityMap()           { v.explorer.ReturnToMap() }
func (v *viewer) recordExplorerView(kind string) { v.explorer.RecordView(kind) }

type explorerHost struct{ v *viewer }

func (h explorerHost) Window() fyne.Window { return h.v.win }
func (h explorerHost) Changed()            { h.v.syncMenus() }
func (h explorerHost) BrowseCohort(paths []string, unassigned bool) {
	h.v.openExplorerGrid(paths, unassigned)
}
func (h explorerHost) LeaveExplorer()              { h.v.LeaveSimilarityMap() }
func (h explorerHost) ReturnToMap()                { h.v.grid.Close() }
func (h explorerHost) ShowToast(message string)    { h.v.ShowToast(message) }
func (h explorerHost) Unfocus()                    { h.v.Unfocus() }
func (h explorerHost) Modifiers() fyne.KeyModifier { return h.v.Modifiers() }
func (h explorerHost) Presentation() explorerui.Presentation {
	v := h.v
	var paths []string
	surface := "map"
	switch {
	case v.win.Canvas().Overlays().Top() != nil:
		surface = "overlay"
	case v.comparisonActive():
		surface = "comparison"
	case v.grid.Visible():
		surface = "grid"
		for _, i := range v.grid.ResultIndexes() {
			paths = append(paths, v.FileAt(i).Path())
		}
	case !v.explorer.Surface().Visible():
		surface = "image"
		if uri, _, ok := v.CurrentFile(); ok {
			paths = append(paths, uri.Path())
		}
	}

	return explorerui.Presentation{Surface: surface, Paths: paths, GridVisible: v.grid.Visible(), ComparisonActive: v.comparisonActive()}
}

func (h explorerHost) Repaint() { h.v.ForceRepaint() }
