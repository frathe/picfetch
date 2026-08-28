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

// visibility reads hide and the installed group snapshot in one lock
// acquisition, for callers that then test many indices against them.
// The walks below used to call IsHiddenExtra per candidate, which is one
// mutex acquisition per candidate; at 50k files with hide on, that was
// the cost of a single arrow key.
func (m *Model) visibility() (hide bool, groups Groups) {
	m.mu.Lock()
	defer m.mu.Unlock()

	return m.hide, m.groups
}

// hiddenExtra is IsHiddenExtra's test against an already-read hide flag
// and group snapshot: no lock, so a walk can call it per index.
func hiddenExtra(hide bool, groups Groups, i int) bool {
	if !hide {
		return false
	}
	if groups.Size(i) < 2 {
		return false
	}

	return i != groups.RepresentativeOf(i)
}

// IsHiddenExtra reports whether i is a non-representative member of a
// duplicate group while hide is on. Unhashed files are never extras:
// their installed group size is 0, which already fails the size check on
// its own.
//
// This is the single-index entry point. A walk over many indices should
// call visibility() once itself and then hiddenExtra per index, rather
// than paying a lock acquisition per candidate here.
func (m *Model) IsHiddenExtra(i int) bool {
	hide, groups := m.visibility()

	return hiddenExtra(hide, groups, i)
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
	hide, groups := m.visibility()
	if !hide || delta == 0 {
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
			if !hiddenExtra(hide, groups, i) {
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
	hide, groups := m.visibility()
	for i := range s.Count() {
		if !hiddenExtra(hide, groups, i) {
			return i
		}
	}

	return 0
}

// LastVisible is the last index that is not a hidden extra, or 0 when
// nothing qualifies. Same non-check of HideDuplicates as FirstVisible.
func (m *Model) LastVisible() int {
	s := m.set.Snapshot()
	hide, groups := m.visibility()
	for i := s.Count() - 1; i >= 0; i-- {
		if !hiddenExtra(hide, groups, i) {
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
	hide, groups := m.visibility()
	var out []int
	for i := range s.Count() {
		if i != current && !hiddenExtra(hide, groups, i) {
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
