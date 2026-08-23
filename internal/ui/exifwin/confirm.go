package exifwin

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/ui/widgets"
)

// cancelChoice and confirmChoice mirror internal/ui/favorites' own
// confirmation: Cancel first/left and so the default selection, the
// confirming action second/right, so a prompt never opens with the action
// already under Return.
const (
	cancelChoice  = 0
	confirmChoice = 1
)

// confirmation describes one keyboard-driven two-choice prompt: a title, a
// message, the confirming button's own label and importance, and what runs
// on each way the prompt can go. Same shape as favorites' own confirmation -
// see that package's showConfirm for the reasoning behind every field here.
type confirmation struct {
	title      string
	message    string
	action     string            // the confirming button's label
	importance widget.Importance // widget.DangerImportance for a destructive action
	onConfirm  func()
	onCancel   func() // the Cancel choice and Escape both; nil for "just close"
	onClosed   func() // whichever way the dialog goes; nil for nothing
}

// showConfirm raises c as a two-choice confirmation and hands it the
// keyboard. Parented on w.win.Window() (the EXIF window itself, not the main
// image window) so it floats above the panel that opened it; a no-op when
// that window is nil, since the EXIF window has since closed.
//
// See favorites.(*Feature).showConfirm's doc comment for why this goes
// through a widgets.ChoicePanel rather than dialog.NewConfirm, and for the
// onClosed-before-onConfirm/onCancel ordering ChoicePanel itself guarantees.
//
// A second call while one confirmation is already up hides it first (the
// same superseded-dialog guard favorites' own callers use), so hideConfirm
// is the only place that needs to know how to tear one down.
func (w *Window) showConfirm(c confirmation) dialog.Dialog {
	win := w.win.Window()
	if win == nil {
		return nil
	}

	w.hideConfirm()

	message := &widget.Label{
		Text:      c.message,
		Alignment: fyne.TextAlignCenter,
	}

	var confirm dialog.Dialog
	panel := widgets.NewChoicePanel(nil,
		widgets.Choice{Label: lang.L("Cancel"), OnChosen: c.onCancel},
		widgets.Choice{
			Label:      c.action,
			Importance: c.importance,
			OnChosen:   c.onConfirm,
		},
	)
	panel.SetOnDismiss(func() { confirm.Hide() })
	panel.SetOnCancel(c.onCancel)

	confirm = dialog.NewCustomWithoutButtons(c.title, container.NewVBox(message, panel), win)
	confirm.SetOnClosed(func() {
		w.confirm = nil
		if c.onClosed != nil {
			c.onClosed()
		}
	})
	w.confirm = confirm
	confirm.Show()
	// After Show: Fyne can only focus an object that is already part of an
	// overlay it can walk to. Load-bearing beyond that - widgets.Singleton
	// registers Canvas().SetOnTypedKey Escape -> win.Close(); with Focused()
	// left nil, Escape would close the EXIF window behind this prompt
	// instead of cancelling it. A focused ChoicePanel swallows Escape
	// itself (see ChoicePanel.TypedKey), so the window never sees it.
	win.Canvas().Focus(panel)

	return confirm
}

// hideConfirm tears down whatever confirmation is currently up, if any.
// Safe to call with nothing showing. It does not clear w.pending: requestStrip
// assigns pending after showConfirm returns, and showConfirm itself calls
// this first to replace a previous prompt — nil-ing pending here would
// drop the URI dismissStalePending needs to notice a navigation.
func (w *Window) hideConfirm() {
	if w.confirm != nil {
		w.confirm.Hide()
		w.confirm = nil
	}
}
