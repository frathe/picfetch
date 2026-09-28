// Package requestlife owns request identity and disposable final delivery.
package requestlife

import (
	"context"
	"sync"
)

// Owner owns one logical request stream. Its zero value is usable and its
// methods are safe for concurrent use. Do not copy it after first use.
type Owner struct {
	mu       sync.Mutex
	revision uint64
	cancel   context.CancelFunc
}

// Token captures one immutable request identity and context. Copies share the
// same cancellation; releasing an old token cannot affect its replacement.
type Token struct {
	owner    *Owner
	ctx      context.Context
	cancel   context.CancelFunc
	revision uint64
}

// Delivery owns one disposable terminal result. Defer Abandon in the worker,
// then Dispatch to transfer release to the existing UI dispatcher. Delivery
// owns neither that queue nor worker/operation completion. Do not copy it.
type Delivery struct {
	mu    sync.Mutex
	token Token
	state deliveryState
}

type deliveryState uint8

const (
	deliveryOpen deliveryState = iota
	deliveryQueued
	deliveryDone
)

// Begin captures parent and supersedes the previous request, even if parent is
// already canceled. Parent must be non-nil, as for context.WithCancel.
func (o *Owner) Begin(parent context.Context) Token {
	ctx, cancel := context.WithCancel(parent)
	o.mu.Lock()
	if o.cancel != nil {
		o.cancel()
	}
	o.revision++
	revision := o.revision
	o.cancel = cancel
	o.mu.Unlock()
	return Token{owner: o, ctx: ctx, cancel: cancel, revision: revision}
}

// Invalidate cancels and supersedes current work without admitting replacement
// work. It is safe before Begin and permits later Begin for close/reopen.
func (o *Owner) Invalidate() uint64 {
	o.mu.Lock()
	o.revision++
	if o.cancel != nil {
		o.cancel()
		o.cancel = nil
	}
	revision := o.revision
	o.mu.Unlock()
	return revision
}

// Revision observes the last admitted Begin or invalidation identity.
func (o *Owner) Revision() uint64 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.revision
}

// Context returns the captured child context; a zero token returns nil.
func (t Token) Context() context.Context { return t.ctx }

// Current is an observation, not a transaction spanning the caller's effects.
// Feature-specific identities and checks after reentrant calls remain required.
func (t Token) Current() bool {
	if t.owner == nil || t.ctx == nil {
		return false
	}
	t.owner.mu.Lock()
	defer t.owner.mu.Unlock()
	return t.revision == t.owner.revision && t.ctx.Err() == nil
}

// Revision is the immutable identity captured at Begin, or zero for a zero token.
func (t Token) Revision() uint64 { return t.revision }

// Release idempotently cancels this token without advancing its owner or
// canceling its parent. Releasing a zero token is harmless.
func (t Token) Release() {
	if t.cancel != nil {
		t.cancel()
	}
}

// FinalDelivery creates one handoff for a wholly disposable terminal result.
// Retained work and committed disk effects must keep their own protocols.
func (t Token) FinalDelivery() *Delivery { return &Delivery{token: t} }

// Abandon releases a result that was never handed to its dispatcher.
func (d *Delivery) Abandon() {
	d.mu.Lock()
	if d.state != deliveryOpen {
		d.mu.Unlock()
		return
	}
	d.state = deliveryDone
	d.mu.Unlock()
	d.token.Release()
}

// Dispatch transfers release before invoking dispatch, which may run inline.
// A stale result skips apply but still releases and calls finish exactly once
// when delivered. Cancellation never runs finish or waits for queue draining.
// Both callbacks run outside module locks; either may start a newer request.
func (d *Delivery) Dispatch(dispatch func(func()), apply func(), finish func()) {
	d.mu.Lock()
	if d.state != deliveryOpen {
		d.mu.Unlock()
		return
	}
	d.state = deliveryQueued
	d.mu.Unlock()

	dispatch(func() {
		d.mu.Lock()
		if d.state != deliveryQueued {
			d.mu.Unlock()
			return
		}
		d.state = deliveryDone
		d.mu.Unlock()

		defer func() {
			d.token.Release()
			if finish != nil {
				finish()
			}
		}()
		if d.token.Current() && apply != nil {
			apply()
		}
	})
}
