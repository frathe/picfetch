package display

import (
	"image"
	"testing"
	"time"

	"fyne.io/fyne/v2/test"
)

// Animation.Start and Stop both reach through fyne.CurrentApp().Driver(),
// so the fade tests need an app - same pattern as internal/ui/infoview's
// tests. The frame/rotation tests would run without one.
func TestMain(m *testing.M) {
	test.NewApp()
	m.Run()
}

func frame(w, h int) image.Image { return image.NewRGBA(image.Rect(0, 0, w, h)) }

func TestZeroValue_IsEmptyAndSafe(t *testing.T) {
	var s State

	if got := s.Count(); got != 0 {
		t.Errorf("Count() = %d, want 0 on a zero State", got)
	}
	if got := s.Index(); got != 0 {
		t.Errorf("Index() = %d, want 0 on a zero State", got)
	}
	if got := s.Rotation(); got != 0 {
		t.Errorf("Rotation() = %d, want 0 on a zero State", got)
	}
	if s.Fade() != nil {
		t.Error("Fade() should be nil on a zero State")
	}

	// None of these may panic with nothing installed.
	s.ResetFade()
	s.Clear()
	if s.ResetRotation() {
		t.Error("ResetRotation() = true on a zero State, want false")
	}
}

func TestRotateBy_NormalizesIntoQuarterTurnRange(t *testing.T) {
	cases := []struct {
		name  string
		start int
		steps int
		want  int
	}{
		{"one clockwise", 0, 1, 1},
		{"one counter-clockwise wraps", 0, -1, 3},
		{"wrap at the top boundary", 3, 1, 0},
		{"full turn is identity", 0, 4, 0},
		{"five is one", 0, 5, 1},
		{"full negative turn is identity", 0, -4, 0},
		{"minus five is three", 0, -5, 3},
		{"negative past a full turn from mid-range", 2, -6, 0},
		{"seven from one wraps twice", 1, 7, 0},
		{"large positive", 0, 4001, 1},
		{"large negative", 0, -4001, 3},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var s State
			s.RotateBy(c.start)
			s.RotateBy(c.steps)

			if got := s.Rotation(); got != c.want {
				t.Errorf("Rotation() after RotateBy(%d) from %d = %d, want %d", c.steps, c.start, got, c.want)
			}
		})
	}
}

func TestRotateBy_AlwaysLandsInRange(t *testing.T) {
	var s State
	for steps := -9; steps <= 9; steps++ {
		s.RotateBy(steps)
		if r := s.Rotation(); r < 0 || r > 3 {
			t.Fatalf("Rotation() = %d after RotateBy(%d), want within 0-3", r, steps)
		}
	}
}

func TestResetRotation_ReportsWhetherItChangedAnything(t *testing.T) {
	var s State

	if s.ResetRotation() {
		t.Error("ResetRotation() = true when already 0, want false")
	}

	s.RotateBy(2)
	if !s.ResetRotation() {
		t.Error("ResetRotation() = false with a rotation pending, want true")
	}
	if got := s.Rotation(); got != 0 {
		t.Errorf("Rotation() = %d after ResetRotation, want 0", got)
	}
	if s.ResetRotation() {
		t.Error("ResetRotation() = true immediately after a reset, want false")
	}
}

func TestCurrentAndReplaceCurrent_FollowTheIndex(t *testing.T) {
	var s State
	first, second := frame(1, 1), frame(2, 2)
	s.SetFrames([]image.Image{first, second})

	if got := s.Current(); got != first {
		t.Fatalf("Current() = %v, want the first frame at index 0", got.Bounds())
	}

	s.SetIndex(1)
	if got := s.Index(); got != 1 {
		t.Fatalf("Index() = %d after SetIndex(1), want 1", got)
	}
	if got := s.Current(); got != second {
		t.Fatalf("Current() = %v, want the second frame at index 1", got.Bounds())
	}

	swapped := frame(3, 3)
	s.ReplaceCurrent(swapped)
	if got := s.Current(); got != swapped {
		t.Fatal("Current() should return the frame ReplaceCurrent just installed")
	}

	s.SetIndex(0)
	if got := s.Current(); got != first {
		t.Fatal("ReplaceCurrent must only touch the frame at the current index")
	}
}

func TestRotated_ComposesTheCurrentIndexWithTheRotation(t *testing.T) {
	var s State
	s.SetFrames([]image.Image{frame(4, 4), frame(4, 2)})
	s.SetIndex(1)
	s.RotateBy(1)

	// A quarter turn of the asymmetric 4x2 frame at index 1 must come out
	// 2x4 - proof Rotated read the current frame, not frame 0.
	if b := s.Rotated().Bounds(); b.Dx() != 2 || b.Dy() != 4 {
		t.Errorf("Rotated().Bounds() = %v, want 2x4 (index 1's 4x2 turned once)", b)
	}
}

func TestRotated_AtZeroRotationIsTheCurrentFrame(t *testing.T) {
	var s State
	only := frame(4, 2)
	s.SetFrames([]image.Image{only})

	// imaging.RotateSteps returns the image itself for a zero rotation, so
	// this pins that no copy or recomposition happens on the common path.
	if got := s.Rotated(); got != only {
		t.Error("Rotated() at rotation 0 should be the current frame itself")
	}
}

func TestClear_ZeroesEverything(t *testing.T) {
	var s State
	s.SetFrames([]image.Image{frame(1, 1), frame(1, 1)})
	s.SetIndex(1)
	s.RotateBy(3)
	s.StartFade(time.Hour, func(float32) {})

	s.Clear()

	if got := s.Count(); got != 0 {
		t.Errorf("Count() = %d after Clear, want 0", got)
	}
	if got := s.Index(); got != 0 {
		t.Errorf("Index() = %d after Clear, want 0", got)
	}
	if got := s.Rotation(); got != 0 {
		t.Errorf("Rotation() = %d after Clear, want 0", got)
	}
	if s.Fade() != nil {
		t.Error("Fade() should be nil after Clear")
	}
}

func TestStartFade_TicksToTheEndUnderTheTestDriver(t *testing.T) {
	var s State
	var last float32 = -1
	s.StartFade(time.Hour, func(t float32) { last = t })

	// The test driver runs the whole animation as one synchronous
	// Tick(1.0) - what internal/ui's picture-frame tests rely on to
	// assert final translucency without waiting.
	if last != 1 {
		t.Errorf("tick's final value = %v, want 1 (ticked straight to the end)", last)
	}
	if s.Fade() == nil {
		t.Error("Fade() should report the fade StartFade installed")
	}
}

func TestStartFade_ReplacesTheFadeInProgress(t *testing.T) {
	var s State
	s.StartFade(time.Hour, func(float32) {})
	first := s.Fade()

	s.StartFade(time.Hour, func(float32) {})

	if s.Fade() == first {
		t.Error("StartFade should install a fresh animation, not reuse the stopped one")
	}
}

func TestResetFade_SafeWithNoFadeAndClearsARunningOne(t *testing.T) {
	var s State

	s.ResetFade() // must not panic with no fade running

	s.StartFade(time.Hour, func(float32) {})
	if s.Fade() == nil {
		t.Fatal("premise: StartFade should install a fade")
	}

	s.ResetFade()
	if s.Fade() != nil {
		t.Error("Fade() should be nil after ResetFade")
	}

	s.ResetFade() // and calling it again stays safe
}
