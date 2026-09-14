package display_test

import (
	"errors"
	"image"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/display"
	"github.com/frathe/picfetch/internal/uitest"
)

func vectorContract(t *testing.T) {
	t.Run("debounce and cancellation", vectorDebounceContract)
	t.Run("failed sharpening", vectorFailureContract)
	queue := &uitest.UIQueue{}
	f := newPresentation(t, display.Config{Queue: queue})
	f.SetVectorOptions(display.VectorOptions{})
	defer func() { f.Stop(); f.Wait(); queue.Drain() }()
	vec, err := imaging.ParseVector([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="200" height="100"><rect width="200" height="100" fill="red"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	b := vec.Logical()
	pixels, err := vec.RasterAt(b.Dx(), b.Dy())
	if err != nil {
		t.Fatal(err)
	}
	loaded := &imaging.LoadedImage{Frames: []image.Image{pixels}, Vector: vec}
	beforeBytes := loaded.DecodedBytes()
	f.Present(storage.NewFileURI("/wide.svg"), loaded, false)
	f.RotateBy(1)
	before := f.Surface().Image
	f.RequestVectorRender(2, func(p fyne.Position) (int, int) { return int(p.X * 2), int(p.Y * 2) })
	f.Wait()
	if f.Surface().Image != before {
		t.Fatal("layout request mutated the surface before UI delivery")
	}
	f.Settle()
	if got := f.Surface().Image.Bounds().Size(); got != image.Pt(1040, 2080) {
		t.Fatalf("device-pixel rotated raster = %v, want 1040x2080", got)
	}
	if f.Snapshot().Size != fyne.NewSize(260, 520) {
		t.Fatal("sharpening changed logical dimensions")
	}
	capture, release, ok := f.CaptureStable()
	if !ok || capture.Vector != vec || capture.LogicalSize != image.Pt(520, 260) || capture.Rotation != 1 || capture.Pixels != f.Surface().Image {
		t.Fatal("vector capture lost logical content or published raster")
	}
	release()
	if loaded.Frames[0] != pixels || loaded.DecodedBytes() != beforeBytes {
		t.Fatal("live raster changed cached content or byte accounting")
	}
	f.RequestVectorRender(8, nil)
	f.Wait()
	f.Clear()
	f.Present(storage.NewFileURI("/replacement.png"), &imaging.LoadedImage{Frames: []image.Image{pixels}}, false)
	f.Settle()
	if f.Surface().Image != pixels || f.Snapshot().Vector {
		t.Fatal("stale vector delivery replaced a raster presentation")
	}
}

func vectorDebounceContract(t *testing.T) {
	queue := &uitest.UIQueue{}
	f := newPresentation(t, display.Config{Queue: queue})
	release := make(chan time.Time)
	var rasterized atomic.Int32
	f.SetVectorOptions(display.VectorOptions{Debounce: time.Hour, After: func(_ time.Duration) <-chan time.Time { return release }, Rasterize: func(vec *imaging.Vector, w, h int) (image.Image, error) {
		rasterized.Add(1)
		return vec.RasterAt(w, h)
	}})
	defer func() { f.Stop(); f.Wait(); queue.Drain() }()
	loaded := vectorRecord(t)
	f.Present(storage.NewFileURI("/debounce.svg"), loaded, false)
	for scale := float32(2); scale <= 6; scale++ {
		f.RequestVectorRender(scale, nil)
	}
	close(release)
	f.Settle()
	if rasterized.Load() != 1 || f.Surface().Image.Bounds().Dx() != 3120 {
		t.Fatal("a burst did not publish only the final requested raster")
	}
	// A second instance parked in debounce must stop without a clock tick or UI drain.
	parked := newPresentation(t, display.Config{Queue: &uitest.UIQueue{}})
	parked.SetVectorOptions(display.VectorOptions{Debounce: time.Hour, After: func(_ time.Duration) <-chan time.Time { return make(chan time.Time) }})
	parked.Present(storage.NewFileURI("/parked.svg"), loaded, false)
	parked.RequestVectorRender(2, nil)
	parked.Stop()
	parked.Wait()
	parked.Settle()
}

func vectorFailureContract(t *testing.T) {
	f := newPresentation(t, display.Config{Queue: &uitest.UIQueue{}})
	f.SetVectorOptions(display.VectorOptions{Rasterize: func(_ *imaging.Vector, _, _ int) (image.Image, error) { return nil, errors.New("raster failed") }})
	defer func() { f.Stop(); f.Wait(); f.Settle() }()
	loaded := vectorRecord(t)
	f.Present(storage.NewFileURI("/failure.svg"), loaded, false)
	writes := f.AppliedFrames()
	f.RequestVectorRender(2, nil)
	f.Settle()
	if f.Surface().Image != loaded.Frames[0] || f.AppliedFrames() != writes {
		t.Fatal("failed sharpening replaced the valid image")
	}
}

func vectorRecord(t *testing.T) *imaging.LoadedImage {
	t.Helper()
	vec, err := imaging.ParseVector([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="200" height="100"><rect width="200" height="100" fill="red"/></svg>`))
	if err != nil {
		t.Fatal(err)
	}
	b := vec.Logical()
	pixels, err := vec.RasterAt(b.Dx(), b.Dy())
	if err != nil {
		t.Fatal(err)
	}
	return &imaging.LoadedImage{Frames: []image.Image{pixels}, Vector: vec}
}
