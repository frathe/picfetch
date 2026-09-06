package imaging

import (
	"image"
	"image/color"
	"testing"

	"fyne.io/fyne/v2/storage"
)

// --- scaleToFit -----------------------------------------------------------

func TestScaleToFit_DownscalesLandscapePreservingAspect(t *testing.T) {
	img := imageOfSize(t, 4000, 2000)

	got := scaleToFit(img, ThumbnailSize)

	b := got.Bounds()
	if b.Dx() != ThumbnailSize {
		t.Errorf("width = %d, want %d (the longer edge)", b.Dx(), ThumbnailSize)
	}
	if b.Dy() != ThumbnailSize/2 {
		t.Errorf("height = %d, want %d (aspect preserved)", b.Dy(), ThumbnailSize/2)
	}
}

func TestScaleToFit_DownscalesPortraitPreservingAspect(t *testing.T) {
	img := imageOfSize(t, 1000, 3000)

	got := scaleToFit(img, ThumbnailSize)

	b := got.Bounds()
	if b.Dy() != ThumbnailSize {
		t.Errorf("height = %d, want %d (the longer edge)", b.Dy(), ThumbnailSize)
	}
	if b.Dx() != ThumbnailSize/3 {
		t.Errorf("width = %d, want %d (aspect preserved)", b.Dx(), ThumbnailSize/3)
	}
}

func TestScaleToFit_LeavesAlreadySmallImageUnscaled(t *testing.T) {
	img := imageOfSize(t, 50, 30)

	got := scaleToFit(img, ThumbnailSize)

	if got != img {
		t.Errorf("scaleToFit changed an already-small image instead of returning it unchanged")
	}
}

func imageOfSize(t *testing.T, w, h int) *image.RGBA {
	t.Helper()
	return image.NewRGBA(image.Rect(0, 0, w, h))
}

// --- NewThumbCache ----------------------------------------------------------

func TestNewThumbCache_IsEmptyAndUsable(t *testing.T) {
	c := NewThumbCache(DefaultThumbCacheBytes)

	if c.Len() != 0 {
		t.Fatalf("Len() = %d, want 0 for a fresh cache", c.Len())
	}

	c.Add("key", imageOfSize(t, 1, 1))
	if c.Len() != 1 {
		t.Errorf("Len() = %d, want 1 after Add", c.Len())
	}
}

// --- LoadThumbnail ----------------------------------------------------------

func TestLoadThumbnail_DecodesAndDownsamples(t *testing.T) {
	path := writeTempFile(t, "photo.jpg", encodeJPEG(t, 800, 400, color.RGBA{R: 200, G: 20, B: 20, A: 255}))

	thumb, err := LoadThumbnail(storage.NewFileURI(path))
	if err != nil {
		t.Fatalf("LoadThumbnail returned error: %v", err)
	}

	b := thumb.Bounds()
	if b.Dx() != ThumbnailSize || b.Dy() != ThumbnailSize/2 {
		t.Errorf("thumbnail size = %dx%d, want %dx%d", b.Dx(), b.Dy(), ThumbnailSize, ThumbnailSize/2)
	}
}

// TestLoadThumbnail_DoesNotCompositeEveryAnimationFrame guards the zero
// animation budget LoadThumbnail passes. The two paths are told apart by the
// concrete type rather than by pixels, since both would end up showing frame
// 0's content: compositing runs every frame through copyRGBA and so yields an
// *image.RGBA, while the static path hands back what gif.Decode produced - a
// paletted first frame, untouched, because ApplyOrientation is a no-op for
// orientation 1 and the image is already inside ThumbnailSize so scaleToFit
// returns it unchanged. Before the budget existed, a long animation composited
// every frame to a full canvas here just to keep one and discard the rest.
func TestLoadThumbnail_DoesNotCompositeEveryAnimationFrame(t *testing.T) {
	red := color.RGBA{R: 255, A: 255}
	blue := color.RGBA{B: 255, A: 255}

	path := writeTempFile(t, "anim.gif",
		encodeAnimatedGIF(t, 20, 20, []color.Color{red, blue, blue}, []int{5, 5, 5}))

	thumb, err := LoadThumbnail(storage.NewFileURI(path))
	if err != nil {
		t.Fatalf("LoadThumbnail returned error: %v", err)
	}

	if _, composited := thumb.(*image.RGBA); composited {
		t.Error("thumbnail is an *image.RGBA, which means every frame was composited before one was kept")
	}

	// Still the first frame's content, which is what the grid should show.
	if r, _, _, _ := thumb.At(10, 10).RGBA(); r == 0 {
		t.Error("thumbnail should be the animation's first (red) frame")
	}
}

