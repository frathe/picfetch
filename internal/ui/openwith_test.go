package ui

import (
	"image/color"
	"slices"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/openwith"
	"github.com/frathe/picfetch/internal/uitest"
)

// These drive the same wiring Run's SetOnStarted does - set pendingInitial,
// installOpenWithHandler, openInitialFiles - and deliver through the real
// openwith queue rather than calling the handler directly, so the flush that
// SetHandler performs on install is part of what is under test. Every test
// here installs a handler at some point, which matters: openwith's queue is
// process-global and SetHandler(nil) (drain, harness_test.go) deliberately
// leaves anything still buffered in place, so a test that delivered without
// ever installing would hand its files to whichever test installed next.

// TestInstallOpenWithHandler_FlushesADeliveryThatArrivedBeforeInstall is the
// cold start: on macOS the Apple Event lands while Fyne is still building
// the first window, so by the time SetOnStarted installs the handler the
// files are already sitting in the queue and only the flush can reach them.
func TestInstallOpenWithHandler_FlushesADeliveryThatArrivedBeforeInstall(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	openwith.Deliver([]fyne.URI{a})

	if v.scanOp.done.Begun() {
		t.Fatal("a delivery with no handler installed must not scan anything yet")
	}

	v.installOpenWithHandler()
	waitForScan(t, v)
	waitForSort(t, v)
	waitUntilLoaded(t, v)

	if len(v.state.files) != 1 || v.state.files[0].Name() != "a.jpg" {
		t.Errorf("files = %v, want the buffered a.jpg opened by the install flush", v.state.files)
	}
	if v.img.Image == nil {
		t.Error("the delivered image should be on screen")
	}
}

func TestOpenWithHandler_DeliveryWhileInstalledOpensTheFiles(t *testing.T) {
	v := newTestViewer(t)
	v.installOpenWithHandler()

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	openwith.Deliver([]fyne.URI{a})
	waitForScan(t, v)
	waitForSort(t, v)
	waitUntilLoaded(t, v)

	if len(v.state.files) != 1 || v.state.files[0].Name() != "a.jpg" {
		t.Errorf("files = %v, want the delivered a.jpg opened", v.state.files)
	}
	if v.img.Image == nil {
		t.Error("the delivered image should be on screen")
	}
}

// TestOpenInitialFiles_ArgvAndADeliveryBecomeOneScanWithArgvFirst is the
// launch that carries both kinds of file. Two handleDrops would mean two
// overlapping scans, the first of them immediately superseded - so the
// second batch would replace the first rather than joining it.
func TestOpenInitialFiles_ArgvAndADeliveryBecomeOneScanWithArgvFirst(t *testing.T) {
	v := newTestViewer(t)

	// Named so the name sort (the default mode) would put the delivered
	// file first: unsortedFiles keeps the order the batch was dropped in,
	// files keeps the sorted one, and only the first proves argv led.
	argv := uitest.TempJPEGURI(t, "z-argv.jpg", 4, 4, color.White)
	delivered := uitest.TempJPEGURI(t, "a-delivered.jpg", 4, 4, color.White)

	openwith.Deliver([]fyne.URI{delivered})

	scansBefore := v.scanOp.lifecycle.currentRevision()

	v.pendingInitial = []fyne.URI{argv}
	v.installOpenWithHandler()
	v.openInitialFiles()

	waitForScan(t, v)
	waitForSort(t, v)
	waitUntilLoaded(t, v)

	if scans := v.scanOp.lifecycle.currentRevision() - scansBefore; scans != 1 {
		t.Errorf("the drop started %d scans, want exactly 1 - argv and the delivery must be one batch", scans)
	}
	if got := namesOfURIs(v.state.unsortedFiles); !slices.Equal(got, []string{"z-argv.jpg", "a-delivered.jpg"}) {
		t.Errorf("dropped order = %v, want the argv file first then the delivered one", got)
	}
	if v.pendingInitial != nil {
		t.Errorf("pendingInitial = %v, want it cleared by whichever path took the batch", v.pendingInitial)
	}
	assertEquivalentFileSlices(t, v)
	assertValidFileIndex(t, v)
}

