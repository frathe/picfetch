package ui

import (
	"image/color"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/uitest"
)

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
