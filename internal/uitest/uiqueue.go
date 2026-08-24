package uitest

import "sync"

// UIQueue collects the callbacks a background goroutine hands over for the
// UI goroutine to run, and runs them on whoever calls Drain.
//
// It exists because Fyne's test driver is not a marshaling point: its
// DoFromGoroutine calls the function inline on the calling goroutine, so a
// worker's fyne.Do body runs *on the worker*, concurrently with the test
// goroutine that spawned it. Under -race that is a genuine data race on
// every widget and every unsynchronized field the body touches - the same
// code being perfectly safe in the app, where the real driver queues the
// callback onto the one UI goroutine.
//
// A feature that hands its completions to a UIQueue gets those production
// semantics back under test: the callback is deferred, and runs serialized
// on the goroutine that drains it - which for a test is the test goroutine
// itself, at a point of its own choosing.
//
// The zero value is ready to use. Do is safe from any goroutine; Drain is
// for one goroutine at a time, the one the test is running on.
type UIQueue struct {
	mu      sync.Mutex
	pending []func()
}

// Do queues f, and never runs it on the calling goroutine. A nil f queues
// fine and panics whoever eventually Drains it - the same way fyneQueue's
// Do(nil) panics inside the real driver, so this seam doesn't quietly
// swallow a caller bug the app wouldn't.
func (q *UIQueue) Do(f func()) {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.pending = append(q.pending, f)
}

// Drain runs everything queued so far, in the order it was queued, on the
// calling goroutine, and reports whether it ran anything.
//
// The lock is dropped before the callbacks run: a callback may queue more
// work, directly or by spawning a worker that does, and holding the lock
// across one would deadlock. Work queued during a Drain therefore lands in
// the *next* one, so a caller that needs quiescence loops until Drain
// reports false.
//
// The batch is already detached from pending by the time the loop below
// runs it, so a callback that panics aborts every callback still queued
// behind it in that same batch - they are simply never reached, not lost
// to a corrupted pending slice.
func (q *UIQueue) Drain() bool {
	q.mu.Lock()
	batch := q.pending
	q.pending = nil
	q.mu.Unlock()

	for _, f := range batch {
		f()
	}

	return len(batch) > 0
}

// Len is how many callbacks are waiting for a Drain. No current adopter's
// tests assert on it - internal/ui/grid's asserts on the drained effect
// instead - but it's here for whichever of the other packages with the
// same latent worker-races-the-test-goroutine exposure (deletion,
// slideshow, favorites, exifwin, spiral, toast) adopts this queue next,
// in case one of them wants to show a worker deferred its paint rather
// than running it.
func (q *UIQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()

	return len(q.pending)
}
