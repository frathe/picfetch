package imaging

import (
	"bytes"
	"encoding/binary"
	"errors"
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

func TestIsICCAPP2(t *testing.T) {
	icc := wrapAPP2([]byte("ICC_PROFILE\x00\x01\x01dummy-icc"))
	if !isICCAPP2(icc) {
		t.Fatal("isICCAPP2: want true for an ICC APP2")
	}
	mpf := wrapAPP2([]byte("MPF\x00x"))
	if isICCAPP2(mpf) {
		t.Fatal("isICCAPP2: want false for MPF")
	}
	if isICCAPP2(nil) || isICCAPP2([]byte{0xFF, 0xE2}) {
		t.Fatal("isICCAPP2: want false for truncated input")
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

func wrapAPP14() []byte {
	return []byte{0xFF, 0xEE, 0x00, 0x0E, 'A', 'd', 'o', 'b', 'e', 0x00, 0x64, 0x00, 0x00, 0x00, 0x00, 0x00}
}

func TestKeepOnStrip(t *testing.T) {
	if !keepOnStrip(0xE0, []byte("JFIF")) {
		t.Fatal("keep APP0")
	}
	if !keepOnStrip(0xEE, []byte("Adobe")) {
		t.Fatal("keep APP14")
	}
	if keepOnStrip(0xFE, []byte("hi")) {
		t.Fatal("drop COM")
	}
	if keepOnStrip(0xE1, []byte("Exif\x00\x00")) {
		t.Fatal("drop Exif APP1")
	}
	icc := []byte("ICC_PROFILE\x00\x01\x01x")
	if !keepOnStrip(0xE2, icc) {
		t.Fatal("keep ICC APP2")
	}
	if keepOnStrip(0xE2, []byte("MPF\x00x")) {
		t.Fatal("drop MPF APP2")
	}
	if keepOnStrip(0xED, []byte("Photoshop")) {
		t.Fatal("drop IPTC/APP13")
	}
}

func TestJPEGHasRemovableMetadata(t *testing.T) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
	plain := buf.Bytes()
	if jpegHasRemovableMetadata(plain) {
		t.Fatal("stdlib JPEG is only JFIF APP0; nothing removable")
	}
	exif := wrapAsAPP1(buildExifSegment(t, 1, false))
	withExif := append([]byte{}, plain[:2]...)
	withExif = append(withExif, exif...)
	withExif = append(withExif, plain[2:]...)
	if !jpegHasRemovableMetadata(withExif) {
		t.Fatal("Exif APP1 is removable")
	}
	if jpegHasRemovableMetadata([]byte("\x89PNG")) {
		t.Fatal("non-JPEG is not removable metadata")
	}

	if jpegHasRemovableMetadata(appendAfterEOI(t, plain, gpsTrailerJPEG(t))) != true {
		t.Fatal("GPS JPEG after primary EOI is removable")
	}
	if jpegHasRemovableMetadata(appendAfterEOI(t, plain, []byte("ftypmp42fake-video"))) != true {
		t.Fatal("motion-photo bytes after primary EOI are removable")
	}
}

func eiffelGPS() gpsFields {
	return gpsFields{
		latRef: "N", lat: [3][2]uint32{{48, 1}, {51, 1}, {2960, 100}},
		lonRef: "E", lon: [3][2]uint32{{2, 1}, {17, 1}, {4020, 100}},
	}
}

func TestStripJPEGSegments(t *testing.T) {
	t.Run("drops Exif XMP COM MPF IPTC; keeps APP0 APP14 ICC and the scan", func(t *testing.T) {
		com := []byte{0xFF, 0xFE, 0x00, 0x05, 'h', 'i', 0x00}
		xmp := wrapAsAPP1([]byte("http://ns.adobe.com/xap/1.0/\x00<x/>"))
		exif := wrapAsAPP1(buildExifSegment(t, 1, false))
		icc := wrapAPP2([]byte("ICC_PROFILE\x00\x01\x01dummy-icc"))
		mpf := wrapAPP2([]byte("MPF\x00not-a-real-mpf"))
		iptc := []byte{0xFF, 0xED, 0x00, 0x0C, 'P', 'h', 'o', 't', 'o', 's', 'h', 'o', 'p', 0x00}
		adobe := wrapAPP14()

		data := spliceMetadataIntoJPEG(t, markedImage(4, 4), [][]byte{
			com, xmp, exif, icc, mpf, iptc, adobe,
		})

		got, err := stripJPEGSegments(data)
		if err != nil {
			t.Fatalf("stripJPEGSegments: %v", err)
		}
		if jpegHasRemovableMetadata(got) {
			t.Fatal("still has removable segments")
		}
		if !bytes.Contains(got, []byte("ICC_PROFILE")) {
			t.Fatal("lost ICC")
		}
		if !bytes.Contains(got, []byte("Adobe")) {
			t.Fatal("lost APP14")
		}
		if bytes.Contains(got, []byte("Exif\x00\x00")) || bytes.Contains(got, []byte("xap/1.0")) || bytes.Contains(got, []byte("MPF\x00")) {
			t.Fatal("left identifying or MPF segments")
		}
		if _, err := jpeg.Decode(bytes.NewReader(got)); err != nil {
			t.Fatalf("stripped file must still decode: %v", err)
		}
	})

	t.Run("orientation-1 GPS JPEG is bit-identical in the scan after strip", func(t *testing.T) {
		exif := wrapAsAPP1(append([]byte("Exif\x00\x00"), buildGPSExifTIFF(t, eiffelGPS())...))
		data := spliceMetadataIntoJPEG(t, markedImage(8, 8), [][]byte{exif})
		if !ReadMetadata(data).HasGPS {
			t.Fatal("setup: want GPS")
		}
		got, err := stripJPEGSegments(data)
		if err != nil {
			t.Fatal(err)
		}
		if !ReadMetadata(got).Empty() {
			t.Fatalf("ReadMetadata after strip = %+v, want empty", ReadMetadata(got))
		}
		sos := bytes.Index(data, []byte{0xFF, 0xDA})
		if sos < 0 {
			t.Fatal("setup: no SOS")
		}
		gotSOS := bytes.Index(got, []byte{0xFF, 0xDA})
		if gotSOS < 0 {
			t.Fatal("stripped file lost SOS")
		}
		if !bytes.Equal(data[sos:], got[gotSOS:]) {
			t.Fatal("lossless strip must copy the entropy-coded scan verbatim")
		}
	})

	t.Run("non-JPEG is errNotJPEG", func(t *testing.T) {
		_, err := stripJPEGSegments([]byte("\x89PNG"))
		if !errors.Is(err, errNotJPEG) {
			t.Fatalf("err = %v, want errNotJPEG", err)
		}
	})

	t.Run("drops a GPS JPEG concatenated after the primary EOI", func(t *testing.T) {
		exif := wrapAsAPP1(append([]byte("Exif\x00\x00"), buildGPSExifTIFF(t, eiffelGPS())...))
		mpf := wrapAPP2([]byte("MPF\x00not-a-real-mpf"))
		primary := spliceMetadataIntoJPEG(t, markedImage(8, 8), [][]byte{exif, mpf})
		data := appendAfterEOI(t, primary, gpsTrailerJPEG(t))
		if !bytes.Contains(data, []byte("Exif\x00\x00")) {
			t.Fatal("setup: want Exif in the trailer file")
		}

		got, err := stripJPEGSegments(data)
		if err != nil {
			t.Fatalf("stripJPEGSegments: %v", err)
		}
		if jpegHasRemovableMetadata(got) {
			t.Fatal("still has removable metadata or a trailer")
		}
		if n := jpegLength(got); n != len(got) {
			t.Fatalf("stripped length %d, jpegLength %d (trailer left)", len(got), n)
		}
		if bytes.Contains(got, []byte("Exif\x00\x00")) || bytes.Contains(got, []byte("MPF\x00")) {
			t.Fatal("left identifying tags or MPF")
		}
		if _, err := jpeg.Decode(bytes.NewReader(got)); err != nil {
			t.Fatalf("stripped file must still decode: %v", err)
		}
		sos := bytes.Index(primary, []byte{0xFF, 0xDA})
		gotSOS := bytes.Index(got, []byte{0xFF, 0xDA})
		if sos < 0 || gotSOS < 0 {
			t.Fatal("missing SOS")
		}
		if !bytes.Equal(primary[sos:], got[gotSOS:]) {
			t.Fatal("lossless strip must copy the primary scan through EOI, not the trailer")
		}
	})

	t.Run("drops a trailer when the primary header has nothing removable", func(t *testing.T) {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
			t.Fatal(err)
		}
		plain := buf.Bytes()
		data := appendAfterEOI(t, plain, gpsTrailerJPEG(t))

		got, err := stripJPEGSegments(data)
		if err != nil {
			t.Fatal(err)
		}
		if jpegHasRemovableMetadata(got) {
			t.Fatal("plain primary plus dropped trailer must not look removable")
		}
		if bytes.Contains(got, []byte("Exif\x00\x00")) {
			t.Fatal("left trailer Exif")
		}
		if !bytes.Equal(got, plain) {
			t.Fatal("header-clean primary must be unchanged aside from dropping the trailer")
		}
	})

	t.Run("copy through EOF when the JPEG has no EOI", func(t *testing.T) {
		var buf bytes.Buffer
		if err := jpeg.Encode(&buf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
			t.Fatal(err)
		}
		truncated := buf.Bytes()
		if n := jpegLength(truncated); n != len(truncated) {
			t.Fatalf("setup: stdlib JPEG should close, jpegLength=%d len=%d", n, len(truncated))
		}
		truncated = truncated[:len(truncated)-2] // drop EOI
		if jpegLength(truncated) != 0 {
			t.Fatal("setup: want jpegLength 0 after chopping EOI")
		}
		got, err := stripJPEGSegments(truncated)
		if err != nil {
			t.Fatalf("stripJPEGSegments: %v", err)
		}
		if !bytes.Equal(got, truncated) {
			t.Fatal("no-EOI JPEG must still copy through EOF")
		}
	})
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
		got := normalizeSavedExif(app1, image.Point{})
		if parseExifOrientation(got[4:]) != 1 {
			t.Errorf("orientation = %d, want 1", parseExifOrientation(got[4:]))
		}
		if parseExifOrientation(app1[4:]) != 6 {
			t.Fatal("normalizeSavedExif mutated the input segment")
		}
	})

	t.Run("big-endian orientation 8 becomes 1", func(t *testing.T) {
		app1 := wrapAsAPP1(buildExifSegment(t, 8, true))
		got := normalizeSavedExif(app1, image.Point{})
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

		got := normalizeSavedExif(app1, image.Point{})
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
		got := normalizeSavedExif(in, image.Point{})
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
		if err := encodeJPEGPreservingMetadata(&out, markedImage(3, 2), orig, image.Point{}); err != nil {
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
		if err := encodeJPEGPreservingMetadata(&out, markedImage(2, 2), origBuf.Bytes(), image.Point{}); err != nil {
			t.Fatal(err)
		}
		if segs := jpegMetadataSegments(out.Bytes()); len(segs) != 0 {
			t.Errorf("invented %d metadata segments", len(segs))
		}
	})
}

