//go:build darwin && appleappstore

package fileaccess

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation
#import <Foundation/Foundation.h>
#include <stdlib.h>
#include <string.h>

static char *acquireBookmark(const void *bytes, size_t length, int directory, void **owner, char **failure) {
 @autoreleasepool {
  NSData *bookmark = [NSData dataWithBytes:bytes length:length];
  NSError *error = nil;
  NSURL *url = [NSURL URLByResolvingBookmarkData:bookmark
   options:NSURLBookmarkResolutionWithSecurityScope | NSURLBookmarkResolutionWithoutUI
   relativeToURL:nil bookmarkDataIsStale:NULL error:&error];
  if (!url || ![url startAccessingSecurityScopedResource]) {
   *failure = strdup(error ? error.description.UTF8String : "security-scoped source access denied");
   return NULL;
  }
  NSNumber *isDirectory = nil;
  if (![url getResourceValue:&isDirectory forKey:NSURLIsDirectoryKey error:&error] ||
      !isDirectory || isDirectory.boolValue != (directory != 0)) {
   [url stopAccessingSecurityScopedResource];
   *failure = strdup("security-scoped source type changed");
   return NULL;
  }
  char *path = strdup(url.fileSystemRepresentation);
  if (!path) { [url stopAccessingSecurityScopedResource]; return NULL; }
  *owner = (__bridge_retained void *)url;
  return path;
 }
}
static void releaseBookmark(void *owner) {
 @autoreleasepool {
  NSURL *url = (__bridge_transfer NSURL *)owner;
  [url stopAccessingSecurityScopedResource];
 }
}
*/
import "C"

import (
	"context"
	"errors"
	"unsafe"
)

func resolveBookmark(_ context.Context, record Record) (string, func(), error) {
	var owner unsafe.Pointer
	var failure *C.char
	directory := C.int(0)
	if record.Directory {
		directory = 1
	}
	path := C.acquireBookmark(unsafe.Pointer(&record.Bookmark[0]), C.size_t(len(record.Bookmark)), directory, &owner, &failure)
	defer C.free(unsafe.Pointer(path))
	defer C.free(unsafe.Pointer(failure))
	if path == nil {
		if failure != nil {
			return "", nil, errors.New(C.GoString(failure))
		}
		return "", nil, errors.New("unable to acquire security-scoped source")
	}
	return C.GoString(path), func() { C.releaseBookmark(owner) }, nil
}
