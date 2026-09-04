// The File menu (menu.go): its structure and each item's wiring, plus
// closeFiles - the viewer.go action its "Close Files" item runs.

package ui

import (
	"errors"
	"image/color"
	"slices"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/theme"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/filepicker"
	"github.com/frathe/picfetch/internal/filesort"
	favoriteui "github.com/frathe/picfetch/internal/ui/favorites"
	"github.com/frathe/picfetch/internal/uitest"
)

// TestBuildMainMenu_Structure checks the bar's shape: File (Open Files…,
// Save Changes, Export image, Close Files, a separator, Settings…) followed
// by Favorites, Actions, Window, and Help - mirroring help's own TestHelpMenu
// (manual_test.go), which covers the Help submenu's own contents.
func TestBuildMainMenu_Structure(t *testing.T) {
	v := newTestViewer(t)

	menu := buildMainMenu(v)

	if len(menu.Items) != 5 {
		t.Fatalf("top-level menus = %d, want 5 (File, Favorites, Actions, Window, Help)", len(menu.Items))
	}

	file := menu.Items[0]
	if file.Label != "File" {
		t.Errorf("first menu label = %q, want %q", file.Label, "File")
	}
	if len(file.Items) != 6 {
		t.Fatalf("File menu items = %d, want 6 (Open Files…, Save Changes, Export image, Close Files, separator, Settings…)", len(file.Items))
	}

	if got := file.Items[0]; got.Label != "Open Files…" || got.Action == nil {
		t.Errorf("File menu item 0 = %+v, want %q with an action", got, "Open Files…")
	}
	if got := file.Items[1]; got.Label != "Save Changes" || got.Action == nil || !got.Disabled {
		t.Errorf("File menu item 1 = %+v, want %q with an action, starting disabled", got, "Save Changes")
	}
	if got := file.Items[2]; got.Label != "Export image" || got.Action == nil || !got.Disabled {
		t.Errorf("File menu item 2 = %+v, want %q with an action, starting disabled", got, "Export image")
	}
	if got := file.Items[3]; got.Label != "Close Files" || got.Action == nil || !got.Disabled {
		t.Errorf("File menu item 3 = %+v, want %q with an action, starting disabled", got, "Close Files")
	}
	if !file.Items[4].IsSeparator {
		t.Error("expected a separator between Close Files and Settings…")
	}
	if got := file.Items[5]; got.Label != "Settings…" || got.Action == nil {
		t.Errorf("File menu item 5 = %+v, want %q with an action", got, "Settings…")
	}

	if got := menu.Items[1]; got.Label != "Favorites" {
		t.Errorf("second menu label = %q, want %q", got.Label, "Favorites")
	}

	actions := menu.Items[2]
	if actions.Label != "Actions" {
		t.Errorf("third menu label = %q, want %q", actions.Label, "Actions")
	}
	if len(actions.Items) != 18 {
		t.Fatalf("Actions menu items = %d, want 18", len(actions.Items))
	}
	wantActionsLabels := []string{
		"Sort order", "Show/Hide duplicates", "Show variants", "Compare selected images", "Generate Image Mosaic...", "",
		"Rotate image (CW)", "Zoom in", "Zoom out", "",
		"Toggle merge mode", "Show/Hide info overlay", "",
		"Copy image", "Copy selection", "Copy image path", "Set as Wallpaper", "Move image to Trash",
	}
	for i, want := range wantActionsLabels {
		got := actions.Items[i]
		if want == "" {
			if !got.IsSeparator {
				t.Errorf("Actions menu item %d: want separator, got %q", i, got.Label)
			}
			continue
		}
		if got.Label != want {
			t.Errorf("Actions menu item %d label = %q, want %q", i, got.Label, want)
		}
		if got.IsSeparator {
			t.Errorf("Actions menu item %d (%q) is a separator, want a normal item", i, want)
		}
	}
	sortParent := actions.Items[0]
	if sortParent.Action != nil {
		t.Error("Sort order parent Action should be nil (never toggleSort)")
	}
	if sortParent.ChildMenu == nil || len(sortParent.ChildMenu.Items) != 5 {
		t.Fatalf("Sort order ChildMenu items = %d, want 5", len(sortParent.ChildMenu.Items))
	}
	modes := filesort.Modes()
	for i, m := range modes {
		child := sortParent.ChildMenu.Items[i]
		if child.Label != filesort.DisplayName(m) {
			t.Errorf("sort child %d label = %q, want %q", i, child.Label, filesort.DisplayName(m))
		}
		if child.Action == nil {
			t.Errorf("sort child %d (%q) has no action", i, child.Label)
		}
	}
	if !sortParent.ChildMenu.Items[0].Checked {
		t.Error("Name sort mode should start checked on a fresh viewer")
	}
	for i := 1; i < 5; i++ {
		if sortParent.ChildMenu.Items[i].Checked {
			t.Errorf("sort child %d (%q) should not start checked", i, sortParent.ChildMenu.Items[i].Label)
		}
	}
	for _, idx := range []int{1, 2, 3, 4, 6, 7, 8, 13, 14, 15, 16} {
		if !actions.Items[idx].Disabled {
			t.Errorf("Actions menu item %d (%q) should start disabled", idx, actions.Items[idx].Label)
		}
		if actions.Items[idx].Action == nil {
			t.Errorf("Actions menu item %d (%q) has no action", idx, actions.Items[idx].Label)
		}
	}
	for _, idx := range []int{10, 11} {
		if actions.Items[idx].Disabled {
			t.Errorf("Actions menu item %d (%q) should start enabled", idx, actions.Items[idx].Label)
		}
		if actions.Items[idx].Action == nil {
			t.Errorf("Actions menu item %d (%q) has no action", idx, actions.Items[idx].Label)
		}
	}
	if actions.Items[10].Checked {
		t.Error("Toggle merge mode should start unchecked")
	}
	if actions.Items[11].Checked {
		t.Error("Show/Hide info overlay should start unchecked")
	}

	window := menu.Items[3]
	if window.Label != "Window" {
		t.Errorf("fourth menu label = %q, want %q", window.Label, "Window")
	}
	if len(window.Items) != 5 {
		t.Fatalf("Window menu items = %d, want 5 (Viewer, EXIF Data, Grid View, Picture-frame mode, Help)", len(window.Items))
	}
	wantWindowLabels := []string{"Viewer", "EXIF Data", "Grid View", "Picture-frame mode", "Help"}
	for i, want := range wantWindowLabels {
		got := window.Items[i]
		if got.Label != want {
			t.Errorf("Window menu item %d label = %q, want %q", i, got.Label, want)
		}
		if got.Action == nil {
			t.Errorf("Window menu item %d (%q) has no action", i, want)
		}
		if got.IsSeparator {
			t.Errorf("Window menu item %d (%q) is a separator, want a normal item", i, want)
		}
	}
	if !window.Items[0].Disabled || !window.Items[1].Disabled || !window.Items[2].Disabled || !window.Items[3].Disabled {
		t.Error("Viewer, EXIF Data, Grid View, and Picture-frame mode should start disabled with no files")
	}
	if window.Items[4].Disabled {
		t.Error("Help should start enabled")
	}

	if got := menu.Items[4]; got.Label != "Help" {
		t.Errorf("fifth menu label = %q, want %q", got.Label, "Help")
	}
}

