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
