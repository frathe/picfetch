package framemenu

import (
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
)

// repaintEvery is how often the bar is redrawn during a slide.
const repaintEvery = 16 * time.Millisecond

var (
	_ fyne.Widget       = (*Chrome)(nil)
	_ desktop.Hoverable = (*Chrome)(nil)
	_ fyne.Tappable     = (*barItem)(nil)
)

// Chrome is the sliding menu bar. Layer is what the window stacks: a
// top-pinned strip, hidden until picture-frame mode asks for it. The
// in-window Fyne menu cannot be translated, so this bar is drawn instead
// and the window menu is detached for the session.
type Chrome struct {
	widget.BaseWidget

	canvas fyne.Canvas
	layer  *fyne.Container
	row    *fyne.Container
	menu   *fyne.MainMenu
	popup  *widget.PopUpMenu
	reveal Reveal

	active         atomic.Bool
	leftDuringMenu bool
	suppress       bool
	manual         bool

	nowFn   func() time.Time
	afterFn func(time.Duration, func()) func() bool
	timerMu sync.Mutex
	cancel  func() bool
	pending sync.WaitGroup
}

// New builds a hidden bar for canvas. Pop-up menus use that canvas.
func New(canvas fyne.Canvas) *Chrome {
	c := &Chrome{
		canvas:  canvas,
		nowFn:   time.Now,
		afterFn: realAfter,
	}
	c.row = container.NewHBox()
	c.ExtendBaseWidget(c)
	c.layer = container.NewVBox(c)
	c.layer.Hide()
	return c
}

// Layer is the overlay placed at the top of the window stack.
func (c *Chrome) Layer() fyne.CanvasObject { return c.layer }

// Activate shows the hidden strip for menu and resets any previous slide.
func (c *Chrome) Activate(menu *fyne.MainMenu) {
	c.active.Store(true)
	c.leftDuringMenu = false
	c.reveal.Reset()
	c.setMenu(menu)
	c.layer.Show()
	c.Show()
	c.kick()
}

// Deactivate hides the strip, closes an open menu, and drops pending timers.
// A callback that already started is left for Wait; it will not touch the bar
// once this returns.
func (c *Chrome) Deactivate() {
	c.timerMu.Lock()
	c.active.Store(false)
	c.disarmLocked()
	c.timerMu.Unlock()
	c.suppress = true
	if c.popup != nil {
		c.popup.Dismiss()
		c.popup = nil
	}
	c.suppress = false
	c.leftDuringMenu = false
	c.reveal.Reset()
	c.layer.Hide()
}

// Wait joins a dwell or slide callback that Deactivate could not prevent.
// Call it off the UI thread. The callback delivers through fyne.Do, so waiting
// on the UI thread can deadlock once that delivery is queued.
func (c *Chrome) Wait() {
	c.pending.Wait()
}

// Slide is how far the bar has traveled right now, from 0 to 1.
func (c *Chrome) Slide() float32 { return c.reveal.Shown(c.now()) }

func (c *Chrome) CreateRenderer() fyne.WidgetRenderer {
	bg := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	return &chromeRenderer{c: c, bg: bg, row: c.row}
}

func (c *Chrome) MouseIn(_ *desktop.MouseEvent) { c.hover(true) }

func (c *Chrome) MouseMoved(_ *desktop.MouseEvent) { c.hover(true) }

func (c *Chrome) MouseOut() {
	if !c.active.Load() {
		return
	}
	// The open menu is a canvas overlay, so it takes the pointer off this
	// strip without the user having left the menu. Remember that and keep
	// the bar pinned until the menu closes.
	if c.popup != nil {
		c.leftDuringMenu = true
		return
	}
	c.hover(false)
}

func (c *Chrome) hover(inside bool) {
	if !c.active.Load() {
		return
	}
	c.reveal.Pointer(inside, c.now())
	c.kick()
}

func (c *Chrome) setMenu(menu *fyne.MainMenu) {
	c.menu = menu
	c.row.Objects = nil
	if menu == nil {
		c.row.Refresh()
		return
	}
	for i, item := range menu.Items {
		if item == nil {
			continue
		}
		index := i
		label := item.Label
		c.row.Add(newBarItem(label, func(anchor fyne.CanvasObject) {
			c.open(index, anchor)
		}))
	}
	c.row.Refresh()
}

