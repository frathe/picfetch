package ui

import (
	"context"
	"encoding/hex"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/mosaic"
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/session"
	"github.com/frathe/picfetch/internal/uitest"
)

type testHEICBackend struct {
	check func(context.Context) error
	read  func(context.Context, []byte, heic.Request) (heic.Result, error)
}

func TestHEICFeatureIntegration(t *testing.T) {
	v := newTestViewer(t)
	exif, err := hex.DecodeString("49492a000800000001000f010200060000001a0000000000000043616e6f6e00")
	if err != nil {
		t.Fatal(err)
	}
	v.configureHEIC(testHEICBackend{
		check: func(_ context.Context) error { return nil },
		read: func(_ context.Context, _ []byte, request heic.Request) (heic.Result, error) {
			result := heic.Result{Width: 2, Height: 1, EXIF: exif}
			if request.Pixels {
				result.Stride = 8
				result.Pixels = []byte{255, 0, 0, 255, 0, 255, 0, 255}
			}
			return result, nil
		},
	})
	v.heic.ui = &uitest.UIQueue{}
	v.startHEICCheck(false)
	v.settleHEIC()
	data, err := os.ReadFile("../imaging/testdata/test_exif.heic")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "photo.HEIC")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	v.handleDrop([]fyne.URI{storage.NewFileURI(path)})
	waitForScan(t, v)
	if !v.sortOp.done.Begun() {
		t.Fatal("verified HEIC photo was not admitted to the ordinary collection")
	}
	waitForSort(t, v)
	waitUntilLoaded(t, v)
	if v.FileCount() != 1 || v.img.Image == nil || v.img.Image.Bounds() != image.Rect(0, 0, 2, 1) {
		t.Fatalf("canonical HEIC did not reach the viewer: files=%d image=%v", v.FileCount(), v.img.Image)
	}
	if err := v.grid.Warm(); err != nil {
		t.Fatalf("HEIC Grid thumbnail: %v", err)
	}
	if thumb, ok := v.grid.CachedThumb(storage.NewFileURI(path)); !ok || thumb.Bounds() != image.Rect(0, 0, 2, 1) {
		t.Fatal("Grid did not retain the canonical HEIC thumbnail")
	}
	v.imgCache.Purge()
	compared, err := v.loadComparedImage(context.Background(), storage.NewFileURI(path))
	if err != nil || compared.Frames[0].Bounds() != image.Rect(0, 0, 2, 1) {
		t.Fatalf("uncached comparison did not use HEIC capability: %v, %v", compared, err)
	}
	favoriteDir := t.TempDir()
	favoritePath := filepath.Join(t.TempDir(), "favorite.heic")
	if err := os.WriteFile(favoritePath, data, 0600); err != nil {
		t.Fatal(err)
	}
	v.SyncFavoritePreviews(favoriteDir, []fyne.URI{storage.NewFileURI(favoritePath)})
	settleFavoritePreviews(t, v)
	if names := previewNames(t, favoriteDir); len(names) != 1 {
		t.Fatalf("HEIC favorite preview missing: %v", names)
	}
	v.exif.Show()
	v.exif.Settle()
	if !strings.Contains(v.exif.Text().Text, "Canon") {
		t.Fatalf("native HEIC metadata did not reach the panel: %q", v.exif.Text().Text)
	}
	request, err := mosaic.NewRequest([]fyne.URI{storage.NewFileURI(path)}, image.Pt(80, 50), mosaic.DefaultSettings(), 2)
	if err != nil {
		t.Fatal(err)
	}
	result, err := v.GenerateMosaic(context.Background(), request, nil)
	if err != nil {
		t.Fatalf("HEIC mosaic generation: %v", err)
	}
	var red, green bool
	for y := range result.Image().Bounds().Dy() {
		for x := range result.Image().Bounds().Dx() {
			r, g, b, _ := result.Image().At(x, y).RGBA()
			red = red || r > 50000 && g < 6000 && b < 6000
			green = green || g > 50000 && r < 6000 && b < 6000
		}
	}
	if !red || !green {
		t.Fatalf("mosaic lost canonical colors: red=%v green=%v", red, green)
	}
}

func (b testHEICBackend) Check(ctx context.Context) error { return b.check(ctx) }
func (b testHEICBackend) Read(ctx context.Context, data []byte, request heic.Request) (heic.Result, error) {
	if b.read != nil {
		return b.read(ctx, data, request)
	}
	return heic.Result{}, heic.ErrUnavailable
}

func TestHEICCapabilityLifecycle(t *testing.T) {
	v := newTestViewer(t)
	v.configureHEIC(testHEICBackend{check: func(_ context.Context) error { return nil }})
	v.heic.ui = &uitest.UIQueue{}
	v.startHEICCheck(false)
	v.settleHEIC()
	if saved := preferences.LoadHEICObservation(v.app); !saved.Available || !saved.Matches(heic.SystemIdentity()) {
		t.Fatalf("startup did not persist verified support: %+v", saved)
	}
	snapshot := v.heic.capability.Snapshot()
	_, _ = snapshot.Backend.Read(context.Background(), []byte{1}, heic.Request{})
	v.settleHEIC()
	if !v.heic.capability.State().Available {
		t.Fatal("backend disappearance did not trigger one support recheck")
	}
}

