package ui

import (
	"errors"
	"image/color"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/frathe/picfetch/internal/uitest"
)

// Per-platform shell-out behavior (open -R, explorer /select, the
// FileManager1 D-Bus call and its xdg-open fallback) is covered by
// internal/filemanager's own tests; the tests below exercise the viewer's
// integration with it - which file it picks, that it dispatches on its own
// goroutine, and how it reports a failure - which can't move since they
// depend on *viewer.

func TestRevealCurrentFile_HandsThePathToTheFileManager(t *testing.T) {
	v, _, _ := newTestUI(t)

	jpegURI := uitest.TempJPEGURI(t, "picked.jpg", 4, 4, color.RGBA{R: 100, A: 255})
	v.state.files = []fyne.URI{jpegURI}
	v.state.index = 0

	var got string
	uitest.StubReveal(t, func(path string) error {
		got = path
		return nil
	})

	v.revealCurrentFile()
	waitForReveal(t, v)

	if got != jpegURI.Path() {
		t.Errorf("revealed path = %q, want %q", got, jpegURI.Path())
	}
}

// TestRevealCurrentFile_RevealsTheDisplayedFileNotTheFirst: the command acts
// on what is on screen, which after navigating is not state.files[0].
func TestRevealCurrentFile_RevealsTheDisplayedFileNotTheFirst(t *testing.T) {
	v, _, _ := newTestUI(t)

	first := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.RGBA{R: 100, A: 255})
	second := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.RGBA{G: 100, A: 255})
	v.state.files = []fyne.URI{first, second}
	v.state.index = 1

	var got string
	uitest.StubReveal(t, func(path string) error {
		got = path
		return nil
	})

	v.revealCurrentFile()
	waitForReveal(t, v)

	if got != second.Path() {
		t.Errorf("revealed path = %q, want the displayed file %q", got, second.Path())
	}
}

func TestRevealCurrentFile_NoFilesIsNoop(t *testing.T) {
	v, _, _ := newTestUI(t)

	uitest.StubReveal(t, func(string) error {
		t.Error("filemanager.Reveal should not run with no files loaded")
		return nil
	})

	v.revealCurrentFile()

	if v.reveal.Begun() {
		t.Error("revealCurrentFile started a worker with no files loaded")
	}
}

func TestRevealCurrentFile_DispatchFailureShowsToast(t *testing.T) {
	v, _, _ := newTestUI(t)

	v.state.files = []fyne.URI{uitest.TempJPEGURI(t, "picked.jpg", 4, 4, color.RGBA{R: 100, A: 255})}
	uitest.StubReveal(t, func(string) error { return errors.New("boom") })

	v.revealCurrentFile()

	// The failure is reported from the background goroutine via fyne.Do;
	// waiting on v.reveal - finished after that goroutine has fully run,
	// error toast included - is what makes reading the toast widgets
	// afterwards race-free, exactly as the clipboard tests document.
	waitForReveal(t, v)

	if !v.toast.card.Visible() {
		t.Error("expected a toast for a failed reveal")
	}
	if !strings.Contains(v.toast.text.Text, "boom") {
		t.Errorf("toast text = %q, want it to mention the underlying error", v.toast.text.Text)
	}
	settleToast(t, v)
}

// TestWireRevealShortcut_RevealsCurrentFile drives the production binding
// through a bare *fyne.ShortcutHandler, the detour shortcutAdder exists for -
// see its comment for why a real Cmd/Ctrl+R can't be simulated through the
// test driver's canvas.
func TestWireRevealShortcut_RevealsCurrentFile(t *testing.T) {
	v, _, _ := newTestUI(t)

	jpegURI := uitest.TempJPEGURI(t, "picked.jpg", 4, 4, color.RGBA{R: 100, A: 255})
	v.state.files = []fyne.URI{jpegURI}

	var got string
	uitest.StubReveal(t, func(path string) error {
		got = path
		return nil
	})

	handler := &fyne.ShortcutHandler{}
	wireGlobalShortcuts(handler, v)
	handler.TypedShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyR,
		Modifier: fyne.KeyModifierShortcutDefault,
	})
	waitForReveal(t, v)

	if got != jpegURI.Path() {
		t.Errorf("revealed path = %q, want %q", got, jpegURI.Path())
	}
}

// TestRevealLink_RevealsCurrentFile checks build.go's info-card wiring
// reaches the command, mirroring TestExifLink_OpensExifWindow's way of
// driving OnTapped directly rather than simulating a click.
func TestRevealLink_RevealsCurrentFile(t *testing.T) {
	v, _, _ := newTestUI(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White)
	dropAndWait(t, v, a)

	var got string
	uitest.StubReveal(t, func(path string) error {
		got = path
		return nil
	})

	v.toggleInfoOverlay()
	if !v.info.RevealLink().Visible() {
		t.Fatal("the reveal link should be showing with the info card")
	}

	v.info.RevealLink().OnTapped()
	waitForReveal(t, v)

	if got != a.Path() {
		t.Errorf("revealed path = %q, want %q", got, a.Path())
	}
}

// TestRevealActionsFile_MenuItemRevealsCurrentFile drives the Actions menu
// item's own callback, which reaches the command through
// yieldingMenuCallbacks rather than through the canvas shortcut above.
func TestRevealActionsFile_MenuItemRevealsCurrentFile(t *testing.T) {
	v, _, _ := newTestUI(t)

	jpegURI := uitest.TempJPEGURI(t, "picked.jpg", 4, 4, color.RGBA{R: 100, A: 255})
	v.state.files = []fyne.URI{jpegURI}

	var got string
	uitest.StubReveal(t, func(path string) error {
		got = path
		return nil
	})

	v.menus.Actions().Reveal().Action()
	waitForReveal(t, v)

	if got != jpegURI.Path() {
		t.Errorf("revealed path = %q, want %q", got, jpegURI.Path())
	}
}
