//go:build linux

package displays

/*
#cgo LDFLAGS: -lX11
#include <X11/Xlib.h>
#include <stdint.h>

static int windowBounds(uintptr_t windowHandle, int *x, int *y, int *width, int *height) {
	Display *display = XOpenDisplay(NULL);
	if (display == NULL) return 0;
	Window window = (Window)windowHandle;
	XWindowAttributes attrs;
	Window child;
	int rootX, rootY;
	int ok = XGetWindowAttributes(display, window, &attrs) &&
		XTranslateCoordinates(display, window, DefaultRootWindow(display), 0, 0, &rootX, &rootY, &child);
	if (ok) {
		*x = rootX; *y = rootY; *width = attrs.width; *height = attrs.height;
	}
	XCloseDisplay(display);
	return ok;
}
*/
import "C"

import (
	"fmt"
	"image"
	"os/exec"

	"fyne.io/fyne/v2/driver"
)

func platformInspect(context any) ([]Display, ID, error) {
	window, ok := context.(driver.X11WindowContext)
	if !ok || window.WindowHandle == 0 {
		return nil, "", &UnsupportedError{Reason: "Wayland does not expose truthful global display topology"}
	}
	output, err := exec.Command("xrandr", "--query", "--current").Output()
	if err != nil {
		return nil, "", &UnsupportedError{Reason: fmt.Sprintf("xrandr unavailable: %v", err)}
	}
	displays, err := parseXRandR(string(output))
	if err != nil {
		return nil, "", err
	}

	var x, y, width, height C.int
	windowKnown := C.windowBounds(C.uintptr_t(window.WindowHandle), &x, &y, &width, &height) != 0
	windowRect := image.Rect(int(x), int(y), int(x+width), int(y+height))

	return displays, defaultForWindow(displays, windowRect, windowKnown), nil
}
