package ui

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"
	fynetest "fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/openwith"
	favoriteui "github.com/frathe/picfetch/internal/ui/favorites"
	"github.com/frathe/picfetch/internal/uitest"
)

func waitForCompare(t *testing.T, v *viewer) {
	t.Helper()
	if !v.compare.Done().Begun() {
		t.Fatal("comparison load never started")
	}
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()
	if err := v.compare.Settle(ctx); err != nil {
		t.Fatal("timed out settling comparison work")
	}
	waitFor(t, "the comparison", v.compare.Done())
}

func fireCompareShortcut(v *viewer) {
	fireCompareShortcutWithModifier(v, fyne.KeyModifierShortcutDefault)
}

func fireCompareShortcutWithModifier(v *viewer, modifier fyne.KeyModifier) {
	handler := &fyne.ShortcutHandler{}
	wireGlobalShortcuts(handler, v)
	handler.TypedShortcut(&desktop.CustomShortcut{
		KeyName:  fyne.KeyD,
		Modifier: modifier,
	})
}

type shortcutCapture struct {
	shortcuts []fyne.Shortcut
}

func (c *shortcutCapture) AddShortcut(shortcut fyne.Shortcut, _ func(fyne.Shortcut)) {
	c.shortcuts = append(c.shortcuts, shortcut)
}

func TestCompareShortcut_RegistersDefaultAndPhysicalControlWithoutDuplicates(t *testing.T) {
	v, _, _ := newTestUI(t)
	capture := &shortcutCapture{}
	wireCompareShortcut(capture, v)

	want := map[fyne.KeyModifier]bool{fyne.KeyModifierShortcutDefault: true}
	want[fyne.KeyModifierControl] = true
	got := make(map[fyne.KeyModifier]int)
	for _, shortcut := range capture.shortcuts {
		custom, ok := shortcut.(*desktop.CustomShortcut)
		if !ok {
			t.Fatalf("comparison shortcut type = %T, want *desktop.CustomShortcut", shortcut)
		}
		if custom.KeyName != fyne.KeyD {
			t.Errorf("comparison shortcut key = %v, want D", custom.KeyName)
		}
		got[custom.Modifier]++
	}
	for modifier := range want {
		if got[modifier] != 1 {
			t.Errorf("comparison D shortcut registrations for modifier %v = %d, want 1", modifier, got[modifier])
		}
	}
	for modifier, count := range got {
		if !want[modifier] || count != 1 {
			t.Errorf("unexpected comparison D shortcut registration: modifier=%v count=%d", modifier, count)
		}
	}
}

func TestCompareShortcut_PhysicalControlOpensComparison(t *testing.T) {
	v := openGridWith(t, "left.jpg", "right.jpg")
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})

	fireCompareShortcutWithModifier(v, fyne.KeyModifierControl)
	waitForCompare(t, v)
	if !v.compare.Visible() {
		t.Fatal("physical Ctrl+D did not open comparison")
	}
}

func TestShutdownClosesActiveComparisonBeforeEventLoopStops(t *testing.T) {
	application := fynetest.NewApp()
	v, win := buildStartupViewer(application)
	v.grid.SetUIQueue(&uitest.UIQueue{})
	v.compare.SetUIQueue(&uitest.UIQueue{})
	t.Cleanup(win.Close)
	t.Cleanup(func() { drain(t, v) })

	uris := []fyne.URI{
		uitest.TempJPEGURI(t, "a.jpg", 4, 4, color.White),
		uitest.TempJPEGURI(t, "b.jpg", 4, 4, color.White),
		uitest.TempJPEGURI(t, "c.jpg", 4, 4, color.White),
	}
	dropAndWait(t, v, uris...)
	warmThumbs(t, v)
	v.grid.Toggle()
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	fireCompareShortcut(v)
	waitForCompare(t, v)

	lifecycle, ok := application.Lifecycle().(interface{ OnStopped() func() })
	if !ok {
		t.Skip("test app lifecycle does not expose its stopped hook")
	}
	original := lifecycle.OnStopped()
	registerShutdown(application, v)
	shutdown := lifecycle.OnStopped()
	application.Lifecycle().SetOnStopped(original)
	if shutdown == nil {
		t.Fatal("registerShutdown did not install a stopped hook")
	}

	shutdown()
	if v.compare.Visible() {
		t.Fatal("shutdown left comparison workers and surface active")
	}
}

func comparisonShaders(root fyne.CanvasObject) []*canvas.Shader {
	var shaders []*canvas.Shader
	var walk func(fyne.CanvasObject)
	walk = func(object fyne.CanvasObject) {
		if shader, ok := object.(*canvas.Shader); ok &&
			strings.HasPrefix(shader.Name, "picfetch-compare-tiled-") && shader.Visible() {
			shaders = append(shaders, shader)
		}
		switch object := object.(type) {
		case *fyne.Container:
			for _, child := range object.Objects {
				walk(child)
			}
		case *container.Clip:
			walk(object.Content)
		}
	}
	walk(root)
	return shaders
}

func comparisonButton(t *testing.T, root fyne.CanvasObject, text string) *widget.Button {
	t.Helper()
	var found *widget.Button
	var walk func(fyne.CanvasObject)
	walk = func(object fyne.CanvasObject) {
		if button, ok := object.(*widget.Button); ok && button.Text == text {
			found = button
		}
		if fyneContainer, ok := object.(*fyne.Container); ok {
			for _, child := range fyneContainer.Objects {
				walk(child)
			}
		}
	}
	walk(root)
	if found == nil {
		t.Fatalf("comparison has no %q button", text)
	}
	return found
}

func comparisonHasVisibleLabel(root fyne.CanvasObject, text string) bool {
	found := false
	var walk func(fyne.CanvasObject)
	walk = func(object fyne.CanvasObject) {
		if label, ok := object.(*widget.Label); ok && label.Visible() && label.Text == text {
			found = true
		}
		switch object := object.(type) {
		case *fyne.Container:
			for _, child := range object.Objects {
				walk(child)
			}
		case *container.Clip:
			walk(object.Content)
		}
	}
	walk(root)
	return found
}

type comparisonKeyHooks struct {
	down func(*fyne.KeyEvent)
	up   func(*fyne.KeyEvent)
}

func (h *comparisonKeyHooks) OnKeyDown() func(*fyne.KeyEvent)      { return h.down }
func (h *comparisonKeyHooks) SetOnKeyDown(fn func(*fyne.KeyEvent)) { h.down = fn }
func (h *comparisonKeyHooks) OnKeyUp() func(*fyne.KeyEvent)        { return h.up }
func (h *comparisonKeyHooks) SetOnKeyUp(fn func(*fyne.KeyEvent))   { h.up = fn }

func comparisonBackButton(t *testing.T, root fyne.CanvasObject) *widget.Button {
	t.Helper()
	return comparisonButton(t, root, lang.L("Back to Grid"))
}

func comparisonDivider(t *testing.T, root fyne.CanvasObject) (fyne.CanvasObject, fyne.Draggable) {
	t.Helper()
	var found fyne.CanvasObject
	var drag fyne.Draggable
	var walk func(fyne.CanvasObject)
	walk = func(object fyne.CanvasObject) {
		if cursorable, ok := object.(desktop.Cursorable); ok && cursorable.Cursor() == desktop.HResizeCursor {
			if draggable, ok := object.(fyne.Draggable); ok {
				found, drag = object, draggable
			}
		}
		switch object := object.(type) {
		case *fyne.Container:
			for _, child := range object.Objects {
				walk(child)
			}
		case *container.Clip:
			walk(object.Content)
		}
	}
	walk(root)
	if found == nil {
		t.Fatal("comparison has no horizontal divider drag target")
	}
	return found, drag
}

