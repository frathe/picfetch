package copyselection

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"

	"github.com/frathe/picfetch/internal/ui/widgets"
)

type overlayLayout struct{}

func (overlayLayout) MinSize(_ []fyne.CanvasObject) fyne.Size {
	return fyne.NewSize(0, 0)
}

func (overlayLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) == 0 {
		return
	}
	objects[0].Move(fyne.NewPos(0, 0))
	objects[0].Resize(size)
	if len(objects) < 2 || !objects[1].Visible() {
		return
	}
	button := objects[1]
	pad := theme.Size(theme.SizeNameInnerPadding)
	buttonSize := button.MinSize()
	button.Resize(buttonSize)
	button.Move(fyne.NewPos(size.Width-buttonSize.Width-pad, size.Height-buttonSize.Height-pad))
}

type overlayRenderer struct {
	input   *inputArea
	dims    [4]*canvas.Rectangle
	border  *canvas.Rectangle
	handles [8]*canvas.Rectangle
	objects []fyne.CanvasObject
}

func newOverlayRenderer(input *inputArea) *overlayRenderer {
	r := &overlayRenderer{input: input, border: widgets.NewFocusRing(widgets.ButtonRingWidth, 0)}
	r.border.Hide()
	for i := range r.dims {
		dim := canvas.NewRectangle(widgets.ScrimColor)
		dim.Hide()
		r.dims[i] = dim
		r.objects = append(r.objects, dim)
	}
	r.objects = append(r.objects, r.border)
	for i := range r.handles {
		handle := canvas.NewRectangle(theme.Color(theme.ColorNamePrimary))
		handle.Hide()
		r.handles[i] = handle
		r.objects = append(r.objects, handle)
	}
	return r
}

func (r *overlayRenderer) Destroy() {}

func (r *overlayRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *overlayRenderer) MinSize() fyne.Size { return fyne.NewSize(0, 0) }

func (r *overlayRenderer) Layout(_ fyne.Size) { r.layoutChrome() }

func (r *overlayRenderer) Refresh() {
	r.border.StrokeColor = theme.Color(theme.ColorNamePrimary)
	for i := range r.handles {
		r.handles[i].FillColor = theme.Color(theme.ColorNamePrimary)
	}
	r.layoutChrome()
	canvas.Refresh(r.input)
}

func (r *overlayRenderer) layoutChrome() {
	f := r.input.feature
	rect, ok := f.displayRect()
	if !ok {
		r.hideAll()
		return
	}

	selPos, selSize := f.canvasRect(rect)
	r.border.Move(selPos)
	r.border.Resize(selSize)
	r.border.Show()

	imgPos := f.view.Position
	imgSize := f.view.Size
	layoutDim(r.dims[0], imgPos, fyne.NewSize(imgSize.Width, selPos.Y-imgPos.Y))
	layoutDim(r.dims[1], fyne.NewPos(imgPos.X, selPos.Y+selSize.Height), fyne.NewSize(imgSize.Width, imgPos.Y+imgSize.Height-(selPos.Y+selSize.Height)))
	layoutDim(r.dims[2], fyne.NewPos(imgPos.X, selPos.Y), fyne.NewSize(selPos.X-imgPos.X, selSize.Height))
	layoutDim(r.dims[3], fyne.NewPos(selPos.X+selSize.Width, selPos.Y), fyne.NewSize(imgPos.X+imgSize.Width-(selPos.X+selSize.Width), selSize.Height))

	showHandles := f.state.HasSelection && !f.gesture.drawing
	extent := handleExtent()
	for i, center := range f.handleCenters(rect) {
		if !showHandles {
			r.handles[i].Hide()
			continue
		}
		r.handles[i].Move(fyne.NewPos(center.X-extent/2, center.Y-extent/2))
		r.handles[i].Resize(fyne.NewSize(extent, extent))
		r.handles[i].Show()
	}
}

func layoutDim(rect *canvas.Rectangle, pos fyne.Position, size fyne.Size) {
	if size.Width <= 0 || size.Height <= 0 {
		rect.Hide()
		return
	}
	rect.Move(pos)
	rect.Resize(size)
	rect.Show()
}

func (r *overlayRenderer) hideAll() {
	r.border.Hide()
	for i := range r.dims {
		r.dims[i].Hide()
	}
	for i := range r.handles {
		r.handles[i].Hide()
	}
}

func themeSizePadding() float32 {
	return theme.Size(theme.SizeNamePadding)
}

func (f *Feature) cursorAt(pos fyne.Position) desktop.Cursor {
	if !f.state.Active || f.state.Busy {
		return desktop.DefaultCursor
	}
	if !f.canvasPointInImage(pos) {
		return desktop.DefaultCursor
	}
	if handle := f.handleAt(pos); handle != handleNone {
		return handleCursor(handle)
	}
	if f.state.HasSelection && f.canvasPointInRect(pos, f.committed) {
		return desktop.PointerCursor
	}
	return desktop.CrosshairCursor
}

func handleCursor(handle handleKind) desktop.Cursor {
	switch handle {
	case handleN, handleS:
		return desktop.VResizeCursor
	case handleE, handleW:
		return desktop.HResizeCursor
	case handleNE, handleSW:
		return desktop.NESWResizeCursor
	case handleNW, handleSE:
		return desktop.NWSEResizeCursor
	default:
		return desktop.DefaultCursor
	}
}
