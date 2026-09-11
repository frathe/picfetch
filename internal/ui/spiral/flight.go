package spiral

import "math"

// flight is immutable after admission. Geometry is also used for retirement
// and route admission; the shader evaluates the same flight on every paint.
type flight struct {
	source                                             int
	born, duration, angle, curve, margin, aspect, size float64
}

type tunnelFrame struct{ width, height, cx, cy float64 }
type flightPose struct{ x, y, width, height, opacity, depth float64 }

func (f tunnelFrame) valid() bool { return f.width > 0 && f.height > 0 }

func photoSize(aspect, edge float64) (float64, float64) {
	if aspect >= 1 {
		return edge, edge / aspect
	}
	return edge * aspect, edge
}

// rayExit is the far intersection with the frame expanded by a card's half
// extents. It also works when a resized frame leaves the core outside it.
func (f tunnelFrame) rayExit(angle, halfW, halfH float64) float64 {
	dx, dy := math.Cos(angle), math.Sin(angle)
	tx, ty := math.Inf(1), math.Inf(1)
	if dx > 1e-8 {
		tx = (f.width + halfW - f.cx) / dx
	}
	if dx < -1e-8 {
		tx = (-halfW - f.cx) / dx
	}
	if dy > 1e-8 {
		ty = (f.height + halfH - f.cy) / dy
	}
	if dy < -1e-8 {
		ty = (-halfH - f.cy) / dy
	}
	return math.Max(0, math.Min(tx, ty))
}

func (f flight) pose(now float64, frame tunnelFrame) flightPose {
	if !frame.valid() || f.duration <= 0 || f.aspect <= 0 || f.size <= 0 {
		return flightPose{}
	}
	unit := math.Min(frame.width, frame.height)
	p := clampFloat((now-f.born)/f.duration, 0, 1)
	depth := 0.2*p + 0.8*p*p
	w0, h0 := photoSize(f.aspect, 0.10*unit*f.size)
	w1, h1 := photoSize(f.aspect, 0.25*unit*f.size)
	start := 0.086*unit + math.Hypot(w0, h0)/2
	end := math.Max(frame.rayExit(f.angle+f.curve, w1/2, h1/2)+0.035*unit*f.margin,
		0.10*unit+math.Hypot(w1, h1)/2)
	angle := f.angle + f.curve*depth
	radius := start + (end-start)*depth
	edge := frame.rayExit(angle, 0, 0)
	opacity := 0.15 + 0.70*clampFloat((radius-start)/math.Max(edge-start, 0.001*unit), 0, 1)
	w, h := photoSize(f.aspect, (0.10+0.15*depth)*unit*f.size)
	return flightPose{frame.cx + radius*math.Cos(angle), frame.cy + radius*math.Sin(angle), w, h, opacity, depth}
}

func (p flightPose) outside(f tunnelFrame) bool {
	return p.x+p.width/2 <= 0 || p.x-p.width/2 >= f.width ||
		p.y+p.height/2 <= 0 || p.y-p.height/2 >= f.height
}

func (p flightPose) inside(f tunnelFrame) bool {
	return p.x-p.width/2 >= 0 && p.x+p.width/2 <= f.width &&
		p.y-p.height/2 >= 0 && p.y+p.height/2 <= f.height
}
