package compare

import (
	"image"
	"math"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"

	"github.com/frathe/picfetch/internal/imaging"
)

const (
	compareZoomStep   = float32(1.25)
	minZoomFactor     = float32(0.05)
	maxZoomFactor     = float32(16)
	scrollSensitivity = float32(0.01)
)

type scaleMode uint8

const (
	fitRelative scaleMode = iota
	absoluteScale
)

type photoTransform struct {
	center fyne.Position
	factor float32
	mode   scaleMode
}

type cameraTransform struct {
	zoom float32
	// offset is measured from the pane center as a fraction of its viewport.
	// Both comparison panes use the same value, so a camera pan is identical
	// in points while their viewports have the same size.
	offset fyne.Position
}

func defaultPhotoTransform() photoTransform {
	return photoTransform{
		center: fyne.NewPos(0.5, 0.5),
		factor: 1,
		mode:   fitRelative,
	}
}

func defaultCameraTransform() cameraTransform {
	return cameraTransform{zoom: 1}
}

func newPane(feature *Feature, index int, renderer paneRenderer) pane {
	input := newPaneInput(feature, index)
	viewport := container.New(paneImageLayout{feature: feature, index: index}, renderer.Object(), input)

	spinner := newPaneSpinner()
	return pane{
		root:     container.NewStack(viewport, container.NewCenter(spinner)),
		renderer: renderer,
		input:    input,
		spinner:  spinner,
	}
}

type paneImageLayout struct {
	feature *Feature
	index   int
}

func (l paneImageLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	l.feature.viewports[l.index] = size
	if len(objects) > 1 {
		l.feature.layoutPaneInput(l.index, objects[1])
	}
	l.feature.applyTransform()
}

func (paneImageLayout) MinSize(_ []fyne.CanvasObject) fyne.Size { return fyne.Size{} }

// HandleKey applies comparison-owned transform keys. The caller still owns
// Escape and F1, and swallows every unsupported key while comparison is active.
func (f *Feature) HandleKey(name fyne.KeyName) {
	f.handleKey(name, f.currentModifiers())
}

func (f *Feature) handleKey(name fyne.KeyName, modifiers fyne.KeyModifier) {
	if !f.active || !f.ready {
		return
	}
	if f.handleDividerKey(name, modifiers) {
		return
	}

	if f.unlinked {
		if f.hoveredPane < 0 || f.hoveredPane >= len(f.photoTransforms) {
			return
		}
		if !f.applyPhotoKey(f.hoveredPane, name) {
			return
		}
		f.applyTransform()
		return
	}

	if !f.applyCameraKey(name) {
		return
	}

	f.applyTransform()
}

func (f *Feature) applyPhotoKey(index int, name fyne.KeyName) bool {
	transform := &f.photoTransforms[index]
	switch name {
	case fyne.Key0:
		*transform = photoTransform{factor: 1 / f.camera.zoom, mode: fitRelative}
		f.centerPhotoInCurrentCamera(index)
	case fyne.Key1:
		*transform = photoTransform{factor: 1 / f.camera.zoom, mode: absoluteScale}
		f.centerPhotoInCurrentCamera(index)
	case fyne.KeyPlus, fyne.KeyEqual:
		f.zoomPhotoAt(index, transform.factor*compareZoomStep, f.paneCenter(index))
	case fyne.KeyMinus:
		f.zoomPhotoAt(index, transform.factor/compareZoomStep, f.paneCenter(index))
	default:
		return false
	}
	return true
}

func (f *Feature) paneCenter(index int) fyne.Position {
	return fyne.NewPos(f.viewports[index].Width/2, f.viewports[index].Height/2)
}

func (f *Feature) centerPhotoInCurrentCamera(index int) {
	transform := &f.photoTransforms[index]
	native := frameSize(f.loaded[index])
	scale := f.scaleForTransform(index, *transform) * f.camera.zoom
	visible := fyne.NewSize(native.Width*scale, native.Height*scale)
	offset := f.cameraOffsetFor(index)
	transform.center = fyne.NewPos(
		0.5+offset.X/visible.Width,
		0.5+offset.Y/visible.Height,
	)
}

