package dupes

import (
	"context"
	"errors"
	"fmt"
	"image"
	"math/bits"
	"slices"
	"strconv"
	"sync"
	"testing"
)

func TestGroups_SizeAndRepresentativeOf_OutOfRange(t *testing.T) {
	g := Groups{Sizes: []int{1, 2}, Reps: []int{0, 0}}

	if got := g.Size(-1); got != 0 {
		t.Errorf("Size(-1) = %d, want 0", got)
	}
	if got := g.Size(2); got != 0 {
		t.Errorf("Size(2) = %d for an index == len(Sizes), want 0", got)
	}
	if got := g.RepresentativeOf(-1); got != -1 {
		t.Errorf("RepresentativeOf(-1) = %d, want -1 (itself)", got)
	}
	if got := g.RepresentativeOf(5); got != 5 {
		t.Errorf("RepresentativeOf(5) = %d for an index == len(Reps), want 5 (itself)", got)
	}
}

func TestGroupSizeAndRepresentativeOf_BeforeAnyInstall(t *testing.T) {
	m := New(newFakeSet(3, 1))

	if got := m.GroupSize(0); got != 0 {
		t.Errorf("GroupSize(0) = %d before any Install, want 0", got)
	}
	if got := m.RepresentativeOf(0); got != 0 {
		t.Errorf("RepresentativeOf(0) = %d before any Install, want 0 (itself)", got)
	}
}

func TestCompute_ExactPairGroupsTogether(t *testing.T) {
	set := newFakeSet(3, 1)
	m := New(set)
	m.PutHash("a", 5)
	m.PutHash("b", 5)
	// c stays unhashed.

	g := m.Compute()

	if g.Size(0) != 2 || g.Size(1) != 2 {
		t.Errorf("Sizes = %v, want both a and b at 2", g.Sizes)
	}
	if g.Size(2) != 0 {
		t.Errorf("Size(2) = %d for an unhashed file, want 0", g.Size(2))
	}
}

func TestCompute_ThreeFileGroup(t *testing.T) {
	set := newFakeSet(3, 1)
	m := New(set)
	// Pairwise Hamming distances 1, 2, 1 - all within the default
	// threshold (imaging.DuplicateMaxDistance = 6).
	m.PutHash("a", 1)
	m.PutHash("b", 3)
	m.PutHash("c", 2)

	g := m.Compute()

	for i := range 3 {
		if g.Size(i) != 3 {
			t.Errorf("Size(%d) = %d, want 3", i, g.Size(i))
		}
	}
}

func TestCompute_TwoDisjointGroups(t *testing.T) {
	set := newFakeSet(4, 1)
	m := New(set)
	m.PutHash("a", 1)
	m.PutHash("b", 1)
	m.PutHash("c", 0xFFFFFFFFFFFFFFFE)
	m.PutHash("d", 0xFFFFFFFFFFFFFFFE)

	g := m.Compute()

	if g.RepresentativeOf(0) != g.RepresentativeOf(1) {
		t.Error("a and b were not grouped together")
	}
	if g.RepresentativeOf(2) != g.RepresentativeOf(3) {
		t.Error("c and d were not grouped together")
	}
	if g.RepresentativeOf(0) == g.RepresentativeOf(2) {
		t.Error("the two far-apart hashes were grouped together, want disjoint groups")
	}
}

func TestCompute_RepresentativeIsHighestPixelCount(t *testing.T) {
	set := newFakeSet(2, 1)
	m := New(set)
	m.PutHash("a", 5)
	m.PutHash("b", 5)
	m.PutNativeSize("a", image.Pt(100, 100)) // 10000 px
	m.PutNativeSize("b", image.Pt(200, 200)) // 40000 px

	g := m.Compute()

	if rep := g.RepresentativeOf(0); rep != 1 {
		t.Errorf("RepresentativeOf(0) = %d, want 1 (higher native pixel count)", rep)
	}
	if rep := g.RepresentativeOf(1); rep != 1 {
		t.Errorf("RepresentativeOf(1) = %d, want 1 (itself, the representative)", rep)
	}
}