func TestLoadThumbnail_PropagatesDecodeError(t *testing.T) {
	path := writeTempFile(t, "broken.jpg", []byte("not an image"))

	if _, err := LoadThumbnail(storage.NewFileURI(path)); err == nil {
		t.Error("LoadThumbnail returned nil error for an unparseable file, want an error")
	}
}

func TestFitEdge(t *testing.T) {
	for _, tc := range []struct {
		name         string
		w, h         int
		wantW, wantH int
	}{
		{"wide", 800, 600, 200, 150},
		{"tall", 600, 800, 150, 200},
		{"already small enough is never upscaled", 120, 80, 120, 80},
		{"extreme aspect floors at one", 100000, 3, 200, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, h := fitEdge(tc.w, tc.h, 200)
			if w != tc.wantW || h != tc.wantH {
				t.Fatalf("fitEdge(%d, %d, 200) = %dx%d, want %dx%d",
					tc.w, tc.h, w, h, tc.wantW, tc.wantH)
			}
		})
	}
}

// TestFitEdge_NoCeilingIsTheSizeItself covers the value the export path
// spells "Original": a maxEdge of zero (or less) is not a ceiling of zero,
// it is no ceiling at all. Without this the arithmetic below would floor
// every side at one and report a 1x1 image.
func TestFitEdge_NoCeilingIsTheSizeItself(t *testing.T) {
	for _, maxEdge := range []int{0, -1} {
		w, h := fitEdge(800, 600, maxEdge)
		if w != 800 || h != 600 {
			t.Errorf("fitEdge(800, 600, %d) = %dx%d, want the size itself", maxEdge, w, h)
		}
	}
}

// TestSizeLimitApplies covers the predicate both the export prompt and the
// encoder read: whether a ceiling would change anything at all. It is what
// decides whether the size joins the suggested filename and the toast, and
// whether the source's dimension tags have been made false.
func TestSizeLimitApplies(t *testing.T) {
	for _, tc := range []struct {
		name    string
		w, h    int
		maxEdge int
		want    bool
	}{
		{"a ceiling under the photo", 3000, 2000, 2400, true},
		{"a ceiling over the photo", 1800, 1200, 2400, false},
		{"a ceiling exactly on the photo", 2400, 1600, 2400, false},
		{"no ceiling at all", 3000, 2000, 0, false},
		{"a ceiling under the photo's short edge only", 2000, 3000, 2400, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := SizeLimitApplies(image.Rect(0, 0, tc.w, tc.h), tc.maxEdge)
			if got != tc.want {
				t.Errorf("SizeLimitApplies(%dx%d, %d) = %v, want %v", tc.w, tc.h, tc.maxEdge, got, tc.want)
			}
		})
	}
}

