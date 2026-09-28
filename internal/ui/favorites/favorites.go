// Package favorites owns the Favorites menu and its dialogs.
package favorites

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/driver/desktop"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/favstore"
)

// ShortcutCount is how many sorted favorites can be opened by keyboard.
const ShortcutCount = 10

var shortcutKeys = [...]fyne.KeyName{
	fyne.Key1,
	fyne.Key2,
	fyne.Key3,
	fyne.Key4,
	fyne.Key5,
	fyne.Key6,
	fyne.Key7,
	fyne.Key8,
	fyne.Key9,
	fyne.Key0,
}

// Command identifies a fresh Favorite intent without exposing root UI policy.
type Command uint8

const (
	OpenCommand Command = iota
	AddCommand
	ManageCommand
)

// Host is the viewer behavior used by the favorites feature.
type Host interface {
	CurrentFiles() []fyne.URI
	OpenFavorite(owner *favstore.Owner, files []fyne.URI)
	ShowToast(msg string)

	// SyncFavoritePreviews brings the previews stored under owner in line
	// with files, in the background. This feature knows nothing about
	// thumbnails or caches; it only reports that a favorite's file list is
	// now this, and leaves what that costs to the host.
	SyncFavoritePreviews(owner *favstore.Owner, files []fyne.URI)

	// RefreshMenus re-publishes the main menu bar. This feature calls it
	// after changing its own menu's items, because fyne.Menu.Refresh is
	// SetMainMenu underneath: on Darwin that rebuilds the native bar, and
	// only the host knows how to fold it back together afterwards.
	RefreshMenus()

	// AdmitFavorite checks admission before storage, capture or dialog entry.
	AdmitFavorite(Command) bool
}

// Feature owns the Favorites menu and its dialogs.
type Feature struct {
	addFiles        []fyne.URI
	onDialogClosed  func()
	onDialogChanged func()
	onSaved         func()
	host            Host
	win             fyne.Window
	dir             string
	storage         Storage
	ui              UIQueue
	entries         []favstore.Entry
	menuDir         string
	viewCtx         context.Context
	viewCancel      context.CancelFunc
	openCancel      context.CancelFunc
	refreshCancel   context.CancelFunc
	refreshRevision uint64
	refreshRunning  bool
	refreshPending  bool
	manageRequested bool
	stopped         bool
	confirmDialog   dialog.Dialog
	mutationTail    <-chan struct{}
	saveRevision    uint64
	removeCancel    context.CancelFunc
	removeDialog    dialog.Dialog

	menu         *fyne.Menu
	addItem      *fyne.MenuItem
	manageItem   *fyne.MenuItem
	names        []string
	availability Availability

	// manageDialog and managePanel are the Manage Favorites dialog while it
	// is up, and nil whenever it is not - see manage.go, where a non-nil
	// manageDialog doubles as the guard against stacking a second one.
	manageDialog dialog.Dialog
	managePanel  *managePanel

	// addDialog and addPanel are the Add to Favorites dialog while it is up,
	// nil whenever it is not - see add.go, where a non-nil addDialog is the
	// same kind of guard against stacking a second one.
	addDialog dialog.Dialog
	addPanel  *addPanel

	workers sync.WaitGroup
}

// New captures the selected root and builds the menu without reading from disk.
func New(host Host, win fyne.Window, dir string) *Feature {
	f := &Feature{host: host, win: win, dir: dir, availability: Availability{Open: true, Manage: true}, storage: &favstore.Store{}, ui: fyneQueue{}}
	f.addItem = fyne.NewMenuItem(lang.L("Add Current List to Favorites…"), f.AddCurrentList)
	f.addItem.Disabled = true
	// Display-only, mirroring Manage Favorites… below: the binding itself
	// is wireAddFavoritesShortcut's AddShortcut call
	// (internal/ui/shortcuts.go). This just shows the same accelerator next
	// to the menu item.
	f.addItem.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyF,
		Modifier: fyne.KeyModifierAlt | fyne.KeyModifierShift,
	}
	f.manageItem = fyne.NewMenuItem(lang.L("Manage Favorites…"), f.ShowManage)
	// Display-only, mirroring how internal/ui/menu.go sets Export image's
	// and Actions' Set as Wallpaper Shortcut fields: the binding itself is
	// wireManageFavoritesShortcut's AddShortcut call
	// (internal/ui/shortcuts.go). This just shows the same accelerator next
	// to the menu item.
	f.manageItem.Shortcut = &desktop.CustomShortcut{
		KeyName:  fyne.KeyF,
		Modifier: fyne.KeyModifierShortcutDefault | fyne.KeyModifierShift,
	}
	f.menu = fyne.NewMenu(lang.L("Favorites"),
		f.addItem, fyne.NewMenuItemSeparator(), f.manageItem)
	return f
}

// Menu returns the feature's top-level menu.
func (f *Feature) Menu() *fyne.Menu {
	return f.menu
}

