package uitest

import (
	"bytes"
	"encoding/binary"
	"testing"
)

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
