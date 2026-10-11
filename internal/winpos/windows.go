//go:build windows

package winpos

import (
	"strconv"
	"syscall"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

var (
	user32             = syscall.NewLazyDLL("user32.dll")
	procClientToScreen = user32.NewProc("ClientToScreen")
	procShowWindow     = user32.NewProc("ShowWindow")
	procIsZoomed       = user32.NewProc("IsZoomed")
)

// swMaximize is SW_MAXIMIZE, the nCmdShow value ShowWindow uses to activate
// a window and size it to fill the work area - exactly what clicking the
// window's own maximize button does.
const swMaximize = 3

// swRestore is SW_RESTORE, the nCmdShow value that undoes swMaximize -
// activating the window and returning it to its pre-maximize size and
// position, exactly what clicking the window's own restore button does.
const swRestore = 9

type point struct {
	x, y int32
}

// platformPosition mirrors what Fyne's glfw driver itself does for its own
// position bookkeeping: ClientToScreen on the client area's (0,0) corner,
// not GetWindowRect, which would include the non-client window frame/border
// and drift from the coordinates RequestPosition/SetWindowPos actually
// place the window at.
func platformPosition(ctx any) (x, y int, ok bool) {
	win, isWin := ctx.(driver.WindowsWindowContext)
	if !isWin || win.HWND == 0 {
		return 0, 0, false
	}

	var pt point
	ret, _, _ := procClientToScreen.Call(win.HWND, uintptr(unsafe.Pointer(&pt)))
	if ret == 0 {
		return 0, 0, false
	}
	return int(pt.x), int(pt.y), true
}

func platformMaximize(ctx any) {
	win, isWin := ctx.(driver.WindowsWindowContext)
	if !isWin || win.HWND == 0 {
		return
	}

	_, _, _ = procShowWindow.Call(win.HWND, uintptr(swMaximize))
}

func platformUnmaximize(ctx any) {
	win, isWin := ctx.(driver.WindowsWindowContext)
	if !isWin || win.HWND == 0 {
		return
	}

	// SW_RESTORE also activates normal windows and restores minimized ones.
	// Empty-state cleanup can run after the user minimizes, so only undo an
	// actual maximize; never change a minimized window's visibility or focus.
	maximized, _, _ := procIsZoomed.Call(win.HWND)
	if maximized == 0 {
		return
	}
	_, _, _ = procShowWindow.Call(win.HWND, uintptr(swRestore))
}

// Windows screenshots exclude the invisible resize border: DWM reports the
// visible frame, while GetClientRect reports the Fyne content surface.
func platformScreenshotContentSize(ctx any, width, height int) (fyne.Size, bool) {
	win, ok := ctx.(driver.WindowsWindowContext)
	if !ok || win.HWND == 0 {
		return fyne.Size{}, false
	}
	// Fixed GLFW limits alone do not remove the native maximize command.
	// Keep close/minimize, but remove resizing and maximize from the title bar.
	getLong, setLong := "GetWindowLongPtrW", "SetWindowLongPtrW"
	if strconv.IntSize == 32 {
		getLong, setLong = "GetWindowLongW", "SetWindowLongW"
	}
	styleIndex := ^uintptr(15) // GWL_STYLE (-16)
	style, _, _ := user32.NewProc(getLong).Call(win.HWND, styleIndex)
	if style == 0 {
		return fyne.Size{}, false
	}
	style &^= 0x00040000 | 0x00010000 // WS_THICKFRAME | WS_MAXIMIZEBOX
	result, _, _ := user32.NewProc(setLong).Call(win.HWND, styleIndex, style)
	if result == 0 {
		return fyne.Size{}, false
	}
	_, _, _ = user32.NewProc("SetWindowPos").Call(win.HWND, 0, 0, 0, 0, 0, 0x37) // frame changed; preserve position/size/z-order/activation
	var frame, client struct{ left, top, right, bottom int32 }
	getClientRect := user32.NewProc("GetClientRect")
	getFrame := syscall.NewLazyDLL("dwmapi.dll").NewProc("DwmGetWindowAttribute")
	result, _, _ = getFrame.Call(win.HWND, 9, uintptr(unsafe.Pointer(&frame)), unsafe.Sizeof(frame))
	if result != 0 {
		return fyne.Size{}, false
	}
	result, _, _ = getClientRect.Call(win.HWND, uintptr(unsafe.Pointer(&client)))
	if result == 0 {
		return fyne.Size{}, false
	}
	chromeWidth := (frame.right - frame.left) - (client.right - client.left)
	chromeHeight := (frame.bottom - frame.top) - (client.bottom - client.top)
	return fyne.NewSize(float32(width)-float32(chromeWidth), float32(height)-float32(chromeHeight)), true
}
