package imaging

import (
	"bytes"
	"context"
	"encoding/binary"
	"slices"
)

const removalICCSignature = "ICC_PROFILE\x00"
const maxRemovalICCBytes = 4 * 1024 * 1024

type removalICC struct {
	total  byte
	chunks map[byte][]byte
	size   int
}

func (c *removalICC) add(p []byte) error {
	if len(p) < 14 || p[12] == 0 || p[13] == 0 || p[12] > p[13] || c.total != 0 && c.total != p[13] {
		return ErrJPEGMetadataProfile
	}
	if c.chunks == nil {
		c.chunks = make(map[byte][]byte)
	}
	if _, exists := c.chunks[p[12]]; exists {
		return ErrJPEGMetadataProfile
	}
	c.size += len(p) - 14
	if c.size > maxRemovalICCBytes {
		return ErrJPEGMetadataProfile
	}
	c.total = p[13]
	c.chunks[p[12]] = p[14:]
	return nil
}

// normalized reports whether the assembled source is already exactly sanitized,
// so callers can retain its original marker packaging without a false rewrite.
func (c *removalICC) normalized(ctx context.Context, components int) ([][]byte, bool, error) {
	if c.total == 0 {
		return nil, true, nil
	}
	if len(c.chunks) != int(c.total) {
		return nil, false, ErrJPEGMetadataProfile
	}
	data := make([]byte, 0, c.size)
	for i := 1; i <= int(c.total); i++ {
		data = append(data, c.chunks[byte(i)]...)
	}
	profile, err := normalizeRemovalICC(ctx, data, components)
	if err != nil {
		return nil, false, err
	}
	const chunkSize = 65533 - 14
	total := (len(profile) + chunkSize - 1) / chunkSize
	var segments [][]byte
	for pos, sequence := 0, 1; pos < len(profile); sequence++ {
		end := min(pos+chunkSize, len(profile))
		payload := append([]byte(removalICCSignature), byte(sequence), byte(total))
		payload = append(payload, profile[pos:end]...)
		segments = append(segments, jpegSegmentBytes(0xe2, payload))
		pos = end
	}
	return segments, bytes.Equal(data, profile), nil
}

