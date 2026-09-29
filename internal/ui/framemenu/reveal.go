// Package framemenu is the picture-frame menu bar on platforms whose menu
// lives inside the window. It stays off the picture until the pointer has
// rested on the top edge, then slides in and out. An open menu holds it
// fully shown.
package framemenu

import "time"

const (
	// dwell is how long the pointer must stay in, or stay away, before a
	// slide starts. Shorter visits are ignored.
	dwell = 500 * time.Millisecond
	// slide is how long the bar takes to travel once a dwell has elapsed.
	slide = 200 * time.Millisecond
	// topBand is the hidden hit strip along the top edge, in Fyne units.
	topBand float32 = 24
)

// Reveal is how far the bar has traveled. 0 is fully above the top edge
// and 1 is fully shown. The clock is the caller's.
type Reveal struct {
	// at is the settled travel, 0 hidden and 1 shown. A slide interpolates
	// from slideFrom toward slideTo and lands on at.
	at            float32
	pointerInside bool
	menuOpen      bool
	dwelling      bool
	dwellFrom     time.Time
	dwellInside   bool
	sliding       bool
	slideFrom     float32
	slideTo       float32
	slideStart    time.Time
}

// Pointer records whether the pointer is over the top band or the bar.
// While a menu is open the bar is pinned, so pointer changes are ignored.
func (r *Reveal) Pointer(inside bool, now time.Time) {
	if r.menuOpen || inside == r.pointerInside {
		return
	}
	r.pointerInside = inside
	r.armDwell(now)
}

// SetMenuOpen pins the bar fully shown while a menu is open. pointerInside
// is the pointer's place when the menu closes, which decides whether the
// hide dwell starts.
func (r *Reveal) SetMenuOpen(open bool, pointerInside bool, now time.Time) {
	if open {
		r.menuOpen = true
		r.pointerInside = true
		r.dwelling = false
		r.sliding = false
		r.at = 1
		r.slideFrom = 1
		r.slideTo = 1
		return
	}
	r.menuOpen = false
	r.pointerInside = pointerInside
	if pointerInside {
		r.dwelling = false
		r.sliding = false
		r.at = 1
		r.slideFrom = 1
		r.slideTo = 1
		return
	}
	r.armDwell(now)
}

// Shown is the eased travel at now. Reading it applies a dwell or slide
// that has come due, so a late poll still lands on the right fraction.
func (r *Reveal) Shown(now time.Time) float32 {
	r.evaluate(now)
	return r.interpolate(now)
}

// Sliding reports an in-progress slide, so the widget can keep repainting.
func (r *Reveal) Sliding() bool { return r.sliding }

// NextDelay is when the next dwell or slide boundary falls. The false
// result means the bar is settled.
func (r *Reveal) NextDelay(now time.Time) (time.Duration, bool) {
	if r.dwelling {
		d := r.dwellFrom.Add(dwell).Sub(now)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	if r.sliding {
		d := r.slideStart.Add(slide).Sub(now)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

// Reset returns the bar to hidden and idle. Picture-frame exit uses it.
func (r *Reveal) Reset() { *r = Reveal{} }

func (r *Reveal) armDwell(now time.Time) {
	want := float32(0)
	if r.pointerInside {
		want = 1
	}
	if !r.sliding && !r.dwelling && r.at == want {
		return
	}
	if r.sliding && r.slideTo == want {
		r.dwelling = false
		return
	}
	r.dwelling = true
	r.dwellFrom = now
	r.dwellInside = r.pointerInside
}

func (r *Reveal) evaluate(now time.Time) {
	if r.menuOpen {
		return
	}
	if r.dwelling && !now.Before(r.dwellFrom.Add(dwell)) {
		want := float32(0)
		if r.dwellInside {
			want = 1
		}
		started := r.dwellFrom.Add(dwell)
		r.dwelling = false
		r.startSlide(want, started)
	}
	if r.sliding && !now.Before(r.slideStart.Add(slide)) {
		r.finishSlide()
	}
}

func (r *Reveal) startSlide(to float32, now time.Time) {
	cur := r.interpolate(now)
	if r.sliding && r.slideTo == to {
		return
	}
	if !r.sliding && cur == to {
		r.at = to
		r.slideTo = to
		return
	}
	r.slideFrom = cur
	r.slideTo = to
	r.slideStart = now
	r.sliding = cur != to
	if cur == to {
		r.at = to
	}
}

func (r *Reveal) finishSlide() {
	r.sliding = false
	r.at = r.slideTo
	r.slideFrom = r.slideTo
}

func (r *Reveal) interpolate(now time.Time) float32 {
	if !r.sliding {
		return r.at
	}
	elapsed := now.Sub(r.slideStart)
	if elapsed <= 0 {
		return r.slideFrom
	}
	if elapsed >= slide {
		return r.slideTo
	}
	t := float32(elapsed) / float32(slide)
	return r.slideFrom + (r.slideTo-r.slideFrom)*easeInOut(t)
}

// easeInOut matches Fyne's curve: slow at both ends, halfway at mid time.
func easeInOut(t float32) float32 {
	if t <= 0.5 {
		return t * t * 2
	}
	return -1 + (4-t*2)*t
}
