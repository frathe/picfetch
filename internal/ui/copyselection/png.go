package copyselection

import (
	"bytes"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
)

const maxPNGBytes = 64 * 1024 * 1024

var errSelectionTooLarge = errors.New("copy selection exceeds the 64 MiB memory limit")

// PNG returns a zero-origin PNG containing bounds from src.
func PNG(src image.Image, bounds image.Rectangle) ([]byte, error) {
	if src == nil {
		return nil, fmt.Errorf("copy selection PNG: nil source")
	}
	if bounds.Empty() || bounds.Intersect(src.Bounds()) != bounds {
		return nil, fmt.Errorf("copy selection PNG: bounds %v outside source %v", bounds, src.Bounds())
	}
	width, height := bounds.Dx(), bounds.Dy()
	if width <= 0 || height <= 0 || width > maxPNGBytes/4 || height > maxPNGBytes/4 ||
		int64(width)*int64(height) > maxPNGBytes/4 {
		return nil, errSelectionTooLarge
	}

	buf := limitedBuffer{remaining: maxPNGBytes}
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

func (c cropImage) ColorModel() color.Model { return c.src.ColorModel() }
func (c cropImage) Bounds() image.Rectangle {
	return image.Rect(0, 0, c.bounds.Dx(), c.bounds.Dy())
}
func (c cropImage) At(x, y int) color.Color {
	return c.src.At(x+c.bounds.Min.X, y+c.bounds.Min.Y)
}

type limitedBuffer struct {
	bytes.Buffer
	remaining int
}

func (b *limitedBuffer) Write(p []byte) (int, error) {
	if len(p) > b.remaining {
		return 0, errSelectionTooLarge
	}
	n, err := b.Buffer.Write(p)
	b.remaining -= n
	return n, err
}
