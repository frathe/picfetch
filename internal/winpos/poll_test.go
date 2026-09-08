package winpos

import (
	"testing"
	"testing/synctest"
	"time"

	"fyne.io/fyne/v2/test"
)

// The fyne test driver's windows are not driver.NativeWindow, so there is
// no handle for a reading to come from - Poll must degrade to "no goroutine
// at all" there rather than spinning a ticker that can only ever fail. That
// is what keeps every headless test in internal/ui (and in this package's
// own consumers) from carrying a poller goroutine behind it.

func TestPoll_NonNativeWindowRecordsNothing(t *testing.T) {
	win := test.NewWindow(nil)
	defer win.Close()

	var tr Tracker
	stop := Poll(win, &tr, nil)
	if stop == nil {
		t.Fatal("Poll should never return a nil handle")
	}
	stop.Wait() // Non-native handles are complete even before Stop.
	stop.Stop()

	if _, _, ok := tr.Get(); ok {
		t.Error("Poll recorded a position for a window with no native handle to read one from")
	}
}

func TestPoll_StopIsSafeToCallOnANonNativeWindow(t *testing.T) {
	win := test.NewWindow(nil)
	defer win.Close()

	var tr Tracker
	Poll(win, &tr, func() bool { return true }).Stop()
}

// PollAt is the general form Poll is built on, and inherits the same
// degradation: no native handle means no goroutine and no callback, rather
// than a ticker spinning on a read that can only ever fail.

func TestPollAt_NonNativeWindowNeverCallsBack(t *testing.T) {
	win := test.NewWindow(nil)
	defer win.Close()

	called := make(chan struct{}, 1)
	stop := PollAt(win, time.Millisecond, nil, func(x, y int) {
		select {
		case called <- struct{}{}:
		default:
		}
	})
	if stop == nil {
		t.Fatal("PollAt should never return a nil handle")
	}
	stop.Wait()
	defer stop.Stop()

	select {
	case <-called:
		t.Error("PollAt called back for a window with no native handle to read a position from")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestPollAt_StopIsSafeToCallOnANonNativeWindow(t *testing.T) {
	win := test.NewWindow(nil)
	defer win.Close()

	PollAt(win, time.Millisecond, func() bool { return true }, func(_, _ int) {}).Stop()
}

func TestPoller_StopDiscardsQueuedReadWithoutDrainingUI(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ticks := make(chan time.Time, 1)
		queued := make(chan func(), 1)
		reads, published := 0, 0
		p := startPoll(ticks, func() {}, nil, func(_, _ int) { published++ }, func() (int, int, bool) { reads++; return 1, 2, true }, func(fn func()) { queued <- fn })
		ticks <- time.Now()
		callback := <-queued
		p.Stop()
		p.Stop()
		synctest.Wait()
		select {
		case <-p.Done():
		default:
			t.Error("stopped poller still needs queued UI to exit")
		}
		callback()
		p.Wait()
		if reads != 0 || published != 0 {
			t.Errorf("stopped read=%d publications=%d", reads, published)
		}
	})
}

func TestPoller_StopDuringNativeReadWaitsForReadWithoutPublishing(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ticks := make(chan time.Time, 1)
		queued := make(chan func(), 1)
		entered, release := make(chan struct{}), make(chan struct{})
		published := 0
		p := startPoll(ticks, func() {}, nil, func(_, _ int) { published++ }, func() (int, int, bool) { close(entered); <-release; return 1, 2, true }, func(fn func()) { queued <- fn })
		ticks <- time.Now()
		callback := <-queued
		go callback()
		<-entered
		p.Stop()
		waited := make(chan struct{})
		go func() { p.Wait(); close(waited) }()
		synctest.Wait()
		select {
		case <-p.Done():
			t.Error("worker completed while native read was still active")
		default:
		}
		select {
		case <-waited:
			t.Error("Wait returned while native read was still active")
		default:
		}
		close(release)
		<-waited
		if published != 0 {
			t.Error("cancelled native result reached the stopped target")
		}
	})
}

func TestPoller_StopWhileDispatchHeldExposesWorkerCompletion(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ticks := make(chan time.Time, 1)
		entered, release := make(chan struct{}), make(chan struct{})
		queued := make(chan func(), 1)
		reads := 0
		p := startPoll(ticks, func() {}, nil, func(_, _ int) {}, func() (int, int, bool) { reads++; return 0, 0, false }, func(fn func()) { close(entered); <-release; queued <- fn })
		ticks <- time.Now()
		<-entered
		p.Stop()
		synctest.Wait()
		select {
		case <-p.Done():
			t.Error("worker completed before dispatch returned")
		default:
		}
		close(release)
		callback := <-queued
		synctest.Wait()
		select {
		case <-p.Done():
		default:
			t.Error("stopped dispatch still awaited its queued callback")
		}
		callback()
		p.Wait()
		if reads != 0 {
			t.Error("queued callback read after cancellation")
		}
	})
}

func TestPoller_StopBeforeTickPreventsAdmission(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ticks := make(chan time.Time, 1)
		queued := make(chan func(), 1)
		p := startPoll(ticks, func() {}, nil, func(_, _ int) {}, func() (int, int, bool) { t.Error("unexpected read"); return 0, 0, false }, func(fn func()) { queued <- fn })
		p.Stop()
		p.Wait()
		ticks <- time.Now()
		synctest.Wait()
		if len(queued) != 0 {
			t.Error("stopped worker queued a read")
		}
	})
}

func TestPoller_QueuedReadChecksStopWhileDispatchIsHeld(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ticks := make(chan time.Time, 1)
		queued := make(chan func(), 1)
		release := make(chan struct{})
		reads := 0
		p := startPoll(ticks, func() {}, nil, func(_, _ int) {},
			func() (int, int, bool) { reads++; return 1, 2, true },
			func(fn func()) { queued <- fn; <-release })
		ticks <- time.Now()
		callback := <-queued
		p.Stop()
		callback() // The worker cannot mark it discarded until dispatch returns.
		if reads != 0 {
			t.Error("queued callback read a stopped target before worker cancellation")
		}
		close(release)
		p.Wait()
	})
}

func TestPoller_ChecksSkipOnUIAndPreservesSuccessfulReads(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		ticks := make(chan time.Time, 1)
		queued := make(chan func(), 1)
		skip, readOK, reads, published := false, true, 0, 0
		p := startPoll(ticks, func() {}, func() bool { return skip }, func(x, y int) {
			if x != 11 || y != 22 {
				t.Error("wrong position")
			}
			published++
		}, func() (int, int, bool) { reads++; return 11, 22, readOK }, func(fn func()) { queued <- fn })
		ticks <- time.Now()
		callback := <-queued
		skip = true
		callback()
		synctest.Wait()
		if reads != 0 {
			t.Error("skip was read before the queued UI decision")
		}
		skip = false
		for _, ok := range []bool{false, true} {
			readOK = ok
			ticks <- time.Now()
			(<-queued)()
			synctest.Wait()
		}
		p.Stop()
		p.Wait()
		if reads != 2 || published != 1 {
			t.Errorf("reads=%d publications=%d, want 2/1", reads, published)
		}
	})
}
