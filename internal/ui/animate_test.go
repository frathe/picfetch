package ui

import (
	"image/color"
	"testing"
	"time"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/uitest"
)

// Root animation tests retain navigation and load integration. Playback pacing
// and stable-capture tests live at display's public contract seam.
type frameClock struct {
	ticks  chan time.Time
	parked chan struct{}
}

func newFrameClock() *frameClock {
	return &frameClock{
		ticks:  make(chan time.Time),
		parked: make(chan struct{}, 1),
	}
}

func (c *frameClock) After(_ time.Duration) <-chan time.Time {
	select {
	case c.parked <- struct{}{}:
	default:
	}
	return c.ticks
}

func (c *frameClock) waitParked(t *testing.T) {
	t.Helper()
	select {
	case <-c.parked:
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for animate to park on the frame clock")
	}
}

func (c *frameClock) tick(t *testing.T) {
	t.Helper()
	select {
	case c.ticks <- time.Time{}:
	case <-time.After(testTimeout):
		t.Fatal("timed out releasing a frame tick")
	}
}

func TestViewerShow_AnimatesGIF(t *testing.T) {
	v := newTestViewer(t)

	path := uitest.WriteTempFile(t, "anim.gif", uitest.EncodeAnimatedGIF(t, 4, 4,
		[]color.Color{color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255}},
		[]int{2, 2})) // 20ms per frame, fast enough to keep the test quick

	dropAndWait(t, v, storage.NewFileURI(path))

	// Observe actual frame delivery, then cancel and join playback so its final
	// applied count and published pixels remain fixed for the color assertion.
	waitForAnimFrame(t, v, 2)

	v.invalidateLoad()
	waitForAnimStopped(t, v)

	// Frame 0 (red) is written on odd counts (display's initial publication
	// is count 1), frame 1 (blue) on even ones - whichever count animate
	// happened to stop on, this checks the frame it left on screen actually
	// matches the data for that count instead of stale or corrupted pixels.
	n := v.display.AppliedFrames()
	wantBlue := n%2 == 0

	r, _, b, _ := v.img.Image.At(0, 0).RGBA()
	if wantBlue && b == 0 {
		t.Fatalf("expected the blue frame at animFrame=%d, got r=%d b=%d", n, r, b)
	}
	if !wantBlue && r == 0 {
		t.Fatalf("expected the red frame at animFrame=%d, got r=%d b=%d", n, r, b)
	}
}

func TestViewerShow_NavigatingAwayStopsAnimation(t *testing.T) {
	v := newTestViewer(t)
	clock := newFrameClock()

	// Write-once, before the drop: the same rule as vector.after. 10s GIF
	// delays so a missing seam cannot pass this by firing time.After on its
	// own inside testTimeout; the clock is what has to advance the frame.
	v.display.SetAnimationClock(clock.After)

	animURI := storage.NewFileURI(uitest.WriteTempFile(t, "anim.gif", uitest.EncodeAnimatedGIF(t, 4, 4,
		[]color.Color{color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255}},
		[]int{1000, 1000})))
	staticURI := uitest.TempJPEGURI(t, "static.jpg", 4, 4, color.RGBA{G: 255, A: 255})

	dropAndWait(t, v, animURI, staticURI)
	clock.waitParked(t)
	clock.tick(t)
	waitForAnimFrame(t, v, 2)
	// After() is only called again once fyne.Do has returned, so parking
	// here is the happens-before that lets this goroutine read
	// the display index (and later call ShowImage) without racing animate's
	// write under the test driver.
	clock.waitParked(t)

	if _, _, b, _ := v.img.Image.At(0, 0).RGBA(); b == 0 {
		t.Fatal("expected the blue frame on screen after one clock tick")
	}

	oldAnim := v.display.AnimationDone()

	v.ShowImage(1)
	waitUntilLoaded(t, v)

	waitHandle(t, "the superseded animation to stop", oldAnim)

	// JPEG is lossy, so a "solid green" square won't decode back to an exact
	// R=0, but green should still clearly dominate; an animation frame
	// bleeding through would show red or blue dominating instead.
	r, g, b, _ := v.img.Image.At(0, 0).RGBA()
	if g <= r || g <= b {
		t.Errorf("expected the static green image to remain displayed, got r=%d g=%d b=%d", r, g, b)
	}
}

// TestInvalidateLoad_WakesAnimateImmediately parks animate on a clock that
// never ticks and checks lifecycle cancellation wakes it immediately rather
// than waiting for the next frame.
func TestInvalidateLoad_WakesAnimateImmediately(t *testing.T) {
	v := newTestViewer(t)
	parkAnimate(v)

	animURI := storage.NewFileURI(uitest.WriteTempFile(t, "slow.gif", uitest.EncodeAnimatedGIF(t, 4, 4,
		[]color.Color{color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255}},
		[]int{2, 2})))

	dropAndWait(t, v, animURI)

	if !v.display.AnimationBegun() {
		t.Fatal("loading an animated GIF should arm the animation signal")
	}

	v.invalidateLoad()

	waitForAnimStopped(t, v)
	v.invalidateLoad() // repeated invalidation must remain safe
}
