package ui

import "testing"

func TestRevisionZeroValueAdvancesAndMatches(t *testing.T) {
	var r revision

	if got := r.current(); got != 0 {
		t.Fatalf("initial revision = %d, want 0", got)
	}
	first := r.advance()
	if first != 1 || !r.matches(first) {
		t.Fatalf("after advance: revision = %d, matches = %t; want 1, true", first, r.matches(first))
	}
	if r.matches(0) {
		t.Fatal("the zero revision must be stale after an advance")
	}
}
