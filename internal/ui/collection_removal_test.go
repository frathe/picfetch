package ui

import (
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	fynetest "fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/session"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/deletion"
	"github.com/frathe/picfetch/internal/uitest"
)

func collectionRemovalReconciliation(t *testing.T) {
	t.Run("repeated_image_origin", func(t *testing.T) {
		v := newTestViewer(t)
		a, b, c := collectionRemovalSources(t)
		v.OpenFavorite("favorite", []fyne.URI{a, b, a, c, a})
		waitForScan(t, v)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		v.ShowImage(1)
		waitUntilLoaded(t, v)
		publish := streamingSearchFrom(t, v)
		publish(similarity.SearchFinal, 3, 4)
		before := v.Generation()
		v.RemoveFiles([]int{4, 0, 0})
		waitUntilLoaded(t, v)
		if v.Generation() != before+1 || v.searchActive() || v.grid.Visible() || v.currentImageOccurrence().Ordinal != 0 || v.state.index != 0 || v.FileAt(0).String() != a.String() {
			t.Fatal("batch survivor mapping lost the retained later image occurrence")
		}
	})
	t.Run("image_origin_single_load", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg")
		v.grid.Close()
		v.ShowImage(1)
		waitUntilLoaded(t, v)
		origin := v.FileAt(1)
		publish := streamingSearchFrom(t, v)
		publish(similarity.SearchFinal, 0, 3)
		v.ShowImage(3)
		waitUntilLoaded(t, v)
		revision := v.display.RequestRevision()
		uitest.StubTrashMove(t, os.Remove)
		collectionTrash(t, v, v.FileAt(0), v.FileAt(2))
		waitUntilLoaded(t, v)
		if v.searchActive() || v.grid.Visible() || v.display.RequestRevision() != revision+1 || v.FileAt(v.state.index).String() != origin.String() {
			t.Fatal("Trash did not restore the explicit image origin with exactly one load")
		}
	})
	for _, owner := range []string{"collection", "cohort", "cluster"} {
		t.Run("partial_"+owner, func(t *testing.T) {
			v := newTestViewer(t)
			v.win.Resize(fyne.NewSize(1000, 700))
			a := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
			b := uitest.TempGPSJPEGURI(t, "b.jpg", 24, 16, 52.52, 13.405)
			c := uitest.TempJPEGURI(t, "c.jpg", 24, 16, color.White)
			dropAndWait(t, v, a, b, c)
			v.display.WaitPreloads()
			v.state.Replace(collectionInput{source: []fyne.URI{a, b, a, c}, display: []fyne.URI{a, a, b, c}, retained: []collectionSource{{a, false}, {a, true}, {b, false}, {b, true}, {a, false}, {c, false}}, favorite: "favorite"})
			switch owner {
			case "collection":
				v.grid.Toggle()
			case "cohort":
				v.OpenSimilarityCohort([]string{a.Path(), b.Path()})
			case "cluster":
				locationMenu(t, v).Action()
				v.locationMap.Settle()
				fynetest.Tap(locationButton(t, v, "3 images"))
			}
			v.grid.Settle()
			v.grid.SimulateHover(2)
			v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
			// Location visits intentionally refuse search; keep that frozen map
			// origin directly while collection/cohort origins nest a search.
			if owner != "cluster" {
				publish := streamingSearchFrom(t, v)
				publish(similarity.SearchFinal, 0, 3)
			}
			before := v.Generation()
			uitest.StubTrashMove(t, func(path string) error {
				if path == b.Path() {
					return errors.New("failed b")
				}
				return os.Remove(path)
			})
			collectionTrash(t, v, a, b)
			waitUntilLoaded(t, v)
			v.grid.Settle()
			v.locationMap.Settle()
			if v.Generation() != before+1 || v.searchActive() || !v.grid.Visible() || !slices.Equal(v.grid.Selection(), []int{0}) || !slices.Equal(v.state.Observe().Capture(collectionSourceOrder), []fyne.URI{b, b, c}) {
				t.Fatal("partial batch lost the failed target, retained unavailable entry or restored Grid selection")
			}
			if owner != "collection" {
				if !slices.Equal(v.grid.ResultIndexes(), []int{0}) {
					t.Fatal("partial batch widened its retained subset")
				}
				fynetest.Tap(explorerButton(t, v, "Back to map"))
				v.locationMap.Settle()
				if owner == "cohort" && !v.explorerMapActive() || owner == "cluster" && !v.locationMapVisible() {
					t.Fatal("partial batch retained an obsolete subset return")
				}
			}
		})
	}
	for _, imageOrigin := range []bool{false, true} {
		t.Run("exhausted_trash/"+fmt.Sprint(imageOrigin), func(t *testing.T) {
			v := explorerFixture(t)
			member := v.FileAt(14)
			v.OpenSimilarityCohort([]string{member.Path()})
			if imageOrigin {
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
				waitUntilLoaded(t, v)
			}
			publish := streamingSearchFrom(t, v)
			publish(similarity.SearchFinal, 0, 1)
			revision := v.display.RequestRevision()
			uitest.StubTrashMove(t, os.Remove)
			collectionTrash(t, v, member)
			v.display.Settle()
			if v.searchActive() || v.grid.Visible() || !v.explorerMapActive() || v.display.RequestRevision() != revision {
				t.Fatal("completed Trash revived an exhausted scope or loaded an unrelated collection image")
			}
		})
	}
}

