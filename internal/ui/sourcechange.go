package ui

import (
	"slices"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/ui/grid"
)

type sourceChangeKind uint8

const (
	sourcesRemoved sourceChangeKind = iota
	sourceLoadFailed
	sourcesRevalidated
	sourceWritten
	analysisPolicyChanged
	duplicatePolicyChanged
)

// sourceChange describes an admitted change. Callers report the fact; this
// boundary owns its ordering relative to analysis, Grid and origin restoration.
type sourceChange struct {
	kind    sourceChangeKind
	removed []int
	written []fyne.URI
}

// Reorder retains collection facts but retires generation-bound producers and
// reconciles retained visits before choosing the one authoritative image load.
func (v *viewer) commitCollectionReorder(ordered []fyne.URI) {
	finishUpdate := v.beginBrowsingUpdate()
	browsing := v.captureBrowsingReconciliation(nil)
	change := v.state.Reorder(ordered)
	v.browsing.reconcile(change.after.Generation(), browsing.survivors)
	v.grid.FilesChanged()
	index := v.finishBrowsingReconciliation(browsing)
	if index < 0 {
		if current, ok := change.after.Bookmark(change.after.index); ok {
			if candidate, eligible := v.captureBrowsingScope().RestoreImage(current.occurrence, change.after.Occurrences()); eligible {
				index = candidate
			}
		}
	}
	if index < 0 {
		v.invalidateLoad()
	}
	v.ForceRepaint()
	finishUpdate()
	if index >= 0 {
		v.loadImage(index)
	}
}

// Open preparation has already retired its feature surfaces. Commit complete
// collection facts before rebinding derived readers or admitting display work.
func (v *viewer) commitOpenedCollection(input collectionInput, merging bool, present func()) {
	finishUpdate := v.beginBrowsingUpdate()
	v.closeVisualSearch()
	v.grid.Close()
	var change collectionChange
	if merging {
		change = v.state.Merge(input)
	} else {
		change = v.state.Replace(input)
	}
	if !merging {
		v.dupes.WipeIfStale()
	}
	v.browsing.rebind(change.after.Generation())
	// The closed Grid rebuilds its generation-bound indexes on next entry.
	// FilesChanged would start fresh duplicate hashing behind a closed surface.
	v.locationMap.SetSources(change.after.DisplayFiles())
	v.ForceRepaint()
	finishUpdate()
	// Display admission must be last: startup tests use Fyne's inline worker
	// delivery, so neither repaint nor deferred menu publication may follow it.
	present()
}

// reconcileSources returns an image requested by origin restoration, or -1.
// A failed load consumes that index through display's existing retry chain;
// all other changes admit any required load here, after reconciliation.
func (v *viewer) reconcileSources(change sourceChange) int {
	indices := slices.Sorted(slices.Values(change.removed))
	indices = slices.Compact(slices.DeleteFunc(indices, func(i int) bool {
		return i < 0 || i >= len(v.state.files)
	}))
	if len(indices) == 0 && (change.kind == sourcesRemoved || change.kind == sourceLoadFailed) {
		return -1
	}
	defer v.beginBrowsingUpdate()()
	browsing := v.captureBrowsingReconciliation(indices)
	if change.kind == sourceWritten {
		v.locationMap.InvalidateSources(change.written)
	}

	restore := browsing.origin != nil
	// Closing comparison can deliver a pending ranking. Detach search first,
	// before any callback can expose the collection being reconciled.
	if v.comparisonActive() && (len(indices) > 0 || change.kind == sourcesRevalidated || restore && !browsing.origin.grid.Visible) {
		v.compare.Close()
	}
	if change.kind == sourcesRevalidated || change.kind == sourceWritten {
		v.favThumbLifecycle.invalidate()
	}
	if change.kind == sourcesRevalidated {
		v.imgCache.Purge()
	}
	for _, i := range slices.Backward(indices) {
		v.invalidateSort()
		removed := v.state.removeFile(i)
		if v.state.snapshot().IndexOf(removed.String()) < 0 {
			v.explorer.RemoveCohortSource(removed.Path())
		}
	}
	v.browsing.reconcile(v.Generation(), browsing.survivors)
	v.cancelExplorerPreparation()
	v.explorer.SourcesChanged()

	switch change.kind {
	case sourcesRemoved, sourceLoadFailed, sourcesRevalidated:
		v.grid.FilesChanged()
		if len(v.state.files) == 0 {
			v.grid.Close()
		}
	case duplicatePolicyChanged:
		v.grid.DuplicateDistanceChanged()
	case sourceWritten, analysisPolicyChanged:
		// Source positions are unchanged; retain Grid's interaction indexes.
	}
	if change.kind == sourcesRevalidated || change.kind == sourceWritten {
		v.grid.InvalidateContent()
	}
	if change.kind == sourceWritten {
		v.compare.Refresh()
	}
	index := v.finishBrowsingReconciliation(browsing)
	if change.kind == sourcesRevalidated && index < 0 && v.FileCount() > 0 {
		scope := v.captureBrowsingScope()
		if !scope.restricted || slices.Contains(scope.indexes, v.state.index) {
			index = v.state.index
		}
	}
	if index >= 0 && change.kind != sourceLoadFailed {
		v.loadImage(index)
	}
	if restore {
		v.syncMenus()
		v.ForceRepaint()
	}
	return index
}

