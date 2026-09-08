package ui

import (
	"image/color"
	"os"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	fynetest "fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/uitest"
)

// The confirmation flow's own state machine - selection, ring visuals, the
// three outcomes of a confirmed delete - is covered in internal/ui/deletion
// against a fake host. What stays here is the viewer's side of it: that the
// key dispatcher hands over while the card is up, that a drop dismisses it,
// that the global shortcut reaches it, and that a real confirmed delete
// moves real files and lands the viewer in the right state afterwards.
//
// These drive the component only through the API the app itself uses
// (Request, HandleKey, Visible) rather than reaching into its internals, so
// nothing here needs an accessor that exists only for tests.

// confirmDelete opens the confirmation and confirms it: Right moves the
// selection onto the danger button, Return commits - exactly the key
// sequence a user performs - then settles the background trash-move
// goroutine performDelete starts (see its doc comment in deletion.go),
// so every caller lands after the whole thing has actually finished.
func confirmDelete(t *testing.T, v *viewer) {
	t.Helper()

	v.deletion.Request()
	if !v.deletion.Visible() {
		t.Fatal("setup: the confirmation card should be up after Request")
	}

	v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	v.deletion.Settle()
}

type deletionCompletionQueue struct {
	uitest.UIQueue
	queued chan struct{}
}

func (q *deletionCompletionQueue) Do(f func()) {
	q.UIQueue.Do(f)
	q.queued <- struct{}{}
}

func TestDeletion_ShutdownDiscardsQueuedCompletion(t *testing.T) {
	uitest.StubTrashMove(t, func(path string) error { return os.Remove(path) })
	// Shutdown persists session/preferences, so it needs its own app/cache.
	application := fynetest.NewApp()
	v, win := buildStartupViewer(application)
	v.grid.SetUIQueue(&uitest.UIQueue{})
	v.compare.SetUIQueue(&uitest.UIQueue{})
	v.mosaicWin.SetUIQueue(&uitest.UIQueue{})
	t.Cleanup(win.Close)
	t.Cleanup(func() { drain(t, v) })
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)
	v.preloads.Wait()
	queue := &deletionCompletionQueue{queued: make(chan struct{}, 1)}
	v.deletion.SetUIQueue(queue)
	v.deletion.Request()
	v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	select {
	case <-queue.queued:
	case <-time.After(testTimeout):
		t.Fatal("trash completion was not queued")
	}
	if _, err := os.Stat(a.Path()); !os.IsNotExist(err) {
		t.Fatalf("confirmed file was not moved: %v", err)
	}
	lifecycle, ok := application.Lifecycle().(interface{ OnStopped() func() })
	if !ok {
		t.Fatal("test app lifecycle does not expose its stopped hook")
	}
	original := lifecycle.OnStopped()
	registerShutdown(application, v)
	shutdown := lifecycle.OnStopped()
	application.Lifecycle().SetOnStopped(original)
	shutdown()
	queue.Drain()
	if len(v.state.files) != 2 || v.state.files[0].String() != a.String() {
		t.Errorf("late deletion changed the stopped viewer: %v", v.state.files)
	}
	v.deletion.Request()
	if v.deletion.Visible() {
		t.Error("stopped deletion feature accepted another confirmation")
	}
	v.deletion.Settle()
}

func TestDeletion_ReorderBeforeConfirmationPreservesIdentity(t *testing.T) {
	uitest.StubTrashMove(t, func(path string) error { return os.Remove(path) })
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)
	v.preloads.Wait()
	v.deletion.Request()
	v.state.reorder([]fyne.URI{b, a})
	v.state.index = 1
	v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	v.deletion.Settle()
	waitUntilLoaded(t, v)
	if _, err := os.Stat(a.Path()); !os.IsNotExist(err) {
		t.Errorf("confirmed file survived: %v", err)
	}
	if _, err := os.Stat(b.Path()); err != nil {
		t.Errorf("unconfirmed file did not survive: %v", err)
	}
	if len(v.state.files) != 1 || v.state.files[0].String() != b.String() || v.imgCache.Contains(a.String()) {
		t.Errorf("deleted identity remains in the model/cache: files=%v cached=%v", v.state.files, v.imgCache.Contains(a.String()))
	}
}

