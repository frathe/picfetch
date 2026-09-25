package ui

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/ui/grid"
	"github.com/frathe/picfetch/internal/ui/locationmap"
)

type locationInput struct {
	order                                  []fileidentity.Occurrence
	prepare                                func()
	prepareOp                              requestLifecycle
	savedGrid, clusterGrid                 *grid.Visit
	image, cluster, transition, rebuilding bool
}

func (v *viewer) showLocationMap() {
	if v.stopping || v.comparisonActive() || v.scanOp.active || v.sortOp.active || !v.yieldCopySelection() {
		return
	}
	if v.locationMap.Active() {
		v.returnLocationMap()
		return
	}
	v.closeExplorer()
	v.closeVisualSearch()
	visit := v.grid.CaptureVisit()
	if visit.Ranked {
		v.grid.Close()
	} else {
		visit.Visible = false
		v.grid.RestoreVisit(visit)
	}
	if v.slides.Active() {
		v.slides.Exit()
		v.resetFade()
	}
	v.locationInput.rebuilding = false
	v.beginLocationTrial()
	v.locationMap.Preparing()
	v.locationMap.ValidateSources(func(changed bool) {
		if changed {
			v.grid.InvalidateContent()
		}
		v.prepareLocationMap()
	})
}

func (v *viewer) prepareLocationMap() {
	v.locationInput.prepareOp.invalidate()
	v.locationInput.prepare = nil
	if v.dupes.HideDuplicates() {
		token := v.locationInput.prepareOp.begin()
		v.locationInput.prepare = func() {
			if token.current() && !v.stopping {
				v.beginLocationMap()
			}
		}
		if !v.grid.PrepareDuplicateGroups() {
			v.syncDuplicatePreparationProgress()
			v.Unfocus()
			return
		}
		v.locationInput.prepare = nil
	}
	v.beginLocationMap()
}

func (v *viewer) beginLocationMap() {
	v.scanLocationTrial()
	v.locationMap.SetFavoritesRoot(v.favorites.Dir())
	identities := fileidentity.NewIndex(v.FileCount(), func(i int) string { return v.FileAt(i).Path() })
	sources := make([]locationmap.Source, 0, v.FileCount())
	visibility := v.dupes.Visibility()
	groups := make(map[int][]locationmap.Source)
	if visibility.Hide {
		for i, uri := range v.state.files {
			if !visibility.HiddenExtra(i) {
				continue
			}
			identity, _ := identities.Capture(uri.Path(), i)
			size, _ := v.dupes.NativeSize(uri.String())
			rep := visibility.RepresentativeOf(i)
			groups[rep] = append(groups[rep], locationmap.Source{URI: uri, Identity: identity, Pixels: int64(size.X) * int64(size.Y)})
		}
	}
	for i, uri := range v.state.files {
		if !visibility.Visible(i) {
			continue
		}
		identity, _ := identities.Capture(uri.Path(), i)
		sources = append(sources, locationmap.Source{URI: uri, Identity: identity, Donors: groups[i]})
	}
	if v.locationInput.rebuilding {
		v.locationMap.Rebuild(sources)
	} else {
		v.locationMap.Open(sources)
	}
	v.syncLocationDisplayed()
	v.Unfocus()
}

func (v *viewer) LeaveLocationMap() {
	v.closeLocationMap()
	v.Unfocus()
}
func (v *viewer) closeLocationMap() {
	v.locationInput.prepareOp.invalidate()
	if v.locationInput.prepare != nil {
		v.locationInput.prepare = nil
		v.grid.Close()
	}
	v.locationInput.image, v.locationInput.order = false, nil
	v.locationInput.cluster = false
	v.locationMap.Close()
	if saved := v.locationInput.savedGrid; saved != nil {
		v.locationInput.savedGrid = nil
		v.grid.RestoreVisit(*saved)
	}
	v.locationInput.clusterGrid = nil
}
func (v *viewer) LocationMapChanged() {
	v.recordLocationTrial()
	if v.locationMap != nil && v.locationMap.Active() && v.locationInput.image && !v.locationInput.cluster && v.locationMap.Counts().Complete && len(v.locationMap.Points()) == 0 {
		v.returnLocationMap()
	}
	v.syncMenus()
}
func (v *viewer) locationMapVisible() bool { return v.locationMap != nil && v.locationMap.Visible() }

