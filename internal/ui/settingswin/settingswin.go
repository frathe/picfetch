// Package settingswin is the Settings window, reachable from the File menu:
// one place to see and change every standing preference the app has - sort
// order, appearance, merge mode, picture-frame shuffle and interval, the
// folder-scan cap, the window-size cap, whether the window stays a fixed
// size, the three memory limits (image cache, thumbnail cache, maximum file
// size), and whether favorite previews are cached to disk - instead of only
// discovering them by stumbling onto their keyboard shortcuts.
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
	"github.com/frathe/picfetch/internal/preferences"
	"github.com/frathe/picfetch/internal/ui/widgets"
)

const (
	windowW = 520.0
	windowH = 520.0

	// time.Duration is an int64 nanosecond count. Reject a larger number of
	// seconds instead of letting the multiplication in build wrap negative
	// and get mistaken for the one-second minimum by ApplySettings.
	maxDurationSeconds = int64(1<<63-1) / int64(time.Second)

	// The memory limits are typed in megabytes and multiplied out to bytes
	// by ApplySettings, so reject anything that couldn't survive that shift -
	// same reasoning as maxDurationSeconds above. A terabyte is already far
	// past anything a machine could honour.
	maxMemoryMB = 1 << 20
)

// Host is what the settings window needs from the app after it has been
// seeded with a preferences.State snapshot: ApplySettings receives the form
// snapshot before and after the one control edit, so the app applies only
// the edited fields (patch semantics — a stale snapshot must never revert a
// preference a main-window shortcut changed while this window was open),
// and the two update verbs stay out of that snapshot because they are
// requests, not standing values.
type Host interface {
	ApplySettings(prev, next preferences.State)
	CheckForUpdatesNow(UpdateCallbacks)
	PerformUpdate() error
}

