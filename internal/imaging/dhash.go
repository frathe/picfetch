package imaging

import (
	"image"
	"image/color"
	"math/bits"

	"golang.org/x/image/draw"
)

// DuplicateMaxDistance is the default Hamming threshold at or below which
// two dHashes count as the same shot. 10 is the usual dHash near-duplicate
// cutoff: bursts and re-exports match, unrelated photos do not. The
// settings slider may pass a different maxDist into DuplicateGroups.
const DuplicateMaxDistance = 10

const dhashWide, dhashHigh = 9, 8

// DifferenceHash is a 64-bit dHash of img: luma resized to 9×8, then one
// bit per adjacent horizontal pair (8 rows × 8 comparisons). Uniform
// images have no horizontal gradient and hash to 0, so two different
// solid colors collide — callers that need “not a duplicate” fixtures
// must use patterned pixels, not solid JPEGs.
func DifferenceHash(img image.Image) uint64 {
	dst := image.NewGray(image.Rect(0, 0, dhashWide, dhashHigh))
	draw.ApproxBiLinear.Scale(dst, dst.Bounds(), img, img.Bounds(), draw.Src, nil)

	var h uint64
	var bit uint64
	for y := range dhashHigh {
		for x := range dhashWide - 1 {
			left, _ := color.GrayModel.Convert(dst.At(x, y)).(color.Gray)
			right, _ := color.GrayModel.Convert(dst.At(x+1, y)).(color.Gray)
			if right.Y > left.Y {
				h |= 1 << bit
			}
			bit++
		}
	}
	return h
}

// Hamming is the number of bits that differ between two dHashes.
func Hamming(a, b uint64) int {
	return bits.OnesCount64(a ^ b)
}

// DuplicateGroups partitions indices into groups of near-duplicates.
// Each group has a representative at the lowest index; every other
// member is within maxDist Hamming distance of that representative.
// Membership is not transitive: A~B and B~C do not put A and C in the
// same group unless both are within maxDist of the representative.
// Groups of size 1 are omitted.
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
		if assigned[i] >= 0 {
			continue
		}
		grp := []int{i}
		assigned[i] = i
		for j := i + 1; j < n; j++ {
			if assigned[j] >= 0 {
				continue
			}
			if Hamming(hashes[i], hashes[j]) <= maxDist {
				assigned[j] = i
				grp = append(grp, j)
			}
		}
		if len(grp) >= 2 {
			out = append(out, grp)
		}
	}
	return out
}
