package ui

import (
	"context"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/session"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/uitest"
)

func collectionValidationRemoval(t *testing.T) {
	v := newTestViewer(t)
	a, b, c := collectionRemovalSources(t)
	u := storage.NewFileURI(uitest.WriteTempFile(t, "u.heic", []byte("unavailable")))
	v.OpenFavorite("favorite", []fyne.URI{a, u, b, a, c})
	waitForScan(t, v)
	waitForSort(t, v)
	waitUntilLoaded(t, v)
	v.ShowImage(2)
	waitUntilLoaded(t, v)
	publish := streamingSearchFrom(t, v)
	publish(similarity.SearchFinal, 0, 3)
	v.grid.Settle()
	v.display.WaitPreloads()
	before, revision := v.state.Observe(), v.display.RequestRevision()
	for _, uri := range []fyne.URI{a, c} {
		if err := os.Remove(uri.Path()); err != nil {
			t.Fatal(err)
		}
	}
	v.reconcileSearchOrigin()
	drainFileWork(t, v)
	waitUntilLoaded(t, v)
	after := v.state.Observe()
	if after.Generation() != before.Generation()+1 || after.Favorite() != "favorite" || !slices.Equal(after.Capture(collectionSourceOrder), []fyne.URI{u, b}) || before.Count() != 4 || len(before.Retained()) != 5 {
		t.Fatal("validation removals bypassed one-publication removal or lost retained gaps")
	}
	if uri, index, ok := after.Current(); !ok || uri != b || index != 0 || v.searchActive() || v.grid.Visible() || v.display.RequestRevision() != revision+1 {
		t.Fatal("validation did not reconcile the retained image origin before one display handoff")
	}
}

func collectionQueuedChooserLifecycle(t *testing.T) {
	for _, action := range []string{"close_reopen", "stop"} {
		t.Run(action, func(t *testing.T) {
			v := newTestViewer(t)
			files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg")
			v.OpenFavorite("original", []fyne.URI{files[0], files[0]})
			waitForScan(t, v)
			waitForSort(t, v)
			waitUntilLoaded(t, v)
			queue := &uitest.UIQueue{}
			v.chooserUI = queue
			uitest.StubChooser(t, []fyne.URI{files[1]}, nil)
			v.openFileDialog()
			waitFor(t, "queued native chooser", &v.chooser)
			if queue.Len() != 1 {
				t.Fatal("chooser did not queue its result")
			}
			if action == "stop" {
				v.closeOpenChooser()
			} else {
				v.closeFiles()
				v.OpenFavorite("current", []fyne.URI{files[1], files[1]})
				waitForScan(t, v)
				waitForSort(t, v)
				waitUntilLoaded(t, v)
			}
			before, scan := v.state.Observe(), v.scanOp.lifecycle.currentRevision()
			queue.Drain()
			settleChooser(t, v)
			if after := v.state.Observe(); after.Generation() != before.Generation() || after.Favorite() != before.Favorite() || !slices.Equal(after.Retained(), before.Retained()) || v.scanOp.lifecycle.currentRevision() != scan {
				t.Fatal("queued retired chooser reopened a collection or changed its binding")
			}
		})
	}
}

func collectionCloseFacts(t *testing.T) {
	for _, action := range []string{"close_files", "reset"} {
		t.Run(action, func(t *testing.T) {
			v := newTestViewer(t)
			files := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg")
			u := storage.NewFileURI(uitest.WriteTempFile(t, "u.heic", []byte("unavailable")))
			v.OpenFavorite("favorite", []fyne.URI{files[1], u, files[0], files[0]})
			waitForScan(t, v)
			waitForSort(t, v)
			waitUntilLoaded(t, v)
			v.ShowImage(1)
			waitUntilLoaded(t, v)
			before := v.state.Observe()
			if action == "close_files" {
				if v.menus.CloseFiles().Disabled {
					t.Fatal("loaded Favorite did not admit Close Files")
				}
				v.menus.CloseFiles().Action()
			} else {
				v.reset()
			}
			after := v.state.Observe()
			if _, _, chosen := after.Current(); chosen || after.Count() != 0 || len(after.SourceFiles()) != 0 || len(after.Retained()) != 0 || after.Favorite() != "" || after.Generation() != before.Generation()+1 {
				t.Fatal("close/reset did not clear every collection fact in one publication")
			}
			mounted := false
			explorerWalk(v.win.Content(), func(object fyne.CanvasObject) { mounted = mounted || object == v.welcomeArt })
			if !mounted || !v.dropzone.Visible() || !v.welcomeArt.Visible() || v.img.Image != nil || v.searchActive() || v.grid.Visible() || v.browsing.current().binding.kind != browsingCollection {
				t.Fatal("close/reset did not retire visits and restore the mounted welcome surface")
			}
			if before.Count() != 3 || len(before.Retained()) != 4 || before.Favorite() != "favorite" || before.index != 1 {
				t.Fatal("close/reset mutated its retained previous observation")
			}
		})
	}
}

