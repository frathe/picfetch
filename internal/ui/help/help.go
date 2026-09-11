// Package help owns the documentation windows and the Finis companion.
// The manual's secret phrase emits a callback; the viewer owns the Spiral.
package help

import (
	"net/url"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/ui/widgets"
)

const DiscussionsURL = "https://github.com/frathe/picfetch/discussions"

// Help owns the documentation windows (manual, About, What's New). Each is
// a widgets.Singleton, so a second request raises the window that's already
// open instead of stacking up duplicates.
type Help struct {
	app   fyne.App
	title string

	// art is the welcome image the About box shows beside the app name -
	// passed in rather than imported so this package doesn't depend on
	// where the app keeps its assets.
	art []byte

	manualWin   widgets.Singleton
	aboutWin    widgets.Singleton
	whatsNewWin widgets.Singleton
	finisWin    widgets.Singleton
	finis       *finisView
	manual      *manualView

	onManualClosed func()
	onManualOpened func()

	onSpiral func()
}

// New returns the help UI for application, showing title as the app's name
// and art as the About box's illustration.
func New(application fyne.App, title string, art []byte) *Help {
	return &Help{app: application, title: title, art: art}
}

// SetOnSpiral registers the manual secret callback. It is read at invocation,
// so replacing it while the manual is open takes effect immediately.
func (h *Help) SetOnSpiral(f func()) { h.onSpiral = f }

func (h *Help) openSpiral() {
	if h.onSpiral != nil {
		h.onSpiral()
	}
}

// ManualOpen reports whether the end-user manual window is currently showing.
func (h *Help) ManualOpen() bool { return h.manualWin.Open() }

// SetOnManualClosed registers f to run when the manual window closes. The
// field is read at close time, so a Set after ShowManual still fires. nil
// is a no-op.
func (h *Help) SetOnManualClosed(f func()) { h.onManualClosed = f }

// SetOnManualOpened registers f to run after ShowManual raises or builds
// the window. The field is read at show time. nil is a no-op. Needed so
// the app can grey Window -> Help on every door into the manual (Help
// menu, About link, F1), not only the ones that already wrap ShowManual.
func (h *Help) SetOnManualOpened(f func()) { h.onManualOpened = f }

// ShowDiscussions opens the public community page without attaching app data.
func (h *Help) ShowDiscussions() {
	target, _ := url.Parse(DiscussionsURL)
	if err := h.app.OpenURL(target); err != nil {
		fyne.LogError("open GitHub Discussions", err)
	}
}

// Menu is the app's Help menu: the manual, and an About screen below a
// separator (the usual place for it in a Help menu). Returns the *fyne.Menu
// itself rather than a whole *fyne.MainMenu, so internal/ui can combine it
// with its own File menu into one bar - composing menus is the app's job,
// not this package's, the same "internal/ui decides how features compose"
// rule the grid/slideshow full-window-mode guard follows (see
// ARCHITECTURE.md).
func (h *Help) Menu() *fyne.Menu {
	manual := fyne.NewMenuItem(lang.L("Manual"), h.ShowManual)
	// Display-only: F1 itself is handleKeyEvent in internal/ui. This is the
	// same menu-hint pattern File uses for Open/Save/Export.
	manual.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyF1}
	about := fyne.NewMenuItem(lang.L("About"), h.ShowAbout)
	discussions := fyne.NewMenuItem(lang.L("GitHub Discussions"), h.ShowDiscussions)

	return fyne.NewMenu(lang.L("Help"), manual, discussions, fyne.NewMenuItemSeparator(), about)
}
