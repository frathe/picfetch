package imaging

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"slices"
)

var errNotJPEG = errors.New("not a JPEG")

// jpegMetadataSegments returns a copy of every COM and APPn segment between
// SOI and SOS, in file order, excluding APP0 (JFIF/JFXX) and APP2 segments
// whose payload starts with "MPF\x00". Each slice includes the 0xFF marker,
// the 2-byte length, and the payload. Later re-encoding can copy these
// segments verbatim into a freshly written JPEG to preserve metadata a
// bare image/jpeg.Encode call would otherwise drop. data that is not a
// JPEG yields nil.
func jpegMetadataSegments(data []byte) [][]byte {
	var segs [][]byte
	walkJPEGSegments(data, func(marker byte, payload []byte) bool {
		if marker == 0xFE || (marker >= 0xE0 && marker <= 0xEF) {
			if !skipSegment(marker, payload) {
				segs = append(segs, jpegSegmentBytes(marker, payload))
			}
		}
		return true
	})
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

// keepOnStrip reports whether a COM/APPn segment should survive
// stripJPEGSegments. APP0 (JFIF/JFXX) and APP14 (Adobe color transform)
// stay; ICC APP2 stays (appearance). Everything else removable — Exif,
// XMP, IPTC, COM, MPF — is dropped.
func keepOnStrip(marker byte, payload []byte) bool {
	if marker == 0xE0 { // APP0
		return true
	}
	if marker == 0xEE { // APP14 Adobe
		return true
	}
	if marker == 0xE2 && len(payload) >= 12 && string(payload[:12]) == "ICC_PROFILE\x00" {
		return true
	}
	return false
}

// jpegHasRemovableMetadata reports whether stripJPEGSegments would drop
// at least one COM/APPn segment or bytes after the primary EOI.
// Non-JPEG data is false.
func jpegHasRemovableMetadata(data []byte) bool {
	if n := jpegLength(data); n > 0 && n < len(data) {
		return true
	}
	found := false
	walkJPEGSegments(data, func(marker byte, payload []byte) bool {
		if marker == 0xFE || (marker >= 0xE0 && marker <= 0xEF) {
			if !keepOnStrip(marker, payload) {
				found = true
				return false
			}
		}
		return true
	})
	return found
}

// stripJPEGSegments returns a copy of a JPEG with removable metadata
// segments (see keepOnStrip) omitted. DQT/DHT/SOF/DRI and the entropy-
// coded scan through the primary EOI are copied verbatim; bytes after
// that EOI (MPF extra pictures, motion-photo video) are dropped. data
// that is not a JPEG yields errNotJPEG. A JPEG with nothing removable
// returns a copy of data.
func stripJPEGSegments(data []byte) ([]byte, error) {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return nil, errNotJPEG
	}

	out := make([]byte, 0, len(data))
	out = append(out, 0xFF, 0xD8)
	pos := 2

	for pos+4 <= len(data) {
		if data[pos] != 0xFF {
			return nil, errNotJPEG
		}
		marker := data[pos+1]
		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD9) {
			out = append(out, data[pos:pos+2]...)
			pos += 2
			continue
		}
		if marker == 0xDA {
			end := jpegLength(data)
			if end > pos && end <= len(data) {
				out = append(out, data[pos:end]...)
			} else {
				out = append(out, data[pos:]...)
			}
			return out, nil
		}
		segLen := int(data[pos+2])<<8 | int(data[pos+3])
		if segLen < 2 || pos+2+segLen > len(data) {
			return nil, errNotJPEG
		}
		segEnd := pos + 2 + segLen
		if marker == 0xFE || (marker >= 0xE0 && marker <= 0xEF) {
			if !keepOnStrip(marker, data[pos+4:segEnd]) {
				pos = segEnd
				continue
			}
		}
		out = append(out, data[pos:segEnd]...)
		pos = segEnd
	}
	return nil, errNotJPEG
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

// Dimension tags: the closed, deliberately narrow set a resize turns into
// a false claim about the file. They are removed rather than rewritten to
// the new values - a reader then falls back to the JPEG frame header, which
// carries the true size for free, instead of trusting a second source of
// truth this app would have to keep correct forever.
//
// MakerNote and the resolution/DPI tags are explicitly *not* here.
// MakerNote may contain pixel geometry but cannot be audited, and removing
// it on suspicion would discard the lens and body detail a user asked to
// keep; DPI states intended print density rather than a pixel count, and
// dropping it would land photos at a default density in layout software.
var (
	ifd0DimensionTags = []uint16{
		0x0100, // ImageWidth
		0x0101, // ImageLength
	}
	exifDimensionTags = []uint16{
		0x9214, // SubjectArea
		0xA002, // PixelXDimension
		0xA003, // PixelYDimension
		0xA214, // SubjectLocation
	}
	interopDimensionTags = []uint16{
		0x1001, // RelatedImageWidth
		0x1002, // RelatedImageLength
	}
)

