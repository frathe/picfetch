package ui

import (
	"context"
	"image/color"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	fynetest "fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/similarity"
	explorerui "github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestCollectionModel(t *testing.T) {
	t.Run("occurrences", collectionRemovalOccurrences)
	t.Run("snapshots", func(t *testing.T) {
		a, b := storage.NewFileURI("/images/a.jpg"), storage.NewFileURI("/images/b.jpg")
		u := storage.NewFileURI("/images/unavailable.heic")
		source := []fyne.URI{b, a, a}
		display := []fyne.URI{a, a, b}
		retained := []collectionSource{{b, false}, {u, true}, {a, false}, {a, false}}
		state := newAppState(0, false)
		state.Replace(collectionInput{source: source, display: display, retained: retained, favorite: "snapshot-favorite"})
		state.Select(1)
		before := state.Observe()
		bookmark, ok := before.Bookmark(1)
		if !ok || bookmark.occurrence != (fileidentity.Occurrence{Path: a.Path(), Ordinal: 1}) {
			t.Fatalf("second occurrence bookmark = %+v, %v", bookmark, ok)
		}
		source[0], display[0], retained[1].uri = u, u, a
		state.Select(2)
		state.Reorder([]fyne.URI{b, a, a})
		after := state.Observe()
		if got := before.SourceFiles(); !slices.EqualFunc(got, []fyne.URI{b, a, a}, sameURI) {
			t.Fatalf("source input changed retained snapshot: %v", got)
		}
		if before.FileAt(0) != a || before.FileSet().KeyAt(0) != a.String() || before.Resolve(bookmark) != 1 {
			t.Fatal("old order, URI keys and occurrence lookup must stay together")
		}
		if got, index, present := before.Current(); got != a || index != 1 || !present {
			t.Fatalf("old requested occurrence = %v, %d, %v", got, index, present)
		}
		if after.Retained()[1].uri != u || before.Retained()[1].uri != u {
			t.Fatal("caller mutation changed retained unavailable membership")
		}
		if before.Favorite() != "snapshot-favorite" || after.Favorite() != before.Favorite() {
			t.Fatal("snapshot association did not stay with its collection")
		}
		if after.Generation() <= before.Generation() || after.FileSet().Generation() != after.Generation() {
			t.Fatal("published generation does not describe the complete order")
		}
		copySource, copyRetained := before.SourceFiles(), before.Retained()
		copySource[0], copyRetained[1].uri = u, a
		if before.SourceFiles()[0] != b || before.Retained()[1].uri != u {
			t.Fatal("observation exposed writable membership")
		}
	})
	t.Run("operations", func(t *testing.T) {
		t.Run("unavailability", collectionModelUnavailability)
		t.Run("reorder", func(t *testing.T) {
			a, b := storage.NewFileURI("/images/a.jpg"), storage.NewFileURI("/images/b.jpg")
			u := storage.NewFileURI("/images/u.heic")
			state := newAppState(0, false)
			state.Replace(collectionInput{source: []fyne.URI{b, a, a}, display: []fyne.URI{a, a, b}, retained: []collectionSource{{b, false}, {u, true}, {a, false}, {u, true}, {a, false}}, favorite: "favorite"})
			state.Select(1)
			before := state.Observe()
			bookmark, _ := before.Bookmark(1)
			ordered := []fyne.URI{b, a, a}
			state.Reorder(ordered)
			after := state.Observe()
			ordered[0] = u
			if uri, index, ok := after.Current(); !ok || uri != a || index != 2 {
				t.Fatalf("reorder requested occurrence = %v/%d/%v; want second a at 2", uri, index, ok)
			}
			if after.Generation() != before.Generation()+1 || after.Favorite() != "favorite" || !slices.Equal(after.SourceFiles(), before.SourceFiles()) || !slices.Equal(after.Retained(), before.Retained()) {
				t.Fatal("reorder changed membership, association or publication count")
			}
			if !slices.Equal(after.Capture(collectionDisplayOrder), []fyne.URI{b, u, a, u, a}) || after.FileAt(0) != b {
				t.Fatal("reorder lost anchored unavailable gaps or aliased caller order")
			}
			if before.Resolve(bookmark) != 1 || after.Resolve(bookmark) != -1 || before.FileAt(0) != a || before.index != 1 {
				t.Fatal("reorder mutated the old snapshot or accepted an unremapped bookmark")
			}
		})
		t.Run("merge", func(t *testing.T) {
			a := storage.NewFileURI("/images/a.jpg")
			u := storage.NewFileURI("/images/u.heic")
			state := newAppState(0, false)
			state.Replace(collectionInput{source: []fyne.URI{a}, display: []fyne.URI{a}, retained: []collectionSource{{a, false}, {u, true}}, favorite: "favorite-A"})
			change := state.Merge(collectionInput{source: []fyne.URI{a}, display: []fyne.URI{a, a}, retained: []collectionSource{{a, false}, {u, true}}, index: 1, favorite: "favorite-B"})
			if got := change.after.SourceFiles(); !slices.Equal(got, []fyne.URI{a, a}) {
				t.Fatalf("merge lost a repeated source occurrence: %v", got)
			}
			if got := change.after.Retained(); !slices.Equal(got, []collectionSource{{a, false}, {u, true}, {a, false}, {u, true}}) {
				t.Fatalf("merge lost retained gap order: %v", got)
			}
			if uri, index, ok := change.after.Current(); !ok || uri != a || index != 1 || change.after.Favorite() != "favorite-B" || change.after.Generation() != change.before.Generation()+1 {
				t.Fatal("merge did not publish the requested repeated occurrence and association together")
			}
			if change.before.Count() != 1 || len(change.before.Retained()) != 2 || change.before.Favorite() != "favorite-A" {
				t.Fatal("merge mutated its old observation")
			}
			retained := state.Merge(collectionInput{retained: []collectionSource{{u, true}}, favorite: "favorite-C"})
			if uri, index, ok := retained.after.Current(); !ok || uri != a || index != 1 || retained.after.Count() != 2 || len(retained.after.Retained()) != 5 || retained.after.Favorite() != "favorite-C" {
				t.Fatal("retained-only merge lost existing selection or incoming association")
			}
			empty := state.Merge(collectionInput{favorite: "must-not-bind"})
			if empty.after.Generation() != retained.after.Generation() || empty.after.Favorite() != "favorite-C" || len(empty.after.Retained()) != 5 || empty.after.Count() != 2 {
				t.Fatal("merge with no admitted entries changed committed facts")
			}
		})
		t.Run("replacement", func(t *testing.T) {
			a, b := storage.NewFileURI("/images/a.jpg"), storage.NewFileURI("/images/b.jpg")
			u := storage.NewFileURI("/images/u.heic")
			state := newAppState(0, false)
			change := state.Replace(collectionInput{
				source: []fyne.URI{b, a}, display: []fyne.URI{a, b},
				retained: []collectionSource{{b, false}, {u, true}, {a, false}},
				index:    1, favorite: "favorite-A",
			})
			if change.before.Count() != 0 || change.after.Generation() != change.before.Generation()+1 {
				t.Fatal("replacement did not publish exactly one complete change")
			}
			if uri, index, ok := change.after.Current(); uri != b || index != 1 || !ok || change.after.Favorite() != "favorite-A" {
				t.Fatal("replacement did not commit association and requested order together")
			}
			state.Replace(collectionInput{retained: []collectionSource{{u, true}}, favorite: "favorite-B"})
			if state.Observe().Favorite() != "favorite-B" || change.after.Favorite() != "favorite-A" || change.after.Count() != 2 {
				t.Fatal("replacement lost unavailable association or mutated an old snapshot")
			}
			state.Replace(collectionInput{favorite: "empty-favorite"})
			if state.Observe().Favorite() != "" {
				t.Fatal("replacement with no retained members kept an association")
			}
		})
		t.Run("clear", func(t *testing.T) {
			u := storage.NewFileURI("/images/u.heic")
			state := newAppState(0, false)
			state.Replace(collectionInput{retained: []collectionSource{{u, true}}, favorite: "favorite"})
			change := state.Clear()
			if change.after.Count() != 0 || len(change.after.Retained()) != 0 || len(change.after.SourceFiles()) != 0 || change.after.Favorite() != "" {
				t.Fatal("clear left collection facts behind")
			}
			if change.before.Favorite() != "favorite" || len(change.before.Retained()) != 1 || change.after.Generation() != change.before.Generation()+1 {
				t.Fatal("clear mutated its old snapshot or published multiple generations")
			}
		})
		t.Run("selection", func(t *testing.T) {
			a, b := storage.NewFileURI("/images/a.jpg"), storage.NewFileURI("/images/b.jpg")
			state := newAppState(0, false)
			if state.Select(0) {
				t.Fatal("empty collection admitted selection")
			}
			state.Replace(collectionInput{source: []fyne.URI{a, b, a}, display: []fyne.URI{a, b, a}})
			before := state.Observe()
			bookmark, _ := before.Bookmark(2)
			if !state.Select(-1) {
				t.Fatal("nonempty collection refused navigation")
			}
			chosen := state.Observe()
			if uri, index, ok := chosen.Current(); uri != a || index != 2 || !ok {
				t.Fatalf("wrapped requested occurrence = %v, %d, %v", uri, index, ok)
			}
			if chosen.Generation() != before.Generation() || chosen.Resolve(bookmark) != 2 {
				t.Fatal("navigation invalidated collection identity")
			}
			if _, index, _ := before.Current(); index != 0 {
				t.Fatal("navigation changed a retained observation")
			}
			state.Replace(collectionInput{source: []fyne.URI{a, b, a}, display: []fyne.URI{a, b, a}})
			replaced := state.Observe()
			if replaced.Resolve(bookmark) != -1 {
				t.Fatal("old bookmark resolved into unrelated replacement")
			}
			if _, index, ok := replaced.Current(); index != 0 || !ok {
				t.Fatalf("replacement selected %d, %v; want first occurrence", index, ok)
			}
		})
	})
}