// Dir returns the configured favorites storage root.
func (f *Feature) Dir() string { return f.dir }

// SetDir selects the storage directory and populates the menu from it.
func (f *Feature) SetDir(dir string) {
	if f.stopped {
		return
	}
	f.Close()
	f.dir = dir
	f.refreshMenu()
}

// Availability is supplied by the host's shared command policy.
type Availability struct{ Open, Add, Manage bool }

// SetAvailability updates items without publishing the main menu. Root owns
// its single refresh after applying all feature decisions.
func (f *Feature) SetAvailability(availability Availability) bool {
	if !availability.Open {
		f.CancelOpen()
	}
	changed := f.availability != availability
	f.availability = availability
	f.syncCommandAvailability()
	return changed
}

func (f *Feature) syncCommandAvailability() {
	for _, item := range f.menu.Items {
		if item.IsSeparator {
			continue
		}
		item.Disabled = !f.availability.Open
	}
	f.addItem.Disabled = !f.availability.Add
	f.manageItem.Disabled = !f.availability.Manage
}

// ShortcutForIndex returns the Cmd/Ctrl+digit accelerator for a zero-based
// favorite index: 1 through 9, then 0 for the tenth.
func ShortcutForIndex(index int) *desktop.CustomShortcut {
	if index < 0 || index >= len(shortcutKeys) {
		return nil
	}
	return &desktop.CustomShortcut{
		KeyName:  shortcutKeys[index],
		Modifier: fyne.KeyModifierShortcutDefault,
	}
}

// Open opens the favorite currently assigned to a zero-based shortcut slot.
func (f *Feature) Open(index int) {
	if f.menuDir != f.dir || index < 0 || index >= ShortcutCount || index >= len(f.names) {
		return
	}
	f.openFavorite(f.names[index])
}

func (f *Feature) installMenu(entries []favstore.Entry) {
	f.entries = entries
	f.menuDir = f.dir
	menuDir := f.dir
	names := make([]string, len(entries))
	for i, entry := range entries {
		names[i] = entry.Name
	}

	items := []*fyne.MenuItem{f.addItem, fyne.NewMenuItemSeparator()}
	for i, name := range names {
		favoriteName := name
		item := fyne.NewMenuItem(f.menuLabel(favoriteName), func() {
			if f.dir == menuDir {
				f.openFavorite(favoriteName)
			}
		})
		if shortcut := ShortcutForIndex(i); shortcut != nil {
			item.Shortcut = shortcut
		}
		items = append(items, item)
	}
	if len(names) > 0 {
		items = append(items, fyne.NewMenuItemSeparator())
	}
	items = append(items, f.manageItem)
	f.names = names
	f.menu.Items = items
	f.syncCommandAvailability()
	f.host.RefreshMenus()
}

// menuLabel uses only the last complete worker snapshot. Unknown counts keep
// the bare name and shortcut slot without producing a toast per invalid list.
func (f *Feature) menuLabel(name string) string {
	for _, entry := range f.entries {
		if entry.Name == name && entry.CountErr == nil {
			return fmt.Sprintf(lang.L("%s (%d)"), name, entry.Count)
		}
	}
	return name
}

// AddCurrentList is the Favorites menu's own "Add Current List to
// Favorites…" item and the Opt/Alt+Shift+F binding - always a fresh, empty
// dialog; showAdd's initial parameter exists for Stage 5's Replace-Cancel,
// which reopens with the name that just clashed still in the field.
func (f *Feature) AddCurrentList() {
	if f.stopped || !f.host.AdmitFavorite(AddCommand) {
		return
	}
	f.addFilesPrompt(f.host.CurrentFiles())
}

func (f *Feature) reportError(format string, args ...any) {
	message := fmt.Sprintf(format, args...)
	fyne.LogError("favorites operation failed", errors.New(message))
	f.host.ShowToast(message)
}

// AddFiles opens naming for an explicit list.
func (f *Feature) AddFiles(files []fyne.URI) {
	if f.stopped || !f.host.AdmitFavorite(AddCommand) {
		return
	}
	f.addFilesPrompt(files)
}

func (f *Feature) addFilesPrompt(files []fyne.URI) {
	if f.addDialog != nil {
		return
	}
	f.saveRevision++
	f.addFiles = append([]fyne.URI{}, files...)
	f.showAdd("")
}

// SetOnDialogClosed observes return to the main browsing surface.
func (f *Feature) SetOnDialogClosed(closed func()) { f.onDialogClosed = closed }

// SetOnDialogChanged reports modal ownership changes without publishing menus.
func (f *Feature) SetOnDialogChanged(changed func()) { f.onDialogChanged = changed }
func (f *Feature) dialogChanged() {
	if f.onDialogChanged != nil {
		f.onDialogChanged()
	}
}

// SetOnSaved observes a successfully committed list, before menu refresh.
func (f *Feature) SetOnSaved(saved func()) { f.onSaved = saved }
