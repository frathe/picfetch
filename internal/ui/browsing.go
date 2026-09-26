package ui

import "slices"

// browsingContext captures cross-feature restrictions on the UI goroutine.
// Menus and command handlers consume the same restriction for source subsets.
// It deliberately does not copy ranked paths just to answer a capability check.
type browsingContext struct {
	ranked, grid, restricted bool
}

func (v *viewer) browsingContext() browsingContext {
	ranked := v.searchActive()
	return browsingContext{ranked: ranked, grid: v.grid.Visible(), restricted: ranked || v.explorer.HasCohort() || v.locationMap.Active()}
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
	paths := v.searchView.imageOrder
	if paths == nil {
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
