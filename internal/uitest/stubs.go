package uitest

import (
	"testing"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/clipboard"
	"github.com/frathe/picfetch/internal/displays"
	"github.com/frathe/picfetch/internal/filemanager"
	"github.com/frathe/picfetch/internal/filepicker"
	"github.com/frathe/picfetch/internal/trash"
	"github.com/frathe/picfetch/internal/wallpaper"
)

// The stubs below swap the exported dispatcher vars that stand in front of
// this app's OS-level shell-outs (a native file dialog, a clipboard copy, a
// trash move, a wallpaper change, a file-manager reveal). Those vars exist
// precisely so tests never actually open a dialog, touch the system
// clipboard, move a file to the real Trash, or raise a file-manager window;
// each restores the real implementation on cleanup, so a test that stubs
// one can't affect the ones that follow.

// StubChooser makes filepicker.Choose return out/err instead of opening the
// OS file browser.
func StubChooser(t *testing.T, out []byte, err error) {
	t.Helper()

	orig := filepicker.Choose
	t.Cleanup(func() { filepicker.Choose = orig })
	filepicker.Choose = func() ([]byte, error) { return out, err }
}

// StubDisplays makes displays.Inspect use a deterministic topology without
// reading the developer's attached monitors.
func StubDisplays(t *testing.T, fn func(fyne.Window) (displays.Snapshot, error)) {
	t.Helper()

	orig := displays.Inspect
	t.Cleanup(func() { displays.Inspect = orig })
	displays.Inspect = fn
}

// StubSaveChooser makes filepicker.ChooseSave call fn instead of opening
// the OS save panel. It takes a function rather than a fixed result the way
// StubChooser does, since a caller usually wants to assert on the suggested
// path it was offered as well as control what comes back.
func StubSaveChooser(t *testing.T, fn func(suggestedPath string) ([]byte, error)) {
	t.Helper()

	orig := filepicker.ChooseSave
	t.Cleanup(func() { filepicker.ChooseSave = orig })
	filepicker.ChooseSave = fn
}

// StubClipboardCopy makes clipboard.CopyImage call fn instead of shelling
// out to the OS clipboard.
func StubClipboardCopy(t *testing.T, fn func(data []byte) error) {
	t.Helper()

	orig := clipboard.CopyImage
	t.Cleanup(func() { clipboard.CopyImage = orig })
	clipboard.CopyImage = fn
}

// StubClipboardCopyFiles makes clipboard.CopyFiles call fn instead of
// shelling out to the OS clipboard - the file-reference twin of
// StubClipboardCopy, for the grid's batch copy.
func StubClipboardCopyFiles(t *testing.T, fn func(paths []string) error) {
	t.Helper()

	orig := clipboard.CopyFiles
	t.Cleanup(func() { clipboard.CopyFiles = orig })
	clipboard.CopyFiles = fn
}

// StubTrashMove makes trash.Move call fn instead of shelling out to the
// OS's real trash/recycle-bin mover.
func StubTrashMove(t *testing.T, fn func(path string) error) {
	t.Helper()

	orig := trash.Move
	t.Cleanup(func() { trash.Move = orig })
	trash.Move = fn
}

// StubReveal makes filemanager.Reveal call fn instead of opening the
// developer's own Finder/Explorer/Nautilus window - the reveal twin of
// StubTrashMove, and, like StubWallpaperSet, a stub whose absence a test run
// would leave visibly behind on their screen.
func StubReveal(t *testing.T, fn func(path string) error) {
	t.Helper()

	orig := filemanager.Reveal
	t.Cleanup(func() { filemanager.Reveal = orig })
	filemanager.Reveal = fn
}

// StubWallpaperSet makes wallpaper.Set call fn instead of changing the
// machine's real desktop wallpaper - the one stub here whose absence a test
// run would leave visibly behind on the developer's own screen.
func StubWallpaperSet(t *testing.T, fn any) {
	t.Helper()

	orig := wallpaper.Set
	t.Cleanup(func() { wallpaper.Set = orig })
	switch set := fn.(type) {
	case func(string) error:
		// Keep older viewer tests terse while the production seam carries the
		// optional display target. Tests concerned with targeting pass the full
		// Request form below.
		wallpaper.Set = func(request wallpaper.Request) error { return set(request.Path) }
	case func(wallpaper.Request) error:
		wallpaper.Set = set
	default:
		t.Fatalf("unsupported wallpaper stub type %T", fn)
	}
}
