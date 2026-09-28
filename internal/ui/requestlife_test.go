package ui

import (
	"context"
	"image"
	"slices"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/session"
	"github.com/frathe/picfetch/internal/ui/display"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestMA032DeliveryIntegration(t *testing.T) {
	t.Run("sort", func(t *testing.T) {
		t.Run("queued_current", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				v := newTestViewer(t)
				queue := &uitest.UIQueue{}
				v.sortDo = queue.Do
				defer queue.Drain()
				a, b := uitest.FakeURI{FileName: "a.jpg"}, uitest.FakeURI{FileName: "b.jpg"}
				original := []fyne.URI{b, a}
				v.state.Replace(collectionInput{source: original, display: original})
				v.startSort(filesort.ByName, original, func(ordered []fyne.URI) { v.state.Reorder(ordered) })
				done := v.sortOp.done.Current()
				synctest.Wait()
				ma032RequirePending(t, done)
				if !v.sortOp.active || !slices.Equal(v.state.Observe().DisplayFiles(), original) {
					t.Fatal("worker exit applied order or finalized progress before UI delivery")
				}
				if !queue.Drain() {
					t.Fatal("sort did not queue its result")
				}
				waitHandle(t, "delivered sort", done)
				if v.sortOp.active || !slices.Equal(v.state.Observe().DisplayFiles(), []fyne.URI{a, b}) {
					t.Fatal("current queued sort did not apply ordering and finish progress")
				}
			})
		})
		t.Run("queued_superseded", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				v := newTestViewer(t)
				oldQueue, currentQueue := &uitest.UIQueue{}, &uitest.UIQueue{}
				defer currentQueue.Drain()
				defer oldQueue.Drain()
				a, b := uitest.FakeURI{FileName: "a.jpg"}, uitest.FakeURI{FileName: "b.jpg"}
				original := []fyne.URI{b, a}
				v.state.Replace(collectionInput{source: original, display: original})
				before := v.Generation()
				v.sortDo = oldQueue.Do
				v.startSort(filesort.ByName, original, func(ordered []fyne.URI) { v.state.Reorder(ordered) })
				old := v.sortOp.done.Current()
				synctest.Wait()
				v.sortDo = currentQueue.Do
				v.startSort(filesort.ByDropOrder, original, func(ordered []fyne.URI) { v.state.Reorder(ordered) })
				current := v.sortOp.done.Current()
				synctest.Wait()
				oldQueue.Drain()
				waitHandle(t, "obsolete delivery", old)
				ma032RequirePending(t, current)
				if v.Generation() != before || !v.sortOp.active || !v.sortOp.spinner.Visible() || !v.sortOp.label.Visible() {
					t.Fatal("stale queued sort changed collection or newer progress")
				}
				currentQueue.Drain()
				waitHandle(t, "current delivery", current)
				if v.sortOp.active || !slices.Equal(v.state.Observe().DisplayFiles(), original) {
					t.Fatal("current sort lost ownership after obsolete delivery")
				}
			})
		})
		t.Run("inline_reentry", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				v := newTestViewer(t)
				queue := &uitest.UIQueue{}
				defer queue.Drain()
				a, b := uitest.FakeURI{FileName: "a.jpg"}, uitest.FakeURI{FileName: "b.jpg"}
				original := []fyne.URI{b, a}
				v.state.Replace(collectionInput{source: original, display: original})
				var newer completion.Handle
				var older completion.Handle
				v.sortDo = func(run func()) { run() }
				v.startSort(filesort.ByName, original, func(ordered []fyne.URI) {
					older = v.sortOp.done.Current()
					v.state.Reorder(ordered)
					v.sortDo = queue.Do
					v.startSort(filesort.ByDropOrder, original, func(current []fyne.URI) { v.state.Reorder(current) })
					newer = v.sortOp.done.Current()
				})
				synctest.Wait()
				waitHandle(t, "inline sort", older)
				ma032RequirePending(t, newer)
				if !v.sortOp.active {
					t.Fatal("old inline finalization cleared reentrant sort progress")
				}
				queue.Drain()
				waitHandle(t, "reentrant sort", newer)
				if v.sortOp.active || !slices.Equal(v.state.Observe().DisplayFiles(), original) {
					t.Fatal("old inline finalization canceled reentrant sort")
				}
			})
		})
	})
	t.Run("production_shutdown", func(t *testing.T) {
		t.Run("held_svg", func(t *testing.T) { ma032SVGShutdown(t, true) })
		t.Run("queued_svg", func(t *testing.T) { ma032SVGShutdown(t, false) })
		t.Run("queued_sort", func(t *testing.T) {
			synctest.Test(t, func(t *testing.T) {
				v := newTestViewer(t)
				queue := &uitest.UIQueue{}
				v.sortDo = queue.Do
				defer queue.Drain()
				a, b := uitest.FakeURI{FileName: "a.jpg"}, uitest.FakeURI{FileName: "b.jpg"}
				original := []fyne.URI{b, a}
				v.state.Replace(collectionInput{source: original, display: original})
				before := v.Generation()
				v.startSort(filesort.ByName, original, func(ordered []fyne.URI) { v.state.Reorder(ordered) })
				done := v.sortOp.done.Current()
				synctest.Wait()
				ma032Shutdown(t, v)
				// The real post-event-loop wait excludes sort delivery. Returning
				// with this callback held must not claim the operation completed.
				v.waitForShutdown()
				ma032RequirePending(t, done)
				queue.Drain()
				waitHandle(t, "sort delivered after shutdown", done)
				if v.Generation() != before || !slices.Equal(v.state.Observe().DisplayFiles(), original) {
					t.Fatal("queued sort changed collection after production shutdown")
				}
			})
		})
	})
}

