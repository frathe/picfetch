// Command appassets derives embedded artwork from retained originals.
// Generation uses cwebp; checking requires only Go and compares decoded pixels.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"

	xdraw "golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

type asset struct {
	source, output string
	width, height  int
	gaze           bool
}

// Ordinary illustrations target twice their maximum logical display size.
// Gaze frames retain their original pixels; icons have separate package needs.
var assets = []asset{
	{"assets/trane/codex-pet/spritesheet.webp", "internal/ui/assets/trane.webp", 17 * 192, 208, true},
	{"assets/finis/spritesheet.webp", "internal/ui/help/finis.webp", 17 * 192, 208, true},
	{"assets/ui-originals/welcome.webp", "internal/ui/assets/welcome.webp", 360, 434, false},
	{"assets/ui-originals/placeholder.webp", "internal/ui/assets/placeholder.webp", 360, 394, false},
	{"assets/ui-originals/digging.webp", "internal/ui/assets/digging.webp", 360, 240, false},
	{"assets/ui-originals/comparingImages.webp", "internal/ui/assets/comparingImages.webp", 520, 475, false},
	{"assets/ui-originals/explorer-intro.png", "internal/ui/assets/explorer-intro.png", 640, 480, false},
	{"assets/trane/TaneWithFrame.webp", "internal/ui/help/TaneWithFrame.webp", 629, 440, false},
	{"assets/trane/trane_digging.webp", "internal/ui/help/trane_digging.webp", 660, 440, false},
	{"assets/trane/trane_wags.webp", "internal/ui/help/trane_wags.webp", 365, 440, false},
}

func main() {
	root := flag.String("root", ".", "repository root")
	check := flag.Bool("check", false, "check dimensions and exact decoded pixels without writing")
	flag.Parse()
	for _, spec := range assets {
		if err := process(*root, spec, *check); err != nil {
			_, _ = fmt.Fprintf(os.Stderr, "%s: %v\n", spec.output, err)
			os.Exit(1)
		}
	}
}

func process(root string, spec asset, check bool) error {
	source, err := readImage(filepath.Join(root, spec.source))
	if err != nil {
		return err
	}
	want, err := derive(source, spec)
	if err != nil {
		return err
	}
	out := filepath.Join(root, spec.output)
	if check {
		got, err := readImage(out)
		if err != nil {
			return err
		}
		if got.Bounds() != want.Bounds() {
			return fmt.Errorf("dimensions %v, want %v; run make generate-app-assets", got.Bounds(), want.Bounds())
		}
		pixels := image.NewNRGBA(got.Bounds())
		draw.Draw(pixels, pixels.Bounds(), got, got.Bounds().Min, draw.Src)
		if !spec.gaze {
			// Lossless optimizers can rewrite RGB beneath fully transparent pixels.
			// Gaze atlases retain their stricter original-pixel contract.
			clearTransparentRGB(pixels)
			clearTransparentRGB(want)
		}
		if !bytes.Equal(pixels.Pix, want.Pix) {
			return fmt.Errorf("decoded pixels differ from source; run make generate-app-assets")
		}
		return nil
	}
	var encoded bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestCompression}
	if err := encoder.Encode(&encoded, want); err != nil {
		return err
	}
	data := encoded.Bytes()
	if filepath.Ext(out) == ".webp" {
		data, err = encodeWebP(data)
		if err != nil {
			return err
		}
	}
	if err := os.WriteFile(out, data, 0644); err != nil {
		return err
	}
	_, _ = fmt.Printf("%s: %dx%d, %d bytes\n", spec.output, spec.width, spec.height, len(data))
	return nil
}

func clearTransparentRGB(pixels *image.NRGBA) {
	for offset := 0; offset < len(pixels.Pix); offset += 4 {
		if pixels.Pix[offset+3] == 0 {
			clear(pixels.Pix[offset : offset+3])
		}
	}
}

func encodeWebP(data []byte) ([]byte, error) {
	dir, err := os.MkdirTemp("", "picfetch-art-")
	if err != nil {
		return nil, err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	input, output := filepath.Join(dir, "input.png"), filepath.Join(dir, "output.webp")
	if err := os.WriteFile(input, data, 0600); err != nil {
		return nil, err
	}
	// -exact retains RGB values beneath transparent pixels as well.
	command := exec.Command("cwebp", "-quiet", "-lossless", "-exact", "-z", "9", input, "-o", output)
	if log, err := command.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("cwebp: %w: %s", err, log)
	}
	return os.ReadFile(output)
}

func readImage(path string) (image.Image, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	picture, _, err := image.Decode(bytes.NewReader(data))
	return picture, err
}

func derive(source image.Image, spec asset) (*image.NRGBA, error) {
	result := image.NewNRGBA(image.Rect(0, 0, spec.width, spec.height))
	if !spec.gaze {
		if spec.width > source.Bounds().Dx() || spec.height > source.Bounds().Dy() {
			return nil, fmt.Errorf("source is smaller than the display target")
		}
		xdraw.CatmullRom.Scale(result, result.Bounds(), source, source.Bounds(), draw.Src, nil)
		return result, nil
	}
	if source.Bounds() != image.Rect(0, 0, 8*192, 11*208) {
		return nil, fmt.Errorf("source must be a full 8x11 Codex atlas")
	}
	for index := range 17 {
		column, row := index%8, 9+index/8
		if index == 16 {
			column, row = 6, 0
		}
		cell := image.Rect(index*192, 0, (index+1)*192, 208)
		draw.Draw(result, cell, source, image.Pt(column*192, row*208), draw.Src)
	}
	return result, nil
}
