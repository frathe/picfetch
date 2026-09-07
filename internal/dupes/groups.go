package dupes

import (
	"context"

	"github.com/frathe/picfetch/internal/imaging"
)

// Groups is a snapshot of duplicate-group membership across a FileSet,
// grouped at the Hamming distance Dist. Sizes[i] is 0 for an unhashed
// file, 1 for a unique hashed file, and >= 2 for a member of a duplicate
// group. Reps[i] defaults to i.
type Groups struct {
	Sizes, Reps []int
	Dist        int
	key         GroupingKey
	valid       bool
}

// Key identifies the immutable inputs captured by this snapshot.
func (g Groups) Key() GroupingKey { return g.key }

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
// worker. A mismatched fact namespace is ignored without erasing facts that
// an explicit UI adoption may retain after an incremental change.
func (m *Model) Compute() Groups {
	g, _ := m.ComputeContext(context.Background())
	return g
}

// ComputeContext returns no installable partial result when cancelled.
func (m *Model) ComputeContext(ctx context.Context) (Groups, error) {
	if err := ctx.Err(); err != nil {
		return Groups{}, err
	}
	m.computes.Add(1)
	s := m.set.Snapshot()
	n := s.Count()
	sizes := make([]int, n)
	reps := make([]int, n)
	for i := range n {
		reps[i] = i
	}
	m.mu.Lock()
	idx := make([]int, 0, n)
	hs := make([]uint64, 0, n)
	hashed := make([]bool, n)
	px := make([]int, n)
	dist := m.dist
	key := m.groupingKeyLocked(s.Generation())
	for i := range n {
		if i&255 == 0 {
			if err := ctx.Err(); err != nil {
				m.mu.Unlock()
				return Groups{}, err
			}
		}
		if m.gen != s.Generation() {
			break
		}
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

	groups, err := imaging.DuplicateGroupsContext(ctx, hs, dist)
	if err != nil {
		return Groups{}, err
	}
	for _, grp := range groups {
		if err := ctx.Err(); err != nil {
			return Groups{}, err
		}
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
	if err := ctx.Err(); err != nil {
		return Groups{}, err
	}
	return Groups{Sizes: sizes, Reps: reps, Dist: dist, key: key, valid: true}, nil
}

// Install replaces the model's live group snapshot with g.
func (m *Model) Install(g Groups) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if !g.valid || g.key != m.groupingKeyLocked(m.set.Snapshot().Generation()) {
		return false
	}
	m.groups = g
	return true
}

// Rebuild computes a fresh snapshot and installs it.
func (m *Model) Rebuild() {
	if _, ok := m.CurrentGroups(); ok {
		return
	}
	m.Install(m.Compute())
}

// GroupSize is the installed snapshot's Size(i).
func (m *Model) GroupSize(i int) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readableGroupsLocked(m.set.Snapshot().Generation()).Size(i)
}

// RepresentativeOf is the installed snapshot's RepresentativeOf(i).
func (m *Model) RepresentativeOf(i int) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.readableGroupsLocked(m.set.Snapshot().Generation()).RepresentativeOf(i)
}

// Members returns set indices sharing i's representative, in ascending
// index order, or nil when the group has fewer than two members.
func (m *Model) Members(i int) []int {
	s := m.set.Snapshot()
	n := s.Count()

	m.mu.Lock()
	groups := m.readableGroupsLocked(s.Generation())
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

// GroupingKey identifies the immutable inputs to one grouping computation.
// The fields are opaque; callers compare keys to coalesce unchanged requests.
type GroupingKey struct {
	generation, facts, reset uint64
	distance                 int
}

func (m *Model) groupingKeyLocked(generation uint64) GroupingKey {
	return GroupingKey{generation: generation, facts: m.factRevision, reset: m.reset, distance: m.dist}
}

func (m *Model) GroupingKey() GroupingKey {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.groupingKeyLocked(m.set.Snapshot().Generation())
}

// CurrentGroups recognizes only an accepted snapshot of the present inputs.
func (m *Model) CurrentGroups() (Groups, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.groups, m.groups.valid && m.groups.key == m.groupingKeyLocked(m.set.Snapshot().Generation())
}

// Facts and distance may change while their replacement is computing, but
// old indices or a reset content namespace must never hide current files.
func (m *Model) readableGroupsLocked(generation uint64) Groups {
	if !m.groups.valid || m.groups.key.generation != generation || m.groups.key.reset != m.reset {
		return Groups{}
	}
	return m.groups
}
