package widgets_test

import (
	"math"
	"testing"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/ui/widgets"
)

type circleSample struct {
	point fyne.Position
	at    time.Time
}

func circleTrace(turns, direction, rx, ry float64, elapsed time.Duration) []circleSample {
	steps := int(turns * 32)
	trace := make([]circleSample, steps+1)
	for i := range trace {
		angle := direction * float64(i) * math.Pi / 16
		trace[i] = circleSample{
			point: fyne.NewPos(float32(rx*math.Cos(angle)), float32(ry*math.Sin(angle))),
			// The first angular movement, not the first stationary sample,
			// starts the deadline. The final sample can be exactly on it.
			at: time.Unix(100, 0).Add(time.Duration(i-1) * elapsed / time.Duration(steps-1)),
		}
	}
	return trace
}

func playCircle(g *widgets.CircleGesture, trace []circleSample) int {
	completed := 0
	for _, sample := range trace {
		if g.Move(sample.point, sample.at) {
			completed++
		}
	}
	return completed
}

func TestCircleGestureRecognition(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		turns, direction, rx, ry float64
		elapsed                  time.Duration
		want                     int
	}{
		{"clockwise inclusive deadline", 10, 1, 3, 3, 20 * time.Second, 1},
		{"counterclockwise", 10, -1, 3, 3, 10 * time.Second, 1},
		{"nine turns", 9, 1, 3, 3, 10 * time.Second, 0},
		{"partial tenth", 9.96875, 1, 3, 3, 10 * time.Second, 0},
		{"late tenth", 10, 1, 3, 3, 20*time.Second + time.Nanosecond, 0},
		{"oval", 10, 1, 7, 2, 10 * time.Second, 1},
		{"no outer radius", 10, -1, 1000, 500, 10 * time.Second, 1},
		{"tiny center orbit", 10, 1, .9, .9, 10 * time.Second, 0},
		{"inner boundary", 10, 1, 1, 1, 10 * time.Second, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var g widgets.CircleGesture
			if got := playCircle(&g, circleTrace(tc.turns, tc.direction, tc.rx, tc.ry, tc.elapsed)); got != tc.want {
				t.Fatalf("completed %d attempts, want %d", got, tc.want)
			}
		})
	}
}

func TestCircleGestureResetAndRetry(t *testing.T) {
	for _, cause := range []string{"explicit reset", "center", "cross center", "discontinuous angle", "reversal", "expired pause", "backwards time"} {
		t.Run(cause, func(t *testing.T) {
			var g widgets.CircleGesture
			trace := circleTrace(10, 1, 3, 3, 10*time.Second)
			if playCircle(&g, trace[:9*32+1]) != 0 {
				t.Fatal("nine turns triggered")
			}
			at := trace[9*32].at
			switch cause {
			case "explicit reset":
				g.Reset()
			case "center":
				g.Move(fyne.NewPos(0, 0), at)
			case "cross center":
				g.Move(fyne.NewPos(-3, 0), at)
			case "discontinuous angle":
				g.Move(fyne.NewPos(-2, 3), at)
			case "reversal":
				g.Move(fyne.NewPos(2.6, -1.5), at)
			case "expired pause":
				g.Move(trace[9*32].point, at.Add(time.Minute))
			case "backwards time":
				g.Move(trace[9*32].point, at.Add(-time.Minute))
			}
			if playCircle(&g, trace[9*32:]) != 0 {
				t.Fatal("break retained nine completed circles")
			}
			// Continue immediately with a fresh complete attempt. No explicit
			// Reset or cooldown may be needed after the invalidating input.
			next := circleTrace(10, 1, 3, 3, 5*time.Second)
			nextStart := next[0].at
			for i := range next {
				next[i].at = trace[len(trace)-1].at.Add(100*time.Millisecond + next[i].at.Sub(nextStart))
			}
			if playCircle(&g, next) != 1 {
				t.Fatal("fresh complete attempt after reset did not trigger")
			}
		})
	}
}

