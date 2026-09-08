package ui

import (
	"errors"
	"image/color"
	"os/exec"
	"runtime"
	"slices"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/filepicker"
	"github.com/frathe/picfetch/internal/ui/widgets"
	"github.com/frathe/picfetch/internal/uitest"
)

// Per-OS dispatch (Zenity/PowerShell/AppKit structured transport)
// is covered by internal/filepicker's own tests; the tests below exercise
// the viewer's integration with it - openFileDialog/runFileChooser wiring,
// error reporting, and the tappable drop-zone widget - which can't move
// since they depend on *viewer.

// The integration path uses UI admission, a native worker and queued delivery.
func TestRunFileChooser_LoadsSelectedImage(t *testing.T) {
	v, _, _ := newTestUI(t)

	jpegURI := uitest.TempJPEGURI(t, "picked.jpg", 20, 15, color.RGBA{R: 100, A: 255})
	uitest.StubChooser(t, []fyne.URI{jpegURI}, nil)

	v.openFileDialog()
	settleChooser(t, v)
	waitForScan(t, v)
	waitForSort(t, v)
	waitUntilLoaded(t, v)

	if !v.img.Visible() || v.img.Image == nil {
		t.Fatal("expected the chooser-selected image to load")
	}
}

func TestRunFileChooser_CancelLeavesStateUntouched(t *testing.T) {
	v, _, _ := newTestUI(t)

	// Cancellation is separate from an execution or transport failure.
	uitest.StubChooser(t, nil, nil)

	v.openFileDialog()
	settleChooser(t, v)

	if v.img.Visible() {
		t.Error("no image should be shown after a cancelled dialog")
	}
	if !v.welcomeArt.Visible() {
		t.Error("welcome art should still be showing after a cancelled dialog")
	}
}

func TestChooserErrorDetail_PrefersStderr(t *testing.T) {
	// A real failing command so err is an *exec.ExitError with Stderr
	// populated by Output(), the same as a failed osascript/zenity/
	// powershell call would produce.
	_, err := exec.Command("sh", "-c", "echo boom >&2; exit 1").Output()
	if err == nil {
		t.Fatal("expected the stub shell command to fail")
	}

	if got := chooserErrorDetail(err); got != "boom" {
		t.Errorf("chooserErrorDetail(err) = %q, want %q", got, "boom")
	}
}

func TestChooserErrorDetail_FallsBackToErrorString(t *testing.T) {
	err := errors.New("some other failure")

	if got := chooserErrorDetail(err); got != "some other failure" {
		t.Errorf("chooserErrorDetail(err) = %q, want %q", got, "some other failure")
	}
}

func TestReportChooserError_ToastsFailures(t *testing.T) {
	v := newTestViewer(t)
	v.reportChooserError(errors.New("boom"))
	if !v.toast.card.Visible() {
		t.Fatal("an execution/transport failure must be reported after cancellation is classified separately")
	}
	settleToast(t, v)
}

// TestOpenFileDialog_RunsChooserInBackground checks that openFileDialog
// actually reaches the native chooser on a background goroutine. The stub
// returns an error immediately, so runFileChooser takes its
// report-and-return path rather than reaching v.handleDrop; settleChooser
// then waits for that goroutine to finish, since the error path still
// renders a toast after the stub has returned.
func TestOpenFileDialog_RunsChooserInBackground(t *testing.T) {
	v, _, _ := newTestUI(t)

	called := make(chan struct{})
	orig := filepicker.Choose
	t.Cleanup(func() { filepicker.Choose = orig })
	filepicker.Choose = func() ([]fyne.URI, error) {
		close(called)
		return nil, errors.New("stub: not exercising the success path here")
	}

	v.openFileDialog()

	select {
	case <-called:
	case <-time.After(testTimeout):
		t.Fatal("expected openFileDialog to invoke the native chooser")
	}

	settleChooser(t, v)
}

