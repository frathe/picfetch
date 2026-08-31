package copyselection

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	"image/png"
)

// PNG returns a zero-origin PNG containing bounds from src.
func PNG(src image.Image, bounds image.Rectangle) ([]byte, error) {
	if src == nil {
		return nil, fmt.Errorf("copy selection PNG: nil source")
	}
	if bounds.Empty() || bounds.Intersect(src.Bounds()) != bounds {
		return nil, fmt.Errorf("copy selection PNG: bounds %v outside source %v", bounds, src.Bounds())
	}

	crop := image.NewNRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
	draw.Draw(crop, crop.Bounds(), src, bounds.Min, draw.Src)

	var buf bytes.Buffer
	if err := png.Encode(&buf, crop); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
