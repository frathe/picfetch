package mosaic

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestMain(m *testing.M) {
	test.NewApp()
	os.Exit(m.Run())
}

func TestGenerate_DeterministicAndCoverage(t *testing.T) {
	source := mosaicPNG(t, "source.png", 12, 8, func(x, y int) color.NRGBA {
		return color.NRGBA{R: uint8(x * 15), G: uint8(y * 20), B: 80, A: 255}
	})
	request := mustRequest(t, []fyne.URI{source}, image.Pt(83, 47), DefaultSettings(), 987)

	first, err := Generate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Generate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Bounds() != image.Rect(0, 0, 83, 47) {
		t.Fatalf("result bounds = %v", first.Bounds())
	}
	a := first.Image().(*image.NRGBA)
	b := second.Image().(*image.NRGBA)
	if !bytes.Equal(a.Pix, b.Pix) {
		t.Fatal("fixed generation inputs produced different pixels")
	}
	for i := 3; i < len(a.Pix); i += 4 {
		if a.Pix[i] != 255 {
			t.Fatalf("output pixel %d has alpha %d, want fully covered", i/4, a.Pix[i])
		}
	}
}

func TestGenerate_CoverageHasNoUntouchedCanvasPixel(t *testing.T) {
	source := mosaicPNG(t, "white.png", 10, 10, func(_, _ int) color.NRGBA {
		return color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	})
	result, err := Generate(context.Background(), mustRequest(
		t, []fyne.URI{source}, image.Pt(83, 47), DefaultSettings(), 987,
	))
	if err != nil {
		t.Fatal(err)
	}
	pixels := result.Image().(*image.NRGBA)
	for y := pixels.Bounds().Min.Y; y < pixels.Bounds().Max.Y; y++ {
		for x := pixels.Bounds().Min.X; x < pixels.Bounds().Max.X; x++ {
			pixel := pixels.NRGBAAt(x, y)
			if pixel == (color.NRGBA{R: 28, G: 30, B: 34, A: 255}) {
				t.Fatalf("canvas pixel at (%d,%d) was never composited", x, y)
			}
		}
	}
}

func TestGenerate_SourceFidelity(t *testing.T) {
	source := mosaicPNG(t, "two-colors.png", 20, 10, func(x, _ int) color.NRGBA {
		if x < 10 {
			return color.NRGBA{R: 255, A: 255}
		}
		return color.NRGBA{B: 255, A: 255}
	})
	settings := DefaultSettings()
	settings.SizeVariation = 0
	settings.Overlap = 0
	settings.MaximumRotation = 0
	request := mustRequest(t, []fyne.URI{source}, image.Pt(80, 50), settings, 2)

	result, err := Generate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	var red, blue bool
	image := result.Image().(*image.NRGBA)
	for y := range image.Bounds().Dy() {
		for x := range image.Bounds().Dx() {
			pixel := image.NRGBAAt(x, y)
			red = red || pixel.R > 230 && pixel.B < 25
			blue = blue || pixel.B > 230 && pixel.R < 25
		}
	}
	if !red || !blue {
		t.Fatalf("source colors missing after generation: red=%v blue=%v", red, blue)
	}
}

