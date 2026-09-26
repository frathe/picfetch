package menus

import (
	"reflect"
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
		{"window.mosaic", m.Window().Mosaic(), true},
		{"actions.rotate", m.Actions().Rotate(), true},
		{"actions.zoomIn", m.Actions().ZoomIn(), true},
		{"actions.zoomOut", m.Actions().ZoomOut(), true},
		{"actions.merge", m.Actions().Merge(), false},
		{"actions.info", m.Actions().Info(), false},
		{"actions.copy", m.Actions().Copy(), true},
		{"actions.copySelection", m.Actions().CopySelection(), true},
		{"actions.copyPath", m.Actions().CopyPath(), true},
		{"actions.reveal", m.Actions().Reveal(), true},
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
		{"window.explorer", m.window.explorer, lang.L("Similarity Explorer"), fyne.KeyS, fyne.KeyModifierShift},
		{"sortParent", m.sortParent, lang.L("Sort order"), fyne.KeyS, 0},
		{"actions.hide", m.Actions().Hide(), lang.L("Show/Hide duplicates"), fyne.KeyD, 0},
		{"actions.showVariant", m.Actions().ShowVariant(), lang.L("Show variants"), fyne.KeyD, fyne.KeyModifierShift},
		{"actions.compare", m.Actions().Compare(), lang.L("Compare selected images"), fyne.KeyD, mod},
		{"window.mosaic", m.Window().Mosaic(), lang.L("Generate Image Mosaic..."), fyne.KeyM, fyne.KeyModifierShift},
		{"actions.rotate", m.Actions().Rotate(), lang.L("Rotate image (CW)"), fyne.KeyR, 0},
		{"actions.zoomIn", m.Actions().ZoomIn(), lang.L("Zoom in"), fyne.KeyPlus, 0},
		{"actions.zoomOut", m.Actions().ZoomOut(), lang.L("Zoom out"), fyne.KeyMinus, 0},
		{"actions.merge", m.Actions().Merge(), lang.L("Toggle merge mode"), fyne.KeyM, 0},
		{"actions.info", m.Actions().Info(), lang.L("Show/Hide info overlay"), fyne.KeyI, 0},
		{"actions.copy", m.Actions().Copy(), lang.L("Copy image"), fyne.KeyC, mod},
		{"actions.copySelection", m.Actions().CopySelection(), lang.L("Copy selection"), fyne.KeyC, fyne.KeyModifierAlt | fyne.KeyModifierShift},
		{"actions.copyPath", m.Actions().CopyPath(), lang.L("Copy image path"), fyne.KeyC, mod | fyne.KeyModifierShift},
		{"actions.reveal", m.Actions().Reveal(), lang.L("Reveal in file manager"), fyne.KeyR, mod},
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
		m.sortParent, a.Hide(), a.ShowVariant(), a.Compare(), a.findMoreLikeThis,
		nil,
		a.Rotate(), a.ZoomIn(), a.ZoomOut(),
		nil,
		a.Merge(), a.Info(),
		nil,
		a.Copy(), a.CopySelection(), a.CopyPath(), a.Reveal(), a.Wallpaper(), a.Trash(),
	}
	assertItems(t, "Actions", menu.Items, want)
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

	if !m.Apply(State{Availability: Availability{CopySelection: true}}) {
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

	if !m.Apply(State{Availability: Availability{Compare: true}}) {
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
	want := []*fyne.MenuItem{w.Viewer(), w.Exif(), w.Grid(), w.PictureFrame(), w.Help(), m.window.explorer, m.window.locationMap, w.Mosaic()}
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
		Reveal:               record("Reveal"),
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
		{m.Window().Mosaic(), "Mosaic"},
		{m.Actions().Rotate(), "Rotate"},
		{m.Actions().ZoomIn(), "ZoomIn"},
		{m.Actions().ZoomOut(), "ZoomOut"},
		{m.Actions().Merge(), "ToggleMergeMode"},
		{m.Actions().Info(), "ToggleInfoOverlay"},
		{m.Actions().Copy(), "CopyImage"},
		{m.Actions().CopySelection(), "CopySelection"},
		{m.Actions().CopyPath(), "CopyPath"},
		{m.Actions().Reveal(), "Reveal"},
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

// Each named availability is a supplied decision, independently rendered.
func TestApply_AvailabilityAndChangeDetection(t *testing.T) {
	m := newMenus()
	items := map[string]*fyne.MenuItem{
		"Open":             m.open,
		"Save":             m.save,
		"Export":           m.export,
		"CloseFiles":       m.closeFiles,
		"Settings":         m.settings,
		"Viewer":           m.window.viewer,
		"Explorer":         m.window.explorer,
		"LocationMap":      m.window.locationMap,
		"Mosaic":           m.window.mosaic,
		"Exif":             m.window.exif,
		"Grid":             m.window.grid,
		"PictureFrame":     m.window.pictureFrame,
		"Help":             m.window.help,
		"Sort":             m.sortParent,
		"HideDuplicates":   m.actions.hide,
		"BrowseDuplicates": m.actions.showVariant,
		"Compare":          m.actions.compare,
		"Search":           m.actions.findMoreLikeThis,
		"Rotate":           m.actions.rotate,
		"Zoom":             m.actions.zoomIn,
		"Merge":            m.actions.merge,
		"Info":             m.actions.info,
		"Copy":             m.actions.copy,
		"CopySelection":    m.actions.copySelection,
		"CopyPath":         m.actions.copyPath,
		"Reveal":           m.actions.reveal,
		"Wallpaper":        m.actions.wallpaper,
		"Trash":            m.actions.trash,
	}
	var state State
	m.Apply(state)
	value := reflect.ValueOf(&state.Availability).Elem()
	if len(items) != value.NumField() {
		t.Fatal("new availability needs rendering coverage")
	}
	for name, item := range items {
		t.Run(name, func(t *testing.T) {
			if !item.Disabled {
				t.Fatal("zero decision enabled an action")
			}
			value.FieldByName(name).SetBool(true)
			if !m.Apply(state) || item.Disabled {
				t.Fatal("admission was not rendered")
			}
			if m.Apply(state) {
				t.Fatal("unchanged snapshot caused refresh")
			}
			value.FieldByName(name).SetBool(false)
			if !m.Apply(state) || !item.Disabled {
				t.Fatal("refusal was not rendered")
			}
		})
	}
	// Shared presentations remain paired, without deriving their own policy.
	state.Availability.Zoom = true
	state.Availability.Sort = true
	m.Apply(state)
	if m.actions.zoomOut.Disabled {
		t.Fatal("Zoom out diverged")
	}
	for _, item := range m.actions.sort {
		if item.Disabled {
			t.Fatal("sort child diverged")
		}
	}
	state.Availability = Availability{}
	m.Apply(state)
	if !m.actions.zoomOut.Disabled {
		t.Fatal("Zoom out missed refusal")
	}
	for _, item := range m.actions.sort {
		if !item.Disabled {
			t.Fatal("sort child missed refusal")
		}
	}
}

func TestApply_PresentationIsIndependentOfAdmission(t *testing.T) {
	m := newMenus()
	for _, mode := range filesort.Modes() {
		s := State{SortMode: mode, MergeMode: true, HideDuplicates: true, BrowsingDuplicates: true, InfoVisible: true, LocationMapActive: true}
		m.Apply(s)
		for i, candidate := range filesort.Modes() {
			checkChecked(t, "sort", m.actions.sort[i], candidate == mode)
		}
		for _, item := range []*fyne.MenuItem{m.actions.merge, m.actions.hide, m.actions.showVariant, m.actions.info, m.window.locationMap} {
			if !item.Checked || !item.Disabled {
				t.Fatal("presentation and refusal are not independent")
			}
		}
		if m.Apply(s) {
			t.Fatal("identical presentation refreshed the bar")
		}
	}
	m.Apply(State{SortMode: filesort.Mode(99)})
	for _, item := range m.actions.sort {
		if item.Checked {
			t.Fatal("unknown sort checked a mode")
		}
	}
}
