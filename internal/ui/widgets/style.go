// Package widgets holds the viewer-free UI mechanics shared across the
// app's feature packages: the tappable area behind the drop zone, the
// singleton-window helper the secondary windows share, and the style
// values below.
package widgets

import (
	"image/color"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
)

// This file gathers the app's shared hardcoded style values - everything
// visual that isn't taken from Fyne's own theme - so one concept can't
// quietly drift apart between call sites. Where two uses of the same concept
// genuinely differ (the delete-confirm rings vs the grid highlight ring),
// the difference is a named parameter here instead of two unrelated
// literals.

const (
	// CardRadius rounds every card-shaped overlay background: the toast,
	// the info overlay, comparison chrome, and the delete-confirmation card.
	CardRadius = 8

	// DropzoneBorderWidth and DropzoneBorderRadius outline the dropzone's
	// rounded border box.
	DropzoneBorderWidth  = 4
	DropzoneBorderRadius = 14

	// WelcomeArtSize is the square box the welcome and empty-state art
	// share, so both occupy the exact same spot regardless of which one is
	// currently shown.
	WelcomeArtSize = 180

	// ScanArtSize is the square box the folder-scan overlay's art occupies,
	// above the spinner and label.
	ScanArtSize = 120

	// ButtonRingWidth and GridRingWidth are the focus rings' stroke widths
	// (see NewFocusRing), RingRadius their shared corner radius: the
	// delete-confirmation buttons use a thinner ring than the grid's cell
	// highlight, whose stroke has to stay visible against a busy thumbnail
	// behind it.
	ButtonRingWidth = 2
	GridRingWidth   = 3
	RingRadius      = 6

	// SelectionTintAlpha is how opaque the grid's multi-select wash is (see
	// NewSelectionTint). Enough to read as "picked" across both themes and
	// over any thumbnail, little enough to still see which image it is.
	SelectionTintAlpha = 90

	// InactiveRingAlpha is how opaque a focus ring is drawn once the
	// keyboard has moved somewhere else (see SetRingActive) - the same
	// opacity as the grid's "picked" wash, because it is saying the same
	// thing: this is still the choice, it is just no longer where you are.
	InactiveRingAlpha = SelectionTintAlpha

	// MarqueeStrokeWidth / MarqueeFillAlpha are the grid drag-select
	// rectangle (see NewMarqueeRect): a hairline of the same primary hue
	// as the focus ring, with a wash light enough to read the thumbnails
	// underneath while the drag is in progress.
	MarqueeStrokeWidth float32 = 1
	MarqueeFillAlpha   uint8   = 40
)

var (
	DropzoneBorderColor = color.NRGBA{R: 100, G: 100, B: 135, A: 200}
	DropzoneHoverColor  = color.NRGBA{R: 150, G: 150, B: 205, A: 255}

	// ToastBGColor and ToastTextColor are the toast's fixed, deliberately loud
	// warning colors: dark text for contrast against the pastel-orange
	// background. Not theme-derived on purpose - the toast should look the
	// same, and stand out the same, in both light and dark themes (contrast
	// the info overlay, which uses the theme's own
	// overlay-background/foreground pairing instead).
	ToastBGColor   = color.NRGBA{R: 255, G: 179, B: 128, A: 235}
	ToastTextColor = color.NRGBA{R: 51, G: 26, B: 0, A: 255}

	// ScrimColor dims the image view behind the delete-confirmation card.
	ScrimColor = color.NRGBA{R: 0, G: 0, B: 0, A: 140}
)

// NewFocusRing returns one of the manually drawn selection rings used where
// this app tracks its own selection instead of Fyne's widget-focus system
// (see grid.Overview.Close's comment for why it never uses that): a
// transparent rectangle with a colored stroke. Returned visible; callers
// hide it when the ringed item starts unselected.
//
// The stroke is the theme's *primary* color, not ColorNameFocus: Fyne's
// focus color is that same hue at 16% alpha (0x2a), a wash meant to be
// painted across a widget's whole area, and drawn instead as a hairline it
// all but disappears - against the delete card's red button, and against a
// grid thumbnail. Selection here has to be unmissable, so it takes the
// opaque color.
func NewFocusRing(strokeWidth, cornerRadius float32) *canvas.Rectangle {
	ring := canvas.NewRectangle(color.Transparent)
	ring.StrokeColor = theme.Color(theme.ColorNamePrimary)
	ring.StrokeWidth = strokeWidth
	ring.CornerRadius = cornerRadius

	return ring
}