func TestGenerate_SourceFidelityCanonicalFormats(t *testing.T) {
	oriented := storage.NewFileURI(uitest.WriteTempFile(
		t, "oriented.jpg", uitest.EncodeOrientedJPEG(t, 12, 6, 6),
	))
	animated := storage.NewFileURI(uitest.WriteTempFile(
		t, "animated.gif", uitest.EncodeAnimatedGIF(t, 9, 7,
			[]color.Color{color.RGBA{R: 255, A: 255}, color.RGBA{B: 255, A: 255}},
			[]int{5, 5}),
	))
	transparent := mosaicPNG(t, "transparent.png", 8, 8, func(_, _ int) color.NRGBA {
		return color.NRGBA{G: 220, A: 80}
	})

	tests := []struct {
		name       string
		source     fyne.URI
		wantBounds image.Rectangle
		wantVector bool
		check      func(*testing.T, image.Image)
	}{
		{
			name:       "EXIF orientation",
			source:     oriented,
			wantBounds: image.Rect(0, 0, 6, 12),
			check: func(t *testing.T, pixels image.Image) {
				t.Helper()
				top := color.NRGBAModel.Convert(pixels.At(3, 2)).(color.NRGBA)
				bottom := color.NRGBAModel.Convert(pixels.At(3, 9)).(color.NRGBA)
				if top.R < 220 || bottom.B < 220 {
					t.Fatalf("orientation pixels top=%v bottom=%v", top, bottom)
				}
			},
		},
		{
			name:       "first animated GIF frame",
			source:     animated,
			wantBounds: image.Rect(0, 0, 9, 7),
			check: func(t *testing.T, pixels image.Image) {
				t.Helper()
				pixel := color.NRGBAModel.Convert(pixels.At(4, 3)).(color.NRGBA)
				if pixel.R < 240 || pixel.B > 15 {
					t.Fatalf("first GIF frame pixel = %v, want red", pixel)
				}
			},
		},
		{name: "SVG vector", source: uitest.TempSVGURI(t, "wide.svg", 20, 10), wantBounds: image.Rect(0, 0, 520, 260), wantVector: true},
		{name: "RAW preview", source: uitest.TempRAWURI(t, "photo.cr2", 13, 7, color.RGBA{R: 210, A: 255}), wantBounds: image.Rect(0, 0, 13, 7)},
		{
			name:       "transparent raster",
			source:     transparent,
			wantBounds: image.Rect(0, 0, 8, 8),
			check: func(t *testing.T, pixels image.Image) {
				t.Helper()
				if alpha := color.NRGBAModel.Convert(pixels.At(4, 4)).(color.NRGBA).A; alpha != 80 {
					t.Fatalf("decoded alpha = %d, want 80", alpha)
				}
			},
		},
		{name: "portrait raster", source: mosaicPNG(t, "portrait.png", 7, 13, func(_, _ int) color.NRGBA { return color.NRGBA{R: 255, A: 255} }), wantBounds: image.Rect(0, 0, 7, 13)},
		{name: "landscape raster", source: mosaicPNG(t, "landscape.png", 13, 7, func(_, _ int) color.NRGBA { return color.NRGBA{B: 255, A: 255} }), wantBounds: image.Rect(0, 0, 13, 7)},
	}

	settings := DefaultSettings()
	settings.SizeVariation = 0
	settings.Overlap = 0
	settings.MaximumRotation = 0
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loaded, err := loadCanonicalSource(context.Background(), tt.source)
			if err != nil {
				t.Fatal(err)
			}
			if loaded.bounds != tt.wantBounds {
				t.Fatalf("canonical bounds = %v, want %v", loaded.bounds, tt.wantBounds)
			}
			if (loaded.vector != nil) != tt.wantVector {
				t.Fatalf("vector present = %v, want %v", loaded.vector != nil, tt.wantVector)
			}
			if tt.check != nil {
				tt.check(t, loaded.pixels)
			}

			result, err := Generate(context.Background(), mustRequest(t, []fyne.URI{tt.source}, image.Pt(61, 37), settings, 8))
			if err != nil {
				t.Fatal(err)
			}
			if result.Bounds() != image.Rect(0, 0, 61, 37) {
				t.Fatalf("generated bounds = %v", result.Bounds())
			}
		})
	}
}

