// Package decodepool bounds background decode work. It combines per-key
// in-flight claims with demand-started workers and a completion counter: Grid
// thumbnails and speculative display preloads share the same small, testable
// ownership shape without creating one waiting goroutine for every source.
//
// It is deliberately viewer-independent: no Fyne types, no UI marshaling. The
// caller decides what a key is, what staleness means, and when to release.
package decodepool

import (
	"context"
	"sync"
)

// Pool bounds work over keys of type K, each claim carrying a value of type V
// that says what the in-flight work is for. The zero Pool is not usable; call
// New.
type Pool[K, V comparable] struct {
	limit    int
	inflight sync.Map
	pending  sync.WaitGroup

	mu            sync.Mutex
	high, low     []job[K, V]
	workers       int
	cancellations []job[K, V]
	cancelling    bool
}

// Queue binds a related set of jobs to one context. A Queue's cancellation
// removes every job still waiting for a worker and calls it with acquired set
// to false. Reusing one Queue for a logical work session therefore makes one
// cancellation cleanup pass, rather than one goroutine per queued job.
type Queue[K, V comparable] struct {
	pool *Pool[K, V]
	ctx  context.Context

	// stop unregisters the cancellation callback for Pool.Go's one-job
	// convenience queue. All fields below are guarded by pool.mu.
	stop        func() bool
	oneShot     bool
	cancelled   bool
	outstanding int
	stopped     bool
}

type job[K, V comparable] struct {
	queue *Queue[K, V]
	fn    func(acquired bool)
}

// New returns a pool that runs at most limit decode functions at once. limit
// must be positive.
func New[K, V comparable](limit int) *Pool[K, V] {
	if limit <= 0 {
		panic("decodepool: limit must be positive")
	}
	return &Pool[K, V]{limit: limit}
}

// Claim records that work is being spawned for key toward v, reporting
// whether the caller should actually spawn it. False means identical work is
// already in flight. A claim for the same key with a different v succeeds and
// overwrites: the caller has moved on to different work for that key, and the
// superseded worker's Release will not clobber the new claim (see Release).
func (p *Pool[K, V]) Claim(key K, v V) bool {
	if existing, ok := p.inflight.Load(key); ok && existing == v {
		return false
	}
	p.inflight.Store(key, v)
	return true
}

// Release drops key's claim, but only if it still names v - so a superseded
// worker finishing late cannot drop a newer claim made over it.
func (p *Pool[K, V]) Release(key K, v V) {
	p.inflight.CompareAndDelete(key, v)
}

// Begin binds a queue of related jobs to ctx. Submit the whole cancellable
// work session through the returned Queue so cancellation can release pending
// claims and accounting without waiting for an occupied decode slot.
func (p *Pool[K, V]) Begin(ctx context.Context) *Queue[K, V] {
	return p.begin(ctx, false)
}

func (p *Pool[K, V]) begin(ctx context.Context, oneShot bool) *Queue[K, V] {
	q := &Queue[K, V]{pool: p, ctx: ctx, oneShot: oneShot}
	q.stop = context.AfterFunc(ctx, func() { p.cancel(q) })
	return q
}

// Go adds interactive work to the pool without blocking its caller. It is a
// convenience for a single job; a logical session with many jobs should call
// Begin once and submit through its Queue instead.
//
// acquired is false when ctx was cancelled while the job waited. fn is called
// either way, so a caller can always undo its Claim, but must not start source
// work when acquired is false. The slot is held for the whole of fn and
// released when it returns.
func (p *Pool[K, V]) Go(ctx context.Context, fn func(acquired bool)) {
	p.begin(ctx, true).Go(fn)
}

// Go adds interactive work to q. Newer interactive jobs run before older ones
// so a current Grid cell is not stuck behind cells that were already recycled
// by scrolling.
func (q *Queue[K, V]) Go(fn func(acquired bool)) {
	q.submit(fn, false)
}

// GoLow adds background work to q. Low-priority jobs run only when no
// interactive job is waiting; duplicate hashing uses this so image cells keep
// responding while a large collection is analysed.
func (q *Queue[K, V]) GoLow(fn func(acquired bool)) {
	q.submit(fn, true)
}