func (v *viewer) locationMapKey(key fyne.KeyName) bool {
	if v.locationInput.image && key == fyne.KeyP {
		return true
	}
	if v.locationInput.cluster && v.locationInput.image && (key == fyne.KeyEscape || key == fyne.KeyG) {
		v.openLocationGrid()
		return true
	}
	if v.locationInput.image && key == fyne.KeyEscape {
		v.returnLocationMap()
		return true
	}
	if v.locationInput.image && key == fyne.KeyG {
		v.LeaveLocationMap()
		v.grid.Toggle()
		return true
	}
	if !v.locationMapVisible() {
		return false
	}
	switch key {
	case fyne.KeyEscape, fyne.KeyV:
		v.LeaveLocationMap()
	case fyne.KeyF1:
		v.help.ShowManual()
	default:
		v.locationMap.HandleKey(key, v.keyModifiers())
	}
	return true
}

func (v *viewer) OpenLocationCluster(members []fileidentity.Occurrence) {
	visit := v.grid.CaptureVisit()
	visit.Visible = false
	v.locationInput.savedGrid = &visit
	v.locationInput.cluster = true
	v.locationInput.order = slices.Clone(members)
	v.locationInput.clusterGrid = nil
	v.locationMap.HideForImage()
	v.openLocationGrid()
}

func (v *viewer) openLocationGrid() {
	v.locationInput.transition = true
	defer func() { v.locationInput.transition = false; v.syncMenus() }()
	v.locationInput.image = false
	v.grid.OpenOccurrences(v.locationInput.order, lang.L("Showing location cluster"), v.returnLocationMap)
	if visit := v.locationInput.clusterGrid; visit != nil {
		v.grid.RestoreVisit(*visit)
	}
	v.Unfocus()
}

func (v *viewer) locationImageOpened(visit grid.Visit) {
	if !v.locationInput.cluster {
		return
	}
	v.locationInput.clusterGrid = &visit
	v.locationInput.image = true
}

func (v *viewer) returnLocationMap() {
	v.locationMap.ValidateSources(func(changed bool) {
		// Keep the browsing visit visible until validation completes. It can
		// still start Copy Selection while the worker is checking sources.
		ready := v.yieldCopySelection()
		if ready {
			v.locationInput.cluster = false
			v.locationInput.image = false
			v.locationInput.order = nil
			if saved := v.locationInput.savedGrid; saved != nil {
				v.locationInput.savedGrid = nil
				v.grid.RestoreVisit(*saved)
			}
			v.locationInput.clusterGrid = nil
		}
		if changed {
			// Reconcile even when an in-flight copy defers the visible return:
			// validation has already recorded these new source versions.
			v.grid.InvalidateContent()
			v.rebuildLocationMap()
		}
		if ready {
			v.locationMap.Return()
		}
	})
}

func (v *viewer) rebuildLocationMap() {
	v.locationMap.SetSources(v.state.files)
	if !v.locationMap.Active() || v.stopping {
		return
	}
	v.locationInput.rebuilding = true
	v.locationMap.PreparingRebuild()
	v.prepareLocationMap()
}