// TestCompute_UnknownPixelsLoseToKnown covers a group where only one member
// has ever had its native size probed: the unprobed member's pixel count
// defaults to 0, so the probed member always wins the representative pick,
// known or not.
func TestCompute_UnknownPixelsLoseToKnown(t *testing.T) {
	set := newFakeSet(2, 1) // a: unprobed, b: probed
	m := New(set)
	const h uint64 = 0x1111111111111111
	m.PutHash("a", h)
	m.PutHash("b", h)
	m.PutNativeSize("b", image.Pt(50, 1))

	g := m.Compute()

	if rep := g.RepresentativeOf(0); rep != 1 {
		t.Errorf("RepresentativeOf(0) = %d, want 1 (known size wins)", rep)
	}
}

// TestCompute_ZeroHashFirstFileIsUnique pins that imaging.DuplicateGroups
// treats hash 0 as ungroupable: a file whose thumbnail happens to hash to 0
// must not silently absorb genuinely sparse, unrelated hashes.
func TestCompute_ZeroHashFirstFileIsUnique(t *testing.T) {
	set := newFakeSet(3, 1) // flat, sparse-a, sparse-b
	m := New(set)
	m.PutHash("a", 0)
	m.PutHash("b", 1)
	m.PutHash("c", 2)

	g := m.Compute()

	if got := g.Size(0); got != 1 {
		t.Fatalf("Size(0) = %d, want 1 (hash 0 must not absorb sparse hashes)", got)
	}
	if g.RepresentativeOf(1) == 0 || g.RepresentativeOf(2) == 0 {
		t.Fatal("sparse hashes must not pick the hash-0 first file as representative")
	}
}

func TestCompute_RepresentativeIsLowestIndexOnTie(t *testing.T) {
	set := newFakeSet(3, 1)
	m := New(set)
	m.PutHash("a", 5)
	m.PutHash("b", 5)
	m.PutHash("c", 5)
	// No native sizes recorded - all pixel counts tie at 0.

	g := m.Compute()

	for i := range 3 {
		if rep := g.RepresentativeOf(i); rep != 0 {
			t.Errorf("RepresentativeOf(%d) = %d, want 0 (lowest index on a tie)", i, rep)
		}
	}
}

// TestCompute_EqualPixelsKeepsLowestIndex is the tie-break at a *recorded*
// pixel count rather than at the 0 an unprobed file defaults to, so the
// lowest-index rule is pinned for the case where both members really were
// probed and really are the same size.
func TestCompute_EqualPixelsKeepsLowestIndex(t *testing.T) {
	set := newFakeSet(2, 1)
	m := New(set)
	const h uint64 = 0x1111111111111111
	m.PutHash("a", h)
	m.PutHash("b", h)
	m.PutNativeSize("a", image.Pt(100, 1))
	m.PutNativeSize("b", image.Pt(100, 1))

	g := m.Compute()

	if rep := g.RepresentativeOf(1); rep != 0 {
		t.Errorf("RepresentativeOf(1) = %d, want 0 (tie-break)", rep)
	}
}

func TestCompute_UnhashedFilesGetSizeZeroAndAreNeverRepresentative(t *testing.T) {
	set := newFakeSet(3, 1)
	m := New(set)
	m.PutHash("a", 5)
	m.PutHash("c", 5)
	// b stays unhashed.

	g := m.Compute()

	if g.Size(1) != 0 {
		t.Errorf("Size(1) = %d for an unhashed file, want 0", g.Size(1))
	}
	if g.RepresentativeOf(1) != 1 {
		t.Errorf("RepresentativeOf(1) = %d, want 1 (itself)", g.RepresentativeOf(1))
	}
	if g.RepresentativeOf(0) == 1 || g.RepresentativeOf(2) == 1 {
		t.Error("a hashed file's representative is the unhashed file, want never")
	}
}

func TestCompute_SnapshotsDistanceIntoGroups(t *testing.T) {
	m := New(newFakeSet(1, 1))
	m.SetDistance(10)

	g := m.Compute()

	if g.Dist != 10 {
		t.Errorf("Groups.Dist = %d, want 10", g.Dist)
	}
}