type browsingReconciliation struct {
	survivors map[fileidentity.Occurrence]fileidentity.Occurrence
	grid      *grid.Visit
	origin    *browsingOrigin
}

// Capture before mutation and detach search before any feature callback. The
// occurrence map is shared by every retained visit and the detached origin.
func (v *viewer) captureBrowsingReconciliation(removed []int) browsingReconciliation {
	change := browsingReconciliation{}
	if len(removed) > 0 {
		change.survivors = make(map[fileidentity.Occurrence]fileidentity.Occurrence, v.FileCount())
		before, after := map[string]int{}, map[string]int{}
		for i, source := range v.state.files {
			path := source.Path()
			old := fileidentity.Occurrence{Path: path, Ordinal: before[path]}
			before[path]++
			if _, deleted := slices.BinarySearch(removed, i); deleted {
				continue
			}
			change.survivors[old] = fileidentity.Occurrence{Path: path, Ordinal: after[path]}
			after[path]++
		}
	}
	kind := v.browsing.current().binding.kind
	if (kind == browsingExplorer || kind == browsingCluster) && v.grid.Visible() {
		bookmark := v.grid.CaptureVisit()
		if change.survivors != nil {
			bookmark = bookmark.RemapOccurrences(change.survivors)
		}
		change.grid = &bookmark
	}
	if v.locationVisitActive() {
		v.locationInput.prepareOp.invalidate()
		v.locationInput.prepare = nil
	}
	if origin, detached := v.detachSearchOrigin(); detached {
		remapped := origin.remap(change.survivors)
		change.origin = &remapped
	}
	return change
}

// Collection and owner rebinding precede this call. Reconcile feature facts and
// current Grid bindings before publishing any restored origin or parent.
func (v *viewer) finishBrowsingReconciliation(change browsingReconciliation) int {
	if change.grid != nil && v.FileCount() > 0 {
		switch visit := v.browsing.current(); visit.binding.kind {
		case browsingExplorer:
			paths, unassigned := v.explorer.Cohort()
			v.presentExplorerGrid(visit.binding, paths, unassigned, change.grid)
		case browsingCluster:
			v.presentLocationGrid(change.grid)
		default:
			// Other visits have no retained subset Grid to refresh.
		}
	}
	// Rebuild retires validation; only the new source generation can return.
	v.rebuildLocationMap()
	if change.origin != nil {
		return v.restoreBrowsingOrigin(*change.origin)
	}
	v.returnExhaustedBrowsingScope(v.captureBrowsingScope())
	return -1
}

func (v *viewer) returnExhaustedBrowsingScope(scope browsingScope) {
	if !scope.restricted || !scope.complete || len(scope.indexes) != 0 {
		return
	}
	switch scope.binding.kind {
	case browsingExplorer:
		v.returnExplorerMap(scope.binding)
	case browsingCluster, browsingLocation:
		v.returnLocationMapFrom(scope.binding)
	default:
		return
	}
}

// restoreBrowsingOrigin restores interaction and selects an image without
// starting a load. Its caller owns either fresh admission or a display retry.
func (v *viewer) restoreBrowsingOrigin(origin browsingOrigin) int {
	v.fileWork.searchLifecycle.invalidate()
	v.grid.Close()
	if v.FileCount() == 0 {
		v.clearToDropzone()
		return -1
	}
	scope := v.captureBrowsingScope()
	if scope.restricted && len(scope.indexes) == 0 {
		v.returnExhaustedBrowsingScope(scope)
		return -1
	}
	if origin.grid.Visible {
		if origin.grid.Subset != nil && v.browsing.has(browsingExplorer) {
			paths, unassigned := v.explorer.Cohort()
			v.presentExplorerGrid(v.browsing.current().binding, paths, unassigned, &origin.grid)
		} else {
			v.grid.RestoreVisit(origin.grid)
		}
		return -1
	}
	i, ok := scope.RestoreImage(origin.image, v.sourceOccurrences(origin.image.Path))
	if !ok {
		return -1
	}
	v.state.Select(i)
	return i
}

// Defer native menu publication across the synchronous root transaction. This
// is an effect barrier, not visit authority or a queue of deferred commands.
func (v *viewer) beginBrowsingUpdate() func() {
	v.browsingUpdates++
	return func() {
		v.browsingUpdates--
		v.syncMenus()
	}
}