func TestDeletion_ComparisonOpenedDuringMoveReturnsToReconciledFiles(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)
	v.preloads.Wait()
	entered, release := make(chan struct{}), make(chan struct{})
	releaseWorker := sync.OnceFunc(func() { close(release) })
	uitest.StubTrashMove(t, func(path string) error {
		close(entered)
		<-release
		return os.Remove(path)
	})
	t.Cleanup(func() { releaseWorker(); v.deletion.Settle() })
	v.deletion.Request()
	v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.deletion.HandleKey(&fyne.KeyEvent{Name: fyne.KeyReturn})
	select {
	case <-entered:
	case <-time.After(testTimeout):
		t.Fatal("trash worker did not start")
	}
	v.compare.Open([2]fyne.URI{a, b})
	waitForCompare(t, v)
	releaseWorker()
	v.deletion.Settle()
	waitUntilLoaded(t, v)
	if v.comparisonActive() || len(v.state.files) != 1 || v.state.files[0].String() != b.String() {
		t.Errorf("completed deletion was suppressed by comparison: active=%v files=%v", v.comparisonActive(), v.state.files)
	}
	if _, err := os.Stat(a.Path()); !os.IsNotExist(err) {
		t.Errorf("confirmed file survived: %v", err)
	}
}

// TestHandleKeyEvent_DeleteConfirmSwallowsNavigationButRespondsToItsOwnKeys
// guards the core safety property of the confirmation: while it's up,
// ordinary navigation must not slip through and change what's on screen,
// but its own keys must still work - which is the dispatcher's job, not the
// component's.
func TestHandleKeyEvent_DeleteConfirmSwallowsNavigationButRespondsToItsOwnKeys(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)

	v.deletion.Request()
	startIndex := v.state.index

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	if v.state.index != startIndex {
		t.Error("arrow-key navigation should be swallowed while the delete confirmation is up")
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if v.deletion.Visible() {
		t.Error("Escape should dismiss the confirmation instead of falling through to its usual meaning")
	}
	if len(v.state.files) != 2 {
		t.Error("Escape on the confirmation must not also reset the loaded file set")
	}
}

// TestPerformDelete_RemovesCurrentFileAndAdvancesToTheNextOne covers the
// common case end to end, through the real viewer: two files loaded, delete
// the first, and the second one - which has shifted down to index 0 -
// should end up on screen.
func TestPerformDelete_RemovesCurrentFileAndAdvancesToTheNextOne(t *testing.T) {
	uitest.StubTrashMove(t, func(path string) error { return os.Remove(path) })
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)

	confirmDelete(t, v)
	waitUntilLoaded(t, v)

	if _, err := os.Stat(a.Path()); !os.IsNotExist(err) {
		t.Errorf("a.jpg should no longer exist on disk, stat = %v", err)
	}
	if len(v.state.files) != 1 || v.state.files[0].String() != b.String() {
		t.Fatalf("files = %v, want just b.jpg left", v.state.files)
	}
	if v.state.index != 0 {
		t.Errorf("index = %d, want 0 (b.jpg took a.jpg's slot)", v.state.index)
	}
	if !v.toast.card.Visible() {
		t.Error("expected a toast confirming the deletion")
	}
	settleToast(t, v)
}

// TestPerformDelete_OnLastImageOfMultipleAdvancesWithoutPanicking is a
// regression test: deleting while positioned on the last image of a
// multi-file set left v.state.index equal to the new (shrunk) length, so the very
// next CurrentFile() call - performDelete's own "did that empty the set?"
// check - indexed v.state.files out of range and crashed the whole app.
func TestPerformDelete_OnLastImageOfMultipleAdvancesWithoutPanicking(t *testing.T) {
	uitest.StubTrashMove(t, func(path string) error { return os.Remove(path) })
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, a, b)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	waitUntilLoaded(t, v)
	if v.state.index != 1 {
		t.Fatalf("setup: index = %d, want 1 (on b.jpg, the last image)", v.state.index)
	}

	confirmDelete(t, v)
	waitUntilLoaded(t, v)

	if len(v.state.files) != 1 || v.state.files[0].String() != a.String() {
		t.Fatalf("files = %v, want just a.jpg left", v.state.files)
	}
	if v.state.index != 0 {
		t.Errorf("index = %d, want 0 (a.jpg took b.jpg's slot)", v.state.index)
	}
	settleToast(t, v)
}