// interopIFDPointer (0xA005) locates the Interoperability IFD - and lives
// in the Exif SubIFD, not in IFD0, which is the one hop in this walk that
// is easy to get wrong.
const interopIFDPointer = 0xA005

// normalizeSavedExif returns a copy of app1 (a full FF E1 Exif segment)
// with IFD0 Orientation (tag 0x0112) set to 1 when that tag is present
// as a SHORT, and with IFD0's next-IFD pointer zeroed so a thumbnail
// IFD1 is no longer linked. When dropDimensions is true - the export path,
// and only when a size limit actually changed the pixels - the tags listed
// above are removed as well. If app1 is not a well-formed Exif APP1, it
// is returned copied and unchanged.
func normalizeSavedExif(app1 []byte, dropDimensions bool) []byte {
	out := make([]byte, len(app1))
	copy(out, app1)

	if !isExifAPP1(app1) {
		return out
	}

	patchSavedTIFF(out[10:], dropDimensions)
	return out
}

// patchSavedTIFF rewrites tiff (an Exif APP1 payload's TIFF portion, i.e.
// app1[10:]) in place: it forces IFD0's Orientation entry to 1, zeroes
// IFD0's next-IFD pointer so a thumbnail IFD1 is unlinked, and - when
// dropDimensions is set - removes the dimension tags from IFD0, the Exif
// SubIFD and the Interoperability IFD. It does not follow the next-IFD
// pointer or compact the freed bytes. Every offset is bounds checked
// against len(tiff); any failure to locate IFD0 leaves tiff unchanged
// rather than panicking.
func patchSavedTIFF(tiff []byte, dropDimensions bool) {
	bo, ok := tiffOrder(tiff)
	if !ok {
		return
	}

	ifd0Offset := uint64(bo.Uint32(tiff[4:8]))
	if ifd0Offset+2 > uint64(len(tiff)) {
		return
	}

	if dropDimensions {
		// Both pointers are read before anything is removed. Their values
		// are absolute offsets from the TIFF start and so survive any
		// entry shift untouched - but the *entries holding them* move, so
		// reading them afterwards would mean chasing them again.
		if exifOffset, ok := savedIFDPointer(tiff, bo, ifd0Offset, exifIFDPointer); ok {
			if interopOffset, ok := savedIFDPointer(tiff, bo, exifOffset, interopIFDPointer); ok {
				removeIFDEntries(tiff, bo, interopOffset, interopDimensionTags)
			}
			removeIFDEntries(tiff, bo, exifOffset, exifDimensionTags)
		}
		removeIFDEntries(tiff, bo, ifd0Offset, ifd0DimensionTags)
	}

	// Re-read after the removal above: IFD0's entry count is exactly what
	// it may just have changed, and both the orientation walk and the
	// next-IFD pointer's position are measured from it.
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

// savedIFDPointer is the offset a sub-IFD pointer entry (tag, a LONG or
// IFD holding one absolute offset) points at, or ok=false when the IFD at
// ifdOffset has no such entry, or one whose target lies outside tiff.
// Deliberately its own small walk rather than walkIFD, which reports values
// but not the entry offsets the removal below needs to reason about.
func savedIFDPointer(tiff []byte, bo binary.ByteOrder, ifdOffset uint64, tag uint16) (uint64, bool) {
	if ifdOffset+2 > uint64(len(tiff)) {
		return 0, false
	}

	numEntries := uint64(bo.Uint16(tiff[ifdOffset : ifdOffset+2]))
	entriesStart := ifdOffset + 2

	for i := range numEntries {
		entryOffset := entriesStart + i*12
		if entryOffset+12 > uint64(len(tiff)) {
			break
		}
		if bo.Uint16(tiff[entryOffset:entryOffset+2]) != tag {
			continue
		}

		typ := bo.Uint16(tiff[entryOffset+2 : entryOffset+4])
		if typ != 4 && typ != 13 { // LONG, IFD
			return 0, false
		}
		target := uint64(bo.Uint32(tiff[entryOffset+8 : entryOffset+12]))
		if target+2 > uint64(len(tiff)) {
			return 0, false
		}

		return target, true
	}

	return 0, false
}

// removeIFDEntries deletes every entry carrying one of tags from the IFD at
// ifdOffset, in place: the surviving entries move back to close the gap,
// the entry count drops by however many went, and the next-IFD pointer is
// rewritten at its new position. The bytes freed off the end are left dead,
// consistent with patchSavedTIFF's own choice not to compact.
//
// Only the 12-byte entry records move. An entry's value either lives inside
// those 12 bytes and travels with it, or lives in the value area at an
// offset absolute from the TIFF start - which no amount of shifting entries
// can invalidate. That is what makes removal safe without rewriting a
// single other offset in the file.
//
// An IFD whose declared entries do not fit inside tiff is left completely
// alone rather than partially rewritten: a truncated or hostile block
// should come out of an export exactly as it went in, which is the same
// failure mode walkIFD chose for reading.
func removeIFDEntries(tiff []byte, bo binary.ByteOrder, ifdOffset uint64, tags []uint16) {
	if ifdOffset+2 > uint64(len(tiff)) {
		return
	}

	numEntries := uint64(bo.Uint16(tiff[ifdOffset : ifdOffset+2]))
	entriesStart := ifdOffset + 2
	nextIFDOffset := entriesStart + numEntries*12
	if nextIFDOffset+4 > uint64(len(tiff)) {
		return
	}
	nextIFD := bo.Uint32(tiff[nextIFDOffset : nextIFDOffset+4])

	kept := uint64(0)
	for i := range numEntries {
		entryOffset := entriesStart + i*12
		if slices.Contains(tags, bo.Uint16(tiff[entryOffset:entryOffset+2])) {
			continue
		}

		if dst := entriesStart + kept*12; dst != entryOffset {
			copy(tiff[dst:dst+12], tiff[entryOffset:entryOffset+12])
		}
		kept++
	}
	if kept == numEntries {
		return
	}

	bo.PutUint16(tiff[ifdOffset:ifdOffset+2], uint16(kept))
	newNext := entriesStart + kept*12
	bo.PutUint32(tiff[newNext:newNext+4], nextIFD)
}

// encodeJPEGPreservingMetadata encodes img at jpegSaveQuality, then
// splices orig's metadata segments after SOI. Exif APP1 segments are
// passed through normalizeSavedExif first, with dropDimensions set when
// img is no longer the size orig's tags describe. A non-JPEG orig, or a
// JPEG with no metadata, encodes exactly as encodeJPEGForSave would.
func encodeJPEGPreservingMetadata(w io.Writer, img image.Image, orig []byte, dropDimensions bool) error {
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
			segs[i] = normalizeSavedExif(s, dropDimensions)
		}
	}
	out, err := injectJPEGMetadata(encoded, segs)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}