func comparisonGridWrap(t *testing.T, root fyne.CanvasObject) *widget.GridWrap {
	t.Helper()
	var found *widget.GridWrap
	var walk func(fyne.CanvasObject)
	walk = func(object fyne.CanvasObject) {
		if gridWrap, ok := object.(*widget.GridWrap); ok {
			found = gridWrap
		}
		if fyneContainer, ok := object.(*fyne.Container); ok {
			for _, child := range fyneContainer.Objects {
				walk(child)
			}
		}
	}
	walk(root)
	if found == nil {
		t.Fatal("grid overlay has no GridWrap")
	}
	return found
}

func openActiveComparisonWithExtra(t *testing.T) *viewer {
	t.Helper()
	v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	fireCompareShortcut(v)
	waitForCompare(t, v)
	return v
}

type compareCommandSnapshot struct {
	files          string
	generation     uint64
	index          int
	selection      string
	query          string
	highlight      int
	scroll         float32
	sort           string
	merge          bool
	grid           bool
	comparison     bool
	slides         bool
	info           bool
	exif           bool
	settings       bool
	deletion       bool
	export         bool
	regionCopy     bool
	rotation       int
	zoom           int
	windows        int
	chooserBegun   bool
	clipboardBegun bool
	wallpaperBegun bool
}

func snapshotCompareCommands(v *viewer) compareCommandSnapshot {
	files := make([]string, v.FileCount())
	for i := range files {
		files[i] = v.FileAt(i).String()
	}
	return compareCommandSnapshot{
		files:          strings.Join(files, "\n"),
		generation:     v.Generation(),
		index:          v.CurrentIndex(),
		selection:      fmt.Sprint(v.grid.Selection()),
		query:          v.grid.Query(),
		highlight:      v.grid.Highlight(),
		scroll:         v.grid.ScrollOffset(),
		sort:           v.SortMode().PrefValue(),
		merge:          v.MergeMode(),
		grid:           v.grid.Visible(),
		comparison:     v.compare.Visible(),
		slides:         v.slides.Active(),
		info:           v.info.Visible(),
		exif:           v.exif.Open(),
		settings:       v.settingsWin.Open(),
		deletion:       v.deletion.Visible(),
		export:         v.exportPrompt.Visible(),
		regionCopy:     v.regionCopy.State().Active,
		rotation:       v.display.Rotation(),
		zoom:           v.zoom.Percent(),
		windows:        len(v.app.Driver().AllWindows()),
		chooserBegun:   v.chooser.Begun(),
		clipboardBegun: v.clipboard.Begun(),
		wallpaperBegun: v.wallpaper.Begun(),
	}
}

func TestCompareEntry_RequiresExactlyTwoExplicitSelections(t *testing.T) {
	v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
	item := v.menus.Actions().Compare()

	assertRejected := func(wantSelection []int) {
		t.Helper()
		if item.Disabled != (len(wantSelection) != 2) {
			t.Errorf("Compare Disabled = %v with selection %v", item.Disabled, wantSelection)
		}
		fireCompareShortcut(v)
		if v.compare.Visible() {
			t.Fatalf("invalid selection %v opened comparison", wantSelection)
		}
		if !v.grid.Visible() {
			t.Fatalf("invalid selection %v closed Grid View", wantSelection)
		}
		if got, want := v.toast.text.Text, lang.L("Select exactly 2 images to compare"); got != want {
			t.Errorf("toast = %q, want %q", got, want)
		}
		settleToast(t, v)
	}

	assertRejected(nil)
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	if got, want := v.grid.Selection(), []int{0}; !slices.Equal(got, want) {
		t.Fatalf("setup selection = %v, want %v", got, want)
	}
	assertRejected([]int{0})

	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	if item.Disabled {
		t.Fatal("Compare menu item stayed disabled with exactly two explicit selections")
	}
	item.Action()
	waitForCompare(t, v)
	if !v.compare.Visible() {
		t.Fatal("Actions -> Compare selected images did not open comparison")
	}
	v.compare.Close()

	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	if got, want := v.grid.Selection(), []int{0, 1, 2}; !slices.Equal(got, want) {
		t.Fatalf("setup selection = %v, want %v", got, want)
	}
	assertRejected([]int{0, 1, 2})
}

func TestCompareRestoration_EscapeRevealsTheUnchangedFilteredGrid(t *testing.T) {
	v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	v.handleTypedRune('/')
	v.handleTypedRune('z') // hides both selected host files

	beforeSelection := v.grid.Selection()
	beforeQuery := v.grid.Query()
	beforeHighlight := v.grid.Highlight()
	beforeScroll := v.grid.ScrollOffset()
	beforeTitle := v.win.Title()
	beforeFiles := append([]fyne.URI(nil), v.state.files...)
	windows := len(v.app.Driver().AllWindows())

	fireCompareShortcut(v)
	waitForCompare(t, v)
	if !v.compare.Visible() || !v.grid.Visible() {
		t.Fatal("comparison must cover, not close, the open grid")
	}
	if got := len(v.app.Driver().AllWindows()); got != windows {
		t.Fatalf("window count = %d after comparison, want unchanged %d", got, windows)
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if v.compare.Visible() {
		t.Fatal("Escape did not close comparison")
	}
	if !v.grid.Visible() {
		t.Fatal("Escape closed Grid View along with comparison")
	}
	if got := v.grid.Selection(); !slices.Equal(got, beforeSelection) {
		t.Errorf("selection after Escape = %v, want %v", got, beforeSelection)
	}
	if v.grid.Query() != beforeQuery || v.grid.Highlight() != beforeHighlight || v.grid.ScrollOffset() != beforeScroll {
		t.Errorf("grid after Escape = {query:%q highlight:%d scroll:%v}, want {%q %d %v}",
			v.grid.Query(), v.grid.Highlight(), v.grid.ScrollOffset(), beforeQuery, beforeHighlight, beforeScroll)
	}
	if v.win.Title() != beforeTitle {
		t.Errorf("title after Escape = %q, want %q", v.win.Title(), beforeTitle)
	}
	if !slices.EqualFunc(v.state.files, beforeFiles, func(a, b fyne.URI) bool { return a.String() == b.String() }) {
		t.Errorf("file set changed across comparison")
	}
}

func TestCompareHelp_F1OpensManualWithoutLeavingComparison(t *testing.T) {
	originalTheme := testApp.Settings().Theme()
	t.Cleanup(func() { testApp.Settings().SetTheme(originalTheme) })
	testApp.Settings().SetTheme(theme.DefaultTheme())

	v := openGridWith(t, "a.jpg", "b.jpg")
	v.grid.SelectAll()
	fireCompareShortcut(v)
	waitForCompare(t, v)

	windowsBefore := make(map[fyne.Window]struct{})
	for _, window := range v.app.Driver().AllWindows() {
		windowsBefore[window] = struct{}{}
	}
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyF1})

	var manual fyne.Window
	for _, window := range v.app.Driver().AllWindows() {
		if _, existed := windowsBefore[window]; !existed {
			manual = window
			break
		}
	}
	if manual == nil || !v.help.ManualOpen() {
		t.Fatal("F1 did not open the manual while comparison was active")
	}
	t.Cleanup(manual.Close)
	if !v.compare.Visible() {
		t.Fatal("F1 closed or replaced the active comparison")
	}
}

func TestCompareAllowedCommands_ToolbarRemainsAboveInputShield(t *testing.T) {
	v := openActiveComparisonWithExtra(t)
	back := comparisonBackButton(t, v.compare.Overlay())
	position := v.app.Driver().AbsolutePositionForObject(back).
		Add(fyne.NewPos(back.Size().Width/2, back.Size().Height/2))

	fynetest.TapCanvas(v.win.Canvas(), position)

	if v.compare.Visible() {
		t.Fatal("the comparison input shield intercepted Back to Grid")
	}
	if !v.grid.Visible() {
		t.Fatal("Back to Grid did not reveal the still-open grid")
	}
}

