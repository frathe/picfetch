package ui

import (
	"slices"

	"github.com/frathe/picfetch/internal/dupes"
	"github.com/frathe/picfetch/internal/fileidentity"
)

type browsingKind uint8

const (
	browsingCollection browsingKind = iota
	browsingExplorer
	browsingSearch
	browsingLocation
	browsingCluster
	browsingExplorerMap
	browsingLocationMap
)

type browsingBinding struct {
	kind       browsingKind
	collection uint64
	visit      uint64
}

// browsingScope is one immutable restricted order. Restriction is independent
// of membership: a visit with no surviving members is not the collection.
type browsingScope struct {
	indexes    []int
	restricted bool
	complete   bool
	binding    browsingBinding
	collection dupes.Snapshot
}

// captureBrowsingScope resolves the foreground owner's data into one immutable
// action scope. Feature visibility and nonempty membership never choose a visit.
func (v *viewer) captureBrowsingScope() browsingScope {
	visit := v.browsing.current()
	collection := v.state.Observe()
	scope := browsingScope{complete: true, collection: collection.FileSet(), binding: visit.binding}
	scope.binding.collection = collection.Generation()
	scope.restricted = visit.binding.kind != browsingCollection
	switch visit.binding.kind {
	case browsingCollection:
		// Ordinary and duplicate browsing retain the baseline model adapter.
	case browsingLocation:
		scope.indexes = v.locationIndexes()
		scope.complete = v.locationMap.Counts().Complete
	case browsingCluster:
		scope.indexes = v.locationIndexes()
	case browsingLocationMap:
		scope.complete = v.locationMap.Counts().Complete
	case browsingSearch:
		scope.indexes = v.captureRankedIndexes(visit)
		scope.complete = v.visualsearch.State().Progress.Complete
	case browsingExplorerMap:
		scope.complete = v.explorer.State().Complete
	case browsingExplorer:
		paths, _ := v.explorer.Cohort()
		members := make(map[string]bool, len(paths))
		for _, path := range paths {
			members[path] = true
		}
		var indexes []int
		for i := range collection.Count() {
			if members[collection.FileAt(i).Path()] {
				indexes = append(indexes, i)
			}
		}
		scope.indexes = indexes
	}
	return scope
}

func (s browsingScope) Next(from, delta int) (int, bool) {
	if len(s.indexes) == 0 {
		return 0, false
	}
	return neighborInOrder(s.indexes, from, delta), true
}

func (s browsingScope) First() (int, bool) {
	if len(s.indexes) == 0 {
		return 0, false
	}
	return s.indexes[0], true
}

func (s browsingScope) Last() (int, bool) {
	if len(s.indexes) == 0 {
		return 0, false
	}
	return s.indexes[len(s.indexes)-1], true
}

// RestoreImage prefers the exact bookmark, then another occurrence of that
// source, then the first eligible image. A restriction never widens implicitly.
func (s browsingScope) RestoreImage(origin fileidentity.Occurrence, identities fileidentity.Index) (int, bool) {
	eligible := func(i int) bool {
		return i >= 0 && i < s.collection.Count() && (!s.restricted || slices.Contains(s.indexes, i))
	}
	if i := identities.Resolve(origin); eligible(i) {
		return i, true
	}
	for ordinal := 0; ; ordinal++ {
		i := identities.Resolve(fileidentity.Occurrence{Path: origin.Path, Ordinal: ordinal})
		if i < 0 {
			break
		}
		if eligible(i) {
			return i, true
		}
	}
	if s.restricted {
		return s.First()
	}
	return 0, s.collection.Count() > 0
}

// Recover follows source-removal policy, not ordinary Next: the removed
// collection position is reused when eligible, then the scoped order wraps.
// An explicitly restored image origin takes priority over that successor.
func (s browsingScope) Recover(failed, restored int) (int, bool) {
	if restored >= 0 {
		return restored, restored < s.collection.Count() && (!s.restricted || slices.Contains(s.indexes, restored))
	}
	if s.restricted {
		for _, index := range s.indexes {
			if index >= failed {
				return index, true
			}
		}
		return s.First()
	}
	if count := s.collection.Count(); count > 0 {
		return ((failed % count) + count) % count, true
	}
	return 0, false
}

// sourceOccurrences captures only the bookmarked path, without retaining an
// index for unrelated collection members.
func (v *viewer) sourceOccurrences(path string) fileidentity.Index {
	collection := v.state.Observe()
	return fileidentity.NewIndex(collection.Count(), func(i int) string {
		if uri := collection.FileAt(i); uri != nil && uri.Path() == path {
			return path
		}
		return ""
	})
}

func (v *viewer) currentImageOccurrence() fileidentity.Occurrence {
	collection := v.state.Observe()
	if _, position, ok := collection.Current(); ok {
		bookmark, _ := collection.Bookmark(position)
		return bookmark.occurrence
	}
	return fileidentity.Occurrence{}
}

// browsingContext observes the source subset for payload/order capture.
// It deliberately does not copy ranked paths just to answer a capability check.
type browsingContext struct {
	ranked, grid bool
}

func (v *viewer) browsingContext() browsingContext {
	ranked := v.searchActive()
	return browsingContext{ranked: ranked, grid: v.grid.Visible()}
}

// An opened search image uses frozen rank; the current Grid supplies live rank.
func (v *viewer) captureRankedIndexes(visit browsingVisit) []int {
	if visit.surface == browsingGrid && v.grid.Visible() {
		return v.grid.ResultIndexes()
	}
	paths := visit.order
	if visit.surface != browsingImage {
		paths = v.visualsearch.State().Visit.Paths
	}
	if len(paths) == 0 {
		return nil
	}
	byPath := make(map[string]int, len(paths))
	for _, path := range paths {
		byPath[path] = -1
	}
	remaining := len(byPath)
	collection := v.state.Observe()
	for i := range collection.Count() {
		uri := collection.FileAt(i)
		if uri == nil {
			continue
		}
		path := uri.Path()
		if index, wanted := byPath[path]; wanted && index < 0 {
			byPath[path] = i
			remaining--
			if remaining == 0 {
				break
			}
		}
	}
	indexes := make([]int, 0, len(paths))
	for _, path := range paths {
		if i := byPath[path]; i >= 0 {
			indexes = append(indexes, i)
		}
	}
	return indexes
}

func neighborInOrder(indexes []int, from, delta int) int {
	if len(indexes) == 0 {
		return from
	}
	pos := slices.Index(indexes, from)
	if pos < 0 {
		pos, delta = 0, 0
	}
	return indexes[((pos+delta)%len(indexes)+len(indexes))%len(indexes)]
}
