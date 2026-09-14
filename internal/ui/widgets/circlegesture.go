package widgets

import (
	"math"
	"time"

	"fyne.io/fyne/v2"
)

// CircleGesture recognizes ten pointer revolutions around a mascot's head.
// Move takes head-relative positions in units of the host's inner dead-zone
// radius. Each host owns its own zero-value-ready instance on the UI thread.
type CircleGesture struct {
	previous  fyne.Position
	last      time.Time
	known     bool
	started   time.Time
	direction float64
	turn      float64
	furthest  float64
}

// Reset forgets the unfinished attempt and its previous pointer sample.
func (g *CircleGesture) Reset() { *g = CircleGesture{} }

// Move consumes actual pointer movement and reports a completed attempt once.
// Layout, surface departure, hide and close must call Reset instead.
func (g *CircleGesture) Move(position fyne.Position, at time.Time) bool {
	x, y := float64(position.X), float64(position.Y)
	if math.IsNaN(x) || math.IsNaN(y) || math.IsInf(x, 0) || math.IsInf(y, 0) || math.Hypot(x, y) <= 1 {
		g.Reset()
		return false
	}
	if !g.known || at.Before(g.last) || (g.direction != 0 && at.Sub(g.started) > 20*time.Second) {
		g.seed(position, at)
		return false
	}
	px, py := float64(g.previous.X), float64(g.previous.Y)
	g.previous, g.last = position, at
	delta := math.Atan2(px*y-py*x, px*x+py*y)
	// Sparse jumps larger than a quarter turn have no reliable direction.
	// A chord through the inner dead zone is a center crossing, even when
	// neither sampled endpoint lands inside it.
	dx, dy := x-px, y-py
	distanceSquared := dx*dx + dy*dy
	if distanceSquared == 0 {
		return false
	}
	nearest := max(0, min(1, -(px*dx+py*dy)/distanceSquared))
	if math.Abs(delta) > math.Pi/2+1e-6 || math.Hypot(px+nearest*dx, py+nearest*dy) <= 1 {
		g.seed(position, at)
		return false
	}
	if g.direction == 0 {
		if math.Abs(delta) < 1e-6 {
			return false
		}
		g.direction = math.Copysign(1, delta)
		g.started = at
	}
	g.turn += delta * g.direction
	g.furthest = max(g.furthest, g.turn)
	// Allow up to 15 degrees of backward wobble from the furthest angle.
	// Signed progress pays that motion back before earning another turn.
	if g.furthest-g.turn > math.Pi/12 {
		g.seed(position, at)
		return false
	}
	if g.turn < 20*math.Pi-1e-6 {
		return false
	}
	g.Reset()
	return true
}

func (g *CircleGesture) seed(position fyne.Position, at time.Time) {
	g.Reset()
	g.previous, g.last, g.known = position, at, true
}
