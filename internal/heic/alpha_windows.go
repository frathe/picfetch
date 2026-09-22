//go:build windows && (amd64 || arm64)

package heic

import (
	"fmt"
	"unsafe"

	"golang.org/x/sys/windows"
)

func windowsAlpha(data []byte) (bool, bool, error) {
	metadata, err := primaryAlpha(data)
	return metadata.item != 0, metadata.premultiplied, err
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
