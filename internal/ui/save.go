// Persisting rotate.go's view-only rotation back to disk.

package ui

import (
	"fmt"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/imaging"
)

// canSaveRotation reports whether the File > Save Changes item (also
// Cmd/Ctrl+S) should be enabled: an image is loaded, its view rotation
// (rotate.go) is non-zero, it isn't mid-load, it's a single frame, and its
// format has a registered encoder. Checked here rather than only inside
// saveRotation so the item is never offered for an action guaranteed to
// fail or do nothing.
//
//   - !v.display.Snapshot().Loading: ShowImage sets v.state.index to the file being
//     navigated to before that file's pixels have finished decoding, so
//     mid-load, CurrentFile() already names the new file while
//     v.display/v.img.Image still hold the old one's frames - saving then
//     would write the wrong file's pixels into the new file's path.
//   - v.display.Count() == 1: an animated GIF's frames would all need
//     re-rotating and re-encoding as a fresh animation, which SaveRotated
//     doesn't attempt.
//   - imaging.CanEncode: WebP/HEIC/ICO/XPM have no encoder in this module's
//     dependencies (see save.go's own doc comment in internal/imaging).
func (v *viewer) canSaveRotation() bool {
	snapshot := v.display.Snapshot()
	u := snapshot.Displayed.Source
	if u == nil {
		return false
	}

	return !v.fileWork.closed && !v.fileWork.savePending && snapshot.Rotation != 0 && !snapshot.Loading && v.display.Count() == 1 && imaging.CanEncode(u)
}

// saveRotation is the File menu's "Save Changes" action (also Cmd/Ctrl+S,
// see wireSaveShortcut in shortcuts.go): it writes the currently displayed,
// already-rotated frame back to the file it came from, in that file's own
// format. A no-op unless canSaveRotation() is currently true - re-checked
// here rather than trusted from the menu item's Disabled state, since the
// shortcut bypasses the menu entirely.
func (v *viewer) saveRotation() {
	if v.comparisonActive() {
		return
	}
	if !v.canSaveRotation() {
		return
	}

	capture, ok := v.display.Capture()
	if !ok {
		return
	}
	u := capture.Identity.Source
	token := v.fileWork.saveLifecycle.begin()
	done := v.fileWork.saveDone.Begin()
	v.fileWork.savePending = true
	v.syncMenus()
	save := v.fileWork.save
	v.fileWork.workers.Go(func() {
		result, err := save(token.context(), u, capture.Pixels)
		if result.Committed {
			v.imgCache.Purge()
		}
		if !token.current() && !result.Committed {
			done()
			return
		}
		v.fileWork.ui.Do(func() {
			current := token.current()
			defer token.cancelContext()
			defer v.afterFileWrite(result, !current, true, done)
			if !token.current() {
				return
			}
			v.fileWork.savePending = false
			defer v.syncMenus()
			if err != nil {
				fyne.LogError("failed to save rotation", err)
				v.ShowToast(fmt.Sprintf(lang.L("could not save %q: %v"), u.Name(), err))
				return
			}
			if result.Committed {
				v.display.ReconcileSaved(capture)
			}
			v.ShowToast(lang.L("Saved"))
		})
	})
}
