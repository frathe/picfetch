package ui

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	fynetest "fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/session"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/grid"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestBrowsingLoadRecovery(t *testing.T) {
	t.Run("frozen_cluster_successor", func(t *testing.T) {
		v := newTestViewer(t)
		a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
		b := uitest.TempJPEGURI(t, "b.jpg", 24, 16, color.Black)
		c := uitest.TempGPSJPEGURI(t, "c.jpg", 24, 16, 52.52, 13.405)
		dropAndWait(t, v, a, b, c)
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		fynetest.Tap(locationButton(t, v, "2 images"))
		v.grid.Settle()
		v.display.WaitPreloads()
		if err := os.WriteFile(a.Path(), []byte("broken image"), 0600); err != nil {
			t.Fatal(err)
		}
		v.imgCache.Purge()
		revision := v.display.RequestRevision()
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		if displayed, ok := v.DisplayedFile(); !ok || displayed.Path() != c.Path() || v.display.RequestRevision() != revision+1 || !v.locationImageVisit() {
			t.Fatal("cluster load failure escaped its frozen members or retry chain")
		}
	})
	for _, imageOrigin := range []bool{false, true} {
		t.Run("explorer_search_origin/"+fmt.Sprint(imageOrigin), func(t *testing.T) {
			v := explorerFixture(t)
			first, second := v.FileAt(14), v.FileAt(15)
			v.OpenSimilarityCohort([]string{first.Path(), second.Path()})
			if imageOrigin {
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
				waitUntilLoaded(t, v)
			}
			publish := streamingSearchFrom(t, v)
			publish(similarity.SearchFinal, 1, 2)
			failed := v.FileAt(1)
			v.display.WaitPreloads()
			if err := os.WriteFile(failed.Path(), []byte("broken image"), 0600); err != nil {
				t.Fatal(err)
			}
			v.imgCache.Purge()
			revision := v.display.RequestRevision()
			v.grid.SimulateHover(1)
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
			waitUntilLoaded(t, v)
			if v.searchActive() || v.grid.Visible() == imageOrigin || v.display.RequestRevision() != revision+1 {
				t.Fatal("recovery lost the origin surface or started a competing display request")
			}
			if displayed, ok := v.DisplayedFile(); !ok || displayed.Path() != first.Path() {
				t.Fatal("failed ranked result escaped the restored Explorer scope")
			}
		})
	}
	t.Run("repeated_failures_exhaust_cohort", func(t *testing.T) {
		v := explorerFixture(t)
		paths := []string{v.FileAt(14).Path(), v.FileAt(15).Path()}
		v.OpenSimilarityCohort(paths)
		v.display.WaitPreloads()
		for _, path := range paths {
			if err := os.WriteFile(path, []byte("broken image"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		v.imgCache.Purge()
		revision := v.display.RequestRevision()
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		if !v.explorerMapActive() || v.grid.Visible() || v.FileCount() != 16 || v.display.RequestRevision() != revision+1 {
			t.Fatal("repeated failures escaped the cohort or display retry chain")
		}
		if len(v.preloadCandidates()) != 0 {
			t.Fatal("exhausted recovery preloaded unrelated images")
		}
	})
}

func TestBrowsingCollectionChanges(t *testing.T) {
	t.Run("coherent_menu_publication", func(t *testing.T) {
		v, publish := streamingSearch(t)
		publish(similarity.SearchFinal, 2, 1)
		if !v.menus.Actions().Hide().Disabled {
			t.Fatal("ranked visit did not disable duplicate toggling")
		}
		observed, leaked := false, false
		v.grid.SetOnResultChanged(func() {
			observed = true
			v.syncMenus()
			leaked = leaked || !v.menus.Actions().Hide().Disabled
		})
		v.RemoveFile(3)
		if !observed || leaked || v.menus.Actions().Hide().Disabled {
			t.Fatal("reconciliation published intermediate or stale menu availability")
		}
	})
	t.Run("committed_write_under_cohort_comparison", func(t *testing.T) {
		v := explorerFixture(t)
		first, second := v.FileAt(14), v.FileAt(15)
		paths := []string{first.Path(), second.Path()}
		v.OpenSimilarityCohort(paths)
		publish := streamingSearchFrom(t, v)
		publish(similarity.SearchFinal, 0, 1)
		for _, i := range []int{0, 1} {
			v.grid.SimulateHover(i)
			v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
		}
		v.compareSelected()
		if err := v.compare.Settle(context.Background()); err != nil {
			t.Fatal(err)
		}
		result, err := imaging.SaveRotatedContext(context.Background(), first, image.NewRGBA(image.Rect(0, 0, 19, 13)))
		if err != nil || !result.Committed {
			t.Fatalf("fixture did not commit: %v", err)
		}
		v.afterFileWrite(result, false, false, func() {})
		drainFileWork(t, v)
		if err := v.compare.Settle(context.Background()); err != nil {
			t.Fatal(err)
		}
		if v.searchActive() || !v.comparisonActive() || !v.grid.Visible() || !slices.Equal(explorerGridPaths(v), paths) {
			t.Fatal("committed reconciliation lost the covered cohort Grid")
		}
		v.compare.Close()
		fynetest.Tap(explorerButton(t, v, "Back to map"))
		if !v.explorerMapActive() {
			t.Fatal("covered reconciliation retained a stale return binding")
		}
	})
	t.Run("surviving_middle_occurrence", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		duplicate := v.FileAt(1)
		v.SetMergeMode(true)
		dropAndWait(t, v, duplicate)
		dropAndWait(t, v, duplicate)
		v.grid.Close()
		v.ShowImage(2)
		waitUntilLoaded(t, v)
		publish := streamingSearchFrom(t, v)
		publish(similarity.SearchFinal, 0, 4)
		v.RemoveFile(1)
		waitUntilLoaded(t, v)
		if v.searchActive() || v.state.index != 1 || v.currentImageOccurrence().Ordinal != 0 {
			t.Fatal("origin ordinal was not remapped to its exact surviving occurrence")
		}
	})
	t.Run("exhausted_search_origin", func(t *testing.T) {
		for _, imageOrigin := range []bool{false, true} {
			t.Run(fmt.Sprint(imageOrigin), func(t *testing.T) {
				v := explorerFixture(t)
				v.OpenSimilarityCohort([]string{v.FileAt(14).Path()})
				if imageOrigin {
					v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
					waitUntilLoaded(t, v)
				}
				publish := streamingSearchFrom(t, v)
				publish(similarity.SearchFinal, 0, 1)
				revision := v.display.RequestRevision()
				v.RemoveFile(14)
				if v.searchActive() || v.grid.Visible() || !v.explorerMapActive() || v.display.RequestRevision() != revision {
					t.Fatal("exhausted origin revived a subset or loaded an unrelated image")
				}
			})
		}
	})
	t.Run("sorted_repeated_image_origin", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		duplicate := v.FileAt(1)
		v.SetMergeMode(true)
		dropAndWait(t, v, duplicate)
		v.grid.Close()
		v.ShowImage(2)
		waitUntilLoaded(t, v)
		publish := streamingSearchFrom(t, v)
		publish(similarity.SearchFinal, 0, 3)
		v.SetSortMode(filesort.ByDropOrder)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		if v.searchActive() || v.state.index != 3 || v.FileAt(v.state.index).Path() != duplicate.Path() {
			t.Fatal("sort replaced the repeated image origin with its first occurrence")
		}
	})
	t.Run("removed_selected_search_origin", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		duplicate := v.FileAt(1)
		v.SetMergeMode(true)
		dropAndWait(t, v, duplicate)
		v.grid.Toggle()
		v.grid.SimulateHover(1)
		v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
		publish := streamingSearchFrom(t, v)
		publish(similarity.SearchFinal, 0, 3)
		v.RemoveFile(1)
		if v.searchActive() || !v.grid.Visible() || v.grid.SelectionCount() != 0 {
			t.Fatal("removed origin selection substituted the surviving duplicate")
		}
	})
	t.Run("restricted_search_origin_fallback", func(t *testing.T) {
		v := explorerFixture(t)
		first, second := v.FileAt(14), v.FileAt(15)
		v.OpenSimilarityCohort([]string{first.Path(), second.Path()})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		publish := streamingSearchFrom(t, v)
		publish(similarity.SearchFinal, 0, 1)
		v.RemoveFile(14)
		waitUntilLoaded(t, v)
		if v.searchActive() || !v.browsing.has(browsingExplorer) || v.FileAt(v.state.index).Path() != second.Path() {
			t.Fatal("missing Explorer origin fell back outside its restored cohort")
		}
	})
}

