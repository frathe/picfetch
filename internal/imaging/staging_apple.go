//go:build darwin && appleappstore

package imaging

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation
#import <Foundation/Foundation.h>
#include <stdlib.h>
#include <string.h>

static char *replacementDirectory(const char *path, char **failure) {
 @autoreleasepool {
  NSURL *destination = [NSURL fileURLWithPath:[NSString stringWithUTF8String:path]];
  NSError *error = nil;
  NSURL *directory = [NSFileManager.defaultManager URLForDirectory:NSItemReplacementDirectory
   inDomain:NSUserDomainMask appropriateForURL:destination create:YES error:&error];
  if (!directory) {
   *failure = strdup(error ? error.description.UTF8String : "replacement directory unavailable");
   return NULL;
  }
  return strdup(directory.fileSystemRepresentation);
 }
}
*/
import "C"

import (
	"errors"
	"os"
	"unsafe"
)

// A file-only sandbox grant does not permit sibling temporary files. Foundation
// supplies a private staging directory on the destination's volume, preserving
// atomic rename without requesting authority over the destination's parent.
func writeStagingDirectory(path string) (string, func(), error) {
	name := C.CString(path)
	defer C.free(unsafe.Pointer(name))
	var failure *C.char
	value := C.replacementDirectory(name, &failure)
	defer C.free(unsafe.Pointer(value))
	defer C.free(unsafe.Pointer(failure))
	if failure != nil {
		return "", nil, errors.New(C.GoString(failure))
	}
	if value == nil {
		return "", nil, errors.New("replacement directory unavailable")
	}
	directory := C.GoString(value)
	return directory, func() { _ = os.RemoveAll(directory) }, nil
}
