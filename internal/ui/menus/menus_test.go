package menus

import (
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/filesort"
)

// newMenus builds the bar the way internal/ui will, minus the callbacks:
// nothing in this file fires an item, it only reads Checked/Disabled. No
// Fyne app is started anywhere here - that is the point of the package.
func newMenus() *Menus { return New(Callbacks{}, filesort.ByName) }

func checkDisabled(t *testing.T, name string, it *fyne.MenuItem, want bool) {
	t.Helper()
	if it.Disabled != want {
		t.Errorf("%s Disabled = %v, want %v", name, it.Disabled, want)
	}
}

func checkChecked(t *testing.T, name string, it *fyne.MenuItem, want bool) {
	t.Helper()
	if it.Checked != want {
		t.Errorf("%s Checked = %v, want %v", name, it.Checked, want)
	}
}

func TestNew_InitialDisabledMatchesTheBarAsBuilt(t *testing.T) {
	m := newMenus()

	for _, tc := range []struct {
		name string
		item *fyne.MenuItem
		want bool
	}{
		{"open", m.open, false},
		{"save", m.Save(), true},
		{"export", m.Export(), true},
		{"closeFiles", m.CloseFiles(), true},
		{"settings", m.settings, false},
		{"window.viewer", m.Window().Viewer(), true},
		{"window.exif", m.Window().Exif(), true},
		{"window.grid", m.Window().Grid(), true},
		{"window.pictureFrame", m.Window().PictureFrame(), true},
		{"window.help", m.Window().Help(), false},
		{"sortParent", m.sortParent, false},
		{"actions.hide", m.Actions().Hide(), true},
		{"actions.showVariant", m.Actions().ShowVariant(), true},
		{"actions.compare", m.Actions().Compare(), true},
		{"actions.mosaic", m.Actions().Mosaic(), true},
		{"actions.rotate", m.Actions().Rotate(), true},
		{"actions.zoomIn", m.Actions().ZoomIn(), true},
		{"actions.zoomOut", m.Actions().ZoomOut(), true},
		{"actions.merge", m.Actions().Merge(), false},
		{"actions.info", m.Actions().Info(), false},
		{"actions.copy", m.Actions().Copy(), true},
		{"actions.copySelection", m.Actions().CopySelection(), true},
		{"actions.copyPath", m.Actions().CopyPath(), true},
		{"actions.wallpaper", m.Actions().Wallpaper(), true},
		{"actions.trash", m.Actions().Trash(), true},
	} {
		checkDisabled(t, tc.name, tc.item, tc.want)
	}

	for _, it := range m.Actions().Sort() {
		checkDisabled(t, "sort item", it, false)
	}
}

