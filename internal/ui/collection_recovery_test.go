package ui

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	fynetest "fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/uitest"
)

func collectionModelUnavailability(t *testing.T) {
	a, b, u := storage.NewFileURI("/a.heic"), storage.NewFileURI("/b.jpg"), storage.NewFileURI("/u.heic")
	state := newAppState(0, false)
	retained := []collectionSource{{a, false}, {u, true}, {b, false}, {a, false}, {u, true}, {a, false}}
	state.Replace(collectionInput{source: []fyne.URI{a, b, a, a}, display: []fyne.URI{a, a, a, b}, retained: retained, index: 1, favorite: "favorite"})
	change := state.MarkUnavailable(1)
	want := slices.Clone(retained)
	want[3].unavailable = true
	if !slices.Equal(change.after.Retained(), want) || !slices.Equal(change.after.SourceFiles(), []fyne.URI{a, b, a}) || !slices.Equal(change.after.DisplayFiles(), []fyne.URI{a, a, b}) || change.after.Favorite() != "favorite" || change.after.Generation() != change.before.Generation()+1 {
		t.Fatal("decoder loss did not retain the exact middle occurrence in one publication")
	}
	if got, ok := change.survivors[fileidentity.Occurrence{Path: a.Path(), Ordinal: 2}]; !ok || got.Ordinal != 1 || got.Path != a.Path() {
		t.Fatal("decoder loss did not return the surviving later occurrence mapping")
	}
	if _, ok := change.survivors[fileidentity.Occurrence{Path: a.Path(), Ordinal: 1}]; ok || !slices.Equal(change.removed, []fyne.URI{a}) {
		t.Fatal("unavailable occurrence survived as browsable or lost its cache effect")
	}
	if !slices.Equal(change.before.Retained(), retained) || !slices.Equal(change.after.Capture(collectionSourceOrder), []fyne.URI{a, u, b, a, u, a}) {
		t.Fatal("decoder loss mutated the old observation or lost persistence placement")
	}
	state.MarkUnavailable(1)
	state.Remove([]int{1})
	last := state.MarkUnavailable(0)
	if _, _, ok := last.after.Current(); ok || last.after.Count() != 0 || len(last.after.SourceFiles()) != 0 || last.after.Favorite() != "favorite" || len(last.after.Retained()) != 5 {
		t.Fatal("final decoder loss erased retained membership or invented selection")
	}
	for _, index := range []int{-1, 0, 9} {
		if invalid := state.MarkUnavailable(index); invalid.after.Generation() != last.after.Generation() {
			t.Fatal("invalid decoder-loss index published a change")
		}
	}
}