func TestHEICUnavailableFiles(t *testing.T) {
	v := newTestViewer(t)
	var available atomic.Bool
	v.configureHEIC(testHEICBackend{check: func(_ context.Context) error {
		if available.Load() {
			return nil
		}
		return heic.ErrUnavailable
	}})
	v.heic.ui = &uitest.UIQueue{}
	v.startHEICCheck(false)
	v.settleHEIC()
	heicPath := filepath.Join(t.TempDir(), "missing-support.HEIC")
	if err := os.WriteFile(heicPath, []byte("not decoded while unavailable"), 0600); err != nil {
		t.Fatal(err)
	}
	heicURI := storage.NewFileURI(heicPath)
	jpegURI := uitest.TempJPEGURI(t, "available.jpg", 2, 1, color.White)
	storeFavorite(t, v, "Mixed", jpegURI, heicURI)
	v.favorites.Menu().Items[2].Action()
	waitForScan(t, v)
	waitForSort(t, v)
	waitUntilLoaded(t, v)
	settleFavoritePreviews(t, v)
	if v.FileCount() != 1 {
		t.Fatalf("mixed unavailable collection = %d visible files", v.FileCount())
	}
	files := (favoriteListHost{v}).CurrentFiles()
	if len(files) != 2 {
		t.Fatalf("saving Favorite loses unavailable HEIC membership: %v", files)
	}
	if !strings.Contains(v.toast.text.Text, "1 HEIC") {
		t.Fatalf("mixed scan has no aggregate HEIC notice: %q", v.toast.text.Text)
	}
	available.Store(true)
	v.startHEICCheck(true)
	v.settleHEIC()
	if v.FileCount() != 1 {
		t.Fatal("support refresh changed the current collection")
	}
}

func TestHEICUnavailableGuide(t *testing.T) {
	v := newTestViewer(t)
	v.startHEICCheck(false)
	v.settleHEIC()
	path := filepath.Join(t.TempDir(), "unavailable.heic")
	if err := os.WriteFile(path, []byte("unavailable"), 0600); err != nil {
		t.Fatal(err)
	}
	v.handleDrop([]fyne.URI{storage.NewFileURI(path)})
	waitForScan(t, v)
	if !strings.Contains(v.toast.text.Text, "HEIC") {
		t.Fatalf("no HEIC-specific explanation: %q", v.toast.text.Text)
	}
	button := explorerDialogButton(t, v, "HEIC installation guide")
	test.Tap(button)
	found := false
	for _, window := range v.app.Driver().AllWindows() {
		found = found || window.Title() == "HEIC installation instructions"
	}
	if !found {
		t.Fatal("unavailable open did not offer the actual installation guide")
	}
}

func TestHEICBackendLossPreservesSession(t *testing.T) {
	v := newTestViewer(t)
	var checks atomic.Int32
	v.configureHEIC(testHEICBackend{check: func(_ context.Context) error {
		if checks.Add(1) == 1 {
			return nil
		}
		return context.DeadlineExceeded
	}})
	v.heic.ui = &uitest.UIQueue{}
	v.startHEICCheck(false)
	v.settleHEIC()
	data, err := os.ReadFile("../imaging/testdata/test_exif.heic")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "retained.heic")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	v.handleCollectionDrop([]fyne.URI{storage.NewFileURI(path)}, t.TempDir())
	waitForScan(t, v)
	waitForSort(t, v)
	waitUntilLoaded(t, v)
	v.settleHEIC()
	if checks.Load() != 2 {
		t.Fatalf("backend loss checks=%d, want one recheck", checks.Load())
	}
	if saved := preferences.LoadHEICObservation(v.app); saved.Available || !saved.CheckedAt.IsZero() {
		t.Fatalf("failed recheck retained invalidated observation: %+v", saved)
	}
	lifecycle := v.app.Lifecycle().(interface{ OnStopped() func() })
	previous := lifecycle.OnStopped()
	registerShutdown(v.app, v)
	shutdown := lifecycle.OnStopped()
	v.app.Lifecycle().SetOnStopped(previous)
	shutdown()
	files := session.Load(v.app)
	if len(files) != 1 || files[0].Path() != path {
		t.Fatalf("provider loss erased saved session membership: %v", files)
	}
}

func TestHEICCaptureDateSort(t *testing.T) {
	v := newTestViewer(t)
	exif, err := hex.DecodeString("49492a0008000000010032010200140000001a00000000000000323032303a30313a30322030333a30343a303500")
	if err != nil {
		t.Fatal(err)
	}
	v.configureHEIC(testHEICBackend{check: func(_ context.Context) error { return nil }, read: func(_ context.Context, _ []byte, request heic.Request) (heic.Result, error) {
		result := heic.Result{Width: 2, Height: 1, EXIF: exif}
		if request.Pixels {
			result.Stride = 8
			result.Pixels = []byte{255, 0, 0, 255, 0, 255, 0, 255}
		}
		return result, nil
	}})
	v.heic.ui = &uitest.UIQueue{}
	v.startHEICCheck(false)
	v.settleHEIC()
	data, err := os.ReadFile("../imaging/testdata/test_exif.heic")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "z.heic")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	jpeg := uitest.TempJPEGURI(t, "a.jpg", 2, 1, color.White)
	modified := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	if err := os.Chtimes(jpeg.Path(), modified, modified); err != nil {
		t.Fatal(err)
	}
	dropAndWait(t, v, jpeg, storage.NewFileURI(path))
	v.SetSortMode(filesort.ByCaptureDate)
	waitForSort(t, v)
	waitUntilLoaded(t, v)
	if v.FileAt(0).Path() != path {
		t.Fatalf("capture-date sort ignored HEIC metadata: first=%s", v.FileAt(0).Name())
	}
}

type unavailableHEICBackend struct{}

func (unavailableHEICBackend) Check(_ context.Context) error { return heic.ErrUnavailable }
func (unavailableHEICBackend) Read(_ context.Context, _ []byte, _ heic.Request) (heic.Result, error) {
	return heic.Result{}, heic.ErrUnavailable
}