func TestNew_LabelsAndAccelerators(t *testing.T) {
	m := newMenus()
	mod := fyne.KeyModifierShortcutDefault

	for _, tc := range []struct {
		name  string
		item  *fyne.MenuItem
		label string
		key   fyne.KeyName
		mods  fyne.KeyModifier
	}{
		{"open", m.open, lang.L("Open Files…"), fyne.KeyO, mod},
		{"save", m.Save(), lang.L("Save Changes"), fyne.KeyS, mod},
		{"export", m.Export(), lang.L("Export image"), fyne.KeyE, mod},
		{"window.viewer", m.Window().Viewer(), lang.L("Viewer"), fyne.KeyV, 0},
		{"window.exif", m.Window().Exif(), lang.L("EXIF Data"), fyne.KeyE, 0},
		{"window.grid", m.Window().Grid(), lang.L("Grid View"), fyne.KeyG, 0},
		{"window.pictureFrame", m.Window().PictureFrame(), lang.L("Picture-frame mode"), fyne.KeyP, 0},
		{"window.help", m.Window().Help(), lang.L("Help"), fyne.KeyF1, 0},
		{"sortParent", m.sortParent, lang.L("Sort order"), fyne.KeyS, 0},
		{"actions.hide", m.Actions().Hide(), lang.L("Show/Hide duplicates"), fyne.KeyD, 0},
		{"actions.showVariant", m.Actions().ShowVariant(), lang.L("Show variants"), fyne.KeyD, fyne.KeyModifierShift},
		{"actions.compare", m.Actions().Compare(), lang.L("Compare selected images"), fyne.KeyD, mod},
		{"actions.rotate", m.Actions().Rotate(), lang.L("Rotate image (CW)"), fyne.KeyR, 0},
		{"actions.zoomIn", m.Actions().ZoomIn(), lang.L("Zoom in"), fyne.KeyPlus, 0},
		{"actions.zoomOut", m.Actions().ZoomOut(), lang.L("Zoom out"), fyne.KeyMinus, 0},
		{"actions.merge", m.Actions().Merge(), lang.L("Toggle merge mode"), fyne.KeyM, 0},
		{"actions.info", m.Actions().Info(), lang.L("Show/Hide info overlay"), fyne.KeyI, 0},
		{"actions.copy", m.Actions().Copy(), lang.L("Copy image"), fyne.KeyC, mod},
		{"actions.copySelection", m.Actions().CopySelection(), lang.L("Copy selection"), fyne.KeyC, fyne.KeyModifierAlt | fyne.KeyModifierShift},
		{"actions.copyPath", m.Actions().CopyPath(), lang.L("Copy image path"), fyne.KeyC, mod | fyne.KeyModifierShift},
		{"actions.wallpaper", m.Actions().Wallpaper(), lang.L("Set as Wallpaper"), fyne.KeyE, mod | fyne.KeyModifierShift},
		{"actions.trash", m.Actions().Trash(), lang.L("Move image to Trash"), fyne.KeyDelete, fyne.KeyModifierShift},
	} {
		if tc.item.Label != tc.label {
			t.Errorf("%s Label = %q, want %q", tc.name, tc.item.Label, tc.label)
		}
		sc, ok := tc.item.Shortcut.(*desktop.CustomShortcut)
		if !ok {
			t.Fatalf("%s Shortcut = %T, want *desktop.CustomShortcut", tc.name, tc.item.Shortcut)
		}
		if sc.KeyName != tc.key || sc.Modifier != tc.mods {
			t.Errorf("%s shortcut = %v+%v, want %v+%v", tc.name, sc.Modifier, sc.KeyName, tc.mods, tc.key)
		}
	}

	if m.settings.Label != lang.L("Settings…") {
		t.Errorf("settings Label = %q, want %q", m.settings.Label, lang.L("Settings…"))
	}
	if m.settings.Shortcut != nil {
		t.Errorf("settings Shortcut = %v, want none", m.settings.Shortcut)
	}
	if m.CloseFiles().Label != lang.L("Close Files") {
		t.Errorf("closeFiles Label = %q, want %q", m.CloseFiles().Label, lang.L("Close Files"))
	}
	if m.CloseFiles().Shortcut != nil {
		t.Errorf("closeFiles Shortcut = %v, want none", m.CloseFiles().Shortcut)
	}
}

func TestNew_SortItemsFollowFilesortModes(t *testing.T) {
	modes := filesort.Modes()

	for _, mode := range modes {
		m := New(Callbacks{}, mode)
		items := m.Actions().Sort()
		if len(items) != len(modes) {
			t.Fatalf("sort items = %d, want %d", len(items), len(modes))
		}
		for i, it := range items {
			if it.Label != filesort.DisplayName(modes[i]) {
				t.Errorf("sort[%d] Label = %q, want %q", i, it.Label, filesort.DisplayName(modes[i]))
			}
			checkChecked(t, "sort item", it, modes[i] == mode)
		}
		if m.sortParent.ChildMenu == nil || len(m.sortParent.ChildMenu.Items) != len(modes) {
			t.Fatalf("sortParent child menu = %v, want %d items", m.sortParent.ChildMenu, len(modes))
		}
		for i, it := range m.sortParent.ChildMenu.Items {
			if it != items[i] {
				t.Errorf("sortParent child %d is not sort item %d", i, i)
			}
		}
	}
}

func TestFileMenu_Composition(t *testing.T) {
	m := newMenus()
	menu := m.FileMenu()

	if menu.Label != lang.L("File") {
		t.Errorf("File menu Label = %q, want %q", menu.Label, lang.L("File"))
	}
	want := []*fyne.MenuItem{m.open, m.Save(), m.Export(), m.CloseFiles(), nil, m.settings}
	assertItems(t, "File", menu.Items, want)
}

func TestActionsMenu_Composition(t *testing.T) {
	m := newMenus()
	menu := m.ActionsMenu()

	if menu.Label != lang.L("Actions") {
		t.Errorf("Actions menu Label = %q, want %q", menu.Label, lang.L("Actions"))
	}
	a := m.Actions()
	want := []*fyne.MenuItem{
		m.sortParent, a.Hide(), a.ShowVariant(), a.Compare(), a.Mosaic(),
		nil,
		a.Rotate(), a.ZoomIn(), a.ZoomOut(),
		nil,
		a.Merge(), a.Info(),
		nil,
		a.Copy(), a.CopySelection(), a.CopyPath(), a.Wallpaper(), a.Trash(),
	}
	assertItems(t, "Actions", menu.Items, want)
}

