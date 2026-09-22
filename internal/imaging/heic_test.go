package imaging

import (
	"context"
	"encoding/hex"
	"errors"
	"image"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/heic"
)

type heicBackend struct {
	read func(context.Context, []byte, heic.Request) (heic.Result, error)
}

func TestHEICNativeQualification(t *testing.T) {
	if os.Getenv("PICFETCH_HEIC_NATIVE_TEST") != "1" {
		t.Skip("requires explicit native provider qualification")
	}
	client := heic.NewClient("")
	t.Cleanup(func() { client.Stop(); client.Wait() })
	if err := client.Check(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx := heic.WithSnapshot(context.Background(), heic.Snapshot{Backend: client, Available: true, Generation: 1})
	for _, tc := range []struct {
		name    string
		samples [4]byte
	}{
		{"probe8", [4]byte{0, 255, 0, 255}},
		{"probe10", [4]byte{0, 255, 0, 255}},
		{"container-rotate", [4]byte{255, 255, 0, 0}},
		{"exif-rotate", [4]byte{0, 0, 255, 255}},
		{"container-and-exif", [4]byte{255, 255, 0, 0}},
		{"nonfirst-primary", [4]byte{255, 0, 255, 0}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			data, err := os.ReadFile("../heic/testdata/" + tc.name + ".heic")
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "renamed.jpg")
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			encoded, bounds, err := ReadAndProbe(ctx, storage.NewFileURI(path))
			if err != nil {
				t.Fatal(err)
			}
			loaded, err := DecodeLoaded(ctx, encoded, 0)
			if err != nil {
				t.Fatal(err)
			}
			if bounds != image.Rect(0, 0, 64, 64) || len(loaded.Frames) != 1 || loaded.Frames[0].Bounds() != bounds {
				t.Fatalf("full canonical still: bounds=%v loaded=%+v", bounds, loaded)
			}
			pixels, ok := loaded.Frames[0].(*image.NRGBA)
			if !ok {
				t.Fatal("native output is not straight RGBA")
			}
			for index, point := range []image.Point{{8, 8}, {48, 8}, {8, 48}, {48, 48}} {
				pixel := pixels.NRGBAAt(point.X, point.Y)
				for _, channel := range []byte{pixel.R, pixel.G, pixel.B} {
					if delta := int(channel) - int(tc.samples[index]); delta < -2 || delta > 2 {
						t.Fatalf("pixel %v=%v, want gray %d", point, pixel, tc.samples[index])
					}
				}
				if pixel.A != 255 {
					t.Fatalf("opaque source lost alpha: %v", pixel)
				}
			}
		})
	}
}

func (b heicBackend) Check(_ context.Context) error { return nil }
func (b heicBackend) Read(ctx context.Context, data []byte, request heic.Request) (heic.Result, error) {
	return b.read(ctx, data, request)
}

func TestHEICDecoderContract(t *testing.T) {
	t.Run("metadata comes from the captured native primary item", func(t *testing.T) {
		data, err := os.ReadFile("testdata/test_exif.heic")
		if err != nil {
			t.Fatal(err)
		}
		exif, err := hex.DecodeString("49492a000800000001000f010200060000001a0000000000000043616e6f6e00")
		if err != nil {
			t.Fatal(err)
		}
		ctx := heic.WithSnapshot(context.Background(), heic.Snapshot{Available: true, Backend: heicBackend{read: func(_ context.Context, _ []byte, request heic.Request) (heic.Result, error) {
			if request.Pixels {
				t.Fatal("metadata requested pixels")
			}
			return heic.Result{Width: 2, Height: 1, EXIF: exif}, nil
		}}})
		metadata, err := ReadMetadataContext(ctx, data)
		if err != nil || metadata.Make != "Canon" {
			t.Fatalf("metadata = %+v, %v", metadata, err)
		}
	})
	t.Run("shared probe and decode use captured backend with canonical pixels", func(t *testing.T) {
		backend := heicBackend{read: func(_ context.Context, data []byte, request heic.Request) (heic.Result, error) {
			if len(data) != 1130 || request.MaxPixels != maxImagePixels || request.MaxEncodedBytes != MaxEncodedBytes() {
				t.Fatalf("source/limits not preserved: bytes=%d request=%+v", len(data), request)
			}
			result := heic.Result{Width: 2, Height: 1, Provider: "test system boundary"}
			if request.Pixels {
				result.Stride = 8
				result.Pixels = []byte{255, 0, 0, 255, 0, 255, 0, 128}
			}
			return result, nil
		}}
		ctx := heic.WithSnapshot(context.Background(), heic.Snapshot{Backend: backend, Available: true, Generation: 1})
		data, bounds, err := ReadAndProbe(ctx, storage.NewFileURI("testdata/test_exif.heic"))
		if err != nil {
			t.Fatal(err)
		}
		if bounds != image.Rect(0, 0, 2, 1) {
			t.Fatalf("oriented bounds = %v", bounds)
		}
		loaded, err := DecodeLoaded(ctx, data, 0)
		if err != nil {
			t.Fatal(err)
		}
		if len(loaded.Frames) != 1 || loaded.Frames[0].Bounds() != bounds {
			t.Fatalf("canonical frame = %+v", loaded)
		}
		pixels, ok := loaded.Frames[0].(*image.NRGBA)
		if !ok || pixels.NRGBAAt(0, 0).R != 255 || pixels.NRGBAAt(1, 0).G != 255 || pixels.NRGBAAt(1, 0).A != 128 {
			t.Fatalf("canonical color/alpha was changed: %v", loaded.Frames[0])
		}
	})
	t.Run("no captured capability refuses actual HEIC bytes", func(t *testing.T) {
		data, err := os.ReadFile("testdata/test_exif.heic")
		if err != nil {
			t.Fatal(err)
		}
		if _, err := DecodeLoaded(context.Background(), data, 0); !errors.Is(err, heic.ErrUnavailable) {
			t.Fatalf("unavailable decode = %v", err)
		}
	})
}
