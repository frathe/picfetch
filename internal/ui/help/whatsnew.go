package help

import (
	"context"
	_ "embed"
	"fmt"
	"net/url"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"
)

const (
	whatsNewW   = 640.0
	whatsNewH   = 480.0
	releasesURL = "https://github.com/frathe/picfetch/releases"
)

// releaseNotesMD is copied from .github/release-notes.md by make release, in
// the same commit as the version bump. The bundle parity test guards drift.
//
//go:embed release-notes.md
var releaseNotesMD string

// ShowReleaseNotes opens this build's bundled notes, including on first installs
// and Store builds where no automatic-update What's New cache is available.
func (h *Help) ShowReleaseNotes() {
	h.ShowWhatsNew(h.app.Metadata().Version, releaseNotesMD)
}

// ShowWhatsNew opens a window with the given release notes body (GitHub
// markdown). A second call while it's still open raises the existing window
// instead of stacking a duplicate (see widgets.Singleton). Empty body still
// opens the window with a translated fallback line. Escape closes; the
// window is not KeepOnTop. Help -> Release Notes reuses this window.
func (h *Help) ShowWhatsNew(version, body string) {
	if h.stopped {
		return
	}
	title := lang.L("Release Notes")
	if version != "" {
		title = fmt.Sprintf(lang.L("What's New in %s"), strings.TrimPrefix(version, "v"))
	}
	if strings.TrimSpace(body) == "" {
		body = lang.L("This release has no notes.")
	}
	h.whatsNewWin.Show(h.app, title, fyne.NewSize(whatsNewW, whatsNewH), func() fyne.CanvasObject {
		ctx, cancel := context.WithCancel(context.Background())
		h.notes = &releaseNotesSession{ctx: ctx, cancel: cancel}
		text := widget.NewRichTextFromMarkdown(body)
		var images []releaseImageRequest
		text.Segments = prepareReleaseImages(text.Segments, &images)
		text.Wrapping = fyne.TextWrapWord
		target, _ := url.Parse(releasesURL)
		history := widget.NewHyperlink(lang.L("Browse all releases"), target)
		history.OnTapped = func() {
			if err := h.app.OpenURL(target); err != nil {
				fyne.LogError("open release history", err)
			}
		}
		h.loadReleaseImages(h.notes, text, images)
		return container.NewBorder(nil, container.NewPadded(history), nil, nil, container.NewScroll(text))
	}, h.cancelReleaseNotes)
}

// WhatsNewOpen reports whether the What's New window is currently showing.
func (h *Help) WhatsNewOpen() bool { return h.whatsNewWin.Open() }
