// The export prompt's extra rows (exportoptions.go): the export size limit
// and the metadata checkbox, the keyboard story over them, and what each
// one changes about the file that reaches disk.
//
// Everything here asserts on what was written - the pixel dimensions read
// back off the file, the tags read back out of it, the name the save panel
// was offered, the text of the toast - never on which helper the export
// path called on the way there.

package ui

import (
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

// --- defaults --------------------------------------------------------------

func TestExportOptions_DefaultToOriginalSizeAndFullMetadata(t *testing.T) {
	v := newTestViewer(t)

	if got := v.exportOptions.Options(); got != (imaging.ExportOptions{}) {
		t.Errorf("Options() = %+v, want the zero value - today's behaviour", got)
	}
}

// TestPromptExport_OpensAtDefaultsAfterANonDefaultExport is user story 11
// and 12 in one: options are not persisted and not remembered within a
// session, so the prompt can never silently shrink a photo months later.
func TestPromptExport_OpensAtDefaultsAfterANonDefaultExport(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 40, 30, color.White))

	v.promptExport()
	selectExportRung(t, v, 1000)
	v.exportOptions.setMetadataIncluded(false)
	if want := (imaging.ExportOptions{MaxEdge: 1000, OmitMetadata: true}); v.exportOptions.Options() != want {
		t.Fatal("setup: both options should be off their defaults - the premise of this test")
	}
	v.exportPrompt.Hide()

	v.promptExport()

	if got := v.exportOptions.Options(); got != (imaging.ExportOptions{}) {
		t.Errorf("Options() = %+v on re-opening, want the defaults back", got)
	}
}

// --- the Original rung's label ---------------------------------------------

// TestExportOptions_OriginalLabelStatesTheFramesLongestEdge is what lets
// someone tell at a glance whether a smaller rung would change anything.
func TestExportOptions_OriginalLabelStatesTheFramesLongestEdge(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "wide.jpg", 90, 30, color.White))

	v.promptExport()

	if got := originalRungLabel(t, v); !strings.Contains(got, "90") {
		t.Errorf("the Original rung reads %q, want the frame's longest edge (90) in it", got)
	}
}

// TestExportOptions_OriginalLabelReportsARAWsPreviewSize is the case the
// label exists for: the frame on screen for a RAW file is the camera's
// embedded JPEG preview, so "Original" means the preview's dimensions
// rather than the sensor's - and the number says so without a special case.
// Asserting the same number against the exported file is what proves the
// label is not merely plausible.
func TestExportOptions_OriginalLabelReportsARAWsPreviewSize(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempRAWURI(t, "shot.cr2", 64, 48, color.White))

	v.promptExport()

	label := originalRungLabel(t, v)
	if !strings.Contains(label, "64") {
		t.Errorf("the Original rung reads %q, want the embedded preview's longest edge (64) in it", label)
	}

	dest := filepath.Join(t.TempDir(), "copy.png")
	uitest.StubSaveChooser(t, func(string) ([]byte, error) { return []byte(dest + "\n"), nil })
	v.exportAs(".png")
	settleChooser(t, v)

	written, err := loadExported(t, dest)
	if err != nil {
		t.Fatalf("load the exported file: %v", err)
	}
	if b := written.Bounds(); b.Dx() != 64 || b.Dy() != 48 {
		t.Errorf("exported bounds = %v, want the 64x48 preview the label promised", b)
	}

	settleToast(t, v)
}

// --- keyboard ---------------------------------------------------------------

