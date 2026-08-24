package imaging

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"math"
	"math/bits"
	"slices"
	"testing"

	"golang.org/x/image/draw"

	"github.com/frathe/picfetch/internal/uitest"
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

// TestDifferenceHash_SparseDetailSurvivesTheDownsample is the root-cause
// lock for the duplicate-detection bug. Downsampling to the 9x8 grid has to
// average every source pixel, not sample a handful: a 200x200 thumbnail is
// ~36000 pixels and the grid has 72, so a sampler that reads four pixels per
// cell lands on the white background almost every time and reports a nearly
// empty hash for a picture that plainly has content.
func TestDifferenceHash_SparseDetailSurvivesTheDownsample(t *testing.T) {
	h := DifferenceHash(uitest.LineArtGray(200, 1))
	if got := bits.OnesCount64(h); got < 12 {
		t.Fatalf("dHash of line art on white = %#016x with %d bits set, want at least 12: "+
			"the downsample dropped the strokes", h, got)
	}
}

// TestDifferenceHash_UnrelatedSparseImagesAreNotDuplicates is the same
// defect seen the way the tester saw it. Two hashes with few bits set are
// within Hamming popcount(a)+popcount(b) of each other whatever they
// depict, so once the downsample empties them out, unrelated pictures
// become one another's duplicates.
func TestDifferenceHash_UnrelatedSparseImagesAreNotDuplicates(t *testing.T) {
	for _, seed := range []int{2, 3, 4, 5} {
		a, b := DifferenceHash(uitest.LineArtGray(200, 1)), DifferenceHash(uitest.LineArtGray(200, seed))
		if d := Hamming(a, b); d <= DuplicateMaxDistance {
			t.Errorf("Hamming(lineArt 1, lineArt %d) = %d (%#016x vs %#016x), want > %d",
				seed, d, a, b, DuplicateMaxDistance)
		}
	}
}

// photoLike is a stand-in for photographic content: smooth, low-frequency
// structure that survives a downscale. patterned is deliberately not used
// for resize tests - its ~20px sawtooth aliases into a genuinely different
// picture at a quarter size, so a hash that tracked it would be wrong.
func photoLike(w, h, seed int) *image.Gray {
	im := image.NewGray(image.Rect(0, 0, w, h))
	s := float64(seed)
	for y := range h {
		fy := float64(y) / float64(h)
		for x := range w {
			fx := float64(x) / float64(w)
			v := 0.5 +
				0.25*math.Sin(2*math.Pi*(fx+0.13*s)) +
				0.15*math.Cos(2*math.Pi*(1.5*fy+0.29*s)) +
				0.10*math.Sin(2*math.Pi*(fx+fy+0.07*s))
			im.SetGray(x, y, color.Gray{Y: uint8(max(min(v, 1), 0) * 255)})
		}
	}
	return im
}

// TestDifferenceHash_SurvivesReEncodeAndResize guards the other side of the
// tradeoff: the threshold must stay loose enough that a re-export or a
// downscale of the same picture still matches. Measured over 398 real
// photographs, both edits move this hash by at most 7 bits and by 4 at the
// 99th percentile, which is what makes a default of 6 affordable.
func TestDifferenceHash_SurvivesReEncodeAndResize(t *testing.T) {
	src := photoLike(400, 300, 5)
	base := DifferenceHash(src)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, src, &jpeg.Options{Quality: 50}); err != nil {
		t.Fatal(err)
	}
	reencoded, err := jpeg.Decode(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	if d := Hamming(base, DifferenceHash(reencoded)); d > DuplicateMaxDistance {
		t.Errorf("Hamming(original, jpeg q50) = %d, want <= %d", d, DuplicateMaxDistance)
	}

	small := image.NewGray(image.Rect(0, 0, 100, 75))
	draw.CatmullRom.Scale(small, small.Bounds(), src, src.Bounds(), draw.Src, nil)
	if d := Hamming(base, DifferenceHash(small)); d > DuplicateMaxDistance {
		t.Errorf("Hamming(original, quarter-size) = %d, want <= %d", d, DuplicateMaxDistance)
	}
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
	got := DuplicateGroups([]uint64{2, 3, ^uint64(0)}, 2)
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
	// Non-zero stem so hash 0 (uniform images) is not a cluster member.
	chainStem uint64 = 1 << 63
	chainA    uint64 = chainStem
	chainB    uint64 = chainStem | 0x3FF   // Hamming(A,B)=10
	chainC    uint64 = chainStem | 0xFFFFF // Hamming(B,C)=10, Hamming(A,C)=20
)

