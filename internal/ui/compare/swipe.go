package compare

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
)

const (
	defaultDivider           = float32(0.5)
	dividerKeyStep           = float32(0.05)
	dividerFineKeyStep       = float32(0.01)
	dividerFallbackThickness = float32(5)
)

type comparisonLayout uint8

const (
	sideBySide comparisonLayout = iota
	swipe
)

type paneReveal struct {
	clip *container.Clip
}

func newPaneReveal(root fyne.CanvasObject) paneReveal {
	// Clip resizes its direct content to the reveal bounds, so keep the
	// full-viewport pane inside a manually laid-out wrapper. This is also the
	// pane's only clip: Fyne's software walker replaces nested clip rectangles
	// instead of intersecting them, which would let the right image overpaint
	// the left reveal.
	surface := container.NewWithoutLayout(root)
	return paneReveal{clip: container.NewClip(surface)}
}

type comparisonContentLayout struct {
	feature *Feature
}

func (l comparisonContentLayout) Layout(_ []fyne.CanvasObject, size fyne.Size) {
	if l.feature != nil {
		l.feature.layoutContent(size)
	}
}

func (comparisonContentLayout) MinSize(_ []fyne.CanvasObject) fyne.Size { return fyne.Size{} }

func (f *Feature) layoutContent(size fyne.Size) {
	if f.layoutMode == swipe {
		f.layoutSwipe(size)
		return
	}
	f.layoutSideBySide(size)
}

func (f *Feature) layoutSideBySide(size fyne.Size) {
	leftWidth := size.Width / 2
	paneSizes := [2]fyne.Size{
		fyne.NewSize(leftWidth, size.Height),
		fyne.NewSize(size.Width-leftWidth, size.Height),
	}
	// Publish both destination viewports before either pane relayouts. A pane
	// resize calls applyTransform, which must never clamp against mixed old/new
	// layout geometry during a mode transition.
	f.viewports = paneSizes
	f.layoutReveal(0, fyne.NewPos(0, 0), paneSizes[0], fyne.NewPos(0, 0), paneSizes[0])
	f.layoutReveal(1, fyne.NewPos(leftWidth, 0), paneSizes[1], fyne.NewPos(0, 0), paneSizes[1])
	f.divider.Hide()
	f.applyTransform()
}

func (f *Feature) layoutSwipe(size fyne.Size) {
	full := fyne.NewSize(size.Width, size.Height)
	f.viewports = [2]fyne.Size{full, full}
	f.layoutSwipeReveal(size)
	f.divider.Show()
	f.applyTransform()
}

func (f *Feature) layoutSwipeReveal(size fyne.Size) {
	boundary := min(max(f.dividerAt, 0), 1) * size.Width
	full := fyne.NewSize(size.Width, size.Height)
	f.layoutReveal(0, fyne.NewPos(0, 0), fyne.NewSize(boundary, size.Height), fyne.NewPos(0, 0), full)
	f.layoutReveal(1, fyne.NewPos(boundary, 0), fyne.NewSize(size.Width-boundary, size.Height), fyne.NewPos(-boundary, 0), full)

	thickness := dividerThickness()
	f.divider.Move(fyne.NewPos(boundary-thickness/2, 0))
	f.divider.Resize(fyne.NewSize(thickness, size.Height))
}

func (f *Feature) paneVisibleArea(index int) (fyne.Position, fyne.Size) {
	viewport := f.viewports[index]
	if f.layoutMode != swipe {
		return fyne.Position{}, viewport
	}
	boundary := min(max(f.dividerAt, 0), 1) * viewport.Width
	if index == 0 {
		return fyne.Position{}, fyne.NewSize(boundary, viewport.Height)
	}
	return fyne.NewPos(boundary, 0), fyne.NewSize(viewport.Width-boundary, viewport.Height)
}

func (f *Feature) layoutReveal(index int, clipPosition fyne.Position, clipSize fyne.Size, rootPosition fyne.Position, rootSize fyne.Size) {
	reveal := f.reveals[index]
	reveal.clip.Move(clipPosition)
	reveal.clip.Resize(clipSize)
	f.panes[index].root.Move(rootPosition)
	f.panes[index].root.Resize(rootSize)
}

func dividerThickness() float32 {
	thickness := theme.Size(theme.SizeNameSplitThickness)
	if thickness <= 0 {
		return dividerFallbackThickness
	}
	return thickness
}

func (f *Feature) resetLayout() {
	f.layoutMode = sideBySide
	f.dividerAt = defaultDivider
	f.layoutToggle.SetText(lang.L("Swipe"))
	if f.content != nil {
		f.content.Refresh()
	}
}

func (f *Feature) toggleLayout() {
	if !f.active || !f.ready {
		return
	}
	if f.layoutMode == sideBySide {
		f.layoutMode = swipe
		f.layoutToggle.SetText(lang.L("Side by side"))
	} else {
		f.layoutMode = sideBySide
		f.layoutToggle.SetText(lang.L("Swipe"))
	}
	f.content.Refresh()
	f.repaint()
}

func (f *Feature) dragDivider(deltaX float32) {
	if !f.active || !f.ready || f.layoutMode != swipe || deltaX == 0 || f.content.Size().Width <= 0 {
		return
	}
	f.setDivider(f.dividerAt + deltaX/f.content.Size().Width)
}

func (f *Feature) setDivider(position float32) {
	position = min(max(position, 0), 1)
	if position == f.dividerAt {
		return
	}
	f.dividerAt = position
	// Divider input changes only reveal geometry. Refreshing either the
	// comparison content or its owner recursively refreshes canvas images;
	// Fyne then discards their textures and can re-decode hidden resource art
	// on every pointer event. The visible clips and pane offset mark the canvas
	// dirty as they move, so the static images and linked transform stay cached.
	f.layoutSwipeReveal(f.content.Size())
}

func (f *Feature) handleDividerKey(name fyne.KeyName, modifiers fyne.KeyModifier) bool {
	if f.layoutMode != swipe {
		return false
	}

	step := dividerKeyStep
	if modifiers&fyne.KeyModifierShift != 0 {
		step = dividerFineKeyStep
	}
	switch name {
	case fyne.KeyLeft:
		f.setDivider(f.dividerAt - step)
	case fyne.KeyRight:
		f.setDivider(f.dividerAt + step)
	case fyne.KeyHome:
		f.setDivider(0)
	case fyne.KeyEnd:
		f.setDivider(1)
	default:
		return false
	}
	return true
}

var _ fyne.Layout = comparisonContentLayout{}
