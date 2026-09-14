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
	pending        *searchui.Visit
	progress       grid.Progress
	revision       uint64
	applying       bool
	overlay        *searchOverlayWait
	overlayWorkers sync.WaitGroup
	overlayUI      searchui.UIQueue
}

func (v *viewer) searchActive() bool { return v.visualsearch != nil && v.visualsearch.State().Active }

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
	if v.searchActive() {
		v.visualsearch.SetCachePolicy(v.searchCachePolicy())
		if v.visualsearch.Explore(reference) {
			v.searchView.imageOrder = nil
			v.presentSearch(v.visualsearch.State().Visit, v.visualsearch.State().Progress)
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
		v.presentSearch(searchui.Visit{ReferencePath: reference}, grid.Progress{Total: len(paths)})
	}
}
func (v *viewer) presentSearch(visit searchui.Visit, progress grid.Progress) {
	if v.searchView.applying {
		return
	}
	v.searchView.applying = true
	defer func() { v.searchView.applying = false }()
	v.searchView.progress = progress
	if !v.searchActive() {
		return
	}
	overlayOpen := v.win.Canvas().Overlays().Top() != nil
	if v.comparisonActive() || overlayOpen || v.deletion.Visible() || v.exportPrompt.Visible() || v.searchView.imageOrder != nil {
		v.searchView.pending = &visit
		if overlayOpen {
			v.watchSearchOverlay()
		}
		return
	}
	v.stopSearchOverlayWait()
	v.searchView.pending = nil
	v.searchView.revision++
	v.grid.OpenRanked(grid.RankedVisit{ReferencePath: visit.ReferencePath, Paths: visit.Paths, Revision: v.searchView.revision, Progress: progress, Back: func() { v.visualsearch.Back() }, Exit: v.visualsearch.Exit, Save: v.saveSearchMatches})
	v.ForceRepaint()
}
func (v *viewer) returnToSearchGrid() {
	v.searchView.imageOrder = nil
	state := v.visualsearch.State()
	v.presentSearch(state.Visit, state.Progress)
	v.grid.RestoreVisit(state.Visit.Grid)
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
	v.stopSearchOverlayWait()
	if v.visualsearch != nil {
		v.visualsearch.Close()
	}
	v.searchView.imageOrder = nil
	v.searchView.pending = nil
}
func (v *viewer) activeSearchIndexes() []int {
	if !v.searchActive() {
		return nil
	}
	if v.grid.Visible() {
		return v.grid.ResultIndexes()
	}
	paths := v.searchView.imageOrder
	if paths == nil {
		paths = v.visualsearch.State().Visit.Paths
	}
	if len(paths) == 0 {
		return nil
	}
	byPath := make(map[string]int, len(paths))
	for _, path := range paths {
		byPath[path] = -1
	}
	remaining := len(byPath)
	for i, uri := range v.state.files {
		if uri == nil {
			continue
		}
		path := uri.Path()
		if index, wanted := byPath[path]; wanted && index < 0 {
			byPath[path] = i
			remaining--
			if remaining == 0 {
				break
			}
		}
	}
	indexes := make([]int, 0, len(paths))
	for _, path := range paths {
		if i := byPath[path]; i >= 0 {
			indexes = append(indexes, i)
		}
	}
	return indexes
}
func (v *viewer) indexOfSearchPath(path string) int {
	for i, uri := range v.state.files {
		if uri != nil && uri.Path() == path {
			return i
		}
	}
	return -1
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
	if uri, _, ok := h.v.CurrentFile(); ok {
		visit.ImagePath = uri.Path()
	}
	if !h.v.searchActive() {
		visit.Grid = h.v.grid.CaptureVisit()
	}
	return visit
}
func (h searchHost) Present(visit searchui.Visit, progress grid.Progress) {
	h.v.presentSearch(visit, progress)
}
func (h searchHost) Restore(visit searchui.Visit, origin bool) {
	v := h.v
	v.stopSearchOverlayWait()
	v.searchView.imageOrder = nil
	v.searchView.pending = nil
	if origin {
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
			i := v.indexOfSearchPath(visit.ImagePath)
			if i < 0 {
				i = 0
			}
			v.ShowImage(i)
		}
	} else {
		v.presentSearch(visit, v.visualsearch.State().Progress)
		v.grid.RestoreVisit(visit.Grid)
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
		h.v.ShowToast(lang.L("Analysis paused while cache space is freed. Start another search to continue."))
		return
	}
	fyne.LogError("visual search", err)
	h.v.ShowToast(lang.L("Could not search this image. Choose another reference or try again."))
	var terminal searchui.SessionError
	if errors.As(err, &terminal) {
		h.v.visualsearch.Exit()
	}
}

type favoriteListHost struct{ *viewer }

// CurrentFiles captures ranked indexes once before the naming dialog opens.
func (h favoriteListHost) CurrentFiles() []fyne.URI {
	if !h.searchActive() {
		return slices.Clone(h.state.files)
	}
	indexes := h.activeSearchIndexes()
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
	if v.searchView.pending != nil && !v.searchView.applying && v.grid != nil && v.grid.Visible() && v.searchActive() {
		v.presentSearch(*v.searchView.pending, v.searchView.progress)
	}
}
