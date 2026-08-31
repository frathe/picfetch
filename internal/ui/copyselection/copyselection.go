// Package copyselection owns the transient interaction for selecting an
// image-space rectangle to copy, including crop and PNG encode of the
// captured source. It is viewer-independent: callers provide current
// presentation geometry, a Source, and callbacks for the three effects that
// cross the package boundary.
package copyselection

import (
	"image"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

// View maps the oriented image bounds into the selection overlay's canvas
// coordinate space.
type View struct {
	ImageBounds image.Rectangle
	Position    fyne.Position
	Size        fyne.Size
}

// State is the viewer-facing Copy Selection mode state.
type State struct {
	Active       bool
	Busy         bool
	HasSelection bool
}

// Callbacks are the effects Copy Selection delegates to its viewer adapter.
// All callbacks are optional.
type Callbacks struct {
	Copy   func(image.Rectangle)
	Ended  func()
	Scroll func(*fyne.ScrollEvent)
}

// Feature owns Copy Selection mode state, its overlay, and the captured
// source Encode uses to produce PNG bytes.
type Feature struct {
	callbacks Callbacks
	overlay   *fyne.Container
	input     *inputArea
	button    *widget.Button

	view      View
	source    Source
	encode    func(image.Image, image.Rectangle) ([]byte, error)
	state     State
	committed rectF
	gesture   gesture
}

// New constructs an inactive Copy Selection feature.
func New(callbacks Callbacks) *Feature {
	f := &Feature{callbacks: callbacks}
	f.input = newInputArea(f)
	f.button = widget.NewButton(lang.L("Copy to clipboard"), func() { f.requestCopy() })
	f.button.Hide()
	f.overlay = container.New(&overlayLayout{}, f.input, f.button)
	f.overlay.Hide()
	return f
}

// Overlay returns the single canvas object the viewer composes.
func (f *Feature) Overlay() fyne.CanvasObject { return f.overlay }

// Start begins a fresh Copy Selection mode for view, holding source for Encode.
func (f *Feature) Start(view View, source Source) {
	if f.state.Active || !validView(view) {
		return
	}

	f.view = view
	f.source = source
	f.committed = rectF{}
	f.gesture = gesture{}
	f.state = State{Active: true}
	f.overlay.Show()
	f.syncChrome()
}

// ViewChanged updates presentation geometry without changing selected image
// coordinates. ImageBounds stay those captured at Start.
func (f *Feature) ViewChanged(view View) {
	if !f.state.Active {
		return
	}
	view.ImageBounds = f.view.ImageBounds
	if !validView(view) {
		return
	}
	f.view = view
	f.syncChrome()
}

// Cancel ends an idle Copy Selection mode without requesting a copy.
func (f *Feature) Cancel() {
	if !f.state.Active || f.state.Busy {
		return
	}
	f.end()
}

// State reports the viewer-facing mode state.
func (f *Feature) State() State { return f.state }

// HandleKey handles or suppresses a key while Copy Selection owns input.
// It returns true for keys the mode consumes: Escape, Return/Enter, image
// navigation, and every key while a copy is pending. Unowned keys return
// false so the viewer can yield or keep the mode.
func (f *Feature) HandleKey(key fyne.KeyName) bool {
	if !f.state.Active {
		return false
	}
	if f.state.Busy {
		return true
	}

	switch key {
	case fyne.KeyEscape:
		f.Cancel()
		return true
	case fyne.KeyReturn, fyne.KeyEnter:
		f.requestCopy()
		return true
	case fyne.KeyLeft, fyne.KeyRight, fyne.KeyUp, fyne.KeyDown, fyne.KeyHome, fyne.KeyEnd:
		return true
	}
	return false
}

// Complete reports the result of the viewer's asynchronous copy request.
func (f *Feature) Complete(err error) {
	if !f.state.Active || !f.state.Busy {
		return
	}
	if err != nil {
		f.state.Busy = false
		f.syncChrome()
		return
	}
	f.end()
}

// Encode returns a zero-origin PNG of bounds from the source captured at Start.
func (f *Feature) Encode(bounds image.Rectangle) ([]byte, error) {
	if f.encode == nil {
		return f.source.Encode(bounds)
	}
	pixels, err := f.source.pixels()
	if err != nil {
		return nil, err
	}
	crop, err := f.source.cropBounds(bounds, pixels.Bounds())
	if err != nil {
		return nil, err
	}
	return f.encode(pixels, crop)
}

// SetEncode replaces PNG encoding after crop. Production leaves this unset.
func (f *Feature) SetEncode(fn func(image.Image, image.Rectangle) ([]byte, error)) {
	f.encode = fn
}

func (f *Feature) requestCopy() {
	if !f.state.HasSelection || f.callbacks.Copy == nil {
		return
	}

	bounds, ok := f.pixelBounds(f.committed)
	if !ok {
		return
	}
	f.state.Busy = true
	f.syncChrome()
	f.callbacks.Copy(bounds)
}

func (f *Feature) end() {
	f.state = State{}
	f.source = Source{}
	f.committed = rectF{}
	f.gesture = gesture{}
	f.overlay.Hide()
	f.syncChrome()
	if f.callbacks.Ended != nil {
		f.callbacks.Ended()
	}
}

func (f *Feature) syncChrome() {
	if f.button == nil {
		return
	}
	if f.state.HasSelection {
		f.button.Show()
	} else {
		f.button.Hide()
	}
	if f.state.Busy {
		f.button.Disable()
	} else {
		f.button.Enable()
	}
	if f.input != nil {
		f.input.Refresh()
	}
	if f.overlay != nil {
		f.overlay.Refresh()
	}
}
