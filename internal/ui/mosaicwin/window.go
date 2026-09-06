// Package mosaicwin owns the secondary image-mosaic configuration, preview,
// lifecycle, export, and wallpaper workflow.
package mosaicwin

import (
	"context"
	"errors"
	"fmt"
	"image"
	"slices"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/displays"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/mosaic"
	"github.com/frathe/picfetch/internal/ui/widgets"
	"github.com/frathe/picfetch/internal/wallpaper"
)

const (
	windowWidth  = 760
	windowHeight = 620
)

type SourceKind string

const (
	SourceSelection SourceKind = "selection"
	SourceResult    SourceKind = "result"
)

// Snapshot freezes the source pool and attached displays at command entry.
type Snapshot struct {
	Sources  []fyne.URI
	Kind     SourceKind
	Displays displays.Snapshot
}

// NewSnapshot validates and defensively copies the command-entry state.
func NewSnapshot(sources []fyne.URI, kind SourceKind, topology displays.Snapshot) (Snapshot, error) {
	if len(sources) == 0 {
		return Snapshot{}, fmt.Errorf("mosaic window requires at least one source")
	}
	for index, source := range sources {
		if source == nil {
			return Snapshot{}, fmt.Errorf("mosaic source %d is nil", index)
		}
	}
	if kind != SourceSelection && kind != SourceResult {
		return Snapshot{}, fmt.Errorf("unknown mosaic source kind %q", kind)
	}
	if len(topology.Displays) == 0 || topology.Default == "" {
		return Snapshot{}, fmt.Errorf("mosaic window requires an attached target display")
	}

	return cloneSnapshot(Snapshot{Sources: sources, Kind: kind, Displays: topology}), nil
}

func cloneSnapshot(snapshot Snapshot) Snapshot {
	snapshot.Sources = slices.Clone(snapshot.Sources)
	snapshot.Displays.Displays = slices.Clone(snapshot.Displays.Displays)

	return snapshot
}

// Host is the narrow set of cross-feature effects owned by internal/ui.
type Host interface {
	GenerateMosaic(context.Context, mosaic.Request) (mosaic.Result, error)
	InspectMosaicDisplays() (displays.Snapshot, error)
	// SetMosaicWallpaper's solo argument confirms the target is currently the
	// only attached display - see wallpaper.Request.Solo for why that lets a
	// platform that can't truthfully address one display among several
	// honor it anyway.
	SetMosaicWallpaper(ctx context.Context, result mosaic.Result, target displays.ID, solo bool) error
}

// Window is the one secondary mosaic workflow window.
type Window struct {
	app  fyne.App
	host Host
	win  widgets.Singleton

	snapshot       Snapshot
	settings       mosaic.Settings
	target         displays.ID
	result         mosaic.Result
	hasResult      bool
	generationBusy bool
	actionBusy     bool
	statusText     string

	lifecycle       revisionLifecycle
	actionLifecycle revisionLifecycle
	workers         workTracker
	ui              UIQueue
	seed            func() int64
	lastSeed        int64
	exportFormat    ExportFormat
	clock           func() time.Time
	exporter        func(fyne.URI, image.Image, fyne.URI, imaging.ExportOptions) error

	root, config, previewPanel   *fyne.Container
	sourceLabel, status          *widget.Label
	previewStatus                *widget.Label
	displaySelect, frameSelect   *namedSelect
	formatSelect                 *namedSelect
	minimum, variation           *namedSlider
	overlap, rotation            *namedSlider
	dropShadow                   *namedCheck
	minimumValue, variationValue *widget.Label
	overlapValue, rotationValue  *widget.Label
	loading                      *widget.ProgressBarInfinite
	advancedButton               *actionButton
	advancedControls             *fyne.Container
	refreshButton                *actionButton
	generateButton, cancelButton *actionButton
	startOverButton, regenerateButton, wallpaperButton,
	saveButton, closeButton *actionButton
	preview *canvas.Image

	displayLabels map[string]displays.ID
}

// New constructs a closed mosaic window with default visual settings.
func New(application fyne.App, host Host) *Window {
	w := &Window{
		app:           application,
		host:          host,
		settings:      mosaic.DefaultSettings(),
		ui:            fyneQueue{},
		seed:          func() int64 { return time.Now().UnixNano() },
		clock:         time.Now,
		exportFormat:  ExportPNG,
		exporter:      imaging.Export,
		displayLabels: make(map[string]displays.ID),
	}
	w.win.SetEscape(w.handleEscape)

	return w
}

