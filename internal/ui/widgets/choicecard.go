package widgets

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// ChoiceCard is a dimmed scrim behind a centered message-and-buttons card -
// the modal prompt shape deletion's Shift+Delete confirmation originated and
// now shares with the export-format prompt. The buttons and everything about
// selection are the ChoicePanel underneath; the card adds the scrim, the
// message above them, and its own visibility.
//
// The card never gives that panel Fyne's keyboard focus, unlike a dialog
// would: this app dispatches every key from the canvas's unfocused handler
// (modified key combos already have to bypass widget focus - see the app's
// wireOpenShortcuts comment), so the app's dispatcher hands the card the keys
// through HandleKey instead while it is up.
type ChoiceCard struct {
	// panel owns the buttons, their focus rings, the selected index and the
	// key rules over them. The card keeps no copy of any of that, so a click
	// on a button - which reaches the panel directly, never this type - can't
	// leave the two disagreeing.
	panel *ChoicePanel

	// rows is the optional block drawn above the buttons, nil for a card
	// that is only a message and a button row (deletion's confirmation).
	rows ExtraRows

	// focus is which vertical stop currently holds the selection: a row
	// index while it is inside rows, len(rows) - i.e. rows.Rows() - once it
	// is back on the button row. Show always leaves it on the buttons, so a
	// prompt raised and immediately confirmed still runs a choice.
	focus int

	// repaint is called after every visibility change - the app has no
	// automatic redraw loop, so a hidden window has to be told to paint again
	// itself (see viewer.ForceRepaint, which every caller so far passes in
	// directly). The panel gets the same hook for its selection changes.
	repaint func()

	visible bool
	overlay *fyne.Container
	message *widget.Label
}

// ExtraRows is the optional block a ChoiceCard draws above its button row -
// the export prompt's size limit and metadata checkbox - together with its
// side of the card's keyboard story. A card built without one is a message
// and a button row, exactly as the delete confirmation has always been.
//
// The vertical stops are rows 0..Rows()-1 and then the button row, so Focus
// is called with -1 when the selection has moved back down onto the buttons
// and there is no row to mark. Reset runs on every Show, so the prompt can
// never open in a state a previous use left behind.
//
// HandleKey reports whether the row used the key. That answer only decides
// anything for Return and Enter: a row that has something to activate (a
// checkbox to tick) takes them, because that is what a user pressing Return
// on a highlighted checkbox means; a row that has nothing to activate
// returns false and the card commits the prompt instead. Escape is never
// offered at all - cancelling belongs to the prompt as a whole, from any
// stop.
type ExtraRows interface {
	Content() fyne.CanvasObject
	Rows() int
	Focus(row int)
	HandleKey(ev *fyne.KeyEvent) bool
	Reset()
}

// NewChoiceCard builds the card (hidden) with the given choices, left to
// right. Index 0 is the leftmost button and the default selection.
func NewChoiceCard(repaint func(), choices ...Choice) *ChoiceCard {
	return NewChoiceCardWithRows(repaint, nil, choices...)
}

// NewChoiceCardWithRows is NewChoiceCard with a block of extra rows drawn
// between the message and the buttons, and reachable with Up/Down. A nil
// rows argument builds exactly the card NewChoiceCard does - nothing is
// added to the layout and Up/Down stay inert.
func NewChoiceCardWithRows(repaint func(), rows ExtraRows, choices ...Choice) *ChoiceCard {
	c := &ChoiceCard{repaint: repaint, rows: rows}

	c.panel = NewChoicePanel(repaint, choices...)
	// The card is what a confirmed or cancelled prompt has to take off
	// screen, and hiding it is all there is to that - see the panel's
	// SetOnDismiss for the ordering this buys.
	c.panel.SetOnDismiss(c.Hide)

	scrim := canvas.NewRectangle(ScrimColor)

	c.message = widget.NewLabel("")
	c.message.Alignment = fyne.TextAlignCenter
	c.message.Wrapping = fyne.TextWrapWord

	cardBG := canvas.NewRectangle(theme.Color(theme.ColorNameOverlayBackground))
	cardBG.CornerRadius = CardRadius
	stacked := []fyne.CanvasObject{c.message}
	if rows != nil {
		stacked = append(stacked, rows.Content())
	}
	stacked = append(stacked, c.panel)
	card := container.NewStack(cardBG, container.NewPadded(container.NewVBox(stacked...)))

	c.overlay = container.NewStack(scrim, container.NewCenter(card))
	c.overlay.Hide()

	return c
}

// runChoice is the card's name for the panel's click path (see
// ChoicePanel.runChoice), so a caller holding a card can run choice i the way
// a click on its button does - card hidden first, keyboard selection ignored.
func (c *ChoiceCard) runChoice(i int) func() {
	return c.panel.runChoice(i)
}

// Overlay is the card, for the caller to place in its window stack.
func (c *ChoiceCard) Overlay() fyne.CanvasObject {
	return c.overlay
}

// Visible reports whether the card is up.
func (c *ChoiceCard) Visible() bool {
	return c.visible
}

// Selected is the index Left/Right (or Select) last moved the ring to - a
// test seam, mirrored by Message and Ring below for consumers built directly
// on the card that need to assert on its rendered state.
func (c *ChoiceCard) Selected() int {
	return c.panel.Selected()
}

