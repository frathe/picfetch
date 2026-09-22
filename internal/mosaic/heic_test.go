package mosaic_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/mosaic"
)

type mosaicHEICBackend struct{ exif []byte }

func (mosaicHEICBackend) Check(_ context.Context) error { return nil }
func (b mosaicHEICBackend) Read(_ context.Context, _ []byte, request heic.Request) (heic.Result, error) {
	result := heic.Result{Width: 2, Height: 3, EXIF: b.exif}
	if request.Pixels {
		result.Stride = 8
		result.Pixels = []byte{
			255, 0, 0, 255, 0, 255, 0, 255,
			0, 0, 255, 255, 255, 255, 0, 255,
			255, 0, 255, 255, 0, 255, 255, 255,
		}
	}
	return result, nil
}

func TestHEICMosaicSource(t *testing.T) {
	data, err := os.ReadFile("../heic/testdata/container-and-exif.heic")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	heicPath := filepath.Join(dir, "canonical.heic")
	if err := os.WriteFile(heicPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	oracle := image.NewNRGBA(image.Rect(0, 0, 2, 3))
	for i, pixel := range []color.NRGBA{
		{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255},
		{R: 255, G: 255, A: 255}, {R: 255, B: 255, A: 255}, {G: 255, B: 255, A: 255},
	} {
		oracle.SetNRGBA(i%2, i/2, pixel)
	}
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, oracle); err != nil {
		t.Fatal(err)
	}
	pngPath := filepath.Join(dir, "canonical.png")
	if err := os.WriteFile(pngPath, encoded.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	exif, err := hex.DecodeString("49492a0008000000010012010300010000000600000000000000")
	if err != nil {
		t.Fatal(err)
	}
	capability := heic.NewCapability(mosaicHEICBackend{exif: exif}, heic.SystemIdentity(), heic.Observation{})
	t.Cleanup(func() { capability.Stop(); capability.Wait() })
	<-capability.Check(t.Context())
	ctx := capability.CaptureContext(t.Context())
	capability.Invalidate(capability.State().Generation)
	for _, layout := range []mosaic.LayoutMode{mosaic.LayoutRandom, mosaic.LayoutShelf} {
		t.Run(string(layout), func(t *testing.T) {
			settings := mosaic.DefaultSettings()
			settings.Layout = layout
			var results []image.Image
			for _, path := range []string{heicPath, pngPath} {
				request, err := mosaic.NewRequest([]fyne.URI{storage.NewFileURI(path)}, image.Pt(80, 60), settings, 23)
				if err != nil {
					t.Fatal(err)
				}
				result, err := mosaic.Generate(ctx, request)
				if err != nil {
					t.Fatalf("mosaic from %s: %v", path, err)
				}
				results = append(results, result.Image())
			}
			for y := range 60 {
				for x := range 80 {
					got, want := color.NRGBAModel.Convert(results[0].At(x, y)), color.NRGBAModel.Convert(results[1].At(x, y))
					if got != want {
						t.Fatalf("canonical HEIC/PNG mosaic differs at (%d,%d): %v / %v", x, y, got, want)
					}
				}
			}
		})
	}
	after, err := os.ReadFile(heicPath)
	if err != nil || !bytes.Equal(after, data) {
		t.Fatalf("mosaic changed the HEIC source: %v", err)
	}
}
