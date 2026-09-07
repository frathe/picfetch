package ui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"testing/synctest"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

type heldSavePixels struct {
	image.Image
	once             sync.Once
	entered, release chan struct{}
}

func TestSaveChangesInvalidatesAliasesAndRejectsOlderPreloads(t *testing.T) {
	v, _, _ := newTestUI(t)
	source := storage.NewFileURI(uitest.WriteTempFile(t, "source.png", uitest.EncodePNG(t, 8, 16, color.White)))
	aliasPath := filepath.Join(t.TempDir(), "alias.png")
	if err := os.Symlink(source.Path(), aliasPath); err != nil {
		t.Fatal(err)
	}
	alias := storage.NewFileURI(aliasPath)
	dropAndWait(t, v, source)
	if _, err := v.loadComparedImage(context.Background(), alias); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(source.Path())
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	controlled := uitest.ReaderURI(alias, func() (io.ReadCloser, error) {
		close(entered)
		<-release
		return io.NopCloser(bytes.NewReader(before)), nil
	})
	token := v.loadLifecycle.begin()
	v.preloadOne(token, controlled)
	<-entered
	v.rotateBy(1)
	v.saveRotation()
	waitForSave(t, v)
	if cached, ok := v.imgCache.Get(alias.String()); ok && cached.Frames[0].Bounds().Size() != image.Pt(16, 8) {
		t.Error("saving through one path retained old pixels under its alias")
	}
	close(release)
	v.preloads.Wait()
	if cached, ok := v.imgCache.Get(controlled.String()); ok && cached.Frames[0].Bounds().Size() != image.Pt(16, 8) {
		t.Error("a pre-commit alias decode repopulated the invalidated cache")
	}
	settleToast(t, v)
}

func TestSaveChangesCancellationKeepsCommittedDiskEffectsAndCurrentView(t *testing.T) {
	for _, committed := range []bool{false, true} {
		for _, change := range []string{"navigate", "clear", "close"} {
			t.Run(fmt.Sprintf("committed=%v/change=%s", committed, change), func(t *testing.T) {
				v, _, _ := newTestUI(t)
				source := storage.NewFileURI(uitest.WriteTempFile(t, "a.png", uitest.EncodePNG(t, 8, 16, color.White)))
				other := uitest.TempJPEGURI(t, "b.jpg", 14, 7, color.RGBA{B: 255, A: 255})
				dropAndWait(t, v, source, other)
				v.rotateBy(1)
				before, err := os.ReadFile(source.Path())
				if err != nil {
					t.Fatal(err)
				}
				entered, release := make(chan struct{}), make(chan struct{})
				v.fileWork.save = func(ctx context.Context, u fyne.URI, img image.Image) (imaging.WriteResult, error) {
					var result imaging.WriteResult
					var err error
					if committed {
						result, err = imaging.SaveRotatedContext(ctx, u, img)
					}
					close(entered)
					<-release
					if !committed {
						result, err = imaging.SaveRotatedContext(ctx, u, img)
					}
					return result, err
				}
				v.saveRotation()
				<-entered
				switch change {
				case "navigate":
					v.ShowImage(1)
					waitUntilLoaded(t, v)
					v.rotateBy(1)
				case "clear":
					v.clearToDropzone()
				case "close":
					v.closeFileWork()
				}
				shown, rotation := v.img.Image, v.display.Rotation()
				ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
				if err := v.fileWork.saveDone.Wait(ctx); err == nil {
					t.Error("save finished before its writer returned")
				}
				cancel()
				close(release)
				waitForSave(t, v)
				after, err := os.ReadFile(source.Path())
				if err != nil {
					t.Fatal(err)
				}
				if committed == bytes.Equal(before, after) {
					t.Errorf("disk change = %v, want committed=%v", !bytes.Equal(before, after), committed)
				}
				if committed && v.imgCache.Contains(source.String()) {
					t.Error("cancelled presentation retained the source's pre-commit cache record")
				}
				if v.img.Image != shown || v.display.Rotation() != rotation {
					t.Error("obsolete save changed the current view or rotation")
				}
				if v.toast.card.Visible() {
					t.Error("obsolete save showed a success or error toast")
				}
			})
		}
	}
}

