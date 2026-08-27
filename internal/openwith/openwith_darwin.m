// Objective-C half of the macOS "Open With" bridge; the Go half is in
// openwith_darwin.go.
//
// macOS never puts files in argv for a bundled .app: "Open With", a drop
// onto the Dock icon, `open -a`, and a double-click on an associated file
// all reach the process as a kAEOpenDocuments Apple Event, which AppKit
// turns into an open-document call on NSApp's delegate. That delegate is
// GLFW's, and GLFWApplicationDelegate implements no open-document method
// at all, which is why PicFetch ignored every "Open With" before this.
//
// The graft is done with class_addMethod on GLFW's class rather than by
// installing a delegate of our own or registering an NSAppleEventManager
// handler. -[NSApplication setDelegate:] caches which selectors the
// delegate answers, so a method added after GLFW's [NSApp setDelegate:]
// call is never consulted; and AppKit installs its own kAEOpenDocuments
// handler during finishLaunching, which would clobber ours.

#import <AppKit/AppKit.h>
#import <objc/runtime.h>

#include <limits.h>
#include <stdlib.h>
#include <string.h>

#include "_cgo_export.h"
#include "openwith_darwin.h"

// deliverAbsoluteStrings hands strings to Go as a C array of UTF-8 buffers.
// A nil array is a no-op: -count on nil returns 0.
//
// Buffer lifetime, which -fobjc-arc makes load-bearing: -[NSString
// UTF8String] returns storage owned by the receiver, valid only until the
// string is released or the enclosing autorelease pool drains, and ARC is
// free to release each string as soon as the loop stops using it. So every
// buffer is strdup'd into memory this function owns outright before it is
// handed over. Go's side copies each entry with C.GoString before
// returning, so the whole array is freed here the moment the call is done.
static void deliverAbsoluteStrings(NSArray<NSString *> *strings) {
	NSUInteger count = [strings count];
	if (count == 0) {
		return;
	}
	// n below is an int, to match the count cgo's generated prototype
	// takes. An NSArray big enough to overflow it cannot arise from
	// LaunchServices, but signed overflow is undefined rather than merely
	// wrong, so the impossible case is refused outright instead of
	// wrapping into a negative count.
	if (count > INT_MAX) {
		return;
	}

	const char **buf = calloc(count, sizeof(char *));
	if (buf == NULL) {
		return;
	}

	int n = 0;
	for (NSString *s in strings) {
		const char *utf = [s UTF8String];
		if (utf == NULL) {
			continue;
		}
		char *owned = strdup(utf);
		if (owned == NULL) {
			continue;
		}
		buf[n] = owned;
		n++;
	}

	if (n > 0) {
		// cgo's generated prototype is char **; const here only records
		// that this side does not rewrite the array.
		picfetchDeliverOpenURLs((char **)buf, n);
	}

	for (int i = 0; i < n; i++) {
		free((void *)buf[i]);
	}
	free(buf);
}

// picfetchOpenURLs is the IMP grafted on as -application:openURLs:, the
// modern (10.13+) open-document callback. Fast enumeration over a nil array
// is a no-op, so a nil urls argument simply delivers nothing.
static void picfetchOpenURLs(id self, SEL _cmd, id sender, NSArray<NSURL *> *urls) {
	@autoreleasepool {
		NSMutableArray<NSString *> *absolute = [NSMutableArray arrayWithCapacity:[urls count]];
		for (NSURL *url in urls) {
			NSString *s = [url absoluteString];
			if (s == nil) {
				// -addObject:nil raises; a URL we cannot render as a
				// string is dropped the way Go's URIsFromFileURLs drops
				// one it cannot parse.
				continue;
			}
			[absolute addObject:s];
		}
		deliverAbsoluteStrings(absolute);
	}
}

