// Package settingswin is the Settings window, reachable from the File menu:
// one place to see and change every standing preference the app has - sort
// order, appearance, merge mode, picture-frame shuffle and interval, the
// folder-scan cap, the window-size cap, the three memory limits (image cache,
// thumbnail cache, maximum file size), and whether favorite previews are
// cached to disk - instead of only discovering them by stumbling onto their
// keyboard shortcuts.
//
// Every control applies live, through its own OnChanged, the same
// immediate-effect behavior the S/M/Shift+P keys already give their own
// preferences - there is no separate Save/Apply step and so nothing here
// needs to track a "dirty" draft state.
package settingswin

import (
	"fmt"
	"strconv"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/data/validation"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/appearance"
	"github.com/frathe/picfetch/internal/filesort"
	"github.com/frathe/picfetch/internal/ui/widgets"
)

const (
	windowW = 520.0
	windowH = 520.0

	// time.Duration is an int64 nanosecond count. Reject a larger number of
	// seconds instead of letting the multiplication in build wrap negative
	// and get mistaken for the one-second minimum by the host.
	maxDurationSeconds = int64(1<<63-1) / int64(time.Second)

	// The memory limits are typed in megabytes and multiplied out to bytes
	// by the host, so reject anything that couldn't survive that shift -
	// same reasoning as maxDurationSeconds above. A terabyte is already far
	// past anything a machine could honour.
	maxMemoryMB = 1 << 20
)

// Host is what the settings window needs from the app: read/write access to
// every standing preference it exposes. Every setter is expected to apply
// its change immediately, the same as the keyboard shortcut that already
// exists for it (where one exists) - the window itself holds no state of
// its own to reconcile later.
type Host interface {
	ThemeMode() appearance.Mode
	SetThemeMode(appearance.Mode)

	SortMode() filesort.Mode
	SetSortMode(filesort.Mode)

	MergeMode() bool
	SetMergeMode(bool)

	SlideShuffle() bool
	SetSlideShuffle(bool)

	SlideInterval() time.Duration
	SetSlideInterval(time.Duration)

	MaxScan() int
	SetMaxScan(int)

	MaxWindowWidth() float32
	SetMaxWindowWidth(float32)

	MaxWindowHeight() float32
	SetMaxWindowHeight(float32)

	MaxImageCacheMB() int
	SetMaxImageCacheMB(int)

	MaxThumbCacheMB() int
	SetMaxThumbCacheMB(int)

	MaxFileSizeMB() int
	SetMaxFileSizeMB(int)

	FavoritePreviewCache() bool
	SetFavoritePreviewCache(bool)

	CheckForUpdates() bool
	SetCheckForUpdates(bool)
	CheckForUpdatesNow(UpdateCallbacks)
	PerformUpdate() error

	DuplicateDistance() int
	SetDuplicateDistance(int)
}

// Window is the settings panel. At most one is open at a time (widgets.
// Singleton): a second request raises the existing window rather than
// stacking up duplicates.
type Window struct {
	app  fyne.App
	host Host

	win widgets.Singleton

	// The controls themselves, live only while the window is open (nil
	// otherwise - the same pattern exifwin.Window's text field uses). Kept
	// as fields rather than locals inside build so this package's own tests
	// can drive them directly, the same way internal/ui/deletion's tests
	// drive that confirmation card's widgets.
	themeSelect, sortSelect       *widget.Select
	mergeCheck, shuffleCheck      *widget.Check
	favPreviewCheck, updateCheck  *widget.Check
	updateNow                     *widget.Button
	intervalEntry, maxScanEntry   *widget.Entry
	maxWidthEntry, maxHeightEntry *widget.Entry
	imgCacheEntry, thumbCacheEntry,
	maxFileSizeEntry *widget.Entry
	dupeDistSlider *widget.Slider
	dupeDistValue  *widget.Label

	// updateFlow identifies the currently live manual-update request. Host
	// callbacks are already delivered on the UI thread, so this narrow
	// monotonically increasing value is enough to reject callbacks from a
	// closed Settings window or a superseded request without a lock.
	updateFlow       uint64
	updateActive     bool
	updatePerforming bool

	// The current update dialog and its visible controls are held so package
	// tests can assert the state machine without reaching through Fyne's
	// overlay internals. They are replaced together on every phase change.
	updateDialog   dialog.Dialog
	updateMessage  *widget.Label
	updateProgress *widget.ProgressBar
	updateInfinite *widget.ProgressBarInfinite
	updateChoices  *widgets.ChoicePanel
}

