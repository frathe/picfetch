package display

import (
	"context"
	"sync"
)

// animationPause serializes animated-frame advancement with the stable
// source capture used by Copy Selection. Pausing does not cancel the load
// token or start a second animation loop: the existing loop waits here and
// resumes with a fresh frame delay when the mode ends.
type animationPause struct {
	mu sync.Mutex

	acquisition    uint64
	resume         chan struct{}
	observed       chan struct{}
	changed        chan struct{}
	paused         bool
	observedClosed bool
}

// pause captures while frame advancement is excluded, then holds the gate.
// It reports false when another owner has already paused the loop.
func (p *animationPause) pause(capture func()) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.paused {
		return false
	}

	capture()
	p.acquisition++
	p.paused = true
	p.resume = make(chan struct{})
	p.observed = make(chan struct{})
	p.observedClosed = false
	p.notifyLocked()
	return true
}

// phase binds a pending delay and its UI delivery to the current acquisition.
func (p *animationPause) phase() (uint64, <-chan struct{}, bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.changed == nil {
		p.changed = make(chan struct{})
	}
	return p.acquisition, p.changed, !p.paused
}

func (p *animationPause) notifyLocked() {
	if p.changed != nil {
		close(p.changed)
	}
	p.changed = make(chan struct{})
}

// wait blocks an animation loop between frames while paused. The context is
// the load token, so navigation and test cleanup still stop the loop at once.
func (p *animationPause) wait(ctx context.Context) bool {
	p.mu.Lock()
	if !p.paused {
		p.mu.Unlock()
		return true
	}
	resume := p.resume
	p.markObservedLocked()
	p.mu.Unlock()

	select {
	case <-resume:
		return true
	case <-ctx.Done():
		return false
	}
}

// advance rejects a delivery if capture began after its delay was scheduled,
// including captures already released before the queued callback runs.
func (p *animationPause) advance(acquisition uint64, fn func()) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.paused || p.acquisition != acquisition {
		return false
	}
	fn()
	return true
}

func (p *animationPause) unpause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.resumeLocked()
}

func (p *animationPause) resumeLocked() {
	if !p.paused {
		return
	}

	p.paused = false
	p.markObservedLocked()
	close(p.resume)
	p.resume = nil
	p.notifyLocked()
}

func (p *animationPause) markObservedLocked() {
	if p.observed != nil && !p.observedClosed {
		close(p.observed)
		p.observedClosed = true
	}
}

// release binds unpausing to the exact acquisition, so retired callbacks cannot
// resume a later selection (even within the same presentation).
func (p *animationPause) release() func() {
	p.mu.Lock()
	acquisition := p.acquisition
	p.mu.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			p.mu.Lock()
			defer p.mu.Unlock()
			if !p.paused || p.acquisition != acquisition {
				return
			}
			p.resumeLocked()
		})
	}
}