func TestApply_MosaicFollowsCanMosaicAndComparisonIsolation(t *testing.T) {
	fired := false
	m := New(Callbacks{Mosaic: func() { fired = true }}, filesort.ByName)
	item := m.Actions().Mosaic()
	if item.Label != lang.L("Generate Image Mosaic...") || !item.Disabled {
		t.Fatalf("initial Mosaic item = {label:%q disabled:%v}", item.Label, item.Disabled)
	}
	if !m.Apply(State{CanMosaic: true}) || item.Disabled {
		t.Fatal("CanMosaic did not enable the mosaic item")
	}
	item.Action()
	if !fired {
		t.Fatal("Mosaic item did not call its callback")
	}
	m.Apply(State{CanMosaic: true, ComparisonActive: true})
	if !item.Disabled {
		t.Fatal("comparison isolation did not disable the mosaic item")
	}
}

func TestActionsMenu_CopySelection(t *testing.T) {
	fired := false
	m := New(Callbacks{CopySelection: func() { fired = true }}, filesort.ByName)
	a := m.Actions()

	menu := m.ActionsMenu()
	copyIndex, selectionIndex, pathIndex := -1, -1, -1
	for i, item := range menu.Items {
		switch item {
		case a.Copy():
			copyIndex = i
		case a.CopySelection():
			selectionIndex = i
		case a.CopyPath():
			pathIndex = i
		}
	}
	if !(copyIndex >= 0 && copyIndex+1 == selectionIndex && selectionIndex+1 == pathIndex) {
		t.Fatalf("clipboard item indexes = copy:%d selection:%d path:%d, want consecutive in that order",
			copyIndex, selectionIndex, pathIndex)
	}

	item := a.CopySelection()
	if item.Label != lang.L("Copy selection") {
		t.Errorf("CopySelection Label = %q, want %q", item.Label, lang.L("Copy selection"))
	}
	shortcut, ok := item.Shortcut.(*desktop.CustomShortcut)
	if !ok {
		t.Fatalf("CopySelection Shortcut = %T, want *desktop.CustomShortcut", item.Shortcut)
	}
	if shortcut.KeyName != fyne.KeyC || shortcut.Modifier != fyne.KeyModifierAlt|fyne.KeyModifierShift {
		t.Errorf("CopySelection shortcut = %v+%v, want Alt+Shift+C", shortcut.Modifier, shortcut.KeyName)
	}
	if !item.Disabled || item.Checked {
		t.Errorf("initial CopySelection state = {Disabled:%v Checked:%v}, want {true false}", item.Disabled, item.Checked)
	}

	if !m.Apply(State{CanCopySelection: true}) {
		t.Fatal("Apply did not report CopySelection becoming enabled")
	}
	if item.Disabled || item.Checked {
		t.Errorf("available CopySelection state = {Disabled:%v Checked:%v}, want {false false}", item.Disabled, item.Checked)
	}
	item.Action()
	if !fired {
		t.Fatal("CopySelection item did not run its callback")
	}
}

func TestCompareEntry_MenuItem(t *testing.T) {
	fired := false
	m := New(Callbacks{Compare: func() { fired = true }}, filesort.ByName)
	item := m.Actions().Compare()

	if item.Label != lang.L("Compare selected images") {
		t.Errorf("Compare Label = %q, want %q", item.Label, lang.L("Compare selected images"))
	}
	shortcut, ok := item.Shortcut.(*desktop.CustomShortcut)
	if !ok {
		t.Fatalf("Compare Shortcut = %T, want *desktop.CustomShortcut", item.Shortcut)
	}
	if shortcut.KeyName != fyne.KeyD || shortcut.Modifier != fyne.KeyModifierShortcutDefault {
		t.Errorf("Compare shortcut = %v+%v, want ShortcutDefault+D", shortcut.Modifier, shortcut.KeyName)
	}
	if !item.Disabled || item.Checked {
		t.Errorf("initial Compare state = {Disabled:%v Checked:%v}, want {true false}", item.Disabled, item.Checked)
	}

	if !m.Apply(State{CanCompare: true}) {
		t.Fatal("Apply did not report Compare becoming enabled")
	}
	if item.Disabled || item.Checked {
		t.Errorf("available Compare state = {Disabled:%v Checked:%v}, want {false false}", item.Disabled, item.Checked)
	}
	item.Action()
	if !fired {
		t.Fatal("Compare item did not run its callback")
	}
}

func TestWindowMenu_Composition(t *testing.T) {
	m := newMenus()
	menu := m.WindowMenu()

	if menu.Label != lang.L("Window") {
		t.Errorf("Window menu Label = %q, want %q", menu.Label, lang.L("Window"))
	}
	w := m.Window()
	want := []*fyne.MenuItem{w.Viewer(), w.Exif(), w.Grid(), w.PictureFrame(), w.Help()}
	assertItems(t, "Window", menu.Items, want)
}

