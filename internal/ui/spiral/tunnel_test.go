package spiral

import (
	"bytes"
	"fmt"
	"image/color"
	"io"
	"math"
	"math/rand/v2"
	"os"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestTunnelSession(t *testing.T) {
	t.Run("first_preview_and_frozen_sources", func(t *testing.T) {
		s := newTestSpiral(t)
		s.st.randomness = 0
		now := time.Unix(1000, 0)
		s.now = func() time.Time { return now }
		u := uitest.TempJPEGURI(t, "first.jpg", 100, 50, color.White)
		sources := []fyne.URI{u}
		s.Show(sources)
		s.win.Resize(fyne.NewSize(800, 600))
		sources[0] = nil
		s.previewWorkers.Wait()
		s.ui.Drain()
		if s.sources[0] != u {
			t.Fatal("source membership was not copied")
		}
		if s.shader.Textures["traveller0"].Bounds().Dx() != 100 {
			t.Fatal("first ready preview did not reach the real shader")
		}
		now = now.Add(10 * time.Second)
		s.frame(10)
		if s.shader.Uniforms["traveller0Active"] != 0 {
			t.Fatal("fully exited traveller was not retired")
		}
		s.Close()
		s.Settle()
		if s.sources != nil {
			t.Fatal("close retained the snapshot")
		}
	})
}

func TestTunnelGIFPlayback(t *testing.T) {
	s := newTestSpiral(t)
	start := time.Unix(1000, 0)
	now := start
	s.now = func() time.Time { return now }
	s.st.randomness, s.st.imageSpeed = 0, .35
	path := t.TempDir() + "/animated.gif"
	if err := os.WriteFile(path, uitest.EncodeAnimatedGIF(t, 80, 40, []color.Color{color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255}}, []int{10, 30}), 0600); err != nil {
		t.Fatal(err)
	}
	s.Show([]fyne.URI{storage.NewFileURI(path)})
	s.win.Resize(fyne.NewSize(800, 600))
	settleTunnelPreviews(s)
	check := func(slot int, blue bool) {
		t.Helper()
		r, _, b, _ := s.shader.Textures[fmt.Sprintf("traveller%d", slot)].At(10, 10).RGBA()
		if (b > r) != blue || r+b == 0 {
			t.Fatalf("slot %d at %v: r=%d b=%d; blue=%v", slot, now.Sub(start), r, b, blue)
		}
	}
	for _, tc := range []struct {
		ms   int
		blue bool
	}{{0, false}, {99, false}, {100, true}, {399, true}, {400, false}, {1700, true}} {
		now = start.Add(time.Duration(tc.ms) * time.Millisecond)
		s.frame(0)
		check(0, tc.blue)
	}
	// Changing order must not restart the animation already in flight.
	s.st.randomOrder = true
	s.frame(0)
	settleTunnelPreviews(s)
	check(0, true)
	now = start.Add(2500 * time.Millisecond)
	s.frame(0)
	settleTunnelPreviews(s)
	check(0, true)
	check(1, false) // Second admission has its own frame-zero origin.
	now = start.Add(2600 * time.Millisecond)
	s.frame(0)
	check(1, true)
	s.Close()
	s.Settle()
	for i := range 3 {
		if s.shader.Textures[fmt.Sprintf("traveller%d", i)] != s.placeholder {
			t.Fatal("close retained GIF texture")
		}
	}
}

func settleTunnelPreviews(s *Spiral) {
	for {
		s.previewWorkers.Wait()
		if !s.ui.Drain() {
			return
		}
	}
}

func newestTraveller(s *Spiral) *flight {
	var newest *flight
	for _, f := range s.tunnel.slots {
		if f != nil && (newest == nil || f.born > newest.born) {
			newest = f
		}
	}
	return newest
}

func TestTunnelDirection(t *testing.T) {
	s := newTestSpiral(t)
	now := time.Unix(1000, 0)
	s.now = func() time.Time { return now }
	s.st.randomness = 0
	s.Show(uitest.TempDirJPEGURIs(t, "a.jpg"))
	s.win.Resize(fyne.NewSize(800, 600))
	settleTunnelPreviews(s)
	for range 12 {
		s.handleKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
	}
	s.handleKey(&fyne.KeyEvent{Name: fyne.KeyRight}) // Reverse, then pause before another admission.
	if math.Abs(s.st.speed()) > 1e-8 {
		t.Fatal("expected paused spiral")
	}
	now = now.Add(10 * time.Second)
	s.frame(10)
	settleTunnelPreviews(s)
	if f := newestTraveller(s); f == nil || f.curve >= 0 {
		t.Fatalf("paused spiral lost its last direction: %+v", f)
	}
}

