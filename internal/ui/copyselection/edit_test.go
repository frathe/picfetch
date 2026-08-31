package copyselection_test

import (
	"image"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/ui/copyselection"
)

func TestMoveSelection(t *testing.T) {
	var copied []image.Rectangle
	feature, selectionCanvas := newFeatureCanvas(t, sampleView, copyselection.Callbacks{
		Copy: func(bounds image.Rectangle) { copied = append(copied, bounds) },
	})
	commitSampleSelection(t, selectionCanvas)

	// Interior canvas point (120,90) is image (50,40). Drag +20,+16 canvas
	// (+10,+8 image) without changing size.
	test.Drag(selectionCanvas, fyne.NewPos(140, 106), 20, 16)
	feature.HandleKey(fyne.KeyReturn)

	if len(copied) != 1 || copied[0] != image.Rect(35, 28, 85, 68) {
		t.Fatalf("copy requests after move = %v, want [(35,28)-(85,68)]", copied)
	}
}

func TestResizeSelection(t *testing.T) {
	var copied []image.Rectangle
	feature, selectionCanvas := newFeatureCanvas(t, sampleView, copyselection.Callbacks{
		Copy: func(bounds image.Rectangle) { copied = append(copied, bounds) },
	})
	commitSampleSelection(t, selectionCanvas)

	// East handle is at canvas (170,90). Drag it +20 canvas (+10 image).
	test.Drag(selectionCanvas, fyne.NewPos(190, 90), 20, 0)
	feature.HandleKey(fyne.KeyReturn)

	if len(copied) != 1 || copied[0] != image.Rect(25, 20, 85, 60) {
		t.Fatalf("copy requests after east resize = %v, want [(25,20)-(85,60)]", copied)
	}
}

func TestCrossedResizeHandle(t *testing.T) {
	var copied []image.Rectangle
	feature, selectionCanvas := newFeatureCanvas(t, sampleView, copyselection.Callbacks{
		Copy: func(bounds image.Rectangle) { copied = append(copied, bounds) },
	})
	commitSampleSelection(t, selectionCanvas)

	// Drag the east handle past the west edge so the rectangle flips.
	test.Drag(selectionCanvas, fyne.NewPos(50, 90), -120, 0)
	feature.HandleKey(fyne.KeyReturn)

	if len(copied) != 1 || copied[0] != image.Rect(15, 20, 25, 60) {
		t.Fatalf("copy requests after crossed handle = %v, want [(15,20)-(25,60)]", copied)
	}
}
