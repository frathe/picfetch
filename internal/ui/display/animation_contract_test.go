package display_test

import (
	"context"
	"image"
	"testing"
	"testing/synctest"
	"time"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/display"
	"github.com/frathe/picfetch/internal/uitest"
)

func animationContract(t *testing.T) {
	t.Run("brief capture restarts delay", animationFreshDelayContract)
	synctest.Test(t, func(t *testing.T) {
		queue := &uitest.UIQueue{}
		ticks := make(chan time.Time, 1)
		delays := make(chan time.Duration, 8)
		f := newPresentation(t, display.Config{Queue: queue, AnimationAfter: func(d time.Duration) <-chan time.Time { delays <- d; return ticks }})
		defer func() { f.Stop(); f.Wait(); queue.Drain() }()
		first, second := image.NewNRGBA(image.Rect(0, 0, 2, 3)), image.NewNRGBA(image.Rect(0, 0, 2, 3))
		loaded := &imaging.LoadedImage{Frames: []image.Image{first, second}, Delays: []time.Duration{time.Second, 2 * time.Second}}
		source := storage.NewFileURI("/animation.gif")
		if _, _, ok := f.CaptureStable(); ok {
			t.Fatal("empty presentation allowed stable capture")
		}
		f.Present(source, loaded, false)
		synctest.Wait()
		expectDelay(t, delays, time.Second)
		ticks <- time.Time{}
		synctest.Wait()
		if queue.Len() != 1 || f.Surface().Image != first {
			t.Fatal("frame application was not deferred")
		}
		select {
		case <-delays:
			t.Fatal("new delay started before frame acknowledgement")
		default:
		}
		queue.Drain()
		synctest.Wait()
		expectDelay(t, delays, 2*time.Second)
		if f.Surface().Image != second || f.AppliedFrames() != 2 {
			t.Fatal("acknowledged frame was not published")
		}
		ordinary, ok := f.Capture()
		if !ok || ordinary.Pixels != second {
			t.Fatal("ordinary capture lost the displayed frame")
		}
		ticks <- time.Time{}
		synctest.Wait()
		captured, release, ok := f.CaptureStable()
		if !ok || captured.Pixels != second {
			t.Fatal("stable capture did not retain displayed frame")
		}
		if _, _, ok := f.CaptureStable(); ok {
			t.Fatal("second acquisition stole the pause")
		}
		queue.Drain()
		synctest.Wait()
		if f.Surface().Image != captured.Pixels {
			t.Fatal("queued frame escaped stable capture")
		}
		select {
		case <-delays:
			t.Fatal("paused playback scheduled another delay")
		default:
		}
		release()
		synctest.Wait()
		expectDelay(t, delays, 2*time.Second)
		ticks <- time.Time{}
		synctest.Wait()
		_, secondRelease, ok := f.CaptureStable()
		if !ok {
			t.Fatal("released pause cannot be reacquired")
		}
		release() // old release cannot resume this acquisition
		queue.Drain()
		synctest.Wait()
		if f.Surface().Image != second {
			t.Fatal("stale release resumed a later acquisition")
		}
		secondRelease()
		synctest.Wait()
		expectDelay(t, delays, 2*time.Second)
		f.RotateBy(1)
		ticks <- time.Time{}
		synctest.Wait()
		queue.Drain()
		synctest.Wait()
		expectDelay(t, delays, time.Second)
		if f.Surface().Image.Bounds().Size() != image.Pt(3, 2) {
			t.Fatal("subsequent frame lost rotation")
		}
		ticks <- time.Time{}
		synctest.Wait()
		retired := f.AnimationDone()
		f.CancelRequest()
		f.Wait() // cancellation must not require draining the queued frame
		if err := retired.Wait(context.Background()); err != nil {
			t.Fatal(err)
		}
		f.Present(source, &imaging.LoadedImage{Frames: []image.Image{first}}, false)
		writes := f.AppliedFrames()
		queue.Drain()
		if f.Surface().Image != first || f.AppliedFrames() != writes {
			t.Fatal("retired queued frame changed reopened content")
		}
		f.Present(source, loaded, false)
		_, oldRelease, ok := f.CaptureStable()
		if !ok {
			t.Fatal("capture failed after reopening")
		}
		f.Present(source, loaded, false)
		_, currentRelease, ok := f.CaptureStable()
		if !ok {
			t.Fatal("retired presentation retained its pause")
		}
		oldRelease()
		if _, _, ok := f.CaptureStable(); ok {
			t.Fatal("retired presentation release resumed a newer pause")
		}
		currentRelease()
	})
}

func animationFreshDelayContract(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		type timer struct {
			delay time.Duration
			fire  chan time.Time
		}
		timers := make(chan timer, 8)
		queue := &uitest.UIQueue{}
		f := newPresentation(t, display.Config{Queue: queue, AnimationAfter: func(delay time.Duration) <-chan time.Time {
			fire := make(chan time.Time, 1)
			timers <- timer{delay, fire}
			return fire
		}})
		first, second := image.NewNRGBA(image.Rect(0, 0, 2, 3)), image.NewNRGBA(image.Rect(0, 0, 2, 3))
		f.Present(storage.NewFileURI("/brief.gif"), &imaging.LoadedImage{Frames: []image.Image{first, second}, Delays: []time.Duration{time.Second, 2 * time.Second}}, false)
		synctest.Wait()
		old := <-timers
		_, release, ok := f.CaptureStable()
		if !ok {
			t.Fatal("capture failed")
		}
		release() // End before playback has an opportunity to observe the pause.
		synctest.Wait()
		var fresh timer
		select {
		case fresh = <-timers:
		case <-time.After(time.Nanosecond):
			t.Fatal("brief capture resumed the old deadline instead of a fresh delay")
		}
		if fresh.delay != time.Second {
			t.Fatal("resume changed the current frame delay")
		}
		old.fire <- time.Time{}
		synctest.Wait()
		queue.Drain()
		synctest.Wait()
		if f.Surface().Image != first {
			t.Fatal("retired deadline advanced playback")
		}
		fresh.fire <- time.Time{}
		synctest.Wait()
		_, release, ok = f.CaptureStable()
		if !ok {
			t.Fatal("queued capture failed")
		}
		release()
		queue.Drain()
		synctest.Wait()
		if f.Surface().Image != first {
			t.Fatal("a frame queued before capture escaped its fresh-delay rule")
		}
	})
}

func expectDelay(t *testing.T, delays <-chan time.Duration, want time.Duration) {
	t.Helper()
	select {
	case got := <-delays:
		if got != want {
			t.Fatalf("delay = %v, want %v", got, want)
		}
	default:
		t.Fatal("playback did not schedule its next delay")
	}
}