func TestSaveChangesCapturesPixelsAndRetainsLaterRotation(t *testing.T) {
	for _, tc := range []struct {
		name                  string
		saved, later, pending int
		reset                 bool
	}{
		{name: "clockwise", saved: 1, later: 1, pending: 1},
		{name: "counterclockwise", saved: 1, later: -1, pending: 3},
		{name: "wraparound", saved: 3, later: 1, pending: 1},
		{name: "reset", saved: 1, reset: true, pending: 3},
		{name: "full cycle", saved: 1, later: 4, pending: 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			v, _, _ := newTestUI(t)
			source := storage.NewFileURI(uitest.WriteTempFile(t, "save.png", uitest.EncodePNG(t, 8, 16, color.White)))
			dropAndWait(t, v, source)
			v.rotateBy(tc.saved)
			wantSaved := v.img.Image.Bounds()
			entered, release := make(chan struct{}), make(chan struct{})
			v.fileWork.save = func(ctx context.Context, u fyne.URI, img image.Image) (imaging.WriteResult, error) {
				close(entered)
				<-release
				return imaging.SaveRotatedContext(ctx, u, img)
			}
			v.saveRotation()
			<-entered
			if tc.reset {
				v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.Key0})
			} else {
				v.rotateBy(tc.later)
			}
			wantShown := v.img.Image
			close(release)
			waitForSave(t, v)
			loaded, err := imaging.LoadImage(source, imaging.DefaultImgCacheBytes)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.Frames[0].Bounds() != wantSaved {
				t.Errorf("saved bounds = %v, want captured %v", loaded.Frames[0].Bounds(), wantSaved)
			}
			if v.img.Image != wantShown {
				t.Error("finishing the earlier save changed the displayed pixels")
			}
			if got := v.display.Rotation(); got != tc.pending {
				t.Errorf("pending rotation = %d, want %d relative to the saved frame", got, tc.pending)
			}
			if v.canSaveRotation() != (tc.pending != 0) {
				t.Error("Save availability does not reflect the remaining rotation")
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.Key0})
			if got := v.img.Image.Bounds(); got != loaded.Frames[0].Bounds() {
				t.Errorf("reset bounds = %v, want saved file bounds %v", got, loaded.Frames[0].Bounds())
			}
			v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyR})
			if got := v.img.Image.Bounds().Size(); got != image.Pt(8, 16) {
				t.Errorf("rotation after reset = %v, want 8x16 relative to the saved frame", got)
			}
			settleToast(t, v)
		})
	}
}

