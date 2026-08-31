package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestCopySelectionPixels(t *testing.T) {
	v := newTestViewer(t)
	src := markedRegionCopyImage(20, 15)
	dropAndWait(t, v, regionCopyPNGURI(t, "literal.png", src))
	v.win.Resize(fyne.NewSize(500, 300))
	v.zoom.ActualSize()
	for range 8 {
		v.zoom.In()
	}
	v.zoom.Widget().(fyne.Draggable).Dragged(&fyne.DragEvent{Dragged: fyne.NewDelta(35, -20)})

	var copied []byte
	uitest.StubClipboardCopy(t, func(data []byte) error {
		copied = append([]byte(nil), data...)
		return nil
	})

	wantBounds := image.Rect(4, 3, 16, 12)
	selectRegion(t, v, wantBounds)
	v.regionCopy.HandleKey(fyne.KeyReturn)
	waitForClipboard(t, v)

	got := decodeRegionCopyPNG(t, copied)
	if got.Bounds() != image.Rect(0, 0, wantBounds.Dx(), wantBounds.Dy()) {
		t.Fatalf("copied bounds = %v, want %v and %dx%d zero-origin PNG",
			got.Bounds(), wantBounds, wantBounds.Dx(), wantBounds.Dy())
	}
	for y := range wantBounds.Dy() {
		for x := range wantBounds.Dx() {
			gotPixel := color.NRGBAModel.Convert(got.At(x, y)).(color.NRGBA)
			wantPixel := src.NRGBAAt(wantBounds.Min.X+x, wantBounds.Min.Y+y)
			if gotPixel != wantPixel {
				t.Errorf("copied pixel (%d,%d) = %#v, want source pixel (%d,%d) %#v",
					x, y, gotPixel, wantBounds.Min.X+x, wantBounds.Min.Y+y, wantPixel)
			}
		}
	}
}

func selectRegion(t *testing.T, v *viewer, bounds image.Rectangle) {
	t.Helper()
	v.startRegionCopy()
	if !v.regionCopy.State().Active {
		t.Fatal("Copy Selection mode did not start")
	}

	geometry := v.zoom.Geometry()
	w, h := v.displayedDimensions()
	toCanvas := func(x, y float32) fyne.Position {
		return fyne.NewPos(
			geometry.Position.X+x*geometry.Size.Width/float32(w),
			geometry.Position.Y+y*geometry.Size.Height/float32(h),
		)
	}
	// Stay inside both edges. The test driver's synthetic drag reconstructs
	// the origin from the end and a float32 delta, so the wider inset at the
	// minimum keeps floor/ceil stable after that extra subtraction.
	start := toCanvas(float32(bounds.Min.X)+0.75, float32(bounds.Min.Y)+0.75)
	end := toCanvas(float32(bounds.Max.X)-0.25, float32(bounds.Max.Y)-0.25)
	test.Drag(v.win.Canvas(), end, end.X-start.X, end.Y-start.Y)
	if !v.regionCopy.State().HasSelection {
		t.Fatalf("drag did not commit image bounds %v", bounds)
	}
}

func markedRegionCopyImage(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, color.NRGBA{
				R: uint8(20 + x*23),
				G: uint8(10 + y*37),
				B: uint8(5 + x*7 + y*11),
				A: uint8(70 + x*13 + y*9),
			})
		}
	}
	return img
}

func regionCopyPNGURI(t *testing.T, name string, img image.Image) fyne.URI {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode literal PNG: %v", err)
	}
	return storage.NewFileURI(uitest.WriteTempFile(t, name, buf.Bytes()))
}

func decodeRegionCopyPNG(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode copied PNG: %v", err)
	}
	return img
}
