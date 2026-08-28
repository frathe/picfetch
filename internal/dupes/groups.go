package dupes

import "github.com/frathe/picfetch/internal/imaging"

// Groups is a snapshot of duplicate-group membership across a FileSet,
// grouped at the Hamming distance Dist. Sizes[i] is 0 for an unhashed
// file, 1 for a unique hashed file, and >= 2 for a member of a duplicate
// group. Reps[i] defaults to i.
type Groups struct {
	Sizes, Reps []int
	Dist        int
}

// Size is 0 if i is unhashed, 1 if it is a unique hashed file, and >= 2
// if it belongs to a duplicate group. Out-of-range indices return 0.
func (g Groups) Size(i int) int {
	if i < 0 || i >= len(g.Sizes) {
		return 0
	}
	return g.Sizes[i]
}

// RepresentativeOf is the highest native pixel count in i's group,
// lowest index on a tie; i itself when unique, unhashed, or out of
// range.
func (g Groups) RepresentativeOf(i int) int {
	if i < 0 || i >= len(g.Reps) {
		return i
	}
	return g.Reps[i]
}

// Compute groups set's hashed files by Hamming distance, choosing the
// highest native pixel count as each group's representative (lowest
// index on a tie). It is pure and safe to call off the UI goroutine -
// hashing workers do exactly that; Install is what replaces the model's
// live snapshot.
//
// The file set is read once, at the top, as an immutable Snapshot: the
// count and the keys this pass groups over cannot disagree, however the
// app rewrites its file list meanwhile. Reading that snapshot under mu
// below is safe precisely because it is a value - it cannot reach back
// into the model or into the app, so it cannot deadlock a hashing
// worker. wipeIfStale locks and unlocks on its own first; the snapshot
// build then takes the lock again rather than nesting it.
func (m *Model) Compute() Groups {
	m.computes.Add(1)
	s := m.set.Snapshot()
	n := s.Count()
	sizes := make([]int, n)
	reps := make([]int, n)
	for i := range n {
		reps[i] = i
	}
	m.wipeIfStale(s.Generation())

	m.mu.Lock()
	idx := make([]int, 0, n)
	hs := make([]uint64, 0, n)
	hashed := make([]bool, n)
	px := make([]int, n)
	dist := m.dist
	for i := range n {
		key := s.KeyAt(i)
		if h, ok := m.hashes[key]; ok {
			idx = append(idx, i)
			hs = append(hs, h)
			hashed[i] = true
		}
		if sz, ok := m.native[key]; ok {
			px[i] = sz.X * sz.Y
		}
	}
	m.mu.Unlock()

	groups := imaging.DuplicateGroups(hs, dist)
	for _, grp := range groups {
		rep := idx[grp[0]]
		repPx := px[rep]
		for _, gi := range grp {
			hi := idx[gi]
			if px[hi] > repPx || (px[hi] == repPx && hi < rep) {
				rep, repPx = hi, px[hi]
			}
		}
		for _, gi := range grp {
			hi := idx[gi]
			sizes[hi] = len(grp)
			reps[hi] = rep
		}
	}
	for i := range n {
		if hashed[i] && sizes[i] == 0 {
			sizes[i] = 1
		}
	}
	return Groups{Sizes: sizes, Reps: reps, Dist: dist}
}

// Install replaces the model's live group snapshot with g.
func (m *Model) Install(g Groups) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.groups = g
}

// Rebuild computes a fresh snapshot and installs it.
func (m *Model) Rebuild() {
	m.Install(m.Compute())
}

// GroupSize is the installed snapshot's Size(i).
func (m *Model) GroupSize(i int) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.groups.Size(i)
}

// RepresentativeOf is the installed snapshot's RepresentativeOf(i).
func (m *Model) RepresentativeOf(i int) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.groups.RepresentativeOf(i)
}

// Members returns set indices sharing i's representative, in ascending
// index order, or nil when the group has fewer than two members.
func (m *Model) Members(i int) []int {
	n := m.set.Snapshot().Count()

	m.mu.Lock()
	groups := m.groups
	m.mu.Unlock()

	return membersOf(groups, n, i)
}

// membersOf is Members' body against an already-read Groups snapshot and
// count, so a caller that has both does not re-take the model mutex.
func membersOf(groups Groups, n, i int) []int {
	if groups.Size(i) < 2 {
		return nil
	}
	rep := groups.RepresentativeOf(i)
	var members []int
	for j := range n {
		if groups.RepresentativeOf(j) == rep {
			members = append(members, j)
		}
	}

	return members
}

// Computes is how many times Compute has run, so tests can prove a
// snapshot was computed off the UI queue rather than inside it.
func (m *Model) Computes() int32 {
	return m.computes.Load()
}
