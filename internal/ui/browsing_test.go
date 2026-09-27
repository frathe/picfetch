package ui

import (
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	fynetest "fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/grid"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestBrowsingVisitTransitions(t *testing.T) {
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
	t.Run("search_shutdown", func(t *testing.T) {
		v, publish := streamingSearch(t)
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
