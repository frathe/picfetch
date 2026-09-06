package imaging

// walkJPEGSegments calls fn for each payload-bearing marker between SOI and
// SOS. No-payload markers (TEM 0x01, RST0–RST7 0xD0–0xD7, SOI 0xD8, EOI 0xD9)
// are skipped without a callback. SOS (0xDA) and any malformed structure
// (a non-0xFF where a marker is required, a length field < 2, or a segment
// that would run past len(data)) stop the walk without a callback.
//
// payload is the bytes after the 2-byte length (data[pos+4:pos+2+segLen]),
// not including the 0xFF marker. It aliases data — callers that retain it
// must copy. Returning false from fn stops the walk; true continues.
// Non-JPEG data (missing FF D8) is a no-op.
//
// This is a literal extract of the four header walks in jpegEXIFOrientation,
// jpegMetadata, jpegMetadataSegments, and jpegHasRemovableMetadata. It does
// not skip fill 0xFF bytes and does not walk the entropy-coded scan — those
// are jpegLength / stripJPEGSegments, which stay separate.
func walkJPEGSegments(data []byte, fn func(marker byte, payload []byte) bool) {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return
	}

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

		if marker == 0xDA {
			break
		}

		segLen := int(data[pos+2])<<8 | int(data[pos+3])
		if segLen < 2 || pos+2+segLen > len(data) {
			break
		}

		segStart := pos + 4
		segEnd := pos + 2 + segLen
		if !fn(marker, data[segStart:segEnd]) {
			return
		}
		pos = segEnd
	}
}

// jpegFrameSize is the pixel size recorded in data's own frame header - the
// size the file's Exif dimension tags describe. ok is false for data that is
// not a JPEG, or whose frame header cannot be read.
func jpegFrameSize(data []byte) (w, h int, ok bool) {
	walkJPEGSegments(data, func(marker byte, payload []byte) bool {
		if marker < 0xC0 || marker > 0xCF || marker == 0xC4 || marker == 0xC8 || marker == 0xCC {
			return true
		}

		// Found the frame header: stop the walk either way. A short or
		// zero-sized payload here means ok stays false rather than us
		// guessing from a later, unrelated marker.
		if len(payload) < 5 {
			return false
		}

		height := int(payload[1])<<8 | int(payload[2])
		width := int(payload[3])<<8 | int(payload[4])
		if width == 0 || height == 0 {
			return false
		}

		w, h, ok = width, height, true
		return false
	})
	return w, h, ok
}

// jpegSegmentBytes returns a standalone on-disk COM/APPn segment: 0xFF,
// marker, 2-byte big-endian length (len(payload)+2), then payload. The
// result is a copy, so the caller may mutate it.
func jpegSegmentBytes(marker byte, payload []byte) []byte {
	n := len(payload) + 2
	seg := make([]byte, 4+len(payload))
	seg[0] = 0xFF
	seg[1] = marker
	seg[2] = byte(n >> 8)
	seg[3] = byte(n)
	copy(seg[4:], payload)
	return seg
}
