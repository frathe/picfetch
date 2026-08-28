// Actions menu (actionmenu.go): structure accelerators, checkmarks, and
// enablement. Do not open the manual here - rendering manual.md panics
// under Fyne's test theme.

package ui

import (
	"image/color"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"

	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/uitest"
)

func actionsMenu(v *viewer) *fyne.Menu {
	bar := v.win.MainMenu()
	if bar == nil || len(bar.Items) < 3 {
		return nil
	}
	return bar.Items[2] // File, Favorites, Actions
}

func actionsItem(m *fyne.Menu, label string) *fyne.MenuItem {
	if m == nil {
		return nil
	}
	for _, it := range m.Items {
		if it.IsSeparator {
			continue
		}
		if it.Label == label {
			return it
		}
	}
	return nil
}

func requireActionsItem(t *testing.T, v *viewer, label string) *fyne.MenuItem {
	t.Helper()
	item := actionsItem(actionsMenu(v), label)
	if item == nil {
		t.Fatalf("Actions menu missing item %q", label)
	}
	return item
}

func requireSortChild(t *testing.T, v *viewer, label string) *fyne.MenuItem {
	t.Helper()
	parent := requireActionsItem(t, v, "Sort order")
	if parent.ChildMenu == nil {
		t.Fatal("Sort order has no ChildMenu")
	}
	for _, it := range parent.ChildMenu.Items {
		if it.Label == label {
			return it
		}
	}
	t.Fatalf("Sort order missing child %q", label)
	return nil
}

func TestBuildMainMenu_ActionsItemsDisplayTheirAccelerators(t *testing.T) {
	v := newTestViewer(t)
	actions := buildMainMenu(v).Items[2]

	want := []struct {
		label    string
		key      fyne.KeyName
		modifier fyne.KeyModifier
	}{
		{"Sort order", fyne.KeyS, 0},
		{"Show/Hide duplicates", fyne.KeyD, 0},
		{"Show variants", fyne.KeyD, fyne.KeyModifierShift},
		{"Rotate image (CW)", fyne.KeyR, 0},
		{"Zoom in", fyne.KeyPlus, 0},
		{"Zoom out", fyne.KeyMinus, 0},
		{"Toggle merge mode", fyne.KeyM, 0},
		{"Show/Hide info overlay", fyne.KeyI, 0},
		{"Copy image", fyne.KeyC, fyne.KeyModifierShortcutDefault},
		{"Copy image path", fyne.KeyC, fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift},
		{"Set as Wallpaper", fyne.KeyE, fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift},
		{"Move image to Trash", fyne.KeyDelete, fyne.KeyModifierShift},
	}
	for _, tc := range want {
		var item *fyne.MenuItem
		for _, it := range actions.Items {
			if it.Label == tc.label {
				item = it
				break
			}
		}
		if item == nil {
			t.Fatalf("Actions menu missing item %q", tc.label)
		}
		shortcut, ok := item.Shortcut.(*desktop.CustomShortcut)
		if !ok {
			t.Fatalf("Actions menu item %q Shortcut = %#v, want a *desktop.CustomShortcut", tc.label, item.Shortcut)
		}
		if shortcut.KeyName != tc.key || shortcut.Modifier != tc.modifier {
			t.Errorf("Actions menu item %q accelerator = {%v, %v}, want {%v, %v}",
				tc.label, shortcut.KeyName, shortcut.Modifier, tc.key, tc.modifier)
		}
	}
}

func TestActionsMenu_FreshViewer(t *testing.T) {
	v := newTestViewer(t)

	name := requireSortChild(t, v, "Name")
	if !name.Checked {
		t.Error("Name should start checked")
	}
	for _, label := range []string{"Capture date", "Modified date", "File size", "Drop order"} {
		if requireSortChild(t, v, label).Checked {
			t.Errorf("%q should start unchecked", label)
		}
	}

	for _, label := range []string{
		"Show/Hide duplicates", "Show variants",
		"Rotate image (CW)", "Zoom in", "Zoom out",
		"Copy image", "Copy image path", "Set as Wallpaper", "Move image to Trash",
	} {
		if !requireActionsItem(t, v, label).Disabled {
			t.Errorf("%q should start disabled", label)
		}
	}

	merge := requireActionsItem(t, v, "Toggle merge mode")
	if merge.Disabled {
		t.Error("Toggle merge mode should start enabled")
	}
	if merge.Checked {
		t.Error("Toggle merge mode should start unchecked")
	}

	info := requireActionsItem(t, v, "Show/Hide info overlay")
	if info.Disabled {
		t.Error("Show/Hide info overlay should start enabled")
	}
	if info.Checked {
		t.Error("Show/Hide info overlay should start unchecked")
	}
}

