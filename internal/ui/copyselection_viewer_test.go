package ui

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/ui/copyselection"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestCopySelectionActivation(t *testing.T) {
	v := newTestViewer(t)

	v.startRegionCopy()
	if got := v.regionCopy.State(); got != (copyselection.State{}) {
		t.Fatalf("State() without a decoded image = %+v, want inactive", got)
	}
	if v.regionCopy.Overlay().Visible() {
		t.Fatal("Copy Selection overlay is visible without a decoded image")
	}

	dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))
	v.startRegionCopy()

	want := copyselection.State{Active: true}
	if got := v.regionCopy.State(); got != want {
		t.Fatalf("State() after activation = %+v, want %+v", got, want)
	}
	if !v.regionCopy.Overlay().Visible() {
		t.Fatal("Copy Selection overlay is hidden after activation")
	}

	v.startRegionCopy()
	if got := v.regionCopy.State(); got != want {
		t.Fatalf("State() after repeated activation = %+v, want unchanged %+v", got, want)
	}

	objects := v.win.Content().(*fyne.Container).Objects
	indexOf := func(want fyne.CanvasObject) int {
		for i, object := range objects {
			if containsObject(object, want) {
				return i
			}
		}
		return -1
	}
	info := indexOf(v.info.Object())
	region := indexOf(v.regionCopy.Overlay())
	grid := indexOf(v.grid.Overlay())
	deletePrompt := indexOf(v.deletion.Overlay())
	exportPrompt := indexOf(v.exportPrompt.Overlay())
	toast := indexOf(v.toast.card)
	if info < 0 || region < 0 || grid < 0 || deletePrompt < 0 || exportPrompt < 0 || toast < 0 {
		t.Fatalf("overlay not found: info=%d region=%d grid=%d delete=%d export=%d toast=%d",
			info, region, grid, deletePrompt, exportPrompt, toast)
	}
	if !(info < region && region < grid && grid < deletePrompt && deletePrompt < exportPrompt && exportPrompt < toast) {
		t.Fatalf("overlay order = info:%d region:%d grid:%d delete:%d export:%d toast:%d, want strictly back-to-front",
			info, region, grid, deletePrompt, exportPrompt, toast)
	}
}

func TestCopySelectionInfoOverlay(t *testing.T) {
	for _, initiallyVisible := range []bool{false, true} {
		t.Run(map[bool]string{false: "initially hidden", true: "initially visible"}[initiallyVisible], func(t *testing.T) {
			v := newTestViewer(t)
			dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))
			if initiallyVisible {
				v.toggleInfoOverlay()
			}
			if got := v.info.Object().Visible(); got != initiallyVisible {
				t.Fatalf("setup: info object visibility = %v, want %v", got, initiallyVisible)
			}

			v.startRegionCopy()
			if v.info.Object().Visible() {
				t.Fatal("information overlay remains painted during Copy Selection mode")
			}
			if got := v.info.Visible(); got != initiallyVisible {
				t.Fatalf("standing info preference during mode = %v, want preserved %v", got, initiallyVisible)
			}

			v.cancelRegionCopy()
			if got := v.info.Object().Visible(); got != initiallyVisible {
				t.Fatalf("info object visibility after cancel = %v, want restored %v", got, initiallyVisible)
			}
			if got := v.info.Visible(); got != initiallyVisible {
				t.Fatalf("standing info preference after cancel = %v, want %v", got, initiallyVisible)
			}
		})
	}
}

func TestCopySelectionZoomPanResize(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 800, 400, color.White))

	geometryUpdates := 0
	v.regionCopyDo = func(f func()) {
		geometryUpdates++
		f()
	}
	v.startRegionCopy()

	before := v.zoom.Scale()
	test.Scroll(v.win.Canvas(), fyne.NewPos(100, 100), 0, 20)
	if got := v.zoom.Scale(); got <= before {
		t.Fatalf("zoom scale after scrolling the selection overlay = %v, want greater than %v", got, before)
	}
	if geometryUpdates == 0 {
		t.Fatal("zoom geometry change was not delivered to Copy Selection")
	}

	updatesBeforeResize := geometryUpdates
	v.win.Resize(fyne.NewSize(900, 600))
	if geometryUpdates <= updatesBeforeResize {
		t.Fatal("window resize did not deliver new zoom geometry to Copy Selection")
	}
	if !v.regionCopy.State().Active {
		t.Fatal("zoom and resize ended Copy Selection mode")
	}
}

