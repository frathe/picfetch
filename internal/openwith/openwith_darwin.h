// Declarations for the Objective-C half of the macOS "Open With" bridge,
// implemented in openwith_darwin.m.
//
// This header is all that openwith_darwin.go's cgo preamble includes.
// That file carries an //export, and cgo copies such a preamble into a
// second C translation unit, so the preamble may contain declarations
// only - never a body. Keeping this header plain C, with no AppKit import,
// makes that rule visible at a glance instead of relying on the reader to
// know which system header hides a definition.

#ifndef PICFETCH_OPENWITH_DARWIN_H
#define PICFETCH_OPENWITH_DARWIN_H

// picfetchInstallOpenHandler grafts -application:openURLs: and
// -application:openFiles: onto GLFW's application delegate class. Returns 1
// if at least one selector was added, 0 if the class is not linked into
// this binary or if both selectors were already present.
int picfetchInstallOpenHandler(void);

// picfetchDelegateRespondsToOpen reports whether GLFW's delegate class
// currently carries both open-document methods. Test helper.
int picfetchDelegateRespondsToOpen(void);

// picfetchTestInvokeOpenURLs and picfetchTestInvokeOpenFiles build a
// synthetic array and call the grafted implementations directly, the way
// the Objective-C runtime would. A NULL array stands in for the nil
// NSArray AppKit could hand a delegate. Test helpers.
void picfetchTestInvokeOpenURLs(const char **urls, int n);
void picfetchTestInvokeOpenFiles(const char **paths, int n);

#endif // PICFETCH_OPENWITH_DARWIN_H