func TestCompareMenuState_DisablesOrdinaryCommandsAndRestoresThemOnExit(t *testing.T) {
	v := openGridWith(t, "a.jpg", "b.jpg")
	v.grid.SelectAll()

	bar := v.win.MainMenu()
	before := make(map[*fyne.MenuItem]bool)
	var remember func(*fyne.Menu)
	remember = func(menu *fyne.Menu) {
		for _, item := range menu.Items {
			if item.IsSeparator {
				continue
			}
			before[item] = item.Disabled
			if item.ChildMenu != nil {
				remember(item.ChildMenu)
			}
		}
	}
	for _, menu := range bar.Items {
		remember(menu)
	}

	fireCompareShortcut(v)
	waitForCompare(t, v)

	for menuIndex, menu := range bar.Items {
		for _, item := range menu.Items {
			if item.IsSeparator {
				continue
			}
			allowedHelp := menuIndex == 3 && item == v.menus.Window().Help() || menuIndex == 4
			if item.Disabled == allowedHelp {
				t.Errorf("%s -> %s Disabled = %v, want %v during comparison",
					menu.Label, item.Label, item.Disabled, !allowedHelp)
			}
			if item.ChildMenu != nil {
				for _, child := range item.ChildMenu.Items {
					if !child.Disabled {
						t.Errorf("%s -> %s -> %s stayed enabled during comparison", menu.Label, item.Label, child.Label)
					}
				}
			}
		}
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
	for item, want := range before {
		if item.Disabled != want {
			t.Errorf("%q Disabled after comparison = %v, want restored %v", item.Label, item.Disabled, want)
		}
	}
}

func TestCompareCommandEntryPoints_MenuAndFeatureCallbacksAreIgnored(t *testing.T) {
	tests := []struct {
		name string
		act  func(*viewer)
	}{
		{"FileClose", func(v *viewer) { v.win.MainMenu().Items[0].Items[3].Action() }},
		{"FileSettings", func(v *viewer) { v.win.MainMenu().Items[0].Items[5].Action() }},
		{"Sort", func(v *viewer) { v.menus.Actions().Sort()[3].Action() }},
		{"Merge", func(v *viewer) { v.menus.Actions().Merge().Action() }},
		{"Export", func(v *viewer) { v.menus.Export().Action() }},
		{"Delete", func(v *viewer) { v.menus.Actions().Trash().Action() }},
		{"Viewer", func(v *viewer) { v.menus.Window().Viewer().Action() }},
		{"PictureFrame", func(v *viewer) { v.menus.Window().PictureFrame().Action() }},
		{"Exif", func(v *viewer) { v.showWindowExif() }},
		{"ShowImageHost", func(v *viewer) { v.ShowImage(2) }},
		{"StepImageHost", func(v *viewer) { v.StepImage(1) }},
		{"RemoveFilesHost", func(v *viewer) { v.RemoveFiles([]int{2}) }},
		{"AdvanceHost", func(v *viewer) { v.Advance() }},
		{"SelectAll", func(v *viewer) { v.selectAllInGrid() }},
		{"SettingsHost", func(v *viewer) {
			prev := v.settingsState()
			next := prev
			next.MergeMode = !prev.MergeMode
			v.ApplySettings(prev, next)
		}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := openActiveComparisonWithExtra(t)
			before := snapshotCompareCommands(v)
			windowsBefore := make(map[fyne.Window]struct{})
			for _, window := range v.app.Driver().AllWindows() {
				windowsBefore[window] = struct{}{}
			}
			tc.act(v)
			for _, window := range v.app.Driver().AllWindows() {
				if _, existed := windowsBefore[window]; !existed {
					t.Cleanup(window.Close)
				}
			}
			if got := snapshotCompareCommands(v); got != before {
				t.Errorf("command changed covered app state\n got: %+v\nwant: %+v", got, before)
			}
		})
	}

	t.Run("FavoritesHostRunner", func(t *testing.T) {
		v := openActiveComparisonWithExtra(t)
		ran := false
		v.RunCommand(func() { ran = true })
		if ran {
			t.Error("Favorites Host.RunCommand ran an action during comparison")
		}
	})

	t.Run("CopyMenu", func(t *testing.T) {
		v := openActiveComparisonWithExtra(t)
		called := false
		uitest.StubClipboardCopyFiles(t, func([]string) error {
			called = true
			return nil
		})
		before := snapshotCompareCommands(v)
		v.menus.Actions().Copy().Action()
		if v.clipboard.Begun() {
			waitForClipboard(t, v)
		}
		if called {
			t.Error("Copy menu reached the OS clipboard during comparison")
		}
		if got := snapshotCompareCommands(v); got != before {
			t.Errorf("Copy menu changed covered app state\n got: %+v\nwant: %+v", got, before)
		}
	})
}

func TestCompareCommandIsolation_ShortcutsAreIgnored(t *testing.T) {
	v := openActiveComparisonWithExtra(t)
	favoriteDir := t.TempDir()
	if err := favstore.Save(favoriteDir, "Trip", []fyne.URI{v.FileAt(2)}); err != nil {
		t.Fatalf("favstore.Save: %v", err)
	}
	v.favorites.SetDir(favoriteDir)

	var clipboardImage, clipboardFiles, wallpaper, revealed bool
	uitest.StubClipboardCopy(t, func([]byte) error { clipboardImage = true; return nil })
	uitest.StubClipboardCopyFiles(t, func([]string) error { clipboardFiles = true; return nil })
	uitest.StubWallpaperSet(t, func(string) error { wallpaper = true; return nil })
	uitest.StubReveal(t, func(string) error { revealed = true; return nil })
	uitest.StubChooser(t, nil, nil)

	compareLoads := 0
	v.compareLoad = func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		compareLoads++
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 4, 4))}}, nil
	}
	before := snapshotCompareCommands(v)
	clipboardText := v.app.Clipboard().Content()

	handler := &fyne.ShortcutHandler{}
	wireGlobalShortcuts(handler, v)
	for _, shortcut := range []fyne.Shortcut{
		favoriteui.ShortcutForIndex(0),
		&desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift},
		&desktop.CustomShortcut{KeyName: fyne.KeyF, Modifier: fyne.KeyModifierAlt | fyne.KeyModifierShift},
		&fyne.ShortcutCopy{},
		&desktop.CustomShortcut{KeyName: fyne.KeyC, Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift},
		&desktop.CustomShortcut{KeyName: fyne.KeyC, Modifier: fyne.KeyModifierAlt | fyne.KeyModifierShift},
		&desktop.CustomShortcut{KeyName: fyne.KeyD, Modifier: fyne.KeyModifierShortcutDefault},
		&fyne.ShortcutCut{Secondary: true},
		&fyne.ShortcutSelectAll{},
		&desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierShortcutDefault},
		&desktop.CustomShortcut{KeyName: fyne.KeyE, Modifier: fyne.KeyModifierShortcutDefault},
		&desktop.CustomShortcut{KeyName: fyne.KeyE, Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift},
		&desktop.CustomShortcut{KeyName: fyne.KeyR, Modifier: fyne.KeyModifierShortcutDefault},
	} {
		handler.TypedShortcut(shortcut)
	}

	if v.clipboard.Begun() {
		waitForClipboard(t, v)
	}
	if v.wallpaper.Begun() {
		waitFor(t, "the wallpaper command", &v.wallpaper)
	}
	if v.reveal.Begun() {
		waitForReveal(t, v)
	}
	if got := snapshotCompareCommands(v); got != before {
		t.Errorf("shortcut changed covered app state\n got: %+v\nwant: %+v", got, before)
	}
	if compareLoads != 0 {
		t.Errorf("Compare shortcut restarted %d source loads", compareLoads)
	}
	if clipboardImage || clipboardFiles || wallpaper || revealed || v.app.Clipboard().Content() != clipboardText {
		t.Errorf("shortcut reached an OS integration: image=%v files=%v wallpaper=%v reveal=%v textChanged=%v",
			clipboardImage, clipboardFiles, wallpaper, revealed, v.app.Clipboard().Content() != clipboardText)
	}
	if v.toast.card.Visible() {
		t.Errorf("ordinary shortcut raised unexpected toast %q", v.toast.text.Text)
		settleToast(t, v)
	}
}

