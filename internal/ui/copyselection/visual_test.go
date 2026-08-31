package copyselection_test

import (
	"image"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/ui/copyselection"
)

func TestModeActivation(t *testing.T) {
	feature, _ := newFeatureCanvas(t, sampleView, copyselection.Callbacks{})

	if got := feature.State(); got != (copyselection.State{Active: true}) {
		t.Fatalf("State() after Start = %+v, want active with no selection", got)
	}
	if !feature.Overlay().Visible() {
		t.Fatal("overlay is hidden after Start")
	}
	if copyButton(t, feature).Visible() {
		t.Fatal("Copy to clipboard is visible before a rectangle is committed")
	}

	feature.Start(sampleView)
	if got := feature.State(); got != (copyselection.State{Active: true}) {
		t.Fatalf("State() after repeated Start = %+v, want unchanged", got)
	}

	feature.Cancel()
	if got := feature.State(); got != (copyselection.State{}) {
		t.Fatalf("State() after Cancel = %+v, want inactive", got)
	}
	if feature.Overlay().Visible() {
		t.Fatal("overlay is visible after Cancel")
	}
}

func TestCopyButtonVisibility(t *testing.T) {
	feature, selectionCanvas := newFeatureCanvas(t, sampleView, copyselection.Callbacks{
		Copy: func(image.Rectangle) {},
	})
	button := copyButton(t, feature)

	if button.Visible() {
		t.Fatal("Copy to clipboard is visible before a rectangle is committed")
	}
	if button.Text != lang.L("Copy to clipboard") {
		t.Fatalf("button text = %q, want %q", button.Text, lang.L("Copy to clipboard"))
	}

	commitSampleSelection(t, selectionCanvas)
	if !button.Visible() {
		t.Fatal("Copy to clipboard stayed hidden after a committed rectangle")
	}
	if button.Disabled() {
		t.Fatal("Copy to clipboard is disabled on an idle committed rectangle")
	}

	pad := theme.Size(theme.SizeNameInnerPadding)
	want := fyne.NewPos(
		selectionCanvas.Size().Width-button.MinSize().Width-pad,
		selectionCanvas.Size().Height-button.MinSize().Height-pad,
	)
	if button.Position() != want {
		t.Fatalf("button position = %v, want lower-right %v", button.Position(), want)
	}

	feature.HandleKey(fyne.KeyReturn)
	if !button.Visible() || !button.Disabled() {
		t.Fatalf("busy button visible=%v disabled=%v, want visible and disabled", button.Visible(), button.Disabled())
	}

	feature.Complete(nil)
	if button.Visible() {
		t.Fatal("Copy to clipboard remained visible after a successful copy")
	}
}

func TestVisualState(t *testing.T) {
	feature, selectionCanvas := newFeatureCanvas(t, sampleView, copyselection.Callbacks{})
	input := overlayInput(t, feature)

	if n := len(visibleChrome(t, input)); n != 0 {
		t.Fatalf("visible chrome before a rectangle = %d objects, want 0", n)
	}

	commitSampleSelection(t, selectionCanvas)
	chrome := visibleChrome(t, input)
	if len(chrome) != 13 {
		t.Fatalf("visible chrome after commit = %d objects, want 4 dim + 1 border + 8 handles", len(chrome))
	}

	border := strokeRect(t, chrome)
	if border.Position() != fyne.NewPos(70, 50) || border.Size() != fyne.NewSize(100, 80) {
		t.Fatalf("border = pos %v size %v, want pos (70,50) size 100x80", border.Position(), border.Size())
	}

	dims := fillRects(chrome, true)
	if len(dims) != 4 {
		t.Fatalf("dim rectangles = %d, want 4 covering image content outside the selection", len(dims))
	}
	for _, dim := range dims {
		if coversPoint(dim, fyne.NewPos(5, 5)) {
			t.Fatalf("dim rectangle at %v size %v covers letterbox (5,5)", dim.Position(), dim.Size())
		}
		if coversPoint(dim, fyne.NewPos(120, 90)) {
			t.Fatalf("dim rectangle at %v size %v covers the selection interior (120,90)", dim.Position(), dim.Size())
		}
	}
	if allAvoidPoint(dims, fyne.NewPos(40, 30)) {
		t.Fatal("image content outside the selection is not dimmed at (40,30)")
	}

	handles := fillRects(chrome, false)
	if len(handles) != 8 {
		t.Fatalf("handles = %d, want 8", len(handles))
	}
}