func TestBuildMainMenu_WindowItemsDisplayTheirAccelerators(t *testing.T) {
	v := newTestViewer(t)
	window := buildMainMenu(v).Items[3]

	if len(window.Items) != 5 {
		t.Fatalf("Window menu items = %d, want 5", len(window.Items))
	}

	want := []struct {
		label string
		key   fyne.KeyName
	}{
		{"Viewer", fyne.KeyV},
		{"EXIF Data", fyne.KeyE},
		{"Grid View", fyne.KeyG},
		{"Picture-frame mode", fyne.KeyP},
		{"Help", fyne.KeyF1},
	}
	for i, tc := range want {
		got := window.Items[i]
		if got.Label != tc.label {
			t.Fatalf("Window menu item %d label = %q, want %q", i, got.Label, tc.label)
		}
		shortcut, ok := got.Shortcut.(*desktop.CustomShortcut)
		if !ok {
			t.Fatalf("Window menu item %q Shortcut = %#v, want a *desktop.CustomShortcut", tc.label, got.Shortcut)
		}
		if shortcut.KeyName != tc.key || shortcut.Modifier != 0 {
			t.Errorf("Window menu item %q accelerator = %+v, want {%v, 0}", tc.label, shortcut, tc.key)
		}
	}
}

func TestBuildMainMenu_ManualOpenedObserverSyncsWindowHelp(t *testing.T) {
	fyne.SetCurrentApp(testApp)
	originalTheme := testApp.Settings().Theme()
	t.Cleanup(func() { testApp.Settings().SetTheme(originalTheme) })
	testApp.Settings().SetTheme(theme.DefaultTheme())

	v := newTestViewer(t)
	windowHelp := v.menus.Window().Help()
	if windowHelp.Disabled {
		t.Fatal("Window > Help should start enabled")
	}

	windowsBefore := make(map[fyne.Window]struct{})
	for _, window := range testApp.Driver().AllWindows() {
		windowsBefore[window] = struct{}{}
	}

	v.help.ShowManual()

	var newWindows []fyne.Window
	for _, window := range testApp.Driver().AllWindows() {
		if _, existed := windowsBefore[window]; !existed {
			newWindows = append(newWindows, window)
		}
	}
	for _, window := range newWindows {
		t.Cleanup(window.Close)
	}
	if len(newWindows) != 1 {
		t.Fatalf("new windows after ShowManual = %d, want 1", len(newWindows))
	}

	if !v.help.ManualOpen() {
		t.Error("manual should be open after ShowManual")
	}
	if !windowHelp.Disabled {
		t.Error("Window > Help should be disabled after the manual opens")
	}
}