func TestCompareInputIsolation_TypedKeysAndRunesCannotReachCoveredGrid(t *testing.T) {
	v := openActiveComparisonWithExtra(t)
	before := snapshotCompareCommands(v)

	v.handleTypedRune('/')
	v.handleTypedRune('c')
	for _, key := range []fyne.KeyName{
		fyne.KeySpace, fyne.KeyReturn, fyne.KeyRight, fyne.KeyHome,
		fyne.KeyS, fyne.KeyM, fyne.KeyP, fyne.KeyR,
	} {
		v.handleKeyEvent(&fyne.KeyEvent{Name: key})
	}

	if got := snapshotCompareCommands(v); got != before {
		t.Errorf("typed input changed the covered grid/viewer\n got: %+v\nwant: %+v", got, before)
	}
}

func TestCompareZoom_KeyboardRoutesToComparisonWithoutChangingCoveredState(t *testing.T) {
	v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
	v.compareLoad = func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "a.jpg" {
			return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 800, 400))}}, nil
		}
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 200, 800))}}, nil
	}
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	fireCompareShortcut(v)
	waitForCompare(t, v)

	images := comparisonShaders(v.compare.Overlay())
	if len(images) != 2 {
		t.Fatalf("comparison images = %d, want 2", len(images))
	}
	beforeSizes := [2]fyne.Size{images[0].Size(), images[1].Size()}
	beforeState := snapshotCompareCommands(v)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})

	images = comparisonShaders(v.compare.Overlay())
	for i, img := range images {
		if img.Size().Width <= beforeSizes[i].Width || img.Size().Height <= beforeSizes[i].Height {
			t.Errorf("comparison image %d after + = %v, want larger than %v", i, img.Size(), beforeSizes[i])
		}
	}
	if got := snapshotCompareCommands(v); got != beforeState {
		t.Errorf("comparison zoom key changed covered app state\n got: %+v\nwant: %+v", got, beforeState)
	}
}

func TestCompareLinkControl_CtrlLAndButtonShareReadyGate(t *testing.T) {
	v := openGridWith(t, "left.jpg", "right.jpg")
	v.grid.SelectAll()

	started := make(chan struct{}, 2)
	release := make(chan struct{})
	releaseLoads := func() {
		select {
		case <-release:
		default:
			close(release)
		}
	}
	t.Cleanup(releaseLoads)
	v.compareLoad = func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		started <- struct{}{}
		<-release
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 20, 10))}}, nil
	}

	fireCompareShortcut(v)
	for range 2 {
		select {
		case <-started:
		case <-time.After(testTimeout):
			t.Fatal("timed out waiting for both comparison loads to start")
		}
	}

	overlay := v.compare.Overlay()
	unlink := comparisonButton(t, overlay, lang.L("Unlink"))
	if !unlink.Disabled() {
		t.Fatal("Unlink is enabled before both comparison images are ready")
	}

	var modifiers fyne.KeyModifier
	v.keyModifiers = func() fyne.KeyModifier { return modifiers }
	hooks := &comparisonKeyHooks{}
	wireComparisonLinkToggleHook(hooks, v)
	modifiers = fyne.KeyModifierControl
	hooks.down(&fyne.KeyEvent{Name: fyne.KeyL})
	modifiers = 0

	if got := comparisonButton(t, overlay, lang.L("Unlink")); got != unlink {
		t.Error("pre-ready Ctrl+L replaced the disabled Unlink control")
	}
	if !unlink.Disabled() {
		t.Error("pre-ready Ctrl+L enabled the Unlink control")
	}
	if comparisonHasVisibleLabel(overlay, lang.L("Unlinked")) {
		t.Error("pre-ready Ctrl+L showed the Unlinked status")
	}

	releaseLoads()
	waitForCompare(t, v)
	if unlink.Disabled() {
		t.Fatal("Unlink stayed disabled after both comparison images became ready")
	}

	modifiers = fyne.KeyModifierControl
	hooks.down(&fyne.KeyEvent{Name: fyne.KeyL})
	modifiers = 0
	link := comparisonButton(t, overlay, lang.L("Link"))
	if link != unlink {
		t.Error("Ctrl+L replaced the comparison link control")
	}
	if !comparisonHasVisibleLabel(overlay, lang.L("Unlinked")) {
		t.Error("ready Ctrl+L did not show the Unlinked status")
	}

	fynetest.Tap(link)
	if got := comparisonButton(t, overlay, lang.L("Unlink")); got != unlink {
		t.Error("Link tap replaced the comparison link control")
	}
	if comparisonHasVisibleLabel(overlay, lang.L("Unlinked")) {
		t.Error("Link tap left the Unlinked status visible")
	}
}

func TestCompareLinkToggle_CanvasOverlayOwnsPhysicalCtrlL(t *testing.T) {
	v := openActiveComparisonWithExtra(t)
	v.keyModifiers = func() fyne.KeyModifier { return fyne.KeyModifierControl }
	hooks := &comparisonKeyHooks{}
	wireComparisonLinkToggleHook(hooks, v)

	modal := canvas.NewRectangle(color.Transparent)
	v.win.Canvas().Overlays().Add(modal)
	hooks.down(&fyne.KeyEvent{Name: fyne.KeyL})
	if comparisonHasVisibleLabel(v.compare.Overlay(), lang.L("Unlinked")) {
		t.Fatal("physical Ctrl+L unlinked comparison behind a canvas overlay")
	}

	v.win.Canvas().Overlays().Remove(modal)
	hooks.down(&fyne.KeyEvent{Name: fyne.KeyL})
	if !comparisonHasVisibleLabel(v.compare.Overlay(), lang.L("Unlinked")) {
		t.Error("physical Ctrl+L stayed blocked after the canvas overlay was removed")
	}
}

func TestCompareLinkToggle_ZoomsOnlyTheLastHoveredPaneWithoutHeldModifier(t *testing.T) {
	v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
	v.compareLoad = func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 800, 400))}}, nil
	}
	var modifiers fyne.KeyModifier
	v.keyModifiers = func() fyne.KeyModifier { return modifiers }
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	fireCompareShortcut(v)
	waitForCompare(t, v)

	images := comparisonShaders(v.compare.Overlay())
	if len(images) != 2 {
		t.Fatalf("comparison images = %d, want 2", len(images))
	}
	before := [2]fyne.Size{images[0].Size(), images[1].Size()}
	overlayPosition := v.app.Driver().AbsolutePositionForObject(v.compare.Overlay())
	leftCenter := overlayPosition.Add(fyne.NewPos(v.compare.Overlay().Size().Width/4, v.compare.Overlay().Size().Height/2))
	fynetest.MoveMouse(v.win.Canvas(), leftCenter)

	hooks := &comparisonKeyHooks{}
	wireComparisonLinkToggleHook(hooks, v)
	modifiers = fyne.KeyModifierControl
	hooks.down(&fyne.KeyEvent{Name: fyne.KeyL})
	modifiers = 0
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})

	images = comparisonShaders(v.compare.Overlay())
	if images[0].Size().Width <= before[0].Width {
		t.Errorf("left unlinked comparison image after + = %v, want larger than %v", images[0].Size(), before[0])
	}
	if images[1].Size() != before[1] {
		t.Errorf("right comparison image after left-targeted + = %v, want unchanged %v", images[1].Size(), before[1])
	}
}

