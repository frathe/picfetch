//go:build darwin

package wallpaper

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit

#import <AppKit/AppKit.h>
#include <stdint.h>
#include <stdlib.h>

// setWallpaper points every screen's desktop picture at path via
// NSWorkspace - the same call the Wallpaper settings pane ends up making.
// Unlike an AppleScript "tell application \"System Events\" to set picture
// of every desktop", this calls a system framework directly rather than
// scripting another app, so it never triggers the one-time Automation
// permission prompt Apple Events would.
//
// setDesktopImageURL:forScreen:options:error: is synchronous and safe to
// call off the main thread, so this needs neither the semaphore
// internal/trash's asynchronous recycleURLs: does nor a fyne.DoAndWait hop
// to the main thread the way internal/filepicker's NSOpenPanel does - there
// is no modal panel and no completion handler involved.
//
// Each screen's existing options (the user's own Fill Screen / Fit to
// Screen / Stretch choice, and their background color) are read back and
// passed through, so setting a wallpaper from this app changes the picture
// and nothing else. Returns NULL on success, or a malloc'd error message.
static char *setWallpaper(const char *path, uint32_t displayID, int targeted) {
	@autoreleasepool {
		NSURL *url = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
		NSArray<NSScreen *> *screens = [NSScreen screens];
		if (screens.count == 0) {
			return strdup("no screen is attached");
		}

		NSScreen *targetScreen = nil;
		if (targeted) {
			for (NSScreen *screen in screens) {
				NSNumber *number = screen.deviceDescription[@"NSScreenNumber"];
				if (number != nil && number.unsignedIntValue == displayID) {
					targetScreen = screen;
					break;
				}
			}
			if (targetScreen == nil) {
				return strdup("the selected display is no longer attached");
			}
			screens = @[targetScreen];
		}

		for (NSScreen *screen in screens) {
			NSDictionary *options = [[NSWorkspace sharedWorkspace] desktopImageOptionsForScreen:screen];
			NSError *error = nil;
			if (![[NSWorkspace sharedWorkspace] setDesktopImageURL:url
			                                            forScreen:screen
			                                              options:options ? options : @{}
			                                                error:&error]) {
				return strdup(error.localizedDescription.UTF8String);
			}
		}
		return NULL;
	}
}
*/
import "C"

import (
	"errors"
	"fmt"
	"strconv"
	"unsafe"
)

// setDarwin makes path the desktop picture on every attached screen via
// AppKit's NSWorkspace, in-process - see setWallpaper above for why not an
// osascript shell-out.
func setDarwin(path string) error {
	return setDarwinNative(path, 0, false)
}

func setDarwinRequest(request Request) error {
	return setDarwinRequestWith(request, setDarwinNative)
}

func setDarwinRequestWith(request Request, native func(string, uint32, bool) error) error {
	if request.Target == "" {
		return native(request.Path, 0, false)
	}
	displayID, err := strconv.ParseUint(string(request.Target), 10, 32)
	if err != nil {
		return fmt.Errorf("invalid macOS display ID %q: %w", request.Target, err)
	}

	return native(request.Path, uint32(displayID), true)
}

func setDarwinNative(path string, displayID uint32, targeted bool) error {
	cPath := C.CString(path)
	defer C.free(unsafe.Pointer(cPath))

	targetedInt := C.int(0)
	if targeted {
		targetedInt = 1
	}
	cErr := C.setWallpaper(cPath, C.uint32_t(displayID), targetedInt)
	if cErr == nil {
		return nil
	}
	defer C.free(unsafe.Pointer(cErr))
	return errors.New(C.GoString(cErr))
}