func TestCollectionCompatibility(t *testing.T) {
	t.Run("navigation_snapshots", func(t *testing.T) {
		v := newTestViewer(t)
		files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg", "c.jpg")
		dropAndWait(t, v, files...)
		v.SetMergeMode(true)
		dropAndWait(t, v, files[0])
		v.ShowImage(2)
		waitUntilLoaded(t, v)
		before := v.state.Observe()
		v.display.WaitPreloads()
		v.imgCache.Purge()
		v.ShowImage(1)
		chosen := v.state.Observe()
		bookmark, _ := chosen.Bookmark(1)
		if uri, index, ok := chosen.Current(); uri != files[0] || index != 1 || !ok {
			t.Fatalf("requested collection observation = %v, %d, %v", uri, index, ok)
		}
		if bookmark.occurrence.Ordinal != 1 || chosen.Generation() != before.Generation() {
			t.Fatal("navigation lost the repeated occurrence or changed collection generation")
		}
		if uri, index, ok := v.CurrentFile(); uri != files[0] || index != 1 || !ok {
			t.Fatalf("Host observation disagrees: %v, %d, %v", uri, index, ok)
		}
		if displayed, ok := v.displayedFile(); !ok || displayed.String() != files[1].String() {
			t.Fatal("requested observation was confused with outgoing displayed pixels")
		}
		waitUntilLoaded(t, v)
		v.StepImage(1)
		waitUntilLoaded(t, v)
		if uri, index, ok := v.CurrentFile(); uri.String() != files[1].String() || index != 2 || !ok {
			t.Fatalf("ordinary next after repeated occurrence = %v, %d, %v", uri, index, ok)
		}
	})
}

