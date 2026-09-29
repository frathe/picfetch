//go:build darwin

package filepicker

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework AppKit

#import <AppKit/AppKit.h>
#include <stdlib.h>
#include <string.h>

// Both panels and the native transport regression use this serializer.
static char *serializePanelURLs(NSArray<NSURL *> *urls, int scopes, char **failure) {
	NSMutableArray *paths = [NSMutableArray array];
	for (NSURL *url in urls) {
		if (!url.path) return NULL;
		if (scopes) {
			NSError *error = nil;
			NSNumber *directory = nil;
			if (![url getResourceValue:&directory forKey:NSURLIsDirectoryKey error:&error] || !directory) {
				*failure = strdup(error ? error.description.UTF8String : "selected URL directory inspection failed");
				return NULL;
			}
			NSData *bookmark = [url bookmarkDataWithOptions:NSURLBookmarkCreationWithSecurityScope
				includingResourceValuesForKeys:nil relativeToURL:nil error:&error];
			if (!bookmark) {
				*failure = strdup(error ? error.description.UTF8String : "selected URL bookmark capture failed");
				return NULL;
			}
			[paths addObject:@{@"uri":url.absoluteString,
				@"bookmark":[bookmark base64EncodedStringWithOptions:0], @"directory":directory}];
		} else {
			[paths addObject:url.path];
		}
	}
	NSData *data = [NSJSONSerialization dataWithJSONObject:paths options:0 error:NULL];
	if (!data) return NULL;
	NSString *json = [[NSString alloc] initWithData:data encoding:NSUTF8StringEncoding];
	return json ? strdup(json.UTF8String) : NULL;
}

// Runs the real NSURL-to-JSON transport without creating a panel or a window.
static char *roundTripPanelPaths(const char *input) {
	@autoreleasepool {
		NSData *data = [NSData dataWithBytes:input length:strlen(input)];
		NSArray<NSString *> *paths = [NSJSONSerialization JSONObjectWithData:data options:0 error:NULL];
		NSMutableArray<NSURL *> *urls = [NSMutableArray array];
		for (NSString *path in paths) [urls addObject:[NSURL fileURLWithPath:path]];
		return serializePanelURLs(urls, 0, NULL);
	}
}

// runOpenPanel shows an app-modal NSOpenPanel that allows files, folders,
// and multi-select all at once - a combination none of AppleScript's
// Standard Additions pickers offer. Must be called on the main thread (an
// AppKit requirement); chooseFilesDarwin below guarantees that. Returns
// retained URLs or an explicit cancellation/failure state. Serialization runs
// on the chooser worker after the modal panel closes.
typedef struct {
	int state; // 0 cancellation, 1 selected URLs, -1 native failure.
	void *urls;
} panelResult;

// Keep original native URL authority through delivery to the chooser worker.
static panelResult retainSelection(NSArray<NSURL *> *urls) {
	if (!urls.count) return (panelResult){-1, NULL};
	return (panelResult){1, (__bridge_retained void *)[urls copy]};
}

static char *serializeSelection(panelResult result, int scopes, char **failure) {
	@autoreleasepool {
		NSArray<NSURL *> *urls = (__bridge_transfer NSArray<NSURL *> *)result.urls;
		char *data = serializePanelURLs(urls, scopes, failure);
		if (scopes) {
			for (NSURL *url in urls) [url stopAccessingSecurityScopedResource];
		}
		return data;
	}
}

static panelResult runOpenPanel(const char *message) {
	NSOpenPanel *panel = [NSOpenPanel openPanel];
	panel.message = [NSString stringWithUTF8String:message];
	panel.canChooseFiles = YES;
	panel.canChooseDirectories = YES;
	panel.allowsMultipleSelection = YES;

	NSModalResponse response = [panel runModal];
	if (response == NSModalResponseCancel) return (panelResult){0, NULL};
	if (response != NSModalResponseOK) return (panelResult){-1, NULL};
	return retainSelection(panel.URLs);
}

// runSavePanel shows an app-modal NSSavePanel pre-filled with name, opened
// on dir. Same main-thread requirement as runOpenPanel above;
// chooseSaveDarwin guarantees it the same way. No allowedContentTypes is
// set: the format is already decided by which "Export as…" item the user
// picked, and constraining the panel would only stop them naming the file
// whatever they want. Returns the same retained-URL contract as runOpenPanel.
static panelResult runSavePanel(const char *message, const char *dir, const char *name) {
	NSSavePanel *panel = [NSSavePanel savePanel];
	panel.message = [NSString stringWithUTF8String:message];
	panel.nameFieldStringValue = [NSString stringWithUTF8String:name];
	panel.canCreateDirectories = YES;
	if (strlen(dir) > 0) {
		panel.directoryURL = [NSURL fileURLWithPath:[NSString stringWithUTF8String:dir] isDirectory:YES];
	}

	NSModalResponse response = [panel runModal];
	if (response == NSModalResponseCancel) return (panelResult){0, NULL};
	if (response != NSModalResponseOK) return (panelResult){-1, NULL};
	return retainSelection(panel.URL ? @[panel.URL] : @[]);
}
// The destination may not exist; retaining the original URL preserves its
// implicit save permission without inspecting or bookmarking a nonexistent file.
static char *saveSelectionPath(panelResult result) {
 @autoreleasepool {
  NSArray<NSURL *> *urls = (__bridge NSArray<NSURL *> *)result.urls;
  if (urls.count != 1 || !urls.firstObject.isFileURL) return NULL;
  const char *path = urls.firstObject.path.UTF8String;
  return path ? strdup(path) : NULL;
 }
}
static void releaseSaveSelection(panelResult result) {
 @autoreleasepool {
  NSArray<NSURL *> *urls = (__bridge_transfer NSArray<NSURL *> *)result.urls;
  for (NSURL *url in urls) [url stopAccessingSecurityScopedResource];
 }
}

