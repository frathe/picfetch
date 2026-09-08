package ui

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"os"
	"reflect"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestImageCacheWriters_PreserveCompleteRecords(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
		exif bool
	}{
		{"plain.jpg", uitest.EncodeJPEG(t, 19, 13, color.White), false},
		{"gps.jpg", uitest.GPSJPEG(t, 19, 13, 50.85, 4.35), true},
		// Orientation alone has no fields to display in the EXIF window.
		{"oriented.jpg", uitest.EncodeOrientedJPEG(t, 19, 13, 6), false},
		{"preview.cr2", uitest.EncodeRAWPreview(t, uitest.RAWPreview{Width: 19, Height: 13, Orientation: 6, Make: "Camera"}), true},
		{"animated.gif", uitest.EncodeAnimatedGIF(t, 19, 13, []color.Color{color.White, color.Black}, []int{100, 100}), false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			u := storage.NewFileURI(uitest.WriteTempFile(t, tc.name, tc.data))
			var foreground *imaging.LoadedImage
			for _, path := range []string{"foreground", "preload", "comparison"} {
				t.Run(path, func(t *testing.T) {
					v := newTestViewer(t)
					parkAnimate(v)
					switch path {
					case "foreground":
						dropAndWait(t, v, u)
					case "preload":
						v.preloadOne(v.loadLifecycle.begin(), u)
						v.preloads.Wait()
					case "comparison":
						if _, err := v.loadComparedImage(context.Background(), u); err != nil {
							t.Fatal(err)
						}
					}
					loaded, ok := v.imgCache.Get(u.String())
					if !ok {
						t.Fatal("image was not cached")
					}
					if loaded.FileSize != int64(len(tc.data)) || loaded.HasEXIF != tc.exif {
						t.Errorf("cached size/EXIF = %d/%v, want %d/%v", loaded.FileSize, loaded.HasEXIF, len(tc.data), tc.exif)
					}
					if path == "foreground" {
						foreground = loaded
					} else if !reflect.DeepEqual(loaded, foreground) {
						t.Error("cached frames, delays, preview markers or file facts differ from foreground")
					}
				})
			}
		})
	}
}

func TestCompareFirstNavigation_ShowsCachedSizeAndEXIF(t *testing.T) {
	v, win, _ := newTestUI(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 19, 13, color.White)
	data := uitest.GPSJPEG(t, 19, 13, 50.85, 4.35)
	b := storage.NewFileURI(uitest.WriteTempFile(t, "b.jpg", data))
	v.state.setFiles([]fyne.URI{a, b}, []fyne.URI{a, b})
	v.compare.Open([2]fyne.URI{a, b})
	waitForCompare(t, v)
	v.compare.Close()
	// Prove navigation serves the comparison record, without a fresh decode
	// repairing its facts as a side effect.
	if err := os.Remove(b.Path()); err != nil {
		t.Fatal(err)
	}
	v.ShowImage(1)
	waitUntilLoaded(t, v)
	v.toggleInfoOverlay()
	want := fmt.Sprintf("b.jpg  (2/2)\n19 x 13\n%s\nZoom: %d%%", wantSizeText(int64(len(data))), v.zoom.Percent())
	if got := v.info.Text().Text; got != want {
		t.Errorf("info = %q, want %q", got, want)
	}
	for _, object := range []fyne.CanvasObject{v.info.Object(), v.info.Text(), v.info.ExifLink()} {
		if !containsObject(win.Content(), object) || !object.Visible() {
			t.Errorf("info object %T must be visible in the window content tree", object)
		}
	}
}

// This file covers the decoded-image cache and speculative neighbor
// preloading, plus the behavior the byte budget produces. The two halves
// belong together because they are the same mechanism seen from two sides:
// imgCache retains decodes, preloadOne speculatively fills it, and the byte
// budget decides what survives.
//
// Two design points worth keeping in mind while reading these tests:
// ByteCache never evicts its most recently added entry, and attemptLoad adds
// the image it is about to display, so an image larger than the entire
// budget still displays - a budget smaller than one decode must not make the
// app unable to show that image. An over-large neighbor, by contrast, is
// refused on the header probe alone, before any decode, so preloading cannot
// multiply one oversized image into several retained decodes. And note the
// deliberate asymmetry between the two failure modes: an oversized file is
// refused outright, but an oversized animation is kept on screen as a static
// frame - the image is valid, so only the motion is given up.
//
// memlimits_test.go is where the settings window's getters/setters for these
// budgets are tested; this file is the behavior those budgets produce, that
// one is the binding surface.

