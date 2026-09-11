package imaging

import (
	"image"
	"image/color"
	"image/draw"
	"testing"
)

func TestApplyOrientationPreservesPixelFormats(t *testing.T) {
	// A subimage exercises both a nonzero origin and a stride wider than its
	// pixels. Fractional alpha and 16-bit colors detect conversion/rounding loss.
	bounds := image.Rect(4, 7, 9, 11)
	region := image.Rect(5, 8, 8, 10)
	ycbcr := image.NewYCbCr(bounds, image.YCbCrSubsampleRatio420)
	for i := range ycbcr.Y {
		ycbcr.Y[i] = uint8(17 * i)
	}
	for i := range ycbcr.Cb {
		ycbcr.Cb[i], ycbcr.Cr[i] = uint8(31*i), uint8(255-23*i)
	}
	colors := []color.Color{
		color.NRGBA64{R: 0xf123, G: 0x8234, B: 0x3567, A: 0x8001},
		color.NRGBA64{R: 0x1234, G: 0xdef0, B: 0x4567, A: 0xffff},
		color.NRGBA64{R: 0xffff, G: 0xffff, A: 0},
		color.NRGBA64{R: 0x2345, G: 0x9876, B: 0xabcd, A: 0x0101},
		color.NRGBA64{R: 0x3456, G: 0x789a, B: 0xfedc, A: 0xff00},
		color.NRGBA64{R: 0xabcd, G: 0x7654, B: 0x3210, A: 0x7fff},
	}
	sources := map[string]image.Image{
		"rgba":     image.NewRGBA(bounds).SubImage(region),
		"nrgba":    image.NewNRGBA(bounds).SubImage(region),
		"rgba64":   image.NewRGBA64(bounds).SubImage(region),
		"nrgba64":  image.NewNRGBA64(bounds).SubImage(region),
		"gray16":   image.NewGray16(bounds).SubImage(region),
		"cmyk":     image.NewCMYK(bounds).SubImage(region),
		"paletted": image.NewPaletted(bounds, color.Palette(colors)).SubImage(region),
		"ycbcr":    ycbcr.SubImage(region),
	}
	orders := [8][6]int{
		{0, 1, 2, 3, 4, 5}, {2, 1, 0, 5, 4, 3},
		{5, 4, 3, 2, 1, 0}, {3, 4, 5, 0, 1, 2},
		{0, 3, 1, 4, 2, 5}, {3, 0, 4, 1, 5, 2},
		{5, 2, 4, 1, 3, 0}, {2, 5, 1, 4, 0, 3},
	}
	for name, source := range sources {
		t.Run(name, func(t *testing.T) {
			if writable, ok := source.(draw.Image); ok {
				for i, c := range colors {
					writable.Set(region.Min.X+i%3, region.Min.Y+i/3, c)
				}
			}
			var original [6]color.RGBA
			for i := range original {
				original[i] = color.RGBAModel.Convert(source.At(region.Min.X+i%3, region.Min.Y+i/3)).(color.RGBA)
			}
			// Hiding optional accessors retains the image.Image compatibility path.
			for _, input := range []image.Image{source, struct{ image.Image }{source}} {
				for index, order := range orders {
					got := ApplyOrientation(input, index+1)
					wantBounds := image.Rect(0, 0, 3, 2)
					if index == 0 {
						wantBounds = region
					} else if index >= 4 {
						wantBounds = image.Rect(0, 0, 2, 3)
					}
					if got.Bounds() != wantBounds {
						t.Fatalf("orientation %d: bounds %v, want %v", index+1, got.Bounds(), wantBounds)
					}
					for i, from := range order {
						pixel := got.At(wantBounds.Min.X+i%wantBounds.Dx(), wantBounds.Min.Y+i/wantBounds.Dx())
						if color.RGBAModel.Convert(pixel) != original[from] {
							t.Fatalf("orientation %d: pixel %d = %v, want %v", index+1, i, pixel, original[from])
						}
						if pixel := source.At(region.Min.X+i%3, region.Min.Y+i/3); color.RGBAModel.Convert(pixel) != original[i] {
							t.Fatalf("orientation %d changed source pixel %d", index+1, i)
						}
					}
				}
			}
		})
	}
}