// A URL fixture uses the same ownership transfer as the save panel.
static panelResult testSaveSelection(const char *path) {
 @autoreleasepool {
  return retainSelection(@[[NSURL fileURLWithPath:[NSString stringWithUTF8String:path]]]);
 }
}
*/
import "C"

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"unsafe"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/distribution"
	"github.com/frathe/picfetch/internal/fileaccess"
)

// chooseFilesDarwin runs AppKit's NSOpenPanel in-process rather than
// shelling out the way the Linux/Windows choosers do. In-process is
// load-bearing, not a style choice: this picker went through two
// subprocess generations first - Standard Additions' "choose file or
// folder" (a command that doesn't exist; it parses as a boolean `or` of a
// bare "choose file" and the `folder` property and dies at the first
// "with"), then AppleScriptObjC driving NSOpenPanel inside osascript,
// which did open a panel, but one owned by a background process that macOS
// refuses to make the active app: it appeared *behind* the app window and
// auto-dismissed on the first click, when that click tried and failed to
// activate osascript. A panel owned by this app - a regular, frontmost GUI
// app - has neither problem, and is how every native app shows this
// dialog. AppKit requires the panel on the main thread, which on darwin is
// exactly Fyne's UI thread, hence the fyne.DoAndWait hop; the UI behind
// the panel freezes for the duration, which is what app-modal means. That
// also means this must never be called *from* the UI goroutine (DoAndWait
// would deadlock) - production only reaches it via the viewer's
// openFileDialog background goroutine. Cancellation emits JSON null; successful
// paths use structured transport, while other modal responses are errors.
func chooseFilesDarwin() ([]byte, error) {
	cMsg := C.CString(lang.L("Open images"))
	defer C.free(unsafe.Pointer(cMsg))

	var result C.panelResult
	fyne.DoAndWait(func() { result = C.runOpenPanel(cMsg) })
	return decodeNativePanel(result, distribution.AppleAppStore)
}

// chooseSaveDarwin is chooseFilesDarwin's save-panel twin, in-process and
// main-thread-bound for exactly the same reasons - see the comment above,
// every word of which applies here too. Splitting suggestedPath with
// filepath is safe in this file in a way it wouldn't be in the Windows
// builder: this only ever compiles for darwin, where filepath's separator
// is already the right one.
func chooseSaveDarwin(suggestedPath string) (fyne.URI, error) {
	cMsg := C.CString(lang.L("Export image"))
	defer C.free(unsafe.Pointer(cMsg))
	cDir := C.CString(filepath.Dir(suggestedPath))
	defer C.free(unsafe.Pointer(cDir))
	cName := C.CString(filepath.Base(suggestedPath))
	defer C.free(unsafe.Pointer(cName))

	var result C.panelResult
	fyne.DoAndWait(func() { result = C.runSavePanel(cMsg, cDir, cName) })
	if distribution.AppleAppStore {
		return decodeNativeSavePanel(result)
	}
	out, err := decodeNativePanel(result, false)
	return decodePickedDestination(out, err)
}

// darwinPathTransport exercises the panel's actual Objective-C NSURL serializer
// with temporary paths. It performs no desktop operation and changes no seams.
func darwinPathTransport(paths []string) ([]byte, error) {
	input, err := json.Marshal(paths)
	if err != nil {
		return nil, err
	}
	cInput := C.CString(string(input))
	defer C.free(unsafe.Pointer(cInput))
	cOut := C.roundTripPanelPaths(cInput)
	defer C.free(unsafe.Pointer(cOut))
	return []byte(C.GoString(cOut)), nil
}

// URL/bookmark inspection runs on the tracked chooser worker, after the modal
// panel releases UI. Native URLs remain owned until this serialization ends.
func decodeNativePanel(result C.panelResult, scoped bool) ([]byte, error) {
	if result.state == 0 {
		return []byte("null"), nil
	}
	if result.state != 1 {
		return nil, errors.New("native file panel failed")
	}
	scopes := C.int(0)
	if scoped {
		scopes = 1
	}
	var failure *C.char
	data := C.serializeSelection(result, scopes, &failure)
	defer C.free(unsafe.Pointer(data))
	defer C.free(unsafe.Pointer(failure))
	if failure != nil {
		return nil, errors.New(C.GoString(failure))
	}
	if data == nil {
		return nil, errors.New("native file panel serialization failed")
	}
	return []byte(C.GoString(data)), nil
}

func decodeNativeSavePanel(result C.panelResult) (fyne.URI, error) {
	if result.state == 0 {
		return nil, nil
	}
	if result.state != 1 {
		return nil, errors.New("native save panel failed")
	}
	path := C.saveSelectionPath(result)
	defer C.free(unsafe.Pointer(path))
	if path == nil {
		C.releaseSaveSelection(result)
		return nil, errors.New("native save panel returned an invalid destination")
	}
	return fileaccess.NewDestination(storage.NewFileURI(C.GoString(path)), func() { C.releaseSaveSelection(result) }), nil
}

func darwinSaveTransport(path string) (fyne.URI, error) {
	name := C.CString(path)
	defer C.free(unsafe.Pointer(name))
	return decodeNativeSavePanel(C.testSaveSelection(name))
}

var _ = darwinSaveTransport
