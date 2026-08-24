package imaging

import (
	"image"
	"image/color"
	"slices"
	"testing"
)

func patterned(w, h, seed int) *image.Gray {
	im := image.NewGray(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			im.SetGray(x, y, color.Gray{Y: uint8((x*13 + y*7 + seed*31) & 0xff)})
		}
	}
	return im
}

func TestDifferenceHash_IdenticalImagesHaveDistanceZero(t *testing.T) {
	a := patterned(64, 48, 1)
	if Hamming(DifferenceHash(a), DifferenceHash(a)) != 0 {
		t.Fatal("identical images must hash with Hamming 0")
	}
}

func TestDifferenceHash_DifferentPatternsAreFar(t *testing.T) {
	a, b := patterned(64, 48, 1), patterned(64, 48, 99)
	if d := Hamming(DifferenceHash(a), DifferenceHash(b)); d <= DuplicateMaxDistance {
		t.Fatalf("Hamming(different) = %d, want > %d", d, DuplicateMaxDistance)
	}
}

func TestDifferenceHash_UniformImagesHashToZero(t *testing.T) {
	white := image.NewGray(image.Rect(0, 0, 32, 32))
	for y := range 32 {
		for x := range 32 {
			white.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	black := image.NewGray(image.Rect(0, 0, 32, 32))
	if DifferenceHash(white) != 0 || DifferenceHash(black) != 0 {
		t.Fatal("uniform images must dHash to 0")
	}
}

func TestHamming_KnownBits(t *testing.T) {
	if Hamming(0, 0) != 0 || Hamming(0, 1) != 1 || Hamming(0, ^uint64(0)) != 64 {
		t.Fatal("Hamming known bits")
	}
}

func TestDuplicateGroups_ClustersByHamming(t *testing.T) {
	got := DuplicateGroups([]uint64{0, 1, ^uint64(0)}, 2)
	if len(got) != 1 || !slices.Equal(got[0], []int{0, 1}) {
		t.Fatalf("got %v, want [[0 1]]", got)
	}
}

func TestDuplicateGroups_SingletonsOmitted(t *testing.T) {
	if got := DuplicateGroups([]uint64{0, ^uint64(0)}, 0); len(got) != 0 {
		t.Fatalf("got %v, want no groups", got)
	}
}

func TestDuplicateGroups_NegativeMaxDistTreatedAsZero(t *testing.T) {
	if got := DuplicateGroups([]uint64{0, 1}, -3); len(got) != 0 {
		t.Fatalf("got %v, want no groups at exact-only", got)
	}
}

const (
	chainA uint64 = 0
	chainB uint64 = 0x3FF   // bits 0–9;  Hamming(A,B)=10
	chainC uint64 = 0xFFFFF // bits 0–19; Hamming(B,C)=10, Hamming(A,C)=20
)

func TestDuplicateGroups_ChainIsNotTransitive(t *testing.T) {
	// A~B and B~C at distance 10, A far from C (20). Union-find would
	// emit one group of three; a star around A must keep C out.
	got := DuplicateGroups([]uint64{chainA, chainB, chainC}, 10)
	if len(got) != 1 || !slices.Equal(got[0], []int{0, 1}) {
		t.Fatalf("got %v, want [[0 1]] (C must not join A's star)", got)
	}
}

func TestDuplicateGroups_ChainJoinsWhenWithinRepDistance(t *testing.T) {
	got := DuplicateGroups([]uint64{chainA, chainB, chainC}, 20)
	if len(got) != 1 || !slices.Equal(got[0], []int{0, 1, 2}) {
		t.Fatalf("got %v, want [[0 1 2]] at maxDist 20", got)
	}
}

func TestDuplicateGroups_MultipleStarsKeepFirstSeenOrder(t *testing.T) {
	got := DuplicateGroups([]uint64{0, 1, ^uint64(0), ^uint64(0) ^ 1}, 2)
	if len(got) != 2 || !slices.Equal(got[0], []int{0, 1}) || !slices.Equal(got[1], []int{2, 3}) {
		t.Fatalf("got %v, want [[0 1] [2 3]]", got)
	}
}

func TestDuplicateGroups_LaterNearDupesDoNotJoinEarlierFarRep(t *testing.T) {
	// X claims A (dist 10); A' is dist 1 from A but 11 from X, so A' is a singleton.
	const x, a, aPrime uint64 = 0, 0x3FF, 0x7FF
	if Hamming(x, a) != 10 || Hamming(a, aPrime) != 1 || Hamming(x, aPrime) != 11 {
		t.Fatalf("fixture Hamming: X-A=%d A-A'=%d X-A'=%d, want 10, 1, 11",
			Hamming(x, a), Hamming(a, aPrime), Hamming(x, aPrime))
	}
	got := DuplicateGroups([]uint64{x, a, aPrime}, 10)
	if len(got) != 1 || !slices.Equal(got[0], []int{0, 1}) {
		t.Fatalf("got %v, want [[0 1]] (A' must not join X's star)", got)
	}
}

func TestDuplicateGroups_SeparatedHashesStaySingletons(t *testing.T) {
	hashes := []uint64{
		0x0000000000000000,
		0xFFFF000000000000,
		0x0000FFFF00000000,
		0x00000000FFFF0000,
		0x000000000000FFFF,
		0x00FF00FF00FF00FF,
		0xFF00FF00FF00FF00,
		0xF0F0F0F0F0F0F0F0,
	}
	got := DuplicateGroups(hashes, DuplicateMaxDistance)
	if len(got) != 0 {
		t.Fatalf("got %v, want no groups (each hash is far from the others)", got)
	}
}