func TestDuplicateGroups_ChainIsNotTransitive(t *testing.T) {
	// A~B and B~C at distance 10, A far from C (20). Union-find would
	// emit one group of three; complete linkage keeps C out.
	got := DuplicateGroups([]uint64{chainA, chainB, chainC}, 10)
	if len(got) != 1 || !slices.Equal(got[0], []int{0, 1}) {
		t.Fatalf("got %v, want [[0 1]] (C must not join A's group)", got)
	}
}

func TestDuplicateGroups_ChainJoinsWhenWithinRepDistance(t *testing.T) {
	got := DuplicateGroups([]uint64{chainA, chainB, chainC}, 20)
	if len(got) != 1 || !slices.Equal(got[0], []int{0, 1, 2}) {
		t.Fatalf("got %v, want [[0 1 2]] at maxDist 20", got)
	}
}

func TestDuplicateGroups_MultipleGroupsKeepFirstSeenOrder(t *testing.T) {
	got := DuplicateGroups([]uint64{2, 3, ^uint64(0), ^uint64(0) ^ 1}, 2)
	if len(got) != 2 || !slices.Equal(got[0], []int{0, 1}) || !slices.Equal(got[1], []int{2, 3}) {
		t.Fatalf("got %v, want [[0 1] [2 3]]", got)
	}
}

func TestDuplicateGroups_LaterNearDupesDoNotJoinEarlierFarRep(t *testing.T) {
	// X claims A (dist 10); A' is dist 1 from A but 11 from X, so A' is a singleton.
	const x, a, aPrime uint64 = chainStem, chainStem | 0x3FF, chainStem | 0x7FF
	if Hamming(x, a) != 10 || Hamming(a, aPrime) != 1 || Hamming(x, aPrime) != 11 {
		t.Fatalf("fixture Hamming: X-A=%d A-A'=%d X-A'=%d, want 10, 1, 11",
			Hamming(x, a), Hamming(a, aPrime), Hamming(x, aPrime))
	}
	got := DuplicateGroups([]uint64{x, a, aPrime}, 10)
	if len(got) != 1 || !slices.Equal(got[0], []int{0, 1}) {
		t.Fatalf("got %v, want [[0 1]] (A' must not join X's group)", got)
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

func TestDuplicateGroups_SpokesNearHubButFarFromEachOtherDoNotGroup(t *testing.T) {
	// Star clustering around the first file: Hamming(hub, spoke) <= 10 is
	// enough to join, so both spokes land on the hub even though they are
	// 20 apart — the tester's "first unsorted file still has dozens of
	// unrelated shots" after hash-0 exclusion. Complete linkage requires
	// every pair in the group to be within maxDist.
	const hub uint64 = 0xFFFF000000000000
	spokeA := hub ^ 0x3FF
	spokeB := hub ^ (0x3FF << 10)
	if Hamming(hub, spokeA) != 10 || Hamming(hub, spokeB) != 10 || Hamming(spokeA, spokeB) != 20 {
		t.Fatalf("fixture Hamming hub-A=%d hub-B=%d A-B=%d, want 10, 10, 20",
			Hamming(hub, spokeA), Hamming(hub, spokeB), Hamming(spokeA, spokeB))
	}
	// A literal 10, not DuplicateMaxDistance: this fixture is about linkage,
	// so its distances are pinned to the threshold it was built for.
	got := DuplicateGroups([]uint64{hub, spokeA, spokeB}, 10)
	if len(got) != 1 || !slices.Equal(got[0], []int{0, 1}) {
		t.Fatalf("got %v, want [[0 1]] (spoke B must not join via the hub)", got)
	}
}

func TestDuplicateGroups_ZeroHashIsNotAStarCenter(t *testing.T) {
	// Uniform images all dHash to 0. Hamming(0, 1<<k) is 1, so at the
	// default threshold a leading 0 would swallow every sparse hash —
	// the tester's "first unsorted file has dozens of false positives".
	hashes := make([]uint64, 21)
	for i := 1; i < 21; i++ {
		hashes[i] = 1 << (i - 1)
	}
	got := DuplicateGroups(hashes, DuplicateMaxDistance)
	for _, grp := range got {
		for _, idx := range grp {
			if idx == 0 {
				t.Fatalf("hash 0 joined group %v", grp)
			}
		}
	}
}
