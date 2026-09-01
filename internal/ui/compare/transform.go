package compare

import (
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"github.com/frathe/picfetch/internal/imaging"
)

const (
	compareZoomStep   = float32(1.25)
	minZoomFactor     = float32(0.05)
	maxZoomFactor     = float32(16)
	comparePanSlack   = float32(0.5)
	scrollSensitivity = float32(0.01)
)

type scaleMode uint8

const (
	fitRelative scaleMode = iota
	absoluteScale
)

type linkedTransform struct {
	center fyne.Position
	factor float32
	mode   scaleMode
}

func defaultLinkedTransform() linkedTransform {
	return linkedTransform{
		center: fyne.NewPos(0.5, 0.5),
		factor: 1,
		mode:   fitRelative,
	}
}

func newPane(feature *Feature, index int) pane {
	img := newPaneImage()
	input := newPaneInput(feature, index)
	viewport := container.New(paneImageLayout{feature: feature, index: index}, img, input)

	spinner := newPaneSpinner()
	return pane{
		root:    container.NewStack(viewport, container.NewCenter(spinner)),
		image:   img,
		input:   input,
		spinner: spinner,
	}
}

type paneImageLayout struct {
	feature *Feature
	index   int
}

func (l paneImageLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	if len(objects) > 1 {
		objects[1].Move(fyne.NewPos(0, 0))
		objects[1].Resize(size)
	}
	l.feature.viewports[l.index] = size
	l.feature.applyTransform()
}

func (paneImageLayout) MinSize(_ []fyne.CanvasObject) fyne.Size { return fyne.Size{} }

// HandleKey applies comparison-owned transform keys. The caller still owns
// Escape and F1, and swallows every unsupported key while comparison is active.
func (f *Feature) HandleKey(name fyne.KeyName) {
	if !f.active || !f.ready {
		return
	}
	if f.handleDividerKey(name) {
		return
	}

	switch name {
	case fyne.Key0:
		f.transform = defaultLinkedTransform()
	case fyne.KeyPlus, fyne.KeyEqual:
		f.transform.factor = min(f.transform.factor*compareZoomStep, maxZoomFactor)
	case fyne.KeyMinus:
		f.transform.factor = max(f.transform.factor/compareZoomStep, minZoomFactor)
	case fyne.Key1:
		f.transform = linkedTransform{
			center: fyne.NewPos(0.5, 0.5),
			factor: 1,
			mode:   absoluteScale,
		}
	default:
		return
	}

	f.applyTransform()
	f.repaint()
}

func (f *Feature) handleScroll(index int, ev *fyne.ScrollEvent) {
	if !f.active || !f.ready || ev == nil || index < 0 || index >= len(f.panes) {
		return
	}
	if f.callbacks.Modifiers != nil && f.callbacks.Modifiers()&fyne.KeyModifierShift != 0 {
		f.panBy(index, ev.Scrolled)
		return
	}
	if ev.Scrolled.DY == 0 {
		return
	}

	oldFactor := f.transform.factor
	newFactor := min(max(
		oldFactor*float32(math.Exp(float64(ev.Scrolled.DY*scrollSensitivity))),
		minZoomFactor,
	), maxZoomFactor)
	if newFactor == oldFactor {
		return
	}

	native := frameSize(f.loaded[index])
	oldScale := f.scaleFor(index, oldFactor)
	newScale := f.scaleFor(index, newFactor)
	oldScaled := fyne.NewSize(native.Width*oldScale, native.Height*oldScale)
	newScaled := fyne.NewSize(native.Width*newScale, native.Height*newScale)
	viewport := f.viewports[index]
	f.transform.center = fyne.NewPos(
		f.transform.center.X+(ev.Position.X-viewport.Width/2)*(1/oldScaled.Width-1/newScaled.Width),
		f.transform.center.Y+(ev.Position.Y-viewport.Height/2)*(1/oldScaled.Height-1/newScaled.Height),
	)
	f.transform.factor = newFactor
	f.applyTransform()
	f.repaint()
}

