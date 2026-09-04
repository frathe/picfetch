package imaging

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/gen2brain/avif"
	"github.com/gen2brain/heic"
)

// readEXIFOrientation returns the orientation tag's value (1-8), or 1 (no
// correction needed). JPEG files store it in an APP1 Exif segment, PNG files
// in an eXIf chunk, WebP files in an EXIF chunk, and TIFF-container RAW files
// in IFD0. A missing or unreadable tag is 1.
func readEXIFOrientation(data []byte) int {
	if len(data) >= 4 && data[0] == 0xFF && data[1] == 0xD8 {
		return jpegEXIFOrientation(data)
	}
	if o := pngEXIFOrientation(data); o != 1 {
		return o
	}
	if o := webpEXIFOrientation(data); o != 1 {
		return o
	}
	if o := tiffIFD0Orientation(data); o != 1 {
		return o
	}
	return 1
}

func webpEXIFOrientation(data []byte) int {
	if len(data) < 12 || string(data[:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return 1
	}

	for offset := 12; offset+8 <= len(data); {
		length := uint64(binary.LittleEndian.Uint32(data[offset+4 : offset+8]))
		payloadStart := offset + 8
		payloadEnd := uint64(payloadStart) + length
		padding := length % 2
		if payloadEnd+padding > uint64(len(data)) {
			return 1
		}
		if string(data[offset:offset+4]) == "EXIF" {
			payload := data[payloadStart:int(payloadEnd)]
			if orientation := parseExifOrientation(payload); orientation != 0 {
				return orientation
			}
			return tiffIFD0Orientation(payload)
		}
		offset = int(payloadEnd + padding)
	}

	return 1
}

func pngEXIFOrientation(data []byte) int {
	const signature = "\x89PNG\r\n\x1a\n"
	if len(data) < len(signature) || string(data[:len(signature)]) != signature {
		return 1
	}

	for offset := len(signature); offset+12 <= len(data); {
		length := uint64(binary.BigEndian.Uint32(data[offset : offset+4]))
		payloadStart := offset + 8
		payloadEnd := uint64(payloadStart) + length
		if payloadEnd+4 > uint64(len(data)) {
			return 1
		}
		if string(data[offset+4:offset+8]) == "eXIf" {
			return tiffIFD0Orientation(data[payloadStart:int(payloadEnd)])
		}
		offset = int(payloadEnd) + 4
	}

	return 1
}

// jpegEXIFOrientation is the APP1 walk that used to be readEXIFOrientation
// itself, split out so TIFF-container RAW files can use IFD0's orientation
// tag without pretending to be a JPEG.
func jpegEXIFOrientation(data []byte) int {
	found := 1
	walkJPEGSegments(data, func(marker byte, payload []byte) bool {
		if marker == 0xE1 {
			if o := parseExifOrientation(payload); o != 0 {
				found = o
				return false
			}
		}
		return true
	})
	return found
}

func tiffIFD0Orientation(data []byte) int {
	bo, ok := tiffOrder(data)
	if !ok {
		return 1
	}

	found := 1
	walkIFD(data, bo, bo.Uint32(data[4:8]), func(tag, typ uint16, val []byte) {
		if tag != 0x0112 {
			return
		}
		if v, ok := uintValue(bo, typ, val); ok && v >= 1 && v <= 8 {
			found = int(v)
		}
	})
	return found
}

// parseExifOrientation reads the orientation tag (0x0112) out of an APP1
// segment's payload, which starts with the "Exif\0\0" marker followed by a
// TIFF header. It returns 0 if the segment isn't Exif data or has no valid
// orientation tag.
func parseExifOrientation(seg []byte) int {
	if len(seg) < 8 || string(seg[:6]) != "Exif\x00\x00" {
		return 0
	}

	tiff := seg[6:]

	if len(tiff) < 8 {
		return 0
	}

	var bo binary.ByteOrder

	switch string(tiff[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return 0
	}

	if bo.Uint16(tiff[2:4]) != 0x002A {
		return 0
	}

	ifdOffset := bo.Uint32(tiff[4:8])

	if ifdOffset+2 > uint32(len(tiff)) {
		return 0
	}

	numEntries := bo.Uint16(tiff[ifdOffset : ifdOffset+2])
	entriesStart := ifdOffset + 2

	for i := uint32(0); i < uint32(numEntries); i++ {
		entryOffset := entriesStart + i*12

		if entryOffset+12 > uint32(len(tiff)) {
			break
		}

		tag := bo.Uint16(tiff[entryOffset : entryOffset+2])

		if tag != 0x0112 {
			continue
		}

		valType := bo.Uint16(tiff[entryOffset+2 : entryOffset+4])

		if valType != 3 { // SHORT
			return 0
		}

		v := bo.Uint16(tiff[entryOffset+8 : entryOffset+10])

		if v < 1 || v > 8 {
			return 0
		}

		return int(v)
	}

	return 0
}

// Metadata is the subset of a photo's Exif tags the EXIF window (see
// internal/ui/exifwin) displays: camera make/model, lens, exposure,
// aperture, ISO, focal length, capture date, and - only where the photo
// carries one - the GPS position its map view centers on. A zero Metadata
// (every field "", no position) means either the file has no Exif data or
// none of these particular tags.
type Metadata struct {
	Make         string
	Model        string
	LensModel    string
	ExposureTime string
	FNumber      string
	ISO          string
	FocalLength  string
	DateTaken    string

	// Latitude and Longitude are the capture position in signed decimal
	// degrees (north and east positive), read from the GPS sub-IFD that
	// IFD0's pointer tag 0x8825 locates. Only meaningful when HasGPS is
	// set: a photo without location tags leaves all three zero, which is
	// what keeps the EXIF window's map collapsed and hidden.
	Latitude  float64
	Longitude float64
	HasGPS    bool

	// DateTakenTime is DateTaken's underlying value, parsed from the same
	// raw Exif tag - for callers that need to compare or sort capture
	// dates (see CaptureDate in loader.go and internal/filesort's
	// captureOrModTime) rather than just display DateTaken's
	// already-formatted string. Zero when DateTaken is empty, or set from a
	// raw value that didn't parse.
	DateTakenTime time.Time
}

// Empty reports whether none of m's fields were populated - either the file
// carried no Exif segment at all, or it did but none of the tags
// ReadMetadata looks for were present.
func (m Metadata) Empty() bool {
	return m == Metadata{}
}

// ReadMetadata scans data (a whole image file's raw bytes) for an Exif APP1
// segment and extracts Metadata from it. Like readEXIFOrientation, it is
// deliberately failure-tolerant throughout: a malformed tag, a truncated
// value, or a file with no Exif data at all just leaves the corresponding
// field (or all of them) blank rather than returning an error - there is no
// error to report, only "nothing to show".
func ReadMetadata(data []byte) Metadata {
	if len(data) >= 4 && data[0] == 0xFF && data[1] == 0xD8 {
		return jpegMetadata(data)
	}
	if _, ok := tiffOrder(data); ok {
		if m := parseExifMetadata(data); !m.Empty() {
			return m
		}
	}
	if m := isobmffMetadata(data); !m.Empty() {
		return m
	}
	if jpegBytes, ok := embeddedJPEGPreview(data); ok {
		return jpegMetadata(jpegBytes)
	}
	return Metadata{}
}

func jpegMetadata(data []byte) Metadata {
	var found Metadata
	walkJPEGSegments(data, func(marker byte, payload []byte) bool {
		if marker != 0xE1 {
			return true
		}
		if len(payload) >= 8 && string(payload[:6]) == "Exif\x00\x00" {
			if m := parseExifMetadata(payload[6:]); !m.Empty() {
				found = m
				return false
			}
		}
		return true
	})
	return found
}

// isobmffMetadata reads Exif metadata out of an ISOBMFF-boxed file (HEIC or
// AVIF) - JPEG APP1 and TIFF IFD0 are handled before this is called. Both
// DecodeExif calls fail fast (pure box-walking, no wasm/cgo invocation) on a
// file that isn't theirs, so trying heic then avif costs nothing extra for
// the common case of a format with no Exif at all (PNG, GIF, WebP, BMP, ICO,
// XPM).
func isobmffMetadata(data []byte) Metadata {
	if ex, err := heic.DecodeExif(bytes.NewReader(data)); err == nil {
		return metadataFromISOBMFFExif(ex.Make, ex.Model, ex.ExposureTime, ex.FNumber, ex.ISOSpeed, ex.FocalLength, ex.DateTimeOriginal, ex.DateTime, ex.GPSLatitude, ex.GPSLongitude)
	}

	if ex, err := avif.DecodeExif(bytes.NewReader(data)); err == nil {
		return metadataFromISOBMFFExif(ex.Make, ex.Model, ex.ExposureTime, ex.FNumber, ex.ISOSpeed, ex.FocalLength, ex.DateTimeOriginal, ex.DateTime, ex.GPSLatitude, ex.GPSLongitude)
	}

	return Metadata{}
}

// metadataFromISOBMFFExif adapts the fields heic.Exif and avif.Exif share
// (the two packages expose identically-shaped structs) into Metadata,
// reusing the same formatting helpers the JPEG APP1 walk uses so a HEIC/AVIF
// photo's EXIF window reads the same as a JPEG's. LensModel has no
// equivalent in either struct, so it's left unset, same as a JPEG missing
// that tag. Both decoders report an absent position as a zero latitude and
// longitude rather than a flag, so an exact (0, 0) is read as "no
// location": Null Island is open ocean, and treating that one point as
// missing is a better trade than showing a map of it for every photo that
// simply has no GPS tags.
func metadataFromISOBMFFExif(cameraMake, model string, exposureTime, fNumber float64, iso int, focalLength float64, dateTimeOriginal, dateTime string, lat, lon float64) Metadata {
	m := Metadata{Make: cameraMake, Model: model}

	if (lat != 0 || lon != 0) && validCoordinates(lat, lon) {
		m.Latitude, m.Longitude, m.HasGPS = lat, lon, true
	}

	if exposureTime > 0 {
		m.ExposureTime = formatExposureTime(exposureTime)
	}
	if fNumber > 0 {
		m.FNumber = fmt.Sprintf("f/%.1f", fNumber)
	}
	if iso > 0 {
		m.ISO = fmt.Sprintf("ISO %d", iso)
	}
	if focalLength > 0 {
		m.FocalLength = formatFocalLength(focalLength)
	}

	if dateTimeOriginal != "" {
		m.DateTaken = formatExifDate(dateTimeOriginal)
		if t, ok := parseExifDateTime(dateTimeOriginal); ok {
			m.DateTakenTime = t
		}
	} else if dateTime != "" {
		m.DateTaken = formatExifDate(dateTime)
		if t, ok := parseExifDateTime(dateTime); ok {
			m.DateTakenTime = t
		}
	}

	return m
}

// exifIFDPointer (0x8769) locates the Exif SubIFD, which - unlike IFD0's
// camera make/model/date - holds the shooting parameters (exposure,
// aperture, ISO, focal length, lens, and the more specific
// DateTimeOriginal).
const exifIFDPointer = 0x8769

// gpsIFDPointer (0x8825) locates the GPS sub-IFD, whose latitude and
// longitude tags the EXIF window's map view centers on.
const gpsIFDPointer = 0x8825

// parseExifMetadata reads tiff - the TIFF header and IFDs following the
// "Exif\x00\x00" marker, same payload parseExifOrientation works from - and
// walks IFD0 plus the Exif SubIFD it points to for the tags Metadata cares
// about.
func parseExifMetadata(tiff []byte) Metadata {
	var m Metadata

	if len(tiff) < 8 {
		return m
	}

	var bo binary.ByteOrder

	switch string(tiff[:2]) {
	case "II":
		bo = binary.LittleEndian
	case "MM":
		bo = binary.BigEndian
	default:
		return m
	}

	if bo.Uint16(tiff[2:4]) != 0x002A {
		return m
	}

	ifd0Offset := bo.Uint32(tiff[4:8])

	var exifIFDOffset uint32
	haveExifIFD := false

	var gpsIFDOffset uint32
	haveGPSIFD := false

	walkIFD(tiff, bo, ifd0Offset, func(tag, typ uint16, val []byte) {
		switch tag {
		case 0x010F: // Make
			if s, ok := asciiValue(typ, val); ok {
				m.Make = s
			}
		case 0x0110: // Model
			if s, ok := asciiValue(typ, val); ok {
				m.Model = s
			}
		case 0x0132: // DateTime - fallback if the Exif SubIFD has no
			// DateTimeOriginal (0x9003), which is preferred where present.
			if s, ok := asciiValue(typ, val); ok {
				m.DateTaken = formatExifDate(s)
				if t, ok := parseExifDateTime(s); ok {
					m.DateTakenTime = t
				}
			}
		case exifIFDPointer:
			if v, ok := uintValue(bo, typ, val); ok {
				exifIFDOffset = v
				haveExifIFD = true
			}
		case gpsIFDPointer:
			if v, ok := uintValue(bo, typ, val); ok {
				gpsIFDOffset = v
				haveGPSIFD = true
			}
		}
	})

	if haveGPSIFD {
		if lat, lon, ok := parseGPSIFD(tiff, bo, gpsIFDOffset); ok {
			m.Latitude, m.Longitude, m.HasGPS = lat, lon, true
		}
	}

	if !haveExifIFD {
		return m
	}

	walkIFD(tiff, bo, exifIFDOffset, func(tag, typ uint16, val []byte) {
		switch tag {
		case 0x829A: // ExposureTime
			if r, ok := rationalValue(bo, typ, val); ok && r > 0 {
				m.ExposureTime = formatExposureTime(r)
			}
		case 0x829D: // FNumber
			if r, ok := rationalValue(bo, typ, val); ok && r > 0 {
				m.FNumber = fmt.Sprintf("f/%.1f", r)
			}
		case 0x8827: // PhotographicSensitivity (ISO)
			if v, ok := uintValue(bo, typ, val); ok {
				m.ISO = fmt.Sprintf("ISO %d", v)
			}
		case 0x920A: // FocalLength
			if r, ok := rationalValue(bo, typ, val); ok && r > 0 {
				m.FocalLength = formatFocalLength(r)
			}
		case 0x9003: // DateTimeOriginal - takes priority over IFD0's DateTime
			if s, ok := asciiValue(typ, val); ok {
				m.DateTaken = formatExifDate(s)
				if t, ok := parseExifDateTime(s); ok {
					m.DateTakenTime = t
				}
			}
		case 0xA434: // LensModel
			if s, ok := asciiValue(typ, val); ok {
				m.LensModel = s
			}
		}
	})

	return m
}

// parseGPSIFD reads the latitude/longitude pair out of the GPS sub-IFD at
// gpsOffset within tiff and converts it to signed decimal degrees. ok is
// false unless all four tags are present and readable and the result lands
// in valid coordinate ranges - a partial or malformed GPS IFD is treated
// as no location at all, in keeping with the rest of this reader.
func parseGPSIFD(tiff []byte, bo binary.ByteOrder, gpsOffset uint32) (lat, lon float64, ok bool) {
	var latRef, lonRef string
	var latDMS, lonDMS []float64

	walkIFD(tiff, bo, gpsOffset, func(tag, typ uint16, val []byte) {
		switch tag {
		case 0x0001: // GPSLatitudeRef: "N" or "S"
			if s, ok := asciiValue(typ, val); ok {
				latRef = strings.ToUpper(s)
			}
		case 0x0002: // GPSLatitude: degrees, minutes, seconds
			if v, ok := rationalsValue(bo, typ, val, 3); ok {
				latDMS = v
			}
		case 0x0003: // GPSLongitudeRef: "E" or "W"
			if s, ok := asciiValue(typ, val); ok {
				lonRef = strings.ToUpper(s)
			}
		case 0x0004: // GPSLongitude
			if v, ok := rationalsValue(bo, typ, val, 3); ok {
				lonDMS = v
			}
		}
	})

	lat, latOK := degreesFromDMS(latDMS, latRef, "N", "S")
	lon, lonOK := degreesFromDMS(lonDMS, lonRef, "E", "W")

	if !latOK || !lonOK || !validCoordinates(lat, lon) {
		return 0, 0, false
	}

	return lat, lon, true
}

// degreesFromDMS converts one Exif degrees/minutes/seconds triple into
// signed decimal degrees, negating it for the southern/western hemisphere
// reference. ok is false for a missing triple or a reference that is
// neither of the two the axis allows - Exif writes the hemisphere only in
// that tag, so without it the sign is unknowable and the coordinate is
// unusable rather than merely ambiguous.
func degreesFromDMS(dms []float64, ref, positive, negative string) (float64, bool) {
	if len(dms) != 3 || (ref != positive && ref != negative) {
		return 0, false
	}

	deg := dms[0] + dms[1]/60 + dms[2]/3600

	if ref == negative {
		deg = -deg
	}

	return deg, true
}

// validCoordinates reports whether lat/lon are real coordinates: in range
// and not NaN or infinite, which a rational with an absurd numerator could
// otherwise produce.
func validCoordinates(lat, lon float64) bool {
	if math.IsNaN(lat) || math.IsNaN(lon) || math.IsInf(lat, 0) || math.IsInf(lon, 0) {
		return false
	}

	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}
