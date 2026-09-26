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
	FindMoreLikeThis func()
	OpenFiles        func()
	SaveRotation     func()
	PromptExport     func()
	CloseFiles       func()
	ShowSettings     func()

	ShowViewer       func()
	ShowExplorer     func()
	ShowLocationMap  func()
	ShowExif         func()
	ShowGrid         func()
	ShowPictureFrame func()
	ShowHelp         func()

	SetSort              func(filesort.Mode)
	ToggleHideDuplicates func()
	ShowVariant          func()
	Compare              func()
	Mosaic               func()
	Rotate               func()
	ZoomIn               func()
	ZoomOut              func()
	ToggleMergeMode      func()
	ToggleInfoOverlay    func()
	CopyImage            func()
	CopySelection        func()
	CopyPath             func()
	Reveal               func()
	SetWallpaper         func()
	Trash                func()
}

// State separates admission (decided by root UI) from menu presentation.
type State struct {
	Availability                                                                  Availability
	SortMode                                                                      filesort.Mode
	MergeMode, HideDuplicates, BrowsingDuplicates, InfoVisible, LocationMapActive bool
}

// Availability contains named decisions, not the cross-feature facts from
// which they were derived. Menus only render them; actions recheck on entry.
type Availability struct {
	Open, Save, Export, CloseFiles, Settings                                           bool
	Viewer, Explorer, LocationMap, Mosaic, Exif, Grid, PictureFrame, Help              bool
	Sort, HideDuplicates, BrowseDuplicates, Compare, Search                            bool
	Rotate, Zoom, Merge, Info, Copy, CopySelection, CopyPath, Reveal, Wallpaper, Trash bool
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
	explorer     *fyne.MenuItem
	locationMap  *fyne.MenuItem
	mosaic       *fyne.MenuItem
	exif         *fyne.MenuItem
	grid         *fyne.MenuItem
	pictureFrame *fyne.MenuItem
	help         *fyne.MenuItem
}

