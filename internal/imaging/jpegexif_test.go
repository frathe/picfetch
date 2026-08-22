package imaging

import (
	"bytes"
	"encoding/binary"
	"image"
	"image/jpeg"
	"testing"
)

func TestIsExifAPP1(t *testing.T) {
	exif := wrapAsAPP1(buildExifSegment(t, 6, false))
	if !isExifAPP1(exif) {
		t.Fatal("isExifAPP1: want true for an Exif APP1")
	}
	xmp := wrapAsAPP1([]byte("http://ns.adobe.com/xap/1.0/\x00<x:xmpmeta/>"))
	if isExifAPP1(xmp) {
		t.Fatal("isExifAPP1: want false for XMP")
	}
	if isExifAPP1(nil) || isExifAPP1([]byte{0xFF, 0xE1}) {
		t.Fatal("isExifAPP1: want false for truncated input")
	}
	// Regression: seg[4:10] requires len(seg) >= 10, so an 8- or 9-byte
	// slice with the right marker prefix must be rejected by the length
	// guard rather than reaching that slice and panicking.
	if isExifAPP1([]byte{0xFF, 0xE1, 0, 0, 0, 0, 0, 0}) {
		t.Fatal("isExifAPP1: want false for an 8-byte APP1 prefix")
	}
	if isExifAPP1([]byte{0xFF, 0xE1, 0, 0, 0, 0, 0, 0, 0}) {
		t.Fatal("isExifAPP1: want false for a 9-byte APP1 prefix")
	}
}

func TestJPEGMetadataSegments(t *testing.T) {
	t.Run("keeps Exif APP1, XMP APP1, ICC APP2, COM; skips JFIF APP0 and MPF", func(t *testing.T) {
		jfif := []byte{0xFF, 0xE0, 0x00, 0x10, 'J', 'F', 'I', 'F', 0x00, 1, 1, 0, 0, 1, 0, 1, 0, 0}
		com := []byte{0xFF, 0xFE, 0x00, 0x05, 'h', 'i', 0x00}
		xmp := wrapAsAPP1([]byte("http://ns.adobe.com/xap/1.0/\x00<x/>"))
		exif := wrapAsAPP1(buildExifSegment(t, 6, false))
		icc := wrapAPP2([]byte("ICC_PROFILE\x00\x01\x01dummy-icc"))
		mpf := wrapAPP2([]byte("MPF\x00not-a-real-mpf"))

		var data []byte
		data = append(data, 0xFF, 0xD8)
		data = append(data, jfif...)
		data = append(data, com...)
		data = append(data, xmp...)
		data = append(data, exif...)
		data = append(data, icc...)
		data = append(data, mpf...)
		data = append(data, 0xFF, 0xDA, 0x00, 0x08, 0, 0, 0, 0, 0, 0) // SOS: walk must stop

		got := jpegMetadataSegments(data)
		if len(got) != 4 {
			t.Fatalf("got %d segments, want 4 (COM, XMP, Exif, ICC)", len(got))
		}
		if !bytes.Equal(got[0], com) || !bytes.Equal(got[1], xmp) || !bytes.Equal(got[2], exif) || !bytes.Equal(got[3], icc) {
			t.Fatalf("segments = %x, want COM, XMP, Exif, ICC in that order", got)
		}
		got[2][4] = 'X'
		if !isExifAPP1(exif) {
			t.Fatal("jpegMetadataSegments must return copies")
		}
	})

	t.Run("finds Exif after jpeg.Encode's JFIF, the way real files look", func(t *testing.T) {
		data := halfRedHalfBlueJPEG(t, 8, 8, 6)
		var exif []byte
		for _, s := range jpegMetadataSegments(data) {
			if isExifAPP1(s) {
				exif = s
			}
		}
		if exif == nil {
			t.Fatal("expected an Exif APP1 in halfRedHalfBlueJPEG")
		}
		if parseExifOrientation(exif[4:]) != 6 {
			t.Errorf("orientation = %d, want 6", parseExifOrientation(exif[4:]))
		}
	})

	t.Run("nil for a JPEG with no metadata and for non-JPEG", func(t *testing.T) {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
			t.Fatal(err)
		}
		if segs := jpegMetadataSegments(buf.Bytes()); len(segs) != 0 {
			t.Errorf("stdlib JPEG metadata = %d segments, want 0 (JFIF APP0 skipped)", len(segs))
		}
		if jpegMetadataSegments([]byte("\x89PNG")) != nil {
			t.Error("want nil for PNG magic")
		}
		if jpegMetadataSegments(nil) != nil {
			t.Error("want nil for nil")
		}
	})
}

func wrapAPP2(payload []byte) []byte {
	length := len(payload) + 2
	return append([]byte{0xFF, 0xE2, byte(length >> 8), byte(length)}, payload...)
}

