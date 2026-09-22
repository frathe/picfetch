package heic

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

// walkBoxes checks sizes before slicing. It never interprets image bitstreams.
func walkBoxes(data []byte, visit func(string, []byte) error) error {
	for count := 0; len(data) != 0; count++ {
		if len(data) < 8 || count >= 65536 {
			return ErrInvalid
		}
		size := uint64(binary.BigEndian.Uint32(data))
		header := uint64(8)
		if size == 1 {
			if len(data) < 16 {
				return ErrInvalid
			}
			size, header = binary.BigEndian.Uint64(data[8:]), 16
		} else if size == 0 {
			size = uint64(len(data))
		}
		if size < header || size > uint64(len(data)) {
			return ErrInvalid
		}
		if err := visit(string(data[4:8]), data[header:size]); err != nil {
			return err
		}
		data = data[size:]
	}
	return nil
}

// inspectContainer establishes whether the designated primary has irot/imir
// properties. Only that fact controls EXIF fallback; dimensions never guess it.
func inspectContainer(data []byte) (bool, error) {
	_, properties, err := primaryItemProperties(data)
	if err != nil {
		return false, err
	}
	for _, property := range properties {
		if property.Kind == "irot" || property.Kind == "imir" {
			return true, nil
		}
	}
	return false, nil
}

type containerProperty struct {
	Kind string
	Data []byte
}

