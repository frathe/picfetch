package imaging

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"testing"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/heicdecode"
	"github.com/frathe/picfetch/internal/heicdecode/client"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestSourceHEICAdmissionAndPixels(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "test_exif.heic"))
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"photo.heic", "photo.heif", "renamed.jpg"} {
		t.Run(name, func(t *testing.T) {
			var read, closed atomic.Int64
			stream := bytes.NewReader(data)
			u := uitest.ReaderURI(storage.NewFileURI(name), func() (io.ReadCloser, error) {
				return uitest.ReadCloser{ReadFunc: func(p []byte) (int, error) {
					n, err := stream.Read(p)
					read.Add(int64(n))
					return n, err
				}, CloseFunc: func() error { closed.Add(1); return nil }}, nil
			})
			pixels := image.NewNRGBA64(image.Rect(0, 0, 2, 3))
			pixels.SetNRGBA64(1, 2, color.NRGBA64{R: 0x1234, G: 0xabcd, B: 0x5678, A: 0x4321})
			calls := 0
			r := NewReader(func(ctx context.Context, op heicdecode.Operation, input client.Input) (heicdecode.Response, error) {
				calls++
				if op != heicdecode.Decode {
					t.Fatalf("operation = %v", op)
				}
				if name != "renamed.jpg" && read.Load() != 0 {
					t.Fatal("extension source read before admission")
				}
				if read.Load() >= int64(len(data)) {
					t.Fatal("full source read before admission")
				}
				got, err := input(ctx, int64(len(data)))
				if err != nil || !bytes.Equal(got, data) {
					t.Fatalf("input = %d bytes, %v", len(got), err)
				}
				return heicdecode.Response{Image: pixels, Config: image.Config{Width: 2, Height: 3, ColorModel: color.NRGBA64Model}, Metadata: &heicdecode.Metadata{Orientation: 6, Make: "Camera", DateTimeOriginal: "2026:09:16 12:34:56"}}, nil
			})
			source, err := r.Read(context.Background(), u)
			if err != nil {
				t.Fatal(err)
			}
			if source.Bounds() != pixels.Bounds() || source.SHA256() != sha256.Sum256(data) {
				t.Fatal("source bounds/digest changed")
			}
			for range 2 {
				loaded, err := source.Decode(context.Background(), 0)
				if err != nil {
					t.Fatal(err)
				}
				if len(loaded.Frames) != 1 || loaded.Frames[0] != pixels || loaded.FileSize != int64(len(data)) || !loaded.HasEXIF {
					t.Fatalf("source record = %+v", loaded)
				}
			}
			if source.Metadata().Make != "Camera" || calls != 1 || read.Load() != int64(len(data)) || closed.Load() != 1 {
				t.Fatal("source decoded/read twice or lost metadata/closure")
			}
		})
	}
}

func TestSourceMetadataUsesIsolatedOperation(t *testing.T) {
	data := []byte("owned source callback bytes")
	path := writeTempFile(t, "photo.heic", data)
	r := NewReader(func(ctx context.Context, op heicdecode.Operation, input client.Input) (heicdecode.Response, error) {
		if op != heicdecode.DecodeExif {
			t.Fatalf("metadata operation = %v", op)
		}
		if _, err := input(ctx, int64(len(data))); err != nil {
			return heicdecode.Response{}, err
		}
		return heicdecode.Response{Metadata: &heicdecode.Metadata{Make: "Owned", GPSLatitude: 50.8, GPSLongitude: 4.3}}, nil
	})
	m, err := r.Metadata(context.Background(), storage.NewFileURI(path))
	if err != nil || m.Make != "Owned" || !m.HasGPS || m.Latitude != 50.8 {
		t.Fatalf("metadata = %+v, %v", m, err)
	}
}

func TestSourceCancellationClosesAdmittedInput(t *testing.T) {
	started, stopped := make(chan struct{}), make(chan struct{})
	var once sync.Once
	u := uitest.ReaderURI(storage.NewFileURI("owned.heic"), func() (io.ReadCloser, error) {
		return uitest.ReadCloser{ReadFunc: func(_ []byte) (int, error) { close(started); <-stopped; return 0, io.ErrClosedPipe }, CloseFunc: func() error { once.Do(func() { close(stopped) }); return nil }}, nil
	})
	r := NewReader(func(ctx context.Context, _ heicdecode.Operation, input client.Input) (heicdecode.Response, error) {
		_, err := input(ctx, 128)
		return heicdecode.Response{}, err
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan error, 1)
	go func() { _, err := r.Read(ctx, u); done <- err }()
	select {
	case err := <-done:
		t.Fatalf("reader returned before admission: %v", err)
	case <-started:
	}
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled read = %v", err)
	}
}

func TestSourceWithoutOwnerRejectsHEIC(t *testing.T) {
	path := writeTempFile(t, "photo.heic", []byte("owned bytes"))
	if _, err := (Reader{}).Read(context.Background(), storage.NewFileURI(path)); !errors.Is(err, client.ErrUnavailable) {
		t.Fatalf("unconfigured source = %v", err)
	}
}

