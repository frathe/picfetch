package ui

import (
	"image"
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/uitest"
)

// stubKeyModifiers swaps v's keyModifiers func for the duration of the
// test - what Shift+R below needs, since a fyne.KeyEvent carries no
// modifier state and the test driver can't synthesize any (see
// defaultKeyModifiers in keys.go). A per-viewer field, so this is just a
// field write with no global to restore; kept as a helper for the
// call-site readability and the t.Helper symmetry with uitest.StubChooser.
func stubKeyModifiers(t *testing.T, v *viewer, mods fyne.KeyModifier) {
	t.Helper()

	v.keyModifiers = func() fyne.KeyModifier { return mods }
}

// --- rotateBy / resetRotation -----------------------------------------------

func TestRotateBy_NoImageIsNoOp(t *testing.T) {
	v := newTestViewer(t)

	v.rotateBy(1)

	if v.display.Rotation() != 0 {
		t.Errorf("rotation = %d, want 0 with no image loaded", v.display.Rotation())
	}
}

func TestRotateBy_SwapsBoundsAndResizesWindow(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 400, 200, color.White) // asymmetric dims
	dropAndWait(t, v, a)

	if b := v.img.Image.Bounds(); b.Dx() != 400 || b.Dy() != 200 {
		t.Fatalf("loaded bounds = %dx%d, want 400x200", b.Dx(), b.Dy())
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyR})

	if v.display.Rotation() != 1 {
		t.Errorf("rotation = %d, want 1 after one R press", v.display.Rotation())
	}
	if b := v.img.Image.Bounds(); b.Dx() != 200 || b.Dy() != 400 {
		t.Errorf("bounds after rotate = %dx%d, want 200x400 (swapped)", b.Dx(), b.Dy())
	}

	want := test.NewWindow(nil)
	defer want.Close()
	resizeToImage(want, image.Rect(0, 0, 200, 400), v.settings.maxWinW, v.settings.maxWinH)

	if got := v.win.Canvas().Size(); got != want.Canvas().Size() {
		t.Errorf("window size after rotate = %v, want %v (resizeToImage against the rotated 200x400 bounds)", got, want.Canvas().Size())
	}
}

func TestRotateBy_CounterClockwiseWraps(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 400, 200, color.White)
	dropAndWait(t, v, a)

	stubKeyModifiers(t, v, fyne.KeyModifierShift)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyR})

	if v.display.Rotation() != 3 {
		t.Errorf("rotation = %d, want 3 (one turn CCW) after Shift+R from 0", v.display.Rotation())
	}
}

func TestRotateBy_FourStepsReturnsToStart(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 400, 200, color.White)
	dropAndWait(t, v, a)

	for range 4 {
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyR})
	}

	if v.display.Rotation() != 0 {
		t.Errorf("rotation = %d, want 0 after four R presses", v.display.Rotation())
	}
	if b := v.img.Image.Bounds(); b.Dx() != 400 || b.Dy() != 200 {
		t.Errorf("bounds after four rotations = %dx%d, want back to 400x200", b.Dx(), b.Dy())
	}
}

func TestRotateBy_ResetsZoomFit(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 400, 200, color.White)
	dropAndWait(t, v, a)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.Key1}) // manual zoom
	if v.zoom.Fitting() {
		t.Fatal("expected manual zoom (fit off) before rotating")
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyR})

	if !v.zoom.Fitting() {
		t.Error("rotating should reset back to fit, same as a fresh navigation")
	}
}

func TestResetRotation_Key0ClearsRotationAndZoom(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 400, 200, color.White)
	dropAndWait(t, v, a)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyR})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.Key1})

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.Key0})

	if v.display.Rotation() != 0 {
		t.Errorf("rotation = %d, want 0 after the 0 key", v.display.Rotation())
	}
	if !v.zoom.Fitting() {
		t.Error("zoom should be back to fit after the 0 key")
	}
	if b := v.img.Image.Bounds(); b.Dx() != 400 || b.Dy() != 200 {
		t.Errorf("bounds after 0 = %dx%d, want back to the native 400x200", b.Dx(), b.Dy())
	}
}

func TestFinishLoad_ResetsRotationOnNavigation(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 400, 200, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 400, 200, color.Black)
	dropAndWait(t, v, a, b)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyR})
	if v.display.Rotation() == 0 {
		t.Fatal("expected a nonzero rotation before navigating")
	}

	v.ShowImage(1)
	waitUntilLoaded(t, v)

	if v.display.Rotation() != 0 {
		t.Errorf("rotation = %d, want reset to 0 on the next image, same as zoom", v.display.Rotation())
	}
}

// TestResetRotation_AfterClearToDropzoneDoesNotPanic pins a crash that was
// reachable in four keystrokes before the display cluster moved into
// internal/ui/display: open an image, R, Escape, 0.
//
// clearToDropzone used to null the frames but leave the rotation alone, and
// the 0 key is handled ahead of handleKeyEvent's navigation guard - by
// design, so it works with one file loaded or mid-decode. resetRotation
// therefore saw a non-zero rotation, decided there was work to do, and
// indexed into a nil frame slice. display.State.Clear zeroes the rotation
// with the frames, so ResetRotation now reports "nothing to do" and returns
// before the redraw. Do not "optimise" Clear to leave the rotation set.
func TestResetRotation_AfterClearToDropzoneDoesNotPanic(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 8, 4, color.White))

	v.rotateBy(1)
	if v.display.Rotation() == 0 {
		t.Fatal("precondition: rotateBy should have set a rotation")
	}

	v.clearToDropzone()

	if got := v.display.Rotation(); got != 0 {
		t.Errorf("rotation after clearToDropzone = %d, want 0 - a stale rotation makes the 0 key index a nil frame slice", got)
	}

	// The panic, if it comes back, lands here.
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.Key0})
}
