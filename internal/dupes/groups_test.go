package dupes

import (
	"image"
	"slices"
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

	m.Install(Groups{Sizes: []int{9, 9}, Reps: []int{1, 1}})
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