// ActionItems are the Actions menu's items: sort, duplicates, image
// transforms, merge/info toggles, clipboard, reveal, wallpaper, and trash.
type ActionItems struct {
	findMoreLikeThis *fyne.MenuItem
	sort             []*fyne.MenuItem // len 5, index matches filesort.Modes()
	hide             *fyne.MenuItem
	showVariant      *fyne.MenuItem
	compare          *fyne.MenuItem
	rotate           *fyne.MenuItem
	zoomIn           *fyne.MenuItem
	zoomOut          *fyne.MenuItem
	merge            *fyne.MenuItem
	info             *fyne.MenuItem
	copy             *fyne.MenuItem
	copySelection    *fyne.MenuItem
	copyPath         *fyne.MenuItem
	reveal           *fyne.MenuItem
	wallpaper        *fyne.MenuItem
	trash            *fyne.MenuItem
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
	m.save.Disabled = true // Apply renders the host's Save admission.
	m.save.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyS,
		Modifier: fyne.KeyModifierShortcutDefault,
	}

	m.export = fyne.NewMenuItem(lang.L("Export image"), c.PromptExport)
	m.export.Disabled = true // Apply renders the host's Export admission.
	// Display-only, like Open's above: the binding itself is
	// wireExportShortcuts's AddShortcut call in internal/ui/shortcuts.go.
	m.export.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyE,
		Modifier: fyne.KeyModifierShortcutDefault,
	}

	m.closeFiles = fyne.NewMenuItem(lang.L("Close Files"), c.CloseFiles)
	m.closeFiles.Disabled = true // Apply enables it once a file is loaded or a scan/sort starts
	m.settings = fyne.NewMenuItem(lang.L("Settings…"), c.ShowSettings)

	m.window.explorer = fyne.NewMenuItem(lang.L("Similarity Explorer"), c.ShowExplorer)
	m.window.locationMap = fyne.NewMenuItem(lang.L("Location Map"), c.ShowLocationMap)
	m.window.locationMap.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyL, Modifier: fyne.KeyModifierShift}
	m.window.explorer.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyS, Modifier: fyne.KeyModifierShift}
	m.window.mosaic = fyne.NewMenuItem(lang.L("Generate Image Mosaic..."), c.Mosaic)
	m.window.mosaic.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyM, Modifier: fyne.KeyModifierShift}
	m.window.mosaic.Disabled = true
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

	m.actions.findMoreLikeThis = fyne.NewMenuItem(lang.L("Find more like this"), c.FindMoreLikeThis)
	m.actions.findMoreLikeThis.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyL, Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift}
	m.actions.findMoreLikeThis.Disabled = true
	m.actions.compare = fyne.NewMenuItem(lang.L("Compare selected images"), c.Compare)
	m.actions.compare.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyD,
		Modifier: fyne.KeyModifierShortcutDefault,
	}
	m.actions.compare.Disabled = true

	m.actions.rotate = fyne.NewMenuItem(lang.L("Rotate image (CW)"), c.Rotate)
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
	// The canvas binding is wireClipboardShortcuts's AddShortcut of
	// fyne.ShortcutCopy (internal/ui/shortcuts.go). Native menu bars can invoke
	// this item's callback from the same accelerator instead; internal/ui gives
	// both paths the same context-aware copy command.
	m.actions.copy.Shortcut = &desktop.CustomShortcut{KeyName: fyne.KeyC, Modifier: fyne.KeyModifierShortcutDefault}
	m.actions.copy.Disabled = true

	m.actions.copySelection = fyne.NewMenuItem(lang.L("Copy selection"), c.CopySelection)
	m.actions.copySelection.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyC,
		Modifier: fyne.KeyModifierAlt | fyne.KeyModifierShift,
	}
	m.actions.copySelection.Disabled = true

	m.actions.copyPath = fyne.NewMenuItem(lang.L("Copy image path"), c.CopyPath)
	// Display-only, like File -> Export: the Cmd/Ctrl+Shift+C binding is
	// wireClipboardShortcuts (internal/ui/shortcuts.go). This just shows
	// the same accelerator.
	m.actions.copyPath.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyC,
		Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift,
	}
	m.actions.copyPath.Disabled = true

	// One label on every platform rather than Finder/Explorer/Files wording
	// per OS: a runtime.GOOS branch has no business in this deliberately
	// viewer-free package, and three labels would be three keys in every
	// catalogue for one command.
	m.actions.reveal = fyne.NewMenuItem(lang.L("Reveal in file manager"), c.Reveal)
	// Display-only, like File -> Export: the Cmd/Ctrl+R binding is
	// wireRevealShortcut (internal/ui/shortcuts.go). Unmodified R stays
	// Rotate, above.
	m.actions.reveal.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyR,
		Modifier: fyne.KeyModifierShortcutDefault,
	}
	m.actions.reveal.Disabled = true

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
// sort, duplicates and comparison, image transforms, the merge/info toggles, then
// what can be done with the current file.
func (m *Menus) ActionsMenu() *fyne.Menu {
	return fyne.NewMenu(lang.L("Actions"),
		m.sortParent, m.actions.hide, m.actions.showVariant, m.actions.compare, m.actions.findMoreLikeThis,
		fyne.NewMenuItemSeparator(),
		m.actions.rotate, m.actions.zoomIn, m.actions.zoomOut,
		fyne.NewMenuItemSeparator(),
		m.actions.merge, m.actions.info,
		fyne.NewMenuItemSeparator(),
		m.actions.copy, m.actions.copySelection, m.actions.copyPath, m.actions.reveal,
		m.actions.wallpaper, m.actions.trash,
	)
}

