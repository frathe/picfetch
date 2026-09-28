package ui

import (
	"context"
	"image"
	"image/color"
	"sync"
	"testing"
	"testing/synctest"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/displays"
	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/similarity"
	explorerui "github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestCommandAdmissionBusyCopy(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, regionCopyPNGURI(t, "photo.png", markedRegionCopyImage(10, 8)))
	v.rotateBy(1)
	release := make(chan struct{})
	finish := sync.OnceFunc(func() { close(release) })
	t.Cleanup(finish)
	uitest.StubClipboardCopy(t, func(_ []byte) error { <-release; return nil })
	selectRegion(t, v, image.Rect(2, 2, 6, 6))
	v.regionCopy.HandleKey(fyne.KeyReturn)
	if !v.regionCopy.State().Busy {
		t.Fatal("premise: region copy is not pending")
	}
	for name, item := range map[string]*fyne.MenuItem{"Save": v.menus.Save(), "Copy Path": v.menus.Actions().CopyPath()} {
		if !item.Disabled {
			t.Errorf("busy region copy did not disable %s", name)
		}
	}
	v.saveRotation()
	if v.fileWork.saveDone.Begun() || !v.toast.card.Visible() {
		t.Error("busy Save did not refuse with feedback")
	}
	finish()
	waitForClipboard(t, v)
	for name, item := range map[string]*fyne.MenuItem{"Save": v.menus.Save(), "Copy Path": v.menus.Actions().CopyPath(), "Copy": v.menus.Actions().Copy()} {
		if item.Disabled {
			t.Errorf("completed region copy did not restore %s", name)
		}
	}
	settleToast(t, v)
	for _, cancelCopy := range []bool{false, true} {
		v.clipboardWork.workers.Wait()
		ordinaryRelease := make(chan struct{})
		ordinaryFinish := sync.OnceFunc(func() { close(ordinaryRelease) })
		t.Cleanup(ordinaryFinish)
		uitest.StubClipboardCopy(t, func(_ []byte) error { <-ordinaryRelease; return nil })
		v.copyImageToClipboard()
		if !v.menus.Actions().Copy().Disabled || !v.menus.Actions().CopyPath().Disabled || v.menus.Save().Disabled {
			t.Error("ordinary copy must disable clipboard commands without blocking Save")
		}
		if cancelCopy {
			v.cancelImageClipboard()
		}
		ordinaryFinish()
		waitForClipboard(t, v)
		if v.menus.Actions().Copy().Disabled || v.menus.Actions().CopyPath().Disabled {
			t.Errorf("ordinary copy cancellation=%v did not restore clipboard menus", cancelCopy)
		}
	}
}

