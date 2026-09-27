package help

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"
)

// Unlike ShowManual (see manual_test.go, which only ever checks the
// embedded markdown string, and the F1/showManual comment at the end of
// internal/ui/e2e_test.go), the About window's RichText content is a
// single plain heading - not manual.md's mix of styles - so it doesn't hit
// the test theme's limited font-combination coverage, and can be exercised
// directly.
func TestShowAbout_OpensAndRaisesSameWindow(t *testing.T) {
	h := New(test.NewApp(), "PicFetch", nil)

	h.ShowAbout()

	win := h.aboutWin.Window()
	if win == nil {
		t.Fatal("ShowAbout did not open a window")
	}

	h.ShowAbout()

	if h.aboutWin.Window() != win {
		t.Error("a second ShowAbout call should raise the existing window, not open a new one")
	}

	original := currentManual
	t.Cleanup(func() { currentManual = original })
	currentManual = func() string { return "manual" }
	h.SetAdmission(func() bool { return false })
	var manualLink *widget.Hyperlink
	var visit func(fyne.CanvasObject)
	visit = func(object fyne.CanvasObject) {
		if link, ok := object.(*widget.Hyperlink); ok && link.Text == lang.L("Open the manual") {
			manualLink = link
		}
		if group, ok := object.(*fyne.Container); ok {
			for _, child := range group.Objects {
				visit(child)
			}
		}
	}
	visit(win.Content())
	if manualLink == nil {
		t.Fatal("About has no manual link")
	}
	test.TapAt(manualLink, fyne.NewPos(8, manualLink.MinSize().Height/2))
	if !h.ManualOpen() {
		t.Fatal("About's manual link was blocked by main-window admission")
	}
	h.manualWin.Window().Close()

	win.Close()

	if h.aboutWin.Open() {
		t.Error("closing the About window should leave the singleton closed")
	}
}