func collectionPreparationLifecycle(t *testing.T) {
	for _, preparation := range []string{"scan", "replay", "sort"} {
		for _, action := range []string{"cancel", "replacement", "close_reopen", "stop"} {
			t.Run(preparation+"/"+action, func(t *testing.T) {
				savedPreferences, savedSession := preferences.Load(testApp), session.Load(testApp)
				t.Cleanup(func() {
					preferences.Save(testApp, savedPreferences)
					session.Save(testApp, savedSession)
				})
				v := newTestViewer(t)
				a := uitest.TempJPEGURI(t, "a.jpg", 4, 5, color.White)
				b := uitest.TempJPEGURI(t, "b.jpg", 6, 7, color.Black)
				u := storage.NewFileURI(uitest.WriteTempFile(t, "u.heic", []byte("unavailable")))
				v.OpenFavorite("original", []fyne.URI{a, u, b, a})
				waitForScan(t, v)
				waitForSort(t, v)
				waitUntilLoaded(t, v)
				v.display.WaitPreloads()
				entered, release := make(chan struct{}), make(chan struct{})
				var releaseOnce, readOnce sync.Once
				unblock := func() { releaseOnce.Do(func() { close(release) }) }
				defer unblock()
				var done completion.Handle
				switch preparation {
				case "scan":
					root := uitest.DirectoryURI(storage.NewFileURI(filepath.Join(t.TempDir(), "held")), func() ([]fyne.URI, error) {
						close(entered)
						<-release
						return []fyne.URI{b}, nil
					})
					v.handleDrop([]fyne.URI{root})
					done = v.scanOp.done.Current()
				case "replay":
					v.configureHEIC(testHEICBackend{check: func(_ context.Context) error {
						close(entered)
						<-release
						return heic.ErrUnavailable
					}})
					v.heic.ui = &uitest.UIQueue{}
					v.OpenFavorite("pending", []fyne.URI{u, b, u})
					done = v.scanOp.done.Current()
				case "sort":
					wrapped := uitest.ReaderURI(a, func() (io.ReadCloser, error) {
						readOnce.Do(func() { close(entered); <-release })
						return os.Open(a.Path())
					})
					v.state.Replace(collectionInput{source: []fyne.URI{wrapped, b, wrapped}, display: []fyne.URI{wrapped, wrapped, b}, retained: []collectionSource{{wrapped, false}, {u, true}, {b, false}, {wrapped, false}}, favorite: "original"})
					v.SetSortMode(filesort.ByCaptureDate)
					done = v.sortOp.done.Current()
				}
				select {
				case <-entered:
				case <-time.After(testTimeout):
					t.Fatal("preparation did not reach its held worker")
				}
				before := v.state.Observe()
				switch action {
				case "cancel":
					if preparation == "sort" {
						v.cancelSort()
					} else {
						v.cancelScan()
					}
				case "replacement", "close_reopen":
					if action == "close_reopen" {
						v.closeFiles()
					}
					v.OpenFavorite("current", []fyne.URI{b, b})
					waitForScan(t, v)
					waitForSort(t, v)
					waitUntilLoaded(t, v)
					v.OpenSimilarityCohort([]string{b.Path()})
					v.grid.Settle()
				case "stop":
					lifecycle := v.app.Lifecycle().(interface{ OnStopped() func() })
					previous := lifecycle.OnStopped()
					registerShutdown(v.app, v)
					shutdown := lifecycle.OnStopped()
					v.app.Lifecycle().SetOnStopped(previous)
					shutdown()
				}
				committed, visit := v.state.Observe(), v.browsing.current().binding
				unblock()
				waitHandle(t, "retired preparation", done)
				v.settleHEIC()
				v.display.Settle()
				v.grid.Settle()
				after := v.state.Observe()
				if after.Generation() != committed.Generation() || after.Favorite() != committed.Favorite() || !slices.Equal(after.Retained(), committed.Retained()) || !slices.Equal(after.DisplayFiles(), committed.DisplayFiles()) || v.browsing.current().binding != visit {
					t.Fatal("retired preparation changed committed collection facts or restored an obsolete visit")
				}
				if action == "cancel" || action == "stop" {
					if after.Generation() != before.Generation() || after.Favorite() != "original" || len(after.Retained()) != 4 {
						t.Fatal("canceled preparation discarded committed unavailable membership")
					}
				} else if after.Favorite() != "current" || after.Count() != 2 || !v.grid.Visible() || !v.browsing.has(browsingExplorer) {
					t.Fatal("fresh repeated collection and cohort did not survive retired preparation")
				}
			})
		}
	}
}
