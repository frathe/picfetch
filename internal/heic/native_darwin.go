//go:build darwin && cgo && (amd64 || arm64)

package heic

/*
#cgo LDFLAGS: -framework ImageIO -framework CoreGraphics -framework CoreFoundation
#include <stdlib.h>
#include "native_darwin.h"
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// nativeRead is entered only by WorkerMain in the bounded native child.
func nativeRead(data []byte, request Request) (Result, error) {
	if err := validateRequest(request, int64(len(data))); err != nil {
		return Result{}, err
	}
	if _, err := inspectContainer(data); err != nil {
		return Result{}, err
	}
	if err := validateDarwinContainer(data); err != nil {
		return Result{}, err
	}
	pixels := C.int(0)
	if request.Pixels {
		pixels = 1
	}
	decoded := C.picfetch_imageio_read((*C.uint8_t)(unsafe.Pointer(&data[0])), C.size_t(len(data)), C.int64_t(request.MaxPixels), pixels)
	defer C.free(unsafe.Pointer(decoded.pixels))
	if decoded.code != 0 {
		message := C.GoString(&decoded.message[0])
		switch decoded.code {
		case 1:
			return Result{}, fmt.Errorf("%w: %s", ErrUnavailable, message)
		case 2:
			return Result{}, fmt.Errorf("%w: %s", ErrUnsupported, message)
		case 3:
			return Result{}, fmt.Errorf("%w: %s", ErrInvalid, message)
		default:
			return Result{}, fmt.Errorf("ImageIO native decode: %s", message)
		}
	}
	result := Result{Width: int(decoded.width), Height: int(decoded.height), Stride: int(decoded.width) * 4,
		Pixels:   C.GoBytes(unsafe.Pointer(decoded.pixels), C.int(decoded.pixel_bytes)),
		Provider: C.GoString(&decoded.provider[0]) + "; Darwin " + systemVersion()}
	// ImageIO reports one intended-display orientation. Apply that value once;
	// neither a thumbnail transform nor a second EXIF fallback is requested.
	orientResult(&result, int(decoded.orientation))
	return result, nil
}
