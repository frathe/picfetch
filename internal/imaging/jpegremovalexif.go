package imaging

import "encoding/binary"

// removalEXIF rebuilds only finite rendering declarations. Preserving those
// alongside unchanged image data avoids guessing EXIF/ICC precedence or changing
// chroma placement. No source directory, pointer, padding or text is copied.
func removalEXIF(tiff []byte) ([]byte, error) {
	bo, ok := tiffOrder(tiff)
	if !ok {
		return nil, ErrJPEGMetadataOrientation
	}
	const (
		primary = iota
		exif
		interop
	)
	// GPS and thumbnail directories have no primary-image rendering values.
	offsets := [3]uint64{uint64(bo.Uint32(tiff[4:8]))}
	var ends [3]uint64
	var orientation, positioning, space uint16
	var index string
	for level := range offsets {
		offset := offsets[level]
		if level != primary && offset == 0 {
			continue
		}
		invalid := ErrJPEGMetadataProcess
		if level == primary {
			invalid = ErrJPEGMetadataOrientation
		}
		header, ok := tiffSpan(tiff, offset, 2)
		if !ok || offset < 8 {
			return nil, invalid
		}
		count := uint64(bo.Uint16(header))
		entries, ok := tiffSpan(tiff, offset+2, count*12+4)
		if !ok {
			return nil, invalid
		}
		ends[level] = offset + 2 + count*12 + 4
		for previous := 0; previous < level; previous++ {
			if offset < ends[previous] && offsets[previous] < ends[level] {
				return nil, ErrJPEGMetadataProcess
			}
		}
		for i := uint64(0); i < count; i++ {
			e := entries[i*12 : i*12+12]
			tag, kind, values := bo.Uint16(e[:2]), bo.Uint16(e[2:4]), bo.Uint32(e[4:8])
			value := bo.Uint16(e[8:10])
			switch {
			case level == primary && tag == 0x0112:
				if orientation != 0 || kind != 3 || values != 1 || value < 1 || value > 8 {
					return nil, ErrJPEGMetadataOrientation
				}
				orientation = value
			case level == primary && tag == 0x0213:
				if positioning != 0 || kind != 3 || values != 1 || value < 1 || value > 2 {
					return nil, ErrJPEGMetadataProcess
				}
				positioning = value
			case level == primary && (tag == 0x012d || tag == 0x013e || tag == 0x013f || tag == 0x0211 || tag == 0x0214), level == exif && tag == 0xa500:
				// Numerical transform extensions need separate qualification.
				return nil, ErrJPEGMetadataProcess
			case level == primary && tag == 0x8769 || level == exif && tag == 0xa005:
				if offsets[level+1] != 0 || kind != 4 || values != 1 || bo.Uint32(e[8:12]) < 8 {
					return nil, ErrJPEGMetadataProcess
				}
				offsets[level+1] = uint64(bo.Uint32(e[8:12]))
			case level == exif && tag == 0xa001:
				if space != 0 || kind != 3 || values != 1 || value != 1 && value != 0xffff {
					return nil, ErrJPEGMetadataProcess
				}
				space = value
			case level == interop && tag == 1:
				if index != "" || kind != 2 || values != 4 || string(e[8:12]) != "R98\x00" && string(e[8:12]) != "R03\x00" {
					return nil, ErrJPEGMetadataProcess
				}
				index = string(e[8:12])
			}
		}
	}
	var root, sub, inter []removalEXIFEntry
	if orientation > 1 {
		root = append(root, removalEXIFEntry{0x0112, 3, 1, uint32(orientation)})
	}
	if positioning == 2 {
		root = append(root, removalEXIFEntry{0x0213, 3, 1, uint32(positioning)})
	}
	if space != 0 {
		sub = append(sub, removalEXIFEntry{0xa001, 3, 1, uint32(space)})
	}
	if index != "" {
		inter = append(inter, removalEXIFEntry{1, 2, 4, binary.LittleEndian.Uint32([]byte(index))})
	}
	if len(sub)+len(inter) != 0 {
		subOffset := uint32(8 + 2 + 12*(len(root)+1) + 4)
		root = append(root, removalEXIFEntry{0x8769, 4, 1, subOffset})
		if len(inter) != 0 {
			interOffset := subOffset + uint32(2+12*(len(sub)+1)+4)
			sub = append(sub, removalEXIFEntry{0xa005, 4, 1, interOffset})
		}
	}
	if len(root) == 0 {
		return nil, nil
	}
	out := []byte{'E', 'x', 'i', 'f', 0, 0, 'I', 'I', 42, 0, 8, 0, 0, 0}
	for _, entries := range [][]removalEXIFEntry{root, sub, inter} {
		if len(entries) == 0 {
			continue
		}
		out = binary.LittleEndian.AppendUint16(out, uint16(len(entries)))
		for _, entry := range entries {
			out = binary.LittleEndian.AppendUint16(out, entry.tag)
			out = binary.LittleEndian.AppendUint16(out, entry.kind)
			out = binary.LittleEndian.AppendUint32(out, entry.count)
			out = binary.LittleEndian.AppendUint32(out, entry.value)
		}
		out = binary.LittleEndian.AppendUint32(out, 0)
	}
	return out, nil
}

type removalEXIFEntry struct {
	tag, kind    uint16
	count, value uint32
}
