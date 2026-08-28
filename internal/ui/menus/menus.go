// Package menus owns the window menu bar's stateful items - the File,
// Window and Actions entries whose Checked/Disabled state has to move as
// the app's state moves - and computes that whole matrix in one place, as
// a pure function of a State value snapshot.
//
// It is Fyne-typed but viewer-free: nothing here reads the app, so the
// enablement matrix is testable with no Fyne app at all. internal/ui
// fills a State in exactly one function and calls Apply.
//
// Deliberately not here: the *fyne.MainMenu assembly (the bar also
// carries the favorites and help feature menus, and internal/ui decides
// how features compose - see ARCHITECTURE.md), the Darwin native-bar
// follow-up after every rebuild (internal/ui/windowmenu.go's
// refreshMainMenu/syncNativeMenuBar and the cgo behind them), the real
// keyboard bindings for the accelerators shown here
// (internal/ui/shortcuts.go), and every action the callbacks below run.
package menus

import (
	"slices"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/filesort"
)

// Callbacks are the actions the menu items run. internal/ui supplies them
// as method values on the viewer, and every handler stays there: this
// package never decides what an item does, only whether it is available.
type Callbacks struct {
	OpenFiles    func()
	SaveRotation func()
	PromptExport func()
	CloseFiles   func()
	ShowSettings func()

	ShowViewer       func()
	ShowExif         func()
	ShowGrid         func()
	ShowPictureFrame func()
	ShowHelp         func()

	SetSort              func(filesort.Mode)
	ToggleHideDuplicates func()
	ShowVariant          func()
	Rotate               func()
	ZoomIn               func()
	ZoomOut              func()
	ToggleMergeMode      func()
	ToggleInfoOverlay    func()
	CopyImage            func()
	CopyPath             func()
	SetWallpaper         func()
	Trash                func()
}

// State is the snapshot Apply reads: every app condition the enablement
// matrix depends on, as one struct literal you can take in at a glance.
// A Host interface for this would need a dozen methods and would leave
// the coupling implicit; a value makes it explicit and testable.
type State struct {
	// SortMode is the mode whose entry in the Sort order submenu is checked.
	SortMode filesort.Mode
	// VariantGroupSize is the duplicate-group size of the file a variant
	// browse would start from. "Show variants" needs at least 2.
	VariantGroupSize int

	NoFiles            bool // nothing is loaded at all
	GridUp             bool // the grid overview is showing
	NoImage            bool // no decoded frame, so nothing to rotate or zoom
	SlidesActive       bool // picture-frame mode is running
	ExifOpen           bool // the EXIF window is already open
	ManualOpen         bool // the manual window is already open
	Displayed          bool // there is a current file on display
	MergeMode          bool
	HideDuplicates     bool
	BrowsingDuplicates bool
	VariantsSession    bool // browsing duplicates, or inspecting a group
	InfoVisible        bool
	CanSave            bool // a pending rotation can be written back
	CanExport          bool
	CanWallpaper       bool
}

// Menus holds every menu item whose Checked or Disabled state moves at
// runtime, which is why they are kept as fields at all rather than being
// local to the construction below. It also keeps the two File items that
// never move (Open, Settings) and the Sort order parent, so the three
// menu compositions can live next to the items they are made of.
type Menus struct {
	open       *fyne.MenuItem
	save       *fyne.MenuItem
	export     *fyne.MenuItem
	closeFiles *fyne.MenuItem
	settings   *fyne.MenuItem

	sortParent *fyne.MenuItem

	window  WindowItems
	actions ActionItems
}

// WindowItems are the Window menu's items: which surface is already
// showing decides which of them is available.
type WindowItems struct {
	viewer       *fyne.MenuItem
	exif         *fyne.MenuItem
	grid         *fyne.MenuItem
	pictureFrame *fyne.MenuItem
	help         *fyne.MenuItem
}

