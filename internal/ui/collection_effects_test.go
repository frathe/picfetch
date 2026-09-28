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
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestCollectionCommittedEffects(t *testing.T) {
	t.Run("callers", collectionCommittedWriteCallers)
	t.Run("chosen_index_revalidation", func(t *testing.T) {
		v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
		v.grid.Close()
		v.ShowImage(1)
		waitUntilLoaded(t, v)
		v.display.WaitPreloads()
		chosen := v.FileAt(1)
		result, err := imaging.ExportContext(context.Background(), chosen, image.NewRGBA(image.Rect(0, 0, 19, 13)), nil, imaging.ExportOptions{})
		if err != nil || !result.Committed {
			t.Fatalf("fixture write did not commit: %v", err)
		}
		v.AfterFileExported(result)
		v.fileWork.workers.Wait()
		if !v.fileWork.ui.Drain() {
			t.Fatal("affected-source delivery was not queued")
		}
		v.fileWork.workers.Wait() // Hold the captured current-index decision.
		revision := v.display.RequestRevision()
		v.RemoveFile(0)
		before := v.state.Observe()
		if uri, index, ok := before.Current(); !ok || uri != chosen || index != 0 || v.display.RequestRevision() != revision {
			t.Fatal("fixture did not shift the chosen occurrence without a fresh load")
		}
		drainFileWork(t, v)
		waitUntilLoaded(t, v)
		after := v.state.Observe()
		if uri, index, ok := after.Current(); !ok || uri != chosen || index != 0 || v.img.Image.Bounds().Size() != image.Pt(19, 13) || after.Generation() != before.Generation() || !slices.Equal(after.Retained(), before.Retained()) {
			t.Fatal("committed refresh used an obsolete collection index instead of its current binding")
		}
	})
}

func collectionCommittedWriteCallers(t *testing.T) {
	for _, caller := range []string{"save", "export", "metadata", "mosaic"} {
		t.Run(caller, func(t *testing.T) {
			v := newTestViewer(t)
			source := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
			other := uitest.TempJPEGURI(t, "z.jpg", 14, 7, color.Black)
			alias := storage.NewFileURI(filepath.Join(t.TempDir(), "alias.jpg"))
			if err := os.Symlink(source.Path(), alias.Path()); err != nil {
				t.Fatal(err)
			}
			u := storage.NewFileURI(uitest.WriteTempFile(t, "u.heic", []byte("unavailable")))
			v.OpenFavorite("original", []fyne.URI{source, other})
			waitForScan(t, v)
			waitForSort(t, v)
			waitUntilLoaded(t, v)
			v.display.WaitPreloads()
			entered, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			defer unblock()
			var result imaging.WriteResult
			var writeErr error
			switch caller {
			case "save":
				v.fileWork.save = func(ctx context.Context, uri fyne.URI, pixels image.Image) (imaging.WriteResult, error) {
					result, writeErr = imaging.SaveRotatedContext(ctx, uri, pixels)
					close(entered)
					<-release
					return result, writeErr
				}
				v.rotateBy(1)
				v.saveRotation()
			case "export":
				v.fileWork.export = func(ctx context.Context, dest fyne.URI, pixels image.Image, original fyne.URI, options imaging.ExportOptions) (imaging.WriteResult, error) {
					result, writeErr = imaging.ExportContext(ctx, dest, pixels, original, options)
					close(entered)
					<-release
					return result, writeErr
				}
				uitest.StubSaveChooser(t, func(_ string) (fyne.URI, error) { return source, nil })
				v.rotateBy(1)
				v.exportAs(".jpg")
			case "metadata":
				result, writeErr = imaging.StripJPEGMetadataContext(context.Background(), source)
				close(entered)
			case "mosaic":
				result, writeErr = imaging.ExportContext(context.Background(), source, image.NewRGBA(image.Rect(0, 0, 19, 13)), nil, imaging.ExportOptions{})
				close(entered)
			}
			select {
			case <-entered:
			case <-time.After(testTimeout):
				t.Fatal("write did not reach the committed-result barrier")
			}
			if writeErr != nil || !result.Committed {
				t.Fatalf("fixture write did not commit: %v", writeErr)
			}
			v.OpenFavorite("current", []fyne.URI{alias, other, u})
			waitForScan(t, v)
			waitForSort(t, v)
			waitUntilLoaded(t, v)
			v.ShowImage(1)
			waitUntilLoaded(t, v)
			v.rotateBy(1)
			before := v.state.Observe()
			pixels, rotation, toast := v.img.Image, v.display.Rotation(), v.toast.text.Text
			v.grid.StoreThumb(alias, image.NewRGBA(image.Rect(0, 0, 24, 16)))
			v.dupes.PutHash(alias.String(), 42)
			thumbs, facts := v.grid.CaptureThumbs(), v.dupes.CaptureFacts()
			preview := v.favThumbLifecycle.begin()
			defer preview.cancel()
			unblock()
			switch caller {
			case "save":
				waitForSave(t, v)
			case "export":
				settleChooser(t, v)
			case "metadata":
				v.AfterMetadataRemoved(source, result)
			case "mosaic":
				v.AfterFileExported(result)
			}
			drainFileWork(t, v)
			after := v.state.Observe()
			if after.Generation() != before.Generation() || after.Favorite() != "current" || !slices.Equal(after.Retained(), before.Retained()) || after.index != before.index || v.img.Image != pixels || v.display.Rotation() != rotation || v.toast.text.Text != toast {
				t.Fatal("stale committed effect rebound collection, navigation, pixels, rotation or toast")
			}
			if preview.current() || thumbs.Current() || facts.Current() {
				t.Fatal("committed effect missed the current collection's noncurrent source alias")
			}
		})
	}
}