func TestCommandAdmissionAsync(t *testing.T) {
	t.Run("duplicate preparation rechecks admission", func(t *testing.T) {
		for _, entry := range []string{"Explorer", "Location Map", "Variants"} {
			t.Run(entry, func(t *testing.T) {
				synctest.Test(t, func(t *testing.T) {
					fixture := newGridAnalysisFixture(t, true)
					v := fixture.v
					t.Cleanup(func() {
						if v.toast.stop != nil {
							settleToast(t, v)
						}
					})
					calls := 0
					configureExplorer(v, func(options *explorerui.Options) {
						options.Supported, options.AssetsReady, options.Settings.IntroSeen = true, true, true
						options.Analyze = func(_ context.Context, _ []string, _ <-chan similarity.Control, _ func(similarity.Event)) error {
							calls++
							return nil
						}
					})
					if entry == "Variants" {
						v.grid.Close()
						v.ShowImage(4)
						v.browseCurrentDuplicates()
					} else if entry == "Explorer" {
						v.showExplorer()
					} else {
						v.showLocationMap()
					}
					fixture.deliver()
					if entry == "Variants" && (!v.grid.BrowsingDuplicates() || v.grid.BrowseReady()) {
						t.Fatal("premise: variant preparation was not held")
					}
					if entry != "Variants" && v.explorerInput.prepare == nil && v.locationInput.prepare == nil {
						t.Fatal("premise: preparation was not held")
					}
					prompt := widget.NewModalPopUp(widget.NewLabel("owner"), v.win.Canvas())
					prompt.Show()
					defer prompt.Hide()
					for _, i := range []int{4, 5, 6} {
						fixture.releaseNext(i)
					}
					fixture.deliver()
					v.grid.Settle()
					v.locationMap.Settle()
					v.settleExplorer()
					if calls != 0 || v.locationMap.Active() || v.explorer.Surface().Visible() {
						t.Fatal("preparation delivered beneath new modal")
					}
					if entry == "Variants" && (v.grid.Visible() || v.grid.BrowsingDuplicates()) {
						t.Fatal("variant preparation opened or retained a refused visit")
					}
					if v.win.Canvas().Overlays().Top() == nil {
						t.Fatal("preparation dismissed the owner")
					}
				})
			})
		}
	})
	t.Run("queued chooser observes modal ownership", func(t *testing.T) {
		v := newTestViewer(t)
		source := uitest.TempJPEGURI(t, "original.jpg", 40, 20, color.White)
		incoming := uitest.TempJPEGURI(t, "incoming.jpg", 40, 20, color.Black)
		dropAndWait(t, v, source)
		queue := &uitest.UIQueue{}
		v.chooserUI = queue
		uitest.StubChooser(t, []fyne.URI{incoming}, nil)
		v.openFileDialog()
		waitFor(t, "queued chooser", &v.chooser)
		v.requestDelete()
		queue.Drain()
		if !v.deletion.Visible() || v.FileAt(0).String() != source.String() {
			t.Fatal("chooser delivery bypassed new modal ownership")
		}
		v.deletion.Cancel()
		queue.Drain()
		if v.FileAt(0).String() != source.String() {
			t.Fatal("refused chooser was replayed")
		}
	})
	t.Run("committed save reconciles beneath modal", func(t *testing.T) {
		v := newTestViewer(t)
		source := regionCopyPNGURI(t, "photo.png", markedRegionCopyImage(10, 8))
		dropAndWait(t, v, source)
		v.rotateBy(1)
		v.fileWork.ui = &uitest.UIQueue{}
		v.saveRotation()
		v.fileWork.workers.Wait()
		v.requestDelete()
		waitForSave(t, v)
		v.display.Settle()
		loaded, err := imaging.LoadImage(source, imaging.DefaultImgCacheBytes)
		if err != nil {
			t.Fatal(err)
		}
		if loaded.Frames[0].Bounds().Size() != image.Pt(8, 10) || v.display.Rotation() != 0 || !v.deletion.Visible() {
			t.Fatal("modal lost committed save reconciliation")
		}
		settleToast(t, v)
	})
	t.Run("navigation and admitted reconciliation", func(t *testing.T) {
		v := newTestViewer(t)
		dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White), uitest.TempJPEGURI(t, "b.jpg", 40, 20, color.Black))
		before := v.state.Observe().index
		v.requestDelete()
		v.ShowImage(before + 1)
		v.display.Settle()
		if v.state.Observe().index != before {
			t.Fatal("fresh image selection navigated beneath a modal")
		}
		// An already-admitted sort/merge reconciliation may reload the same
		// source. The modal must not strand its collection/display handoff.
		if !v.showFileIfPresent(v.state.Observe().DisplayFiles()[before]) {
			t.Fatal("lost captured source")
		}
		v.display.Settle()
		if !v.deletion.Visible() || v.state.Observe().index != before {
			t.Fatal("reconciliation changed prompt ownership")
		}
	})
}

