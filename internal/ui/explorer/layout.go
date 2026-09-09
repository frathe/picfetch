package explorer

import (
	"math"
	"sort"

	"fyne.io/fyne/v2"
)

const (
	pileWidth  float32 = 380
	pileHeight float32 = 310
	pileGap    float32 = 48
)

// Match by shared source identities, because a cohort's membership-derived ID
// changes during discovery. Continuing piles stay exactly where the user left
// them; new piles are placed near related groups without moving those anchors.
func anchorPiles(piles, previous []*Pile) {
	type match struct {
		current, old int
		score        float32
	}
	var matches []match
	projected := make([]fyne.Position, len(piles))
	oldOwners := map[string]int{}
	oldSizes := make([]int, len(previous))
	for i, p := range previous {
		for _, path := range p.members {
			if _, exists := oldOwners[path]; !exists {
				oldOwners[path] = i
				oldSizes[i]++
			}
		}
	}
	for i, p := range piles {
		projected[i] = p.world
		shared := map[int]int{}
		seen := map[string]bool{}
		for _, path := range p.members {
			if seen[path] {
				continue
			}
			seen[path] = true
			if old, ok := oldOwners[path]; ok {
				shared[old]++
			}
		}
		for old, count := range shared {
			matches = append(matches, match{i, old, float32(count) / float32(len(seen)+oldSizes[old]-count)})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]
		if a.score != b.score {
			return a.score > b.score
		}
		if a.current != b.current {
			return a.current < b.current
		}
		return a.old < b.old
	})
	used, oldUsed := map[int]bool{}, map[int]bool{}
	var ordered []*Pile
	var anchors []int
	for _, pair := range matches {
		if used[pair.current] || oldUsed[pair.old] {
			continue
		}
		used[pair.current], oldUsed[pair.old] = true, true
		p := piles[pair.current]
		p.world = previous[pair.old].world
		ordered = append(ordered, p)
		anchors = append(anchors, pair.current)
	}
	for i, p := range piles {
		if used[i] {
			continue
		}
		nearest, distance := -1, float32(math.Inf(1))
		for _, anchor := range anchors {
			delta := projected[i].Subtract(projected[anchor])
			if d := delta.X*delta.X + delta.Y*delta.Y; d < distance {
				nearest, distance = anchor, d
			}
		}
		if nearest >= 0 {
			p.world = piles[nearest].world.Add(projected[i].Subtract(projected[nearest]))
		}
		ordered = append(ordered, p)
	}
	placePiles(ordered)
}

// placePiles keeps the requested position when free, otherwise chooses a nearby
// free position. Earlier piles remain anchored; every resulting pair has a gap.
func placePiles(piles []*Pile) {
	var placed []*Pile
	for _, p := range piles {
		origin := p.world
		free := func(at fyne.Position) bool {
			for _, other := range placed {
				if abs(at.X-other.world.X) < pileWidth+pileGap && abs(at.Y-other.world.Y) < pileHeight+pileGap {
					return false
				}
			}
			return true
		}
		for ring := 1; !free(p.world); ring++ {
			best := float32(math.Inf(1))
			for x := -ring; x <= ring; x++ {
				for y := -ring; y <= ring; y++ {
					if x != -ring && x != ring && y != -ring && y != ring {
						continue
					}
					dx, dy := float32(x)*(pileWidth+pileGap), float32(y)*(pileHeight+pileGap)
					at := origin.Add(fyne.NewPos(dx, dy))
					if score := dx*dx + dy*dy; score < best && free(at) {
						p.world, best = at, score
					}
				}
			}
		}
		placed = append(placed, p)
	}
}

func abs(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

// Projection axes have no intrinsic screen orientation. Put their main spread
// along the window's longer dimension before packing, avoiding a tiny column of
// piles in an otherwise empty landscape window (or the converse in portrait).
func orientPiles(piles []*Pile, size fyne.Size) {
	if len(piles) < 2 {
		return
	}
	var x, y float64
	for _, p := range piles {
		x += float64(p.world.X)
		y += float64(p.world.Y)
	}
	x, y = x/float64(len(piles)), y/float64(len(piles))
	var xx, yy, xy float64
	for _, p := range piles {
		dx, dy := float64(p.world.X)-x, float64(p.world.Y)-y
		xx, yy, xy = xx+dx*dx, yy+dy*dy, xy+dx*dy
	}
	angle := .5 * math.Atan2(2*xy, xx-yy)
	if size.Height > size.Width {
		angle -= math.Pi / 2
	}
	c, s := math.Cos(angle), math.Sin(angle)
	for _, p := range piles {
		dx, dy := float64(p.world.X)-x, float64(p.world.Y)-y
		p.world = fyne.NewPos(float32(c*dx+s*dy), float32(-s*dx+c*dy))
	}
}
