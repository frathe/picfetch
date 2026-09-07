package ui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"strings"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/uitest"
)

type heldClipboardImage struct {
	image.Image
	once             sync.Once
	entered, release chan struct{}
}

func (i *heldClipboardImage) At(x, y int) color.Color {
	i.once.Do(func() { close(i.entered); <-i.release })
	return i.Image.At(x, y)
}

func TestClipboardEncodingLeavesQueuedUIResponsive(t *testing.T) {
	v := newTestViewer(t)
	uitest.StubClipboardCopy(t, func(_ []byte) error { return nil })
	synctest.Test(t, func(t *testing.T) {
		frame := &heldClipboardImage{Image: image.NewRGBA(image.Rect(0, 0, 16, 16)), entered: make(chan struct{}), release: make(chan struct{})}
		v.img.Image = frame
		actions := make(chan func(), 2)
		uiDone := make(chan struct{})
		go func() {
			defer close(uiDone)
			for fn := range actions {
				fn()
			}
		}()
		actions <- v.copyImageToClipboard
		<-frame.entered
		interaction := make(chan struct{})
		actions <- func() { v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyM}); close(interaction) }
		synctest.Wait()
		select {
		case <-interaction:
		default:
			t.Error("queued unrelated UI action did not run during PNG encoding")
		}
		close(frame.release)
		<-interaction
		close(actions)
		<-uiDone
		waitForClipboard(t, v)
		v.img.Image = frame.Image
		v.clipboard = completion.Signal{}
	})
}

func TestClipboardNavigationCancelsHeldEncoding(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 16, 16, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 16, 16, color.Black)
	dropAndWait(t, v, a, b)
	frame := &heldClipboardImage{Image: v.img.Image, entered: make(chan struct{}), release: make(chan struct{})}
	var once sync.Once
	unblock := func() { once.Do(func() { close(frame.release) }) }
	defer unblock()
	v.img.Image = frame
	calls := 0
	uitest.StubClipboardCopy(t, func(_ []byte) error { calls++; return nil })
	v.copyImageToClipboard()
	select {
	case <-frame.entered:
	case <-time.After(testTimeout):
		t.Fatal("PNG encoder did not reach the captured image")
	}
	v.ShowImage(1)
	waitUntilLoaded(t, v)
	unblock()
	waitForClipboard(t, v)
	if calls != 0 {
		t.Errorf("cancelled encoding dispatched %d clipboard writes", calls)
	}
}

func TestClipboardBusyPreservesActiveOperationAcrossCopyRoutes(t *testing.T) {
	for _, route := range []string{"image", "grid", "region", "path"} {
		t.Run(route, func(t *testing.T) {
			v := newTestViewer(t)
			dropAndWait(t, v, uitest.TempJPEGURI(t, "source.jpg", 16, 16, color.White))
			uitest.StubClipboardCopy(t, func(_ []byte) error { return nil })
			uitest.StubClipboardCopyFiles(t, func(_ []string) error { return nil })
			frame := &heldClipboardImage{Image: v.img.Image, entered: make(chan struct{}), release: make(chan struct{})}
			var once sync.Once
			unblock := func() { once.Do(func() { close(frame.release) }) }
			defer unblock()
			v.img.Image = frame
			v.app.Clipboard().SetContent("previous clipboard")
			v.copyImageToClipboard()
			original := v.clipboard.Current()
			select {
			case <-frame.entered:
			case <-time.After(testTimeout):
				t.Fatal("encoder did not reach captured pixels")
			}
			switch route {
			case "image":
				v.copyImageToClipboard()
			case "grid":
				v.grid.Toggle()
				v.copySelection()
			case "region":
				v.startRegionCopy()
				selectRegion(t, v, image.Rect(0, 0, 8, 8))
				v.regionCopy.HandleKey(fyne.KeyReturn)
			case "path":
				v.copyPathToClipboard()
				if v.app.Clipboard().Content() != "previous clipboard" {
					t.Error("path copy ran while an older native image write was pending")
				}
			}
			if v.clipboard.Current() != original {
				t.Error("competing copy replaced the active operation's completion")
			}
			unblock()
			waitForClipboard(t, v)
			ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
			defer cancel()
			if err := original.Wait(ctx); err != nil {
				t.Fatal("original clipboard operation did not finish:", err)
			}
			switch route {
			case "region":
				if v.regionCopy.State().Busy || !v.regionCopy.State().Active {
					t.Fatal("refused copy must leave its selection available for retry")
				}
				v.regionCopy.HandleKey(fyne.KeyReturn)
			case "path":
				v.copyPathToClipboard()
				current, _, ok := v.CurrentFile()
				if !ok || v.app.Clipboard().Content() != current.Path() {
					t.Error("path copy did not resume after image copy finished")
				}
				return
			default:
				v.copySelection()
			}
			if v.clipboard.Current() == original {
				t.Error("copy retry did not begin a fresh operation")
			}
			waitForClipboard(t, v)
		})
	}
}