func TestCommandAdmissionTextEditing(t *testing.T) {
	t.Run("in-tree prompt releases underlying editor", func(t *testing.T) {
		v := newTestViewer(t)
		dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))
		entry := widget.NewEntry()
		entry.SetText("underlying editor")
		v.win.SetContent(container.NewStack(v.win.Content(), entry))
		v.win.Canvas().Focus(entry)
		v.requestDelete()
		if v.win.Canvas().Focused() == entry {
			t.Fatal("modal left underlying Entry on the native shortcut path")
		}
	})
	t.Run("native Copy bypasses menu interception", func(t *testing.T) {
		v := newTestViewer(t)
		if !nativeEditingAccelerator(v.menus.Actions().Copy()) {
			t.Fatal("native image Copy would intercept the editing accelerator")
		}
		if nativeEditingAccelerator(v.menus.Actions().CopyPath()) || nativeEditingAccelerator(v.menus.Save()) {
			t.Fatal("unrelated explicit accelerator was stripped")
		}
	})
	t.Run("explicit image action retains image intent", func(t *testing.T) {
		v := newTestViewer(t)
		dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))
		entry := widget.NewEntry()
		entry.SetText("editor")
		v.win.SetContent(container.NewStack(v.win.Content(), entry))
		v.win.Canvas().Focus(entry)
		var copied int
		uitest.StubClipboardCopy(t, func(data []byte) error { copied = len(data); return nil })
		v.menus.Actions().Copy().Action()
		waitForClipboard(t, v)
		if copied == 0 || entry.Text != "editor" {
			t.Fatal("explicit menu action lost its image intent")
		}
	})
	for _, modal := range []bool{false, true} {
		t.Run(map[bool]string{false: "viewer", true: "dialog"}[modal], func(t *testing.T) {
			v := newTestViewer(t)
			dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))
			v.showWindowGrid()
			v.grid.Settle()
			entry := widget.NewEntry()
			entry.SetText("distinctive editor text")
			if modal {
				popup := widget.NewModalPopUp(entry, v.win.Canvas())
				popup.Show()
				defer popup.Hide()
			} else {
				v.win.SetContent(container.NewStack(v.win.Content(), entry))
			}
			v.win.Canvas().Focus(entry)
			handler := &fyne.ShortcutHandler{}
			wireGlobalShortcuts(handler, v)
			handler.TypedShortcut(&fyne.ShortcutSelectAll{})
			handler.TypedShortcut(&fyne.ShortcutCopy{Clipboard: v.app.Clipboard()})
			if got := v.app.Clipboard().Content(); got != "distinctive editor text" {
				t.Fatalf("editing shortcuts copied %q, want selected text", got)
			}
			if v.clipboard.Begun() || v.grid.SelectionCount() != 0 {
				t.Fatal("editing changed underlying Grid or image clipboard")
			}
			test.Type(entry, "replacement")
			if entry.Text != "replacement" {
				t.Fatal("Select All did not select field contents")
			}
		})
	}
}