func TestInjectJPEGMetadata(t *testing.T) {
	t.Run("inserts immediately after SOI, before JFIF APP0", func(t *testing.T) {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
			t.Fatal(err)
		}
		encoded := buf.Bytes()
		com := []byte{0xFF, 0xFE, 0x00, 0x05, 'h', 'i', 0x00}
		exif := wrapAsAPP1(buildExifSegment(t, 8, false))

		out, err := injectJPEGMetadata(encoded, [][]byte{com, exif})
		if err != nil {
			t.Fatalf("injectJPEGMetadata: %v", err)
		}
		if out[0] != 0xFF || out[1] != 0xD8 {
			t.Fatal("output lost SOI")
		}
		if !bytes.Equal(out[2:2+len(com)], com) {
			t.Fatal("COM was not placed immediately after SOI")
		}
		if !bytes.Equal(out[2+len(com):2+len(com)+len(exif)], exif) {
			t.Fatal("Exif APP1 was not placed after COM")
		}
		if !bytes.Equal(out[2+len(com)+len(exif):], encoded[2:]) {
			t.Fatal("bytes after SOI were not preserved")
		}

		got := jpegMetadataSegments(out)
		if len(got) != 2 || !bytes.Equal(got[0], com) || !isExifAPP1(got[1]) {
			t.Fatal("extract did not round-trip the injected segments")
		}
	})

	t.Run("empty segs returns a copy of encoded", func(t *testing.T) {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
			t.Fatal(err)
		}
		encoded := buf.Bytes()
		out, err := injectJPEGMetadata(encoded, nil)
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(out, encoded) {
			t.Fatal("empty segs must preserve the encoded JPEG")
		}
		out[0] = 0
		if encoded[0] != 0xFF {
			t.Fatal("must not alias encoded")
		}
	})

	t.Run("rejects a non-JPEG encoded buffer", func(t *testing.T) {
		exif := wrapAsAPP1(buildExifSegment(t, 1, false))
		if _, err := injectJPEGMetadata([]byte("\x89PNG"), [][]byte{exif}); err == nil {
			t.Fatal("want error")
		}
	})
}

// buildExifWithThumbnailIFD builds the payload of an APP1 Exif segment
// (starting with the "Exif\0\0" marker) whose little-endian TIFF has an
// IFD0 with a single Orientation=6 entry, a next-IFD pointer to IFD1, and
// an IFD1 with a single dummy ImageWidth=16 entry and next-IFD 0 - the
// shape a thumbnail-carrying Exif segment has before saving.
func buildExifWithThumbnailIFD(t *testing.T) []byte {
	t.Helper()

	bo := binary.LittleEndian

	u16 := func(v uint16) []byte { b := make([]byte, 2); bo.PutUint16(b, v); return b }
	u32 := func(v uint32) []byte { b := make([]byte, 4); bo.PutUint32(b, v); return b }

	const headerSize = 8
	ifd0Offset := uint32(headerSize)
	ifd0Size := 2 + 1*12 + 4
	ifd1Offset := ifd0Offset + uint32(ifd0Size)

	buf := new(bytes.Buffer)
	buf.WriteString("Exif\x00\x00")
	buf.WriteString("II")
	buf.Write(u16(0x002A))
	buf.Write(u32(ifd0Offset))

	buf.Write(u16(1)) // IFD0: one entry

	buf.Write(u16(0x0112)) // Orientation tag
	buf.Write(u16(3))      // type SHORT
	buf.Write(u32(1))      // count 1
	orientationValue := make([]byte, 4)
	bo.PutUint16(orientationValue, 6)
	buf.Write(orientationValue)

	buf.Write(u32(ifd1Offset)) // IFD0 next-IFD -> IFD1

	if buf.Len() != int(6+ifd1Offset) {
		t.Fatalf("IFD0 layout mismatch: wrote %d bytes, want %d", buf.Len(), 6+ifd1Offset)
	}

	buf.Write(u16(1)) // IFD1: one entry

	buf.Write(u16(0x0100)) // ImageWidth tag (dummy)
	buf.Write(u16(3))      // type SHORT
	buf.Write(u32(1))      // count 1
	widthValue := make([]byte, 4)
	bo.PutUint16(widthValue, 16)
	buf.Write(widthValue)

	buf.Write(u32(0)) // IFD1 next-IFD: none

	return buf.Bytes()
}

