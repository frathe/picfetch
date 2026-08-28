package dupes

import "testing"

func TestNewSnapshot_CountKeyAtAndGeneration(t *testing.T) {
	s := NewSnapshot([]string{"a", "b", "c"}, 7)

	if got := s.Count(); got != 3 {
		t.Errorf("Count() = %d, want 3", got)
	}
	if got := s.KeyAt(1); got != "b" {
		t.Errorf("KeyAt(1) = %q, want %q", got, "b")
	}
	if got := s.Generation(); got != 7 {
		t.Errorf("Generation() = %d, want 7", got)
	}
}

func TestSnapshotKeyAt_OutOfRangeIsEmpty(t *testing.T) {
	s := NewSnapshot([]string{"a"}, 1)

	for _, i := range []int{-1, 1, 99} {
		if got := s.KeyAt(i); got != "" {
			t.Errorf("KeyAt(%d) = %q, want %q", i, got, "")
		}
	}
}

func TestSnapshotIndexOf(t *testing.T) {
	s := NewSnapshot([]string{"a", "b", "a"}, 1)

	tests := []struct {
		key  string
		want int
	}{
		{"a", 0}, // duplicates keep the lowest index, matching the scan this replaces
		{"b", 1},
		{"zzz", -1},
		{"", -1},
	}
	for _, tt := range tests {
		if got := s.IndexOf(tt.key); got != tt.want {
			t.Errorf("IndexOf(%q) = %d, want %d", tt.key, got, tt.want)
		}
	}
}

// The whole point of the type: a caller mutating its own slice afterwards
// must not be able to change what the snapshot reports.
func TestNewSnapshot_CopiesKeys(t *testing.T) {
	keys := []string{"a", "b"}
	s := NewSnapshot(keys, 1)

	keys[0] = "mutated"

	if got := s.KeyAt(0); got != "a" {
		t.Errorf("KeyAt(0) = %q after caller mutation, want %q", got, "a")
	}
}

func TestZeroSnapshot_IsAValidEmptySet(t *testing.T) {
	var s Snapshot

	if got := s.Count(); got != 0 {
		t.Errorf("Count() = %d, want 0", got)
	}
	if got := s.KeyAt(0); got != "" {
		t.Errorf("KeyAt(0) = %q, want %q", got, "")
	}
	if got := s.Generation(); got != 0 {
		t.Errorf("Generation() = %d, want 0", got)
	}
	if got := s.IndexOf("a"); got != -1 {
		t.Errorf("IndexOf(\"a\") = %d, want -1", got)
	}
}
