package exifwin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/color"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestMetadataRemovalLeavesQueuedWindowInputResponsive(t *testing.T) {
	app, host := gpsApp(t)
	source, _ := host.DisplayedFile()
	synctest.Test(t, func(t *testing.T) {
		w := newTestWindow(t, app, host)
		w.Show()
		w.Settle()
		entered, release := make(chan struct{}), make(chan struct{})
		w.stripFile = func(ctx context.Context, u fyne.URI) (imaging.WriteResult, error) {
			close(entered)
			<-release
			return imaging.StripJPEGMetadataContext(ctx, u)
		}
		actions := make(chan func(), 2)
		uiDone := make(chan struct{})
		go func() {
			defer close(uiDone)
			for action := range actions {
				action()
			}
		}()
		actions <- func() { w.performStrip(source) }
		<-entered
		interaction := make(chan struct{})
		actions <- func() { w.Window().Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyRight}); close(interaction) }
		synctest.Wait()
		select {
		case <-interaction:
		default:
			t.Error("queued window input blocked behind metadata removal")
		}
		close(release)
		<-interaction
		close(actions)
		<-uiDone
		w.Settle()
		if len(host.steps) != 1 || host.after != 1 {
			t.Errorf("navigation/removal effects = %v/%d", host.steps, host.after)
		}
		w.Window().Close()
		w.Settle()
	})
}

func TestMetadataRemovalCancellationDistinguishesCommittedDiskChanges(t *testing.T) {
	for _, committed := range []bool{false, true} {
		for _, change := range []string{"navigate", "close", "reopen", "reset", "stop"} {
			t.Run(fmt.Sprintf("committed=%v/change=%s", committed, change), func(t *testing.T) {
				app, host := gpsApp(t)
				source, _ := host.DisplayedFile()
				before, err := os.ReadFile(source.Path())
				if err != nil {
					t.Fatal(err)
				}
				other := uitest.TempJPEGURI(t, "other.jpg", 8, 8, color.White)
				w := newTestWindow(t, app, host)
				w.Show()
				w.Settle()
				entered, release := make(chan struct{}), make(chan struct{})
				w.stripFile = func(ctx context.Context, u fyne.URI) (imaging.WriteResult, error) {
					var result imaging.WriteResult
					var err error
					if committed {
						result, err = imaging.StripJPEGMetadataContext(ctx, u)
					}
					close(entered)
					<-release
					if !committed {
						result, err = imaging.StripJPEGMetadataContext(ctx, u)
					}
					return result, err
				}
				w.performStrip(source)
				<-entered
				done := w.MutationDone().Current()
				switch change {
				case "navigate":
					host.current = func() (fyne.URI, bool) { return other, true }
					w.Refresh()
					settleMetadata(w)
				case "close", "reopen":
					w.Window().Close()
					if change == "reopen" {
						host.current = func() (fyne.URI, bool) { return other, true }
						w.Show()
						settleMetadata(w)
					}
				case "reset":
					w.Invalidate()
					host.current = func() (fyne.URI, bool) { return nil, false }
					w.Refresh()
				case "stop":
					w.Stop()
				}
				ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
				if err := done.Wait(ctx); err == nil {
					t.Error("removal finished before its underlying operation returned")
				}
				cancel()
				close(release)
				w.stripWork.workers.Wait()
				if host.after != 0 || len(host.toasts) != 0 {
					t.Errorf("worker applied host effects before UI: after=%d, toasts=%v", host.after, host.toasts)
				}
				w.Settle()
				if err := done.Wait(context.Background()); err != nil {
					t.Fatal(err)
				}
				after, err := os.ReadFile(source.Path())
				if err != nil {
					t.Fatal(err)
				}
				if committed {
					if bytes.Equal(before, after) || !imaging.ReadMetadata(after).Empty() {
						t.Error("accomplished removal disappeared after cancellation")
					}
				} else if !bytes.Equal(before, after) {
					t.Error("cancelled uncommitted removal changed its source")
				}
				wantNotify := 0
				if committed && change != "stop" {
					wantNotify = 1
				}
				if host.after != wantNotify {
					t.Errorf("host notifications = %d, want %d", host.after, wantNotify)
				}
				if len(host.toasts) != 0 {
					t.Errorf("stale removal showed toasts: %v", host.toasts)
				}
				if w.Open() {
					if change != "stop" && (w.Location().Visible() || northHolds(w.north, w.stripBar)) {
						t.Error("old removal restored GPS/action on another or empty source")
					}
					w.Window().Close()
				}
			})
		}
	}
}

func TestMetadataRemovalBusyErrorDeliveryAndRetry(t *testing.T) {
	app, host := gpsApp(t)
	source, _ := host.DisplayedFile()
	w := newTestWindow(t, app, host)
	w.Show()
	w.Settle()
	defer w.Window().Close()
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	var calls atomic.Int32
	w.stripFile = func(_ context.Context, _ fyne.URI) (imaging.WriteResult, error) {
		calls.Add(1)
		once.Do(func() { close(entered) })
		<-release
		return imaging.WriteResult{}, errors.New("remove failed")
	}
	w.performStrip(source)
	<-entered
	handle := w.MutationDone().Current()
	w.performStrip(source)
	if handle != w.MutationDone().Current() {
		t.Error("repeated removal replaced the active completion")
	}
	if !w.StripButton().Disabled() {
		t.Error("removal action remained enabled during its write")
	}
	close(release)
	w.stripWork.workers.Wait()
	if calls.Load() != 1 {
		t.Errorf("overlapping removals=%d", calls.Load())
	}
	if len(host.toasts) != 0 {
		t.Error("removal error was presented by a worker")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	if err := handle.Wait(ctx); err == nil {
		t.Error("removal completed before its queued failure")
	}
	cancel()
	w.Settle()
	if len(host.toasts) != 1 || host.after != 0 {
		t.Errorf("failure effects = %v, after=%d", host.toasts, host.after)
	}
	if w.StripButton().Disabled() {
		t.Error("failed removal cannot be retried")
	}
	w.stripFile = imaging.StripJPEGMetadataContext
	w.performStrip(source)
	w.Settle()
	if len(host.toasts) != 2 || host.after != 1 {
		t.Errorf("retry effects = %v, after=%d", host.toasts, host.after)
	}
	if northHolds(w.north, w.stripBar) {
		t.Error("successful retry left removal in the panel")
	}
}