func TestCollectionMerge(t *testing.T) {
	t.Run("per_input_limit", func(t *testing.T) {
		v := newTestViewer(t)
		files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg")
		v.SetMaxScan(2)
		dropAndWait(t, v, files[:2]...)
		v.SetMergeMode(true)
		dropAndWait(t, v, files[2:]...)
		if got := v.state.Observe().SourceFiles(); !slices.EqualFunc(got, files, sameURI) {
			t.Fatalf("per-input admission became an aggregate merge cap: %v", got)
		}
	})
	t.Run("unavailable_existing", func(t *testing.T) {
		v := newTestViewer(t)
		v.startHEICCheck(false)
		v.settleHEIC()
		unavailable := storage.NewFileURI(uitest.WriteTempFile(t, "saved.heic", []byte("unavailable")))
		v.OpenFavorite("favorite-A", []fyne.URI{unavailable})
		waitForScan(t, v)
		fynetest.Tap(explorerDialogButton(t, v, "Close"))
		before := v.state.Observe()
		v.SetMergeMode(true)
		files := uitest.TempDirJPEGURIs(t, "added.jpg", "uninvited-sibling.jpg")
		dropAndWait(t, v, files[0])
		after := v.state.Observe()
		if after.Count() != 1 || after.FileAt(0).String() != files[0].String() {
			t.Fatal("merge into unavailable membership expanded replacement-style siblings")
		}
		if retained := after.Retained(); len(retained) != 2 || retained[0].uri != unavailable || !retained[0].unavailable || retained[1].uri.String() != files[0].String() {
			t.Fatal("merge did not preserve unavailable membership before the addition")
		}
		if before.Count() != 0 || len(before.Retained()) != 1 || before.Favorite() != "favorite-A" || after.Favorite() != "" || after.Generation() != before.Generation()+1 {
			t.Fatal("merge changed its old snapshot or did not atomically apply ordinary association")
		}
	})
}

