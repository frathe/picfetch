package mosaicwin

import (
	"errors"
	"image"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestMosaicExport_UsesTimestampDirectoryFormatAndExactPixels(t *testing.T) {
	tests := []struct {
		name       string
		format     ExportFormat
		pickedName string
		wantName   string
	}{
		{name: "default PNG missing extension", format: ExportPNG, pickedName: "chosen", wantName: "chosen.png"},
		{name: "JPEG missing extension", format: ExportJPEG, pickedName: "chosen", wantName: "chosen.jpg"},
		{name: "typed JPEG overrides PNG", format: ExportPNG, pickedName: "chosen.jpeg", wantName: "chosen.jpeg"},
		{name: "typed PNG overrides JPEG", format: ExportJPEG, pickedName: "chosen.png", wantName: "chosen.png"},
		{name: "unsupported suffix appends choice", format: ExportPNG, pickedName: "chosen.webp", wantName: "chosen.webp.png"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := generatedWindow(t)
			w.SetExportFormat(tt.format)
			w.SetClock(func() time.Time { return time.Date(2026, 9, 4, 15, 6, 7, 0, time.Local) })
			result, _ := w.Result()
			var suggested string
			destination := filepath.Join(t.TempDir(), tt.pickedName)
			uitest.StubSaveChooser(t, func(path string) ([]byte, error) {
				suggested = path
				return []byte(destination + "\n"), nil
			})
			var gotDest fyne.URI
			var gotImage image.Image
			var gotSource fyne.URI
			w.SetExporter(func(dest fyne.URI, pixels image.Image, source fyne.URI, _ imaging.ExportOptions) error {
				gotDest, gotImage, gotSource = dest, pixels, source
				return nil
			})

			w.SaveImage()
			settleWindow(t, w)

			firstSourceDir := filepath.Dir(w.Snapshot().Sources[0].Path())
			wantSuggestion := filepath.Join(firstSourceDir, "PicFetch-Mosaic-20260904-150607"+string(tt.format))
			if suggested != wantSuggestion {
				t.Fatalf("suggested path = %q, want %q", suggested, wantSuggestion)
			}
			if gotDest == nil || gotDest.Name() != tt.wantName {
				t.Fatalf("destination = %v, want %q", gotDest, tt.wantName)
			}
			if gotSource != nil {
				t.Fatalf("metadata source = %v, want nil", gotSource)
			}
			if !sameImage(gotImage, result.Image()) {
				t.Fatal("export did not receive the current result pixels")
			}
			if !w.PreviewActionsEnabled() {
				t.Fatal("successful export did not re-enable preview actions")
			}
			w.Close()
		})
	}
}

func TestMosaicExport_CancelAndFailuresKeepPreview(t *testing.T) {
	tests := []struct {
		name      string
		pickerOut []byte
		pickerErr error
		exportErr error
		wantError bool
	}{
		{name: "cancel"},
		{name: "picker failure", pickerErr: errors.New("picker failed"), wantError: true},
		{name: "write failure", pickerOut: []byte("/tmp/mosaic.png\n"), exportErr: errors.New("disk full"), wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := generatedWindow(t)
			before, _ := w.Result()
			uitest.StubSaveChooser(t, func(string) ([]byte, error) { return tt.pickerOut, tt.pickerErr })
			exports := 0
			w.SetExporter(func(fyne.URI, image.Image, fyne.URI, imaging.ExportOptions) error { exports++; return tt.exportErr })

			w.SaveImage()
			settleWindow(t, w)

			after, ok := w.Result()
			if !ok || !samePixels(before, after) || !w.PreviewActionsEnabled() {
				t.Fatal("cancel/failure replaced or disabled the preview")
			}
			if tt.pickerOut == nil && exports != 0 {
				t.Fatalf("cancel/picker failure called exporter %d times", exports)
			}
			if (w.Status() != "") != tt.wantError {
				t.Fatalf("status = %q, wantError=%v", w.Status(), tt.wantError)
			}
			w.Close()
		})
	}
}

func TestMosaicExport_FormatControlSelectsJPEG(t *testing.T) {
	w := generatedWindow(t)
	w.formatSelect.SetSelected("JPEG")
	if w.exportFormat != ExportJPEG {
		t.Fatalf("format selection = %q, want JPEG", w.exportFormat)
	}
	var suggested string
	uitest.StubSaveChooser(t, func(path string) ([]byte, error) {
		suggested = path
		return nil, nil
	})
	w.SaveImage()
	settleWindow(t, w)
	if filepath.Ext(suggested) != ".jpg" {
		t.Fatalf("JPEG suggested path = %q", suggested)
	}
	w.Close()
}