// assertItems compares a composed menu against the items it should hold,
// in order. A nil entry in want means "a separator belongs here".
func assertItems(t *testing.T, menu string, got, want []*fyne.MenuItem) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s menu has %d items, want %d", menu, len(got), len(want))
	}
	for i := range want {
		if want[i] == nil {
			if !got[i].IsSeparator {
				t.Errorf("%s menu item %d = %q, want a separator", menu, i, got[i].Label)
			}
			continue
		}
		if got[i] != want[i] {
			t.Errorf("%s menu item %d = %q, want %q", menu, i, got[i].Label, want[i].Label)
		}
	}
}

func TestApply_SortCheckFollowsSortMode(t *testing.T) {
	modes := filesort.Modes()

	for _, mode := range modes {
		// Start on a different mode so Apply has to move the check,
		// rather than agreeing with what New already set.
		m := New(Callbacks{}, mode.Next())
		m.Apply(State{SortMode: mode})
		for i, it := range m.Actions().Sort() {
			checkChecked(t, filesort.DisplayName(modes[i]), it, modes[i] == mode)
			checkDisabled(t, filesort.DisplayName(modes[i]), it, false)
		}
	}
}

func TestApply_SortChecksNothingForAnUnknownMode(t *testing.T) {
	for _, mode := range []filesort.Mode{-1, filesort.Mode(len(filesort.Modes())), 99} {
		m := newMenus()
		m.Apply(State{SortMode: mode})
		for i, it := range m.Actions().Sort() {
			checkChecked(t, filesort.DisplayName(filesort.Modes()[i]), it, false)
			checkDisabled(t, filesort.DisplayName(filesort.Modes()[i]), it, false)
		}
	}
}

// TestApply_SortIsNeverDisabled pins the one item in the Actions menu
// that stays available no matter what: reordering an empty set is a
// no-op, not an error.
func TestApply_SortIsNeverDisabled(t *testing.T) {
	m := newMenus()
	m.Apply(everythingOn())
	for i, it := range m.Actions().Sort() {
		checkDisabled(t, filesort.DisplayName(filesort.Modes()[i]), it, false)
	}
}

// everythingOn is the State with every bool set, used to prove the items
// that are never disabled really never are.
func everythingOn() State {
	return State{
		SortMode:           filesort.ByName,
		VariantGroupSize:   9,
		NoFiles:            true,
		GridUp:             true,
		NoImage:            true,
		SlidesActive:       true,
		ExifOpen:           true,
		ManualOpen:         true,
		Displayed:          true,
		MergeMode:          true,
		HideDuplicates:     true,
		BrowsingDuplicates: true,
		VariantsSession:    true,
		InfoVisible:        true,
		CanSave:            true,
		CanExport:          true,
		CanWallpaper:       true,
		CanCopySelection:   true,
		CanCompare:         true,
		CanMosaic:          true,
	}
}

func TestApply_Hide(t *testing.T) {
	for _, tc := range []struct {
		name            string
		state           State
		wantDisabled    bool
		wantCheckedFlag bool
	}{
		{"idle", State{}, false, false},
		{"hiding", State{HideDuplicates: true}, false, true},
		{"no files", State{NoFiles: true}, true, false},
		{"no files while hiding", State{NoFiles: true, HideDuplicates: true}, true, true},
		{"variants session", State{VariantsSession: true}, true, false},
		{"variants session while hiding", State{VariantsSession: true, HideDuplicates: true}, true, true},
		{"both", State{NoFiles: true, VariantsSession: true}, true, false},
		{"both while hiding", State{NoFiles: true, VariantsSession: true, HideDuplicates: true}, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMenus()
			m.Apply(tc.state)
			checkDisabled(t, "hide", m.Actions().Hide(), tc.wantDisabled)
			checkChecked(t, "hide", m.Actions().Hide(), tc.wantCheckedFlag)
		})
	}
}

