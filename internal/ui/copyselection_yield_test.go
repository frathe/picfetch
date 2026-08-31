package ui

import (
	"image"
	"image/color"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/uitest"
)

// The EXIF window's arrow keys call StepImage directly, bypassing the key
// dispatcher's yield. The chokepoint yield in ShowImage must cancel an idle
// selection there too, instead of swapping the image under it.
func TestStepImageYieldsIdleCopySelection(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 40, 20, color.White)
	dropAndWait(t, v, a, b)
	start := v.CurrentIndex()

	v.startRegionCopy()
	if !v.regionCopy.State().Active {
		t.Fatal("precondition: Copy Selection did not start")
	}

	v.StepImage(1) // what exifwin's Left/Right handler calls
	waitUntilLoaded(t, v)

	if v.regionCopy.State().Active {
		t.Error("navigation left Copy Selection active over a different image")
	}
	if v.CurrentIndex() == start {
		t.Error("yielded navigation did not advance the image")
	}
}

// A pending copy blocks navigation instead of being cancelled, and the
// refusal is visible: the shared yield shows a toast rather than silently
// dropping the command.
func TestStepImageBlockedWhileCopyPending(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v,
		regionCopyPNGURI(t, "a.png", markedRegionCopyImage(10, 8)),
		regionCopyPNGURI(t, "b.png", markedRegionCopyImage(10, 8)))
	start := v.CurrentIndex()

	release := make(chan struct{})
	uitest.StubClipboardCopy(t, func([]byte) error { <-release; return nil })
	selectRegion(t, v, image.Rect(2, 2, 8, 6))
	v.regionCopy.HandleKey(fyne.KeyReturn)
	if !v.regionCopy.State().Busy {
		t.Fatal("precondition: no copy is pending")
	}

	v.StepImage(1)

	if v.CurrentIndex() != start {
		t.Errorf("navigation during a pending copy moved the image: index %d, want %d", v.CurrentIndex(), start)
	}
	if !v.toast.card.Visible() {
		t.Error("blocked navigation gave no feedback toast")
	}

	close(release)
	waitForClipboard(t, v)
	settleToast(t, v)
}

// An OS drop during a pending copy is refused early (before any state is
// touched) — but no longer silently.
func TestHandleDropBlockedWhileCopyPendingShowsToast(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, regionCopyPNGURI(t, "photo.png", markedRegionCopyImage(10, 8)))

	release := make(chan struct{})
	uitest.StubClipboardCopy(t, func([]byte) error { <-release; return nil })
	selectRegion(t, v, image.Rect(2, 2, 8, 6))
	v.regionCopy.HandleKey(fyne.KeyReturn)
	if !v.regionCopy.State().Busy {
		t.Fatal("precondition: no copy is pending")
	}

	v.handleDrop([]fyne.URI{uitest.TempJPEGURI(t, "late.jpg", 8, 8, color.White)})

	if v.FileCount() != 1 {
		t.Errorf("blocked drop changed the file set: count = %d, want 1", v.FileCount())
	}
	if !v.toast.card.Visible() {
		t.Error("blocked drop gave no feedback toast")
	}

	close(release)
	waitForClipboard(t, v)
	settleToast(t, v)
}

// 0 is not a pure zoom key: its handler also clears view rotation. Under
// an active selection that would change orientation while the captured
// bounds stay axis-swapped, so 0 must yield the mode the same way R does.
func TestKey0YieldsCopySelectionAndResetsRotation(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))

	v.rotateBy(1)
	if v.display.Rotation() == 0 {
		t.Fatal("precondition: rotation did not apply")
	}

	v.startRegionCopy()
	if !v.regionCopy.State().Active {
		t.Fatal("precondition: Copy Selection did not start")
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.Key0})

	if v.regionCopy.State().Active {
		t.Error("0 changed orientation but left Copy Selection active")
	}
	if v.display.Rotation() != 0 {
		t.Errorf("rotation after 0 = %d, want 0", v.display.Rotation())
	}
}

// The Favorites menu's own items must yield like every other menu command;
// they were the gap in the hand-wrapped menu callbacks.
func TestFavoritesMenuYieldsCopySelection(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))

	v.startRegionCopy()
	if !v.regionCopy.State().Active {
		t.Fatal("precondition: Copy Selection did not start")
	}

	// Items[0] is "Add Current List to Favorites…" — see favorites.New.
	v.favorites.Menu().Items[0].Action()

	if v.regionCopy.State().Active {
		t.Error("the Favorites menu ran a command without yielding Copy Selection")
	}
}

// A capture that produces no frame must release the animation pause it
// acquired: the caller only cleans up after a failed Start, not a failed
// capture, and a held pause freezes the animation loop for the session.
func TestCaptureRegionCopySourceReleasesPauseWhenCaptureFails(t *testing.T) {
	v := newTestViewer(t)
	path := uitest.WriteTempFile(t, "anim.gif", uitest.EncodeAnimatedGIF(t, 4, 3,
		[]color.Color{color.NRGBA{R: 255, A: 255}, color.NRGBA{B: 255, A: 255}},
		[]int{1000, 1000}))
	dropAndWait(t, v, storage.NewFileURI(path))
	if v.display.Count() < 2 {
		t.Fatal("precondition: the dropped GIF is not animated")
	}

	v.img.Image = nil // simulate the display layer handing back no frame
	_, animated, ok := v.captureRegionCopySource()
	if ok || animated {
		t.Fatalf("capture with no frame = (animated=%v, ok=%v), want (false, false)", animated, ok)
	}

	if !v.animationPause.pause(func() {}) {
		t.Fatal("failed capture left the animation pause held")
	}
	v.animationPause.unpause()
}

// Zoom geometry changes fire for the app's whole lifetime; while Copy
// Selection is inactive they must not queue UI work at all.
func TestZoomGeometryCallbackSkipsInactiveMode(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 800, 400, color.White))

	dispatches := 0
	v.regionCopyDo = func(f func()) {
		dispatches++
		f()
	}

	test.Scroll(v.win.Canvas(), fyne.NewPos(100, 100), 0, 20)

	if dispatches != 0 {
		t.Fatalf("geometry dispatches while Copy Selection is inactive = %d, want 0", dispatches)
	}
}
