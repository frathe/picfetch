//go:build windows && (amd64 || arm64)

package heic

import (
	"encoding/binary"
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/frathe/picfetch/internal/wincom"
)

// nativeRead runs only inside the bounded worker. Direct Microsoft class
// activation avoids registry arbitration among third-party HEIF codecs.
func nativeRead(data []byte, request Request) (Result, error) {
	if err := validateRequest(request, int64(len(data))); err != nil {
		return Result{}, err
	}
	itemType, properties, err := primaryItemProperties(data)
	if err != nil {
		return Result{}, err
	}
	if itemType != "hvc1" && itemType != "grid" {
		return Result{}, ErrUnsupported
	}
	transformed := false
	for _, property := range properties {
		switch property.Kind {
		case "ispe":
			if len(property.Data) != 12 {
				return Result{}, ErrInvalid
			}
			width, height := binary.BigEndian.Uint32(property.Data[4:]), binary.BigEndian.Uint32(property.Data[8:])
			if width == 0 || height == 0 || uint64(width)*uint64(height) > uint64(request.MaxPixels) {
				return Result{}, fmt.Errorf("%w: declared dimensions exceed pixel limit", ErrInvalid)
			}
		case "irot", "imir":
			transformed = true
		}
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	ole := windows.NewLazySystemDLL("ole32.dll")
	if err := wicCall("initialize COM", ErrUnavailable, ole.NewProc("CoInitializeEx").Addr(), 0, wincom.COInitApartmentThreaded); err != nil {
		return Result{}, err
	}
	defer func() { _, _, _ = ole.NewProc("CoUninitialize").Call() }()
	create := ole.NewProc("CoCreateInstance").Addr()
	factoryClass := windows.GUID{Data1: 0xcacaf262, Data2: 0x9370, Data3: 0x4615, Data4: [8]byte{0xa1, 0x3b, 0x9f, 0x55, 0x39, 0xda, 0x4c, 0x0a}}
	factoryID := windows.GUID{Data1: 0xec5ec8a9, Data2: 0xc395, Data3: 0x4314, Data4: [8]byte{0x9c, 0x77, 0x54, 0xd7, 0xa9, 0x35, 0xff, 0x70}}
	var factory *wicFactory
	if err := wicCall("create factory", ErrUnavailable, create, uintptr(unsafe.Pointer(&factoryClass)), 0, 1, uintptr(unsafe.Pointer(&factoryID)), uintptr(unsafe.Pointer(&factory))); err != nil {
		return Result{}, err
	}
	if factory == nil {
		return Result{}, ErrUnavailable
	}
	defer wicRelease(unsafe.Pointer(factory))
	var stream *wicStream
	if err := wicCall("create stream", ErrUnavailable, factory.vtable.createStream, uintptr(unsafe.Pointer(factory)), uintptr(unsafe.Pointer(&stream))); err != nil {
		return Result{}, err
	}
	if stream == nil {
		return Result{}, ErrUnavailable
	}
	var pinned runtime.Pinner
	pinned.Pin(&data[0])
	defer pinned.Unpin()
	defer wicRelease(unsafe.Pointer(stream))
	if err := wicCall("initialize stream", ErrInvalid, stream.vtable.initializeFromMemory, uintptr(unsafe.Pointer(stream)), uintptr(unsafe.Pointer(&data[0])), uintptr(len(data))); err != nil {
		return Result{}, err
	}
	decoderClass := windows.GUID{Data1: 0xe9a4a80a, Data2: 0x44fe, Data3: 0x4de4, Data4: [8]byte{0x89, 0x71, 0x71, 0x50, 0xb1, 0x0a, 0x51, 0x99}}
	decoderID := windows.GUID{Data1: 0x9edde9e7, Data2: 0x8dee, Data3: 0x47ea, Data4: [8]byte{0x99, 0xdf, 0xe6, 0xfa, 0xf2, 0xed, 0x44, 0xbf}}
	var decoder *wicDecoder
	if err := wicCall("activate Microsoft HEIF decoder", ErrUnavailable, create, uintptr(unsafe.Pointer(&decoderClass)), 0, 1, uintptr(unsafe.Pointer(&decoderID)), uintptr(unsafe.Pointer(&decoder))); err != nil {
		return Result{}, err
	}
	if decoder == nil {
		return Result{}, ErrUnavailable
	}
	defer wicRelease(unsafe.Pointer(decoder))
	if err := wicCall("initialize HEIF decoder", ErrUnsupported, decoder.vtable.initialize, uintptr(unsafe.Pointer(decoder)), uintptr(unsafe.Pointer(stream)), 0); err != nil {
		return Result{}, err
	}
	provider, err := wicProvider(decoder, decoderClass)
	if err != nil {
		return Result{}, err
	}
	// Microsoft's HEIF decoder presents the designated primary at frame zero.
	// Native tests vary pitm independently of storage order and item identifiers.
	var frame *wicFrame
	if err := wicCall("get primary frame", ErrUnsupported, decoder.vtable.getFrame, uintptr(unsafe.Pointer(decoder)), 0, uintptr(unsafe.Pointer(&frame))); err != nil {
		return Result{}, err
	}
	if frame == nil {
		return Result{}, ErrInvalid
	}
	defer wicRelease(unsafe.Pointer(frame))
	result, err := wicPixels(frame, request)
	if err != nil {
		return Result{}, err
	}
	result.Provider, result.EXIF = provider, primaryEXIF(data)
	alpha, premultiplied, err := windowsAlpha(data)
	if err != nil {
		return Result{}, err
	}
	if alpha {
		if err := wicApplyAlpha(frame, request, &result, premultiplied); err != nil {
			return Result{}, err
		}
	}
	orientation := exifOrientation(result.EXIF)
	if transformed {
		orientation, err = wicOrientation(frame)
		if err != nil {
			return Result{}, err
		}
	}
	orientResult(&result, orientation)
	runtime.KeepAlive(data)
	return result, nil
}

func wicPixels(frame *wicFrame, request Request) (Result, error) {
	var width, height uint32
	if err := wicCall("get dimensions", ErrInvalid, frame.vtable.getSize, uintptr(unsafe.Pointer(frame)), uintptr(unsafe.Pointer(&width)), uintptr(unsafe.Pointer(&height))); err != nil {
		return Result{}, err
	}
	if width == 0 || height == 0 || uint64(width)*uint64(height) > uint64(request.MaxPixels) {
		return Result{}, fmt.Errorf("%w: WIC dimensions exceed pixel limit", ErrInvalid)
	}
	result := Result{Width: int(width), Height: int(height), Stride: int(width) * 4}
	if !request.Pixels {
		return result, nil
	}
	rgba := windows.GUID{Data1: 0xf5c7ad2d, Data2: 0x6a8d, Data3: 0x43dd, Data4: [8]byte{0xa7, 0xa8, 0xa2, 0x99, 0x35, 0x26, 0x1a, 0xe9}}
	var converted *wicSource
	convert := windows.NewLazySystemDLL("windowscodecs.dll").NewProc("WICConvertBitmapSource")
	if err := wicCall("convert to straight RGBA", ErrUnsupported, convert.Addr(), uintptr(unsafe.Pointer(&rgba)), uintptr(unsafe.Pointer(frame)), uintptr(unsafe.Pointer(&converted))); err != nil {
		return Result{}, err
	}
	if converted == nil {
		return Result{}, ErrInvalid
	}
	defer wicRelease(unsafe.Pointer(converted))
	result.Pixels = make([]byte, result.Stride*result.Height)
	if err := wicCall("copy pixels", ErrUnsupported, converted.vtable.copyPixels, uintptr(unsafe.Pointer(converted)), 0, uintptr(result.Stride), uintptr(len(result.Pixels)), uintptr(unsafe.Pointer(&result.Pixels[0]))); err != nil {
		return Result{}, err
	}
	return result, nil
}

func wicOrientation(frame *wicFrame) (int, error) {
	var reader *wicMetadataReader
	if err := wicCall("get orientation reader", ErrUnsupported, frame.vtable.getMetadataQueryReader, uintptr(unsafe.Pointer(frame)), uintptr(unsafe.Pointer(&reader))); err != nil {
		return 0, err
	}
	if reader == nil {
		return 0, ErrInvalid
	}
	defer wicRelease(unsafe.Pointer(reader))
	query, err := windows.UTF16PtrFromString("/heifProps/Orientation")
	if err != nil {
		return 0, err
	}
	// PROPVARIANT is 24 bytes on both supported 64-bit Windows architectures.
	var value struct {
		kind     uint16
		reserved [3]uint16
		data     [2]uint64
	}
	defer func() {
		_, _, _ = windows.NewLazySystemDLL("ole32.dll").NewProc("PropVariantClear").Call(uintptr(unsafe.Pointer(&value)))
	}()
	if err := wicCall("read HEIF orientation", ErrUnsupported, reader.vtable.getMetadataByName, uintptr(unsafe.Pointer(reader)), uintptr(unsafe.Pointer(query)), uintptr(unsafe.Pointer(&value))); err != nil {
		return 0, err
	}
	if value.kind != 18 || value.data[0] < 1 || value.data[0] > 8 { // VT_UI2
		return 0, fmt.Errorf("%w: invalid HEIF orientation", ErrInvalid)
	}
	return int(value.data[0]), nil
}

func wicProvider(decoder *wicDecoder, expected windows.GUID) (string, error) {
	var info *wicInfo
	if err := wicCall("get decoder identity", ErrUnavailable, decoder.vtable.getDecoderInfo, uintptr(unsafe.Pointer(decoder)), uintptr(unsafe.Pointer(&info))); err != nil {
		return "", err
	}
	if info == nil {
		return "", ErrUnavailable
	}
	defer wicRelease(unsafe.Pointer(info))
	var classID, vendor windows.GUID
	if err := wicCall("get decoder class", ErrUnavailable, info.vtable.getCLSID, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(&classID))); err != nil {
		return "", err
	}
	if err := wicCall("get decoder vendor", ErrUnavailable, info.vtable.getVendorGUID, uintptr(unsafe.Pointer(info)), uintptr(unsafe.Pointer(&vendor))); err != nil {
		return "", err
	}
	microsoft := windows.GUID{Data1: 0xf0e749ca, Data2: 0xedef, Data3: 0x4589, Data4: [8]byte{0xa7, 0x3a, 0xee, 0x0e, 0x62, 0x6a, 0x2a, 0x2b}}
	if classID != expected || vendor != microsoft {
		return "", fmt.Errorf("%w: unexpected HEIF provider", ErrUnavailable)
	}
	var version [256]uint16
	var written uint32
	if err := wicCall("get decoder version", ErrUnavailable, info.vtable.getVersion, uintptr(unsafe.Pointer(info)), uintptr(len(version)), uintptr(unsafe.Pointer(&version[0])), uintptr(unsafe.Pointer(&written))); err != nil {
		return "", err
	}
	return "Microsoft HEIF WIC " + windows.UTF16ToString(version[:]) + "; Windows " + systemVersion(), nil
}

//go:uintptrescapes
func wicCall(name string, cause error, function uintptr, arguments ...uintptr) error {
	hr, _, _ := syscall.SyscallN(function, arguments...)
	if !wincom.FailedHRESULT(hr) {
		return nil
	}
	switch uint32(hr) {
	case 0x80040154, 0x88982f50, 0xc00d5212: // Class/component/codec not found.
		cause = ErrUnavailable
	}
	return fmt.Errorf("%w: %s: HRESULT 0x%08x", cause, name, uint32(hr))
}

func wicRelease(object unsafe.Pointer) {
	if object != nil {
		_, _, _ = syscall.SyscallN((*wicUnknown)(object).vtable.release, uintptr(object))
	}
}
