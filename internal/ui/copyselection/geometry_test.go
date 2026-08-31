package copyselection_test

import (
	"image"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/ui/copyselection"
)

func TestPixelBounds(t *testing.T) {
	var copied []image.Rectangle
	view := copyselection.View{
		ImageBounds: image.Rect(0, 0, 100, 80),
		Size:        fyne.NewSize(100, 80),
	}
	feature, selectionCanvas := newFeatureCanvas(t, view, copyselection.Callbacks{
		Copy: func(bounds image.Rectangle) { copied = append(copied, bounds) },
	})

	test.Drag(selectionCanvas, fyne.NewPos(11, 11), 1, 1)
	feature.HandleKey(fyne.KeyReturn)

	if len(copied) != 1 || copied[0] != image.Rect(10, 10, 11, 11) {
		t.Fatalf("copy requests = %v, want the outward-rounded 1x1 crop [(10,10)-(11,11)]", copied)
	}
}

func TestViewportTransform(t *testing.T) {
	var copied []image.Rectangle
	feature, selectionCanvas := newFeatureCanvas(t, sampleView, copyselection.Callbacks{
		Copy: func(bounds image.Rectangle) { copied = append(copied, bounds) },
	})
	commitSampleSelection(t, selectionCanvas)

	feature.ViewChanged(copyselection.View{
		ImageBounds: sampleView.ImageBounds,
		Position:    fyne.NewPos(30, 15),
		Size:        sampleView.Size,
	})
	feature.HandleKey(fyne.KeyReturn)

	if len(copied) != 1 || copied[0] != image.Rect(25, 20, 75, 60) {
		t.Fatalf("copy requests after pan = %v, want the original image pixels [(25,20)-(75,60)]", copied)
	}
}

func TestHiDPIGeometry(t *testing.T) {
	var copied []image.Rectangle
	feature, selectionCanvas := newFeatureCanvas(t, sampleView, copyselection.Callbacks{
		Copy: func(bounds image.Rectangle) { copied = append(copied, bounds) },
	})
	commitSampleSelection(t, selectionCanvas)

	feature.ViewChanged(copyselection.View{
		ImageBounds: sampleView.ImageBounds,
		Position:    sampleView.Position,
		Size:        fyne.NewSize(400, 320),
	})
	feature.HandleKey(fyne.KeyReturn)

	if len(copied) != 1 || copied[0] != image.Rect(25, 20, 75, 60) {
		t.Fatalf("copy requests after HiDPI scale = %v, want the original image pixels [(25,20)-(75,60)]", copied)
	}
}
