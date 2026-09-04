//go:build windows

package displays

import (
	"fmt"
	"image"
	"runtime"
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2/driver"
)

type winGUID struct {
	data1 uint32
	data2 uint16
	data3 uint16
	data4 [8]byte
}

type winRect struct {
	left, top, right, bottom int32
}

type desktopWallpaper struct {
	vtbl *desktopWallpaperVtbl
}

type desktopWallpaperVtbl struct {
	queryInterface, addRef, release                   uintptr
	setWallpaper, getWallpaper                        uintptr
	getMonitorDevicePathAt, getMonitorDevicePathCount uintptr
	getMonitorRECT                                    uintptr
}

var (
	ole32Display              = syscall.NewLazyDLL("ole32.dll")
	user32Display             = syscall.NewLazyDLL("user32.dll")
	procCoInitializeEx        = ole32Display.NewProc("CoInitializeEx")
	procCoUninitialize        = ole32Display.NewProc("CoUninitialize")
	procCoCreateInstance      = ole32Display.NewProc("CoCreateInstance")
	procCoTaskMemFree         = ole32Display.NewProc("CoTaskMemFree")
	procGetWindowRectDisplays = user32Display.NewProc("GetWindowRect")
)

var (
	clsidDesktopWallpaper = winGUID{0xc2cf3110, 0x460e, 0x4fc1, [8]byte{0xb9, 0xd0, 0x8a, 0x1c, 0x0c, 0x9c, 0xc4, 0xbd}}
	iidDesktopWallpaper   = winGUID{0xb92b56a9, 0x8b55, 0x4e14, [8]byte{0x9a, 0x89, 0x01, 0x99, 0xbb, 0xb6, 0xf9, 0x3b}}
)

const (
	coinitApartmentThreaded = 0x2
	clsctxLocalServer       = 0x4
)

func platformInspect(context any) (found []Display, defaultID ID, err error) {
	window, ok := context.(driver.WindowsWindowContext)
	if !ok || window.HWND == 0 {
		return nil, "", &UnsupportedError{Reason: "not a Windows desktop window"}
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := procCoInitializeEx.Call(0, coinitApartmentThreaded)
	if failedHRESULT(hr) {
		return nil, "", fmt.Errorf("initialize COM for display inspection: HRESULT 0x%08x", uint32(hr))
	}
	defer procCoUninitialize.Call()

	var api *desktopWallpaper
	hr, _, _ = procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&clsidDesktopWallpaper)), 0, clsctxLocalServer,
		uintptr(unsafe.Pointer(&iidDesktopWallpaper)), uintptr(unsafe.Pointer(&api)),
	)
	if failedHRESULT(hr) || api == nil {
		return nil, "", &UnsupportedError{Reason: fmt.Sprintf("IDesktopWallpaper unavailable (HRESULT 0x%08x)", uint32(hr))}
	}
	defer syscall.SyscallN(api.vtbl.release, uintptr(unsafe.Pointer(api)))

	var count uint32
	hr, _, _ = syscall.SyscallN(api.vtbl.getMonitorDevicePathCount, uintptr(unsafe.Pointer(api)), uintptr(unsafe.Pointer(&count)))
	if failedHRESULT(hr) {
		return nil, "", fmt.Errorf("count attached monitors: HRESULT 0x%08x", uint32(hr))
	}
	for index := uint32(0); index < count; index++ {
		var devicePath *uint16
		hr, _, _ = syscall.SyscallN(api.vtbl.getMonitorDevicePathAt, uintptr(unsafe.Pointer(api)), uintptr(index), uintptr(unsafe.Pointer(&devicePath)))
		if failedHRESULT(hr) || devicePath == nil {
			return nil, "", fmt.Errorf("read monitor %d device path: HRESULT 0x%08x", index, uint32(hr))
		}
		path := windowsUTF16String(devicePath)
		procCoTaskMemFree.Call(uintptr(unsafe.Pointer(devicePath)))
		pathUTF16, conversionErr := syscall.UTF16PtrFromString(path)
		if conversionErr != nil {
			return nil, "", fmt.Errorf("prepare monitor %d device path: %w", index, conversionErr)
		}
		var bounds winRect
		hr, _, _ = syscall.SyscallN(api.vtbl.getMonitorRECT, uintptr(unsafe.Pointer(api)), uintptr(unsafe.Pointer(pathUTF16)), uintptr(unsafe.Pointer(&bounds)))
		if failedHRESULT(hr) {
			return nil, "", fmt.Errorf("read monitor %d bounds: HRESULT 0x%08x", index, uint32(hr))
		}
		found = append(found, Display{
			ID:     ID(path),
			Name:   fmt.Sprintf("Display %d", index+1),
			Bounds: image.Rect(int(bounds.left), int(bounds.top), int(bounds.right), int(bounds.bottom)),
		})
	}

	var windowBounds winRect
	windowKnown, _, _ := procGetWindowRectDisplays.Call(window.HWND, uintptr(unsafe.Pointer(&windowBounds)))
	defaultID = defaultForWindow(found,
		image.Rect(int(windowBounds.left), int(windowBounds.top), int(windowBounds.right), int(windowBounds.bottom)),
		windowKnown != 0)

	return found, defaultID, nil
}

func windowsUTF16String(value *uint16) string {
	if value == nil {
		return ""
	}
	units := unsafe.Slice(value, 32768)
	length := 0
	for length < len(units) && units[length] != 0 {
		length++
	}

	return syscall.UTF16ToString(units[:length])
}

func failedHRESULT(value uintptr) bool {
	return int32(uint32(value)) < 0
}
