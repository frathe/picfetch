package ui

import (
	"image"
	"image/color"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestCopySelectionTransparency(t *testing.T) {
	v := newTestViewer(t)
	src := markedRegionCopyImage(5, 4)
	dropAndWait(t, v, regionCopyPNGURI(t, "transparent.png", src))

	var copied []byte
	uitest.StubClipboardCopy(t, func(data []byte) error {
		copied = append([]byte(nil), data...)
		return nil
	})

	wantBounds := image.Rect(1, 1, 5, 4)
	selectRegion(t, v, wantBounds)
	v.regionCopy.HandleKey(fyne.KeyReturn)
	waitForClipboard(t, v)

	got := decodeRegionCopyPNG(t, copied)
	if got.Bounds() != image.Rect(0, 0, wantBounds.Dx(), wantBounds.Dy()) {
		t.Fatalf("copied bounds = %v, want %dx%d zero-origin PNG", got.Bounds(), wantBounds.Dx(), wantBounds.Dy())
	}
	for y := range wantBounds.Dy() {
		for x := range wantBounds.Dx() {
			gotPixel := color.NRGBAModel.Convert(got.At(x, y)).(color.NRGBA)
			wantPixel := src.NRGBAAt(wantBounds.Min.X+x, wantBounds.Min.Y+y)
			if gotPixel != wantPixel {
				t.Errorf("copied pixel (%d,%d) = %#v, want alpha-preserving source pixel %#v",
					x, y, gotPixel, wantPixel)
			}
		}
	}
}

func TestCopySelectionRotation(t *testing.T) {
	v := newTestViewer(t)
	src := markedRegionCopyImage(5, 4)
	for y := range src.Bounds().Dy() {
		for x := range src.Bounds().Dx() {
			pixel := src.NRGBAAt(x, y)
			pixel.A = 255
			src.SetNRGBA(x, y, pixel)
		}
	}
	dropAndWait(t, v, regionCopyPNGURI(t, "rotation.png", src))

	v.rotateBy(1)

	var copied []byte
	uitest.StubClipboardCopy(t, func(data []byte) error {
		copied = append([]byte(nil), data...)
		return nil
	})
	wantBounds := image.Rect(0, 1, 3, 4)
	selectRegion(t, v, wantBounds)
	v.regionCopy.HandleKey(fyne.KeyReturn)
	waitForClipboard(t, v)

	got := decodeRegionCopyPNG(t, copied)
	if got.Bounds() != image.Rect(0, 0, wantBounds.Dx(), wantBounds.Dy()) {
		t.Fatalf("copied bounds after unsaved quarter turn = %v, want %dx%d", got.Bounds(), wantBounds.Dx(), wantBounds.Dy())
	}
	for y := range wantBounds.Dy() {
		for x := range wantBounds.Dx() {
			displayX := wantBounds.Min.X + x
			displayY := wantBounds.Min.Y + y
			wantPixel := src.NRGBAAt(displayY, src.Bounds().Dy()-1-displayX)
			if gotPixel := color.NRGBAModel.Convert(got.At(x, y)).(color.NRGBA); gotPixel != wantPixel {
				t.Errorf("rotated copy pixel (%d,%d) = %#v, want %#v", x, y, gotPixel, wantPixel)
			}
		}
	}
}

func TestCopySelectionAnimatedFrame(t *testing.T) {
	v := newTestViewer(t)
	clock := newFrameClock()
	v.frameAfter = clock.After

	path := uitest.WriteTempFile(t, "animated.gif", uitest.EncodeAnimatedGIF(t, 4, 3,
		[]color.Color{color.NRGBA{R: 255, A: 255}, color.NRGBA{B: 255, A: 255}},
		[]int{1000, 1000}))
	dropAndWait(t, v, storage.NewFileURI(path))
	clock.waitParked(t)

	initialFrame := v.animFrame.Load()
	clock.tick(t)
	waitForAnimFrame(t, v, initialFrame+1)
	clock.waitParked(t)

	var copied []byte
	uitest.StubClipboardCopy(t, func(data []byte) error {
		copied = append([]byte(nil), data...)
		return nil
	})
	selectRegion(t, v, image.Rect(0, 0, 4, 3))
	pausedFrame := v.animFrame.Load()

	clock.tick(t)
	v.animationPause.mu.Lock()
	observed := v.animationPause.observed
	v.animationPause.mu.Unlock()
	if observed == nil {
		t.Fatal("animated selection did not install a pause observation signal")
	}
	select {
	case <-observed:
	case <-time.After(testTimeout):
		t.Fatal("timed out waiting for animation to observe Copy Selection pause")
	}
	if got := v.animFrame.Load(); got != pausedFrame {
		t.Fatalf("animation advanced while selecting: animFrame = %d, want paused %d", got, pausedFrame)
	}

	v.regionCopy.HandleKey(fyne.KeyReturn)
	waitForClipboard(t, v)
	got := decodeRegionCopyPNG(t, copied)
	if got.Bounds() != image.Rect(0, 0, 4, 3) {
		t.Fatalf("animated frame copy bounds = %v, want 4x3", got.Bounds())
	}
	if gotPixel := color.NRGBAModel.Convert(got.At(2, 1)).(color.NRGBA); gotPixel != (color.NRGBA{B: 255, A: 255}) {
		t.Fatalf("copied animated pixel = %#v, want the visible blue frame", gotPixel)
	}

	clock.waitParked(t)
	clock.tick(t)
	waitForAnimFrame(t, v, pausedFrame+1)
	clock.waitParked(t)

	selectRegion(t, v, image.Rect(0, 0, 4, 3))
	beforeCancel := v.animFrame.Load()
	v.cancelRegionCopy()
	clock.tick(t)
	waitForAnimFrame(t, v, beforeCancel+1)
	clock.waitParked(t)
}

func TestCopySelectionSVG(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempSVGURI(t, "wide.svg", 200, 100))

	var copied []byte
	uitest.StubClipboardCopy(t, func(data []byte) error {
		copied = append([]byte(nil), data...)
		return nil
	})
	wantBounds := image.Rect(10, 10, 510, 250)
	selectRegion(t, v, wantBounds)

	for range 6 {
		v.zoom.In()
	}
	v.vector.pending.Wait()
	canvasBounds := v.img.Image.Bounds()
	if canvasBounds.Dx() <= 520 || canvasBounds.Dy() <= 260 {
		t.Fatalf("zoomed canvas raster = %v, want denser than the 520x260 logical SVG", canvasBounds)
	}

	v.regionCopy.HandleKey(fyne.KeyReturn)
	waitForClipboard(t, v)
	got := decodeRegionCopyPNG(t, copied)
	if got.Bounds() != image.Rect(0, 0, wantBounds.Dx(), wantBounds.Dy()) {
		t.Fatalf("SVG copy bounds = %v with canvas raster %v, want logical selection %dx%d",
			got.Bounds(), canvasBounds, wantBounds.Dx(), wantBounds.Dy())
	}
	if pixels := int64(got.Bounds().Dx()) * int64(got.Bounds().Dy()); pixels > imaging.MaxVectorRasterPixels() {
		t.Fatalf("SVG copy has %d pixels, over the %d raster cap", pixels, imaging.MaxVectorRasterPixels())
	}
	if gotPixel := color.NRGBAModel.Convert(got.At(250, 120)).(color.NRGBA); gotPixel != (color.NRGBA{R: 0xcc, A: 255}) {
		t.Fatalf("SVG copy center pixel = %#v, want opaque #cc0000", gotPixel)
	}
}