// Window is the settings panel. At most one is open at a time (widgets.
// Singleton): a second request raises the existing window rather than
// stacking up duplicates.
type Window struct {
	app                   fyne.App
	host                  Host
	updatesManagedByStore bool

	// prefs is the form snapshot Show seeded, mutated by each control, and
	// pushed back through Host.ApplySettings. Ignored while the window is
	// already open: a second Show raises rather than rebuilding.
	prefs preferences.State

	win widgets.Singleton

	// The controls themselves, live only while the window is open (nil
	// otherwise - the same pattern exifwin.Window's text field uses). Kept
	// as fields rather than locals inside build so this package's own tests
	// can drive them directly, the same way internal/ui/deletion's tests
	// drive that confirmation card's widgets.
	themeSelect, sortSelect       *widget.Select
	mergeCheck, shuffleCheck      *widget.Check
	favPreviewCheck, updateCheck  *widget.Check
	staticSizeCheck               *widget.Check
	updateNow                     *widget.Button
	updateVersion                 *widget.Label
	updateManaged                 *widget.Label
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

// New returns the settings window for application. Show seeds the form
// from a preferences.State snapshot; later edits go through host.
func New(application fyne.App, host Host) *Window {
	return &Window{app: application, host: host}
}

// Show opens the settings window, or raises it if it's already open.
// prefs is the standing-preferences snapshot used to seed the form; it is
// ignored when the window is already showing, so in-flight edits stay put.
func (w *Window) Show(prefs preferences.State, updatesManagedByStore bool) {
	if !w.win.Open() {
		w.prefs = prefs
		w.updatesManagedByStore = updatesManagedByStore
	}
	w.win.Show(w.app, lang.L("Settings"), fyne.NewSize(windowW, windowH), w.build, func() {
		w.closeUpdateFlow()
		w.themeSelect = nil
		w.sortSelect = nil
		w.mergeCheck, w.shuffleCheck = nil, nil
		w.favPreviewCheck, w.updateCheck = nil, nil
		w.staticSizeCheck = nil
		w.updateNow = nil
		w.updateVersion = nil
		w.updateManaged = nil
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
// current value back into ApplySettings. OnChanged ignores anything that isn't a
// positive int, and when max > 0 also anything above that ceiling — the same
// mid-edit "leave the last good value in the snapshot" behaviour the six copies had.
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

// apply mutates the seeded snapshot and pushes the before/after pair
// through the host, which applies only what actually changed between the
// two. Invalid mid-edit input never reaches here.
func (w *Window) apply(mutate func(*preferences.State)) {
	prev := w.prefs
	mutate(&w.prefs)
	w.host.ApplySettings(prev, w.prefs)
}

// selectFrom builds a Select whose options are displayName(modes[i]) and
// whose OnChanged maps the chosen label back to modes[i] via set. Selected
// is seeded from displayName(current) without SetSelected, matching
// newPositiveIntEntry's no-round-trip rule.
func selectFrom[T any](modes []T, displayName func(T) string, current T, set func(T)) *widget.Select {
	labels := make([]string, len(modes))
	for i, m := range modes {
		labels[i] = displayName(m)
	}
	sel := widget.NewSelect(labels, func(selected string) {
		for i, label := range labels {
			if label == selected {
				set(modes[i])
				return
			}
		}
	})
	sel.Selected = displayName(current)
	return sel
}

// build lays out every control, each one seeded from the snapshot Show
// stored and wired to push a change straight back through ApplySettings.
// Initial seeding sets the widgets' fields directly rather than through
// their own SetSelected/SetChecked/SetText - those fire OnChanged themselves,
// which would otherwise round-trip the freshly read value straight back
// into ApplySettings before the window has even been shown.
func (w *Window) build() fyne.CanvasObject {
	positiveInt := validation.NewRegexp(`^[1-9][0-9]*$`, lang.L("must be a positive whole number"))

	w.themeSelect = selectFrom(appearance.Modes(), appearance.DisplayName, w.prefs.ThemeMode, func(mode appearance.Mode) {
		w.apply(func(s *preferences.State) { s.ThemeMode = mode })
	})
	w.sortSelect = selectFrom(filesort.Modes(), filesort.DisplayName, filesort.FromPref(w.prefs.SortMode), func(mode filesort.Mode) {
		w.apply(func(s *preferences.State) { s.SortMode = mode.PrefValue() })
	})

	w.intervalEntry = widget.NewEntry()
	w.intervalEntry.Validator = positiveInt
	w.intervalEntry.Text = strconv.Itoa(int(w.prefs.SlideInterval.Seconds()))
	w.intervalEntry.OnChanged = func(s string) {
		if n, err := strconv.ParseInt(s, 10, 64); err == nil && n > 0 && n <= maxDurationSeconds {
			w.apply(func(p *preferences.State) { p.SlideInterval = time.Duration(n) * time.Second })
		}
	}

	w.maxScanEntry = newPositiveIntEntry(
		func() int { return w.prefs.MaxScanFiles },
		func(n int) { w.apply(func(s *preferences.State) { s.MaxScanFiles = n }) },
		0,
		positiveInt,
	)

	maxScanItem := widget.NewFormItem(lang.L("Max files per folder scan"), w.maxScanEntry)
	maxScanItem.HintText = lang.L("Caps how many images a single recursive folder scan will gather")

	w.maxWidthEntry = newPositiveIntEntry(
		func() int { return int(w.prefs.MaxWindowWidth) },
		func(n int) { w.apply(func(s *preferences.State) { s.MaxWindowWidth = float32(n) }) },
		0,
		positiveInt,
	)

	w.maxHeightEntry = newPositiveIntEntry(
		func() int { return int(w.prefs.MaxWindowHeight) },
		func(n int) { w.apply(func(s *preferences.State) { s.MaxWindowHeight = float32(n) }) },
		0,
		positiveInt,
	)

	w.imgCacheEntry = newPositiveIntEntry(
		func() int { return w.prefs.MaxImageCacheMB },
		func(n int) { w.apply(func(s *preferences.State) { s.MaxImageCacheMB = n }) },
		maxMemoryMB,
		positiveInt,
	)

	imgCacheItem := widget.NewFormItem(lang.L("Max image cache (MB)"), w.imgCacheEntry)
	imgCacheItem.HintText = lang.L("Memory kept for recently viewed images")

	w.thumbCacheEntry = newPositiveIntEntry(
		func() int { return w.prefs.MaxThumbCacheMB },
		func(n int) { w.apply(func(s *preferences.State) { s.MaxThumbCacheMB = n }) },
		maxMemoryMB,
		positiveInt,
	)

	thumbCacheItem := widget.NewFormItem(lang.L("Max thumbnail cache (MB)"), w.thumbCacheEntry)
	thumbCacheItem.HintText = lang.L("Memory kept for grid-view thumbnails")

	w.maxFileSizeEntry = newPositiveIntEntry(
		func() int { return w.prefs.MaxFileSizeMB },
		func(n int) { w.apply(func(s *preferences.State) { s.MaxFileSizeMB = n }) },
		maxMemoryMB,
		positiveInt,
	)

	maxFileSizeItem := widget.NewFormItem(lang.L("Max file size (MB)"), w.maxFileSizeEntry)
	maxFileSizeItem.HintText = lang.L("Larger files are not opened at all")

	dist := w.prefs.DuplicateDistance
	w.dupeDistValue = widget.NewLabel(strconv.Itoa(dist))
	w.dupeDistSlider = widget.NewSlider(0, 32)
	w.dupeDistSlider.Step = 1
	w.dupeDistSlider.Value = float64(dist)
	w.dupeDistSlider.OnChanged = func(v float64) {
		n := int(v)
		w.dupeDistValue.SetText(strconv.Itoa(n))
		w.apply(func(s *preferences.State) {
			s.DuplicateDistance = n
			s.DuplicateDistanceSet = true
		})
	}
	dupeDistItem := widget.NewFormItem(lang.L("Duplicate match distance"), container.NewBorder(nil, nil, nil, w.dupeDistValue, w.dupeDistSlider))
	dupeDistItem.HintText = lang.L("Lower is stricter; 0 is an exact thumbnail hash")

	generalForm := widget.NewForm(
		widget.NewFormItem(lang.L("Sort order"), w.sortSelect),
		widget.NewFormItem(lang.L("Picture-frame interval (seconds)"), w.intervalEntry),
		dupeDistItem,
	)
	windowSizeForm := widget.NewForm(
		widget.NewFormItem(lang.L("Max window width"), w.maxWidthEntry),
		widget.NewFormItem(lang.L("Max window height"), w.maxHeightEntry),
	)
	limitsForm := widget.NewForm(maxScanItem, imgCacheItem, thumbCacheItem, maxFileSizeItem)

	w.mergeCheck = widget.NewCheck(lang.L("Merge newly dropped files into the current set"), func(on bool) {
		w.apply(func(s *preferences.State) { s.MergeMode = on })
	})
	w.mergeCheck.Checked = w.prefs.MergeMode

	w.shuffleCheck = widget.NewCheck(lang.L("Shuffle picture-frame order"), func(on bool) {
		w.apply(func(s *preferences.State) { s.SlideShuffle = on })
	})
	w.shuffleCheck.Checked = w.prefs.SlideShuffle

	w.favPreviewCheck = widget.NewCheck(lang.L("Cache favorite previews on disk"), func(on bool) {
		w.apply(func(s *preferences.State) { s.FavoritePreviewCache = on })
	})
	w.favPreviewCheck.Checked = w.prefs.FavoritePreviewCache

	w.staticSizeCheck = widget.NewCheck(lang.L("Keep a fixed window size"), func(on bool) {
		w.apply(func(s *preferences.State) { s.StaticWindowSize = on })
	})
	w.staticSizeCheck.Checked = w.prefs.StaticWindowSize

	meta := w.app.Metadata()
	w.updateVersion = widget.NewLabel(fmt.Sprintf(lang.L("Version %s (Build %d)"), meta.Version, meta.Build))

	general := container.NewVBox(generalForm, widget.NewSeparator(), w.mergeCheck, w.shuffleCheck, w.favPreviewCheck)
	appearanceSettings := container.NewVBox(w.themeSelect, widget.NewSeparator(), windowSizeForm, w.staticSizeCheck)
	updates := container.NewVBox(w.updateVersion)
	if w.updatesManagedByStore {
		w.updateManaged = widget.NewLabel(lang.L("Updates are managed by Microsoft Store."))
		w.updateManaged.Wrapping = fyne.TextWrapWord
		updates.Add(w.updateManaged)
	} else {
		w.updateCheck = widget.NewCheck(lang.L("Check for updates"), func(on bool) {
			w.apply(func(s *preferences.State) { s.CheckForUpdates = on })
		})
		w.updateCheck.Checked = w.prefs.CheckForUpdates
		w.updateNow = widget.NewButton(lang.L("Check now"), w.startUpdateCheck)
		updates.Add(w.updateCheck)
		updates.Add(w.updateNow)
	}

	return container.NewAppTabs(
		container.NewTabItem(lang.L("General"), container.NewPadded(container.NewVScroll(general))),
		container.NewTabItem(lang.L("Appearance"), container.NewPadded(container.NewVScroll(appearanceSettings))),
		container.NewTabItem(lang.L("Updates"), container.NewPadded(container.NewVScroll(updates))),
		container.NewTabItem(lang.L("Limits"), container.NewPadded(container.NewVScroll(limitsForm))),
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
