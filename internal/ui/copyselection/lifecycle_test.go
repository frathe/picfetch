package copyselection_test

import (
	"errors"
	"image"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/ui/copyselection"
)

func TestHandleKey_OwnsModeKeys(t *testing.T) {
	t.Run("idle mode consumes its own keys", func(t *testing.T) {
		for _, key := range []fyne.KeyName{
			fyne.KeyReturn, fyne.KeyEnter,
			fyne.KeyLeft, fyne.KeyRight, fyne.KeyUp, fyne.KeyDown, fyne.KeyHome, fyne.KeyEnd,
		} {
			feature, _ := newFeatureCanvas(t, sampleView, copyselection.Callbacks{})
			if !feature.HandleKey(key) {
				t.Fatalf("HandleKey(%s) = false, want the active mode to consume it", key)
			}
			if !feature.State().Active {
				t.Fatalf("HandleKey(%s) cancelled idle mode", key)
			}
		}
	})

	t.Run("Escape cancels idle mode", func(t *testing.T) {
		feature, _ := newFeatureCanvas(t, sampleView, copyselection.Callbacks{})
		if !feature.HandleKey(fyne.KeyEscape) {
			t.Fatal("HandleKey(Escape) = false, want the active mode to consume it")
		}
		if feature.State().Active {
			t.Fatal("HandleKey(Escape) left the mode active")
		}
	})

	t.Run("busy mode consumes every key", func(t *testing.T) {
		feature, selectionCanvas := newFeatureCanvas(t, sampleView, copyselection.Callbacks{
			Copy: func(image.Rectangle) {},
		})
		commitSampleSelection(t, selectionCanvas)
		feature.HandleKey(fyne.KeyReturn)
		if !feature.State().Busy {
			t.Fatal("HandleKey(Return) after a selection did not enter busy mode")
		}
		if !feature.HandleKey(fyne.KeyG) {
			t.Fatal("HandleKey(G) while busy = false, want busy mode to consume every key")
		}
		if !feature.State().Busy {
			t.Fatal("HandleKey(G) while busy ended the copy")
		}
	})

	t.Run("unowned keys stay with the viewer", func(t *testing.T) {
		feature, _ := newFeatureCanvas(t, sampleView, copyselection.Callbacks{})
		for _, key := range []fyne.KeyName{fyne.KeyG, fyne.KeyR, fyne.KeyPlus, fyne.KeyF1, fyne.KeyC} {
			if feature.HandleKey(key) {
				t.Fatalf("HandleKey(%s) = true, want unowned keys left to the viewer", key)
			}
			if !feature.State().Active {
				t.Fatalf("HandleKey(%s) cancelled the mode", key)
			}
		}
	})
}

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