func TestCompute_WipesStaleHashesFirst(t *testing.T) {
	set := newFakeSet(2, 1)
	m := New(set)
	m.PutHash("a", 5)
	m.PutHash("b", 5)

	set.gen = 2
	g := m.Compute()

	if g.Size(0) != 0 || g.Size(1) != 0 {
		t.Errorf("Sizes = %v after a generation change, want both 0 (hashes wiped)", g.Sizes)
	}
}

func TestMembers_NilBelowTwo(t *testing.T) {
	set := newFakeSet(2, 1)
	m := New(set)
	m.PutHash("a", 5)
	m.PutHash("b", 0xFFFFFFFFFFFFFFFE)
	m.Rebuild()

	if got := m.Members(0); got != nil {
		t.Errorf("Members(0) = %v for a unique file, want nil", got)
	}
	if got := m.Members(1); got != nil {
		t.Errorf("Members(1) = %v for a unique file, want nil", got)
	}
}

func TestMembers_NilForUnhashedFile(t *testing.T) {
	set := newFakeSet(2, 1)
	m := New(set)
	m.PutHash("a", 5)
	// b stays unhashed.
	m.Rebuild()

	if got := m.Members(1); got != nil {
		t.Errorf("Members(1) = %v for an unhashed file, want nil", got)
	}
}

func TestMembers_AscendingOrderAtThree(t *testing.T) {
	set := newFakeSet(4, 1)
	m := New(set)
	m.PutHash("a", 1)
	m.PutHash("b", 3)
	m.PutHash("c", 2)
	m.PutHash("d", 0xFFFFFFFFFFFFFFFE) // unrelated, own group of 1
	m.Rebuild()

	want := []int{0, 1, 2}
	for _, i := range []int{0, 1, 2} {
		if got := m.Members(i); !slices.Equal(got, want) {
			t.Errorf("Members(%d) = %v, want %v", i, got, want)
		}
	}
	if got := m.Members(3); got != nil {
		t.Errorf("Members(3) = %v, want nil", got)
	}
}

func TestInstallAndRebuild(t *testing.T) {
	set := newFakeSet(2, 1)
	m := New(set)
	m.PutHash("a", 5)
	m.PutHash("b", 5)

	if got := m.GroupSize(0); got != 0 {
		t.Errorf("GroupSize(0) = %d before Install/Rebuild, want 0", got)
	}

	m.Rebuild()

	if got := m.GroupSize(0); got != 2 {
		t.Errorf("GroupSize(0) = %d after Rebuild, want 2", got)
	}

	installFixtureGroups(m, Groups{Sizes: []int{9, 9}, Reps: []int{1, 1}})
	if got := m.GroupSize(0); got != 9 {
		t.Errorf("GroupSize(0) = %d after a manual Install, want 9", got)
	}
	if got := m.RepresentativeOf(0); got != 1 {
		t.Errorf("RepresentativeOf(0) = %d after a manual Install, want 1", got)
	}
}

func TestComputes_CountsEachComputeCall(t *testing.T) {
	m := New(newFakeSet(1, 1))

	if got := m.Computes(); got != 0 {
		t.Errorf("Computes() = %d before any Compute, want 0", got)
	}

	m.Compute()
	m.Compute()

	if got := m.Computes(); got != 2 {
		t.Errorf("Computes() = %d after two Compute calls, want 2", got)
	}

	m.Rebuild() // Rebuild calls Compute once more.
	if got := m.Computes(); got != 3 {
		t.Errorf("Computes() = %d after Rebuild, want 3", got)
	}
}

