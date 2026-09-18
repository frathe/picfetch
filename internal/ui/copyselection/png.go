package copyselection

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"

	"github.com/frathe/picfetch/internal/imaging"
)

// PNG returns a zero-origin, eight-bit PNG containing bounds from src.
// The current Max file size setting bounds the pixel footprint, encoder rows
// and encoded output independently; compression state and buffer growth add
// overhead. One encode retains its captured limit if the setting changes.
func PNG(src image.Image, bounds image.Rectangle) ([]byte, error) {
	if src == nil {
		return nil, fmt.Errorf("copy selection PNG: nil source")
	}
	if bounds.Empty() || bounds.Intersect(src.Bounds()) != bounds {
		return nil, fmt.Errorf("copy selection PNG: bounds %v outside source %v", bounds, src.Bounds())
	}
	maxPNGBytes := imaging.MaxEncodedBytes()
	width, height := bounds.Dx(), bounds.Dy()
	// image/png retains five filter rows and one previous row, each with a
	// filter byte. Division keeps both checks safe from dimension overflow.
	pixelLimit := maxPNGBytes / 4
	rowPixelLimit := (maxPNGBytes/6 - 1) / 4
	if width <= 0 || height <= 0 || int64(width) > pixelLimit/int64(height) || int64(width) > rowPixelLimit {
		return nil, selectionTooLargeError{limit: maxPNGBytes}
	}

	buf := limitedBuffer{maxBytes: maxPNGBytes}
	crop := cropImage{src: src, bounds: bounds}
	if err := png.Encode(&buf, crop); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// cropImage presents the selection at a zero origin without materializing a
// second four-byte-per-pixel raster beside the decoded source.
type cropImage struct {
	src    image.Image
	bounds image.Rectangle
}

func (_ cropImage) ColorModel() color.Model { return color.NRGBAModel }
func (c cropImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, c.bounds.Dx(), c.bounds.Dy())
}
func (c cropImage) At(x, y int) color.Color {
	return c.src.At(x+c.bounds.Min.X, y+c.bounds.Min.Y)
}

type limitedBuffer struct {
	bytes.Buffer
	maxBytes int64
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if int64(len(p)) > b.maxBytes-int64(b.Len()) {
		return 0, selectionTooLargeError{limit: b.maxBytes}
	}
	return b.Buffer.Write(p)
}

type selectionTooLargeError struct{ limit int64 }

func (e selectionTooLargeError) Error() string {
	return fmt.Sprintf("copy selection exceeds the %d-byte size limit", e.limit)
}