func TestCommandAdmissionPolicy(t *testing.T) {
	t.Run("every command has capability and ownership coverage", func(t *testing.T) {
		ready := commandContext{hasFiles: true, hasImage: true, hasPixels: true, canSave: true, canExport: true, canWallpaper: true,
			canNavigate: true, hasSession: true, canCompare: true, gridTargets: true, displayed: true, canMosaic: true,
			hasSearchTarget: true, hideDuplicates: true, variantGroupSize: 2}
		for id := commandSave; id <= commandInterval; id++ {
			c := ready
			request := commandRequest{command: id, intent: intentToggle}
			// Only these commands need a different surface; all others use ready.
			//goland:noinspection GoSwitchMissingCasesForIotaConsts
			switch id {
			case commandCopyFiles, commandSelectAll, commandViewer:
				c.surface = surfaceGrid
			case commandInterval:
				c.surface = surfacePictureFrame
			}
			decision := decideCommand(request, c)
			if !decision.allowed {
				t.Fatalf("ready command %d refused: %+v", id, decision)
			}
			c.input = inputModal
			if got := decideCommand(request, c); got.allowed || got.yieldRegion || got.refusal != refusalModal {
				t.Fatalf("modal command %d: %+v", id, got)
			}
			c = ready
			c.stopping = true
			if decideCommand(request, c).allowed {
				t.Fatalf("stopped command %d admitted", id)
			}
		}
	})
	t.Run("intent and target", func(t *testing.T) {
		for _, tc := range []struct {
			name    string
			request commandRequest
			context commandContext
			target  commandTarget
		}{
			{"editor", commandRequest{command: commandCopy, intent: intentEditing}, commandContext{editorFocused: true, input: inputModal}, targetEditor},
			{"image with focused editor", commandRequest{command: commandCopyImage}, commandContext{editorFocused: true, hasPixels: true}, targetDisplayedImage},
			{"region before Grid", commandRequest{command: commandCopy}, commandContext{regionActive: true, surface: surfaceGrid, gridTargets: true}, targetRegion},
			{"Grid files", commandRequest{command: commandCopy}, commandContext{surface: surfaceGrid, gridTargets: true, hasImage: true}, targetGridFiles},
			{"displayed image", commandRequest{command: commandCopy}, commandContext{hasPixels: true}, targetDisplayedImage},
			{"current path", commandRequest{command: commandCopyPath}, commandContext{hasFiles: true}, targetCurrentFile},
		} {
			t.Run(tc.name, func(t *testing.T) {
				got := decideCommand(tc.request, tc.context)
				if !got.allowed || got.target != tc.target {
					t.Fatalf("decision=%+v, target=%v", got, tc.target)
				}
			})
		}
	})
	for _, tc := range []struct {
		name    string
		context commandContext
		want    commandDecision
	}{
		{"unavailable preserves selection", commandContext{regionActive: true}, commandDecision{refusal: refusalUnavailable}},
		{"save yields selection", commandContext{canSave: true, regionActive: true}, commandDecision{allowed: true, yieldRegion: true, target: targetDisplayedImage}},
		{"modal owns input", commandContext{canSave: true, input: inputModal}, commandDecision{refusal: refusalModal}},
		{"menu permits action", commandContext{canSave: true, input: inputMenu}, commandDecision{allowed: true, target: targetDisplayedImage}},
		{"editor preserves explicit save intent", commandContext{canSave: true, input: inputEditor}, commandDecision{allowed: true, target: targetDisplayedImage}},
		{"comparison refuses", commandContext{canSave: true, surface: surfaceComparison}, commandDecision{refusal: refusalSurface}},
		{"visible map refuses", commandContext{canSave: true, surface: surfaceLocationMap, locationVisit: true}, commandDecision{refusal: refusalSurface}},
		{"retained map visit permits save", commandContext{canSave: true, locationVisit: true}, commandDecision{allowed: true, target: targetDisplayedImage}},
		{"busy region refuses", commandContext{canSave: true, regionActive: true, regionBusy: true}, commandDecision{refusal: refusalRegionBusy}},
		{"ordinary clipboard permits save", commandContext{canSave: true, clipboardBusy: true}, commandDecision{allowed: true, target: targetDisplayedImage}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			before := tc.context
			for range 3 {
				if got := decideCommand(commandRequest{command: commandSave}, tc.context); got != tc.want {
					t.Fatalf("decision = %+v, want %+v", got, tc.want)
				}
				if tc.context != before {
					t.Fatal("decision mutated its input")
				}
			}
		})
	}
}

