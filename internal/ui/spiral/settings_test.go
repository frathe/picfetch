package spiral

import (
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/uitest"
)

// sliderRowSliders returns the *widget.Slider objects found directly in
// box, in the order they were added - i.e. the order addSliderRow's calls
// wired up the panel's rows.
func sliderRowSliders(box *fyne.Container) []*widget.Slider {
	var sliders []*widget.Slider
	for _, obj := range box.Objects {
		if s, ok := obj.(*widget.Slider); ok {
			sliders = append(sliders, s)
		}
	}
	return sliders
}

// TestAddSliderRowOnChangedUpdatesTargetShaderUniformAndActivity exercises
// addSliderRow in isolation from the rest of newSettingsPanel's layout, so
// this test only fails when the wiring itself - target pointer, shader
// uniform, activity timer - breaks, not when unrelated layout changes.
func TestAddSliderRowOnChangedUpdatesTargetShaderUniformAndActivity(t *testing.T) {
	test.NewApp()
	st := newState()
	shader := newShader(st)

	p := &settingsPanel{content: container.NewWithoutLayout()}
	target := 0.0
	p.addSliderRow(0, "Test", 0, 10, 3, 1, "arms", &target, shader)

	sliders := sliderRowSliders(p.content)
	if len(sliders) != 1 {
		t.Fatalf("box has %d sliders after one addSliderRow call; want 1", len(sliders))
	}
	s := sliders[0]

	p.lastMove.Store(0) // zero out so a nonzero value after SetValue is unambiguous
	s.SetValue(7)

	if target != s.Value {
		t.Errorf("target = %f; want slider's post-clamp value %f", target, s.Value)
	}
	if got := shader.Uniforms["arms"]; got != float32(s.Value) {
		t.Errorf(`Uniforms["arms"] = %f; want %f`, got, s.Value)
	}
	if p.lastMove.Load() == 0 {
		t.Error("lastMove still 0 after OnChanged fired; want it to mark activity")
	}
}

func TestPanelVisible(t *testing.T) {
	const timeout = 5 * time.Second
	now := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	if !panelVisible(now.Add(-timeout), now, timeout) {
		t.Error("panelVisible at exactly the timeout boundary = false; want true (just inside)")
	}
	if panelVisible(now.Add(-timeout-time.Millisecond), now, timeout) {
		t.Error("panelVisible just past the timeout = true; want false")
	}
}

func TestPanelAnchor(t *testing.T) {
	got := panelAnchor(fyne.NewSize(800, 600), fyne.NewSize(settingsPanelWidth, settingsPanelHeight))
	want := fyne.NewPos(800-settingsPanelWidth-settingsPanelMargin, settingsPanelMargin)
	if got != want {
		t.Errorf("panelAnchor(800x600) = %v; want %v", got, want)
	}
}

func TestTunnelControls(t *testing.T) {
	s := newTestSpiral(t)
	sources := uitest.TempDirJPEGURIs(t, "controls.jpg")
	s.Show(sources)
	sliders := sliderRowSliders(s.panel.content)
	if len(sliders) != 8 {
		t.Fatalf("got %d sliders, want existing three plus speed/gap/randomness/transparency/size", len(sliders))
	}
	speed, gap := sliders[3], sliders[4]
	if speed.Min != .35 || speed.Max != 2 || speed.Step != .05 || speed.Value != 1 {
		t.Fatalf("image speed range/default = %+v", speed)
	}
	if gap.Min != .75 || gap.Max != 6 || gap.Step != .05 || gap.Value != 2.5 {
		t.Fatalf("image gap range/default = %+v", gap)
	}
	speed.SetValue(1.5)
	gap.SetValue(3)
	s.Close()
	s.Settle()
	s.Show(sources)
	sliders = sliderRowSliders(s.panel.content)
	if math.Abs(sliders[3].Value-1.5) > 1e-6 || math.Abs(sliders[4].Value-3) > 1e-6 {
		t.Fatalf("reopened speed=%g gap=%g", sliders[3].Value, sliders[4].Value)
	}
	if sliders[5].Min != 0 || sliders[5].Max != 100 || sliders[5].Step != 1 || sliders[5].Value != 35 {
		t.Fatal("randomness range/default")
	}
	var order *widget.Select
	for _, o := range s.panel.content.Objects {
		if selectBox, ok := o.(*widget.Select); ok {
			order = selectBox
		}
	}
	if order == nil || order.Selected != "Main order" {
		t.Fatal("order control absent from panel")
	}
	order.SetSelected("Random")
	if !s.st.randomOrder {
		t.Fatal("order control did not change mode")
	}
}

