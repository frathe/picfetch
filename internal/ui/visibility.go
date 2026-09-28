// Which files the viewer can navigate to: the adapter that lets
// internal/dupes group over this viewer's file set, the navigation
// helpers plain stepping asks it through, the push that turns
// hide-duplicates on or off, and the jump that takes the display off a
// file the model has just classified as a hidden duplicate extra.
//
// Everything here reads the model the viewer owns, never the grid
// overlay. That is the point of the file: arrow keys, Home/End and the
// slideshow's shuffle answer "which file comes next" without a closed
// overlay having to be consulted per index.

package ui

import (
	"math/rand/v2"

	"github.com/frathe/picfetch/internal/dupes"
)

// dupeFileSet adapts the viewer to dupes.FileSet, so the model can group
// over the loaded files while staying Fyne-free: every fact it stores is
// keyed by the URI string the rest of the app already keys its caches by.
//
// It forwards the collection's immutable URI-key projection, so workers retain
// the keys and generation from one observation. Collection indexing represents
// nil URIs as empty keys before any duplicate consumer sees them.
type dupeFileSet struct {
	v *viewer
}

// Snapshot is the viewer's published, immutable view of the file set:
// what dupes.Model reads instead of walking a live count and key lookup
// while the UI goroutine replaces the slice underneath it.
func (s dupeFileSet) Snapshot() dupes.Snapshot { return s.v.state.Observe().FileSet() }

// jumpIfHiddenExtra moves the display to the current file's group
// representative when the model has just made that file a hidden extra -
// what stops hide-duplicates, a distance change, or a hash landing from
// leaving the viewer parked on a file navigation can no longer reach.
//
// It lives here rather than in internal/ui/grid because it calls
// ShowImage, which is this package's job; the grid reaches it by firing
// the model's observers once it has re-filtered itself (see
// registerFeatures for the registration, and the fire sites in
// internal/ui/grid/dupes.go).
//
// An active inspect session is exactly the state where the user asked to
// sit on an extra - the file committed out of the variants grid - so it
// is left alone.
func (v *viewer) jumpIfHiddenExtra() {
	if v.dupes.Inspecting() || v.browsing.has(browsingExplorer) || v.searchActive() || v.locationVisitActive() {
		return
	}
	vis := v.dupes.Visibility()
	if i := v.CurrentIndex(); vis.HiddenExtra(i) {
		v.loadImage(vis.RepresentativeOf(i))
	}
}

// pushHideDuplicates hands the hide flag to the model and, when the
// stored value actually moved, lets the grid re-apply its own view of it -
// hashing whatever is still unhashed when hide just turned on, then
// re-filtering - before the model's observers run. Exactly the shape of
// pushDuplicateDistance (memlimits.go), and for the same reason: those two
// halves have to stay in that order, because jumpIfHiddenExtra must see
// the group snapshot the grid's re-filter installed.
//
// on is passed on rather than left for the grid to re-read because the
// grid's work differs by direction - see Overview.HideDuplicatesChanged.
// The grid's own D key still goes through Overview.SetHideDuplicates,
// which does the same two steps in the same order from the other side.
func (v *viewer) pushHideDuplicates(on bool) {
	if !v.dupes.SetHideDuplicates(on) {
		return
	}

	v.grid.HideDuplicatesChanged(on)
}

// nextVisibleIndex is where plain navigation asks the duplicate model
// which file lies delta steps from here. The inspect-members ring, the
// hide-off arithmetic and the skip-the-extras walk all live in
// dupes.Model.NextVisible, so StepImage and Advance no longer poll a
// closed grid overlay once per index to find out who exists.
//
// With hide off the result is deliberately unclamped - NextVisible hands
// back from+delta as it is, and ShowImage is what folds it into range.
// Do not add a bounds check on this path.
func (v *viewer) nextVisibleIndex(from, delta int) (int, bool) {
	if scope := v.captureBrowsingScope(); scope.restricted {
		return scope.Next(from, delta)
	}
	return v.dupes.NextVisible(from, delta), v.FileCount() > 0
}

// firstVisibleIndex is where Home lands: the first file that is not a
// hidden duplicate extra, or 0 when nothing qualifies.
func (v *viewer) firstVisibleIndex() (int, bool) {
	if scope := v.captureBrowsingScope(); scope.restricted {
		return scope.First()
	}
	return v.dupes.FirstVisible(), v.FileCount() > 0
}

// lastVisibleIndex is End's counterpart to firstVisibleIndex.
func (v *viewer) lastVisibleIndex() (int, bool) {
	if scope := v.captureBrowsingScope(); scope.restricted {
		return scope.Last()
	}
	return v.dupes.LastVisible(), v.FileCount() > 0
}

// randomVisibleOther picks the slideshow's next shuffle target. The draw
// stays on this side of the boundary: internal/dupes must not import
// math/rand, so it hands back the candidates and this makes the choice.
//
// With hide off there is no candidate list worth building - every index
// but current qualifies - so it keeps going through randomOtherIndex
// (load.go), the same draw the shuffle has always used.
func (v *viewer) randomVisibleOther(current int) (int, bool) {
	if scope := v.captureBrowsingScope(); scope.restricted {
		if len(scope.indexes) == 0 {
			return 0, false
		}
		return scope.indexes[rand.IntN(len(scope.indexes))], true
	}
	if !v.dupes.HideDuplicates() {
		return randomOtherIndex(v.state.Observe().Count(), current), v.FileCount() > 0
	}

	vis := v.dupes.VisibleIndexesExcept(current)
	if len(vis) == 0 {
		return current, v.FileCount() > 0
	}

	return vis[rand.IntN(len(vis))], true
}
