package ui

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
	"github.com/frathe/picfetch/internal/uitest"
)

func TestHEICNativeImageOperations(t *testing.T) {
	if os.Getenv("PICFETCH_HEIC_NATIVE_TEST") != "1" {
		t.Skip("requires explicit installed-provider consumer qualification")
	}
	black, white := color.NRGBA{A: 255}, color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	for _, fixture := range []struct {
		name   string
		colors []color.NRGBA
	}{
		{"probe8", []color.NRGBA{black, white, black, white}},
		{"probe10", []color.NRGBA{black, white, black, white}},
		{"container-and-exif", []color.NRGBA{white, white, black, black}},
		{"nonfirst-primary", []color.NRGBA{white, black, white, black}},
	} {
		t.Run(fixture.name, func(t *testing.T) {
			v := newTestViewer(t)
			client := heic.NewClient("")
			v.configureHEIC(client)
			v.heic.ui = &uitest.UIQueue{}
			v.startHEICCheck(true)
			v.settleHEIC()
			if state := v.heic.capability.State(); !state.Available {
				t.Fatalf("native provider did not qualify: %+v", state)
			}
			data, err := os.ReadFile("../heic/testdata/" + fixture.name + ".heic")
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(t.TempDir(), "写真.heic")
			if err := os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			dropAndWait(t, v, storage.NewFileURI(path))
			capture, release, ok := v.display.CaptureStable()
			if !ok {
				t.Fatal("native HEIC did not reach display capture")
			}
			release()
			assertHEICOperationPixels(t, capture.Pixels, image.Pt(64, 64), 2, fixture.colors, 2)
			var copied []byte
			uitest.StubClipboardCopy(t, func(data []byte) error { copied = bytes.Clone(data); return nil })
			v.copyImageToClipboard()
			waitForClipboard(t, v)
			clipboard, err := png.Decode(bytes.NewReader(copied))
			if err != nil {
				t.Fatal(err)
			}
			assertHEICOperationPixels(t, clipboard, image.Pt(64, 64), 2, fixture.colors, 2)
			for _, extension := range []string{".png", ".jpg"} {
				destination := filepath.Join(t.TempDir(), "copy"+extension)
				uitest.StubSaveChooser(t, func(_ string) (fyne.URI, error) { return storage.NewFileURI(destination), nil })
				v.exportAs(extension)
				settleChooser(t, v)
				exported, err := loadExported(t, destination)
				if err != nil {
					t.Fatal(err)
				}
				assertHEICOperationPixels(t, exported, image.Pt(64, 64), 2, fixture.colors, 2)
				settleToast(t, v)
			}
			request, err := mosaic.NewRequest([]fyne.URI{storage.NewFileURI(path)}, image.Pt(80, 60), mosaic.DefaultSettings(), 23)
			if err != nil {
				t.Fatal(err)
			}
			result, err := v.GenerateMosaic(t.Context(), request, nil)
			if err != nil {
				t.Fatal(err)
			}
			oracle := image.NewNRGBA(image.Rect(0, 0, 64, 64))
			for y := range 64 {
				for x := range 64 {
					oracle.SetNRGBA(x, y, fixture.colors[y/32*2+x/32])
				}
			}
			var oraclePNG bytes.Buffer
			if err := png.Encode(&oraclePNG, oracle); err != nil {
				t.Fatal(err)
			}
			oraclePath := uitest.WriteTempFile(t, "oracle.png", oraclePNG.Bytes())
			oracleRequest, err := mosaic.NewRequest([]fyne.URI{storage.NewFileURI(oraclePath)}, image.Pt(80, 60), mosaic.DefaultSettings(), 23)
			if err != nil {
				t.Fatal(err)
			}
			oracleResult, err := v.GenerateMosaic(t.Context(), oracleRequest, nil)
			if err != nil {
				t.Fatal(err)
			}
			for y := range 60 {
				for x := range 80 {
					got := color.NRGBAModel.Convert(result.Image().At(x, y)).(color.NRGBA)
					want := color.NRGBAModel.Convert(oracleResult.Image().At(x, y)).(color.NRGBA)
					for i, value := range []byte{got.R, got.G, got.B, got.A} {
						expected := []byte{want.R, want.G, want.B, want.A}[i]
						if delta := int(value) - int(expected); delta < -2 || delta > 2 {
							t.Fatalf("native mosaic pixel (%d,%d)=%v, oracle=%v", x, y, got, want)
						}
					}
				}
			}
			provider, err := client.Read(t.Context(), data, heic.Request{MaxEncodedBytes: 64 * 1024, MaxPixels: 4096})
			if err != nil {
				t.Fatal(err)
			}
			t.Logf("provider=%q identity=%+v", provider.Provider, heic.SystemIdentity())
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(after, data) {
				t.Fatalf("native consumer changed source bytes: %v", err)
			}
		})
	}
}

