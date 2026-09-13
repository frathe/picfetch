package main

import (
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

func TestGazeAtlasKeepsOnlyUsedCells(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 8*192, 11*208))
	for y := range source.Bounds().Dy() {
		for x := range source.Bounds().Dx() {
			source.SetNRGBA(x, y, color.NRGBA{R: uint8(x), G: uint8(y), B: uint8(y/208*8 + x/192), A: uint8(x + y)})
		}
	}
	got, err := derive(source, asset{width: 17 * 192, height: 208, gaze: true})
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds() != image.Rect(0, 0, 17*192, 208) {
		t.Fatalf("compact bounds = %v", got.Bounds())
	}
	for i, cell := range []int{72, 73, 74, 75, 76, 77, 78, 79, 80, 81, 82, 83, 84, 85, 86, 87, 6} {
		for y := range 208 {
			for x := range 192 {
				if got.NRGBAAt(i*192+x, y) != source.NRGBAAt(cell%8*192+x, cell/8*208+y) {
					t.Fatalf("frame %d pixel %d,%d changed", i, x, y)
				}
			}
		}
	}
}

func TestAssetCheckRejectsStalePixels(t *testing.T) {
	root := t.TempDir()
	spec := asset{source: "source.png", output: "output.png", width: 10, height: 10}
	write := func(name string, picture image.Image) {
		t.Helper()
		file, err := os.Create(filepath.Join(root, name))
		if err != nil {
			t.Fatal(err)
		}
		encodeErr := png.Encode(file, picture)
		closeErr := file.Close()
		if encodeErr != nil {
			t.Fatal(encodeErr)
		}
		if closeErr != nil {
			t.Fatal(closeErr)
		}
	}
	write(spec.source, image.NewNRGBA(image.Rect(0, 0, 20, 20)))
	if err := process(root, spec, false); err != nil {
		t.Fatal(err)
	}
	if err := process(root, spec, true); err != nil {
		t.Fatal(err)
	}
	wrong := image.NewNRGBA(image.Rect(0, 0, 10, 10))
	wrong.SetNRGBA(5, 5, color.NRGBA{R: 255, A: 255})
	write(spec.output, wrong)
	if err := process(root, spec, true); err == nil {
		t.Fatal("check accepted different pixels with the correct dimensions")
	}
}

func TestDeriveRejectsUnexpectedSources(t *testing.T) {
	for _, spec := range []asset{{width: 17 * 192, height: 208, gaze: true}, {width: 640, height: 480}} {
		if _, err := derive(image.NewNRGBA(image.Rect(0, 0, 100, 100)), spec); err == nil {
			t.Fatal("accepted undersized source")
		}
	}
}

func TestResizePreservesTransparencyAndSolidInterior(t *testing.T) {
	source := image.NewNRGBA(image.Rect(0, 0, 100, 100))
	for y := 20; y < 80; y++ {
		for x := 20; x < 80; x++ {
			source.SetNRGBA(x, y, color.NRGBA{R: 240, G: 180, B: 30, A: 255})
		}
	}
	got, err := derive(source, asset{width: 50, height: 50})
	if err != nil {
		t.Fatal(err)
	}
	if got.Bounds() != image.Rect(0, 0, 50, 50) || got.NRGBAAt(0, 0).A != 0 || got.NRGBAAt(25, 25) != source.NRGBAAt(50, 50) {
		t.Fatal("resizing changed dimensions, background alpha, or solid interior color")
	}
}

func TestAssetCheckAfterLosslessOptimization(t *testing.T) {
	for _, tc := range []struct {
		name     string
		x, y     int
		change   func(color.NRGBA) color.NRGBA
		wantPass bool
	}{
		{"transparent RGB", 0, 0, func(c color.NRGBA) color.NRGBA { c.R ^= 255; return c }, true},
		{"opaque RGB", 2, 2, func(c color.NRGBA) color.NRGBA { c.R ^= 1; return c }, false},
		{"low-alpha RGB", 1, 1, func(c color.NRGBA) color.NRGBA { c.R ^= 1; return c }, false},
		{"transparent to visible", 0, 0, func(c color.NRGBA) color.NRGBA { c.A = 1; return c }, false},
		{"visible to transparent", 1, 1, func(c color.NRGBA) color.NRGBA { c.A = 0; return c }, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := t.TempDir()
			spec := asset{source: "source.png", output: "output.png", width: 4, height: 4}
			source := image.NewNRGBA(image.Rect(0, 0, 4, 4))
			source.SetNRGBA(1, 1, color.NRGBA{R: 50, G: 80, B: 100, A: 1})
			source.SetNRGBA(2, 2, color.NRGBA{R: 150, G: 80, B: 100, A: 255})
			writeAssetTestPNG(t, filepath.Join(root, spec.source), source)
			if err := process(root, spec, false); err != nil {
				t.Fatal(err)
			}
			if err := process(root, spec, true); err != nil {
				t.Fatal(err)
			}
			output, err := readImage(filepath.Join(root, spec.output))
			if err != nil {
				t.Fatal(err)
			}
			optimized := image.NewNRGBA(output.Bounds())
			draw.Draw(optimized, optimized.Bounds(), output, output.Bounds().Min, draw.Src)
			optimized.SetNRGBA(tc.x, tc.y, tc.change(optimized.NRGBAAt(tc.x, tc.y)))
			writeAssetTestPNG(t, filepath.Join(root, spec.output), optimized)
			if err := process(root, spec, true); (err == nil) != tc.wantPass {
				t.Fatalf("check error = %v, want pass = %v", err, tc.wantPass)
			}
		})
	}
}

func TestGazeAssetCheckRejectsTransparentRGBChanges(t *testing.T) {
	root := t.TempDir()
	spec := asset{source: "source.png", output: "output.png", width: 17 * 192, height: 208, gaze: true}
	writeAssetTestPNG(t, filepath.Join(root, spec.source), image.NewNRGBA(image.Rect(0, 0, 8*192, 11*208)))
	if err := process(root, spec, false); err != nil {
		t.Fatal(err)
	}
	if err := process(root, spec, true); err != nil {
		t.Fatal(err)
	}
	changed := image.NewNRGBA(image.Rect(0, 0, spec.width, spec.height))
	changed.SetNRGBA(0, 0, color.NRGBA{R: 255})
	writeAssetTestPNG(t, filepath.Join(root, spec.output), changed)
	if err := process(root, spec, true); err == nil {
		t.Fatal("gaze check accepted changed RGB beneath a fully transparent pixel")
	}
}

func writeAssetTestPNG(t *testing.T, path string, picture image.Image) {
	t.Helper()
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	encodeErr := png.Encode(file, picture)
	closeErr := file.Close()
	if encodeErr != nil {
		t.Fatal(encodeErr)
	}
	if closeErr != nil {
		t.Fatal(closeErr)
	}
}