func TestNormalizeSavedExif(t *testing.T) {
	t.Run("sets orientation 6 to 1 and leaves the rest of the payload intact", func(t *testing.T) {
		app1 := wrapAsAPP1(buildExifSegment(t, 6, false))
		got := normalizeSavedExif(app1)
		if parseExifOrientation(got[4:]) != 1 {
			t.Errorf("orientation = %d, want 1", parseExifOrientation(got[4:]))
		}
		if parseExifOrientation(app1[4:]) != 6 {
			t.Fatal("normalizeSavedExif mutated the input segment")
		}
	})

	t.Run("big-endian orientation 8 becomes 1", func(t *testing.T) {
		app1 := wrapAsAPP1(buildExifSegment(t, 8, true))
		got := normalizeSavedExif(app1)
		if parseExifOrientation(got[4:]) != 1 {
			t.Errorf("orientation = %d, want 1", parseExifOrientation(got[4:]))
		}
	})

	t.Run("zeros IFD0 next-IFD so IFD1 is unlinked", func(t *testing.T) {
		app1 := wrapAsAPP1(buildExifWithThumbnailIFD(t))
		tiff := app1[10:]
		le := binary.LittleEndian
		ifd0 := le.Uint32(tiff[4:8])
		num := le.Uint16(tiff[ifd0 : ifd0+2])
		nextOff := ifd0 + 2 + uint32(num)*12
		if le.Uint32(tiff[nextOff:nextOff+4]) == 0 {
			t.Fatal("fixture: IFD1 pointer should be non-zero before normalize")
		}

		got := normalizeSavedExif(app1)
		gtiff := got[10:]
		gnum := le.Uint16(gtiff[ifd0 : ifd0+2])
		gnext := ifd0 + 2 + uint32(gnum)*12
		if le.Uint32(gtiff[gnext:gnext+4]) != 0 {
			t.Errorf("next IFD = %d, want 0", le.Uint32(gtiff[gnext:gnext+4]))
		}
		if parseExifOrientation(got[4:]) != 1 {
			t.Errorf("orientation = %d, want 1", parseExifOrientation(got[4:]))
		}
	})

	t.Run("XMP APP1 is returned copied, not rewritten as Exif", func(t *testing.T) {
		in := wrapAsAPP1([]byte("http://ns.adobe.com/xap/1.0/\x00<x/>"))
		got := normalizeSavedExif(in)
		if !bytes.Equal(got, in) {
			t.Fatalf("got %x, want copy of XMP", got)
		}
		got[4] = 'x'
		if in[4] == 'x' {
			t.Fatal("must not alias the input")
		}
	})
}

func TestEncodeJPEGPreservingMetadata(t *testing.T) {
	t.Run("copies GPS, XMP, ICC, and COM from orig", func(t *testing.T) {
		exif := wrapAsAPP1(append([]byte("Exif\x00\x00"), buildGPSExifTIFF(t, gpsFields{
			latRef: "N", lat: [3][2]uint32{{48, 1}, {51, 1}, {2960, 100}},
			lonRef: "E", lon: [3][2]uint32{{2, 1}, {17, 1}, {4020, 100}},
		})...))
		xmp := wrapAsAPP1([]byte("http://ns.adobe.com/xap/1.0/\x00<x/>"))
		icc := wrapAPP2([]byte("ICC_PROFILE\x00\x01\x01dummy-icc"))
		com := []byte{0xFF, 0xFE, 0x00, 0x05, 'h', 'i', 0x00}
		orig := spliceMetadataIntoJPEG(t, markedImage(4, 3), [][]byte{com, xmp, exif, icc})

		var out bytes.Buffer
		if err := encodeJPEGPreservingMetadata(&out, markedImage(3, 2), orig); err != nil {
			t.Fatal(err)
		}
		m := ReadMetadata(out.Bytes())
		if !m.HasGPS {
			t.Fatal("saved JPEG lost GPS")
		}
		got := jpegMetadataSegments(out.Bytes())
		if len(got) != 4 {
			t.Fatalf("saved metadata segments = %d, want 4", len(got))
		}
		if !bytes.Equal(got[0], com) {
			t.Fatal("lost COM")
		}
		if !bytes.Equal(got[1], xmp) {
			t.Fatal("lost XMP")
		}
		if !bytes.Contains(got[3], []byte("ICC_PROFILE")) {
			t.Fatal("lost ICC")
		}
	})

	t.Run("orig without metadata encodes a JPEG without extra APPn", func(t *testing.T) {
		var origBuf bytes.Buffer
		if err := jpeg.Encode(&origBuf, markedImage(2, 2), nil); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if err := encodeJPEGPreservingMetadata(&out, markedImage(2, 2), origBuf.Bytes()); err != nil {
			t.Fatal(err)
		}
		if segs := jpegMetadataSegments(out.Bytes()); len(segs) != 0 {
			t.Errorf("invented %d metadata segments", len(segs))
		}
	})
}

func spliceMetadataIntoJPEG(t *testing.T, img image.Image, segs [][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 100}); err != nil {
		t.Fatal(err)
	}
	out, err := injectJPEGMetadata(buf.Bytes(), segs)
	if err != nil {
		t.Fatal(err)
	}
	return out
}
