package widgets

import (
	"bytes"
	"image"
	"image/draw"
	"os"
	"testing"

	"golang.org/x/image/webp"
)

func TestCompactGazeAtlasesPreserveFrames(t *testing.T) {
	for _, spec := range []struct{ name, embedded, original string }{
		{"Trane", "../assets/trane.webp", "../../../assets/trane/codex-pet/spritesheet.webp"},
		{"Finis", "../help/finis.webp", "../../../assets/finis/spritesheet.webp"},
	} {
		t.Run(spec.name, func(t *testing.T) {
			data, err := os.ReadFile(spec.embedded)
			if err != nil {
				t.Fatal(err)
			}
			config, err := webp.DecodeConfig(bytes.NewReader(data))
			if err != nil {
				t.Fatal(err)
			}
			if config.Width != 17*192 || config.Height != 208 {
				t.Fatalf("embedded atlas is %dx%d; want only 17 original cells in one row", config.Width, config.Height)
			}
			frames, err := DecodeGazeAtlas(data, nil)
			if err != nil {
				t.Fatal(err)
			}
			originalData, err := os.ReadFile(spec.original)
			if err != nil {
				t.Fatal(err)
			}
			original, err := webp.Decode(bytes.NewReader(originalData))
			if err != nil {
				t.Fatal(err)
			}
			for i, cell := range []int{72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 6} {
				bounds := image.Rect(0, 0, 192, 208)
				if frames[i].Bounds() != bounds {
					t.Fatalf("frame %d bounds = %v", i, frames[i].Bounds())
				}
				want, got := image.NewNRGBA(bounds), image.NewNRGBA(bounds)
				draw.Draw(want, bounds, original, image.Pt(cell%8*192, cell/8*208), draw.Src)
				draw.Draw(got, bounds, frames[i], image.Point{}, draw.Src)
				if !bytes.Equal(got.Pix, want.Pix) {
					t.Fatalf("frame %d changed original pixels", i)
				}
			}
		})
	}
}

func TestGazeDecoderRejectsFullAtlas(t *testing.T) {
	data, err := os.ReadFile("../../../assets/trane/codex-pet/spritesheet.webp")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeGazeAtlas(data, nil); err == nil {
		t.Fatal("accepted unused animation rows in the app gaze atlas")
	}
}