func TestSettingsPanelEmptySources(t *testing.T) {
	s := newTestSpiral(t)
	sources := uitest.TempDirJPEGURIs(t, "controls.jpg")
	for _, files := range [][]fyne.URI{nil, sources, {}, sources, nil} {
		s.Show(files)
		s.win.Resize(fyne.NewSize(800, 600))
		s.frame(0)

		var labels []string
		var sliders, orders int
		var viewport *container.Scroll
		var walk func(fyne.CanvasObject)
		walk = func(object fyne.CanvasObject) {
			if !object.Visible() {
				return
			}
			switch object := object.(type) {
			case *fyne.Container:
				for _, child := range object.Objects {
					walk(child)
				}
			case *container.Scroll:
				viewport = object
				walk(object.Content)
			case *widget.Label:
				labels = append(labels, object.Text)
			case *widget.Slider:
				sliders++
			case *widget.Select:
				orders++
			}
		}
		walk(s.win.Canvas().Overlays().Top())
		wantSize := fyne.NewSize(520, 370)
		wantSliders, wantOrders := 8, 1
		if len(files) == 0 {
			wantSize = fyne.NewSize(260, 250)
			wantSliders, wantOrders = 3, 0
			if !slices.Equal(labels, []string{"Arms", "Twists", "Pixel Density"}) {
				t.Errorf("empty-list panel labels = %v; want only spiral controls", labels)
			}
		}
		if sliders != wantSliders || orders != wantOrders {
			t.Errorf("%d sources: live panel has %d sliders and %d order controls; want %d and %d", len(files), sliders, orders, wantSliders, wantOrders)
		}
		if got := s.panel.box.Size(); got != wantSize {
			t.Errorf("%d sources: panel size = %v; want %v", len(files), got, wantSize)
		}
		if viewport == nil || viewport.Content.MinSize() != wantSize {
			t.Errorf("%d sources: scrollable surface does not match panel size %v", len(files), wantSize)
		}
		if got := s.panel.box.Position(); got != fyne.NewPos(800-wantSize.Width-settingsPanelMargin, settingsPanelMargin) {
			t.Errorf("%d sources: panel is not anchored to the right edge: %v", len(files), got)
		}
		s.Close()
		s.Settle()
	}
}

func TestTunnelTransparency(t *testing.T) {
	s := newTestSpiral(t)
	s.st.randomness = 0
	s.now = func() time.Time { return time.Unix(1000, 0) }
	sources := uitest.TempDirJPEGURIs(t, "opacity.jpg")
	s.Show(sources)
	s.win.Resize(fyne.NewSize(800, 600))
	settleTunnelPreviews(s)
	active := s.tunnel.slots
	if active[0] == nil {
		t.Fatal("no in-flight picture for the live transparency check")
	}
	sliders := sliderRowSliders(s.panel.content)
	if len(sliders) != 8 {
		t.Fatalf("transparency control missing: %d sliders", len(sliders))
	}
	control := sliders[6]
	var contains func(fyne.CanvasObject) bool
	contains = func(o fyne.CanvasObject) bool {
		if o == control {
			return true
		}
		switch c := o.(type) {
		case *fyne.Container:
			if slices.ContainsFunc(c.Objects, contains) {
				return true
			}
		case *container.Scroll:
			return contains(c.Content)
		}
		return false
	}
	if !contains(s.win.Canvas().Overlays().Top()) {
		t.Fatal("transparency slider is absent from the live surface")
	}
	if control.Min != -70 || control.Max != 84 || control.Step != 1 || control.Value != 0 {
		t.Fatal("transparency range/default")
	}
	for _, tc := range []struct{ shift, lo, hi float64 }{{0, .15, .85}, {10, .05, .75}, {-10, .25, .85}, {84, .01, .01}, {-70, .85, .85}} {
		s.panel.lastMove.Store(0)
		control.SetValue(tc.shift)
		if s.tunnel.slots != active {
			t.Fatal("transparency change restarted an active flight")
		}
		for key, want := range map[string]float64{"imageOpacityMin": tc.lo, "imageOpacityMax": tc.hi} {
			if got := float64(s.shader.Uniforms[key]); math.Abs(got-want) > 1e-6 {
				t.Fatalf("shift %g: %s=%g want %g", tc.shift, key, got, want)
			}
		}
		if tc.shift != 0 && s.panel.lastMove.Load() == 0 {
			t.Fatal("transparency change did not keep controls visible")
		}
	}
	control.SetValue(10)
	s.Close()
	s.Settle()
	s.Show(sources)
	sliders = sliderRowSliders(s.panel.content)
	if sliders[6].Value != 10 || math.Abs(float64(s.shader.Uniforms["imageOpacityMax"])-.75) > 1e-6 {
		t.Fatal("reopen lost transparency range")
	}
	var label *widget.Label
	for _, o := range s.panel.content.Objects {
		if l, ok := o.(*widget.Label); ok && l.Text == "Image transparency: 25-95%" {
			label = l
		}
	}
	if label == nil {
		t.Fatal("transparency control does not display its resulting range")
	}
}

