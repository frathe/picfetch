package imaging_test

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"math"
	"os"
	"path/filepath"
	"testing"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

func fixtureTIFF(t *testing.T, lat, lon float64, date string) []byte {
	t.Helper()
	jpeg := uitest.GPSDateJPEG(t, 2, 1, lat, lon, date)
	if len(jpeg) < 12 || !bytes.Equal(jpeg[:4], []byte{0xff, 0xd8, 0xff, 0xe1}) {
		t.Fatal("GPS fixture has no first APP1 segment")
	}
	end := 4 + int(binary.BigEndian.Uint16(jpeg[4:6]))
	if end > len(jpeg) || !bytes.Equal(jpeg[6:12], []byte("Exif\x00\x00")) {
		t.Fatal("GPS fixture has no EXIF TIFF payload")
	}
	return jpeg[12:end]
}

func dateOnlyTIFF() []byte {
	tiff := []byte{'I', 'I', 42, 0, 8, 0, 0, 0}
	tiff = binary.LittleEndian.AppendUint16(tiff, 1)
	tiff = binary.LittleEndian.AppendUint16(tiff, 0x0132)
	tiff = binary.LittleEndian.AppendUint16(tiff, 2)
	tiff = binary.LittleEndian.AppendUint32(tiff, 20)
	tiff = binary.LittleEndian.AppendUint32(tiff, 26)
	tiff = binary.LittleEndian.AppendUint32(tiff, 0)
	return append(tiff, "2021:03:04 05:06:07\x00"...)
}

func pngWithEXIF(t *testing.T, payload []byte) []byte {
	t.Helper()
	png := uitest.EncodePNG(t, 2, 1, color.White)
	firstEnd := 8 + 12 + int(binary.BigEndian.Uint32(png[8:12]))
	if firstEnd > len(png) || !bytes.Equal(png[12:16], []byte("IHDR")) {
		t.Fatal("encoded PNG has no IHDR")
	}
	chunk := make([]byte, 0, len(payload)+12)
	chunk = binary.BigEndian.AppendUint32(chunk, uint32(len(payload)))
	chunk = append(chunk, "eXIf"...)
	chunk = append(chunk, payload...)
	chunk = binary.BigEndian.AppendUint32(chunk, crc32.ChecksumIEEE(chunk[4:]))
	return append(append(append([]byte(nil), png[:firstEnd]...), chunk...), png[firstEnd:]...)
}

func webpWithEXIF(t *testing.T, payload []byte) []byte {
	t.Helper()
	base, err := os.ReadFile("../../assets/ui-originals/placeholder.webp")
	if err != nil {
		t.Fatal(err)
	}
	if len(base) < 30 || !bytes.Equal(base[:4], []byte("RIFF")) || !bytes.Equal(base[8:16], []byte("WEBPVP8X")) {
		t.Fatal("WebP fixture has no RIFF/VP8X header")
	}
	data := append([]byte(nil), base...)
	data[20] |= 0x08 // VP8X EXIF feature flag.
	chunk := make([]byte, 0, len(payload)+9)
	chunk = append(chunk, "EXIF"...)
	chunk = binary.LittleEndian.AppendUint32(chunk, uint32(len(payload)))
	chunk = append(chunk, payload...)
	if len(payload)%2 != 0 {
		chunk = append(chunk, 0)
	}
	data = append(data, chunk...)
	binary.LittleEndian.PutUint32(data[4:8], uint32(len(data)-8))
	return data
}

func TestReadMetadataPNGEXIF(t *testing.T) {
	data := pngWithEXIF(t, fixtureTIFF(t, 48.858222, 2.2945, "2024:07:08 09:10:11"))
	if _, _, err := image.DecodeConfig(bytes.NewReader(data)); err != nil {
		t.Fatalf("authored PNG is invalid: %v", err)
	}
	m := imaging.ReadMetadata(data)
	if !m.HasGPS || math.Abs(m.Latitude-48.858222) > 0.000001 || math.Abs(m.Longitude-2.2945) > 0.000001 || m.DateTakenTime.Year() != 2024 {
		t.Fatalf("PNG metadata = %+v, want GPS and capture date", m)
	}
	contextMetadata, err := imaging.ReadMetadataContext(context.Background(), data)
	if err != nil || contextMetadata != m {
		t.Fatalf("PNG context metadata = %+v, %v; want %+v", contextMetadata, err, m)
	}
}

func TestReadMetadataTIFFGPS(t *testing.T) {
	for _, position := range [][2]float64{{0, 0}, {52.52, 13.405}} {
		data := uitest.GPSMetadataTIFF(t, position[0], position[1], "2024:07:08 09:10:11")
		source := storage.NewFileURI(uitest.WriteTempFile(t, "metadata.tiff", data))
		metadata, err := imaging.ReadMetadataURIContext(context.Background(), source)
		if err != nil || !metadata.HasGPS || math.Abs(metadata.Latitude-position[0]) > .000001 || math.Abs(metadata.Longitude-position[1]) > .000001 || metadata.DateTakenTime.Year() != 2024 {
			t.Fatalf("shared TIFF GPS contract: %+v, %v", metadata, err)
		}
	}
}