// Show opens a new workflow or raises the existing one without retargeting its
// source/display snapshot.
func (w *Window) Show(snapshot Snapshot) {
	if w.win.Open() {
		w.win.Show(w.app, lang.L("Image Mosaic"), fyne.NewSize(windowWidth, windowHeight), w.build, nil)
		return
	}
	w.snapshot = cloneSnapshot(snapshot)
	w.target = snapshot.Displays.Default
	w.hasResult = false
	w.result = mosaic.Result{}
	w.generationBusy = false
	w.actionBusy = false
	w.statusText = ""
	w.win.Show(w.app, lang.L("Image Mosaic"), fyne.NewSize(windowWidth, windowHeight), w.build, w.closed)
	if window := w.win.Window(); window != nil && w.displaySelect != nil {
		w.syncActions()
		window.Canvas().Focus(w.displaySelect)
	}
}

func (w *Window) build() fyne.CanvasObject {
	w.sourceLabel = widget.NewLabel(w.sourceDescription())
	w.status = widget.NewLabel(w.statusText)
	w.previewStatus = widget.NewLabel(w.statusText)
	w.displaySelect = newNamedSelect(lang.L("Target display"), nil, func(label string) {
		if id, ok := w.displayLabels[label]; ok {
			w.target = id
		}
		w.syncActions()
	})
	w.syncDisplayOptions()
	w.refreshButton = newActionButton(lang.L("Refresh Displays"), w.RefreshTargets)

	w.minimum = newNamedSlider(lang.L("Minimum image size"), 0.10, 0.30, 0.01, w.settings.MinimumShortEdge, percentValue, func(value float64) {
		settings := w.settings
		settings.MinimumShortEdge = value
		w.acceptSettings(settings)
	})
	w.variation = newNamedSlider(lang.L("Size variation"), 0, 0.25, 0.01, w.settings.SizeVariation, percentValue, func(value float64) {
		settings := w.settings
		settings.SizeVariation = value
		w.acceptSettings(settings)
	})
	w.overlap = newNamedSlider(lang.L("Overlap"), 0, 0.20, 0.01, w.settings.Overlap, percentValue, func(value float64) {
		settings := w.settings
		settings.Overlap = value
		w.acceptSettings(settings)
	})
	w.rotation = newNamedSlider(lang.L("Maximum rotation"), 0, 12, 1, w.settings.MaximumRotation, degreeValue, func(value float64) {
		settings := w.settings
		settings.MaximumRotation = value
		w.acceptSettings(settings)
	})

	frames := []string{lang.L("None"), lang.L("Thin light"), lang.L("Thin dark"), lang.L("Polaroid")}
	w.frameSelect = newNamedSelect(lang.L("Frame"), frames, func(label string) {
		settings := w.settings
		switch label {
		case lang.L("Thin light"):
			settings.Frame = mosaic.FrameThinLight
		case lang.L("Thin dark"):
			settings.Frame = mosaic.FrameThinDark
		case lang.L("Polaroid"):
			settings.Frame = mosaic.FramePolaroid
		default:
			settings.Frame = mosaic.FrameNone
		}
		w.acceptSettings(settings)
	})
	w.frameSelect.SetSelected(frameLabel(w.settings.Frame))
	w.dropShadow = newNamedCheck(lang.L("Drop shadow"), w.settings.DropShadow, func(checked bool) {
		settings := w.settings
		settings.DropShadow = checked
		w.acceptSettings(settings)
	})

	w.minimumValue = w.minimum.valueLabel
	w.variationValue = w.variation.valueLabel
	w.overlapValue = w.overlap.valueLabel
	w.rotationValue = w.rotation.valueLabel
	w.advancedControls = container.NewVBox(
		labelledSlider(lang.L("Minimum image size"), w.minimum),
		labelledControl(lang.L("Frame"), w.frameSelect),
		labelledSlider(lang.L("Size variation"), w.variation),
		labelledSlider(lang.L("Overlap"), w.overlap),
		labelledSlider(lang.L("Maximum rotation"), w.rotation),
		w.dropShadow,
	)
	w.advancedControls.Hide()
	w.advancedButton = newActionButton(lang.L("Advanced"), func() {
		if w.advancedControls.Visible() {
			w.advancedControls.Hide()
			w.advancedButton.SetText(lang.L("Advanced"))
		} else {
			w.advancedControls.Show()
			w.advancedButton.SetText(lang.L("Hide Advanced"))
		}
	})
	w.generateButton = newActionButton(lang.L("Generate"), w.Generate)
	w.cancelButton = newActionButton(lang.L("Cancel"), w.Cancel)
	w.config = container.NewBorder(nil, container.NewHBox(w.generateButton, w.cancelButton), nil, nil,
		container.NewVBox(
			w.sourceLabel,
			widget.NewSeparator(),
			labelledControl(lang.L("Target display"), container.NewBorder(nil, nil, nil, w.refreshButton, w.displaySelect)),
			w.advancedButton,
			w.advancedControls,
			w.status,
		),
	)

	w.preview = canvas.NewImageFromImage(nil)
	w.preview.FillMode = canvas.ImageFillContain
	w.preview.ScaleMode = canvas.ImageScaleSmooth
	w.formatSelect = newNamedSelect(lang.L("Image format"), []string{lang.L("PNG"), lang.L("JPEG")}, func(label string) {
		if label == lang.L("JPEG") {
			w.exportFormat = ExportJPEG
		} else {
			w.exportFormat = ExportPNG
		}
	})
	if w.exportFormat == ExportJPEG {
		w.formatSelect.SetSelected(lang.L("JPEG"))
	} else {
		w.formatSelect.SetSelected(lang.L("PNG"))
	}
	w.regenerateButton = newActionButton(lang.L("Regenerate"), w.Regenerate)
	w.startOverButton = newActionButton(lang.L("Start Over"), w.StartOver)
	w.wallpaperButton = newActionButton(lang.L("Set as Wallpaper"), w.SetWallpaper)
	w.saveButton = newActionButton(lang.L("Save Image"), w.SaveImage)
	w.closeButton = newActionButton(lang.L("Close"), w.Close)
	w.previewPanel = container.NewBorder(container.NewHBox(w.startOverButton),
		container.NewVBox(
			labelledControl(lang.L("Image format"), w.formatSelect),
			container.NewHBox(w.regenerateButton, w.wallpaperButton, w.saveButton, w.closeButton),
		),
		nil, nil,
		container.NewStack(w.preview, container.NewVBox(w.previewStatus)),
	)
	w.previewPanel.Hide()
	w.loading = widget.NewProgressBarInfinite()
	w.loading.Hide()
	w.root = container.NewBorder(w.loading, nil, nil, nil, container.NewStack(w.config, w.previewPanel))
	w.syncActions()

	return w.root
}