func TestTunnelImageSize(t *testing.T) {
	s := newTestSpiral(t)
	now := time.Unix(1000, 0)
	s.now = func() time.Time { return now }
	s.st.randomness, s.st.imageSpeed, s.st.imageGap = 0, .35, .75
	sources := uitest.TempDirJPEGURIs(t, "size.jpg")
	s.Show(sources)
	s.win.Resize(fyne.NewSize(800, 600))
	settleTunnelPreviews(s)
	sliders := sliderRowSliders(s.panel.content)
	if len(sliders) != 8 {
		t.Fatalf("image-size control missing: %d sliders", len(sliders))
	}
	size := sliders[7]
	if size.Min != .5 || size.Max != 2 || size.Step != .05 || size.Value != 1 {
		t.Fatal("image-size range/default")
	}
	first := *s.tunnel.slots[0]
	size.SetValue(2)
	if *s.tunnel.slots[0] != first {
		t.Fatal("size change moved an in-flight image")
	}
	now = now.Add(time.Second)
	s.frame(1)
	settleTunnelPreviews(s)
	size.SetValue(.5)
	now = now.Add(time.Second)
	s.frame(1)
	settleTunnelPreviews(s)
	for i, want := range []float64{1, 2, .5} {
		if got := s.shader.Uniforms[fmt.Sprintf("traveller%dSize", i)]; got != float32(want) {
			t.Fatalf("slot %d size=%g want %g", i, got, want)
		}
		f := s.tunnel.slots[i]
		if f == nil {
			t.Fatalf("slot %d not admitted", i)
		}
		pose := f.pose(f.born, s.tunnelFrame())
		if math.Abs(math.Max(pose.width, pose.height)-.1*600*want) > 1e-6 {
			t.Fatalf("slot %d: CPU footprint %gx%g disagrees with selected size", i, pose.width, pose.height)
		}
	}
	s.Close()
	s.Settle()
	s.Show(sources)
	if sliderRowSliders(s.panel.content)[7].Value != .5 {
		t.Fatal("reopen lost image size")
	}
}

func TestTunnelSmallControls(t *testing.T) {
	s := newTestSpiral(t)
	s.Show(uitest.TempDirJPEGURIs(t, "controls.jpg"))
	s.win.Resize(fyne.NewSize(320, 240))
	s.frame(0)
	box, size := s.panel.box, s.win.Canvas().Size()
	if box.Position().X+box.Size().Width > size.Width || box.Position().Y+box.Size().Height > size.Height {
		t.Fatal("controls extend beyond the resized window")
	}
	var scroll *container.Scroll
	for _, object := range box.Objects {
		if viewport, ok := object.(*container.Scroll); ok {
			scroll = viewport
		}
	}
	if scroll == nil {
		t.Fatal("small-window controls have no scrollable viewport")
	}
	s.panel.lastMove.Store(0)
	scroll.Scrolled(&fyne.ScrollEvent{Scrolled: fyne.Delta{DY: -100}})
	if s.panel.lastMove.Load() == 0 {
		t.Fatal("scrolling the controls did not refresh the idle timer")
	}
}

