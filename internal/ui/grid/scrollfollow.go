package grid

import (
	"image/color"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// scrollFollower receives only wheel/trackpad input. All other pointer
// interfaces belong to the cells, marquee catcher and native scrollbar.
type scrollFollower struct {
	widget.BaseWidget
	g *Overview
}

var _ fyne.Scrollable = (*scrollFollower)(nil)

func newScrollFollower(g *Overview) *scrollFollower {
	f := &scrollFollower{g: g}
	f.ExtendBaseWidget(f)
	return f
}

func (f *scrollFollower) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(canvas.NewRectangle(color.Transparent))
}

func (f *scrollFollower) Scrolled(ev *fyne.ScrollEvent) {
	g := f.g
	if !g.visible || g.count() == 0 {
		return
	}
	before := g.wrap.GetScrollOffset()
	g.wrap.ScrollToOffset(before - ev.Scrolled.DY)
	after := g.wrap.GetScrollOffset()
	if after == before {
		return
	}
	direction := 1
	if after < before {
		direction = -1
	}
	id := g.viewportHighlight(g.highlight, direction)
	if id != g.highlight {
		g.setHighlight(id)
	}
}

// viewportHighlight retains id only while its entire cell fits, matching
// GridWrap.Highlight's reveal rule. Otherwise it keeps the column on the
// nearest fully visible row, so highlighting cannot undo the scroll. A
// scroll direction chooses the leading edge; zero chooses the nearest edge
// to id when restoring a cell after a filter reflow.
func (g *Overview) viewportHighlight(id, direction int) int {
	if g.count() == 0 {
		return 0
	}
	id = max(0, min(id, g.count()-1))
	cols := g.wrap.ColumnCount()
	pad := g.wrap.Theme().Size(theme.SizeNamePadding)
	pitch := float64(cellSize + pad)
	offset := float64(g.wrap.GetScrollOffset())
	height := float64(g.wrap.Size().Height)
	first := int(math.Ceil(offset / pitch))
	last := int(math.Floor((offset + height - cellSize) / pitch))
	if first <= id/cols && id/cols <= last {
		return id
	}
	if first > last {
		// A short viewport can intersect cells without containing a full
		// row. Highlight may make its minimum corrective reveal here.
		first = int(math.Floor(offset / pitch))
		if float64(first)*pitch+cellSize <= offset {
			first++
		}
		last = int(math.Ceil((offset+height)/pitch)) - 1
	}
	row := max(first, min(id/cols, last))
	if direction > 0 {
		row = first
	} else if direction < 0 {
		row = last
	}
	row = max(0, min(row, (g.count()-1)/cols))
	return min(row*cols+id%cols, g.count()-1)
}