func TestCommandAdmissionVisits(t *testing.T) {
	t.Run("empty-view duplicate preference keeps key versus menu intent", func(t *testing.T) {
		v := newTestViewer(t)
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyD})
		if !v.dupes.HideDuplicates() {
			t.Fatal("D no longer pre-arms the duplicate preference before loading files")
		}
		if !v.menus.Actions().Hide().Disabled {
			t.Fatal("empty-view menu should remain unavailable")
		}
		v.menus.Actions().Hide().Action()
		if !v.dupes.HideDuplicates() {
			t.Fatal("unavailable menu changed the standing preference")
		}
	})
	t.Run("cohort selection handoff", func(t *testing.T) {
		facts := commandContext{surface: surfaceExplorer, cohortVisit: true, hasFiles: true}
		if !decideCommand(commandRequest{command: commandSelectImage}, facts).allowed {
			t.Fatal("closing cohort Grid blocked its selected image handoff")
		}
		facts.input = inputModal
		if decideCommand(commandRequest{command: commandSelectImage}, facts).allowed {
			t.Fatal("cohort handoff bypassed a modal")
		}
	})
	t.Run("retained visit restrictions", func(t *testing.T) {
		for _, visit := range []string{"Explorer", "Location", "Search"} {
			for _, surface := range []commandSurface{surfaceViewer, surfaceGrid} {
				c := commandContext{surface: surface, hasFiles: true, hasImage: true, hasSearchTarget: true, canSave: true}
				c.cohortVisit, c.locationVisit, c.searchVisit = visit == "Explorer", visit == "Location", visit == "Search"
				for _, id := range []commandID{commandHideDuplicates, commandBrowseDuplicates} {
					if decideCommand(commandRequest{command: id}, c).allowed {
						t.Fatalf("%s visit admitted duplicate command %d", visit, id)
					}
				}
				if visit != "Search" && decideCommand(commandRequest{command: commandPictureFrame}, c).allowed {
					t.Fatalf("%s visit admitted Picture-frame", visit)
				}
				if visit == "Location" && decideCommand(commandRequest{command: commandSearch}, c).allowed {
					t.Fatal("hidden Location Map admitted search")
				}
				if !decideCommand(commandRequest{command: commandSave}, c).allowed {
					t.Fatal("retained visit became a visible-map block")
				}
			}
		}
	})
	t.Run("visible map and retained image are different", func(t *testing.T) {
		v := newTestViewer(t)
		source := uitest.TempGPSJPEGURI(t, "a.jpg", 24, 16, 52.52, 13.405)
		dropAndWait(t, v, source)
		v.showLocationMap()
		v.locationMap.Settle()
		if v.queryCommand(commandRequest{command: commandRotate}).allowed {
			t.Fatal("visible map admitted image rotation")
		}
		test.Tap(locationPhoto(t, v, source.Name()))
		waitUntilLoaded(t, v)
		if !v.locationMap.Active() || v.locationMap.Visible() {
			t.Fatal("premise: retained image visit missing")
		}
		v.rotateBy(1)
		if v.display.Rotation() != 1 {
			t.Fatal("retained image visit blocked rotation")
		}
		v.toggleHideDuplicates()
		v.findMoreLikeThis()
		if v.dupes.HideDuplicates() || v.searchActive() {
			t.Fatal("retained visit lost browsing restrictions")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		v.locationMap.Settle()
		if !v.locationMap.Visible() {
			t.Fatal("Escape lost the retained map")
		}
	})
}

func TestCommandAdmissionQueries(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))
	v.rotateBy(1)
	v.startRegionCopy()
	for range 3 {
		v.syncMenus()
		if !v.queryCommand(commandRequest{command: commandSave}).allowed || v.menus.Save().Disabled {
			t.Fatal("savable rotation was not offered")
		}
	}
	if !v.regionCopy.State().Active || v.fileWork.saveDone.Begun() || v.clipboard.Begun() || v.toast.card.Visible() {
		t.Fatal("availability query had side effects")
	}
	// Change capability without rebuilding the menu. The old menu callback must
	// observe the new facts and leave the selection intact.
	v.fileWork.savePending = true
	v.menus.Save().Action()
	v.fileWork.savePending = false
	if !v.regionCopy.State().Active || v.fileWork.saveDone.Begun() {
		t.Fatal("stale menu availability authorized a save")
	}
}

