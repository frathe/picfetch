package imaging

import "context"

// DuplicateGroups partitions nonzero hashes by greedy complete linkage.
// Members and groups retain first-seen order; singletons are omitted.
func DuplicateGroups(hashes []uint64, maxDist int) [][]int {
	groups, _ := DuplicateGroupsContext(context.Background(), hashes, maxDist)
	return groups
}

// DuplicateGroupsContext checks bounded work intervals and never returns a
// partial partition after cancellation. A later hash joins the earliest group
// compatible with every distinct member. Adding members can only reject more
// candidates, so this is equivalent to the original first-unassigned scan.
func DuplicateGroupsContext(ctx context.Context, hashes []uint64, maxDist int) ([][]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(hashes) == 0 {
		return nil, nil
	}
	maxDist = max(0, maxDist)
	g := hashGrouping{ctx: ctx, distance: maxDist, capacity: len(hashes), groups: make([]hashGroup, 0, min(len(hashes), 64))}
	assigned := make(map[uint64]int, min(len(hashes), 1024))
	for i, h := range hashes {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if ungroupable(h) {
			continue
		}
		if id, ok := assigned[h]; ok {
			g.groups[id].appendMember(i)
			continue
		}
		id, err := g.find(h, i+1)
		if err != nil {
			return nil, err
		}
		if id == len(g.groups) {
			g.add(h, i)
		} else {
			g.groups[id].appendMember(i)
			g.groups[id].others = append(g.groups[id].others, h)
		}
		assigned[h] = id
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var out [][]int
	for _, group := range g.groups {
		if len(group.members) > 1 {
			out = append(out, group.members)
		}
	}
	return out, nil
}

type hashGroup struct {
	first   int
	anchor  uint64
	others  []uint64
	members []int
	next    [4]int
	seen    int
}

func (g *hashGroup) appendMember(index int) {
	if len(g.members) == 0 {
		g.members = append(g.members, g.first)
	}
	g.members = append(g.members, index)
}

// The projection index is allocated only once enough distinct groups need it.
// Heads and links use group index + 1, leaving zero as the empty-list sentinel.
// Dense inputs often need just one group and never allocate these heads.
type hashGrouping struct {
	ctx                      context.Context
	distance, capacity, work int
	groups                   []hashGroup
	heads                    *[4][1 << 16]int
}

func (g *hashGrouping) checkpoint() error {
	g.work++
	if g.work&255 == 0 {
		return g.ctx.Err()
	}
	return nil
}

func (g *hashGrouping) fits(id int, h uint64) (bool, error) {
	if err := g.checkpoint(); err != nil {
		return false, err
	}
	group := &g.groups[id]
	if Hamming(group.anchor, h) > g.distance {
		return false, nil
	}
	for _, other := range group.others {
		if err := g.checkpoint(); err != nil {
			return false, err
		}
		if Hamming(other, h) > g.distance {
			return false, nil
		}
	}
	return true, nil
}

func (g *hashGrouping) find(h uint64, stamp int) (int, error) {
	if g.distance == 0 {
		return len(g.groups), nil
	} // equal hashes already joined
	if g.distance >= 64 && len(g.groups) > 0 {
		return 0, nil
	}
	if g.heads == nil {
		for id := range g.groups {
			fits, err := g.fits(id, h)
			if err != nil || fits {
				return id, err
			}
		}
		return len(g.groups), nil
	}
	best := len(g.groups)
	for part := range 4 {
		key := uint16(h >> uint(16*part))
		if err := g.visit(part, key, h, stamp, &best); err != nil {
			return 0, err
		}
		if g.distance >= 4 {
			for bit := range 16 {
				if err := g.visit(part, key^(uint16(1)<<uint(bit)), h, stamp, &best); err != nil {
					return 0, err
				}
			}
		}
	}
	return best, nil
}

// Within distance 0..3 at least one 16-bit block matches; within 4..7 at
// least one differs by at most one bit. These are candidates only: the full
// distance and complete-linkage checks still decide membership.
func (g *hashGrouping) visit(part int, key uint16, h uint64, stamp int, best *int) error {
	for entry := g.heads[part][key]; entry != 0; entry = g.groups[entry-1].next[part] {
		if err := g.checkpoint(); err != nil {
			return err
		}
		id := entry - 1
		if id >= *best || g.groups[id].seen == stamp {
			continue
		}
		g.groups[id].seen = stamp
		fits, err := g.fits(id, h)
		if err != nil {
			return err
		}
		if fits {
			*best = id
		}
	}
	return nil
}

func (g *hashGrouping) add(h uint64, index int) {
	if g.heads == nil && g.distance >= 1 && g.distance <= 7 && g.capacity >= 256 && len(g.groups) == 64 {
		g.heads = new([4][1 << 16]int)
		groups := make([]hashGroup, len(g.groups), g.capacity)
		copy(groups, g.groups)
		g.groups = groups
		for id := range g.groups {
			g.index(id)
		}
	}
	g.groups = append(g.groups, hashGroup{first: index, anchor: h})
	if g.heads != nil {
		g.index(len(g.groups) - 1)
	}
}

func (g *hashGrouping) index(id int) {
	group := &g.groups[id]
	for part := range 4 {
		key := uint16(group.anchor >> uint(16*part))
		group.next[part] = g.heads[part][key]
		g.heads[part][key] = id + 1
	}
}
