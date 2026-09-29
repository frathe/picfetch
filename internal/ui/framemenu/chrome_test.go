package framemenu

import (
	"os"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
)

func TestMain(m *testing.M) {
	test.NewApp()
	os.Exit(m.Run())
}

type manualClock struct{ now time.Time }

func attachClock(c *Chrome, start time.Time) *manualClock {
	clk := &manualClock{now: start}
	c.manual = true
	c.nowFn = func() time.Time { return clk.now }
	c.afterFn = func(time.Duration, func()) func() { return func() {} }
	return clk
}

func newTestChrome(t *testing.T) (*Chrome, fyne.Window) {
	t.Helper()
	w := test.NewWindow(nil)
	t.Cleanup(w.Close)
	w.SetPadded(false)
	c := New(w.Canvas())
	t.Cleanup(c.Deactivate)
	w.SetContent(c.Layer())
	w.Resize(fyne.NewSize(800, 600))
	return c, w
}

func menuWithFile() *fyne.MainMenu {
	return fyne.NewMainMenu(fyne.NewMenu("File", fyne.NewMenuItem("Open", nil)))
}

func TestChrome_HiddenBarSitsAboveTheTopEdge(t *testing.T) {
	c, _ := newTestChrome(t)
	start := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	clk := attachClock(c, start)
	c.Activate(menuWithFile())
	c.Resize(c.MinSize())

	if h := c.MinSize().Height; h != 24 {
		t.Fatalf("hot zone height = %v, want 24", h)
	}
	if y := c.row.Position().Y; y >= 0 {
		t.Fatalf("hidden bar Y = %v, want it above the top edge", y)
	}

	c.MouseIn(&desktop.MouseEvent{})
	clk.now = start.Add(700 * time.Millisecond)
	c.Resize(c.MinSize())
	if got := c.Slide(); got != 1 {
		t.Fatalf("slide after a 700ms dwell = %v, want 1", got)
	}
	if y := c.row.Position().Y; y != 0 {
		t.Fatalf("shown bar Y = %v, want 0", y)
	}
}

func TestChrome_RightSideOfTopEdgeCounts(t *testing.T) {
	c, w := newTestChrome(t)
	start := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	clk := attachClock(c, start)
	c.Activate(menuWithFile())
	w.Resize(fyne.NewSize(800, 600))

	test.MoveMouse(w.Canvas(), fyne.NewPos(790, 8))
	clk.now = start.Add(700 * time.Millisecond)
	if got := c.Slide(); got != 1 {
		t.Fatalf("slide after dwelling at the right edge = %v, want 1", got)
	}
}

func TestChrome_OpenMenuStaysShownUntilDismissed(t *testing.T) {
	c, _ := newTestChrome(t)
	start := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	clk := attachClock(c, start)
	c.Activate(menuWithFile())
	c.MouseIn(&desktop.MouseEvent{})
	clk.now = start.Add(700 * time.Millisecond)
	c.Resize(c.MinSize())
	if got := c.Slide(); got != 1 {
		t.Fatalf("slide before opening = %v, want 1", got)
	}

	c.row.Objects[0].(*barItem).Tapped(nil)
	c.MouseOut()
	clk.now = clk.now.Add(5 * time.Second)
	if got := c.Slide(); got != 1 {
		t.Fatalf("slide while the menu is open = %v, want 1", got)
	}
	if c.canvas.Overlays().Top() == nil {
		t.Fatal("opening a menu item should show its menu")
	}

	closed := clk.now
	c.popup.Dismiss()
	if got := c.Slide(); got != 1 {
		t.Fatalf("slide just after the menu closed = %v, want 1", got)
	}
	clk.now = closed.Add(499 * time.Millisecond)
	if got := c.Slide(); got != 1 {
		t.Fatalf("slide 499ms after close = %v, want 1", got)
	}
	clk.now = closed.Add(700 * time.Millisecond)
	if got := c.Slide(); got != 0 {
		t.Fatalf("slide 700ms after close = %v, want 0", got)
	}
}