type namedCheck struct {
	widget.Check
	name string
}

func newNamedCheck(name string, checked bool, changed func(bool)) *namedCheck {
	check := &namedCheck{name: name}
	check.Text = name
	check.OnChanged = changed
	check.ExtendBaseWidget(check)
	check.SetChecked(checked)

	return check
}

func (c *namedCheck) AccessibilityLabel() string { return c.name }

func (*namedCheck) AccessibilityRole() fyne.AccessibleRole { return fyne.AccessibleRoleButton }

func labelledControl(label string, control fyne.CanvasObject) fyne.CanvasObject {
	return container.NewBorder(nil, nil, widget.NewLabel(label), nil, control)
}

func labelledSlider(label string, slider *namedSlider) fyne.CanvasObject {
	return container.NewBorder(nil, nil, widget.NewLabel(label), slider.valueLabel, slider)
}

func percentValue(value float64) string { return fmt.Sprintf("%.0f%%", value*100) }

func degreeValue(value float64) string { return fmt.Sprintf(lang.L("%.0f degrees"), value) }

func (w *Window) sourceDescription() string {
	if w.snapshot.Kind == SourceSelection {
		return fmt.Sprintf(lang.L("Using %d selected images"), len(w.snapshot.Sources))
	}

	return fmt.Sprintf(lang.L("Using %d images from the current Grid result"), len(w.snapshot.Sources))
}