func TestHEICImageOperations(t *testing.T) {
	v := newTestViewer(t)
	const cell = 400
	original := []color.NRGBA{
		{R: 255, A: 255}, {G: 255, A: 255}, {B: 255, A: 255},
		{R: 255, G: 255, A: 255}, {R: 255, B: 255, A: 255}, {G: 255, B: 255, A: 255},
	}
	pixels := make([]byte, 2*cell*3*cell*4)
	for y := range 3 * cell {
		for x := range 2 * cell {
			pixel := original[y/cell*2+x/cell]
			copy(pixels[(y*2*cell+x)*4:], []byte{pixel.R, pixel.G, pixel.B, pixel.A})
		}
	}
	exif, err := hex.DecodeString("49492a0008000000010012010300010000000600000000000000")
	if err != nil {
		t.Fatal(err)
	}
	v.configureHEIC(testHEICBackend{
		check: func(_ context.Context) error { return nil },
		read: func(_ context.Context, _ []byte, request heic.Request) (heic.Result, error) {
			result := heic.Result{Width: 2 * cell, Height: 3 * cell, EXIF: exif}
			if request.Pixels {
				result.Stride, result.Pixels = 2*cell*4, bytes.Clone(pixels)
			}
			return result, nil
		},
	})
	v.heic.ui = &uitest.UIQueue{}
	v.startHEICCheck(false)
	v.settleHEIC()
	data, err := os.ReadFile("../heic/testdata/container-and-exif.heic")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "操作.heic")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	dropAndWait(t, v, storage.NewFileURI(path))
	capture, release, ok := v.display.CaptureStable()
	if !ok {
		t.Fatal("displayed HEIC was not available to stable capture")
	}
	release()
	assertHEICOperationPixels(t, capture.Pixels, image.Pt(2*cell, 3*cell), 2, original, 0)
	v.rotateBy(1)
	rotated := []color.NRGBA{original[4], original[2], original[0], original[5], original[3], original[1]}
	assertHEICOperationPixels(t, v.img.Image, image.Pt(3*cell, 2*cell), 3, rotated, 0)
	assertHEICOperationPixels(t, capture.Pixels, image.Pt(2*cell, 3*cell), 2, original, 0)
	if v.canSaveRotation() {
		t.Fatal("Save Changes became available for a HEIC source")
	}
	v.saveRotation()

	var copied []byte
	uitest.StubClipboardCopy(t, func(data []byte) error { copied = bytes.Clone(data); return nil })
	v.copyImageToClipboard()
	waitForClipboard(t, v)
	clipboard, err := png.Decode(bytes.NewReader(copied))
	if err != nil {
		t.Fatal(err)
	}
	assertHEICOperationPixels(t, clipboard, image.Pt(3*cell, 2*cell), 3, rotated, 0)
	v.win.Resize(fyne.NewSize(600, 450))
	v.zoom.ResetToFit()
	selectRegion(t, v, image.Rect(cell+100, 100, 3*cell-100, cell-100))
	v.regionCopy.HandleKey(fyne.KeyReturn)
	waitForClipboard(t, v)
	region, err := png.Decode(bytes.NewReader(copied))
	if err != nil {
		t.Fatal(err)
	}
	assertHEICOperationPixels(t, region, image.Pt(2*cell-200, cell-200), 2, []color.NRGBA{original[2], original[0]}, 0)

	for _, extension := range []string{".png", ".jpg"} {
		t.Run("export"+extension, func(t *testing.T) {
			destination := filepath.Join(t.TempDir(), "copy"+extension)
			uitest.StubSaveChooser(t, func(_ string) (fyne.URI, error) { return storage.NewFileURI(destination), nil })
			v.promptExport()
			v.exportOptions.setMetadataIncluded(false)
			wantSize, tolerance := image.Pt(3*cell, 2*cell), 0
			if extension == ".jpg" {
				v.exportOptions.selectRung(3)
				v.exportPrompt.Select(jpegChoice)
				wantSize, tolerance = image.Pt(1000, 666), 3
			} else {
				v.exportPrompt.Select(pngChoice)
			}
			v.exportPrompt.Confirm()
			settleChooser(t, v)
			exported, err := loadExported(t, destination)
			if err != nil {
				t.Fatal(err)
			}
			assertHEICOperationPixels(t, exported, wantSize, 3, rotated, tolerance)
			settleToast(t, v)
		})
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(after, data) {
		t.Fatalf("image operations changed the HEIC source: %v", err)
	}
	if v.display.Rotation() != 1 {
		t.Fatal("export discarded the pending view rotation")
	}
}

func assertHEICOperationPixels(t *testing.T, pixels image.Image, size image.Point, columns int, colors []color.NRGBA, tolerance int) {
	t.Helper()
	if pixels == nil {
		t.Fatal("operation returned no pixels")
	}
	if pixels.Bounds() != (image.Rectangle{Max: size}) {
		t.Fatalf("operation pixels have unexpected bounds: %v, want %v", pixels.Bounds(), size)
	}
	rows := len(colors) / columns
	for i, want := range colors {
		x, y := (i%columns*2+1)*size.X/(columns*2), (i/columns*2+1)*size.Y/(rows*2)
		got := color.NRGBAModel.Convert(pixels.At(x, y)).(color.NRGBA)
		for channel, value := range []byte{got.R, got.G, got.B, got.A} {
			expected := []byte{want.R, want.G, want.B, want.A}[channel]
			if delta := int(value) - int(expected); delta < -tolerance || delta > tolerance {
				t.Fatalf("operation pixel (%d,%d) = %v, want %v within %d", x, y, got, want, tolerance)
			}
		}
	}
}
