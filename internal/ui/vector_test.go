package ui

import (
	"image/color"
	"strings"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestVectorFloorMatchesStartWindowSize(t *testing.T) {
	gotW, wantW := startW, float64(imaging.MinVectorWidth)
	if gotW != wantW {
		t.Fatalf("startW = %v, imaging.MinVectorWidth = %v", startW, imaging.MinVectorWidth)
	}
	gotH, wantH := startH, float64(imaging.MinVectorHeight)
	if gotH != wantH {
		t.Fatalf("startH = %v, imaging.MinVectorHeight = %v", startH, imaging.MinVectorHeight)
	}
}

func TestSVGDisplaysAtLogicalSize(t *testing.T) {
	v, _, _ := newTestUI(t)
	dropAndWait(t, v, uitest.TempSVGURI(t, "icon.svg", 24, 24))

	if !v.display.Snapshot().Vector {
		t.Fatal("a loaded SVG must leave a Vector on the viewer")
	}
	if got := vectorLogicalSize(v); got.Width != 340 || got.Height != 340 {
		t.Fatalf("logical size = %v, want 340x340", got)
	}
	if b := v.img.Image.Bounds(); b.Dx() != 340 || b.Dy() != 340 {
		t.Fatalf("first raster = %dx%d, want 340x340", b.Dx(), b.Dy())
	}
}

func TestSVGReRendersAtHigherDensityOnZoom(t *testing.T) {
	v, _, _ := newTestUI(t)
	dropAndWait(t, v, uitest.TempSVGURI(t, "icon.svg", 24, 24))

	before := v.img.Image.Bounds().Size()

	for range 6 { // 1.25^6 ~= 3.8x
		v.zoom.In()
	}
	v.display.Settle()

	if v.img.Image.Bounds().Size().X <= before.X {
		t.Fatalf("raster stayed at %v after zooming in from %v", v.img.Image.Bounds().Size(), before)
	}

	if got := vectorLogicalSize(v); got.Width != 340 || got.Height != 340 {
		t.Fatalf("logical size drifted to %v", got)
	}
}

func TestSVGReRenderNeverMutatesTheCachedEntry(t *testing.T) {
	v, _, _ := newTestUI(t)
	uri := uitest.TempSVGURI(t, "icon.svg", 24, 24)
	dropAndWait(t, v, uri)

	cached, ok := v.imgCache.Get(uri.String())
	if !ok {
		t.Fatal("the loaded SVG should be in the image cache")
	}
	cachedBefore := cached.Frames[0].Bounds()

	for range 6 {
		v.zoom.In()
	}
	v.display.Settle()

	if got := cached.Frames[0].Bounds(); got != cachedBefore {
		t.Fatalf("re-render mutated the cached frame: %v -> %v", cachedBefore, got)
	}
	if v.img.Image == cached.Frames[0] {
		t.Fatal("the display frame must not share the cached entry's backing array")
	}
}

func TestZoomKeysDriveVectorRerenders(t *testing.T) {
	v, _, _ := newTestUI(t)
	dropAndWait(t, v, uitest.TempSVGURI(t, "icon.svg", 24, 24))
	before := v.img.Image.Bounds().Size()
	for range 12 {
		v.zoom.In()
	}
	v.display.Settle()
	if v.img.Image.Bounds().Dx() <= before.X {
		t.Fatal("zoom keys did not sharpen the vector")
	}
}

func TestRasterFormatKeepsNoVectorState(t *testing.T) {
	v, _, _ := newTestUI(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 30, color.White))

	if v.display.Snapshot().Vector {
		t.Fatal("a JPEG must leave no Vector behind")
	}
	if got := vectorLogicalSize(v); got != (fyne.Size{}) {
		t.Fatalf("vector.logical = %v, want zero", got)
	}
}

func TestSVGThenRasterClearsVectorState(t *testing.T) {
	v, _, _ := newTestUI(t)
	dropAndWait(t, v, uitest.TempSVGURI(t, "icon.svg", 24, 24))
	dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 30, color.White))

	if v.display.Snapshot().Vector {
		t.Fatal("navigating from an SVG to a JPEG must clear the Vector")
	}
}

func TestSVGRotationSwapsLogicalAxes(t *testing.T) {
	v, _, _ := newTestUI(t)
	dropAndWait(t, v, uitest.TempSVGURI(t, "wide.svg", 200, 100)) // logical 520x260

	v.rotateBy(1)
	v.display.Settle()

	if b := v.img.Image.Bounds(); b.Dy() <= b.Dx() {
		t.Fatalf("after a quarter turn the frame is %dx%d, want taller than wide", b.Dx(), b.Dy())
	}
}

func TestSVGRotationSwapsTheLogicalSizeZoomMeasuresAgainst(t *testing.T) {
	v, _, _ := newTestUI(t)
	dropAndWait(t, v, uitest.TempSVGURI(t, "wide.svg", 200, 100)) // logical 520x260

	if got := v.zoom.LogicalSize(); got.Width != 520 || got.Height != 260 {
		t.Fatalf("before rotation zoom measures against %v, want 520x260", got)
	}

	v.rotateBy(1)
	v.display.Settle()

	if got := v.zoom.LogicalSize(); got.Width != 260 || got.Height != 520 {
		t.Fatalf("after a quarter turn zoom measures against %v, want 260x520", got)
	}

	if got := vectorLogicalSize(v); got.Width != 520 || got.Height != 260 {
		t.Fatalf("vector.logical moved to %v, must stay the unrotated 520x260", got)
	}

	v.rotateBy(1) // 180 degrees: back to the original axes

	if got := v.zoom.LogicalSize(); got.Width != 520 || got.Height != 260 {
		t.Fatalf("at 180 degrees zoom measures against %v, want 520x260", got)
	}
}

