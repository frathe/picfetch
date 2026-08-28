package dupes

import (
	"slices"
	"testing"
)

func TestHideDuplicates_DefaultsToFalse(t *testing.T) {
	m := New(newFakeSet(1, 1))

	if m.HideDuplicates() {
		t.Error("HideDuplicates() = true on a new Model, want false")
	}
}

func TestSetHideDuplicates_ReturnsWhetherItChanged(t *testing.T) {
	m := New(newFakeSet(1, 1))

	if changed := m.SetHideDuplicates(true); !changed {
		t.Error("SetHideDuplicates(true) on a model with hide off = false, want true")
	}
	if !m.HideDuplicates() {
		t.Error("HideDuplicates() = false after SetHideDuplicates(true), want true")
	}

	if changed := m.SetHideDuplicates(true); changed {
		t.Error("SetHideDuplicates(true) on a model already on = true, want false (no-op)")
	}

	if changed := m.SetHideDuplicates(false); !changed {
		t.Error("SetHideDuplicates(false) on a model with hide on = false, want true")
	}
	if m.HideDuplicates() {
		t.Error("HideDuplicates() = true after SetHideDuplicates(false), want false")
	}
}

// TestSetHideDuplicates_DoesNotRebuildGroupsOrNotify is the invariant
// Stage 5 depends on: SetHideDuplicates must be a pure flag setter, with
// hashing/filtering/jumping/notifying left to whatever observes the
// change.
func TestSetHideDuplicates_DoesNotRebuildGroupsOrNotify(t *testing.T) {
	m := New(newFakeSet(3, 1))
	var fired bool
	m.OnChange(func() { fired = true })
	before := m.Computes()

	m.SetHideDuplicates(true)

	if got := m.Computes(); got != before {
		t.Errorf("Computes() = %d after SetHideDuplicates, want unchanged at %d: it must not rebuild groups", got, before)
	}
	if fired {
		t.Error("an OnChange observer fired after SetHideDuplicates, want none: notifying is Stage 5's job")
	}
}

func TestBeginInspect_And_Inspecting_And_ClearInspect(t *testing.T) {
	m := New(newFakeSet(3, 1))

	if m.Inspecting() {
		t.Error("Inspecting() = true on a new Model, want false")
	}

	m.BeginInspect(1)

	if !m.Inspecting() {
		t.Error("Inspecting() = false after BeginInspect, want true")
	}

	m.ClearInspect()

	if m.Inspecting() {
		t.Error("Inspecting() = true after ClearInspect, want false")
	}
	if src := m.InspectSource(); src != -1 {
		t.Errorf("InspectSource() = %d after ClearInspect, want -1", src)
	}
}

func TestBeginInspect_OutOfRangeClearsInspect(t *testing.T) {
	m := New(newFakeSet(2, 1))
	m.BeginInspect(0)
	if !m.Inspecting() {
		t.Fatal("Inspecting() = false after BeginInspect(0), want true")
	}

	m.BeginInspect(5)

	if m.Inspecting() {
		t.Error("Inspecting() = true after BeginInspect with an out-of-range index, want false (cleared)")
	}
}

func TestBeginInspect_NegativeIndexClearsInspect(t *testing.T) {
	m := New(newFakeSet(2, 1))
	m.BeginInspect(0)

	m.BeginInspect(-1)

	if m.Inspecting() {
		t.Error("Inspecting() = true after BeginInspect(-1), want false (cleared)")
	}
}

// TestBeginInspect_EmptyKeyClearsInspect covers this package's equivalent
// of the original grid code's nil-URI guard: KeyAt returning "" for the
// requested index must clear inspect rather than store the empty key.
func TestBeginInspect_EmptyKeyClearsInspect(t *testing.T) {
	set := &fakeSet{keys: []string{"a", ""}, gen: 1}
	m := New(set)
	m.BeginInspect(0)
	if !m.Inspecting() {
		t.Fatal("Inspecting() = false after BeginInspect(0), want true")
	}

	m.BeginInspect(1)

	if m.Inspecting() {
		t.Error("Inspecting() = true after BeginInspect at an empty-key index, want false (cleared)")
	}
}

func TestInspectSource_NegativeOneWhenInspectIsOff(t *testing.T) {
	m := New(newFakeSet(3, 1))

	if src := m.InspectSource(); src != -1 {
		t.Errorf("InspectSource() = %d with inspect off, want -1", src)
	}
}