// TestNewSettingsPanelOverlayHoldsTrackerAndBoxAsSeparateChildren is a
// regression guard for the hit-testing trap the donor's comment on
// newSettingsPanel describes: Fyne's hit-testing stops descending into a
// CanvasObject's children the moment that object reports Visible() ==
// false, so a mouse tracker nested inside the (auto-hiding) box would go
// deaf whenever the box hides, and nothing would be left listening for the
// movement that's supposed to bring it back. The tracker must live in the
// never-hidden overlay as a sibling of the box, not inside it.
func TestNewSettingsPanelOverlayHoldsTrackerAndBoxAsSeparateChildren(t *testing.T) {
	test.NewApp()
	st := newState()
	shader := newShader(st)
	p := newSettingsPanel(st, shader, true)

	if len(p.overlay.Objects) != 2 {
		t.Fatalf("overlay has %d children; want 2 (the mouse tracker and the box)", len(p.overlay.Objects))
	}
	if _, ok := p.overlay.Objects[0].(*hoverRect); !ok {
		t.Errorf("overlay.Objects[0] = %T; want *hoverRect (the mouse tracker)", p.overlay.Objects[0])
	}
	if p.overlay.Objects[1] != fyne.CanvasObject(p.box) {
		t.Error("overlay.Objects[1] is not p.box; the box must be a direct sibling of the tracker in the overlay, not nested inside it")
	}
	for _, obj := range p.box.Objects {
		if _, ok := obj.(*hoverRect); ok {
			t.Error("found the mouse tracker nested inside p.box; it must live in the overlay instead so hiding the box doesn't silence it")
		}
	}
}

func TestNewSettingsPanelStartsVisibleAndTickHidesAfterIdleTimeout(t *testing.T) {
	a := test.NewApp()
	w := a.NewWindow("")
	defer w.Close()
	w.Resize(fyne.NewSize(640, 480))

	st := newState()
	shader := newShader(st)
	p := newSettingsPanel(st, shader, true)

	if !p.box.Visible() {
		t.Fatal("box.Visible() = false immediately after newSettingsPanel; want true - a freshly built panel must start visible")
	}

	// Drive the idle timer by writing a stale timestamp directly, rather
	// than sleeping past settingsPanelIdleTimeout.
	stale := time.Now().Add(-settingsPanelIdleTimeout - time.Second).UnixMilli()
	p.lastMove.Store(stale)

	p.tick(w)

	if p.box.Visible() {
		t.Error("box.Visible() = true after tick() saw a stale lastMove; want false (auto-hidden)")
	}
}

func TestTickAnchorsBoxToWindowRightEdge(t *testing.T) {
	a := test.NewApp()
	w := a.NewWindow("")
	defer w.Close()
	w.Resize(fyne.NewSize(640, 480))

	st := newState()
	shader := newShader(st)
	p := newSettingsPanel(st, shader, true)

	p.tick(w)

	want := panelAnchor(w.Canvas().Size(), p.surfaceSize)
	if got := p.box.Position(); got != want {
		t.Errorf("box.Position() after tick() = %v; want %v (panelAnchor of the window's canvas size)", got, want)
	}
}

// TestAddSliderRowStepSmallerThanRange guards the reasoning in addSliderRow's
// doc comment: widget.NewSlider defaults Step to 1, and Fyne's snapping
// rounds every drag to a multiple of Step that can fall outside [Min, Max]
// when Step exceeds the slider's range, sticking the handle. Every row
// newSettingsPanel wires up must keep Step below its own range.
func TestAddSliderRowStepSmallerThanRange(t *testing.T) {
	test.NewApp()
	st := newState()
	shader := newShader(st)
	p := newSettingsPanel(st, shader, true)

	sliders := sliderRowSliders(p.content)
	if len(sliders) < 3 {
		t.Fatalf("box has %d sliders; want at least Arms, Twists, Pixel Density", len(sliders))
	}
	for i, s := range sliders {
		if rng := s.Max - s.Min; s.Step >= rng {
			t.Errorf("slider %d: Min=%f Max=%f Step=%f; want Step < range %f", i, s.Min, s.Max, s.Step, rng)
		}
	}
}