func TestFinishLoad_PreloadsBothNeighbors(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	c := uitest.TempJPEGURI(t, "c.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b, c)

	v.ShowImage(1) // b, with a genuine neighbor on each side
	waitUntilLoaded(t, v)

	waitForCached(t, v, a)
	waitForCached(t, v, c)
}

// TestAttemptLoad_CacheHitServesFileRemovedFromDisk proves a cache hit
// really does skip the disk read: b's file is deleted from disk right after
// it's preloaded, so a real (non-cached) load of it would fail and trigger
// retryAfterLoadFailure, dropping it from v.state.files. Navigating to it
// succeeding instead demonstrates the display came from imgCache.
func TestAttemptLoad_CacheHitServesFileRemovedFromDisk(t *testing.T) {
	v := newTestViewer(t)

	aPath := uitest.WriteTempFile(t, "a.jpg", uitest.EncodeJPEG(t, 4, 4, color.White))
	bPath := uitest.WriteTempFile(t, "b.jpg", uitest.EncodeJPEG(t, 4, 4, color.White))
	a := storage.NewFileURI(aPath)
	b := storage.NewFileURI(bPath)

	dropAndWait(t, v, a, b)

	waitForCached(t, v, b)

	if err := os.Remove(bPath); err != nil {
		t.Fatal(err)
	}

	v.ShowImage(1)
	waitUntilLoaded(t, v)

	if v.state.index != 1 {
		t.Fatalf("index = %d, want 1 - a cache hit must not fall through to retryAfterLoadFailure", v.state.index)
	}
	if len(v.state.files) != 2 {
		t.Fatalf("files = %v, want b still present - a cache hit must not treat it as broken", v.state.files)
	}
}

func TestRemoveFile_PurgesCacheEntry(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	v.state.files = []fyne.URI{a, b}
	v.state.unsortedFiles = []fyne.URI{a, b}
	v.imgCache.Add(a.String(), &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 1, 1))}})

	v.RemoveFile(0)

	if v.imgCache.Contains(a.String()) {
		t.Error("RemoveFile should purge the removed file's imgCache entry")
	}
}

// TestAppState_RemoveFileEvictsCacheWithoutCallerAsking proves the eviction
// is appState's own invariant, not RemoveFile's - it calls the low-level
// mutator v.state.removeFile directly, bypassing v.RemoveFile entirely, so a
// future mutator that goes through appState gets the same guarantee for free.
func TestAppState_RemoveFileEvictsCacheWithoutCallerAsking(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	v.state.files = []fyne.URI{a, b}
	v.state.unsortedFiles = []fyne.URI{a, b}
	v.imgCache.Add(a.String(), &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 1, 1))}})

	v.state.removeFile(0)

	if v.imgCache.Contains(a.String()) {
		t.Error("appState.removeFile should evict the removed file's imgCache entry via its onRemove hook")
	}
}

// TestAttemptLoad_DisplaysAnImageLargerThanTheWholeCacheBudget is the
// completion criterion the byte budget had to be designed around: a budget
// smaller than a single decode must not make the app unable to show that
// image. ByteCache never evicts its most recently added entry, and
// attemptLoad adds the image it is about to display, so the one on screen is
// always the survivor.
func TestAttemptLoad_DisplaysAnImageLargerThanTheWholeCacheBudget(t *testing.T) {
	v := newTestViewer(t)

	// One byte - past anything the settings window allows (it floors at
	// 1 MB), so this is the extreme the cache itself still has to handle.
	v.imgCache.SetBudget(1)

	u := uitest.TempJPEGURI(t, "big.jpg", 64, 64, color.White)
	dropAndWait(t, v, u)

	if v.img.Image == nil {
		t.Fatal("no image displayed - an image larger than the whole cache budget must still show")
	}
	if v.display.Count() != 1 {
		t.Errorf("display.Count() = %d, want 1", v.display.Count())
	}
	if !v.imgCache.Contains(u.String()) {
		t.Error("the displayed image should still be cached - ByteCache never evicts its newest entry")
	}
}

