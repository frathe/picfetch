//go:build wasip1 && wasm

// fixturegen writes one ordinary PicFetch-owned 16x16 gradient with explicit
// ten-bit samples. It is a development tool, not application encoding support.
package main

import (
	"image"
	"image/color"
	"os"

	"github.com/gen2brain/h265/heic"
)

func main() {
	img := image.NewNRGBA64(image.Rect(0, 0, 16, 16))
	for y := 0; y < 16; y++ {
		for x := 0; x < 16; x++ {
			gray := uint16((x + y) * 2000)
			img.SetNRGBA64(x, y, color.NRGBA64{R: gray, G: gray, B: gray, A: 65535})
		}
	}
	if err := heic.Encode(os.Stdout, img, heic.EncodeOptions{BitDepth: 10, Chroma: heic.Chroma444, Lossless: true}); err != nil {
		os.Exit(1)
	}
}
