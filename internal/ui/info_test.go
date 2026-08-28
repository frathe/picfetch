package ui

import (
	"fmt"
	"image/color"
	"os"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/uitest"
)

// This file owns the info overlay that the I key toggles
// (internal/ui/info.go): its content - filename and position, pixel
// dimensions, file size, zoom level - that it stays current across a
// navigation instead of freezing on whatever the first image showed, and
// that the zoom line tracks every zoom mutator (ActualSize, In,
// FitToWindow), not just the value at load time. The I preference itself
// is a standing one, like naturalSort/mergeMode:
// TestClearToDropzone_HidesInfoCardButKeepsThePreference checks it
// survives a reset even though the card itself is one of the things that
// reset hides. The pure byte-count formatting helper underneath the
// card's text (formatFileSize) now lives in internal/ui/infoview, with its
// own boundary tests there; wantSizeText below mirrors it just to build
// this file's expected strings from a real file's on-disk size.

// wantSizeText mirrors infoview's own (unexported, already
// boundary-tested) formatFileSize, purely so the tests below can build an
// expected info-card string from a real file's actual on-disk size.
func wantSizeText(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}

	div, exp := int64(unit), 0
	for m := n / unit; m >= unit; m /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(n)/float64(div), "KMGTPE"[exp])
}

// TestToggleInfoOverlay_HiddenUntilAnImageIsLoaded guards against I turning
// the card on before there's anything for it to describe: pressed before
// the first drop (allowed, like M/P - see handleKeyEvent), the preference
// should be recorded but the card must stay hidden until an image actually
// loads.
func TestToggleInfoOverlay_HiddenUntilAnImageIsLoaded(t *testing.T) {
	v := newTestViewer(t)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyI})
	if !v.info.Visible() {
		t.Fatal("the info preference should flip on right away, even with nothing loaded")
	}
	if v.info.Object().Visible() {
		t.Fatal("infoCard should stay hidden until an image is actually on screen")
	}

	a := uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White)
	dropAndWait(t, v, a)

	if !v.info.Object().Visible() {
		t.Error("infoCard should appear once the first image loads, since the toggle was already on")
	}
}

// TestToggleInfoOverlay_ContentAndPersistenceAcrossNavigation covers the
// card's actual content (filename+position, pixel dimensions, file size,
// zoom) and that it keeps itself current across a navigation instead of
// freezing on whatever the first image showed.
func TestToggleInfoOverlay_ContentAndPersistenceAcrossNavigation(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 80, 10, color.White)
	dropAndWait(t, v, a, b)

	v.toggleInfoOverlay()
	if !v.info.Object().Visible() {
		t.Fatal("infoCard should be visible right after toggling on with an image already loaded")
	}

	aInfo, err := os.Stat(a.Path())
	if err != nil {
		t.Fatalf("stat a.jpg: %v", err)
	}
	// The zoom line's own value is whatever fit scale the test window's
	// size works out to, so it's read back rather than pinned: what this
	// test is about is the card's content and that it stays current. The
	// fit math itself is internal/ui/zoom's to test, against a viewport it
	// can actually control.
	want := fmt.Sprintf("a.jpg  (1/2)\n40 x 20\n%s\nZoom: %d%%", wantSizeText(aInfo.Size()), v.zoom.Percent())
	if got := v.info.Text().Text; got != want {
		t.Errorf("infoText = %q, want %q", got, want)
	}

	// Step to the second file: the card must refresh, not keep showing a's
	// info.
	v.ShowImage(v.state.index + 1)
	waitUntilLoaded(t, v)
	v.updateInfoOverlay()

	bInfo, err := os.Stat(b.Path())
	if err != nil {
		t.Fatalf("stat b.jpg: %v", err)
	}
	want = fmt.Sprintf("b.jpg  (2/2)\n80 x 10\n%s\nZoom: %d%%", wantSizeText(bInfo.Size()), v.zoom.Percent())
	if got := v.info.Text().Text; got != want {
		t.Errorf("infoText after navigating = %q, want %q", got, want)
	}

	// Toggling off hides it; toggling back on immediately re-shows current info.
	v.toggleInfoOverlay()
	if v.info.Object().Visible() {
		t.Fatal("infoCard should hide once toggled off")
	}
	v.toggleInfoOverlay()
	if !v.info.Object().Visible() {
		t.Fatal("infoCard should reappear once toggled back on")
	}
	if got := v.info.Text().Text; got != want {
		t.Errorf("infoText after re-enabling = %q, want %q (still on b.jpg)", got, want)
	}
}

