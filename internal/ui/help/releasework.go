package help

import (
	"context"
	"net/http"

	"fyne.io/fyne/v2"
)

// UIQueue delivers release artwork on UI. Tests install a drainable queue
// because the Fyne test driver runs fyne.Do inline on the worker.
type UIQueue interface {
	Do(func())
	Drain() bool
}

type fyneQueue struct{}

func (fyneQueue) Do(f func()) { fyne.Do(f) }
func (fyneQueue) Drain() bool { return false }

// SetUIQueue configures image delivery before opening release notes.
func (h *Help) SetUIQueue(queue UIQueue) {
	if queue == nil {
		queue = fyneQueue{}
	}
	h.imageUI = queue
}

type releaseNotesSession struct {
	ctx    context.Context
	cancel context.CancelFunc
}

// SetImageClient configures the HTTP client before opening release notes.
// Tests use a local transport; production keeps the default timed client.
func (h *Help) SetImageClient(client *http.Client) {
	if client != nil {
		h.imageClient = client
	}
}

func (h *Help) cancelReleaseNotes() {
	if h.notes != nil {
		h.notes.cancel()
		h.notes = nil
	}
}

// Stop permanently ends image admission and cancels downloads without waiting.
func (h *Help) Stop() {
	h.stopped = true
	h.cancelReleaseNotes()
}

// Wait joins current and retired image workers after Stop, off UI.
func (h *Help) Wait() { h.imageWorkers.Wait() }

// Settle joins finite image work and drains delivery on the test goroutine.
func (h *Help) Settle() {
	for {
		h.Wait()
		if !h.imageUI.Drain() {
			return
		}
	}
}
