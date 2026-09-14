package help

import (
	"bytes"
	"encoding/json"
	"image"
	"image/draw"
	"math"
	"os"
	"reflect"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
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

func revealFinis(window fyne.Window) {
	size := window.Canvas().Size()
	center := fyne.NewPos(size.Width/2, size.Height/2-40)
	for step := 0; step <= 320; step++ {
		angle := float64(step) * math.Pi / 16
		test.MoveMouse(window.Canvas(), center.Add(fyne.NewPos(float32(60*math.Cos(angle)), float32(60*math.Sin(angle)))))
	}
}

func TestFinisClueLayoutAndLocale(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	// Bind each catalogue's hint to the current locale through Fyne's public
	// loader; switching the machine locale is neither needed nor available.
	current := lang.SystemLocale()
	english, err := os.ReadFile("../../../translations/en.json")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := lang.AddTranslationsForLocale(english, current); err != nil {
			t.Error(err)
		}
	})
	for _, locale := range []string{"en", "de"} {
		data, err := os.ReadFile("../../../translations/" + locale + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var catalog map[string]string
		if err := json.Unmarshal(data, &catalog); err != nil {
			t.Fatal(err)
		}
		if catalog[secretPhrase] != secretPhrase {
			t.Fatalf("%s: clue phrase translation = %q, want canonical search trigger %q", locale, catalog[secretPhrase], secretPhrase)
		}
		var baseline map[string]string
		if err := json.Unmarshal(english, &baseline); err != nil {
			t.Fatal(err)
		}
		baseline["(search for it)"] = catalog["(search for it)"]
		hint, err := json.Marshal(baseline)
		if err != nil {
			t.Fatal(err)
		}
		if err := lang.AddTranslationsForLocale(hint, current); err != nil {
			t.Fatal(err)
		}
		// Exercise both variants independently of the machine preference.
		//noinspection GoDeprecation
		for _, th := range []fyne.Theme{theme.LightTheme(), theme.DarkTheme()} {
			a.Settings().SetTheme(th)
			h := New(a, "PicFetch", nil)
			h.ShowFinis()
			window := h.finisWin.Window()
			revealFinis(window)
			want := "please hypnotize me (search for it)"
			if locale == "de" {
				want = "please hypnotize me (suche danach)"
			}
			if !h.finis.clue.Visible() || h.finis.clue.text.Text != want {
				t.Fatalf("%s: clue = %q, visible %v", locale, h.finis.clue.text.Text, h.finis.clue.Visible())
			}
			for _, size := range []fyne.Size{fyne.NewSize(280, 320), fyne.NewSize(480, 420), fyne.NewSize(900, 700)} {
				// The native driver enforces the content minimum; the test
				// window accepts unsupported smaller sizes.
				window.Resize(size.Max(window.Content().MinSize()))
				clue, label := h.finis.clue, h.finis.clue.text
				origin := a.Driver().AbsolutePositionForObject(clue)
				portrait := a.Driver().AbsolutePositionForObject(finisImage(window.Content()))
				if origin.X < 0 || origin.Y < 0 || origin.X+clue.Size().Width > window.Canvas().Size().Width || origin.Y+clue.Size().Height > portrait.Y {
					t.Fatalf("%s at %v: bubble %v/%v is outside the canvas or overlaps portrait %v", locale, size, origin, clue.Size(), portrait)
				}
				if label.Wrapping != fyne.TextWrapWord || label.Size().Height < label.MinSize().Height || label.Position().X+label.Size().Width > clue.Size().Width {
					t.Fatalf("%s at %v: text clipped: %v / minimum %v", locale, size, label.Size(), label.MinSize())
				}
			}
			window.Close()
		}
	}
}

func TestFinisClueClearsManualSearch(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	original := currentManual
	t.Cleanup(func() { currentManual = original })
	currentManual = func() string { return searchFixture }
	h := New(a, "PicFetch", nil)
	h.ShowManual()
	manual := h.manualWin.Window()
	h.manual.entry.SetText("alpha")
	h.manual.entry.TypedKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	if len(hitTexts(h.manual.text.Segments)) != 2 {
		t.Fatal("precondition: ordinary search did not highlight")
	}
	manual.Canvas().Unfocus()
	h.ShowFinis()
	companion := h.finisWin.Window()
	revealFinis(companion)
	clue := h.finis.clue
	point := a.Driver().AbsolutePositionForObject(clue).Add(fyne.NewPos(clue.Size().Width/2, clue.Size().Height/2))
	test.TapCanvas(companion.Canvas(), point)
	if h.manualWin.Window() != manual || h.manual.entry.Text != "" || manual.Canvas().Focused() != h.manual.entry || len(hitTexts(h.manual.text.Segments)) != 0 || h.manual.current != nil || h.manual.state != (searchState{}) {
		t.Fatal("clue click failed to clear and focus existing manual search")
	}
	if !h.finisWin.Open() || !clue.Visible() {
		t.Fatal("clue click dismissed Finis")
	}
	companion.Close()
	manual.Close()
}

func TestFinisCirclesReachWindowEdges(t *testing.T) {
	a := test.NewApp()
	t.Cleanup(a.Quit)
	h := New(a, "PicFetch", nil)
	h.ShowFinis()
	window := h.finisWin.Window()
	t.Cleanup(window.Close)
	size := window.Canvas().Size()
	vertices := []fyne.Position{{X: size.Width - 1, Y: 1}, {X: size.Width - 1, Y: size.Height - 1}, {X: 1, Y: size.Height - 1}, {X: 1, Y: 1}, {X: size.Width - 1, Y: 1}}
	for range 10 {
		for i := 1; i < len(vertices); i++ {
			from, to := vertices[i-1], vertices[i]
			for step := 0; step <= 40; step++ {
				test.MoveMouse(window.Canvas(), fyne.NewPos(from.X+(to.X-from.X)*float32(step)/40, from.Y+(to.Y-from.Y)*float32(step)/40))
			}
		}
	}
	if !h.finis.clue.Visible() {
		t.Fatal("circles just inside the window edges did not reveal the clue")
	}
}

func assertFinisPose(t *testing.T, window fyne.Window, column, row int) {
	t.Helper()
	picture := finisImage(window.Content())
	if picture == nil || picture.Image == nil {
		t.Fatal("companion has no character image")
	}
	data, err := os.ReadFile("../../../assets/finis/spritesheet.webp")
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
	h.ShowFinis()
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
	test.MoveMouse(window.Canvas(), origin.Add(fyne.NewPos(20, 0)))
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
	h.ShowFinis()
	first := h.finisWin.Window()
	test.MoveMouse(first.Canvas(), fyne.NewPos(450, 170))
	assertFinisPose(t, first, 4, 9)
	h.ShowFinis()
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
	h.ShowFinis()
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