// TestInspectSource_FollowsKeyAfterFileSetReorders proves InspectSource
// scans for the stored key rather than trusting the index it was stored
// at - the entire reason inspect is a key, not an index.
func TestInspectSource_FollowsKeyAfterFileSetReorders(t *testing.T) {
	set := newFakeSet(3, 1) // keys: a, b, c
	m := New(set)
	m.BeginInspect(1) // "b"

	if src := m.InspectSource(); src != 1 {
		t.Fatalf("InspectSource() = %d before reorder, want 1", src)
	}

	set.keys = []string{"b", "c", "a"} // b moved from index 1 to index 0

	if src := m.InspectSource(); src != 0 {
		t.Errorf("InspectSource() = %d after reorder, want 0 (follows the key, not the old index)", src)
	}
}

func TestInspectSource_NegativeOneWhenKeyIsGone(t *testing.T) {
	set := newFakeSet(3, 1) // keys: a, b, c
	m := New(set)
	m.BeginInspect(1) // "b"

	set.keys = []string{"a", "c"} // b removed entirely

	if src := m.InspectSource(); src != -1 {
		t.Errorf("InspectSource() = %d after the inspected key was removed, want -1", src)
	}
}

func TestInspectMembers_NilWhenInspectOff(t *testing.T) {
	m := New(newFakeSet(3, 1))

	if got := m.InspectMembers(); got != nil {
		t.Errorf("InspectMembers() = %v with inspect off, want nil", got)
	}
}

func TestInspectMembers_FullGroupWhenOn(t *testing.T) {
	set := newFakeSet(3, 1) // keys: a, b, c
	m := New(set)
	m.Install(Groups{Sizes: []int{2, 2, 0}, Reps: []int{0, 0, 2}})
	m.BeginInspect(1)

	want := []int{0, 1}
	if got := m.InspectMembers(); !slices.Equal(got, want) {
		t.Errorf("InspectMembers() = %v, want %v", got, want)
	}
}

func TestInspectMembers_NilWhenInspectedFileIsGone(t *testing.T) {
	set := newFakeSet(3, 1)
	m := New(set)
	m.Install(Groups{Sizes: []int{2, 2, 0}, Reps: []int{0, 0, 2}})
	m.BeginInspect(1) // "b"

	set.keys = []string{"a", "c"} // b removed

	if got := m.InspectMembers(); got != nil {
		t.Errorf("InspectMembers() = %v after the inspected file was removed, want nil", got)
	}
}

// TestInspectMembers_ReadsSnapshotWithoutRebuild proves InspectMembers only
// reads the already-installed snapshot: it must never trigger a fresh
// Compute, which would be expensive linkage work run on every step while
// inspecting.
func TestInspectMembers_ReadsSnapshotWithoutRebuild(t *testing.T) {
	set := newFakeSet(3, 1) // keys: a, b, c
	m := New(set)
	m.Install(Groups{Sizes: []int{2, 2, 0}, Reps: []int{0, 0, 2}})
	m.BeginInspect(1)

	got := m.InspectMembers()
	if len(got) != 2 || got[0] != 0 || got[1] != 1 {
		t.Fatalf("InspectMembers() = %v, want [0 1]", got)
	}

	before := m.Computes()
	_ = m.InspectMembers()
	if n := m.Computes(); n != before {
		t.Fatalf("InspectMembers incremented Computes() from %d to %d", before, n)
	}
}

func TestIsHiddenExtra_FalseWhenHideOff(t *testing.T) {
	m := New(newFakeSet(3, 1))
	m.Install(Groups{Sizes: []int{2, 2, 0}, Reps: []int{0, 0, 2}})
	// hide stays off

	if m.IsHiddenExtra(1) {
		t.Error("IsHiddenExtra(1) = true with hide off, want false")
	}
}

func TestIsHiddenExtra_FalseForRepresentative(t *testing.T) {
	m := New(newFakeSet(3, 1))
	m.Install(Groups{Sizes: []int{2, 2, 0}, Reps: []int{0, 0, 2}})
	m.SetHideDuplicates(true)

	if m.IsHiddenExtra(0) {
		t.Error("IsHiddenExtra(0) = true for the group representative, want false")
	}
}