func TestOpenInitialFiles_OpensPendingWhenNothingWasDelivered(t *testing.T) {
	v := newTestViewer(t)

	argv := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)

	v.pendingInitial = []fyne.URI{argv}
	v.installOpenWithHandler()
	v.openInitialFiles()

	waitForScan(t, v)
	waitForSort(t, v)
	waitUntilLoaded(t, v)

	if len(v.state.files) != 1 || v.state.files[0].Name() != "a.jpg" {
		t.Errorf("files = %v, want the command-line a.jpg opened", v.state.files)
	}
	if v.pendingInitial != nil {
		t.Errorf("pendingInitial = %v, want it cleared once opened", v.pendingInitial)
	}
}

// TestOpenInitialFiles_PlainLaunchStartsNoScan is the empty launch - no
// arguments, no delivery - which must leave the drop zone exactly as
// buildViewer left it rather than run a scan over nothing.
func TestOpenInitialFiles_PlainLaunchStartsNoScan(t *testing.T) {
	v := newTestViewer(t)

	scansBefore := v.scanOp.lifecycle.currentRevision()

	v.installOpenWithHandler()
	v.openInitialFiles()

	if scans := v.scanOp.lifecycle.currentRevision() - scansBefore; scans != 0 {
		t.Errorf("a launch with nothing to open started %d scans, want 0", scans)
	}
	if v.scanOp.done.Begun() {
		t.Error("a launch with nothing to open must not begin the scan signal")
	}
	if !v.dropzone.Visible() {
		t.Error("the drop zone should still be showing after a launch with nothing to open")
	}
}

// TestOpenWithHandler_DeliveryClosesTheGridAndCancelsAPendingDelete proves
// the delivery really goes through handleDrop rather than some parallel
// ingestion path: closing the grid and dismissing the confirmation are
// handleDrop's own first two statements, and nothing in openwith.go repeats
// them.
func TestOpenWithHandler_DeliveryClosesTheGridAndCancelsAPendingDelete(t *testing.T) {
	v := openGridWith(t, "a.jpg", "b.jpg")

	v.deletion.Request()
	if !v.deletion.Visible() {
		t.Fatal("setup: the delete confirmation should be up")
	}

	v.installOpenWithHandler()

	c := uitest.TempJPEGURI(t, "c.jpg", 4, 4, color.White)
	openwith.Deliver([]fyne.URI{c})
	waitForScan(t, v)
	waitForSort(t, v)
	waitUntilLoaded(t, v)

	if v.grid.Visible() {
		t.Error("the grid should have closed - it was showing the file set the delivery replaced")
	}
	if v.deletion.Visible() {
		t.Error("the pending delete confirmation should have been cancelled")
	}
	if len(v.state.files) != 1 || v.state.files[0].Name() != "c.jpg" {
		t.Errorf("files = %v, want the delivered c.jpg to have replaced the set", v.state.files)
	}
}

// TestOpenWithHandler_DeliveryMergesWhenMergeModeIsOn is parity with
// drag-and-drop: merge mode is read by handleDrop, so an "Open With" honors
// it for free - as long as the delivery keeps going through handleDrop.
func TestOpenWithHandler_DeliveryMergesWhenMergeModeIsOn(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	dropAndWait(t, v, a)

	v.state.SetMergeMode(true)
	v.installOpenWithHandler()

	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	openwith.Deliver([]fyne.URI{b})
	waitForScan(t, v)
	waitForSort(t, v)
	waitUntilLoaded(t, v)

	if len(v.state.files) != 2 {
		t.Fatalf("files = %v, want a.jpg and b.jpg - a delivery should merge, not replace", v.state.files)
	}
	if got := v.state.files[v.state.index].Name(); got != "b.jpg" {
		t.Errorf("displayed file = %q, want the just-merged b.jpg", got)
	}
}
