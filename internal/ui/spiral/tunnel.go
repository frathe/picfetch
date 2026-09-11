package spiral

import (
	"context"
	"fmt"
	"image"
	"math"
	"math/rand/v2"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/imaging"
)

const tunnelPreviewEdge = 512

type tunnelPreview struct {
	source int
	pixels image.Image
}

// tunnelSession belongs to one opening. The decoder's busy flag and waitgroup
// belong to Spiral, so an uninterruptible old read cannot multiply on reopen.
type tunnelSession struct {
	ctx                                context.Context
	cancel                             context.CancelFunc
	start                              time.Time
	slots                              [3]*flight
	ready                              *tunnelPreview
	next                               int
	admissions                         uint64
	lastAdmission                      float64
	cycleSuccess                       int
	retryAt                            float64
	failed                             []bool
	usable                             map[string]bool
	profile                            flowProfile
	remaining                          int
	randomness, gapFactor, lastBearing float64
	randomOrder                        bool
	order                              []int
	orderRevision                      uint64
	decodeCancel                       context.CancelFunc
	lastSource                         string
	rng                                *rand.Rand
}

func (s *Spiral) startTunnel() {
	ctx, cancel := context.WithCancel(context.Background())
	s.tunnel = &tunnelSession{ctx: ctx, cancel: cancel, start: s.now(),
		rng:    rand.New(rand.NewPCG(rand.Uint64(), rand.Uint64())),
		failed: make([]bool, len(s.sources)), usable: make(map[string]bool), gapFactor: 1,
		randomOrder: s.st.randomOrder}
	s.advanceTunnel()
}

func (s *Spiral) tunnelFrame() tunnelFrame {
	size := s.win.Canvas().Size()
	w, h := s.win.Canvas().PixelCoordinateForPosition(fyne.NewPos(size.Width, size.Height))
	x, y := s.st.centerOffset()
	return tunnelFrame{float64(w), float64(h), float64(w)/2 + x, float64(h)/2 + y}
}

func (s *Spiral) advanceTunnel() {
	t := s.tunnel
	if t == nil || s.win == nil {
		return
	}
	s.resetTunnelOrder()
	now := s.now().Sub(t.start).Seconds()
	// Only tunnel time rebases. The background's established animation keeps
	// its phase; birth times use the same epoch as this small shader clock.
	epoch := math.Floor(now/60) * 60
	s.shader.Uniforms["tunnelTime"] = float32(now - epoch)
	frame := s.tunnelFrame()
	if !frame.valid() {
		return
	}
	for i, f := range t.slots {
		if f == nil {
			continue
		}
		if now >= f.born+f.duration || f.pose(now, frame).outside(frame) {
			t.slots[i] = nil
			s.clearTraveller(i)
		} else {
			s.shader.Uniforms[fmt.Sprintf("traveller%dBorn", i)] = float32(f.born - epoch)
		}
	}
	slot := -1
	for i, f := range t.slots {
		if f == nil {
			slot = i
			break
		}
	}
	repeated := false
	if t.ready != nil && len(t.usable) >= 3 {
		for _, f := range t.slots {
			if f != nil && s.sources[f.source].String() == s.sources[t.ready.source].String() {
				repeated = true
			}
		}
	}
	due := t.admissions == 0 || now-t.lastAdmission >= s.st.imageGap*t.gapFactor
	if t.ready != nil && slot >= 0 && due && !repeated {
		t.nextProfile(s.st.randomness)
		b := t.ready.pixels.Bounds()
		f, valid := t.chooseRoute(flight{source: t.ready.source, born: now, duration: 9 / (s.st.imageSpeed * t.profile.speed),
			curve: t.profile.curve * s.st.turnDirection, margin: t.profile.margin, aspect: float64(b.Dx()) / float64(b.Dy())}, frame)
		if valid {
			t.slots[slot] = &f
			s.installTraveller(slot, f, t.ready.pixels, epoch)
			t.ready = nil
			t.admissions++
			t.lastAdmission = now
			t.remaining--
			t.gapFactor = t.profile.gap
			t.lastBearing = f.angle
			t.lastSource = s.sources[f.source].String()
		}
	}
	if t.ready == nil && !s.previewBusy && len(s.sources) > 0 && now >= t.retryAt {
		if t.next == len(s.sources) {
			t.next = 0
			t.order = nil
			if t.cycleSuccess == 0 {
				t.retryAt = now + 2
				return
			}
			t.cycleSuccess = 0
		}
		if t.order == nil {
			s.makeTunnelOrder()
		}
		index := t.order[t.next]
		t.next++
		s.loadTunnelPreview(t, index)
	}
}

func (s *Spiral) loadTunnelPreview(t *tunnelSession, index int) {
	u := s.sources[index]
	revision := t.orderRevision
	ctx, cancel := context.WithCancel(t.ctx)
	t.decodeCancel = cancel
	s.previewBusy = true
	queue := s.ui
	s.previewWorkers.Go(func() {
		defer cancel()
		pixels, err := imaging.LoadThumbnailAtEdgeContext(ctx, u, tunnelPreviewEdge)
		if ctx.Err() != nil {
			pixels = nil
		}
		queue.Do(func() {
			s.previewBusy = false
			if s.tunnel == t && t.ctx.Err() == nil && revision == t.orderRevision {
				if err != nil {
					if !t.failed[index] {
						fyne.LogError("load spiral preview", err)
					}
					t.failed[index] = true
					delete(t.usable, u.String())
				} else {
					t.failed[index] = false
					t.usable[u.String()] = true
					t.cycleSuccess++
					t.ready = &tunnelPreview{index, pixels}
				}
			}
			s.advanceTunnel()
		})
	})
}

func (s *Spiral) installTraveller(i int, f flight, pixels image.Image, epoch float64) {
	name := fmt.Sprintf("traveller%d", i)
	s.shader.Textures[name] = pixels
	for key, value := range map[string]float64{"Active": 1, "Born": f.born - epoch,
		"Duration": f.duration, "Angle": f.angle, "Curve": f.curve, "Margin": f.margin, "Aspect": f.aspect} {
		s.shader.Uniforms[name+key] = float32(value)
	}
	s.shader.Refresh()
}

func (s *Spiral) clearTraveller(i int) {
	name := fmt.Sprintf("traveller%d", i)
	s.shader.Textures[name] = s.placeholder
	s.shader.Uniforms[name+"Active"] = 0
	s.shader.Refresh()
}
