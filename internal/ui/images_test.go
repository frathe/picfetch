package ui

import (
	"context"
	"errors"
	"image"
	"os"
	"os/exec"
	"strings"
	"sync/atomic"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/heicdecode"
	heicclient "github.com/frathe/picfetch/internal/heicdecode/client"
	"github.com/frathe/picfetch/internal/imaging"
	mosaiccore "github.com/frathe/picfetch/internal/mosaic"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestHEICSourceConsumers(t *testing.T) {
	var foreground, background atomic.Int64
	reader := func(calls *atomic.Int64) imaging.Reader {
		return imaging.NewReader(func(ctx context.Context, op heicdecode.Operation, input heicclient.Input) (heicdecode.Response, error) {
			calls.Add(1)
			data, err := input(ctx, 128)
			if err != nil {
				return heicdecode.Response{}, err
			}
			date := "2026:09:16 12:34:56"
			if string(data) == "early" {
				date = "2026:09:15 12:34:56"
			}
			result := heicdecode.Response{Metadata: &heicdecode.Metadata{Make: "Owned camera", DateTimeOriginal: date}}
			if op == heicdecode.Decode {
				result.Image = image.NewNRGBA64(image.Rect(0, 0, 8, 6))
				result.Config = image.Config{Width: 8, Height: 6}
			}
			return result, nil
		})
	}
	v, _, _ := newTestUIWithImages(t, imageServices{foreground: reader(&foreground), background: reader(&background)})
	late := storage.NewFileURI(uitest.WriteTempFile(t, "late.heic", []byte("late")))
	early := storage.NewFileURI(uitest.WriteTempFile(t, "early.heic", []byte("early")))
	files := []fyne.URI{late, early}
	// Format advertisement stays disabled during qualification; install this
	// owned source set directly and exercise the ordinary viewer load path.
	v.state.replaceFiles(files, files)
	v.ShowImage(0)
	waitUntilLoaded(t, v)
	v.display.Settle()
	if foreground.Load() == 0 || !v.display.Snapshot().HasEXIF {
		t.Fatal("viewer did not inject its foreground reader")
	}
	v.imgCache.Purge()
	compared, err := v.loadComparedImage(context.Background(), early)
	if err != nil || compared.FileSize != 5 || !compared.HasEXIF {
		t.Fatalf("comparison source = %v", err)
	}
	v.exif.Show()
	v.exif.Settle()
	if !strings.Contains(v.exif.Text().Text, "Owned camera") || v.exif.StripButton().Visible() {
		t.Fatal("EXIF source values or editing capability were lost")
	}
	if err := v.grid.Warm(); err != nil {
		t.Fatal(err)
	}
	if !v.grid.Cached(early) || background.Load() == 0 {
		t.Fatal("grid did not use the background reader")
	}
	v.SetSortMode(filesort.ByCaptureDate)
	waitForSort(t, v)
	waitUntilLoaded(t, v)
	if v.state.files[0].String() != early.String() {
		t.Fatal("capture-date sort ignored isolated metadata")
	}
	request, err := mosaiccore.NewRequest(files, image.Pt(320, 180), mosaiccore.DefaultSettings(), 1)
	if err != nil {
		t.Fatal(err)
	}
	result, err := v.GenerateMosaic(context.Background(), request, nil)
	if err != nil || result.Bounds().Size() != image.Pt(320, 180) {
		t.Fatalf("mosaic source = %v", err)
	}
	// Use an uncached third source so Favorite persistence must decode.
	favorite := storage.NewFileURI(uitest.WriteTempFile(t, "favorite.heic", []byte("favorite")))
	favDir := t.TempDir()
	v.SetFavoritePreviewCache(true)
	v.SyncFavoritePreviews(favDir, []fyne.URI{favorite})
	settleFavoritePreviews(t, v)
	if got := previewNames(t, favDir); len(got) != 1 {
		t.Fatalf("Favorite previews = %v", got)
	}
}

func TestHEICOwnerStopsWithViewer(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	owner, err := heicclient.New(heicclient.Config{Executable: executable, SHA256: [32]byte{1}, Limits: heicdecode.DefaultLimits(0)})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { owner.Stop(); owner.Wait() }()
	v, _, _ := newTestUIWithImages(t, newImageServices(owner))
	if v.explorer.Options().Client.HEIC != owner {
		t.Fatal("Explorer received a different admission owner")
	}
	link, err := owner.Attach(context.Background(), exec.Command(executable))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { link.Stop(); link.Wait() }()
	lifecycle, ok := testApp.Lifecycle().(interface{ OnStopped() func() })
	if !ok {
		t.Fatal("test lifecycle has no stopped hook")
	}
	previous := lifecycle.OnStopped()
	t.Cleanup(func() { testApp.Lifecycle().SetOnStopped(previous) })
	registerShutdown(testApp, v)
	lifecycle.OnStopped()()
	v.waitForShutdown()
	_, err = owner.Do(context.Background(), heicdecode.Decode, func(_ context.Context, _ int64) ([]byte, error) {
		t.Fatal("stopped owner read a source")
		return nil, nil
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("viewer left owner admission open: %v", err)
	}
}