// normalizeRemovalICC supports matrix/TRC RGB and monochrome input/display
// profiles with XYZ PCS. It rebuilds every byte, retaining only qualified
// numerical transform tags, and never copies gaps, padding or private data.
func normalizeRemovalICC(ctx context.Context, p []byte, components int) ([]byte, error) {
	if len(p) < 132 || len(p) > maxRemovalICCBytes || uint64(binary.BigEndian.Uint32(p)) != uint64(len(p)) || string(p[36:40]) != "acsp" {
		return nil, ErrJPEGMetadataProfile
	}
	version := p[8]
	if version != 2 && version != 4 || p[9] > 0x40 || p[9]&15 > 9 || !allZero(p[10:12]) || (string(p[12:16]) != "mntr" && string(p[12:16]) != "scnr") || string(p[20:24]) != "XYZ " {
		return nil, ErrJPEGMetadataProfile
	}
	if components == 1 && string(p[16:20]) != "GRAY" || components == 3 && string(p[16:20]) != "RGB " {
		return nil, ErrJPEGMetadataProfile
	}
	// Attributes 0-3 describe the media; 4-31 are reserved. The high word
	// belongs to the vendor and is identity data, removed during rebuilding.
	if binary.BigEndian.Uint32(p[44:48]) & ^uint32(3) != 0 || binary.BigEndian.Uint32(p[60:64]) & ^uint32(15) != 0 || binary.BigEndian.Uint32(p[64:68]) > 3 || !allZero(p[100:128]) {
		return nil, ErrJPEGMetadataProfile
	}
	// ICC PCS illuminant is D50, encoded as the specified s15Fixed16 XYZ.
	if !bytes.Equal(p[68:80], []byte{0, 0, 0xf6, 0xd6, 0, 1, 0, 0, 0, 0, 0xd3, 0x2d}) {
		return nil, ErrJPEGMetadataProfile
	}
	count := int(binary.BigEndian.Uint32(p[128:132]))
	if count == 0 || count > 128 || count > (len(p)-132)/12 {
		return nil, ErrJPEGMetadataProfile
	}
	endTable := 132 + 12*count
	tags := make(map[string][]byte, count)
	type span struct{ start, end int }
	var spans []span
	for i := 0; i < count; i++ {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		e := p[132+i*12 : 144+i*12]
		name := string(e[:4])
		start64, size64 := uint64(binary.BigEndian.Uint32(e[4:8])), uint64(binary.BigEndian.Uint32(e[8:12]))
		if _, exists := tags[name]; exists {
			return nil, ErrJPEGMetadataProfile
		}
		if start64 < uint64(endTable) || start64%4 != 0 || size64 < 8 || start64+size64 > uint64(len(p)) {
			return nil, ErrJPEGMetadataProfile
		}
		start, end := int(start64), int(start64+size64)
		for _, other := range spans {
			if start < other.end && other.start < end && (start != other.start || end != other.end) {
				return nil, ErrJPEGMetadataProfile
			}
		}
		spans = append(spans, span{start, end})
		value := p[start:end]
		if !allZero(value[4:8]) {
			return nil, ErrJPEGMetadataProfile
		}
		switch name {
		case "desc", "cprt", "dmnd", "dmdd", "vued":
			if !validICCDescription(value, version) {
				return nil, ErrJPEGMetadataProfile
			}
		case "wtpt", "bkpt", "rXYZ", "gXYZ", "bXYZ":
			if len(value) != 20 || string(value[:4]) != "XYZ " {
				return nil, ErrJPEGMetadataProfile
			}
		case "lumi":
			if len(value) != 20 || string(value[:4]) != "XYZ " || !allZero(value[8:12]) || int32(binary.BigEndian.Uint32(value[12:16])) < 0 || !allZero(value[16:20]) {
				return nil, ErrJPEGMetadataProfile
			}
		case "meas":
			if !validICCMeasurement(value) {
				return nil, ErrJPEGMetadataProfile
			}
		case "tech":
			if !validICCTechnology(value) {
				return nil, ErrJPEGMetadataProfile
			}
		case "chad":
			if len(value) != 44 || string(value[:4]) != "sf32" {
				return nil, ErrJPEGMetadataProfile
			}
		case "rTRC", "gTRC", "bTRC", "kTRC":
			if !validICCCurve(value, version) {
				return nil, ErrJPEGMetadataProfile
			}
		case "chrm":
			if len(value) != 12+8*components || string(value[:4]) != "chrm" || int(binary.BigEndian.Uint16(value[8:10])) != components || binary.BigEndian.Uint16(value[10:12]) != 0 {
				return nil, ErrJPEGMetadataProfile
			}
		default:
			// Even a well-formed unfamiliar tag may change CMM interpretation.
			return nil, ErrJPEGMetadataProfile
		}
		tags[name] = value
	}
	required := []string{"desc", "cprt", "wtpt"}
	if components == 1 {
		required = append(required, "kTRC")
	} else {
		required = append(required, "rXYZ", "gXYZ", "bXYZ", "rTRC", "gTRC", "bTRC")
	}
	for _, name := range required {
		if _, ok := tags[name]; !ok {
			return nil, ErrJPEGMetadataProfile
		}
	}
	if components == 1 {
		for _, name := range []string{"rXYZ", "gXYZ", "bXYZ", "rTRC", "gTRC", "bTRC"} {
			if tags[name] != nil {
				return nil, ErrJPEGMetadataProfile
			}
		}
	} else if tags["kTRC"] != nil {
		return nil, ErrJPEGMetadataProfile
	}
	delete(tags, "dmnd")
	delete(tags, "dmdd")
	delete(tags, "vued")
	tags["desc"] = neutralICCText(version, true)
	tags["cprt"] = neutralICCText(version, false)
	names := make([]string, 0, len(tags))
	for name := range tags {
		names = append(names, name)
	}
	slices.Sort(names)
	out := make([]byte, 132+12*len(names))
	copy(out[8:24], p[8:24])
	// A fixed valid date carries no source creation time.
	copy(out[24:36], []byte{7, 0xd0, 0, 1, 0, 1, 0, 0, 0, 0, 0, 0})
	copy(out[36:40], "acsp")
	copy(out[44:48], p[44:48])
	copy(out[60:80], p[60:80])
	binary.BigEndian.PutUint32(out[128:132], uint32(len(names)))
	for i, name := range names {
		value := tags[name]
		e := out[132+i*12 : 144+i*12]
		copy(e[:4], name)
		binary.BigEndian.PutUint32(e[4:8], uint32(len(out)))
		binary.BigEndian.PutUint32(e[8:12], uint32(len(value)))
		out = append(out, value...)
		for len(out)%4 != 0 {
			out = append(out, 0)
		}
	}
	binary.BigEndian.PutUint32(out[:4], uint32(len(out)))
	return out, nil
}

func validICCMeasurement(p []byte) bool {
	// ICC measurementType: observer, backing XYZ, geometry, flare and
	// illuminant. Only fixed numerical values and standard enums survive.
	return len(p) == 36 && string(p[:4]) == "meas" &&
		binary.BigEndian.Uint32(p[8:12]) <= 2 &&
		binary.BigEndian.Uint32(p[24:28]) <= 2 &&
		binary.BigEndian.Uint32(p[28:32]) <= 65536 &&
		binary.BigEndian.Uint32(p[32:36]) <= 8
}