// ActionItems are the Actions menu's items: sort, duplicates, image
// transforms, merge/info toggles, clipboard, wallpaper, and trash.
type ActionItems struct {
	sort        []*fyne.MenuItem // len 5, index matches filesort.Modes()
	hide        *fyne.MenuItem
	showVariant *fyne.MenuItem
	rotate      *fyne.MenuItem
	zoomIn      *fyne.MenuItem
	zoomOut     *fyne.MenuItem
	merge       *fyne.MenuItem
	info        *fyne.MenuItem
	copy        *fyne.MenuItem
	copyPath    *fyne.MenuItem
	wallpaper   *fyne.MenuItem
	trash       *fyne.MenuItem
}

// New builds every item with its label, its accelerator, and the Disabled
// state it starts in. sortMode is the mode checked in the Sort order
// submenu to begin with; Apply moves that check from there on.
func New(c Callbacks, sortMode filesort.Mode) *Menus {
	m := &Menus{}

	m.open = fyne.NewMenuItem(lang.L("Open Files…"), c.OpenFiles)
	// Display-only: the Cmd/Ctrl+O binding itself is wireOpenShortcuts's
	// AddShortcut call in internal/ui/shortcuts.go. This just shows the
	// same accelerator as a hint next to the menu item.
	m.open.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyO,
		Modifier: fyne.KeyModifierShortcutDefault,
	}

	m.save = fyne.NewMenuItem(lang.L("Save Changes"), c.SaveRotation)
	m.save.Disabled = true // Apply enables it once State.CanSave reports a pending rotation to save
	m.save.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyS,
		Modifier: fyne.KeyModifierShortcutDefault,
	}

	m.export = fyne.NewMenuItem(lang.L("Export image"), c.PromptExport)
	m.export.Disabled = true // Apply enables it once State.CanExport reports an image is loaded
	// Display-only, like Open's above: the binding itself is
	// wireExportShortcuts's AddShortcut call in internal/ui/shortcuts.go.
	m.export.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyE,
		Modifier: fyne.KeyModifierShortcutDefault,
	}

	m.closeFiles = fyne.NewMenuItem(lang.L("Close Files"), c.CloseFiles)
	m.closeFiles.Disabled = true // Apply enables it once State.NoFiles is false, i.e. a file is loaded
	m.settings = fyne.NewMenuItem(lang.L("Settings…"), c.ShowSettings)

	m.window.viewer = fyne.NewMenuItem(lang.L("Viewer"), c.ShowViewer)
	m.window.viewer.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyV}
	m.window.viewer.Disabled = true

	m.window.exif = fyne.NewMenuItem(lang.L("EXIF Data"), c.ShowExif)
	m.window.exif.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyE}
	m.window.exif.Disabled = true

	m.window.grid = fyne.NewMenuItem(lang.L("Grid View"), c.ShowGrid)
	m.window.grid.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyG}
	m.window.grid.Disabled = true

	m.window.pictureFrame = fyne.NewMenuItem(lang.L("Picture-frame mode"), c.ShowPictureFrame)
	m.window.pictureFrame.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyP}
	m.window.pictureFrame.Disabled = true

	m.window.help = fyne.NewMenuItem(lang.L("Help"), c.ShowHelp)
	m.window.help.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyF1}

	modes := filesort.Modes()
	sortItems := make([]*fyne.MenuItem, len(modes))
	for i, mode := range modes {
		it := fyne.NewMenuItem(filesort.DisplayName(mode), func() { c.SetSort(mode) })
		if mode == sortMode {
			it.Checked = true
		}
		sortItems[i] = it
	}
	m.actions.sort = sortItems

	m.sortParent = fyne.NewMenuItem(lang.L("Sort order"), nil)
	m.sortParent.ChildMenu = fyne.NewMenu("", sortItems...)
	m.sortParent.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyS}

	m.actions.hide = fyne.NewMenuItem(lang.L("Show/Hide duplicates"), c.ToggleHideDuplicates)
	m.actions.hide.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyD}
	m.actions.hide.Disabled = true

	m.actions.showVariant = fyne.NewMenuItem(lang.L("Show variants"), c.ShowVariant)
	m.actions.showVariant.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyD,
		Modifier: fyne.KeyModifierShift,
	}
	m.actions.showVariant.Disabled = true

	m.actions.rotate = fyne.NewMenuItem(lang.L("Rotate image"), c.Rotate)
	m.actions.rotate.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyR}
	m.actions.rotate.Disabled = true

	m.actions.zoomIn = fyne.NewMenuItem(lang.L("Zoom in"), c.ZoomIn)
	m.actions.zoomIn.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyPlus}
	m.actions.zoomIn.Disabled = true

	m.actions.zoomOut = fyne.NewMenuItem(lang.L("Zoom out"), c.ZoomOut)
	m.actions.zoomOut.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyMinus}
	m.actions.zoomOut.Disabled = true

	m.actions.merge = fyne.NewMenuItem(lang.L("Toggle merge mode"), c.ToggleMergeMode)
	// Unmodified M. Fyne's Darwin native menus leave a zero modifier mask
	// unset, so AppKit would default this to ⌘M (Minimize);
	// internal/ui/windowmenu.go's refreshMainMenu clears that via
	// applyUnmodifiedNativeAccelerators.
	m.actions.merge.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyM}

	m.actions.info = fyne.NewMenuItem(lang.L("Show/Hide info overlay"), c.ToggleInfoOverlay)
	m.actions.info.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyI}

	m.actions.copy = fyne.NewMenuItem(lang.L("Copy image"), c.CopyImage)
	// Display-only: the Cmd/Ctrl+C binding is wireClipboardShortcuts's
	// AddShortcut of fyne.ShortcutCopy (internal/ui/shortcuts.go). A second
	// CustomShortcut here would double-fire copy.
	m.actions.copy.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyC, Modifier: fyne.KeyModifierShortcutDefault}
	m.actions.copy.Disabled = true

	m.actions.copyPath = fyne.NewMenuItem(lang.L("Copy image path"), c.CopyPath)
	// Display-only, like File -> Export: the Cmd/Ctrl+Shift+C binding is
	// wireClipboardShortcuts (internal/ui/shortcuts.go). This just shows
	// the same accelerator.
	m.actions.copyPath.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyC,
		Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift,
	}
	m.actions.copyPath.Disabled = true

	m.actions.wallpaper = fyne.NewMenuItem(lang.L("Set as Wallpaper"), c.SetWallpaper)
	// Display-only: the Cmd/Ctrl+Shift+E binding is wireExportShortcuts
	// (internal/ui/shortcuts.go).
	m.actions.wallpaper.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyE,
		Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift,
	}
	m.actions.wallpaper.Disabled = true

	m.actions.trash = fyne.NewMenuItem(lang.L("Move image to Trash"), c.Trash)
	// Display-only: the Shift+Delete binding is wireDeleteShortcut's
	// AddShortcut of fyne.ShortcutCut (internal/ui/shortcuts.go). A
	// CustomShortcut{KeyDelete, Shift} would never be reached by the
	// driver anyway.
	m.actions.trash.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyDelete, Modifier: fyne.KeyModifierShift}
	m.actions.trash.Disabled = true

	return m
}