// New returns the settings window for application, reading and writing its
// preferences through host.
func New(application fyne.App, host Host) *Window {
	return &Window{app: application, host: host}
}

// Show opens the settings window, or raises it if it's already open.
func (w *Window) Show() {
	w.win.Show(w.app, lang.L("Settings"), fyne.NewSize(windowW, windowH), w.build, func() {
		w.closeUpdateFlow()
		w.themeSelect = nil
		w.sortSelect = nil
		w.mergeCheck, w.shuffleCheck = nil, nil
		w.favPreviewCheck, w.updateCheck = nil, nil
		w.updateNow = nil
		w.intervalEntry, w.maxScanEntry = nil, nil
		w.maxWidthEntry, w.maxHeightEntry = nil, nil
		w.imgCacheEntry, w.thumbCacheEntry, w.maxFileSizeEntry = nil, nil, nil
		w.dupeDistSlider, w.dupeDistValue = nil, nil
	})
}

// Open reports whether the settings window is currently showing.
func (w *Window) Open() bool {
	return w.win.Open()
}

// RestoreGeometry makes the window remember where and how large it was,
// seeded with what the last run left it at. Called once during internal/ui's
// startup restoration; the app reads the current values back out of
// Geometry at shutdown. Without it the window opens at windowW x windowH
// wherever the OS puts it, which is what it always did.
func (w *Window) RestoreGeometry(g widgets.Geometry) {
	w.win.Remember(g)
}

// Geometry is where the window currently is and how large - or where it was
// last, since it outlives the window being closed. What internal/ui hands
// preferences.Save at shutdown.
func (w *Window) Geometry() widgets.Geometry {
	return w.win.Geometry()
}

// StopTracking stops following the window's position, for a shutdown that
// finds it still open - see widgets.Singleton.StopTracking.
func (w *Window) StopTracking() {
	w.win.StopTracking()
}

// newPositiveIntEntry is the numeric form field used by every standing integer
// preference except the picture-frame interval (that one is an int64-second
// count that has to survive a Duration multiply). Text is seeded from get
// without going through SetText, so opening the window does not round-trip the
// current value back into the host. OnChanged ignores anything that isn't a
// positive int, and when max > 0 also anything above that ceiling — the same
// mid-edit "leave the last good value in the host" behaviour the six copies had.
func newPositiveIntEntry(get func() int, set func(int), max int, validate fyne.StringValidator) *widget.Entry {
	e := widget.NewEntry()
	e.Validator = validate
	e.Text = strconv.Itoa(get())
	e.OnChanged = func(s string) {
		if n, err := strconv.Atoi(s); err == nil && n > 0 && (max <= 0 || n <= max) {
			set(n)
		}
	}
	return e
}