func TestActionsMenu_SetSortModeWithNoFiles(t *testing.T) {
	v := newTestViewer(t)

	v.SetSortMode(filesort.BySize)

	if !requireSortChild(t, v, "File size").Checked {
		t.Error("File size should be checked after SetSortMode(BySize)")
	}
	if requireSortChild(t, v, "Name").Checked {
		t.Error("Name should be unchecked after SetSortMode(BySize)")
	}
}

func TestActionsMenu_AfterOneJPEGDrop(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	waitUntilLoaded(t, v)

	if requireActionsItem(t, v, "Show/Hide duplicates").Disabled {
		t.Error("Show/Hide duplicates should be enabled after a file is loaded")
	}
	if !requireActionsItem(t, v, "Show variants").Disabled {
		t.Error("Show variants should stay disabled while hide is off")
	}
	for _, label := range []string{
		"Rotate image (CW)", "Zoom in", "Zoom out",
		"Copy image", "Copy image path", "Set as Wallpaper", "Move image to Trash",
	} {
		if requireActionsItem(t, v, label).Disabled {
			t.Errorf("%q should be enabled after a file is loaded", label)
		}
	}
}

func TestActionsMenu_SWithTwoFilesChecksCaptureDate(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v,
		uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White),
		uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White),
	)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyS})
	waitForSort(t, v)

	if !requireSortChild(t, v, "Capture date").Checked {
		t.Error("Capture date should be checked after one S")
	}
	if requireSortChild(t, v, "Name").Checked {
		t.Error("Name should be unchecked after one S")
	}
}

func TestActionsMenu_HideDuplicatesEnablesVariantsOnPairNotUnique(t *testing.T) {
	v := loadPatternedTriple(t)
	v.grid.SetHideDuplicates(true)
	waitUntilLoaded(t, v)

	if !requireActionsItem(t, v, "Show/Hide duplicates").Checked {
		t.Error("Show/Hide duplicates should be checked after SetHideDuplicates(true)")
	}
	if requireActionsItem(t, v, "Show variants").Disabled {
		t.Error("Show variants should be enabled on a duplicate pair")
	}

	v.ShowImage(2)
	waitUntilLoaded(t, v)

	if !requireActionsItem(t, v, "Show variants").Disabled {
		t.Error("Show variants should be disabled on a unique file")
	}
}

func TestActionsMenu_ToggleBrowseDuplicatesChecksVariants(t *testing.T) {
	v := loadPatternedTriple(t)
	v.grid.SetHideDuplicates(true)
	waitUntilLoaded(t, v)

	v.grid.ToggleBrowseDuplicates()
	v.grid.Settle()

	variants := requireActionsItem(t, v, "Show variants")
	if !variants.Checked {
		t.Error("Show variants should be checked while browsing duplicates")
	}
	if variants.Disabled {
		t.Error("Show variants should stay enabled while browsing")
	}
}

func TestActionsMenu_PictureFrameLeavesRotateEnabled(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	t.Cleanup(func() { settleSlideshow(t, v) })

	v.togglePictureFrameMode()
	if !v.slides.Active() {
		t.Fatal("togglePictureFrameMode should enter picture-frame mode")
	}

	if !requireActionsItem(t, v, "Show variants").Disabled {
		t.Error("Show variants should be disabled in picture-frame mode")
	}
	if requireActionsItem(t, v, "Show/Hide duplicates").Disabled {
		t.Error("Show/Hide duplicates should stay enabled in picture-frame mode")
	}
	if requireActionsItem(t, v, "Rotate image (CW)").Disabled {
		t.Error("Rotate image should stay enabled in picture-frame mode")
	}
}

func TestActionsMenu_CloseFiles(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))

	v.SetSortMode(filesort.BySize)
	waitForSort(t, v)
	waitUntilLoaded(t, v)
	v.toggleMergeMode()

	v.closeFiles()

	for _, label := range []string{
		"Show/Hide duplicates", "Show variants",
		"Rotate image (CW)", "Zoom in", "Zoom out",
		"Copy image", "Copy image path", "Set as Wallpaper", "Move image to Trash",
	} {
		if !requireActionsItem(t, v, label).Disabled {
			t.Errorf("%q should be disabled after closeFiles", label)
		}
	}
	if !requireSortChild(t, v, "File size").Checked {
		t.Error("sort checkmark should be unchanged after closeFiles")
	}
	if !requireActionsItem(t, v, "Toggle merge mode").Checked {
		t.Error("merge checkmark should be unchanged after closeFiles")
	}
}