func TestIsHiddenExtra_TrueForNonRepresentativeMember(t *testing.T) {
	m := New(newFakeSet(3, 1))
	m.Install(Groups{Sizes: []int{2, 2, 0}, Reps: []int{0, 0, 2}})
	m.SetHideDuplicates(true)

	if !m.IsHiddenExtra(1) {
		t.Error("IsHiddenExtra(1) = false for a non-representative member, want true")
	}
}

func TestIsHiddenExtra_FalseForUnhashedFile(t *testing.T) {
	m := New(newFakeSet(3, 1))
	m.Install(Groups{Sizes: []int{2, 2, 0}, Reps: []int{0, 0, 2}})
	m.SetHideDuplicates(true)

	if m.IsHiddenExtra(2) {
		t.Error("IsHiddenExtra(2) = true for an unhashed file (size 0), want false")
	}
}

// TestIsHiddenExtra_ChainDoesNotHideUnrelated proves the model surfaces
// imaging.DuplicateGroups' star clustering correctly through IsHiddenExtra:
// a chain of hashes that are pairwise near but not all mutually near must
// not collapse into one group.
func TestIsHiddenExtra_ChainDoesNotHideUnrelated(t *testing.T) {
	set := newFakeSet(3, 1) // a, b, c
	m := New(set)
	m.PutHash("a", 1<<63)
	m.PutHash("b", 1<<63|0x3FF)
	m.PutHash("c", 1<<63|0xFFFFF)
	// A literal 10: this fixture is built at Hamming 10/10/20 to exercise
	// linkage, so it pins the threshold it was written for rather than
	// tracking the shipped default.
	m.SetDistance(10)
	m.Rebuild()

	m.SetHideDuplicates(true)

	if !m.IsHiddenExtra(1) {
		t.Error("B is within distance 10 of A and must be an extra")
	}
	if m.IsHiddenExtra(2) {
		t.Error("C is Hamming 20 from A and must not be hidden as A's extra")
	}
	if got := m.RepresentativeOf(2); got != 2 {
		t.Errorf("RepresentativeOf(2) = %d, want 2", got)
	}
	if got := m.GroupSize(2); got != 1 {
		t.Errorf("GroupSize(2) = %d, want 1 (C is hashed-and-unique, not unhashed)", got)
	}
	if got := m.GroupSize(0); got != 2 {
		t.Errorf("GroupSize(0) = %d, want 2 (not the whole set)", got)
	}
}

// TestIsHiddenExtra_HubSpokesDoNotHideUnrelated is
// TestIsHiddenExtra_ChainDoesNotHideUnrelated's hub-and-spoke shape: two
// spokes that are each near the hub but Hamming 20 from each other must
// group with the hub one at a time, never both at once.
func TestIsHiddenExtra_HubSpokesDoNotHideUnrelated(t *testing.T) {
	set := newFakeSet(3, 1) // hub, spoke-a, spoke-b
	m := New(set)
	const hub uint64 = 0xFFFF000000000000
	m.PutHash("a", hub)
	m.PutHash("b", hub^0x3FF)
	m.PutHash("c", hub^(0x3FF<<10))
	m.SetDistance(10)
	m.Rebuild()

	m.SetHideDuplicates(true)

	if got := m.GroupSize(0); got != 2 {
		t.Fatalf("GroupSize(0) = %d, want 2 (one spoke, not both)", got)
	}
	if m.IsHiddenExtra(2) {
		t.Error("spoke B is 20 from spoke A and must not hide as hub's extra")
	}
	if m.RepresentativeOf(2) == 0 {
		t.Error("spoke B must not list the hub as representative")
	}
}

func TestIsVisible_IsNegationOfIsHiddenExtra(t *testing.T) {
	m := New(newFakeSet(3, 1))
	m.Install(Groups{Sizes: []int{2, 2, 0}, Reps: []int{0, 0, 2}})
	m.SetHideDuplicates(true)

	if !m.IsVisible(0) {
		t.Error("IsVisible(0) = false for the representative, want true")
	}
	if m.IsVisible(1) {
		t.Error("IsVisible(1) = true for a hidden extra, want false")
	}
}