func TestCommandAdmissionModalOwnership(t *testing.T) {
	t.Run("prompt keyboard focus returns to viewer", func(t *testing.T) {
		for _, prompt := range []string{"delete", "export"} {
			t.Run(prompt, func(t *testing.T) {
				v := newTestViewer(t)
				dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))
				if prompt == "delete" {
					v.requestDelete()
				} else {
					v.promptExport()
				}
				c := v.win.Canvas()
				if prompt == "export" {
					test.Tap(v.exportOptions.metaCheck)
					if c.Focused() == v.exportOptions.metaCheck {
						t.Fatal("clicked metadata checkbox retained keyboard ownership")
					}
					if c.Focused() != nil {
						c.Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyUp})
						if v.exportOptions.focus != exportMetadataRow {
							t.Fatal("focused card did not route Up to its extra rows")
						}
					}
				}
				if focused, ok := c.Focused().(fyne.Tabbable); ok && focused.AcceptsTab() {
					c.Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyTab})
				} else {
					c.FocusNext()
				}
				focused := c.Focused()
				if focused == nil {
					t.Fatal("Tab did not focus a prompt control")
				}
				t.Logf("Tab focused %T", focused)
				focused.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
				if v.deletion.Visible() || v.exportPrompt.Visible() {
					t.Fatal("focused prompt control swallowed Escape")
				}
				if c.Focused() != nil {
					t.Fatalf("hidden prompt retained focus on %T", c.Focused())
				}
				c.OnTypedKey()(&fyne.KeyEvent{Name: fyne.KeyEscape})
				if v.FileCount() != 0 {
					t.Fatal("Escape did not return to the viewer after prompt dismissal")
				}
			})
		}
	})
	t.Run("favorite dialog notifications", func(t *testing.T) {
		v := newTestViewer(t)
		dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))
		v.favorites.AddCurrentList()
		if !v.menus.FileMenu().Items[0].Disabled || !v.help.Menu().Items[0].Disabled {
			t.Fatal("dialog entry did not refresh menus")
		}
		v.win.Canvas().Focused().TypedKey(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if v.win.Canvas().Overlays().Top() != nil || v.menus.FileMenu().Items[0].Disabled || v.help.Menu().Items[0].Disabled {
			t.Fatal("dialog cancel did not restore menus")
		}
	})
	t.Run("whole bar", func(t *testing.T) {
		v := newTestViewer(t)
		dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))
		v.requestDelete()
		var check func(*fyne.Menu, bool)
		check = func(menu *fyne.Menu, disabled bool) {
			for _, item := range menu.Items {
				if item.IsSeparator {
					continue
				}
				if disabled && !item.Disabled {
					t.Errorf("%s remains enabled beneath modal", item.Label)
				}
				if item.ChildMenu != nil {
					check(item.ChildMenu, disabled)
				}
			}
		}
		for _, menu := range v.win.MainMenu().Items {
			check(menu, true)
		}
		v.deletion.Cancel()
		if v.menus.FileMenu().Items[0].Disabled || v.favorites.Menu().Items[0].Disabled {
			t.Fatal("prompt exit did not restore menus")
		}
	})
	t.Run("save", func(t *testing.T) {
		v := newTestViewer(t)
		dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))
		v.rotateBy(1)
		v.requestDelete()
		if !v.menus.Save().Disabled {
			t.Error("Save remains enabled under delete confirmation")
		}
		v.menus.Save().Action()
		v.saveRotation()
		if v.fileWork.saveDone.Begun() || !v.deletion.Visible() {
			t.Fatal("Save acted under delete confirmation")
		}
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
		if v.deletion.Visible() || v.menus.Save().Disabled {
			t.Fatal("cancel did not restore availability")
		}
	})
}

