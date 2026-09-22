package heic

import (
	"bytes"
	"encoding/binary"
)

type alphaMetadata struct {
	primary, item uint32
	premultiplied bool
}

// primaryAlpha checks references to the declared primary. Depth maps and other
// auxiliaries must not become alpha, and an unsupported alpha provider must not
// silently return an opaque image.
func primaryAlpha(data []byte) (alphaMetadata, error) {
	var primary uint64
	var references []byte
	err := walkBoxes(data, func(kind string, payload []byte) error {
		if kind != "meta" || len(payload) < 4 {
			return nil
		}
		return walkBoxes(payload[4:], func(kind string, payload []byte) error {
			switch kind {
			case "pitm":
				r := metadataReader{data: payload}
				version := r.uint(4)
				width := 2
				if version == 1<<24 {
					width = 4
				} else if version != 0 {
					return ErrInvalid
				}
				primary = r.uint(width)
				if !r.complete() {
					return ErrInvalid
				}
			case "iref":
				if references != nil {
					return ErrInvalid
				}
				references = payload
			}
			return nil
		})
	})
	if err != nil || references == nil {
		return alphaMetadata{}, err
	}
	r := metadataReader{data: references}
	version := r.uint(4)
	width := 2
	if version == 1<<24 {
		width = 4
	} else if version != 0 || r.failed {
		return alphaMetadata{}, ErrInvalid
	}
	auxiliary, premultiplied := make(map[uint32]bool), make(map[uint32]bool)
	err = walkBoxes(r.data, func(kind string, payload []byte) error {
		if kind != "auxl" && kind != "prem" {
			return nil
		}
		ref := metadataReader{data: payload}
		from, count := ref.uint(width), ref.uint(2)
		for i := uint64(0); i < count && !ref.failed; i++ {
			to := ref.uint(width)
			if kind == "auxl" && to == primary {
				auxiliary[uint32(from)] = true
			}
			if kind == "prem" && from == primary {
				premultiplied[uint32(to)] = true
			}
		}
		if !ref.complete() {
			return ErrInvalid
		}
		return nil
	})
	if err != nil {
		return alphaMetadata{}, err
	}
	if len(auxiliary) > 16 {
		return alphaMetadata{}, ErrUnsupported
	}
	var alpha uint32
	for id := range auxiliary {
		_, properties, err := itemProperties(data, id)
		if err != nil {
			return alphaMetadata{}, err
		}
		for _, property := range properties {
			if property.Kind != "auxC" {
				continue
			}
			if len(property.Data) < 5 {
				return alphaMetadata{}, ErrInvalid
			}
			if binary.BigEndian.Uint32(property.Data) != 0 {
				return alphaMetadata{}, ErrUnsupported
			}
			end := bytes.IndexByte(property.Data[4:], 0)
			if end < 0 {
				return alphaMetadata{}, ErrInvalid
			}
			if string(property.Data[4:4+end]) == "urn:mpeg:hevc:2015:auxid:1" {
				if alpha != 0 {
					return alphaMetadata{}, ErrUnsupported
				}
				alpha = id
			}
		}
	}
	return alphaMetadata{primary: uint32(primary), item: alpha, premultiplied: premultiplied[alpha]}, nil
}
