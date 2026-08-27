package ui

import "testing"

// The jump off a hidden extra used to live in internal/ui/grid, where it
// reached the app through Host.ShowImage; it is production code here now
// (visibility.go) and the grid reaches it by firing the duplicate model's
// observers. These three tests are the ones that moved with it: the jump
// itself, the inspect guard that lets a variant committed out of the
// browse grid stay on screen, and the guard lifting again on ClearInspect.
//
// The wiring that makes the grid fire it - a hide toggle, a distance
// change, a hash pass landing - is asserted in step_test.go, which drives
// those transitions end to end.

func TestJumpIfHiddenExtra_MovesToRepresentative(t *testing.T) {
	v := hiddenExtraViewer(t)

	v.jumpIfHiddenExtra()
	waitUntilLoaded(t, v)

	if v.state.index != 0 {
		t.Errorf("index after jumpIfHiddenExtra = %d, want 0 (the representative)", v.state.index)
	}
}

func TestJumpIfHiddenExtra_NoopWhileInspecting(t *testing.T) {
	v := hiddenExtraViewer(t)
	v.grid.BeginInspect(1)
	if !v.dupes.Inspecting() {
		t.Fatal("premises: BeginInspect(1) should start an inspect session")
	}

	v.jumpIfHiddenExtra()
	waitUntilLoaded(t, v)

	if v.state.index != 1 {
		t.Errorf("index = %d, want 1 - inspect is the state where sitting on an extra is the point", v.state.index)
	}
}

func TestJumpIfHiddenExtra_JumpsAgainAfterClearInspect(t *testing.T) {
	v := hiddenExtraViewer(t)
	v.grid.BeginInspect(1)
	v.grid.ClearInspect()

	v.jumpIfHiddenExtra()
	waitUntilLoaded(t, v)

	if v.state.index != 0 {
		t.Errorf("index after ClearInspect and jumpIfHiddenExtra = %d, want 0", v.state.index)
	}
}

// hiddenExtraViewer parks the viewer on file 1 of the patterned triple with
// hide-duplicates on, which is exactly the state jumpIfHiddenExtra exists to
// undo: 0 and 1 are the same seed at the same size, so 0 is the
// representative and 1 the extra.
//
// Settle first, so the hashing pass the hide toggle started is fully drained
// before the display is moved: a landing snapshot fires the model's
// observers too, and waiting it out here is what keeps the jump under test
// the only one in the test.
func hiddenExtraViewer(t *testing.T) *viewer {
	t.Helper()

	v := loadPatternedTriple(t)
	v.grid.SetHideDuplicates(true)
	v.grid.Settle()
	waitUntilLoaded(t, v)
	if !v.dupes.IsHiddenExtra(1) || v.dupes.RepresentativeOf(1) != 0 {
		t.Fatalf("premises: extra(1)=%v rep(1)=%d, want true/0",
			v.dupes.IsHiddenExtra(1), v.dupes.RepresentativeOf(1))
	}

	// ShowImage does not consult visibility - the grid, the favorites list
	// and a jump all go through it - so this is how the viewer legitimately
	// ends up parked on an extra.
	v.ShowImage(1)
	waitUntilLoaded(t, v)
	if v.state.index != 1 {
		t.Fatalf("premises: index = %d, want 1 (the extra) before the jump", v.state.index)
	}

	return v
}