func (q *Queue[K, V]) submit(fn func(acquired bool), low bool) {
	p := q.pool
	p.pending.Add(1)

	p.mu.Lock()
	item := job[K, V]{queue: q, fn: fn}
	q.outstanding++
	if q.cancelled || q.ctx.Err() != nil {
		p.cancelLocked(q)
		p.queueCancellationLocked(item)
		p.mu.Unlock()
		return
	}
	if low {
		p.low = append(p.low, item)
	} else {
		p.high = append(p.high, item)
	}
	p.startWorkersLocked()
	p.mu.Unlock()
}

func (p *Pool[K, V]) cancel(q *Queue[K, V]) {
	p.mu.Lock()
	p.cancelLocked(q)
	p.mu.Unlock()
}

func (p *Pool[K, V]) cancelLocked(q *Queue[K, V]) {
	if q.cancelled {
		return
	}
	q.cancelled = true
	p.high = p.removeQueueLocked(p.high, q)
	p.low = p.removeQueueLocked(p.low, q)
	p.startCancellationWorkerLocked()
}

func (p *Pool[K, V]) removeQueueLocked(jobs []job[K, V], q *Queue[K, V]) []job[K, V] {
	kept := 0
	for _, item := range jobs {
		if item.queue == q {
			p.cancellations = append(p.cancellations, item)
			continue
		}
		jobs[kept] = item
		kept++
	}
	var zero job[K, V]
	for i := kept; i < len(jobs); i++ {
		jobs[i] = zero
	}
	if kept == 0 {
		return nil
	}
	return jobs[:kept]
}

func (p *Pool[K, V]) queueCancellationLocked(item job[K, V]) {
	p.cancellations = append(p.cancellations, item)
	p.startCancellationWorkerLocked()
}

func (p *Pool[K, V]) startCancellationWorkerLocked() {
	if p.cancelling || len(p.cancellations) == 0 {
		return
	}
	p.cancelling = true
	go p.drainCancellations()
}

func (p *Pool[K, V]) drainCancellations() {
	for {
		p.mu.Lock()
		if len(p.cancellations) == 0 {
			p.cancelling = false
			p.mu.Unlock()
			return
		}
		items := p.cancellations
		p.cancellations = nil
		p.mu.Unlock()

		for _, item := range items {
			p.run(item, false)
		}
	}
}

func (p *Pool[K, V]) startWorkersLocked() {
	for p.workers < p.limit && (len(p.high) != 0 || len(p.low) != 0) {
		p.workers++
		go p.work()
	}
}

func (p *Pool[K, V]) work() {
	for {
		p.mu.Lock()
		item, ok := p.nextLocked()
		if !ok {
			p.workers--
			p.mu.Unlock()
			return
		}
		p.mu.Unlock()

		p.run(item, item.queue.ctx.Err() == nil)
	}
}

func (p *Pool[K, V]) nextLocked() (job[K, V], bool) {
	if n := len(p.high); n != 0 {
		item := p.high[n-1]
		var zero job[K, V]
		p.high[n-1] = zero
		p.high = p.high[:n-1]
		return item, true
	}
	if n := len(p.low); n != 0 {
		item := p.low[0]
		var zero job[K, V]
		p.low[0] = zero
		p.low = p.low[1:]
		if len(p.low) == 0 {
			p.low = nil
		}
		return item, true
	}
	return job[K, V]{}, false
}

func (p *Pool[K, V]) run(item job[K, V], acquired bool) {
	defer p.finished(item)
	item.fn(acquired)
}

func (p *Pool[K, V]) finished(item job[K, V]) {
	q := item.queue
	var stop func() bool
	p.mu.Lock()
	q.outstanding--
	if q.oneShot && q.outstanding == 0 && !q.stopped {
		q.stopped = true
		stop = q.stop
	}
	p.mu.Unlock()
	if stop != nil {
		_ = stop()
	}
	p.pending.Done()
}

// Wait blocks until every function submitted so far has returned. The
// application never needs this; tests do.
func (p *Pool[K, V]) Wait() {
	p.pending.Wait()
}
