package spiral

import "fyne.io/fyne/v2"

// UIQueue delivers frame and preview work on the UI goroutine. Tests install
// a drainable queue before Show, because Fyne's test driver runs Do inline.
type UIQueue interface {
	Do(func())
	Drain() bool
}

// SetUIQueue configures dispatch before the first session. Nil restores Fyne.
func (s *Spiral) SetUIQueue(q UIQueue) {
	if q == nil {
		q = fyneQueue{}
	}
	s.ui = q
}

type fyneQueue struct{}

func (fyneQueue) Do(f func()) { fyne.Do(f) }
func (fyneQueue) Drain() bool { return false }