// TestExportPrompt_KeyboardReachesTheSizeRow covers the whole keyboard
// story the rows add: Up leaves the buttons for the options, Left/Right
// walk the rungs there instead of the format buttons, and Return still
// commits from where the selection happens to be.
func TestExportPrompt_KeyboardReachesTheSizeRow(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 40, 30, color.White))

	var suggested string
	uitest.StubSaveChooser(t, func(s string) ([]byte, error) {
		suggested = s
		return nil, nil // cancelled: this test is about the keyboard, not the file
	})

	v.promptExport()

	// Right on the button row is still the format ring, not the rungs.
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	if got := v.exportPrompt.Selected(); got != jpegChoice {
		t.Fatalf("selection = %d after Right on the button row, want jpegChoice (%d)", got, jpegChoice)
	}
	if got := v.exportOptions.Options().MaxEdge; got != 0 {
		t.Errorf("MaxEdge = %d after Right on the button row, want the rungs untouched (0)", got)
	}

	// Up twice, past the metadata row, into the size row: now Left/Right
	// walk the rungs.
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyUp})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyUp})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	if got := v.exportOptions.Options().MaxEdge; got != 2400 {
		t.Errorf("MaxEdge = %d after Right in the size row, want the second rung (2400)", got)
	}
	if got := v.exportPrompt.Selected(); got != jpegChoice {
		t.Errorf("selection = %d, want the format ring left where it was (%d)", got, jpegChoice)
	}
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyLeft})
	if got := v.exportOptions.Options().MaxEdge; got != 0 {
		t.Errorf("MaxEdge = %d after Left back to the first rung, want Original (0)", got)
	}

	// The size row has nothing to activate, so Return still commits from it
	// without going back down to the buttons first.
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
	settleChooser(t, v)

	if v.exportPrompt.Visible() {
		t.Error("Return from the size row should still commit and take the prompt down")
	}
	if filepath.Ext(suggested) != exportJPEGExt {
		t.Errorf("suggested %q, want the JPEG the format ring was left on", suggested)
	}
}

// TestExportPrompt_EscapeFromTheSizeRowCancels pins the other key that
// belongs to the prompt as a whole rather than to the focused row.
func TestExportPrompt_EscapeFromTheSizeRowCancels(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 40, 30, color.White))

	called := false
	uitest.StubSaveChooser(t, func(string) ([]byte, error) {
		called = true
		return nil, nil
	})

	v.promptExport()
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyUp})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyUp})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})

	if v.exportPrompt.Visible() {
		t.Error("Escape from the size row should dismiss the prompt")
	}
	if called {
		t.Error("Escape must not open the save panel")
	}
}

// --- what lands on disk -----------------------------------------------------

func TestExportAs_SizeLimitCapsTheWrittenFile(t *testing.T) {
	for _, tc := range []struct {
		name         string
		w, h         int
		rung         int
		wantW, wantH int
	}{
		{"caps the longest edge", 1200, 800, 1000, 1000, 666},
		{"never enlarges a photo already inside the rung", 800, 600, 1000, 800, 600},
		{"original writes the frame's own size", 1200, 800, 0, 1200, 800},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := newTestViewer(t)
			dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", tc.w, tc.h, color.White))

			dest := filepath.Join(t.TempDir(), "copy.png")
			uitest.StubSaveChooser(t, func(string) ([]byte, error) { return []byte(dest + "\n"), nil })

			v.promptExport()
			selectExportRung(t, v, tc.rung)
			v.exportPrompt.Select(pngChoice)
			v.exportPrompt.Confirm()
			settleChooser(t, v)

			written, err := loadExported(t, dest)
			if err != nil {
				t.Fatalf("load the exported file: %v", err)
			}
			if b := written.Bounds(); b.Dx() != tc.wantW || b.Dy() != tc.wantH {
				t.Errorf("exported bounds = %v, want %dx%d", b, tc.wantW, tc.wantH)
			}

			settleToast(t, v)
		})
	}
}

// --- the suggested name -----------------------------------------------------

// TestExportAs_SuggestedNameCarriesTheAppliedSize covers the rule that
// keeps a resized copy from colliding with the original it was exported
// from - and the two cases where nothing was resized and so nothing is
// added.
func TestExportAs_SuggestedNameCarriesTheAppliedSize(t *testing.T) {
	for _, tc := range []struct {
		name string
		w, h int
		rung int
		want string
	}{
		{"a limit that applied", 1200, 800, 1000, "holiday-1000.png"},
		{"a limit larger than the photo", 800, 600, 1000, "holiday.png"},
		{"original size", 1200, 800, 0, "holiday.png"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := newTestViewer(t)
			path := uitest.WriteTempFile(t, "holiday.jpg", uitest.EncodeJPEG(t, tc.w, tc.h, color.White))
			dropAndWait(t, v, storage.NewFileURI(path))

			var suggested string
			uitest.StubSaveChooser(t, func(s string) ([]byte, error) {
				suggested = s
				return nil, nil // cancelled: the suggested name is all this asserts on
			})

			v.promptExport()
			selectExportRung(t, v, tc.rung)
			v.exportPrompt.Select(pngChoice)
			v.exportPrompt.Confirm()
			settleChooser(t, v)

			if want := filepath.Join(filepath.Dir(path), tc.want); suggested != want {
				t.Errorf("suggested %q, want %q", suggested, want)
			}
		})
	}
}

