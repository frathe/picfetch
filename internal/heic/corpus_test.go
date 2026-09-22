package heic

import (
	"errors"
	"image/color"
	"os"
	"strings"
	"testing"
)

func TestHEICNativeCorpus(t *testing.T) {
	if os.Getenv("PICFETCH_HEIC_NATIVE_TEST") != "1" {
		t.Skip("requires explicit installed-provider corpus qualification")
	}
	client := NewClient("")
	t.Cleanup(func() { client.Stop(); client.Wait() })
	colors := []color.NRGBA{{R: 192, G: 32, B: 64, A: 255}, {R: 32, G: 160, B: 64, A: 255},
		{R: 32, G: 64, B: 192, A: 255}, {R: 160, G: 128, B: 32, A: 255}}
	for _, fixture := range []struct {
		name  string
		size  int
		kind  string
		order [4]int
	}{
		{name: "icc-srgb8", size: 64, kind: "gray"},
		{name: "icc-srgb10", size: 64, kind: "gray"},
		{name: "icc-p3-linear8", size: 64, kind: "icc"},
		{name: "icc-p3-linear10", size: 64, kind: "icc"},
		{name: "color8", size: 64, order: [4]int{0, 1, 2, 3}},
		{name: "color10", size: 64, order: [4]int{0, 1, 2, 3}},
		{name: "grid8", size: 128, order: [4]int{0, 1, 2, 3}},
		{name: "grid10", size: 128, order: [4]int{0, 1, 2, 3}},
		{name: "mirror-horizontal8", size: 64, order: [4]int{1, 0, 3, 2}},
		{name: "mirror-horizontal10", size: 64, order: [4]int{1, 0, 3, 2}},
		{name: "mirror-rotate8", size: 64, order: [4]int{3, 1, 2, 0}},
		{name: "alpha-straight8", size: 64, kind: "alpha"},
		{name: "alpha-straight10", size: 64, kind: "alpha"},
		{name: "alpha-premultiplied8", size: 64, kind: "alpha"},
		{name: "alpha-premultiplied10", size: 64, kind: "alpha"},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			data, err := os.ReadFile("testdata/" + fixture.name + ".heic")
			if err != nil {
				t.Fatal(err)
			}
			request := Request{Pixels: true, MaxEncodedBytes: 64 * 1024, MaxPixels: int64(fixture.size * fixture.size)}
			result, err := client.Read(t.Context(), data, request)
			if fixture.name == "mirror-horizontal10" && errors.Is(err, ErrUnsupported) &&
				strings.Contains(err.Error(), "libheif error 4/0:") &&
				strings.Contains(err.Error(), "Can currently only mirror images with 8 bits per pixel") {
				// libheif 1.17.6 cannot render this transform. The native
				// limitation remains a per-file error, not a color-policy gate.
				t.Logf("native provider limitation (no successful rendition claimed): %v", err)
				if checkErr := client.Check(t.Context()); checkErr != nil {
					t.Fatalf("unsupported transform damaged decoder availability: %v", checkErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if result.Width != fixture.size || result.Height != fixture.size || result.Stride != fixture.size*4 || len(result.Pixels) != fixture.size*fixture.size*4 {
				t.Fatalf("native geometry: %dx%d stride=%d bytes=%d", result.Width, result.Height, result.Stride, len(result.Pixels))
			}
			for quadrant := range 4 {
				x, y := fixture.size/4+(quadrant%2)*fixture.size/2, fixture.size/4+(quadrant/2)*fixture.size/2
				pixel := result.Pixels[y*result.Stride+x*4:][:4]
				switch fixture.kind {
				case "gray":
					assertGrayPixel(t, result, x, y, (quadrant%2)*255)
				case "icc":
					// The product delegates color management to the OS. These
					// shapes remain recognizable with or without profile correction.
					r, g, b := int(pixel[0]), int(pixel[1]), int(pixel[2])
					recognizable := []bool{r > g+30 && r > b+30, g > r+30 && g > b+30,
						b > r+30 && b > g+30, r > b+30 && g > b+30}
					if !recognizable[quadrant] || pixel[3] != 255 {
						t.Fatalf("ICC quadrant %d lost recognizable colored pixels: %v", quadrant, pixel)
					}
				case "alpha":
					alpha := []byte{0, 128, 192, 255}[quadrant]
					assertCorpusChannels(t, pixel[3:], []byte{alpha}, 2)
					if alpha != 0 {
						assertCorpusChannels(t, pixel[:3], []byte{160, 80, 40}, 6)
					}
				default:
					want := colors[fixture.order[quadrant]]
					assertCorpusChannels(t, pixel, []byte{want.R, want.G, want.B, want.A}, 3)
				}
			}
			request.Pixels = false
			metadata, err := client.Read(t.Context(), data, request)
			if err != nil || metadata.Width != result.Width || metadata.Height != result.Height || len(metadata.Pixels) != 0 {
				t.Fatalf("native metadata/full pixel geometry differs: %dx%d, %v", metadata.Width, metadata.Height, err)
			}
			t.Log(result.Provider)
		})
	}
}

func assertCorpusChannels(t *testing.T, got, want []byte, tolerance int) {
	t.Helper()
	for i, value := range got {
		if delta := int(value) - int(want[i]); delta < -tolerance || delta > tolerance {
			t.Fatalf("native channels %v, want %v within %d levels", got, want, tolerance)
		}
	}
}
