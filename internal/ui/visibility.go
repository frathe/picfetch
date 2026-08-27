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

import "math/rand/v2"

// dupeFileSet adapts the viewer to dupes.FileSet, so the model can group
// over the loaded files while staying Fyne-free: every fact it stores is
// keyed by the URI string the rest of the app already keys its caches by.
//
// It wraps the same FileCount/FileAt/Generation the feature packages'
// Host interfaces bind to (see viewer.go), rather than reading
// v.state.files directly, so there is one definition of "the file set"
// for every consumer.
//
// KeyAt has to stay a plain lookup. dupes.Model.Compute calls it while
// holding the model's own mutex - faithfully to the code this replaced,
// which read the host's FileAt(i) under hashMu - so anything here that
// took a lock, or reached back into the model, would deadlock a hashing
// worker. If it ever needs to, hoist the key slice out of the lock in
// Compute instead.
type dupeFileSet struct {
	v *viewer
}

func (s dupeFileSet) Count() int { return s.v.FileCount() }

// KeyAt is the URI string of the file at i, or "" when there is no URI
// there: the nil-URI guard every duplicate helper used to apply before it
// touched a fyne.URI, moved to the one place the model reaches through.
func (s dupeFileSet) KeyAt(i int) string {
	u := s.v.FileAt(i)
	if u == nil {
		return ""
	}

	return u.String()
}

func (s dupeFileSet) Generation() uint64 { return s.v.Generation() }

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
	if v.dupes.Inspecting() {
		return
	}
	if i := v.state.index; v.dupes.IsHiddenExtra(i) {
		v.ShowImage(v.dupes.RepresentativeOf(i))
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
func (v *viewer) nextVisibleIndex(from, delta int) int {
	return v.dupes.NextVisible(from, delta)
}

// firstVisibleIndex is where Home lands: the first file that is not a
// hidden duplicate extra, or 0 when nothing qualifies.
func (v *viewer) firstVisibleIndex() int {
	return v.dupes.FirstVisible()
}

// lastVisibleIndex is End's counterpart to firstVisibleIndex.
func (v *viewer) lastVisibleIndex() int {
	return v.dupes.LastVisible()
}

// randomVisibleOther picks the slideshow's next shuffle target. The draw
// stays on this side of the boundary: internal/dupes must not import
// math/rand, so it hands back the candidates and this makes the choice.
//
// With hide off there is no candidate list worth building - every index
// but current qualifies - so it keeps going through randomOtherIndex
// (load.go), the same draw the shuffle has always used.
func (v *viewer) randomVisibleOther(current int) int {
	if !v.dupes.HideDuplicates() {
		return randomOtherIndex(len(v.state.files), current)
	}

	vis := v.dupes.VisibleIndexesExcept(current)
	if len(vis) == 0 {
		return current
	}

	return vis[rand.IntN(len(vis))]
}