func TestSourceOrdinaryOrientationAndFacts(t *testing.T) {
	data := halfRedHalfBlueJPEG(t, 20, 10, 6)
	path := writeTempFile(t, "photo.jpg", data)
	source, err := (Reader{}).Read(context.Background(), storage.NewFileURI(path))
	if err != nil {
		t.Fatal(err)
	}
	wantData, wantBounds, err := ReadAndProbe(context.Background(), storage.NewFileURI(path))
	if err != nil {
		t.Fatal(err)
	}
	want, err := DecodeRecord(context.Background(), wantData, 0)
	if err != nil {
		t.Fatal(err)
	}
	got, err := source.Decode(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}
	if source.Bounds() != wantBounds || got.Frames[0].Bounds() != want.Frames[0].Bounds() || got.FileSize != want.FileSize || got.HasEXIF != want.HasEXIF || source.SHA256() != sha256.Sum256(data) {
		t.Fatal("ordinary source facts changed")
	}
}

func TestSourceDispatchTypes(t *testing.T) {
	for _, c := range []struct {
		name   string
		brands []string
		want   sourceKind
	}{
		{"heic", []string{"heic", "mif1"}, sourceIsolated},
		{"generic HEIF", []string{"mif1"}, sourceIsolated},
		{"avif", []string{"avif", "mif1"}, sourceAVIF},
		{"cr3", []string{"crx "}, sourceCR3},
		{"mixed", []string{"avif", "heic"}, sourceIsolated},
	} {
		t.Run(c.name, func(t *testing.T) {
			data := make([]byte, 12+4*len(c.brands))
			binary.BigEndian.PutUint32(data[:4], uint32(len(data)))
			copy(data[4:8], "ftyp")
			copy(data[8:12], c.brands[0])
			for i, brand := range c.brands[1:] {
				copy(data[16+4*i:], brand)
			}
			if got := routeSource(data); got != c.want {
				t.Fatalf("route = %v, want %v", got, c.want)
			}
		})
	}
}

func TestByteMetadataAndPreviewRefuseHEIC(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "test_exif.heic"))
	if err != nil {
		t.Fatal(err)
	}
	if !ReadMetadata(data).Empty() {
		t.Fatal("byte metadata reached native HEIC container parsing")
	}
	if looksLikePreviewContainer(data) {
		t.Fatal("HEIC admitted to native embedded-preview scan")
	}
}

func TestSourceDerivedViewsAndCaptureDate(t *testing.T) {
	path := writeTempFile(t, "photo.heic", []byte("owned read bytes"))
	u := storage.NewFileURI(path)
	var operations []heicdecode.Operation
	r := NewReader(func(ctx context.Context, op heicdecode.Operation, input client.Input) (heicdecode.Response, error) {
		operations = append(operations, op)
		if _, err := input(ctx, 128); err != nil {
			return heicdecode.Response{}, err
		}
		return heicdecode.Response{Image: image.NewNRGBA64(image.Rect(0, 0, 40, 20)), Config: image.Config{Width: 40, Height: 20}, Metadata: &heicdecode.Metadata{DateTimeOriginal: "2026:09:16 12:34:56"}}, nil
	})
	thumb, bounds, err := r.Thumbnail(context.Background(), u, 10)
	if err != nil || thumb.Bounds().Size() != image.Pt(10, 5) || bounds.Size() != image.Pt(40, 20) {
		t.Fatalf("thumbnail = %v, %v", bounds, err)
	}
	preview, err := r.Preview(context.Background(), u, 20, 1024)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Frames) != 1 || preview.Frames[0].Bounds().Size() != image.Pt(20, 10) || len(preview.Delays) != 0 {
		t.Fatal("HEIC preview changed geometry or became animated")
	}
	date, ok, err := r.CaptureDate(context.Background(), u)
	if err != nil || !ok || date.Format("2006-01-02") != "2026-09-16" {
		t.Fatalf("capture date = %v, %v, %v", date, ok, err)
	}
	if len(operations) != 3 || operations[0] != heicdecode.Decode || operations[1] != heicdecode.Decode || operations[2] != heicdecode.DecodeExif {
		t.Fatalf("operations = %v", operations)
	}
}

func TestSourceAdmittedCancellationClosesInput(t *testing.T) {
	started, stopped := make(chan struct{}), make(chan struct{})
	var once sync.Once
	u := uitest.ReaderURI(storage.NewFileURI("owned.heic"), func() (io.ReadCloser, error) {
		return uitest.ReadCloser{ReadFunc: func(_ []byte) (int, error) { close(started); <-stopped; return 0, io.ErrClosedPipe }, CloseFunc: func() error { once.Do(func() { close(stopped) }); return nil }}, nil
	})
	admitted, cancel := context.WithCancel(context.Background())
	defer cancel()
	r := NewReader(func(_ context.Context, _ heicdecode.Operation, input client.Input) (heicdecode.Response, error) {
		_, err := input(admitted, 128)
		return heicdecode.Response{}, err
	})
	done := make(chan error, 1)
	go func() { _, err := r.Read(context.Background(), u); done <- err }()
	<-started
	cancel()
	if err := <-done; !errors.Is(err, context.Canceled) {
		t.Fatalf("admitted cancellation = %v", err)
	}
}