func TestCompareLinkToggle_ChainsHookPersistsAndRelinksOnSecondPress(t *testing.T) {
	v := openActiveComparisonWithExtra(t)
	var modifiers fyne.KeyModifier
	v.keyModifiers = func() fyne.KeyModifier { return modifiers }

	downCalls, upCalls := 0, 0
	hooks := &comparisonKeyHooks{
		down: func(*fyne.KeyEvent) { downCalls++ },
		up:   func(*fyne.KeyEvent) { upCalls++ },
	}
	wireComparisonLinkToggleHook(hooks, v)

	modifiers = fyne.KeyModifierControl
	hooks.down(&fyne.KeyEvent{Name: fyne.KeyL})
	if !comparisonHasVisibleLabel(v.compare.Overlay(), lang.L("Unlinked")) {
		t.Fatal("first Ctrl+L press did not show the comparison unlink status")
	}
	for range 3 {
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyL})
	}
	if !comparisonHasVisibleLabel(v.compare.Overlay(), lang.L("Unlinked")) {
		t.Error("repeated typed L events retriggered the physical Ctrl+L toggle")
	}

	modifiers = 0
	hooks.up(&fyne.KeyEvent{Name: desktop.KeyControlLeft})
	if !comparisonHasVisibleLabel(v.compare.Overlay(), lang.L("Unlinked")) {
		t.Error("releasing Control relinked panes toggled apart by Ctrl+L")
	}

	modifiers = fyne.KeyModifierControl
	hooks.down(&fyne.KeyEvent{Name: fyne.KeyL})
	if comparisonHasVisibleLabel(v.compare.Overlay(), lang.L("Unlinked")) {
		t.Error("second Ctrl+L press did not relink the comparison panes")
	}
	if downCalls != 2 || upCalls != 1 {
		t.Errorf("existing key-hook calls = down %d up %d, want 2 and 1", downCalls, upCalls)
	}
}

func TestCompareLinkToggle_ControlAloneDoesNotChangeMode(t *testing.T) {
	v := openActiveComparisonWithExtra(t)
	var modifiers fyne.KeyModifier
	v.keyModifiers = func() fyne.KeyModifier { return modifiers }

	hooks := &comparisonKeyHooks{}
	wireComparisonLinkToggleHook(hooks, v)

	modifiers = fyne.KeyModifierControl
	hooks.down(&fyne.KeyEvent{Name: desktop.KeyControlLeft})
	if comparisonHasVisibleLabel(v.compare.Overlay(), lang.L("Unlinked")) {
		t.Error("pressing Control by itself unlinked the comparison panes")
	}
}

func TestCompareLinkToggle_RequiresExactPhysicalControlL(t *testing.T) {
	tests := []struct {
		name      string
		modifiers fyne.KeyModifier
	}{
		{name: "no modifier"},
		{name: "Shift", modifiers: fyne.KeyModifierShift},
		{name: "Command", modifiers: fyne.KeyModifierSuper},
		{name: "Control Shift", modifiers: fyne.KeyModifierControl | fyne.KeyModifierShift},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := openActiveComparisonWithExtra(t)
			v.keyModifiers = func() fyne.KeyModifier { return tc.modifiers }
			hooks := &comparisonKeyHooks{}
			wireComparisonLinkToggleHook(hooks, v)

			hooks.down(&fyne.KeyEvent{Name: fyne.KeyL})
			if comparisonHasVisibleLabel(v.compare.Overlay(), lang.L("Unlinked")) {
				t.Errorf("L with modifiers %v toggled comparison linking", tc.modifiers)
			}
		})
	}
}

func TestComparePanInputs_CanvasDragAndShiftWheelStayInComparison(t *testing.T) {
	v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
	v.compareLoad = func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "a.jpg" {
			return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 800, 400))}}, nil
		}
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 200, 800))}}, nil
	}
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	fireCompareShortcut(v)
	waitForCompare(t, v)
	for range 7 {
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})
	}

	images := comparisonShaders(v.compare.Overlay())
	beforeState := snapshotCompareCommands(v)
	leftBefore := images[0].Position()
	canvasSize := v.win.Canvas().Size()
	leftCenter := fyne.NewPos(canvasSize.Width/4, canvasSize.Height/2)
	fynetest.Drag(v.win.Canvas(), leftCenter, 40, 20)
	images = comparisonShaders(v.compare.Overlay())
	if images[0].Position() == leftBefore {
		t.Fatal("canvas drag did not pan the active comparison view")
	}

	rightBefore := images[1].Position()
	v.keyModifiers = func() fyne.KeyModifier { return fyne.KeyModifierShift }
	rightCenter := fyne.NewPos(canvasSize.Width*3/4, canvasSize.Height/2)
	fynetest.Scroll(v.win.Canvas(), rightCenter, -15, 25)
	v.keyModifiers = func() fyne.KeyModifier { return 0 }
	images = comparisonShaders(v.compare.Overlay())
	if images[1].Position() == rightBefore {
		t.Fatal("canvas Shift+wheel did not pan the active comparison view")
	}
	if got := snapshotCompareCommands(v); got != beforeState {
		t.Errorf("comparison pointer input changed covered app state\n got: %+v\nwant: %+v", got, beforeState)
	}
}

func TestCompareSwipePointer_CanvasRoutesDividerAndPaneDrag(t *testing.T) {
	v := openGridWith(t, "a.jpg", "b.jpg", "c.jpg")
	v.compareLoad = func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 1600, 800))}}, nil
	}
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	fireCompareShortcut(v)
	waitForCompare(t, v)
	fynetest.Tap(comparisonButton(t, v.compare.Overlay(), lang.L("Swipe")))
	for range 5 {
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})
	}

	images := comparisonShaders(v.compare.Overlay())
	if len(images) != 2 {
		t.Fatalf("comparison images = %d, want 2", len(images))
	}
	beforePositions := [2]fyne.Position{images[0].Position(), images[1].Position()}
	beforeState := snapshotCompareCommands(v)
	divider, _ := comparisonDivider(t, v.compare.Overlay())
	beforeDivider := divider.Position().X + divider.Size().Width/2
	dividerPosition := v.app.Driver().AbsolutePositionForObject(divider)
	dividerPoint := dividerPosition.Add(fyne.NewPos(divider.Size().Width/2, divider.Size().Height/2))
	fynetest.Drag(v.win.Canvas(), dividerPoint, 80, 0)

	afterDivider := divider.Position().X + divider.Size().Width/2
	if !uitest.ApproxEqual(afterDivider, beforeDivider+80) {
		t.Errorf("canvas divider drag moved center to %.2f, want %.2f", afterDivider, beforeDivider+80)
	}
	images = comparisonShaders(v.compare.Overlay())
	for i, img := range images {
		if img.Position() != beforePositions[i] {
			t.Errorf("comparison image %d moved during canvas divider drag: %v, want %v", i, img.Position(), beforePositions[i])
		}
	}
	afterDividerPositions := [2]fyne.Position{images[0].Position(), images[1].Position()}

	overlayPosition := v.app.Driver().AbsolutePositionForObject(v.compare.Overlay())
	panePoint := overlayPosition.Add(fyne.NewPos(v.compare.Overlay().Size().Width/4, v.compare.Overlay().Size().Height/2))
	fynetest.Drag(v.win.Canvas(), panePoint, 40, 20)
	images = comparisonShaders(v.compare.Overlay())
	if images[0].Position() == afterDividerPositions[0] {
		t.Fatal("canvas drag away from the divider did not pan the comparison images")
	}
	if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, afterDivider) {
		t.Errorf("pane drag moved divider center to %.2f, want unchanged %.2f", center, afterDivider)
	}
	if got := snapshotCompareCommands(v); got != beforeState {
		t.Errorf("swipe pointer input changed covered app state\n got: %+v\nwant: %+v", got, beforeState)
	}
}