func TestRotatedNonSquareSVGKeepsItsAspectRatio(t *testing.T) {
	v, _, _ := newTestUI(t)
	dropAndWait(t, v, uitest.TempSVGURI(t, "wide.svg", 200, 100)) // logical 520x260, 2:1

	v.rotateBy(1)
	for range 6 {
		v.zoom.In()
	}
	v.display.Settle()

	b := v.img.Image.Bounds()
	if got, want := float64(b.Dx())/float64(b.Dy()), 0.5; got < want*0.98 || got > want*1.02 {
		t.Fatalf("unrotated raster is %dx%d (aspect %.3f), want aspect ~2.0 - a swapped target stretches the drawing",
			b.Dx(), b.Dy(), got)
	}
}

func TestSVGReRendersAfterRotation(t *testing.T) {
	v, _, _ := newTestUI(t)
	dropAndWait(t, v, uitest.TempSVGURI(t, "icon.svg", 24, 24))

	v.rotateBy(1)
	for range 6 {
		v.zoom.In()
	}
	v.display.Settle()

}

func TestInfoOverlayReportsLogicalSizeNotLiveRaster(t *testing.T) {
	v, _, _ := newTestUI(t)
	dropAndWait(t, v, uitest.TempSVGURI(t, "wide.svg", 200, 100)) // logical 520x260

	if w, h := v.displayedDimensions(); w != 520 || h != 260 {
		t.Fatalf("dimensions = %dx%d, want 520x260", w, h)
	}

	for range 6 {
		v.zoom.In()
	}
	v.display.Settle()

	if b := v.img.Image.Bounds(); b.Dx() <= 520 {
		t.Fatalf("test setup: raster did not grow after zooming in, got %dx%d", b.Dx(), b.Dy())
	}
	if w, h := v.displayedDimensions(); w != 520 || h != 260 {
		t.Fatalf("dimensions after zooming in = %dx%d, want unchanged 520x260 (raster is now %v)", w, h, v.img.Image.Bounds())
	}

	v.rotateBy(1)
	v.display.Settle()

	w, h := v.displayedDimensions()
	if w != 260 || h != 520 {
		t.Fatalf("dimensions after a quarter turn = %dx%d, want swapped 260x520", w, h)
	}

	v.toggleInfoOverlay()
	if got := strings.Split(v.info.Text().Text, "\n")[1]; got != "260 x 520" {
		t.Fatalf("infoText dimensions = %q, want %q", got, "260 x 520")
	}
}

func TestCloseFilesClearsVector(t *testing.T) {
	v, _, _ := newTestUI(t)
	dropAndWait(t, v, uitest.TempSVGURI(t, "icon.svg", 24, 24))

	v.closeFiles()

	if v.display.Snapshot().Vector {
		t.Fatal("closing files must clear the vector state")
	}
}

func TestRotatingAZoomedSVGSizesTheWindowFromItsLogicalSize(t *testing.T) {
	v, win, _ := newTestUI(t)
	dropAndWait(t, v, uitest.TempSVGURI(t, "wide.svg", 200, 100)) // logical 520x260

	v.rotateBy(1)
	v.display.Settle()

	unzoomed := win.Canvas().Size()

	v.resetRotation()

	for range 6 { // ~3.8x, well inside zoom's maxScale clamp
		v.zoom.In()
	}
	v.display.Settle()

	if v.img.Image.Bounds().Size().X <= int(vectorLogicalSize(v).Width) {
		t.Fatalf("raster is %v, expected zooming to have made it denser than the logical %v - "+
			"the rest of this test proves nothing otherwise", v.img.Image.Bounds().Size(), vectorLogicalSize(v))
	}

	v.rotateBy(1)
	v.display.Settle()

	if got := win.Canvas().Size(); got != unzoomed {
		t.Fatalf("rotating a zoomed SVG sized the window to %v, want %v - the same window "+
			"an unzoomed rotation produces", got, unzoomed)
	}
}

func TestSVGThumbnailsInTheGridOverview(t *testing.T) {
	v := newTestViewer(t)

	svg := uitest.TempSVGURI(t, "icon.svg", 24, 24) // logical 340x340
	dropAndWait(t, v, svg, uitest.TempJPEGURI(t, "photo.jpg", 8, 8, color.White))

	warmThumbs(t, v)

	v.grid.Toggle()
	if !v.grid.Visible() {
		t.Fatal("the grid should be open")
	}

	thumb, err := imaging.LoadThumbnail(svg)
	if err != nil {
		t.Fatalf("LoadThumbnail on an SVG: %v", err)
	}

	b := thumb.Bounds()
	if b.Dx() <= 0 || b.Dy() <= 0 {
		t.Fatalf("SVG thumbnail is %dx%d, want real pixels", b.Dx(), b.Dy())
	}
	if b.Dx() >= imaging.MinVectorWidth {
		t.Fatalf("SVG thumbnail is %dx%d, want it downsampled well below the %d logical size",
			b.Dx(), b.Dy(), imaging.MinVectorWidth)
	}
	if b.Dx() != b.Dy() {
		t.Fatalf("SVG thumbnail is %dx%d, want the source's square aspect preserved", b.Dx(), b.Dy())
	}
}

func vectorLogicalSize(v *viewer) fyne.Size {
	capture, ok := v.display.Capture()
	if !ok || capture.Vector == nil {
		return fyne.Size{}
	}
	return fyne.NewSize(float32(capture.LogicalSize.X), float32(capture.LogicalSize.Y))
}