// TestToggleInfoOverlay_ZoomLineTracksZoomChanges checks the last line
// updates with every zoom mutator (ActualSize, In, FitToWindow), not just
// at load time.
func TestToggleInfoOverlay_ZoomLineTracksZoomChanges(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 400, 200, color.White)
	dropAndWait(t, v, a)

	// The fit percentage depends on the test window's size, so it's read
	// back once here and used as the anchor the last step returns to.
	// Actual size (100%) is the one value that's the same everywhere.
	fitPct := fmt.Sprintf("Zoom: %d%%", v.zoom.Percent())

	v.toggleInfoOverlay()

	if !strings.HasSuffix(v.info.Text().Text, fitPct) {
		t.Errorf("infoText = %q, want it to end with the %q fit scale", v.info.Text().Text, fitPct)
	}

	v.zoom.ActualSize()
	if !strings.HasSuffix(v.info.Text().Text, "Zoom: 100%") {
		t.Errorf("infoText after ActualSize = %q, want it to end with 100%%", v.info.Text().Text)
	}

	v.zoom.In()
	if v.zoom.Percent() <= 100 {
		t.Fatalf("setup: zoom percent after In = %d, want more than 100", v.zoom.Percent())
	}
	want := fmt.Sprintf("Zoom: %d%%", v.zoom.Percent())
	if !strings.HasSuffix(v.info.Text().Text, want) {
		t.Errorf("infoText after In = %q, want it to end with %q", v.info.Text().Text, want)
	}

	v.zoom.FitToWindow()
	if !strings.HasSuffix(v.info.Text().Text, fitPct) {
		t.Errorf("infoText after FitToWindow = %q, want back to %q", v.info.Text().Text, fitPct)
	}
}

// TestClearToDropzone_HidesInfoCardButKeepsThePreference guards the reset
// (Escape) path: the card must disappear along with the image, but the I
// preference itself is a standing one - like naturalSort/mergeMode - so a
// fresh drop afterward should bring the card straight back.
func TestClearToDropzone_HidesInfoCardButKeepsThePreference(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White)
	dropAndWait(t, v, a)
	v.toggleInfoOverlay()

	v.reset()

	if !v.info.Visible() {
		t.Error("the info preference should survive a reset")
	}
	if v.info.Object().Visible() {
		t.Error("infoCard should be hidden once reset back to the empty drop zone")
	}

	b := uitest.TempJPEGURI(t, "b.jpg", 40, 20, color.White)
	dropAndWait(t, v, b)

	if !v.info.Object().Visible() {
		t.Error("infoCard should reappear on the next load since the preference was still on")
	}
}

func TestToggleInfoOverlay_RAWMarksPreview(t *testing.T) {
	v := newTestViewer(t)

	raw := uitest.TempRAWURI(t, "photo.nef", 16, 8, color.White)
	dropAndWait(t, v, raw)
	v.toggleInfoOverlay()

	if !strings.Contains(v.info.Text().Text, lang.L("(preview)")) {
		t.Errorf("infoText = %q, want %q on a RAW preview", v.info.Text().Text, lang.L("(preview)"))
	}
	if !strings.Contains(v.info.Text().Text, "photo.nef") {
		t.Errorf("infoText = %q, want the RAW filename", v.info.Text().Text)
	}
}
