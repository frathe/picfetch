//go:build darwin

package displays

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit -framework CoreGraphics

#import <AppKit/AppKit.h>
#import <CoreGraphics/CoreGraphics.h>
#include <stdint.h>
#include <stdlib.h>
#include <string.h>

static int attachedScreenCount(void) {
	@autoreleasepool {
		return (int)[NSScreen screens].count;
	}
}

static int screenInfoAt(int index, uint32_t *displayID, int *width,
	int *height, char **name) {
	@autoreleasepool {
		NSArray<NSScreen *> *screens = [NSScreen screens];
		if (index < 0 || index >= (int)screens.count) {
			return 0;
		}
		NSScreen *screen = screens[(NSUInteger)index];
		NSNumber *number = screen.deviceDescription[@"NSScreenNumber"];
		if (number == nil) {
			return 0;
		}
		CGDirectDisplayID ident = (CGDirectDisplayID)number.unsignedIntValue;
		CGDisplayModeRef mode = CGDisplayCopyDisplayMode(ident);
		if (mode == NULL) {
			return 0;
		}
		NSString *label = screen.localizedName;
		const char *utf8 = label.UTF8String;
		*displayID = (uint32_t)ident;
		*width = (int)CGDisplayModeGetPixelWidth(mode);
		*height = (int)CGDisplayModeGetPixelHeight(mode);
		CGDisplayModeRelease(mode);
		*name = strdup(utf8 == NULL ? "Display" : utf8);
		return *name != NULL;
	}
}

static uint32_t defaultScreenForWindow(uintptr_t nsWindowPtr) {
	@autoreleasepool {
		NSWindow *window = (__bridge NSWindow *)(void *)nsWindowPtr;
		if (window == nil) {
			return 0;
		}
		NSRect frame = window.frame;
		CGFloat largest = -1;
		uint32_t selected = 0;
		for (NSScreen *screen in [NSScreen screens]) {
			NSRect overlap = NSIntersectionRect(frame, screen.frame);
			CGFloat area = overlap.size.width * overlap.size.height;
			if (area > largest) {
				NSNumber *number = screen.deviceDescription[@"NSScreenNumber"];
				selected = number.unsignedIntValue;
				largest = area;
			}
		}
		return selected;
	}
}
*/
import "C"

import (
	"fmt"
	"image"
	"strconv"
	"unsafe"

	"fyne.io/fyne/v2/driver"
)

func platformInspect(context any) ([]Display, ID, error) {
	window, ok := context.(driver.MacWindowContext)
	if !ok || window.NSWindow == 0 {
		return nil, "", &UnsupportedError{Reason: "not an AppKit window"}
	}

	count := int(C.attachedScreenCount())
	displays := make([]Display, 0, count)
	for index := range count {
		var displayID C.uint32_t
		var width, height C.int
		var name *C.char
		if C.screenInfoAt(C.int(index), &displayID, &width, &height, &name) == 0 {
			return nil, "", fmt.Errorf("inspect macOS display %d", index)
		}
		label := C.GoString(name)
		C.free(unsafe.Pointer(name))
		displays = append(displays, Display{
			ID:     ID(strconv.FormatUint(uint64(displayID), 10)),
			Name:   label,
			Bounds: image.Rect(0, 0, int(width), int(height)),
		})
	}
	defaultNative := C.defaultScreenForWindow(C.uintptr_t(window.NSWindow))
	defaultID := ID("")
	if defaultNative != 0 {
		defaultID = ID(strconv.FormatUint(uint64(defaultNative), 10))
	}

	return displays, defaultID, nil
}
