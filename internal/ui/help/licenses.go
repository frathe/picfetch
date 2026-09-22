package help

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

// ShowLicenses renders the embedded release document without reading files or
// fetching license texts. Repeated requests raise the same window; Escape closes.
func (h *Help) ShowLicenses() {
	if h.stopped {
		return
	}
	h.licensesWin.Show(h.app, lang.L("Licenses"), fyne.NewSize(760, 720), func() fyne.CanvasObject {
		text := widget.NewRichTextFromMarkdown(h.licenses)
		for i, segment := range text.Segments {
			if block, ok := segment.(*widget.CodeBlockSegment); ok {
				// Fyne's code-block panels contain horizontal scrollers that
				// swallow vertical wheel events. These are license prose, not
				// code: keep the verbatim monospace text, but wrap and scroll
				// it with the document instead of using a nested viewport.
				text.Segments[i] = &widget.TextSegment{Text: block.Text, Style: widget.RichTextStyleCodeBlock}
			}
		}
		text.Wrapping = fyne.TextWrapWord
		return container.NewVScroll(text)
	}, nil)
}