func TestBrowsingVisitTransitions(t *testing.T) {
	t.Run("frozen_cluster_and_retired_return", func(t *testing.T) {
		var visits browsingVisits
		visits.enterLocation(7)
		parent := visits.current().binding
		members := []fileidentity.Occurrence{{Path: "/a", Ordinal: 1}, {Path: "/b"}}
		if !visits.openLocationCluster(parent, 7, members, grid.Visit{Query: "ordinary"}) {
			t.Fatal("current map refused its cluster")
		}
		binding := visits.current().binding
		members[0].Ordinal = 0
		if visits.current().occurrences[0].Ordinal != 1 {
			t.Fatal("cluster membership aliases live map discovery")
		}
		visits.openImage(binding, 7, grid.Visit{Query: "cluster", ScrollOffset: 200})
		toGrid, ok := visits.planReturn(binding, 7, browsingReturnGrid)
		if !ok || toGrid.grid.Query != "cluster" || toGrid.grid.ScrollOffset != 200 || !visits.commitReturn(toGrid, 7) {
			t.Fatal("cluster image lost its Grid bookmark")
		}
		toMap, ok := visits.planReturn(binding, 7, browsingReturnParent)
		if !ok || toMap.origin.grid.Query != "ordinary" || !visits.commitReturn(toMap, 7) || visits.current().binding != parent {
			t.Fatal("cluster map return lost its ordinary Grid origin")
		}
		visits.openLocationCluster(parent, 7, members, grid.Visit{})
		if visits.commitReturn(toMap, 7) || visits.current().binding == binding {
			t.Fatal("old cluster return replaced a newly opened visit")
		}
	})
	t.Run("direct_map_parent_and_validation", func(t *testing.T) {
		var visits browsingVisits
		visits.enterLocation(7)
		parent := visits.current().binding
		member := fileidentity.Occurrence{Path: "/a", Ordinal: 1}
		members := []fileidentity.Occurrence{member}
		if visits.openLocationImage(parent, 7, fileidentity.Occurrence{Path: "/unmapped"}, members) {
			t.Fatal("map admitted an image outside its discovered membership")
		}
		if !visits.openLocationImage(parent, 7, member, members) || visits.current().surface != browsingImage || !visits.has(browsingLocationMap) {
			t.Fatal("direct image entry lost its retained map")
		}
		members[0].Ordinal = 0
		if visits.current().occurrences[0] != member {
			t.Fatal("direct discovery fallback aliases the feature's occurrences")
		}
		binding := visits.current().binding
		if _, ok := visits.planReturn(binding, 7, browsingReturnGrid); ok {
			t.Fatal("direct image acquired a cluster Grid destination")
		}
		plan, ok := visits.planReturn(binding, 7, browsingReturnParent)
		if !ok || !visits.commitReturn(plan, 7) || visits.current().binding != parent {
			t.Fatal("validated direct return did not retain its map parent")
		}
		visits.openLocationImage(parent, 7, member, []fileidentity.Occurrence{member})
		if visits.commitReturn(plan, 7) {
			t.Fatal("old validation replaced a newly opened image visit")
		}
		visits.leaveLocation()
		visits.enterLocation(7)
		if visits.current().binding == parent {
			t.Fatal("map reopen reused a retired binding")
		}
	})
	t.Run("ranked_origin_is_independent", func(t *testing.T) {
		var visits browsingVisits
		cohort := visits.openExplorerCohort(7)
		visits.openImage(cohort, 7, grid.Visit{Query: "cohort"})
		origin := browsingOrigin{grid: grid.Visit{Selected: []string{"/a"}}, image: fileidentity.Occurrence{Path: "/a", Ordinal: 1}}
		search, entered := visits.enterSearch(7, origin)
		if !entered || !visits.has(browsingExplorer) || visits.current().binding.kind != browsingSearch {
			t.Fatal("search entry lost the retained Explorer image visit")
		}
		origin.grid.Selected[0] = "/changed"
		bookmark := grid.Visit{Results: []string{"/a", "/c", "/b"}}
		visits.openImage(search, 7, bookmark)
		bookmark.Results[0] = "/changed"
		if !slices.Equal(visits.current().order, []string{"/a", "/c", "/b"}) {
			t.Fatal("search image order aliases the Grid's bookmark")
		}
		toGrid, ok := visits.planReturn(search, 7, browsingReturnGrid)
		if !ok || !visits.commitReturn(toGrid, 7) || visits.current().order != nil || !visits.has(browsingExplorer) {
			t.Fatal("ranked Grid return lost the independent origin or retained frozen order")
		}
		exit, ok := visits.planReturn(search, 7, browsingReturnParent)
		if !ok || exit.origin.grid.Selected[0] != "/a" || exit.origin.image != origin.image {
			t.Fatal("ranked transitions changed the original occurrence/bookmark")
		}
		if !visits.commitReturn(exit, 7) || visits.current().binding != cohort || visits.current().surface != browsingImage {
			t.Fatal("search exit failed to restore the retained Explorer image visit")
		}
		if visits.commitReturn(exit, 7) {
			t.Fatal("search origin could be restored twice")
		}
	})
	var visits browsingVisits
	visits.enterExplorer(7)
	parent := visits.current()
	binding := visits.openExplorerCohort(7)
	bookmark := grid.Visit{Query: "sunset", Highlight: "/b.jpg", ScrollOffset: 240}
	if !visits.openImage(binding, 7, bookmark) {
		t.Fatal("current cohort image was refused")
	}
	if visits.current().surface != browsingImage || !visits.has(browsingExplorerMap) {
		t.Fatal("image opening lost its retained Explorer parent")
	}
	plan, ok := visits.planReturn(binding, 7, browsingReturnGrid)
	if !ok || plan.grid.Query != bookmark.Query || plan.grid.ScrollOffset != bookmark.ScrollOffset {
		t.Fatal("current return lost its Grid interaction")
	}
	if visits.commitReturn(plan, 8) {
		t.Fatal("old collection return was accepted")
	}
	if !visits.commitReturn(plan, 7) || visits.current().surface != browsingGrid {
		t.Fatal("valid image-to-Grid return refused")
	}
	if visits.commitReturn(plan, 7) {
		t.Fatal("already consumed transition was replayed")
	}
	toMap, ok := visits.planReturn(binding, 7, browsingReturnParent)
	if !ok || !visits.commitReturn(toMap, 7) || visits.current().binding != parent.binding || visits.has(browsingExplorer) {
		t.Fatal("Grid return did not retire the cohort to its retained parent")
	}
	binding = visits.openExplorerCohort(7)
	old, _ := visits.planReturn(binding, 7, browsingReturnParent)
	visits.rebind(8)
	if visits.commitReturn(old, 8) {
		t.Fatal("collection rebinding retained an obsolete return")
	}
	visits.leaveExplorer()
	visits.enterExplorer(8)
	visits.openExplorerCohort(8)
	if visits.commitReturn(old, 8) || visits.current().binding.visit == binding.visit {
		t.Fatal("close/reopen revived an old visit")
	}
}