func (f *Feature) panBy(index int, delta fyne.Delta) {
	if !f.active || !f.ready || index < 0 || index >= len(f.panes) {
		return
	}
	native := frameSize(f.loaded[index])
	scale := f.scaleFor(index, f.transform.factor)
	scaled := fyne.NewSize(native.Width*scale, native.Height*scale)
	if !validViewport(scaled) {
		return
	}
	f.transform.center = fyne.NewPos(
		f.transform.center.X-delta.DX/scaled.Width,
		f.transform.center.Y-delta.DY/scaled.Height,
	)
	f.applyTransform()
	f.repaint()
}

func (f *Feature) applyTransform() {
	if !f.ready || !validViewport(f.viewports[0]) || !validViewport(f.viewports[1]) {
		return
	}
	f.clampCenter()

	for i := range f.panes {
		f.panes[i].input.Move(fyne.NewPos(0, 0))
		f.panes[i].input.Resize(f.viewports[i])
		native := frameSize(f.loaded[i])
		if !validViewport(native) {
			continue
		}
		scale := f.scaleFor(i, f.transform.factor)
		scaled := fyne.NewSize(native.Width*scale, native.Height*scale)
		position := fyne.NewPos(
			f.viewports[i].Width/2-f.transform.center.X*scaled.Width,
			f.viewports[i].Height/2-f.transform.center.Y*scaled.Height,
		)
		f.panes[i].image.Resize(scaled)
		f.panes[i].image.Move(position)
		f.requestVectorRender(i, scaled)
	}
}

func (f *Feature) clampCenter() {
	minX, maxX, minY, maxY := f.sharedPanRange()
	f.transform.center = fyne.NewPos(
		min(max(f.transform.center.X, minX), maxX),
		min(max(f.transform.center.Y, minY), maxY),
	)
}

func (f *Feature) sharedPanRange() (minX, maxX, minY, maxY float32) {
	minX, maxX = 0, 1
	minY, maxY = 0, 1
	for i := range f.panes {
		native := frameSize(f.loaded[i])
		scale := f.scaleFor(i, f.transform.factor)
		scaled := fyne.NewSize(native.Width*scale, native.Height*scale)
		lowX, highX := normalizedPanRange(scaled.Width, f.viewports[i].Width)
		lowY, highY := normalizedPanRange(scaled.Height, f.viewports[i].Height)
		minX, maxX = max(minX, lowX), min(maxX, highX)
		minY, maxY = max(minY, lowY), min(maxY, highY)
	}
	return minX, maxX, minY, maxY
}

func (f *Feature) canPan() bool {
	if !f.active || !f.ready || !validViewport(f.viewports[0]) || !validViewport(f.viewports[1]) {
		return false
	}
	minX, maxX, minY, maxY := f.sharedPanRange()
	return minX < maxX || minY < maxY
}

func normalizedPanRange(scaled, viewport float32) (float32, float32) {
	if scaled <= viewport+comparePanSlack {
		return 0.5, 0.5
	}
	halfVisible := viewport / (2 * scaled)
	return halfVisible, 1 - halfVisible
}

func (f *Feature) scaleFor(index int, factor float32) float32 {
	if f.transform.mode == fitRelative {
		return fitScale(frameSize(f.loaded[index]), f.viewports[index]) * factor
	}
	return factor
}

func frameSize(loaded *imaging.LoadedImage) fyne.Size {
	if loaded == nil || len(loaded.Frames) == 0 || loaded.Frames[0] == nil {
		return fyne.Size{}
	}
	bounds := loaded.Frames[0].Bounds()
	return fyne.NewSize(float32(bounds.Dx()), float32(bounds.Dy()))
}

func fitScale(native, viewport fyne.Size) float32 {
	return min(viewport.Width/native.Width, viewport.Height/native.Height)
}

func validViewport(size fyne.Size) bool {
	return size.Width > 0 && size.Height > 0
}

var _ fyne.Layout = paneImageLayout{}
