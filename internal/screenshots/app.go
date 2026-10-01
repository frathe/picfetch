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
	w := &window{Window: a.App.NewWindow(title), app: a.App, width: a.width, height: a.height, measure: winpos.ScreenshotContentSize}
	w.Window.SetContent(container.New(fixedLayout{window: w}))
	w.Window.Resize(fyne.NewSize(float32(a.width), float32(a.height)))
	w.Window.SetFixedSize(true)
	w.Window.SetOnClosed(func() {
		w.closed, w.shown = true, false
		if w.onClosed != nil {
			w.onClosed()
		}
	})
	return w
}

type window struct {
	fyne.Window
	app            fyne.App
	width, height  int
	content        fyne.CanvasObject
	shown, closed  bool
	sizing, queued bool
	warned         bool
	applied        fyne.Size
	onClosed       func()
	measure        func(fyne.Window, int, int) (fyne.Size, bool)
}

// fixedLayout keeps a panel's natural minimum from expanding the native window.
// Its children still get normal layout inside the available screenshot surface.
type fixedLayout struct{ window *window }

func (fixedLayout) MinSize(_ []fyne.CanvasObject) fyne.Size { return fyne.Size{} }
func (layout fixedLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, object := range objects {
		object.Move(fyne.Position{})
		object.Resize(size)
	}
	// Scale changes refresh content. Check native metrics too, because macOS
	// backing scale is independent of Canvas.Scale. Queue correction after layout.
	if w := layout.window; w != nil && w.shown && !w.closed && !w.sizing && !w.queued {
		if target := w.targetSize(); target != w.applied {
			w.queued = true
			fyne.Do(func() {
				w.queued = false
				if w.shown && !w.closed {
					w.applySize()
				}
			})
		}
	}
}

func (w *window) Resize(_ fyne.Size)          {}
func (w *window) SetFullScreen(_ bool)        {}
func (w *window) SetFixedSize(_ bool)         {}
func (w *window) RequestFullScreenSecondary() {}
func (w *window) Content() fyne.CanvasObject  { return w.content }
func (w *window) SetOnClosed(fn func())       { w.onClosed = fn }
func (w *window) Hide()                       { w.shown = false; w.Window.Hide() }
func (w *window) Close()                      { w.closed = true; w.Window.Close() }
func (w *window) SetContent(content fyne.CanvasObject) {
	w.content = content
	root := container.New(fixedLayout{window: w})
	if content != nil {
		root.Add(content)
	}
	w.Window.SetContent(root)
}
func (w *window) Show() {
	w.Window.Show()
	w.shown = true
	w.applySize()
}
func (w *window) targetSize() fyne.Size {
	size, ok := w.measure(w.Window, w.width, w.height)
	_, native := w.Window.(driver.NativeWindow)
	if !ok {
		// Unknown chrome cannot be measured, but physical content pixels still
		// require scale conversion. Report the decoration limitation once.
		if native && !w.warned {
			fyne.LogError("screenshot window geometry unavailable", nil)
			w.warned = true
		}
		size = fyne.NewSize(float32(w.width), float32(w.height))
	}
	scale := w.Window.Canvas().Scale()
	if scale <= 0 {
		scale = 1
	}
	logical := fyne.NewSize(size.Width/scale, size.Height/scale)
	// Fyne rounds native coordinates upward. Avoid an extra pixel caused by
	// fractional floating-point division at that boundary.
	if native {
		logical = fyne.NewSize(math.Nextafter32(logical.Width, 0), math.Nextafter32(logical.Height, 0))
	}
	return logical
}
func (w *window) applySize() {
	if w.sizing || w.closed {
		return
	}
	w.sizing = true
	defer func() { w.sizing = false }()
	logical := w.targetSize()
	w.applied = logical
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
