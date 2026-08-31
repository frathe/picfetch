package ui

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

	paused         bool
	resume         chan struct{}
	observed       chan struct{}
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
	p.paused = true
	p.resume = make(chan struct{})
	p.observed = make(chan struct{})
	p.observedClosed = false
	return true
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

// advance runs a single frame mutation unless a pause began after the loop's
// last wait and before its timer fired.
func (p *animationPause) advance(fn func()) bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.paused {
		return false
	}
	fn()
	return true
}

func (p *animationPause) unpause() {
	p.mu.Lock()
	defer p.mu.Unlock()
	if !p.paused {
		return
	}

	p.paused = false
	p.markObservedLocked()
	close(p.resume)
	p.resume = nil
}

func (p *animationPause) markObservedLocked() {
	if p.observed != nil && !p.observedClosed {
		close(p.observed)
		p.observedClosed = true
	}
}