func TestBrowsingProgressiveScopes(t *testing.T) {
	t.Run("direct_map_live_navigation_and_preloads", func(t *testing.T) {
		v, release, queue := locationProgressFixture(t)
		defer release()
		queue.Drain()
		fynetest.Tap(locationPhoto(t, v, "a.jpg"))
		waitUntilLoaded(t, v)
		before := v.captureBrowsingScope()
		if !before.restricted || before.complete || !slices.Equal(before.indexes, []int{0}) || len(v.preloadCandidates()) != 0 {
			t.Fatalf("incomplete direct visit is not bounded to discovered members: %+v", before)
		}
		release()
		v.locationMap.Settle()
		after := v.captureBrowsingScope()
		if !after.complete || !slices.Equal(after.indexes, []int{0, 1}) || !slices.Equal(before.indexes, []int{0}) {
			t.Fatalf("discovery did not extend only the new action's scope: before=%+v after=%+v", before, after)
		}
		neighbors := v.preloadCandidates()
		if len(neighbors) != 1 || neighbors[0].Path() != v.FileAt(1).Path() {
			t.Fatalf("newly mapped member is not a preload neighbor: %v", neighbors)
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
		waitUntilLoaded(t, v)
		if v.state.index != 1 {
			t.Fatal("new mapped member did not become navigable")
		}
	})
	t.Run("search_image_before_first_result", func(t *testing.T) {
		v, publish := streamingSearch(t)
		binding := v.browsing.current().binding
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		publish(similarity.SearchFinal, 2, 1)
		if scope := v.captureBrowsingScope(); !scope.restricted || !slices.Equal(scope.indexes, []int{0}) || scope.binding != binding {
			t.Fatalf("first publication retargeted the pending image visit: %+v", scope)
		}
		if len(v.preloadCandidates()) != 0 {
			t.Fatal("first publication added neighbors to the frozen image visit")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if !v.grid.Visible() || !v.searchActive() || !slices.Equal(v.grid.ResultIndexes(), []int{0, 2, 1}) {
			t.Fatal("image return did not reveal the latest rank without leaving search")
		}
		v.visualsearch.Back()
		if v.searchActive() || !v.grid.Visible() || len(v.grid.ResultIndexes()) != 4 {
			t.Fatal("query-history Back did not restore the initial ordinary Grid")
		}
	})
}

func TestBrowsingRoundTrips(t *testing.T) {
	t.Run("cluster_removed_selection", func(t *testing.T) {
		v := newTestViewer(t)
		a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
		outside := uitest.TempGPSJPEGURI(t, "outside.jpg", 24, 16, 40.7, -74)
		dropAndWait(t, v, a, outside)
		v.state.SetMergeMode(true)
		dropAndWait(t, v, a)
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		fynetest.Tap(locationButton(t, v, "2 images"))
		v.grid.Settle()
		v.grid.SimulateHover(0)
		v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
		removed := v.grid.Selection()[0]
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		v.reconcileSources(sourceChange{kind: sourcesRemoved, removed: []int{removed}})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.grid.Settle()
		if !v.grid.Visible() || v.grid.SelectionCount() != 0 || !slices.Equal(v.grid.ResultIndexes(), []int{0}) {
			t.Fatal("removed selected occurrence was replaced by its surviving duplicate")
		}
	})
	for _, route := range []string{"escape", "g", "show"} {
		t.Run("cluster_bookmark/"+route, func(t *testing.T) {
			v := newTestViewer(t)
			v.win.Resize(fyne.NewSize(480, 280))
			var sources []fyne.URI
			for i := range 20 {
				sources = append(sources, uitest.TempGPSJPEGURI(t, fmt.Sprintf("photo-%02d.jpg", i), 24, 16, 52.52, 13.405))
			}
			dropAndWait(t, v, sources...)
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			fynetest.Tap(locationButton(t, v, "20 images"))
			v.grid.Settle()
			v.grid.SimulateHover(1)
			v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
			v.grid.HandleRune('/')
			v.grid.HandleRune('.')
			v.grid.SimulateHover(19)
			before := v.grid.CaptureVisit()
			if before.ScrollOffset == 0 {
				t.Fatal("cluster fixture did not scroll")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
			waitUntilLoaded(t, v)
			switch route {
			case "escape":
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			case "g":
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyG})
			case "show":
				v.showWindowGrid()
			}
			after := v.grid.CaptureVisit()
			if !v.grid.Visible() || after.Query != before.Query || after.Searching != before.Searching ||
				!slices.Equal(after.Selected, before.Selected) || after.Highlight != before.Highlight || after.ScrollOffset != before.ScrollOffset {
				t.Fatal("cluster image return lost filter, selection, highlight or scroll")
			}
			fynetest.Tap(explorerButton(t, v, "Back to map"))
			v.locationMap.Settle()
			if !locationSurface(t, v).Visible() || !v.locationMapVisible() || v.grid.Visible() {
				t.Fatal("cluster did not return to its mounted map")
			}
		})
	}
	t.Run("explorer_surviving_selection", func(t *testing.T) {
		v := explorerFixture(t)
		paths := []string{v.FileAt(0).Path(), v.FileAt(1).Path(), v.FileAt(2).Path()}
		v.OpenSimilarityCohort(paths)
		v.grid.SelectAll()
		v.grid.SimulateHover(0)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		v.reconcileSources(sourceChange{kind: sourcesRemoved, removed: []int{1}})
		v.returnExplorerGrid()
		bookmark := v.grid.CaptureVisit()
		if !slices.Equal(bookmark.Selected, []string{paths[0], paths[2]}) {
			t.Fatalf("return did not preserve only surviving selections: %v", bookmark.Selected)
		}
		fynetest.Tap(explorerButton(t, v, "Back to map"))
		if !v.explorerMapActive() {
			t.Fatal("image bookmark revived an obsolete return callback")
		}
	})
	for _, unassigned := range []bool{false, true} {
		name := "cohort"
		if unassigned {
			name = "unassigned"
		}
		t.Run("explorer/"+name, func(t *testing.T) {
			v := explorerFixture(t)
			v.win.Resize(fyne.NewSize(480, 280))
			camera := v.explorer.Surface().View()
			var paths []string
			for i := range 16 {
				paths = append(paths, v.FileAt(i).Path())
			}
			if unassigned {
				v.OpenSimilarityUnassigned(paths)
			} else {
				v.OpenSimilarityCohort(paths)
			}
			v.grid.SimulateHover(1)
			v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
			v.grid.HandleRune('/')
			v.grid.HandleRune('.')
			v.grid.SimulateHover(15)
			before := v.grid.CaptureVisit()
			if before.ScrollOffset == 0 {
				t.Fatal("fixture did not scroll the Explorer Grid")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
			waitUntilLoaded(t, v)
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			after := v.grid.CaptureVisit()
			if !v.grid.Visible() || after.Query != before.Query || after.Searching != before.Searching ||
				!slices.Equal(after.Selected, before.Selected) || after.Highlight != before.Highlight || after.ScrollOffset != before.ScrollOffset {
				t.Fatalf("Explorer Grid bookmark lost: query=%q/%q selection=%v/%v highlight=%s/%s scroll=%v/%v", before.Query, after.Query, before.Selected, after.Selected, before.Highlight, after.Highlight, before.ScrollOffset, after.ScrollOffset)
			}
			fynetest.Tap(explorerButton(t, v, "Back to map"))
			if !v.explorerMapActive() || v.grid.Visible() || v.explorer.Surface().View().Center != camera.Center {
				t.Fatal("cohort return did not retain its Explorer camera")
			}
		})
	}
}

func TestBrowsingEmptyScope(t *testing.T) {
	t.Run("cluster_filter", func(t *testing.T) {
		v := newTestViewer(t)
		a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
		dropAndWait(t, v, a)
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		v.OpenLocationCluster([]fileidentity.Occurrence{v.locationMap.Points()[0].Source.Identity})
		for _, r := range "/absent" {
			v.handleTypedRune(r)
		}
		if !v.grid.Visible() || len(v.grid.ResultIndexes()) != 0 || !slices.Equal(v.captureBrowsingScope().indexes, []int{0}) {
			t.Fatal("empty filter retired or widened the frozen cluster")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if !v.grid.Visible() || len(v.grid.ResultIndexes()) != 1 {
			t.Fatal("filter Escape lost its cluster")
		}
	})
	t.Run("direct_map_last_member", func(t *testing.T) {
		v := newTestViewer(t)
		a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
		b := uitest.TempJPEGURI(t, "b.jpg", 24, 16, color.White)
		c := uitest.TempJPEGURI(t, "c.jpg", 24, 16, color.Black)
		dropAndWait(t, v, a, b, c)
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		fynetest.Tap(locationPhoto(t, v, "a.jpg"))
		waitUntilLoaded(t, v)
		v.reconcileSources(sourceChange{kind: sourcesRemoved, removed: []int{0}})
		v.locationMap.Settle()
		if !v.locationMapVisible() {
			t.Fatal("exhausted direct visit did not return to its map")
		}
		if scope := v.captureBrowsingScope(); !scope.restricted || len(scope.indexes) != 0 || len(v.preloadCandidates()) != 0 {
			t.Fatalf("exhausted map widened to the collection: %+v", scope)
		}
	})
	t.Run("explorer_filter", func(t *testing.T) {
		v := explorerFixture(t)
		v.OpenSimilarityCohort([]string{v.FileAt(0).Path()})
		for _, r := range "/absent" {
			v.handleTypedRune(r)
		}
		if len(v.grid.ResultIndexes()) != 0 || !v.grid.Visible() || !v.browsing.has(browsingExplorer) {
			t.Fatal("empty filter retired the editable cohort")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if !v.grid.Visible() || len(v.grid.ResultIndexes()) != 1 {
			t.Fatal("filter Escape lost its cohort")
		}
	})
	t.Run("explorer_last_member_load_failure", func(t *testing.T) {
		v := explorerFixture(t)
		path := v.FileAt(16).Path()
		v.OpenSimilarityCohort([]string{path})
		if err := os.Remove(path); err != nil {
			t.Fatal(err)
		}
		v.imgCache.Purge()
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		if !v.explorerMapActive() || v.display.Snapshot().Displayed.Source.Path() == v.FileAt(16).Path() {
			t.Fatal("failed cohort member loaded an unrelated successor instead of returning to Explorer")
		}
	})
	t.Run("explorer_last_member", func(t *testing.T) {
		v := explorerFixture(t)
		v.OpenSimilarityCohort([]string{v.FileAt(16).Path()})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		revision := v.display.RequestRevision()
		v.reconcileSources(sourceChange{kind: sourcesRemoved, removed: []int{16}})
		v.settleExplorer()
		if !v.explorerMapActive() || v.grid.Visible() || v.explorer.State().SessionCurrent {
			t.Fatal("exhausted cohort did not return to the retired Explorer map")
		}
		if got := v.preloadCandidates(); len(got) != 0 {
			t.Fatalf("exhausted Explorer preloaded the collection: %v", got)
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
		if v.display.RequestRevision() != revision {
			t.Fatal("exhausted Explorer navigated into the collection")
		}
	})
}

func TestBrowsingVisitLifecycle(t *testing.T) {
	for _, route := range []string{"leave_parent", "empty_drop", "ranked_v"} {
		t.Run("nested_search_retirement/"+route, func(t *testing.T) {
			v := explorerFixture(t)
			v.OpenSimilarityCohort([]string{v.FileAt(0).Path(), v.FileAt(1).Path()})
			publish := streamingSearchFrom(t, v)
			publish(similarity.SearchFinal, 2, 1)
			retired := searchDelivery{visit: v.visualsearch.State().Visit, binding: v.browsing.current().binding, revision: v.browsing.revision}
			switch route {
			case "leave_parent":
				v.LeaveSimilarityMap()
			case "empty_drop":
				v.SetMergeMode(true)
				v.handleDrop([]fyne.URI{storage.NewFileURI(t.TempDir())})
				waitForScan(t, v)
				if v.FileCount() != 18 {
					t.Fatal("empty merge fixture replaced its existing collection")
				}
			case "ranked_v":
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyV})
			}
			v.visualsearch.Settle()
			if v.searchActive() || v.visualsearch.Active() {
				t.Fatal("retired root search left its feature/producer active")
			}
			v.applySearchDelivery(retired)
			if route == "ranked_v" {
				if !v.browsing.has(browsingExplorer) || !v.grid.Visible() {
					t.Fatal("ranked V no longer returns to its retained cohort")
				}
			} else if v.browsing.has(browsingExplorerMap) || v.grid.Visible() {
				t.Fatal("retired search delivery revived its closed parent")
			}
		})
	}
	for _, key := range []fyne.KeyName{fyne.KeyEscape, fyne.KeyG} {
		t.Run("explorer_to_map_then_ordinary_grid/"+string(key), func(t *testing.T) {
			v := explorerFixture(t)
			v.OpenSimilarityCohort([]string{v.FileAt(0).Path(), v.FileAt(1).Path()})
			locationMenu(t, v).Action()
			v.locationMap.Settle()
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
			v.showWindowGrid()
			v.grid.Settle()
			if visit := v.grid.CaptureVisit(); visit.Subset != nil || len(v.grid.ResultIndexes()) != v.FileCount() {
				t.Error("ordinary Grid revived a retired Explorer subset")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: key})
			if v.grid.Visible() || v.browsing.has(browsingExplorerMap) {
				t.Fatal("stale subset return trapped the ordinary Grid")
			}
		})
	}
	t.Run("retired_cluster_request", func(t *testing.T) {
		v := newTestViewer(t)
		a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
		dropAndWait(t, v, a)
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		members := []fileidentity.Occurrence{v.locationMap.Points()[0].Source.Identity}
		v.LeaveLocationMap()
		v.OpenLocationCluster(members)
		v.grid.Settle()
		if v.grid.Visible() || v.locationVisitActive() {
			t.Fatal("retired map request revived a cluster Grid")
		}
	})
	t.Run("search_shutdown", func(t *testing.T) {
		v, publish := streamingSearch(t)
		savedSession, savedPreferences := session.Load(v.app), preferences.Load(v.app)
		t.Cleanup(func() {
			session.Save(v.app, savedSession)
			preferences.Save(v.app, savedPreferences)
		})
		publish(similarity.SearchFinal, 2, 1)
		retired := searchDelivery{visit: v.visualsearch.State().Visit, binding: v.browsing.current().binding, revision: v.browsing.revision}
		lifecycle := v.app.Lifecycle().(interface{ OnStopped() func() })
		previous := lifecycle.OnStopped()
		registerShutdown(v.app, v)
		shutdown := lifecycle.OnStopped()
		v.app.Lifecycle().SetOnStopped(previous)
		shutdown()
		v.visualsearch.Settle()
		if v.searchActive() || v.visualsearch.Active() {
			t.Fatal("shutdown retained search visit authority")
		}
		v.applySearchDelivery(retired)
		if v.searchActive() {
			t.Fatal("shutdown accepted a retired search presentation")
		}
	})
	for _, retirement := range []string{"close_reopen", "collection_replacement", "source_removal"} {
		t.Run("explorer/"+retirement, func(t *testing.T) {
			v := explorerFixture(t)
			v.OpenSimilarityCohort([]string{v.FileAt(0).Path(), v.FileAt(1).Path()})
			old := v.browsing.current().binding
			switch retirement {
			case "close_reopen":
				v.LeaveSimilarityMap()
				v.OpenSimilarityCohort([]string{v.FileAt(2).Path()})
			case "collection_replacement":
				dropAndWait(t, v, uitest.TempJPEGURI(t, "replacement.jpg", 24, 16, color.White))
			case "source_removal":
				v.reconcileSources(sourceChange{kind: sourcesRemoved, removed: []int{1}})
			}
			before := v.browsing.current()
			v.returnExplorerMap(old)
			if v.browsing.current().binding != before.binding || v.explorerMapActive() {
				t.Fatal("obsolete Explorer return revived a retired presentation")
			}
			if retirement == "source_removal" {
				fynetest.Tap(explorerButton(t, v, "Back to map"))
				if !v.explorerMapActive() {
					t.Fatal("reconciled Grid did not install a current return binding")
				}
			}
		})
	}
}

func TestBrowsingDeferredReturns(t *testing.T) {
	t.Run("search_exit_refused", func(t *testing.T) {
		v, publish := streamingSearch(t)
		publish(similarity.SearchFinal, 2, 1)
		binding := v.browsing.current().binding
		prompt := dialog.NewInformation("Test", "Cover", v.win)
		prompt.Show()
		v.visualsearch.Exit()
		if !v.searchActive() || !v.visualsearch.Active() || v.browsing.current().binding != binding {
			t.Fatal("refused Exit retired the producer or its retained browsing origin")
		}
		prompt.Hide()
		if !v.searchActive() {
			t.Fatal("refused Exit replayed on dismissal")
		}
		v.visualsearch.Exit()
		if v.searchActive() || !v.grid.Visible() || len(v.grid.ResultIndexes()) != 4 {
			t.Fatal("fresh Exit did not restore the retained Grid origin")
		}
	})
	t.Run("search_retired_delivery", func(t *testing.T) {
		v, publish := streamingSearch(t)
		publish(similarity.SearchPartial, 2, 1)
		prompt := dialog.NewInformation("Test", "Cover", v.win)
		prompt.Show()
		publish(similarity.SearchFinal, 3, 2)
		if v.searchView.pending == nil {
			t.Fatal("fixture did not defer ranked delivery")
		}
		retired := *v.searchView.pending
		prompt.Hide()
		v.visualsearch.Exit()
		v.visualsearch.Settle()
		publish = streamingSearchFrom(t, v)
		publish(similarity.SearchFinal, 1)
		before := v.grid.ResultIndexes()
		v.applySearchDelivery(retired)
		if !slices.Equal(v.grid.ResultIndexes(), before) {
			t.Fatal("retired ranked delivery replaced a newer search visit")
		}
	})
	t.Run("explorer_admission_rechecked", func(t *testing.T) {
		v := explorerFixture(t)
		v.OpenSimilarityCohort([]string{v.FileAt(0).Path()})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		binding := v.browsing.current().binding
		prompt := dialog.NewInformation("Test", "Cover", v.win)
		prompt.Show()
		before, _ := v.explorer.Cohort()
		v.OpenSimilarityCohort([]string{v.FileAt(1).Path()})
		if after, _ := v.explorer.Cohort(); !slices.Equal(after, before) {
			t.Fatal("refused cohort delivery replaced the retained visit membership")
		}
		v.returnExplorerMap(binding)
		v.returnExplorerGrid()
		if v.explorerMapActive() || v.grid.Visible() || v.browsing.current().surface != browsingImage {
			t.Fatal("covered Explorer return bypassed current admission")
		}
		prompt.Hide()
		if v.grid.Visible() || v.explorerMapActive() {
			t.Fatal("refused Explorer return replayed automatically")
		}
		v.returnExplorerGrid()
		if !v.grid.Visible() {
			t.Fatal("fresh admitted Grid return refused")
		}
	})
}

func TestBrowsingScope(t *testing.T) {
	t.Run("recovery_is_not_next", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg", "e.jpg")
		scope := v.captureBrowsingScope()
		scope.restricted, scope.indexes = true, []int{1, 4}
		for _, tc := range []struct{ failed, restored, want int }{
			{2, -1, 4}, {4, -1, 4}, {5, -1, 1}, {2, 1, 1},
		} {
			if got, ok := scope.Recover(tc.failed, tc.restored); !ok || got != tc.want {
				t.Fatalf("Recover(%d,%d) = (%d,%t), want %d", tc.failed, tc.restored, got, ok, tc.want)
			}
		}
		if got, _ := scope.Next(2, 1); got != 1 {
			t.Fatal("ordinary Next's missing-current rule changed")
		}
		scope.indexes = nil
		if _, ok := scope.Recover(2, -1); ok {
			t.Fatal("empty recovery scope widened to the collection")
		}
		if _, ok := scope.Recover(2, 1); ok {
			t.Fatal("restored image outside the reconciled restriction was accepted")
		}
	})
	t.Run("snapshot_binding_and_discovery", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg")
		ordinary := v.captureBrowsingScope()
		if ordinary.restricted || !ordinary.complete || ordinary.binding.kind != browsingCollection || ordinary.binding.collection != v.Generation() {
			t.Fatalf("ordinary collection observation: %+v", ordinary)
		}
		publish := streamingSearchFrom(t, v)
		pending := v.captureBrowsingScope()
		if !pending.restricted || pending.complete || pending.binding.kind != browsingSearch {
			t.Fatalf("pending discovery observation: %+v", pending)
		}
		publish(similarity.SearchPartial, 2, 1)
		captured := v.captureBrowsingScope()
		binding := captured.binding
		key := v.FileAt(2).String()
		publish(similarity.SearchFinal, 3, 2)
		if !slices.Equal(captured.indexes, []int{0, 2, 1}) || captured.complete || captured.binding != binding || captured.collection.KeyAt(2) != key {
			t.Fatalf("publication changed the captured order/identity: %+v", captured)
		}
		latest := v.captureBrowsingScope()
		if !latest.complete || !slices.Equal(latest.indexes, []int{0, 3, 2}) {
			t.Fatalf("new capture did not observe the completed ranking: %+v", latest)
		}
		v.visualsearch.Exit()
		v.visualsearch.Settle()
		streamingSearchFrom(t, v)
		if next := v.captureBrowsingScope(); next.binding == binding {
			t.Fatal("a reopened search reused the retired visit binding")
		}
	})
	t.Run("restricted_without_current_members", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		// A delivered cohort can outlive its last source in the collection.
		// Keep the existing image visible while that empty visit is retained.
		v.OpenSimilarityCohort([]string{filepath.Join(t.TempDir(), "missing.jpg")})
		v.grid.Close()
		v.ShowImage(0)
		waitUntilLoaded(t, v)
		if got := v.preloadCandidates(); len(got) != 0 {
			t.Errorf("empty cohort preloaded unrelated collection images: %v", got)
		}
		before := v.display.RequestRevision()
		for _, key := range []fyne.KeyName{fyne.KeyRight, fyne.KeyLeft, fyne.KeyHome, fyne.KeyEnd} {
			v.handleKeyEvent(&fyne.KeyEvent{Name: key})
			v.display.Settle()
			if v.state.index != 0 || v.display.RequestRevision() != before {
				t.Fatalf("%s selected an image outside an empty cohort: index=%d revision=%d", key, v.state.index, v.display.RequestRevision())
			}
		}
	})
}

