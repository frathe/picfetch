package copyselection_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/copyselection"
)

func TestSourceEncode_CropsLiteralRasterPixelsAndPreservesAlpha(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 6, 4))
	src.SetNRGBA(1, 1, color.NRGBA{R: 255, A: 255})
	src.SetNRGBA(2, 1, color.NRGBA{G: 200, A: 127})
	src.SetNRGBA(3, 1, color.NRGBA{B: 180, A: 64})
	src.SetNRGBA(1, 2, color.NRGBA{R: 220, G: 180, A: 255})
	src.SetNRGBA(2, 2, color.NRGBA{R: 130, B: 210, A: 200})
	src.SetNRGBA(3, 2, color.NRGBA{G: 140, B: 230, A: 96})

	data, err := copyselection.RasterSource(src).Encode(image.Rect(1, 1, 4, 3))
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	got := decodeSourcePNG(t, data)
	if got.Bounds() != image.Rect(0, 0, 3, 2) {
		t.Fatalf("decoded bounds = %v, want (0,0)-(3,2)", got.Bounds())
	}

	want := [][]color.NRGBA{
		{{R: 255, A: 255}, {G: 200, A: 127}, {B: 180, A: 64}},
		{{R: 220, G: 180, A: 255}, {R: 130, B: 210, A: 200}, {G: 140, B: 230, A: 96}},
	}
	for y := range 2 {
		for x := range 3 {
			if pixel := color.NRGBAModel.Convert(got.At(x, y)).(color.NRGBA); pixel != want[y][x] {
				t.Errorf("pixel (%d,%d) = %#v, want %#v", x, y, pixel, want[y][x])
			}
		}
	}
}

func TestSourceEncode_MapsLogicalSVGSelectionToCappedRaster(t *testing.T) {
	capped := image.NewNRGBA(image.Rect(0, 0, 20, 10))
	capped.SetNRGBA(5, 2, color.NRGBA{R: 255, A: 255})
	capped.SetNRGBA(15, 8, color.NRGBA{B: 255, A: 127})

	source := copyselection.VectorSource(&imaging.Vector{}, image.Pt(100, 50), 0,
		func(_ *imaging.Vector, w, h int) (image.Image, error) {
			if w != 100 || h != 50 {
				t.Fatalf("rasterize size = %dx%d, want logical 100x50", w, h)
			}
			return capped, nil
		})

	data, err := source.Encode(image.Rect(25, 10, 76, 41))
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	got := decodeSourcePNG(t, data)
	if got.Bounds() != image.Rect(0, 0, 11, 7) {
		t.Fatalf("decoded bounds = %v, want outward-scaled capped crop 11x7", got.Bounds())
	}
	if pixel := color.NRGBAModel.Convert(got.At(0, 0)).(color.NRGBA); pixel != (color.NRGBA{R: 255, A: 255}) {
		t.Errorf("crop origin pixel = %#v, want marked capped pixel (5,2)", pixel)
	}
	if pixel := color.NRGBAModel.Convert(got.At(10, 6)).(color.NRGBA); pixel != (color.NRGBA{B: 255, A: 127}) {
		t.Errorf("crop far pixel = %#v, want marked capped pixel (15,8)", pixel)
	}
}

func TestSourceEncode_MapsRotatedSVGSelectionToCappedRaster(t *testing.T) {
	capped := image.NewNRGBA(image.Rect(0, 0, 20, 10))
	source := copyselection.VectorSource(&imaging.Vector{}, image.Pt(100, 50), 1,
		func(_ *imaging.Vector, w, h int) (image.Image, error) {
			if w != 100 || h != 50 {
				t.Fatalf("rasterize size = %dx%d, want unrotated logical 100x50", w, h)
			}
			return capped, nil
		})

	data, err := source.Encode(image.Rect(10, 25, 41, 76))
	if err != nil {
		t.Fatalf("Encode() error = %v", err)
	}

	got := decodeSourcePNG(t, data)
	if got.Bounds() != image.Rect(0, 0, 7, 11) {
		t.Fatalf("decoded bounds = %v, want rotated outward-scaled capped crop 7x11", got.Bounds())
	}
}

func TestSourceEncode_RejectsInvalidBounds(t *testing.T) {
	src := copyselection.RasterSource(image.NewNRGBA(image.Rect(0, 0, 4, 3)))
	if _, err := src.Encode(image.Rect(0, 0, 5, 3)); err == nil {
		t.Fatal("Encode() error = nil, want bounds outside the raster")
	}

	vector := copyselection.VectorSource(&imaging.Vector{}, image.Pt(100, 50), 0,
		func(*imaging.Vector, int, int) (image.Image, error) {
			return image.NewNRGBA(image.Rect(0, 0, 20, 10)), nil
		})
	if _, err := vector.Encode(image.Rect(0, 0, 101, 50)); err == nil {
		t.Fatal("Encode() error = nil, want bounds outside the SVG logical size")
	}
}

func TestSourceBounds(t *testing.T) {
	raster := image.NewNRGBA(image.Rect(0, 0, 6, 4))
	if got := copyselection.RasterSource(raster).Bounds(); got != image.Rect(0, 0, 6, 4) {
		t.Fatalf("RasterSource.Bounds() = %v, want (0,0)-(6,4)", got)
	}

	vector := copyselection.VectorSource(&imaging.Vector{}, image.Pt(100, 50), 0, nil)
	if got := vector.Bounds(); got != image.Rect(0, 0, 100, 50) {
		t.Fatalf("VectorSource.Bounds() = %v, want logical (0,0)-(100,50)", got)
	}

	rotated := copyselection.VectorSource(&imaging.Vector{}, image.Pt(100, 50), 1, nil)
	if got := rotated.Bounds(); got != image.Rect(0, 0, 50, 100) {
		t.Fatalf("rotated VectorSource.Bounds() = %v, want swapped logical (0,0)-(50,100)", got)
	}
}

func decodeSourcePNG(t *testing.T, data []byte) image.Image {
	t.Helper()
	got, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("png.Decode() error = %v", err)
	}
	return got
}
