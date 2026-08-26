package help

import (
	"fmt"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

const (
	whatsNewW = 640.0
	whatsNewH = 480.0
)

// ShowWhatsNew opens a window with the given release notes body (GitHub
// markdown). A second call while it's still open raises the existing window
// instead of stacking a duplicate (see widgets.Singleton). Empty body still
// opens the window with a translated fallback line. Escape closes; the
// window is not KeepOnTop and has no Help menu entry.
func (h *Help) ShowWhatsNew(version, body string) {
	title := fmt.Sprintf(lang.L("What's New in %s"), strings.TrimPrefix(version, "v"))
	if strings.TrimSpace(body) == "" {
		body = lang.L("This release has no notes.")
	}
	h.whatsNewWin.Show(h.app, title, fyne.NewSize(whatsNewW, whatsNewH), func() fyne.CanvasObject {
		text := widget.NewRichTextFromMarkdown(body)
		text.Wrapping = fyne.TextWrapWord
		return container.NewScroll(text)
	}, nil)
}

// WhatsNewOpen reports whether the What's New window is currently showing.
func (h *Help) WhatsNewOpen() bool { return h.whatsNewWin.Open() }