// TestConcurrentPutHashAndCompute_NoRace exercises the exact scenario
// Compute's lock-release-relock shape exists for: hashing workers call
// Compute off the UI goroutine while other workers are still recording
// facts.
func TestConcurrentPutHashAndCompute_NoRace(t *testing.T) {
	set := newFakeSet(20, 1)
	m := New(set)

	var wg sync.WaitGroup
	for i := range 20 {
		wg.Go(func() {
			m.PutHash(set.keys[i], uint64(i))
			m.PutNativeSize(set.keys[i], image.Pt(i+1, i+1))
		})
	}
	for range 5 {
		wg.Go(func() {
			m.Compute()
		})
	}
	wg.Wait()

	m.Rebuild()
	if got := m.GroupSize(0); got == 0 {
		t.Error("GroupSize(0) = 0 after concurrent hashing and Compute, want a hashed file's group to be recorded")
	}
}

func TestGroupingReusesOnlyUnchangedAcceptedFacts(t *testing.T) {
	m := New(newFakeSet(2, 1))
	m.PutHash("a", 5)
	m.PutHash("b", 5)
	m.PutNativeSize("a", image.Pt(2, 2))
	if _, ok := m.CurrentGroups(); ok {
		t.Fatal("uncomputed groups reported current")
	}
	m.Rebuild()
	key, computes := m.GroupingKey(), m.Computes()
	for range 3 {
		m.PutHash("a", 5)
		m.PutNativeSize("a", image.Pt(2, 2))
		m.Rebuild()
	}
	if m.GroupingKey() != key || m.Computes() != computes {
		t.Errorf("unchanged grouping recomputed: %d -> %d", computes, m.Computes())
	}
	if g, ok := m.CurrentGroups(); !ok || g.Size(0) != 2 {
		t.Error("accepted groups were not reusable")
	}
	m.PutNativeSize("b", image.Pt(8, 8))
	if _, ok := m.CurrentGroups(); ok {
		t.Error("changed native facts still reported current")
	}
	m.Rebuild()
	if m.RepresentativeOf(0) != 1 || m.Computes() != computes+1 {
		t.Error("changed representative did not compute once")
	}
}

func TestGroupingRejectsObsoleteAndUntaggedSnapshots(t *testing.T) {
	changes := map[string]func(*Model, *fakeSet){
		"hash":        func(m *Model, _ *fakeSet) { m.PutHash("b", ^uint64(0)) },
		"native":      func(m *Model, _ *fakeSet) { m.PutNativeSize("b", image.Pt(30, 30)) },
		"distance":    func(m *Model, _ *fakeSet) { m.SetDistance(0) },
		"replacement": func(_ *Model, s *fakeSet) { s.gen++ },
		"adoption":    func(m *Model, s *fakeSet) { s.gen++; s.keys = s.keys[:1]; m.AdoptGeneration() },
		"reset":       func(m *Model, _ *fakeSet) { m.Clear() },
	}
	for name, change := range changes {
		t.Run(name, func(t *testing.T) {
			set := newFakeSet(2, 1)
			m := New(set)
			m.PutHash("a", 5)
			m.PutHash("b", 7)
			old := m.Compute()
			key := m.GroupingKey()
			if !m.Install(old) {
				t.Fatal("fresh snapshot rejected")
			}
			change(m, set)
			if key == m.GroupingKey() {
				t.Error("changed input reused grouping identity")
			}
			if _, ok := m.CurrentGroups(); ok {
				t.Error("old accepted groups reported current after input change")
			}
			if m.Install(old) {
				t.Error("obsolete snapshot installed")
			}
			fresh := m.Compute()
			if !m.Install(fresh) {
				t.Fatal("fresh snapshot rejected after input change")
			}
			if m.Install(Groups{Sizes: []int{99}}) {
				t.Error("untagged fabricated snapshot installed")
			}
			if got, ok := m.CurrentGroups(); !ok || !slices.Equal(got.Sizes, fresh.Sizes) || !slices.Equal(got.Reps, fresh.Reps) {
				t.Error("obsolete install replaced current groups")
			}
		})
	}
}

func TestGroupingCancelledComputationCannotInstall(t *testing.T) {
	m := New(newFakeSet(2, 1))
	m.PutHash("a", 5)
	m.PutHash("b", 5)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	g, err := m.ComputeContext(ctx)
	if !errors.Is(err, context.Canceled) || m.Install(g) {
		t.Errorf("cancelled compute published a snapshot: %v", err)
	}
}

