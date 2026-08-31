package ui

import (
	"image"
	"image/color"
	"testing"

	"fyne.io/fyne/v2"

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