func TestClipboardCancellationStopsEncoderWrites(t *testing.T) {
	v := newTestViewer(t)
	v.img.Image = image.NewRGBA(image.Rect(0, 0, 2, 2))
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	var writeError error
	v.clipboardWork.encode = func(out io.Writer, _ image.Image) error {
		if _, err := out.Write([]byte("first PNG chunk")); err != nil {
			return err
		}
		close(entered)
		<-release
		_, writeError = out.Write([]byte("next PNG chunk"))
		return writeError
	}
	uitest.StubClipboardCopy(t, func(_ []byte) error { t.Error("cancelled encoding reached OS dispatch"); return nil })
	v.copyImageToClipboard()
	select {
	case <-entered:
	case <-time.After(testTimeout):
		t.Fatal("encoder did not write its first chunk")
	}
	v.reset()
	unblock()
	waitForClipboard(t, v)
	if !errors.Is(writeError, context.Canceled) {
		t.Errorf("next encoder write returned %v, want cancellation", writeError)
	}
}

func TestClipboardQueuedResultsOwnCompletionAndDiscardStaleEffects(t *testing.T) {
	for _, failure := range []string{"encode", "dispatch", "none"} {
		for _, cancelResult := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/cancel=%v", failure, cancelResult), func(t *testing.T) {
				v := newTestViewer(t)
				v.img.Image = image.NewRGBA(image.Rect(0, 0, 2, 2))
				if failure == "encode" {
					v.clipboardWork.encode = func(_ io.Writer, _ image.Image) error { return errors.New("encode failure") }
				}
				calls := 0
				uitest.StubClipboardCopy(t, func(_ []byte) error {
					calls++
					if failure == "dispatch" {
						return errors.New("dispatch failure")
					}
					return nil
				})
				synctest.Test(t, func(t *testing.T) {
					queue := &uitest.UIQueue{}
					v.clipboardWork.ui = queue
					before := v.toast.gen.Load()
					v.copyImageToClipboard()
					v.clipboardWork.workers.Wait()
					if (failure == "encode" && calls != 0) || (failure != "encode" && calls != 1) {
						t.Errorf("clipboard dispatches=%d for %s", calls, failure)
					}
					if v.toast.gen.Load() != before {
						t.Error("result touched UI before queue delivery")
					}
					completed := make(chan struct{})
					go func() { _ = v.clipboard.Wait(context.Background()); close(completed) }()
					synctest.Wait()
					select {
					case <-completed:
						t.Error("operation completed before its queued result was delivered")
					default:
					}
					if cancelResult {
						v.reset()
					}
					before = v.toast.gen.Load()
					if !queue.Drain() {
						t.Error("worker did not queue its result")
					}
					<-completed
					want := before
					if failure != "none" && !cancelResult {
						want++
						if !strings.Contains(v.toast.text.Text, failure+" failure") {
							t.Errorf("toast=%q, want %s failure", v.toast.text.Text, failure)
						}
					}
					if v.toast.gen.Load() != want || queue.Drain() {
						t.Error("result effects were stale, missing, or delivered more than once")
					}
					if v.clipboardWork.pending.Load() {
						t.Error("completed result retained clipboard admission")
					}
					if v.toast.stop != nil {
						settleToast(t, v)
						v.toast.hidden = completion.Signal{}
					}
					v.clipboard = completion.Signal{}
				})
			})
		}
	}
}

