//go:build linux && cgo && (amd64 || arm64)

package heic

/*
#cgo LDFLAGS: -ldl
#include <stdlib.h>
#include "native_linux.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func nativeRead(data []byte, request Request) (Result, error) {
	if err := validateRequest(request, int64(len(data))); err != nil {
		return Result{}, err
	}
	transformed, err := inspectContainer(data)
	if err != nil {
		return Result{}, err
	}
	// Each platform keeps its own cgo request/allocation boundary.
	//goland:noinspection DuplicatedCode
	pixels := C.int(0)
	if request.Pixels {
		pixels = 1
	}
	decoded := C.picfetch_heic_read((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)), C.int64_t(request.MaxPixels), pixels)
	defer C.free(unsafe.Pointer(decoded.pixels))
	defer C.free(unsafe.Pointer(decoded.exif))
	if decoded.code != 0 {
		var message string = C.GoString(&decoded.message[0])
		switch decoded.code {
		case 1:
			return Result{}, fmt.Errorf("%w: %s", ErrUnavailable, message)
		case 2:
			return Result{}, fmt.Errorf("%w: %s", ErrUnsupported, message)
		case 3:
			return Result{}, fmt.Errorf("%w: %s", ErrInvalid, message)
		default:
			return Result{}, fmt.Errorf("HEIC native decode: %s", message)
		}
	}
	result := Result{Width: int(decoded.width), Height: int(decoded.height), Stride: int(decoded.width) * 4,
		Pixels: C.GoBytes(unsafe.Pointer(decoded.pixels), C.int(decoded.pixel_bytes)),
		EXIF:   C.GoBytes(unsafe.Pointer(decoded.exif), C.int(decoded.exif_bytes)), Provider: C.GoString(&decoded.provider[0])}
	result.EXIF = tiffMetadata(result.EXIF)
	if !transformed {
		orientResult(&result, exifOrientation(result.EXIF))
	}
	return result, nil
}