func (f *Feature) zoomPhotoAt(index int, factor float32, anchor fyne.Position) {
	transform := &f.photoTransforms[index]
	minimum, maximum := f.photoFactorRange()
	factor = min(max(factor, minimum), maximum)
	if factor == transform.factor {
		return
	}

	native := frameSize(f.loaded[index])
	oldScale := f.scaleForTransform(index, *transform) * f.camera.zoom
	updated := *transform
	updated.factor = factor
	newScale := f.scaleForTransform(index, updated) * f.camera.zoom
	oldSize := fyne.NewSize(native.Width*oldScale, native.Height*oldScale)
	newSize := fyne.NewSize(native.Width*newScale, native.Height*newScale)
	viewportCenter := f.paneCenter(index)
	offset := f.cameraOffsetFor(index)
	updated.center = fyne.NewPos(
		transform.center.X+(anchor.X-viewportCenter.X-offset.X)*(1/oldSize.Width-1/newSize.Width),
		transform.center.Y+(anchor.Y-viewportCenter.Y-offset.Y)*(1/oldSize.Height-1/newSize.Height),
	)
	updated.center = f.clampPhotoCenter(index, updated, updated.center)
	*transform = updated
}

func (f *Feature) applyCameraKey(name fyne.KeyName) bool {
	switch name {
	case fyne.Key0:
		f.fitCamera()
	case fyne.Key1:
		f.camera = defaultCameraTransform()
	case fyne.KeyPlus, fyne.KeyEqual:
		f.zoomCamera(f.camera.zoom*compareZoomStep, fyne.Position{})
	case fyne.KeyMinus:
		f.zoomCamera(f.camera.zoom/compareZoomStep, fyne.Position{})
	default:
		return false
	}
	return true
}

func (f *Feature) handleScroll(index int, ev *fyne.ScrollEvent) {
	if !f.active || !f.ready || ev == nil || index < 0 || index >= len(f.panes) {
		return
	}
	f.setHoveredPane(index)
	if f.callbacks.Modifiers != nil && f.callbacks.Modifiers()&fyne.KeyModifierShift != 0 {
		f.panBy(index, ev.Scrolled)
		return
	}
	if ev.Scrolled.DY == 0 {
		return
	}
	if f.unlinked {
		transform := &f.photoTransforms[index]
		oldFactor := transform.factor
		minFactor, maxFactor := f.photoFactorRange()
		newFactor := min(max(
			oldFactor*float32(math.Exp(float64(ev.Scrolled.DY*scrollSensitivity))),
			minFactor,
		), maxFactor)
		if newFactor == oldFactor {
			return
		}

		f.zoomPhotoAt(index, newFactor, ev.Position)
		f.applyTransform()
		return
	}

	oldZoom := f.camera.zoom
	newZoom := oldZoom * float32(math.Exp(float64(ev.Scrolled.DY*scrollSensitivity)))
	if newZoom == oldZoom {
		return
	}
	viewport := f.viewports[index]
	f.zoomCamera(newZoom, fyne.NewPos(
		ev.Position.X/viewport.Width-0.5,
		ev.Position.Y/viewport.Height-0.5,
	))
	f.applyTransform()
}

func (f *Feature) photoFactorRange() (float32, float32) {
	zoom := f.camera.zoom
	if zoom <= 0 {
		zoom = 1
	}
	return minZoomFactor / zoom, maxZoomFactor / zoom
}

func (f *Feature) cameraZoomRange() (float32, float32) {
	minimum, maximum := float32(0), float32(math.MaxFloat32)
	for _, transform := range f.photoTransforms {
		if transform.factor <= 0 {
			continue
		}
		minimum = max(minimum, minZoomFactor/transform.factor)
		maximum = min(maximum, maxZoomFactor/transform.factor)
	}
	if maximum == float32(math.MaxFloat32) {
		maximum = maxZoomFactor
	}
	if minimum > maximum {
		minimum = maximum
	}
	return minimum, maximum
}

func (f *Feature) zoomCamera(zoom float32, anchor fyne.Position) {
	minimum, maximum := f.cameraZoomRange()
	zoom = min(max(zoom, minimum), maximum)
	if f.camera.zoom <= 0 || zoom == f.camera.zoom {
		return
	}
	ratio := zoom / f.camera.zoom
	f.camera.offset = fyne.NewPos(
		f.camera.offset.X*ratio+(1-ratio)*anchor.X,
		f.camera.offset.Y*ratio+(1-ratio)*anchor.Y,
	)
	f.camera.zoom = zoom
	f.clampCameraOffset()
}