func collectionCommittedWriteLifecycle(t *testing.T) {
	for _, stage := range []string{"sources", "current"} {
		for _, action := range []string{"same_membership", "replacement", "close_reopen", "stop"} {
			for _, committed := range []bool{false, true} {
				t.Run(stage+"/"+action+"/"+fmt.Sprint(committed), func(t *testing.T) {
					v := newTestViewer(t)
					source := uitest.TempJPEGURI(t, "a.jpg", 24, 16, color.White)
					other := uitest.TempJPEGURI(t, "b.jpg", 14, 7, color.Black)
					u := storage.NewFileURI(uitest.WriteTempFile(t, "u.heic", []byte("unavailable")))
					v.OpenFavorite("old", []fyne.URI{source, other, u})
					waitForScan(t, v)
					waitForSort(t, v)
					waitUntilLoaded(t, v)
					v.display.WaitPreloads()
					result := imaging.WriteResult{}
					if committed {
						var err error
						result, err = imaging.ExportContext(context.Background(), source, image.NewRGBA(image.Rect(0, 0, 19, 13)), nil, imaging.ExportOptions{})
						if err != nil || !result.Committed {
							t.Fatalf("fixture write did not commit: %v", err)
						}
						v.imgCache.Purge() // The production writer owns this commit-time effect.
					}
					var completions atomic.Int32
					v.afterFileWrite(result, true, true, func() { completions.Add(1) })
					v.fileWork.workers.Wait()
					if stage == "current" && committed {
						if !v.fileWork.ui.Drain() {
							t.Fatal("source decision was not queued")
						}
						v.fileWork.workers.Wait()
					}
					if got := completions.Load(); committed && got != 0 || !committed && got != 1 {
						t.Fatal("completion does not include its queued reconciliation")
					}
					if action == "stop" {
						v.closeFileWork()
					} else {
						if action == "close_reopen" {
							v.clearToDropzone()
						}
						files := []fyne.URI{other, u}
						if action == "same_membership" {
							files = []fyne.URI{source, other, u}
						}
						v.OpenFavorite("current", files)
						waitForScan(t, v)
						waitForSort(t, v)
						waitUntilLoaded(t, v)
					}
					before := v.state.Observe()
					preview := v.favThumbLifecycle.begin()
					defer preview.cancel()
					drainFileWork(t, v)
					v.display.Settle()
					after := v.state.Observe()
					wantInvalidation := committed && stage == "sources" && action == "same_membership"
					if completions.Load() != 1 || preview.current() == wantInvalidation || after.Generation() != before.Generation() || after.Favorite() != before.Favorite() || !slices.Equal(after.Retained(), before.Retained()) || after.index != before.index {
						t.Fatal("queued reconciliation lost completion, revalidation or current collection facts")
					}
					written, err := imaging.LoadImage(source, imaging.DefaultImgCacheBytes)
					wantSize := image.Pt(24, 16)
					if committed {
						wantSize = image.Pt(19, 13)
					}
					if err != nil || written.Frames[0].Bounds().Size() != wantSize {
						t.Fatal("retiring reconciliation changed committed disk truth")
					}
				})
			}
		}
	}
}