// TestPerformDelete_LastFileReturnsToEmptyDropzone covers deleting the only
// remaining file: the app should fall back to the empty-state screen, the
// same place a last decode failure already lands it.
func TestPerformDelete_LastFileReturnsToEmptyDropzone(t *testing.T) {
	uitest.StubTrashMove(t, func(path string) error { return os.Remove(path) })
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	dropAndWait(t, v, a)

	confirmDelete(t, v)

	if len(v.state.files) != 0 {
		t.Error("v.state.files should be empty after deleting the last file")
	}
	if v.dropzone == nil || !v.dropzone.Visible() {
		t.Error("expected the drop zone to reappear once nothing is left")
	}
	if !v.emptyStateArt.Visible() {
		t.Error("expected the empty-state art, matching a last decode failure")
	}
	settleToast(t, v)
}

// TestPerformDelete_OSFailureKeepsTheFileAndToastsAnError guards the
// trash.Move error path through the real viewer: if the move fails, the
// file must stay in v.state.files (nothing silently dropped from the set for a
// file that's actually still there) and the user must be told.
func TestPerformDelete_OSFailureKeepsTheFileAndToastsAnError(t *testing.T) {
	uitest.StubTrashMove(t, func(path string) error { return os.Remove(path) })
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	dropAndWait(t, v, a)

	// Remove the file out from under the app first, so the stubbed
	// trash.Move deterministically fails with "no such file", regardless of
	// what permissions the test process happens to run with.
	if err := os.Remove(a.Path()); err != nil {
		t.Fatalf("pre-removing the file: %v", err)
	}

	confirmDelete(t, v)

	if len(v.state.files) != 1 {
		t.Error("a file that failed to delete must stay in v.state.files")
	}
	if !v.toast.card.Visible() {
		t.Error("expected a toast reporting the deletion failure")
	}
	settleToast(t, v)
}

// TestHandleDrop_CancelsAnyPendingDeleteConfirmation guards against a fresh
// drop landing while the confirmation card is still up over a file list
// that drop is about to replace.
func TestHandleDrop_CancelsAnyPendingDeleteConfirmation(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	dropAndWait(t, v, a)

	v.deletion.Request()
	if !v.deletion.Visible() {
		t.Fatal("setup: expected the confirmation to be visible")
	}

	b := uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White)
	dropAndWait(t, v, b)

	if v.deletion.Visible() {
		t.Error("a fresh drop should dismiss any pending delete confirmation")
	}
}

// TestWireDeleteShortcut_ShiftDeleteOpensConfirmation_PlainCutIgnored is the
// same class of regression test as TestWireClipboardShortcuts's: Fyne's
// glfw driver special-cases bare Shift+Delete into &fyne.ShortcutCut{
// Secondary: true} before a desktop.CustomShortcut registration would ever
// see it, so wireDeleteShortcut must bind &fyne.ShortcutCut{} and check
// Secondary itself. Firing a CustomShortcut here would pass even if
// wireDeleteShortcut bound the wrong type, so this fires the exact shortcut
// shape the real driver produces instead.
func TestWireDeleteShortcut_ShiftDeleteOpensConfirmation_PlainCutIgnored(t *testing.T) {
	v, _, _ := newTestUI(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	dropAndWait(t, v, a)

	handler := &fyne.ShortcutHandler{}
	wireDeleteShortcut(handler, v)

	// A genuine Ctrl/Cmd+X reaches the same handler (ShortcutName() ==
	// "Cut" either way) with Secondary false; this app has no cut action,
	// so it must be ignored rather than opening a delete prompt.
	handler.TypedShortcut(&fyne.ShortcutCut{})
	if v.deletion.Visible() {
		t.Fatal("a plain Cut shortcut (Secondary=false) must not open the delete confirmation")
	}

	// Shift+Delete: Secondary true.
	handler.TypedShortcut(&fyne.ShortcutCut{Secondary: true})
	if !v.deletion.Visible() {
		t.Error("Shift+Delete (ShortcutCut with Secondary=true) should open the delete confirmation")
	}
}