// Visibility tests supply membership directly; tag it with their current
// fixture input without invoking the algorithm their assertions do not test.
func installFixtureGroups(m *Model, g Groups) {
	g.key, g.valid = m.GroupingKey(), true
	m.Install(g)
}

// benchmarkSet mirrors production's immutable published snapshot; fakeSet is
// unsuitable here because it rebuilds and copies its keys on every call.
type benchmarkSet struct{ snapshot Snapshot }

func (s benchmarkSet) Snapshot() Snapshot { return s.snapshot }

func benchmarkGroupingModel(n int, dense bool) *Model {
	keys := make([]string, n)
	for i := range keys {
		keys[i] = "image-" + strconv.Itoa(i)
	}
	m := New(benchmarkSet{snapshot: NewSnapshot(keys, 1)})
	facts := m.CaptureFacts()
	seed := uint64(1)
	for i, key := range keys {
		seed ^= seed << 13
		seed ^= seed >> 7
		seed ^= seed << 17
		hash := seed
		if dense {
			hash = 0xf0f0f0f0f0f0f0f0 ^ (uint64(1) << uint(i%64))
		}
		facts.PutHash(key, hash)
		facts.PutNativeSize(key, image.Pt(1+i%64, 1+i%64))
	}
	return m
}

func BenchmarkGrouping(b *testing.B) {
	for _, distribution := range []string{"unrelated", "dense"} {
		b.Run(distribution, func(b *testing.B) {
			for _, n := range []int{10000, 50000, 200000} {
				b.Run(strconv.Itoa(n), func(b *testing.B) {
					b.Run("cold", func(b *testing.B) {
						m := benchmarkGroupingModel(n, distribution == "dense")
						b.ReportAllocs()
						for b.Loop() {
							g := m.Compute()
							if len(g.Sizes) != n {
								b.Fatal("grouping lost files")
							}
							if distribution == "dense" && (g.Size(0) != n || g.RepresentativeOf(0) != 63) {
								b.Fatal("dense grouping changed membership or representative")
							}
						}
					})
					b.Run("reuse", func(b *testing.B) {
						m := benchmarkGroupingModel(n, distribution == "dense")
						m.Rebuild()
						b.ReportAllocs()
						for b.Loop() {
							m.Rebuild()
						}
						if m.Computes() != 1 {
							b.Fatal("unchanged accepted grouping recomputed")
						}
					})
				})
			}
		})
	}
}

func TestGroupingMatchesGreedyOracleAcrossDistributionsAndDistances(t *testing.T) {
	for _, n := range []int{0, 1, 8, 63, 257, 1024} {
		for _, distance := range []int{0, 1, 3, 4, 6, 7, 8, 16, 32} {
			t.Run(fmt.Sprintf("n=%d/distance=%d", n, distance), func(t *testing.T) {
				keys := make([]string, n)
				hashes := make([]uint64, n)
				known := make([]bool, n)
				native := make([]image.Point, n)
				seed := uint64(42)
				for i := range keys {
					keys[i] = strconv.Itoa(i)
					seed ^= seed << 13
					seed ^= seed >> 7
					seed ^= seed << 17
					hashes[i] = seed
					switch i % 6 {
					case 0:
						hashes[i] = 0
					case 1:
						hashes[i] = 0xf0f0f0f0f0f0f0f0
					case 2:
						hashes[i] = 0xf0f0f0f0f0f0f0f0 ^ (uint64(1) << uint(i%64))
					case 3:
						hashes[i] = 0xf0f0f0f0f0f0f0f0 ^ (seed & 255)
					}
					known[i] = i%11 != 0
					native[i] = image.Pt(int(seed%5), int((seed>>8)%5))
				}
				m := New(benchmarkSet{snapshot: NewSnapshot(keys, 1)})
				m.SetDistance(distance)
				facts := m.CaptureFacts()
				for i, key := range keys {
					if known[i] {
						facts.PutHash(key, hashes[i])
					}
					facts.PutNativeSize(key, native[i])
				}
				got := m.Compute()
				want := greedyGroupingOracle(hashes, known, native, distance)
				if !slices.Equal(got.Sizes, want.Sizes) || !slices.Equal(got.Reps, want.Reps) {
					t.Fatalf("group membership/representatives differ from original greedy complete linkage")
				}
			})
		}
	}
}