func TestCompareDividerKeys_ViewerRoutesShiftWithoutChangingCoveredState(t *testing.T) {
	v := openActiveComparisonWithExtra(t)
	fynetest.Tap(comparisonButton(t, v.compare.Overlay(), lang.L("Swipe")))
	divider, _ := comparisonDivider(t, v.compare.Overlay())
	width := v.compare.Overlay().Size().Width
	before := snapshotCompareCommands(v)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, width*0.55) {
		t.Fatalf("viewer Right routed divider center to %.2f, want %.2f", center, width*0.55)
	}
	v.keyModifiers = func() fyne.KeyModifier { return fyne.KeyModifierShift }
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyLeft})
	v.keyModifiers = func() fyne.KeyModifier { return 0 }
	if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, width*0.54) {
		t.Errorf("viewer Shift+Left routed divider center to %.2f, want %.2f", center, width*0.54)
	}
	if got := snapshotCompareCommands(v); got != before {
		t.Errorf("divider keys changed covered app state\n got: %+v\nwant: %+v", got, before)
	}
}

func TestCompareDividerKeys_RemainActiveWhilePanesAreUnlinked(t *testing.T) {
	v := openActiveComparisonWithExtra(t)
	fynetest.Tap(comparisonButton(t, v.compare.Overlay(), lang.L("Swipe")))
	divider, _ := comparisonDivider(t, v.compare.Overlay())
	width := v.compare.Overlay().Size().Width
	var modifiers fyne.KeyModifier
	v.keyModifiers = func() fyne.KeyModifier { return modifiers }
	hooks := &comparisonKeyHooks{}
	wireComparisonLinkToggleHook(hooks, v)

	modifiers = fyne.KeyModifierControl
	hooks.down(&fyne.KeyEvent{Name: fyne.KeyL})
	modifiers = 0
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})
	if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, width*0.55) {
		t.Fatalf("unlinked Right divider center = %.2f, want %.2f", center, width*0.55)
	}
	modifiers = fyne.KeyModifierShift
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyLeft})
	if center := divider.Position().X + divider.Size().Width/2; !uitest.ApproxEqual(center, width*0.54) {
		t.Errorf("unlinked Shift+Left divider center = %.2f, want fine step to %.2f", center, width*0.54)
	}
	if !comparisonHasVisibleLabel(v.compare.Overlay(), lang.L("Unlinked")) {
		t.Error("divider keys relinked panes toggled apart by Ctrl+L")
	}
}

func TestCompareGridLeak_PointerGesturesCannotScrollCoveredGrid(t *testing.T) {
	names := make([]string, 40)
	for i := range names {
		names[i] = fmt.Sprintf("photo-%02d.jpg", i)
	}
	v := openGridWith(t, names...)
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	fireCompareShortcut(v)
	waitForCompare(t, v)

	before := snapshotCompareCommands(v)
	size := v.win.Canvas().Size()
	center := fyne.NewPos(size.Width/2, size.Height/2)
	fynetest.TapCanvas(v.win.Canvas(), center)
	fynetest.Drag(v.win.Canvas(), center, 120, 80)
	fynetest.Scroll(v.win.Canvas(), center, 0, -280)
	fynetest.MoveMouse(v.win.Canvas(), center.Add(fyne.NewPos(130, 0)))

	if got := snapshotCompareCommands(v); got != before {
		t.Errorf("pointer input reached the covered grid\n got: %+v\nwant: %+v", got, before)
	}
}

func TestCompareOpenRefusal_DropDialogShortcutAndOpenWithAreDiscarded(t *testing.T) {
	wantToast := lang.L("Return to Grid View before opening files")
	assertRefused := func(t *testing.T, v *viewer, before compareCommandSnapshot) {
		t.Helper()
		if got := snapshotCompareCommands(v); got != before {
			t.Errorf("open request changed comparison state\n got: %+v\nwant: %+v", got, before)
		}
		if got := v.toast.text.Text; got != wantToast || !v.toast.card.Visible() {
			t.Errorf("refusal toast = %q (visible %v), want visible %q", got, v.toast.card.Visible(), wantToast)
			return
		}
		settleToast(t, v)
	}

	t.Run("Drop", func(t *testing.T) {
		v := openActiveComparisonWithExtra(t)
		incoming := uitest.TempJPEGURI(t, "drop.jpg", 4, 4, color.White)
		before := snapshotCompareCommands(v)
		v.handleDrop([]fyne.URI{incoming})
		assertRefused(t, v, before)
	})

	t.Run("FileDialog", func(t *testing.T) {
		v := openActiveComparisonWithExtra(t)
		uitest.StubChooser(t, nil, nil)
		before := snapshotCompareCommands(v)
		v.openFileDialog()
		if v.chooser.Begun() {
			settleChooser(t, v)
		}
		assertRefused(t, v, before)
	})

	for _, modifier := range []fyne.KeyModifier{
		fyne.KeyModifierShortcutDefault,
		fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift,
	} {
		t.Run(fmt.Sprintf("FileDialogShortcut-%d", modifier), func(t *testing.T) {
			v := openActiveComparisonWithExtra(t)
			uitest.StubChooser(t, nil, nil)
			before := snapshotCompareCommands(v)
			handler := &fyne.ShortcutHandler{}
			wireGlobalShortcuts(handler, v)
			handler.TypedShortcut(&desktop.CustomShortcut{KeyName: fyne.KeyO, Modifier: modifier})
			if v.chooser.Begun() {
				settleChooser(t, v)
			}
			assertRefused(t, v, before)
		})
	}

	t.Run("OpenWith", func(t *testing.T) {
		v := openActiveComparisonWithExtra(t)
		pending := uitest.TempJPEGURI(t, "pending.jpg", 4, 4, color.White)
		delivered := uitest.TempJPEGURI(t, "delivered.jpg", 4, 4, color.White)
		v.pendingInitial = []fyne.URI{pending}
		v.installOpenWithHandler()
		before := snapshotCompareCommands(v)
		openwith.Deliver([]fyne.URI{delivered})
		assertRefused(t, v, before)
		if v.pendingInitial != nil {
			t.Errorf("pendingInitial = %v, want discarded rather than queued", v.pendingInitial)
		}
	})
}