// TestBuildMainMenu_ExportItemDisplaysItsAccelerator covers the display-only
// *desktop.CustomShortcut menu.go sets on File -> Export image - distinct
// from wireExportShortcuts (export_test.go), which is what actually binds
// Cmd/Ctrl+E; this only pins what the menu shows next to the item as a hint.
// Actions -> Set as Wallpaper's Cmd/Ctrl+Shift+E hint is covered by
// TestBuildMainMenu_ActionsItemsDisplayTheirAccelerators.
func TestBuildMainMenu_ExportItemDisplaysItsAccelerator(t *testing.T) {
	v := newTestViewer(t)
	file := buildMainMenu(v).Items[0]

	if len(file.Items) < 3 {
		t.Fatalf("File menu items = %d, want at least 3 (through Export image)", len(file.Items))
	}

	export, ok := file.Items[2].Shortcut.(*desktop.CustomShortcut)
	if !ok {
		t.Fatalf("Export image item's Shortcut = %#v, want a *desktop.CustomShortcut", file.Items[2].Shortcut)
	}
	if export.KeyName != fyne.KeyE || export.Modifier != fyne.KeyModifierShortcutDefault {
		t.Errorf("Export image accelerator = %+v, want {KeyE, KeyModifierShortcutDefault}", export)
	}
}

// TestBuildMainMenu_OpenFilesItemInvokesTheNativeChooser mirrors
// TestOpenFileDialog_RunsChooserInBackground (openfiles_test.go): the menu
// item must reach the same openFileDialog/runFileChooser path Cmd/Ctrl+O
// and the drop-zone tap already do.
func TestBuildMainMenu_OpenFilesItemInvokesTheNativeChooser(t *testing.T) {
	v := newTestViewer(t)
	menu := buildMainMenu(v)

	called := make(chan struct{})
	orig := filepicker.Choose
	t.Cleanup(func() { filepicker.Choose = orig })
	filepicker.Choose = func() ([]byte, error) {
		close(called)
		return nil, errors.New("stub: not exercising the success path here")
	}

	menu.Items[0].Items[0].Action()

	select {
	case <-called:
	case <-time.After(testTimeout):
		t.Fatal("expected the Open Files… action to invoke the native chooser")
	}

	settleChooser(t, v)
}

func TestBuildMainMenu_CloseFilesItemResetsToWelcomeState(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	dropAndWait(t, v, a)

	menu := buildMainMenu(v)
	menu.Items[0].Items[3].Action()

	if v.state.files != nil {
		t.Errorf("files = %v, want nil after the Close Files action", v.state.files)
	}
	if !v.welcomeArt.Visible() {
		t.Error("expected the welcome drop zone back after the Close Files action")
	}
}

