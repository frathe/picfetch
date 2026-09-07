// The background sampler that keeps a Tracker current. It lives here rather
// than at each call site because there is nothing app-specific about it: any
// window whose manually-dragged position should outlive it needs exactly
// this loop (internal/ui's main window, and every widgets.Singleton window
// that remembers where it was).

package winpos

import (
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

// PollInterval is how often Poll samples a window's on-screen position when
// all that is wanted is the last known one - a second is ample for a value
// only read back at shutdown.
const PollInterval = 1 * time.Second

// GestureInterval is the far shorter interval used when the *path* the
// window took matters rather than just where it ended up (see
// internal/wingesture). It is deliberately faster than the roughly ten
// updates a second the OS itself reports a dragged window at, so no genuine
// movement is missed to sampling luck; the duplicate readings that fall out
// of oversampling are the consumer's to discard.
const GestureInterval = 60 * time.Millisecond

// PollAt launches a background goroutine that samples win's on-screen
// position every interval for as long as the window lives, handing each
// successful reading to fn. Failed readings are dropped rather than
// reported: "couldn't ask right now" is not a position.
//
// It has to be a poller rather than an event hook: unlike a resize, a pure
// window drag-move triggers no layout pass at all, and fyne.Window has
// neither a position getter nor a "window moved" callback in the first place
// (see this package's Get, whose RunNative read is the only way to ask the
// OS directly).
//
// win must satisfy driver.NativeWindow for Get to ever succeed; checked once
// up front so a window that can't - the fyne test driver's windows, which is
// what every headless test gets - never has a poller goroutine running
// behind it at all.
//
// skip, when non-nil, suppresses a single reading whenever it reports true:
// internal/ui passes the slideshow's Active, because picture-frame mode
// full-screens the window and a full-screen reading is not the
// manually-placed position this exists to remember - it would clobber the
// value the slideshow captured on the way in for its own exit to restore.
//
// Each native read and its callback run through fyne.Do on UI. Darwin's
// RunNative does not marshal an NSWindow frame read to AppKit's main thread.
// Only one reading may be queued at a time, and skip is checked inside it.
//
// Stop is nonblocking and idempotent. It can discard a queued read and finish
// the worker without the event loop running that callback. If a native read
// has already begun, Done closes only after it returns; its cancelled result
// is discarded. Call Stop while shutting down UI, and Wait off UI when actual
// worker completion is needed. A non-native window returns a completed handle.
func PollAt(win fyne.Window, interval time.Duration, skip func() bool, fn func(x, y int)) *Poller {
	if _, ok := win.(driver.NativeWindow); !ok {
		p := &Poller{stop: make(chan struct{}), done: make(chan struct{})}
		close(p.done)
		return p
	}

	ticker := time.NewTicker(interval)
	p := startPoll(ticker.C, ticker.Stop, skip, fn, func() (int, int, bool) { return Get(win) }, fyne.Do)
	return p
}

// Poll keeps t current with win's position at PollInterval - the common
// case, and a thin binding of PollAt above, which owns the loop itself and
// every reason it has to be a poller at all. The tracker keeps its last
// reading after the returned Poller is stopped, so a save afterwards still has
// a value.
func Poll(win fyne.Window, t *Tracker, skip func() bool) *Poller {
	return PollAt(win, PollInterval, skip, t.Store)
}

// Poller owns cancellation and actual worker completion. Stop never waits.
type Poller struct {
	stop chan struct{}
	done chan struct{}
	once sync.Once
}

// Stop requests cancellation without waiting for dispatch or an active read.
func (p *Poller) Stop() {
	if p != nil {
		p.once.Do(func() { close(p.stop) })
	}
}

// Done closes when the worker and any already-started read have finished.
func (p *Poller) Done() <-chan struct{} { return p.done }

// Wait observes Done; call it off UI, after Stop.
func (p *Poller) Wait() {
	if p != nil {
		<-p.done
	}
}

// startPoll owns the same loop as PollAt; each dependency is captured per call.
func startPoll(ticks <-chan time.Time, stopTicks func(), skip func() bool, publish func(int, int), read func() (int, int, bool), dispatch func(func())) *Poller {
	p := &Poller{stop: make(chan struct{}), done: make(chan struct{})}
	go func() {
		defer close(p.done)
		defer stopTicks()
		for {
			select {
			case <-ticks:
			case <-p.stop:
				return
			}

			if p.stopped() {
				return
			}
			// 0 pending, 1 executing, 2 discarded. Stop can abandon a queued
			// callback, but Done must still cover an already active native read.
			var admission atomic.Uint32
			applied := make(chan struct{})
			dispatch(func() {
				if !admission.CompareAndSwap(0, 1) {
					return
				}
				defer close(applied)
				if p.stopped() || (skip != nil && skip()) {
					return
				}
				x, y, ok := read()
				if ok && !p.stopped() {
					publish(x, y)
				}
			})
			select {
			case <-applied:
			case <-p.stop:
				if admission.CompareAndSwap(0, 2) {
					return
				}
				<-applied
				return
			}
		}
	}()
	return p
}

func (p *Poller) stopped() bool {
	select {
	case <-p.stop:
		return true
	default:
		return false
	}
}