func TestCopySelectionRAWPreview(t *testing.T) {
	v := newTestViewer(t)
	path := uitest.WriteTempFile(t, "preview.cr2", uitest.EncodeRAWPreview(t, uitest.RAWPreview{
		Width:       6,
		Height:      4,
		Color:       color.NRGBA{B: 255, A: 255},
		Orientation: 6,
	}))
	dropAndWait(t, v, storage.NewFileURI(path))

	var copied []byte
	uitest.StubClipboardCopy(t, func(data []byte) error {
		copied = append([]byte(nil), data...)
		return nil
	})
	selectRegion(t, v, image.Rect(0, 0, 4, 6))
	v.regionCopy.HandleKey(fyne.KeyReturn)
	waitForClipboard(t, v)

	got := decodeRegionCopyPNG(t, copied)
	if got.Bounds() != image.Rect(0, 0, 4, 6) {
		t.Fatalf("RAW preview copy bounds = %v, want oriented embedded preview 4x6", got.Bounds())
	}
	pixel := color.NRGBAModel.Convert(got.At(2, 3)).(color.NRGBA)
	if pixel.B < 240 || pixel.R > 15 || pixel.G > 15 || pixel.A != 255 {
		t.Fatalf("RAW preview copy pixel = %#v, want opaque embedded blue preview", pixel)
	}
}

func TestRegionCopySourceMapsLogicalSVGSelectionToCappedRaster(t *testing.T) {
	source := regionCopySource{
		vector:  &imaging.Vector{},
		logical: image.Pt(100, 50),
	}

	got, err := source.cropBounds(image.Rect(25, 10, 76, 41), image.Rect(0, 0, 20, 10))
	if err != nil {
		t.Fatalf("cropBounds() error = %v", err)
	}
	if want := image.Rect(5, 2, 16, 9); got != want {
		t.Fatalf("cropBounds() = %v, want outward-scaled capped raster bounds %v", got, want)
	}

	source.rotation = 1
	got, err = source.cropBounds(image.Rect(10, 25, 41, 76), image.Rect(0, 0, 10, 20))
	if err != nil {
		t.Fatalf("rotated cropBounds() error = %v", err)
	}
	if want := image.Rect(2, 5, 9, 16); got != want {
		t.Fatalf("rotated cropBounds() = %v, want outward-scaled capped raster bounds %v", got, want)
	}
}
