package ui

import "slices"

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

	origin, restore := v.visualsearch.DetachOrigin()
	if restore {
		v.resetSearchPresentation()
	}
	// Closing comparison can deliver a pending ranking. Detach search first,
	// before any callback can expose the collection being reconciled.
	if v.comparisonActive() && (len(indices) > 0 || change.kind == sourcesRevalidated || restore && !origin.Grid.Visible) {
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

	index := -1
	if restore {
		index = v.restoreSearchOrigin(origin)
	}
	if change.kind == sourcesRevalidated && index < 0 && v.FileCount() > 0 {
		index = v.state.index
	}
	if index >= 0 && change.kind != sourceLoadFailed {
		v.ShowImage(index)
	}
	if restore {
		v.syncMenus()
		v.ForceRepaint()
	}
	return index
}
