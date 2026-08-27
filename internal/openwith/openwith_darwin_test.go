//go:build darwin

package openwith

import (
	"testing"

	"fyne.io/fyne/v2"
)

// captureDelivered installs a handler on the package-level queue and hands
// back a reader for everything delivered through it. The Objective-C
// helpers call back into Go synchronously on the calling goroutine, so the
// closure needs no locking.
func captureDelivered(t *testing.T) func() []fyne.URI {
	t.Helper()
	reset()
	t.Cleanup(reset)

	var got []fyne.URI
	SetHandler(func(uris []fyne.URI) {
		got = append(got, uris...)
	})

	return func() []fyne.URI { return got }
}

func assertPaths(t *testing.T, got []fyne.URI, want ...string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("delivered %v, want %d URI(s) with paths %v", got, len(want), want)
	}
	for i, u := range got {
		if u.Path() != want[i] {
			t.Fatalf("delivered[%d].Path() = %q, want %q", i, u.Path(), want[i])
		}
	}
}

// TestInstall_ReportsFalseWithoutGLFWDelegateClass pins down what false
// means here. internal/openwith does not import glfw, so this test binary
// never links GLFWApplicationDelegate and there is genuinely nothing to
// graft onto - objc_getClass returns Nil and Install must say so rather
// than crash. Do not add a glfw import to turn this green the other way:
// the real graft is asserted from package main's test binary, which does
// link the Cocoa driver.
func TestInstall_ReportsFalseWithoutGLFWDelegateClass(t *testing.T) {
	if Install() {
		t.Fatal("Install() = true in a binary that does not link GLFW")
	}
	if DelegateRespondsToOpen() {
		t.Fatal("DelegateRespondsToOpen() = true with no GLFWApplicationDelegate class")
	}

	// A second call must be just as harmless as the first.
	if Install() {
		t.Fatal("second Install() = true in a binary that does not link GLFW")
	}
}

func TestInvokeOpenURLs_DeliversDecodedPaths(t *testing.T) {
	delivered := captureDelivered(t)

	testInvokeOpenURLs([]string{"file:///tmp/a.jpg", "file:///tmp/b%20c.png"})

	assertPaths(t, delivered(), "/tmp/a.jpg", "/tmp/b c.png")
}

func TestInvokeOpenURLs_DecodesPercentEncodedUnicode(t *testing.T) {
	delivered := captureDelivered(t)

	testInvokeOpenURLs([]string{"file:///tmp/%C3%BCber.png"})

	assertPaths(t, delivered(), "/tmp/über.png")
}

// TestInvokeOpenFiles_DeliversEquivalentURIsFromPlainPaths drives the
// pre-10.13 callback, whose argument is an array of paths rather than
// URLs. It must land on exactly the same URIs as the openURLs: path above:
// +fileURLWithPath: percent-encodes the space on the way out and
// URIsFromFileURLs decodes it again on the way in.
func TestInvokeOpenFiles_DeliversEquivalentURIsFromPlainPaths(t *testing.T) {
	delivered := captureDelivered(t)

	testInvokeOpenFiles([]string{"/tmp/a.jpg", "/tmp/b c.png"})

	assertPaths(t, delivered(), "/tmp/a.jpg", "/tmp/b c.png")
}

func TestInvokeOpenURLs_NonFileURLIsSkipped(t *testing.T) {
	delivered := captureDelivered(t)

	testInvokeOpenURLs([]string{"https://example.com/x.jpg"})

	assertPaths(t, delivered())
}

func TestInvokeOpenURLs_NonFileURLDoesNotCostTheRestOfTheBatch(t *testing.T) {
	delivered := captureDelivered(t)

	testInvokeOpenURLs([]string{
		"file:///tmp/a.jpg",
		"https://example.com/x.jpg",
		"file:///tmp/c.jpg",
	})

	assertPaths(t, delivered(), "/tmp/a.jpg", "/tmp/c.jpg")
}

// TestInvokeOpenURLs_SchemelessEntryIsSkipped covers what +URLWithString:
// does with something that is not a URL at all: it returns a non-nil
// relative URL ("not a url" becomes "not%20a%20url"), so the entry
// survives Objective-C and has to be dropped on the Go side for having no
// file scheme.
func TestInvokeOpenURLs_SchemelessEntryIsSkipped(t *testing.T) {
	delivered := captureDelivered(t)

	testInvokeOpenURLs([]string{"not a url", "file:///tmp/a.jpg"})

	assertPaths(t, delivered(), "/tmp/a.jpg")
}

func TestInvokeOpenURLs_EmptyArrayDeliversNothing(t *testing.T) {
	delivered := captureDelivered(t)

	testInvokeOpenURLs([]string{})

	assertPaths(t, delivered())
}

// TestInvokeOpenURLs_NilArrayDeliversNothing drives the branch where the
// delegate is handed no array at all - a nil slice reaches Objective-C as
// a NULL pointer, which the helper turns into the nil NSArray AppKit could
// in principle pass.
func TestInvokeOpenURLs_NilArrayDeliversNothing(t *testing.T) {
	delivered := captureDelivered(t)

	testInvokeOpenURLs(nil)

	assertPaths(t, delivered())
}

func TestInvokeOpenFiles_EmptyArrayDeliversNothing(t *testing.T) {
	delivered := captureDelivered(t)

	testInvokeOpenFiles([]string{})

	assertPaths(t, delivered())
}

func TestInvokeOpenFiles_NilArrayDeliversNothing(t *testing.T) {
	delivered := captureDelivered(t)

	testInvokeOpenFiles(nil)

	assertPaths(t, delivered())
}

// TestInvokeOpenFiles_EmptyPathIsSkipped pins the contract rather than one
// guard: +fileURLWithPath: has answered nil for an empty path on some
// macOS releases and logged or raised on others, so an empty entry must
// vanish without taking the rest of the batch with it either way.
func TestInvokeOpenFiles_EmptyPathIsSkipped(t *testing.T) {
	delivered := captureDelivered(t)

	testInvokeOpenFiles([]string{"", "/tmp/a.jpg", ""})

	assertPaths(t, delivered(), "/tmp/a.jpg")
}

// TestInvokeOpenURLs_BuffersWhenNoHandlerIsInstalledYet is the cold-start
// case this whole package exists for: LaunchServices delivers during
// glfw.Init, long before the viewer has installed a handler, so the URIs
// have to wait in the queue until it does.
func TestInvokeOpenURLs_BuffersWhenNoHandlerIsInstalledYet(t *testing.T) {
	reset()
	t.Cleanup(reset)

	testInvokeOpenURLs([]string{"file:///tmp/a.jpg"})

	var got []fyne.URI
	SetHandler(func(uris []fyne.URI) {
		got = append(got, uris...)
	})

	assertPaths(t, got, "/tmp/a.jpg")
}
