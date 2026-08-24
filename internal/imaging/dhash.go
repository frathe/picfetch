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

// DuplicateGroups returns connected components of indices whose hashes
// are within maxDist Hamming distance. Groups of size 1 are omitted.
func DuplicateGroups(hashes []uint64, maxDist int) [][]int {
	n := len(hashes)
	if n == 0 {
		return nil
	}
	if maxDist < 0 {
		maxDist = 0
	}
	parent := make([]int, n)
	for i := range parent {
		parent[i] = i
	}
	var find func(int) int
	find = func(i int) int {
		if parent[i] != i {
			parent[i] = find(parent[i])
		}
		return parent[i]
	}
	union := func(a, b int) {
		ra, rb := find(a), find(b)
		if ra != rb {
			parent[rb] = ra
		}
	}
	for i := range n {
		for j := i + 1; j < n; j++ {
			if Hamming(hashes[i], hashes[j]) <= maxDist {
				union(i, j)
			}
		}
	}
	buckets := make(map[int][]int, n)
	order := make([]int, 0, n)
	for i := range n {
		r := find(i)
		if _, ok := buckets[r]; !ok {
			order = append(order, r)
		}
		buckets[r] = append(buckets[r], i)
	}
	var out [][]int
	for _, r := range order {
		if len(buckets[r]) >= 2 {
			out = append(out, buckets[r])
		}
	}
	return out
}
