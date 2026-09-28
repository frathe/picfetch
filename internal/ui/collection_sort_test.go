package ui

import (
	"bytes"
	"fmt"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	fynetest "fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestCollectionSortHandoff(t *testing.T) {
	for _, pending := range []bool{false, true} {
		name := "loaded_latest"
		if pending {
			name = "pending_latest"
		}
		t.Run(name, func(t *testing.T) {
			v := newTestViewer(t)
			a := uitest.TempJPEGURI(t, "a.jpg", 4, 5, color.White)
			b := uitest.TempJPEGURI(t, "b.jpg", 6, 7, color.Black)
			dropAndWait(t, v, b)
			v.display.WaitPreloads()
			for i, uri := range []fyne.URI{b, a} {
				stamp := time.Unix(int64(1000+i), 0)
				if err := os.Chtimes(uri.Path(), stamp, stamp); err != nil {
					t.Fatal(err)
				}
			}
			sortEntered, releaseSort := make(chan struct{}), make(chan struct{})
			var sortOnce, releaseSortOnce sync.Once
			unblockSort := func() { releaseSortOnce.Do(func() { close(releaseSort) }) }
			defer unblockSort()
			sortedSource := uitest.ReaderURI(b, func() (io.ReadCloser, error) {
				sortOnce.Do(func() { close(sortEntered); <-releaseSort })
				return os.Open(b.Path())
			})
			cached, _ := v.imgCache.Get(b.String())
			v.imgCache.Add(sortedSource.String(), cached)
			loadEntered, releaseLoad := make(chan struct{}), make(chan struct{})
			oldPixels := uitest.EncodePNG(t, 11, 12, color.Black)
			var loadOpens atomic.Int32
			var releaseLoadOnce sync.Once
			unblockLoad := func() { releaseLoadOnce.Do(func() { close(releaseLoad) }) }
			defer unblockLoad()
			chosen := uitest.ReaderURI(a, func() (io.ReadCloser, error) {
				if pending && loadOpens.Add(1) == 1 {
					close(loadEntered)
					<-releaseLoad
					// A retired decode must never replace the fresh 4x5 source.
					return io.NopCloser(bytes.NewReader(oldPixels)), nil
				}
				return os.Open(a.Path())
			})
			v.state.Replace(collectionInput{source: []fyne.URI{sortedSource, chosen, chosen}, display: []fyne.URI{chosen, chosen, sortedSource}, index: 2, favorite: "favorite"})
			before := v.state.Observe()
			v.SetSortMode(filesort.ByCaptureDate)
			select {
			case <-sortEntered:
			case <-time.After(testTimeout):
				t.Fatal("sort did not reach its held metadata read")
			}
			v.ShowImage(1)
			if pending {
				select {
				case <-loadEntered:
				case <-time.After(testTimeout):
					t.Fatal("navigation did not reach its independently held decode")
				}
				if outgoing, ok := v.displayedFile(); !ok || outgoing.String() != b.String() {
					t.Fatal("pending fixture has no distinct outgoing pixels")
				}
			} else {
				waitUntilLoaded(t, v)
			}
			oldLoad, revision := v.display.LoadDone(), v.display.RequestRevision()
			unblockSort()
			waitForSort(t, v)
			if current, index, ok := v.CurrentFile(); !ok || current.String() != chosen.String() || index != 2 {
				t.Fatalf("sort restored start-time/outgoing choice: %v/%d/%v; want second a at 2", current, index, ok)
			}
			if v.Generation() != before.Generation()+1 || v.state.Observe().Favorite() != "favorite" {
				t.Fatal("sort did not preserve association with one publication")
			}
			if v.display.RequestRevision() != revision+1 {
				t.Fatal("sort did not admit exactly one authoritative display handoff")
			}
			waitHandle(t, "retired pre-sort load", oldLoad)
			unblockLoad()
			waitUntilLoaded(t, v)
			if got := v.display.Snapshot(); got.Displayed.Source.String() != chosen.String() || v.img.Image.Bounds().Dx() != 4 || got.Loading {
				t.Fatal("retired decode published after the sort handoff")
			}
			if cached, ok := v.imgCache.Get(chosen.String()); !ok || cached.Frames[0].Bounds().Dx() != 4 {
				t.Fatal("retired decode replaced the post-handoff cache entry")
			}
			v.StepImage(-1)
			waitUntilLoaded(t, v)
			if _, index, _ := v.CurrentFile(); index != 1 {
				t.Fatal("navigation did not continue from the later repeated occurrence")
			}
		})
	}
	t.Run("canceled", TestCaptureSort_CancellationRestoresMenuAndAllowsRetry)
	t.Run("superseded", TestCaptureSort_SupersededCompletionPreservesCancellationBaseline)
}

