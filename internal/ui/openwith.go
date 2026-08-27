// The viewer's side of macOS "Open With": internal/openwith owns the
// Objective-C bridge and the queue that buffers a delivery until someone
// can take it; this file is what the viewer does with one.

package ui

import (
	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/openwith"
)

// installOpenWithHandler routes every "Open With" delivery into handleDrop.
// Because openwith.SetHandler installs and flushes in one critical section,
// this also picks up whatever the cold-start Apple Event already queued -
// which on macOS is the normal case, since AppKit dispatches that event
// while GLFW is still inside glfw.Init(), long before SetOnStarted fires.
//
// The fyne.Do here inverts this package's usual rule and is deliberate. The
// handler runs on whichever thread delivered, and for the production caller
// that is AppKit's main thread - which *is* Fyne's UI goroutine, where
// marshaling would normally be the wrong thing to do. It is safe because Do
// makes no claim about the calling thread: gLDriver.DoFromGoroutine only
// wraps the wait == true path in async.EnsureNotMain, and
// runOnMainWithWait(f, false) (fyne v2.8.0, internal/driver/glfw/loop.go)
// just enqueues onto funcQueue. So this one call is correct from the Apple
// Event callback and from a background goroutine alike. It must stay Do and
// never DoAndWait, which asserts it was called off the main goroutine and
// would fail exactly where this is used.
func (v *viewer) installOpenWithHandler() {
	openwith.SetHandler(func(uris []fyne.URI) {
		fyne.Do(func() { v.openFilesFromOS(uris) })
	})
}

// openInitialFiles opens the launch's own file set, and is a no-op when
// installOpenWithHandler's flush already swept it up along with an
// "Open With" delivery - the usual macOS cold start.
func (v *viewer) openInitialFiles() {
	fyne.Do(func() { v.openFilesFromOS(nil) })
}

// openFilesFromOS opens uris together with whatever the launch is still
// carrying, as one batch, and clears that pending set so the other caller
// finds nothing left to open.
//
// This is what keeps a cold start that has both command-line paths *and* an
// "Open With" Apple Event to a single scan rather than two overlapping ones,
// with the command-line files first. Both callers hand their work to
// fyne.Do, so the two arrive in call order on one goroutine and whichever
// runs first takes everything. Calling this straight from the handler
// instead of through Do would break that: the flush would then run ahead of
// an openInitialFiles that Do had merely queued, and both batches would be
// dropped separately.
func (v *viewer) openFilesFromOS(uris []fyne.URI) {
	files := append(append([]fyne.URI(nil), v.pendingInitial...), uris...)
	v.pendingInitial = nil

	v.handleDrop(files)
}