// TestApply_ShowVariant walks the one genuinely compound rule in the
// matrix: the item is live only when there is somewhere to go (a
// duplicate group of at least two while hiding) or somewhere to come back
// from (already browsing), and never while there are no files or the
// slideshow is running.
func TestApply_ShowVariant(t *testing.T) {
	for _, tc := range []struct {
		name         string
		state        State
		wantDisabled bool
		wantChecked  bool
	}{
		{"idle", State{}, true, false},
		{"hiding, group of 2", State{HideDuplicates: true, VariantGroupSize: 2}, false, false},
		{"hiding, group of 3", State{HideDuplicates: true, VariantGroupSize: 3}, false, false},
		{"hiding, group of 1", State{HideDuplicates: true, VariantGroupSize: 1}, true, false},
		{"hiding, group of 0", State{HideDuplicates: true, VariantGroupSize: 0}, true, false},
		{"hiding, negative group", State{HideDuplicates: true, VariantGroupSize: -1}, true, false},
		{"group of 5 but not hiding", State{VariantGroupSize: 5}, true, false},
		{"browsing with no group at all", State{BrowsingDuplicates: true}, false, true},
		{"browsing while hiding", State{BrowsingDuplicates: true, HideDuplicates: true, VariantGroupSize: 2}, false, true},
		{"no files kills the group case", State{NoFiles: true, HideDuplicates: true, VariantGroupSize: 2}, true, false},
		{"no files kills the browse case", State{NoFiles: true, BrowsingDuplicates: true}, true, true},
		{"slides kill the group case", State{SlidesActive: true, HideDuplicates: true, VariantGroupSize: 2}, true, false},
		{"slides kill the browse case", State{SlidesActive: true, BrowsingDuplicates: true}, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMenus()
			m.Apply(tc.state)
			checkDisabled(t, "showVariant", m.Actions().ShowVariant(), tc.wantDisabled)
			checkChecked(t, "showVariant", m.Actions().ShowVariant(), tc.wantChecked)
		})
	}
}

func TestApply_RotateAndZoomShareOneCondition(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state State
		want  bool
	}{
		{"image showing", State{}, false},
		{"no image", State{NoImage: true}, true},
		{"grid up", State{GridUp: true}, true},
		{"no image and grid up", State{NoImage: true, GridUp: true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMenus()
			m.Apply(tc.state)
			checkDisabled(t, "rotate", m.Actions().Rotate(), tc.want)
			checkDisabled(t, "zoomIn", m.Actions().ZoomIn(), tc.want)
			checkDisabled(t, "zoomOut", m.Actions().ZoomOut(), tc.want)
		})
	}
}

func TestApply_Info(t *testing.T) {
	for _, tc := range []struct {
		name         string
		state        State
		wantDisabled bool
		wantChecked  bool
	}{
		{"hidden", State{}, false, false},
		{"visible", State{InfoVisible: true}, false, true},
		{"grid up", State{GridUp: true}, true, false},
		{"grid up while visible", State{GridUp: true, InfoVisible: true}, true, true},
		{"no image does not disable it", State{NoImage: true}, false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMenus()
			m.Apply(tc.state)
			checkDisabled(t, "info", m.Actions().Info(), tc.wantDisabled)
			checkChecked(t, "info", m.Actions().Info(), tc.wantChecked)
		})
	}
}

// TestApply_MergeIsNeverDisabled: merge mode is a preference about the
// next drop, so it stays reachable with nothing loaded.
func TestApply_MergeIsNeverDisabled(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state State
		want  bool
	}{
		{"off", State{}, false},
		{"on", State{MergeMode: true}, true},
		{"on with everything else on too", everythingOn(), true},
		{"off with no files", State{NoFiles: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMenus()
			m.Apply(tc.state)
			checkDisabled(t, "merge", m.Actions().Merge(), false)
			checkChecked(t, "merge", m.Actions().Merge(), tc.want)
		})
	}
}

func TestApply_ClipboardWallpaperAndTrash(t *testing.T) {
	for _, tc := range []struct {
		name          string
		state         State
		wantNoFiles   bool
		wantWallpaper bool
	}{
		{"files, wallpaper allowed", State{CanWallpaper: true}, false, false},
		{"files, wallpaper not allowed", State{}, false, true},
		{"no files, wallpaper allowed", State{NoFiles: true, CanWallpaper: true}, true, false},
		{"no files, wallpaper not allowed", State{NoFiles: true}, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMenus()
			m.Apply(tc.state)
			checkDisabled(t, "copy", m.Actions().Copy(), tc.wantNoFiles)
			checkDisabled(t, "copyPath", m.Actions().CopyPath(), tc.wantNoFiles)
			checkDisabled(t, "trash", m.Actions().Trash(), tc.wantNoFiles)
			checkDisabled(t, "wallpaper", m.Actions().Wallpaper(), tc.wantWallpaper)
		})
	}
}

func TestApply_FileItems(t *testing.T) {
	for _, tc := range []struct {
		name                            string
		state                           State
		wantSave, wantExport, wantClose bool
	}{
		{"nothing loaded", State{NoFiles: true}, true, true, true},
		{"loaded, nothing pending", State{}, true, true, false},
		{"pending rotation", State{CanSave: true}, false, true, false},
		{"exportable", State{CanExport: true}, true, false, false},
		{"both", State{CanSave: true, CanExport: true}, false, false, false},
		// CanSave/CanExport are computed by internal/ui, so a State that
		// claims both while claiming no files is not one the app builds -
		// the matrix still has to answer for it, item by item.
		{"contradictory", State{NoFiles: true, CanSave: true, CanExport: true}, false, false, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMenus()
			m.Apply(tc.state)
			checkDisabled(t, "save", m.Save(), tc.wantSave)
			checkDisabled(t, "export", m.Export(), tc.wantExport)
			checkDisabled(t, "closeFiles", m.CloseFiles(), tc.wantClose)
		})
	}
}

