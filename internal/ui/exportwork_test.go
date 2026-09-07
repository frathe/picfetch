package ui

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestExportCancellationBeforeDestinationLeavesFilesUntouched(t *testing.T) {
	v, _, _ := newTestUI(t)
	source := storage.NewFileURI(uitest.WriteTempFile(t, "a.png", uitest.EncodePNG(t, 8, 16, color.White)))
	other := storage.NewFileURI(uitest.WriteTempFile(t, "b.png", uitest.EncodePNG(t, 14, 7, color.Black)))
	dropAndWait(t, v, source, other)
	before, err := os.ReadFile(source.Path())
	if err != nil {
		t.Fatal(err)
	}
	entered, release := make(chan struct{}), make(chan struct{})
	uitest.StubSaveChooser(t, func(_ string) (fyne.URI, error) { close(entered); <-release; return source, nil })
	v.rotateBy(1)
	v.exportAs(".png")
	<-entered
	v.ShowImage(1)
	waitUntilLoaded(t, v)
	close(release)
	settleChooser(t, v)
	after, err := os.ReadFile(source.Path())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(before, after) {
		t.Error("export whose source was abandoned wrote after its native panel returned")
	}
	if v.toast.card.Visible() {
		t.Error("cancelled export showed a completion toast")
		settleToast(t, v)
	}
}

func TestExportCommittedAliasRefreshesCurrentPixelsAfterDelivery(t *testing.T) {
	for _, navigate := range []bool{false, true} {
		t.Run(fmt.Sprintf("navigate=%v", navigate), func(t *testing.T) {
			v, _, _ := newTestUI(t)
			source := storage.NewFileURI(uitest.WriteTempFile(t, "a.png", uitest.EncodePNG(t, 8, 16, color.White)))
			aliasPath := filepath.Join(t.TempDir(), "alias.png")
			if err := os.Symlink(source.Path(), aliasPath); err != nil {
				t.Fatal(err)
			}
			alias := storage.NewFileURI(aliasPath)
			dropAndWait(t, v, source, alias)
			v.preloads.Wait()
			if _, err := v.loadComparedImage(context.Background(), alias); err != nil {
				t.Fatal(err)
			}
			if err := v.grid.Warm(); err != nil {
				t.Fatal(err)
			}
			v.grid.StoreThumb(alias, image.NewRGBA(image.Rect(0, 0, 8, 16)))
			v.dupes.PutNativeSize(alias.String(), image.Pt(8, 16))
			entered, release := make(chan struct{}), make(chan struct{})
			v.fileWork.export = func(ctx context.Context, dest fyne.URI, pixels image.Image, src fyne.URI, opts imaging.ExportOptions) (imaging.WriteResult, error) {
				result, err := imaging.ExportContext(ctx, dest, pixels, src, opts)
				close(entered)
				<-release
				return result, err
			}
			uitest.StubSaveChooser(t, func(_ string) (fyne.URI, error) { return source, nil })
			v.rotateBy(1)
			v.exportAs(".png")
			<-entered
			if navigate {
				v.ShowImage(1)
				waitUntilLoaded(t, v)
			}
			close(release)
			settleChooser(t, v)
			waitUntilLoaded(t, v)
			if got := v.img.Image.Bounds().Size(); got != image.Pt(16, 8) || v.display.Rotation() != 0 {
				t.Errorf("committed alias did not refresh view: size=%v rotation=%v", got, v.display.Rotation())
			}
			if thumb, ok := v.grid.CachedThumb(alias); ok && thumb.Bounds().Dx() < thumb.Bounds().Dy() {
				t.Error("alias retained pre-export thumbnail")
			}
			if size, ok := v.dupes.NativeSize(alias.String()); ok && size == image.Pt(8, 16) {
				t.Error("alias retained pre-export native dimensions")
			}
			if navigate && v.toast.card.Visible() {
				t.Error("obsolete export delivered a toast")
			}
			if v.toast.card.Visible() {
				settleToast(t, v)
			}
		})
	}
}

func TestFileMutationReconciliationSurvivesUnrelatedCommit(t *testing.T) {
	v, _, _ := newTestUI(t)
	source := storage.NewFileURI(uitest.WriteTempFile(t, "source.png", uitest.EncodePNG(t, 8, 16, color.White)))
	other := storage.NewFileURI(uitest.WriteTempFile(t, "other.png", uitest.EncodePNG(t, 4, 4, color.Black)))
	dropAndWait(t, v, source)
	pixels := image.NewRGBA(image.Rect(0, 0, 16, 8))
	first, err := imaging.ExportContext(context.Background(), source, pixels, nil, imaging.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	v.AfterFileExported(first)
	v.fileWork.workers.Wait() // The current-file decision is queued, not applied.
	second, err := imaging.ExportContext(context.Background(), other, pixels, nil, imaging.ExportOptions{})
	if err != nil {
		t.Fatal(err)
	}
	v.AfterFileExported(second)
	drainFileWork(t, v)
	waitUntilLoaded(t, v)
	if got := v.img.Image.Bounds().Size(); got != image.Pt(16, 8) {
		t.Errorf("unrelated commit discarded required current-source refresh: %v", got)
	}
}

func TestExportBusyAndQueuedFailureAllowRetry(t *testing.T) {
	v, _, _ := newTestUI(t)
	source := uitest.TempJPEGURI(t, "a.jpg", 8, 16, color.White)
	other := uitest.TempJPEGURI(t, "b.jpg", 14, 7, color.Black)
	dest := storage.NewFileURI(filepath.Join(t.TempDir(), "export.png"))
	dropAndWait(t, v, source, other)
	uitest.StubSaveChooser(t, func(_ string) (fyne.URI, error) { return dest, nil })
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	var calls atomic.Int32
	v.fileWork.export = func(_ context.Context, _ fyne.URI, _ image.Image, _ fyne.URI, _ imaging.ExportOptions) (imaging.WriteResult, error) {
		calls.Add(1)
		once.Do(func() { close(entered) })
		<-release
		return imaging.WriteResult{}, errors.New("export failed")
	}
	v.exportAs(".png")
	<-entered
	handle := v.chooser.Current()
	v.exportAs(".png")
	if v.canExport() || handle != v.chooser.Current() {
		t.Error("repeated Export replaced active operation or remained enabled")
	}
	close(release)
	v.fileWork.workers.Wait()
	if calls.Load() != 1 || v.toast.card.Visible() {
		t.Error("worker admitted a duplicate export or presented its error")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Millisecond)
	if err := handle.Wait(ctx); err == nil {
		t.Error("Export completion preceded its queued error")
	}
	cancel()
	v.ShowImage(1)
	waitUntilLoaded(t, v)
	v.rotateBy(1)
	pixels := v.img.Image
	settleChooser(t, v)
	if v.toast.card.Visible() || v.img.Image != pixels || v.display.Rotation() != 1 {
		t.Error("obsolete export error changed the navigated view")
	}
	v.fileWork.export = imaging.ExportContext
	v.exportAs(".png")
	settleChooser(t, v)
	if !v.canExport() || !v.toast.card.Visible() {
		t.Error("export retry did not finish normally")
	}
	settleToast(t, v)
	v.closeFileWork()
	handle = v.chooser.Current()
	v.exportAs(".png")
	if v.canExport() || v.chooser.Current() != handle {
		t.Error("shutdown allowed a fresh export")
	}
}