// FileMenu is the File menu: open, the three state-driven items, then
// settings behind a separator.
func (m *Menus) FileMenu() *fyne.Menu {
	return fyne.NewMenu(lang.L("File"),
		m.open, m.save, m.export, m.closeFiles, fyne.NewMenuItemSeparator(), m.settings)
}

// ActionsMenu is the Actions menu, in four separator-delimited groups:
// sort and duplicates, image transforms, the merge/info toggles, then
// what can be done with the current file.
func (m *Menus) ActionsMenu() *fyne.Menu {
	return fyne.NewMenu(lang.L("Actions"),
		m.sortParent, m.actions.hide, m.actions.showVariant,
		fyne.NewMenuItemSeparator(),
		m.actions.rotate, m.actions.zoomIn, m.actions.zoomOut,
		fyne.NewMenuItemSeparator(),
		m.actions.merge, m.actions.info,
		fyne.NewMenuItemSeparator(),
		m.actions.copy, m.actions.copyPath, m.actions.wallpaper, m.actions.trash,
	)
}

// WindowMenu is the Window menu: one item per surface the app can show.
func (m *Menus) WindowMenu() *fyne.Menu {
	return fyne.NewMenu(lang.L("Window"),
		m.window.viewer, m.window.exif, m.window.grid, m.window.pictureFrame, m.window.help)
}

