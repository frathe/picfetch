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
// something meaningless.
func (m *Model) BeginInspect(i int) {
	var key string
	if i >= 0 && i < m.set.Count() {
		key = m.set.KeyAt(i)
	}
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

// InspectSource is a linear scan of the current file set for the key
// recorded by BeginInspect, so a caller can find where the inspected file
// lives now even after a sort or filter moved it away from the index it
// was inspected at. Returns -1 when inspect is off or the file is gone.
func (m *Model) InspectSource() int {
	m.mu.Lock()
	key := m.inspectKey
	m.mu.Unlock()
	if key == "" {
		return -1
	}
	for i := range m.set.Count() {
		if m.set.KeyAt(i) == key {
			return i
		}
	}
	return -1
}

// InspectMembers returns the inspected file's duplicate group in
// ascending index order, or nil when inspect is off or its file is gone.
func (m *Model) InspectMembers() []int {
	src := m.InspectSource()
	if src < 0 {
		return nil
	}
	return m.Members(src)
}

// IsHiddenExtra reports whether i is a non-representative member of a
// duplicate group while hide is on. Unhashed files are never extras:
// their installed group size is 0, which already fails the size check
// below on its own.
//
// hide and the installed Groups snapshot are read under a single lock
// acquisition and then used via Groups' own value methods, not Model's -
// GroupSize/RepresentativeOf take mu themselves, and mu is not
// reentrant.
func (m *Model) IsHiddenExtra(i int) bool {
	m.mu.Lock()
	hide := m.hide
	groups := m.groups
	m.mu.Unlock()
	if !hide {
		return false
	}
	if groups.Size(i) < 2 {
		return false
	}
	return i != groups.RepresentativeOf(i)
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
	n := m.set.Count()
	if n == 0 {
		return 0
	}
	if members := m.InspectMembers(); len(members) >= 2 && delta != 0 {
		return stepInMembers(members, from, delta)
	}
	if !m.HideDuplicates() || delta == 0 {
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
			if !m.IsHiddenExtra(i) {
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
	for i := range m.set.Count() {
		if !m.IsHiddenExtra(i) {
			return i
		}
	}
	return 0
}

// LastVisible is the last index that is not a hidden extra, or 0 when
// nothing qualifies. Same non-check of HideDuplicates as FirstVisible.
func (m *Model) LastVisible() int {
	n := m.set.Count()
	for i := n - 1; i >= 0; i-- {
		if !m.IsHiddenExtra(i) {
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
	var out []int
	for i := range m.set.Count() {
		if i != current && !m.IsHiddenExtra(i) {
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