func TestActionsMenu_GridToggleDisablesRotateZoomInfo(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	warmThumbs(t, v)

	v.grid.Toggle()
	if !v.grid.Visible() {
		t.Fatal("Toggle should open the grid")
	}
	for _, label := range []string{"Rotate image (CW)", "Zoom in", "Zoom out", "Show/Hide info overlay"} {
		if !requireActionsItem(t, v, label).Disabled {
			t.Errorf("%q should be disabled while the grid is open", label)
		}
	}
	for _, label := range []string{"Copy image", "Copy image path", "Move image to Trash"} {
		if requireActionsItem(t, v, label).Disabled {
			t.Errorf("%q should stay enabled while the grid is open", label)
		}
	}

	v.grid.Toggle()
	if v.grid.Visible() {
		t.Fatal("a second Toggle should close the grid")
	}
	for _, label := range []string{"Rotate image (CW)", "Zoom in", "Zoom out", "Show/Hide info overlay"} {
		if requireActionsItem(t, v, label).Disabled {
			t.Errorf("%q should be enabled again after the grid closes", label)
		}
	}
}

func TestActionsMenu_ToggleMergeMode(t *testing.T) {
	v := newTestViewer(t)

	v.toggleMergeMode()
	if !requireActionsItem(t, v, "Toggle merge mode").Checked {
		t.Error("Toggle merge mode should be checked after turning merge on")
	}

	v.toggleMergeMode()
	if requireActionsItem(t, v, "Toggle merge mode").Checked {
		t.Error("Toggle merge mode should be unchecked after turning merge off")
	}
}

func TestActionsMenu_ToggleInfoOverlay(t *testing.T) {
	v := newTestViewer(t)

	v.toggleInfoOverlay()
	if !requireActionsItem(t, v, "Show/Hide info overlay").Checked {
		t.Error("Show/Hide info overlay should be checked after turning info on")
	}

	v.toggleInfoOverlay()
	if requireActionsItem(t, v, "Show/Hide info overlay").Checked {
		t.Error("Show/Hide info overlay should be unchecked after turning info off")
	}
}

func TestActionsMenu_SortItemJumpsWithoutCycling(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White), uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White))
	waitForSort(t, v)
	waitUntilLoaded(t, v)

	// Modes()[3] is BySize — do not cycle S three times.
	v.menus.Actions().Sort()[3].Action()
	waitForSort(t, v)
	if v.SortMode() != filesort.BySize {
		t.Fatalf("SortMode = %v, want BySize", v.SortMode())
	}
	if !v.menus.Actions().Sort()[3].Checked || v.menus.Actions().Sort()[0].Checked {
		t.Fatal("File size should be the only checked sort item")
	}
}

func TestActionsMenu_SortItemNoopWhenAlreadySelected(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White), uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White))
	waitForSort(t, v)
	waitUntilLoaded(t, v)
	if p := filesort.Label(v.SortMode()); p != "" {
		t.Fatalf("premises: default ByName must have an empty title prefix, got %q", p)
	}
	v.menus.Actions().Sort()[0].Action() // Name, the default
	if v.SortMode() != filesort.ByName {
		t.Fatal("re-choosing Name must leave ByName")
	}
	if p := filesort.Label(v.SortMode()); p != "" {
		t.Fatalf("re-choosing Name must not start a sort or change the title prefix, got %q", p)
	}
	if v.sortOp.active {
		t.Fatal("re-choosing Name must not start a sort")
	}
}

func TestActionsMenu_HideTogglesLikeD(t *testing.T) {
	v := loadPatternedTriple(t)
	v.menus.Actions().Hide().Action()
	if !v.dupes.HideDuplicates() || !v.menus.Actions().Hide().Checked {
		t.Fatal("Show/Hide duplicates should turn hide on and checkmark")
	}
	v.menus.Actions().Hide().Action()
	if v.dupes.HideDuplicates() || v.menus.Actions().Hide().Checked {
		t.Fatal("second click should turn hide off")
	}
}

func TestActionsMenu_HideNoopsWithoutFiles(t *testing.T) {
	v := newTestViewer(t)
	v.menus.Actions().Hide().Action()
	if v.dupes.HideDuplicates() {
		t.Fatal("no files: hide must stay off")
	}
}

