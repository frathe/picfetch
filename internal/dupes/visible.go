package dupes

// HideDuplicates reports whether duplicate extras are currently hidden
// from navigation.
func (m *Model) HideDuplicates() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.hide
}

// SetHideDuplicates sets the hide flag and reports whether it actually
// changed. It is a pure flag setter: it does not hash anything, rebuild
// groups, filter, or jump the caller off a now-hidden file. Those are the
// jobs of whatever observes the change - see OnChange/Notify - not of
// this method.
func (m *Model) SetHideDuplicates(on bool) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.hide == on {
		return false
	}
	m.hide = on
	return true
}

// BeginInspect starts an inspect session on the file at index i, storing
// its key rather than i itself: the key survives the file set shifting
// under it (a sort, a filter, a deletion), where a stored index would
// silently end up pointing at a different file after such a shift. An
// out-of-range i, or an empty key at i (this package's equivalent of the
// original code's nil-URI guard), clears inspect instead of storing
// something meaningless - Snapshot.KeyAt answers "" for both.
func (m *Model) BeginInspect(i int) {
	key := m.set.Snapshot().KeyAt(i)
	m.mu.Lock()
	m.inspectKey = key
	m.mu.Unlock()
}

// ClearInspect ends the inspect session.
func (m *Model) ClearInspect() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.inspectKey = ""
}

// Inspecting reports whether an inspect session is active.
func (m *Model) Inspecting() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.inspectKey != ""
}

// InspectSource is the current index of the file BeginInspect recorded,
// or -1 when inspect is off or that file is no longer in the set.
//
// A map lookup in the published Snapshot, not a scan: the arrow keys ask
// this once per step, and at 50k files a scan per keystroke was the
// dominant cost of moving through a drop with inspect on. IndexOf
// returns -1 for the empty key, which folds in the inspect-off case.
func (m *Model) InspectSource() int {
	m.mu.Lock()
	key := m.inspectKey
	m.mu.Unlock()

	return m.set.Snapshot().IndexOf(key)
}

// InspectMembers returns the inspected file's duplicate group in
// ascending index order, or nil when inspect is off or its file is gone.
func (m *Model) InspectMembers() []int {
	return m.inspectMembers(m.set.Snapshot())
}

// inspectMembers is InspectMembers against a snapshot the caller already
// holds. The inspect key and the group snapshot come out of one lock
// acquisition, so a NextVisible step pays one mutex for the whole ring
// lookup instead of one per member.
func (m *Model) inspectMembers(s Snapshot) []int {
	m.mu.Lock()
	key := m.inspectKey
	groups := m.groups
	m.mu.Unlock()

	src := s.IndexOf(key)
	if src < 0 {
		return nil
	}

	return membersOf(groups, s.Count(), src)
}

// Visibility is the model's hide flag and installed group snapshot frozen
// at one read. Groups' slices are never mutated in place - Install always
// replaces the struct wholesale, never one of its fields - so a caller
// holding a Visibility keeps answering off the exact pair it was handed,
// however the model's hide flag or groups change underneath it afterward.
// See Model.Visibility for why a caller wants that.
type Visibility struct {
	Hide   bool
	Groups Groups
}

// HiddenExtra reports whether i is a non-representative member of a
// duplicate group while hide is on, in three steps: hide off is always
// false; Groups.Size(i) < 2 is always false too - unhashed files are
// never extras, because their installed group size is 0, which already
// fails this check on its own; otherwise i is an extra exactly when it
// is not its group's representative.
func (v Visibility) HiddenExtra(i int) bool {
	if !v.Hide {
		return false
	}
	if v.Groups.Size(i) < 2 {
		return false
	}

	return i != v.Groups.RepresentativeOf(i)
}

// Visible is the negation of HiddenExtra.
func (v Visibility) Visible(i int) bool {
	return !v.HiddenExtra(i)
}

// RepresentativeOf delegates to the frozen Groups.
func (v Visibility) RepresentativeOf(i int) int {
	return v.Groups.RepresentativeOf(i)
}

// Size delegates to the frozen Groups.
func (v Visibility) Size(i int) int {
	return v.Groups.Size(i)
}