func TestCommandAdmissionRoutes(t *testing.T) {
	t.Run("clipboard", func(t *testing.T) {
		for _, route := range []string{"menu", "shortcut", "bare"} {
			t.Run(route, func(t *testing.T) {
				v := openGridWith(t, "a.jpg", "b.jpg")
				uitest.StubClipboardCopy(t, func(_ []byte) error { t.Error("modal image copy reached OS"); return nil })
				uitest.StubClipboardCopyFiles(t, func(_ []string) error { t.Error("modal file copy reached OS"); return nil })
				v.requestDelete()
				before := v.grid.SelectionCount()
				switch route {
				case "menu":
					v.menus.Actions().Copy().Action()
					v.menus.Actions().CopySelection().Action()
					v.menus.Actions().CopyPath().Action()
				case "shortcut":
					handler := &fyne.ShortcutHandler{}
					wireGlobalShortcuts(handler, v)
					handler.TypedShortcut(&fyne.ShortcutCopy{Clipboard: v.app.Clipboard()})
					handler.TypedShortcut(&fyne.ShortcutSelectAll{})
					handler.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyC, Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift})
				case "bare":
					v.copySelection()
					v.copyImageToClipboard()
					v.copyPathToClipboard()
					v.copyGridSelection()
					v.startRegionCopy()
					v.selectAllInGrid()
				}
				if v.clipboard.Begun() || v.regionCopy.State().Active || v.grid.SelectionCount() != before || !v.deletion.Visible() {
					t.Fatal("clipboard route acted beneath modal")
				}
			})
		}
	})
	t.Run("navigation", func(t *testing.T) {
		for _, name := range []string{"rotate", "merge", "info", "zoom", "shuffle"} {
			t.Run(name, func(t *testing.T) {
				v := newTestViewer(t)
				dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White))
				beforeZoom, beforeInfo := v.zoom.Scale(), v.info.Visible()
				v.requestDelete()
				switch name {
				case "rotate":
					v.rotateBy(1)
				case "merge":
					v.toggleMergeMode()
				case "info":
					v.toggleInfoOverlay()
				case "zoom":
					v.zoomActionsIn()
				case "shuffle":
					v.toggleSlideshowShuffle()
				}
				if v.display.Rotation() != 0 || v.MergeMode() || v.info.Visible() != beforeInfo || v.zoom.Scale() != beforeZoom || v.SlideShuffle() {
					t.Error("presentation command acted beneath confirmation")
				}
			})
		}
	})
	t.Run("search", func(t *testing.T) {
		for _, name := range []string{"sort", "duplicates"} {
			t.Run(name, func(t *testing.T) {
				v := newTestViewer(t)
				dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White))
				before := v.SortMode()
				v.requestDelete()
				if name == "sort" {
					v.setActionsSort(filesort.ByModTime)
				} else {
					v.toggleHideDuplicates()
				}
				if v.SortMode() != before || v.dupes.HideDuplicates() {
					t.Error("sort/duplicate entry acted below confirmation")
				}
			})
		}
	})
	t.Run("maps", func(t *testing.T) {
		for _, name := range []string{"explorer", "location", "mosaic"} {
			t.Run(name, func(t *testing.T) {
				v := newTestViewer(t)
				dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White))
				configureExplorer(v, func(options *explorerui.Options) {
					options.Analyze = func(_ context.Context, _ []string, _ <-chan similarity.Control, _ func(similarity.Event)) error {
						return nil
					}
				})
				uitest.StubDisplays(t, func(_ fyne.Window) (displays.Snapshot, error) {
					return displays.Snapshot{Displays: []displays.Display{{ID: "main", Name: "Main", Bounds: image.Rect(0, 0, 80, 50)}}, Default: "main"}, nil
				})
				v.requestDelete()
				switch name {
				case "explorer":
					v.showExplorer()
				case "location":
					v.showLocationMap()
				case "mosaic":
					v.showMosaic()
				}
				if v.explorerMapActive() || v.locationMap.Active() || v.mosaicWin.Opened() {
					t.Error("feature entry acted below confirmation")
				}
			})
		}
	})
	t.Run("windows", func(t *testing.T) {
		for _, name := range []string{"grid", "frame", "exif", "about"} {
			t.Run(name, func(t *testing.T) {
				v := newTestViewer(t)
				dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White))
				before := len(v.app.Driver().AllWindows())
				popup := widget.NewModalPopUp(widget.NewLabel("owned prompt"), v.win.Canvas())
				popup.Show()
				defer popup.Hide()
				switch name {
				case "grid":
					v.showWindowGrid()
				case "frame":
					v.showWindowPictureFrame()
				case "exif":
					v.showWindowExif()
				case "about":
					v.help.ShowAbout()
				}
				if v.grid.Visible() || v.slides.Active() || len(v.app.Driver().AllWindows()) != before {
					t.Error("window opened under a modal dialog")
				}
				for _, w := range v.app.Driver().AllWindows()[before:] {
					w.Close()
				}
			})
		}
	})
	t.Run("files", func(t *testing.T) {
		for _, name := range []string{"export", "trash", "reveal", "wallpaper"} {
			for _, route := range []string{"menu", "bare"} {
				t.Run(name+"/"+route, func(t *testing.T) {
					v := newTestViewer(t)
					uitest.StubReveal(t, func(_ string) error { return nil })
					uitest.StubWallpaperSet(t, func(_ string) error { return nil })
					dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White))
					popup := widget.NewModalPopUp(widget.NewLabel("owned prompt"), v.win.Canvas())
					popup.Show()
					defer popup.Hide()
					invoke, menu := v.promptExport, v.menus.Export().Action
					switch name {
					case "trash":
						invoke, menu = v.requestDelete, v.menus.Actions().Trash().Action
					case "reveal":
						invoke, menu = v.revealCurrentFile, v.menus.Actions().Reveal().Action
					case "wallpaper":
						invoke, menu = v.setAsWallpaper, v.menus.Actions().Wallpaper().Action
					}
					if route == "menu" {
						invoke = menu
					}
					invoke()
					if v.deletion.Visible() || v.exportPrompt.Visible() || v.reveal.Begun() || v.wallpaper.Begun() {
						t.Fatal("file action acted under a modal dialog")
					}
				})
			}
		}
	})
	t.Run("open", func(t *testing.T) {
		t.Run("favorites bare entries", func(t *testing.T) {
			for _, action := range []string{"add", "manage", "open"} {
				t.Run(action, func(t *testing.T) {
					v := newTestViewer(t)
					a := uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White)
					b := uitest.TempJPEGURI(t, "b.jpg", 40, 20, color.White)
					dropAndWait(t, v, a)
					dir := t.TempDir()
					if err := favstore.Save(dir, "favorite", []fyne.URI{b}); err != nil {
						t.Fatal(err)
					}
					v.favorites.SetDir(dir)
					v.requestDelete()
					switch action {
					case "add":
						v.favorites.AddCurrentList()
					case "manage":
						v.favorites.ShowManage()
					case "open":
						v.favorites.Open(0)
					}
					if v.win.Canvas().Overlays().Top() != nil || !v.deletion.Visible() || v.state.Observe().DisplayFiles()[0] != a || v.favThumb.Begun() {
						t.Fatal("Favorite entry acted below confirmation")
					}
				})
			}
		})
		for _, route := range []string{"drop", "bare close", "menu close", "restore"} {
			t.Run(route, func(t *testing.T) {
				v := newTestViewer(t)
				a := uitest.TempJPEGURI(t, "a.jpg", 40, 20, color.White)
				b := uitest.TempJPEGURI(t, "b.jpg", 40, 20, color.White)
				dropAndWait(t, v, a)
				v.savedSession = []fyne.URI{b}
				v.requestDelete()
				switch route {
				case "drop":
					v.handleDrop([]fyne.URI{b})
				case "bare close":
					v.closeFiles()
				case "menu close":
					v.menus.CloseFiles().Action()
				case "restore":
					v.restoreSession()
				}
				if !v.deletion.Visible() || v.FileCount() != 1 || v.state.Observe().DisplayFiles()[0] != a || len(v.savedSession) != 1 {
					t.Fatal("collection command acted beneath confirmation")
				}
			})
		}
	})
	t.Run("save", func(t *testing.T) {
		t.Run("open menu is not a dialog", func(t *testing.T) {
			v := newTestViewer(t)
			dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))
			v.rotateBy(1)
			popup := widget.NewPopUpMenu(fyne.NewMenu("", fyne.NewMenuItem("Save", v.menus.Save().Action)), v.win.Canvas())
			popup.Show()
			defer popup.Hide()
			v.menus.Save().Action()
			drainFileWork(t, v)
			if v.display.Rotation() != 0 {
				t.Fatal("open application menu blocked Save")
			}
			settleToast(t, v)
		})
	})
}