func TestBrowsingActionTargets(t *testing.T) {
	t.Run("frozen_image_favorite_during_new_rankings", func(t *testing.T) {
		v, publish := streamingSearch(t)
		v.favorites.SetDir(t.TempDir())
		publish(similarity.SearchPartial, 2, 1)
		v.grid.SimulateHover(1)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		want := []fyne.URI{v.FileAt(0), v.FileAt(2), v.FileAt(1)}
		publish(similarity.SearchFinal, 3, 2)
		v.favorites.AddCurrentList()
		publish(similarity.SearchFinal, 3, 1)
		saveBrowsingFavorite(t, v, want)
	})
	t.Run("filtered_ranked_grid_favorite", func(t *testing.T) {
		v, publish := streamingSearch(t)
		v.favorites.SetDir(t.TempDir())
		publish(similarity.SearchPartial, 2, 1)
		v.grid.HandleRune('/')
		v.grid.HandleRune('b')
		want := []fyne.URI{v.FileAt(1)}
		v.favorites.AddCurrentList()
		publish(similarity.SearchFinal, 3, 2)
		saveBrowsingFavorite(t, v, want)
	})
	for _, visit := range []string{"explorer", "map_direct", "map_cluster"} {
		t.Run("ordinary_favorite/"+visit, func(t *testing.T) {
			v := newTestViewer(t)
			v.favorites.SetDir(t.TempDir())
			a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
			b := uitest.TempJPEGURI(t, "b.jpg", 24, 16, color.White)
			dropAndWait(t, v, a, b)
			want := []fyne.URI{a, b}
			if visit == "explorer" {
				v.OpenSimilarityCohort([]string{a.Path()})
			} else {
				locationMenu(t, v).Action()
				v.locationMap.Settle()
				point := v.locationMap.Points()[0]
				if visit == "map_direct" {
					v.OpenLocationImage(point.Source.Identity)
					waitUntilLoaded(t, v)
				} else {
					v.OpenLocationCluster([]fileidentity.Occurrence{point.Source.Identity})
				}
			}
			v.favorites.AddCurrentList()
			saveBrowsingFavorite(t, v, want)
		})
	}
	t.Run("highlighted_copy_survives_rank_and_visit_change", func(t *testing.T) {
		v, publish := streamingSearch(t)
		publish(similarity.SearchPartial, 2, 1)
		v.grid.ClearSelection()
		v.grid.SimulateHover(1)
		want := v.FileAt(2).Path()
		started, release := make(chan struct{}), make(chan struct{})
		var once sync.Once
		unblock := func() { once.Do(func() { close(release) }) }
		defer unblock()
		var copied []string
		uitest.StubClipboardCopyFiles(t, func(paths []string) error {
			close(started)
			<-release
			copied = slices.Clone(paths)
			return nil
		})
		v.copySelection()
		select {
		case <-started:
		case <-time.After(testTimeout):
			t.Fatal("highlighted-file copy did not reach the clipboard")
		}
		publish(similarity.SearchFinal, 3, 1)
		v.showViewer()
		unblock()
		waitForClipboard(t, v)
		if !slices.Equal(copied, []string{want}) {
			t.Fatalf("highlighted copy was retargeted: %v", copied)
		}
		if v.searchActive() {
			t.Fatal("fixture did not leave the original search visit")
		}
	})
}

func saveBrowsingFavorite(t *testing.T, v *viewer, want []fyne.URI) {
	t.Helper()
	entry := v.win.Canvas().Focused()
	if entry == nil || v.win.Canvas().Overlays().Top() == nil {
		t.Fatal("Favorite action did not open its naming interaction")
	}
	for _, r := range "Captured" {
		entry.TypedRune(r)
	}
	entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	got, err := favstore.Load(v.favorites.Dir(), "Captured")
	if err != nil {
		t.Fatal(err)
	}
	if !slices.EqualFunc(got, want, func(a, b fyne.URI) bool { return a.String() == b.String() }) {
		t.Fatalf("Favorite sources changed after capture: got %v, want %v", got, want)
	}
}
