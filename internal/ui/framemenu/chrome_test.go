package framemenu

import (
	"math"
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
	c.afterFn = func(_ time.Duration, _ func()) func() bool { return func() bool { return true } }
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

func TestChrome_ReturnToBarWhileMenuIsOpenStaysShown(t *testing.T) {
	c, _ := newTestChrome(t)
	start := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	clk := attachClock(c, start)
	c.Activate(menuWithFile())
	c.MouseIn(&desktop.MouseEvent{})
	clk.now = start.Add(700 * time.Millisecond)
	c.Resize(c.MinSize())
	c.row.Objects[0].(*barItem).Tapped(nil)
	c.MouseOut()
	c.MouseIn(&desktop.MouseEvent{})

	closed := clk.now
	c.popup.Dismiss()
	clk.now = closed.Add(2 * time.Second)
	if got := c.Slide(); got != 1 {
		t.Fatalf("slide after returning to the bar = %v, want 1", got)
	}
}

func TestChrome_DeactivatePreventsTheDwellTimer(t *testing.T) {
	c, _ := newTestChrome(t)
	stops := 0
	c.afterFn = func(_ time.Duration, _ func()) func() bool {
		return func() bool {
			stops++
			return true
		}
	}
	c.Activate(menuWithFile())
	c.MouseIn(&desktop.MouseEvent{})
	c.Deactivate()
	waitChrome(t, c)
	if stops != 1 {
		t.Fatalf("timer stops = %d, want 1", stops)
	}
	if c.Layer().Visible() {
		t.Fatal("deactivating should hide the bar")
	}
}

func TestChrome_StartedTimerDoesNotTouchTheBarAfterDeactivate(t *testing.T) {
	c, _ := newTestChrome(t)
	started := make(chan struct{})
	release := make(chan struct{})
	c.afterFn = func(_ time.Duration, fn func()) func() bool {
		go func() {
			close(started)
			<-release
			fn()
		}()
		return func() bool { return false }
	}
	enqueued := 0
	c.beforeEnqueue = func() { enqueued++ }
	c.Activate(menuWithFile())
	c.MouseIn(&desktop.MouseEvent{})
	<-started
	c.Deactivate()
	close(release)
	waitChrome(t, c)
	if enqueued != 0 {
		t.Fatalf("repaints submitted after deactivation = %d, want 0", enqueued)
	}
	if c.Layer().Visible() {
		t.Fatal("a timer that already started should still leave the bar hidden")
	}
	if got := c.Slide(); got != 0 {
		t.Fatalf("slide after a late timer = %v, want 0", got)
	}
}

func TestChrome_EnqueueHoldsDeactivationUntilItSubmits(t *testing.T) {
	c, _ := newTestChrome(t)
	var fire func()
	first := true
	c.afterFn = func(_ time.Duration, fn func()) func() bool {
		if first {
			first = false
			fire = fn
			return func() bool { return false }
		}
		return func() bool { return true }
	}
	c.beforeEnqueue = func() {
		if c.admit.TryLock() {
			c.admit.Unlock()
			t.Error("timer callback submitted a repaint without holding deactivation")
		}
	}
	c.Activate(menuWithFile())
	c.MouseIn(&desktop.MouseEvent{})
	if fire == nil {
		t.Fatal("dwelling should arm a timer")
	}
	fire()
}

func TestChrome_LeaveDuringSlideKeepsRepainting(t *testing.T) {
	c, _ := newTestChrome(t)
	start := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	clk := &manualClock{now: start}
	c.nowFn = func() time.Time { return clk.now }
	var delay time.Duration
	c.afterFn = func(d time.Duration, _ func()) func() bool {
		delay = d
		return func() bool { return true }
	}
	c.Activate(menuWithFile())
	c.MouseIn(&desktop.MouseEvent{})
	clk.now = start.Add(550 * time.Millisecond)
	c.kick()
	c.MouseOut()
	if delay != repaintEvery {
		t.Fatalf("delay after leaving mid-slide = %v, want a repaint every %v", delay, repaintEvery)
	}
	if got := c.Slide(); math.Abs(float64(got-0.125)) > 0.001 {
		t.Fatalf("slide just after leaving mid-slide = %v, want 0.125", got)
	}
}

func waitChrome(t *testing.T, c *Chrome) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		c.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for the menu timer")
	}
}
