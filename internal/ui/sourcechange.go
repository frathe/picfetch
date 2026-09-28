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
	sourcesTrashed
	sourceLoadFailed
	sourceUnavailable
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
	targets []fyne.URI
	written []fyne.URI
}

// Reorder retains collection facts but retires generation-bound producers and
// reconciles retained visits before choosing the one authoritative image load.
func (v *viewer) commitCollectionReorder(ordered []fyne.URI) {
	v.favorites.CancelOpen()
	finishUpdate := v.beginBrowsingUpdate()
	browsing := v.captureBrowsingReconciliation()
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
	v.favorites.CancelOpen()
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
// Trash consumes it after its outcome notification. Other changes admit any
// required load here, after reconciliation.
func (v *viewer) reconcileSources(change sourceChange) int {
	before := v.state.Observe()
	failedLoad := change.kind == sourceLoadFailed || change.kind == sourceUnavailable
	indices := slices.Sorted(slices.Values(change.removed))
	indices = slices.Compact(slices.DeleteFunc(indices, func(i int) bool {
		return i < 0 || i >= before.Count()
	}))
	if (change.kind == sourcesRemoved || change.kind == sourcesTrashed || failedLoad) && len(indices) == 0 && !before.ContainsTargets(change.targets) {
		return -1
	}
	defer v.beginBrowsingUpdate()()
	// Settings reconciliation also runs before Favorites is constructed.
	if v.favorites != nil {
		v.favorites.CancelOpen()
	}
	browsing := v.captureBrowsingReconciliation()
	if change.kind == sourceWritten {
		v.locationMap.InvalidateSources(change.written)
	}

	restore := browsing.origin != nil
	// Closing comparison can deliver a pending ranking. Detach search first,
	// before any callback can expose the collection being reconciled.
	if v.comparisonActive() && (len(indices) > 0 || len(change.targets) > 0 || change.kind == sourcesRevalidated || restore && !browsing.origin.grid.Visible) {
		v.compare.Close()
	}
	if change.kind == sourcesRevalidated || change.kind == sourceWritten {
		v.favThumbLifecycle.Invalidate()
	}
	if change.kind == sourcesRevalidated {
		v.imgCache.Purge()
	}
	var committed collectionChange
	if len(indices) > 0 || len(change.targets) > 0 {
		v.invalidateSort()
		if change.kind == sourceUnavailable {
			committed = v.state.MarkUnavailable(indices[0])
		} else if len(change.targets) > 0 {
			committed = v.state.RemoveTargets(change.targets)
		} else {
			committed = v.state.Remove(indices)
		}
		browsing = browsing.remap(committed.survivors)
	}
	after := v.state.Observe()
	for _, removed := range committed.removed {
		v.imgCache.Remove(removed.String())
		if after.Occurrences().Resolve(fileidentity.Occurrence{Path: removed.Path()}) < 0 {
			v.explorer.RemoveCohortSource(removed.Path())
		}
	}
	v.browsing.reconcile(after.Generation(), browsing.survivors)
	v.cancelExplorerPreparation()
	v.explorer.SourcesChanged()

	switch change.kind {
	case sourcesRemoved, sourcesTrashed, sourceLoadFailed, sourceUnavailable, sourcesRevalidated:
		v.grid.FilesChanged()
		if after.Count() == 0 {
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
	if change.kind == sourcesRevalidated && index < 0 && after.Count() > 0 {
		scope := v.captureBrowsingScope()
		_, current, _ := after.Current()
		if !scope.restricted || slices.Contains(scope.indexes, current) {
			index = current
		}
	}
	if index >= 0 && !failedLoad && change.kind != sourcesTrashed {
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

// Capture before mutation and detach search before any feature callback. After
// publication, remap applies the model's one survivor result to these values.
func (v *viewer) captureBrowsingReconciliation() browsingReconciliation {
	change := browsingReconciliation{}
	kind := v.browsing.current().binding.kind
	if (kind == browsingExplorer || kind == browsingCluster) && v.grid.Visible() {
		bookmark := v.grid.CaptureVisit()
		change.grid = &bookmark
	}
	if v.locationVisitActive() {
		v.locationInput.prepareOp.Invalidate()
		v.locationInput.prepare = nil
	}
	if origin, detached := v.detachSearchOrigin(); detached {
		change.origin = &origin
	}
	return change
}

func (change browsingReconciliation) remap(survivors map[fileidentity.Occurrence]fileidentity.Occurrence) browsingReconciliation {
	change.survivors = survivors
	if change.grid != nil {
		bookmark := change.grid.RemapOccurrences(survivors)
		change.grid = &bookmark
	}
	if change.origin != nil {
		origin := change.origin.remap(survivors)
		change.origin = &origin
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
	v.fileWork.searchLifecycle.Invalidate()
	v.grid.Close()
	if v.FileCount() == 0 {
		v.presentCommittedEmptyCollection()
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
	i, ok := scope.RestoreImage(origin.image, v.state.Observe().Occurrences())
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