// --- the completion toast ---------------------------------------------------

// TestExportAs_ToastReportsASizeOnlyWhenOneApplied keeps a routine export
// as quiet as it is today while making a non-default one a receipt of what
// actually went out.
func TestExportAs_ToastReportsASizeOnlyWhenOneApplied(t *testing.T) {
	for _, tc := range []struct {
		name     string
		rung     int
		wantSize bool
	}{
		{"a limit that applied", 1000, true},
		{"a limit larger than the photo", 2400, false},
		{"original size", 0, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := newTestViewer(t)
			dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 1200, 800, color.White))

			dest := filepath.Join(t.TempDir(), "copy.png")
			uitest.StubSaveChooser(t, func(string) ([]byte, error) { return []byte(dest + "\n"), nil })

			v.promptExport()
			selectExportRung(t, v, tc.rung)
			v.exportPrompt.Select(pngChoice)
			v.exportPrompt.Confirm()
			settleChooser(t, v)

			got := v.toast.text.Text
			if !strings.Contains(got, "copy.png") {
				t.Errorf("toast = %q, want the written file's name in it", got)
			}
			if size := strconv.Itoa(tc.rung); tc.wantSize != strings.Contains(got, size) {
				t.Errorf("toast = %q, want size %q reported = %v", got, size, tc.wantSize)
			}

			settleToast(t, v)
		})
	}
}

// --- helpers ----------------------------------------------------------------

// selectExportRung moves the export prompt's size limit onto the rung
// capping the longest edge at maxEdge (0 for Original), failing the test if
// the prompt does not offer it.
func selectExportRung(t *testing.T, v *viewer, maxEdge int) {
	t.Helper()

	for i, rung := range exportSizeRungs {
		if rung == maxEdge {
			v.exportOptions.selectRung(i)
			return
		}
	}

	t.Fatalf("the export prompt offers no %d px rung; it has %v", maxEdge, exportSizeRungs)
}

// exportRowContents is what the prompt's rows actually draw: every label in
// tree order (the size row's heading, then one per rung) and the metadata
// row's checkbox. Walked from the block's root rather than read off the
// widget's own fields, because a control that was built and then left out of
// its container still reports Visible() - only finding it under the root
// proves a user can see it.
func exportRowContents(t *testing.T, v *viewer) ([]string, *widget.Check) {
	t.Helper()

	var labels []string
	var check *widget.Check
	var walk func(fyne.CanvasObject)
	walk = func(object fyne.CanvasObject) {
		switch obj := object.(type) {
		case *widget.Label:
			labels = append(labels, obj.Text)
		case *widget.Check:
			check = obj
		case *fyne.Container:
			for _, child := range obj.Objects {
				walk(child)
			}
		case fyne.Widget:
			for _, child := range test.WidgetRenderer(obj).Objects() {
				walk(child)
			}
		}
	}
	walk(v.exportOptions.Content())

	if want := len(exportSizeRungs) + 1; len(labels) != want {
		t.Fatalf("the prompt's rows draw %d labels (%q), want %d: a heading and %d rungs",
			len(labels), labels, want, len(exportSizeRungs))
	}
	if check == nil {
		t.Fatal("the prompt's rows draw no checkbox at all")
	}

	return labels, check
}

// originalRungLabel is the text the prompt draws for the Original rung -
// the first one after the size row's heading.
func originalRungLabel(t *testing.T, v *viewer) string {
	t.Helper()

	labels, _ := exportRowContents(t, v)

	return labels[1]
}