// primaryItemProperties returns only properties associated with pitm, retaining
// their payloads for provider-specific admission. It never parses HEVC data.
func primaryItemProperties(data []byte) (string, []containerProperty, error) {
	var meta []byte
	branded := false
	err := walkBoxes(data, func(kind string, payload []byte) error {
		switch kind {
		case "moov":
			return fmt.Errorf("%w: image sequences are not supported", ErrUnsupported)
		case "meta":
			if meta != nil || len(payload) < 4 || binary.BigEndian.Uint32(payload) != 0 {
				return ErrInvalid
			}
			meta = payload[4:]
		case "ftyp":
			if len(payload) < 8 || len(payload)%4 != 0 {
				return ErrInvalid
			}
			for offset := 0; offset < len(payload); offset += 4 {
				if offset == 4 {
					continue
				}
				switch string(payload[offset : offset+4]) {
				case "heic", "heix", "mif1":
					branded = true
				case "hevc", "hevx", "msf1", "hevm", "hevs", "heim", "heis", "avif", "avis":
					return ErrUnsupported
				}
			}
		}
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	if !branded || meta == nil {
		return "", nil, ErrUnsupported
	}
	var primary uint32
	var properties []byte
	var itemInfo []byte
	err = walkBoxes(meta, func(kind string, payload []byte) error {
		switch kind {
		case "pitm":
			if len(payload) < 6 || primary != 0 {
				return ErrInvalid
			}
			switch payload[0] {
			case 0:
				primary = uint32(binary.BigEndian.Uint16(payload[4:]))
			case 1:
				if len(payload) < 8 {
					return ErrInvalid
				}
				primary = binary.BigEndian.Uint32(payload[4:])
			default:
				return ErrUnsupported
			}
		case "iprp":
			if properties != nil {
				return ErrInvalid
			}
			properties = payload
		case "iinf":
			if itemInfo != nil {
				return ErrInvalid
			}
			itemInfo = payload
		}
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	if primary == 0 || properties == nil {
		return "", nil, ErrInvalid
	}
	itemType, err := primaryItemType(itemInfo, primary)
	if err != nil {
		return "", nil, err
	}
	var allProperties []containerProperty
	var associations [][]byte
	err = walkBoxes(properties, func(kind string, payload []byte) error {
		switch kind {
		case "ipco":
			if allProperties != nil {
				return ErrInvalid
			}
			allProperties = []containerProperty{}
			return walkBoxes(payload, func(kind string, data []byte) error {
				allProperties = append(allProperties, containerProperty{Kind: kind, Data: data})
				return nil
			})
		case "ipma":
			associations = append(associations, payload)
		}
		return nil
	})
	if err != nil {
		return "", nil, err
	}
	var selected []containerProperty
	seen := make(map[int]bool)
	for _, table := range associations {
		indices, err := primaryPropertyIndices(table, primary, len(allProperties))
		if err != nil {
			return "", nil, err
		}
		for _, index := range indices {
			if seen[index] {
				return "", nil, ErrInvalid
			}
			seen[index] = true
			selected = append(selected, allProperties[index-1])
		}
	}
	return itemType, selected, nil
}

func primaryPropertyIndices(table []byte, primary uint32, propertyCount int) ([]int, error) {
	if len(table) < 8 || table[0] > 1 {
		return nil, ErrInvalid
	}
	version, wide := table[0], table[3]&1 != 0
	count := binary.BigEndian.Uint32(table[4:])
	table = table[8:]
	var selected []int
	for range count {
		idBytes := 2
		if version == 1 {
			idBytes = 4
		}
		if len(table) < idBytes+1 {
			return nil, ErrInvalid
		}
		id := uint32(binary.BigEndian.Uint16(table))
		if idBytes == 4 {
			id = binary.BigEndian.Uint32(table)
		}
		n := int(table[idBytes])
		table = table[idBytes+1:]
		for range n {
			propertyBytes := 1
			if wide {
				propertyBytes = 2
			}
			if len(table) < propertyBytes {
				return nil, ErrInvalid
			}
			property := int(table[0] & 0x7f)
			if wide {
				property = int(binary.BigEndian.Uint16(table) & 0x7fff)
			}
			table = table[propertyBytes:]
			if property > propertyCount {
				return nil, ErrInvalid
			}
			if id == primary && property != 0 {
				selected = append(selected, property)
			}
		}
	}
	if len(table) != 0 {
		return nil, ErrInvalid
	}
	return selected, nil
}

func primaryItemType(info []byte, primary uint32) (string, error) {
	// The earlier transform-only inspection did not require iinf. An absent
	// type stays unknown; a provider that requires it refuses that input.
	if info == nil {
		return "", nil
	}
	var itemType string
	err := walkItemInfo(info, func(id uint32, kind string, protected bool) error {
		if id == primary {
			if protected {
				return ErrUnsupported
			}
			itemType = kind
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	if itemType == "" {
		return "", ErrInvalid
	}
	return itemType, nil
}

func walkItemInfo(info []byte, visit func(uint32, string, bool) error) error {
	if len(info) < 6 || info[0] > 1 {
		return ErrInvalid
	}
	count := uint32(binary.BigEndian.Uint16(info[4:]))
	header := 6
	if info[0] == 1 {
		if len(info) < 8 {
			return ErrInvalid
		}
		count, header = binary.BigEndian.Uint32(info[4:]), 8
	}
	seen := make(map[uint32]bool)
	err := walkBoxes(info[header:], func(kind string, payload []byte) error {
		if kind != "infe" || len(payload) < 4 {
			return ErrInvalid
		}
		idBytes := 2
		if payload[0] == 3 {
			idBytes = 4
		} else if payload[0] != 2 {
			return ErrUnsupported
		}
		typeOffset := 4 + idBytes + 2
		if len(payload) < typeOffset+5 || bytes.IndexByte(payload[typeOffset+4:], 0) < 0 {
			return ErrInvalid
		}
		id := uint32(binary.BigEndian.Uint16(payload[4:]))
		if idBytes == 4 {
			id = binary.BigEndian.Uint32(payload[4:])
		}
		if id == 0 || seen[id] {
			return ErrInvalid
		}
		seen[id] = true
		return visit(id, string(payload[typeOffset:typeOffset+4]), binary.BigEndian.Uint16(payload[4+idBytes:]) != 0)
	})
	if err != nil {
		return err
	}
	if uint32(len(seen)) != count {
		return ErrInvalid
	}
	return nil
}

func tiffMetadata(metadata []byte) []byte {
	if len(metadata) < 4 {
		return nil
	}
	offset := uint64(binary.BigEndian.Uint32(metadata)) + 4
	if offset > uint64(len(metadata)) {
		return nil
	}
	return metadata[offset:]
}

func exifOrientation(tiff []byte) int {
	if len(tiff) < 8 {
		return 1
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 1
	}
	if order.Uint16(tiff[2:]) != 42 {
		return 1
	}
	offset := uint64(order.Uint32(tiff[4:]))
	if offset+2 > uint64(len(tiff)) {
		return 1
	}
	count := uint64(order.Uint16(tiff[offset:]))
	offset += 2
	if count*12 > uint64(len(tiff))-offset {
		return 1
	}
	for index := range count {
		entry := tiff[offset+index*12:]
		if order.Uint16(entry) == 0x112 && order.Uint16(entry[2:]) == 3 && order.Uint32(entry[4:]) == 1 {
			orientation := int(order.Uint16(entry[8:]))
			if orientation >= 1 && orientation <= 8 {
				return orientation
			}
		}
	}
	return 1
}

func orientResult(result *Result, orientation int) {
	if orientation <= 1 || orientation > 8 {
		return
	}
	width, height, oldStride := result.Width, result.Height, result.Stride
	if orientation >= 5 {
		result.Width, result.Height = height, width
		result.Stride = result.Width * 4
	}
	if len(result.Pixels) == 0 {
		return
	}
	pixels := make([]byte, len(result.Pixels))
	for y := 0; y < result.Height; y++ {
		for x := 0; x < result.Width; x++ {
			sx, sy := x, y
			switch orientation {
			case 2:
				sx = width - 1 - x
			case 3:
				sx, sy = width-1-x, height-1-y
			case 4:
				sy = height - 1 - y
			case 5:
				sx, sy = y, x
			case 6:
				sx, sy = y, height-1-x
			case 7:
				sx, sy = width-1-y, height-1-x
			default: // The remaining validated orientation is 8.
				sx, sy = width-1-y, x
			}
			copy(pixels[y*result.Stride+x*4:][:4], result.Pixels[sy*oldStride+sx*4:][:4])
		}
	}
	result.Pixels = pixels
}