// markedImage builds a w x h RGBA image where each pixel's color encodes its
// own coordinates, so transforms can be checked by looking up where a given
// source pixel ended up.
func markedImage(w, h int) *image.RGBA {
	return markedImageBounds(image.Rect(0, 0, w, h))
}

func markedImageBounds(bounds image.Rectangle) *image.RGBA {
	img := image.NewRGBA(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			img.SetRGBA(x, y, color.RGBA{
				R: uint8(x - bounds.Min.X),
				G: uint8(y - bounds.Min.Y),
				A: 255,
			})
		}
	}

	return img
}

func at(t *testing.T, img image.Image, x, y int) (int, int) {
	t.Helper()
	c := img.At(x, y).(color.RGBA)
	return int(c.R), int(c.G)
}

func TestOrientationTransformsNonZeroBounds(t *testing.T) {
	sourceBounds := image.Rect(5, 7, 8, 9)
	src := markedImageBounds(sourceBounds)
	w, h := sourceBounds.Dx(), sourceBounds.Dy()

	tests := []struct {
		name       string
		transform  func(image.Image) image.Image
		wantBounds image.Rectangle
		dest       func(x, y int) (int, int)
	}{
		{"flip horizontal", flipH, image.Rect(0, 0, w, h), func(x, y int) (int, int) { return w - 1 - x, y }},
		{"flip vertical", flipV, image.Rect(0, 0, w, h), func(x, y int) (int, int) { return x, h - 1 - y }},
		{"rotate 180", rotate180, image.Rect(0, 0, w, h), func(x, y int) (int, int) { return w - 1 - x, h - 1 - y }},
		{"rotate 90 clockwise", rotate90CW, image.Rect(0, 0, h, w), func(x, y int) (int, int) { return h - 1 - y, x }},
		{"rotate 270 clockwise", rotate270CW, image.Rect(0, 0, h, w), func(x, y int) (int, int) { return y, w - 1 - x }},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got := test.transform(src)
			if got.Bounds() != test.wantBounds {
				t.Fatalf("bounds = %v, want %v", got.Bounds(), test.wantBounds)
			}

			for y := range h {
				for x := range w {
					destX, destY := test.dest(x, y)
					gotX, gotY := at(t, got, destX, destY)
					if gotX != x || gotY != y {
						t.Errorf("source (%d,%d): pixel at (%d,%d) = (%d,%d), want (%d,%d)",
							x, y, destX, destY, gotX, gotY, x, y)
					}
				}
			}
		})
	}
}

func TestApplyOrientation(t *testing.T) {
	const w, h = 3, 2 // asymmetric so transposed cases are unambiguous

	cases := []struct {
		name        string
		orientation int
		wantW       int
		wantH       int
		// check maps a source (x,y) to the (x,y) it should land at in the
		// corrected image.
		check func(x, y int) (int, int)
	}{
		{"1: identity", 1, w, h, func(x, y int) (int, int) { return x, y }},
		{"2: flip horizontal", 2, w, h, func(x, y int) (int, int) { return w - 1 - x, y }},
		{"3: rotate 180", 3, w, h, func(x, y int) (int, int) { return w - 1 - x, h - 1 - y }},
		{"4: flip vertical", 4, w, h, func(x, y int) (int, int) { return x, h - 1 - y }},
		{"6: rotate 90 CW", 6, h, w, func(x, y int) (int, int) { return h - 1 - y, x }},
		{"8: rotate 270 CW", 8, h, w, func(x, y int) (int, int) { return y, w - 1 - x }},
		{"unknown value: treated as identity", 99, w, h, func(x, y int) (int, int) { return x, y }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := markedImage(w, h)
			out := ApplyOrientation(src, c.orientation)

			b := out.Bounds()
			if b.Dx() != c.wantW || b.Dy() != c.wantH {
				t.Fatalf("bounds = %dx%d, want %dx%d", b.Dx(), b.Dy(), c.wantW, c.wantH)
			}

			for y := range h {
				for x := range w {
					wantX, wantY := c.check(x, y)
					gotX, gotY := at(t, out, wantX, wantY)
					if gotX != x || gotY != y {
						t.Errorf("source (%d,%d): pixel at (%d,%d) = (%d,%d), want (%d,%d)",
							x, y, wantX, wantY, gotX, gotY, x, y)
					}
				}
			}
		})
	}

	// 5 (transpose) and 7 (transverse) swap dimensions like 6 and 8, but
	// mirror across a diagonal rather than rotating; check them by
	// composition against the primitives they're defined in terms of.
	t.Run("5: transpose matches flipH+rotate270CW", func(t *testing.T) {
		src := markedImage(w, h)
		want := rotate270CW(flipH(src))
		got := ApplyOrientation(src, 5)

		if got.Bounds() != want.Bounds() {
			t.Fatalf("bounds = %v, want %v", got.Bounds(), want.Bounds())
		}

		for y := range h {
			for x := range w {
				gx, gy := at(t, got, x, y)
				wx, wy := at(t, want, x, y)
				if gx != wx || gy != wy {
					t.Errorf("(%d,%d) = (%d,%d), want (%d,%d)", x, y, gx, gy, wx, wy)
				}
			}
		}
	})

	t.Run("7: transverse matches flipH+rotate90CW", func(t *testing.T) {
		src := markedImage(w, h)
		want := rotate90CW(flipH(src))
		got := ApplyOrientation(src, 7)

		if got.Bounds() != want.Bounds() {
			t.Fatalf("bounds = %v, want %v", got.Bounds(), want.Bounds())
		}

		for y := range h {
			for x := range w {
				gx, gy := at(t, got, x, y)
				wx, wy := at(t, want, x, y)
				if gx != wx || gy != wy {
					t.Errorf("(%d,%d) = (%d,%d), want (%d,%d)", x, y, gx, gy, wx, wy)
				}
			}
		}
	})
}