func collectionCaptureAfterRemoval(t *testing.T) {
	v := newTestViewer(t)
	v.favorites.SetDir(t.TempDir())
	a, b, c := collectionRemovalSources(t)
	u := storage.NewFileURI(filepath.Join(t.TempDir(), "unavailable.heic"))
	dropAndWait(t, v, a, b, c)
	v.state.Replace(collectionInput{source: []fyne.URI{c, a, b, a}, display: []fyne.URI{a, a, b, c}, retained: []collectionSource{{c, false}, {u, true}, {a, false}, {b, false}, {a, false}, {u, true}}, favorite: "favorite"})
	v.RemoveFile(1)
	v.ReconcileDeletedFiles([]fyne.URI{c}, "removed fixture")
	want := []fyne.URI{u, a, b, u}
	if got := v.state.Observe().Capture(collectionSourceOrder); !slices.Equal(got, want) {
		t.Fatalf("capture resurrected removals or moved gaps: %v", got)
	}
	v.favorites.AddCurrentList()
	saveBrowsingFavorite(t, v, want)
	lifecycle := v.app.Lifecycle().(interface{ OnStopped() func() })
	previous := lifecycle.OnStopped()
	registerShutdown(v.app, v)
	shutdown := lifecycle.OnStopped()
	v.app.Lifecycle().SetOnStopped(previous)
	shutdown()
	if got := session.Load(v.app); !slices.EqualFunc(got, want, sameURI) {
		t.Fatalf("session save resurrected removed members: %v", got)
	}
	t.Cleanup(func() { session.Save(v.app, nil) })
}

func collectionRemovalEffects(t *testing.T) {
	v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
	v.display.WaitPreloads()
	v.grid.Settle()
	removed, kept := v.FileAt(0), v.FileAt(1)
	cached, ok := v.imgCache.Get(kept.String())
	if !ok {
		t.Fatal("survivor fixture was not cached")
	}
	writer := v.imgCache.Capture()
	facts := v.dupes.CaptureFacts()
	facts.PutHash(kept.String(), 123)
	v.RemoveFiles([]int{0, 2})
	v.grid.Settle()
	if v.imgCache.Contains(removed.String()) {
		t.Fatal("root did not evict the committed removed identity")
	}
	if got, ok := v.imgCache.Get(kept.String()); !ok || got != cached || !writer.AddIfRoom("unrelated-entry", cached) {
		t.Fatal("removal purged unrelated content or invalidated its writer")
	}
	if hash, ok := v.dupes.Hash(kept.String()); !ok || hash != 123 || facts.PutHash(kept.String(), 456) {
		t.Fatal("removal lost surviving duplicate facts or admitted old-generation work")
	}
}

