package ui

import (
	"path/filepath"
	"slices"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/similarity"
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