// TestVisibility_AgreesWithPerIndexAccessors proves the batched Visibility
// value answers exactly what the existing per-index accessors already did,
// for every kind of index this package distinguishes: a group
// representative, that group's hidden extra, a hashed-and-unique file, an
// unhashed file, and both flavors of out-of-range index.
func TestVisibility_AgreesWithPerIndexAccessors(t *testing.T) {
	set := newFakeSet(4, 1) // a, b, c, d
	m := New(set)
	// 0: representative of a group of two. 1: that group's hidden extra.
	// 2: hashed and unique. 3: unhashed (size 0).
	m.Install(Groups{Sizes: []int{2, 2, 1, 0}, Reps: []int{0, 0, 2, 3}})
	indexes := []int{-1, 0, 1, 2, 3, set.Snapshot().Count()}

	for _, hide := range []bool{true, false} {
		m.SetHideDuplicates(hide)
		vis := m.Visibility()

		for _, i := range indexes {
			if got, want := vis.HiddenExtra(i), m.IsHiddenExtra(i); got != want {
				t.Errorf("hide=%v: vis.HiddenExtra(%d) = %v, want %v (must agree with IsHiddenExtra)", hide, i, got, want)
			}
			if got, want := vis.Visible(i), m.IsVisible(i); got != want {
				t.Errorf("hide=%v: vis.Visible(%d) = %v, want %v (must agree with IsVisible)", hide, i, got, want)
			}
			if got, want := vis.RepresentativeOf(i), m.RepresentativeOf(i); got != want {
				t.Errorf("hide=%v: vis.RepresentativeOf(%d) = %d, want %d (must agree with Model.RepresentativeOf)", hide, i, got, want)
			}
			if got, want := vis.Size(i), m.GroupSize(i); got != want {
				t.Errorf("hide=%v: vis.Size(%d) = %d, want %d (must agree with GroupSize)", hide, i, got, want)
			}
		}
	}
}

// TestVisibility_IsAFrozenRead proves a held Visibility keeps answering off
// the hide flag and Groups it was read with, even after the model's hide
// flag flips and a different Groups is installed - Install replaces the
// struct wholesale rather than mutating it in place, so the old value's
// slices are never touched.
func TestVisibility_IsAFrozenRead(t *testing.T) {
	set := newFakeSet(3, 1) // a, b, c
	m := New(set)
	m.Install(Groups{Sizes: []int{2, 2, 0}, Reps: []int{0, 0, 2}})
	m.SetHideDuplicates(true)

	vis := m.Visibility()

	m.SetHideDuplicates(false)
	m.Install(Groups{Sizes: []int{0, 2, 2}, Reps: []int{0, 1, 1}})

	if !vis.Hide {
		t.Error("vis.Hide = false after the model's hide flag flipped, want true (frozen at read time)")
	}
	if !vis.HiddenExtra(1) {
		t.Error("vis.HiddenExtra(1) = false after the model's groups were replaced, want true (frozen at read time)")
	}
	if got := vis.RepresentativeOf(0); got != 0 {
		t.Errorf("vis.RepresentativeOf(0) = %d after the model's groups were replaced, want 0 (frozen at read time)", got)
	}
	if got := vis.Size(0); got != 2 {
		t.Errorf("vis.Size(0) = %d after the model's groups were replaced, want 2 (frozen at read time)", got)
	}
}

// TestVisibility_TakesOneModelReadPerCall is the regression guard for the
// whole point of this type: a caller testing many indices must pay one
// model-mutex acquisition, not one per index.
func TestVisibility_TakesOneModelReadPerCall(t *testing.T) {
	m := New(newFakeSet(3, 1))
	before := m.VisibilityReads()

	vis := m.Visibility()

	if got, want := m.VisibilityReads(), before+1; got != want {
		t.Errorf("VisibilityReads() = %d after one Visibility() call, want %d", got, want)
	}

	for i := range 64 {
		_ = vis.HiddenExtra(i)
		_ = vis.Visible(i)
		_ = vis.RepresentativeOf(i)
		_ = vis.Size(i)
	}

	if got, want := m.VisibilityReads(), before+1; got != want {
		t.Errorf("VisibilityReads() = %d after testing 64 indices off one held value, want unchanged at %d", got, want)
	}

	m.Visibility()

	if got, want := m.VisibilityReads(), before+2; got != want {
		t.Errorf("VisibilityReads() = %d after a second Visibility() call, want %d", got, want)
	}
}

