package imaging

import (
	"image"
	"image/color"
	"math/bits"
)

// DuplicateMaxDistance is the default Hamming threshold at or below which
// two dHashes count as the same shot. The settings slider may pass a
// different maxDist into DuplicateGroups.
//
// 6, not the 10 usually quoted for dHash. That folklore figure assumes a
// hash that drifts under re-encoding, which this one no longer does: over
// a 13k-image library, a JPEG re-export or a downscale to a quarter size
// moves the hash by at most 7 bits and by 4 at the 99th percentile, so the
// extra slack in 10 buys almost no genuine matches while admitting a great
// many wrong ones. Measured on that library, moving 10 → 6 dropped false
// pairs from 1150 to 66 and the largest wrong group from 26 files to 4,
// at a cost of 17 correct pairs out of 5930.
const DuplicateMaxDistance = 6

const dhashWide, dhashHigh = 9, 8

// DifferenceHash is a 64-bit dHash of img: luma reduced to a 9×8 grid, then
// one bit per adjacent horizontal pair (8 rows × 8 comparisons). Uniform
// images have no horizontal gradient and hash to 0, so two different solid
// colors collide — callers that need “not a duplicate” fixtures must use
// patterned pixels, not solid JPEGs.
func DifferenceHash(img image.Image) uint64 {
	grid := dhashLuma(img)

	var h uint64
	var bit uint64
	for y := range dhashHigh {
		for x := range dhashWide - 1 {
			if grid[y*dhashWide+x+1] > grid[y*dhashWide+x] {
				h |= 1 << bit
			}
			bit++
		}
	}
	return h
}

// dhashLuma reduces img to a 9×8 grid of average luma, every source pixel
// counting toward exactly one cell.
//
// Averaging every pixel is the whole point, and it is why this does not
// reach for draw.ApproxBiLinear the way the rest of the package does.
// That interpolator's cost — and therefore its sample count — is
// independent of the source size, so reducing a 200px thumbnail to 9×8
// read four pixels per cell out of the ~500 the cell covers. Any picture
// whose subject is thin against a wide flat background (a sketch, a
// screenshot, a logo, anything on white) sampled as background nearly
// everywhere and hashed to a handful of bits; since two hashes are always
// within Hamming popcount(a)+popcount(b) of each other, every such picture
// then matched every other one regardless of content.
func dhashLuma(img image.Image) [dhashWide * dhashHigh]uint8 {
	var grid [dhashWide * dhashHigh]uint8

	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return grid
	}

	for gy := range dhashHigh {
		y0, y1 := cellSpan(gy, dhashHigh, h)
		for gx := range dhashWide {
			x0, x1 := cellSpan(gx, dhashWide, w)

			var sum, n uint32
			for y := y0; y < y1; y++ {
				for x := x0; x < x1; x++ {
					c, _ := color.GrayModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.Gray)
					sum += uint32(c.Y)
					n++
				}
			}
			grid[gy*dhashWide+gx] = uint8(sum / n)
		}
	}
	return grid
}

// cellSpan is the half-open source range that cell i of n covers across
// size pixels. Never empty: a source narrower or shorter than the grid
// reuses a pixel rather than leaving a cell unsampled and black.
func cellSpan(i, n, size int) (int, int) {
	lo := i * size / n
	hi := (i + 1) * size / n
	if hi <= lo {
		hi = lo + 1
	}
	return lo, hi
}

// Hamming is the number of bits that differ between two dHashes.
func Hamming(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

// ungroupable reports hashes that must not participate in DuplicateGroups.
// DifferenceHash returns 0 for any image with no horizontal gradient at
// all, which says nothing about what the image contains: a solid red and a
// solid blue both hash to 0 and are not the same picture.
//
// This is a guard against a genuinely featureless image, not against the
// near-empty hashes that used to come out of the downsample — those are
// dhashLuma's problem and were never exactly 0. Over a 13k-image library
// this rule matches no file at all; it earns its place only for the solid
// colour that would otherwise collect every other solid colour.
func ungroupable(h uint64) bool {
	return h == 0
}

// DuplicateGroups partitions indices into groups of near-duplicates.
// Each group has a representative at the lowest index. A later file
// joins only if it is within maxDist of *every* current member
// (complete linkage), so neighbors of the first file that are far
// from each other do not become one giant group. Hash 0 (uniform
// images) is omitted; groups of size 1 are omitted.
func DuplicateGroups(hashes []uint64, maxDist int) [][]int {
	n := len(hashes)
	if n == 0 {
		return nil
	}
	if maxDist < 0 {
		maxDist = 0
	}

	assigned := make([]int, n)
	for i := range assigned {
		assigned[i] = -1
	}

	var out [][]int
	for i := range n {
		if assigned[i] >= 0 || ungroupable(hashes[i]) {
			continue
		}
		grp := []int{i}
		assigned[i] = i
		for j := i + 1; j < n; j++ {
			if assigned[j] >= 0 || ungroupable(hashes[j]) {
				continue
			}
			fits := true
			for _, m := range grp {
				if Hamming(hashes[m], hashes[j]) > maxDist {
					fits = false
					break
				}
			}
			if !fits {
				continue
			}
			assigned[j] = i
			grp = append(grp, j)
		}
		if len(grp) >= 2 {
			out = append(out, grp)
		}
	}
	return out
}
