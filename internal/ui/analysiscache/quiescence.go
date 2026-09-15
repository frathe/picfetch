package analysiscache

import (
	"context"
	"sync/atomic"
)

// joinRevokedWriters requests UI retirement after the manager has revoked
// shared write leases, then joins the returned producer barriers off UI.
func joinRevokedWriters(ctx context.Context, queue UIQueue, notify func(), retire func() []<-chan struct{}) error {
	const (
		queued uint32 = iota
		claimed
		abandoned
	)
	var state atomic.Uint32
	ready := make(chan []<-chan struct{}, 1)
	queue.Do(func() {
		if !state.CompareAndSwap(queued, claimed) {
			return
		}
		if ctx.Err() != nil {
			ready <- nil
			return
		}
		ready <- retire()
	})
	notify()
	var barriers []<-chan struct{}
	select {
	case barriers = <-ready:
	case <-ctx.Done():
		if state.CompareAndSwap(queued, abandoned) {
			return ctx.Err()
		}
		// UI owns the request now. Its reply may still be in progress, and
		// any producers it retires remain part of maintenance completion.
		barriers = <-ready
	}
	for _, barrier := range barriers {
		if barrier != nil {
			<-barrier
		}
	}
	return ctx.Err()
}
