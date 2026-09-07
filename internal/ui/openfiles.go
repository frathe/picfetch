package ui

import (
	"errors"
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/filepicker"
)

// openFileDialog admits an open request on UI, then runs its native panel
// on a worker. Capture the native function here as part of this request.
func (v *viewer) openFileDialog() {
	if v.openChooserClosed || v.refuseOpenDuringComparison() {
		return
	}
	token := v.openChooserLifecycle.begin()
	choose := filepicker.Choose
	done := v.chooser.Begin()
	v.openChooserWorkers.Go(func() {
		defer done()
		v.runFileChooser(token, choose)
	})
}

// runFileChooser reads no UI mode state. Its tracked lifetime ends after
// native work and queue submission; the queue owns result application.
func (v *viewer) runFileChooser(token requestToken, choose func() ([]fyne.URI, error)) {
	if !token.current() {
		return
	}
	uris, err := choose()
	if !token.current() || (err == nil && len(uris) == 0) {
		return
	}
	uris = slices.Clone(uris)
	v.chooserUI.Do(func() {
		if !token.current() || v.comparisonActive() {
			return
		}
		if err != nil {
			v.showChooserError(err)
			return
		}
		v.handleDrop(uris)
	})
}

// closeOpenChooser prevents new admission and invalidates held/queued results.
// The native API itself is blocking: an open panel's worker remains tracked
// until it returns, so shutdown must not wait for that external interaction.
func (v *viewer) closeOpenChooser() {
	v.openChooserClosed = true
	v.openChooserLifecycle.invalidate()
}

// reportChooserError is used by other native panel workers (export). Open
// delivery already runs on UI and calls showChooserError without another hop.
func (v *viewer) reportChooserError(err error) {
	fyne.Do(func() { v.showChooserError(err) })
}

func (v *viewer) showChooserError(err error) {
	detail := chooserErrorDetail(err)
	fyne.LogError("file chooser failed", errors.New(detail))
	v.ShowToast(fmt.Sprintf(lang.L("could not open the file browser: %v"), detail))
}

// chooserErrorDetail pulls the most useful message out of err: an
// *exec.ExitError's own Error() is just "exit status N", but Output()
// populates its Stderr field with whatever the failed command printed,
// which is almost always the actually useful part (an AppleScript error
// message, a missing-binary complaint, and so on). Shared with
// reportClipboardError in clipboard.go and reportRevealError in reveal.go,
// the same kind of chooser-adjacent shell-out failure.
func chooserErrorDetail(err error) string {
	if exitErr, ok := errors.AsType[*exec.ExitError](err); ok {
		if msg := strings.TrimSpace(string(exitErr.Stderr)); msg != "" {
			return msg
		}
	}
	return err.Error()
}

// chooserUIQueue owns result delivery for open requests. Native workers finish
// before test code drains the queue; production delivery uses Fyne's UI loop.
type chooserUIQueue interface {
	Do(func())
	Drain() bool
}
type fyneChooserQueue struct{}

func (fyneChooserQueue) Do(f func()) { fyne.Do(f) }
func (fyneChooserQueue) Drain() bool { return false }
