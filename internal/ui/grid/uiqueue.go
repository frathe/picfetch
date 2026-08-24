// The seam a finished background job hands its widget work across, and the
// app's own implementation of it.

package grid

import "fyne.io/fyne/v2"

// uiQueue is how a decode-pool worker gets its result onto the UI
// goroutine. fyneQueue in the app; this package's tests install
// uitest.UIQueue instead, because Fyne's test driver is not a marshaling
// point - its DoFromGoroutine runs the callback inline on the calling
// (worker) goroutine, so the completion bodies below would otherwise touch
// canvas.Image, widget.Label, g.matches and g.groupSizes concurrently with
// the test goroutine that spawned the worker.
//
// A field on Overview rather than a package-level var, for the reason
// AGENTS.md gives: it is per-instance configuration. internal/ui builds a
// real Overview under that same test driver and must keep the app's own
// behaviour, which it does because New installs fyneQueue.
type uiQueue interface {
	// Do arranges for f to run on the UI goroutine. It may return before
	// f has run, and must not run f on the calling goroutine.
	Do(f func())

	// Drain runs whatever Do deferred, on the calling goroutine, and
	// reports whether it ran anything. Always false for a queue backed by
	// a real UI goroutine: that goroutine drains itself.
	Drain() bool
}

// fyneQueue is the app's uiQueue - hand the callback to Fyne and let the
// driver marshal it onto the UI goroutine. Nothing here to drain.
type fyneQueue struct{}

func (fyneQueue) Do(f func()) { fyne.Do(f) }

func (fyneQueue) Drain() bool { return false }
