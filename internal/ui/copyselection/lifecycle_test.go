package copyselection_test

import (
	"errors"
	"image"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/ui/copyselection"
)

func TestCopyLifecycle(t *testing.T) {
	var copied []image.Rectangle
	feature, selectionCanvas := newFeatureCanvas(t, sampleView, copyselection.Callbacks{
		Copy: func(bounds image.Rectangle) { copied = append(copied, bounds) },
	})
	commitSampleSelection(t, selectionCanvas)
	feature.HandleKey(fyne.KeyReturn)

	if got := feature.State(); !got.Busy || !got.HasSelection {
		t.Fatalf("State() after copy request = %+v, want busy with a selection", got)
	}

	feature.Complete(errors.New("clipboard full"))
	if got := feature.State(); got != (copyselection.State{Active: true, HasSelection: true}) {
		t.Fatalf("State() after copy failure = %+v, want idle with the selection retained", got)
	}

	feature.HandleKey(fyne.KeyReturn)
	feature.Complete(nil)
	if got := feature.State(); got != (copyselection.State{}) {
		t.Fatalf("State() after successful copy = %+v, want inactive", got)
	}
	if len(copied) != 2 {
		t.Fatalf("copy requests = %d, want 2 (retry after failure)", len(copied))
	}
}

func TestScrollForwarding(t *testing.T) {
	var scrolled int
	feature, _ := newFeatureCanvas(t, sampleView, copyselection.Callbacks{
		Scroll: func(*fyne.ScrollEvent) { scrolled++ },
	})
	input, ok := overlayInput(t, feature).(fyne.Scrollable)
	if !ok {
		t.Fatal("overlay input is not fyne.Scrollable")
	}

	input.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.NewDelta(0, -10)})
	if scrolled != 1 {
		t.Fatalf("scroll forwards = %d, want 1", scrolled)
	}
}