func TestCircleGestureToleranceBoundaries(t *testing.T) {
	for _, degrees := range []float64{14.9, 15.1} {
		var g widgets.CircleGesture
		trace := circleTrace(10, 1, 3, 3, 10*time.Second)
		playCircle(&g, trace[:9*32+1])
		angle := -degrees * math.Pi / 180
		g.Move(fyne.NewPos(float32(3*math.Cos(angle)), float32(3*math.Sin(angle))), trace[9*32].at)
		got := playCircle(&g, trace[9*32:])
		want := 0
		if degrees < 15 {
			want = 1
		}
		if got != want {
			t.Fatalf("%g-degree reversal: %d completions, want %d", degrees, got, want)
		}
	}
	for _, degrees := range []float64{90, 91} {
		var g widgets.CircleGesture
		completed := 0
		for i := range 41 {
			angle := float64(i) * degrees * math.Pi / 180
			if g.Move(fyne.NewPos(float32(3*math.Cos(angle)), float32(3*math.Sin(angle))), time.Unix(100, int64(i)*1000000)) {
				completed++
			}
		}
		want := 0
		if degrees == 90 {
			want = 1
		}
		if completed != want {
			t.Fatalf("%g-degree sample gaps: %d completions, want %d", degrees, completed, want)
		}
	}
	var g widgets.CircleGesture
	trace := circleTrace(10, 1, 3, 3, 10*time.Second)
	playCircle(&g, trace[:9*32+1])
	at := trace[9*32].at
	g.Move(fyne.NewPos(1.1, 0), at)
	// A 60-degree chord near the center crosses the dead zone even though
	// both endpoints are outside it and the angle is below the jump limit.
	g.Move(fyne.NewPos(.55, .95262796), at)
	if playCircle(&g, trace[9*32:]) != 0 {
		t.Fatal("unsampled center crossing retained progress")
	}
}

func TestCircleGestureWobbleAndChangingRadius(t *testing.T) {
	for _, turns := range []float64{9, 10} {
		var g widgets.CircleGesture
		trace := circleTrace(turns, 1, 3, 3, 10*time.Second)
		var wobbly []circleSample
		for i, sample := range trace {
			angle := float64(i) * math.Pi / 16
			radius := 3 + .8*math.Sin(float64(i)*.13)
			sample.point = fyne.NewPos(float32(radius*math.Cos(angle)), float32(radius*math.Sin(angle)))
			wobbly = append(wobbly, sample)
			// A backward 10-degree wobble and return must not earn progress.
			if i > 0 && i%7 == 0 {
				back := sample
				back.point = fyne.NewPos(float32(radius*math.Cos(angle-math.Pi/18)), float32(radius*math.Sin(angle-math.Pi/18)))
				wobbly = append(wobbly, back, sample)
			}
		}
		want := 0
		if turns == 10 {
			want = 1
		}
		if got := playCircle(&g, wobbly); got != want {
			t.Fatalf("%g wobbly changing-radius turns completed %d attempts, want %d", turns, got, want)
		}
	}
	var g widgets.CircleGesture
	for i := range 1000 {
		// Repeating a small arc, even with tolerated reversals, never makes
		// a complete circle. Larger cumulative reversals must reset too.
		angle := float64(i%2) * math.Pi / 18
		if g.Move(fyne.NewPos(float32(3*math.Cos(angle)), float32(3*math.Sin(angle))), time.Unix(100, int64(i))) {
			t.Fatal("repeated incomplete arcs manufactured circles")
		}
	}
}

func TestCircleGestureAttemptStartsOnMovement(t *testing.T) {
	var g widgets.CircleGesture
	trace := circleTrace(10, 1, 3, 3, 20*time.Second)
	g.Move(trace[0].point, trace[0].at.Add(-time.Hour))
	if playCircle(&g, trace) != 1 {
		t.Fatal("stationary time before angular movement consumed the attempt")
	}
	// Completion consumes the attempt; nine more turns cannot retrigger.
	next := circleTrace(10, 1, 3, 3, 10*time.Second)
	for i := range next {
		next[i].at = next[i].at.Add(time.Minute)
	}
	if playCircle(&g, next[:9*32+1]) != 0 || playCircle(&g, next[9*32+1:]) != 1 {
		t.Fatal("completion did not require a fresh ten-turn attempt")
	}
}
