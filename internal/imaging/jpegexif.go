package imaging

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/jpeg"
	"io"
	"math"
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

// Dimension tags: the closed, deliberately narrow set that a written frame
// of a different shape turns into a false claim about the file. Each one
// states a size, so each one has a true value - the frame actually being
// written - and is corrected to it rather than dropped. What cannot be
// corrected is removed instead (see patchIFDDimension): a tag holding a
// number that is simply wrong is worse than an absent one, and absence
// still leaves a reader the JPEG frame header, which carries the true size
// for free.
//
// MakerNote and the resolution/DPI tags are explicitly *not* here.
// MakerNote may contain pixel geometry but cannot be audited, and removing
// it on suspicion would discard the lens and body detail a user asked to
// keep; DPI states intended print density rather than a pixel count, and
// dropping it would land photos at a default density in layout software.
// dimensionTag pairs a tag with which edge of the written frame it states.
// Carrying the axis on the tag rather than inferring it from position keeps
// a list and its meaning from drifting apart: adding a tag forces whoever
// adds it to say which edge it claims, instead of relying on every list
// staying in width-then-height order forever.
type dimensionTag struct {
	tag uint16

	// vertical marks a tag that states the frame's height; the zero value
	// states its width, which is the first of every pair below.
	vertical bool
}

// edge is the value tag should now hold, given the frame being written.
func (d dimensionTag) edge(corrected image.Point) int {
	if d.vertical {
		return corrected.Y
	}

	return corrected.X
}

var (
	ifd0DimensionTags = []dimensionTag{
		{tag: 0x0100},                 // ImageWidth
		{tag: 0x0101, vertical: true}, // ImageLength
	}
	exifDimensionTags = []dimensionTag{
		{tag: 0xA002},                 // PixelXDimension
		{tag: 0xA003, vertical: true}, // PixelYDimension
	}
	interopDimensionTags = []dimensionTag{
		{tag: 0x1001},                 // RelatedImageWidth
		{tag: 0x1002, vertical: true}, // RelatedImageLength
	}

	// exifCoordinateTags are the two the same invalidation reaches that a
	// corrected width and height cannot repair: they are positions *inside*
	// the frame, not sizes of it, and rotating or scaling one needs the
	// transform that produced the new frame - which this module never sees,
	// since it infers that something changed from the pixels rather than
	// being told what. They are removed, as all eight used to be.
	exifCoordinateTags = []uint16{
		0x9214, // SubjectArea
		0xA214, // SubjectLocation
	}
)

// interopIFDPointer (0xA005) locates the Interoperability IFD - and lives
// in the Exif SubIFD, not in IFD0, which is the one hop in this walk that
// is easy to get wrong.
const interopIFDPointer = 0xA005

// normalizeSavedExif returns a copy of app1 (a full FF E1 Exif segment)
// with IFD0 Orientation (tag 0x0112) set to 1 when that tag is present
// as a SHORT, and with IFD0's next-IFD pointer zeroed so a thumbnail
// IFD1 is no longer linked. corrected is the size of the frame being
// written when that frame no longer matches what the source's dimension
// tags describe, and the zero value when it still does - a written frame is
// never 0x0, so nothing else has to be passed to say "leave them alone". If
// app1 is not a well-formed Exif APP1, it is returned copied and unchanged.
func normalizeSavedExif(app1 []byte, corrected image.Point) []byte {
	out := make([]byte, len(app1))
	copy(out, app1)

	if !isExifAPP1(app1) {
		return out
	}

	patchSavedTIFF(out[10:], corrected)
	return out
}

// patchSavedTIFF rewrites tiff (an Exif APP1 payload's TIFF portion, i.e.
// app1[10:]) in place: it forces IFD0's Orientation entry to 1, zeroes
// IFD0's next-IFD pointer so a thumbnail IFD1 is unlinked, and - when
// corrected is not the zero value - rewrites the dimension tags across
// IFD0, the Exif SubIFD and the Interoperability IFD to that size,
// removing the ones it cannot rewrite honestly along with the two
// coordinate tags no size can repair. It does not follow the next-IFD
// pointer or compact the freed bytes. Every offset is bounds checked
// against len(tiff); any failure to locate IFD0 leaves tiff unchanged
// rather than panicking.
func patchSavedTIFF(tiff []byte, corrected image.Point) {
	bo, ok := tiffOrder(tiff)
	if !ok {
		return
	}

	ifd0Offset := uint64(bo.Uint32(tiff[4:8]))
	if ifd0Offset+2 > uint64(len(tiff)) {
		return
	}

	if corrected != (image.Point{}) {
		correctSavedDimensions(tiff, bo, ifd0Offset, corrected)
	}

	// Re-walked after the correction above: IFD0's entry count is exactly
	// what it may just have changed, and both the orientation entry and the
	// next-IFD pointer's position are measured from it.
	entries, nextIFDOffset, ok := ifdEntryOffsets(tiff, bo, ifd0Offset)
	if !ok {
		return
	}

	for _, entryOffset := range entries {
		if bo.Uint16(tiff[entryOffset:entryOffset+2]) != 0x0112 {
			continue
		}

		typ := bo.Uint16(tiff[entryOffset+2 : entryOffset+4])
		count := bo.Uint32(tiff[entryOffset+4 : entryOffset+8])
		if typ == 3 && count == 1 {
			bo.PutUint16(tiff[entryOffset+8:entryOffset+10], 1)
		}
	}

	bo.PutUint32(tiff[nextIFDOffset:nextIFDOffset+4], 0)
}