func collectionRuntimeLoss(t *testing.T) {
	t.Run("retained_search_origin", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		files := v.state.Observe().DisplayFiles()
		v.OpenSimilarityCohort([]string{files[1].Path(), files[2].Path()})
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
		waitUntilLoaded(t, v)
		publish := streamingSearchFrom(t, v)
		publish(similarity.SearchFinal, 0, 2)
		v.grid.Settle()
		v.display.WaitPreloads()
		v.state.Select(0)
		generation := v.Generation()
		if retry := v.imageLoadFailed(files[0], heic.ErrUnavailable); retry != nil {
			t.Fatal("unavailable guide admitted a scoped or ordinary successor")
		}
		if v.searchActive() || !v.browsing.has(browsingExplorer) || v.img.Image != nil || v.Generation() != generation+1 || !v.state.Observe().Retained()[0].unavailable || !slices.Equal(v.state.Observe().Capture(collectionSourceOrder), files) {
			t.Fatal("decoder-loss guide lost retained origin or collection membership")
		}
		_ = explorerDialogButton(t, v, "HEIC installation guide")
	})
	for _, sibling := range []bool{false, true} {
		t.Run(fmt.Sprint(sibling), func(t *testing.T) {
			v := newTestViewer(t)
			var restored atomic.Bool
			v.configureHEIC(testHEICBackend{
				check: func(_ context.Context) error { return nil },
				read: func(_ context.Context, _ []byte, request heic.Request) (heic.Result, error) {
					if !restored.Load() {
						return heic.Result{}, heic.ErrUnavailable
					}
					result := heic.Result{Width: 2, Height: 1}
					if request.Pixels {
						result.Stride = 8
						result.Pixels = []byte{255, 0, 0, 255, 0, 255, 0, 255}
					}
					return result, nil
				},
			})
			v.heic.ui = &uitest.UIQueue{}
			v.startHEICCheck(false)
			v.settleHEIC()
			data, err := os.ReadFile("../imaging/testdata/test_exif.heic")
			if err != nil {
				t.Fatal(err)
			}
			source := storage.NewFileURI(uitest.WriteTempFile(t, "a.heic", data))
			files := []fyne.URI{source}
			if sibling {
				files = append(files, uitest.TempDirJPEGURIs(t, "b.jpg")...)
			}
			favorite := t.TempDir()
			v.OpenFavorite(favorite, files)
			waitForScan(t, v)
			waitForSort(t, v)
			before := v.state.Observe()
			observed, incoherent := false, false
			v.grid.SetOnResultChanged(func() {
				observed = true
				current := v.state.Observe()
				incoherent = incoherent || current.Favorite() != favorite || len(current.Retained()) != len(files) || !current.Retained()[0].unavailable
			})
			waitUntilLoaded(t, v)
			v.settleHEIC()
			after := v.state.Observe()
			if after.Favorite() != favorite || !slices.Equal(after.Capture(collectionSourceOrder), files) || after.Generation() != before.Generation()+1 || sibling && !observed || incoherent || !after.Retained()[0].unavailable {
				t.Fatalf("runtime loss did not atomically retain facts: favorite=%q capture=%v generation=%d->%d observed=%v incoherent=%v", after.Favorite(), namesOfURIs(after.Capture(collectionSourceOrder)), before.Generation(), after.Generation(), observed, incoherent)
			}
			if _, _, chosen := after.Current(); chosen != sibling || after.Count() != len(files)-1 || v.img.Image != nil || !v.dropzone.Visible() || !v.emptyStateArt.Visible() {
				t.Fatal("runtime loss invented a chosen image, auto-advanced or lost empty guidance")
			}
			_ = explorerDialogButton(t, v, "HEIC installation guide")
			fynetest.Tap(explorerDialogButton(t, v, "Close"))
			v.grid.SetOnResultChanged(nil)
			v.favorites.SetDir(t.TempDir())
			if sibling {
				v.favorites.AddCurrentList()
				saveBrowsingFavorite(t, v, files)
			} else if captured := (favoriteListHost{v}).CurrentFiles(); !slices.Equal(captured, files) {
				// MA-028 keeps Add Current List disabled without a browsable
				// image. Its capture still retains every unavailable member.
				t.Fatal("Favorite capture discarded the unavailable-only collection")
			}
			restored.Store(true)
			v.startHEICCheck(false)
			v.settleHEIC()
			if current := v.state.Observe(); current.Generation() != after.Generation() || current.Count() != after.Count() || current.Favorite() != favorite || !current.Retained()[0].unavailable {
				t.Fatal("support check automatically readmitted a committed unavailable member")
			}
			v.OpenFavorite(favorite, files)
			waitForScan(t, v)
			waitForSort(t, v)
			waitUntilLoaded(t, v)
			if current := v.state.Observe(); current.Count() != len(files) || current.Favorite() != favorite || current.Retained()[0].unavailable || v.img.Image == nil {
				t.Fatal("explicit reopen did not readmit the restored decoder through normal loading")
			}
		})
	}
}

