package compare

import (
	"context"
	"sync"
)

// workTracker is a reusable, context-selectable worker barrier. Each busy
// epoch owns its own channel, so a timed-out waiter never leaves a goroutine
// behind and cannot race a later Add after the prior epoch settles.
type workTracker struct {
	mu     sync.Mutex
	active int
	idle   chan struct{}
}

func (w *workTracker) Add(delta int) {
	if delta == 0 {
		return
	}

	w.mu.Lock()
	defer w.mu.Unlock()
	previous := w.active
	next := previous + delta
	if next < 0 {
		panic("compare: negative work tracker count")
	}
	if previous == 0 && next > 0 {
		w.idle = make(chan struct{})
	}
	w.active = next
	if previous > 0 && next == 0 {
		close(w.idle)
		w.idle = nil
	}
}

func (w *workTracker) Done() { w.Add(-1) }

func (w *workTracker) Go(run func()) {
	w.Add(1)
	go func() {
		defer w.Done()
		run()
	}()
}

func (w *workTracker) Wait() {
	if done := w.done(); done != nil {
		<-done
	}
}

func (w *workTracker) WaitContext(ctx context.Context) error {
	done := w.done()
	if done == nil {
		return nil
	}
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *workTracker) done() <-chan struct{} {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.idle
}
