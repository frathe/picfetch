//go:build darwin && cgo

package worker

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/SecTask.h>
#include <CoreFoundation/CoreFoundation.h>

static int picfetchAppSandbox(void) {
    SecTaskRef task = SecTaskCreateFromSelf(kCFAllocatorDefault);
    if (task == NULL) return 0;
    CFTypeRef value = SecTaskCopyValueForEntitlement(task, CFSTR("com.apple.security.app-sandbox"), NULL);
    int enabled = value != NULL && CFGetTypeID(value) == CFBooleanGetTypeID() && CFBooleanGetValue((CFBooleanRef)value);
    if (value != NULL) CFRelease(value);
    CFRelease(task);
    return enabled;
}
*/
import "C"

import (
	"errors"

	"github.com/frathe/picfetch/internal/heicdecode"
)

func isolate(_ heicdecode.Limits) (int64, error) {
	// App Sandbox is established by macOS at exec from the signed helper
	// bundle's entitlement. No deprecated custom Seatbelt API is used.
	if C.picfetchAppSandbox() != 1 {
		return 0, errors.New("HEIC helper lacks App Sandbox entitlement")
	}
	// Capability probes in Main must also observe actual file/network denial.
	// Ronin explicitly accepted no guaranteed whole-native-process memory cap.
	return 0, nil
}