// metadataCheck is the checkbox the prompt draws for the metadata option.
func metadataCheck(t *testing.T, v *viewer) *widget.Check {
	t.Helper()

	_, check := exportRowContents(t, v)

	return check
}

// exportedFileBytes reads a written export back as raw bytes, for the tests
// that assert on tags rather than on pixels.
func exportedFileBytes(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read the exported file: %v", err)
	}

	return data
}

// --- metadata omission ------------------------------------------------------

// TestExportOptions_MetadataRowStatesItsJPEGOnlyScope pins the wording as
// much as the default. PNG carries no metadata either way, and the format
// is not known until a button commits, so the label says so permanently
// rather than greying itself out - and it must never read like the EXIF
// window's irreversible metadata *removal*, which is a different operation
// on a different file.
func TestExportOptions_MetadataRowStatesItsJPEGOnlyScope(t *testing.T) {
	v := newTestViewer(t)

	if v.exportOptions.Options().OmitMetadata {
		t.Error("the metadata box should start checked - today's behaviour")
	}

	check := metadataCheck(t, v)
	if !check.Checked {
		t.Error("the metadata checkbox is drawn unticked, want it ticked by default")
	}
	label := check.Text
	if !strings.Contains(label, "JPEG") {
		t.Errorf("the metadata row reads %q, want its JPEG-only scope stated", label)
	}
	for _, word := range []string{"strip", "Strip", "remove", "Remove"} {
		if strings.Contains(label, word) {
			t.Errorf("the metadata row reads %q, which borrows metadata removal's wording (%q)", label, word)
		}
	}
}

// TestExportPrompt_SpaceTogglesTheMetadataRow is the keyboard half of the
// checkbox: Return is spoken for by the export itself, so Space is what
// toggles - and only while the selection is actually on that row.
func TestExportPrompt_SpaceTogglesTheMetadataRow(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 40, 30, color.White))

	v.promptExport()

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeySpace})
	if v.exportOptions.Options().OmitMetadata {
		t.Error("Space on the button row must not toggle the metadata box")
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyUp})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeySpace})
	if !v.exportOptions.Options().OmitMetadata {
		t.Error("Space on the metadata row should uncheck the box")
	}
	if metadataCheck(t, v).Checked {
		t.Error("the metadata checkbox is still drawn ticked after Space, want it unticked")
	}
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeySpace})
	if v.exportOptions.Options().OmitMetadata {
		t.Error("Space again should check the box back")
	}
}

// TestExportAs_OmittingMetadataWritesACleanCopyAndKeepsTheOriginal is the
// user story the whole option exists for: a clean copy goes out, the
// photographer keeps their own capture data, and nothing asks them to
// confirm anything - export cannot harm the file it reads.
func TestExportAs_OmittingMetadataWritesACleanCopyAndKeepsTheOriginal(t *testing.T) {
	v := newTestViewer(t)
	path := uitest.WriteTempFile(t, "geo.jpg", uitest.GPSJPEG(t, 8, 4, 48.858, 2.294))
	dropAndWait(t, v, storage.NewFileURI(path))

	dest := filepath.Join(t.TempDir(), "copy.jpg")
	uitest.StubSaveChooser(t, func(string) ([]byte, error) { return []byte(dest + "\n"), nil })

	v.promptExport()
	v.exportOptions.setMetadataIncluded(false)
	v.exportPrompt.Select(jpegChoice)
	v.exportPrompt.Confirm()

	if top := v.win.Canvas().Overlays().Top(); top != nil {
		t.Errorf("omitting metadata raised %T, want no confirmation at all", top)
	}
	settleChooser(t, v)

	if imaging.ReadMetadata(exportedFileBytes(t, dest)).HasGPS {
		t.Error("the exported copy still carries the source's GPS position")
	}
	if !imaging.ReadMetadata(exportedFileBytes(t, path)).HasGPS {
		t.Error("the source file lost its GPS position - export must only ever write the copy")
	}

	settleToast(t, v)
}

