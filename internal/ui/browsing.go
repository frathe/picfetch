package ui

import (
	"slices"

	"github.com/frathe/picfetch/internal/dupes"
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

// captureBrowsingScope is the temporary adapter for visits whose ownership has
// not migrated yet. Consumers never infer restriction from a nonempty slice.
func (v *viewer) captureBrowsingScope() browsingScope {
	scope := browsingScope{complete: true, collection: v.state.snapshot()}
	scope.binding.collection = v.Generation()
	if v.locationImageVisit() || v.browsing.current().binding.kind == browsingCluster {
		scope.indexes, scope.restricted = v.locationIndexes(), true
		scope.binding = v.browsing.current().binding
		scope.complete = v.locationMap.Counts().Complete
		if scope.binding.kind == browsingCluster {
			scope.complete = true
		}
		return scope
	}
	if v.browsing.current().binding.kind == browsingLocationMap {
		scope.restricted, scope.binding = true, v.browsing.current().binding
		scope.complete = v.locationMap.Counts().Complete
		return scope
	}
	if order := v.captureSearchOrder(); order.active {
		state := v.visualsearch.State()
		scope.indexes, scope.restricted = order.indexes, true
		scope.binding = v.browsing.current().binding
		scope.complete = state.Progress.Complete
		return scope
	}
	if v.browsing.current().binding.kind == browsingExplorerMap {
		scope.restricted, scope.binding = true, v.browsing.current().binding
		scope.complete = v.explorer.State().Complete
		return scope
	}
	if v.browsing.has(browsingExplorer) {
		paths, _ := v.explorer.Cohort()
		members := make(map[string]bool, len(paths))
		for _, path := range paths {
			members[path] = true
		}
		var indexes []int
		for i := range v.FileCount() {
			if members[v.FileAt(i).Path()] {
				indexes = append(indexes, i)
			}
		}
		scope.indexes, scope.restricted = indexes, true
		scope.binding = v.browsing.current().binding
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

// browsingContext observes the source subset for payload/order capture.
// It deliberately does not copy ranked paths just to answer a capability check.
type browsingContext struct {
	ranked, grid bool
}

func (v *viewer) browsingContext() browsingContext {
	ranked := v.searchActive()
	return browsingContext{ranked: ranked, grid: v.grid.Visible()}
}

// searchOrder is one immutable index snapshot for an action or both preloads.
// An opened image uses its frozen order; the Grid uses its current visible rank.
type searchOrder struct {
	indexes []int
	active  bool
}

func (v *viewer) captureSearchOrder() searchOrder {
	mode := v.browsingContext()
	order := searchOrder{active: mode.ranked}
	if !order.active {
		return order
	}
	if mode.grid {
		order.indexes = v.grid.ResultIndexes()
		return order
	}
	visit := v.browsing.current()
	paths := visit.order
	if visit.surface != browsingImage {
		paths = v.visualsearch.State().Visit.Paths
	}
	if len(paths) == 0 {
		return order
	}
	byPath := make(map[string]int, len(paths))
	for _, path := range paths {
		byPath[path] = -1
	}
	remaining := len(byPath)
	for i, uri := range v.state.files {
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
	order.indexes = indexes
	return order
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