func (f *Feature) fitCamera() {
	minimum, maximum := f.cameraZoomRange()
	minX, maxX, minY, maxY, feasible := f.cameraFitRange(minimum)
	if !feasible {
		f.camera.zoom = minimum
		f.camera.offset = fyne.Position{}
		return
	}

	low, high := minimum, maximum
	for range 48 {
		candidate := low + (high-low)/2
		if _, _, _, _, ok := f.cameraFitRange(candidate); ok {
			low = candidate
		} else {
			high = candidate
		}
	}
	f.camera.zoom = low
	minX, maxX, minY, maxY, _ = f.cameraFitRange(low)
	f.camera.offset = fyne.NewPos(
		min(max(float32(0), minX), maxX),
		min(max(float32(0), minY), maxY),
	)
}

func (f *Feature) cameraFitRange(zoom float32) (minX, maxX, minY, maxY float32, feasible bool) {
	minX, maxX = -float32(math.MaxFloat32), float32(math.MaxFloat32)
	minY, maxY = -float32(math.MaxFloat32), float32(math.MaxFloat32)
	for i := range f.panes {
		viewport := f.viewports[i]
		native := frameSize(f.loaded[i])
		if !validViewport(viewport) || !validViewport(native) {
			return 0, 0, 0, 0, false
		}
		transform := f.photoTransforms[i]
		baseScale := f.scaleForTransform(i, transform)
		scaled := fyne.NewSize(native.Width*baseScale*zoom, native.Height*baseScale*zoom)
		withoutOffset := fyne.NewPos(
			viewport.Width/2-transform.center.X*scaled.Width,
			viewport.Height/2-transform.center.Y*scaled.Height,
		)
		lowX := -withoutOffset.X / viewport.Width
		highX := (viewport.Width - withoutOffset.X - scaled.Width) / viewport.Width
		lowY := -withoutOffset.Y / viewport.Height
		highY := (viewport.Height - withoutOffset.Y - scaled.Height) / viewport.Height
		minX, maxX = max(minX, lowX), min(maxX, highX)
		minY, maxY = max(minY, lowY), min(maxY, highY)
	}
	return minX, maxX, minY, maxY, minX <= maxX && minY <= maxY
}

func (f *Feature) panBy(index int, delta fyne.Delta) {
	if !f.active || !f.ready || index < 0 || index >= len(f.panes) {
		return
	}
	f.setHoveredPane(index)
	if f.unlinked {
		transform := &f.photoTransforms[index]
		native := frameSize(f.loaded[index])
		scale := f.scaleForTransform(index, *transform)
		scaled := fyne.NewSize(native.Width*scale*f.camera.zoom, native.Height*scale*f.camera.zoom)
		if !validViewport(scaled) {
			return
		}
		transform.center = f.clampPhotoCenter(index, *transform, fyne.NewPos(
			transform.center.X-delta.DX/scaled.Width,
			transform.center.Y-delta.DY/scaled.Height,
		))
		f.applyTransform()
		return
	}
	f.camera.offset = fyne.NewPos(
		f.camera.offset.X+delta.DX/f.viewports[index].Width,
		f.camera.offset.Y+delta.DY/f.viewports[index].Height,
	)
	f.clampCameraOffset()
	f.applyTransform()
}

func (f *Feature) clampPhotoCenter(index int, transform photoTransform, center fyne.Position) fyne.Position {
	native := frameSize(f.loaded[index])
	scale := f.scaleForTransform(index, transform) * f.camera.zoom
	scaled := fyne.NewSize(native.Width*scale, native.Height*scale)
	if !validViewport(scaled) {
		return center
	}
	offset := f.cameraOffsetFor(index)
	minX, minY := offset.X/scaled.Width, offset.Y/scaled.Height
	return fyne.NewPos(
		min(max(center.X, minX), 1+minX),
		min(max(center.Y, minY), 1+minY),
	)
}

