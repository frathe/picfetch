package ui

import (
	"errors"
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/grid"
	searchui "github.com/frathe/picfetch/internal/ui/visualsearch"
)

func (v *viewer) searchReference() string {
	if v.stopping || v.analysisMaintenanceBusy() || v.FileCount() == 0 || v.scanOp.active || v.sortOp.active || v.comparisonActive() || v.explorerMapActive() || v.slides.Active() || v.win.Canvas().Overlays().Top() != nil || v.deletion.Visible() || v.exportPrompt.Visible() {
		return ""
	}
	if _, editing := v.win.Canvas().Focused().(*widget.Entry); editing {
		return ""
	}
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
	imageOrder     []string
	pending        *searchDelivery
	revision       uint64
	applying       bool
	overlay        *searchOverlayWait
	overlayWorkers sync.WaitGroup
	overlayUI      searchui.UIQueue
}

func (v *viewer) searchActive() bool { return v.visualsearch != nil && v.visualsearch.Active() }

func (v *viewer) findMoreLikeThis() {
	reference := v.searchReference()
	if reference == "" || !v.yieldCopySelection() {
		return
	}
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
	if v.analysisMaintenanceBusy() {
		return
	}
	v.fileWork.searchLifecycle.invalidate()
	if v.searchActive() {
		v.visualsearch.SetCachePolicy(v.searchCachePolicy())
		if v.visualsearch.Explore(reference) {
			v.searchView.imageOrder = nil
			state := v.visualsearch.State()
			v.presentSearch(state.Visit)
		}
		return
	}
	var paths []string
	seen := map[string]bool{}
	for _, uri := range v.state.files {
		if uri != nil && !seen[uri.Path()] {
			seen[uri.Path()] = true
			paths = append(paths, uri.Path())
		}
	}
	origin := searchHost{v}.CaptureVisit()
	after := v.explorer.Suspend()
	v.explorer.Surface().Hide()
	v.searchView.imageOrder = nil
	policy := v.searchCachePolicy()
	if v.visualsearch.Start(searchui.StartRequest{Paths: paths, ReferencePath: reference, Origin: origin, Cache: policy, After: after}) {
		v.presentSearch(searchui.Visit{ReferencePath: reference})
	}
}

// A deferred restoration carries its saved Grid interaction, while progress
// remains owned by the live search session. Read progress at delivery time:
// preparation can complete while a popup is holding this visit.
type searchDelivery struct {
	visit   searchui.Visit
	restore bool
}

func (v *viewer) presentSearch(visit searchui.Visit) {
	v.applySearchDelivery(searchDelivery{visit: visit})
}

func (v *viewer) applySearchDelivery(delivery searchDelivery) {
	if v.searchView.applying || !v.searchActive() {
		return
	}
	v.searchView.applying = true
	defer func() { v.searchView.applying = false }()
	overlayOpen := v.win.Canvas().Overlays().Top() != nil
	if v.comparisonActive() || overlayOpen || v.deletion.Visible() || v.exportPrompt.Visible() || v.searchView.imageOrder != nil {
		v.searchView.pending = &delivery
		if overlayOpen {
			v.watchSearchOverlay()
		}
		return
	}
	v.stopSearchOverlayWait()
	v.searchView.pending = nil
	v.searchView.revision++
	visit := delivery.visit
	v.grid.OpenRanked(grid.RankedVisit{ReferencePath: visit.ReferencePath, Paths: visit.Paths, Revision: v.searchView.revision, Progress: v.visualsearch.State().Progress, Back: func() { v.visualsearch.Back() }, Exit: v.visualsearch.Exit, Save: v.saveSearchMatches})
	if delivery.restore {
		v.grid.RestoreVisit(visit.Grid)
	}
	v.ForceRepaint()
}

func (v *viewer) resetSearchPresentation() {
	v.stopSearchOverlayWait()
	v.searchView.imageOrder = nil
	v.searchView.pending = nil
}

func (v *viewer) restoreSearchVisit(visit searchui.Visit) {
	v.resetSearchPresentation()
	v.applySearchDelivery(searchDelivery{visit: visit, restore: true})
}

func (v *viewer) returnToSearchGrid() {
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
	v.resetSearchPresentation()
	if v.visualsearch != nil {
		v.visualsearch.Close()
	}
}
func (v *viewer) indexOfSearchOccurrence(path string, occurrence int) int {
	first := -1
	for i, uri := range v.state.files {
		if uri != nil && uri.Path() == path {
			if first < 0 {
				first = i
			}
			if occurrence == 0 {
				return i
			}
			occurrence--
		}
	}
	return first
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
	if uri, index, ok := h.v.CurrentFile(); ok {
		visit.ImagePath = uri.Path()
		visit.ImageOccurrence = 0
		for _, prior := range h.v.state.files[:index] {
			if prior != nil && prior.Path() == visit.ImagePath {
				visit.ImageOccurrence++
			}
		}
	}
	if !h.v.searchActive() {
		visit.Grid = h.v.grid.CaptureVisit()
	}
	return visit
}
func (h searchHost) Present(visit searchui.Visit, _ grid.Progress) {
	h.v.presentSearch(visit)
}
func (h searchHost) Restore(visit searchui.Visit, origin bool) {
	v := h.v
	v.resetSearchPresentation()
	if origin {
		v.fileWork.searchLifecycle.invalidate()
		v.grid.Close()
		if v.FileCount() == 0 {
			v.clearToDropzone()
			return
		}
		if visit.Grid.Visible {
			v.grid.RestoreVisit(visit.Grid)
			if visit.Grid.Subset != nil && v.explorer.HasCohort() {
				v.explorer.Surface().Show()
			}
		} else {
			i := v.indexOfSearchOccurrence(visit.ImagePath, visit.ImageOccurrence)
			if i < 0 {
				i = 0
			}
			v.ShowImage(i)
		}
	} else {
		v.restoreSearchVisit(visit)
	}
	v.syncMenus()
	v.ForceRepaint()
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
		if !h.v.visualsearch.State().Progress.Complete {
			h.v.ShowToast(lang.L("Analysis paused while cache space is freed. Start another search to continue."))
		}
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

// CurrentFiles captures ranked indexes once before the naming dialog opens.
func (h favoriteListHost) CurrentFiles() []fyne.URI {
	order := h.captureSearchOrder()
	if !order.active {
		return slices.Clone(h.state.files)
	}
	indexes := order.indexes
	files := make([]fyne.URI, len(indexes))
	for i, index := range indexes {
		files[i] = h.viewer.FileAt(index)
	}
	return files
}

func (v *viewer) searchImageOpened(visit grid.Visit) {
	v.visualsearch.CaptureGrid(visit)
	v.searchView.imageOrder = slices.Clone(visit.Results)
}

func (v *viewer) flushSearchPresentation() {
	if v.searchView.pending != nil && !v.searchView.applying && v.searchActive() {
		v.applySearchDelivery(*v.searchView.pending)
	}
}
