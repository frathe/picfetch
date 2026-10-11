//go:build darwin && appleappstore

package openwith

/*
#include <stdint.h>
#include "openwith_darwin.h"
*/
import "C"

import (
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver"
)

// InstallWindowDrop preserves the native NSURLs before GLFW reduces them to
// paths. Call after Show, when the main window's native content view exists.
func InstallWindowDrop(window fyne.Window) bool {
	native, ok := window.(driver.NativeWindow)
	if !ok {
		return false
	}
	installed := false
	native.RunNative(func(value any) {
		if mac, ok := value.(driver.MacWindowContext); ok && mac.NSWindow != 0 {
			installed = C.picfetchInstallWindowDrop(C.uintptr_t(mac.NSWindow)) != 0
		}
	})
	return installed
}

func testWindowDrop(path string) int {
	paths, release := cStrings([]string{path})
	defer release()
	return int(C.picfetchTestWindowDrop(paths))
}

var _ = testWindowDrop
