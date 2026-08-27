// Package openwith buffers files the OS hands the process before the
// viewer is ready to receive them, and parses the file:// URLs
// LaunchServices delivers them as.
//
// macOS never puts files in argv for a bundled .app. "Open With", a
// drag-and-drop onto the Dock icon, and `open -a` all go through
// LaunchServices, which sends a kAEOpenDocuments Apple Event that AppKit
// turns into an application:openURLs: call on the app's delegate. A later
// stage grafts that delegate method onto GLFW's delegate class from
// Objective-C - this package is the viewer-independent core both that
// bridge and the UI wiring sit on.
//
// The timing fact that dictates this package's shape: GLFW calls
// [NSApp run] *inside* glfw.Init(), and AppKit dispatches the cold-start
// Apple Event during that call, which happens while Fyne is still creating
// its first window - before the viewer exists and before
// fyne.App.Lifecycle().SetOnStarted fires. Files that arrive that early
// have nowhere to go yet, so Deliver buffers them until SetHandler installs
// the viewer's real handler, which drains anything buffered in the same
// critical section it installs itself in.
//
// Ordering holds per caller, not across concurrent ones: both Deliver and
// SetHandler release the lock before invoking the handler, so a flush and a
// direct delivery racing from two goroutines can arrive in either order.
// That is not a problem for the only production caller - the Apple Event
// callback and the SetOnStarted install both run on the main thread - but
// a future caller delivering from a background goroutine should not assume
// otherwise.
package openwith

import (
	"net/url"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
)

// queue buffers URIs that arrive before a handler is installed, and hands
// them straight to the handler once one is set.
type queue struct {
	mu      sync.Mutex
	pending []fyne.URI
	handler func([]fyne.URI)
}

// Deliver adds uris to the queue: passed to the installed handler
// immediately if one is set, buffered until SetHandler otherwise. A nil or
// empty slice is a no-op. The handler, if any, is called outside the lock,
// so it may itself call Deliver or SetHandler without deadlocking.
func (q *queue) Deliver(uris []fyne.URI) {
	if len(uris) == 0 {
		return
	}

	q.mu.Lock()
	h := q.handler
	if h == nil {
		q.pending = append(q.pending, uris...)
	}
	q.mu.Unlock()

	if h != nil {
		h(uris)
	}
}

// SetHandler installs h as the handler and, in the same critical section,
// takes whatever is currently buffered; that batch is then delivered to h
// outside the lock. Doing both under one lock acquisition is the point: a
// separate "drain, then install" pair would lose anything that arrives via
// Deliver between the two steps. SetHandler(nil) clears the handler
// (used at shutdown) without discarding whatever is currently pending -
// there's no handler to flush it to, so it's left in the queue rather than
// taken and dropped, and picks up right where it left off the next time
// something is delivered or a real handler is installed.
func (q *queue) SetHandler(h func([]fyne.URI)) {
	q.mu.Lock()
	q.handler = h

	var pending []fyne.URI
	if h != nil {
		pending = q.pending
		q.pending = nil
	}
	q.mu.Unlock()

	if len(pending) > 0 {
		h(pending)
	}
}

// defaultQueue is process-global state, but it is not the kind of "mutable
// package-level test seam" AGENTS.md forbids: that rule targets hooks
// installed purely so a test can swap behavior, whereas this is a
// mutex-guarded model of a genuinely process-global OS resource - there is
// exactly one NSApp, and hence exactly one open-files delegate callback,
// per process. That callback is invoked from Objective-C with no Go
// receiver to thread a *queue through, which is why Deliver and SetHandler
// exist as package-level wrappers at all. Tests get isolation from
// reset (openwith_test.go), not from an exported reset.
var defaultQueue queue

// Deliver adds uris to the default queue - see (*queue).Deliver.
func Deliver(uris []fyne.URI) {
	defaultQueue.Deliver(uris)
}

// SetHandler installs h as the default queue's handler - see
// (*queue).SetHandler.
func SetHandler(h func([]fyne.URI)) {
	defaultQueue.SetHandler(h)
}

// URIsFromFileURLs parses each of raw as a file:// URL and returns the
// resulting fyne.URIs, in order. An entry is skipped, not fatal, if it
// fails to parse, its scheme isn't "file", or its decoded path is empty -
// mirroring main.go's argsToURIs, which skips a bad command-line path
// rather than aborting the rest of the batch for the same reason: one bad
// entry from LaunchServices shouldn't stop the rest of an "Open With"
// batch from loading. url.Parse decodes percent-escapes into URL.Path, so
// "file:///a/with%20space.jpg" yields the path "/a/with space.jpg".
func URIsFromFileURLs(raw []string) []fyne.URI {
	uris := make([]fyne.URI, 0, len(raw))

	for _, r := range raw {
		u, err := url.Parse(r)
		if err != nil {
			continue
		}

		if u.Scheme != "file" {
			continue
		}

		// An authority component means the URL names a file on another
		// host, which storage.NewFileURI cannot express - it would build a
		// local path from u.Path alone and silently open the wrong file.
		// LaunchServices only ever hands over local files, as an empty
		// authority or the explicit "localhost", so anything else is
		// skipped the same way a foreign scheme is.
		if u.Host != "" && u.Host != "localhost" {
			continue
		}

		if u.Path == "" {
			continue
		}

		uris = append(uris, storage.NewFileURI(u.Path))
	}

	return uris
}
