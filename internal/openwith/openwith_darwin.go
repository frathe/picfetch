//go:build darwin

package openwith

/*
#cgo CFLAGS: -fobjc-arc
#cgo LDFLAGS: -framework AppKit

#include <stdlib.h>

#include "openwith_darwin.h"
*/
import "C"

import "unsafe"

// picfetchDeliverOpenURLs is the single entry point from Objective-C: both
// grafted delegate methods funnel their absolute file:// URL strings here.
// Every C string is copied into Go memory before this returns, so the
// caller is free to release its buffer as soon as the call completes.
//
//export picfetchDeliverOpenURLs
func picfetchDeliverOpenURLs(urls **C.char, n C.int) {
	Deliver(URIsFromFileURLs(goStrings(urls, int(n))))
}

// goStrings copies n NUL-terminated C strings into Go strings.
//
// Neither guard is reachable through the current caller - the .m side only
// calls back with an array it just built and a count it wrote itself - and
// both stay because this is an unsafe boundary: unsafe.Slice panics on a
// negative length and hands back an unusable slice from a NULL base, and a
// NULL element would otherwise reach C.GoString. Dropping a bad entry
// rather than failing the call also matches URIsFromFileURLs, so one bad
// path from LaunchServices never costs the rest of the batch.
func goStrings(p **C.char, n int) []string {
	if p == nil || n <= 0 {
		return nil
	}

	out := make([]string, 0, n)
	for _, c := range unsafe.Slice(p, n) {
		if c == nil {
			continue
		}
		out = append(out, C.GoString(c))
	}
	return out
}

// Install grafts application:openURLs: and application:openFiles: onto
// GLFW's application delegate class and reports whether either was added.
//
// It has to run before glfw.Init reaches [NSApp setDelegate:], because
// NSApplication caches which selectors its delegate answers at that
// moment; a method added afterwards is never consulted, no matter that the
// runtime now finds it on the class.
//
// false is not an error the caller can act on, only a fact worth logging.
// It means either that this binary has no GLFWApplicationDelegate class
// (nothing links the Cocoa driver) or that both selectors were already
// grafted by an earlier call. In both cases the app keeps its previous
// behaviour of ignoring "Open With" rather than failing to start. Ask
// DelegateRespondsToOpen, not Install's result, whether the methods are
// actually on the class.
func Install() bool {
	return C.picfetchInstallOpenHandler() != 0
}

// DelegateRespondsToOpen reports whether GLFW's delegate class currently
// carries both open-document methods. This, not Install's return value, is
// the honest answer to "is the bridge in place?": Install reports false
// both when the class is absent and when an earlier call already grafted
// the methods on. Exported because the only binary that links the Cocoa
// driver - and so the only one where this can be true - is package main.
func DelegateRespondsToOpen() bool {
	return C.picfetchDelegateRespondsToOpen() != 0
}

func testInvokeOpenURLs(urls []string) {
	buf, release := cStrings(urls)
	defer release()
	C.picfetchTestInvokeOpenURLs(buf, C.int(len(urls)))
}

func testInvokeOpenFiles(paths []string) {
	buf, release := cStrings(paths)
	defer release()
	C.picfetchTestInvokeOpenFiles(buf, C.int(len(paths)))
}

// cStrings copies ss into an array of C strings and returns it alongside
// the func that frees every element. The backing array is Go memory
// holding only C pointers, which cgo permits passing for the duration of
// the call; the returned closure keeps it reachable until the C side is
// done with it.
//
// A nil slice yields a NULL pointer on purpose, so a test can drive the
// Objective-C side's "no array at all" branch - the nil NSArray AppKit
// could hand a delegate. A non-nil empty slice yields a valid pointer to a
// zero-length array instead, which is a different branch over there.
func cStrings(ss []string) (**C.char, func()) {
	if ss == nil {
		return nil, func() {}
	}

	buf := make([]*C.char, max(len(ss), 1))
	for i, s := range ss {
		buf[i] = C.CString(s)
	}

	return &buf[0], func() {
		for i := range ss {
			C.free(unsafe.Pointer(buf[i]))
		}
	}
}

// Referenced so `go build` (which skips tests) still compiles the bridge
// harness openwith_darwin_test.go needs. Go forbids cgo in _test.go files.
var _ = []any{
	testInvokeOpenURLs, testInvokeOpenFiles,
}

// picfetchDeliverOpenURLs has no Go caller by design - it is reached only
// from openwith_darwin.m, through the //export above - so referencing it
// here records that the missing call site is the point rather than an
// oversight, and keeps an unused-function inspection from flagging it.
var _ = picfetchDeliverOpenURLs
