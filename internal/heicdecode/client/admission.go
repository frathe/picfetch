package client

import (
	"context"
	"sync"
)

// Priority separates an interactive image from speculative/background work.
type Priority uint8

const (
	Foreground Priority = iota
	Background
)

// Callers enforce the 64-waiter limit before entering. The queue does not start
// workers: a caller retains its grant until helper and transport cleanup ends.
type admissionQueue struct {
	mu            sync.Mutex
	busy          bool
	foregroundRun int
	waiting       []*admission
}

type admission struct {
	ready    chan struct{}
	ctx      context.Context
	once     sync.Once
	queue    *admissionQueue
	priority Priority
	granted  bool
}

func (q *admissionQueue) enter(ctx context.Context, priority Priority) *admission {
	a := &admission{ready: make(chan struct{}), ctx: ctx, queue: q, priority: priority}
	q.mu.Lock()
	q.waiting = append(q.waiting, a)
	if !q.busy {
		q.promote()
	}
	q.mu.Unlock()
	return a
}

func (a *admission) wait() error {
	select {
	case <-a.ready:
		return a.ctx.Err()
	case <-a.ctx.Done():
		return a.ctx.Err()
	}
}

func (a *admission) release() {
	a.once.Do(func() {
		q := a.queue
		q.mu.Lock()
		defer q.mu.Unlock()
		if a.granted {
			q.promote()
			return
		}
		for i, waiting := range q.waiting {
			if waiting == a {
				copy(q.waiting[i:], q.waiting[i+1:])
				q.waiting[len(q.waiting)-1] = nil
				q.waiting = q.waiting[:len(q.waiting)-1]
				return
			}
		}
	})
}

// Called under mu when the previous grant has fully retired. Preserve FIFO
// within each priority, with a background grant after at most three foreground
// grants. Cancelled requests never consume a grant or a fairness turn.
func (q *admissionQueue) promote() {
	active := q.waiting[:0]
	for _, a := range q.waiting {
		if a.ctx.Err() == nil {
			active = append(active, a)
		}
	}
	clear(q.waiting[len(active):])
	q.waiting = active
	foreground, background := -1, -1
	for i, a := range q.waiting {
		if a.priority == Foreground && foreground < 0 {
			foreground = i
		}
		if a.priority == Background && background < 0 {
			background = i
		}
	}
	index := foreground
	if foreground < 0 || (background >= 0 && q.foregroundRun >= 3) {
		index = background
	}
	if index < 0 {
		q.busy = false
		q.foregroundRun = 0
		return
	}
	a := q.waiting[index]
	copy(q.waiting[index:], q.waiting[index+1:])
	q.waiting[len(q.waiting)-1] = nil
	q.waiting = q.waiting[:len(q.waiting)-1]
	if a.priority == Foreground {
		q.foregroundRun = min(q.foregroundRun+1, 3)
	} else {
		q.foregroundRun = 0
	}
	q.busy = true
	a.granted = true
	close(a.ready)
}