func TestBuildMainMenu_SettingsItemOpensTheSettingsWindow(t *testing.T) {
	v := newTestViewer(t)
	menu := buildMainMenu(v)

	if v.settingsWin.Open() {
		t.Fatal("settings window should not be open yet")
	}

	menu.Items[0].Items[5].Action()

	if !v.settingsWin.Open() {
		t.Error("the Settings… action should open the settings window")
	}
}

func TestFavoritesMenuItemOpensStoredFilesThroughViewer(t *testing.T) {
	v := newTestViewer(t)
	dir := t.TempDir()
	image := uitest.TempJPEGURI(t, "favorite.jpg", 4, 4, color.White)
	if err := favstore.Save(dir, "Trip", []fyne.URI{image}); err != nil {
		t.Fatalf("favstore.Save: %v", err)
	}
	v.favorites.SetDir(dir)

	v.favorites.Menu().Items[2].Action()
	waitForScan(t, v)
	waitForSort(t, v)
	waitUntilLoaded(t, v)

	if len(v.state.files) != 1 || v.state.files[0].Path() != image.Path() {
		t.Errorf("files = %v, want favorite image %q", v.state.files, image.Path())
	}
}

func TestWireFavoriteShortcutsMapsDigitsToFavoriteSlots(t *testing.T) {
	handler := &fyne.ShortcutHandler{}
	var opened []int
	wireFavoriteShortcuts(handler, func(index int) {
		opened = append(opened, index)
	})

	for i := range favoriteui.ShortcutCount {
		handler.TypedShortcut(favoriteui.ShortcutForIndex(i))
	}

	want := []int{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
	if !slices.Equal(opened, want) {
		t.Errorf("opened slots = %v, want %v", opened, want)
	}
}

func TestFavoriteShortcutOpensStoredFilesThroughViewer(t *testing.T) {
	v := newTestViewer(t)
	dir := t.TempDir()
	image := uitest.TempJPEGURI(t, "shortcut-favorite.jpg", 4, 4, color.White)
	if err := favstore.Save(dir, "Trip", []fyne.URI{image}); err != nil {
		t.Fatalf("favstore.Save: %v", err)
	}
	v.favorites.SetDir(dir)

	handler := &fyne.ShortcutHandler{}
	wireFavoriteShortcuts(handler, v.favorites.Open)
	handler.TypedShortcut(favoriteui.ShortcutForIndex(0))
	waitForScan(t, v)
	waitForSort(t, v)
	waitUntilLoaded(t, v)

	if len(v.state.files) != 1 || v.state.files[0].Path() != image.Path() {
		t.Errorf("files = %v, want shortcut favorite %q", v.state.files, image.Path())
	}
}

func TestGlobalFavoriteShortcutOpensStoredFilesThroughViewer(t *testing.T) {
	v := newTestViewer(t)
	dir := t.TempDir()
	image := uitest.TempJPEGURI(t, "global-shortcut-favorite.jpg", 4, 4, color.White)
	if err := favstore.Save(dir, "Trip", []fyne.URI{image}); err != nil {
		t.Fatalf("favstore.Save: %v", err)
	}
	v.favorites.SetDir(dir)

	handler := &fyne.ShortcutHandler{}
	wireGlobalShortcuts(handler, v)
	handler.TypedShortcut(favoriteui.ShortcutForIndex(0))
	waitForScan(t, v)
	waitForSort(t, v)
	waitUntilLoaded(t, v)

	if len(v.state.files) != 1 || v.state.files[0].Path() != image.Path() {
		t.Errorf("files = %v, want favorite image %q", v.state.files, image.Path())
	}
}

// mainMenuRecorder is a fyne.Window that notes, every time something reads
// its main menu, which items the Favorites menu held at that moment. Reading
// the bar is what refreshMainMenu (windowmenu.go) does and the only trace it
// leaves on this platform: fyne.MainMenu.Refresh re-hands the very same
// pointer to test.window.SetMainMenu, which just stores it, and both halves
// of syncNativeMenuBar dead-end on NSApp's nil mainMenu/windowsMenu in a test
// binary with no GLFW app - so neither the window's menu identity nor its
// contents move when the bar is re-published. Recording at read time rather
// than afterwards is deliberate: it is what pins the ordering inside
// favorites' refreshMenu, which must rewrite its items before asking the
// host to publish them.
type mainMenuRecorder struct {
	fyne.Window

	favorites *fyne.Menu
	published [][]string
}

func (w *mainMenuRecorder) MainMenu() *fyne.MainMenu {
	bar := w.Window.MainMenu()
	if bar == nil {
		return nil
	}
	for _, menu := range bar.Items {
		if menu != w.favorites {
			continue
		}
		labels := make([]string, len(menu.Items))
		for i, item := range menu.Items {
			labels[i] = item.Label
		}
		w.published = append(w.published, labels)
	}
	return bar
}

// TestRefreshMenus_FavoriteChangeRepublishesTheBarThroughTheViewer covers
// viewer.RefreshMenus (windowmenu.go), the Host method internal/ui/favorites
// calls at the end of every refreshMenu - the one funnel an add, a delete and
// this SetDir rebuild all pass through. The feature is viewer-independent and
// so must not call fyne.Menu.Refresh itself: that is SetMainMenu underneath,
// which on Darwin rebuilds the native bar and undoes both the Window-menu
// merge and the unmodified-letter accelerator fixups. Stub RefreshMenus out,
// or let the Host method quietly stop being wired to refreshMainMenu, and
// adding or deleting a favorite leaves a duplicate "Window" menu and
// Command-prefixed accelerators on the bar until some unrelated syncMenus
// happens to fold it back.
func TestRefreshMenus_FavoriteChangeRepublishesTheBarThroughTheViewer(t *testing.T) {
	v, win, _ := newTestUI(t)

	dir := t.TempDir()
	image := uitest.TempJPEGURI(t, "trip.jpg", 4, 4, color.White)
	if err := favstore.Save(dir, "Trip", []fyne.URI{image}); err != nil {
		t.Fatalf("favstore.Save: %v", err)
	}

	// Swapped for the length of the rebuild only, the same per-viewer
	// override style newTestUI uses for toast.duration and frameAfter.
	recorder := &mainMenuRecorder{Window: win, favorites: v.favorites.Menu()}
	v.win = recorder
	defer func() { v.win = win }()
	v.favorites.SetDir(dir)

	if len(recorder.published) == 0 {
		t.Fatal("rebuilding the Favorites menu never reached refreshMainMenu - viewer.RefreshMenus has to re-publish the bar")
	}
	if !slices.ContainsFunc(recorder.published, func(labels []string) bool {
		return slices.Contains(labels, "Trip (1)")
	}) {
		t.Errorf("Favorites items per publish = %v, want one publish already holding %q", recorder.published, "Trip (1)")
	}
}

// --- Cmd/Ctrl+Shift+F (Manage Favorites) shortcut --------------------------

// TestWireManageFavoritesShortcut_OpensTheDialog covers wireManageFavoritesShortcut
// (shortcuts.go): F isn't one of the glfw driver's specially-cased bare
// shortcuts, so the combo reaches it as a plain desktop.CustomShortcut the
// way Cmd/Ctrl+S reaches wireSaveShortcut (see
// TestWireSaveShortcut_SavesTheCurrentRotation, save_test.go). The dialog is
// a real fyne.Window overlay (dialog.NewCustom), not one of this app's own
// stacked cards, so its presence shows up on the canvas's own overlay stack -
// the same signal TestHandleDrop_EmptyDrop (drop_test.go) checks for "no
// dialog open".
func TestWireManageFavoritesShortcut_OpensTheDialog(t *testing.T) {
	v := newTestViewer(t)

	handler := &fyne.ShortcutHandler{}
	wireManageFavoritesShortcut(handler, v)

	handler.TypedShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyF,
		Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift,
	})

	if n := len(v.win.Canvas().Overlays().List()); n != 1 {
		t.Errorf("overlay count = %d, want 1 (the Manage Favorites dialog)", n)
	}
}

