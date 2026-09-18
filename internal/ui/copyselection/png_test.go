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

// Literal pixel expectations deliberately remain beside this encoder's test.
//
//goland:noinspection DuplicatedCode
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

func TestPNG_RejectsSelectionThatExceedsWorkingMemoryBudget(t *testing.T) {
	setPNGByteLimit(t, 4096)
	src := image.NewUniform(color.White)

	if _, err := copyselection.PNG(src, image.Rect(0, 0, 32, 33)); err == nil {
		t.Fatal("PNG() error = nil, want a recoverable size-limit error")
	}
}

func TestPNG_UsesConfiguredSizeLimit(t *testing.T) {
	setPNGByteLimit(t, 4095)
	src := image.NewUniform(color.White)
	bounds := image.Rect(0, 0, 32, 32)
	if data, err := copyselection.PNG(src, bounds); err == nil || data != nil {
		t.Fatalf("PNG below pixel budget = (%d bytes, %v), want rejection", len(data), err)
	}
	imaging.SetMaxEncodedBytes(4096)
	if _, err := copyselection.PNG(src, bounds); err != nil {
		t.Fatalf("PNG at pixel budget: %v", err)
	}
}

func TestPNG_RejectsScanlinesOverBudget(t *testing.T) {
	setPNGByteLimit(t, 4096)
	src := image.NewUniform(color.NRGBA{R: 200, A: 128})
	if data, err := copyselection.PNG(src, image.Rect(0, 0, 200, 1)); err == nil || data != nil {
		t.Fatalf("PNG over scanline budget = (%d bytes, %v), want rejection", len(data), err)
	}
}

func TestPNG_BoundsEncodedOutput(t *testing.T) {
	setPNGByteLimit(t, 1024)
	src := image.NewUniform(color.White)
	bounds := image.Rect(0, 0, 1, 1)
	want, err := copyselection.PNG(src, bounds)
	if err != nil {
		t.Fatal(err)
	}
	imaging.SetMaxEncodedBytes(int64(len(want) - 1))
	if data, err := copyselection.PNG(src, bounds); err == nil || data != nil {
		t.Fatalf("PNG over output limit = (%d bytes, %v), want rejection", len(data), err)
	}
	imaging.SetMaxEncodedBytes(int64(len(want)))
	if got, err := copyselection.PNG(src, bounds); err != nil || !bytes.Equal(got, want) {
		t.Fatalf("PNG at output limit = (%d bytes, %v), want %d bytes", len(got), err, len(want))
	}
}

func TestPNG_PreservesEightBitOutput(t *testing.T) {
	for _, c := range []color.Color{
		color.NRGBA64{R: 0xabcd, G: 0x1234, B: 0x5678, A: 0xffff},
		color.Gray16{Y: 0xabcd},
		color.YCbCr{Y: 100, Cb: 50, Cr: 150},
	} {
		src := image.NewUniform(c)
		data, err := copyselection.PNG(src, image.Rect(0, 0, 2, 2))
		if err != nil {
			t.Fatal(err)
		}
		if data[24] != 8 { // PNG IHDR bit depth.
			t.Errorf("%T: PNG depth = %d, want 8", c, data[24])
		}
		got, err := png.Decode(bytes.NewReader(data))
		if err != nil {
			t.Fatal(err)
		}
		if gotPixel, want := color.NRGBAModel.Convert(got.At(0, 0)), color.NRGBAModel.Convert(c); gotPixel != want {
			t.Errorf("%T: pixel = %v, want %v", c, gotPixel, want)
		}
	}
}

func setPNGByteLimit(t *testing.T, limit int64) {
	t.Helper()
	previous := imaging.MaxEncodedBytes()
	t.Cleanup(func() { imaging.SetMaxEncodedBytes(previous) })
	imaging.SetMaxEncodedBytes(limit)
}

func TestPNG_CapturesSizeLimitBeforeReadingPixels(t *testing.T) {
	setPNGByteLimit(t, 1024)
	src := pngPolicyChangeImage{Image: image.NewUniform(color.White)}
	if _, err := copyselection.PNG(src, image.Rect(0, 0, 2, 2)); err != nil {
		t.Fatalf("PNG changed its captured limit during encoding: %v", err)
	}
}

type pngPolicyChangeImage struct{ image.Image }

func (p pngPolicyChangeImage) At(x, y int) color.Color {
	imaging.SetMaxEncodedBytes(1)
	return p.Image.At(x, y)
}