func TestCommandAdmissionYieldOrdering(t *testing.T) {
	for _, route := range []string{"menu", "shortcut", "bare"} {
		t.Run("save/"+route, func(t *testing.T) {
			v := newTestViewer(t)
			dropAndWait(t, v, uitest.TempJPEGURI(t, "photo.jpg", 40, 20, color.White))
			v.startRegionCopy()
			if !v.regionCopy.State().Active {
				t.Fatal("premise: selection did not start")
			}
			invoke := func() { v.saveRotation() }
			switch route {
			case "menu":
				invoke = v.menus.Save().Action
			case "shortcut":
				handler := &fyne.ShortcutHandler{}
				wireGlobalShortcuts(handler, v)
				invoke = func() {
					handler.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierShortcutDefault})
				}
			}
			invoke()
			if !v.regionCopy.State().Active || v.fileWork.saveDone.Begun() {
				t.Fatal("unavailable Save discarded selection or started a write")
			}
			v.cancelRegionCopy()
			v.rotateBy(1)
			v.startRegionCopy()
			invoke()
			if v.regionCopy.State().Active {
				t.Fatal("available Save did not yield selection before writing")
			}
			drainFileWork(t, v)
			if v.display.Rotation() != 0 {
				t.Fatal("admitted Save did not persist the rotation")
			}
			settleToast(t, v)
		})
	}
}