// WindowMenu is the Window menu: one item per surface the app can show.
func (m *Menus) WindowMenu() *fyne.Menu {
	return fyne.NewMenu(lang.L("Window"),
		m.window.viewer, m.window.exif, m.window.grid, m.window.pictureFrame, m.window.help, m.window.explorer, m.window.locationMap, m.window.mosaic)
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

// Compare is the Actions menu's "Compare selected images" item.
func (a ActionItems) Compare() *fyne.MenuItem { return a.compare }

// Mosaic is the Window menu's "Generate Image Mosaic..." item.
func (w WindowItems) Mosaic() *fyne.MenuItem { return w.mosaic }

// Rotate is the Actions menu's "Rotate image (CW)" item.
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

// CopySelection is the Actions menu's "Copy selection" item.
func (a ActionItems) CopySelection() *fyne.MenuItem { return a.copySelection }

// CopyPath is the Actions menu's "Copy image path" item.
func (a ActionItems) CopyPath() *fyne.MenuItem { return a.copyPath }

// Reveal is the Actions menu's "Reveal in file manager" item.
func (a ActionItems) Reveal() *fyne.MenuItem { return a.reveal }

// Wallpaper is the Actions menu's "Set as Wallpaper" item.
func (a ActionItems) Wallpaper() *fyne.MenuItem { return a.wallpaper }

// Trash is the Actions menu's "Move image to Trash" item.
func (a ActionItems) Trash() *fyne.MenuItem { return a.trash }

// Apply writes the whole Checked/Disabled matrix from s and reports
// whether any item actually moved, so internal/ui can rebuild the native
// menu bar only when there is something new to show.
//
// It recomputes every item on every call rather than trusting a caller to
// say what changed: the matrix is compact boolean arithmetic, and a caller
// that guesses wrong is exactly how a menu goes stale.
func (m *Menus) Apply(s State) (changed bool) {
	before := m.pairs()
	a := s.Availability
	m.open.Disabled = !a.Open
	m.save.Disabled = !a.Save
	m.export.Disabled = !a.Export
	m.closeFiles.Disabled = !a.CloseFiles
	m.settings.Disabled = !a.Settings
	m.window.viewer.Disabled = !a.Viewer
	m.window.explorer.Disabled = !a.Explorer
	m.window.locationMap.Disabled = !a.LocationMap
	m.window.locationMap.Checked = s.LocationMapActive
	m.window.mosaic.Disabled = !a.Mosaic
	m.window.exif.Disabled = !a.Exif
	m.window.grid.Disabled = !a.Grid
	m.window.pictureFrame.Disabled = !a.PictureFrame
	m.window.help.Disabled = !a.Help
	m.sortParent.Disabled = !a.Sort
	for i, mode := range filesort.Modes() {
		m.actions.sort[i].Checked = mode == s.SortMode
		m.actions.sort[i].Disabled = !a.Sort
	}
	m.actions.hide.Checked = s.HideDuplicates
	m.actions.hide.Disabled = !a.HideDuplicates
	m.actions.showVariant.Checked = s.BrowsingDuplicates
	m.actions.showVariant.Disabled = !a.BrowseDuplicates
	m.actions.compare.Disabled = !a.Compare
	m.actions.findMoreLikeThis.Disabled = !a.Search
	m.actions.rotate.Disabled = !a.Rotate
	m.actions.zoomIn.Disabled = !a.Zoom
	m.actions.zoomOut.Disabled = !a.Zoom
	m.actions.merge.Checked = s.MergeMode
	m.actions.merge.Disabled = !a.Merge
	m.actions.info.Checked = s.InfoVisible
	m.actions.info.Disabled = !a.Info
	m.actions.copy.Disabled = !a.Copy
	m.actions.copySelection.Disabled = !a.CopySelection
	m.actions.copyPath.Disabled = !a.CopyPath
	m.actions.reveal.Disabled = !a.Reveal
	m.actions.wallpaper.Disabled = !a.Wallpaper
	m.actions.trash.Disabled = !a.Trash
	return !slices.Equal(before, m.pairs())
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
	items := make([]*fyne.MenuItem, 0, len(m.actions.sort)+24)
	items = append(items, m.open, m.save, m.export, m.closeFiles, m.settings)
	items = append(items, m.window.viewer, m.window.explorer, m.window.locationMap, m.window.exif, m.window.grid,
		m.window.pictureFrame, m.window.help)
	items = append(items, m.sortParent)
	items = append(items, m.actions.sort...)
	items = append(items, m.actions.hide, m.actions.showVariant, m.actions.compare, m.actions.findMoreLikeThis, m.window.mosaic, m.actions.rotate,
		m.actions.zoomIn, m.actions.zoomOut, m.actions.merge, m.actions.info,
		m.actions.copy, m.actions.copySelection, m.actions.copyPath, m.actions.reveal,
		m.actions.wallpaper, m.actions.trash)

	out := make([]pair, len(items))
	for i, it := range items {
		out[i] = pair{checked: it.Checked, disabled: it.Disabled}
	}
	return out
}