func TestCopySelectionSurvivesZoomPanAndResize(t *testing.T) {
	TestCopySelectionZoomPanResize(t)
}

func TestCopySelectionCancel(t *testing.T) {
	v := newTestViewer(t)
	u := uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White)
	dropAndWait(t, v, u)
	wantGeometry := v.zoom.Geometry()

	v.startRegionCopy()
	v.cancelRegionCopy()

	if got := v.regionCopy.State(); got != (copyselection.State{}) {
		t.Fatalf("State() after cancel = %+v, want inactive", got)
	}
	if v.regionCopy.Overlay().Visible() {
		t.Fatal("Copy Selection overlay remains visible after cancel")
	}
	current, _, ok := v.CurrentFile()
	if v.FileCount() != 1 || !ok || current.String() != u.String() {
		t.Fatalf("cancel changed the file set/current file: count=%d current=%v ok=%v", v.FileCount(), current, ok)
	}
	if got := v.zoom.Geometry(); got != wantGeometry {
		t.Fatalf("zoom geometry after cancel = %+v, want unchanged %+v", got, wantGeometry)
	}
	if v.clipboard.Begun() {
		t.Fatal("cancel touched the clipboard worker")
	}
}

func TestWireGlobalShortcuts_CopySelectionWithCopyShortcut(t *testing.T) {
	v := newTestViewer(t)
	u := uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White)
	dropAndWait(t, v, u)

	var copiedImage []byte
	uitest.StubClipboardCopy(t, func(data []byte) error {
		copiedImage = data
		return nil
	})
	v.app.Clipboard().SetContent("untouched")

	handler := &fyne.ShortcutHandler{}
	wireGlobalShortcuts(handler, v)

	handler.TypedShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyC,
		Modifier: fyne.KeyModifierAlt | fyne.KeyModifierShift,
	})
	if !v.regionCopy.State().Active {
		t.Fatal("Alt+Shift+C did not start Copy Selection mode")
	}
	if copiedImage != nil || v.app.Clipboard().Content() != "untouched" {
		t.Fatalf("Alt+Shift+C fired an existing copy action: image copied=%v text=%q",
			copiedImage != nil, v.app.Clipboard().Content())
	}

	selectRegion(t, v, copySelectionBounds)
	v.handleKeyEvent(&fyne.KeyEvent{Name: desktop.KeySuperLeft})
	handler.TypedShortcut(&fyne.ShortcutCopy{})
	waitForClipboard(t, v)
	copied, err := png.Decode(bytes.NewReader(copiedImage))
	if err != nil {
		t.Fatalf("decode Cmd/Ctrl+C clipboard PNG: %v", err)
	}
	if got := copied.Bounds(); got != image.Rect(0, 0, 12, 9) {
		t.Fatalf("Cmd/Ctrl+C copied bounds = %v, want selected 12x9 region", got)
	}
	if v.regionCopy.State().Active {
		t.Fatal("Cmd/Ctrl+C left Copy Selection active after copying")
	}

	handler.TypedShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyC,
		Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift,
	})
	if got := v.app.Clipboard().Content(); got != u.Path() {
		t.Fatalf("Cmd/Ctrl+Shift+C clipboard text = %q, want %q", got, u.Path())
	}
}

func TestCopyImageMenuActionCopiesActiveSelection(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))

	var copiedImage []byte
	uitest.StubClipboardCopy(t, func(data []byte) error {
		copiedImage = data
		return nil
	})
	selectRegion(t, v, copySelectionBounds)

	v.menus.Actions().Copy().Action()
	waitForClipboard(t, v)

	copied, err := png.Decode(bytes.NewReader(copiedImage))
	if err != nil {
		t.Fatalf("decode menu copy clipboard PNG: %v", err)
	}
	if got := copied.Bounds(); got != image.Rect(0, 0, 12, 9) {
		t.Fatalf("menu copy bounds = %v, want selected 12x9 region", got)
	}
}
