// canSetWallpaper/setAsWallpaper (wallpaper.go): the Actions > "Set as
// Wallpaper" action, which writes the frame on screen into the app's own
// cache directory and points the OS at that copy.
//
// Per-OS dispatch (gsettings/plasma-apply-wallpaperimage/PowerShell/AppKit)
// is covered by internal/wallpaper's own tests; what's here is the viewer's
// side - the enable rule, the copy it hands over, the sweep of the file it
// replaces, and the error path.

package ui

import (
	"context"
	"errors"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/displays"
	"github.com/frathe/picfetch/internal/mosaic"
	"github.com/frathe/picfetch/internal/uitest"
	"github.com/frathe/picfetch/internal/wallpaper"
)

// settleWallpaper waits out the goroutine setAsWallpaper runs on - wallpaper
// is finished once that goroutine has fully run, toast included, so
// reading widget state afterwards is race-free.
func settleWallpaper(t *testing.T, v *viewer) {
	t.Helper()

	if !v.wallpaper.Begun() {
		t.Fatal("no wallpaper goroutine pending to settle")
	}

	waitFor(t, "the wallpaper change", &v.wallpaper)
}

// wallpaperFiles lists what setAsWallpaper has left in the viewer's cache
// directory, which is a t.TempDir() in every test here.
func wallpaperFiles(t *testing.T, v *viewer) []string {
	t.Helper()

	found, err := filepath.Glob(filepath.Join(v.wallpaperDir, "wallpaper-*.png"))
	if err != nil {
		t.Fatalf("glob the wallpaper directory: %v", err)
	}
	return found
}

// --- canSetWallpaper -----------------------------------------------------

func TestCanSetWallpaper_FalseWithNoImage(t *testing.T) {
	v := newTestViewer(t)

	if v.canSetWallpaper() {
		t.Error("canSetWallpaper should be false with nothing loaded")
	}
}

func TestCanSetWallpaper_TrueForALoadedImage(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))

	if !v.canSetWallpaper() {
		t.Error("canSetWallpaper should be true once an image is loaded")
	}
}

// TestCanSetWallpaper_TrueForAFormatWithNoEncoder mirrors canExport's own
// case, and for the same reason: the copy this writes is a PNG whatever the
// source was, so a decode-only format is no obstacle.
func TestCanSetWallpaper_TrueForAFormatWithNoEncoder(t *testing.T) {
	v := newTestViewer(t)
	path := uitest.WriteTempFile(t, "a.webp", uitest.EncodeJPEG(t, 4, 4, color.White))
	dropAndWait(t, v, storage.NewFileURI(path))

	if v.canSaveRotation() {
		t.Fatal("canSaveRotation should be false for .webp - the premise of this test")
	}
	if !v.canSetWallpaper() {
		t.Error("canSetWallpaper should be true for a format with no encoder of its own")
	}
}

// TestCanSetWallpaper_FalseWhileLoading mirrors canExport's own: mid-load
// v.img.Image still holds the previous file's pixels, and those are what
// would land on the desktop.
func TestCanSetWallpaper_FalseWhileLoading(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))

	v.loading.Store(true)
	t.Cleanup(func() { v.loading.Store(false) })

	if v.canSetWallpaper() {
		t.Error("canSetWallpaper should be false while a load is in flight")
	}
}

// --- setAsWallpaper ------------------------------------------------------

// TestSetAsWallpaper_PointsTheOSAtACopyOfTheFrameOnScreen is the whole
// feature in one test: what the OS is handed is a PNG this app wrote, not
// the user's own file, and it holds the frame as displayed - rotation
// included, which is why the source here is asymmetric and rotated first.
func TestSetAsWallpaper_PointsTheOSAtACopyOfTheFrameOnScreen(t *testing.T) {
	v := newTestViewer(t)
	path := uitest.WriteTempFile(t, "a.png", uitest.EncodePNG(t, 4, 2, color.White)) // asymmetric
	dropAndWait(t, v, storage.NewFileURI(path))
	v.rotateBy(1) // 4x2 -> 2x4

	var got string
	uitest.StubWallpaperSet(t, func(p string) error {
		got = p
		return nil
	})

	v.setAsWallpaper()
	settleWallpaper(t, v)

	if got == path {
		t.Fatal("the OS was pointed at the source file itself, want a copy this app owns")
	}
	if dir := filepath.Dir(got); dir != v.wallpaperDir {
		t.Errorf("wallpaper written to %q, want it inside the app's own cache directory %q", dir, v.wallpaperDir)
	}

	written, err := loadExported(t, got)
	if err != nil {
		t.Fatalf("load the written wallpaper: %v", err)
	}
	if b := written.Bounds(); b.Dx() != 2 || b.Dy() != 4 {
		t.Errorf("wallpaper bounds = %v, want 2x4 (the rotation on screen carried into the file)", b)
	}

	settleToast(t, v) // a successful change toasts
}