// correctSavedDimensions rewrites every dimension tag across IFD0, the Exif
// SubIFD and the Interoperability IFD to corrected, and removes the ones
// that cannot be rewritten honestly plus the two coordinate tags that no
// width and height can repair.
//
// Patching happens before any removal, because a removal shifts the entries
// after it and every patch below addresses an entry by walking to it. Both
// sub-IFD pointers are read first for the same reason, and because their
// values are absolute offsets from the TIFF start: they survive an entry
// shift untouched, but the entries *holding* them move.
func correctSavedDimensions(tiff []byte, bo binary.ByteOrder, ifd0Offset uint64, corrected image.Point) {
	exifOffset, hasExif := savedIFDPointer(tiff, bo, ifd0Offset, exifIFDPointer)
	interopOffset, hasInterop := uint64(0), false
	if hasExif {
		interopOffset, hasInterop = savedIFDPointer(tiff, bo, exifOffset, interopIFDPointer)
	}

	dropIFD0 := correctIFDDimensions(tiff, bo, ifd0Offset, ifd0DimensionTags, corrected)

	var dropExif, dropInterop []uint16
	if hasExif {
		dropExif = append(
			correctIFDDimensions(tiff, bo, exifOffset, exifDimensionTags, corrected),
			exifCoordinateTags...)
	}
	if hasInterop {
		dropInterop = correctIFDDimensions(tiff, bo, interopOffset, interopDimensionTags, corrected)
	}

	// The three removals are independent of each other: every offset above
	// was read before any of them, and an entry shift can only move entries
	// within one IFD, never the IFDs themselves, which are addressed
	// absolutely from the TIFF start. The order is just the order they were
	// walked in.
	if hasInterop {
		removeIFDEntries(tiff, bo, interopOffset, dropInterop)
	}
	if hasExif {
		removeIFDEntries(tiff, bo, exifOffset, dropExif)
	}
	removeIFDEntries(tiff, bo, ifd0Offset, dropIFD0)
}

// correctIFDDimensions rewrites each of tags in the IFD at ifdOffset to the
// edge of corrected it states, and returns the ones that could not be
// rewritten honestly, for the caller to remove instead.
func correctIFDDimensions(tiff []byte, bo binary.ByteOrder, ifdOffset uint64, tags []dimensionTag, corrected image.Point) []uint16 {
	var failed []uint16
	for _, t := range tags {
		if !patchIFDDimension(tiff, bo, ifdOffset, t.tag, t.edge(corrected)) {
			failed = append(failed, t.tag)
		}
	}

	return failed
}

// patchIFDDimension overwrites the inline value of tag in the IFD at
// ifdOffset with v, honouring the entry's declared type so a reader parsing
// by type is not handed four bytes where it expects two.
//
// It reports false only when the entry is there and cannot hold v honestly
// - a type this does not write, a count other than 1, or a SHORT that v
// overflows - so the caller removes it rather than leaving a wrong number
// behind. A tag that is simply absent reports true: there is nothing to
// correct and nothing to remove.
func patchIFDDimension(tiff []byte, bo binary.ByteOrder, ifdOffset uint64, tag uint16, v int) bool {
	// A truncated IFD is reported as success so the caller does not try to
	// remove from it either - see ifdEntryOffsets for why it is left whole.
	entries, _, ok := ifdEntryOffsets(tiff, bo, ifdOffset)
	if !ok {
		return true
	}

	corrected := true
	for _, entryOffset := range entries {
		if bo.Uint16(tiff[entryOffset:entryOffset+2]) != tag {
			continue
		}

		// Every match, not the first: a well-formed IFD carries a tag once,
		// but a file that repeats one would otherwise keep the later copy
		// at the old size, reported as a success so nothing removed it -
		// exactly the wrong number this is here to prevent. One failure
		// condemns them all, and removeIFDEntries takes every copy.
		if !correctInlineDimension(tiff, bo, entryOffset, v) {
			corrected = false
		}
	}

	return corrected
}

