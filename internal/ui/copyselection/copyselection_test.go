package copyselection_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/ui/copyselection"
)

func TestDrawSelection(t *testing.T) {
	var copied []image.Rectangle
	view := copyselection.View{
		ImageBounds: image.Rect(0, 0, 100, 80),
		Position:    fyne.NewPos(20, 10),
		Size:        fyne.NewSize(200, 160),
	}
	feature, selectionCanvas := newFeatureCanvas(t, view, copyselection.Callbacks{
		Copy: func(bounds image.Rectangle) {
			copied = append(copied, bounds)
		},
	})

	// Drag from image pixel (75, 60) back to (25, 20). The literal
	// expectation proves direction normalization and the canvas-to-image
	// transform without repeating the production formula in the test.
	test.Drag(selectionCanvas, fyne.NewPos(70, 50), -100, -80)

	if got := feature.State(); got != (copyselection.State{Active: true, HasSelection: true}) {
		t.Fatalf("State() after drag = %+v, want active with a committed image-region selection", got)
	}
	if handled := feature.HandleKey(fyne.KeyReturn); !handled {
		t.Fatal("HandleKey(Return) = false, want the active mode to consume it")
	}
	if len(copied) != 1 || copied[0] != image.Rect(25, 20, 75, 60) {
		t.Fatalf("copy requests = %v, want [(25,20)-(75,60)]", copied)
	}
	if got := feature.State(); got != (copyselection.State{Active: true, Busy: true, HasSelection: true}) {
		t.Fatalf("State() after copy request = %+v, want active, busy, and selected", got)
	}
}

func TestFeatureEncode_UsesSourceCapturedAtStart(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 100, 80))
	src.SetNRGBA(25, 20, color.NRGBA{R: 255, A: 200})
	src.SetNRGBA(74, 59, color.NRGBA{G: 180, A: 127})

	view := copyselection.View{
		ImageBounds: image.Rect(0, 0, 100, 80),
		Position:    fyne.NewPos(20, 10),
		Size:        fyne.NewSize(200, 160),
	}
	var copied []image.Rectangle
	test.NewTempApp(t)
	feature := copyselection.New(copyselection.Callbacks{
		Copy: func(bounds image.Rectangle) { copied = append(copied, bounds) },
	})
	selectionCanvas := test.NewCanvas()
	selectionCanvas.SetPadded(false)
	selectionCanvas.SetContent(feature.Overlay())
	selectionCanvas.Resize(fyne.NewSize(300, 220))
	feature.Start(view, copyselection.RasterSource(src))

	test.Drag(selectionCanvas, fyne.NewPos(70, 50), -100, -80)
	feature.HandleKey(fyne.KeyReturn)
	if len(copied) != 1 {
		t.Fatalf("copy requests = %d, want 1", len(copied))
	}

	data, err := feature.Encode(copied[0])
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}
	got, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("png.Decode() error = %v", err)
	}
	if pixel := color.NRGBAModel.Convert(got.At(0, 0)).(color.NRGBA); pixel != (color.NRGBA{R: 255, A: 200}) {
		t.Errorf("encoded origin pixel = %#v, want marked source (25,20)", pixel)
	}
	if pixel := color.NRGBAModel.Convert(got.At(49, 39)).(color.NRGBA); pixel != (color.NRGBA{G: 180, A: 127}) {
		t.Errorf("encoded far pixel = %#v, want marked source (74,59)", pixel)
	}
}

func TestInvalidReplacement(t *testing.T) {
	var copied []image.Rectangle
	view := copyselection.View{
		ImageBounds: image.Rect(0, 0, 100, 80),
		Size:        fyne.NewSize(100, 80),
	}
	feature, selectionCanvas := newFeatureCanvas(t, view, copyselection.Callbacks{
		Copy: func(bounds image.Rectangle) {
			copied = append(copied, bounds)
		},
	})

	test.Drag(selectionCanvas, fyne.NewPos(40, 40), 30, 30)
	// This replacement starts outside the committed rectangle but spans less
	// than one image pixel on each axis, so the old rectangle must survive.
	test.Drag(selectionCanvas, fyne.NewPos(60.5, 60.5), .5, .5)
	feature.HandleKey(fyne.KeyReturn)

	if len(copied) != 1 || copied[0] != image.Rect(10, 10, 40, 40) {
		t.Fatalf("copy requests after invalid replacement = %v, want [(10,10)-(40,40)]", copied)
	}
}

func newFeatureCanvas(t *testing.T, view copyselection.View, callbacks copyselection.Callbacks) (*copyselection.Feature, fyne.Canvas) {
	t.Helper()
	test.NewTempApp(t)
	feature := copyselection.New(callbacks)
	selectionCanvas := test.NewCanvas()
	selectionCanvas.SetPadded(false)
	selectionCanvas.SetContent(feature.Overlay())
	selectionCanvas.Resize(fyne.NewSize(300, 220))
	feature.Start(view, copyselection.Source{})
	return feature, selectionCanvas
}
