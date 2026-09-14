package ui

import (
	"context"
	"time"

	"fyne.io/fyne/v2"
)

type searchOverlayWait struct {
	cancel context.CancelFunc
	notice chan struct{}
}

type searchOverlayQueue struct{}

func (searchOverlayQueue) Do(fn func()) { fyne.Do(fn) }
func (searchOverlayQueue) Drain() bool  { return false }

// Fyne exposes no overlay-close observer. Only while a ranked result is held
// behind an overlay, check for its return on UI with one acknowledged callback.
// Cancellation releases the worker without needing the UI queue to drain.
func (v *viewer) watchSearchOverlay() {
	if v.stopping || v.searchView.overlay != nil {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	wait := &searchOverlayWait{cancel: cancel, notice: make(chan struct{}, 1)}
	v.searchView.overlay = wait
	queue := v.searchView.overlayUI
	if queue == nil {
		queue = searchOverlayQueue{}
	}
	v.searchView.overlayWorkers.Add(1)
	go func() {
		defer v.searchView.overlayWorkers.Done()
		ticker := time.NewTicker(25 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
			applied := make(chan struct{})
			queue.Do(func() {
				defer close(applied)
				if ctx.Err() != nil || v.stopping || v.searchView.overlay != wait {
					return
				}
				if !v.searchActive() || v.searchView.pending == nil || v.win.Canvas().Overlays().Top() == nil {
					v.stopSearchOverlayWait()
					v.flushSearchPresentation()
				}
			})
			select {
			case wait.notice <- struct{}{}:
			default:
			}
			select {
			case <-ctx.Done():
				return
			case <-applied:
			}
		}
	}()
}
func (v *viewer) stopSearchOverlayWait() {
	if wait := v.searchView.overlay; wait != nil {
		wait.cancel()
		v.searchView.overlay = nil
	}
}