// build lays out every control, each one seeded from the host's current
// value and wired to push a change straight back through it. Initial
// seeding sets the widgets' fields directly rather than through their own
// SetSelected/SetChecked/SetText - those fire OnChanged themselves, which
// would otherwise round-trip the freshly read value straight back into the
// host before the window has even been shown.
func (w *Window) build() fyne.CanvasObject {
	positiveInt := validation.NewRegexp(`^[1-9][0-9]*$`, lang.L("must be a positive whole number"))
	themeModes := appearance.Modes()
	themeLabels := make([]string, len(themeModes))
	for i, mode := range themeModes {
		themeLabels[i] = appearance.DisplayName(mode)
	}

	w.themeSelect = widget.NewSelect(themeLabels, func(selected string) {
		for i, label := range themeLabels {
			if label == selected {
				w.host.SetThemeMode(themeModes[i])
				return
			}
		}
	})
	w.themeSelect.Selected = appearance.DisplayName(w.host.ThemeMode())

	modes := filesort.Modes()
	labels := make([]string, len(modes))
	for i, m := range modes {
		labels[i] = filesort.DisplayName(m)
	}

	w.sortSelect = widget.NewSelect(labels, func(s string) {
		for i, l := range labels {
			if l == s {
				w.host.SetSortMode(modes[i])
				return
			}
		}
	})
	w.sortSelect.Selected = filesort.DisplayName(w.host.SortMode())

	w.intervalEntry = widget.NewEntry()
	w.intervalEntry.Validator = positiveInt
	w.intervalEntry.Text = strconv.Itoa(int(w.host.SlideInterval().Seconds()))
	w.intervalEntry.OnChanged = func(s string) {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 && n <= maxDurationSeconds {
			w.host.SetSlideInterval(time.Duration(n) * time.Second)
		}
	}

	w.maxScanEntry = newPositiveIntEntry(w.host.MaxScan, w.host.SetMaxScan, 0, positiveInt)

	maxScanItem := widget.NewFormItem(lang.L("Max files per folder scan"), w.maxScanEntry)
	maxScanItem.HintText = lang.L("Caps how many images a single recursive folder scan will gather")

	w.maxWidthEntry = newPositiveIntEntry(
		func() int { return int(w.host.MaxWindowWidth()) },
		func(n int) { w.host.SetMaxWindowWidth(float32(n)) },
		0,
		positiveInt,
	)

	w.maxHeightEntry = newPositiveIntEntry(
		func() int { return int(w.host.MaxWindowHeight()) },
		func(n int) { w.host.SetMaxWindowHeight(float32(n)) },
		0,
		positiveInt,
	)

	w.imgCacheEntry = newPositiveIntEntry(w.host.MaxImageCacheMB, w.host.SetMaxImageCacheMB, maxMemoryMB, positiveInt)

	imgCacheItem := widget.NewFormItem(lang.L("Max image cache (MB)"), w.imgCacheEntry)
	imgCacheItem.HintText = lang.L("Memory kept for recently viewed images")

	w.thumbCacheEntry = newPositiveIntEntry(w.host.MaxThumbCacheMB, w.host.SetMaxThumbCacheMB, maxMemoryMB, positiveInt)

	thumbCacheItem := widget.NewFormItem(lang.L("Max thumbnail cache (MB)"), w.thumbCacheEntry)
	thumbCacheItem.HintText = lang.L("Memory kept for grid-view thumbnails")

	w.maxFileSizeEntry = newPositiveIntEntry(w.host.MaxFileSizeMB, w.host.SetMaxFileSizeMB, maxMemoryMB, positiveInt)

	maxFileSizeItem := widget.NewFormItem(lang.L("Max file size (MB)"), w.maxFileSizeEntry)
	maxFileSizeItem.HintText = lang.L("Larger files are not opened at all")

	dist := w.host.DuplicateDistance()
	w.dupeDistValue = widget.NewLabel(strconv.Itoa(dist))
	w.dupeDistSlider = widget.NewSlider(0, 32)
	w.dupeDistSlider.Step = 1
	w.dupeDistSlider.Value = float64(dist)
	w.dupeDistSlider.OnChanged = func(v float64) {
		n := int(v)
		w.dupeDistValue.SetText(strconv.Itoa(n))
		w.host.SetDuplicateDistance(n)
	}
	dupeDistItem := widget.NewFormItem(lang.L("Duplicate match distance"), container.NewBorder(nil, nil, nil, w.dupeDistValue, w.dupeDistSlider))
	dupeDistItem.HintText = lang.L("Lower is stricter; 0 is an exact thumbnail hash")

	generalForm := widget.NewForm(
		widget.NewFormItem(lang.L("Sort order"), w.sortSelect),
		widget.NewFormItem(lang.L("Picture-frame interval (seconds)"), w.intervalEntry),
		maxScanItem,
		widget.NewFormItem(lang.L("Max window width"), w.maxWidthEntry),
		widget.NewFormItem(lang.L("Max window height"), w.maxHeightEntry),
		imgCacheItem,
		thumbCacheItem,
		maxFileSizeItem,
		dupeDistItem,
	)

	w.mergeCheck = widget.NewCheck(lang.L("Merge newly dropped files into the current set"), w.host.SetMergeMode)
	w.mergeCheck.Checked = w.host.MergeMode()

	w.shuffleCheck = widget.NewCheck(lang.L("Shuffle picture-frame order"), w.host.SetSlideShuffle)
	w.shuffleCheck.Checked = w.host.SlideShuffle()

	w.favPreviewCheck = widget.NewCheck(lang.L("Cache favorite previews on disk"), w.host.SetFavoritePreviewCache)
	w.favPreviewCheck.Checked = w.host.FavoritePreviewCache()

	w.updateCheck = widget.NewCheck(lang.L("Check for updates"), w.host.SetCheckForUpdates)
	w.updateCheck.Checked = w.host.CheckForUpdates()
	w.updateNow = widget.NewButton(lang.L("Check now"), w.startUpdateCheck)

	general := container.NewVBox(generalForm, widget.NewSeparator(), w.mergeCheck, w.shuffleCheck, w.favPreviewCheck)
	appearanceSettings := container.NewVBox(w.themeSelect)
	updates := container.NewVBox(w.updateCheck, w.updateNow)

	return container.NewAppTabs(
		container.NewTabItem(lang.L("General"), container.NewPadded(container.NewVScroll(general))),
		container.NewTabItem(lang.L("Appearance"), container.NewPadded(container.NewVScroll(appearanceSettings))),
		container.NewTabItem(lang.L("Updates"), container.NewPadded(container.NewVScroll(updates))),
	)
}

