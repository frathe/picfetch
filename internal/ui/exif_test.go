package ui

import (
	"image/color"
	"os"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/widgets"
	"github.com/frathe/picfetch/internal/uitest"
)

// --- EXIF panel wiring ------------------------------------------------
//
// The panel itself lives in internal/ui/exifwin; these cover the viewer's
// side of it - the E key, the info-overlay link, and staying in sync as
// navigation changes which file is current.

// TestShowExifWindow_NoopWithNothingLoaded guards against E (or the info
// overlay link, before it's even reachable - see
// TestToggleInfoOverlay_HiddenUntilAnImageIsLoaded for the same idea applied
// to the info card itself) trying to open a window with no file to read
// metadata from.
func TestShowExifWindow_NoopWithNothingLoaded(t *testing.T) {
	v, _, _ := newTestUI(t)

	v.exif.Show()

	if v.exif.Open() {
		t.Error("showExifWindow should no-op with nothing loaded")
	}
}

// TestShowExifWindow_OpensAndRaisesSameWindow mirrors
// TestShowAbout_OpensAndRaisesSameWindow (help/about_test.go): a plain
// widget.Label content, like About's single heading, so it doesn't hit the
// test theme's font-combination limits the way manual.md's RichText does
// (see the F1/showManual note at the end of e2e_test.go) and can be
// exercised directly.
func TestShowExifWindow_OpensAndRaisesSameWindow(t *testing.T) {
	v, _, _ := newTestUI(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White)
	dropAndWait(t, v, a)

	v.exif.Show()

	win := v.exif.Window()
	if win == nil {
		t.Fatal("showExifWindow did not open a window")
	}

	v.exif.Show()

	if v.exif.Window() != win {
		t.Error("a second showExifWindow call should raise the existing window, not open a new one")
	}

	win.Close()

	if v.exif.Open() {
		t.Error("closing the EXIF window should leave the singleton closed")
	}
}

// TestShowExifWindow_ContentAndRefreshOnNavigation checks the window shows
// the current file's metadata (uitest.EncodeJPEG embeds none, so the
// no-metadata message - imaging.ReadMetadata's own behavior is covered
// directly in internal/imaging) and, per exifwin.Window.Refresh's comment,
// keeps itself current across navigation while still open, mirroring how the
// info overlay behaves (TestToggleInfoOverlay_ContentAndPersistenceAcrossNavigation).
func TestShowExifWindow_ContentAndRefreshOnNavigation(t *testing.T) {
	v, _, _ := newTestUI(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 40, 20, color.White)
	dropAndWait(t, v, a, b)

	v.exif.Show()

	want := "No EXIF metadata found in this file."
	if got := v.exif.Text().Text; got != want {
		t.Errorf("exifText = %q, want %q", got, want)
	}

	v.ShowImage(v.state.index + 1)
	waitUntilLoaded(t, v)

	if got := v.exif.Text().Text; got != want {
		t.Errorf("exifText after navigating should stay in sync, got %q, want %q", got, want)
	}
}

func TestExifWindow_LeftRightChangeImage(t *testing.T) {
	v, _, _ := newTestUI(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 40, 20, color.White)
	dropAndWait(t, v, a, b)

	v.exif.Show()
	start := v.state.index
	canvas := v.exif.Window().Canvas()
	if got := canvas.Focused(); got != nil {
		t.Fatalf("EXIF canvas focused %T after Show, want nil", got)
	}
	handler := canvas.OnTypedKey()
	if handler == nil {
		t.Fatal("EXIF window has no OnTypedKey handler")
	}
	handler(&fyne.KeyEvent{Name: fyne.KeyRight})
	waitUntilLoaded(t, v)
	if v.state.index != (start+1)%2 {
		t.Fatalf("index = %d, want next file", v.state.index)
	}
	handler(&fyne.KeyEvent{Name: fyne.KeyLeft})
	waitUntilLoaded(t, v)
	if v.state.index != start {
		t.Fatalf("index = %d, want start %d", v.state.index, start)
	}
}

// TestHandleKeyEvent_EOpensExifWindow checks the E keybinding reaches
// v.exif.Show, mirroring how the I/M/P keys are each tested against
// their own handler.
func TestHandleKeyEvent_EOpensExifWindow(t *testing.T) {
	v, _, _ := newTestUI(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White)
	dropAndWait(t, v, a)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyE})

	if !v.exif.Open() {
		t.Error("E should open the EXIF window")
	}
}

// TestExifLink_OpensExifWindow checks the info overlay's "Show EXIF data"
// link (build.go's info-card wiring) reaches v.exif.Show, mirroring how
// e2e_test.go drives restoreLink's own OnTapped directly rather than a real
// simulated click.
func TestExifLink_OpensExifWindow(t *testing.T) {
	v, _, _ := newTestUI(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White)
	dropAndWait(t, v, a)

	v.info.ExifLink().OnTapped()

	if !v.exif.Open() {
		t.Error("the info overlay's EXIF link should open the EXIF window")
	}
}