func TestSaveChangesBusyPreservesCompletionAndAllowsRetry(t *testing.T) {
	v, _, _ := newTestUI(t)
	source := uitest.TempJPEGURI(t, "save.jpg", 8, 16, color.White)
	dropAndWait(t, v, source)
	shared, ok := v.imgCache.Get(source.String())
	if !ok {
		t.Fatal("source cache fixture missing")
	}
	v.rotateBy(1)
	entered, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int32
	var once sync.Once
	v.fileWork.save = func(_ context.Context, _ fyne.URI, _ image.Image) (imaging.WriteResult, error) {
		calls.Add(1)
		once.Do(func() { close(entered) })
		<-release
		return imaging.WriteResult{}, errors.New("save failed")
	}
	v.saveRotation()
	<-entered
	handle := v.fileWork.saveDone.Current()
	v.saveRotation()
	if v.canSaveRotation() || !v.menus.Save().Disabled {
		t.Error("Save Changes remained enabled during its active write")
	}
	if handle != v.fileWork.saveDone.Current() {
		t.Error("repeated Save replaced the active completion")
	}
	close(release)
	v.fileWork.workers.Wait()
	if calls.Load() != 1 {
		t.Errorf("overlapping Save calls = %d, want 1", calls.Load())
	}
	if v.toast.card.Visible() {
		t.Error("worker presented its save error")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	if err := handle.Wait(ctx); err == nil {
		t.Error("Save completion preceded the queued error")
	}
	cancel()
	waitForSave(t, v)
	if !v.canSaveRotation() || v.display.Rotation() != 1 || !v.toast.card.Visible() {
		t.Error("failed Save lost rotation, its error, or retry admission")
	}
	settleToast(t, v)
	v.fileWork.save = imaging.SaveRotatedContext
	v.saveRotation()
	waitForSave(t, v)
	if v.display.Rotation() != 0 || v.canSaveRotation() {
		t.Error("successful retry did not settle the saved rotation")
	}
	if shared.Frames[0].Bounds().Size() != image.Pt(8, 16) {
		t.Error("save mutated decoded frames already shared with another reader")
	}
	settleToast(t, v)
}

func TestSaveChangesQueuedFailureCannotAffectNavigatedView(t *testing.T) {
	v, _, _ := newTestUI(t)
	source := uitest.TempJPEGURI(t, "a.jpg", 8, 16, color.White)
	other := uitest.TempJPEGURI(t, "b.jpg", 14, 7, color.Black)
	dropAndWait(t, v, source, other)
	v.rotateBy(1)
	v.fileWork.save = func(_ context.Context, _ fyne.URI, _ image.Image) (imaging.WriteResult, error) {
		return imaging.WriteResult{}, errors.New("old failure")
	}
	v.saveRotation()
	v.fileWork.workers.Wait()
	v.ShowImage(1)
	waitUntilLoaded(t, v)
	v.rotateBy(1)
	want, rotation := v.img.Image, v.display.Rotation()
	waitForSave(t, v)
	if v.toast.card.Visible() || v.img.Image != want || v.display.Rotation() != rotation {
		t.Error("queued obsolete error changed the navigated view")
	}
}

func (p *heldSavePixels) At(x, y int) color.Color {
	p.once.Do(func() { close(p.entered); <-p.release })
	return p.Image.At(x, y)
}

func TestSaveChangesLeavesQueuedUIResponsive(t *testing.T) {
	v, _, _ := newTestUI(t)
	source := storage.NewFileURI(uitest.WriteTempFile(t, "save.png", uitest.EncodePNG(t, 8, 16, color.White)))
	dropAndWait(t, v, source)
	v.rotateBy(1)
	synctest.Test(t, func(t *testing.T) {
		pixels := &heldSavePixels{Image: v.img.Image, entered: make(chan struct{}), release: make(chan struct{})}
		v.img.Image = pixels
		actions := make(chan func(), 2)
		uiDone := make(chan struct{})
		go func() {
			defer close(uiDone)
			for fn := range actions {
				fn()
			}
		}()
		actions <- v.saveRotation
		<-pixels.entered
		interaction := make(chan struct{})
		actions <- func() { v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyM}); close(interaction) }
		synctest.Wait()
		select {
		case <-interaction:
		default:
			t.Error("queued unrelated UI input blocked behind Save Changes encoding")
		}
		close(pixels.release)
		<-interaction
		close(actions)
		<-uiDone
		waitForSave(t, v)
		settleToast(t, v)
		v.toast.hidden = completion.Signal{}
		v.fileWork.saveDone = completion.Signal{}
		// Save invalidation now admits grouping inside this synctest bubble.
		// Cancel and drain that work before outer viewer cleanup runs.
		v.grid.Stop()
		v.grid.Settle()
	})
}

func waitForSave(t *testing.T, v *viewer) {
	t.Helper()
	if !v.fileWork.saveDone.Begun() {
		t.Fatal("Save Changes never started")
	}
	drainFileWork(t, v)
	waitFor(t, "Save Changes", &v.fileWork.saveDone)
}

func drainFileWork(t *testing.T, v *viewer) {
	t.Helper()
	for {
		finished := make(chan struct{})
		go func() { v.fileWork.workers.Wait(); close(finished) }()
		select {
		case <-finished:
		case <-time.After(testTimeout):
			t.Fatal("timed out waiting for file mutations")
		}
		if !v.fileWork.ui.Drain() {
			return
		}
	}
}
