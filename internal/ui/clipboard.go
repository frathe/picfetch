package ui

import (
	"bytes"
	"errors"
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/clipboard"
)

// copyPathToClipboard puts the current file's absolute path on the system
// text clipboard. No shell-out needed here, unlike copyImageToClipboard
// below - fyne.Clipboard already handles text on every platform.
func (v *viewer) copyPathToClipboard() {
	if v.comparisonActive() || v.explorerMapActive() {
		return
	}
	if len(v.state.files) == 0 {
		return
	}
	if v.clipboardWork.closed || v.clipboardBusy() {
		return
	}
	v.app.Clipboard().SetContent(v.state.files[v.state.index].Path())
}

// copyImageToClipboard puts the currently displayed frame onto the system
// clipboard as real image data, via internal/clipboard's per-OS shell-out -
// the same kind openfiles.go already established for the file/folder
// dialog. The displayed image is immutable after publication; capture that
// reference on UI, then encode and dispatch on the operation's worker.
func (v *viewer) copyImageToClipboard() {
	if v.comparisonActive() || v.explorerMapActive() {
		return
	}
	img := v.img.Image
	if img == nil {
		return
	}

	// Admission owns the completion signal until encoding/dispatch finishes
	// and the current result reaches UI, or cancelled work returns.
	token, done, ok := v.beginClipboardCopy(true)
	if !ok {
		return
	}
	encode := v.clipboardWork.encode

	v.clipboardWork.workers.Go(func() {
		if !token.current() {
			done()
			return
		}
		var buf bytes.Buffer
		err := encode(clipboardContextWriter{ctx: token.context(), out: &buf}, img)
		if err == nil && token.current() {
			err = clipboard.CopyImage(buf.Bytes())
		}
		v.completeClipboardCopy(token, done, func() {
			if err != nil {
				v.reportClipboardError(err)
			}
		})
	})
}

// reportClipboardError logs a failed clipboard copy and shows its toast on UI.
func (v *viewer) reportClipboardError(err error) {
	detail := chooserErrorDetail(err)
	fyne.LogError("clipboard image copy failed", errors.New(detail))

	v.ShowToast(fmt.Sprintf(lang.L("could not copy the image: %v"), detail))
}
