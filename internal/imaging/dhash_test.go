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