// TestOpenShortcuts_InvokeFileDialog checks that wireOpenShortcuts
// (shortcuts.go) binds Cmd/Ctrl+O and Cmd/Ctrl+Shift+O to openFileDialog. It
// drives a bare *fyne.ShortcutHandler through the real wiring function rather
// than a full window/canvas: Fyne's test driver canvas (fyne.io/fyne/v2/test)
// embeds software.WindowlessCanvas by interface, which doesn't include
// TypedShortcut, so a real key-plus-modifier press can't be simulated
// through it - only the production glfw driver's canvas exposes that
// method (see wireOpenShortcuts's own comment). A bare ShortcutHandler is
// exactly what that driver's canvas embeds to do its own dispatch, so
// firing TypedShortcut on it exercises the same lookup-by-ShortcutName path
// a real press would.
func TestOpenShortcuts_InvokeFileDialog(t *testing.T) {
	tests := []struct {
		name     string
		modifier fyne.KeyModifier
	}{
		{"CmdOrCtrl+O", fyne.KeyModifierShortcutDefault},
		{"CmdOrCtrl+Shift+O", fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v, _, _ := newTestUI(t)

			called := make(chan struct{})
			orig := filepicker.Choose
			t.Cleanup(func() { filepicker.Choose = orig })
			filepicker.Choose = func() ([]fyne.URI, error) {
				close(called)
				return nil, errors.New("stub: not exercising the success path here")
			}

			handler := &fyne.ShortcutHandler{}
			wireOpenShortcuts(handler, v)
			handler.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: tt.modifier})

			select {
			case <-called:
			case <-time.After(2 * time.Second):
				t.Fatalf("expected the %s shortcut to invoke the native chooser", tt.name)
			}

			settleChooser(t, v)
		})
	}
}

func TestTappableArea_TappedInvokesCallback(t *testing.T) {
	called := false
	rect := canvas.NewRectangle(color.Black)
	ta := widgets.NewTappableArea(rect, func() { called = true })

	test.Tap(ta)

	if !called {
		t.Error("expected onTapped to be invoked on tap")
	}
}

func TestTappableArea_HoverInvokesOnHoverCallback(t *testing.T) {
	rect := canvas.NewRectangle(color.Black)
	ta := widgets.NewTappableArea(rect, func() {})

	var got []bool
	ta.OnHover = func(hovering bool) { got = append(got, hovering) }

	ta.MouseIn(&desktop.MouseEvent{})
	ta.MouseOut()

	if len(got) != 2 || got[0] != true || got[1] != false {
		t.Errorf("onHover calls = %v, want [true false] for MouseIn then MouseOut", got)
	}
}

func TestTappableArea_HoverIsOptional(t *testing.T) {
	rect := canvas.NewRectangle(color.Black)
	ta := widgets.NewTappableArea(rect, func() {})

	// onHover is left nil - MouseIn/MouseOut must not panic when no caller
	// has opted into hover feedback.
	ta.MouseIn(&desktop.MouseEvent{})
	ta.MouseMoved(&desktop.MouseEvent{})
	ta.MouseOut()
}

func TestE2E_TappingDropzoneArtOpensFileDialog(t *testing.T) {
	v, _, _ := newTestUI(t)

	called := make(chan struct{})
	orig := filepicker.Choose
	t.Cleanup(func() { filepicker.Choose = orig })
	filepicker.Choose = func() ([]fyne.URI, error) {
		close(called)
		return nil, errors.New("stub: not exercising the success path here")
	}

	test.Tap(v.dropzoneArt)

	select {
	case <-called:
	case <-time.After(testTimeout):
		t.Fatal("expected tapping the dropzone art to open the file dialog")
	}

	settleChooser(t, v)
}

func TestRunFileChooser_PreservesPathIdentity(t *testing.T) {
	v := newTestViewer(t)
	names := []string{"café 東京 😀.jpg", " spaced .jpg"}
	if runtime.GOOS != "windows" {
		names = []string{"line\nnext.jpg", "tail\r.jpg"}
	}
	paths := []fyne.URI{uitest.TempJPEGURI(t, names[0], 8, 4, color.White), uitest.TempJPEGURI(t, names[1], 4, 8, color.Black)}
	uitest.StubChooser(t, paths, nil)
	v.openFileDialog()
	settleChooser(t, v)
	waitForScan(t, v)
	waitForSort(t, v)
	waitUntilLoaded(t, v)
	if len(v.state.unsortedFiles) != len(paths) {
		t.Fatalf("opened files = %v", v.state.unsortedFiles)
	}
	for i, path := range paths {
		if v.state.unsortedFiles[i].Path() != path.Path() {
			t.Errorf("opened %q, want %q in selection order", v.state.unsortedFiles[i].Path(), path.Path())
		}
	}
}

func TestOpenChooser_ReverseCompletionKeepsLatestRequest(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.Black)
	release := make(chan struct{})
	called := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	original := filepicker.Choose
	t.Cleanup(func() { filepicker.Choose = original })
	filepicker.Choose = func() ([]fyne.URI, error) { close(called); <-release; return []fyne.URI{a}, nil }
	v.openFileDialog()
	first := v.chooser.Current()
	select {
	case <-called:
	case <-time.After(testTimeout):
		t.Fatal("first chooser did not start")
	}
	filepicker.Choose = func() ([]fyne.URI, error) { return []fyne.URI{b}, nil }
	v.openFileDialog()
	waitFor(t, "second chooser worker", &v.chooser)
	unblock()
	waitHandle(t, "first chooser worker", first)
	settleChooser(t, v)
	waitForScan(t, v)
	waitForSort(t, v)
	waitUntilLoaded(t, v)
	if len(v.state.files) != 1 || v.state.files[0].String() != b.String() {
		t.Fatalf("older chooser replaced the latest selection: %v", v.state.files)
	}
}

