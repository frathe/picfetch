//go:build windows && (amd64 || arm64)

package heic

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

// windowsAlpha checks references to the declared primary. Depth maps and other
// auxiliaries must not become alpha, and an unsupported alpha provider must not
// silently return an opaque image.
func windowsAlpha(data []byte) (bool, bool, error) {
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
		return false, false, err
	}
	r := metadataReader{data: references}
	version := r.uint(4)
	width := 2
	if version == 1<<24 {
		width = 4
	} else if version != 0 || r.failed {
		return false, false, ErrInvalid
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
		return false, false, err
	}
	if len(auxiliary) > 16 {
		return false, false, ErrUnsupported
	}
	var alpha uint32
	for id := range auxiliary {
		_, properties, err := itemProperties(data, id)
		if err != nil {
			return false, false, err
		}
		for _, property := range properties {
			if property.Kind != "auxC" {
				continue
			}
			if len(property.Data) < 5 {
				return false, false, ErrInvalid
			}
			if binary.BigEndian.Uint32(property.Data) != 0 {
				return false, false, ErrUnsupported
			}
			end := bytes.IndexByte(property.Data[4:], 0)
			if end < 0 {
				return false, false, ErrInvalid
			}
			if string(property.Data[4:4+end]) == "urn:mpeg:hevc:2015:auxid:1" {
				if alpha != 0 {
					return false, false, ErrUnsupported
				}
				alpha = id
			}
		}
	}
	return alpha != 0, premultiplied[alpha], nil
}

func wicApplyAlpha(frame *wicFrame, request Request, result *Result, premultiplied bool) error {
	id := windows.GUID{Data1: 0x0c599495, Data2: 0xa120, Data3: 0x4222, Data4: [8]byte{0x91, 0x30, 0xa8, 0xc2, 0x94, 0x10, 0xbd, 0x0b}}
	var reader *wicChainReader
	if err := wicCall("get HEIF alpha reader", ErrUnsupported, frame.vtable.queryInterface, uintptr(unsafe.Pointer(frame)), uintptr(unsafe.Pointer(&id)), uintptr(unsafe.Pointer(&reader))); err != nil {
		return err
	}
	if reader == nil {
		return ErrInvalid
	}
	defer wicRelease(unsafe.Pointer(reader))
	const alphaMap = 5 // WICBitmapChainType_AlphaMap
	var count uint32
	if err := wicCall("get alpha count", ErrUnsupported, reader.vtable.getChainedFrameCount, uintptr(unsafe.Pointer(reader)), alphaMap, uintptr(unsafe.Pointer(&count))); err != nil {
		return err
	}
	if count != 1 {
		return fmt.Errorf("%w: HEIF alpha plane is unavailable or ambiguous", ErrUnsupported)
	}
	var alpha *wicFrame
	if err := wicCall("get alpha plane", ErrUnsupported, reader.vtable.getChainedFrame, uintptr(unsafe.Pointer(reader)), alphaMap, 0, uintptr(unsafe.Pointer(&alpha))); err != nil {
		return err
	}
	if alpha == nil {
		return ErrInvalid
	}
	defer wicRelease(unsafe.Pointer(alpha))
	plane, err := wicPixels(alpha, request)
	if err != nil {
		return err
	}
	if plane.Width != result.Width || plane.Height != result.Height {
		return fmt.Errorf("%w: alpha geometry differs from primary", ErrUnsupported)
	}
	for i := 0; i < len(result.Pixels); i += 4 {
		opacity := plane.Pixels[i]
		result.Pixels[i+3] = opacity
		if premultiplied {
			for channel := range 3 {
				value := 0
				if opacity != 0 {
					value = min(255, (int(result.Pixels[i+channel])*255+int(opacity)/2)/int(opacity))
				}
				result.Pixels[i+channel] = byte(value)
			}
		}
	}
	return nil
}
