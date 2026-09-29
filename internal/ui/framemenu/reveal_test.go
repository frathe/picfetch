package framemenu

import (
	"math"
	"testing"
	"time"
)

// Picture-frame menu motion, driven by a clock the test owns. Fractions are
// the bar's travel: 0 is fully above the screen, 1 is fully in.

func TestReveal_StaysHiddenUntilPointerDwellsThenSlidesIn(t *testing.T) {
	start := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	var reveal Reveal

	reveal.Pointer(true, start)
	if got := reveal.Shown(start.Add(499 * time.Millisecond)); got != 0 {
		t.Fatalf("shown after 499ms = %v, want 0", got)
	}

	// Dwell ends at 500ms and the slide begins there, so the bar is still
	// fully hidden at that instant.
	if got := reveal.Shown(start.Add(500 * time.Millisecond)); got != 0 {
		t.Fatalf("shown at 500ms = %v, want 0", got)
	}

	// 50ms into a 200ms ease-in-out is a quarter of the timeline and an
	// eighth of the travel (t*t*2 at t=0.25).
	quarter := start.Add(550 * time.Millisecond)
	if got := reveal.Shown(quarter); math.Abs(float64(got-0.125)) > 0.001 {
		t.Fatalf("shown at 550ms = %v, want 0.125", got)
	}

	if got := reveal.Shown(start.Add(700 * time.Millisecond)); got != 1 {
		t.Fatalf("shown at 700ms = %v, want 1", got)
	}
}

func TestReveal_LeaveBeforeDwellStaysHidden(t *testing.T) {
	start := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	var reveal Reveal

	reveal.Pointer(true, start)
	reveal.Pointer(false, start.Add(400*time.Millisecond))

	if got := reveal.Shown(start.Add(2 * time.Second)); got != 0 {
		t.Fatalf("shown after a 400ms visit = %v, want 0", got)
	}
}

func TestReveal_LeaveAfterShownSlidesOutAfterDwell(t *testing.T) {
	start := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	var reveal Reveal

	reveal.Pointer(true, start)
	shown := start.Add(700 * time.Millisecond)
	if got := reveal.Shown(shown); got != 1 {
		t.Fatalf("shown before leaving = %v, want 1", got)
	}

	reveal.Pointer(false, shown)
	if got := reveal.Shown(shown.Add(499 * time.Millisecond)); got != 1 {
		t.Fatalf("shown 499ms after leaving = %v, want 1", got)
	}

	// Hide dwell ends at +500ms and the 200ms slide finishes at +700ms.
	if got := reveal.Shown(shown.Add(700 * time.Millisecond)); got != 0 {
		t.Fatalf("shown 700ms after leaving = %v, want 0", got)
	}
}

func TestReveal_OpenMenuStaysFullyShown(t *testing.T) {
	start := time.Date(2026, 9, 29, 12, 0, 0, 0, time.UTC)
	var reveal Reveal

	reveal.Pointer(true, start)
	opened := start.Add(600 * time.Millisecond) // mid-slide
	reveal.SetMenuOpen(true, true, opened)
	reveal.Pointer(false, opened.Add(10*time.Millisecond))

	if got := reveal.Shown(opened.Add(5 * time.Second)); got != 1 {
		t.Fatalf("shown while a menu is open = %v, want 1", got)
	}

	// Closing with the pointer away starts the same 500ms hide dwell.
	closed := opened.Add(5 * time.Second)
	reveal.SetMenuOpen(false, false, closed)
	if got := reveal.Shown(closed.Add(499 * time.Millisecond)); got != 1 {
		t.Fatalf("shown 499ms after the menu closed = %v, want 1", got)
	}
	if got := reveal.Shown(closed.Add(700 * time.Millisecond)); got != 0 {
		t.Fatalf("shown 700ms after the menu closed = %v, want 0", got)
	}
}
