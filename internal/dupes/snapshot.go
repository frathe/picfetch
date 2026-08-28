// The immutable view of a file set that every Model method reads through.

package dupes

// Snapshot is a FileSet frozen at one generation: the file keys in index
// order, the generation they belong to, and a key-to-index map.
//
// It exists because the model is read from hashing workers while the UI
// goroutine replaces the file set underneath them. The Count()/KeyAt(i)
// pair this replaces could not be read consistently: a shrink landing
// between the two, or part-way through a loop over them, indexed past the
// end of a slice whose header was itself being written unsynchronized.
// A method that takes one Snapshot at its top holds a count and a key
// list that cannot disagree.
//
// byKey is what makes IndexOf O(1), which is what makes InspectSource -
// and so every arrow key while inspecting - cheap enough to call per
// keystroke on a 50k-file drop.
//
// The zero value is a valid empty snapshot: Count 0, generation 0, no
// keys. That is what a viewer with no files loaded publishes.
type Snapshot struct {
	keys  []string
	byKey map[string]int
	gen   uint64
}

// NewSnapshot builds a Snapshot over keys at generation gen. keys is
// copied, so the caller keeps ownership of its own slice and the
// snapshot stays immutable however that slice is mutated later.
//
// A duplicate key keeps its lowest index, which is what the linear scan
// this replaces returned (InspectSource stopped at the first match).
func NewSnapshot(keys []string, gen uint64) Snapshot {
	cp := append([]string(nil), keys...)
	byKey := make(map[string]int, len(cp))
	for i, k := range cp {
		if _, seen := byKey[k]; !seen {
			byKey[k] = i
		}
	}

	return Snapshot{keys: cp, byKey: byKey, gen: gen}
}

// Count is how many files the snapshot holds.
func (s Snapshot) Count() int { return len(s.keys) }

// KeyAt is the key at i, or "" when i is out of range - the same empty
// key an absent URI produced before, which every caller already reads as
// "no facts are stored about this index".
func (s Snapshot) KeyAt(i int) string {
	if i < 0 || i >= len(s.keys) {
		return ""
	}

	return s.keys[i]
}

// Generation is the file-set revision these keys belong to. A change
// invalidates stored hashes - see Model.WipeIfStale.
func (s Snapshot) Generation() uint64 { return s.gen }

// IndexOf is the index of key, or -1 when key is empty or absent.
func (s Snapshot) IndexOf(key string) int {
	if key == "" {
		return -1
	}
	i, ok := s.byKey[key]
	if !ok {
		return -1
	}

	return i
}
