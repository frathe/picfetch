// Package screenshots constrains PicFetch-owned windows for a screenshot launch.
package screenshots

import (
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/frathe/picfetch/internal/winpos"
)

type application struct {
	fyne.App
	width, height int
}

// App returns an application whose windows share physical screenshot dimensions.
// The caller supplies a validated launch preset. The native driver remains owned
// by the original application; lifecycle and persistence operations pass through.
func App(app fyne.App, width, height int) fyne.App {
	return &application{App: app, width: width, height: height}
}

func (a *application) NewWindow(title string) fyne.Window {
	w := &window{Window: a.App.NewWindow(title), app: a.App, width: a.width, height: a.height}
	w.Window.SetContent(container.New(fixedLayout{}))
	w.Window.SetPadded(false)
	w.Window.Resize(fyne.NewSize(float32(a.width), float32(a.height)))
	w.Window.SetFixedSize(true)
	return w
}

type window struct {
	fyne.Window
	app           fyne.App
	width, height int
	content       fyne.CanvasObject
}

// fixedLayout keeps a panel's natural minimum from expanding the native window.
// Its children still get normal layout inside the available screenshot surface.
type fixedLayout struct{}

func (fixedLayout) MinSize(_ []fyne.CanvasObject) fyne.Size { return fyne.Size{} }
func (fixedLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, object := range objects {
		object.Move(fyne.Position{})
		object.Resize(size)
	}
}

func (w *window) Resize(_ fyne.Size)          {}
func (w *window) SetFullScreen(_ bool)        {}
func (w *window) SetFixedSize(_ bool)         {}
func (w *window) RequestFullScreenSecondary() {}
func (w *window) Content() fyne.CanvasObject  { return w.content }
func (w *window) SetContent(content fyne.CanvasObject) {
	w.content = content
	root := container.New(fixedLayout{})
	if content != nil {
		root.Add(content)
	}
	w.Window.SetContent(root)
}
func (w *window) Show() {
	w.Window.Show()
	size, ok := winpos.ScreenshotContentSize(w.Window, w.width, w.height)
	if !ok {
		// The test driver has no chrome or backing scale. Native unsupported
		// backends retain the bounded canvas size and report the missing metric.
		if _, native := w.Window.(driver.NativeWindow); native {
			fyne.LogError("screenshot window geometry unavailable", nil)
		}
		size = fyne.NewSize(float32(w.width), float32(w.height))
	}
	scale := w.Window.Canvas().Scale()
	if scale <= 0 {
		scale = 1
	}
	// Fyne rounds native coordinates upward. Bias by one float step so division
	// by a fractional UI scale cannot add a physical pixel at that boundary.
	logical := fyne.NewSize(math.Nextafter32(size.Width/scale, 0), math.Nextafter32(size.Height/scale, 0))
	if !ok {
		logical = size
	} // exact dimensions for the headless test driver
	w.Window.SetFixedSize(false)
	w.Window.Resize(logical)
	w.Window.SetFixedSize(true)
}
func (w *window) ShowAndRun() { w.Show(); w.app.Run() }
func (w *window) RunNative(fn func(any)) {
	if native, ok := w.Window.(driver.NativeWindow); ok {
		native.RunNative(fn)
	}
}
func (w *window) RequestPosition(x, y int) {
	if native, ok := w.Window.(desktop.Window); ok {
		native.RequestPosition(x, y)
	}
}
func (w *window) RequestAlwaysOnTop() {
	if native, ok := w.Window.(desktop.Window); ok {
		native.RequestAlwaysOnTop()
	}
}
