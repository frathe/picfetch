package deletion

import "fyne.io/fyne/v2"

// UIQueue marshals completed Trash operations onto UI. Tests install a
// drainable queue because Fyne's test driver otherwise runs callbacks on workers.
type UIQueue interface {
	Do(func())
	Drain() bool
}

// SetUIQueue configures completion dispatch before starting any operation.
// A nil queue restores Fyne dispatch.
func (c *Confirmer) SetUIQueue(queue UIQueue) {
	if queue == nil {
		c.ui = fyneQueue{}
	} else {
		c.ui = queue
	}
}

type fyneQueue struct{}

func (fyneQueue) Do(f func()) { fyne.Do(f) }
func (fyneQueue) Drain() bool { return false }
