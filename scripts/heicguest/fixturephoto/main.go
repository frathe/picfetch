//go:build wasip1 && wasm

// fixturephoto writes one ordinary 12-megapixel color gradient for bounded
// decoder throughput qualification. No input, fuzzing, or malformed data.
package main

import (
	"image"
	"os"

	"github.com/gen2brain/h265/heic"
)

func main() {
	const width, height = 4032, 3024
	img := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			offset := y*img.Stride + x*4
			img.Pix[offset] = byte(x * 255 / (width - 1))
			img.Pix[offset+1] = byte(y * 255 / (height - 1))
			img.Pix[offset+2] = 128
			img.Pix[offset+3] = 255
		}
	}
	if err := heic.Encode(os.Stdout, img, heic.EncodeOptions{BitDepth: 8, Chroma: heic.Chroma420, Lossless: true}); err != nil {
		os.Exit(1)
	}
}