// This retains the original first-unassigned, forward-scan algorithm. It is
// intentionally independent of the indexed implementation and its helpers.
func greedyGroupingOracle(hashes []uint64, known []bool, native []image.Point, distance int) Groups {
	g := Groups{Sizes: make([]int, len(hashes)), Reps: make([]int, len(hashes))}
	assigned := make([]bool, len(hashes))
	for i := range hashes {
		g.Reps[i] = i
		if known[i] {
			g.Sizes[i] = 1
		}
	}
	for i, h := range hashes {
		if !known[i] || h == 0 || assigned[i] {
			continue
		}
		members := []int{i}
		assigned[i] = true
		for j := i + 1; j < len(hashes); j++ {
			if !known[j] || hashes[j] == 0 || assigned[j] {
				continue
			}
			fits := true
			for _, member := range members {
				if bits.OnesCount64(hashes[member]^hashes[j]) > distance {
					fits = false
					break
				}
			}
			if fits {
				members = append(members, j)
				assigned[j] = true
			}
		}
		representative := i
		for _, member := range members {
			if native[member].X*native[member].Y > native[representative].X*native[representative].Y {
				representative = member
			}
		}
		for _, member := range members {
			g.Sizes[member] = len(members)
			g.Reps[member] = representative
		}
	}
	return g
}

func TestGroupingIndexedCandidatesKeepGreedyCompleteLinkage(t *testing.T) {
	base := uint64(0x0123456789abcdef)
	cases := map[string][]uint64{
		"one changed bit in every projection": {base, base ^ 1 ^ (1 << 16) ^ (1 << 32) ^ (1 << 48)},
		"earliest compatible group":           {base ^ 7, base ^ (7 << 16), base},
		"every member must fit":               {base, base ^ 7, base ^ (7 << 16)},
	}
	for name, tail := range cases {
		t.Run(name, func(t *testing.T) {
			const prefix = 257
			n := prefix + len(tail)
			keys := make([]string, n)
			hashes := make([]uint64, n)
			known := make([]bool, n)
			native := make([]image.Point, n)
			seed := uint64(42)
			for i := range n {
				keys[i] = strconv.Itoa(i)
				known[i] = true
				native[i] = image.Pt(1+i%3, 1+i%3)
				seed ^= seed << 13
				seed ^= seed >> 7
				seed ^= seed << 17
				hashes[i] = seed
			}
			copy(hashes[prefix:], tail)
			m := New(benchmarkSet{snapshot: NewSnapshot(keys, 1)})
			m.SetDistance(4)
			facts := m.CaptureFacts()
			for i, key := range keys {
				facts.PutHash(key, hashes[i])
				facts.PutNativeSize(key, native[i])
			}
			got, want := m.Compute(), greedyGroupingOracle(hashes, known, native, 4)
			if !slices.Equal(got.Sizes, want.Sizes) || !slices.Equal(got.Reps, want.Reps) {
				t.Fatalf("indexed candidates changed original greedy membership or representatives")
			}
		})
	}
}

func TestGroupingReadersRejectDifferentFileIdentity(t *testing.T) {
	for _, reset := range []bool{false, true} {
		t.Run(fmt.Sprint(reset), func(t *testing.T) {
			set := &fakeSet{keys: []string{"a", "b"}}
			m := New(set)
			m.PutHash("a", 7)
			m.PutHash("b", 7)
			m.Rebuild()
			m.SetHideDuplicates(true)
			m.BeginInspect(0)
			if reset {
				m.Clear()
			} else {
				set.gen++
				m.AdoptGeneration()
			}
			if m.GroupSize(0) != 0 || m.RepresentativeOf(1) != 1 || len(m.Members(0)) != 0 || len(m.InspectMembers()) != 0 || m.IsHiddenExtra(1) {
				t.Fatal("old file identity still controls group readers")
			}
		})
	}
}
