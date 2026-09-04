//go:build windows

package wallpaper

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/frathe/picfetch/internal/wincom"
)

type wallpaperRect struct {
	left, top, right, bottom int32
}

var (
	ole32Wallpaper          = syscall.NewLazyDLL("ole32.dll")
	coInitializeWallpaper   = ole32Wallpaper.NewProc("CoInitializeEx")
	coUninitializeWallpaper = ole32Wallpaper.NewProc("CoUninitialize")
	coCreateWallpaper       = ole32Wallpaper.NewProc("CoCreateInstance")
)

const (
	regdbEClassNotRegistered = 0x80040154
	eNoInterface             = 0x80004002
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

	hr, _, _ := coInitializeWallpaper.Call(0, wincom.COInitApartmentThreaded)
	if wincom.FailedHRESULT(hr) {
		return fmt.Errorf("initialize COM for targeted wallpaper: HRESULT 0x%08x", uint32(hr))
	}
	defer func() { _, _, _ = coUninitializeWallpaper.Call() }()

	classID, interfaceID := wincom.DesktopWallpaperIDs()
	var api *wincom.DesktopWallpaper
	hr, _, _ = coCreateWallpaper.Call(
		uintptr(unsafe.Pointer(&classID)), 0, wincom.CLSCTXLocalServer,
		uintptr(unsafe.Pointer(&interfaceID)), uintptr(unsafe.Pointer(&api)),
	)
	if wincom.FailedHRESULT(hr) || api == nil {
		if uint32(hr) == regdbEClassNotRegistered || uint32(hr) == eNoInterface {
			return &TargetUnsupportedError{Platform: "Windows", Reason: fmt.Sprintf("IDesktopWallpaper is unavailable (HRESULT 0x%08x)", uint32(hr))}
		}
		return fmt.Errorf("open IDesktopWallpaper: HRESULT 0x%08x", uint32(hr))
	}
	defer syscall.SyscallN(api.ReleaseProc(), uintptr(unsafe.Pointer(api)))

	return applyWindowsTarget(target, path,
		func(selected *uint16) error {
			var bounds wallpaperRect
			validateHR, _, _ := syscall.SyscallN(
				api.MonitorRECTProc(),
				uintptr(unsafe.Pointer(api)),
				uintptr(unsafe.Pointer(selected)),
				uintptr(unsafe.Pointer(&bounds)),
			)
			if wincom.FailedHRESULT(validateHR) {
				return fmt.Errorf("validate selected display: HRESULT 0x%08x", uint32(validateHR))
			}
			return nil
		},
		func(selected, imagePath *uint16) error {
			setHR, _, _ := syscall.SyscallN(
				api.SetWallpaperProc(),
				uintptr(unsafe.Pointer(api)),
				uintptr(unsafe.Pointer(selected)),
				uintptr(unsafe.Pointer(imagePath)),
			)
			if wincom.FailedHRESULT(setHR) {
				return fmt.Errorf("set wallpaper on selected display: HRESULT 0x%08x", uint32(setHR))
			}
			return nil
		},
	)
}

func applyWindowsTarget(target, path *uint16, validate func(*uint16) error, set func(*uint16, *uint16) error) error {
	// This injected seam proves that a detached or stale opaque ID is validated
	// before any Windows desktop mutation without touching a real desktop.
	if err := validate(target); err != nil {
		return err
	}

	return set(target, path)
}
