package help

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

// ShowPrivacyPolicy displays the bundled policy offline. Repeated requests raise
// the existing window; Escape closes it.
func (h *Help) ShowPrivacyPolicy() {
	if !h.admitted() {
		return
	}
	if h.stopped {
		return
	}
	h.privacyWin.Show(h.app, lang.L("Privacy policy"), fyne.NewSize(640, 480), func() fyne.CanvasObject {
		text := widget.NewRichTextFromMarkdown(h.privacy)
		text.Wrapping = fyne.TextWrapWord
		return container.NewVScroll(text)
	}, nil)
}