// picfetchOpenFiles is the IMP grafted on as -application:openFiles:, the
// pre-10.13 callback. Both methods are installed because fyne's Info.plist
// template stamps LSMinimumSystemVersion 10.11; AppKit prefers openURLs:
// when a delegate answers both, so there is no double delivery.
static void picfetchOpenFiles(id self, SEL _cmd, id sender, NSArray<NSString *> *paths) {
	@autoreleasepool {
		NSMutableArray<NSString *> *absolute = [NSMutableArray arrayWithCapacity:[paths count]];
		for (NSString *path in paths) {
			// +fileURLWithPath: answers nil for an empty path on current
			// macOS and has logged or raised on older ones; dropping it
			// here means the outcome does not depend on which. The nil
			// check below would catch today's behaviour on its own.
			if ([path length] == 0) {
				continue;
			}
			NSString *s = [[NSURL fileURLWithPath:path] absoluteString];
			if (s == nil) {
				continue;
			}
			[absolute addObject:s];
		}
		deliverAbsoluteStrings(absolute);

		// sender is nil when the test helper drives this IMP directly.
		// Messaging nil is a no-op in Objective-C, which is what lets this
		// reply be exercised without a live NSApp.
		[(NSApplication *)sender replyToOpenOrPrint:NSApplicationDelegateReplySuccess];
	}
}

int picfetchInstallOpenHandler(void) {
	// Looked up by name rather than referenced at compile time: GLFW's
	// delegate class only exists in a binary that links the Cocoa driver,
	// and this must return cleanly rather than crash in one that does not.
	Class delegate = objc_getClass("GLFWApplicationDelegate");
	if (delegate == Nil) {
		return 0;
	}

	// "v@:@@" - void return, then the two implicit arguments every method
	// carries (id self, SEL _cmd), then the two object arguments
	// (NSApplication *sender and the NSArray). A wrong encoding is not a
	// compile error; it silently feeds the IMP garbage at runtime.
	BOOL addedURLs = class_addMethod(delegate, @selector(application:openURLs:),
	                                 (IMP)picfetchOpenURLs, "v@:@@");
	BOOL addedFiles = class_addMethod(delegate, @selector(application:openFiles:),
	                                  (IMP)picfetchOpenFiles, "v@:@@");

	// NO means the selector was already on the class, which is what a
	// second call looks like - routine, not a failure worth reporting.
	return (addedURLs || addedFiles) ? 1 : 0;
}

int picfetchDelegateRespondsToOpen(void) {
	Class delegate = objc_getClass("GLFWApplicationDelegate");
	if (delegate == Nil) {
		return 0;
	}
	if (class_getInstanceMethod(delegate, @selector(application:openURLs:)) == NULL) {
		return 0;
	}
	if (class_getInstanceMethod(delegate, @selector(application:openFiles:)) == NULL) {
		return 0;
	}
	return 1;
}

void picfetchTestInvokeOpenURLs(const char **urls, int n) {
	@autoreleasepool {
		NSMutableArray<NSURL *> *arr = nil;
		if (urls != NULL) {
			arr = [NSMutableArray arrayWithCapacity:(n > 0 ? (NSUInteger)n : 0)];
			for (int i = 0; i < n; i++) {
				if (urls[i] == NULL) {
					continue;
				}
				NSString *s = [NSString stringWithUTF8String:urls[i]];
				NSURL *u = (s == nil) ? nil : [NSURL URLWithString:s];
				if (u == nil) {
					continue;
				}
				[arr addObject:u];
			}
		}
		picfetchOpenURLs(nil, @selector(application:openURLs:), nil, arr);
	}
}

void picfetchTestInvokeOpenFiles(const char **paths, int n) {
	@autoreleasepool {
		NSMutableArray<NSString *> *arr = nil;
		if (paths != NULL) {
			arr = [NSMutableArray arrayWithCapacity:(n > 0 ? (NSUInteger)n : 0)];
			for (int i = 0; i < n; i++) {
				if (paths[i] == NULL) {
					continue;
				}
				NSString *s = [NSString stringWithUTF8String:paths[i]];
				if (s == nil) {
					continue;
				}
				[arr addObject:s];
			}
		}
		picfetchOpenFiles(nil, @selector(application:openFiles:), nil, arr);
	}
}
