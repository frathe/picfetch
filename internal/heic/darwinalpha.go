package heic

import (
	"bytes"
	"encoding/binary"
	"math"
)

// darwinAlphaInput lets ImageIO compose the primary's auxiliary alpha plane.
// ImageIO otherwise returns opaque pixels when that primary has a prem
// reference. Removing that reference makes it render the stored color channels
// as straight; the caller must then undo their original premultiplication.
// No item data or offsets change, and all other references remain intact.
func darwinAlphaInput(data []byte) ([]byte, bool, error) {
	alpha, err := primaryAlpha(data)
	if err != nil || !alpha.premultiplied {
		return data, false, err
	}
	prepared := bytes.Clone(data)
	err = walkRawBoxes(prepared, func(kind string, _ []byte, payload []byte) error {
		if kind != "meta" {
			return nil
		}
		if len(payload) < 4 {
			return ErrInvalid
		}
		return walkRawBoxes(payload[4:], func(kind string, raw, payload []byte) error {
			if kind != "iref" {
				return nil
			}
			width := 2
			if payload[0] == 1 {
				width = 4
			}
			used := 4
			if err := walkRawBoxes(payload[4:], func(kind string, raw, body []byte) error {
				if kind == "prem" {
					reader := metadataReader{data: body}
					if reader.uint(width) == uint64(alpha.primary) && !reader.failed {
						return nil
					}
				}
				used += copy(payload[used:], raw)
				return nil
			}); err != nil {
				return err
			}
			header := len(raw) - len(payload)
			size := header + used
			removed := len(raw) - size
			if removed == 0 {
				return nil
			}
			if removed < 8 || uint64(removed) > math.MaxUint32 || header == 8 && uint64(size) > math.MaxUint32 {
				return ErrUnsupported
			}
			if header == 16 {
				binary.BigEndian.PutUint64(raw[8:], uint64(size))
			} else {
				binary.BigEndian.PutUint32(raw, uint32(size))
			}
			clear(raw[size:])
			binary.BigEndian.PutUint32(raw[size:], uint32(removed))
			copy(raw[size+4:], "free")
			return nil
		})
	})
	return prepared, err == nil, err
}

// walkRawBoxes retains each validated header for size-preserving metadata edits.
func walkRawBoxes(data []byte, visit func(string, []byte, []byte) error) error {
	offset := 0
	return walkBoxes(data, func(kind string, payload []byte) error {
		header := 8
		if binary.BigEndian.Uint32(data[offset:]) == 1 {
			header = 16
		}
		end := offset + header + len(payload)
		raw := data[offset:end]
		offset = end
		return visit(kind, raw, payload)
	})
}

func undoPremultiplication(pixels []byte) {
	for offset := 0; offset+4 <= len(pixels); offset += 4 {
		pixel := pixels[offset : offset+4]
		for channel := range 3 {
			value := 0
			if pixel[3] != 0 {
				value = min(255, (int(pixel[channel])*255+int(pixel[3])/2)/int(pixel[3]))
			}
			pixel[channel] = byte(value)
		}
	}
}