func collectionRemovalOccurrences(t *testing.T) {
	t.Run("uri_keys_and_path_bookmarks", func(t *testing.T) {
		a, b := storage.NewFileURI("/a.jpg"), storage.NewFileURI("/b.jpg")
		aliasKey := uitest.ReaderURI(a, nil)
		state := newAppState(0, false)
		state.Replace(collectionInput{source: []fyne.URI{a, b, aliasKey}, display: []fyne.URI{a, aliasKey, b}, index: 1})
		change := state.RemoveTargets([]fyne.URI{a})
		if !slices.Equal(change.after.SourceFiles(), []fyne.URI{b, aliasKey}) || change.survivors[fileidentity.Occurrence{Path: a.Path(), Ordinal: 1}] != (fileidentity.Occurrence{Path: a.Path()}) {
			t.Fatal("URI target matching was collapsed into path bookmark identity")
		}
		if uri, index, ok := change.after.Current(); !ok || uri != aliasKey || index != 0 {
			t.Fatal("removing an earlier path occurrence lost the surviving requested occurrence")
		}
	})
	for _, remove := range []int{0, 1} {
		name := "earlier"
		if remove == 1 {
			name = "later"
		}
		t.Run(name, func(t *testing.T) {
			a, b := storage.NewFileURI("/a.jpg"), storage.NewFileURI("/b.jpg")
			u, w := storage.NewFileURI("/u.heic"), storage.NewFileURI("/w.heic")
			state := newAppState(0, false)
			state.Replace(collectionInput{source: []fyne.URI{a, b, a}, display: []fyne.URI{a, a, b}, retained: []collectionSource{{a, false}, {u, true}, {b, false}, {a, false}, {w, true}}, index: 1, favorite: "favorite"})
			before := state.Observe()
			change := state.Remove([]int{remove, remove, -1, 10})
			wantSource, wantCapture := []fyne.URI{b, a}, []fyne.URI{u, b, a, w}
			if remove == 1 {
				wantSource, wantCapture = []fyne.URI{a, b}, []fyne.URI{a, u, b, w}
			}
			if change.after.Generation() != before.Generation()+1 || !slices.Equal(change.after.SourceFiles(), wantSource) || !slices.Equal(change.after.Capture(collectionSourceOrder), wantCapture) || change.after.Favorite() != "favorite" {
				t.Fatal("one-occurrence removal lost its exact source/gap membership")
			}
			kept := fileidentity.Occurrence{Path: a.Path(), Ordinal: 1 - remove}
			if len(change.survivors) != 2 || change.survivors[kept] != (fileidentity.Occurrence{Path: a.Path(), Ordinal: 0}) || change.survivors[fileidentity.Occurrence{Path: b.Path()}] != (fileidentity.Occurrence{Path: b.Path()}) {
				t.Fatalf("survivor map = %v", change.survivors)
			}
			if _, found := change.survivors[fileidentity.Occurrence{Path: a.Path(), Ordinal: remove}]; found {
				t.Fatal("removed occurrence was remapped to its surviving repeat")
			}
			if before.Count() != 3 || len(before.Retained()) != 5 || before.index != 1 {
				t.Fatal("removal mutated a retained old observation")
			}
			if noChange := state.Remove([]int{-1, 99}); noChange.after.Generation() != change.after.Generation() {
				t.Fatal("invalid removal published a new collection")
			}
		})
	}
	t.Run("targets", func(t *testing.T) {
		a, b := storage.NewFileURI("/a.jpg"), storage.NewFileURI("/b.jpg")
		state := newAppState(0, false)
		state.Replace(collectionInput{source: []fyne.URI{a, b, a}, display: []fyne.URI{a, a, b}, retained: []collectionSource{{a, false}, {a, true}, {b, false}, {a, false}, {b, true}}, favorite: "favorite"})
		change := state.RemoveTargets([]fyne.URI{a, a})
		if change.after.Generation() != change.before.Generation()+1 || !slices.Equal(change.after.Capture(collectionSourceOrder), []fyne.URI{b, b}) || change.after.Count() != 1 || change.after.Favorite() != "favorite" || len(change.survivors) != 1 {
			t.Fatal("target removal did not remove every matching available/unavailable occurrence once")
		}
		if empty := state.RemoveTargets([]fyne.URI{b}); empty.after.Count() != 0 || len(empty.after.Retained()) != 0 || empty.after.Favorite() != "" {
			t.Fatal("last full-member deletion did not clear association")
		}
		if noChange := state.RemoveTargets([]fyne.URI{a}); noChange.after.Generation() != noChange.before.Generation() {
			t.Fatal("unmatched completed target published a new collection")
		}
	})
}