// TestExifLink_HiddenWhenTheImageHasNoExifData covers the other half of the
// link's job: offering "Show EXIF data" for a file that has none is a
// promise the panel can only answer with "No EXIF metadata found in this
// file." uitest.EncodeJPEG embeds no metadata at all, so a plain test JPEG
// is exactly that case.
func TestExifLink_HiddenWhenTheImageHasNoExifData(t *testing.T) {
	v, _, _ := newTestUI(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White)
	dropAndWait(t, v, a)

	v.toggleInfoOverlay()

	if !v.info.Object().Visible() {
		t.Fatal("the info card itself should be showing")
	}
	if v.info.ExifLink().Visible() {
		t.Error("the EXIF link should be hidden for a file with no EXIF metadata")
	}
}

func TestExifLink_ShownWhenTheImageHasExifData(t *testing.T) {
	v, _, _ := newTestUI(t)

	withExif := uitest.WriteTempFile(t, "dated.jpg", uitest.CaptureDateJPEG(t, 40, 20, "2024:08:12 14:33:02"))
	dropAndWait(t, v, storage.NewFileURI(withExif))

	v.toggleInfoOverlay()

	if !v.info.ExifLink().Visible() {
		t.Error("the EXIF link should be shown for a file that has EXIF metadata")
	}
}

// The link has to follow the file on screen, not just whatever was loaded
// when the overlay was switched on.
func TestExifLink_VisibilityFollowsNavigation(t *testing.T) {
	v, _, _ := newTestUI(t)

	withExif := uitest.WriteTempFile(t, "dated.jpg", uitest.CaptureDateJPEG(t, 40, 20, "2024:08:12 14:33:02"))
	plain := uitest.TempJPEGURI(t, "plain.jpg", 40, 20, color.White)
	dropAndWait(t, v, storage.NewFileURI(withExif), plain)

	v.toggleInfoOverlay()
	if !v.info.ExifLink().Visible() {
		t.Fatal("the EXIF link should be shown for the first file, which has EXIF metadata")
	}

	v.ShowImage(v.state.index + 1)
	waitUntilLoaded(t, v)

	if v.info.ExifLink().Visible() {
		t.Error("the EXIF link should hide again after navigating to a file with no EXIF metadata")
	}

	v.ShowImage(v.state.index - 1)
	waitUntilLoaded(t, v)

	if !v.info.ExifLink().Visible() {
		t.Error("the EXIF link should come back when navigating to the file that does have EXIF metadata")
	}
}

// --- Remove Metadata ---------------------------------------------------
//
// The button itself lives in exifwin (exifwin_test.go, confirm_test.go);
// these cover the viewer's side of a successful strip - AfterMetadataRemoved
// hiding the EXIF link, refreshing the info card's FileSize/HasEXIF facts,
// and evicting the decode cache so a later visit can't revive stale state.

// TestStripMetadata_HidesExifLinkAndShrinksReportedSize drives a strip
// end-to-end through the same keyboard path a user would: open the panel,
// tap Remove Metadata, then answer the "Remove Metadata?" confirmation with
// Right+Return (the confirming choice, per exifwin's cancelChoice/
// confirmChoice - see confirm.go), exactly as
// TestShowConfirmGivesTheKeyboardToItsPanelStartingOnCancel expects the
// panel to start on Cancel.
func TestStripMetadata_HidesExifLinkAndShrinksReportedSize(t *testing.T) {
	v, _, _ := newTestUI(t)

	u := uitest.TempGPSJPEGURI(t, "gps.jpg", 40, 20, 48.858222, 2.2945)
	dropAndWait(t, v, u)

	beforeSize := v.info.FileSize()
	v.toggleInfoOverlay()
	if !v.info.ExifLink().Visible() {
		t.Fatal("setup: the EXIF link should be shown")
	}

	v.exif.Show()
	v.exif.StripButton().OnTapped()
	panel, ok := v.exif.Window().Canvas().Focused().(*widgets.ChoicePanel)
	if !ok {
		t.Fatalf("focused = %v, want the confirmation's choice panel", v.exif.Window().Canvas().Focused())
	}
	panel.TypedKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	panel.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})

	settleToast(t, v)

	if v.info.HasEXIF() {
		t.Fatal("HasEXIF still true")
	}
	if v.info.ExifLink().Visible() {
		t.Fatal("EXIF link should hide")
	}

	// GPS APP1 is only a few hundred bytes, so the stripped file should
	// come back strictly smaller; if some platform's re-encode ever
	// leaves the reported size unchanged, ReadMetadata coming back empty
	// still proves the strip did its job - see the brief's own note on
	// this being the fallback assertion.
	if v.info.FileSize() <= 0 || v.info.FileSize() >= beforeSize {
		data, err := os.ReadFile(u.Path())
		if err != nil {
			t.Fatalf("file size %d, want smaller than %d; also failed to re-read file: %v", v.info.FileSize(), beforeSize, err)
		}
		m := imaging.ReadMetadata(data)
		if !m.Empty() {
			t.Fatalf("file size %d, want smaller than %d, and metadata is still present: %+v", v.info.FileSize(), beforeSize, m)
		}
		info, err := os.Stat(u.Path())
		if err != nil {
			t.Fatalf("could not Stat the stripped file: %v", err)
		}
		if v.info.FileSize() != info.Size() {
			t.Errorf("FileSize() = %d, want it to match the file's actual size %d after the strip", v.info.FileSize(), info.Size())
		}
	}

	if v.imgCache.Contains(u.String()) {
		t.Fatal("imgCache should have been evicted")
	}
}