func TestCursorState(t *testing.T) {
	feature, selectionCanvas := newFeatureCanvas(t, sampleView, copyselection.Callbacks{})
	input := overlayInput(t, feature)
	hover, ok := input.(desktop.Hoverable)
	if !ok {
		t.Fatal("overlay input is not desktop.Hoverable")
	}
	cursor, ok := input.(desktop.Cursorable)
	if !ok {
		t.Fatal("overlay input is not desktop.Cursorable")
	}

	hover.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(5, 5)}})
	if got := cursor.Cursor(); got != desktop.DefaultCursor {
		t.Fatalf("letterbox cursor = %v, want DefaultCursor", got)
	}
	hover.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(120, 90)}})
	if got := cursor.Cursor(); got != desktop.CrosshairCursor {
		t.Fatalf("image cursor before commit = %v, want CrosshairCursor", got)
	}

	commitSampleSelection(t, selectionCanvas)
	hover.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(120, 90)}})
	if got := cursor.Cursor(); got != desktop.PointerCursor {
		t.Fatalf("selection interior cursor = %v, want PointerCursor (Fyne has no move cursor)", got)
	}
	hover.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(40, 30)}})
	if got := cursor.Cursor(); got != desktop.CrosshairCursor {
		t.Fatalf("image outside selection cursor = %v, want CrosshairCursor", got)
	}
	hover.MouseMoved(&desktop.MouseEvent{PointEvent: fyne.PointEvent{Position: fyne.NewPos(170, 90)}})
	if got := cursor.Cursor(); got != desktop.HResizeCursor {
		t.Fatalf("east handle cursor = %v, want HResizeCursor", got)
	}
}

var sampleView = copyselection.View{
	ImageBounds: image.Rect(0, 0, 100, 80),
	Position:    fyne.NewPos(20, 10),
	Size:        fyne.NewSize(200, 160),
}

func commitSampleSelection(t *testing.T, selectionCanvas fyne.Canvas) {
	t.Helper()
	test.Drag(selectionCanvas, fyne.NewPos(70, 50), -100, -80)
}

func copyButton(t *testing.T, feature *copyselection.Feature) *widget.Button {
	t.Helper()
	overlay, ok := feature.Overlay().(*fyne.Container)
	if !ok {
		t.Fatalf("Overlay() type %T, want *fyne.Container", feature.Overlay())
	}
	for _, object := range overlay.Objects {
		if button, ok := object.(*widget.Button); ok {
			return button
		}
	}
	t.Fatal("Copy to clipboard button is missing from the overlay")
	return nil
}

func overlayInput(t *testing.T, feature *copyselection.Feature) fyne.CanvasObject {
	t.Helper()
	overlay, ok := feature.Overlay().(*fyne.Container)
	if !ok {
		t.Fatalf("Overlay() type %T, want *fyne.Container", feature.Overlay())
	}
	if len(overlay.Objects) == 0 {
		t.Fatal("overlay has no input object")
	}
	return overlay.Objects[0]
}

func visibleChrome(t *testing.T, input fyne.CanvasObject) []*canvas.Rectangle {
	t.Helper()
	wid, ok := input.(fyne.Widget)
	if !ok {
		t.Fatalf("overlay input type %T, want fyne.Widget", input)
	}
	var out []*canvas.Rectangle
	for _, object := range test.WidgetRenderer(wid).Objects() {
		rect, ok := object.(*canvas.Rectangle)
		if !ok || !rect.Visible() {
			continue
		}
		out = append(out, rect)
	}
	return out
}

func strokeRect(t *testing.T, rects []*canvas.Rectangle) *canvas.Rectangle {
	t.Helper()
	for _, rect := range rects {
		if rect.StrokeWidth > 0 {
			return rect
		}
	}
	t.Fatal("selection border is not painted")
	return nil
}

func fillRects(rects []*canvas.Rectangle, dim bool) []*canvas.Rectangle {
	var out []*canvas.Rectangle
	for _, rect := range rects {
		if rect.StrokeWidth > 0 {
			continue
		}
		_, _, _, a := rect.FillColor.RGBA()
		isDim := a > 0 && a < 0xffff
		if dim == isDim {
			out = append(out, rect)
		}
	}
	return out
}

func coversPoint(rect *canvas.Rectangle, p fyne.Position) bool {
	pos := rect.Position()
	size := rect.Size()
	return p.X >= pos.X && p.Y >= pos.Y && p.X < pos.X+size.Width && p.Y < pos.Y+size.Height
}

func allAvoidPoint(rects []*canvas.Rectangle, p fyne.Position) bool {
	for _, rect := range rects {
		if coversPoint(rect, p) {
			return false
		}
	}
	return true
}