// TestWireManageFavoritesShortcut_DoesNothingWhileTheDeleteCardIsUp covers
// showManageFavorites' guard: Cmd/Ctrl+Shift+F is a shortcut and arrives
// without passing handleKeyEvent, so without the guard the dialog would open
// over a delete confirmation that still believes it owns the keyboard - see
// TestPromptExport_DoesNothingWhileTheDeleteCardIsUp (export_test.go) for the
// same shape guarding the export prompt instead.
func TestWireManageFavoritesShortcut_DoesNothingWhileTheDeleteCardIsUp(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))

	v.requestDelete()
	if !v.deletion.Visible() {
		t.Fatal("the delete card should be up - the premise of this test")
	}

	v.showManageFavorites()

	if n := len(v.win.Canvas().Overlays().List()); n != 0 {
		t.Errorf("overlay count = %d, want 0 - the dialog must not open over the delete card", n)
	}
	if !v.deletion.Visible() {
		t.Error("the delete card should still be up")
	}
}

// TestWireManageFavoritesShortcut_DoesNothingWhileTheExportPromptIsUp is the
// mirror case: the export-format prompt is the other card a shortcut can
// arrive over.
func TestWireManageFavoritesShortcut_DoesNothingWhileTheExportPromptIsUp(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))

	v.promptExport()
	if !v.exportPrompt.Visible() {
		t.Fatal("the export prompt should be up - the premise of this test")
	}

	v.showManageFavorites()

	if n := len(v.win.Canvas().Overlays().List()); n != 0 {
		t.Errorf("overlay count = %d, want 0 - the dialog must not open over the export prompt", n)
	}
	if !v.exportPrompt.Visible() {
		t.Error("the export prompt should still be up")
	}
}