func TestTunnelOrder(t *testing.T) {
	s := newTestSpiral(t)
	now := time.Unix(1000, 0)
	s.now = func() time.Time { return now }
	s.st.randomness, s.st.randomOrder = 0, true
	s.Show(uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg"))
	s.win.Resize(fyne.NewSize(800, 600))
	seen, previous := map[int]bool{}, -1
	for i := range 24 {
		settleTunnelPreviews(s)
		f := newestTraveller(s)
		if f == nil || seen[f.source] || (i%4 == 0 && f.source == previous) {
			t.Fatalf("invalid shuffled cycle at %d: %+v, seen %v", i, f, seen)
		}
		seen[f.source] = true
		previous = f.source
		if i%4 == 3 {
			clear(seen)
		}
		now = now.Add(10 * time.Second)
		s.frame(10)
	}
	settleTunnelPreviews(s)
	active, admissions, last := s.tunnel.slots, s.tunnel.admissions, s.tunnel.lastAdmission
	s.st.randomOrder = false
	s.resetTunnelOrder()
	s.frame(0)
	settleTunnelPreviews(s)
	if s.tunnel.slots != active || s.tunnel.admissions != admissions || s.tunnel.lastAdmission != last {
		t.Fatal("order change restarted admitted flights or timing")
	}
	if s.tunnel.ready == nil || s.tunnel.ready.source != 0 {
		t.Fatal("Main order did not restart pending source at zero")
	}
}

func TestTunnelRepeatedIdentity(t *testing.T) {
	s := newTestSpiral(t)
	now := time.Unix(1000, 0)
	s.now = func() time.Time { return now }
	s.st.randomness = 0
	uris := uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg")
	s.Show([]fyne.URI{uris[0], uris[0], uris[1]})
	s.win.Resize(fyne.NewSize(800, 600))
	s.tunnel.rng = rand.New(rand.NewPCG(2, 3))
	s.st.randomOrder = true
	s.resetTunnelOrder()
	previous := ""
	seen := map[int]bool{}
	for i := range 90 {
		settleTunnelPreviews(s)
		f := newestTraveller(s)
		if f == nil || seen[f.source] {
			t.Fatalf("cycle lost or repeated a source index at %d: %+v", i, f)
		}
		identity := s.sources[f.source].String()
		if i%3 == 0 && identity == previous {
			t.Fatalf("cycle %d repeated the previous URI despite another available source", i/3)
		}
		seen[f.source] = true
		previous = identity
		if i%3 == 2 {
			clear(seen)
		}
		now = now.Add(10 * time.Second)
		s.frame(10)
	}
}

func TestTunnelBackpressure(t *testing.T) {
	s := newTestSpiral(t)
	now := time.Unix(1000, 0)
	s.now = func() time.Time { return now }
	s.st.randomness, s.st.imageGap, s.st.imageSpeed = 0, .75, .35
	s.Show(uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg"))
	s.win.Resize(fyne.NewSize(800, 600))
	settleTunnelPreviews(s)
	for range 2 {
		now = now.Add(time.Second)
		s.frame(1)
		settleTunnelPreviews(s)
	}
	if s.tunnel.admissions != 3 {
		t.Fatal("expected three occupied slots")
	}
	now = now.Add(time.Second)
	s.frame(1)
	settleTunnelPreviews(s)
	if s.tunnel.admissions != 3 {
		t.Fatal("admitted above capacity")
	}
	// A late UI callback retires old flights but admits only one, immediately.
	now = now.Add(30 * time.Second)
	s.frame(30)
	settleTunnelPreviews(s)
	if f := newestTraveller(s); s.tunnel.admissions != 4 || f == nil || f.born != 33 {
		t.Fatal("capacity release added a pause or catch-up burst")
	}
	s.st.imageGap = 6
	now = now.Add(time.Second)
	s.frame(1)
	settleTunnelPreviews(s)
	if s.tunnel.admissions != 4 {
		t.Fatal("live gap change did not update the pending deadline")
	}
	s.st.imageGap = .75
	s.frame(0)
	settleTunnelPreviews(s)
	if s.tunnel.admissions != 5 {
		t.Fatal("shortened gap was not immediately eligible")
	}
}

func TestTunnelCancelledDecode(t *testing.T) {
	s := newTestSpiral(t)
	u := uitest.TempJPEGURI(t, "old.jpg", 100, 50, color.White)
	data, err := os.ReadFile(u.Path())
	if err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	held := uitest.ReaderURI(u, func() (io.ReadCloser, error) {
		close(started)
		<-release
		return io.NopCloser(bytes.NewReader(data)), nil
	})
	s.Show([]fyne.URI{held})
	<-started
	s.Close()
	s.running.Wait() // Frame cancellation must not depend on UI delivery or I/O.
	fresh := uitest.TempJPEGURI(t, "fresh.jpg", 64, 32, color.Black)
	s.Show([]fyne.URI{fresh})
	s.win.Resize(fyne.NewSize(800, 600))
	if !s.previewBusy || s.tunnel.admissions != 0 {
		t.Fatal("reopen bypassed the occupied single decode lane")
	}
	close(release)
	settleTunnelPreviews(s)
	if got := s.shader.Textures["traveller0"].Bounds().Dx(); got != 64 {
		t.Fatalf("stale preview reached reopened window: width %d", got)
	}
	s.Close()
	s.Settle()
	for _, tex := range s.shader.Textures {
		if tex.Bounds().Dx() != 1 {
			t.Fatal("close retained a photo texture")
		}
	}
}

func TestTunnelFailedSources(t *testing.T) {
	s := newTestSpiral(t)
	now := time.Unix(1000, 0)
	s.now = func() time.Time { return now }
	path := t.TempDir() + "/recover.jpg"
	s.Show([]fyne.URI{storage.NewFileURI(path)})
	s.win.Resize(fyne.NewSize(800, 600))
	settleTunnelPreviews(s)
	if s.tunnel.admissions != 0 || s.tunnel.retryAt != 2 || s.previewBusy {
		t.Fatal("all-failed cycle did not back off")
	}
	u := uitest.TempJPEGURI(t, "good.jpg", 100, 50, color.White)
	data, err := os.ReadFile(u.Path())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	now = now.Add(time.Second)
	s.frame(1)
	settleTunnelPreviews(s)
	if s.tunnel.admissions != 0 {
		t.Fatal("retried before the backoff deadline")
	}
	now = now.Add(time.Second)
	s.frame(1)
	settleTunnelPreviews(s)
	if s.tunnel.admissions != 1 {
		t.Fatal("restored source was not retried")
	}
}

type tunnelNoticeQueue struct {
	uitest.UIQueue
	submitted chan struct{}
}

func (q *tunnelNoticeQueue) Do(f func()) {
	q.UIQueue.Do(f)
	q.submitted <- struct{}{}
}

func TestTunnelQueuedFrame(t *testing.T) {
	s := newTestSpiral(t)
	q := &tunnelNoticeQueue{submitted: make(chan struct{}, 8)}
	s.SetUIQueue(q)
	s.frameInterval = time.Millisecond
	s.Show(nil)
	select {
	case <-q.submitted:
	case <-time.After(5 * time.Second):
		t.Fatal("frame never queued")
	}
	s.Close()
	s.running.Wait()
	if q.Len() != 1 {
		t.Fatalf("frame queue grew without an acknowledgement: %d", q.Len())
	}
	s.frameInterval = time.Minute
	s.Show(nil)
	s.st.toggleFollow()
	s.st.setMouse(50, 70)
	x, y := s.st.centerOffset()
	q.Drain()
	if xx, yy := s.st.centerOffset(); xx != x || yy != y {
		t.Fatal("closed session's frame moved the new session")
	}
}

func TestTunnelClockAndResize(t *testing.T) {
	s := newTestSpiral(t)
	now := time.Unix(1000, 0)
	s.now = func() time.Time { return now }
	s.st.randomness = 0
	s.Show(uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg"))
	s.win.Resize(fyne.NewSize(800, 600))
	settleTunnelPreviews(s)
	// Rebase after days, then straddle a minute boundary with a live flight.
	now = now.Add(48*time.Hour + 59*time.Second)
	s.frame(1)
	settleTunnelPreviews(s)
	current := newestTraveller(s)
	if current == nil {
		t.Fatal("no flight admitted after the long interval")
	}
	f := *current
	now = now.Add(1500 * time.Millisecond)
	s.win.Resize(fyne.NewSize(500, 900))
	s.st.setMouse(250, 450)
	s.st.toggleFollow()
	s.frame(.01)
	if current := newestTraveller(s); current == nil || current.born != f.born || current.duration != f.duration {
		t.Fatal("resize/Follow restarted the live flight")
	}
	if got := s.shader.Uniforms["tunnelTime"]; got != .5 {
		t.Fatalf("shader clock did not rebase: %g", got)
	}
	for i, active := range s.tunnel.slots {
		if active == nil {
			continue
		}
		birth := s.shader.Uniforms[fmt.Sprintf("traveller%dBorn", i)]
		if math.Abs(float64(s.shader.Uniforms["tunnelTime"]-birth)-1.5) > 1e-6 {
			t.Fatal("GPU and scheduler clocks disagree")
		}
	}
}

func TestTunnelResizeRecovery(t *testing.T) {
	for _, follow := range []bool{false, true} {
		t.Run(fmt.Sprintf("follow=%v", follow), func(t *testing.T) {
			s := newTestSpiral(t)
			now := time.Unix(1000, 0)
			s.now = func() time.Time { return now }
			s.st.randomness = 0
			s.Show(uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg"))
			s.win.Resize(fyne.NewSize(1600, 1200))
			s.st.toggleFollow()
			s.st.setMouse(1520, 1140)
			s.frame(0)
			settleTunnelPreviews(s)
			if f := newestTraveller(s); f == nil || f.source != 0 {
				t.Fatal("first source was not admitted before resize")
			}
			if !follow {
				s.st.toggleFollow()
			}
			s.win.Resize(fyne.NewSize(400, 300))
			now = now.Add(10 * time.Second)
			s.frame(10)
			settleTunnelPreviews(s)
			f := newestTraveller(s)
			if f == nil || f.source != 1 || f.born != 10 {
				t.Fatalf("resize stalled the next source: %+v", f)
			}
			frame := s.tunnelFrame()
			if !f.pose(f.born, frame).inside(frame) {
				t.Fatal("recovered arrival starts outside the resized viewport")
			}
			x, y := s.st.centerOffset()
			assertUniform(t, s, "centerOffsetX", float32(x))
			assertUniform(t, s, "centerOffsetY", float32(y))
		})
	}
}

func TestTunnelFlight(t *testing.T) {
	for _, size := range []float64{.5, 1, 2} {
		for _, aspect := range []float64{0.15, 0.5, 1, 2, 8} {
			for _, frame := range []tunnelFrame{{800, 600, 400, 300}, {600, 900, 300, 450}} {
				for _, angle := range []float64{0, 0.7, 2, 4, 5.5} {
					f := flight{born: 0, duration: 9, angle: angle, curve: .35, margin: .7, aspect: aspect, size: size}
					unit := math.Min(frame.width, frame.height)
					entry := f.pose(0, frame)
					if math.Abs(entry.opacity-.15) > 1e-6 || math.Max(entry.width, entry.height) > .100001*unit*size {
						t.Fatalf("bad entry: %+v", entry)
					}
					for step := range 101 {
						p := f.pose(float64(step)*.09, frame)
						dx := math.Max(math.Abs(p.x-frame.cx)-p.width/2, 0)
						dy := math.Max(math.Abs(p.y-frame.cy)-p.height/2, 0)
						if math.Hypot(dx, dy) < .08*unit || p.opacity > .850001 || p.opacity < .149999 {
							t.Fatalf("unsafe flight at step %d: %+v", step, p)
						}
					}
					if !f.pose(9, frame).outside(frame) {
						t.Fatal("flight ends on screen")
					}
				}
			}
		}
	}
}

func TestTunnelStream(t *testing.T) {
	s := newTestSpiral(t)
	s.st.randomness = 0
	now := time.Unix(1000, 0)
	s.now = func() time.Time { return now }
	s.Show(uitest.TempDirJPEGURIs(t, "a.jpg", "b.jpg", "c.jpg", "d.jpg"))
	s.win.Resize(fyne.NewSize(800, 600))
	settlePreviews := func() {
		for {
			s.previewWorkers.Wait()
			if !s.ui.Drain() {
				break
			}
		}
	}
	settlePreviews()
	for i := 1; i < 12; i++ {
		now = now.Add(3 * time.Second)
		s.frame(3)
		settlePreviews()
		var newest *flight
		for _, f := range s.tunnel.slots {
			if f != nil && (newest == nil || f.born > newest.born) {
				newest = f
			}
		}
		if newest == nil || newest.source != i%4 || newest.born != float64(i*3) {
			t.Fatalf("entry %d: newest = %+v; want source %d at %d", i, newest, i%4, i*3)
		}
	}
}

func TestTunnelProfiles(t *testing.T) {
	s := newTestSpiral(t)
	now := time.Unix(1000, 0)
	s.now = func() time.Time { return now }
	s.st.randomness = 100
	s.Show(uitest.TempDirJPEGURIs(t, "a.jpg"))
	s.win.Resize(fyne.NewSize(800, 600))
	s.tunnel.rng = rand.New(rand.NewPCG(4, 5))
	var previous flowProfile
	run := 0
	for i := range 60 {
		s.previewWorkers.Wait()
		s.ui.Drain()
		f := s.tunnel.profile
		if f.speed < .7 || f.speed > 1.3 || f.gap < .7 || f.gap > 1.3 || f.curve < .23 || f.curve > .47 {
			t.Fatalf("profile out of range: %+v", f)
		}
		if i > 0 && f != previous {
			if run < 3 || run > 7 {
				t.Fatalf("batch length=%d", run)
			}
			if math.Abs(f.speed-previous.speed) > .100001 || math.Abs(f.curve-previous.curve) > .040001 {
				t.Fatal("abrupt profile jump")
			}
			run = 0
		}
		run++
		previous = f
		now = now.Add(15 * time.Second)
		s.frame(15)
	}
}