func collectionReorderReconciliation(t *testing.T) {
	t.Run("empty_scope", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		v.OpenSimilarityCohort([]string{filepath.Join(t.TempDir(), "missing.jpg")})
		v.grid.Close()
		v.ShowImage(0)
		waitUntilLoaded(t, v)
		before := v.display.Snapshot().Displayed
		v.SetSortMode(filesort.ByDropOrder)
		waitForSort(t, v)
		v.display.Settle()
		if scope := v.captureBrowsingScope(); !scope.restricted || len(scope.indexes) != 0 || !v.explorerMapActive() {
			t.Fatal("reorder widened an exhausted cohort instead of returning to its parent")
		}
		if v.display.Snapshot().Displayed != before || len(v.preloadCandidates()) != 0 {
			t.Fatal("empty-scope reorder loaded or preloaded an unrelated image")
		}
	})
	t.Run("explicit_origin", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		duplicate := v.FileAt(1)
		v.SetMergeMode(true)
		dropAndWait(t, v, duplicate)
		v.grid.Close()
		v.ShowImage(2)
		waitUntilLoaded(t, v)
		publish := streamingSearchFrom(t, v)
		publish(similarity.SearchFinal, 0, 3)
		v.ShowImage(0)
		waitUntilLoaded(t, v)
		v.SetSortMode(filesort.ByDropOrder)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		if v.searchActive() || v.state.Observe().index != 3 || v.currentImageOccurrence().Ordinal != 1 || v.FileAt(3).Path() != duplicate.Path() {
			t.Fatal("latest ranked image displaced the explicit repeated image origin")
		}
	})
	t.Run("notifications", func(t *testing.T) {
		v, publish := streamingSearch(t)
		publish(similarity.SearchFinal, 2, 1)
		before := v.state.Observe()
		observed, leaked := false, false
		v.grid.SetOnResultChanged(func() {
			observed = true
			v.syncMenus()
			leaked = leaked || !v.menus.Actions().Hide().Disabled || v.Generation() != before.Generation()+1
		})
		v.SetSortMode(filesort.ByDropOrder)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		if !observed || leaked || v.searchActive() || v.menus.Actions().Hide().Disabled || !v.grid.Visible() {
			t.Fatal("reorder notified intermediate generation/menu state or lost the Grid origin")
		}
	})
	t.Run("cohort", func(t *testing.T) {
		v := explorerFixture(t)
		paths := []string{v.FileAt(14).Path(), v.FileAt(15).Path()}
		v.OpenSimilarityCohort(paths)
		v.grid.SelectAll()
		v.grid.SimulateHover(1)
		before := v.grid.CaptureVisit()
		v.SetSortMode(filesort.ByDropOrder)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		v.grid.Settle()
		after := v.grid.CaptureVisit()
		if !v.grid.Visible() || !slices.Equal(explorerGridPaths(v), paths) || !slices.Equal(after.Selected, before.Selected) || after.Highlight != before.Highlight {
			t.Fatal("reorder failed to reconcile the retained cohort Grid")
		}
		fynetest.Tap(explorerButton(t, v, "Back to map"))
		if !v.explorerMapActive() {
			t.Fatal("reorder retained an obsolete cohort return binding")
		}
	})
	t.Run("cluster", func(t *testing.T) {
		v := newTestViewer(t)
		v.win.Resize(fyne.NewSize(1000, 700))
		a := uitest.TempGPSJPEGURI(t, "same.jpg", 24, 16, 52.52, 13.405)
		b := uitest.TempGPSJPEGURI(t, "outside.jpg", 24, 16, 51.507, -.128)
		v.OpenFavorite("cluster-favorite", []fyne.URI{a, b, a})
		waitForScan(t, v)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		locationMenu(t, v).Action()
		v.locationMap.Settle()
		fynetest.Tap(locationButton(t, v, fmt.Sprintf(lang.L("%d images"), 2)))
		v.grid.Settle()
		v.grid.SimulateHover(1)
		v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
		before := v.grid.CaptureVisit()
		v.SetSortMode(filesort.ByDropOrder)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		v.grid.Settle()
		v.locationMap.Settle()
		after := v.grid.CaptureVisit()
		if !v.grid.Visible() || !slices.Equal(v.grid.ResultIndexes(), []int{0, 2}) || !slices.Equal(after.Selected, before.Selected) || !slices.Equal(v.grid.Selection(), []int{2}) {
			t.Fatal("reorder widened a frozen cluster or lost its repeated selection")
		}
		fynetest.Tap(explorerButton(t, v, "Back to map"))
		v.locationMap.Settle()
		if !v.locationMapVisible() || !locationSurface(t, v).Visible() {
			t.Fatal("reorder retained an obsolete location-map return")
		}
	})
}

func TestCollectionChangeKinds(t *testing.T) {
	t.Run("content_and_policy", collectionContentAndPolicyEffects)
	t.Run("removal", collectionRemovalEffects)
	t.Run("reorder", func(t *testing.T) {
		v := openGridWith(t, "b.jpg", "a.jpg", "c.jpg")
		v.display.WaitPreloads()
		v.grid.Settle()
		key := v.FileAt(0).String()
		cached, ok := v.imgCache.Get(key)
		if !ok {
			t.Fatal("fixture source was not cached")
		}
		writer := v.imgCache.Capture()
		facts := v.dupes.CaptureFacts()
		facts.PutHash(key, 123)
		v.grid.SimulateHover(1)
		v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
		v.SetSortMode(filesort.ByDropOrder)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		v.grid.Settle()
		if v.grid.SelectionCount() != 0 || !slices.Equal(v.grid.ResultIndexes(), []int{0, 1, 2}) {
			t.Fatal("ordinary Grid reorder became bookmark restoration")
		}
		if got, ok := v.imgCache.Get(key); !ok || got != cached || !writer.AddIfRoom("unrelated-cache-entry", cached) {
			t.Fatal("reorder purged content cache or invalidated unrelated content writers")
		}
		if hash, ok := v.dupes.Hash(key); !ok || hash != 123 || facts.PutHash(key, 456) {
			t.Fatal("reorder lost established duplicate facts or admitted an obsolete producer")
		}
	})
}