// TestSetAsWallpaper_SweepsTheFileItReplaced keeps the cache directory from
// growing one PNG per invocation. The name carries a timestamp rather than
// being fixed because macOS caches the desktop picture by path and can
// ignore a same-path content change, so the sweep is what bounds it.
func TestSetAsWallpaper_SweepsTheFileItReplaced(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	uitest.StubWallpaperSet(t, func(string) error { return nil })

	v.setAsWallpaper()
	settleWallpaper(t, v)
	settleToast(t, v)
	first := wallpaperFiles(t, v)
	if len(first) != 1 {
		t.Fatalf("wallpaper files after one call = %v, want exactly 1", first)
	}

	v.setAsWallpaper()
	settleWallpaper(t, v)
	settleToast(t, v)

	second := wallpaperFiles(t, v)
	if len(second) != 1 {
		t.Errorf("wallpaper files after two calls = %v, want exactly 1 - the previous one should be swept", second)
	}
	if second[0] == first[0] {
		t.Errorf("second call reused %q, want a fresh name", second[0])
	}
	if _, err := os.Stat(first[0]); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("os.Stat(%q) error = %v, want the replaced file to be gone", first[0], err)
	}
}

// TestSetAsWallpaper_ReportsAFailure covers the Linux "no tool installed"
// case and every other refusal the OS can hand back: the file is written
// either way, so without a toast the user would have no sign at all that
// their desktop didn't change.
func TestSetAsWallpaper_ReportsAFailure(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	uitest.StubWallpaperSet(t, func(string) error { return errors.New("no such schema") })

	v.setAsWallpaper()
	settleWallpaper(t, v)

	if !v.toast.card.Visible() {
		t.Fatal("no toast shown after a failed wallpaper change")
	}
	if msg := v.toast.text.Text; !strings.Contains(msg, "no such schema") {
		t.Errorf("toast = %q, want it to carry the reason the change failed", msg)
	}

	settleToast(t, v)
}

func TestSetAsWallpaper_DoesNothingWithoutAnImage(t *testing.T) {
	v := newTestViewer(t)
	uitest.StubWallpaperSet(t, func(string) error {
		t.Error("the wallpaper should not be changed with nothing loaded")
		return nil
	})

	v.setAsWallpaper()

	if v.wallpaper.Begun() {
		t.Error("setAsWallpaper started a goroutine with nothing loaded")
	}
}

