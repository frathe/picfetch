package explorer_test

import (
	"context"
	"image/color"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/explorerpresets"
	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestFeatureFavoriteSaveRollback(t *testing.T) {
	for _, kind := range []string{"cohort", "preset"} {
		t.Run(kind, func(t *testing.T) {
			app := test.NewApp()
			t.Cleanup(app.Quit)
			host := &featureHost{win: app.NewWindow("Explorer")}
			files := []fyne.URI{storage.NewFileURI(filepath.Join(t.TempDir(), "a.jpg")), storage.NewFileURI(filepath.Join(t.TempDir(), "b.jpg"))}
			paths := []string{files[0].Path(), files[1].Path()}
			favorites := t.TempDir()
			if err := favstore.Save(favorites, "Cats", files); err != nil {
				t.Fatal(err)
			}
			favorite := favstore.Dir(favorites, "Cats")
			preview := uitest.EncodeJPEG(t, 32, 24, color.White)
			provider := func(_ context.Context, sources []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
				var items []similarity.Item
				for _, path := range sources {
					items = append(items, similarity.Item{Path: path, Cohort: "unassigned", Tags: []string{"cat"}, Preview: preview, Facts: similarity.ImageFacts{Version: similarity.FactsVersion, Make: "Canon"}})
				}
				emit(similarity.Event{Complete: true, Total: len(items), Successful: len(items), Items: items})
				return nil
			}
			f := explorer.NewFeature(host, explorer.Options{App: app, Queue: &uitest.UIQueue{}, Analyze: provider, Presets: &explorerpresets.Store{Dir: t.TempDir()}})
			t.Cleanup(func() { f.Stop(); f.Settle() })
			host.win.SetContent(f.Surface().Overlay())
			f.Open(explorer.OpenRequest{Sources: paths, FavoriteDir: favorite})
			f.Settle()
			before := f.Surface().Cohorts()
			var save *widget.Button
			if kind == "cohort" {
				f.OpenSimilarityUnassigned(paths)
				f.AnalyzeSelection(paths)
				featureEntry(t, host, "Cohort name", "Saved cats")
				save = featureButton(t, host, "Create cohort")
			} else {
				f.ShowSimilarityPresets()
				f.Settle()
				test.Tap(featureButton(t, host, "New preset"))
				featureEntry(t, host, "Preset name", "Saved cats")
				featureEntry(t, host, "Camera make", "Canon")
				test.Tap(featureButton(t, host, "Save preset"))
				f.Settle()
				test.Tap(featureButton(t, host, "Saved cats"))
				test.Tap(featureButton(t, host, "Preview preset"))
				f.Settle()
				save = featureButton(t, host, "Apply preset")
			}
			if save.Disabled() {
				t.Fatal("fixture did not prepare an applicable edit")
			}
			if err := os.RemoveAll(favorite); err != nil {
				t.Fatal(err)
			}
			test.Tap(save)
			f.Settle()
			if !reflect.DeepEqual(f.Surface().Cohorts(), before) {
				t.Fatal("failed Favorite save retained proposed memberships")
			}
			explained := false
			walkFeature(host.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
				if label, ok := o.(*widget.Label); ok && label.Text == lang.L("Could not save the cohort. Reopen the favorite and try again.") {
					explained = true
				}
			})
			if !explained || !f.State().DialogOpen || f.State().CohortSaving {
				t.Fatal("failed Favorite edit did not remain recoverable")
			}
		})
	}
}

func TestFeatureCohortRetainsCapturedMembership(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	host := &featureHost{win: app.NewWindow("Explorer")}
	f := explorer.NewFeature(host, explorer.Options{Queue: &uitest.UIQueue{}})
	t.Cleanup(func() { f.Stop(); f.Settle() })
	paths := []string{"/a.jpg", "/b.jpg"}
	f.OpenSimilarityCohort(paths)
	paths[0] = "/changed.jpg"
	captured, _ := f.Cohort()
	captured[1] = "/changed-too.jpg"
	again, unassigned := f.Cohort()
	if unassigned || !slices.Equal(again, []string{"/a.jpg", "/b.jpg"}) {
		t.Fatalf("cohort membership changed with a caller: %v", again)
	}
	f.RemoveCohortSource("/a.jpg")
	again, _ = f.Cohort()
	if !slices.Equal(again, []string{"/b.jpg"}) {
		t.Fatalf("removed source survived cohort reconciliation: %v", again)
	}
}