func TestReadMetadataWebPEXIF(t *testing.T) {
	data := webpWithEXIF(t, fixtureTIFF(t, -33.8568, 151.2153, "2023:06:05 12:13:14"))
	if _, _, err := image.DecodeConfig(bytes.NewReader(data)); err != nil {
		t.Fatalf("authored WebP is invalid: %v", err)
	}
	m := imaging.ReadMetadata(data)
	if !m.HasGPS || math.Abs(m.Latitude-(-33.8568)) > 0.000001 || math.Abs(m.Longitude-151.2153) > 0.000001 || m.DateTakenTime.Year() != 2023 {
		t.Fatalf("WebP metadata = %+v, want GPS and capture date", m)
	}
	contextMetadata, err := imaging.ReadMetadataContext(context.Background(), data)
	if err != nil || contextMetadata != m {
		t.Fatalf("WebP context metadata = %+v, %v; want %+v", contextMetadata, err, m)
	}
}

func TestReadMetadataURIContext(t *testing.T) {
	data := pngWithEXIF(t, fixtureTIFF(t, 0, 0, "2022:01:02 03:04:05"))
	path := uitest.WriteTempFile(t, "position.png", data)
	uri := storage.NewFileURI(path)
	m, err := imaging.ReadMetadataURIContext(context.Background(), uri)
	if err != nil || !m.HasGPS || m.Latitude != 0 || m.Longitude != 0 || m.DateTakenTime.Year() != 2022 {
		t.Fatalf("URI metadata = %+v, %v; want explicit zero GPS and date", m, err)
	}
	unlocated := storage.NewFileURI(uitest.WriteTempFile(t, "unlocated.png", uitest.EncodePNG(t, 2, 1, color.White)))
	if m, err := imaging.ReadMetadataURIContext(context.Background(), unlocated); err != nil || !m.Empty() {
		t.Fatalf("unlocated URI metadata = %+v, %v; want empty without error", m, err)
	}

	missing := storage.NewFileURI(filepath.Join(t.TempDir(), "missing.png"))
	if _, err := imaging.ReadMetadataURIContext(context.Background(), missing); err == nil {
		t.Fatal("missing URI returned no error")
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := imaging.ReadMetadataURIContext(ctx, uri); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled URI read error = %v, want context.Canceled", err)
	}

	imaging.SetMaxEncodedBytes(int64(len(data) - 1))
	t.Cleanup(func() { imaging.SetMaxEncodedBytes(0) })
	if _, err := imaging.ReadMetadataURIContext(context.Background(), uri); err == nil {
		t.Fatal("oversized URI returned no error")
	} else {
		var limitErr *imaging.InputTooLargeError
		if !errors.As(err, &limitErr) {
			t.Fatalf("oversized URI error = %v, want InputTooLargeError", err)
		}
	}
}

func TestReadMetadataContainerGPSPresenceAndBounds(t *testing.T) {
	for _, container := range []struct {
		name  string
		build func(*testing.T, []byte) []byte
	}{
		{"PNG", pngWithEXIF},
		{"WebP", webpWithEXIF},
	} {
		t.Run(container.name, func(t *testing.T) {
			zero := imaging.ReadMetadata(container.build(t, fixtureTIFF(t, 0, 0, "")))
			if !zero.HasGPS || zero.Latitude != 0 || zero.Longitude != 0 {
				t.Fatalf("explicit zero position = %+v", zero)
			}
			absent := imaging.ReadMetadata(container.build(t, dateOnlyTIFF()))
			if absent.HasGPS || absent.DateTakenTime.Year() != 2021 {
				t.Fatalf("date without GPS tags = %+v", absent)
			}
		})
	}

	payload := fixtureTIFF(t, 48.858222, 2.2945, "")
	png := pngWithEXIF(t, payload)
	chunkStart := 8 + 12 + int(binary.BigEndian.Uint32(png[8:12]))
	badPNGLength := append([]byte(nil), png...)
	binary.BigEndian.PutUint32(badPNGLength[chunkStart:chunkStart+4], uint32(len(payload)+1000))
	for name, data := range map[string][]byte{
		"PNG oversized eXIf length": badPNGLength,
		"PNG truncated eXIf":        png[:chunkStart+8+len(payload)-1],
	} {
		t.Run(name, func(t *testing.T) {
			if got := imaging.ReadMetadata(data); !got.Empty() {
				t.Fatalf("malformed PNG metadata = %+v, want empty", got)
			}
		})
	}

	webp := webpWithEXIF(t, payload)
	chunkStart = len(webp) - 8 - len(payload) - len(payload)%2
	badWebPLength := append([]byte(nil), webp...)
	binary.LittleEndian.PutUint32(badWebPLength[chunkStart+4:chunkStart+8], uint32(len(payload)+1000))
	for name, data := range map[string][]byte{
		"WebP oversized EXIF length": badWebPLength,
		"WebP truncated EXIF":        webp[:len(webp)-8],
	} {
		t.Run(name, func(t *testing.T) {
			if got := imaging.ReadMetadata(data); !got.Empty() {
				t.Fatalf("malformed WebP metadata = %+v, want empty", got)
			}
		})
	}
}