func TestApply_WindowViewer(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state State
		want  bool
	}{
		{"already in the viewer", State{}, true},
		{"grid up", State{GridUp: true}, false},
		{"slides active", State{SlidesActive: true}, false},
		{"both", State{GridUp: true, SlidesActive: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMenus()
			m.Apply(tc.state)
			checkDisabled(t, "window.viewer", m.Window().Viewer(), tc.want)
		})
	}
}

func TestApply_WindowExif(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state State
		want  bool
	}{
		{"file displayed, window closed", State{Displayed: true}, false},
		{"file displayed, window open", State{Displayed: true, ExifOpen: true}, true},
		{"nothing displayed", State{}, true},
		{"nothing displayed, window open", State{ExifOpen: true}, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMenus()
			m.Apply(tc.state)
			checkDisabled(t, "window.exif", m.Window().Exif(), tc.want)
		})
	}
}

func TestApply_WindowGrid(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state State
		want  bool
	}{
		{"files loaded, viewer showing", State{}, false},
		{"grid already up", State{GridUp: true}, true},
		{"no files", State{NoFiles: true}, true},
		{"slides active", State{SlidesActive: true}, true},
		{"all three", State{GridUp: true, NoFiles: true, SlidesActive: true}, true},
		{"variants session does not disable it", State{VariantsSession: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMenus()
			m.Apply(tc.state)
			checkDisabled(t, "window.grid", m.Window().Grid(), tc.want)
		})
	}
}

func TestApply_WindowPictureFrame(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state State
		want  bool
	}{
		{"files loaded, viewer showing", State{}, false},
		{"slides already active", State{SlidesActive: true}, true},
		{"no files", State{NoFiles: true}, true},
		{"variants session", State{VariantsSession: true}, true},
		{"all three", State{SlidesActive: true, NoFiles: true, VariantsSession: true}, true},
		{"grid up does not disable it", State{GridUp: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMenus()
			m.Apply(tc.state)
			checkDisabled(t, "window.pictureFrame", m.Window().PictureFrame(), tc.want)
		})
	}
}

func TestApply_WindowHelp(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state State
		want  bool
	}{
		{"manual closed", State{}, false},
		{"manual open", State{ManualOpen: true}, true},
		{"manual closed with nothing loaded", State{NoFiles: true}, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMenus()
			m.Apply(tc.state)
			checkDisabled(t, "window.help", m.Window().Help(), tc.want)
		})
	}
}

func TestApply_ChangedIsFalseWhenNothingMoves(t *testing.T) {
	for _, tc := range []struct {
		name  string
		state State
	}{
		{"zero state", State{}},
		{"everything on", everythingOn()},
		{"a loaded, displayed file", State{SortMode: filesort.ByModTime, Displayed: true, CanSave: true, CanExport: true, CanWallpaper: true, CanCopySelection: true, CanCompare: true}},
		{"empty drop zone", State{NoFiles: true, NoImage: true}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMenus()
			m.Apply(tc.state) // settle
			if m.Apply(tc.state) {
				t.Error("Apply reported a change for an identical re-apply")
			}
			if m.Apply(tc.state) {
				t.Error("Apply reported a change on a third identical apply")
			}
		})
	}
}