// captureLocationReconciliation runs before indexes change. It preserves exact
// occurrence bookmarks and the Grid's ordinary selection/search escape stages.
func (v *viewer) captureLocationReconciliation(removed []int) func() {
	if !v.locationMap.Active() {
		return func() { v.locationMap.SetSources(v.state.files) }
	}
	v.locationInput.prepareOp.invalidate()
	v.locationInput.prepare = nil
	var current *grid.Visit
	if v.locationInput.cluster && v.grid.Visible() {
		visit := v.grid.CaptureVisit()
		current = &visit
	}
	old := fileidentity.NewIndex(v.FileCount(), func(i int) string { return v.FileAt(i).Path() })
	ordinals := map[string]int{}
	survivors := map[fileidentity.Occurrence]fileidentity.Occurrence{}
	for i, source := range v.state.files {
		if _, deleted := slices.BinarySearch(removed, i); deleted {
			continue
		}
		before, _ := old.Capture(source.Path(), i)
		after := fileidentity.Occurrence{Path: source.Path(), Ordinal: ordinals[source.Path()]}
		ordinals[source.Path()]++
		survivors[before] = after
	}
	var order []fileidentity.Occurrence
	for _, identity := range v.locationInput.order {
		if next, ok := survivors[identity]; ok {
			order = append(order, next)
		}
	}
	v.locationInput.order = order
	for _, visit := range []*grid.Visit{current, v.locationInput.savedGrid, v.locationInput.clusterGrid} {
		if visit != nil {
			*visit = visit.RemapOccurrences(survivors)
		}
	}
	v.locationInput.transition = true
	return func() {
		if current != nil {
			v.grid.RestoreVisit(*current)
		}
		v.locationInput.transition = false
		// Rebuild retires validation, so admit an exhausted visit's return
		// only after the new source generation owns the map.
		v.rebuildLocationMap()
		if v.locationInput.cluster && len(v.locationInput.order) == 0 {
			v.returnLocationMap()
		}
	}
}

func (v *viewer) locationGridChanged() {
	if v.locationInput.cluster && !v.locationInput.transition && !v.locationInput.image && !v.grid.Visible() {
		v.returnLocationMap()
	}
}

func (v *viewer) OpenLocationImage(identity fileidentity.Occurrence) {
	indexes := fileidentity.NewIndex(v.FileCount(), func(i int) string { return v.FileAt(i).Path() })
	i := indexes.Resolve(identity)
	if i < 0 {
		return
	}
	v.locationInput.image, v.locationInput.order = true, nil
	for _, point := range v.locationMap.Points() {
		v.locationInput.order = append(v.locationInput.order, point.Source.Identity)
	}
	v.locationMap.HideForImage()
	v.Unfocus()
	v.ShowImage(i)
}

func (v *viewer) locationIndexes() []int {
	if !v.locationMap.Active() || !v.locationInput.image {
		return nil
	}
	index := fileidentity.NewIndex(v.FileCount(), func(i int) string { return v.FileAt(i).Path() })
	var result []int
	order := v.locationInput.order
	if !v.locationInput.cluster {
		if points := v.locationMap.Points(); len(points) > 0 || v.locationMap.Counts().Complete {
			order = nil
			for _, point := range points {
				order = append(order, point.Source.Identity)
			}
		}
	}
	for _, identity := range order {
		if i := index.Resolve(identity); i >= 0 {
			result = append(result, i)
		}
	}
	if !v.locationInput.cluster {
		slices.Sort(result)
	}
	return result
}

func (v *viewer) syncLocationDisplayed() {
	if v.locationMap == nil {
		return
	}
	uri, ok := v.displayedFile()
	identity := fileidentity.Occurrence{}
	if ok {
		index := fileidentity.NewIndex(v.FileCount(), func(i int) string { return v.FileAt(i).Path() })
		position := v.state.index
		if position < 0 || position >= v.FileCount() || v.FileAt(position).Path() != uri.Path() {
			position = index.Resolve(fileidentity.Occurrence{Path: uri.Path()})
		}
		if position >= 0 && v.locationMap.Active() && v.dupes.HideDuplicates() {
			position = v.dupes.Visibility().RepresentativeOf(position)
		}
		if position >= 0 {
			identity, _ = index.Capture(v.FileAt(position).Path(), position)
		}
	}
	v.locationMap.SetDisplayed(identity)
}
