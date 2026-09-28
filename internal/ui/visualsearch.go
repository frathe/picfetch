package ui

import (
	"errors"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/grid"
	searchui "github.com/frathe/picfetch/internal/ui/visualsearch"
)

func (v *viewer) searchReference() string {
	if !v.queryCommand(commandRequest{command: commandSearch}).allowed {
		return ""
	}
	return v.searchTarget()
}

// searchTarget resolves only the prospective source. Cross-feature admission
// is decided before the caller captures that source for a search session.
func (v *viewer) searchTarget() string {
	if v.grid.Visible() {
		if v.grid.SelectionCount() > 1 {
			return ""
		}
		if targets := v.grid.Targets(); len(targets) == 1 && targets[0] >= 0 && targets[0] < v.FileCount() {
			return v.FileAt(targets[0]).Path()
		}
		return ""
	}
	if source, ok := v.DisplayedFile(); ok {
		return source.Path()
	}
	return ""
}

type searchPresentation struct {
	pending        *searchDelivery
	revision       uint64
	applying       bool
	overlay        *searchOverlayWait
	overlayWorkers sync.WaitGroup
	overlayUI      searchui.UIQueue
}

func (v *viewer) searchActive() bool { return v.browsing.current().binding.kind == browsingSearch }

func (v *viewer) findMoreLikeThis() {
	if _, ok := v.admitCommand(commandRequest{command: commandSearch}); !ok {
		return
	}
	reference := v.searchTarget()
	generation := v.Generation()
	if !v.explorer.EnsureReady(func() {
		preferences.Save(v.app, v.currentPreferences())
		if generation == v.Generation() && v.searchReference() == reference {
			v.startVisualSearch(reference)
		}
	}) {
		return
	}
	v.startVisualSearch(reference)
}
func (v *viewer) startVisualSearch(reference string) {
	if _, ok := v.admitCommand(commandRequest{command: commandSearch, route: routeDelivery}); !ok {
		return
	}
	v.fileWork.searchLifecycle.invalidate()
	if v.searchActive() {
		v.visualsearch.SetCachePolicy(v.searchCachePolicy())
		if v.visualsearch.Explore(reference) {
			state := v.visualsearch.State()
			v.restoreSearchVisit(state.Visit)
		}
		return
	}
	var paths []string
	seen := map[string]bool{}
	collection := v.state.Observe()
	for i := range collection.Count() {
		uri := collection.FileAt(i)
		if uri != nil && !seen[uri.Path()] {
			seen[uri.Path()] = true
			paths = append(paths, uri.Path())
		}
	}
	origin := searchHost{v}.CaptureVisit()
	if _, entered := v.browsing.enterSearch(v.Generation(), browsingOrigin{grid: origin.Grid, image: origin.Image}); !entered {
		return
	}
	after := v.explorer.Suspend()
	v.explorer.Surface().Hide()
	policy := v.searchCachePolicy()
	if v.visualsearch.Start(searchui.StartRequest{Paths: paths, ReferencePath: reference, Cache: policy, After: after}) {
		v.presentSearch(searchui.Visit{ReferencePath: reference})
	} else {
		v.browsing.detachSearch()
	}
}

// A deferred restoration carries its saved Grid interaction, while progress
// remains owned by the live search session. Read progress at delivery time:
// preparation can complete while a popup is holding this visit.
type searchDelivery struct {
	visit    searchui.Visit
	restore  bool
	binding  browsingBinding
	revision uint64
}

func (v *viewer) presentSearch(visit searchui.Visit) {
	v.applySearchDelivery(searchDelivery{visit: visit, binding: v.browsing.current().binding, revision: v.browsing.revision})
}

func (v *viewer) applySearchDelivery(delivery searchDelivery) {
	if v.searchView.applying || !v.searchActive() || !v.browsing.matches(delivery.binding, v.Generation()) || delivery.revision != v.browsing.revision {
		return
	}
	v.searchView.applying = true
	defer func() { v.searchView.applying = false }()
	overlayOpen := v.win.Canvas().Overlays().Top() != nil
	if v.comparisonActive() || overlayOpen || v.deletion.Visible() || v.exportPrompt.Visible() || v.regionCopy.State().Active ||
		v.browsing.current().surface == browsingImage && !delivery.restore || !v.queryCommand(commandRequest{command: commandGrid, route: routeDelivery}).allowed {
		v.searchView.pending = &delivery
		if overlayOpen {
			v.watchSearchOverlay()
		}
		return
	}
	if delivery.restore {
		plan, ok := v.browsing.planReturn(delivery.binding, v.Generation(), browsingReturnGrid)
		if !ok || !v.browsing.commitReturn(plan, v.Generation()) {
			return
		}
	}
	v.stopSearchOverlayWait()
	v.searchView.pending = nil
	v.searchView.revision++
	visit := delivery.visit
	binding := v.browsing.current().binding
	v.grid.OpenRanked(grid.RankedVisit{ReferencePath: visit.ReferencePath, Paths: visit.Paths, Revision: v.searchView.revision, Progress: v.visualsearch.State().Progress,
		Back: func() {
			if v.browsing.matches(binding, v.Generation()) {
				v.visualsearch.Back()
			}
		}, Exit: func() { v.exitVisualSearch(binding) }, Save: v.saveSearchMatches})
	if delivery.restore {
		v.grid.RestoreInteraction(visit.Grid)
	}
	v.ForceRepaint()
}

func (v *viewer) resetSearchPresentation() {
	v.stopSearchOverlayWait()
	v.searchView.pending = nil
}

func (v *viewer) restoreSearchVisit(visit searchui.Visit) {
	v.applySearchDelivery(searchDelivery{visit: visit, restore: true, binding: v.browsing.current().binding, revision: v.browsing.revision})
}

