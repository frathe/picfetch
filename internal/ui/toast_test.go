package ui

import (
	"fmt"
	"image/color"
	"strings"
	"sync/atomic"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/appearance"
	"github.com/frathe/picfetch/internal/ui/widgets"
)

type toastFontReads struct {
	fyne.Theme
	reads atomic.Int32
}

func TestToast_WrappedTextRemainsReadable(t *testing.T) {
	previous := testApp.Settings().Theme()
	t.Cleanup(func() { testApp.Settings().SetTheme(previous) })
	v := newTestViewer(t)
	v.win.Resize(fyne.NewSize(360, 400))
	for _, mode := range []appearance.Mode{appearance.Light, appearance.Dark} {
		v.SetThemeMode(mode)
		for _, message := range []string{"Short error", strings.Repeat("A longer image error needs room. ", 5), "Another short error"} {
			v.ShowToast(message)
			v.ForceRepaint()
			v.win.Canvas().Capture()
			var lines []*canvas.Text
			var walk func(fyne.CanvasObject)
			walk = func(object fyne.CanvasObject) {
				if !object.Visible() {
					return
				}
				switch object := object.(type) {
				case *canvas.Text:
					if object.Text != "" {
						lines = append(lines, object)
					}
				case *fyne.Container:
					for _, child := range object.Objects {
						walk(child)
					}
				case fyne.Widget:
					for _, child := range test.WidgetRenderer(object).Objects() {
						walk(child)
					}
				}
			}
			walk(v.toast.card)
			if len(lines) == 0 || len(message) > 100 && len(lines) < 2 {
				t.Fatalf("toast did not render the expected wrapped text: %d lines", len(lines))
			}
			for _, line := range lines {
				if !line.TextStyle.Bold || color.NRGBAModel.Convert(line.Color) != color.NRGBAModel.Convert(widgets.ToastTextColor) {
					t.Fatalf("toast lost its bold readable text in mode %v: style=%+v color=%v", mode, line.TextStyle, line.Color)
				}
			}
			if v.toast.card.Size().Width > v.win.Canvas().Size().Width {
				t.Fatal("wrapped toast extends beyond the window")
			}
		}
	}
}

func (t *toastFontReads) Font(style fyne.TextStyle) fyne.Resource {
	return &toastFontResource{Resource: t.Theme.Font(style), reads: &t.reads}
}

type toastFontResource struct {
	fyne.Resource
	reads *atomic.Int32
}

func (r *toastFontResource) Content() []byte {
	r.reads.Add(1)
	return r.Resource.Content()
}

func TestToast_RepaintsReuseParsedFonts(t *testing.T) {
	previous := testApp.Settings().Theme()
	fonts := &toastFontReads{Theme: previous}
	testApp.Settings().SetTheme(fonts)
	t.Cleanup(func() { testApp.Settings().SetTheme(previous) })
	v := newTestViewer(t)
	v.win.Resize(fyne.NewSize(640, 480))
	v.ShowToast("Could not read image 0")
	v.win.Canvas().Capture()
	before := fonts.reads.Load()
	if before == 0 {
		t.Fatal("canvas did not read the instrumented font")
	}
	for i := range 5 {
		v.ShowToast(fmt.Sprintf("Could not read image %d", i+1))
		v.ForceRepaint()
		v.win.Canvas().Capture()
	}
	if got := fonts.reads.Load() - before; got != 0 {
		t.Fatalf("repainting the toast parsed font resources %d more times", got)
	}
}