func ma032SVGShutdown(t *testing.T, holdRaster bool) {
	t.Helper()
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempSVGURI(t, "shutdown.svg", 24, 24))
	v.display.Settle()
	queue := &uitest.UIQueue{}
	v.display.SetUIQueue(queue)
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	v.display.SetVectorOptions(display.VectorOptions{Rasterize: func(_ *imaging.Vector, w, h int) (image.Image, error) {
		close(entered)
		if holdRaster {
			<-release
		}
		return image.NewNRGBA(image.Rect(0, 0, w, h)), nil
	}})
	before := v.img.Image
	v.display.RequestVectorRender(4, nil)
	select {
	case <-entered:
	case <-time.After(testTimeout):
		t.Fatal("SVG worker did not reach the controlled rasterizer")
	}
	if !holdRaster {
		v.display.Wait()
	}
	ma032Shutdown(t, v)
	// Production joins its backends without joining blocked display raster work
	// or draining UI. The stronger test cleanup joins it only after release.
	v.waitForShutdown()
	if v.img.Image == before {
		t.Fatal("production shutdown did not clear display presentation")
	}
	stoppedPixels := v.img.Image
	unblock()
	v.display.Wait()
	queue.Drain()
	if v.img.Image != stoppedPixels || v.display.Snapshot().Vector {
		t.Fatal("retired SVG result revived display after production shutdown")
	}
}

func ma032Shutdown(t *testing.T, v *viewer) {
	t.Helper()
	savedPreferences, savedSession := preferences.Load(v.app), session.Load(v.app)
	t.Cleanup(func() {
		preferences.Save(v.app, savedPreferences)
		session.Save(v.app, savedSession)
	})
	lifecycle := v.app.Lifecycle().(interface{ OnStopped() func() })
	previous := lifecycle.OnStopped()
	registerShutdown(v.app, v)
	shutdown := lifecycle.OnStopped()
	v.app.Lifecycle().SetOnStopped(previous)
	shutdown()
}

// Called only inside a synctest bubble: quiescence proves the waiter is blocked
// on its own completion generation, independently of worker/queue counters.
func ma032RequirePending(t *testing.T, handle completion.Handle) {
	t.Helper()
	finished := false
	go func() {
		_ = handle.Wait(context.Background())
		finished = true
	}()
	synctest.Wait()
	if finished {
		t.Fatal("operation completed before its queued delivery")
	}
}