func (v *viewer) returnToSearchGrid() {
	if _, ok := v.admitCommand(commandRequest{command: commandGrid, route: routeDelivery}); !ok {
		return
	}
	state := v.visualsearch.State()
	v.restoreSearchVisit(state.Visit)
}
func (v *viewer) searchKey(key fyne.KeyName) bool {
	if !v.searchActive() {
		return false
	}
	if !v.grid.Visible() && (key == fyne.KeyEscape || key == fyne.KeyG) {
		v.returnToSearchGrid()
		return true
	}
	if v.grid.Visible() && (key == fyne.KeyG || key == fyne.KeyV) && !v.grid.Searching() {
		v.visualsearch.Exit()
		return true
	}
	return false
}
func (v *viewer) closeVisualSearch() {
	v.fileWork.searchLifecycle.invalidate()
	v.browsing.detachSearch()
	v.resetSearchPresentation()
	if v.visualsearch != nil {
		v.visualsearch.Close()
	}
}

func (v *viewer) saveSearchMatches() {
	if !v.searchActive() || !v.grid.Visible() {
		return
	}
	selected := v.grid.Selection()
	wanted := map[int]bool{}
	for _, i := range selected {
		wanted[i] = true
	}
	var files []fyne.URI
	for _, i := range v.grid.ResultIndexes() {
		if len(selected) == 0 || wanted[i] {
			files = append(files, v.FileAt(i))
		}
	}
	v.favorites.AddFiles(files)
}

type searchHost struct{ v *viewer }

func (h searchHost) CaptureVisit() searchui.Visit {
	visit := searchui.Visit{}
	if h.v.searchActive() {
		visit = h.v.visualsearch.State().Visit
	}
	if h.v.grid.Visible() {
		visit.Grid = h.v.grid.CaptureVisit()
	}
	visit.Image = h.v.currentImageOccurrence()
	if !h.v.searchActive() {
		visit.Grid = h.v.grid.CaptureVisit()
	}
	return visit
}
func (h searchHost) Present(visit searchui.Visit, _ grid.Progress) {
	h.v.presentSearch(visit)
}
func (h searchHost) Restore(visit searchui.Visit) {
	v := h.v
	v.restoreSearchVisit(visit)
	v.syncMenus()
	v.ForceRepaint()
}

func (h searchHost) LeaveSearch() { h.v.exitVisualSearch(h.v.browsing.current().binding) }

func (v *viewer) exitVisualSearch(binding browsingBinding) {
	plan, ok := v.browsing.planReturn(binding, v.Generation(), browsingReturnParent)
	if !ok || binding.kind != browsingSearch {
		return
	}
	if _, admitted := v.admitCommand(commandRequest{command: commandViewer, route: routeDelivery}); !admitted {
		return
	}
	if !v.browsing.commitReturn(plan, v.Generation()) {
		return
	}
	v.visualsearch.Detach()
	v.resetSearchPresentation()
	if i := v.restoreBrowsingOrigin(*plan.origin); i >= 0 {
		v.loadImage(i)
	}
	v.syncMenus()
	v.ForceRepaint()
}

func (v *viewer) detachSearchOrigin() (browsingOrigin, bool) {
	origin, active := v.browsing.detachSearch()
	if !active {
		return browsingOrigin{}, false
	}
	v.visualsearch.Detach()
	v.resetSearchPresentation()
	return origin, true
}

func (h searchHost) Changed() {
	v := h.v
	if v.searchActive() {
		v.grid.SetRankedProgress(v.visualsearch.State().Progress)
	}
	v.syncMenus()
}
func (h searchHost) Failed(err error) {
	var pressure similarity.CachePressureError
	if errors.As(err, &pressure) {
		h.v.analysisCache.SetRoots(h.v.analysisRoots())
		h.v.analysisCache.MakeRoom(pressure.NeedBytes)
		return
	}
	fyne.LogError("visual search", err)
	h.v.ShowToast(lang.L("Could not search this image. Choose another reference or try again."))
	var terminal searchui.SessionError
	if errors.As(err, &terminal) {
		h.v.reconcileSearchOrigin()
	}
}

type favoriteListHost struct{ *viewer }

func (h favoriteListHost) SyncFavoritePreviews(owner *favstore.Owner, files []fyne.URI) {
	h.syncCapturedFavoritePreviews(owner, files)
}

// OpenFavorite receives the owner captured by the complete storage read. Root
// keeps preview policy and ordinary collection replay in their existing paths.
func (h favoriteListHost) OpenFavorite(owner *favstore.Owner, files []fyne.URI) {
	h.syncCapturedFavoritePreviews(owner, files)
	h.viewer.OpenFavorite(owner.Path(), files)
}

// CurrentFiles captures ranked indexes once before the naming dialog opens.
func (h favoriteListHost) CurrentFiles() []fyne.URI {
	if !h.browsingContext().ranked {
		return h.state.Observe().Capture(collectionDisplayOrder)
	}
	indexes := h.captureBrowsingScope().indexes
	collection := h.state.Observe()
	files := make([]fyne.URI, len(indexes))
	for i, index := range indexes {
		files[i] = collection.FileAt(index)
	}
	return files
}

func (v *viewer) searchImageOpened(visit grid.Visit) {
	if v.browsing.openImage(v.browsing.current().binding, v.Generation(), visit) {
		v.visualsearch.CaptureGrid(visit)
	}
}

func (v *viewer) flushSearchPresentation() {
	if v.searchView.pending != nil && !v.searchView.applying && v.searchActive() {
		v.applySearchDelivery(*v.searchView.pending)
	}
}
