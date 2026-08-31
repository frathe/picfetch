package ui

import (
	"errors"
	"image"
	"image/color"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/ui/copyselection"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestCopySelectionBusy(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, regionCopyPNGURI(t, "busy.png", markedRegionCopyImage(8, 6)))

	started := make(chan struct{})
	release := make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() { releaseOnce.Do(func() { close(release) }) })
	calls := 0
	uitest.StubClipboardCopy(t, func([]byte) error {
		calls++
		close(started)
		<-release
		return nil
	})

	selectRegion(t, v, image.Rect(1, 1, 6, 5))
	v.regionCopy.HandleKey(fyne.KeyReturn)
	select {
	case <-started:
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for Copy Selection clipboard dispatch")
	}
	if got := v.regionCopy.State(); got != (copySelectionBusyState()) {
		t.Fatalf("state while worker is pending = %+v, want active, busy, and selected", got)
	}

	// Busy mode consumes editing, cancellation, and another copy request.
	selectRegionDrag(t, v, image.Rect(0, 0, 2, 2))
	v.regionCopy.HandleKey(fyne.KeyEscape)
	v.regionCopy.HandleKey(fyne.KeyReturn)
	if calls != 1 {
		t.Fatalf("clipboard calls while busy = %d, want 1", calls)
	}
	if got := v.regionCopy.State(); got != (copySelectionBusyState()) {
		t.Fatalf("state after input while busy = %+v, want unchanged", got)
	}

	releaseOnce.Do(func() { close(release) })
	waitForClipboard(t, v)
	if got := v.regionCopy.State(); got.Active {
		t.Fatalf("state after successful worker = %+v, want inactive", got)
	}
}

func TestCopySelectionSuccess(t *testing.T) {
	v := newTestViewer(t)
	original := markedRegionCopyImage(10, 8)
	dropAndWait(t, v, regionCopyPNGURI(t, "stable.png", original))
	v.toggleInfoOverlay()

	var copied []byte
	uitest.StubClipboardCopy(t, func(data []byte) error {
		copied = append([]byte(nil), data...)
		return nil
	})

	selection := image.Rect(2, 2, 8, 6)
	selectRegion(t, v, selection)
	// Replace the live display after activation. Copy Selection must still
	// use the source presentation captured at entry.
	v.display.ReplaceCurrent(uniformRegionCopyImage(10, 8, color.NRGBA{G: 255, A: 255}))
	v.redrawRotatedFrame()
	v.regionCopy.HandleKey(fyne.KeyReturn)
	waitForClipboard(t, v)

	got := decodeRegionCopyPNG(t, copied)
	for y := range selection.Dy() {
		for x := range selection.Dx() {
			want := original.NRGBAAt(selection.Min.X+x, selection.Min.Y+y)
			if pixel := color.NRGBAModel.Convert(got.At(x, y)).(color.NRGBA); pixel != want {
				t.Fatalf("stable source pixel (%d,%d) = %#v, want %#v", x, y, pixel, want)
			}
		}
	}
	if v.regionCopy.State().Active || v.regionCopy.Overlay().Visible() {
		t.Fatal("successful copy did not finish Copy Selection mode")
	}
	if !v.info.Object().Visible() {
		t.Fatal("successful copy did not restore the information overlay")
	}
	if v.toast.card.Visible() {
		t.Fatal("successful copy showed an error toast")
	}
	if _, err := v.regionCopy.Encode(selection); err == nil {
		t.Fatal("successful copy retained its captured source")
	}
}

func TestCopySelectionEncodeFailure(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, regionCopyPNGURI(t, "failure.png", markedRegionCopyImage(10, 8)))

	dispatches := 0
	uitest.StubClipboardCopy(t, func([]byte) error {
		dispatches++
		return nil
	})

	selectRegion(t, v, image.Rect(2, 2, 8, 6))
	v.regionCopy.SetEncode(func(image.Image, image.Rectangle) ([]byte, error) {
		return nil, errors.New("synthetic PNG failure")
	})
	v.regionCopy.HandleKey(fyne.KeyReturn)
	waitForClipboard(t, v)

	assertRecoverableRegionCopyFailure(t, v)
	if dispatches != 0 {
		t.Fatalf("clipboard dispatches after encode failure = %d, want 0", dispatches)
	}
	settleToast(t, v)

	v.regionCopy.SetEncode(nil)
	v.regionCopy.HandleKey(fyne.KeyReturn)
	waitForClipboard(t, v)
	if dispatches != 1 || v.regionCopy.State().Active {
		t.Fatalf("retry = {dispatches:%d state:%+v}, want one dispatch and inactive", dispatches, v.regionCopy.State())
	}
}

func TestCopySelectionClipboardFailure(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, regionCopyPNGURI(t, "clipboard.png", markedRegionCopyImage(10, 8)))

	attempts := 0
	uitest.StubClipboardCopy(t, func([]byte) error {
		attempts++
		if attempts == 1 {
			return errors.New("synthetic clipboard failure")
		}
		return nil
	})

	selectRegion(t, v, image.Rect(2, 2, 8, 6))
	v.regionCopy.HandleKey(fyne.KeyReturn)
	waitForClipboard(t, v)
	assertRecoverableRegionCopyFailure(t, v)
	settleToast(t, v)

	v.regionCopy.HandleKey(fyne.KeyReturn)
	waitForClipboard(t, v)
	if attempts != 2 || v.regionCopy.State().Active {
		t.Fatalf("clipboard retry = {attempts:%d state:%+v}, want two attempts and inactive", attempts, v.regionCopy.State())
	}
}

func assertRecoverableRegionCopyFailure(t *testing.T, v *viewer) {
	t.Helper()
	state := v.regionCopy.State()
	if !state.Active || state.Busy || !state.HasSelection {
		t.Fatalf("state after recoverable failure = %+v, want active, idle, and selected", state)
	}
	if !v.regionCopy.Overlay().Visible() {
		t.Fatal("recoverable failure hid the selection overlay")
	}
	if !v.toast.card.Visible() {
		t.Fatal("recoverable failure did not show the clipboard-style error toast")
	}
}

func selectRegionDrag(t *testing.T, v *viewer, bounds image.Rectangle) {
	t.Helper()
	geometry := v.zoom.Geometry()
	w, h := v.displayedDimensions()
	toCanvas := func(x, y float32) fyne.Position {
		return fyne.NewPos(
			geometry.Position.X+x*geometry.Size.Width/float32(w),
			geometry.Position.Y+y*geometry.Size.Height/float32(h),
		)
	}
	start := toCanvas(float32(bounds.Min.X)+0.75, float32(bounds.Min.Y)+0.75)
	end := toCanvas(float32(bounds.Max.X)-0.25, float32(bounds.Max.Y)-0.25)
	test.Drag(v.win.Canvas(), end, end.X-start.X, end.Y-start.Y)
}

func uniformRegionCopyImage(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func copySelectionBusyState() copyselection.State {
	return copyselection.State{Active: true, Busy: true, HasSelection: true}
}
