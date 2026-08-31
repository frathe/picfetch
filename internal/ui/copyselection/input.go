package copyselection

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/widget"
)

type gesture struct {
	drawing   bool
	moving    bool
	resizing  bool
	handle    handleKind
	origin    pointF
	start     rectF
	candidate rectF
}

// inputArea is the one full-overlay pointer target. Keeping the event surface
// separate from the visuals lets the latter remain renderer details.
type inputArea struct {
	widget.BaseWidget
	feature *Feature
	pressed bool
	pressAt fyne.Position
	hover   fyne.Position
}

var (
	_ fyne.Draggable     = (*inputArea)(nil)
	_ desktop.Mouseable  = (*inputArea)(nil)
	_ desktop.Hoverable  = (*inputArea)(nil)
	_ fyne.Scrollable    = (*inputArea)(nil)
	_ desktop.Cursorable = (*inputArea)(nil)
)

func newInputArea(feature *Feature) *inputArea {
	input := &inputArea{feature: feature}
	input.ExtendBaseWidget(input)
	return input
}

func (i *inputArea) CreateRenderer() fyne.WidgetRenderer {
	return newOverlayRenderer(i)
}

func (i *inputArea) MouseDown(event *desktop.MouseEvent) {
	if event.Button != desktop.MouseButtonPrimary || !i.feature.state.Active || i.feature.state.Busy {
		return
	}
	i.pressed = true
	i.pressAt = event.Position
}

func (i *inputArea) MouseUp(_ *desktop.MouseEvent) {
	i.pressed = false
}

func (i *inputArea) Dragged(event *fyne.DragEvent) {
	f := i.feature
	if !f.state.Active || f.state.Busy {
		return
	}
	if !f.gesture.drawing && !f.gesture.moving && !f.gesture.resizing {
		origin := event.Position.Subtract(fyne.NewPos(event.Dragged.DX, event.Dragged.DY))
		if i.pressed {
			origin = i.pressAt
		}
		handle := f.handleAt(origin)
		switch {
		case handle != handleNone:
			f.gesture.resizing = true
			f.gesture.handle = handle
			f.gesture.start = f.committed
			f.gesture.candidate = f.committed
		case f.state.HasSelection && f.canvasPointInRect(origin, f.committed):
			f.gesture.moving = true
			f.gesture.origin = f.clampImagePoint(f.canvasToImage(origin))
			f.gesture.start = f.committed
			f.gesture.candidate = f.committed
		case f.canvasPointInImage(origin):
			f.gesture.drawing = true
			f.gesture.origin = f.clampImagePoint(f.canvasToImage(origin))
		default:
			return
		}
	}

	at := f.clampImagePoint(f.canvasToImage(event.Position))
	switch {
	case f.gesture.moving:
		f.gesture.candidate = moveRect(f.gesture.start, f.gesture.origin, at, f.view.ImageBounds)
	case f.gesture.resizing:
		f.gesture.candidate = resizeRect(f.gesture.start, f.gesture.handle, at)
	default:
		f.gesture.candidate = normalizedRect(f.gesture.origin, at)
	}
	i.Refresh()
}

func (i *inputArea) DragEnd() {
	f := i.feature
	if !f.gesture.drawing && !f.gesture.moving && !f.gesture.resizing {
		return
	}
	if f.validSelection(f.gesture.candidate) {
		f.committed = f.gesture.candidate
		f.state.HasSelection = true
	}
	f.gesture = gesture{}
	i.pressed = false
	f.syncChrome()
}

func (i *inputArea) Scrolled(event *fyne.ScrollEvent) {
	if i.feature.state.Active && !i.feature.state.Busy && i.feature.callbacks.Scroll != nil {
		i.feature.callbacks.Scroll(event)
	}
}

func (i *inputArea) Cursor() desktop.Cursor {
	return i.feature.cursorAt(i.hover)
}

func (i *inputArea) MouseIn(event *desktop.MouseEvent) {
	i.hover = event.Position
}

func (i *inputArea) MouseMoved(event *desktop.MouseEvent) {
	i.hover = event.Position
}

func (i *inputArea) MouseOut() {
	i.hover = fyne.Position{}
}