// Message is the card's headline label.
func (c *ChoiceCard) Message() *widget.Label {
	return c.message
}

// Ring is the selection ring drawn behind choice i, or nil for an
// out-of-range index.
func (c *ChoiceCard) Ring(i int) *canvas.Rectangle {
	return c.panel.Ring(i)
}

// SetOnCancel registers what Escape runs once the card is hidden, in
// addition to hiding it. Optional: a card whose index-0 choice already does
// nothing beyond dismissing has nothing more for Escape to do.
func (c *ChoiceCard) SetOnCancel(onCancel func()) {
	c.panel.SetOnCancel(onCancel)
}

// Show raises the card with the given message, resetting the selection to
// index 0 - never carried over from whatever a previous prompt left it at.
func (c *ChoiceCard) Show(message string) {
	c.message.SetText(message)
	c.Select(0)
	if c.rows != nil {
		c.rows.Reset()
		c.focusStop(c.rows.Rows())
	}

	c.visible = true
	c.overlay.Show()
	if c.repaint != nil {
		c.repaint()
	}
}

// Hide dismisses the card without running any choice - Escape, a caller's
// own guarded dismissal (deletion.Confirmer.Cancel), or the panel taking the
// card down before a chosen action runs. Always repaints, even when the card
// is already hidden: a caller that just hid it through some other path still
// wants the window redrawn.
func (c *ChoiceCard) Hide() {
	c.visible = false
	c.overlay.Hide()
	if c.repaint != nil {
		c.repaint()
	}
}

// Select moves the selection to index i, clamping to the choice range rather
// than wrapping - see ChoicePanel.Select, which owns that rule.
func (c *ChoiceCard) Select(i int) {
	c.panel.Select(i)
}

// Confirm runs whichever choice is currently selected - Return/Enter while
// the card is up, or deletion's own confirmSelection test seam calling it
// directly. The card hides before the choice's OnChosen runs, so an action
// that shows something else of its own doesn't have to hide this card first.
func (c *ChoiceCard) Confirm() {
	c.panel.Confirm()
}

// HandleKey handles a key press while the card is up: Left/Right move the
// selection (clamping at either end), Return/Enter runs whichever is
// selected, Escape hides the card and runs onCancel if one is registered.
// Every other key is deliberately left to the caller.
//
// A card carrying extra rows adds one dimension to that: Up and Down move
// between the rows and the button row (clamping at both ends), and every key
// the card doesn't claim for itself goes to whichever stop currently holds
// the selection. Up and Down are free to mean this precisely because the app
// dispatcher hands the card *every* key while it is up, so nothing outside
// is competing for them - and a card built without rows leaves both as inert
// as they have always been.
//
// The app's key dispatcher calls this rather than Fyne delivering it to the
// panel, because nothing on this card ever holds widget focus - see the type
// comment.
func (c *ChoiceCard) HandleKey(ev *fyne.KeyEvent) {
	if c.rows == nil {
		c.panel.TypedKey(ev)
		return
	}

	switch ev.Name {
	case fyne.KeyUp:
		c.focusStop(c.focus - 1)
	case fyne.KeyDown:
		c.focusStop(c.focus + 1)
	case fyne.KeyEscape:
		// Cancelling is the prompt's, never a row's: Escape backs out of the
		// whole card from whichever stop the selection is on.
		c.panel.TypedKey(ev)
	case fyne.KeyReturn, fyne.KeyEnter:
		// Offered to the focused row first, so Return on a highlighted
		// checkbox ticks it rather than committing the prompt out from under
		// someone who was reaching for the control they were looking at. A
		// row with nothing to activate declines, and the buttons commit -
		// which is still the case the moment the card opens, so the prompt
		// stays two keystrokes deep.
		if c.rowsHoldSelection() && c.rows.HandleKey(ev) {
			return
		}
		c.panel.TypedKey(ev)
	default:
		if c.rowsHoldSelection() {
			c.rows.HandleKey(ev)
			return
		}
		c.panel.TypedKey(ev)
	}
}

// rowsHoldSelection reports whether the selection is currently up in the
// extra rows rather than on the button row.
func (c *ChoiceCard) rowsHoldSelection() bool {
	return c.rows != nil && c.focus < c.rows.Rows()
}

// focusStop moves the selection to vertical stop i - a row index, or
// rows.Rows() for the button row - clamping at both ends rather than
// wrapping, the same rule ChoicePanel.Select applies horizontally. The rows
// are told which of theirs is marked, or -1 once the selection is back on
// the buttons.
func (c *ChoiceCard) focusStop(i int) {
	buttons := c.rows.Rows()
	if i > buttons {
		i = buttons
	}
	if i < 0 {
		i = 0
	}

	c.focus = i
	if i == buttons {
		c.rows.Focus(-1)
	} else {
		c.rows.Focus(i)
	}
	// Exactly one ring at full strength on the card: the buttons keep
	// showing which format is selected while the keyboard is up in the
	// rows, but muted, so the bright one is always the answer to "where am
	// I?" - see SetRingActive.
	c.panel.SetSelectionActive(i == buttons)

	if c.repaint != nil {
		c.repaint()
	}
}
