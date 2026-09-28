// asyncOpUI: the progress-UI bookkeeping shared by the folder scan
// (drop.go) and the background reorder (sort.go).

package ui

import (
	"context"

	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/requestlife"
)

// asyncOpUI combines the scan's local request owner with the progress state
// also used by sorting. The pilot separates request mechanics from presentation
// without changing active flags, completion generations or progress widgets.
//
// Deliberately viewer-independent: what to *do* about a cancelled operation
// - put the drop zone back, repaint, toast - differs between the two and
// stays at the call sites.
//
// A value field on viewer, never copied: it holds a lifecycle mutex and a
// completion mutex.
type asyncOpUI struct {
	asyncProgressUI
	lifecycle requestLifecycle
}

// sortOpUI is the pilot owner; scan keeps its local lifecycle until ticket 04
// adopts the same request contract (or ticket 08 restores the local pilot).
type sortOpUI struct {
	asyncProgressUI
	lifecycle requestlife.Owner
}

// asyncProgressUI owns only presentation and per-operation completion. Request
// ownership is separate so the pilot can leave scan's lifetime unchanged.
type asyncProgressUI struct {
	active  bool
	done    completion.Signal
	art     *canvas.Image // the scan's Trane-digging art; nil for the sort
	spinner *widget.ProgressBarInfinite
	label   *widget.Label
}

func (o *sortOpUI) begin(parent context.Context) (requestlife.Token, func()) {
	token := o.lifecycle.Begin(parent)
	o.active = true
	return token, o.done.Begin()
}

func (o *sortOpUI) invalidate() uint64 {
	revision := o.lifecycle.Invalidate()
	if o.active {
		o.finish()
	}
	return revision
}

// begin supersedes any request already in flight, marks the operation
// active, and begins a fresh completion generation. The finisher is
// returned so the caller can capture it: a superseded request must still
// finish its own generation without touching the one a newer request now
// owns - see internal/completion.
func (o *asyncOpUI) begin() (requestToken, func()) {
	token := o.lifecycle.begin()
	o.active = true

	return token, o.done.Begin()
}

// show reveals the progress widgets. Separate from begin because the scan
// sets its label's text first. Nil-guarded: the sort instance has no art.
func (o *asyncProgressUI) show() {
	if o.art != nil {
		o.art.Show()
	}
	o.spinner.Show()
	o.label.Show()
}

// finish clears the active flag and hides the progress widgets. Called by
// the completion step of whichever token is still current - never by a
// stale one, which must not report "nothing in flight" while a newer
// request is still running.
func (o *asyncProgressUI) finish() {
	o.active = false
	if o.art != nil {
		o.art.Hide()
	}
	o.spinner.Hide()
	o.label.Hide()
}

// invalidate supersedes and cancels the current request, finishing the UI
// only if this operation was actually active. Returns the new revision.
func (o *asyncOpUI) invalidate() uint64 {
	revision := o.lifecycle.invalidate()
	if o.active {
		o.finish()
	}
	return revision
}

// cancel is invalidate guarded by the flag, reporting whether there was
// anything to cancel. The caller decides what the cancellation means.
func (o *asyncOpUI) cancel() bool {
	if !o.active {
		return false
	}
	o.invalidate()
	return true
}
