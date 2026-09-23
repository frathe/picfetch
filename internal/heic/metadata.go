package heic

import "encoding/binary"

// primaryEXIF extracts bounded metadata associated with the designated still.
// The native worker is its only production caller. Metadata is best effort:
// malformed, ambiguous, external or unsupported storage leaves it absent,
// without rejecting pixels that the system can decode.
func primaryEXIF(data []byte) []byte {
	var meta []byte
	err := walkBoxes(data, func(kind string, payload []byte) error {
		if kind == "meta" {
			if meta != nil || len(payload) < 4 || binary.BigEndian.Uint32(payload) != 0 {
				return ErrInvalid
			}
			meta = payload[4:]
		}
		return nil
	})
	if err != nil || meta == nil {
		return nil
	}
	boxes := make(map[string][]byte, 5)
	err = walkBoxes(meta, func(kind string, payload []byte) error {
		switch kind {
		case "pitm", "iinf", "iref", "iloc", "idat":
			if _, exists := boxes[kind]; exists {
				return ErrInvalid
			}
			boxes[kind] = payload
		}
		return nil
	})
	if err != nil {
		return nil
	}
	primaryReader := metadataReader{data: boxes["pitm"]}
	version := primaryReader.uint(4)
	width := 2
	if version == 1<<24 {
		width = 4
	} else if version != 0 {
		return nil
	}
	primary := primaryReader.uint(width)
	if primary == 0 || !primaryReader.complete() {
		return nil
	}
	associated := metadataAssociations(boxes["iref"], primary)
	var selected uint32
	err = walkItemInfo(boxes["iinf"], func(id uint32, kind string, protected bool) error {
		if kind == "Exif" && !protected && associated[id] {
			if selected != 0 {
				return ErrInvalid
			}
			selected = id
		}
		return nil
	})
	if err != nil || selected == 0 {
		return nil
	}
	metadata := metadataItemData(boxes["iloc"], selected, data, boxes["idat"])
	tiff := tiffMetadata(metadata)
	if len(tiff) < 8 || (string(tiff[:4]) != "II\x2a\x00" && string(tiff[:4]) != "MM\x00\x2a") {
		return nil
	}
	return tiff
}

func metadataAssociations(data []byte, primary uint64) map[uint32]bool {
	r := metadataReader{data: data}
	version := r.uint(4)
	width := 2
	if version == 1<<24 {
		width = 4
	} else if version != 0 || r.failed {
		return nil
	}
	associated := make(map[uint32]bool)
	err := walkBoxes(r.data, func(kind string, payload []byte) error {
		if kind != "cdsc" {
			return nil
		}
		ref := metadataReader{data: payload}
		from, count := ref.uint(width), ref.uint(2)
		for index := uint64(0); index < count && !ref.failed; index++ {
			if ref.uint(width) == primary {
				associated[uint32(from)] = true
			}
		}
		if !ref.complete() {
			return ErrInvalid
		}
		return nil
	})
	if err != nil {
		return nil
	}
	return associated
}

// metadataItemData supports iloc file offsets and idat-relative extents. It
// never follows external data references or recursively resolves other items.
func metadataItemData(locations []byte, selected uint32, file, idat []byte) []byte {
	r := metadataReader{data: locations}
	version, flags := r.uint(1), r.uint(3)
	if version > 2 || flags != 0 || r.failed {
		return nil
	}
	sizes := r.uint(2)
	offsetSize, lengthSize := int(sizes>>12), int(sizes>>8&15)
	baseSize, indexSize := int(sizes>>4&15), int(sizes&15)
	if offsetSize > 8 || lengthSize == 0 || lengthSize > 8 || baseSize > 8 || indexSize > 8 || (version == 0 && indexSize != 0) {
		return nil
	}
	idSize := 2
	if version == 2 {
		idSize = 4
	}
	count := r.uint(idSize)
	if count > 65536 {
		return nil
	}
	var metadata []byte
	found := false
	for index := uint64(0); index < count && !r.failed; index++ {
		id := r.uint(idSize)
		method := uint64(0)
		if version != 0 {
			method = r.uint(2)
		}
		reference, base, extents := r.uint(2), r.uint(baseSize), r.uint(2)
		selectedItem := id == uint64(selected)
		if selectedItem {
			if found || reference != 0 || method > 1 {
				return nil
			}
			found = true
		}
		for extent := uint64(0); extent < extents && !r.failed; extent++ {
			if version != 0 {
				_ = r.uint(indexSize)
			}
			offset, length := r.uint(offsetSize), r.uint(lengthSize)
			if !selectedItem || r.failed {
				continue
			}
			source := file
			if method == 1 {
				source = idat
			}
			size := uint64(len(source))
			if base > size || offset > size-base || length > size-base-offset || length == 0 || length > uint64(metadataLimit-len(metadata)) {
				return nil
			}
			start := base + offset
			metadata = append(metadata, source[start:start+length]...)
		}
	}
	if !r.complete() {
		return nil
	}
	return metadata
}

type metadataReader struct {
	data   []byte
	failed bool
}

func (r *metadataReader) uint(size int) uint64 {
	if r.failed || size < 0 || size > 8 || len(r.data) < size {
		r.failed = true
		return 0
	}
	var value uint64
	for _, b := range r.data[:size] {
		value = value<<8 | uint64(b)
	}
	r.data = r.data[size:]
	return value
}

func (r *metadataReader) complete() bool { return !r.failed && len(r.data) == 0 }