// SetRingActive draws ring at full strength or muted, without moving or
// hiding it: full while the surface it marks holds the keyboard, muted once
// the keyboard has moved to another row of the same prompt.
//
// Two rings at full strength on one card is the thing this exists to
// prevent - the eye reads both as "you are here" and neither as the answer.
// Muting rather than hiding keeps the answer to "which format will Return
// write?" on screen while the keyboard is up in the options.
func SetRingActive(ring *canvas.Rectangle, active bool) {
	stroke := color.NRGBAModel.Convert(theme.Color(theme.ColorNamePrimary)).(color.NRGBA)
	if !active {
		stroke.A = InactiveRingAlpha
	}

	ring.StrokeColor = stroke
	ring.Refresh()
}

// Ringed pairs a widget with its selection ring: the ring fills the cell,
// the widget is inset by one padding step inside it, so the ring's stroke
// lands in that gap instead of underneath the widget. Stacking the two at
// the same size hides the ring entirely - a Fyne button paints an opaque
// background across its whole area, including the DangerImportance red -
// and the card then looks identical whichever button is selected. Behind
// rather than on top so the ring can never sit between the pointer and the
// widget it marks.
//
// Any canvas object rather than a button specifically: the export prompt
// rings a whole row of options to say where the keyboard is, which is the
// same idea one step larger.
func Ringed(ring *canvas.Rectangle, obj fyne.CanvasObject) *fyne.Container {
	return container.NewStack(ring, container.NewPadded(obj))
}

// MarkOnly shows mark i and hides every other one - the show-one, hide-rest
// half of every selection this app draws for itself, shared so the choice
// panel's rings and the export prompt's rung tints cannot drift apart. An
// out-of-range index hides all of them.
//
// Callers still Refresh the container afterwards: Fyne only registers an
// object with its canvas the first time it is painted while visible, so a
// mark hidden since its surface went up has no canvas to mark dirty and
// would silently fail to appear (see viewer.ForceRepaint for the same trap).
func MarkOnly(marks []*canvas.Rectangle, i int) {
	for idx, mark := range marks {
		if idx == i {
			mark.Show()
		} else {
			mark.Hide()
		}
	}
}

// NewSelectionTint returns the translucent wash the grid overview draws over
// a cell that is part of a multi-select. Returned visible; callers hide it on
// the cells that start unselected.
//
// A filled wash rather than a second ring, because a cell can be both
// selected and the keyboard's current position at the same time, and two
// rings differing only in color would be unreadable. The tint is the same
// primary hue NewFocusRing strokes with, so the pair reads as one idea: the
// ring says "here", the wash says "picked".
func NewSelectionTint() *canvas.Rectangle {
	c := color.NRGBAModel.Convert(theme.Color(theme.ColorNamePrimary)).(color.NRGBA)
	c.A = SelectionTintAlpha

	tint := canvas.NewRectangle(c)
	tint.CornerRadius = RingRadius

	return tint
}

// NewMarqueeRect returns the drag-select rectangle drawn over the grid
// overview while a marquee is in progress. Hidden until the first Dragged;
// the grid shows it for the gesture and hides it again on DragEnd, Escape,
// or Close.
func NewMarqueeRect() *canvas.Rectangle {
	c := color.NRGBAModel.Convert(theme.Color(theme.ColorNamePrimary)).(color.NRGBA)
	fill := c
	fill.A = MarqueeFillAlpha
	r := canvas.NewRectangle(fill)
	r.StrokeColor = c
	r.StrokeWidth = MarqueeStrokeWidth
	r.CornerRadius = RingRadius
	r.Hide()
	return r
}
