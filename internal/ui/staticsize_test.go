package ui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/uitest"
)

// Static window size: when the settings toggle is on, load / zoom / rotate
// must leave the main window alone. Manual resizes still flow through
// windowSizeTracker (covered in preferences_wiring_test.go).

func TestStaticWindowSize_LoadDoesNotResize(t *testing.T) {
	v := newTestViewer(t)
	v.SetStaticWindowSize(true)

	before := v.win.Canvas().Size()
	a := uitest.TempJPEGURI(t, "a.jpg", 800, 600, color.White)
	dropAndWait(t, v, a)

	if got := v.win.Canvas().Size(); got != before {
		t.Errorf("window after load = %v, want unchanged %v with static size on", got, before)
	}
}

func TestSyncWindowToZoom_StaticSizeDoesNotResize(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 800, 600, color.White)
	dropAndWait(t, v, a)

	v.SetStaticWindowSize(true)
	before := v.win.Canvas().Size()

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})

	if got := v.win.Canvas().Size(); got != before {
		t.Errorf("window after zoom-in = %v, want unchanged %v with static size on", got, before)
	}
}

func TestRotateBy_StaticSizeDoesNotResize(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 400, 200, color.White)
	dropAndWait(t, v, a)

	v.SetStaticWindowSize(true)
	before := v.win.Canvas().Size()

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyR})

	if v.display.Rotation() != 1 {
		t.Fatalf("rotation = %d, want 1 after R", v.display.Rotation())
	}
	if got := v.win.Canvas().Size(); got != before {
		t.Errorf("window after rotate = %v, want unchanged %v with static size on", got, before)
	}
}

func TestStaticWindowSize_ResetKeepsWindowSize(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 800, 600, color.White)
	dropAndWait(t, v, a)

	v.SetStaticWindowSize(true)
	before := v.win.Canvas().Size()

	v.reset()

	if got := v.win.Canvas().Size(); got != before {
		t.Errorf("window after Escape/reset = %v, want unchanged %v with static size on", got, before)
	}
	if !v.dropzone.Visible() {
		t.Error("dropzone should be visible again after reset")
	}
}