func collectionLoadFailureReconciliation(t *testing.T) {
	t.Run("scoped_retry", TestBrowsingLoadRecovery)
	t.Run("repeated_origin", func(t *testing.T) {
		v := newTestViewer(t)
		a, b, c := collectionRemovalSources(t)
		v.OpenFavorite("favorite", []fyne.URI{a, b, a, c, a})
		waitForScan(t, v)
		waitForSort(t, v)
		waitUntilLoaded(t, v)
		v.ShowImage(2)
		waitUntilLoaded(t, v)
		publish := streamingSearchFrom(t, v)
		publish(similarity.SearchFinal, 0, 4)
		v.grid.Settle()
		v.display.WaitPreloads()
		var reads atomic.Int32
		failed := uitest.ReaderURI(a, func() (io.ReadCloser, error) {
			reads.Add(1)
			return nil, errors.New("failed selected occurrence")
		})
		before := v.state.Observe()
		// Same-path/different-URI first occurrence fails; the retained last
		// origin must beat the ordinary successor and keep its exact ordinal.
		source, display, retained := before.SourceFiles(), before.DisplayFiles(), before.Retained()
		source[0], display[0], retained[0].uri = failed, failed, failed
		v.state.Replace(collectionInput{source: source, display: display, retained: retained, index: 2, favorite: before.Favorite()})
		v.browsing.rebind(v.Generation())
		revision, generation := v.display.RequestRevision(), v.Generation()
		v.ShowImage(0)
		load := v.display.LoadDone()
		waitUntilLoaded(t, v)
		waitHandle(t, "single failure retry chain", load)
		if reads.Load() == 0 || v.searchActive() || v.grid.Visible() || v.currentImageOccurrence().Ordinal != 1 || v.state.Observe().index != 1 || v.display.RequestRevision() != revision+1 || v.Generation() != generation+1 || v.state.Observe().Favorite() != "favorite" {
			t.Fatalf("load failure: reads=%d search=%v grid=%v occurrence=%v index=%d revision=%d->%d generation=%d->%d favorite=%q", reads.Load(), v.searchActive(), v.grid.Visible(), v.currentImageOccurrence(), v.state.Observe().index, revision, v.display.RequestRevision(), generation, v.Generation(), v.state.Observe().Favorite())
		}
	})
}

func TestCollectionLifecycle(t *testing.T) {
	t.Run("queued_chooser", collectionQueuedChooserLifecycle)
	t.Run("close_facts", collectionCloseFacts)
	t.Run("preparation", collectionPreparationLifecycle)
	t.Run("committed_writes", collectionCommittedWriteLifecycle)
	t.Run("load_recovery", func(t *testing.T) {
		for _, action := range []string{"replacement", "close_reopen", "stop"} {
			for _, delivery := range []string{"held", "queued"} {
				for _, unavailable := range []bool{false, true} {
					t.Run(action+"/"+delivery+"/"+fmt.Sprint(unavailable), func(t *testing.T) {
						v := newTestViewer(t)
						files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg")
						u := storage.NewFileURI(uitest.WriteTempFile(t, "u.heic", []byte("unavailable")))
						entered, release := make(chan struct{}), make(chan struct{})
						var once sync.Once
						unblock := func() { once.Do(func() { close(release) }) }
						defer unblock()
						var reads atomic.Int32
						failed := uitest.ReaderURI(files[0], func() (io.ReadCloser, error) {
							if reads.Add(1) == 1 {
								close(entered)
								if delivery == "held" {
									<-release
								}
							}
							if unavailable {
								return nil, heic.ErrUnavailable
							}
							return nil, errors.New("retired failure")
						})
						v.state.Replace(collectionInput{source: []fyne.URI{failed, files[1]}, display: []fyne.URI{failed, files[1]}, retained: []collectionSource{{failed, false}, {u, true}, {files[1], false}}, favorite: "old"})
						v.ShowImage(0)
						oldLoad := v.display.LoadDone()
						select {
						case <-entered:
						case <-time.After(testTimeout):
							t.Fatal("load did not reach controlled reader")
						}
						if delivery == "queued" {
							v.display.Wait()
						}
						if action == "stop" {
							lifecycle := v.app.Lifecycle().(interface{ OnStopped() func() })
							previous := lifecycle.OnStopped()
							registerShutdown(v.app, v)
							shutdown := lifecycle.OnStopped()
							v.app.Lifecycle().SetOnStopped(previous)
							shutdown()
						} else {
							if action == "close_reopen" {
								v.clearToDropzone()
							}
							v.OpenFavorite("current", []fyne.URI{files[1], u})
							waitForScan(t, v)
							waitForSort(t, v)
						}
						waitHandle(t, "retired failed request", oldLoad)
						current := v.state.Observe()
						unblock()
						v.display.Settle()
						v.settleHEIC()
						after := v.state.Observe()
						if reads.Load() != 1 || after.Generation() != current.Generation() || after.Favorite() != current.Favorite() || !slices.Equal(after.Retained(), current.Retained()) || !slices.Equal(after.DisplayFiles(), current.DisplayFiles()) || v.win.Canvas().Overlays().Top() != nil {
							t.Fatal("obsolete failure mutated current retained facts, retried or opened guidance")
						}
						if action != "stop" && (after.Count() != 1 || after.Favorite() != "current" || len(after.Retained()) != 2 || v.img.Image == nil) {
							t.Fatal("current collection did not complete independently of retired failure")
						}
					})
				}
			}
		}
	})
}
