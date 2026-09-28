package display_test

import (
	"image"
	"sync/atomic"
	"testing"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/display"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestMA032DeliveryIntegration(t *testing.T) {
	t.Run("inline_reentry", func(t *testing.T) {
		queue := &uitest.UIQueue{}
		var f *presentationFixture
		armed := false
		f = newPresentation(t, display.Config{Queue: ma032InlineQueue{}, Callbacks: display.Callbacks{Repaint: func() {
			if armed {
				armed = false
				f.SetUIQueue(queue)
				f.RequestVectorRender(3, nil)
			}
		}}})
		f.SetVectorOptions(display.VectorOptions{Rasterize: func(_ *imaging.Vector, w, h int) (image.Image, error) {
			return image.NewNRGBA(image.Rect(0, 0, w, h)), nil
		}})
		f.Present(storage.NewFileURI("/reentrant.svg"), vectorRecord(t), false)
		armed = true
		f.RequestVectorRender(2, nil)
		f.Wait()
		if armed || f.Surface().Image.Bounds().Size() != image.Pt(1040, 520) {
			t.Fatal("inline SVG delivery did not publish and admit reentrant work")
		}
		queue.Drain()
		if f.Surface().Image.Bounds().Size() != image.Pt(1560, 780) {
			t.Fatal("old SVG finalization canceled its reentrant replacement")
		}
	})
	t.Run("queued_superseded", func(t *testing.T) {
		oldQueue, currentQueue := &uitest.UIQueue{}, &uitest.UIQueue{}
		f := newPresentation(t, display.Config{Queue: oldQueue})
		f.SetVectorOptions(display.VectorOptions{Rasterize: func(_ *imaging.Vector, w, h int) (image.Image, error) {
			return image.NewNRGBA(image.Rect(0, 0, w, h)), nil
		}})
		loaded := vectorRecord(t)
		f.Present(storage.NewFileURI("/queued.svg"), loaded, false)
		baseline := f.Surface().Image
		f.RequestVectorRender(2, nil)
		f.Wait()
		if f.Surface().Image != baseline {
			t.Fatal("raster worker published before delivery")
		}
		f.SetUIQueue(currentQueue)
		f.RequestVectorRender(3, nil)
		f.Wait()
		if !oldQueue.Drain() {
			t.Fatal("old raster never reached the held queue")
		}
		if f.Surface().Image != baseline {
			t.Fatal("superseded queued raster replaced current pixels")
		}
		if !currentQueue.Drain() || f.Surface().Image.Bounds().Size() != image.Pt(1560, 780) {
			t.Fatal("current queued raster lost its release handoff")
		}
	})
	for _, action := range []string{"clear_reopen", "stop"} {
		t.Run(action, func(t *testing.T) {
			queue := &uitest.UIQueue{}
			f := newPresentation(t, display.Config{Queue: queue})
			var rasterized atomic.Int32
			f.SetVectorOptions(display.VectorOptions{Rasterize: func(_ *imaging.Vector, w, h int) (image.Image, error) {
				rasterized.Add(1)
				return image.NewNRGBA(image.Rect(0, 0, w, h)), nil
			}})
			loaded := vectorRecord(t)
			source := storage.NewFileURI("/reopened.svg")
			f.Present(source, loaded, false)
			f.RequestVectorRender(2, nil)
			f.Wait()
			if action == "stop" {
				f.Stop()
			} else {
				f.Clear()
			}
			f.Present(source, loaded, false)
			before := f.Surface().Image
			if !queue.Drain() || f.Surface().Image != before {
				t.Fatal("retired SVG delivery changed the reopened or stopped surface")
			}
			f.RequestVectorRender(3, nil)
			f.Settle()
			if action == "stop" {
				if f.Surface().Image != nil || rasterized.Load() != 1 {
					t.Fatal("terminal Stop admitted new presentation or raster work")
				}
			} else if rasterized.Load() != 2 || f.Surface().Image.Bounds().Size() != image.Pt(1560, 780) {
				t.Fatal("Clear did not permit fresh SVG work on reopen")
			}
		})
	}
}

type ma032InlineQueue struct{}

func (ma032InlineQueue) Do(run func()) { run() }
func (ma032InlineQueue) Drain() bool   { return false }