func TestGenerate_FrameStyles(t *testing.T) {
	source := mosaicPNG(t, "black.png", 10, 10, func(_, _ int) color.NRGBA {
		return color.NRGBA{A: 255}
	})
	want := map[FrameStyle]string{
		FrameNone:      "636c3260bfc04cd76f8a2fadfe929919cd4d31be1f73fc60f40ffc5d4217676d",
		FrameThinLight: "417f9e5d48bf016ea21c38bc98c4f360e03ff545eda4bef2388cea3efd492e47",
		FrameThinDark:  "35d228e2e51658c8f4826d1c1aa4761a5dd76e06c5dc11d26c53bff9884e7ce2",
		FramePolaroid:  "7988064d06d63d8d3345b7c7bf8ffbccfed7e047772b384a1937c31f35440d81",
	}
	seen := make(map[[32]byte]FrameStyle)
	for _, frame := range []FrameStyle{FrameNone, FrameThinLight, FrameThinDark, FramePolaroid} {
		settings := DefaultSettings()
		settings.Frame = frame
		request := mustRequest(t, []fyne.URI{source}, image.Pt(64, 40), settings, 4)
		result, err := Generate(context.Background(), request)
		if err != nil {
			t.Fatalf("Generate(frame=%s) = %v", frame, err)
		}
		hash := sha256.Sum256(result.Image().(*image.NRGBA).Pix)
		if fmt.Sprintf("%x", hash) != want[frame] {
			t.Errorf("frame %s hash = %x", frame, hash)
		}
		if previous, ok := seen[hash]; ok {
			t.Fatalf("frames %s and %s rendered identically", previous, frame)
		}
		seen[hash] = frame
	}
}

func TestGenerate_UnreadableSources(t *testing.T) {
	missing := storage.NewFileURI(filepath.Join(t.TempDir(), "missing.png"))
	valid := mosaicPNG(t, "valid.png", 4, 4, func(_, _ int) color.NRGBA {
		return color.NRGBA{G: 255, A: 255}
	})

	if _, err := Generate(context.Background(), mustRequest(t, []fyne.URI{missing, valid}, image.Pt(30, 20), DefaultSettings(), 1)); err != nil {
		t.Fatalf("broken source followed by readable source failed: %v", err)
	}

	_, err := Generate(context.Background(), mustRequest(t, []fyne.URI{missing}, image.Pt(30, 20), DefaultSettings(), 1))
	var unreadable *NoReadableSourcesError
	if !errors.As(err, &unreadable) || len(unreadable.Attempts) != 1 {
		t.Fatalf("all-broken Generate() = %v, want one-attempt NoReadableSourcesError", err)
	}
}

func TestGenerate_LazyPool(t *testing.T) {
	sources := make([]fyne.URI, 10_000)
	for i := range sources {
		sources[i] = storage.NewFileURI(filepath.Join("/virtual", string(rune(i+1))+".png"))
	}
	request := mustRequest(t, sources, image.Pt(120, 80), DefaultSettings(), 7)
	loads := 0
	generator := New()
	generator.load = func(context.Context, fyne.URI) (*loadedSource, error) {
		loads++
		pixels := image.NewNRGBA(image.Rect(0, 0, 8, 6))
		for i := 3; i < len(pixels.Pix); i += 4 {
			pixels.Pix[i] = 255
		}
		return &loadedSource{pixels: pixels, bounds: pixels.Bounds()}, nil
	}

	if _, err := generator.Generate(context.Background(), request); err != nil {
		t.Fatal(err)
	}
	if loads == 0 || loads >= len(sources) {
		t.Fatalf("loader calls = %d for %d sources, want a lazy prefix", loads, len(sources))
	}
}

func TestGenerate_SourcesUnchanged(t *testing.T) {
	source := mosaicPNG(t, "immutable.png", 8, 5, func(x, y int) color.NRGBA {
		return color.NRGBA{R: uint8(x * 20), B: uint8(y * 30), A: 255}
	})
	before, err := os.ReadFile(source.Path())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(context.Background(), mustRequest(t, []fyne.URI{source}, image.Pt(50, 30), DefaultSettings(), 3)); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(source.Path())
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(before) != sha256.Sum256(after) {
		t.Fatal("source bytes changed during generation")
	}
}

func mosaicPNG(t *testing.T, name string, width, height int, pixel func(int, int) color.NRGBA) fyne.URI {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	image := image.NewNRGBA(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			image.SetNRGBA(x, y, pixel(x, y))
		}
	}
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(file, image); err != nil {
		_ = file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}

	return storage.NewFileURI(path)
}

func mustRequest(t *testing.T, sources []fyne.URI, target image.Point, settings Settings, seed int64) Request {
	t.Helper()
	request, err := NewRequest(sources, target, settings, seed)
	if err != nil {
		t.Fatal(err)
	}

	return request
}