// --- Opt/Alt+Shift+F (Add Current List to Favorites) shortcut --------------

func TestWireAddFavoritesShortcut_OpensTheDialog(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))

	handler := &fyne.ShortcutHandler{}
	wireAddFavoritesShortcut(handler, v)

	handler.TypedShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyF,
		Modifier: fyne.KeyModifierAlt | fyne.KeyModifierShift,
	})

	if n := len(v.win.Canvas().Overlays().List()); n != 1 {
		t.Errorf("overlay count = %d, want 1 (the Add to Favorites dialog)", n)
	}
}

func TestWireAddFavoritesShortcut_DoesNothingWithoutFiles(t *testing.T) {
	v := newTestViewer(t)

	handler := &fyne.ShortcutHandler{}
	wireAddFavoritesShortcut(handler, v)

	handler.TypedShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyF,
		Modifier: fyne.KeyModifierAlt | fyne.KeyModifierShift,
	})

	if n := len(v.win.Canvas().Overlays().List()); n != 0 {
		t.Errorf("overlay count = %d, want 0 - no files, the dialog must not open", n)
	}
}

func TestWireAddFavoritesShortcut_DoesNothingWhileTheDeleteCardIsUp(t *testing.T) {
	v := newTestViewer(t)
	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))

	v.requestDelete()
	if !v.deletion.Visible() {
		t.Fatal("the delete card should be up - the premise of this test")
	}

	v.showAddFavorites()

	if n := len(v.win.Canvas().Overlays().List()); n != 0 {
		t.Errorf("overlay count = %d, want 0 - the dialog must not open over the delete card", n)
	}
	if !v.deletion.Visible() {
		t.Error("the delete card should still be up")
	}
}

// --- Close Files menu item state ------------------------------------------

// TestCloseFilesItem_DisabledWithNoFilesLoaded mirrors the other three
// image-dependent File-menu items' own "starts disabled" tests
// (save_test.go/export_test.go/wallpaper_test.go's TestCanSaveRotation_
// FalseWithNoImage and friends): there's nothing to close with an empty
// file set.
func TestCloseFilesItem_DisabledWithNoFilesLoaded(t *testing.T) {
	v := newTestViewer(t)

	if !v.menus.CloseFiles().Disabled {
		t.Error("Close Files menu item should be disabled with nothing loaded")
	}
}

func TestCloseFilesItem_EnabledAfterFilesLoaded(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	dropAndWait(t, v, a)

	if v.menus.CloseFiles().Disabled {
		t.Error("Close Files menu item should be enabled once a file is loaded")
	}
	if v.favorites.Menu().Items[0].Disabled {
		t.Error("Add Current List to Favorites should be enabled once files are loaded")
	}
}

