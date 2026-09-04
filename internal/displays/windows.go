//go:build windows

package displays

import (
	"fmt"
	"image"
	"runtime"
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2/driver"

	"github.com/frathe/picfetch/internal/wincom"
)

type winRect struct {
	left, top, right, bottom int32
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

func platformInspect(context any) (found []Display, defaultID ID, err error) {
	window, ok := context.(driver.WindowsWindowContext)
	if !ok || window.HWND == 0 {
		return nil, "", &UnsupportedError{Reason: "not a Windows desktop window"}
	}

	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := procCoInitializeEx.Call(0, wincom.COInitApartmentThreaded)
	if wincom.FailedHRESULT(hr) {
		return nil, "", fmt.Errorf("initialize COM for display inspection: HRESULT 0x%08x", uint32(hr))
	}
	defer func() { _, _, _ = procCoUninitialize.Call() }()

	classID, interfaceID := wincom.DesktopWallpaperIDs()
	var api *wincom.DesktopWallpaper
	hr, _, _ = procCoCreateInstance.Call(
		uintptr(unsafe.Pointer(&classID)), 0, wincom.CLSCTXLocalServer,
		uintptr(unsafe.Pointer(&interfaceID)), uintptr(unsafe.Pointer(&api)),
	)
	if wincom.FailedHRESULT(hr) || api == nil {
		return nil, "", &UnsupportedError{Reason: fmt.Sprintf("IDesktopWallpaper unavailable (HRESULT 0x%08x)", uint32(hr))}
	}
	defer syscall.SyscallN(api.ReleaseProc(), uintptr(unsafe.Pointer(api)))

	var count uint32
	hr, _, _ = syscall.SyscallN(api.MonitorDevicePathCountProc(), uintptr(unsafe.Pointer(api)), uintptr(unsafe.Pointer(&count)))
	if wincom.FailedHRESULT(hr) {
		return nil, "", fmt.Errorf("count attached monitors: HRESULT 0x%08x", uint32(hr))
	}
	for index := uint32(0); index < count; index++ {
		var devicePath *uint16
		hr, _, _ = syscall.SyscallN(api.MonitorDevicePathAtProc(), uintptr(unsafe.Pointer(api)), uintptr(index), uintptr(unsafe.Pointer(&devicePath)))
		if wincom.FailedHRESULT(hr) || devicePath == nil {
			return nil, "", fmt.Errorf("read monitor %d device path: HRESULT 0x%08x", index, uint32(hr))
		}
		path := windowsUTF16String(devicePath)
		_, _, _ = procCoTaskMemFree.Call(uintptr(unsafe.Pointer(devicePath)))
		pathUTF16, conversionErr := syscall.UTF16PtrFromString(path)
		if conversionErr != nil {
			return nil, "", fmt.Errorf("prepare monitor %d device path: %w", index, conversionErr)
		}
		var bounds winRect
		hr, _, _ = syscall.SyscallN(api.MonitorRECTProc(), uintptr(unsafe.Pointer(api)), uintptr(unsafe.Pointer(pathUTF16)), uintptr(unsafe.Pointer(&bounds)))
		if wincom.FailedHRESULT(hr) {
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