func TestOpenChooser_InterveningInputDiscardsHeldResult(t *testing.T) {
	for _, action := range []string{"drop", "clear"} {
		t.Run(action, func(t *testing.T) {
			v := newTestViewer(t)
			a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
			b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.Black)
			c := uitest.TempJPEGURI(t, "c.jpg", 4, 4, color.White)
			dropAndWait(t, v, b)
			release := make(chan struct{})
			called := make(chan struct{})
			var once sync.Once
			unblock := func() { once.Do(func() { close(release) }) }
			defer unblock()
			original := filepicker.Choose
			t.Cleanup(func() { filepicker.Choose = original })
			filepicker.Choose = func() ([]fyne.URI, error) { close(called); <-release; return []fyne.URI{a}, nil }
			v.openFileDialog()
			select {
			case <-called:
			case <-time.After(testTimeout):
				t.Fatal("chooser did not start")
			}
			if action == "drop" {
				dropAndWait(t, v, c)
			} else {
				v.closeFiles()
			}
			expected := slices.Clone(v.state.files)
			unblock()
			settleChooser(t, v)
			waitForScan(t, v)
			waitForSort(t, v)
			waitUntilLoaded(t, v)
			if !slices.EqualFunc(v.state.files, expected, func(a, b fyne.URI) bool { return a.String() == b.String() }) {
				t.Fatalf("obsolete result replaced %s state: %v", action, v.state.files)
			}
		})
	}
}

func TestOpenChooser_CurrentErrorWaitsForUIDelivery(t *testing.T) {
	v := newTestViewer(t)
	queue := &uitest.UIQueue{}
	v.chooserUI = queue
	uitest.StubChooser(t, nil, errors.New("native chooser failed"))
	v.openFileDialog()
	waitFor(t, "native chooser worker", &v.chooser)
	if v.toast.card.Visible() {
		t.Fatal("worker painted the error before UI delivery")
	}
	if queue.Len() != 1 {
		t.Fatalf("queued results = %d", queue.Len())
	}
	queue.Drain()
	if !v.toast.card.Visible() {
		t.Fatal("current failure did not reach UI")
	}
	settleToast(t, v)
}

func TestOpenChooser_SupersededErrorIsSilent(t *testing.T) {
	v := newTestViewer(t)
	release := make(chan struct{})
	called := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	original := filepicker.Choose
	t.Cleanup(func() { filepicker.Choose = original })
	filepicker.Choose = func() ([]fyne.URI, error) { close(called); <-release; return nil, errors.New("obsolete failure") }
	v.openFileDialog()
	first := v.chooser.Current()
	select {
	case <-called:
	case <-time.After(testTimeout):
		t.Fatal("first chooser did not start")
	}
	filepicker.Choose = func() ([]fyne.URI, error) { return nil, nil }
	v.openFileDialog()
	waitFor(t, "new chooser worker", &v.chooser)
	unblock()
	waitHandle(t, "old chooser worker", first)
	settleChooser(t, v)
	if v.toast.card.Visible() {
		t.Fatal("obsolete chooser painted an error")
	}
}

func TestOpenChooser_QueuedDeliveryRevalidatesComparison(t *testing.T) {
	for _, failure := range []bool{false, true} {
		name := "selection"
		if failure {
			name = "failure"
		}
		t.Run(name, func(t *testing.T) {
			v := newTestViewer(t)
			a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
			b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.Black)
			incoming := uitest.TempJPEGURI(t, "incoming.jpg", 4, 4, color.White)
			dropAndWait(t, v, a, b)
			queue := &uitest.UIQueue{}
			v.chooserUI = queue
			var resultErr error
			if failure {
				resultErr = errors.New("obsolete panel error")
			}
			uitest.StubChooser(t, []fyne.URI{incoming}, resultErr)
			v.openFileDialog()
			waitFor(t, "native chooser worker", &v.chooser)
			if queue.Len() != 1 {
				t.Fatalf("queued deliveries = %d", queue.Len())
			}
			v.compare.Open([2]fyne.URI{a, b})
			waitForCompare(t, v)
			expected := snapshotCompareCommands(v)
			queue.Drain()
			if got := snapshotCompareCommands(v); got != expected {
				t.Error("queued result changed comparison state")
			}
			if v.toast.card.Visible() {
				t.Fatal("comparison-covered delivery painted a stale toast")
			}
		})
	}
}

