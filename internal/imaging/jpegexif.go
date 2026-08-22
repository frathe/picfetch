package imaging

import (
	"bytes"
	"errors"
	"image"
	"image/jpeg"
	"io"
)

var errNotJPEG = errors.New("not a JPEG")

// jpegMetadataSegments returns a copy of every COM and APPn segment between
// SOI and SOS, in file order, excluding APP0 (JFIF/JFXX) and APP2 segments
// whose payload starts with "MPF\x00". Each slice includes the 0xFF marker,
// the 2-byte length, and the payload. Later re-encoding can copy these
// segments verbatim into a freshly written JPEG to preserve metadata a
// bare image/jpeg.Encode call would otherwise drop. data that is not a
// JPEG yields nil.
//
// This walks JPEG markers the same way jpegEXIFOrientation (exif.go) does,
// duplicated rather than shared: that walker returns an orientation int and
// stops at the first APP1 with a valid tag, while this one collects a
// different, filtered subset of segments as byte slices.
func jpegMetadataSegments(data []byte) [][]byte {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return nil
	}

	var segs [][]byte

	pos := 2

	for pos+4 <= len(data) {
		if data[pos] != 0xFF {
			break
		}

		marker := data[pos+1]

		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD9) {
			pos += 2
			continue
		}

		if marker == 0xDA { // start of scan: header segments are over
			break
		}

		segLen := int(data[pos+2])<<8 | int(data[pos+3])

		if segLen < 2 || pos+2+segLen > len(data) {
			break
		}

		segEnd := pos + 2 + segLen

		if marker == 0xFE || (marker >= 0xE0 && marker <= 0xEF) {
			if !skipSegment(marker, data[pos+4:segEnd]) {
				seg := make([]byte, segEnd-pos)
				copy(seg, data[pos:segEnd])
				segs = append(segs, seg)
			}
		}

		pos = segEnd
	}

	return segs
}

// skipSegment reports whether a COM/APPn segment with this marker and
// payload should be excluded from jpegMetadataSegments' result: APP0
// (JFIF/JFXX, which jpeg.Encode always writes itself) and MPF (APP2 whose
// payload starts with "MPF\x00" - multi-picture data tied to the exact
// bytes of the encoded frames, not portable metadata). ICC APP2
// ("ICC_PROFILE\x00...") is a different payload shape and is kept.
func skipSegment(marker byte, payload []byte) bool {
	if marker == 0xE0 {
		return true
	}
	if marker == 0xE2 && len(payload) >= 4 && string(payload[:4]) == "MPF\x00" {
		return true
	}
	return false
}

// isExifAPP1 reports whether seg is a full APP1 whose payload starts with
// "Exif\x00\x00".
func isExifAPP1(seg []byte) bool {
	if len(seg) < 10 || seg[0] != 0xFF || seg[1] != 0xE1 {
		return false
	}
	return string(seg[4:10]) == "Exif\x00\x00"
}

// injectJPEGMetadata returns a JPEG that is encoded with segs inserted
// immediately after the SOI marker, in slice order. encoded must start
// with FF D8. Empty segs returns a copy of encoded. Each seg must be a
// COM or APPn segment starting with 0xFF.
func injectJPEGMetadata(encoded []byte, segs [][]byte) ([]byte, error) {
	if len(encoded) < 2 || encoded[0] != 0xFF || encoded[1] != 0xD8 {
		return nil, errNotJPEG
	}
	extra := 0
	for _, s := range segs {
		if len(s) < 2 || s[0] != 0xFF {
			return nil, errNotJPEG
		}
		m := s[1]
		if m != 0xFE && (m < 0xE0 || m > 0xEF) {
			return nil, errNotJPEG
		}
		extra += len(s)
	}
	out := make([]byte, 0, 2+extra+len(encoded)-2)
	out = append(out, 0xFF, 0xD8)
	for _, s := range segs {
		out = append(out, s...)
	}
	out = append(out, encoded[2:]...)
	return out, nil
}

// normalizeSavedExif returns a copy of app1 (a full FF E1 Exif segment)
// with IFD0 Orientation (tag 0x0112) set to 1 when that tag is present
// as a SHORT, and with IFD0's next-IFD pointer zeroed so a thumbnail
// IFD1 is no longer linked. If app1 is not a well-formed Exif APP1, it
// is returned copied and unchanged.
func normalizeSavedExif(app1 []byte) []byte {
	out := make([]byte, len(app1))
	copy(out, app1)

	if !isExifAPP1(app1) {
		return out
	}

	patchSavedTIFF(out[10:])
	return out
}

// patchSavedTIFF rewrites tiff (an Exif APP1 payload's TIFF portion, i.e.
// app1[10:]) in place: it forces IFD0's Orientation entry to 1 and zeroes
// IFD0's next-IFD pointer so a thumbnail IFD1 is unlinked. It does not
// follow that pointer or compact the freed bytes. Every offset is bounds
// checked against len(tiff); any failure to locate IFD0 leaves tiff
// unchanged rather than panicking.
func patchSavedTIFF(tiff []byte) {
	bo, ok := tiffOrder(tiff)
	if !ok {
		return
	}

	ifd0Offset := uint64(bo.Uint32(tiff[4:8]))
	if ifd0Offset+2 > uint64(len(tiff)) {
		return
	}

	numEntries := uint64(bo.Uint16(tiff[ifd0Offset : ifd0Offset+2]))
	entriesStart := ifd0Offset + 2

	for i := range numEntries {
		entryOffset := entriesStart + i*12
		if entryOffset+12 > uint64(len(tiff)) {
			break
		}

		if bo.Uint16(tiff[entryOffset:entryOffset+2]) != 0x0112 {
			continue
		}

		typ := bo.Uint16(tiff[entryOffset+2 : entryOffset+4])
		count := bo.Uint32(tiff[entryOffset+4 : entryOffset+8])
		if typ == 3 && count == 1 {
			bo.PutUint16(tiff[entryOffset+8:entryOffset+10], 1)
		}
	}

	nextIFDOffset := entriesStart + numEntries*12
	if nextIFDOffset+4 <= uint64(len(tiff)) {
		bo.PutUint32(tiff[nextIFDOffset:nextIFDOffset+4], 0)
	}
}

// encodeJPEGPreservingMetadata encodes img at jpegSaveQuality, then
// splices orig's metadata segments after SOI. Exif APP1 segments are
// passed through normalizeSavedExif first. A non-JPEG orig, or a JPEG
// with no metadata, encodes exactly as encodeJPEGForSave would.
func encodeJPEGPreservingMetadata(w io.Writer, img image.Image, orig []byte) error {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegSaveQuality}); err != nil {
		return err
	}
	encoded := buf.Bytes()
	segs := jpegMetadataSegments(orig)
	if len(segs) == 0 {
		_, err := w.Write(encoded)
		return err
	}
	for i, s := range segs {
		if isExifAPP1(s) {
			segs[i] = normalizeSavedExif(s)
		}
	}
	out, err := injectJPEGMetadata(encoded, segs)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}
