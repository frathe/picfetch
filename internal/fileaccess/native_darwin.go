//go:build darwin && appleappstore

package fileaccess

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework Foundation
#import <Foundation/Foundation.h>
#include <stdlib.h>
#include <string.h>
#include <limits.h>

static char *acquireBookmark(const void *bytes, size_t length, int directory, void **owner, void **renewed, int *renewedLength, char **failure) {
 @autoreleasepool {
  NSData *bookmark = [NSData dataWithBytes:bytes length:length];
  NSError *error = nil;
  BOOL stale = NO;
  NSURL *url = [NSURL URLByResolvingBookmarkData:bookmark
   options:NSURLBookmarkResolutionWithSecurityScope | NSURLBookmarkResolutionWithoutUI
   relativeToURL:nil bookmarkDataIsStale:&stale error:&error];
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
  NSData *replacement = nil;
  if (stale) {
   replacement = [url bookmarkDataWithOptions:NSURLBookmarkCreationWithSecurityScope
    includingResourceValuesForKeys:nil relativeToURL:nil error:&error];
   if (!replacement || replacement.length == 0 || replacement.length > INT_MAX) {
    [url stopAccessingSecurityScopedResource];
    *failure = strdup(error ? error.description.UTF8String : "unable to renew security-scoped bookmark");
    return NULL;
   }
  }
  char *path = strdup(url.fileSystemRepresentation);
  if (!path) { [url stopAccessingSecurityScopedResource]; return NULL; }
  if (replacement) {
   *renewed = malloc(replacement.length);
   if (!*renewed) { free(path); [url stopAccessingSecurityScopedResource]; return NULL; }
   memcpy(*renewed, replacement.bytes, replacement.length);
   *renewedLength = (int)replacement.length;
  }
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

func resolveBookmark(_ context.Context, record Record) (resolution, func(), error) {
	var owner, renewed unsafe.Pointer
	var renewedLength C.int
	var failure *C.char
	directory := C.int(0)
	if record.Directory {
		directory = 1
	}
	path := C.acquireBookmark(unsafe.Pointer(&record.Bookmark[0]), C.size_t(len(record.Bookmark)), directory, &owner, &renewed, &renewedLength, &failure)
	defer C.free(unsafe.Pointer(path))
	defer C.free(renewed)
	defer C.free(unsafe.Pointer(failure))
	if path == nil {
		if failure != nil {
			return resolution{}, nil, errors.New(C.GoString(failure))
		}
		return resolution{}, nil, errors.New("unable to acquire security-scoped source")
	}
	return resolution{path: C.GoString(path), bookmark: C.GoBytes(renewed, renewedLength)}, func() { C.releaseBookmark(owner) }, nil
}