func TestOpenChooser_AdmissionRunsBeforeNativeWork(t *testing.T) {
	for _, entry := range []string{"direct", "menu", "shortcut"} {
		t.Run(entry, func(t *testing.T) {
			v := openActiveComparisonWithExtra(t)
			var calls atomic.Int32
			original := filepicker.Choose
			t.Cleanup(func() { filepicker.Choose = original })
			filepicker.Choose = func() ([]fyne.URI, error) { calls.Add(1); return nil, nil }
			switch entry {
			case "direct":
				v.openFileDialog()
			case "menu":
				buildMainMenu(v).Items[0].Items[0].Action()
			case "shortcut":
				handler := &fyne.ShortcutHandler{}
				wireOpenShortcuts(handler, v)
				handler.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: fyne.KeyModifierShortcutDefault})
			}
			if !v.toast.card.Visible() {
				t.Fatal("UI entry did not paint comparison refusal")
			}
			if v.chooser.Begun() {
				settleChooser(t, v)
			}
			if calls.Load() != 0 {
				t.Fatal("comparison refusal reached the native chooser")
			}
			settleToast(t, v)
		})
	}
}

func TestOpenChooser_ShutdownDiscardsHeldResult(t *testing.T) {
	application := test.NewApp()
	v, win := buildStartupViewer(application)
	v.grid.SetUIQueue(&uitest.UIQueue{})
	v.compare.SetUIQueue(&uitest.UIQueue{})
	v.mosaicWin.SetUIQueue(&uitest.UIQueue{})
	v.chooserUI = &uitest.UIQueue{}
	t.Cleanup(win.Close)
	t.Cleanup(func() { drain(t, v) })
	before := uitest.TempJPEGURI(t, "before.jpg", 4, 4, color.White)
	incoming := uitest.TempJPEGURI(t, "incoming.jpg", 4, 4, color.Black)
	dropAndWait(t, v, before)
	v.preloads.Wait()
	release := make(chan struct{})
	called := make(chan struct{})
	var once sync.Once
	unblock := func() { once.Do(func() { close(release) }) }
	defer unblock()
	var calls atomic.Int32
	originalChooser := filepicker.Choose
	t.Cleanup(func() { filepicker.Choose = originalChooser })
	filepicker.Choose = func() ([]fyne.URI, error) {
		if calls.Add(1) == 1 {
			close(called)
			<-release
		}
		return []fyne.URI{incoming}, nil
	}
	v.openFileDialog()
	select {
	case <-called:
	case <-time.After(testTimeout):
		t.Fatal("native chooser did not start")
	}
	lifecycle, ok := application.Lifecycle().(interface{ OnStopped() func() })
	if !ok {
		t.Fatal("test lifecycle has no stopped hook")
	}
	original := lifecycle.OnStopped()
	registerShutdown(application, v)
	shutdown := lifecycle.OnStopped()
	application.Lifecycle().SetOnStopped(original)
	// Returns while the external panel is still held. Its worker remains
	// tracked until it returns, then its invalid token suppresses delivery.
	shutdown()
	scanRevision := v.scanOp.lifecycle.currentRevision()
	unblock()
	settleChooser(t, v)
	if v.scanOp.lifecycle.currentRevision() != scanRevision {
		t.Fatal("held result started a scan after shutdown")
	}
	if len(v.state.files) != 1 || v.state.files[0].String() != before.String() {
		t.Fatal("held chooser changed the stopped viewer")
	}
	v.openFileDialog()
	settleChooser(t, v)
	if calls.Load() != 1 {
		t.Fatal("shutdown admitted another native chooser")
	}
}

func TestOpenChooser_QueuedDeliveryIsDiscardedAfterReset(t *testing.T) {
	v := newTestViewer(t)
	before := uitest.TempJPEGURI(t, "before.jpg", 4, 4, color.White)
	incoming := uitest.TempJPEGURI(t, "incoming.jpg", 4, 4, color.Black)
	dropAndWait(t, v, before)
	queue := &uitest.UIQueue{}
	v.chooserUI = queue
	uitest.StubChooser(t, []fyne.URI{incoming}, nil)
	v.openFileDialog()
	waitFor(t, "chooser worker before reset", &v.chooser)
	if queue.Len() != 1 {
		t.Fatal("result was not queued")
	}
	v.reset()
	revision := v.scanOp.lifecycle.currentRevision()
	queue.Drain()
	if len(v.state.files) != 0 || v.scanOp.lifecycle.currentRevision() != revision {
		t.Fatal("queued obsolete result restarted opening files after reset")
	}
}
