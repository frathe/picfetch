// Package fileidentity identifies repeated paths within an ordered file list.
// These bookmarks describe list positions, independently of filesystem versions.
package fileidentity

import "slices"

// Occurrence identifies the zero-based ordinal of a path in a captured list.
type Occurrence struct {
	Path    string
	Ordinal int
}

// Index is an immutable snapshot of path occurrences. Its zero value is empty.
type Index struct {
	byPath map[string][]int
}

// NewIndex captures each nonempty path once. Empty paths represent entries
// outside the caller's scope, such as nil sources or unrelated bookmarks.
func NewIndex(count int, pathAt func(int) string) Index {
	index := Index{byPath: make(map[string][]int)}
	for i := range count {
		if path := pathAt(i); path != "" {
			index.byPath[path] = append(index.byPath[path], i)
		}
	}
	return index
}

// Capture returns the occurrence at a known collection position.
func (x Index) Capture(path string, position int) (Occurrence, bool) {
	ordinal, found := slices.BinarySearch(x.byPath[path], position)
	if !found {
		return Occurrence{}, false
	}
	return Occurrence{Path: path, Ordinal: ordinal}, true
}

// Resolve returns the exact occurrence's position, or -1 when it is absent.
// Choosing a different occurrence as fallback belongs to the caller.
func (x Index) Resolve(identity Occurrence) int {
	positions := x.byPath[identity.Path]
	if identity.Ordinal < 0 || identity.Ordinal >= len(positions) {
		return -1
	}
	return positions[identity.Ordinal]
}