func collectionContentAndPolicyEffects(t *testing.T) {
	limit := imaging.MaxEncodedBytes()
	t.Cleanup(func() { imaging.SetMaxEncodedBytes(limit) })
	for _, kind := range []string{"content", "validation", "analysis", "duplicate", "same_analysis", "same_duplicate"} {
		t.Run(kind, func(t *testing.T) {
			v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
			v.display.WaitPreloads()
			v.grid.Settle()
			v.SetDuplicateDistance(0)
			for i, hash := range []uint64{0, 65535, 4294967295} {
				v.dupes.PutHash(v.FileAt(i).String(), hash)
			}
			v.dupes.SetHideDuplicates(true)
			v.grid.DuplicateDistanceChanged()
			v.grid.Settle()
			v.grid.SimulateHover(1)
			v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
			before := v.state.Observe()
			selection := v.grid.Selection()
			if !slices.Equal(selection, []int{1}) {
				t.Fatal("fixture did not select a visible nonduplicate source")
			}
			writer, thumbs, facts := v.imgCache.Capture(), v.grid.CaptureThumbs(), v.dupes.CaptureFacts()
			preview := v.favThumbLifecycle.begin()
			defer preview.cancel()
			visit := v.browsing.revision
			switch kind {
			case "content":
				// Writers already purge at disk commit; reconciliation must not
				// invent another blanket image-cache invalidation.
				v.reconcileSources(sourceChange{kind: sourceWritten, written: []fyne.URI{v.FileAt(0)}})
			case "validation":
				v.reconcileSources(sourceChange{kind: sourcesRevalidated})
			case "analysis":
				v.SetMaxFileSizeMB(v.MaxFileSizeMB() + 1)
			case "duplicate":
				v.SetDuplicateDistance(1)
			case "same_analysis":
				v.SetMaxFileSizeMB(v.MaxFileSizeMB())
			case "same_duplicate":
				v.SetDuplicateDistance(v.DuplicateDistance())
			}
			v.grid.Settle()
			v.display.Settle()
			after := v.state.Observe()
			content := kind == "content" || kind == "validation"
			noop := kind == "same_analysis" || kind == "same_duplicate"
			if writer.Current() == (kind == "validation") || thumbs.Current() == content || facts.Current() == content || preview.current() == content {
				t.Fatal("content, validation and policy invalidations lost their distinct cache/producer semantics")
			}
			if after.Generation() != before.Generation() || after.Favorite() != before.Favorite() || !slices.Equal(after.Retained(), before.Retained()) || (v.browsing.revision == visit) != noop {
				t.Fatal("named effect changed membership or no-op policy rebound browsing")
			}
			if kind == "validation" {
				if v.grid.SelectionCount() != 0 {
					t.Fatal("validation did not rebuild positional interaction state")
				}
			} else if !slices.Equal(v.grid.Selection(), selection) {
				t.Fatal("nonmembership effect discarded unaffected Grid selection")
			}
		})
	}
}