// TestExportAs_OmittingMetadataStillWritesAnUprightCopy: the orientation
// tag goes with the rest of the metadata, so the pixels have to be the
// upright ones already - which they are, since the export writes the frame
// on screen.
func TestExportAs_OmittingMetadataStillWritesAnUprightCopy(t *testing.T) {
	v := newTestViewer(t)
	path := uitest.WriteTempFile(t, "sideways.jpg", uitest.EncodeOrientedJPEG(t, 40, 20, 6))
	dropAndWait(t, v, storage.NewFileURI(path))

	dest := filepath.Join(t.TempDir(), "copy.jpg")
	uitest.StubSaveChooser(t, func(string) ([]byte, error) { return []byte(dest + "\n"), nil })

	v.promptExport()
	v.exportOptions.setMetadataIncluded(false)
	v.exportPrompt.Select(jpegChoice)
	v.exportPrompt.Confirm()
	settleChooser(t, v)

	written, err := loadExported(t, dest)
	if err != nil {
		t.Fatalf("load the exported file: %v", err)
	}
	if b := written.Bounds(); b.Dx() != 20 || b.Dy() != 40 {
		t.Fatalf("exported bounds = %v, want the upright 20x40", b)
	}
	topR, _, topB, _ := written.At(10, 10).RGBA()
	bottomR, _, bottomB, _ := written.At(10, 30).RGBA()
	if topR <= topB || bottomB <= bottomR {
		t.Errorf("exported pixels top R:B=%d:%d bottom R:B=%d:%d, want red above blue",
			topR, topB, bottomR, bottomB)
	}

	settleToast(t, v)
}

// TestExportAs_ToastReportsOmissionOnlyWhenTheBoxWasUnchecked keeps a
// routine export's confirmation exactly as short as it is today.
func TestExportAs_ToastReportsOmissionOnlyWhenTheBoxWasUnchecked(t *testing.T) {
	for _, tc := range []struct {
		name     string
		included bool
	}{
		{"metadata included", true},
		{"metadata omitted", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v := newTestViewer(t)
			dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 40, 30, color.White))

			dest := filepath.Join(t.TempDir(), "copy.jpg")
			uitest.StubSaveChooser(t, func(string) ([]byte, error) { return []byte(dest + "\n"), nil })

			v.promptExport()
			v.exportOptions.setMetadataIncluded(tc.included)
			v.exportPrompt.Select(jpegChoice)
			v.exportPrompt.Confirm()
			settleChooser(t, v)

			got := v.toast.text.Text
			if mentions := strings.Contains(got, lang.L("no camera metadata")); mentions == tc.included {
				t.Errorf("toast = %q with metadata included = %v", got, tc.included)
			}

			settleToast(t, v)
		})
	}
}

// TestExportPrompt_ClickingTheMetadataCheckboxLeavesTheKeyboardWorking is
// the guard the checkbox needs to exist at all. Fyne focuses a widget.Check
// on the tap that toggles it, and this app dispatches every key from the
// canvas's *unfocused* handler - so without the Unfocus in the box's own
// OnChanged, one click would leave the Check holding Return and Escape and
// strand a prompt that can no longer be committed or cancelled.
//
// The Focused() assertion is the one that catches that. The key presses
// below go through handleKeyEvent directly, which is the path Fyne takes
// only while nothing is focused - so they document what has to keep working
// but cannot themselves fail on a focused Check.
func TestExportPrompt_ClickingTheMetadataCheckboxLeavesTheKeyboardWorking(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 40, 30, color.White))

	var suggested string
	uitest.StubSaveChooser(t, func(s string) ([]byte, error) {
		suggested = s
		return nil, nil // cancelled: this test is about the keyboard
	})

	v.promptExport()
	test.Tap(metadataCheck(t, v))

	if !v.exportOptions.Options().OmitMetadata {
		t.Fatal("clicking the checkbox should untick it")
	}
	if focused := v.win.Canvas().Focused(); focused != nil {
		t.Errorf("the canvas is focused on %T after the click, want the keyboard handed back to the app", focused)
	}

	// The keys the prompt owns must still reach it: Right moves the format
	// ring, and Return commits from wherever the selection stands.
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	if got := v.exportPrompt.Selected(); got != jpegChoice {
		t.Errorf("selection = %d after Right, want jpegChoice (%d) - the click swallowed the keyboard", got, jpegChoice)
	}
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
	settleChooser(t, v)

	if v.exportPrompt.Visible() {
		t.Error("Return after clicking the checkbox should still commit the export")
	}
	if filepath.Ext(suggested) != exportJPEGExt {
		t.Errorf("suggested %q, want the JPEG the format ring was left on", suggested)
	}
}