// isICCAPP2 reports whether seg is a full APP2 whose payload starts with
// "ICC_PROFILE\x00".
func isICCAPP2(seg []byte) bool {
	if len(seg) < 16 || seg[0] != 0xFF || seg[1] != 0xE2 {
		return false
	}
	return string(seg[4:16]) == "ICC_PROFILE\x00"
}

// jpegICCSegments returns the ICC APP2 segments jpegMetadataSegments
// collected from data, in file order. Used by the orientation 2–8 strip
// path, which must re-encode pixels and therefore cannot keep APP14 (that
// marker describes the original entropy-coded color transform, not
// image/jpeg.Encode's output).
func jpegICCSegments(data []byte) [][]byte {
	var icc [][]byte
	for _, s := range jpegMetadataSegments(data) {
		if isICCAPP2(s) {
			icc = append(icc, s)
		}
	}
	return icc
}

// encodeJPEGKeepingICC encodes img at jpegSaveQuality, then splices orig's
// ICC APP2 segments after SOI. Identifying metadata is not copied. A
// non-JPEG orig, or a JPEG with no ICC, encodes exactly as
// encodeJPEGForSave would.
func encodeJPEGKeepingICC(w io.Writer, img image.Image, orig []byte) error {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: jpegSaveQuality}); err != nil {
		return err
	}
	encoded := buf.Bytes()
	segs := jpegICCSegments(orig)
	if len(segs) == 0 {
		_, err := w.Write(encoded)
		return err
	}
	out, err := injectJPEGMetadata(encoded, segs)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}
