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
// text clipboard; fyne.Clipboard handles text on every platform.
func (v *viewer) copyPathToClipboard() {
	if _, ok := v.admitCommand(commandRequest{command: commandCopyPath}); !ok {
		return
	}
	source, _, _ := v.CurrentFile()
	v.app.Clipboard().SetContent(source.Path())
}

// copyImageToClipboard puts the currently displayed frame onto the system
// clipboard as real image data, via internal/clipboard's platform backend.
// The displayed image is immutable after publication; capture that
// reference on UI, then encode and dispatch on the operation's worker.
func (v *viewer) copyImageToClipboard() {
	if _, ok := v.admitCommand(commandRequest{command: commandCopyImage}); !ok {
		return
	}
	capture, captured := v.display.Capture()
	if !captured {
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
		if !token.Current() {
			v.completeClipboardCopy(token, done, nil)
			return
		}
		var buf bytes.Buffer
		err := encode(clipboardContextWriter{ctx: token.Context(), out: &buf}, capture.Pixels)
		if err == nil && token.Current() {
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