// startUpdateCheck owns the Settings window's one manual request. Disabling
// both update controls makes a rapid double tap harmless and stops a setting
// toggle from starting a competing automatic request while this UI flow is
// visible.
func (w *Window) startUpdateCheck() {
	if w.updateActive || w.updateNow == nil || w.win.Window() == nil {
		return
	}

	w.updateFlow++
	flow := w.updateFlow
	w.updateActive = true
	w.updatePerforming = false
	w.setUpdateControlsEnabled(false)
	w.showChecking(flow)

	w.host.CheckForUpdatesNow(UpdateCallbacks{
		Downloading: func(version string) {
			if w.flowActive(flow) {
				w.showDownloading(flow, version)
			}
		},
		Progress: func(downloaded, total int64) {
			if w.flowActive(flow) {
				w.showDownloadProgress(flow, downloaded, total)
			}
		},
		Current: func() {
			if w.flowActive(flow) {
				w.showCurrent(flow)
			}
		},
		Ready: func(version string) {
			if w.flowActive(flow) {
				w.showReady(flow, version)
			}
		},
		Failed: func(err error) {
			if w.flowActive(flow) {
				w.showFailed(flow, err)
			}
		},
	})
}

func (w *Window) flowActive(flow uint64) bool {
	return w.updateActive && w.updateFlow == flow && w.win.Window() != nil
}

func (w *Window) setUpdateControlsEnabled(enabled bool) {
	for _, control := range []fyne.Disableable{w.updateNow, w.updateCheck} {
		if control == nil {
			continue
		}
		if enabled {
			control.Enable()
		} else {
			control.Disable()
		}
	}
}

// finishUpdateFlow admits the controls for a later manual check and makes
// every later callback from this request stale. The terminal dialog stays up
// independently, which is safe because it is modal and all callback paths
// first test flowActive.
func (w *Window) finishUpdateFlow(flow uint64) {
	if w.updateFlow != flow {
		return
	}
	w.updateActive = false
	w.updatePerforming = false
	w.setUpdateControlsEnabled(true)
}

// closeUpdateFlow is the Settings-window teardown boundary. A completed
// backend stage deliberately remains available, but it no longer has any
// live widgets to update.
func (w *Window) closeUpdateFlow() {
	w.hideUpdateDialog()
	w.updateActive = false
	w.updatePerforming = false
	w.updateFlow++
	w.updateMessage = nil
	w.updateProgress = nil
	w.updateInfinite = nil
	w.updateChoices = nil
}

func (w *Window) showChecking(flow uint64) {
	if !w.flowActive(flow) {
		return
	}

	w.updateMessage = widget.NewLabel(lang.L("Checking for updates…"))
	w.updateMessage.Alignment = fyne.TextAlignCenter
	w.updateProgress = nil
	w.updateInfinite = widget.NewProgressBarInfinite()
	w.updateChoices = nil
	w.showUpdateDialog(container.NewVBox(w.updateMessage, w.updateInfinite))
}

func (w *Window) showDownloading(flow uint64, version string) {
	if !w.flowActive(flow) {
		return
	}

	w.updateMessage = widget.NewLabel(fmt.Sprintf(lang.L("Downloading version %s"), version))
	w.updateMessage.Alignment = fyne.TextAlignCenter
	w.updateProgress = widget.NewProgressBar()
	w.updateProgress.Hide()
	w.updateInfinite = widget.NewProgressBarInfinite()
	w.updateChoices = nil
	w.showUpdateDialog(container.NewVBox(w.updateMessage, w.updateProgress, w.updateInfinite))
}

