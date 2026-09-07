package exifwin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/color"
	"io"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

// settleMetadata applies available read results without waiting for map HTTP
// work a result may start. Tile tests control that work independently.
func settleMetadata(w *Window) {
	w.metadata.workers.Wait()
	w.ui.Drain()
}

func TestMetadataReadDiscardsHeldSourcesAcrossPanelChanges(t *testing.T) {
	for _, change := range []string{"navigate", "same source", "close", "reopen", "no source", "stop"} {
		t.Run(change, func(t *testing.T) {
			app, host := gpsApp(t)
			file, _ := host.DisplayedFile()
			data, err := os.ReadFile(file.Path())
			if err != nil {
				t.Fatal(err)
			}
			synctest.Test(t, func(t *testing.T) {
				w := newTestWindow(t, app, host)
				entered, release := make(chan struct{}), make(chan struct{})
				var reads, closes atomic.Int32
				source := uitest.ReaderURI(file, func() (io.ReadCloser, error) {
					return uitest.ReadCloser{
						ReadFunc: func(p []byte) (int, error) {
							if reads.Add(1) == 1 {
								close(entered)
								<-release
								return copy(p, data[:1]), nil
							}
							return copy(p, data[1:]), io.EOF
						},
						CloseFunc: func() error { closes.Add(1); return nil },
					}, nil
				})
				host.current = func() (fyne.URI, bool) { return source, true }
				w.Show()
				<-entered
				old := w.MetadataDone().Current()
				oldDone := make(chan struct{})
				go func() { _ = old.Wait(context.Background()); close(oldDone) }()

				// A replacement read stays held while the old operation exits,
				// including a reread of exactly the same source identity.
				freshEntered, freshRelease := make(chan struct{}), make(chan struct{})
				fresh := uitest.ReaderURI(file, func() (io.ReadCloser, error) {
					close(freshEntered)
					<-freshRelease
					return io.NopCloser(bytes.NewReader(data)), nil
				})
				switch change {
				case "navigate", "same source":
					if change == "navigate" {
						fresh = uitest.ReaderURI(uitest.TempJPEGURI(t, "other.jpg", 8, 8, color.White), func() (io.ReadCloser, error) {
							close(freshEntered)
							<-freshRelease
							return nil, errors.New("new source unavailable")
						})
					}
					host.current = func() (fyne.URI, bool) { return fresh, true }
					w.Refresh()
				case "close", "reopen":
					w.Window().Close()
					if change == "reopen" {
						host.current = func() (fyne.URI, bool) { return fresh, true }
						w.Show()
					}
				case "no source":
					host.current = func() (fyne.URI, bool) { return nil, false }
					w.Refresh()
				case "stop":
					w.Stop()
					w.Refresh()
					w.Show()
				}
				freshRead := change == "navigate" || change == "same source" || change == "reopen"
				if freshRead {
					<-freshEntered
				}
				var settled chan struct{}
				if !freshRead {
					settled = make(chan struct{})
					go func() { w.Settle(); close(settled) }()
				}
				synctest.Wait()
				select {
				case <-oldDone:
					t.Error("cancelled read reported completion before its underlying read returned")
				default:
				}
				select {
				case <-settled:
					t.Error("Settle returned while a superseded metadata read was still active")
				default:
				}
				close(release)
				if settled != nil {
					<-settled
				}
				synctest.Wait()
				// Cancellation needs no UI delivery after the actual read exits.
				select {
				case <-oldDone:
				default:
					t.Error("cancelled read did not complete independently of the UI queue")
				}
				if got := reads.Load(); got != 1 {
					t.Errorf("cancelled source read %d chunks, want 1", got)
				}
				if got := closes.Load(); got != 1 {
					t.Errorf("source closes = %d, want 1", got)
				}
				if freshRead {
					freshDone := make(chan struct{})
					handle := w.MetadataDone().Current()
					go func() { _ = handle.Wait(context.Background()); close(freshDone) }()
					synctest.Wait()
					select {
					case <-freshDone:
						t.Error("old completion finished the replacement request")
					default:
					}
					if w.Text().Text != lang.L("Loading...") {
						t.Errorf("held replacement text = %q", w.Text().Text)
					}
					close(freshRelease)
				}
				w.Settle()
				<-oldDone
				if change == "no source" && w.Text().Text != "" {
					t.Errorf("no-source text = %q", w.Text().Text)
				}
				if change == "close" && w.Open() {
					t.Error("old read reopened the closed panel")
				}
				if w.Open() {
					w.Window().Close()
				}
				w.Settle()
				// Bubble-owned signals cannot escape into a later test's cleanup.
				w.metadata.done = completion.Signal{}
			})
		})
	}
}