// TestNextVisible_StepsWithinInspectMembersRing is branch 1: inspecting a
// group of two or more overrides everything else, in both step
// directions, wrapping at either end of the ring. Hide is deliberately
// left off to prove branch 1 is checked before branch 2.
func TestNextVisible_StepsWithinInspectMembersRing(t *testing.T) {
	set := newFakeSet(5, 1) // a b c d e
	m := New(set)
	// Group {0, 2, 4} sharing representative 0; 1 and 3 belong elsewhere.
	m.Install(Groups{
		Sizes: []int{3, 0, 3, 0, 3},
		Reps:  []int{0, 1, 0, 3, 0},
	})
	m.BeginInspect(0)

	if got := m.NextVisible(0, 1); got != 2 {
		t.Errorf("NextVisible(0, 1) = %d, want 2 (next member in the ring)", got)
	}
	if got := m.NextVisible(4, 1); got != 0 {
		t.Errorf("NextVisible(4, 1) = %d, want 0 (ring wraps forward)", got)
	}
	if got := m.NextVisible(0, -1); got != 4 {
		t.Errorf("NextVisible(0, -1) = %d, want 4 (ring wraps backward)", got)
	}
	if got := m.NextVisible(2, -1); got != 0 {
		t.Errorf("NextVisible(2, -1) = %d, want 0 (previous member in the ring)", got)
	}
}

// TestNextVisible_DeltaZeroReturnsFromUnchanged is branch 2's delta==0
// half - it fires even while inspecting, because branch 1 requires
// delta != 0.
func TestNextVisible_DeltaZeroReturnsFromUnchanged(t *testing.T) {
	m := New(newFakeSet(5, 1))
	m.SetHideDuplicates(true)

	if got := m.NextVisible(3, 0); got != 3 {
		t.Errorf("NextVisible(3, 0) = %d, want 3 (delta 0 is a no-op)", got)
	}
}

// TestNextVisible_HideOffReturnsUnclampedFromPlusDelta is branch 2's
// hide-off half. The unclamped, possibly out-of-range result is
// deliberate: ShowImage in the viewer is what normalises it. Do not
// "fix" this into a bounds check.
func TestNextVisible_HideOffReturnsUnclampedFromPlusDelta(t *testing.T) {
	set := newFakeSet(3, 1) // Count() == 3
	m := New(set)
	// hide stays off

	if got := m.NextVisible(2, 5); got != 7 {
		t.Errorf("NextVisible(2, 5) = %d, want 7 (unclamped from+delta past Count())", got)
	}
	if got := m.NextVisible(0, -1); got != -1 {
		t.Errorf("NextVisible(0, -1) = %d, want -1 (unclamped, negative)", got)
	}
}

// TestNextVisible_SkipsHiddenExtras is branch 3's ordinary case: it walks
// past a hidden extra to land on the next visible file.
func TestNextVisible_SkipsHiddenExtras(t *testing.T) {
	set := newFakeSet(4, 1) // a b c d
	m := New(set)
	// Group {0, 1}: rep 0, so 1 is a hidden extra. 2 and 3 are unhashed.
	m.Install(Groups{Sizes: []int{2, 2, 0, 0}, Reps: []int{0, 0, 2, 3}})
	m.SetHideDuplicates(true)

	if got := m.NextVisible(0, 1); got != 2 {
		t.Errorf("NextVisible(0, 1) = %d, want 2 (skips hidden extra at 1)", got)
	}
}

// TestNextVisible_SkipsHiddenExtrasBackward is branch 3's negative-delta
// walk: it must step backward and still skip a hidden extra.
func TestNextVisible_SkipsHiddenExtrasBackward(t *testing.T) {
	set := newFakeSet(4, 1) // a b c d
	m := New(set)
	// Group {2, 3}: rep 3, so 2 is a hidden extra. 0 and 1 are unhashed.
	m.Install(Groups{Sizes: []int{0, 0, 2, 2}, Reps: []int{0, 1, 3, 3}})
	m.SetHideDuplicates(true)

	if got := m.NextVisible(3, -1); got != 1 {
		t.Errorf("NextVisible(3, -1) = %d, want 1 (skips hidden extra at 2)", got)
	}
}

// TestNextVisible_AllHiddenReturnsFrom is branch 3's wrap-around case.
// The installed Reps deliberately point outside the current file set (a
// stale snapshot), which is the only way to make every index, including
// from itself, a hidden extra: a real group always keeps one member -
// its representative - visible.
func TestNextVisible_AllHiddenReturnsFrom(t *testing.T) {
	set := newFakeSet(2, 1)
	m := New(set)
	m.Install(Groups{Sizes: []int{2, 2}, Reps: []int{5, 5}})
	m.SetHideDuplicates(true)

	if got := m.NextVisible(0, 1); got != 0 {
		t.Errorf("NextVisible(0, 1) = %d, want 0 (from unchanged: nothing is visible)", got)
	}
}