func (w *Window) showDownloadProgress(flow uint64, downloaded, total int64) {
	if !w.flowActive(flow) || w.updateProgress == nil || w.updateInfinite == nil {
		return
	}

	if total <= 0 {
		w.updateProgress.Hide()
		w.updateInfinite.Show()
		return
	}

	value := float64(downloaded) / float64(total)
	if value < 0 {
		value = 0
	} else if value > 1 {
		value = 1
	}
	w.updateInfinite.Hide()
	w.updateProgress.Show()
	w.updateProgress.SetValue(value)
}

func (w *Window) showCurrent(flow uint64) {
	if !w.flowActive(flow) {
		return
	}
	w.finishUpdateFlow(flow)
	w.showTerminalMessage(lang.L("You are on the current version."), func() {})
}

func (w *Window) showFailed(flow uint64, err error) {
	if !w.flowActive(flow) {
		return
	}
	w.finishUpdateFlow(flow)
	w.showTerminalMessage(fmt.Sprintf(lang.L("Could not check for updates: %v"), err), func() {})
}

func (w *Window) showReady(flow uint64, _ string) {
	if !w.flowActive(flow) {
		return
	}

	w.hideUpdateDialog()
	w.updateMessage = widget.NewLabel(lang.L("Update downloaded successfully."))
	w.updateMessage.Alignment = fyne.TextAlignCenter
	w.updateProgress, w.updateInfinite = nil, nil
	w.updateChoices = widgets.NewChoicePanel(nil,
		widgets.Choice{Label: lang.L("Later"), OnChosen: func() { w.finishUpdateFlow(flow) }},
		widgets.Choice{Label: lang.L("Perform update"), OnChosen: func() { w.performUpdate(flow) }},
	)
	w.updateChoices.SetOnDismiss(func() { w.hideUpdateDialog() })
	w.updateChoices.SetOnCancel(func() { w.finishUpdateFlow(flow) })
	w.showUpdateDialog(container.NewVBox(w.updateMessage, w.updateChoices))
}

func (w *Window) performUpdate(flow uint64) {
	if !w.flowActive(flow) || w.updatePerforming {
		return
	}
	// ChoicePanel hides before running this action. Keep the flow active until
	// the host accepts the request so a second callback or tap cannot begin a
	// competing action in the small failure-recovery window.
	w.updatePerforming = true
	if err := w.host.PerformUpdate(); err != nil {
		fyne.LogError("perform update failed", err)
		w.updatePerforming = false
		w.finishUpdateFlow(flow)
		w.showTerminalMessage(fmt.Sprintf(lang.L("Could not perform update: %v"), err), func() {})
	}
}

// showTerminalMessage uses ChoicePanel even for the single OK choice so the
// modal owns keyboard focus and Return/Escape cannot leak into Settings.
func (w *Window) showTerminalMessage(message string, onOK func()) {
	w.hideUpdateDialog()
	w.updateMessage = widget.NewLabel(message)
	w.updateMessage.Alignment = fyne.TextAlignCenter
	w.updateProgress, w.updateInfinite = nil, nil
	w.updateChoices = widgets.NewChoicePanel(nil, widgets.Choice{Label: lang.L("OK"), OnChosen: onOK})
	w.updateChoices.SetOnDismiss(func() { w.hideUpdateDialog() })
	w.updateChoices.SetOnCancel(onOK)
	w.showUpdateDialog(container.NewVBox(w.updateMessage, w.updateChoices))
}

// showUpdateDialog replaces the previous phase's modal and parents the new
// one to the Settings window. No callback reaches here after close because
// flowActive protects all asynchronous entries; terminal paths additionally
// tolerate a user closing Settings between phase changes.
func (w *Window) showUpdateDialog(content fyne.CanvasObject) {
	win := w.win.Window()
	if win == nil {
		return
	}

	w.hideUpdateDialog()
	var d dialog.Dialog
	d = dialog.NewCustomWithoutButtons(lang.L("Software Update"), content, win)
	d.SetOnClosed(func() {
		if w.updateDialog == d {
			w.updateDialog = nil
		}
	})
	w.updateDialog = d
	d.Show()
	if w.updateChoices != nil {
		win.Canvas().Focus(w.updateChoices)
	}
}

func (w *Window) hideUpdateDialog() {
	d := w.updateDialog
	w.updateDialog = nil
	if d != nil {
		d.Hide()
	}
}