// Visibility reads hide and the installed group snapshot in one mutex
// acquisition, for a caller about to test many indices against them - the
// same reasoning Snapshot already applies to the file set itself. Read the
// value once at the top of such a pass and call its methods per index
// instead of Model's IsHiddenExtra/RepresentativeOf/GroupSize.
//
// The walks below used to call IsHiddenExtra per candidate, which is one
// mutex acquisition per candidate; at 50k files with hide on, that was
// the cost of a single arrow key.
func (m *Model) Visibility() Visibility {
	m.visibilityReads.Add(1)
	m.mu.Lock()
	defer m.mu.Unlock()

	return Visibility{Hide: m.hide, Groups: m.groups}
}

// VisibilityReads is how many times Visibility has run, so tests can prove
// a filter pass over many indices paid one model-mutex acquisition rather
// than one per index.
func (m *Model) VisibilityReads() int32 {
	return m.visibilityReads.Load()
}

// IsHiddenExtra is Visibility.HiddenExtra's single-index entry point. A
// caller that will test many indices should take a Visibility once
// instead of paying a lock acquisition per candidate here.
func (m *Model) IsHiddenExtra(i int) bool {
	return m.Visibility().HiddenExtra(i)
}

// IsVisible is the negation of IsHiddenExtra.
func (m *Model) IsVisible(i int) bool {
	return !m.IsHiddenExtra(i)
}

// NextVisible reproduces the pre-extraction viewer's nextVisibleIndex
// exactly, including its branch order:
//
//  1. Inspecting a group of two or more with a non-zero delta: step
//     within that group's members ring instead of the whole file set.
//  2. Hide off, or delta == 0: return from+delta unclamped. This looks
//     like a bug - it is not. Normalising the result (wrapping, clamping
//     to range) is the caller's job, ShowImage in the viewer; do not add
//     a bounds check here.
//  3. Otherwise, walk |delta| steps modulo Count(), skipping hidden
//     extras. If a single step wraps all the way back to where it
//     started without finding a visible file, that step - and so the
//     whole call - returns from unchanged.
func (m *Model) NextVisible(from, delta int) int {
	s := m.set.Snapshot()
	n := s.Count()
	if n == 0 {
		return 0
	}
	if members := m.inspectMembers(s); len(members) >= 2 && delta != 0 {
		return stepInMembers(members, from, delta)
	}
	vis := m.Visibility()
	if !vis.Hide || delta == 0 {
		return from + delta
	}
	step := 1
	if delta < 0 {
		step = -1
	}
	i := from
	for k := 0; k < absInt(delta); k++ {
		start := i
		for {
			i = (i + step + n) % n
			if !vis.HiddenExtra(i) {
				break
			}
			if i == start {
				return from
			}
		}
	}

	return i
}

// FirstVisible is the first index that is not a hidden extra, or 0 when
// nothing qualifies (today's fallback). It does not check HideDuplicates
// itself - IsHiddenExtra is already false for every index while hide is
// off, so this loop degrades to "the first index" on its own.
func (m *Model) FirstVisible() int {
	s := m.set.Snapshot()
	vis := m.Visibility()
	for i := range s.Count() {
		if !vis.HiddenExtra(i) {
			return i
		}
	}

	return 0
}

// LastVisible is the last index that is not a hidden extra, or 0 when
// nothing qualifies. Same non-check of HideDuplicates as FirstVisible.
func (m *Model) LastVisible() int {
	s := m.set.Snapshot()
	vis := m.Visibility()
	for i := s.Count() - 1; i >= 0; i-- {
		if !vis.HiddenExtra(i) {
			return i
		}
	}

	return 0
}

// VisibleIndexesExcept returns every index that is neither current nor a
// hidden extra, in ascending order. It is the candidate list for a
// caller's own random draw - randomness stays in internal/ui, which is
// why this package must not import math/rand.
func (m *Model) VisibleIndexesExcept(current int) []int {
	s := m.set.Snapshot()
	vis := m.Visibility()
	var out []int
	for i := range s.Count() {
		if i != current && !vis.HiddenExtra(i) {
			out = append(out, i)
		}
	}

	return out
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func stepInMembers(members []int, from, delta int) int {
	n := len(members)
	if n == 0 {
		return from
	}
	if delta == 0 {
		return from
	}
	pos := 0
	found := false
	for i, m := range members {
		if m == from {
			pos = i
			found = true
			break
		}
	}
	if !found {
		pos = 0
	}
	step := 1
	if delta < 0 {
		step = -1
	}
	for k := 0; k < absInt(delta); k++ {
		pos = (pos + step + n) % n
	}
	return members[pos]
}