func TestCompareSelection_HiddenHostIndicesDetermineLeftAndRight(t *testing.T) {
	v := openGridWith(t, "a-visible.jpg", "b-selected.jpg", "c-visible.jpg", "d-selected.jpg")

	// Select the later file first, then the earlier one. The grid's selection
	// set, not gesture order, owns comparison order.
	for range 3 {
		v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	}
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace}) // host 3
	for range 2 {
		v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyLeft})
	}
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace}) // host 1
	if got, want := v.grid.Selection(), []int{1, 3}; !slices.Equal(got, want) {
		t.Fatalf("setup selection = %v, want %v", got, want)
	}

	v.handleTypedRune('/')
	v.handleTypedRune('a') // only a-visible remains; both selected files hide
	if v.menus.Actions().Compare().Disabled {
		t.Fatal("filename filter disabled Compare even though two host files remain selected")
	}

	v.compareLoad = func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		width := 222
		if uri.Name() == "b-selected.jpg" {
			width = 111
		}
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, width, 40))}}, nil
	}
	fireCompareShortcut(v)
	waitForCompare(t, v)

	images := comparisonShaders(v.compare.Overlay())
	if len(images) != 2 {
		t.Fatalf("comparison images = %d, want 2", len(images))
	}
	if left, right := images[0].Textures["overview"].Bounds().Dx(), images[1].Textures["overview"].Bounds().Dx(); left != 111 || right != 222 {
		t.Errorf("left/right source widths = %d/%d, want 111/222 in ascending host order", left, right)
	}
	if got, want := v.grid.Selection(), []int{1, 3}; !slices.Equal(got, want) {
		t.Errorf("covered grid selection = %v, want %v", got, want)
	}
	if v.grid.Query() != "a" {
		t.Errorf("covered grid query = %q, want %q", v.grid.Query(), "a")
	}
	v.compare.Close()
}

func TestCompareSelection_DuplicateFilterKeepsHiddenSelectedHostFile(t *testing.T) {
	v := loadPatternedTriple(t)
	v.grid.Toggle()
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace}) // duplicate extra, host 1
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyRight})
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace}) // unique, host 2

	v.grid.SetHideDuplicates(true)
	v.grid.Settle()
	if !v.dupes.IsHiddenExtra(1) {
		t.Fatal("setup: host file 1 should be hidden by the duplicate filter")
	}
	if got, want := v.grid.Selection(), []int{1, 2}; !slices.Equal(got, want) {
		t.Fatalf("selection after duplicate filter = %v, want %v", got, want)
	}
	if v.menus.Actions().Compare().Disabled {
		t.Fatal("duplicate filter disabled Compare with two selected host files")
	}

	v.compareLoad = func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		width := 202
		if uri.Name() == "b.jpg" {
			width = 101
		}
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, width, 30))}}, nil
	}
	fireCompareShortcut(v)
	waitForCompare(t, v)
	images := comparisonShaders(v.compare.Overlay())
	if len(images) != 2 || images[0].Textures["overview"].Bounds().Dx() != 101 || images[1].Textures["overview"].Bounds().Dx() != 202 {
		t.Errorf("comparison did not retain hidden host selection in file order")
	}
	v.compare.Close()
}

func TestCompareCancel_EscapeStopsPendingLoadsAndPreservesSelection(t *testing.T) {
	v := openGridWith(t, "a.jpg", "b.jpg")
	v.grid.SelectAll()
	beforeTitle := v.win.Title()
	beforeFiles := append([]fyne.URI(nil), v.state.files...)

	started := make(chan string, 2)
	v.compareLoad = func(ctx context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		started <- uri.Name()
		<-ctx.Done()
		return nil, ctx.Err()
	}
	fireCompareShortcut(v)
	for range 2 {
		select {
		case <-started:
		case <-time.After(testTimeout):
			t.Fatal("timed out waiting for both comparison loads to start")
		}
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})
	waitForCompare(t, v)
	if v.compare.Visible() || !v.grid.Visible() {
		t.Fatal("Escape should reveal the still-open grid")
	}
	if got, want := v.grid.Selection(), []int{0, 1}; !slices.Equal(got, want) {
		t.Errorf("selection after cancellation = %v, want %v", got, want)
	}
	if v.win.Title() != beforeTitle {
		t.Errorf("title after cancellation = %q, want %q", v.win.Title(), beforeTitle)
	}
	if !slices.EqualFunc(v.state.files, beforeFiles, func(a, b fyne.URI) bool { return a.String() == b.String() }) {
		t.Error("file set changed after comparison cancellation")
	}
	if v.toast.stop != nil {
		t.Error("user cancellation reported a failure toast")
		settleToast(t, v)
	}
}

func TestCompareFailure_ReturnsToGridWithoutRemovingEitherFile(t *testing.T) {
	v := openGridWith(t, "a.jpg", "b.jpg")
	v.grid.SelectAll()
	beforeTitle := v.win.Title()
	wantErr := errors.New("broken comparison source")

	v.compareLoad = func(ctx context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		if uri.Name() == "b.jpg" {
			return nil, wantErr
		}
		<-ctx.Done()
		return nil, ctx.Err()
	}
	fireCompareShortcut(v)
	waitForCompare(t, v)

	if v.compare.Visible() || !v.grid.Visible() {
		t.Fatal("decode failure should close comparison and reveal Grid View")
	}
	if got, want := v.grid.Selection(), []int{0, 1}; !slices.Equal(got, want) {
		t.Errorf("selection after failure = %v, want %v", got, want)
	}
	if v.FileCount() != 2 || v.FileAt(0).Name() != "a.jpg" || v.FileAt(1).Name() != "b.jpg" {
		t.Errorf("file set after failure = %v, want both original files", v.state.files)
	}
	if v.win.Title() != beforeTitle {
		t.Errorf("title after failure = %q, want %q", v.win.Title(), beforeTitle)
	}
	if !strings.Contains(v.toast.text.Text, wantErr.Error()) {
		t.Errorf("failure toast = %q, want it to contain %q", v.toast.text.Text, wantErr)
	}
	settleToast(t, v)
}

func TestCompareExit_BackToGridLeavesTheGridSelectionAndTitleUntouched(t *testing.T) {
	v := openGridWith(t, "a.jpg", "b.jpg")
	v.grid.SelectAll()
	beforeTitle := v.win.Title()

	fireCompareShortcut(v)
	waitForCompare(t, v)
	fynetest.Tap(comparisonBackButton(t, v.compare.Overlay()))

	if v.compare.Visible() || !v.grid.Visible() {
		t.Fatal("Back to Grid should remove only the comparison overlay")
	}
	if got, want := v.grid.Selection(), []int{0, 1}; !slices.Equal(got, want) {
		t.Errorf("selection after Back to Grid = %v, want %v", got, want)
	}
	if v.win.Title() != beforeTitle {
		t.Errorf("title after Back to Grid = %q, want %q", v.win.Title(), beforeTitle)
	}
}

func TestCompareTitle_TracksIdentityOrderAndRestoresGridTitle(t *testing.T) {
	v := openGridWith(t, "left.jpg", "right.jpg")
	v.grid.SelectAll()
	beforeTitle := v.win.Title()
	started := make(chan struct{}, 2)
	release := make(chan struct{})
	v.compareLoad = func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		started <- struct{}{}
		<-release
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 20, 10))}}, nil
	}

	fireCompareShortcut(v)
	for range 2 {
		select {
		case <-started:
		case <-time.After(testTimeout):
			t.Fatal("timed out waiting for comparison title probe loads")
		}
	}
	if got, want := v.win.Title(), "Compare: left.jpg | right.jpg - PicFetch"; got != want {
		t.Fatalf("comparison title while loading = %q, want %q", got, want)
	}
	close(release)
	waitForCompare(t, v)

	fynetest.Tap(comparisonButton(t, v.compare.Overlay(), lang.L("Swap")))
	if got, want := v.win.Title(), "Compare: right.jpg | left.jpg - PicFetch"; got != want {
		t.Errorf("title after Swap = %q, want %q", got, want)
	}

	fynetest.Tap(comparisonBackButton(t, v.compare.Overlay()))
	if got := v.win.Title(); got != beforeTitle {
		t.Errorf("title after Back to Grid = %q, want restored %q", got, beforeTitle)
	}
}

