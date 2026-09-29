//go:build darwin && cgo

package macbundle

/*
#cgo LDFLAGS: -framework Security -framework CoreFoundation
#include <Security/Security.h>
#include <CoreFoundation/CoreFoundation.h>
#include <string.h>
#include <stdlib.h>
#include <libproc.h>
#include <unistd.h>

static char *currentExecutablePath(void) {
 char path[PROC_PIDPATHINFO_MAXSIZE];
 if (proc_pidpath(getpid(),path,sizeof(path)) <= 0) return NULL;
 return strdup(path);
}

static OSStatus validateCode(const char *path, SecRequirementRef requirement, SecCSFlags flags) {
 CFURLRef url = CFURLCreateFromFileSystemRepresentation(NULL, (const UInt8 *)path, strlen(path), false);
 if (!url) return errSecParam;
 SecStaticCodeRef code = NULL;
 OSStatus result = SecStaticCodeCreateWithPath(url, kSecCSDefaultFlags, &code);
 CFRelease(url);
 if (result == errSecSuccess) {
  result = SecStaticCodeCheckValidity(code, flags, requirement);
  CFRelease(code);
 }
 return result;
}
static OSStatus verifyRuntimeCode(const char *app, const char *library) {
 SecCSFlags flags = kSecCSStrictValidate | kSecCSCheckAllArchitectures;
 OSStatus result = validateCode(library, NULL, flags);
 if (result != errSecSuccess) return result;
 SecRequirementRef requirement = NULL;
 result = SecRequirementCreateWithString(CFSTR("identifier \"io.github.frathe.picfetch\""), kSecCSDefaultFlags, &requirement);
 if (result != errSecSuccess) return result;
 result = validateCode(app, requirement, flags | kSecCSCheckNestedCode);
 CFRelease(requirement);
 return result;
}
*/
import "C"

import (
	"fmt"
	"unsafe"
)

func verifyNative(app, library string) error {
	appPath := C.CString(app)
	defer C.free(unsafe.Pointer(appPath))
	libraryPath := C.CString(library)
	defer C.free(unsafe.Pointer(libraryPath))
	if status := C.verifyRuntimeCode(appPath, libraryPath); status != 0 {
		return fmt.Errorf("bundled runtime code signature is invalid (Security status %d)", int(status))
	}
	return nil
}

func executablePath() (string, error) {
	path := C.currentExecutablePath()
	defer C.free(unsafe.Pointer(path))
	if path == nil {
		return "", fmt.Errorf("cannot locate current native executable")
	}
	return C.GoString(path), nil
}