// TestApply_ChangedIsTrueForASingleFlip walks one State field at a time,
// in both directions, and demands Apply notice. A missed flip here is a
// menu that stays stale on screen because internal/ui was told there was
// nothing to redraw.
func TestApply_ChangedIsTrueForASingleFlip(t *testing.T) {
	base := State{}

	withSort := base
	withSort.SortMode = filesort.ByCaptureDate

	hiding := base
	hiding.HideDuplicates = true
	hidingWithGroup := hiding
	hidingWithGroup.VariantGroupSize = 2

	displayed := base
	displayed.Displayed = true
	displayedExifOpen := displayed
	displayedExifOpen.ExifOpen = true

	flip := func(mutate func(*State)) State {
		s := base
		mutate(&s)
		return s
	}

	for _, tc := range []struct {
		name string
		from State
		to   State
	}{
		{"SortMode", base, withSort},
		{"NoFiles", base, flip(func(s *State) { s.NoFiles = true })},
		{"GridUp", base, flip(func(s *State) { s.GridUp = true })},
		{"NoImage", base, flip(func(s *State) { s.NoImage = true })},
		{"SlidesActive", base, flip(func(s *State) { s.SlidesActive = true })},
		{"ExifOpen", displayed, displayedExifOpen},
		{"ManualOpen", base, flip(func(s *State) { s.ManualOpen = true })},
		{"Displayed", base, displayed},
		{"MergeMode", base, flip(func(s *State) { s.MergeMode = true })},
		{"HideDuplicates", base, hiding},
		{"BrowsingDuplicates", base, flip(func(s *State) { s.BrowsingDuplicates = true })},
		{"VariantsSession", base, flip(func(s *State) { s.VariantsSession = true })},
		{"InfoVisible", base, flip(func(s *State) { s.InfoVisible = true })},
		{"CanSave", base, flip(func(s *State) { s.CanSave = true })},
		{"CanExport", base, flip(func(s *State) { s.CanExport = true })},
		{"CanWallpaper", base, flip(func(s *State) { s.CanWallpaper = true })},
		{"CanCopySelection", base, flip(func(s *State) { s.CanCopySelection = true })},
		{"CanCompare", base, flip(func(s *State) { s.CanCompare = true })},
		{"CanMosaic", base, flip(func(s *State) { s.CanMosaic = true })},
		{"VariantGroupSize crossing 2", hiding, hidingWithGroup},
	} {
		t.Run(tc.name, func(t *testing.T) {
			m := newMenus()
			m.Apply(tc.from) // settle
			if !m.Apply(tc.to) {
				t.Error("Apply reported no change on the flip")
			}
			if !m.Apply(tc.from) {
				t.Error("Apply reported no change flipping back")
			}
		})
	}
}

// TestApply_ChangedIgnoresStateFieldsNoItemReads guards the other
// direction: a State field that moves without moving an item must not
// make internal/ui rebuild the native bar for nothing.
func TestApply_ChangedIgnoresStateFieldsNoItemReads(t *testing.T) {
	m := newMenus()
	from := State{HideDuplicates: true, VariantGroupSize: 3}
	m.Apply(from)

	to := from
	to.VariantGroupSize = 4 // still >= 2, so "Show variants" does not move
	if m.Apply(to) {
		t.Error("Apply reported a change for a group size that stayed above the threshold")
	}
}

