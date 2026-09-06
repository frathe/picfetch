package uitest

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/color"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/displays"
)

func TestStubDisplays(t *testing.T) {
	want := displays.Snapshot{
		Displays: []displays.Display{{ID: "fixture", Name: "Fixture", Bounds: image.Rect(0, 0, 1920, 1080)}},
		Default:  "fixture",
	}
	StubDisplays(t, func(_ fyne.Window) (displays.Snapshot, error) {
		return want, nil
	})

	got, err := displays.Inspect(nil)
	if err != nil {
		t.Fatal(err)
	}
	if got.Default != want.Default || got.Displays[0] != want.Displays[0] {
		t.Fatalf("Inspect() = %+v, want %+v", got, want)
	}
}

func TestLittleEndianTIFF_WritesHeaderAndIntegers(t *testing.T) {
	buf := newLittleEndianTIFF()
	buf.u16(0x1234)
	buf.u32(0x89ABCDEF)

	want := []byte{
		'I', 'I', 0x2A, 0x00, 0x08, 0x00, 0x00, 0x00,
		0x34, 0x12, 0xEF, 0xCD, 0xAB, 0x89,
	}
	if !bytes.Equal(buf.Bytes(), want) {
		t.Fatalf("little-endian TIFF bytes = % X, want % X", buf.Bytes(), want)
	}
}

func TestWrapAPP1_SplicesExifAfterSOI(t *testing.T) {
	jpegData := []byte{0xFF, 0xD8, 0xFF, 0xD9}
	tiff := bytes.Repeat([]byte{0xAB}, 250)

	got := wrapAPP1(jpegData, tiff)

	const (
		markerAndLengthBytes = 4
		exifID               = "Exif\x00\x00"
	)
	wantLen := len(jpegData) + markerAndLengthBytes + len(exifID) + len(tiff)
	if len(got) != wantLen {
		t.Fatalf("len(wrapAPP1()) = %d, want %d", len(got), wantLen)
	}

	if !bytes.Equal(got[:2], jpegData[:2]) {
		t.Errorf("SOI = % X, want % X", got[:2], jpegData[:2])
	}
	if !bytes.Equal(got[2:4], []byte{0xFF, 0xE1}) {
		t.Errorf("marker = % X, want FF E1", got[2:4])
	}

	gotLength := int(binary.BigEndian.Uint16(got[4:6]))
	wantLength := 2 + len(exifID) + len(tiff)
	if gotLength != wantLength {
		t.Errorf("APP1 length = %d, want %d", gotLength, wantLength)
	}

	payloadStart := 6
	tiffStart := payloadStart + len(exifID)
	tiffEnd := tiffStart + len(tiff)
	if string(got[payloadStart:tiffStart]) != exifID {
		t.Errorf("Exif identifier = %q, want %q", got[payloadStart:tiffStart], exifID)
	}
	if !bytes.Equal(got[tiffStart:tiffEnd], tiff) {
		t.Error("TIFF payload changed while wrapping APP1")
	}
	if !bytes.Equal(got[tiffEnd:], jpegData[2:]) {
		t.Errorf("JPEG remainder = % X, want % X", got[tiffEnd:], jpegData[2:])
	}
}

// TestDimensionTaggedJPEG_RoundTripsThroughExifIFD0Tag covers the pair
// together, since a test asserting what an export left in a tag is only as
// trustworthy as the reader it asks: the builder writes the two dimension
// tags, and the reader reads back exactly those values and finds nothing
// the fixture never wrote.
func TestDimensionTaggedJPEG_RoundTripsThroughExifIFD0Tag(t *testing.T) {
	data := DimensionTaggedJPEG(t, 900, 600)

	for _, want := range []struct {
		tag   uint16
		value int
	}{{0x0100, 900}, {0x0101, 600}} {
		got, ok := ExifIFD0Tag(data, want.tag)
		if !ok {
			t.Errorf("tag %#04x is missing from a freshly built fixture", want.tag)
			continue
		}
		if got != want.value {
			t.Errorf("tag %#04x reads %d, want %d", want.tag, got, want.value)
		}
	}
	if _, ok := ExifIFD0Tag(data, 0x0112); ok { // Orientation, never written here
		t.Error("ExifIFD0Tag found a tag the fixture never wrote")
	}
}

func TestExifIFD0Tag_NotFoundWithoutReadableExif(t *testing.T) {
	for _, tc := range []struct {
		name string
		data []byte
	}{
		{"not a JPEG at all", []byte("plainly not a JPEG")},
		{"empty", nil},
		{"a JPEG with no Exif segment", EncodeJPEG(t, 4, 4, color.White)},
		{"truncated mid-segment", DimensionTaggedJPEG(t, 900, 600)[:12]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, ok := ExifIFD0Tag(tc.data, 0x0100); ok {
				t.Error("ExifIFD0Tag reported a tag it could not have read")
			}
		})
	}
}
