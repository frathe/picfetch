package help

import (
	"bytes"
	"image"
	"image/draw"
	"os"
	"reflect"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
	"golang.org/x/image/webp"
)

func finisImage(object fyne.CanvasObject) *canvas.Image {
	if picture, ok := object.(*canvas.Image); ok {
		return picture
	}
	var children []fyne.CanvasObject
	switch object := object.(type) {
	case *fyne.Container:
		children = object.Objects
	case fyne.Widget:
		children = test.WidgetRenderer(object).Objects()
	}
	for _, child := range children {
		if picture := finisImage(child); picture != nil {
			return picture
		}
	}
	return nil
}

func assertFinisPose(t *testing.T, window fyne.Window, column, row int) {
	t.Helper()
	picture := finisImage(window.Content())
	if picture == nil || picture.Image == nil {
		t.Fatal("companion has no character image")
	}
	data, err := os.ReadFile("finis.webp")
	if err != nil {
		t.Fatal(err)
	}
	atlas, err := webp.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	bounds := image.Rect(0, 0, 192, 208)
	want := image.NewNRGBA(bounds)
	draw.Draw(want, bounds, atlas, image.Pt(column*192, row*208), draw.Src)
	got := image.NewNRGBA(bounds)
	if picture.Image.Bounds() != bounds {
		t.Fatalf("pose bounds = %v", picture.Image.Bounds())
	}
	draw.Draw(got, bounds, picture.Image, image.Point{}, draw.Src)
	if !reflect.DeepEqual(got.Pix, want.Pix) {
		t.Fatalf("wrong gaze: want atlas column %d, row %d", column, row)
	}
}

func TestFinisGazeDirectionsAndRest(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	h := New(a, "PicFetch", nil)
	h.showFinis()
	window := h.finisWin.Window()
	window.Resize(fyne.NewSize(480, 420))
	assertFinisPose(t, window, 6, 0)
	// Offsets clockwise from up, relative to the face, not the sprite's feet.
	offsets := []fyne.Position{
		{X: 0, Y: -120}, {X: 46, Y: -111}, {X: 85, Y: -85}, {X: 111, Y: -46},
		{X: 120, Y: 0}, {X: 111, Y: 46}, {X: 85, Y: 85}, {X: 46, Y: 111},
		{X: 0, Y: 120}, {X: -46, Y: 111}, {X: -85, Y: 85}, {X: -111, Y: 46},
		{X: -120, Y: 0}, {X: -111, Y: -46}, {X: -85, Y: -85}, {X: -46, Y: -111},
	}
	size := window.Canvas().Size()
	origin := fyne.NewPos(size.Width/2, size.Height/2-40)
	for direction, offset := range offsets {
		test.MoveMouse(window.Canvas(), origin.Add(offset))
		assertFinisPose(t, window, direction%8, 9+direction/8)
	}
	test.MoveMouse(window.Canvas(), origin.Add(fyne.NewPos(10, 10)))
	assertFinisPose(t, window, 6, 0)
	test.MoveMouse(window.Canvas(), origin.Add(fyne.NewPos(120, 0)))
	assertFinisPose(t, window, 4, 9)
	test.MoveMouse(window.Canvas(), fyne.NewPos(-10, -10))
	assertFinisPose(t, window, 6, 0)
	// Resizing moves the face; the same screen point must be re-evaluated.
	test.MoveMouse(window.Canvas(), origin.Add(fyne.NewPos(120, 0)))
	window.Resize(fyne.NewSize(720, 420))
	assertFinisPose(t, window, 6, 0)
	test.MoveMouse(window.Canvas(), fyne.NewPos(100, origin.Y))
	assertFinisPose(t, window, 4, 10)
}

func TestFinisCloseAndReopen(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	main := a.NewWindow("PicFetch")
	main.Show()
	windowsBefore := len(a.Driver().AllWindows())
	mainClosed := false
	main.SetOnClosed(func() { mainClosed = true })
	h := New(a, "PicFetch", nil)
	h.showFinis()
	first := h.finisWin.Window()
	test.MoveMouse(first.Canvas(), fyne.NewPos(450, 170))
	assertFinisPose(t, first, 4, 9)
	h.showFinis()
	if h.finisWin.Window() != first {
		t.Fatal("repeat activation replaced the window")
	}
	first.Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if h.finisWin.Open() {
		t.Fatal("Escape left the companion open")
	}
	if mainClosed || len(a.Driver().AllWindows()) != windowsBefore {
		t.Fatal("Escape affected the main window")
	}
	h.showFinis()
	if h.finisWin.Window() == first {
		t.Fatal("reopen reused a closed window")
	}
	assertFinisPose(t, h.finisWin.Window(), 6, 0)
}

func TestFinisSearchKeepsOrdinaryQueries(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	calls := 0
	view := newManualView("Finis likes pictures. Finis is happy.", nil)
	view.onFinis = func() { calls++ }
	view.submit("Finis likes")
	if calls != 0 || len(hitTexts(view.text.Segments)) != 1 {
		t.Fatal("ordinary search did not highlight normally")
	}
	view.submit(" FINIS ")
	if calls != 1 || view.current != nil || view.state != (searchState{}) || len(hitTexts(view.text.Segments)) != 0 {
		t.Fatal("Finis trigger did not clear the previous search")
	}
}