// TestApply_IsIdempotent proves the matrix is a function of State alone:
// the same snapshot lands on the same items no matter what came before.
func TestApply_IsIdempotent(t *testing.T) {
	target := State{
		SortMode:         filesort.BySize,
		VariantGroupSize: 2,
		GridUp:           true,
		HideDuplicates:   true,
		Displayed:        true,
		CanExport:        true,
		CanCopySelection: true,
		CanCompare:       true,
	}

	fresh := newMenus()
	fresh.Apply(target)

	viaOtherStates := newMenus()
	viaOtherStates.Apply(everythingOn())
	viaOtherStates.Apply(State{NoFiles: true, NoImage: true})
	viaOtherStates.Apply(target)

	got, want := viaOtherStates.pairs(), fresh.pairs()
	if len(got) != len(want) {
		t.Fatalf("matrix length %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("item %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}

// TestPairs_CoversEveryStatefulItem keeps the change detector honest: if
// an item is added to the struct but not to pairs, Apply would silently
// stop reporting its moves.
func TestPairs_CoversEveryStatefulItem(t *testing.T) {
	m := newMenus()
	items := []*fyne.MenuItem{
		m.open, m.Save(), m.Export(), m.CloseFiles(), m.settings,
		m.Window().Viewer(), m.Window().Exif(), m.Window().Grid(),
		m.Window().PictureFrame(), m.Window().Help(),
		m.sortParent,
		m.Actions().Hide(), m.Actions().ShowVariant(), m.Actions().Compare(), m.Actions().Mosaic(), m.Actions().Rotate(),
		m.Actions().ZoomIn(), m.Actions().ZoomOut(), m.Actions().Merge(),
		m.Actions().Info(), m.Actions().Copy(), m.Actions().CopySelection(), m.Actions().CopyPath(),
		m.Actions().Wallpaper(), m.Actions().Trash(),
	}
	items = append(items, m.Actions().Sort()...)

	if got, want := len(m.pairs()), len(items); got != want {
		t.Fatalf("pairs covers %d items, want %d", got, want)
	}

	for _, it := range items {
		before := m.pairs()
		it.Disabled = !it.Disabled
		after := m.pairs()
		it.Disabled = !it.Disabled

		moved := false
		for i := range before {
			if before[i] != after[i] {
				moved = true
			}
		}
		if !moved {
			t.Errorf("%q is not covered by pairs", it.Label)
		}
	}
}

func TestCompareMenuState_DisablesEveryOrdinaryItemButHelp(t *testing.T) {
	m := newMenus()
	m.Apply(State{
		ComparisonActive: true,
		CanSave:          true,
		CanExport:        true,
		CanWallpaper:     true,
		CanCopySelection: true,
		CanCompare:       true,
		Displayed:        true,
	})

	for _, menu := range []*fyne.Menu{m.FileMenu(), m.ActionsMenu()} {
		for _, item := range menu.Items {
			if item.IsSeparator {
				continue
			}
			checkDisabled(t, menu.Label+" -> "+item.Label, item, true)
			if item.ChildMenu != nil {
				for _, child := range item.ChildMenu.Items {
					checkDisabled(t, item.Label+" -> "+child.Label, child, true)
				}
			}
		}
	}
	for _, item := range m.WindowMenu().Items {
		checkDisabled(t, "Window -> "+item.Label, item, item != m.Window().Help())
	}
}

// TestNew_ItemsRunTheirOwnCallback pins the wiring internal/ui depends on:
// every item runs the callback it was given, and each sort item passes its
// own mode rather than the last one in the loop.
func TestNew_ItemsRunTheirOwnCallback(t *testing.T) {
	var fired []string
	record := func(name string) func() { return func() { fired = append(fired, name) } }

	var sorted []filesort.Mode
	m := New(Callbacks{
		OpenFiles:            record("OpenFiles"),
		SaveRotation:         record("SaveRotation"),
		PromptExport:         record("PromptExport"),
		CloseFiles:           record("CloseFiles"),
		ShowSettings:         record("ShowSettings"),
		ShowViewer:           record("ShowViewer"),
		ShowExif:             record("ShowExif"),
		ShowGrid:             record("ShowGrid"),
		ShowPictureFrame:     record("ShowPictureFrame"),
		ShowHelp:             record("ShowHelp"),
		SetSort:              func(mode filesort.Mode) { sorted = append(sorted, mode) },
		ToggleHideDuplicates: record("ToggleHideDuplicates"),
		ShowVariant:          record("ShowVariant"),
		Compare:              record("Compare"),
		Mosaic:               record("Mosaic"),
		Rotate:               record("Rotate"),
		ZoomIn:               record("ZoomIn"),
		ZoomOut:              record("ZoomOut"),
		ToggleMergeMode:      record("ToggleMergeMode"),
		ToggleInfoOverlay:    record("ToggleInfoOverlay"),
		CopyImage:            record("CopyImage"),
		CopySelection:        record("CopySelection"),
		CopyPath:             record("CopyPath"),
		SetWallpaper:         record("SetWallpaper"),
		Trash:                record("Trash"),
	}, filesort.ByName)

	for _, tc := range []struct {
		item *fyne.MenuItem
		want string
	}{
		{m.open, "OpenFiles"},
		{m.Save(), "SaveRotation"},
		{m.Export(), "PromptExport"},
		{m.CloseFiles(), "CloseFiles"},
		{m.settings, "ShowSettings"},
		{m.Window().Viewer(), "ShowViewer"},
		{m.Window().Exif(), "ShowExif"},
		{m.Window().Grid(), "ShowGrid"},
		{m.Window().PictureFrame(), "ShowPictureFrame"},
		{m.Window().Help(), "ShowHelp"},
		{m.Actions().Hide(), "ToggleHideDuplicates"},
		{m.Actions().ShowVariant(), "ShowVariant"},
		{m.Actions().Compare(), "Compare"},
		{m.Actions().Mosaic(), "Mosaic"},
		{m.Actions().Rotate(), "Rotate"},
		{m.Actions().ZoomIn(), "ZoomIn"},
		{m.Actions().ZoomOut(), "ZoomOut"},
		{m.Actions().Merge(), "ToggleMergeMode"},
		{m.Actions().Info(), "ToggleInfoOverlay"},
		{m.Actions().Copy(), "CopyImage"},
		{m.Actions().CopySelection(), "CopySelection"},
		{m.Actions().CopyPath(), "CopyPath"},
		{m.Actions().Wallpaper(), "SetWallpaper"},
		{m.Actions().Trash(), "Trash"},
	} {
		fired = nil
		if tc.item.Action == nil {
			t.Errorf("%q has no action, want %s", tc.item.Label, tc.want)
			continue
		}
		tc.item.Action()
		if len(fired) != 1 || fired[0] != tc.want {
			t.Errorf("%q ran %v, want [%s]", tc.item.Label, fired, tc.want)
		}
	}

	if m.sortParent.Action != nil {
		t.Error("the Sort order parent should have no action of its own")
	}
	for i, it := range m.Actions().Sort() {
		sorted = nil
		it.Action()
		if len(sorted) != 1 || sorted[0] != filesort.Modes()[i] {
			t.Errorf("sort[%d] %q asked for %v, want [%v]", i, it.Label, sorted, filesort.Modes()[i])
		}
	}
}