func TestEncodeJPEGKeepingICC(t *testing.T) {
	t.Run("keeps ICC and drops Exif", func(t *testing.T) {
		orig := halfRedHalfBlueJPEG(t, 8, 8, 6)
		icc := wrapAPP2([]byte("ICC_PROFILE\x00\x01\x01dummy-icc"))
		adobe := wrapAPP14()
		withICC, err := injectJPEGMetadata(orig, [][]byte{icc, adobe})
		if err != nil {
			t.Fatal(err)
		}

		var out bytes.Buffer
		if err := encodeJPEGKeepingICC(&out, markedImage(3, 2), withICC); err != nil {
			t.Fatal(err)
		}
		got := out.Bytes()
		if !bytes.Contains(got, []byte("ICC_PROFILE")) {
			t.Fatal("want ICC APP2 on the re-encode")
		}
		if bytes.Contains(got, []byte("Exif")) {
			t.Fatal("must not copy Exif onto the strip re-encode")
		}
		if bytes.Contains(got, []byte("Adobe")) {
			t.Fatal("must not copy APP14 onto the strip re-encode")
		}
	})

	t.Run("no ICC encodes like encodeJPEGForSave", func(t *testing.T) {
		var origBuf bytes.Buffer
		if err := jpeg.Encode(&origBuf, markedImage(2, 2), &jpeg.Options{Quality: 90}); err != nil {
			t.Fatal(err)
		}
		var out bytes.Buffer
		if err := encodeJPEGKeepingICC(&out, markedImage(2, 2), origBuf.Bytes()); err != nil {
			t.Fatal(err)
		}
		if segs := jpegICCSegments(out.Bytes()); len(segs) != 0 {
			t.Errorf("invented %d ICC segments", len(segs))
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

// appendAfterEOI returns jpeg with trailer appended after its primary EOI.
// jpeg must be a closed JPEG (jpegLength(jpeg) == len(jpeg)) with no
// trailer of its own already.
func appendAfterEOI(t *testing.T, jpeg, trailer []byte) []byte {
	t.Helper()
	n := jpegLength(jpeg)
	if n == 0 || n != len(jpeg) {
		t.Fatalf("setup: primary must be a closed JPEG with no trailer, jpegLength=%d len=%d", n, len(jpeg))
	}
	out := make([]byte, 0, len(jpeg)+len(trailer))
	return append(append(out, jpeg...), trailer...)
}

// gpsTrailerJPEG builds a small closed JPEG carrying GPS Exif, suitable as
// the kind of concatenated "second JPEG" a trailer strip test appends
// after a primary image's EOI.
func gpsTrailerJPEG(t *testing.T) []byte {
	t.Helper()
	exif := wrapAsAPP1(append([]byte("Exif\x00\x00"), buildGPSExifTIFF(t, eiffelGPS())...))
	return spliceMetadataIntoJPEG(t, markedImage(4, 4), [][]byte{exif})
}

// --- dropping dimension tags a resize invalidated ---------------------------

// dimensionExif is the fixture the tag-dropping tests work from: the tags a
// resize makes false, spread across all three IFDs that carry them, mixed
// in with the ones that must survive it - camera, lens, exposure, date,
// GPS, MakerNote and the resolution/DPI trio.
type dimensionExif struct {
	width, height uint32
	orientation   uint16
	makerNote     []byte
}

// tiffEntry is one IFD entry for buildDimensionExifTIFF. Exactly one of
// value (an inline value, already in the entry's own 4 bytes) and data (an
// oversized value, placed in the trailing value area) is used; ifd names a
// sub-IFD whose offset becomes the value instead.
type tiffEntry struct {
	tag, typ uint16
	count    uint32
	value    uint32
	data     []byte
	ifd      int // index into the IFD list below, or -1
}

const (
	tiffIFD0 = iota
	tiffExifIFD
	tiffInteropIFD
	tiffGPSIFD
	tiffIFDCount
)

// buildDimensionExifTIFF lays out a little-endian TIFF with IFD0, an Exif
// SubIFD, an Interoperability IFD (reached through the 0xA005 pointer in
// the Exif SubIFD, not through IFD0) and a GPS IFD, followed by one shared
// value area every oversized value points into with an absolute offset.
// The four IFDs are written back to back, so removing an entry from one
// leaves the ones after it exactly where they were - which is the property
// the removal has to preserve.
func buildDimensionExifTIFF(t *testing.T, f dimensionExif) []byte {
	t.Helper()

	bo := binary.LittleEndian
	u16 := func(v uint16) []byte { b := make([]byte, 2); bo.PutUint16(b, v); return b }
	u32 := func(v uint32) []byte { b := make([]byte, 4); bo.PutUint32(b, v); return b }
	asciiZ := func(s string) []byte { return append([]byte(s), 0) }
	rational := func(num, den uint32) []byte { return append(u32(num), u32(den)...) }
	shorts := func(a, b uint16) uint32 { return uint32(a) | uint32(b)<<16 }

	ifds := make([][]tiffEntry, tiffIFDCount)
	ifds[tiffIFD0] = []tiffEntry{
		{tag: 0x0100, typ: 4, count: 1, value: f.width, ifd: -1},  // ImageWidth
		{tag: 0x0101, typ: 4, count: 1, value: f.height, ifd: -1}, // ImageLength
		{tag: 0x010F, typ: 2, data: asciiZ("Canon"), ifd: -1},     // Make
		{tag: 0x0110, typ: 2, data: asciiZ("EOS 90D"), ifd: -1},   // Model
		{tag: 0x0112, typ: 3, count: 1, value: uint32(f.orientation), ifd: -1},
		{tag: 0x011A, typ: 5, count: 1, data: rational(300, 1), ifd: -1}, // XResolution
		{tag: 0x011B, typ: 5, count: 1, data: rational(300, 1), ifd: -1}, // YResolution
		{tag: 0x0128, typ: 3, count: 1, value: 2, ifd: -1},               // ResolutionUnit
		{tag: 0x8769, typ: 4, count: 1, ifd: tiffExifIFD},                // Exif SubIFD
		{tag: 0x8825, typ: 4, count: 1, ifd: tiffGPSIFD},                 // GPS IFD
	}
	ifds[tiffExifIFD] = []tiffEntry{
		{tag: 0x829A, typ: 5, count: 1, data: rational(1, 200), ifd: -1},    // ExposureTime
		{tag: 0x9003, typ: 2, data: asciiZ("2024:08:12 14:33:02"), ifd: -1}, // DateTimeOriginal
		{tag: 0x9214, typ: 3, count: 2, value: shorts(120, 80), ifd: -1},    // SubjectArea
		{tag: 0x927C, typ: 7, count: uint32(len(f.makerNote)), data: f.makerNote, ifd: -1},
		{tag: 0xA002, typ: 4, count: 1, value: f.width, ifd: -1},        // PixelXDimension
		{tag: 0xA003, typ: 4, count: 1, value: f.height, ifd: -1},       // PixelYDimension
		{tag: 0xA005, typ: 4, count: 1, ifd: tiffInteropIFD},            // Interoperability
		{tag: 0xA214, typ: 3, count: 2, value: shorts(60, 40), ifd: -1}, // SubjectLocation
		{tag: 0xA434, typ: 2, data: asciiZ("EF50mm f/1.8"), ifd: -1},    // LensModel
	}
	ifds[tiffInteropIFD] = []tiffEntry{
		{tag: 0x0001, typ: 2, count: 4, value: bo.Uint32(asciiZ("R98")), ifd: -1}, // InteropIndex
		{tag: 0x1001, typ: 4, count: 1, value: f.width, ifd: -1},                  // RelatedImageWidth
		{tag: 0x1002, typ: 4, count: 1, value: f.height, ifd: -1},                 // RelatedImageLength
	}
	ifds[tiffGPSIFD] = []tiffEntry{
		{tag: 0x0001, typ: 2, count: 2, value: bo.Uint32([]byte{'N', 0, 0, 0}), ifd: -1},
		{tag: 0x0002, typ: 5, count: 3, data: append(append(rational(48, 1), rational(51, 1)...), rational(29, 1)...), ifd: -1},
		{tag: 0x0003, typ: 2, count: 2, value: bo.Uint32([]byte{'E', 0, 0, 0}), ifd: -1},
		{tag: 0x0004, typ: 5, count: 3, data: append(append(rational(2, 1), rational(17, 1)...), rational(38, 1)...), ifd: -1},
	}

	offsets := make([]uint32, tiffIFDCount)
	next := uint32(tiffHeaderBytes)
	for i, entries := range ifds {
		offsets[i] = next
		next += uint32(2 + len(entries)*12 + 4)
	}
	valueAreaStart := next

	var valueArea []byte
	place := func(b []byte) uint32 {
		offset := valueAreaStart + uint32(len(valueArea))
		valueArea = append(valueArea, b...)
		return offset
	}

	buf := new(bytes.Buffer)
	buf.WriteString("II")
	buf.Write(u16(0x002A))
	buf.Write(u32(offsets[tiffIFD0]))

	for _, entries := range ifds {
		buf.Write(u16(uint16(len(entries))))
		for _, e := range entries {
			count := e.count
			value := e.value
			switch {
			case e.ifd >= 0:
				value = offsets[e.ifd]
			case len(e.data) > 4:
				if count == 0 {
					count = uint32(len(e.data))
				}
				value = place(e.data)
			case len(e.data) > 0:
				if count == 0 {
					count = uint32(len(e.data))
				}
				padded := make([]byte, 4)
				copy(padded, e.data)
				value = bo.Uint32(padded)
			}
			buf.Write(u16(e.tag))
			buf.Write(u16(e.typ))
			buf.Write(u32(count))
			buf.Write(u32(value))
		}
		buf.Write(u32(0)) // next-IFD pointer
	}

	if buf.Len() != int(valueAreaStart) {
		t.Fatalf("IFD layout mismatch: wrote %d bytes, want %d", buf.Len(), valueAreaStart)
	}
	buf.Write(valueArea)

	return buf.Bytes()
}

// tiffHeaderBytes is the byte order marker, magic and IFD0 offset that open
// every TIFF payload.
const tiffHeaderBytes = 8

// dimensionTagJPEG is buildDimensionExifTIFF spliced into a real JPEG, the
// shape a camera file reaches Export in.
func dimensionTagJPEG(t *testing.T, w, h int) []byte {
	t.Helper()

	tiff := buildDimensionExifTIFF(t, dimensionExif{
		width:       uint32(w),
		height:      uint32(h),
		orientation: 6,
		makerNote:   []byte("MAKERNOTE-8"),
	})

	return spliceMetadataIntoJPEG(t, markedImage(w, h),
		[][]byte{wrapAsAPP1(append([]byte("Exif\x00\x00"), tiff...))})
}

// exifTIFFOf is the TIFF payload of data's Exif APP1 segment.
func exifTIFFOf(t *testing.T, data []byte) []byte {
	t.Helper()

	var tiff []byte
	walkJPEGSegments(data, func(marker byte, payload []byte) bool {
		if marker == 0xE1 && len(payload) >= 8 && string(payload[:6]) == "Exif\x00\x00" {
			tiff = payload[6:]
			return false
		}
		return true
	})
	if tiff == nil {
		t.Fatal("no Exif APP1 segment in the written file")
	}

	return tiff
}

// readIFDs walks a written file's IFD0, Exif SubIFD and Interoperability
// IFD and returns each one's tags mapped to their values - what a reader of
// the exported file actually sees.
func readIFDs(t *testing.T, data []byte) map[int]map[uint16][]byte {
	t.Helper()

	tiff := exifTIFFOf(t, data)
	bo, ok := tiffOrder(tiff)
	if !ok {
		t.Fatal("the written file's Exif payload is not a TIFF")
	}

	out := map[int]map[uint16][]byte{}
	collect := func(offset uint32) map[uint16][]byte {
		tags := map[uint16][]byte{}
		walkIFD(tiff, bo, offset, func(tag, _ uint16, val []byte) {
			tags[tag] = append([]byte(nil), val...)
		})
		return tags
	}

	out[tiffIFD0] = collect(bo.Uint32(tiff[4:8]))
	if v, ok := out[tiffIFD0][0x8769]; ok && len(v) >= 4 {
		out[tiffExifIFD] = collect(bo.Uint32(v))
		if iv, ok := out[tiffExifIFD][0xA005]; ok && len(iv) >= 4 {
			out[tiffInteropIFD] = collect(bo.Uint32(iv))
		}
	}

	return out
}

// TestRemoveIFDEntries covers the entry removal itself - the one piece of
// genuinely new machinery in the export options feature, and the only place
// in this module that has ever moved an IFD entry rather than overwriting
// one in place.
func TestRemoveIFDEntries(t *testing.T) {
	bo := binary.LittleEndian

	// The dimension tags spelled out rather than taken from
	// ifd0DimensionTags, which now carries an axis per tag: what is under
	// test here is the removal mechanics, over a plain list of tags.
	dropped := []uint16{0x0100, 0x0101}

	// A four-entry IFD at offset 8 with a non-zero next-IFD pointer, so a
	// removal that forgot to rewrite that pointer at its new position would
	// show up as a zero rather than as the value below.
	const nextIFD = 0x1234
	build := func(tags ...uint16) []byte {
		buf := new(bytes.Buffer)
		buf.Write([]byte("II"))
		_ = binary.Write(buf, bo, uint16(0x002A))
		_ = binary.Write(buf, bo, uint32(8))
		_ = binary.Write(buf, bo, uint16(len(tags)))
		for i, tag := range tags {
			_ = binary.Write(buf, bo, tag)
			_ = binary.Write(buf, bo, uint16(4))        // LONG
			_ = binary.Write(buf, bo, uint32(1))        // count
			_ = binary.Write(buf, bo, uint32(0xA000+i)) // a recognizable value
		}
		_ = binary.Write(buf, bo, uint32(nextIFD))
		return buf.Bytes()
	}

	t.Run("survivors keep their order, values and next-IFD pointer", func(t *testing.T) {
		tiff := build(0x0100, 0x010F, 0x0101, 0x0110)

		removeIFDEntries(tiff, bo, 8, dropped)

		if got := bo.Uint16(tiff[8:10]); got != 2 {
			t.Fatalf("entry count = %d, want 2 after removing two of four", got)
		}
		for i, want := range []struct {
			tag   uint16
			value uint32
		}{{0x010F, 0xA001}, {0x0110, 0xA003}} {
			entry := 10 + i*12
			if tag := bo.Uint16(tiff[entry : entry+2]); tag != want.tag {
				t.Errorf("entry %d tag = %#04x, want %#04x", i, tag, want.tag)
			}
			if value := bo.Uint32(tiff[entry+8 : entry+12]); value != want.value {
				t.Errorf("entry %d value = %#x, want %#x - the entry moved but its contents must not", i, value, want.value)
			}
		}
		if got := bo.Uint32(tiff[10+2*12 : 10+2*12+4]); got != nextIFD {
			t.Errorf("next-IFD pointer at its new position = %#x, want %#x", got, nextIFD)
		}
	})

	t.Run("an IFD with nothing to remove is untouched", func(t *testing.T) {
		tiff := build(0x010F, 0x0110)
		before := append([]byte(nil), tiff...)

		removeIFDEntries(tiff, bo, 8, dropped)

		if !bytes.Equal(tiff, before) {
			t.Error("an IFD carrying none of the tags was rewritten anyway")
		}
	})

	t.Run("a truncated IFD is left alone rather than panicking", func(t *testing.T) {
		full := build(0x0100, 0x010F, 0x0101, 0x0110)
		for _, cut := range []int{len(full) - 1, len(full) - 13, 12, 9} {
			tiff := append([]byte(nil), full[:cut]...)
			before := append([]byte(nil), tiff...)

			removeIFDEntries(tiff, bo, 8, dropped)

			if !bytes.Equal(tiff, before) {
				t.Errorf("a block truncated to %d bytes was rewritten, want it left exactly as it arrived", cut)
			}
		}
	})
}

// TestNormalizeSavedExif_CorrectionSurvivesAMalformedBlock is the same
// failure mode one level up, where a real file's damage would arrive: a
// segment this walk cannot make sense of comes back out of an export byte
// for byte, rather than half-rewritten or panicking mid-splice. It binds
// the patching path as much as the removal one - an entry that happens to
// sit inside a truncated IFD must not be corrected either.
func TestNormalizeSavedExif_CorrectionSurvivesAMalformedBlock(t *testing.T) {
	full := buildDimensionExifTIFF(t, dimensionExif{width: 900, height: 600, orientation: 1, makerNote: []byte("MN")})

	for _, tc := range []struct {
		name string
		tiff []byte
	}{
		{"not a TIFF at all", []byte("this is not a TIFF payload at all")},
		{"header only", full[:8]},
		{"cut inside IFD0's entries", full[:20]},
		{"cut between IFD0 and the Exif SubIFD", full[:60]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			app1 := wrapAsAPP1(append([]byte("Exif\x00\x00"), tc.tiff...))
			before := append([]byte(nil), app1...)

			got := normalizeSavedExif(app1, image.Pt(600, 900))

			if !bytes.Equal(app1, before) {
				t.Error("normalizeSavedExif mutated the input segment")
			}
			if !bytes.Equal(got, before) {
				t.Error("a block this walk cannot make sense of came back rewritten, want it passed through untouched")
			}
		})
	}
}

// --- correcting dimension tags rather than dropping them --------------------

// buildIFD0TIFF builds a little-endian TIFF whose IFD0 holds exactly the
// given inline entries and nothing else - enough to exercise
// patchSavedTIFF's value rules directly, without the four-IFD camera
// fixture above.
func buildIFD0TIFF(t *testing.T, entries ...tiffEntry) []byte {
	t.Helper()

	bo := binary.LittleEndian
	buf := new(bytes.Buffer)
	buf.WriteString("II")
	_ = binary.Write(buf, bo, uint16(0x002A))
	_ = binary.Write(buf, bo, uint32(tiffHeaderBytes))
	_ = binary.Write(buf, bo, uint16(len(entries)))
	for _, e := range entries {
		_ = binary.Write(buf, bo, e.tag)
		_ = binary.Write(buf, bo, e.typ)
		_ = binary.Write(buf, bo, e.count)
		_ = binary.Write(buf, bo, e.value)
	}
	_ = binary.Write(buf, bo, uint32(0)) // next-IFD pointer

	return buf.Bytes()
}

// ifd0Entry is one entry read back out of a patched TIFF: its declared type
// and its raw value bytes, so a test can assert that a patch corrected the
// number without quietly changing the type it is stored as.
type ifd0Entry struct {
	typ uint16
	val []byte
}

func readIFD0Entries(t *testing.T, tiff []byte) map[uint16]ifd0Entry {
	t.Helper()

	bo, ok := tiffOrder(tiff)
	if !ok {
		t.Fatal("fixture is not a TIFF")
	}

	entries := map[uint16]ifd0Entry{}
	walkIFD(tiff, bo, bo.Uint32(tiff[4:8]), func(tag, typ uint16, val []byte) {
		entries[tag] = ifd0Entry{typ: typ, val: append([]byte(nil), val...)}
	})

	return entries
}

// TestPatchSavedTIFF_CorrectsDimensionTagsKeepingTheirType is the heart of
// correcting rather than dropping: the number becomes the frame that was
// actually written, and the entry stays the type it was declared as, so a
// reader parsing by type is not handed four bytes where it expects two.
func TestPatchSavedTIFF_CorrectsDimensionTagsKeepingTheirType(t *testing.T) {
	const shortType, longType = 3, 4

	for _, tc := range []struct {
		name string
		typ  uint16
	}{
		{"LONG entries", longType},
		{"SHORT entries", shortType},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tiff := buildIFD0TIFF(t,
				tiffEntry{tag: 0x0100, typ: tc.typ, count: 1, value: 900},
				tiffEntry{tag: 0x0101, typ: tc.typ, count: 1, value: 600},
				tiffEntry{tag: 0x010F, typ: longType, count: 1, value: 42}, // Make-ish: untouched
			)

			patchSavedTIFF(tiff, image.Pt(600, 900))

			entries := readIFD0Entries(t, tiff)
			for _, want := range []struct {
				tag   uint16
				value int
			}{{0x0100, 600}, {0x0101, 900}} {
				e, present := entries[want.tag]
				if !present {
					t.Fatalf("tag %#04x was dropped, want it corrected to %d", want.tag, want.value)
				}
				if got := tagValue(t, e.val); got != want.value {
					t.Errorf("tag %#04x reads %d, want the written frame's %d", want.tag, got, want.value)
				}
				if e.typ != tc.typ {
					t.Errorf("tag %#04x is stored as type %d, want its declared type %d left alone", want.tag, e.typ, tc.typ)
				}
			}
			if _, present := entries[0x010F]; !present {
				t.Error("a tag that is not a dimension was touched")
			}
		})
	}
}

// TestPatchSavedTIFF_RemovesADimensionTagItCannotCorrect is the safety
// property the whole change rests on: where the truth will not fit the
// entry as declared, the entry goes rather than being left holding a
// number that is now simply wrong.
func TestPatchSavedTIFF_RemovesADimensionTagItCannotCorrect(t *testing.T) {
	for _, tc := range []struct {
		name  string
		entry tiffEntry
		size  image.Point
	}{
		{
			"a SHORT too small for the new value",
			tiffEntry{tag: 0x0100, typ: 3, count: 1, value: 900},
			image.Pt(70000, 600), // over a SHORT's 65535 ceiling
		},
		{
			"a type this patcher will not write",
			tiffEntry{tag: 0x0100, typ: 5, count: 1, value: 900}, // RATIONAL
			image.Pt(600, 900),
		},
		{
			"a count other than one",
			tiffEntry{tag: 0x0100, typ: 4, count: 2, value: 900},
			image.Pt(600, 900),
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			tiff := buildIFD0TIFF(t, tc.entry,
				tiffEntry{tag: 0x0101, typ: 4, count: 1, value: 600},
			)

			patchSavedTIFF(tiff, tc.size)

			if _, present := readIFD0Entries(t, tiff)[0x0100]; present {
				t.Error("tag 0x0100 survived a patch it could not honestly hold, want it removed instead")
			}
		})
	}
}