func TestMosaicExport_PNGAndJPEGDecodeAtExactTargetWithoutSourceMetadata(t *testing.T) {
	for _, format := range []ExportFormat{ExportPNG, ExportJPEG} {
		t.Run(string(format), func(t *testing.T) {
			w := New(test.NewApp(), successfulHost(t))
			w.SetUIQueue(&uitest.UIQueue{})
			snapshot, err := NewSnapshot(
				[]fyne.URI{uitest.TempGPSJPEGURI(t, "source.jpg", 10, 8, 48.8, 2.3)},
				SourceResult,
				testTopology("one", 80, 50),
			)
			if err != nil {
				t.Fatal(err)
			}
			w.Show(snapshot)
			w.Generate()
			settleWindow(t, w)
			w.SetExportFormat(format)
			picked := filepath.Join(t.TempDir(), "mosaic")
			uitest.StubSaveChooser(t, func(string) ([]byte, error) { return []byte(picked + "\n"), nil })

			w.SaveImage()
			settleWindow(t, w)
			path := picked + string(format)
			file, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			decoded, _, decodeErr := image.Decode(file)
			_ = file.Close()
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			if decoded.Bounds() != image.Rect(0, 0, 80, 50) {
				t.Fatalf("export bounds = %v", decoded.Bounds())
			}
			if format == ExportPNG {
				result, _ := w.Result()
				if !sameImage(decoded, result.Image()) {
					t.Fatal("PNG export differs from the exact retained result")
				}
			} else {
				result, _ := w.Result()
				// JPEG is intentionally lossy. Quality 95 should keep the mean
				// per-channel error well below this visible-difference guard.
				const maxMeanRGBDelta = 16.0
				if delta := meanRGBDelta(decoded, result.Image()); delta > maxMeanRGBDelta {
					t.Fatalf("JPEG mean RGB delta = %.2f, want <= %.2f", delta, maxMeanRGBDelta)
				}
				data, readErr := os.ReadFile(path)
				if readErr != nil {
					t.Fatal(readErr)
				}
				if metadata := imaging.ReadMetadata(data); metadata.HasGPS {
					t.Fatal("JPEG mosaic copied GPS metadata from its source")
				}
			}
			w.Close()
		})
	}
}

func TestMosaicExport_CloseAndReopenRejectsLateCompletion(t *testing.T) {
	w := generatedWindow(t)
	started := make(chan struct{})
	release := make(chan struct{})
	uitest.StubSaveChooser(t, func(string) ([]byte, error) {
		return []byte(filepath.Join(t.TempDir(), "mosaic.png") + "\n"), nil
	})
	w.SetExporter(func(fyne.URI, image.Image, fyne.URI, imaging.ExportOptions) error {
		close(started)
		<-release
		return nil
	})

	w.SaveImage()
	<-started
	w.Close()
	w.Show(mustSnapshot(t))
	close(release)
	settleWindow(t, w)

	if status := w.Status(); status != "" {
		t.Fatalf("late export completion mutated reopened window status: %q", status)
	}
	if w.Busy() {
		t.Fatal("late export completion mutated reopened window busy state")
	}
	w.Close()
}

func generatedWindow(t *testing.T) *Window {
	t.Helper()
	w := New(test.NewApp(), successfulHost(t))
	w.SetUIQueue(&uitest.UIQueue{})
	w.Show(mustSnapshot(t))
	w.Generate()
	settleWindow(t, w)
	return w
}

func sameImage(a, b image.Image) bool {
	if a == nil || b == nil || a.Bounds() != b.Bounds() {
		return false
	}
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			if a.At(x, y) != b.At(x, y) {
				ar, ag, ab, aa := a.At(x, y).RGBA()
				br, bg, bb, ba := b.At(x, y).RGBA()
				if ar != br || ag != bg || ab != bb || aa != ba {
					return false
				}
			}
		}
	}
	return true
}

func meanRGBDelta(a, b image.Image) float64 {
	if a == nil || b == nil || a.Bounds() != b.Bounds() {
		return 255
	}
	var total uint64
	for y := a.Bounds().Min.Y; y < a.Bounds().Max.Y; y++ {
		for x := a.Bounds().Min.X; x < a.Bounds().Max.X; x++ {
			ar, ag, ab, _ := a.At(x, y).RGBA()
			br, bg, bb, _ := b.At(x, y).RGBA()
			total += channelDelta(ar, br) + channelDelta(ag, bg) + channelDelta(ab, bb)
		}
	}
	channels := uint64(a.Bounds().Dx() * a.Bounds().Dy() * 3)
	return float64(total) / float64(channels) / 257
}

func channelDelta(a, b uint32) uint64 {
	if a >= b {
		return uint64(a - b)
	}
	return uint64(b - a)
}