// TestNextVisible_EmptyFileSetReturnsZero covers the n == 0 guard ported
// from nextVisibleIndex's own top-of-function check.
func TestNextVisible_EmptyFileSetReturnsZero(t *testing.T) {
	m := New(newFakeSet(0, 1))

	if got := m.NextVisible(0, 1); got != 0 {
		t.Errorf("NextVisible(0, 1) = %d on an empty file set, want 0", got)
	}
}

// One arrow key is one snapshot, however many indices the walk skips.
// Before this, InspectSource rescanned the whole set for the inspect key
// on every step, and the skip-hidden-extras walk took the model mutex
// once per candidate.
func TestNextVisible_TakesOneSnapshotPerCall(t *testing.T) {
	set := &countingSet{inner: newFakeSet(64, 1)}
	m := New(set)
	set.snapshots = 0

	m.NextVisible(0, 1)

	if set.snapshots != 1 {
		t.Errorf("NextVisible took %d snapshots, want 1", set.snapshots)
	}
}

// The test above reaches branch 2 only, where the walk never runs. These
// two hold the same one-snapshot budget on the other two branches: the
// inspect ring, and the walk that skips hidden extras index by index.
func TestNextVisible_TakesOneSnapshotWhileInspecting(t *testing.T) {
	set := &countingSet{inner: newFakeSet(4, 1)}
	m := New(set)
	// Group {0, 2}: rep 0. 1 and 3 are unhashed.
	m.Install(Groups{Sizes: []int{2, 0, 2, 0}, Reps: []int{0, 1, 0, 3}})
	m.BeginInspect(0)
	set.snapshots = 0

	if got := m.NextVisible(0, 1); got != 2 {
		t.Fatalf("NextVisible(0, 1) = %d, want 2 (next member in the ring)", got)
	}
	if set.snapshots != 1 {
		t.Errorf("NextVisible took %d snapshots, want 1", set.snapshots)
	}
}

func TestNextVisible_TakesOneSnapshotWhileSkippingHiddenExtras(t *testing.T) {
	set := &countingSet{inner: newFakeSet(4, 1)}
	m := New(set)
	// Group {0, 2}: rep 0, so 2 is a hidden extra. 1 and 3 are unhashed.
	m.Install(Groups{Sizes: []int{2, 0, 2, 0}, Reps: []int{0, 1, 0, 3}})
	m.SetHideDuplicates(true)
	set.snapshots = 0

	if got := m.NextVisible(1, 1); got != 3 {
		t.Fatalf("NextVisible(1, 1) = %d, want 3 (skips the hidden extra at 2)", got)
	}
	if set.snapshots != 1 {
		t.Errorf("NextVisible took %d snapshots, want 1", set.snapshots)
	}
}

// TestStepInMembers_EmptyMembersReturnsFrom and the two tests below call
// stepInMembers directly: NextVisible's branch 1 guard (len(members) >= 2
// && delta != 0) never lets an empty or zero-delta case reach it, so
// those defensive branches - ported verbatim from viewer.go - are only
// reachable as direct calls to the helper.
func TestStepInMembers_EmptyMembersReturnsFrom(t *testing.T) {
	if got := stepInMembers(nil, 5, 1); got != 5 {
		t.Errorf("stepInMembers(nil, 5, 1) = %d, want 5", got)
	}
}

func TestStepInMembers_DeltaZeroReturnsFrom(t *testing.T) {
	if got := stepInMembers([]int{1, 2, 3}, 2, 0); got != 2 {
		t.Errorf("stepInMembers(members, 2, 0) = %d, want 2", got)
	}
}

// TestStepInMembers_Wraps came across from internal/ui's step_test.go
// with the helper itself: the ring walk is this package's code now, and
// the viewer only asks for it through NextVisible.
func TestStepInMembers_Wraps(t *testing.T) {
	members := []int{0, 3, 5}
	if got := stepInMembers(members, 3, 1); got != 5 {
		t.Errorf("step +1 from 3 = %d, want 5", got)
	}
	if got := stepInMembers(members, 5, 1); got != 0 {
		t.Errorf("wrap +1 from 5 = %d, want 0", got)
	}
	if got := stepInMembers(members, 0, -1); got != 5 {
		t.Errorf("wrap -1 from 0 = %d, want 5", got)
	}
	if got := stepInMembers(members, 99, 1); got != 3 {
		t.Errorf("from missing, +1 from pos 0 = %d, want 3", got)
	}
}

