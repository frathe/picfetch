// The export prompt's extra rows - what the exported copy is, beyond the
// format its buttons name: the export size limit and whether the source's
// camera metadata rides along. Built in features.go, handed to the prompt's
// widgets.ChoiceCard as its widgets.ExtraRows, and read by exportAs
// (export.go) at the moment a format button commits.

package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/widgets"
)

// exportSizeRungs are the ceilings the export prompt offers for the
// exported copy's longest edge, in the order they are drawn. 0 is the
// Original rung - not a ceiling of zero but no ceiling at all, which is
// exactly what imaging.FitEdge and imaging.ScaleForExport read it as, so
// nothing between here and the encoder has to special-case it.
//
// Four rungs rather than a free-text field: this is a prompt someone is
// passing through on the way to mailing a photo, and a number they have to
// think of is a decision they did not ask for.
var exportSizeRungs = []int{0, 2400, 1600, 1000}

// exportSizeRow and exportMetadataRow are the rows' indices among the
// prompt's extra rows - the stops widgets.ChoiceCard's Up/Down move onto.
// Rows are numbered from the top down, so the size row is the one furthest
// from the buttons and a single Up from them lands on the metadata row.
const (
	exportSizeRow     = 0
	exportMetadataRow = 1
	exportRowCount    = 2
)

// exportOptions is the block of controls the export prompt draws between
// its message and its format buttons, and the widgets.ExtraRows
// implementation behind it.
//
// The metadata row is a real widget.Check; the size rungs are tappable
// labels with hand-drawn marks. Both Fyne widgets grab canvas focus on every
// tap (focusIfNotMobile, in Fyne's own check.go and radio_item.go), and this
// app dispatches every key from the canvas's *unfocused* handler - so a
// focused control left holding the keyboard would swallow Return and Escape
// with the prompt still on screen and no way to commit or cancel it. What
// separates the two is whether there is anywhere to hand the keyboard back
// from: every effective tap on a Check toggles it, so OnChanged fires on all
// of them and Unfocus there closes the hole (grid.Close does the same thing
// after a tap, one surface deeper). A radio item focuses on every tap but
// only fires OnChanged when the value actually changes, so tapping the
// already-selected rung would focus with nothing to hook - which is why the
// rungs draw their own selection instead.
type exportOptions struct {
	// repaint forces the window to redraw after a selection change, the way
	// every other overlay in this app has to - see viewer.ForceRepaint.
	repaint func()

	// unfocus releases Fyne's canvas focus - viewer.Unfocus. Called whenever
	// the checkbox changes, because a tap on it focuses the widget first and
	// the app's key dispatcher only runs while nothing is focused.
	unfocus func()

	content *fyne.Container

	// sizeRing marks the size row when the keyboard is on it; the tints
	// mark which rung is picked. Ring says "here", wash says "picked" -
	// widgets.NewSelectionTint's own pairing, which the grid established.
	sizeRing *canvas.Rectangle
	rungText []*widget.Label
	rungTint []*canvas.Rectangle

	// metaRing marks the metadata row the same way. The checkbox is the one
	// place its own state lives - Options reads metaCheck.Checked rather
	// than a parallel bool, so what is drawn and what is exported cannot
	// disagree (the rule ChoicePanel.SetChoiceEnabled documents for buttons).
	metaRing  *canvas.Rectangle
	metaCheck *widget.Check

	// rung indexes exportSizeRungs. Reset puts it back to 0 on every open,
	// so a limit set once can never quietly shrink a later export.
	rung int

	// sourceEdge is the longest edge of the frame the prompt was opened
	// over, in pixels - what the Original rung's label states. Set from the
	// image in hand rather than from the file's own header, which is what
	// makes a RAW honest: the frame on screen for a RAW is the camera's
	// embedded JPEG preview, so Original means the preview's size, and the
	// number says so.
	sourceEdge int

	// focus is which row the card has moved the selection onto, or -1 when
	// the format buttons hold it - the value HandleKey routes on.
	focus int
}

// newExportOptions builds the prompt's extra rows, unfocused and at their
// defaults.
func newExportOptions(repaint, unfocus func()) *exportOptions {
	o := &exportOptions{repaint: repaint, unfocus: unfocus, focus: -1}

	o.sizeRing = widgets.NewFocusRing(widgets.ButtonRingWidth, widgets.RingRadius)
	o.sizeRing.Hide()

	cells := make([]fyne.CanvasObject, 0, len(exportSizeRungs)+1)
	cells = append(cells, widget.NewLabel(lang.L("Export size limit")))
	for i := range exportSizeRungs {
		text := widget.NewLabel("")
		tint := widgets.NewSelectionTint()
		if i != 0 {
			tint.Hide()
		}

		o.rungText = append(o.rungText, text)
		o.rungTint = append(o.rungTint, tint)
		cells = append(cells, widgets.NewTappableArea(
			container.NewStack(tint, container.NewPadded(text)),
			func() { o.selectRung(i) },
		))
	}

	o.metaRing = widgets.NewFocusRing(widgets.ButtonRingWidth, widgets.RingRadius)
	o.metaRing.Hide()
	o.metaCheck = widget.NewCheck(lang.L("Include camera metadata (JPEG only)"), o.metadataChanged)
	// The field rather than SetChecked: ticked is what the box is born with,
	// and SetChecked would fire OnChanged into a half-built widget - the
	// content it repaints does not exist yet.
	o.metaCheck.Checked = true

	o.content = container.NewVBox(
		widgets.Ringed(o.sizeRing, container.NewHBox(cells...)),
		widgets.Ringed(o.metaRing, o.metaCheck),
	)
	o.refreshRungText()

	return o
}

