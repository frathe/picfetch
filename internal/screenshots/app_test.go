package screenshots

import (
	"image/color"
	"math"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/software"
	"fyne.io/fyne/v2/test"
)

func TestWindowConstraint(t *testing.T) {
	application := test.NewApp()
	defer application.Quit()
	constrained := App(application, 1280, 800)
	for _, title := range []string{"main", "secondary"} {
		w := constrained.NewWindow(title)
		content := canvas.NewRectangle(nil)
		content.SetMinSize(fyne.NewSize(4000, 3000))
		w.SetContent(content)
		w.Resize(fyne.NewSize(6000, 5000))
		w.SetFullScreen(true)
		w.SetFixedSize(false)
		w.Show()
		if got := w.Canvas().Size(); got != fyne.NewSize(1280, 800) {
			t.Fatalf("%s size=%v", title, got)
		}
		if !w.FixedSize() || w.FullScreen() {
			t.Fatalf("%s fixed=%v fullscreen=%v", title, w.FixedSize(), w.FullScreen())
		}
		if w.Content() != content {
			t.Fatal("content identity lost")
		}
		w.SetContent(canvas.NewRectangle(nil))
		w.Resize(fyne.NewSize(200, 200))
		w.Show()
		if got := w.Canvas().Size(); got != fyne.NewSize(1280, 800) {
			t.Fatalf("reopen size=%v", got)
		}
		w.Close()
	}
}

func TestWindowConstraintRetainsPadding(t *testing.T) {
	application := test.NewApp()
	defer application.Quit()
	w := App(application, 1280, 800).NewWindow("padded screenshot")
	if !w.Canvas().(software.WindowlessCanvas).Padded() {
		t.Fatal("screenshot mode removed normal window padding")
	}
	w.Close()
}

type fallbackApp struct{ fyne.App }

func (a fallbackApp) NewWindow(title string) fyne.Window {
	return fallbackWindow{Window: a.App.NewWindow(title)}
}

type fallbackWindow struct{ fyne.Window }

func (w fallbackWindow) RunNative(fn func(any)) { fn(struct{}{}) }

func TestWindowConstraintTracksScale(t *testing.T) {
	application := test.NewApp()
	defer application.Quit()
	for _, native := range []bool{false, true} {
		underlying := application
		if native {
			underlying = fallbackApp{App: application}
		}
		w := App(underlying, 1280, 800).NewWindow("scaled screenshot")
		c := w.Canvas().(software.WindowlessCanvas)
		w.SetContent(canvas.NewRectangle(color.Black))
		c.SetScale(2)
		w.Show()
		if got := c.Capture().Bounds().Size(); got.X != 1280 || got.Y != 800 {
			t.Fatalf("native=%v at 2x screenshot=%v", native, got)
		}
		c.SetScale(1)
		c.Content().Refresh()
		if got := c.Capture().Bounds().Size(); got.X != 1280 || got.Y != 800 {
			t.Fatalf("native=%v after 1x move screenshot=%v", native, got)
		}
		w.Close()
		c.SetScale(2)
		c.Content().Refresh()
	}
}

func TestWindowConstraintTracksNativeBackingWithoutCanvasScaleChange(t *testing.T) {
	application := test.NewApp()
	defer application.Quit()
	w := App(fallbackApp{App: application}, 1280, 800).NewWindow("backing scale").(*window)
	backing := float32(2)
	w.measure = func(_ fyne.Window, width, height int) (fyne.Size, bool) {
		return fyne.NewSize(float32(width)/backing, float32(height)/backing), true
	}
	w.SetContent(canvas.NewRectangle(color.Black))
	w.Show()
	for _, factor := range []float32{2, 1, 2} {
		backing = factor
		w.Canvas().Content().Refresh()
		physical := fyne.NewSize(float32(math.Ceil(float64(w.Canvas().Size().Width)))*backing,
			float32(math.Ceil(float64(w.Canvas().Size().Height)))*backing)
		if physical != fyne.NewSize(1280, 800) {
			t.Fatalf("backing=%v canvas scale=%v frame=%v", backing, w.Canvas().Scale(), physical)
		}
	}
	w.Close()
}
