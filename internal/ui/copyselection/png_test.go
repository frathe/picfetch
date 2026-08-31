package copyselection_test

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/frathe/picfetch/internal/ui/copyselection"
)

func TestPNG_CropsLiteralPixelsAndPreservesAlpha(t *testing.T) {
	src := image.NewNRGBA(image.Rect(10, 20, 14, 23))
	src.SetNRGBA(11, 20, color.NRGBA{R: 255, A: 255})
	src.SetNRGBA(12, 20, color.NRGBA{G: 200, A: 127})
	src.SetNRGBA(13, 20, color.NRGBA{B: 180, A: 64})
	src.SetNRGBA(11, 21, color.NRGBA{R: 220, G: 180, A: 255})
	src.SetNRGBA(12, 21, color.NRGBA{R: 130, B: 210, A: 200})
	src.SetNRGBA(13, 21, color.NRGBA{G: 140, B: 230, A: 96})

	data, err := copyselection.PNG(src, image.Rect(11, 20, 14, 22))
	if err != nil {
		t.Fatalf("PNG() error = %v", err)
	}

	got, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("png.Decode() error = %v", err)
	}
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

func TestPNG_RejectsInvalidBounds(t *testing.T) {
	src := image.NewNRGBA(image.Rect(10, 20, 14, 23))

	for _, test := range []struct {
		name   string
		src    image.Image
		bounds image.Rectangle
	}{
		{name: "empty", src: src, bounds: image.Rect(11, 21, 11, 22)},
		{name: "outside", src: src, bounds: image.Rect(0, 0, 2, 2)},
		{name: "partially outside", src: src, bounds: image.Rect(9, 20, 12, 22)},
		{name: "nil source", bounds: image.Rect(0, 0, 1, 1)},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := copyselection.PNG(test.src, test.bounds); err == nil {
				t.Fatal("PNG() error = nil, want a recoverable validation error")
			}
		})
	}
}
