package mosaicwin

import (
	"context"
	"sync"

	"fyne.io/fyne/v2"
)

type UIQueue interface {
	Do(func())
	Drain() bool
}

type fyneQueue struct{}

func (fyneQueue) Do(apply func()) { fyne.Do(apply) }
func (fyneQueue) Drain() bool     { return false }

type revisionLifecycle struct {
	mu     sync.Mutex
	value  uint64
	cancel context.CancelFunc
}

func (l *revisionLifecycle) begin() (context.Context, uint64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cancel != nil {
		l.cancel()
	}
	l.value++
	ctx, cancel := context.WithCancel(context.Background())
	l.cancel = cancel

	return ctx, l.value
}

func (l *revisionLifecycle) invalidate() {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.cancel != nil {
		l.cancel()
		l.cancel = nil
	}
	l.value++
}

func (l *revisionLifecycle) current(value uint64) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	return value == l.value
}

type workTracker struct {
	mu     sync.Mutex
	active int
	idle   chan struct{}
}

func (w *workTracker) Go(run func()) {
	w.mu.Lock()
	if w.active == 0 {
		w.idle = make(chan struct{})
	}
	w.active++
	w.mu.Unlock()

	go func() {
		defer func() {
			w.mu.Lock()
			w.active--
			if w.active == 0 {
				close(w.idle)
				w.idle = nil
			}
			w.mu.Unlock()
		}()
		run()
	}()
}

func (w *workTracker) wait(ctx context.Context) error {
	w.mu.Lock()
	idle := w.idle
	w.mu.Unlock()
	if idle == nil {
		return nil
	}
	select {
	case <-idle:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (w *workTracker) activeCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.active
}
