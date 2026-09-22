package heic

import (
	"context"
	"errors"
	"sync"
	"time"
)

// State separates the last operation from the still-valid effective capability.
type State struct {
	Known, Available, Checking bool
	Observation                Observation
	Err                        error
	Generation                 uint64
}

// Capability owns one app instance's refreshable support observation.
type Capability struct {
	mu            sync.Mutex
	backend       Backend
	identity      Identity
	state         State
	active        chan struct{}
	cancel        context.CancelFunc
	stopped       bool
	attempted     bool
	workers       sync.WaitGroup
	onInvalidated func()
}

func NewCapability(backend Backend, identity Identity, stored Observation) *Capability {
	c := &Capability{backend: backend, identity: identity}
	if stored.Matches(identity) {
		c.state = State{Known: true, Available: stored.Available, Observation: stored, Generation: 1}
	}
	return c
}
func (c *Capability) State() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}
func (c *Capability) Snapshot() Snapshot {
	state := c.State()
	return Snapshot{Backend: observedBackend{Backend: c.backend, capability: c, generation: state.Generation}, Available: state.Available, Generation: state.Generation}
}

type observedBackend struct {
	Backend
	capability *Capability
	generation uint64
}

func (b observedBackend) Read(ctx context.Context, data []byte, request Request) (Result, error) {
	if b.Backend == nil {
		return Result{}, ErrUnavailable
	}
	result, err := b.Backend.Read(ctx, data, request)
	if ctx.Err() == nil && errors.Is(err, ErrUnavailable) {
		b.capability.Invalidate(b.generation)
	}
	return result, err
}

// SetOnInvalidated installs a notification before image work is admitted. The
// callback runs on the reader's worker and must enqueue any UI work.
func (c *Capability) SetOnInvalidated(notify func()) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.onInvalidated = notify
}
func (c *Capability) Ensure(ctx context.Context) <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.active != nil {
		return c.active
	}
	if c.state.Known || c.attempted {
		return completedCheck()
	}
	return c.startCheck(ctx)
}
func (c *Capability) Check(ctx context.Context) <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.startCheck(ctx)
}

func completedCheck() <-chan struct{} {
	done := make(chan struct{})
	close(done)
	return done
}

// startCheck is called with mu held, so admission and worker tracking cannot
// race Stop followed by Wait.
func (c *Capability) startCheck(parent context.Context) <-chan struct{} {
	if c.active != nil {
		return c.active
	}
	if c.stopped {
		return completedCheck()
	}
	ctx, cancel := context.WithCancel(parent)
	done := make(chan struct{})
	c.active, c.cancel = done, cancel
	c.state.Checking = true
	c.attempted = true
	c.state.Err = nil
	c.workers.Go(func() {
		defer cancel()
		err := ErrUnavailable
		if c.backend != nil {
			err = c.backend.Check(ctx)
		}
		if ctx.Err() != nil {
			err = ctx.Err()
		}
		c.mu.Lock()
		c.state.Checking = false
		if !c.stopped {
			c.state.Err = err
			if err == nil || errors.Is(err, ErrUnavailable) {
				c.state.Known = true
				c.state.Available = err == nil
				c.state.Observation = Observation{Identity: c.identity, CheckedAt: time.Now().UTC(), Available: err == nil}
				c.state.Generation++
				c.state.Err = nil
			}
		}
		c.active, c.cancel = nil, nil
		close(done)
		c.mu.Unlock()
	})
	return done
}

// Stop closes check admission and cancels native work without waiting on UI.
func (c *Capability) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stopped = true
	if c.cancel != nil {
		c.cancel()
	}
}

// Wait joins admitted checks. Call Stop first when ending the app lifetime.
func (c *Capability) Wait() { c.workers.Wait() }

// Invalidate retires a missing backend observation once for its generation.
func (c *Capability) Invalidate(generation uint64) bool {
	c.mu.Lock()
	if c.stopped || !c.state.Available || c.state.Generation != generation {
		c.mu.Unlock()
		return false
	}
	c.state.Known, c.state.Available = false, false
	c.state.Observation = Observation{}
	c.state.Generation++
	c.attempted = false
	notify := c.onInvalidated
	c.mu.Unlock()
	if notify != nil {
		notify()
	}
	return true
}