// TestStepInMembers_FromNotInMembersStartsAtFirst covers the !found
// branch: when from is not itself a member, the walk starts as if
// positioned just before the first member.
func TestStepInMembers_FromNotInMembersStartsAtFirst(t *testing.T) {
	if got := stepInMembers([]int{2, 4, 6}, 99, 1); got != 4 {
		t.Errorf("stepInMembers(members, 99, 1) = %d, want 4 (starts at index 0, steps once)", got)
	}
}

func TestFirstVisible_SkipsHiddenFirstFile(t *testing.T) {
	set := newFakeSet(3, 1)
	m := New(set)
	// 0 is a hidden extra of the group represented by 1.
	m.Install(Groups{Sizes: []int{2, 2, 0}, Reps: []int{1, 1, 2}})
	m.SetHideDuplicates(true)

	if got := m.FirstVisible(); got != 1 {
		t.Errorf("FirstVisible() = %d, want 1 (0 is a hidden extra)", got)
	}
}

func TestLastVisible_SkipsHiddenLastFile(t *testing.T) {
	set := newFakeSet(3, 1)
	m := New(set)
	// 2 is a hidden extra of the group represented by 1.
	m.Install(Groups{Sizes: []int{0, 2, 2}, Reps: []int{0, 1, 1}})
	m.SetHideDuplicates(true)

	if got := m.LastVisible(); got != 1 {
		t.Errorf("LastVisible() = %d, want 1 (2 is a hidden extra)", got)
	}
}

func TestFirstVisible_LastVisible_HideOffDoNotFilter(t *testing.T) {
	set := newFakeSet(3, 1)
	m := New(set)
	// 0 and 2 would be extras if hide were on; it is not.
	m.Install(Groups{Sizes: []int{2, 2, 0}, Reps: []int{1, 1, 2}})

	if got := m.FirstVisible(); got != 0 {
		t.Errorf("FirstVisible() = %d with hide off, want 0 (no filtering)", got)
	}
	if got := m.LastVisible(); got != 2 {
		t.Errorf("LastVisible() = %d with hide off, want 2 (no filtering)", got)
	}
}

// TestFirstVisible_LastVisible_FallBackToZeroWhenNothingVisible uses the
// same out-of-range-Reps trick as TestNextVisible_AllHiddenReturnsFrom to
// make every index a hidden extra.
func TestFirstVisible_LastVisible_FallBackToZeroWhenNothingVisible(t *testing.T) {
	set := newFakeSet(2, 1)
	m := New(set)
	m.Install(Groups{Sizes: []int{2, 2}, Reps: []int{5, 5}})
	m.SetHideDuplicates(true)

	if got := m.FirstVisible(); got != 0 {
		t.Errorf("FirstVisible() = %d with nothing visible, want 0 (fallback)", got)
	}
	if got := m.LastVisible(); got != 0 {
		t.Errorf("LastVisible() = %d with nothing visible, want 0 (fallback)", got)
	}
}

func TestVisibleIndexesExcept_ExcludesCurrentAndHiddenExtras(t *testing.T) {
	set := newFakeSet(5, 1)
	m := New(set)
	// Group {1, 3}: rep 1, so 3 is a hidden extra. 0, 2, 4 are unhashed.
	m.Install(Groups{Sizes: []int{0, 2, 0, 2, 0}, Reps: []int{0, 1, 2, 1, 4}})
	m.SetHideDuplicates(true)

	got := m.VisibleIndexesExcept(1)
	want := []int{0, 2, 4}
	if !slices.Equal(got, want) {
		t.Errorf("VisibleIndexesExcept(1) = %v, want %v", got, want)
	}
}

func TestVisibleIndexesExcept_HideOffOnlyExcludesCurrent(t *testing.T) {
	m := New(newFakeSet(3, 1))

	got := m.VisibleIndexesExcept(1)
	want := []int{0, 2}
	if !slices.Equal(got, want) {
		t.Errorf("VisibleIndexesExcept(1) = %v, want %v", got, want)
	}
}
