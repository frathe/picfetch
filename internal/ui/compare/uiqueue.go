package compare

import "fyne.io/fyne/v2"

// UIQueue is how comparison load and vector workers hand completed widget
// work to the UI goroutine. Production uses fyneQueue; tests install a
// drainable queue because Fyne's test driver runs fyne.Do inline on the worker.
type UIQueue interface {
	Do(func())
	Drain() bool
}

// SetUIQueue replaces the per-instance UI marshaler. Tests use a drainable
// queue so Settle applies completions on the test goroutine. Nil restores the
// production Fyne queue.
func (f *Feature) SetUIQueue(queue UIQueue) {
	if queue == nil {
		f.ui = fyneQueue{}
		return
	}
	f.ui = queue
}

func (f *Feature) queueUI(apply func()) {
	f.uiPending.Add(1)
	f.uiCount.Add(1)
	f.ui.Do(func() {
		defer f.uiPending.Done()
		defer f.uiCount.Add(-1)
		apply()
	})
}

type fyneQueue struct{}

func (fyneQueue) Do(apply func()) { fyne.Do(apply) }

func (fyneQueue) Drain() bool { return false }