// Save is the File menu's "Save Changes" item.
func (m *Menus) Save() *fyne.MenuItem { return m.save }

// Export is the File menu's "Export image" item.
func (m *Menus) Export() *fyne.MenuItem { return m.export }

// CloseFiles is the File menu's "Close Files" item.
func (m *Menus) CloseFiles() *fyne.MenuItem { return m.closeFiles }

// Window returns the Window menu's items.
func (m *Menus) Window() WindowItems { return m.window }

// Actions returns the Actions menu's items.
func (m *Menus) Actions() ActionItems { return m.actions }

// Viewer is the Window menu's "Viewer" item.
func (w WindowItems) Viewer() *fyne.MenuItem { return w.viewer }

// Exif is the Window menu's "EXIF Data" item.
func (w WindowItems) Exif() *fyne.MenuItem { return w.exif }

// Grid is the Window menu's "Grid View" item.
func (w WindowItems) Grid() *fyne.MenuItem { return w.grid }

// PictureFrame is the Window menu's "Picture-frame mode" item.
func (w WindowItems) PictureFrame() *fyne.MenuItem { return w.pictureFrame }

// Help is the Window menu's "Help" item.
func (w WindowItems) Help() *fyne.MenuItem { return w.help }

// Sort returns the Sort order submenu's items, one per filesort.Modes()
// entry and in that order. The slice is the live one, not a copy - it is
// read to assert on an item, not to be reordered.
func (a ActionItems) Sort() []*fyne.MenuItem { return a.sort }

// Hide is the Actions menu's "Show/Hide duplicates" item.
func (a ActionItems) Hide() *fyne.MenuItem { return a.hide }

// ShowVariant is the Actions menu's "Show variants" item.
func (a ActionItems) ShowVariant() *fyne.MenuItem { return a.showVariant }

// Rotate is the Actions menu's "Rotate image" item.
func (a ActionItems) Rotate() *fyne.MenuItem { return a.rotate }

// ZoomIn is the Actions menu's "Zoom in" item.
func (a ActionItems) ZoomIn() *fyne.MenuItem { return a.zoomIn }

// ZoomOut is the Actions menu's "Zoom out" item.
func (a ActionItems) ZoomOut() *fyne.MenuItem { return a.zoomOut }

// Merge is the Actions menu's "Toggle merge mode" item.
func (a ActionItems) Merge() *fyne.MenuItem { return a.merge }

// Info is the Actions menu's "Show/Hide info overlay" item.
func (a ActionItems) Info() *fyne.MenuItem { return a.info }

// Copy is the Actions menu's "Copy image" item.
func (a ActionItems) Copy() *fyne.MenuItem { return a.copy }

// CopyPath is the Actions menu's "Copy image path" item.
func (a ActionItems) CopyPath() *fyne.MenuItem { return a.copyPath }

// Wallpaper is the Actions menu's "Set as Wallpaper" item.
func (a ActionItems) Wallpaper() *fyne.MenuItem { return a.wallpaper }

// Trash is the Actions menu's "Move image to Trash" item.
func (a ActionItems) Trash() *fyne.MenuItem { return a.trash }

