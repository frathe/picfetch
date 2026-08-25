package imaging

import (
	"encoding/binary"
	"strings"
)

// walkIFD calls fn once per readable entry in the IFD at ifdOffset within
// tiff. Entries with an unrecognized type, an implausible count, or a
// value/offset that doesn't fit inside tiff are silently skipped rather
// than reported - see ReadMetadata's comment on why that's the right
// failure mode here.
func walkIFD(tiff []byte, bo binary.ByteOrder, ifdOffset uint32, fn func(tag, typ uint16, val []byte)) {
	if ifdOffset+2 > uint32(len(tiff)) {
		return
	}

	numEntries := bo.Uint16(tiff[ifdOffset : ifdOffset+2])
	entriesStart := ifdOffset + 2

	for i := uint32(0); i < uint32(numEntries); i++ {
		entryOffset := entriesStart + i*12

		if entryOffset+12 > uint32(len(tiff)) {
			break
		}

		tag := bo.Uint16(tiff[entryOffset : entryOffset+2])
		typ := bo.Uint16(tiff[entryOffset+2 : entryOffset+4])
		count := bo.Uint32(tiff[entryOffset+4 : entryOffset+8])

		size := tagComponentSize(typ)
		// A count this large is either a corrupt file or a hostile one -
		// either way the tags this reader looks for are all single values
		// or short strings, so anything past a generous cap is skipped
		// rather than trusted enough to compute a byte length from.
		if size == 0 || count == 0 || count > 1<<16 {
			continue
		}

		total := uint64(size) * uint64(count)
		if total > 1<<20 {
			continue
		}

		var val []byte
		if total <= 4 {
			val = tiff[entryOffset+8 : uint64(entryOffset)+8+total]
		} else {
			offset := bo.Uint32(tiff[entryOffset+8 : entryOffset+12])
			if uint64(offset)+total > uint64(len(tiff)) {
				continue
			}
			val = tiff[offset : uint64(offset)+total]
		}

		fn(tag, typ, val)
	}
}

// tagComponentSize returns the byte size of one component of Exif type typ,
// or 0 for a type this reader doesn't know how to decode.
func tagComponentSize(typ uint16) int {
	switch typ {
	case 1, 2, 6, 7: // BYTE, ASCII, SBYTE, UNDEFINED
		return 1
	case 3, 8: // SHORT, SSHORT
		return 2
	case 4, 9, 13: // LONG, SLONG, IFD (SubIFD pointers)
		return 4
	case 5, 10: // RATIONAL, SRATIONAL
		return 8
	default:
		return 0
	}
}

// asciiValue decodes val as an Exif ASCII value (type 2): NUL-terminated,
// often with trailing padding. Returns ok=false for a wrong type or a
// value that's empty once trimmed, so callers can just skip setting the
// field.
func asciiValue(typ uint16, val []byte) (string, bool) {
	if typ != 2 {
		return "", false
	}

	s := strings.TrimRight(string(val), "\x00")
	s = strings.TrimSpace(s)

	if s == "" {
		return "", false
	}

	return s, true
}

// uintValue decodes val as an unsigned integer from a SHORT or LONG entry
// (the two Exif types this reader treats as plain counts: ISO and the
// Exif SubIFD pointer).
func uintValue(bo binary.ByteOrder, typ uint16, val []byte) (uint32, bool) {
	switch typ {
	case 3: // SHORT
		if len(val) < 2 {
			return 0, false
		}
		return uint32(bo.Uint16(val[:2])), true
	case 4, 13: // LONG, IFD
		if len(val) < 4 {
			return 0, false
		}
		return bo.Uint32(val[:4]), true
	}

	return 0, false
}

// rationalValue decodes val as an unsigned RATIONAL (type 5: a numerator and
// denominator, each a LONG) - the type Exif uses for exposure time,
// aperture, and focal length. ok is false for a wrong type, a truncated
// value, or a zero denominator.
func rationalValue(bo binary.ByteOrder, typ uint16, val []byte) (float64, bool) {
	if typ != 5 || len(val) < 8 {
		return 0, false
	}

	num := bo.Uint32(val[0:4])
	den := bo.Uint32(val[4:8])

	if den == 0 {
		return 0, false
	}

	return float64(num) / float64(den), true
}

// rationalsValue decodes val as n consecutive unsigned RATIONALs - the
// shape Exif uses for a GPS coordinate's degrees/minutes/seconds triple.
// ok is false for a wrong type, a value holding fewer than n rationals, or
// any zero denominator among them.
func rationalsValue(bo binary.ByteOrder, typ uint16, val []byte, n int) ([]float64, bool) {
	if typ != 5 || len(val) < n*8 {
		return nil, false
	}

	out := make([]float64, n)

	for i := range out {
		num := bo.Uint32(val[i*8 : i*8+4])
		den := bo.Uint32(val[i*8+4 : i*8+8])

		if den == 0 {
			return nil, false
		}

		out[i] = float64(num) / float64(den)
	}

	return out, true
}