// correctInlineDimension overwrites the single inline value of the entry at
// entryOffset with v, in the type the entry declares. It reports false for
// anything it will not write honestly: a count other than 1, a type that is
// neither SHORT nor LONG, or a value that does not fit the one declared -
// including a non-positive v, which is no size at all.
//
// Both types hold that value inside the entry's own four bytes, so
// correcting one never touches the value area or any offset pointing into
// it.
func correctInlineDimension(tiff []byte, bo binary.ByteOrder, entryOffset uint64, v int) bool {
	if count := bo.Uint32(tiff[entryOffset+4 : entryOffset+8]); count != 1 {
		return false
	}

	switch typ := bo.Uint16(tiff[entryOffset+2 : entryOffset+4]); {
	case typ == 3 && v > 0 && v <= math.MaxUint16: // SHORT
		bo.PutUint16(tiff[entryOffset+8:entryOffset+10], uint16(v))
	case typ == 4 && v > 0 && uint64(v) <= math.MaxUint32: // LONG
		bo.PutUint32(tiff[entryOffset+8:entryOffset+12], uint32(v))
	default:
		return false
	}

	return true
}

// ifdEntryOffsets is where each entry of the IFD at ifdOffset begins, plus
// the offset of the next-IFD pointer that follows them. ok is false for an
// IFD whose declared entries do not all fit inside tiff.
//
// That all-or-nothing rule is the one every writer here shares: a block
// this cannot see the whole of is left exactly as it arrived, because an
// entry that happens to fit inside a truncated IFD is not evidence that it
// is the entry it claims to be. Deliberately its own walk rather than
// walkIFD, which reports values but not the entry offsets a writer needs.
func ifdEntryOffsets(tiff []byte, bo binary.ByteOrder, ifdOffset uint64) (entries []uint64, nextIFD uint64, ok bool) {
	if ifdOffset+2 > uint64(len(tiff)) {
		return nil, 0, false
	}

	numEntries := uint64(bo.Uint16(tiff[ifdOffset : ifdOffset+2]))
	entriesStart := ifdOffset + 2

	nextIFD = entriesStart + numEntries*12
	if nextIFD+4 > uint64(len(tiff)) {
		return nil, 0, false
	}

	entries = make([]uint64, 0, numEntries)
	for i := range numEntries {
		entries = append(entries, entriesStart+i*12)
	}

	return entries, nextIFD, true
}

// savedIFDPointer is the offset a sub-IFD pointer entry (tag, a LONG or
// IFD holding one absolute offset) points at, or ok=false when the IFD at
// ifdOffset has no such entry, or one whose target lies outside tiff.
func savedIFDPointer(tiff []byte, bo binary.ByteOrder, ifdOffset uint64, tag uint16) (uint64, bool) {
	entries, _, ok := ifdEntryOffsets(tiff, bo, ifdOffset)
	if !ok {
		return 0, false
	}

	for _, entryOffset := range entries {
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
	entries, nextIFDOffset, ok := ifdEntryOffsets(tiff, bo, ifdOffset)
	if !ok {
		return
	}

	entriesStart := ifdOffset + 2
	nextIFD := bo.Uint32(tiff[nextIFDOffset : nextIFDOffset+4])

	kept := uint64(0)
	for _, entryOffset := range entries {
		if slices.Contains(tags, bo.Uint16(tiff[entryOffset:entryOffset+2])) {
			continue
		}

		if dst := entriesStart + kept*12; dst != entryOffset {
			copy(tiff[dst:dst+12], tiff[entryOffset:entryOffset+12])
		}
		kept++
	}
	if kept == uint64(len(entries)) {
		return
	}

	bo.PutUint16(tiff[ifdOffset:ifdOffset+2], uint16(kept))
	newNext := entriesStart + kept*12
	bo.PutUint32(tiff[newNext:newNext+4], nextIFD)
}

// encodeJPEGPreservingMetadata encodes img at jpegSaveQuality, then
// splices orig's metadata segments after SOI. Exif APP1 segments are
// passed through normalizeSavedExif first, carrying corrected - the size
// img is being written at when that is no longer the size orig's tags
// describe, and the zero value when it still is. A non-JPEG orig, or a
// JPEG with no metadata, encodes exactly as encodeJPEGForSave would.
func encodeJPEGPreservingMetadata(w io.Writer, img image.Image, orig []byte, corrected image.Point) error {
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
			segs[i] = normalizeSavedExif(s, corrected)
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
