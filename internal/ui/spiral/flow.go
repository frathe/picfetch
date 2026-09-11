package spiral

import "math"

// A flowProfile remains fixed for 3-7 admitted photographs. Drift bounds keep
// neighbouring batches calm without introducing per-frame noise.
type flowProfile struct{ speed, gap, margin, curve float64 }

func (t *tunnelSession) nextProfile(randomness float64) {
	if t.remaining > 0 {
		return
	}
	r := randomness / 100
	unchanged := t.admissions > 0 && randomness == t.randomness
	sample := func(base, spread, drift, previous float64) float64 {
		lo, hi := base-spread*r, base+spread*r
		if unchanged {
			lo = math.Max(lo, previous-drift*r)
			hi = math.Min(hi, previous+drift*r)
		}
		return lo + t.rng.Float64()*(hi-lo)
	}
	t.profile = flowProfile{
		sample(1, .30, .10, t.profile.speed), sample(1, .30, .10, t.profile.gap),
		sample(1, .30, .10, t.profile.margin), sample(.35, .12, .04, t.profile.curve),
	}
	t.randomness = randomness
	t.remaining = 3 + t.rng.IntN(5)
}

func (t *tunnelSession) chooseRoute(f flight, frame tunnelFrame) (flight, bool) {
	best, separation := f, -1.0
	for range 7 { // One bearing and at most six resamples.
		f.angle = t.rng.Float64() * 2 * math.Pi
		if !f.pose(f.born, frame).inside(frame) {
			continue
		}
		distance := math.Abs(math.Remainder(f.angle-t.lastBearing, 2*math.Pi))
		if t.admissions == 0 || distance >= math.Pi/4 {
			return f, true
		}
		if distance > separation {
			best, separation = f, distance
		}
	}
	if separation >= 0 {
		return best, true
	}
	// Follow can put the core against any edge. A deterministic inward
	// fallback avoids an unbounded rejection loop there.
	f.angle = math.Atan2(frame.height/2-frame.cy, frame.width/2-frame.cx)
	return f, f.pose(f.born, frame).inside(frame)
}

func (s *Spiral) resetTunnelOrder() {
	t := s.tunnel
	if t == nil || t.randomOrder == s.st.randomOrder {
		return
	}
	t.randomOrder = s.st.randomOrder
	t.orderRevision++
	if t.decodeCancel != nil {
		t.decodeCancel()
	}
	t.ready = nil
	t.order = nil
	t.next, t.cycleSuccess, t.retryAt = 0, 0, 0
}

func (s *Spiral) makeTunnelOrder() {
	t := s.tunnel
	t.order = make([]int, len(s.sources))
	for i := range t.order {
		t.order[i] = i
	}
	if !t.randomOrder {
		return
	}
	t.rng.Shuffle(len(t.order), func(i, j int) { t.order[i], t.order[j] = t.order[j], t.order[i] })
	// Leave the previous successful source until the other candidates have
	// had a chance. This also avoids boundary repeats after decode failures.
	for i, index := range t.order {
		if s.sources[index].String() == t.lastSource {
			copy(t.order[i:], t.order[i+1:])
			t.order[len(t.order)-1] = index
			break
		}
	}
}
