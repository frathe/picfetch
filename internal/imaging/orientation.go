package imaging

import (
	"image"
	"image/color"
)

// orientationPixels avoids boxing a color.Color for every source pixel. The
// standard image types expose the same premultiplied channels through RGBA64At;
// SetRGBA64 retains their upper eight bits, exactly as RGBA.Set does.
func orientationPixels(src image.Image) image.RGBA64Image {
	if pixels, ok := src.(image.RGBA64Image); ok {
		return pixels
	}
	return rgba64Adapter{src}
}

type rgba64Adapter struct{ image.Image }

func (a rgba64Adapter) RGBA64At(x, y int) color.RGBA64 {
	r, g, b, alpha := a.At(x, y).RGBA()
	return color.RGBA64{R: uint16(r), G: uint16(g), B: uint16(b), A: uint16(alpha)}
}

// ApplyOrientation returns img corrected for the given Exif orientation tag
// value (1-8, per the Exif spec). Orientation 1, and any value outside that
// range, means no correction is needed.
func ApplyOrientation(img image.Image, orientation int) image.Image {
	switch orientation {
	case 2:
		return flipH(img)
	case 3:
		return rotate180(img)
	case 4:
		return flipV(img)
	case 5:
		return rotate270CW(flipH(img))
	case 6:
		return rotate90CW(img)
	case 7:
		return rotate90CW(flipH(img))
	case 8:
		return rotate270CW(img)
	default:
		return img
	}
}

// RotateSteps rotates img clockwise by steps quarter turns, wrapping any
// value outside 0-3 into that range first (so -1 and 3 both mean "one turn
// counter-clockwise"). It's the primitive behind the app's view-only R/
// Shift+R rotation: composed on top of an image ApplyOrientation has
// already corrected, it never touches the EXIF tag or file data, and being
// a plain 90-degree-multiple permutation (no resampling), applying it
// repeatedly never degrades the pixels the way a resampled rotation would.
func RotateSteps(img image.Image, steps int) image.Image {
	switch ((steps % 4) + 4) % 4 {
	case 1:
		return rotate90CW(img)
	case 2:
		return rotate180(img)
	case 3:
		return rotate270CW(img)
	default:
		return img
	}
}

// flipH keeps a direct pixel loop because a shared coordinate callback or
// transform branch would add work to every pixel in this hot path.
//
//goland:noinspection DuplicatedCode
func flipH(src image.Image) image.Image {
	pixels := orientationPixels(src)
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewRGBA(image.Rect(0, 0, w, h))

	for y := range h {
		for x := range w {
			out.SetRGBA64(w-1-x, y, pixels.RGBA64At(b.Min.X+x, b.Min.Y+y))
		}
	}

	return out
}

// flipV keeps a direct pixel loop because a shared coordinate callback or
// transform branch would add work to every pixel in this hot path.
//
//goland:noinspection DuplicatedCode
func flipV(src image.Image) image.Image {
	pixels := orientationPixels(src)
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewRGBA(image.Rect(0, 0, w, h))

	for y := range h {
		for x := range w {
			out.SetRGBA64(x, h-1-y, pixels.RGBA64At(b.Min.X+x, b.Min.Y+y))
		}
	}

	return out
}

func rotate180(src image.Image) image.Image {
	pixels := orientationPixels(src)
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewRGBA(image.Rect(0, 0, w, h))

	for y := range h {
		for x := range w {
			out.SetRGBA64(w-1-x, h-1-y, pixels.RGBA64At(b.Min.X+x, b.Min.Y+y))
		}
	}

	return out
}

// rotate90CW rotates the image 90 degrees clockwise, swapping width and
// height. It keeps a direct pixel loop because a shared coordinate callback
// or transform branch would add work to every pixel in this hot path.
//
//goland:noinspection DuplicatedCode
func rotate90CW(src image.Image) image.Image {
	pixels := orientationPixels(src)
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewRGBA(image.Rect(0, 0, h, w))

	for y := range h {
		for x := range w {
			out.SetRGBA64(h-1-y, x, pixels.RGBA64At(b.Min.X+x, b.Min.Y+y))
		}
	}

	return out
}

// rotate270CW rotates the image 270 degrees clockwise (90 counterclockwise),
// swapping width and height. It keeps a direct pixel loop because a shared
// coordinate callback or transform branch would add work to every pixel in
// this hot path.
//
//goland:noinspection DuplicatedCode
func rotate270CW(src image.Image) image.Image {
	pixels := orientationPixels(src)
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewRGBA(image.Rect(0, 0, h, w))

	for y := range h {
		for x := range w {
			out.SetRGBA64(y, w-1-x, pixels.RGBA64At(b.Min.X+x, b.Min.Y+y))
		}
	}

	return out
}
