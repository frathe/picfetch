package grid

import (
	"image/color"
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// marqueeRect is an axis-aligned rectangle in wrap-content coordinates
// (origin at the top-left of cell 0, y increasing down, scroll already added).
type marqueeRect struct {
	minX, minY, maxX, maxY float32
}

func normRect(a, b fyne.Position) marqueeRect {
	return marqueeRect{
		minX: min(a.X, b.X),
		minY: min(a.Y, b.Y),
		maxX: max(a.X, b.X),
		maxY: max(a.Y, b.Y),
	}
}

// marqueeGrid is the laid-out cell lattice cellsIntersecting tests against.
// cols is GridWrap.ColumnCount(); cell and pad must match itemMin and
// theme padding (GridWrap lays out at pitch cell+pad).
type marqueeGrid struct {
	cols, count int
	cell, pad   float32
}

func (g marqueeGrid) pitch() float32 { return g.cell + g.pad }

// cellsIntersecting returns display indices whose cell boxes overlap r,
// ascending. A rect that only covers the padding gutter between cells
// selects nothing. A purely horizontal or vertical drag is a line
// (max == min on one axis); inflate that axis by 1px so it still hits
// the row or column the pointer is in.
func cellsIntersecting(r marqueeRect, g marqueeGrid) []int {
	if g.cols < 1 || g.count < 1 || g.cell <= 0 {
		return nil
	}
	if r.maxX == r.minX {
		r.maxX++
	}
	if r.maxY == r.minY {
		r.maxY++
	}

	pitch := g.pitch()
	rows := (g.count + g.cols - 1) / g.cols

	col0 := max(0, int(r.minX/pitch))
	col1 := min(g.cols-1, int(r.maxX/pitch))
	row0 := max(0, int(r.minY/pitch))
	row1 := min(rows-1, int(r.maxY/pitch))

	out := make([]int, 0)
	for row := row0; row <= row1; row++ {
		for col := col0; col <= col1; col++ {
			id := row*g.cols + col
			if id < 0 || id >= g.count {
				continue
			}
			x0 := float32(col) * pitch
			y0 := float32(row) * pitch
			x1 := x0 + g.cell
			y1 := y0 + g.cell
			if r.maxX <= x0 || r.minX >= x1 || r.maxY <= y0 || r.minY >= y1 {
				continue
			}
			out = append(out, id)
		}
	}
	return out
}

func (g *Overview) applyMarquee(origin, at fyne.Position, add bool) {
	grid := g.marqueeLattice()
	hostIDs := make([]int, 0)
	for _, d := range cellsIntersecting(normRect(origin, at), grid) {
		if i := g.fileIndex(d); i >= 0 {
			hostIDs = append(hostIDs, i)
		}
	}

	if add {
		hostIDs = unionSorted(g.marqueeSaved, hostIDs)
	}

	changed := !slices.Equal(hostIDs, g.sel.Indices())
	if changed {
		g.sel.Replace(hostIDs)
	}

	if i := g.fileIndex(cellAtPoint(origin, grid)); i >= 0 {
		g.sel.SetAnchor(i)
	} else if len(hostIDs) > 0 {
		g.sel.SetAnchor(hostIDs[0])
	}

	if !changed {
		return
	}
	g.wrap.Refresh()
	g.syncTopBar()
	g.host.ForceRepaint()
	g.fireSelectionChanged()
}

func (g *Overview) marqueeLattice() marqueeGrid {
	return marqueeGrid{
		cols:  g.wrap.ColumnCount(),
		count: g.count(),
		cell:  cellSize,
		pad:   g.wrap.Theme().Size(theme.SizeNamePadding),
	}
}

func cellAtPoint(p fyne.Position, grid marqueeGrid) int {
	ids := cellsIntersecting(marqueeRect{minX: p.X, minY: p.Y, maxX: p.X + 1, maxY: p.Y + 1}, grid)
	if len(ids) == 0 {
		return -1
	}
	return ids[0]
}

func unionSorted(a, b []int) []int {
	seen := make(map[int]struct{}, len(a)+len(b))
	for _, xs := range [][]int{a, b} {
		for _, i := range xs {
			seen[i] = struct{}{}
		}
	}
	out := make([]int, 0, len(seen))
	for i := range seen {
		out = append(out, i)
	}
	slices.Sort(out)
	return out
}

// marqueeCatcher is the transparent drag target under the padded GridWrap.
// It is fyne.Draggable and nothing else: Tappable/Hoverable/Scrollable/
// Mouseable would steal cell taps, hover, wheel, or the scrollbar.
type marqueeCatcher struct {
	widget.BaseWidget
	g *Overview
}

var _ fyne.Draggable = (*marqueeCatcher)(nil)

func newMarqueeCatcher(g *Overview) *marqueeCatcher {
	c := &marqueeCatcher{g: g}
	c.ExtendBaseWidget(c)
	return c
}

func (c *marqueeCatcher) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (c *marqueeCatcher) Dragged(ev *fyne.DragEvent) { c.g.marqueeDragged(ev) }
func (c *marqueeCatcher) DragEnd()                   { c.g.marqueeDragEnd() }

func (g *Overview) marqueeDragged(ev *fyne.DragEvent) {
	if g.marqueeDisarmed || !g.visible {
		return
	}
	pad := g.wrap.Theme().Size(theme.SizeNamePadding)
	content := fyne.NewPos(
		ev.Position.X-pad,
		ev.Position.Y-pad+g.wrap.GetScrollOffset(),
	)
	if !g.marqueeDragging {
		g.marqueeOrigin = content.Subtract(fyne.NewPos(ev.Dragged.DX, ev.Dragged.DY))
		g.marqueeSaved = append([]int(nil), g.sel.Indices()...)
		g.marqueeDragging = true
	}

	add := false
	if toggle, extend := pickModifier(g.host.Modifiers()); toggle || extend {
		add = true
	}
	g.applyMarquee(g.marqueeOrigin, content, add)

	originCatcher := fyne.NewPos(
		g.marqueeOrigin.X+pad,
		g.marqueeOrigin.Y+pad-g.wrap.GetScrollOffset(),
	)
	g.placeMarqueeRect(originCatcher, ev.Position)
}

func (g *Overview) marqueeDragEnd() {
	if g.marqueeDisarmed || !g.marqueeDragging {
		g.marqueeDisarmed = false
		g.marqueeDragging = false
		g.marqueeSaved = nil
		return
	}
	g.marqueeDragging = false
	g.hideMarqueeRect()
	g.host.Unfocus()
	grid := g.marqueeLattice()
	if d := cellAtPoint(g.marqueeOrigin, grid); d >= 0 {
		g.setHighlight(d)
	} else if ids := g.sel.Indices(); len(ids) > 0 {
		if d := g.displayIndex(ids[0]); d >= 0 {
			g.setHighlight(d)
		}
	}
	g.marqueeSaved = nil
}

func (g *Overview) placeMarqueeRect(a, b fyne.Position) {
	r := normRect(a, b)
	g.marqueeRect.Move(fyne.NewPos(r.minX, r.minY))
	size := fyne.NewSize(r.maxX-r.minX, r.maxY-r.minY)
	if size.Width == 0 {
		size.Width = 1
	}
	if size.Height == 0 {
		size.Height = 1
	}
	g.marqueeRect.Resize(size)
	g.marqueeRect.Show()
	g.marqueeRect.Refresh()
}

func (g *Overview) hideMarqueeRect() {
	if g.marqueeRect == nil {
		return
	}
	g.marqueeRect.Hide()
	g.marqueeRect.Refresh()
}

func (g *Overview) cancelMarquee() {
	changed := !slices.Equal(g.sel.Indices(), g.marqueeSaved)
	g.sel.Replace(g.marqueeSaved)
	g.wrap.Refresh()
	g.syncTopBar()
	g.host.ForceRepaint()
	g.hideMarqueeRect()
	g.marqueeDisarmed = true
	g.marqueeDragging = false
	g.marqueeSaved = nil
	if changed {
		g.fireSelectionChanged()
	}
}