// TestExportPrompt_ClickingTheMetadataCheckboxTwiceReturnsToTheDefault
// covers the click path's own round trip, since the box is the one place
// that state lives.
func TestExportPrompt_ClickingTheMetadataCheckboxTwiceReturnsToTheDefault(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 40, 30, color.White))

	v.promptExport()
	test.Tap(metadataCheck(t, v))
	test.Tap(metadataCheck(t, v))

	if got := v.exportOptions.Options(); got != (imaging.ExportOptions{}) {
		t.Errorf("Options() = %+v after two clicks, want the defaults back", got)
	}
	if !metadataCheck(t, v).Checked {
		t.Error("the checkbox is drawn unticked after two clicks, want it ticked")
	}
}

// TestExportPrompt_ReturnOnTheMetadataRowTicksTheBoxInsteadOfExporting is
// the rule a highlighted checkbox creates: whatever Return means elsewhere
// on this card, on a checkbox it means that checkbox. Committing the export
// out from under someone who was reaching for the control they were looking
// at is the surprise this prevents.
func TestExportPrompt_ReturnOnTheMetadataRowTicksTheBoxInsteadOfExporting(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 40, 30, color.White))

	opened := false
	uitest.StubSaveChooser(t, func(string) ([]byte, error) {
		opened = true
		return nil, nil
	})

	v.promptExport()
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyUp}) // onto the metadata row
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})

	if opened {
		t.Error("Return on the metadata row opened the save panel, want it to tick the box instead")
	}
	if !v.exportPrompt.Visible() {
		t.Error("Return on the metadata row took the prompt down, want it left up")
	}
	if !v.exportOptions.Options().OmitMetadata {
		t.Error("Return on the metadata row did not untick the box")
	}
	if metadataCheck(t, v).Checked {
		t.Error("the checkbox is still drawn ticked after Return on its row")
	}

	// And back again, so Return is a toggle there rather than a one-way trip.
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if v.exportOptions.Options().OmitMetadata {
		t.Error("a second Return should tick the box back")
	}

	// Down to the buttons, where Return means what it always did.
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyDown})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
	settleChooser(t, v)

	if !opened {
		t.Error("Return on the button row should still commit the export")
	}
}

// TestExportPrompt_OnlyTheFocusedRowIsHighlighted covers the other half of
// reading this card: the format ring dims while the keyboard is up in the
// options, so the bright mark is always where the user is - and it is still
// there, dim, answering which format Return would write.
func TestExportPrompt_OnlyTheFocusedRowIsHighlighted(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 40, 30, color.White))

	v.promptExport()
	ring := v.exportPrompt.Ring(pngChoice)
	full := ringAlpha(t, ring)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyUp}) // metadata row
	if got := ringAlpha(t, ring); got >= full {
		t.Errorf("the format ring is at alpha %d with the keyboard on the metadata row, want it muted below %d", got, full)
	}
	if !ring.Visible() {
		t.Error("the format ring should stay visible while muted - it still says which format Return would write")
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyUp}) // size row
	if got := ringAlpha(t, ring); got >= full {
		t.Errorf("the format ring is at alpha %d with the keyboard on the size row, want it muted", got)
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyDown})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyDown}) // back on the buttons
	if got := ringAlpha(t, ring); got != full {
		t.Errorf("the format ring is at alpha %d back on the button row, want the full %d", got, full)
	}
}

// ringAlpha is a selection ring's stroke opacity.
func ringAlpha(t *testing.T, ring *canvas.Rectangle) uint8 {
	t.Helper()

	if ring == nil {
		t.Fatal("no ring to read a stroke from")
	}
	_, _, _, a := ring.StrokeColor.RGBA()

	return uint8(a >> 8)
}