func validICCTechnology(p []byte) bool {
	if len(p) != 12 || string(p[:4]) != "sig " {
		return false
	}
	// ICC technologyTag signatures, not arbitrary four-byte vendor text.
	switch string(p[8:12]) {
	case "fscn", "dcam", "rscn", "ijet", "twax", "epho", "esta", "dsub",
		"rpho", "fprn", "vidm", "vidc", "pjtv", "CRT ", "PMD ", "AMD ",
		"LCD ", "OLED", "KPCD", "imgs", "grav", "offs", "silk", "flex",
		"mpfs", "mpfr", "dmpc", "dcpj":
		return true
	}
	return false
}

func allZero(p []byte) bool {
	for _, v := range p {
		if v != 0 {
			return false
		}
	}
	return true
}

func validICCCurve(p []byte, version byte) bool {
	if len(p) < 12 {
		return false
	}
	switch string(p[:4]) {
	case "curv":
		n := uint64(binary.BigEndian.Uint32(p[8:12]))
		if n > 65536 || uint64(len(p)) != 12+2*n {
			return false
		}
		if n == 1 {
			return binary.BigEndian.Uint16(p[12:14]) != 0
		}
		for i := 14; i < len(p); i += 2 {
			if binary.BigEndian.Uint16(p[i:]) < binary.BigEndian.Uint16(p[i-2:]) {
				return false
			}
		}
		return true
	case "para":
		if version != 4 || !allZero(p[10:12]) {
			return false
		}
		kind := binary.BigEndian.Uint16(p[8:10])
		// Gamma and the sRGB piecewise curve are the qualified families.
		if kind != 0 && kind != 3 || kind == 0 && len(p) != 16 || kind == 3 && len(p) != 32 {
			return false
		}
		if int32(binary.BigEndian.Uint32(p[12:16])) <= 0 {
			return false
		}
		if kind == 3 {
			a, b, c, d := int64(int32(binary.BigEndian.Uint32(p[16:20]))), int64(int32(binary.BigEndian.Uint32(p[20:24]))), int64(int32(binary.BigEndian.Uint32(p[24:28]))), int64(int32(binary.BigEndian.Uint32(p[28:32])))
			if a <= 0 || c < 0 || d < 0 || d > 65536 || a*d+b*65536 < 0 {
				return false
			}
		}
		return true
	}
	return false
}

func validICCDescription(p []byte, version byte) bool {
	if len(p) < 9 {
		return false
	}
	if version == 4 {
		if string(p[:4]) != "mluc" || len(p) < 16 || binary.BigEndian.Uint32(p[12:16]) != 12 {
			return false
		}
		n := uint64(binary.BigEndian.Uint32(p[8:12]))
		end := uint64(16) + 12*n
		if end > uint64(len(p)) {
			return false
		}
		for i := uint64(0); i < n; i++ {
			e := p[16+i*12 : 28+i*12]
			size, start := uint64(binary.BigEndian.Uint32(e[4:8])), uint64(binary.BigEndian.Uint32(e[8:12]))
			if size%2 != 0 || start < end || start+size > uint64(len(p)) {
				return false
			}
		}
		return true
	}
	if string(p[:4]) == "text" {
		return p[len(p)-1] == 0
	}
	if len(p) < 12 || string(p[:4]) != "desc" {
		return false
	}
	n := uint64(binary.BigEndian.Uint32(p[8:12]))
	end := 12 + n
	if n == 0 || end+8 > uint64(len(p)) || p[end-1] != 0 {
		return false
	}
	unicodeCount := uint64(binary.BigEndian.Uint32(p[end+4 : end+8]))
	end += 8 + 2*unicodeCount
	return end+70 <= uint64(len(p)) && uint64(len(p)) <= end+73 && p[end+2] <= 67 && allZero(p[end+70:])
}

func neutralICCText(version byte, description bool) []byte {
	text := ""
	if description {
		text = "Color profile"
	}
	if version == 4 {
		p := make([]byte, 28+2*len(text))
		copy(p, "mluc")
		binary.BigEndian.PutUint32(p[8:12], 1)
		binary.BigEndian.PutUint32(p[12:16], 12)
		copy(p[16:20], "enUS")
		binary.BigEndian.PutUint32(p[20:24], uint32(2*len(text)))
		binary.BigEndian.PutUint32(p[24:28], 28)
		for i, c := range []byte(text) {
			p[29+2*i] = c
		}
		return p
	}
	if !description {
		return []byte{'t', 'e', 'x', 't', 0, 0, 0, 0, 0}
	}
	p := make([]byte, 12+len(text)+1+8+70)
	copy(p, "desc")
	binary.BigEndian.PutUint32(p[8:12], uint32(len(text)+1))
	copy(p[12:], text)
	return p
}
