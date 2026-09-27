package ui

import (
	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/preferences"
	explorerui "github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/ui/grid"
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
	if _, ok := v.admitCommand(commandRequest{command: commandExplorer}); !ok {
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
	v.closeLocationMap()
	v.closeVisualSearch()
	v.grid.Close()
	v.browsing.enterExplorer(v.Generation())
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
			v.syncDuplicatePreparationProgress()
			return
		}
		v.explorerInput.prepare = nil
	}
	v.beginExplorerAnalysis()
}
func (v *viewer) beginExplorerAnalysis() {
	if _, ok := v.admitCommand(commandRequest{command: commandExplorer, route: routeDelivery}); !ok {
		v.closeExplorer()
		return
	}
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
func (v *viewer) openExplorerGrid(paths []string, unassigned bool) bool {
	if _, ok := v.admitCommand(commandRequest{command: commandExplorer, route: routeDelivery}); !ok {
		return false
	}
	v.grid.Close()
	binding := v.browsing.openExplorerCohort(v.Generation())
	v.presentExplorerGrid(binding, paths, unassigned, nil)
	return true
}

// presentExplorerGrid is the only Explorer bookmark restoration boundary.
// Reinstall live bindings before restoring interaction; saved callbacks are data
// from a retired presentation and never authorize a return.
func (v *viewer) presentExplorerGrid(binding browsingBinding, paths []string, unassigned bool, bookmark *grid.Visit) {
	back := func() { v.returnExplorerMap(binding) }
	if unassigned {
		v.grid.OpenUnassigned(paths, back, v.analyzeSimilaritySelection)
	} else {
		v.grid.OpenSubset(paths, back)
	}
	if bookmark != nil {
		v.grid.RestoreInteraction(*bookmark)
	}
	v.explorer.Surface().Show()
	v.syncMenus()
	v.ForceRepaint()
}

func (v *viewer) returnExplorerGrid() {
	v.restoreExplorerGrid(nil)
}

func (v *viewer) restoreExplorerGrid(bookmark *grid.Visit) {
	plan, ok := v.browsing.planReturn(v.browsing.current().binding, v.Generation(), browsingReturnGrid)
	if !ok {
		return
	}
	if _, admitted := v.admitCommand(commandRequest{command: commandGrid, route: routeDelivery}); !admitted {
		return
	}
	if !v.browsing.commitReturn(plan, v.Generation()) {
		return
	}
	paths, unassigned := v.explorer.Cohort()
	if bookmark != nil {
		plan.grid = bookmark
	}
	v.presentExplorerGrid(plan.source, paths, unassigned, plan.grid)
}

// captureExplorerReconciliation keeps saved occurrence interaction independent
// of index shifts. Capturing precedes collection publication; restoring uses the
// rebound owner and fresh callbacks, never the bookmark's old closures.
func (v *viewer) captureExplorerReconciliation(removed []int) func() {
	if !v.browsing.has(browsingExplorerMap) {
		return func() {}
	}
	var bookmark *grid.Visit
	if !v.searchActive() && v.browsing.has(browsingExplorer) && v.grid.Visible() {
		captured := v.grid.CaptureVisit()
		bookmark = &captured
	}
	var survivors map[fileidentity.Occurrence]fileidentity.Occurrence
	if len(removed) > 0 {
		survivors = make(map[fileidentity.Occurrence]fileidentity.Occurrence, v.FileCount())
		before, after := map[string]int{}, map[string]int{}
		nextRemoved := 0
		for i, source := range v.state.files {
			path := source.Path()
			old := fileidentity.Occurrence{Path: path, Ordinal: before[path]}
			before[path]++
			if nextRemoved < len(removed) && removed[nextRemoved] == i {
				nextRemoved++
				continue
			}
			survivors[old] = fileidentity.Occurrence{Path: path, Ordinal: after[path]}
			after[path]++
		}
	}
	return func() {
		v.browsing.reconcile(v.Generation(), survivors)
		if bookmark != nil {
			if survivors != nil {
				remapped := bookmark.RemapOccurrences(survivors)
				bookmark = &remapped
			}
			v.restoreExplorerGrid(bookmark)
		}
	}
}

func (v *viewer) returnExplorerMap(binding browsingBinding) {
	plan, ok := v.browsing.planReturn(binding, v.Generation(), browsingReturnParent)
	if !ok {
		return
	}
	if _, admitted := v.admitCommand(commandRequest{command: commandExplorer, route: routeDelivery}); !admitted {
		return
	}
	if !v.browsing.commitReturn(plan, v.Generation()) {
		return
	}
	v.grid.Close()
	v.explorer.Surface().Show()
	v.syncMenus()
	v.ForceRepaint()
	v.recordExplorerView("map-return")
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
	v.browsing.leaveExplorer()
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
func (v *viewer) explorerGridChanged() {
	v.syncMenus()
}
func (v *viewer) explorerCanRetry() bool {
	return v.explorer.State().CanRetry && v.explorerInput.prepare == nil
}
func (v *viewer) explorerMapActive() bool {
	return v.browsing.current().binding.kind == browsingExplorerMap && !v.searchActive() && !v.grid.Visible()
}

func (v *viewer) browsingImageOpened(bookmark grid.Visit) {
	if kind := v.browsing.current().binding.kind; kind == browsingExplorer || kind == browsingCluster {
		v.browsing.openImage(v.browsing.current().binding, v.Generation(), bookmark)
	}
}

func (v *viewer) explorerImageOpened() {
	if v.browsing.has(browsingExplorer) && !v.searchActive() {
		v.explorer.Surface().Hide()
	}
}
func (v *viewer) explorerKey(key fyne.KeyName) bool {
	if v.explorerMapActive() {
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
	if v.browsing.has(browsingExplorer) && (key == fyne.KeyEscape || key == fyne.KeyG) {
		v.returnExplorerGrid()
		return true
	}
	return false
}

func (v *viewer) cohortIndexes() []int {
	return v.captureBrowsingScope().indexes
}

func (v *viewer) backToSimilarityMap()           { v.returnExplorerMap(v.browsing.current().binding) }
func (v *viewer) recordExplorerView(kind string) { v.explorer.RecordView(kind) }

type explorerHost struct{ v *viewer }

func (h explorerHost) Window() fyne.Window { return h.v.win }
func (h explorerHost) Changed()            { h.v.syncMenus() }
func (h explorerHost) BrowseCohort(paths []string, unassigned bool) bool {
	return h.v.openExplorerGrid(paths, unassigned)
}
func (h explorerHost) LeaveExplorer()              { h.v.LeaveSimilarityMap() }
func (h explorerHost) ReturnToMap()                { h.v.backToSimilarityMap() }
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
	case v.browsing.current().surface == browsingImage || v.searchActive():
		surface = "image"
		if uri, _, ok := v.CurrentFile(); ok {
			paths = append(paths, uri.Path())
		}
	}

	return explorerui.Presentation{Surface: surface, Paths: paths, GridVisible: v.grid.Visible(), ComparisonActive: v.comparisonActive()}
}

func (h explorerHost) Repaint() { h.v.ForceRepaint() }