func TestCloseFilesItem_DisabledAgainAfterCloseFiles(t *testing.T) {
	v := newTestViewer(t)
	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	dropAndWait(t, v, a)

	v.closeFiles()

	if !v.menus.CloseFiles().Disabled {
		t.Error("Close Files menu item should be disabled again once files are closed")
	}
	if !v.favorites.Menu().Items[0].Disabled {
		t.Error("Add Current List to Favorites should be disabled again once files are closed")
	}
}

// --- closeFiles ----------------------------------------------------------

func TestCloseFiles_ResetsLoadedFilesToWelcomeState(t *testing.T) {
	v := newTestViewer(t)

	a := uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White)
	dropAndWait(t, v, a)

	v.closeFiles()

	if v.state.files != nil {
		t.Errorf("files = %v, want nil after closeFiles", v.state.files)
	}
	if !v.welcomeArt.Visible() || !v.dropzone.Visible() {
		t.Error("expected the welcome drop zone back after closeFiles")
	}
}

// TestCloseFiles_NeverClosesTheWindow is the one behavior that sets
// closeFiles apart from Escape's own reset branch (handleKeyEvent): with
// nothing loaded, Escape closes the window, but File > Close Files must
// not - it is a distinct action from quitting the app.
func TestCloseFiles_NeverClosesTheWindow(t *testing.T) {
	v, _, closed := newTestUI(t)

	v.closeFiles()

	if closed() {
		t.Error("closeFiles must never close the window, unlike Escape with nothing loaded")
	}
}

// TestCloseFiles_CancelsScanInProgress mirrors
// TestCancelScan_CancelsInFlightScanWithNoFilesYet (drop_test.go): it
// drives cancelScan's target state directly rather than racing handleDrop's
// own background goroutine.
func TestCloseFiles_CancelsScanInProgress(t *testing.T) {
	v := newTestViewer(t)

	v.scanOp.lifecycle.begin()
	v.scanOp.active = true
	v.scanOp.spinner.Show()
	v.scanOp.label.Show()
	v.dropzone.Hide()
	v.welcomeArt.Hide()

	v.closeFiles()

	if v.scanOp.active {
		t.Error("closeFiles should cancel a scan in progress")
	}
	if v.scanOp.spinner.Visible() || v.scanOp.label.Visible() {
		t.Error("scan spinner/label should be hidden after closeFiles cancels a scan")
	}
	if !v.dropzone.Visible() || !v.welcomeArt.Visible() {
		t.Error("expected the welcome drop zone back after closeFiles cancels a scan")
	}

	settleToast(t, v) // cancelScan raises a "cancelled scanning" toast
}

// TestSyncMenus_KeepsFavoritesAddItemInStep guards the placement of
// SetHasFiles inside syncMenus' changed branch (menu.go). Skipping it on an
// unchanged sync is only safe because the Favorites "Add Current List" item
// can move only on a turn where Close Files moved too - both are driven by
// FileCount. If someone lifts SetHasFiles out of that branch, or Apply stops
// assigning closeFiles.Disabled from NoFiles outright, this catches the drift
// in whichever direction it happens.
func TestSyncMenus_KeepsFavoritesAddItemInStep(t *testing.T) {
	v := newTestViewer(t)

	addItem := v.favorites.Menu().Items[0]
	if !addItem.Disabled || !v.menus.CloseFiles().Disabled {
		t.Fatal("both the Favorites add item and Close Files should start disabled with no files")
	}

	dropAndWait(t, v, uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White))

	if addItem.Disabled != v.menus.CloseFiles().Disabled {
		t.Errorf("after a drop: Favorites add Disabled = %v, Close Files Disabled = %v - they must move together",
			addItem.Disabled, v.menus.CloseFiles().Disabled)
	}
	if addItem.Disabled {
		t.Error("the Favorites add item should be enabled once a file is loaded")
	}

	v.clearToDropzone()

	if addItem.Disabled != v.menus.CloseFiles().Disabled {
		t.Errorf("after clearing: Favorites add Disabled = %v, Close Files Disabled = %v - they must move together",
			addItem.Disabled, v.menus.CloseFiles().Disabled)
	}
	if !addItem.Disabled {
		t.Error("the Favorites add item should be disabled again once the files are closed")
	}
}