func (f *Feature) applyTransform() {
	if !f.ready || !validViewport(f.viewports[0]) || !validViewport(f.viewports[1]) {
		return
	}
	for i := range f.panes {
		native := frameSize(f.loaded[i])
		if !validViewport(native) {
			continue
		}
		transform := f.photoTransforms[i]
		scale := f.scaleForTransform(i, transform)
		baseSize := fyne.NewSize(native.Width*scale, native.Height*scale)
		scaled := fyne.NewSize(baseSize.Width*f.camera.zoom, baseSize.Height*f.camera.zoom)
		cameraOffset := f.cameraOffsetFor(i)
		position := fyne.NewPos(
			f.viewports[i].Width/2-transform.center.X*scaled.Width+cameraOffset.X,
			f.viewports[i].Height/2-transform.center.Y*scaled.Height+cameraOffset.Y,
		)
		revealPosition, revealSize := f.paneVisibleArea(i)
		width, height := f.vectorPixels(f.panes[i].input, scaled)
		f.panes[i].renderer.Present(paneScene{
			source:         f.renderSources[i],
			viewport:       f.viewports[i],
			revealSet:      true,
			revealPosition: revealPosition,
			revealSize:     revealSize,
			imagePosition:  position,
			imageSize:      scaled,
			displaySize:    image.Pt(width, height),
		})
		f.requestVectorRender(i, scaled)
	}
}

func (f *Feature) clampCameraOffset() {
	minX, maxX, minY, maxY := f.cameraPanRange()
	f.camera.offset = fyne.NewPos(
		min(max(f.camera.offset.X, minX), maxX),
		min(max(f.camera.offset.Y, minY), maxY),
	)
}

func (f *Feature) cameraOffsetFor(index int) fyne.Position {
	return fyne.NewPos(
		f.camera.offset.X*f.viewports[index].Width,
		f.camera.offset.Y*f.viewports[index].Height,
	)
}

func (f *Feature) visiblePhotoTransform(index int) photoTransform {
	transform := f.photoTransforms[index]
	native := frameSize(f.loaded[index])
	baseScale := f.scaleForTransform(index, transform)
	visible := fyne.NewSize(
		native.Width*baseScale*f.camera.zoom,
		native.Height*baseScale*f.camera.zoom,
	)
	offset := f.cameraOffsetFor(index)
	transform.center = fyne.NewPos(
		transform.center.X-offset.X/visible.Width,
		transform.center.Y-offset.Y/visible.Height,
	)
	transform.factor *= f.camera.zoom
	return transform
}

func (f *Feature) cameraPanRange() (minX, maxX, minY, maxY float32) {
	minX, maxX = -float32(math.MaxFloat32), float32(math.MaxFloat32)
	minY, maxY = -float32(math.MaxFloat32), float32(math.MaxFloat32)
	for i := range f.panes {
		native := frameSize(f.loaded[i])
		transform := f.photoTransforms[i]
		scale := f.scaleForTransform(i, transform) * f.camera.zoom
		scaled := fyne.NewSize(native.Width*scale, native.Height*scale)
		withoutOffset := fyne.NewPos(
			f.viewports[i].Width/2-transform.center.X*scaled.Width,
			f.viewports[i].Height/2-transform.center.Y*scaled.Height,
		)
		paneCenter := fyne.NewPos(f.viewports[i].Width/2, f.viewports[i].Height/2)
		lowX, highX := paneCenter.X-(withoutOffset.X+scaled.Width), paneCenter.X-withoutOffset.X
		lowY, highY := paneCenter.Y-(withoutOffset.Y+scaled.Height), paneCenter.Y-withoutOffset.Y
		lowX, highX = lowX/f.viewports[i].Width, highX/f.viewports[i].Width
		lowY, highY = lowY/f.viewports[i].Height, highY/f.viewports[i].Height
		minX, maxX = max(minX, lowX), min(maxX, highX)
		minY, maxY = max(minY, lowY), min(maxY, highY)
	}
	if minX > maxX {
		minX, maxX = f.camera.offset.X, f.camera.offset.X
	}
	if minY > maxY {
		minY, maxY = f.camera.offset.Y, f.camera.offset.Y
	}
	return minX, maxX, minY, maxY
}

func (f *Feature) canPan(index int) bool {
	if !f.active || !f.ready || index < 0 || index >= len(f.panes) ||
		!validViewport(f.viewports[0]) || !validViewport(f.viewports[1]) {
		return false
	}
	if f.unlinked {
		return validViewport(frameSize(f.loaded[index]))
	}
	minX, maxX, minY, maxY := f.cameraPanRange()
	return minX < maxX || minY < maxY
}

func (f *Feature) scaleForTransform(index int, transform photoTransform) float32 {
	if transform.mode == fitRelative {
		return fitScale(frameSize(f.loaded[index]), f.viewports[index]) * transform.factor
	}
	return transform.factor
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