func (w *Window) syncDisplayOptions() {
	w.displayLabels = make(map[string]displays.ID, len(w.snapshot.Displays.Displays))
	options := make([]string, 0, len(w.snapshot.Displays.Displays))
	labelCounts := make(map[string]int)
	for _, display := range w.snapshot.Displays.Displays {
		width, height := display.Bounds.Dx(), display.Bounds.Dy()
		aspect := float64(width) / float64(height)
		label := fmt.Sprintf(lang.L("%s - %d x %d (%.2f:1)"), display.Name, width, height, aspect)
		options = append(options, label)
		labelCounts[label]++
	}
	selected := ""
	for index, display := range w.snapshot.Displays.Displays {
		label := options[index]
		if labelCounts[label] > 1 {
			label = fmt.Sprintf(lang.L("%s (%d)"), label, index+1)
			options[index] = label
		}
		w.displayLabels[label] = display.ID
		if display.ID == w.target {
			selected = label
		}
	}
	w.displaySelect.Options = options
	w.displaySelect.Refresh()
	if selected != "" {
		w.displaySelect.SetSelected(selected)
	} else {
		w.displaySelect.ClearSelected()
	}
}

func (w *Window) acceptSettings(settings mosaic.Settings) {
	if settings.Validate() == nil {
		w.settings = settings
	}
}

// Generate starts a generation from the current validated UI snapshot.
func (w *Window) Generate() {
	w.startGeneration(false)
}

func (w *Window) startGeneration(supersede bool) {
	if !w.Opened() || w.host == nil || len(w.snapshot.Sources) == 0 || w.target == "" || (!supersede && w.Busy()) {
		return
	}
	display, ok := w.refreshSelectedDisplay()
	if !ok {
		return
	}
	seed := w.nextSeed()
	request, err := mosaic.NewRequest(w.snapshot.Sources, image.Pt(display.Bounds.Dx(), display.Bounds.Dy()), w.settings, seed)
	if err != nil {
		w.setStatus(err.Error())
		return
	}

	ctx, revision := w.lifecycle.begin()
	w.generationBusy = true
	w.loading.Show()
	w.setStatus(lang.L("Generating mosaic..."))
	w.syncActions()
	w.workers.Go(func() {
		result, generateErr := w.host.GenerateMosaic(ctx, request)
		if !w.lifecycle.current(revision) {
			return
		}
		w.ui.Do(func() {
			if !w.lifecycle.current(revision) || !w.win.Open() {
				return
			}
			w.generationBusy = false
			w.loading.Hide()
			if generateErr != nil {
				fyne.LogError("failed to generate mosaic", generateErr)
				w.setStatus(fmt.Sprintf(lang.L("Mosaic generation failed: %v"), generateErr))
				w.syncActions()
				return
			}
			w.result = result
			w.hasResult = true
			w.preview.Image = result.Image()
			w.preview.Refresh()
			w.config.Hide()
			w.previewPanel.Show()
			w.setStatus("")
			w.syncActions()
			w.win.Window().Canvas().Focus(w.startOverButton)
		})
	})
}

// Regenerate preserves sources, target, and settings while choosing a new
// seed. Direct calls may supersede active work; disabled widgets prevent an
// accidental duplicate click in normal UI use.
func (w *Window) Regenerate() { w.startGeneration(true) }

// StartOver returns an idle finished workflow to configuration while retaining
// the command-entry inputs needed to generate for another display.
func (w *Window) StartOver() {
	if !w.PreviewActionsEnabled() {
		return
	}
	w.result = mosaic.Result{}
	w.hasResult = false
	w.preview.Image = nil
	w.preview.Refresh()
	w.previewPanel.Hide()
	w.config.Show()
	w.setStatus("")
	w.syncActions()
	w.win.Window().Canvas().Focus(w.displaySelect)
}

// Cancel supersedes active work and returns to the configuration state.
func (w *Window) Cancel() {
	if !w.generationBusy {
		w.Close()
		return
	}
	w.lifecycle.invalidate()
	w.generationBusy = false
	if w.loading != nil {
		w.loading.Hide()
	}
	w.setStatus(lang.L("Mosaic generation cancelled"))
	w.syncActions()
}

func (w *Window) handleEscape() bool {
	if w.generationBusy {
		w.Cancel()
		return true
	}
	if w.actionBusy {
		w.setStatus(lang.L("Another wallpaper change is already in progress."))
		return true
	}

	return false
}

func (w *Window) nextSeed() int64 {
	seed := w.seed()
	if seed == w.lastSeed {
		seed++
	}
	w.lastSeed = seed

	return seed
}

