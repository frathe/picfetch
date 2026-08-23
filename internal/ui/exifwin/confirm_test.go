package exifwin

import (
	"testing"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/ui/widgets"
)

// typeKey sends a key event to whatever the window's canvas currently has
// focused, the same package-local helper favorites/manage_test.go defines
// for its own tests.
func typeKey(t *testing.T, win fyne.Window, name fyne.KeyName) {
	t.Helper()

	focused := win.Canvas().Focused()
	if focused == nil {
		t.Fatalf("no focused object to send %s to", name)
	}
	focused.TypedKey(&fyne.KeyEvent{Name: name})
}

// TestShowConfirmGivesTheKeyboardToItsPanelStartingOnCancel pins the same
// bug fix favorites' own showConfirm carries: the panel, not nothing, must
// hold the keyboard once the confirmation is up, and Cancel is the default
// selection.
func TestShowConfirmGivesTheKeyboardToItsPanelStartingOnCancel(t *testing.T) {
	app, host := testApp(t)
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.showConfirm(confirmation{title: "Title", message: "Message", action: "Confirm"})

	panel, ok := w.Window().Canvas().Focused().(*widgets.ChoicePanel)
	if !ok {
		t.Fatalf("focused = %v, want the confirmation's choice panel", w.Window().Canvas().Focused())
	}
	if got := panel.Selected(); got != cancelChoice {
		t.Errorf("selected = %d, want Cancel (%d): a prompt never opens with the action already under Return", got, cancelChoice)
	}
	if !panel.Ring(cancelChoice).Visible() || panel.Ring(confirmChoice).Visible() {
		t.Error("the ring is not drawn on Cancel")
	}
}

// TestShowConfirmEscapeRunsOnCancelAfterTheDialogCloses is the other way to
// say Cancel, and has to leave onCancel looking at already-closed state.
func TestShowConfirmEscapeRunsOnCancelAfterTheDialogCloses(t *testing.T) {
	app, host := testApp(t)
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	var ran int
	w.showConfirm(confirmation{
		title:   "Title",
		message: "Message",
		action:  "Confirm",
		onCancel: func() {
			ran++
			if n := len(w.Window().Canvas().Overlays().List()); n != 0 {
				t.Errorf("overlay count = %d while onCancel ran, want the confirmation already gone", n)
			}
		},
	})

	typeKey(t, w.Window(), fyne.KeyEscape)

	if ran != 1 {
		t.Errorf("onCancel ran %d times, want 1", ran)
	}
}

// TestShowConfirmEscapeDoesNotCloseTheEXIFWindow is this package's own trap:
// widgets.Singleton registers Canvas().SetOnTypedKey Escape -> win.Close()
// for the EXIF window itself. Without focusing the confirmation's panel,
// Escape would close the window behind the prompt rather than cancel it.
func TestShowConfirmEscapeDoesNotCloseTheEXIFWindow(t *testing.T) {
	app, host := testApp(t)
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.showConfirm(confirmation{title: "Title", message: "Message", action: "Confirm"})

	typeKey(t, w.Window(), fyne.KeyEscape)

	if !w.Open() {
		t.Fatal("Escape on the confirmation closed the EXIF window, want it to stay open")
	}
	if n := len(w.Window().Canvas().Overlays().List()); n != 0 {
		t.Errorf("overlay count = %d after Escape, want the confirmation gone and nothing else up", n)
	}
}

func TestShowConfirmEscapeLeavesCanvasUnfocused(t *testing.T) {
	app, host := testApp(t)
	w := New(app, host)
	w.Show()
	t.Cleanup(func() { w.Window().Close() })

	w.showConfirm(confirmation{title: "Title", message: "Message", action: "Confirm"})
	typeKey(t, w.Window(), fyne.KeyEscape)

	if got := w.Window().Canvas().Focused(); got != nil {
		t.Errorf("Focused() after cancelling confirm = %T, want nil so Left/Right reach OnTypedKey", got)
	}
}