func (c *Chrome) open(index int, anchor fyne.CanvasObject) {
	if !c.active.Load() || c.menu == nil || index < 0 || index >= len(c.menu.Items) || c.canvas == nil {
		return
	}
	c.suppress = true
	if c.popup != nil {
		c.popup.Dismiss()
		c.popup = nil
	}
	c.suppress = false

	c.leftDuringMenu = false
	c.reveal.SetMenuOpen(true, true, c.now())
	pop := widget.NewPopUpMenu(c.menu.Items[index], c.canvas)
	if pop == nil {
		return
	}
	prev := pop.OnDismiss
	pop.OnDismiss = func() {
		if prev != nil {
			prev()
		}
		c.onMenuDismissed()
	}
	c.popup = pop
	c.kick()
	pop.ShowAtRelativePosition(fyne.NewPos(0, anchor.Size().Height), anchor)
}

func (c *Chrome) onMenuDismissed() {
	if c.suppress || !c.active.Load() {
		return
	}
	inside := !c.leftDuringMenu
	c.leftDuringMenu = false
	c.popup = nil
	c.reveal.SetMenuOpen(false, inside, c.now())
	c.kick()
}

func (c *Chrome) kick() {
	c.timerMu.Lock()
	if !c.active.Load() {
		c.disarmLocked()
		c.timerMu.Unlock()
		return
	}
	c.timerMu.Unlock()
	_ = c.reveal.Shown(c.now())
	c.Refresh()
	if c.layer != nil {
		c.layer.Refresh()
	}
	c.schedule()
}

func (c *Chrome) schedule() {
	c.timerMu.Lock()
	defer c.timerMu.Unlock()
	c.disarmLocked()
	if !c.active.Load() || c.manual {
		return
	}
	delay, ok := c.reveal.NextDelay(c.now())
	if !ok {
		return
	}
	if c.reveal.Sliding() && (delay <= 0 || delay > repaintEvery) {
		delay = repaintEvery
	}
	if delay <= 0 {
		return
	}
	c.pending.Add(1)
	c.cancel = c.afterFn(delay, func() {
		c.timerMu.Lock()
		active := c.active.Load()
		c.timerMu.Unlock()
		if !active {
			c.pending.Done()
			return
		}
		fyne.Do(func() {
			defer c.pending.Done()
			c.kick()
		})
	})
}

// disarmLocked stops the armed timer. The caller holds timerMu. A callback
// that already started owns the pending count and must finish it.
func (c *Chrome) disarmLocked() {
	if c.cancel == nil {
		return
	}
	prevented := c.cancel()
	c.cancel = nil
	if prevented {
		c.pending.Done()
	}
}

func (c *Chrome) now() time.Time {
	if c.nowFn != nil {
		return c.nowFn()
	}
	return time.Now()
}

func realAfter(d time.Duration, fn func()) func() bool {
	timer := time.AfterFunc(d, fn)
	return timer.Stop
}

type chromeRenderer struct {
	c   *Chrome
	bg  *canvas.Rectangle
	row *fyne.Container
}

func (r *chromeRenderer) Destroy() {}

func (r *chromeRenderer) Layout(size fyne.Size) {
	barH := r.row.MinSize().Height
	if barH < 1 {
		barH = 1
	}
	y := (r.c.reveal.Shown(r.c.now()) - 1) * barH
	r.bg.Move(fyne.NewPos(0, y))
	r.bg.Resize(fyne.NewSize(size.Width, barH))
	r.row.Move(fyne.NewPos(0, y))
	r.row.Resize(fyne.NewSize(size.Width, barH))
}

func (r *chromeRenderer) MinSize() fyne.Size {
	if !r.c.active.Load() {
		return fyne.NewSize(0, 0)
	}
	bar := r.row.MinSize()
	h := topBand
	if vis := r.c.reveal.Shown(r.c.now()) * bar.Height; vis > h {
		h = vis
	}
	width := bar.Width
	if width < 1 {
		width = 1
	}
	return fyne.NewSize(width, h)
}

func (r *chromeRenderer) Objects() []fyne.CanvasObject {
	return []fyne.CanvasObject{r.bg, r.row}
}

func (r *chromeRenderer) Refresh() {
	if fyne.CurrentApp() != nil {
		th := r.c.Theme()
		v := fyne.CurrentApp().Settings().ThemeVariant()
		r.bg.FillColor = th.Color(theme.ColorNameBackground, v)
	}
	r.bg.Refresh()
	r.Layout(r.c.Size())
	canvas.Refresh(r.c)
}

type barItem struct {
	widget.BaseWidget
	label *widget.Label
	onTap func(fyne.CanvasObject)
}

func newBarItem(label string, onTap func(fyne.CanvasObject)) *barItem {
	b := &barItem{label: widget.NewLabel(label), onTap: onTap}
	b.ExtendBaseWidget(b)
	return b
}

func (b *barItem) CreateRenderer() fyne.WidgetRenderer {
	return widget.NewSimpleRenderer(container.NewPadded(b.label))
}

func (b *barItem) Tapped(_ *fyne.PointEvent) {
	if b.onTap != nil {
		b.onTap(b)
	}
}