func (w *Window) selectedDisplay() (displays.Display, bool) {
	for _, display := range w.snapshot.Displays.Displays {
		if display.ID == w.target {
			return display, true
		}
	}

	return displays.Display{}, false
}

// RefreshTargets reinspects displays. A disappeared explicit target is cleared
// and must be chosen again; it never silently falls back.
func (w *Window) RefreshTargets() {
	if !w.Opened() || w.host == nil || w.Busy() {
		return
	}
	_, _ = w.refreshSelectedDisplay()
}

func (w *Window) refreshSelectedDisplay() (displays.Display, bool) {
	topology, err := w.host.InspectMosaicDisplays()
	if err != nil {
		fyne.LogError("failed to refresh mosaic displays", err)
		w.setStatus(fmt.Sprintf(lang.L("Could not refresh displays: %v"), err))
		return displays.Display{}, false
	}
	w.snapshot.Displays = displays.Snapshot{Displays: slices.Clone(topology.Displays), Default: topology.Default}
	display, attached := w.selectedDisplay()
	if !attached {
		w.target = ""
		w.setStatus(lang.L("The selected display is no longer attached. Choose another display."))
	}
	w.syncDisplayOptions()
	w.syncActions()

	return display, attached
}

func (w *Window) syncActions() {
	canGenerate := w.Opened() && w.target != "" && !w.Busy() && len(w.snapshot.Sources) > 0
	configurationEnabled := w.Opened() && !w.Busy()
	if w.displaySelect != nil {
		setDisableableEnabled(w.displaySelect, configurationEnabled)
	}
	if w.minimum != nil {
		setDisableableEnabled(w.minimum, configurationEnabled)
	}
	if w.frameSelect != nil {
		setDisableableEnabled(w.frameSelect, configurationEnabled)
	}
	if w.advancedButton != nil {
		setDisableableEnabled(w.advancedButton, configurationEnabled)
	}
	for _, slider := range []*namedSlider{w.variation, w.overlap, w.rotation} {
		if slider != nil {
			setDisableableEnabled(slider, configurationEnabled)
		}
	}
	if w.dropShadow != nil {
		setDisableableEnabled(w.dropShadow, configurationEnabled)
	}
	setEnabled(w.generateButton, canGenerate)
	setEnabled(w.refreshButton, w.Opened() && !w.Busy())
	setEnabled(w.startOverButton, w.hasResult && !w.Busy())
	setEnabled(w.regenerateButton, canGenerate && w.hasResult)
	setEnabled(w.wallpaperButton, w.hasResult && !w.Busy())
	setEnabled(w.saveButton, w.hasResult && !w.Busy())
}

func setEnabled(button *actionButton, enabled bool) {
	if button == nil {
		return
	}
	if enabled {
		button.Enable()
	} else {
		button.Disable()
	}
}

func setDisableableEnabled(control fyne.Disableable, enabled bool) {
	if enabled {
		control.Enable()
	} else {
		control.Disable()
	}
}

func (w *Window) setStatus(text string) {
	w.statusText = text
	if w.status != nil {
		w.status.SetText(text)
	}
	if w.previewStatus != nil {
		w.previewStatus.SetText(text)
	}
}

// SetWallpaper starts the host's target-aware wallpaper effect from a captured
// immutable result. The full behavior is completed by the wallpaper ticket.
func (w *Window) SetWallpaper() {
	if !w.PreviewActionsEnabled() || w.host == nil {
		return
	}
	result, target := w.result, w.target
	solo := len(w.snapshot.Displays.Displays) == 1
	ctx, revision := w.actionLifecycle.begin()
	w.actionBusy = true
	w.syncActions()
	w.workers.Go(func() {
		err := w.host.SetMosaicWallpaper(ctx, result, target, solo)
		w.ui.Do(func() {
			if !w.actionLifecycle.current(revision) || !w.Opened() {
				return
			}
			w.actionBusy = false
			if err == nil {
				w.setStatus(lang.L("Mosaic set as wallpaper."))
			} else {
				fyne.LogError("failed to set mosaic wallpaper", err)
				var unsupported *wallpaper.TargetUnsupportedError
				switch {
				case errors.As(err, &unsupported):
					w.setStatus(lang.L("This desktop cannot set wallpaper for one display. Save Image remains available."))
				case errors.Is(err, wallpaper.ErrBusy):
					w.setStatus(lang.L("Another wallpaper change is already in progress."))
				default:
					w.setStatus(fmt.Sprintf(lang.L("Could not set mosaic wallpaper: %v"), err))
				}
			}
			w.syncActions()
		})
	})
}