func TestCompareTitle_OwnsTheWindowUntilComparisonCloses(t *testing.T) {
	v := openGridWith(t, "left.jpg", "right.jpg")
	v.grid.SelectAll()
	fireCompareShortcut(v)
	waitForCompare(t, v)
	want := "Compare: left.jpg | right.jpg - PicFetch"
	if got := v.win.Title(); got != want {
		t.Fatalf("comparison title = %q, want %q", got, want)
	}

	v.applyTitle()
	if got := v.win.Title(); got != want {
		t.Errorf("ordinary title refresh replaced active comparison title with %q", got)
	}
	v.compare.Close()
}

func TestCompareSwapPreservesGrid_StateOrderAndSelectionAnchor(t *testing.T) {
	names := make([]string, 40)
	for i := range names {
		names[i] = fmt.Sprintf("photo-%02d.jpg", i)
	}
	v := openGridWith(t, names...)
	v.handleTypedRune('/')
	v.handleTypedRune('p')

	wrap := comparisonGridWrap(t, v.grid.Overlay())
	v.keyModifiers = func() fyne.KeyModifier { return fyne.KeyModifierShortcutDefault }
	wrap.Select(1)
	wrap.Select(3) // selection anchor: host file 3
	v.keyModifiers = func() fyne.KeyModifier { return 0 }
	for range 2 {
		v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyPageDown})
	}

	beforeFiles := append([]fyne.URI(nil), v.state.files...)
	beforeSelection := v.grid.Selection()
	beforeQuery := v.grid.Query()
	beforeHighlight := v.grid.Highlight()
	beforeScroll := v.grid.ScrollOffset()
	beforeTitle := v.win.Title()
	if beforeScroll <= 0 || !slices.Equal(beforeSelection, []int{1, 3}) {
		t.Fatalf("setup grid = {scroll:%v selection:%v}, want nonzero scroll and [1 3]", beforeScroll, beforeSelection)
	}

	fireCompareShortcut(v)
	waitForCompare(t, v)
	fynetest.Tap(comparisonButton(t, v.compare.Overlay(), lang.L("Swap")))
	fynetest.Tap(comparisonBackButton(t, v.compare.Overlay()))

	if !slices.EqualFunc(v.state.files, beforeFiles, func(a, b fyne.URI) bool { return a.String() == b.String() }) {
		t.Error("file order changed across comparison Swap")
	}
	if got := v.grid.Selection(); !slices.Equal(got, beforeSelection) {
		t.Errorf("selection after Swap = %v, want %v", got, beforeSelection)
	}
	if v.grid.Query() != beforeQuery || v.grid.Highlight() != beforeHighlight || v.grid.ScrollOffset() != beforeScroll {
		t.Errorf("grid after Swap = {query:%q highlight:%d scroll:%v}, want {%q %d %v}",
			v.grid.Query(), v.grid.Highlight(), v.grid.ScrollOffset(), beforeQuery, beforeHighlight, beforeScroll)
	}
	if v.win.Title() != beforeTitle {
		t.Errorf("title after Swap and return = %q, want %q", v.win.Title(), beforeTitle)
	}

	// A new comparison follows file order again; the prior session's swapped
	// presentation order never leaks into the grid or the next Open.
	fireCompareShortcut(v)
	if got, want := v.win.Title(), "Compare: photo-01.jpg | photo-03.jpg - PicFetch"; got != want {
		t.Errorf("new comparison title = %q, want reset file order %q", got, want)
	}
	v.compare.Close()
	waitForCompare(t, v)

	// Shift-selection must still extend from host file 3, the last selection
	// gesture before comparison. Membership alone cannot observe this anchor.
	v.keyModifiers = func() fyne.KeyModifier { return fyne.KeyModifierShift }
	wrap.Select(5)
	v.keyModifiers = func() fyne.KeyModifier { return 0 }
	if got, want := v.grid.Selection(), []int{1, 3, 4, 5}; !slices.Equal(got, want) {
		t.Errorf("selection after post-Swap Shift-click = %v, want %v from the preserved anchor", got, want)
	}
}

func TestCompareTransitionPreservesGrid_StateAndTitle(t *testing.T) {
	names := make([]string, 40)
	for i := range names {
		names[i] = fmt.Sprintf("photo-%02d.jpg", i)
	}
	v := openGridWith(t, names...)
	v.handleTypedRune('/')
	v.handleTypedRune('p')

	wrap := comparisonGridWrap(t, v.grid.Overlay())
	v.keyModifiers = func() fyne.KeyModifier { return fyne.KeyModifierShortcutDefault }
	wrap.Select(1)
	wrap.Select(3)
	v.keyModifiers = func() fyne.KeyModifier { return 0 }
	for range 2 {
		v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyPageDown})
	}

	before := snapshotCompareCommands(v)
	beforeTitle := v.win.Title()
	beforeSize := v.win.Canvas().Size()
	if before.scroll <= 0 || before.selection != "[1 3]" || before.query == "" {
		t.Fatalf("setup grid = {scroll:%v selection:%s query:%q}, want nontrivial covered state", before.scroll, before.selection, before.query)
	}

	fireCompareShortcut(v)
	waitForCompare(t, v)
	activeBefore := snapshotCompareCommands(v)
	for range 7 {
		v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})
	}
	fynetest.Tap(comparisonButton(t, v.compare.Overlay(), lang.L("Swipe")))
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyRight})

	shrunken := fyne.NewSize(beforeSize.Width-40, beforeSize.Height-40)
	v.win.Resize(shrunken)
	v.win.Resize(beforeSize)
	fynetest.Tap(comparisonButton(t, v.compare.Overlay(), lang.L("Swap")))
	if got, want := v.win.Title(), "Compare: photo-03.jpg | photo-01.jpg - PicFetch"; got != want {
		t.Errorf("title after transitioned Swap = %q, want %q", got, want)
	}
	fynetest.Tap(comparisonButton(t, v.compare.Overlay(), lang.L("Side by side")))
	if got := snapshotCompareCommands(v); got != activeBefore {
		t.Errorf("comparison transitions changed covered grid state\n got: %+v\nwant: %+v", got, activeBefore)
	}

	fynetest.Tap(comparisonBackButton(t, v.compare.Overlay()))
	if got := snapshotCompareCommands(v); got != before {
		t.Errorf("grid after comparison transition round trip\n got: %+v\nwant: %+v", got, before)
	}
	if got := v.win.Title(); got != beforeTitle {
		t.Errorf("title after comparison transition round trip = %q, want restored %q", got, beforeTitle)
	}
}

func TestCompareRestoration_PreservesANonzeroGridScrollPosition(t *testing.T) {
	names := make([]string, 40)
	for i := range names {
		names[i] = fmt.Sprintf("%02d.jpg", i)
	}
	v := openGridWith(t, names...)
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	for range 2 {
		v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeyPageDown})
	}
	v.grid.HandleKey(&fyne.KeyEvent{Name: fyne.KeySpace})

	beforeScroll := v.grid.ScrollOffset()
	beforeHighlight := v.grid.Highlight()
	beforeSelection := v.grid.Selection()
	if beforeScroll <= 0 || len(beforeSelection) != 2 {
		t.Fatalf("setup grid = {scroll:%v selection:%v}, want nonzero scroll and two files", beforeScroll, beforeSelection)
	}

	fireCompareShortcut(v)
	waitForCompare(t, v)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyEscape})

	if got := v.grid.ScrollOffset(); got != beforeScroll {
		t.Errorf("scroll offset after comparison = %v, want %v", got, beforeScroll)
	}
	if got := v.grid.Highlight(); got != beforeHighlight {
		t.Errorf("highlight after comparison = %d, want %d", got, beforeHighlight)
	}
	if got := v.grid.Selection(); !slices.Equal(got, beforeSelection) {
		t.Errorf("selection after comparison = %v, want %v", got, beforeSelection)
	}
}