// TestPatchSavedTIFF_ZeroSizeLeavesDimensionTagsAlone covers the value that
// says "the tags still describe this file": a written frame is never 0x0,
// so the zero point is unambiguous, and SaveRotated passes exactly that.
func TestPatchSavedTIFF_ZeroSizeLeavesDimensionTagsAlone(t *testing.T) {
	tiff := buildIFD0TIFF(t,
		tiffEntry{tag: 0x0100, typ: 4, count: 1, value: 900},
		tiffEntry{tag: 0x0101, typ: 4, count: 1, value: 600},
	)

	patchSavedTIFF(tiff, image.Point{})

	entries := readIFD0Entries(t, tiff)
	for _, want := range []struct {
		tag   uint16
		value int
	}{{0x0100, 900}, {0x0101, 600}} {
		e, present := entries[want.tag]
		if !present {
			t.Fatalf("tag %#04x was dropped by a no-op patch", want.tag)
		}
		if got := tagValue(t, e.val); got != want.value {
			t.Errorf("tag %#04x reads %d, want it left at %d", want.tag, got, want.value)
		}
	}
}

// TestPatchSavedTIFF_CorrectsEveryCopyOfARepeatedTag covers a file that
// breaks the rule that a tag appears once per IFD. Correcting only the
// first copy would leave the second holding the old size and report success,
// so nothing would remove it either - the one outcome this whole design
// exists to make impossible.
func TestPatchSavedTIFF_CorrectsEveryCopyOfARepeatedTag(t *testing.T) {
	t.Run("both copies corrected", func(t *testing.T) {
		tiff := buildIFD0TIFF(t,
			tiffEntry{tag: 0x0100, typ: 4, count: 1, value: 900},
			tiffEntry{tag: 0x0100, typ: 4, count: 1, value: 900},
		)

		patchSavedTIFF(tiff, image.Pt(600, 900))

		// readIFD0Entries keys by tag, so the entry it reports is the last
		// one in the IFD - exactly the copy a first-match-only patch skips.
		e, present := readIFD0Entries(t, tiff)[0x0100]
		if !present {
			t.Fatal("tag 0x0100 disappeared entirely")
		}
		if got := tagValue(t, e.val); got != 600 {
			t.Errorf("the last copy of 0x0100 reads %d, want the written frame's 600", got)
		}
	})

	t.Run("one bad copy condemns them all", func(t *testing.T) {
		tiff := buildIFD0TIFF(t,
			tiffEntry{tag: 0x0100, typ: 4, count: 1, value: 900},
			tiffEntry{tag: 0x0100, typ: 5, count: 1, value: 900}, // RATIONAL: uncorrectable
		)

		patchSavedTIFF(tiff, image.Pt(600, 900))

		if _, present := readIFD0Entries(t, tiff)[0x0100]; present {
			t.Error("a copy of 0x0100 survived that could not be corrected, want every copy removed")
		}
	})
}
