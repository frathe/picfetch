// Actions > "Reveal in file manager" (Cmd/Ctrl+R), and the info overlay's
// link of the same name: the viewer-side glue over internal/filemanager,
// which owns the per-OS dispatch.

package ui

import (
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/filemanager"
)

// revealCurrentFile opens the OS file manager with the current image
// selected. Always runs on its own goroutine, mirroring
// copyImageToClipboard: every backing command starts an external process,
// and two of the three wait for it.
//
// The subject is deliberately the current file rather than batch.go's
// grid-selection routing: delete and copy act on a set because a set has a
// meaning there, while revealing twelve files across nine folders does not.
func (v *viewer) revealCurrentFile() {
	if v.comparisonActive() {
		return
	}
	if len(v.state.files) == 0 {
		return
	}

	// Read the path here, not in the goroutine: state.files/state.index
	// carry no synchronization of their own and belong to the UI goroutine.
	path := v.state.files[v.state.index].Path()

	// reveal is finished once this goroutine has fully run, error
	// reporting included, so a test can wait for the whole operation
	// instead of polling widget state the goroutine may still be writing.
	done := v.reveal.Begin()

	go func() {
		defer done()

		if err := filemanager.Reveal(path); err != nil {
			v.reportRevealError(err)
		}
	}()
}

// reportRevealError logs the failure and shows it as a toast on every
// platform, like reportClipboardError and unlike reportChooserError: there
// is no cancel to mistake a failure for here, because nothing about this
// command asks the user a question.
func (v *viewer) reportRevealError(err error) {
	detail := chooserErrorDetail(err)
	fyne.LogError("file manager reveal failed", errors.New(detail))

	fyne.Do(func() {
		v.ShowToast(fmt.Sprintf(lang.L("could not show the file in your file manager: %v"), detail))
	})
}