// Content is the block the card draws above its buttons.
func (o *exportOptions) Content() fyne.CanvasObject {
	return o.content
}

// Rows is how many vertical stops the card's Up/Down have to walk.
func (o *exportOptions) Rows() int {
	return exportRowCount
}

// Focus marks the row the card has moved the selection onto, or clears the
// marking for -1, which is the card saying the format buttons have it back.
func (o *exportOptions) Focus(row int) {
	o.focus = row

	widgets.MarkOnly([]*canvas.Rectangle{
		exportSizeRow:     o.sizeRing,
		exportMetadataRow: o.metaRing,
	}, row)
	o.content.Refresh()
}

// HandleKey handles a key the card decided belongs to the focused row, and
// reports whether the row used it. Escape never arrives here, and neither do
// Up and Down, which move between rows.
//
// The answer matters for Return: the metadata row takes it and ticks the
// box, because a user pressing Return on a highlighted checkbox means that
// checkbox, not "export now". The size row has nothing to activate - its
// rungs are picked with Left and Right - so it declines and the card commits
// the export instead.
func (o *exportOptions) HandleKey(ev *fyne.KeyEvent) bool {
	switch o.focus {
	case exportSizeRow:
		switch ev.Name {
		case fyne.KeyLeft:
			o.selectRung(o.rung - 1)
		case fyne.KeyRight:
			o.selectRung(o.rung + 1)
		default:
			return false
		}

		return true
	case exportMetadataRow:
		switch ev.Name {
		case fyne.KeySpace, fyne.KeyReturn, fyne.KeyEnter:
			o.setMetadataIncluded(!o.metaCheck.Checked)

			return true
		}
	}

	return false
}

// Reset returns every control to its default - the behaviour export has
// always had. Run by the card on every Show, so the prompt always states
// the whole truth about what it is about to write and is never in a state
// the user has forgotten setting.
func (o *exportOptions) Reset() {
	o.selectRung(0)
	o.setMetadataIncluded(true)
}

// Options is what the prompt currently asks imaging.Export to do, read by
// exportAs on the UI goroutine at the moment a format button commits.
func (o *exportOptions) Options() imaging.ExportOptions {
	return imaging.ExportOptions{
		MaxEdge:      exportSizeRungs[o.rung],
		OmitMetadata: !o.metaCheck.Checked,
	}
}

// SetSourceEdge tells the prompt the longest edge of the frame it is about
// to be opened over, which is what the Original rung's label states. Called
// before Show rather than from Reset, since the rows have no way to reach
// the frame themselves.
func (o *exportOptions) SetSourceEdge(edge int) {
	o.sourceEdge = edge
	o.refreshRungText()
}

// selectRung moves the size limit onto rung i, clamping to the rung range
// rather than wrapping - the same rule ChoicePanel.Select applies to the
// buttons below.
func (o *exportOptions) selectRung(i int) {
	if last := len(exportSizeRungs) - 1; i > last {
		i = last
	}
	if i < 0 {
		i = 0
	}

	o.rung = i
	widgets.MarkOnly(o.rungTint, i)
	o.redraw()
}

// setMetadataIncluded checks or unchecks the metadata box - the keyboard's
// Space, or Reset putting it back to checked. A click on the box reaches
// the widget directly and arrives at metadataChanged below instead, which
// is why nothing here is repeated there: SetChecked runs it for us.
func (o *exportOptions) setMetadataIncluded(included bool) {
	o.metaCheck.SetChecked(included)
}

// metadataChanged is the checkbox's OnChanged, so it runs however the box
// was ticked - the keyboard through SetChecked, or a click straight on the
// widget.
//
// The Unfocus is what makes a real widget.Check usable inside this prompt:
// Fyne focuses the box on the tap that toggles it, and the app dispatches
// every key from the canvas's *unfocused* handler, so leaving it focused
// would hand it Return and Escape - which it ignores - and strand a prompt
// the user can no longer commit or cancel from the keyboard.
func (o *exportOptions) metadataChanged(_ bool) {
	if o.unfocus != nil {
		o.unfocus()
	}
	o.redraw()
}

// redraw repaints the rows after a selection change: the container first,
// because Fyne only registers an object with its canvas the first time it is
// painted while visible (see widgets.MarkOnly), then the window, because
// this app has no automatic redraw loop.
func (o *exportOptions) redraw() {
	o.content.Refresh()

	if o.repaint != nil {
		o.repaint()
	}
}

// refreshRungText re-renders every rung's label, which only the Original
// one can actually change: it carries the frame's own longest edge.
func (o *exportOptions) refreshRungText() {
	for i, rung := range exportSizeRungs {
		o.rungText[i].SetText(rungLabel(rung, o.sourceEdge))
	}
}

// rungLabel is one rung's text: a plain pixel ceiling, or Original carrying
// the frame's real longest edge so the reader can tell at a glance whether
// a smaller rung would change anything. A frame whose size isn't known yet
// (nothing has opened the prompt) says just Original rather than claiming
// zero pixels.
func rungLabel(rung, sourceEdge int) string {
	if rung > 0 {
		return fmt.Sprintf(lang.L("%d px"), rung)
	}
	if sourceEdge <= 0 {
		return lang.L("Original")
	}

	return fmt.Sprintf(lang.L("Original (%d px)"), sourceEdge)
}
