package ui

import (
	"image/color"
	"testing"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/uitest"
)

// Exercise real callbacks, including a stale/disabled item's Action. A wrapper
// around a dummy function cannot prove the bare command guards its effects.
func TestMenuCallbacksRecheckAdmission(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))
	uitest.StubReveal(t, func(_ string) error { t.Error("modal reveal reached OS"); return nil })
	uitest.StubWallpaperSet(t, func(_ string) error { t.Error("modal wallpaper reached OS"); return nil })
	uitest.StubClipboardCopy(t, func(_ []byte) error { t.Error("modal copy reached OS"); return nil })
	v.requestDelete()
	before := snapshotCompareCommands(v)
	windows := len(v.app.Driver().AllWindows())
	var invoke func(*fyne.Menu)
	invoke = func(menu *fyne.Menu) {
		for _, item := range menu.Items {
			if item.IsSeparator {
				continue
			}
			if item.ChildMenu != nil {
				invoke(item.ChildMenu)
			}
			if item.Action == nil {
				continue
			}
			t.Run(item.Label, func(t *testing.T) {
				item.Action()
				if got := snapshotCompareCommands(v); got != before {
					t.Errorf("modal callback changed state: %+v", got)
				}
				if !v.deletion.Visible() || v.win.Canvas().Overlays().Top() != nil {
					t.Error("callback replaced the prompt")
				}
				if len(v.app.Driver().AllWindows()) != windows {
					t.Error("callback opened a window")
				}
			})
		}
	}
	for _, menu := range v.win.MainMenu().Items {
		invoke(menu)
	}
	if v.fileWork.saveDone.Begun() || v.fileWork.exportPending || v.clipboard.Begun() || v.chooser.Begun() {
		t.Fatal("modal callback started work")
	}
}