// TestPreloadOne_SkipsANeighborTooLargeForTheBudget covers the other
// completion criterion: neighbor preloading must not multiply one oversized
// image into several retained decodes. The bail happens on the header alone,
// before the decode, so an over-large neighbor costs nothing but the probe.
func TestPreloadOne_SkipsANeighborTooLargeForTheBudget(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 64, 64, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 64, 64, color.White)

	// 64x64 estimates at 16,384 decoded bytes (4 per pixel). A 16 KiB budget
	// puts that exactly at the budget and so past the half-budget line
	// preloadOne bails at - the point where the current image and one
	// neighbor stop both fitting.
	v.imgCache.SetBudget(16 * 1024)

	dropAndWait(t, v, a, b)

	if !v.imgCache.Contains(a.String()) {
		t.Error("the displayed image should be cached")
	}
	if v.imgCache.Contains(b.String()) {
		t.Error("a neighbor whose decode would evict the current image should not have been preloaded")
	}
}

// TestAttemptLoad_ReportsAFileTooLargeToOpen wires imaging's
// *InputTooLargeError through attemptLoad's errors.As branch to its own
// message - distinct from the "invalid image dimensions" one, since the file
// here is a perfectly valid JPEG that is merely bigger than the limit.
func TestAttemptLoad_ReportsAFileTooLargeToOpen(t *testing.T) {
	v := newTestViewer(t)

	u := uitest.TempJPEGURI(t, "big.jpg", 64, 64, color.White)

	imaging.SetMaxEncodedBytes(1)
	t.Cleanup(func() { imaging.SetMaxEncodedBytes(0) })

	dropAndWait(t, v, u)

	if v.img.Image != nil {
		t.Error("no image should be loaded after a file is refused for its size")
	}
	if len(v.state.files) != 0 {
		t.Errorf("files = %v, want the refused file dropped from the set", v.state.files)
	}
	if !v.toast.card.Visible() {
		t.Fatal("expected a toast after a file was refused for its size")
	}
	if got, want := v.toast.text.Text, fmt.Sprintf(lang.L("%q is too large to open"), "big.jpg"); got != want {
		t.Errorf("toast text = %q, want %q", got, want)
	}
	settleToast(t, v)
}

// TestAttemptLoad_ToastsAndFallsBackToAStaticFrameForAnOversizedAnimation
// covers the deliberate choice not to refuse an over-budget animation the way
// an oversized file is refused: the image is valid, so it stays in the set and
// on screen, and only the motion is given up.
func TestAttemptLoad_ToastsAndFallsBackToAStaticFrameForAnOversizedAnimation(t *testing.T) {
	v := newTestViewer(t)

	anim := storage.NewFileURI(uitest.WriteTempFile(t, "anim.gif", uitest.EncodeAnimatedGIF(t, 20, 20,
		[]color.Color{color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255}},
		[]int{50, 50})))

	// One frame of this GIF is 20*20*4 = 1600 bytes, so a 1000-byte budget
	// can't hold even one - let alone both composited frames.
	v.imgCache.SetBudget(1000)

	dropAndWait(t, v, anim)

	if v.img.Image == nil {
		t.Fatal("an over-budget animation must still display its first frame")
	}
	if v.display.Count() != 1 {
		t.Errorf("display.Count() = %d, want 1 - the animation should not have been composited", v.display.Count())
	}
	if v.anim.Begun() {
		t.Error("the animation signal is armed, want no animation goroutine for a refused animation")
	}
	if len(v.state.files) != 1 {
		t.Errorf("files = %v, want the file kept - it is valid, just too big to animate", v.state.files)
	}
	if !v.toast.card.Visible() {
		t.Fatal("expected a toast explaining why the animation isn't playing")
	}
	if got, want := v.toast.text.Text, fmt.Sprintf(lang.L("animation in %q is too large to play"), "anim.gif"); got != want {
		t.Errorf("toast text = %q, want %q", got, want)
	}
	settleToast(t, v)
}

func TestClearToDropzone_PurgesTheImageCache(t *testing.T) {
	v := newTestViewer(t)

	u := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	dropAndWait(t, v, u)

	if !v.imgCache.Contains(u.String()) {
		t.Fatal("the displayed image should be cached before the file set is closed")
	}

	v.closeFiles()

	if v.imgCache.Bytes() != 0 {
		t.Errorf("imgCache holds %d bytes after closing the file set, want 0", v.imgCache.Bytes())
	}
}