func TestRotateSteps(t *testing.T) {
	const w, h = 3, 2 // asymmetric so transposed cases are unambiguous

	cases := []struct {
		name  string
		steps int
		wantW int
		wantH int
		check func(x, y int) (int, int)
	}{
		{"0: identity", 0, w, h, func(x, y int) (int, int) { return x, y }},
		{"1: rotate 90 CW", 1, h, w, func(x, y int) (int, int) { return h - 1 - y, x }},
		{"2: rotate 180", 2, w, h, func(x, y int) (int, int) { return w - 1 - x, h - 1 - y }},
		{"3: rotate 270 CW", 3, h, w, func(x, y int) (int, int) { return y, w - 1 - x }},
		{"4: wraps back to identity", 4, w, h, func(x, y int) (int, int) { return x, y }},
		{"-1: one turn counter-clockwise matches 3", -1, h, w, func(x, y int) (int, int) { return y, w - 1 - x }},
		{"-4: wraps back to identity", -4, w, h, func(x, y int) (int, int) { return x, y }},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			src := markedImage(w, h)
			out := RotateSteps(src, c.steps)

			b := out.Bounds()
			if b.Dx() != c.wantW || b.Dy() != c.wantH {
				t.Fatalf("bounds = %dx%d, want %dx%d", b.Dx(), b.Dy(), c.wantW, c.wantH)
			}

			for y := range h {
				for x := range w {
					wantX, wantY := c.check(x, y)
					gotX, gotY := at(t, out, wantX, wantY)
					if gotX != x || gotY != y {
						t.Errorf("source (%d,%d): pixel at (%d,%d) = (%d,%d), want (%d,%d)",
							x, y, wantX, wantY, gotX, gotY, x, y)
					}
				}
			}
		})
	}

	t.Run("identity returns the same image, not a copy", func(t *testing.T) {
		src := markedImage(w, h)
		out := RotateSteps(src, 0)
		if out != image.Image(src) {
			t.Error("RotateSteps(img, 0) should return img unchanged, not a new buffer")
		}
	})

	t.Run("composing two 90 CW steps matches one 180 step", func(t *testing.T) {
		src := markedImage(w, h)
		want := RotateSteps(src, 2)
		got := RotateSteps(RotateSteps(src, 1), 1)

		if got.Bounds() != want.Bounds() {
			t.Fatalf("bounds = %v, want %v", got.Bounds(), want.Bounds())
		}

		for y := 0; y < want.Bounds().Dy(); y++ {
			for x := 0; x < want.Bounds().Dx(); x++ {
				gx, gy := at(t, got, x, y)
				wx, wy := at(t, want, x, y)
				if gx != wx || gy != wy {
					t.Errorf("(%d,%d) = (%d,%d), want (%d,%d)", x, y, gx, gy, wx, wy)
				}
			}
		}
	})
}