func TestCollectionRemoval(t *testing.T) {
	t.Run("unmatched_keeps_search", func(t *testing.T) {
		v, publish := streamingSearch(t)
		publish(similarity.SearchFinal, 2, 1)
		before, revision := v.state.Observe(), v.display.RequestRevision()
		v.ReconcileDeletedFiles([]fyne.URI{storage.NewFileURI(filepath.Join(t.TempDir(), "outside.jpg"))}, "removed outside collection")
		if v.Generation() != before.Generation() || !v.searchActive() || !v.grid.Visible() || v.display.RequestRevision() != revision {
			t.Fatal("unmatched committed target retired the current collection or search")
		}
	})
	t.Run("partial_batch", func(t *testing.T) {
		v := newTestViewer(t)
		a, b, c := collectionRemovalSources(t)
		dropAndWait(t, v, a, b, c)
		v.display.WaitPreloads()
		v.state.Replace(collectionInput{
			source: []fyne.URI{a, b, a, c}, display: []fyne.URI{a, a, b, c},
			retained: []collectionSource{{a, false}, {a, true}, {b, false}, {a, false}, {c, false}, {c, true}}, favorite: "favorite",
		})
		v.grid.Toggle()
		v.grid.Settle()
		before := v.state.Observe()
		var moved []string
		uitest.StubTrashMove(t, func(path string) error {
			moved = append(moved, path)
			if path == b.Path() {
				return errors.New("held failure")
			}
			return os.Remove(path)
		})
		observed, intermediate := false, false
		v.grid.SetOnResultChanged(func() {
			observed = true
			after := v.state.Observe()
			intermediate = intermediate || after.Generation() != before.Generation()+1 || after.Count() != 1 || len(after.Retained()) != 1 || after.Favorite() != "favorite"
		})
		collectionTrash(t, v, a, a, b, c)
		waitUntilLoaded(t, v)
		after := v.state.Observe()
		if len(moved) != 3 || !strings.Contains(v.toast.text.Text, "2 of 3") {
			t.Fatalf("partial batch did not report unique successful targets: %v / %q", moved, v.toast.text.Text)
		}
		if !observed || intermediate || after.Generation() != before.Generation()+1 || !slices.Equal(after.Capture(collectionSourceOrder), []fyne.URI{b}) {
			t.Fatalf("batch did not publish final retained membership once: generation %d -> %d, retained %v, callback %v/%v", before.Generation(), after.Generation(), after.Retained(), observed, intermediate)
		}
		if before.Count() != 4 || len(before.Retained()) != 6 || before.Favorite() != "favorite" {
			t.Fatal("Trash mutated its pre-commit observation")
		}
		if _, err := os.Stat(b.Path()); err != nil {
			t.Fatal("failed target did not survive on disk", err)
		}
	})
	for _, retained := range []bool{false, true} {
		name := "empty"
		if retained {
			name = "unavailable_survivor"
		}
		t.Run(name, func(t *testing.T) {
			v := newTestViewer(t)
			a := uitest.TempJPEGURI(t, "a.jpg", 4, 5, color.White)
			u := storage.NewFileURI(filepath.Join(t.TempDir(), "u.heic"))
			dropAndWait(t, v, a)
			v.display.WaitPreloads()
			order := []collectionSource{{a, false}, {a, true}}
			if retained {
				order = append(order, collectionSource{u, true})
			}
			v.state.Replace(collectionInput{source: []fyne.URI{a}, display: []fyne.URI{a}, retained: order, favorite: "favorite"})
			before := v.state.Observe()
			uitest.StubTrashMove(t, os.Remove)
			collectionTrash(t, v, a)
			after := v.state.Observe()
			wantFavorite := ""
			var want []fyne.URI
			if retained {
				wantFavorite, want = "favorite", []fyne.URI{u}
			}
			if after.Generation() != before.Generation()+1 || after.Count() != 0 || after.Favorite() != wantFavorite || !slices.Equal(after.Capture(collectionSourceOrder), want) {
				t.Fatalf("empty presentation changed committed retained facts: generation %d -> %d, favorite %q, capture %v", before.Generation(), after.Generation(), after.Favorite(), after.Capture(collectionSourceOrder))
			}
			if v.img.Image != nil || !v.dropzone.Visible() || !v.emptyStateArt.Visible() || v.grid.Visible() {
				t.Fatal("last browsable deletion did not present the existing empty/error surface")
			}
		})
	}
	t.Run("stale_delivery", func(t *testing.T) {
		v := newTestViewer(t)
		a, b, c := collectionRemovalSources(t)
		dropAndWait(t, v, a, b)
		v.display.WaitPreloads()
		queue := &deletionCompletionQueue{queued: make(chan struct{}, 1)}
		v.deletion.SetUIQueue(queue)
		uitest.StubTrashMove(t, os.Remove)
		v.deletion.RequestFiles([]deletion.Target{{URI: a}})
		v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
		v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
		select {
		case <-queue.queued:
		case <-time.After(testTimeout):
			t.Fatal("Trash completion was not queued")
		}
		v.state.Replace(collectionInput{source: []fyne.URI{c}, display: []fyne.URI{c}, retained: []collectionSource{{a, true}, {c, false}}, favorite: "replacement"})
		before := v.state.Observe()
		v.deletion.Settle()
		waitUntilLoaded(t, v)
		after := v.state.Observe()
		if after.Generation() != before.Generation()+1 || after.Favorite() != "replacement" || !slices.Equal(after.Capture(collectionSourceOrder), []fyne.URI{c}) {
			t.Fatal("stale confirmation did not reconcile the committed URI against current unavailable membership")
		}
	})
	t.Run("symlink_target", func(t *testing.T) {
		v := newTestViewer(t)
		destination := uitest.TempJPEGURI(t, "destination.jpg", 4, 5, color.White)
		path := filepath.Join(t.TempDir(), "link.jpg")
		if err := os.Symlink(destination.Path(), path); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		link := storage.NewFileURI(path)
		v.OpenFavorite("favorite", []fyne.URI{link, destination, link})
		waitForScan(t, v)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		v.display.WaitPreloads()
		before := v.Generation()
		uitest.StubTrashMove(t, os.Remove)
		collectionTrash(t, v, link)
		waitUntilLoaded(t, v)
		if v.Generation() != before+1 || !slices.Equal(v.state.Observe().SourceFiles(), []fyne.URI{destination}) {
			t.Fatal("Trash confused symlink identity with its surviving destination")
		}
		if _, err := os.Stat(destination.Path()); err != nil {
			t.Fatal("symlink destination did not survive", err)
		}
	})
}

func collectionRemovalSources(t *testing.T) (fyne.URI, fyne.URI, fyne.URI) {
	t.Helper()
	files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg", "c.jpg")
	return files[0], files[1], files[2]
}

func collectionTrash(t *testing.T, v *viewer, uris ...fyne.URI) {
	t.Helper()
	targets := make([]deletion.Target, len(uris))
	for i, uri := range uris {
		targets[i] = deletion.Target{URI: uri}
	}
	v.deletion.RequestFiles(targets)
	if !v.deletion.Visible() {
		t.Fatal("Trash confirmation did not open")
	}
	v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	v.deletion.Settle()
}