func TestMetadataResultsApplyOnceOnUIAndRejectQueuedOldData(t *testing.T) {
	for _, failed := range []bool{false, true} {
		for _, replace := range []bool{false, true} {
			t.Run(fmt.Sprintf("failure=%v/replaced=%v", failed, replace), func(t *testing.T) {
				app, host := gpsApp(t)
				file, _ := host.DisplayedFile()
				if failed {
					host.current = func() (fyne.URI, bool) {
						return uitest.ReaderURI(file, func() (io.ReadCloser, error) { return nil, errors.New("read failed") }), true
					}
				}
				w := newTestWindow(t, app, host)
				w.Show()
				defer w.Window().Close()
				w.metadata.workers.Wait()
				if w.Text().Text != lang.L("Loading...") {
					t.Errorf("result applied before UI drain: %q", w.Text().Text)
				}
				ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
				if err := w.MetadataDone().Wait(ctx); err == nil {
					t.Error("read completed before UI result delivery")
				}
				cancel()
				if replace {
					host.current = func() (fyne.URI, bool) { return nil, false }
					w.Refresh()
				}
				if !w.ui.Drain() {
					t.Error("metadata result did not use the panel UI queue")
				}
				if w.ui.Drain() {
					t.Error("result queued more than once")
				}
				if err := w.MetadataDone().Wait(context.Background()); err != nil {
					t.Fatal(err)
				}
				want := lang.L("Could not read this file's metadata.")
				if replace {
					want = ""
				} else if !failed {
					data, err := os.ReadFile(file.Path())
					if err != nil {
						t.Fatal(err)
					}
					want = formatExifMetadata(imaging.ReadMetadata(data))
				}
				if w.Text().Text != want {
					t.Errorf("text = %q, want %q", w.Text().Text, want)
				}
				wantGPS := !replace && !failed
				if w.Location().Visible() != wantGPS {
					t.Errorf("GPS visibility = %v, want %v", w.Location().Visible(), wantGPS)
				}
				if northHolds(w.north, w.stripBar) != wantGPS {
					t.Errorf("strip row membership = %v, want %v", northHolds(w.north, w.stripBar), wantGPS)
				}
			})
		}
	}
}

type refreshingMetadataHost struct {
	*stubHost
	refresh func()
}

func (h *refreshingMetadataHost) AfterMetadataRemoved(u fyne.URI, result imaging.WriteResult) {
	h.stubHost.AfterMetadataRemoved(u, result)
	if h.refresh != nil {
		h.refresh()
	}
}

func TestMetadataRemovalSupersedesReadAndRefreshesExactlyOnce(t *testing.T) {
	for _, hostRefresh := range []bool{false, true} {
		t.Run(fmt.Sprintf("host refresh=%v", hostRefresh), func(t *testing.T) {
			app, host := gpsApp(t)
			file, _ := host.DisplayedFile()
			data, err := os.ReadFile(file.Path())
			if err != nil {
				t.Fatal(err)
			}
			var opens atomic.Int32
			entered, release := make(chan struct{}), make(chan struct{})
			postRemoval := make(chan struct{})
			source := uitest.ReaderURI(file, func() (io.ReadCloser, error) {
				n := opens.Add(1)
				if n == 2 {
					close(entered)
					<-release
					return io.NopCloser(bytes.NewReader(data)), nil
				}
				if n == 3 {
					close(postRemoval)
				}
				return os.Open(file.Path())
			})
			host.current = func() (fyne.URI, bool) { return source, true }
			h := &refreshingMetadataHost{stubHost: host}
			w := newTestWindow(t, app, h)
			if hostRefresh {
				h.refresh = func() { w.Refresh(); <-postRemoval }
			}
			w.Show()
			settleMetadata(w)
			defer w.Window().Close()
			w.Refresh()
			<-entered
			if w.Text().Text != lang.L("Loading...") || w.Location().Visible() || northHolds(w.north, w.stripBar) {
				t.Error("old metadata/GPS/action remains while replacement read is held")
			}
			w.performStrip(source)
			close(release)
			w.Settle()
			if got := opens.Load(); got != 3 {
				t.Errorf("metadata opens = %d, want initial + held + one post-removal read", got)
			}
			if host.after != 1 || host.afterU.String() != source.String() {
				t.Errorf("removal notification = %d, %v", host.after, host.afterU)
			}
			if len(host.toasts) != 1 || host.toasts[0] != lang.L("Metadata removed") {
				t.Errorf("toasts = %v", host.toasts)
			}
			if w.Text().Text != formatExifMetadata(imaging.Metadata{}) {
				t.Errorf("obsolete pre-removal text = %q", w.Text().Text)
			}
			if w.Location().Visible() || northHolds(w.north, w.stripBar) {
				t.Error("obsolete GPS/strip action survived metadata removal")
			}
		})
	}
}

func TestMetadataReadLeavesQueuedWindowInputResponsive(t *testing.T) {
	app, host := testApp(t)
	file, _ := host.DisplayedFile()
	data, err := os.ReadFile(file.Path())
	if err != nil {
		t.Fatal(err)
	}
	synctest.Test(t, func(t *testing.T) {
		w := newTestWindow(t, app, host)
		entered, release := make(chan struct{}), make(chan struct{})
		source := uitest.ReaderURI(file, func() (io.ReadCloser, error) {
			reader := bytes.NewReader(data)
			var once sync.Once
			return uitest.ReadCloser{
				ReadFunc: func(p []byte) (int, error) {
					once.Do(func() { close(entered); <-release })
					return reader.Read(p)
				},
				CloseFunc: func() error { return nil },
			}, nil
		})
		host.current = func() (fyne.URI, bool) { return source, true }
		actions := make(chan func(), 2)
		uiDone := make(chan struct{})
		go func() {
			defer close(uiDone)
			for action := range actions {
				action()
			}
		}()
		actions <- w.Show
		<-entered
		interaction := make(chan struct{})
		actions <- func() {
			w.Window().Canvas().OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyRight})
			close(interaction)
		}
		synctest.Wait()
		select {
		case <-interaction:
		default:
			t.Error("queued window input blocked behind metadata source read")
		}
		close(release)
		<-interaction
		close(actions)
		<-uiDone
		w.Settle()
		if len(host.steps) != 1 || host.steps[0] != 1 {
			t.Errorf("queued key did not reach navigation: %v", host.steps)
		}
		w.Window().Close()
		w.Settle()
	})
}
