package compare

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// inputShield is the comparison surface's transparent pointer target. Fyne
// dispatches each pointer event to the topmost object implementing the
// matching interface; implementing the complete pointer set here prevents a
// tap, drag, wheel, or hover from falling through non-interactive comparison
// images into the still-open grid/viewer underneath. Toolbar buttons are
// painted above it and remain the topmost tappable objects at their positions.
type inputShield struct {
	widget.BaseWidget
}

var (
	_ fyne.Tappable          = (*inputShield)(nil)
	_ fyne.SecondaryTappable = (*inputShield)(nil)
	_ fyne.DoubleTappable    = (*inputShield)(nil)
	_ fyne.Draggable         = (*inputShield)(nil)
	_ fyne.Scrollable        = (*inputShield)(nil)
	_ desktop.Mouseable      = (*inputShield)(nil)
	_ desktop.Hoverable      = (*inputShield)(nil)
	_ desktop.Cursorable     = (*inputShield)(nil)
)

func newInputShield() *inputShield {
	shield := &inputShield{}
	shield.ExtendBaseWidget(shield)
	return shield
}

func (s *inputShield) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (*inputShield) Tapped(*fyne.PointEvent)          {}
func (*inputShield) TappedSecondary(*fyne.PointEvent) {}
func (*inputShield) DoubleTapped(*fyne.PointEvent)    {}
func (*inputShield) Dragged(*fyne.DragEvent)          {}
func (*inputShield) DragEnd()                         {}
func (*inputShield) Scrolled(*fyne.ScrollEvent)       {}
func (*inputShield) MouseDown(*desktop.MouseEvent)    {}
func (*inputShield) MouseUp(*desktop.MouseEvent)      {}
func (*inputShield) MouseIn(*desktop.MouseEvent)      {}
func (*inputShield) MouseMoved(*desktop.MouseEvent)   {}
func (*inputShield) MouseOut()                        {}

func (*inputShield) Cursor() desktop.Cursor { return desktop.DefaultCursor }

// paneInput is the interactive layer inside one clipped image viewport. It
// forwards pointer intent to the Feature, which owns the single transform
// shared by both panes.
type paneInput struct {
	widget.BaseWidget

	feature *Feature
	index   int
}

var (
	_ fyne.Draggable     = (*paneInput)(nil)
	_ fyne.Scrollable    = (*paneInput)(nil)
	_ desktop.Cursorable = (*paneInput)(nil)
)

func newPaneInput(feature *Feature, index int) *paneInput {
	input := &paneInput{feature: feature, index: index}
	input.ExtendBaseWidget(input)
	return input
}

func (p *paneInput) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (p *paneInput) Scrolled(ev *fyne.ScrollEvent) {
	p.feature.handleScroll(p.index, ev)
}

func (p *paneInput) Dragged(ev *fyne.DragEvent) {
	if ev != nil {
		p.feature.panBy(p.index, ev.Dragged)
	}
}

func (*paneInput) DragEnd() {}

func (p *paneInput) Cursor() desktop.Cursor {
	if p.feature.canPan() {
		return desktop.PointerCursor
	}
	return desktop.DefaultCursor
}

// swipeDivider is the only pointer target that changes the reveal boundary.
// Its renderer paints a narrow full-height line inside a theme-sized hit area.
type swipeDivider struct {
	widget.BaseWidget

	feature *Feature
}

var (
	_ fyne.Draggable     = (*swipeDivider)(nil)
	_ desktop.Cursorable = (*swipeDivider)(nil)
)

func newSwipeDivider(feature *Feature) *swipeDivider {
	divider := &swipeDivider{feature: feature}
	divider.ExtendBaseWidget(divider)
	return divider
}

func (d *swipeDivider) CreateRenderer() fyne.WidgetRenderer {
	background := canvas.NewRectangle(theme.Color(theme.ColorNameShadow))
	line := canvas.NewRectangle(theme.Color(theme.ColorNameForeground))
	return &swipeDividerRenderer{
		divider:    d,
		background: background,
		line:       line,
		objects:    []fyne.CanvasObject{background, line},
	}
}

func (d *swipeDivider) Dragged(ev *fyne.DragEvent) {
	if ev != nil {
		d.feature.dragDivider(ev.Dragged.DX)
	}
}

func (*swipeDivider) DragEnd() {}

func (*swipeDivider) Cursor() desktop.Cursor { return desktop.HResizeCursor }

type swipeDividerRenderer struct {
	divider    *swipeDivider
	background *canvas.Rectangle
	line       *canvas.Rectangle
	objects    []fyne.CanvasObject
}

func (*swipeDividerRenderer) Destroy() {}

func (r *swipeDividerRenderer) Layout(size fyne.Size) {
	r.background.Resize(size)
	lineWidth := theme.Size(theme.SizeNameSeparatorThickness)
	if lineWidth <= 0 {
		lineWidth = 1
	}
	r.line.Move(fyne.NewPos((size.Width-lineWidth)/2, 0))
	r.line.Resize(fyne.NewSize(lineWidth, size.Height))
}

func (r *swipeDividerRenderer) MinSize() fyne.Size {
	return fyne.NewSize(dividerThickness(), 0)
}

func (r *swipeDividerRenderer) Objects() []fyne.CanvasObject { return r.objects }

func (r *swipeDividerRenderer) Refresh() {
	r.background.FillColor = theme.Color(theme.ColorNameShadow)
	r.background.Refresh()
	r.line.FillColor = theme.Color(theme.ColorNameForeground)
	r.line.Refresh()
	r.Layout(r.divider.Size())
}

var _ fyne.WidgetRenderer = (*swipeDividerRenderer)(nil)
