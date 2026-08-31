package copyselection

import (
	"errors"
	"fmt"
	"image"

	"github.com/frathe/picfetch/internal/imaging"
)

// Source is the oriented image captured when Copy Selection mode begins.
// Raster and RAW inputs retain the displayed frame. SVG retains the parsed
// vector plus logical size and view rotation so Encode rasterizes at source
// resolution instead of the zoom-dependent canvas size.
type Source struct {
	raster    image.Image
	vector    *imaging.Vector
	logical   image.Point
	rotation  int
	rasterize func(*imaging.Vector, int, int) (image.Image, error)
}

// RasterSource captures an oriented raster or RAW preview frame.
func RasterSource(img image.Image) Source {
	return Source{raster: img}
}

// VectorSource captures a parsed SVG, its logical size, and the view-only
// rotation Encode must compose after rasterizing at that logical size.
func VectorSource(vector *imaging.Vector, logical image.Point, rotation int, rasterize func(*imaging.Vector, int, int) (image.Image, error)) Source {
	return Source{vector: vector, logical: logical, rotation: rotation, rasterize: rasterize}
}

// orientedLogical is the captured SVG's logical size with view-only quarter
// turns applied — the single definition of the mode's vector image-space.
// Bounds and cropBounds both read it, so the rectangle the selection is
// drawn against and the rectangle the crop is validated against cannot
// drift apart.
func (s Source) orientedLogical() (w, h int) {
	w, h = s.logical.X, s.logical.Y
	if s.rotation%2 != 0 {
		w, h = h, w
	}
	return w, h
}

// Encode returns a zero-origin PNG of bounds from the captured source.
func (s Source) Encode(bounds image.Rectangle) ([]byte, error) {
	return s.encodeWith(PNG, bounds)
}

// encodeWith is the one crop-and-encode pipeline: resolve pixels, map
// bounds into them, hand both to encode. Feature.Encode injects its
// configurable encoder here; any step added to this pipeline reaches the
// seam-installed test encoders too, which is the point of there being
// exactly one.
func (s Source) encodeWith(encode func(image.Image, image.Rectangle) ([]byte, error), bounds image.Rectangle) ([]byte, error) {
	pixels, err := s.pixels()
	if err != nil {
		return nil, err
	}
	crop, err := s.cropBounds(bounds, pixels.Bounds())
	if err != nil {
		return nil, err
	}
	return encode(pixels, crop)
}

// Bounds is the oriented image rectangle Copy Selection mode uses as its
// image-space. Raster frames report their pixel size; SVG reports logical
// size with view-only quarter turns applied.
func (s Source) Bounds() image.Rectangle {
	if s.vector != nil {
		w, h := s.orientedLogical()
		if w <= 0 || h <= 0 {
			return image.Rectangle{}
		}
		return image.Rect(0, 0, w, h)
	}
	if s.raster == nil {
		return image.Rectangle{}
	}
	b := s.raster.Bounds()
	return image.Rect(0, 0, b.Dx(), b.Dy())
}

func (s Source) pixels() (image.Image, error) {
	if s.vector == nil {
		if s.raster == nil {
			return nil, errors.New("copy selection source is unavailable")
		}
		return s.raster, nil
	}
	if s.logical.X <= 0 || s.logical.Y <= 0 || s.rasterize == nil {
		return nil, errors.New("copy selection vector source is unavailable")
	}

	frame, err := s.rasterize(s.vector, s.logical.X, s.logical.Y)
	if err != nil {
		return nil, err
	}
	return imaging.RotateSteps(frame, s.rotation), nil
}

// cropBounds maps the feature's oriented logical SVG coordinates onto the
// actual raster returned by RasterAt. They are identical below the safety
// ceiling; above it, RasterAt scales both axes down and the crop must follow
// that same scale. Raster sources already use literal pixel coordinates.
func (s Source) cropBounds(bounds, pixels image.Rectangle) (image.Rectangle, error) {
	if s.vector == nil {
		return bounds, nil
	}

	w, h := s.orientedLogical()
	logical := image.Rect(0, 0, w, h)
	if logical.Empty() || bounds.Empty() || bounds.Intersect(logical) != bounds || pixels.Empty() {
		return image.Rectangle{}, fmt.Errorf("copy selection bounds %v outside SVG source %v", bounds, logical)
	}

	return image.Rect(
		scaleFloor(bounds.Min.X, logical.Dx(), pixels.Dx()),
		scaleFloor(bounds.Min.Y, logical.Dy(), pixels.Dy()),
		scaleCeil(bounds.Max.X, logical.Dx(), pixels.Dx()),
		scaleCeil(bounds.Max.Y, logical.Dy(), pixels.Dy()),
	), nil
}

func scaleFloor(value, from, to int) int {
	return int(int64(value) * int64(to) / int64(from))
}

func scaleCeil(value, from, to int) int {
	numerator := int64(value) * int64(to)
	return int((numerator + int64(from) - 1) / int64(from))
}