// Apply writes the whole Checked/Disabled matrix from s and reports
// whether any item actually moved, so internal/ui can rebuild the native
// menu bar only when there is something new to show.
//
// It recomputes every item on every call rather than trusting a caller to
// say what changed: the matrix is 20 items of boolean arithmetic, and a
// caller that guesses wrong is exactly how a menu goes stale.
func (m *Menus) Apply(s State) (changed bool) {
	before := m.pairs()

	m.applyFile(s)
	m.applyWindow(s)
	m.applyActions(s)

	return !slices.Equal(before, m.pairs())
}

// applyFile is the File menu's share: what can be saved, exported or
// closed right now. Whether there are any files at all also drives the
// Favorites menu, but that stays in internal/ui - it is a feature menu,
// not one of these items.
func (m *Menus) applyFile(s State) {
	m.save.Disabled = !s.CanSave

	m.export.Disabled = !s.CanExport

	m.closeFiles.Disabled = s.NoFiles
}

// applyWindow greys out whichever surface is already showing.
func (m *Menus) applyWindow(s State) {
	m.window.viewer.Disabled = !s.GridUp && !s.SlidesActive
	m.window.exif.Disabled = s.ExifOpen || !s.Displayed
	m.window.grid.Disabled = s.GridUp || s.NoFiles || s.SlidesActive
	m.window.pictureFrame.Disabled = s.SlidesActive || s.NoFiles || s.VariantsSession
	m.window.help.Disabled = s.ManualOpen
}

// applyActions is the Actions menu's share of the matrix.
func (m *Menus) applyActions(s State) {
	modes := filesort.Modes()
	for i, item := range m.actions.sort {
		if item == nil || i >= len(modes) {
			continue
		}
		item.Checked = modes[i] == s.SortMode
		item.Disabled = false
	}
	noFiles := s.NoFiles
	gridUp := s.GridUp
	noImage := s.NoImage

	m.actions.hide.Checked = s.HideDuplicates
	m.actions.hide.Disabled = noFiles || s.VariantsSession
	m.actions.showVariant.Checked = s.BrowsingDuplicates
	canShowVariants := s.HideDuplicates && s.VariantGroupSize >= 2
	m.actions.showVariant.Disabled = noFiles || s.SlidesActive || !(canShowVariants || s.BrowsingDuplicates)

	rotZoomOff := noImage || gridUp
	m.actions.rotate.Disabled = rotZoomOff
	m.actions.zoomIn.Disabled = rotZoomOff
	m.actions.zoomOut.Disabled = rotZoomOff

	m.actions.merge.Checked = s.MergeMode
	m.actions.merge.Disabled = false
	m.actions.info.Checked = s.InfoVisible
	m.actions.info.Disabled = gridUp

	m.actions.copy.Disabled = noFiles
	m.actions.copyPath.Disabled = noFiles
	m.actions.wallpaper.Disabled = !s.CanWallpaper
	m.actions.trash.Disabled = noFiles
}

// pair is one item's observable menu state.
type pair struct {
	checked  bool
	disabled bool
}

// pairs snapshots every stateful item, in a fixed order, so Apply can
// diff the whole matrix instead of each assignment reporting for itself -
// an assignment added later is then covered without being told to be.
func (m *Menus) pairs() []pair {
	items := make([]*fyne.MenuItem, 0, len(m.actions.sort)+19)
	items = append(items, m.save, m.export, m.closeFiles)
	items = append(items, m.window.viewer, m.window.exif, m.window.grid,
		m.window.pictureFrame, m.window.help)
	items = append(items, m.actions.sort...)
	items = append(items, m.actions.hide, m.actions.showVariant, m.actions.rotate,
		m.actions.zoomIn, m.actions.zoomOut, m.actions.merge, m.actions.info,
		m.actions.copy, m.actions.copyPath, m.actions.wallpaper, m.actions.trash)

	out := make([]pair, len(items))
	for i, it := range items {
		out[i] = pair{checked: it.Checked, disabled: it.Disabled}
	}
	return out
}
