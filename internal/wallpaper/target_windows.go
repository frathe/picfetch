//go:build windows

package wallpaper

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"
)

type wallpaperGUID struct {
	data1 uint32
	data2 uint16
	data3 uint16
	data4 [8]byte
}

type wallpaperRect struct {
	left, top, right, bottom int32
}

type desktopWallpaperAPI struct {
	vtbl *desktopWallpaperVtbl
}

type desktopWallpaperVtbl struct {
	queryInterface, addRef, release                   uintptr
	setWallpaper, getWallpaper                        uintptr
	getMonitorDevicePathAt, getMonitorDevicePathCount uintptr
	getMonitorRECT                                    uintptr
}

var (
	ole32Wallpaper          = syscall.NewLazyDLL("ole32.dll")
	coInitializeWallpaper   = ole32Wallpaper.NewProc("CoInitializeEx")
	coUninitializeWallpaper = ole32Wallpaper.NewProc("CoUninitialize")
	coCreateWallpaper       = ole32Wallpaper.NewProc("CoCreateInstance")
)

var (
	clsidDesktopWallpaper = wallpaperGUID{0xc2cf3110, 0x460e, 0x4fc1, [8]byte{0xb9, 0xd0, 0x8a, 0x1c, 0x0c, 0x9c, 0xc4, 0xbd}}
	iidDesktopWallpaper   = wallpaperGUID{0xb92b56a9, 0x8b55, 0x4e14, [8]byte{0x9a, 0x89, 0x01, 0x99, 0xbb, 0xb6, 0xf9, 0x3b}}
)

const (
	coinitApartmentThreadedWallpaper = 0x2
	clsctxLocalServerWallpaper       = 0x4
	regdbEClassNotRegistered         = 0x80040154
	eNoInterface                     = 0x80004002
)

func setWindowsTarget(request Request) error {
	return setWindowsTargetWith(request, setWindowsTargetNative)
}

func setWindowsTargetWith(request Request, native func(*uint16, *uint16) error) error {
	target, err := syscall.UTF16PtrFromString(string(request.Target))
	if err != nil {
		return fmt.Errorf("prepare selected display ID: %w", err)
	}
	path, err := syscall.UTF16PtrFromString(request.Path)
	if err != nil {
		return fmt.Errorf("prepare wallpaper path: %w", err)
	}

	return native(target, path)
}

func setWindowsTargetNative(target, path *uint16) error {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	hr, _, _ := coInitializeWallpaper.Call(0, coinitApartmentThreadedWallpaper)
	if failedWallpaperHRESULT(hr) {
		return fmt.Errorf("initialize COM for targeted wallpaper: HRESULT 0x%08x", uint32(hr))
	}
	defer coUninitializeWallpaper.Call()

	var api *desktopWallpaperAPI
	hr, _, _ = coCreateWallpaper.Call(
		uintptr(unsafe.Pointer(&clsidDesktopWallpaper)), 0, clsctxLocalServerWallpaper,
		uintptr(unsafe.Pointer(&iidDesktopWallpaper)), uintptr(unsafe.Pointer(&api)),
	)
	if failedWallpaperHRESULT(hr) || api == nil {
		if uint32(hr) == regdbEClassNotRegistered || uint32(hr) == eNoInterface {
			return &TargetUnsupportedError{Platform: "Windows", Reason: fmt.Sprintf("IDesktopWallpaper is unavailable (HRESULT 0x%08x)", uint32(hr))}
		}
		return fmt.Errorf("open IDesktopWallpaper: HRESULT 0x%08x", uint32(hr))
	}
	defer syscall.SyscallN(api.vtbl.release, uintptr(unsafe.Pointer(api)))

	return applyWindowsTarget(target, path,
		func(selected *uint16) error {
			var bounds wallpaperRect
			validateHR, _, _ := syscall.SyscallN(
				api.vtbl.getMonitorRECT,
				uintptr(unsafe.Pointer(api)),
				uintptr(unsafe.Pointer(selected)),
				uintptr(unsafe.Pointer(&bounds)),
			)
			if failedWallpaperHRESULT(validateHR) {
				return fmt.Errorf("validate selected display: HRESULT 0x%08x", uint32(validateHR))
			}
			return nil
		},
		func(selected, imagePath *uint16) error {
			setHR, _, _ := syscall.SyscallN(
				api.vtbl.setWallpaper,
				uintptr(unsafe.Pointer(api)),
				uintptr(unsafe.Pointer(selected)),
				uintptr(unsafe.Pointer(imagePath)),
			)
			if failedWallpaperHRESULT(setHR) {
				return fmt.Errorf("set wallpaper on selected display: HRESULT 0x%08x", uint32(setHR))
			}
			return nil
		},
	)
}

func applyWindowsTarget(target, path *uint16, validate func(*uint16) error, set func(*uint16, *uint16) error) error {
	// Validation completes before SetWallpaper so a detached or stale opaque
	// ID cannot turn into a partial or global mutation.
	if err := validate(target); err != nil {
		return err
	}

	return set(target, path)
}

func failedWallpaperHRESULT(value uintptr) bool {
	return int32(uint32(value)) < 0
}
