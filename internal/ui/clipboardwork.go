package ui

import (
	"context"
	"image"
	"image/png"
	"io"
	"sync"
	"sync/atomic"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/requestlife"
)

type clipboardWork struct {
	lifecycle  requestlife.Owner
	workers    sync.WaitGroup
	pending    atomic.Bool
	closed     atomic.Bool
	imageBound bool
	ui         clipboardUIQueue
	encode     func(io.Writer, image.Image) error
}

type clipboardUIQueue interface {
	Do(func())
	Drain() bool
}

type fyneClipboardQueue struct{}

func (fyneClipboardQueue) Do(fn func()) { fyne.Do(fn) }
func (fyneClipboardQueue) Drain() bool  { return false }

func newClipboardWork() clipboardWork {
	return clipboardWork{ui: fyneClipboardQueue{}, encode: png.Encode}
}

// clipboardBusy keeps all clipboard writes in the viewer behind the same
// admission rule. A competing command leaves the current completion intact.
func (v *viewer) clipboardBusy() bool {
	if !v.clipboardWork.pending.Load() {
		return false
	}
	v.ShowToast(lang.L("finishing the copy - try again in a moment"))
	return true
}

func (v *viewer) beginClipboardCopy(imageBound bool) (requestlife.Token, func(), bool) {
	if v.clipboardWork.closed.Load() || v.clipboardBusy() {
		return requestlife.Token{}, nil, false
	}
	v.clipboardWork.pending.Store(true)
	v.clipboardWork.imageBound = imageBound
	token := v.clipboardWork.lifecycle.Begin(context.Background())
	finished := v.clipboard.Begin()
	v.syncMenus()
	done := sync.OnceFunc(func() {
		token.Release()
		v.clipboardWork.pending.Store(false)
		finished()
	})
	return token, done, true
}

// completeClipboardCopy submits the only result effect. The worker can finish
// before UI delivery; the operation's Signal includes that delivery and restored
// menu availability. Shutdown cancellation finishes directly without UI delivery.
// Other cancellations still refresh availability, but discard their result effect.
func (v *viewer) completeClipboardCopy(token requestlife.Token, done func(), apply func()) {
	if !token.Current() && v.clipboardWork.closed.Load() {
		done()
		return
	}
	v.clipboardWork.ui.Do(func() {
		defer done()
		if token.Current() && apply != nil {
			apply()
		}
		// Release admission before observing it for menus, on the same UI turn
		// as done so an older completion cannot clear a newer operation.
		v.clipboardWork.pending.Store(false)
		v.syncMenus()
	})
}

func (v *viewer) cancelImageClipboard() {
	if v.clipboardWork.imageBound {
		v.clipboardWork.lifecycle.Invalidate()
	}
}

func (v *viewer) closeClipboardWork() {
	v.clipboardWork.closed.Store(true)
	v.clipboardWork.lifecycle.Invalidate()
}

// clipboardContextWriter checks cancellation at encoder output boundaries.
// An encoder already reading pixels must return to a Write before it can stop.
type clipboardContextWriter struct {
	ctx context.Context
	out io.Writer
}

func (w clipboardContextWriter) Write(p []byte) (int, error) {
	if err := w.ctx.Err(); err != nil {
		return 0, err
	}
	n, err := w.out.Write(p)
	if err == nil {
		err = w.ctx.Err()
	}
	return n, err
}
