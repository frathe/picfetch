package ui

import (
	"context"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/requestlife"
	"github.com/frathe/picfetch/internal/ui/grid"
	"github.com/frathe/picfetch/internal/ui/locationmap"
)

type locationInput struct {
	prepare    func()
	prepareOp  requestlife.Owner
	rebuilding bool
	maximized  bool
}

func (v *viewer) showLocationMap() {
	if _, ok := v.admitCommand(commandRequest{command: commandLocationMap}); !ok {
		return
	}
	if v.locationVisitActive() {
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
	v.browsing.enterLocation(v.Generation())
	binding := v.browsing.current().binding
	v.beginLocationTrial()
	v.maximizeLocationMap()
	v.locationMap.Preparing()
	v.locationMap.ValidateSources(func(changed bool) {
		if !v.browsing.matches(binding, v.Generation()) {
			return
		}
		if _, ok := v.admitCommand(commandRequest{command: commandLocationMap, route: routeDelivery}); !ok {
			v.closeLocationMap()
			return
		}
		if changed {
			v.grid.InvalidateContent()
		}
		v.prepareLocationMap()
	})
	v.ForceRepaint()
}

func (v *viewer) maximizeLocationMap() {
	// Native qualification compares fixed-size captured frames across visits.
	if v.locationTrial != nil {
		return
	}
	v.maximizeWindow(v.win)
	v.locationInput.maximized = true
}

func (v *viewer) prepareLocationMap() {
	v.locationInput.prepareOp.Invalidate()
	v.locationInput.prepare = nil
	if v.dupes.HideDuplicates() {
		token := v.locationInput.prepareOp.Begin(context.Background())
		v.locationInput.prepare = func() {
			if token.Current() && !v.stopping {
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
	// Committed-source rebuilds retain their reconciliation ownership.
	if !v.locationInput.rebuilding {
		if _, ok := v.admitCommand(commandRequest{command: commandLocationMap, route: routeDelivery}); !ok {
			v.closeLocationMap()
			return
		}
	}
	v.scanLocationTrial()
	v.locationMap.SetFavoritesRoot(v.favorites.Dir())
	collection := v.state.Observe()
	identities := collection.Occurrences()
	sources := make([]locationmap.Source, 0, collection.Count())
	visibility := v.dupes.Visibility()
	groups := make(map[int][]locationmap.Source)
	if visibility.Hide {
		for i := range collection.Count() {
			uri := collection.FileAt(i)
			if !visibility.HiddenExtra(i) {
				continue
			}
			identity, _ := identities.Capture(uri.Path(), i)
			size, _ := v.dupes.NativeSize(uri.String())
			rep := visibility.RepresentativeOf(i)
			groups[rep] = append(groups[rep], locationmap.Source{URI: uri, Identity: identity, Pixels: int64(size.X) * int64(size.Y)})
		}
	}
	for i := range collection.Count() {
		uri := collection.FileAt(i)
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
	visit := v.browsing.current()
	v.browsing.leaveLocation()
	v.locationInput.prepareOp.Invalidate()
	if v.locationInput.prepare != nil {
		v.locationInput.prepare = nil
		v.grid.Close()
	}
	v.locationMap.Close()
	if visit.binding.kind == browsingCluster && visit.origin != nil {
		v.grid.RestoreVisit(visit.origin.grid)
	}
}
func (v *viewer) LocationMapChanged() {
	v.recordLocationTrial()
	if v.browsing.current().binding.kind == browsingLocation && v.locationMap.Counts().Complete && len(v.locationMap.Points()) == 0 {
		v.returnLocationMap()
	}
	v.syncMenus()
}
func (v *viewer) locationVisitActive() bool { return v.browsing.has(browsingLocationMap) }
func (v *viewer) locationImageVisit() bool {
	visit := v.browsing.current()
	return visit.binding.kind == browsingLocation || visit.binding.kind == browsingCluster && visit.surface == browsingImage
}
func (v *viewer) locationMapVisible() bool {
	return v.browsing.current().binding.kind == browsingLocationMap
}

func (v *viewer) locationMapKey(key fyne.KeyName) bool {
	if v.locationImageVisit() && key == fyne.KeyP {
		return true
	}
	if v.browsing.current().binding.kind == browsingCluster && v.locationImageVisit() && (key == fyne.KeyEscape || key == fyne.KeyG) {
		v.openLocationGrid()
		return true
	}
	if v.locationImageVisit() && key == fyne.KeyEscape {
		v.returnLocationMap()
		return true
	}
	if v.locationImageVisit() && key == fyne.KeyG {
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
	if _, ok := v.admitCommand(commandRequest{command: commandLocationMap, route: routeDelivery}); !ok {
		return
	}
	visit := v.grid.CaptureVisit()
	visit.Visible = false
	if !v.browsing.openLocationCluster(v.browsing.current().binding, v.Generation(), members, visit) {
		return
	}
	v.locationMap.HideForImage()
	v.presentLocationGrid(nil)
}

func (v *viewer) openLocationGrid() {
	plan, ok := v.browsing.planReturn(v.browsing.current().binding, v.Generation(), browsingReturnGrid)
	if !ok || plan.source.kind != browsingCluster {
		return
	}
	if _, ok := v.admitCommand(commandRequest{command: commandGrid, route: routeDelivery}); !ok || !v.browsing.commitReturn(plan, v.Generation()) {
		return
	}
	v.presentLocationGrid(plan.grid)
}

func (v *viewer) presentLocationGrid(bookmark *grid.Visit) {
	visit := v.browsing.current()
	v.grid.OpenOccurrences(visit.occurrences, lang.L("Showing location cluster"), func() { v.returnLocationMapFrom(visit.binding) })
	if bookmark != nil {
		v.grid.RestoreInteraction(*bookmark)
	}
	v.Unfocus()
	v.syncMenus()
}

func (v *viewer) returnLocationMap() {
	v.returnLocationMapFrom(v.browsing.current().binding)
}

func (v *viewer) returnLocationMapFrom(binding browsingBinding) {
	destination := browsingReturnMap
	if binding.kind == browsingLocation || binding.kind == browsingCluster {
		destination = browsingReturnParent
	}
	plan, ok := v.browsing.planReturn(binding, v.Generation(), destination)
	if !ok {
		return
	}
	v.locationMap.ValidateSources(func(changed bool) {
		if !v.browsing.matches(plan.source, v.Generation()) || plan.revision != v.browsing.revision {
			return
		}
		// Keep the browsing visit visible until validation completes. It can
		// still start Copy Selection while the worker is checking sources.
		_, ready := v.admitCommand(commandRequest{command: commandLocationMap, route: routeDelivery})
		if ready {
			ready = v.browsing.commitReturn(plan, v.Generation())
		}
		if ready && plan.source.kind == browsingCluster && plan.origin != nil {
			v.grid.RestoreVisit(plan.origin.grid)
		}
		if changed {
			// Reconcile even when an in-flight copy defers the visible return:
			// validation has already recorded these new source versions.
			v.grid.InvalidateContent()
			v.rebuildLocationMap()
		}
		if ready {
			v.maximizeLocationMap()
			v.locationMap.Return()
		}
	})
}

func (v *viewer) rebuildLocationMap() {
	v.locationMap.SetSources(v.state.Observe().DisplayFiles())
	if !v.locationVisitActive() || v.stopping {
		return
	}
	v.locationInput.rebuilding = true
	v.locationMap.PreparingRebuild()
	v.prepareLocationMap()
}

func (v *viewer) OpenLocationImage(identity fileidentity.Occurrence) {
	indexes := v.state.Observe().Occurrences()
	i := indexes.Resolve(identity)
	if i < 0 {
		return
	}
	if _, ok := v.admitCommand(commandRequest{command: commandLocationMap, route: routeDelivery}); !ok {
		return
	}
	var discovered []fileidentity.Occurrence
	for _, point := range v.locationMap.Points() {
		discovered = append(discovered, point.Source.Identity)
	}
	if !v.browsing.openLocationImage(v.browsing.current().binding, v.Generation(), identity, discovered) {
		return
	}
	v.locationMap.HideForImage()
	v.Unfocus()
	v.ShowImage(i)
}

func (v *viewer) locationIndexes() []int {
	if !v.locationImageVisit() && v.browsing.current().binding.kind != browsingCluster {
		return nil
	}
	index := v.state.Observe().Occurrences()
	var result []int
	order := v.browsing.current().occurrences
	if v.browsing.current().binding.kind != browsingCluster {
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
	if v.browsing.current().binding.kind != browsingCluster {
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
		collection := v.state.Observe()
		index := collection.Occurrences()
		_, position, _ := collection.Current()
		if position < 0 || position >= collection.Count() || collection.FileAt(position).Path() != uri.Path() {
			position = index.Resolve(fileidentity.Occurrence{Path: uri.Path()})
		}
		if position >= 0 && v.locationVisitActive() && v.dupes.HideDuplicates() {
			position = v.dupes.Visibility().RepresentativeOf(position)
		}
		if position >= 0 {
			identity, _ = index.Capture(collection.FileAt(position).Path(), position)
		}
	}
	v.locationMap.SetDisplayed(identity)
}