func TestActionsMenu_ShowVariantsOpensGridOnPairAfterHide(t *testing.T) {
	v := loadPatternedTriple(t)
	v.menus.Actions().Hide().Action()
	if v.menus.Actions().ShowVariant().Disabled {
		t.Fatal("premises: hide on + pair should enable Show variants")
	}
	v.menus.Actions().ShowVariant().Action()
	v.grid.Settle()
	if !v.grid.Visible() || !v.grid.BrowsingDuplicates() {
		t.Fatal("Show variants on a duplicate (hide on) should browse and open the grid")
	}
	if !v.dupes.HideDuplicates() {
		t.Fatal("Show variants must leave hide on")
	}
	if !v.menus.Actions().ShowVariant().Checked {
		t.Fatal("Show variants should be checked while browsing")
	}
}

func TestActionsMenu_ShowVariantsNoopsWhileHideOff(t *testing.T) {
	v := loadPatternedTriple(t)
	if !v.menus.Actions().ShowVariant().Disabled {
		t.Fatal("premises: hide off must grey Show variants")
	}
	v.menus.Actions().ShowVariant().Action()
	v.grid.Settle()
	if v.grid.Visible() || v.grid.BrowsingDuplicates() {
		t.Fatal("hide off: Show variants must not start browse")
	}
}

func TestActionsMenu_ShowVariantsNoopOnUnique(t *testing.T) {
	v := loadPatternedTriple(t)
	v.grid.SetHideDuplicates(true) // rebuild groups so size==1 is known
	v.ShowImage(2)
	waitUntilLoaded(t, v)
	if !v.menus.Actions().ShowVariant().Disabled {
		t.Fatal("premises: unique should grey Show variants")
	}
	v.menus.Actions().ShowVariant().Action()
	v.grid.Settle()
	if v.grid.Visible() || v.grid.BrowsingDuplicates() {
		t.Fatal("unique: must not open browse")
	}
}

func TestActionsMenu_ShowVariantsNoopsDuringPictureFrame(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	t.Cleanup(func() { settleSlideshow(t, v) })
	v.togglePictureFrameMode()
	v.menus.Actions().ShowVariant().Action()
	if v.grid.Visible() || v.grid.BrowsingDuplicates() {
		t.Fatal("picture-frame: Show variants must no-op, like Shift+D")
	}
}

func TestActionsMenu_ShowVariantsSecondClickLeavesBrowse(t *testing.T) {
	v := loadPatternedTriple(t)
	v.menus.Actions().Hide().Action()
	v.menus.Actions().ShowVariant().Action()
	v.grid.Settle()
	v.menus.Actions().ShowVariant().Action()
	v.grid.Settle()
	if v.grid.BrowsingDuplicates() {
		t.Fatal("second click should leave browse, like Shift+D")
	}
}

func TestActionsMenu_RotateTurnsImageClockwise(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	waitUntilLoaded(t, v)
	if v.display.Rotation() != 0 {
		t.Fatal("premises: unrotated")
	}
	v.menus.Actions().Rotate().Action()
	if v.display.Rotation() != 1 {
		t.Fatalf("rotation = %d, want 1 (clockwise R)", v.display.Rotation())
	}
}

func TestActionsMenu_RotateNoopsWithoutImage(t *testing.T) {
	v := newTestViewer(t)
	v.menus.Actions().Rotate().Action()
	if v.display.Rotation() != 0 {
		t.Fatal("no image: rotate must no-op")
	}
}

func TestActionsMenu_RotateNoopsWhileGridVisible(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	waitUntilLoaded(t, v)
	v.grid.Toggle()
	v.menus.Actions().Rotate().Action()
	if v.display.Rotation() != 0 {
		t.Fatal("grid up: rotate must no-op")
	}
}

func TestActionsMenu_ZoomNoopsWhileGridVisible(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 400, 200, color.White))
	waitUntilLoaded(t, v)
	v.grid.Toggle()
	start := v.zoom.Percent()
	v.menus.Actions().ZoomIn().Action()
	v.menus.Actions().ZoomOut().Action()
	if v.zoom.Percent() != start {
		t.Fatal("grid up: zoom must no-op")
	}
}

func TestActionsMenu_InfoNoopsWhileGridVisible(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	waitUntilLoaded(t, v)
	v.grid.Toggle()
	v.menus.Actions().Info().Action()
	if v.info.Visible() {
		t.Fatal("grid up: info overlay must no-op")
	}
}

