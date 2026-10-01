package screenshots

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/test"
)

func TestWindowConstraint(t *testing.T) {
	application := test.NewApp()
	defer application.Quit()
	constrained := App(application, 1280, 800)
	for _, title := range []string{"main", "secondary"} {
		w := constrained.NewWindow(title)
		content := canvas.NewRectangle(nil)
		content.SetMinSize(fyne.NewSize(4000, 3000))
		w.SetContent(content)
		w.Resize(fyne.NewSize(6000, 5000))
		w.SetFullScreen(true)
		w.SetFixedSize(false)
		w.Show()
		if got := w.Canvas().Size(); got != fyne.NewSize(1280, 800) {
			t.Fatalf("%s size=%v", title, got)
		}
		if !w.FixedSize() || w.FullScreen() {
			t.Fatalf("%s fixed=%v fullscreen=%v", title, w.FixedSize(), w.FullScreen())
		}
		if w.Content() != content {
			t.Fatal("content identity lost")
		}
		w.SetContent(canvas.NewRectangle(nil))
		w.Resize(fyne.NewSize(200, 200))
		w.Show()
		if got := w.Canvas().Size(); got != fyne.NewSize(1280, 800) {
			t.Fatalf("reopen size=%v", got)
		}
		w.Close()
	}
}