func TestCollectionUnavailable(t *testing.T) {
	t.Run("runtime_loss", collectionRuntimeLoss)
	t.Run("favorite_open", func(t *testing.T) {
		v := newTestViewer(t)
		v.startHEICCheck(false)
		v.settleHEIC()
		uri := storage.NewFileURI(uitest.WriteTempFile(t, "saved.heic", []byte("unavailable")))
		dir := t.TempDir()
		v.OpenFavorite(dir, []fyne.URI{uri})
		waitForScan(t, v)
		if v.state.Observe().Favorite() != dir {
			t.Fatal("unavailable-only Favorite lost its candidate association")
		}
		collection := v.state.Observe()
		if collection.Count() != 0 || len(collection.Retained()) != 1 || !collection.Retained()[0].unavailable {
			t.Fatal("unavailable-only Favorite lost its retained membership")
		}
		if _, _, chosen := collection.Current(); chosen {
			t.Fatal("unavailable collection has a browsable selection")
		}
		mounted := false
		explorerWalk(v.win.Content(), func(object fyne.CanvasObject) { mounted = mounted || object == v.emptyStateArt })
		if !mounted || !v.dropzone.Visible() || !v.emptyStateArt.Visible() || v.win.Canvas().Overlays().Top() == nil {
			t.Fatal("unavailable collection lost its empty-state guidance")
		}
	})
}

func TestCollectionAdmission(t *testing.T) {
	t.Run("merge", func(t *testing.T) {
		for _, existing := range []string{"browsable", "unavailable"} {
			t.Run(existing, func(t *testing.T) {
				v := newTestViewer(t)
				v.startHEICCheck(false)
				v.settleHEIC()
				files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg")
				if existing == "unavailable" {
					files = []fyne.URI{storage.NewFileURI(uitest.WriteTempFile(t, "a.heic", []byte("unavailable")))}
				}
				v.OpenFavorite("favorite-A", files)
				waitForScan(t, v)
				if existing == "browsable" {
					waitForSort(t, v)
					waitUntilLoaded(t, v)
					v.ShowImage(1)
					waitUntilLoaded(t, v)
				} else {
					fynetest.Tap(explorerDialogButton(t, v, "Close"))
				}
				before := v.state.Observe()
				v.SetMergeMode(true)
				unsupported := storage.NewFileURI(uitest.WriteTempFile(t, "notes.txt", []byte("unsupported")))
				v.OpenFavorite("must-not-bind", []fyne.URI{unsupported})
				waitForScan(t, v)
				after := v.state.Observe()
				if after.Generation() != before.Generation() || after.Favorite() != before.Favorite() || after.index != before.index || !slices.Equal(after.Retained(), before.Retained()) || !slices.Equal(after.SourceFiles(), before.SourceFiles()) {
					t.Fatal("merge with no admitted entries changed committed facts")
				}
				if existing == "unavailable" && (!v.dropzone.Visible() || !v.emptyStateArt.Visible()) {
					t.Fatal("no-op merge left an unavailable collection without guidance")
				}
			})
		}
	})
	t.Run("empty_input", func(t *testing.T) {
		v := newTestViewer(t)
		files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg")
		v.OpenFavorite("favorite-A", files)
		waitForScan(t, v)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		before := v.state.Observe()
		token := v.scanOp.lifecycle.begin()
		defer token.cancelContext()
		v.OpenFavorite("favorite-B", nil)
		if !token.current() || v.state.Observe().Generation() != before.Generation() || v.state.Observe().Favorite() != "favorite-A" {
			t.Fatal("literal empty input changed committed facts or canceled work")
		}
		unsupported := storage.NewFileURI(uitest.WriteTempFile(t, "notes.txt", []byte("not an image")))
		v.OpenFavorite("favorite-B", []fyne.URI{unsupported})
		waitForScan(t, v)
		after := v.state.Observe()
		if after.Count() != 0 || len(after.Retained()) != 0 || after.Favorite() != "" || !v.emptyStateArt.Visible() {
			t.Fatal("unsuccessful nonempty replacement did not clear collection and association")
		}
	})
}

