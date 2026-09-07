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
)

type clipboardWork struct {
	lifecycle  requestLifecycle
	workers    sync.WaitGroup
	pending    atomic.Bool
	closed     bool
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

func (v *viewer) beginClipboardCopy(imageBound bool) (requestToken, func(), bool) {
	if v.clipboardWork.closed || v.clipboardBusy() {
		return requestToken{}, nil, false
	}
	v.clipboardWork.pending.Store(true)
	v.clipboardWork.imageBound = imageBound
	token := v.clipboardWork.lifecycle.begin()
	finished := v.clipboard.Begin()
	done := sync.OnceFunc(func() {
		token.cancelContext()
		v.clipboardWork.pending.Store(false)
		finished()
	})
	return token, done, true
}

// completeClipboardCopy submits the only result effect. The worker can finish
// before UI delivery; the operation's Signal includes that delivery. Cancelled
// work finishes directly, and a queued callback rechecks before touching UI.
func (v *viewer) completeClipboardCopy(token requestToken, done func(), apply func()) {
	if !token.current() {
		done()
		return
	}
	v.clipboardWork.ui.Do(func() {
		defer done()
		if token.current() {
			apply()
		}
	})
}

func (v *viewer) cancelImageClipboard() {
	if v.clipboardWork.imageBound {
		v.clipboardWork.lifecycle.invalidate()
	}
}

func (v *viewer) closeClipboardWork() {
	v.clipboardWork.closed = true
	v.clipboardWork.lifecycle.invalidate()
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
