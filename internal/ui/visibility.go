// Which files the viewer can navigate to: the adapter that lets
// internal/dupes group over this viewer's file set, and the jump that
// takes the display off a file the model has just classified as a hidden
// duplicate extra.

package ui

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
