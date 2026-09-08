package exifwin

import "fyne.io/fyne/v2"

// UIQueue marshals completed EXIF work onto UI. Tests drain callbacks instead
// of allowing Fyne's test driver to paint inline on a fetching worker.
type UIQueue interface {
	Do(func())
	Drain() bool
}

// SetUIQueue configures dispatch before starting work. Nil restores Fyne.
func (w *Window) SetUIQueue(queue UIQueue) {
	if queue == nil {
		w.ui = fyneQueue{}
	} else {
		w.ui = queue
	}
}

type fyneQueue struct{}

func (fyneQueue) Do(fn func()) { fyne.Do(fn) }
func (fyneQueue) Drain() bool  { return false }

// Settle includes metadata, warm passes and tile workers, then drains results.
// A metadata result can start a warm pass; a map can request more tiles.
func (w *Window) Settle() {
	for {
		w.stripWork.workers.Wait()
		w.metadata.workers.Wait()
		w.warmWorkers.Wait()
		w.tiles.Wait()
		if !w.ui.Drain() {
			return
		}
	}
}

// Stop ends admission and cancels reads/map work without waiting on UI or I/O.
// Settle observes completion after underlying operations return.
func (w *Window) Stop() {
	w.stopped = true
	w.cancelStrip()
	w.cancelMetadata()
	w.warmGen++
	w.tiles.Stop()
}

func (w *Window) cancelTiles() {
	w.warmGen++
	w.warming = false
	w.tiles.Cancel()
}
