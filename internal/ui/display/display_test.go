package display_test

import (
	"image"
	"testing"

	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/display"
)

func TestMain(m *testing.M) { test.NewApp(); m.Run() }
func rotationFixture(t *testing.T) *presentationFixture {
	f := newPresentation(t, display.Config{})
	f.Present(storage.NewFileURI("/rotation.png"), &imaging.LoadedImage{Frames: []image.Image{image.NewNRGBA(image.Rect(0, 0, 2, 3))}}, false)
	return f
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
			s := rotationFixture(t)
			s.RotateBy(c.start)
			s.RotateBy(c.steps)

			if got := s.Rotation(); got != c.want {
				t.Errorf("Rotation() after RotateBy(%d) from %d = %d, want %d", c.steps, c.start, got, c.want)
			}
		})
	}
}

func TestRotateBy_AlwaysLandsInRange(t *testing.T) {
	s := rotationFixture(t)
	for steps := -9; steps <= 9; steps++ {
		s.RotateBy(steps)
		if r := s.Rotation(); r < 0 || r > 3 {
			t.Fatalf("Rotation() = %d after RotateBy(%d), want within 0-3", r, steps)
		}
	}
}

func TestResetRotation_ReportsWhetherItChangedAnything(t *testing.T) {
	s := rotationFixture(t)

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
