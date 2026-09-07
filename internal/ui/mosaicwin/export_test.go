package mosaicwin

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
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
		{name: "default PNG missing extension", format: ExportPNG, pickedName: "chosen", wantName: "chosen"},
		{name: "JPEG missing extension", format: ExportJPEG, pickedName: "chosen", wantName: "chosen"},
		{name: "typed JPEG overrides PNG", format: ExportPNG, pickedName: "chosen.jpeg", wantName: "chosen.jpeg"},
		{name: "typed PNG overrides JPEG", format: ExportJPEG, pickedName: "chosen.png", wantName: "chosen.png"},
		{name: "unsupported suffix preserves exact name", format: ExportPNG, pickedName: "chosen.webp", wantName: "chosen.webp"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := generatedWindow(t)
			w.SetExportFormat(tt.format)
			w.SetClock(func() time.Time { return time.Date(2026, 9, 4, 15, 6, 7, 0, time.Local) })
			result, _ := w.Result()
			var suggested string
			destination := filepath.Join(t.TempDir(), tt.pickedName)
			uitest.StubSaveChooser(t, func(path string) (fyne.URI, error) {
				suggested = path
				return storage.NewFileURI(destination), nil
			})
			var gotDest fyne.URI
			var gotImage image.Image
			var gotSource fyne.URI
			w.SetExporter(func(_ context.Context, dest fyne.URI, pixels image.Image, source fyne.URI, _ imaging.ExportOptions) (imaging.WriteResult, error) {
				gotDest, gotImage, gotSource = dest, pixels, source
				return imaging.WriteResult{}, nil
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
		pickerOut fyne.URI
		pickerErr error
		exportErr error
		wantError bool
	}{
		{name: "cancel"},
		{name: "picker failure", pickerErr: errors.New("picker failed"), wantError: true},
		{name: "write failure", pickerOut: storage.NewFileURI("/tmp/mosaic.png"), exportErr: errors.New("disk full"), wantError: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := generatedWindow(t)
			before, _ := w.Result()
			uitest.StubSaveChooser(t, func(_ string) (fyne.URI, error) { return tt.pickerOut, tt.pickerErr })
			exports := 0
			w.SetExporter(func(_ context.Context, _ fyne.URI, _ image.Image, _ fyne.URI, _ imaging.ExportOptions) (imaging.WriteResult, error) {
				exports++
				return imaging.WriteResult{}, tt.exportErr
			})

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
	uitest.StubSaveChooser(t, func(path string) (fyne.URI, error) {
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
			uitest.StubSaveChooser(t, func(_ string) (fyne.URI, error) { return storage.NewFileURI(picked), nil })

			w.SaveImage()
			settleWindow(t, w)
			path := picked
			file, err := os.Open(path)
			if err != nil {
				t.Fatal(err)
			}
			decoded, decodedFormat, decodeErr := image.Decode(file)
			_ = file.Close()
			if decodeErr != nil {
				t.Fatal(decodeErr)
			}
			wantFormat := "png"
			if format == ExportJPEG {
				wantFormat = "jpeg"
			}
			if decodedFormat != wantFormat {
				t.Fatalf("encoded format = %q, want %q", decodedFormat, wantFormat)
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
	uitest.StubSaveChooser(t, func(_ string) (fyne.URI, error) {
		return storage.NewFileURI(filepath.Join(t.TempDir(), "mosaic.png")), nil
	})
	w.SetExporter(func(_ context.Context, _ fyne.URI, _ image.Image, _ fyne.URI, _ imaging.ExportOptions) (imaging.WriteResult, error) {
		close(started)
		<-release
		return imaging.WriteResult{}, nil
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

func TestMosaicExport_CancellationReportsOnlyCommittedDiskChanges(t *testing.T) {
	for _, committed := range []bool{false, true} {
		t.Run(fmt.Sprintf("committed=%v", committed), func(t *testing.T) {
			w := generatedWindow(t)
			dest := storage.NewFileURI(uitest.WriteTempFile(t, "existing.png", uitest.EncodePNG(t, 3, 7, color.White)))
			before, err := os.ReadFile(dest.Path())
			if err != nil {
				t.Fatal(err)
			}
			uitest.StubSaveChooser(t, func(_ string) (fyne.URI, error) { return dest, nil })
			entered, release := make(chan struct{}), make(chan struct{})
			w.SetExporter(func(ctx context.Context, u fyne.URI, pixels image.Image, source fyne.URI, opts imaging.ExportOptions) (imaging.WriteResult, error) {
				var result imaging.WriteResult
				var err error
				if committed {
					result, err = imaging.ExportContext(ctx, u, pixels, source, opts)
				}
				close(entered)
				<-release
				if !committed {
					result, err = imaging.ExportContext(ctx, u, pixels, source, opts)
				}
				return result, err
			})
			w.SaveImage()
			<-entered
			w.Close()
			w.Show(mustSnapshot(t))
			close(release)
			settleWindow(t, w)
			after, err := os.ReadFile(dest.Path())
			if err != nil {
				t.Fatal(err)
			}
			if bytes.Equal(before, after) == committed {
				t.Errorf("disk mutation does not match committed=%v", committed)
			}
			results := w.host.(*fakeHost).exported
			want := 0
			if committed {
				want = 1
			}
			if len(results) != want {
				t.Errorf("host notifications=%d, want %d", len(results), want)
			}
			resolved, err := filepath.EvalSymlinks(dest.Path())
			if err != nil {
				t.Fatal(err)
			}
			if len(results) == 1 && (results[0].Path != resolved || !results[0].Committed) {
				t.Errorf("wrong committed identity: %+v", results[0])
			}
			if w.Status() != "" || w.Busy() {
				t.Error("obsolete export changed reopened window")
			}
			w.Close()
		})
	}
}