func TestLoadThumbnailRastersSVGAtThumbnailSize(t *testing.T) {
	// 800x600 is its own logical size (over the 520x340 floor), so the
	// thumbnail must come back at 200x150 - and with drawn pixels, proving
	// the direct RasterAt path actually rendered rather than scaled.
	path := writeTempFile(t, "thumb.svg", []byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 600"><rect width="800" height="600" fill="#ff0000"/></svg>`))

	img, err := LoadThumbnail(storage.NewFileURI(path))
	if err != nil {
		t.Fatalf("LoadThumbnail: %v", err)
	}
	if b := img.Bounds(); b.Dx() != 200 || b.Dy() != 150 {
		t.Fatalf("thumbnail = %dx%d, want 200x150", b.Dx(), b.Dy())
	}
	if _, _, _, a := img.At(100, 75).RGBA(); a == 0 {
		t.Fatal("centre pixel is transparent - nothing was drawn")
	}
}

func TestLoadThumbnailAndBounds_NativeSizeIsNotThumbnailSize(t *testing.T) {
	path := writeTempFile(t, "photo.jpg", encodeJPEG(t, 800, 400, color.RGBA{R: 200, G: 20, B: 20, A: 255}))
	u := storage.NewFileURI(path)

	thumb, native, err := LoadThumbnailAndBounds(u)
	if err != nil {
		t.Fatalf("LoadThumbnailAndBounds: %v", err)
	}
	if native.Dx() != 800 || native.Dy() != 400 {
		t.Errorf("native = %dx%d, want 800x400", native.Dx(), native.Dy())
	}
	tb := thumb.Bounds()
	if tb.Dx() != ThumbnailSize || tb.Dy() != ThumbnailSize/2 {
		t.Errorf("thumb = %dx%d, want %dx%d", tb.Dx(), tb.Dy(), ThumbnailSize, ThumbnailSize/2)
	}
}

func TestLoadThumbnailAndBounds_AccountsForEXIFOrientation(t *testing.T) {
	path := writeTempFile(t, "rotated.jpg", halfRedHalfBlueJPEG(t, 20, 10, 6))

	_, native, err := LoadThumbnailAndBounds(storage.NewFileURI(path))
	if err != nil {
		t.Fatalf("LoadThumbnailAndBounds: %v", err)
	}
	if native.Dx() != 10 || native.Dy() != 20 {
		t.Errorf("native = %dx%d, want 10x20 after orientation 6", native.Dx(), native.Dy())
	}
}

func TestLoadThumbnail_WrapsLoadThumbnailAndBounds(t *testing.T) {
	path := writeTempFile(t, "photo.jpg", encodeJPEG(t, 800, 400, color.RGBA{R: 200, G: 20, B: 20, A: 255}))
	u := storage.NewFileURI(path)

	a, err := LoadThumbnail(u)
	if err != nil {
		t.Fatalf("LoadThumbnail: %v", err)
	}
	b, _, err := LoadThumbnailAndBounds(u)
	if err != nil {
		t.Fatalf("LoadThumbnailAndBounds: %v", err)
	}
	if a.Bounds() != b.Bounds() {
		t.Errorf("LoadThumbnail bounds %v, LoadThumbnailAndBounds thumb %v", a.Bounds(), b.Bounds())
	}
}

// --- ScaleForExport --------------------------------------------------------

// TestScaleForExport_AppliesTheCeilingToTheLongestEdge covers the rule the
// export size limit is: the longest edge lands exactly on the ceiling, the
// short one follows the aspect ratio, and an image already inside the
// ceiling comes back as itself rather than as a re-encoded copy of itself.
func TestScaleForExport_AppliesTheCeilingToTheLongestEdge(t *testing.T) {
	for _, tc := range []struct {
		name         string
		w, h         int
		maxEdge      int
		wantW, wantH int
	}{
		{"landscape", 3000, 2000, 1000, 1000, 666},
		{"portrait", 2000, 3000, 1000, 666, 1000},
		{"square", 2400, 2400, 1600, 1600, 1600},
		{"already inside the ceiling", 800, 600, 1000, 800, 600},
		{"exactly on the ceiling", 1000, 500, 1000, 1000, 500},
		{"no ceiling at all", 3000, 2000, 0, 3000, 2000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			src := markedImage(tc.w, tc.h)

			got := ScaleForExport(src, tc.maxEdge)

			if b := got.Bounds(); b.Dx() != tc.wantW || b.Dy() != tc.wantH {
				t.Errorf("ScaleForExport(%dx%d, %d) = %v, want %dx%d",
					tc.w, tc.h, tc.maxEdge, b, tc.wantW, tc.wantH)
			}
		})
	}
}

// TestScaleForExport_ReturnsTheSameImageWhenNothingChanges is what keeps an
// Original-size export byte-identical to what it always wrote: no ceiling
// means no re-sampling pass at all, not a resample to the same size.
func TestScaleForExport_ReturnsTheSameImageWhenNothingChanges(t *testing.T) {
	src := markedImage(800, 600)

	for _, maxEdge := range []int{0, 1000} {
		if got := ScaleForExport(src, maxEdge); got != image.Image(src) {
			t.Errorf("ScaleForExport(800x600, %d) returned a copy, want the source image itself", maxEdge)
		}
	}
}

// TestScaleForExport_DoesNotWriteTheThumbnailScalersPixels is the guard
// behind "an export is not a thumbnail": the two paths exist at opposite
// ends of a speed/quality tradeoff, and a photo someone is about to mail
// must not come out of the export path resampled the way a 200px grid cell
// is. Both outputs are compared as pixels, over a hard black/white edge
// where any difference in resampling shows - nothing here reaches into how
// either one arrived at them.
func TestScaleForExport_DoesNotWriteTheThumbnailScalersPixels(t *testing.T) {
	const w, h = 400, 400
	src := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			c := color.RGBA{A: 255}
			if x >= w/2 {
				c = color.RGBA{R: 255, G: 255, B: 255, A: 255}
			}
			src.Set(x, y, c)
		}
	}

	export := ScaleForExport(src, 100)
	thumb := scaleToFit(src, 100)

	same := true
	for y := 0; y < 100 && same; y++ {
		for x := range 100 {
			er, eg, eb, _ := export.At(x, y).RGBA()
			tr, tg, tb, _ := thumb.At(x, y).RGBA()
			if er != tr || eg != tg || eb != tb {
				same = false
				break
			}
		}
	}
	if same {
		t.Error("the export path wrote the thumbnail path's pixels, want an export resampled for a photo rather than for a grid cell")
	}
}