func TestCollectionFavoriteAssociation(t *testing.T) {
	t.Run("merge", func(t *testing.T) {
		t.Run("containment", func(t *testing.T) {
			for _, kind := range []string{"mixed", "repeated", "subset"} {
				t.Run(kind, func(t *testing.T) {
					v := newTestViewer(t)
					files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg", "outside.jpg")
					dir := t.TempDir()
					if err := favstore.Save(dir, "Saved", files[:2]); err != nil {
						t.Fatal(err)
					}
					favorite := favstore.Dir(dir, "Saved")
					store, _, err := favstore.OpenCohorts(context.Background(), favorite)
					if err != nil {
						t.Fatal(err)
					}
					if err := store.Save(context.Background(), favstore.CohortState{Groups: []favstore.Cohort{{Name: "Saved group", Paths: []string{files[0].Path()}}}}); err != nil {
						t.Fatal(err)
					}
					initial := files[:1]
					if kind == "mixed" {
						initial = files[2:]
					}
					v.OpenFavorite("initial-favorite", initial)
					waitForScan(t, v)
					waitForSort(t, v)
					waitUntilLoaded(t, v)
					v.SetMergeMode(true)
					incoming := files[:2]
					if kind == "subset" {
						incoming = files[:1]
					}
					v.OpenFavorite(favorite, incoming)
					waitForScan(t, v)
					waitForSort(t, v)
					waitUntilLoaded(t, v)
					preview := uitest.EncodeJPEG(t, 16, 16, color.White)
					configureExplorer(v, func(options *explorerui.Options) {
						options.Settings.CacheFavorites = false
						options.Analyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
							var items []similarity.Item
							for _, path := range paths {
								items = append(items, similarity.Item{Path: path, Cohort: "unassigned", Preview: preview})
							}
							emit(similarity.Event{Complete: true, Total: len(paths), Successful: len(paths), Items: items})
							return nil
						}
					})
					v.showExplorer()
					v.settleExplorer()
					groups := v.explorer.Surface().Cohorts().Groups
					if got := len(groups) > 0; got != (kind != "mixed") {
						t.Fatalf("Favorite cohort ownership for %s: %+v", kind, groups)
					}
					if v.state.Observe().Favorite() != favorite {
						t.Fatal("containment changed the committed candidate association")
					}
				})
			}
		})
		for _, existing := range []string{"browsable", "unavailable"} {
			for _, incoming := range []string{"browsable", "unavailable"} {
				for _, kind := range []string{"ordinary", "favorite"} {
					t.Run(existing+"/"+incoming+"/"+kind, func(t *testing.T) {
						v := newTestViewer(t)
						v.startHEICCheck(false)
						v.settleHEIC()
						files := uitest.TempDirJPEGURIs(t, "old.jpg", "added.jpg")
						old, added := files[0], files[1]
						if existing == "unavailable" {
							old = storage.NewFileURI(uitest.WriteTempFile(t, "old.heic", []byte("unavailable")))
						}
						if incoming == "unavailable" {
							added = storage.NewFileURI(uitest.WriteTempFile(t, "added.heic", []byte("unavailable")))
						}
						v.OpenFavorite("favorite-A", []fyne.URI{old})
						waitForScan(t, v)
						if existing == "browsable" {
							waitForSort(t, v)
							waitUntilLoaded(t, v)
						} else {
							fynetest.Tap(explorerDialogButton(t, v, "Close"))
						}
						before := v.state.Observe()
						v.SetMergeMode(true)
						want := ""
						if kind == "favorite" {
							want = "favorite-B"
							v.OpenFavorite(want, []fyne.URI{added})
						} else {
							v.OpenFiles([]fyne.URI{added})
						}
						waitForScan(t, v)
						if incoming == "browsable" {
							waitForSort(t, v)
							waitUntilLoaded(t, v)
						}
						after := v.state.Observe()
						if after.Favorite() != want || after.Generation() != before.Generation()+1 {
							t.Fatal("admitted merge did not atomically apply incoming association")
						}
						retained := after.Retained()
						if len(retained) != 2 || retained[0].uri.String() != old.String() || retained[1].uri.String() != added.String() {
							t.Fatalf("merge changed retained input order: %v", retained)
						}
						if incoming == "browsable" {
							if uri, _, ok := after.Current(); !ok || uri.String() != added.String() {
								t.Fatal("merge did not choose its first browsable addition")
							}
						} else if uri, _, ok := after.Current(); ok != (existing == "browsable") || ok && uri.String() != old.String() {
							t.Fatal("unavailable-only addition changed the chosen image")
						}
					})
				}
			}
		}
	})
	t.Run("replacement", func(t *testing.T) {
		for _, favorite := range []bool{false, true} {
			t.Run(map[bool]string{false: "ordinary", true: "favorite"}[favorite], func(t *testing.T) {
				v := newTestViewer(t)
				files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg")
				v.OpenFavorite("favorite-A", files)
				waitForScan(t, v)
				waitForSort(t, v)
				waitUntilLoaded(t, v)
				before := v.state.Observe()
				want := ""
				if favorite {
					want = "favorite-B"
					v.OpenFavorite(want, files)
				} else {
					v.OpenFiles(files)
				}
				waitForScan(t, v)
				waitForSort(t, v)
				waitUntilLoaded(t, v)
				if v.state.Observe().Favorite() != want || before.Favorite() != "favorite-A" {
					t.Fatal("incoming replacement did not select its candidate association")
				}
				if v.state.Observe().Generation() != before.Generation()+1 {
					t.Fatal("replacement published incomplete intermediate facts")
				}
			})
		}
	})
}

