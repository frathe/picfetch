//go:build darwin && appleappstore

package fileaccess

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation
#import <Foundation/Foundation.h>
#include <stdlib.h>
#include <string.h>

static void *exportTransfer(const char *path, size_t *length, char **failure) {
 @autoreleasepool {
  NSURL *url = [NSURL fileURLWithFileSystemRepresentation:path isDirectory:NO relativeToURL:nil];
  NSError *error = nil;
  NSData *data = [url bookmarkDataWithOptions:0 includingResourceValuesForKeys:nil relativeToURL:nil error:&error];
  if (!data) { *failure = strdup(error.description.UTF8String); return NULL; }
  void *bytes = malloc(data.length);
  if (bytes) { memcpy(bytes, data.bytes, data.length); *length = data.length; }
  return bytes;
 }
}
static void *importTransfer(const void *bytes, size_t length, char **failure) {
 @autoreleasepool {
  NSError *error = nil;
  NSURL *url = [NSURL URLByResolvingBookmarkData:[NSData dataWithBytes:bytes length:length]
   options:NSURLBookmarkResolutionWithoutUI relativeToURL:nil bookmarkDataIsStale:NULL error:&error];
  if (!url) { *failure = strdup(error.description.UTF8String); return NULL; }
  return (__bridge_retained void *)url;
 }
}
static void stopTransfer(void *owner) {
 @autoreleasepool {
  NSURL *url = (__bridge_transfer NSURL *)owner;
  [url stopAccessingSecurityScopedResource];
 }
}
*/
import "C"

import (
	"errors"
	"strings"
	"unsafe"
)

func createTransfer(path string) ([]byte, error) {
	if strings.ContainsRune(path, 0) {
		return nil, errors.New("invalid worker path")
	}
	native := C.CString(path)
	defer C.free(unsafe.Pointer(native))
	var length C.size_t
	var failure *C.char
	bytes := C.exportTransfer(native, &length, &failure)
	defer C.free(bytes)
	defer C.free(unsafe.Pointer(failure))
	if bytes == nil {
		if failure != nil {
			return nil, errors.New(C.GoString(failure))
		}
		return nil, errors.New("unable to create worker grant")
	}
	if length == 0 || length > 1024*1024 {
		return nil, errors.New("invalid worker bookmark size")
	}
	return C.GoBytes(bytes, C.int(length)), nil
}

func resolveTransfer(transfer Transfer) (func(), error) {
	var failure *C.char
	owner := C.importTransfer(unsafe.Pointer(&transfer.Bookmark[0]), C.size_t(len(transfer.Bookmark)), &failure)
	defer C.free(unsafe.Pointer(failure))
	if owner == nil {
		if failure != nil {
			return nil, errors.New(C.GoString(failure))
		}
		return nil, errors.New("unable to resolve worker grant")
	}
	return func() { C.stopTransfer(owner) }, nil
}