func TestClipboardCapturesPixelsBeforeAViewRotation(t *testing.T) {
	v := newTestViewer(t)
	original := image.NewRGBA(image.Rect(0, 0, 2, 3))
	original.Set(0, 0, color.RGBA{R: 255, A: 255})
	original.Set(1, 2, color.RGBA{B: 255, A: 255})
	dropAndWait(t, v, regionCopyPNGURI(t, "original.png", original))
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	v.clipboardWork.encode = func(w io.Writer, captured image.Image) error {
		close(entered)
		<-release
		return png.Encode(w, captured)
	}
	var copied []byte
	uitest.StubClipboardCopy(t, func(data []byte) error { copied = bytes.Clone(data); return nil })
	v.copyImageToClipboard()
	select {
	case <-entered:
	case <-time.After(testTimeout):
		t.Fatal("encoding did not start")
	}
	v.rotateBy(1)
	if got := v.img.Image.Bounds().Size(); got != image.Pt(3, 2) {
		t.Fatalf("fixture did not rotate the displayed image: %v", got)
	}
	unblock()
	waitForClipboard(t, v)
	decoded, err := png.Decode(bytes.NewReader(copied))
	if err != nil {
		t.Fatal(err)
	}
	if decoded.Bounds().Size() != image.Pt(2, 3) {
		t.Errorf("copy retargeted the rotated view: %v", decoded.Bounds())
	}
	for _, pixel := range []struct {
		point image.Point
		want  color.RGBA
	}{{image.Pt(0, 0), color.RGBA{R: 255, A: 255}}, {image.Pt(1, 2), color.RGBA{B: 255, A: 255}}} {
		if got := color.RGBAModel.Convert(decoded.At(pixel.point.X, pixel.point.Y)); got != pixel.want {
			t.Errorf("captured pixel %v = %v, want %v", pixel.point, got, pixel.want)
		}
	}
}

func TestClipboardCloseWaitsActiveDispatchAndDiscardsItsError(t *testing.T) {
	v := newTestViewer(t)
	v.img.Image = image.NewRGBA(image.Rect(0, 0, 2, 2))
	synctest.Test(t, func(t *testing.T) {
		entered, release := make(chan struct{}), make(chan struct{})
		uitest.StubClipboardCopy(t, func(_ []byte) error { close(entered); <-release; return errors.New("late native failure") })
		before := v.toast.gen.Load()
		v.copyImageToClipboard()
		<-entered
		original := v.clipboard.Current()
		v.closeClipboardWork()
		v.copyImageToClipboard()
		if v.clipboard.Current() != original {
			t.Error("closed viewer admitted another clipboard operation")
		}
		completed := make(chan struct{})
		go func() { _ = original.Wait(context.Background()); close(completed) }()
		synctest.Wait()
		select {
		case <-completed:
			t.Error("copy completed while its native dispatch was still held")
		default:
		}
		close(release)
		v.clipboardWork.workers.Wait()
		synctest.Wait()
		select {
		case <-completed:
		default:
			t.Error("cancelled work still needs queued UI to finish its operation")
		}
		if v.clipboardWork.ui.Drain() || v.toast.gen.Load() != before {
			t.Error("closed clipboard request submitted stale presentation")
		}
		<-completed
		if v.toast.stop != nil {
			settleToast(t, v)
			v.toast.hidden = completion.Signal{}
		}
		v.clipboard = completion.Signal{}
	})
}

func TestShutdownStopsClipboardAdmission(t *testing.T) {
	application := test.NewApp()
	v, win := buildStartupViewer(application)
	t.Cleanup(win.Close)
	t.Cleanup(func() { drain(t, v) })
	v.img.Image = image.NewRGBA(image.Rect(0, 0, 2, 2))
	uitest.StubClipboardCopy(t, func(_ []byte) error { t.Error("shutdown admitted clipboard dispatch"); return nil })
	lifecycle, ok := application.Lifecycle().(interface{ OnStopped() func() })
	if !ok {
		t.Fatal("test lifecycle has no stopped hook")
	}
	previous := lifecycle.OnStopped()
	registerShutdown(application, v)
	shutdown := lifecycle.OnStopped()
	application.Lifecycle().SetOnStopped(previous)
	shutdown()
	v.copyImageToClipboard()
	drainClipboard(t, v)
	if v.clipboard.Begun() {
		t.Error("shutdown began a new clipboard operation")
	}
}