func TestMosaicWallpaper_PassesExactResultAndOpaqueTarget(t *testing.T) {
	v := newTestViewer(t)
	result := testMosaicResult(t, color.NRGBA{R: 230, G: 40, B: 20, A: 255})
	var got wallpaper.Request
	uitest.StubWallpaperSet(t, func(request wallpaper.Request) error {
		got = request
		return nil
	})

	if err := v.SetMosaicWallpaper(context.Background(), result, displays.ID("opaque/display\\path")); err != nil {
		t.Fatal(err)
	}
	if got.Target != displays.ID("opaque/display\\path") {
		t.Fatalf("wallpaper target = %q, want the opaque selected ID", got.Target)
	}
	written, err := loadExported(t, got.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !imagesMatch(written, result.Image()) {
		t.Fatal("wallpaper copy differs from the latest mosaic result")
	}
	if strings.Contains(filepath.Base(got.Path), string(got.Target)) {
		t.Fatalf("wallpaper filename %q exposes the raw display ID", filepath.Base(got.Path))
	}
}

func TestMosaicWallpaper_CleanupIsScopedPerTarget(t *testing.T) {
	v := newTestViewer(t)
	var requests []wallpaper.Request
	uitest.StubWallpaperSet(t, func(request wallpaper.Request) error {
		requests = append(requests, request)
		return nil
	})
	first := testMosaicResult(t, color.NRGBA{R: 255, A: 255})
	second := testMosaicResult(t, color.NRGBA{B: 255, A: 255})

	if err := v.SetMosaicWallpaper(context.Background(), first, "display-a"); err != nil {
		t.Fatal(err)
	}
	if err := v.SetMosaicWallpaper(context.Background(), second, "display-b"); err != nil {
		t.Fatal(err)
	}
	if got := wallpaperFiles(t, v); len(got) != 2 {
		t.Fatalf("two target files = %v, want 2", got)
	}
	firstA, firstB := requests[0].Path, requests[1].Path
	if err := v.SetMosaicWallpaper(context.Background(), second, "display-a"); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(firstA); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("replaced target A file still exists: %v", err)
	}
	if _, err := os.Stat(firstB); err != nil {
		t.Fatalf("target B file was swept while replacing A: %v", err)
	}
	if got := wallpaperFiles(t, v); len(got) != 2 {
		t.Fatalf("files after replacing A = %v, want current A and B", got)
	}

	// The ordinary no-target action deliberately restores the legacy
	// one-file cache policy and may sweep every older targeted copy.
	dropAndWait(t, v, uitest.TempJPEGURI(t, "ordinary.jpg", 4, 4, color.White))
	v.setAsWallpaper()
	settleWallpaper(t, v)
	settleToast(t, v)
	if got := wallpaperFiles(t, v); len(got) != 1 {
		t.Fatalf("files after global replacement = %v, want 1", got)
	}
}

func TestMosaicWallpaper_RejectsOverlapAndDeletesOnlyFailedCopy(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "ordinary.jpg", 4, 4, color.White))
	result := testMosaicResult(t, color.White)
	started := make(chan struct{})
	release := make(chan struct{})
	uitest.StubWallpaperSet(t, func(wallpaper.Request) error {
		close(started)
		<-release
		return errors.New("native failure")
	})
	done := make(chan error, 1)
	go func() { done <- v.SetMosaicWallpaper(context.Background(), result, "display-a") }()
	<-started

	err := v.SetMosaicWallpaper(context.Background(), result, "display-b")
	if !errors.Is(err, errWallpaperBusy) {
		t.Fatalf("overlapping wallpaper call = %v, want errWallpaperBusy", err)
	}
	v.setAsWallpaper()
	if v.wallpaper.Begun() {
		t.Fatal("overlapping main-window action started a second wallpaper worker")
	}
	if !v.toast.card.Visible() || !strings.Contains(v.toast.text.Text, "already in progress") {
		t.Fatalf("overlapping main-window status = %q", v.toast.text.Text)
	}
	close(release)
	if err := <-done; err == nil || !strings.Contains(err.Error(), "native failure") {
		t.Fatalf("first wallpaper result = %v", err)
	}
	if got := wallpaperFiles(t, v); len(got) != 0 {
		t.Fatalf("failed/overlapping calls left cache files: %v", got)
	}
	settleToast(t, v)
}

func testMosaicResult(t *testing.T, fill color.Color) mosaic.Result {
	t.Helper()
	request, err := mosaic.NewRequest(
		[]fyne.URI{uitest.TempJPEGURI(t, "mosaic-source.jpg", 10, 8, fill)},
		image.Pt(40, 24),
		mosaic.DefaultSettings(),
		42,
	)
	if err != nil {
		t.Fatal(err)
	}
	result, err := mosaic.Generate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func imagesMatch(a, b image.Image) bool {
	if a == nil || b == nil || a.Bounds() != b.Bounds() {
		return false
	}
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			ar, ag, ab, aa := a.At(x, y).RGBA()
			br, bg, bb, ba := b.At(x, y).RGBA()
			if ar != br || ag != bg || ab != bb || aa != ba {
				return false
			}
		}
	}
	return true
}

// --- menu state ----------------------------------------------------------

func TestSyncMenus_TracksTheWallpaperItem(t *testing.T) {
	v := newTestViewer(t)

	if !v.menus.Actions().Wallpaper().Disabled {
		t.Error("the Set as Wallpaper item should start disabled")
	}

	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	if v.menus.Actions().Wallpaper().Disabled {
		t.Error("the Set as Wallpaper item should be enabled once an image is loaded")
	}

	v.closeFiles()
	if !v.menus.Actions().Wallpaper().Disabled {
		t.Error("the Set as Wallpaper item should be disabled again once the files are closed")
	}
}