func TestActionsMenu_ZoomInThenOutChangesPercent(t *testing.T) {
	v := newTestViewer(t)
	// 400x200, not the brief's 8x8: fit of a tiny raster already exceeds
	// zoom.maxScale (16x), so In() clamps Percent down. Same size as
	// TestHandleKeyEvent_ZoomShortcuts.
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 400, 200, color.White))
	waitUntilLoaded(t, v)
	start := v.zoom.Percent()
	v.menus.Actions().ZoomIn().Action()
	if v.zoom.Percent() <= start {
		t.Fatalf("zoom in percent = %d, want > %d", v.zoom.Percent(), start)
	}
	afterIn := v.zoom.Percent()
	v.menus.Actions().ZoomOut().Action()
	if v.zoom.Percent() >= afterIn {
		t.Fatalf("zoom out percent = %d, want < %d", v.zoom.Percent(), afterIn)
	}
}

func TestActionsMenu_MergeToggleChecksItem(t *testing.T) {
	v := newTestViewer(t)
	v.menus.Actions().Merge().Action()
	if !v.MergeMode() || !v.menus.Actions().Merge().Checked {
		t.Fatal("merge should turn on and checkmark")
	}
	v.menus.Actions().Merge().Action()
	if v.MergeMode() || v.menus.Actions().Merge().Checked {
		t.Fatal("second click should turn merge off")
	}
}

func TestActionsMenu_InfoToggleChecksItem(t *testing.T) {
	v := newTestViewer(t)
	v.menus.Actions().Info().Action()
	if !v.info.Visible() || !v.menus.Actions().Info().Checked {
		t.Fatal("info overlay preference should turn on")
	}
}

func TestActionsMenu_CopyPathWritesClipboard(t *testing.T) {
	v := newTestViewer(t)
	u := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	dropAndWait(t, v, u)
	waitUntilLoaded(t, v)
	v.menus.Actions().CopyPath().Action()
	if got := v.app.Clipboard().Content(); got != u.Path() {
		t.Fatalf("clipboard = %q, want %q", got, u.Path())
	}
}

func TestActionsMenu_CopyImageUsesStub(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	waitUntilLoaded(t, v)
	called := make(chan struct{}, 1)
	uitest.StubClipboardCopy(t, func([]byte) error { called <- struct{}{}; return nil })
	v.menus.Actions().Copy().Action()
	select {
	case <-called:
	case <-time.After(testTimeout):
		t.Fatal("Copy image should encode and dispatch clipboard.CopyImage")
	}
}

func TestActionsMenu_WallpaperSetsDesktop(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	waitUntilLoaded(t, v)

	var got string
	uitest.StubWallpaperSet(t, func(p string) error {
		got = p
		return nil
	})
	v.menus.Actions().Wallpaper().Action()
	settleWallpaper(t, v)

	if got == "" {
		t.Fatal("Set as Wallpaper should dispatch wallpaper.Set")
	}
	settleToast(t, v)
}

func TestActionsMenu_WallpaperNoopsWithoutImage(t *testing.T) {
	v := newTestViewer(t)
	uitest.StubWallpaperSet(t, func(string) error {
		t.Fatal("no image: Set as Wallpaper must not call wallpaper.Set")
		return nil
	})
	v.menus.Actions().Wallpaper().Action()
	if v.wallpaper.Begun() {
		t.Fatal("no image: Set as Wallpaper must not start a wallpaper change")
	}
}

func TestActionsMenu_TrashOpensConfirmation(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))
	waitUntilLoaded(t, v)
	v.menus.Actions().Trash().Action()
	if !v.deletion.Visible() {
		t.Fatal("Move image to Trash should open the confirmation")
	}
}

func TestActionsMenu_TrashNoopsWithoutFiles(t *testing.T) {
	v := newTestViewer(t)
	v.menus.Actions().Trash().Action()
	if v.deletion.Visible() {
		t.Fatal("no files: trash must not open")
	}
}

func TestActionsMenu_HideDisabledDuringVariantsSession(t *testing.T) {
	v := loadBrowsePair(t)
	hide := requireActionsItem(t, v, "Show/Hide duplicates")
	if !hide.Disabled {
		t.Fatal("Show/Hide duplicates should be disabled while browsing variants")
	}
	v.menus.Actions().Hide().Action()
	if !v.dupes.HideDuplicates() {
		t.Fatal("Action must not toggle hide while browsing")
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyReturn})
	waitUntilLoaded(t, v)
	hide = requireActionsItem(t, v, "Show/Hide duplicates")
	if !hide.Disabled {
		t.Fatal("Show/Hide duplicates should be disabled while inspecting")
	}
	v.menus.Actions().Hide().Action()
	if !v.dupes.HideDuplicates() {
		t.Fatal("Action must not toggle hide while inspecting")
	}
	if !v.dupes.Inspecting() {
		t.Fatal("inspect must stay on")
	}
}