func (w *Window) closed() {
	w.lifecycle.invalidate()
	w.actionLifecycle.invalidate()
	w.generationBusy = false
	w.actionBusy = false
	w.snapshot = Snapshot{}
	w.result = mosaic.Result{}
	w.hasResult = false
	w.root, w.config, w.previewPanel = nil, nil, nil
	w.sourceLabel, w.status, w.previewStatus = nil, nil, nil
	w.displaySelect, w.frameSelect = nil, nil
	w.formatSelect = nil
	w.minimum, w.variation, w.overlap, w.rotation = nil, nil, nil, nil
	w.dropShadow = nil
	w.minimumValue, w.variationValue, w.overlapValue, w.rotationValue = nil, nil, nil, nil
	w.loading = nil
	w.advancedButton, w.advancedControls = nil, nil
	w.refreshButton = nil
	w.generateButton, w.cancelButton = nil, nil
	w.startOverButton, w.regenerateButton, w.wallpaperButton, w.saveButton, w.closeButton = nil, nil, nil, nil, nil
	w.preview = nil
}

// Close cancels work and closes the secondary window.
func (w *Window) Close() {
	w.lifecycle.invalidate()
	w.actionLifecycle.invalidate()
	if window := w.win.Window(); window != nil {
		window.Close()
		return
	}
	w.closed()
}

func (w *Window) Opened() bool        { return w.win.Open() }
func (w *Window) Window() fyne.Window { return w.win.Window() }
func (w *Window) Busy() bool          { return w.generationBusy || w.actionBusy }
func (w *Window) CanGenerate() bool {
	_, attached := w.selectedDisplay()
	return w.Opened() && attached && !w.Busy() && len(w.snapshot.Sources) > 0
}
func (w *Window) PreviewActionsEnabled() bool   { return w.hasResult && !w.Busy() && w.Opened() }
func (w *Window) Target() displays.ID           { return w.target }
func (w *Window) Snapshot() Snapshot            { return cloneSnapshot(w.snapshot) }
func (w *Window) Result() (mosaic.Result, bool) { return w.result, w.hasResult }
func (w *Window) Status() string                { return w.statusText }

func (w *Window) SelectTarget(id displays.ID) bool {
	if w.Busy() {
		return false
	}
	for _, display := range w.snapshot.Displays.Displays {
		if display.ID == id {
			w.target = id
			w.syncDisplayOptions()
			w.syncActions()
			return true
		}
	}

	return false
}

func (w *Window) SetSettings(settings mosaic.Settings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	w.settings = settings

	return nil
}

func (w *Window) Settings() mosaic.Settings                { return w.settings }
func (w *Window) RestoreSettings(settings mosaic.Settings) { w.settings = settings.Normalized() }
func (w *Window) SetSeedSource(source func() int64) {
	if source == nil {
		w.seed = func() int64 { return time.Now().UnixNano() }
		return
	}
	w.seed = source
}

func (w *Window) SetUIQueue(queue UIQueue) {
	if queue == nil {
		w.ui = fyneQueue{}
		return
	}
	w.ui = queue
}

func (w *Window) Settle(ctx context.Context) error {
	for {
		if err := w.workers.wait(ctx); err != nil {
			return err
		}
		drained := w.ui.Drain()
		if !drained && w.workers.activeCount() == 0 {
			return nil
		}
	}
}

func (w *Window) RestoreGeometry(geometry widgets.Geometry) { w.win.Remember(geometry) }
func (w *Window) Geometry() widgets.Geometry                { return w.win.Geometry() }
func (w *Window) StopTracking()                             { w.win.StopTracking() }

func frameLabel(frame mosaic.FrameStyle) string {
	switch frame {
	case mosaic.FrameThinLight:
		return lang.L("Thin light")
	case mosaic.FrameThinDark:
		return lang.L("Thin dark")
	case mosaic.FramePolaroid:
		return lang.L("Polaroid")
	default:
		return lang.L("None")
	}
}
