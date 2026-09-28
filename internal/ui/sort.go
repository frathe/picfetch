package ui

import (
	"context"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/filesort"
)

// toggleSort is the S key: it cycles the state sort mode to the next mode - see
// SetSortMode below, which does the actual work.
func (v *viewer) toggleSort() {
	if _, ok := v.admitCommand(commandRequest{command: commandSort, intent: intentToggle}); !ok {
		return
	}
	v.SetSortMode(v.state.SortMode().Next())
}

// SetSortMode prepares display order from a captured source order. Commit keeps
// the latest requested occurrence, including navigation made during preparation.
// Metadata-heavy modes run off UI; empty collections only change the preference.
func (v *viewer) SetSortMode(m filesort.Mode) {
	collection := v.state.Observe()
	if collection.Count() == 0 {
		v.invalidateSort()
		v.state.SetSortMode(m)
		v.applyTitle()
		v.syncMenus()

		return
	}

	// The title's sort-mode prefix updates immediately, even before the
	// reorder itself finishes - there's no reason to make the user wait for
	// a large sort just to see that their choice registered.
	if v.sortModeBefore == nil {
		previous := v.state.SortMode()
		v.sortModeBefore = &previous
	}
	v.state.SetSortMode(m)
	v.applyTitle()
	v.syncMenus()

	v.startSort(m, collection.SourceFiles(), v.commitCollectionReorder)
}

// invalidateSort advances sortOp.lifecycle and, if a reorder is currently in
// flight, cancels its context so filesort.Order's per-file stat/Exif loop
// notices and stops promptly instead of running to completion for a result
// that's already guaranteed to be discarded - see sortOp's field comment for
// every caller (a newer sort superseding an older one, Escape via
// cancelSort, RemoveFile, clearToDropzone). It returns the new revision for
// tests and diagnostics.
func (v *viewer) invalidateSort() uint64 {
	if v.sortOp.active {
		v.pendingPictureFrame = false
		v.explorerInput.pendingLaunch = false
	}
	revision := v.sortOp.invalidate()
	if v.sortModeBefore != nil {
		v.state.SetSortMode(*v.sortModeBefore)
		v.sortModeBefore = nil
		v.applyTitle()
		v.syncMenus()
	}
	return revision
}

// startSort reorders unsorted under mode in the background, showing the sort
// spinner/label while it computes - shared by SetSortMode and
// applyScannedFiles (drop.go), the two places that call filesort.Order over a
// potentially large file set: its capture-date/modified/size modes stat or
// Exif-read every file, which freezes the UI for as long as that takes if
// done inline on the UI goroutine (see filesort.Order's own doc comment).
// Any sort already in flight is cancelled by sortOp.begin, rather
// than left to keep computing a result this call already supersedes - so
// pressing S repeatedly cycles straight through modes instead of queuing up
// wasted background work behind whichever one happened to be slowest.
// onDone runs once, and only if this call's token is still current once
// the reorder finishes - see sortOp's field comment for every way it can be
// superseded.
func (v *viewer) startSort(mode filesort.Mode, unsorted []fyne.URI, onDone func(ordered []fyne.URI)) {
	token, sortDone := v.sortOp.begin(v.heicContext(context.Background()))
	v.syncMenus()

	v.sortOp.show()
	// A widget hidden since construction has never been painted, so it has
	// no canvas of its own to mark dirty on Show/Refresh - see
	// ForceRepaint's own doc comment.
	v.ForceRepaint()
	dispatch := v.sortDo
	if dispatch == nil {
		dispatch = fyne.Do
	}

	go func() {
		delivery := token.FinalDelivery()
		defer delivery.Abandon()
		ordered := filesort.Order(token.Context(), mode, unsorted)
		delivery.Dispatch(dispatch, func() { v.finishSort(ordered, onDone) }, sortDone)
	}()
}

// finishSort applies a current result on UI. FinalDelivery owns currentness,
// token release and this operation's completion, including discarded results.
func (v *viewer) finishSort(ordered []fyne.URI, onDone func([]fyne.URI)) {
	// Only current delivery can clear progress. The collection commit in onDone
	// publishes its generation together with the reordered file set.
	v.sortOp.finish()
	v.sortModeBefore = nil

	v.syncMenus()
	onDone(ordered)
}

// cancelSort aborts a reorder in progress (Escape while v.sortOp.active is
// true), mirroring cancelScan (drop.go) for the analogous scan-gathering
// phase. invalidateSort's context cancellation makes filesort.Order's
// per-file stat/Exif loop notice and stop promptly instead of running to
// completion in the background for a result nobody will see.
//
// The pending mode is restored by invalidateSort. Collection facts are
// unchanged until a reorder's own onDone callback runs (see
// applyScannedFiles's and SetSortMode's own comments on why the pairing is
// atomic), so cancelling before that lands leaves them exactly as they
// already were - the untouched pre-sort file set, still fully intact and on
// screen, if there was one; nothing, still showing the dropzone, for a
// first-ever drop's cancelled reorder.
func (v *viewer) cancelSort() {
	if !v.sortOp.active {
		return
	}
	v.invalidateSort()

	if v.state.Observe().Count() == 0 {
		v.showWelcomeState()
		v.dropzone.Show()
	}

	v.syncMenus()
	v.ForceRepaint()
	v.ShowToast(lang.L("cancelled sorting"))
}

// SortMode reports the current sort order - the settings window's getter.
func (v *viewer) SortMode() filesort.Mode {
	return v.state.SortMode()
}