func TestCollectionReconciliation(t *testing.T) {
	t.Run("validation_removal", collectionValidationRemoval)
	t.Run("load_failure", collectionLoadFailureReconciliation)
	t.Run("reorder", collectionReorderReconciliation)
	t.Run("removal", collectionRemovalReconciliation)
	for _, kind := range []string{"replacement", "merge"} {
		t.Run(kind, func(t *testing.T) {
			v, publish := streamingSearch(t)
			publish(similarity.SearchFinal, 2, 1)
			before := v.state.Observe()
			v.SetMergeMode(kind == "merge")
			files := uitest.TempDirJPEGURIs(t, "replacement-a.jpg", "replacement-b.jpg")
			v.OpenFavorite("replacement-favorite", files)
			waitForScan(t, v)
			waitForSort(t, v)
			waitUntilLoaded(t, v)
			v.visualsearch.Settle()
			after := v.state.Observe()
			wantCount := 2
			if kind == "merge" {
				wantCount += before.Count()
			}
			if after.Generation() != before.Generation()+1 || after.Count() != wantCount || after.Favorite() != "replacement-favorite" {
				t.Fatal("replacement did not publish complete collection facts once")
			}
			if v.searchActive() || v.grid.Visible() || v.browsing.current().binding.kind != browsingCollection {
				t.Fatal("retired ranking restored its obsolete browsing surface")
			}
			mounted := false
			explorerWalk(v.win.Content(), func(object fyne.CanvasObject) { mounted = mounted || object == v.img })
			if uri, ok := v.DisplayedFile(); !ok || uri.String() != files[0].String() || !mounted {
				t.Fatal("replacement did not hand off its chosen image to the mounted display")
			}
		})
	}
}
