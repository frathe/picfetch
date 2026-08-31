package copyselection

import (
	"image"
	"math"

	"fyne.io/fyne/v2"
)

type pointF struct {
	x float64
	y float64
}

type rectF struct {
	min pointF
	max pointF
}

func normalizedRect(a, b pointF) rectF {
	return rectF{
		min: pointF{x: min(a.x, b.x), y: min(a.y, b.y)},
		max: pointF{x: max(a.x, b.x), y: max(a.y, b.y)},
	}
}

func validView(view View) bool {
	return !view.ImageBounds.Empty() &&
		view.Size.Width > 0 && view.Size.Height > 0 &&
		finite32(view.Position.X) && finite32(view.Position.Y) &&
		finite32(view.Size.Width) && finite32(view.Size.Height)
}

func finite32(v float32) bool {
	value := float64(v)
	return !math.IsInf(value, 0) && !math.IsNaN(value)
}

func (f *Feature) canvasToImage(pos fyne.Position) pointF {
	bounds := f.view.ImageBounds
	return pointF{
		x: float64(bounds.Min.X) + float64(pos.X-f.view.Position.X)*float64(bounds.Dx())/float64(f.view.Size.Width),
		y: float64(bounds.Min.Y) + float64(pos.Y-f.view.Position.Y)*float64(bounds.Dy())/float64(f.view.Size.Height),
	}
}

func (f *Feature) clampImagePoint(p pointF) pointF {
	bounds := f.view.ImageBounds
	p.x = max(float64(bounds.Min.X), min(p.x, float64(bounds.Max.X)))
	p.y = max(float64(bounds.Min.Y), min(p.y, float64(bounds.Max.Y)))
	return p
}

func (f *Feature) canvasPointInImage(pos fyne.Position) bool {
	return pos.X >= f.view.Position.X && pos.Y >= f.view.Position.Y &&
		pos.X < f.view.Position.X+f.view.Size.Width &&
		pos.Y < f.view.Position.Y+f.view.Size.Height
}

func (f *Feature) validSelection(r rectF) bool {
	return r.max.x-r.min.x >= 1 && r.max.y-r.min.y >= 1
}

func (f *Feature) pixelBounds(r rectF) (image.Rectangle, bool) {
	bounds := f.view.ImageBounds
	out := image.Rect(
		max(bounds.Min.X, int(math.Floor(r.min.x))),
		max(bounds.Min.Y, int(math.Floor(r.min.y))),
		min(bounds.Max.X, int(math.Ceil(r.max.x))),
		min(bounds.Max.Y, int(math.Ceil(r.max.y))),
	)
	return out, f.validSelection(r) && !out.Empty()
}

func (f *Feature) imageToCanvas(p pointF) fyne.Position {
	bounds := f.view.ImageBounds
	if bounds.Empty() || f.view.Size.Width == 0 || f.view.Size.Height == 0 {
		return f.view.Position
	}
	return fyne.NewPos(
		f.view.Position.X+float32((p.x-float64(bounds.Min.X))*float64(f.view.Size.Width)/float64(bounds.Dx())),
		f.view.Position.Y+float32((p.y-float64(bounds.Min.Y))*float64(f.view.Size.Height)/float64(bounds.Dy())),
	)
}

func (f *Feature) canvasRect(r rectF) (fyne.Position, fyne.Size) {
	min := f.imageToCanvas(r.min)
	max := f.imageToCanvas(r.max)
	return min, fyne.NewSize(max.X-min.X, max.Y-min.Y)
}

func (f *Feature) canvasPointInRect(pos fyne.Position, r rectF) bool {
	p, s := f.canvasRect(r)
	return pos.X >= p.X && pos.Y >= p.Y && pos.X < p.X+s.Width && pos.Y < p.Y+s.Height
}

func (f *Feature) displayRect() (rectF, bool) {
	if f.gesture.drawing || f.gesture.moving || f.gesture.resizing {
		return f.gesture.candidate, true
	}
	if f.state.HasSelection {
		return f.committed, true
	}
	return rectF{}, false
}

type handleKind int

const (
	handleNone handleKind = iota
	handleNW
	handleN
	handleNE
	handleE
	handleSE
	handleS
	handleSW
	handleW
)

func handleExtent() float32 {
	size := themeSizePadding()
	if size < 8 {
		return 8
	}
	return size
}

func (f *Feature) handleCenters(r rectF) [8]fyne.Position {
	pos, size := f.canvasRect(r)
	x0, y0 := pos.X, pos.Y
	x1, y1 := pos.X+size.Width, pos.Y+size.Height
	xm, ym := (x0+x1)/2, (y0+y1)/2
	return [8]fyne.Position{
		fyne.NewPos(x0, y0),
		fyne.NewPos(xm, y0),
		fyne.NewPos(x1, y0),
		fyne.NewPos(x1, ym),
		fyne.NewPos(x1, y1),
		fyne.NewPos(xm, y1),
		fyne.NewPos(x0, y1),
		fyne.NewPos(x0, ym),
	}
}

func (f *Feature) handleAt(pos fyne.Position) handleKind {
	if !f.state.HasSelection || f.gesture.drawing {
		return handleNone
	}
	half := handleExtent() / 2
	for i, center := range f.handleCenters(f.committed) {
		if pos.X >= center.X-half && pos.X < center.X+half &&
			pos.Y >= center.Y-half && pos.Y < center.Y+half {
			return handleKind(i + 1)
		}
	}
	return handleNone
}

func moveRect(start rectF, grab, at pointF, bounds image.Rectangle) rectF {
	width := start.max.x - start.min.x
	height := start.max.y - start.min.y
	min := pointF{x: start.min.x + at.x - grab.x, y: start.min.y + at.y - grab.y}
	maxX := float64(bounds.Max.X)
	maxY := float64(bounds.Max.Y)
	minX := float64(bounds.Min.X)
	minY := float64(bounds.Min.Y)
	if min.x < minX {
		min.x = minX
	}
	if min.y < minY {
		min.y = minY
	}
	if min.x+width > maxX {
		min.x = maxX - width
	}
	if min.y+height > maxY {
		min.y = maxY - height
	}
	return rectF{min: min, max: pointF{x: min.x + width, y: min.y + height}}
}

func resizeRect(start rectF, handle handleKind, at pointF) rectF {
	r := start
	switch handle {
	case handleNW:
		r.min = at
	case handleN:
		r.min.y = at.y
	case handleNE:
		r.min.y = at.y
		r.max.x = at.x
	case handleE:
		r.max.x = at.x
	case handleSE:
		r.max = at
	case handleS:
		r.max.y = at.y
	case handleSW:
		r.min.x = at.x
		r.max.y = at.y
	case handleW:
		r.min.x = at.x
	}
	return normalizedRect(r.min, r.max)
}
