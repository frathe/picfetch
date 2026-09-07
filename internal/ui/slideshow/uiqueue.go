package slideshow

import "fyne.io/fyne/v2"

// UIQueue marshals timed advances onto UI. Tests use a drainable queue to
// preserve delayed application with Fyne's otherwise inline test driver.
type UIQueue interface {
	Do(func())
	Drain() bool
}

// SetUIQueue configures dispatch before entering picture-frame mode.
// Nil restores production Fyne dispatch.
func (c *Controller) SetUIQueue(queue UIQueue) {
	if queue == nil {
		c.ui = fyneQueue{}
	} else {
		c.ui = queue
	}
}

type fyneQueue struct{}

func (fyneQueue) Do(f func()) { fyne.Do(f) }
func (fyneQueue) Drain() bool { return false }
