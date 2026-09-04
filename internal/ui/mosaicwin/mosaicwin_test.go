package mosaicwin

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/displays"
	"github.com/frathe/picfetch/internal/mosaic"
	"github.com/frathe/picfetch/internal/ui/widgets"
	"github.com/frathe/picfetch/internal/uitest"
	"github.com/frathe/picfetch/internal/wallpaper"
)

type fakeHost struct {
	generate  func(context.Context, mosaic.Request) (mosaic.Result, error)
	inspect   func() (displays.Snapshot, error)
	wallpaper func(context.Context, mosaic.Result, displays.ID) error
}

func (h *fakeHost) GenerateMosaic(ctx context.Context, request mosaic.Request) (mosaic.Result, error) {
	return h.generate(ctx, request)
}

func (h *fakeHost) InspectMosaicDisplays() (displays.Snapshot, error) {
	return h.inspect()
}

func (h *fakeHost) SetMosaicWallpaper(ctx context.Context, result mosaic.Result, target displays.ID) error {
	return h.wallpaper(ctx, result, target)
}

func TestMosaicWindow_IsSingletonAndSnapshotsInputs(t *testing.T) {
	application := test.NewApp()
	host := successfulHost(t)
	w := New(application, host)
	sources := []fyne.URI{uitest.TempJPEGURI(t, "first.jpg", 8, 6, color.White)}
	topology := testTopology("one", 1920, 1080)
	snapshot, err := NewSnapshot(sources, SourceSelection, topology)
	if err != nil {
		t.Fatal(err)
	}
	w.Show(snapshot)
	firstWindow := w.Window()

	sources[0] = uitest.TempJPEGURI(t, "changed.jpg", 8, 6, color.Black)
	topology.Displays[0].Name = "Changed"
	other, err := NewSnapshot([]fyne.URI{sources[0]}, SourceResult, testTopology("two", 2560, 1440))
	if err != nil {
		t.Fatal(err)
	}
	w.Show(other)

	if w.Window() != firstWindow {
		t.Fatal("a second Show opened a different window")
	}
	got := w.Snapshot()
	if len(got.Sources) != 1 || got.Sources[0].Name() != "first.jpg" || got.Displays.Displays[0].Name != "Display one" {
		t.Fatalf("retained snapshot = %+v", got)
	}
	w.Close()
}

func TestMosaicControls_RejectInvalidSettings(t *testing.T) {
	w := New(test.NewApp(), successfulHost(t))
	w.RestoreSettings(mosaic.DefaultSettings())
	w.Show(mustSnapshot(t))

	invalid := mosaic.DefaultSettings()
	invalid.MinimumShortEdge = 0.31
	if err := w.SetSettings(invalid); err == nil {
		t.Fatal("SetSettings accepted an invalid minimum size")
	}
	if w.Settings() != mosaic.DefaultSettings() {
		t.Fatal("invalid in-progress settings replaced the last valid settings")
	}

	valid := mosaic.DefaultSettings()
	valid.SizeVariation = 0
	valid.Overlap = 0
	valid.MaximumRotation = 0
	valid.Frame = mosaic.FramePolaroid
	if err := w.SetSettings(valid); err != nil || w.Settings() != valid {
		t.Fatalf("SetSettings(valid) = %v, settings=%+v", err, w.Settings())
	}
	w.Close()
}

func TestMosaicConfiguration_OpensEnabledAndRespondsToPointerInput(t *testing.T) {
	inspections := 0
	host := successfulHost(t)
	host.inspect = func() (displays.Snapshot, error) {
		inspections++
		return testTopology("one", 80, 50), nil
	}
	application := test.NewApp()
	w := New(application, host)
	w.Show(mustSnapshot(t))

	controls := []struct {
		name    string
		control fyne.Disableable
	}{
		{name: "target display", control: w.displaySelect},
		{name: "refresh displays", control: w.refreshButton},
		{name: "minimum image size", control: w.minimum},
		{name: "frame", control: w.frameSelect},
		{name: "drop shadow", control: w.dropShadow},
		{name: "advanced", control: w.advancedButton},
		{name: "generate", control: w.generateButton},
	}
	for _, item := range controls {
		if item.control.Disabled() {
			t.Errorf("%s control is disabled immediately after opening", item.name)
		}
	}
	if w.advancedControls.Visible() {
		t.Error("Advanced controls are visible before the user expands them")
	}
	if w.Window().Canvas().Overlays().Top() != nil {
		t.Fatal("configuration window opened with an unexpected overlay")
	}

	tap := func(object fyne.CanvasObject) {
		t.Helper()
		size := object.Size()
		position := application.Driver().AbsolutePositionForObject(object).
			Add(fyne.NewPos(size.Width/2, size.Height/2))
		test.TapCanvas(w.Window().Canvas(), position)
	}
	tap(w.displaySelect)
	if w.Window().Canvas().Overlays().Top() == nil {
		t.Fatal("tapping Target display did not open its options")
	}
	w.displaySelect.Hide()
	w.displaySelect.Show()
	tap(w.advancedButton)
	if !w.advancedControls.Visible() {
		t.Error("tapping Advanced did not reveal its controls")
	}
	w.root.Refresh()
	_ = w.Window().Canvas().Capture()
	checkPosition := application.Driver().AbsolutePositionForObject(w.dropShadow).
		Add(fyne.NewPos(theme.Padding()*2, w.dropShadow.Size().Height/2))
	test.TapCanvas(w.Window().Canvas(), checkPosition)
	if w.Settings().DropShadow {
		t.Error("tapping Drop shadow did not disable it")
	}
	tap(w.refreshButton)
	if inspections != 1 {
		t.Errorf("tapping Refresh Displays made %d inspections, want 1", inspections)
	}
	w.Close()
}

func TestMosaicConfiguration_AdvancedOwnsAllVisualSettings(t *testing.T) {
	w := New(test.NewApp(), successfulHost(t))
	w.Show(mustSnapshot(t))

	visualControls := []fyne.CanvasObject{
		w.minimum, w.frameSelect, w.variation, w.overlap, w.rotation, w.dropShadow,
	}
	for _, control := range visualControls {
		if !containsMosaicObject(w.advancedControls, control) {
			t.Errorf("visual control %T is not inside Advanced", control)
		}
	}
	if containsMosaicObject(w.advancedControls, w.displaySelect) {
		t.Error("target display is inside Advanced")
	}
	if w.advancedControls.Visible() {
		t.Fatal("Advanced controls are visible before expansion")
	}

	if !w.dropShadow.Checked {
		t.Fatal("Drop shadow should default to checked")
	}
	settings := w.Settings()
	settings.DropShadow = false
	w.dropShadow.SetChecked(false)
	if got := w.Settings(); got != settings {
		t.Fatalf("Drop shadow change settings = %+v, want %+v", got, settings)
	}

	w.advancedButton.OnTapped()
	if !w.advancedControls.Visible() {
		t.Fatal("Advanced controls stayed hidden after expansion")
	}
	w.Close()
}

func TestMosaicAccessibleControls_RedrawAsEnabledAfterInitialDisabledRender(t *testing.T) {
	customSelect := newNamedSelect("Target display", []string{"Display one"}, func(string) {})
	customSelect.SetSelected("Display one")
	standardSelect := widget.NewSelect([]string{"Display one"}, func(string) {})
	standardSelect.SetSelected("Display one")

	customSlider := newNamedSlider("Minimum image size", 0.1, 0.3, 0.01, 0.2, percentValue, func(float64) {})
	standardSlider := widget.NewSlider(0.1, 0.3)
	standardSlider.Step = 0.01
	standardSlider.Value = 0.2
	customCheck := newNamedCheck("Drop shadow", true, func(bool) {})
	standardCheck := widget.NewCheck("Drop shadow", func(bool) {})
	standardCheck.SetChecked(true)

	tests := []struct {
		name     string
		custom   fyne.CanvasObject
		standard fyne.CanvasObject
	}{
		{name: "button", custom: newActionButton("Generate", func() {}), standard: widget.NewButton("Generate", func() {})},
		{name: "select", custom: customSelect, standard: standardSelect},
		{name: "slider", custom: customSlider, standard: standardSlider},
		{name: "check", custom: customCheck, standard: standardCheck},
	}
	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			custom := testCase.custom.(fyne.Disableable)
			custom.Disable()
			customWindow := test.NewWindow(testCase.custom)
			customWindow.Resize(fyne.NewSize(300, 48))
			t.Cleanup(customWindow.Close)
			_ = customWindow.Canvas().Capture()
			custom.Enable()
			got := customWindow.Canvas().Capture()

			standardWindow := test.NewWindow(testCase.standard)
			standardWindow.Resize(fyne.NewSize(300, 48))
			t.Cleanup(standardWindow.Close)
			want := standardWindow.Canvas().Capture()
			if !sameImage(got, want) {
				t.Fatalf("enabled mosaic %s retained disabled rendering", testCase.name)
			}
		})
	}
}

func TestMosaicAccessibility_InteractiveControlsHaveMeaningfulNames(t *testing.T) {
	w := New(test.NewApp(), successfulHost(t))
	w.Show(mustSnapshot(t))
	controls := []fyne.CanvasObject{
		w.displaySelect, w.refreshButton, w.minimum, w.frameSelect, w.advancedButton,
		w.variation, w.overlap, w.rotation, w.dropShadow, w.generateButton, w.cancelButton,
		w.startOverButton, w.formatSelect, w.regenerateButton, w.wallpaperButton, w.saveButton, w.closeButton,
	}
	for _, control := range controls {
		accessible, ok := control.(fyne.Accessible)
		if !ok {
			t.Errorf("%T does not implement fyne.Accessible", control)
			continue
		}
		if accessible.AccessibilityLabel() == "" || accessible.AccessibilityRole() == "" {
			t.Errorf("%T has empty accessibility metadata", control)
		}
	}
	if w.minimumValue.Text == "" || w.variationValue.Text == "" || w.overlapValue.Text == "" || w.rotationValue.Text == "" {
		t.Fatal("numeric controls do not expose visible values")
	}
	w.Close()
}

func TestMosaicKeyboard_ConfigAdvancedAndPreviewOrder(t *testing.T) {
	w := generatedWindowFromConfig(t)
	canvas := w.Window().Canvas()
	assertFocusCycle(t, canvas, []fyne.Focusable{
		w.displaySelect, w.refreshButton, w.advancedButton, w.generateButton, w.cancelButton,
	})

	w.advancedButton.OnTapped()
	assertFocusCycle(t, canvas, []fyne.Focusable{
		w.displaySelect, w.refreshButton, w.advancedButton, w.minimum, w.frameSelect,
		w.variation, w.overlap, w.rotation, w.dropShadow, w.generateButton, w.cancelButton,
	})

	w.Generate()
	settleWindow(t, w)
	assertFocusCycle(t, canvas, []fyne.Focusable{
		w.startOverButton, w.formatSelect, w.regenerateButton, w.wallpaperButton, w.saveButton, w.closeButton,
	})
	w.Close()
}

func TestMosaicKeyboard_EnterAndSpaceReachEveryPreviewAction(t *testing.T) {
	host := successfulHost(t)
	generations, wallpaperCalls := 0, 0
	host.generate = func(ctx context.Context, request mosaic.Request) (mosaic.Result, error) {
		generations++
		return mosaic.Generate(ctx, request)
	}
	host.wallpaper = func(context.Context, mosaic.Result, displays.ID) error {
		wallpaperCalls++
		return nil
	}
	w := New(test.NewApp(), host)
	w.SetUIQueue(&uitest.UIQueue{})
	w.Show(mustSnapshot(t))
	w.Window().Canvas().Focus(w.generateButton)
	w.generateButton.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnter})
	settleWindow(t, w)
	if generations != 1 {
		t.Fatalf("Enter on Generate calls=%d", generations)
	}

	w.Window().Canvas().Focus(w.regenerateButton)
	w.regenerateButton.TypedKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	settleWindow(t, w)
	if generations != 2 {
		t.Fatalf("Space on Regenerate calls=%d", generations)
	}
	w.Window().Canvas().Focus(w.wallpaperButton)
	w.wallpaperButton.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnter})
	settleWindow(t, w)
	if wallpaperCalls != 1 {
		t.Fatalf("Enter on Set as Wallpaper calls=%d", wallpaperCalls)
	}

	uitest.StubSaveChooser(t, func(string) ([]byte, error) { return []byte(t.TempDir() + "/mosaic.png\n"), nil })
	exports := 0
	w.SetExporter(func(fyne.URI, image.Image, fyne.URI) error { exports++; return nil })
	w.Window().Canvas().Focus(w.saveButton)
	w.saveButton.TypedKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	settleWindow(t, w)
	if exports != 1 {
		t.Fatalf("Space on Save Image calls=%d", exports)
	}
	w.Window().Canvas().Focus(w.startOverButton)
	w.startOverButton.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnter})
	if !w.config.Visible() || w.previewPanel.Visible() {
		t.Fatal("Enter on Start Over did not return to configuration")
	}
	w.Window().Canvas().Focus(w.generateButton)
	w.generateButton.TypedKey(&fyne.KeyEvent{Name: fyne.KeySpace})
	settleWindow(t, w)
	if generations != 3 {
		t.Fatalf("Space on Generate after Start Over calls=%d", generations)
	}
	w.Window().Canvas().Focus(w.closeButton)
	w.closeButton.TypedKey(&fyne.KeyEvent{Name: fyne.KeyEnter})
	if w.Opened() {
		t.Fatal("Enter on Close left the mosaic window open")
	}
}

func TestMosaicKeyboard_EscapeCancelsBusyThenCloses(t *testing.T) {
	started := make(chan struct{})
	host := successfulHost(t)
	host.generate = func(ctx context.Context, _ mosaic.Request) (mosaic.Result, error) {
		close(started)
		<-ctx.Done()
		return mosaic.Result{}, ctx.Err()
	}
	w := New(test.NewApp(), host)
	w.SetUIQueue(&uitest.UIQueue{})
	w.Show(mustSnapshot(t))
	w.Generate()
	<-started
	handler := w.Window().Canvas().OnTypedKey()
	logged := captureMosaicLog(t, func() {
		handler(&fyne.KeyEvent{Name: fyne.KeyEscape})
		settleWindow(t, w)
	})
	if !w.Opened() || w.Busy() {
		t.Fatalf("first Escape open=%v busy=%v, want cancelled and open", w.Opened(), w.Busy())
	}
	if logged != "" {
		t.Fatalf("cancelled generation logged an error: %q", logged)
	}
	handler(&fyne.KeyEvent{Name: fyne.KeyEscape})
	if w.Opened() {
		t.Fatal("second Escape did not close the idle mosaic window")
	}
}

func TestMosaicCancel_IdleClosesConfigurationWindow(t *testing.T) {
	w := generatedWindowFromConfig(t)
	w.Cancel()
	if w.Opened() {
		t.Fatal("idle Cancel left the configuration window open")
	}
}

func TestMosaicTarget_IdenticalDisplaysRemainIndividuallySelectable(t *testing.T) {
	topology := displays.Snapshot{
		Displays: []displays.Display{
			{ID: "one", Name: "DELL U2723QE", Bounds: image.Rect(0, 0, 80, 50)},
			{ID: "two", Name: "DELL U2723QE", Bounds: image.Rect(80, 0, 160, 50)},
		},
		Default: "one",
	}
	host := successfulHost(t)
	host.inspect = func() (displays.Snapshot, error) { return topology, nil }
	var wallpaperTarget displays.ID
	host.wallpaper = func(_ context.Context, _ mosaic.Result, target displays.ID) error {
		wallpaperTarget = target
		return nil
	}
	w := New(test.NewApp(), host)
	w.SetUIQueue(&uitest.UIQueue{})
	snapshot := mustSnapshot(t)
	snapshot.Displays = topology
	w.Show(snapshot)
	defer w.Close()
	if w.Target() != "one" {
		t.Fatalf("initial target = %q, want the default display one", w.Target())
	}
	if w.displaySelect.Options[0] == w.displaySelect.Options[1] {
		t.Fatal("identical displays have indistinguishable choices")
	}
	w.displaySelect.SetSelected(w.displaySelect.Options[1])
	if w.Target() != "two" {
		t.Fatalf("second choice targets %q, want two", w.Target())
	}
	topology.Displays[0], topology.Displays[1] = topology.Displays[1], topology.Displays[0]
	w.RefreshTargets()
	if w.Target() != "two" {
		t.Fatalf("reordered refresh changed the target to %q", w.Target())
	}
	w.displaySelect.SetSelected(w.displaySelect.Options[1])
	w.Generate()
	settleWindow(t, w)
	w.SetWallpaper()
	settleWindow(t, w)
	if wallpaperTarget != "one" {
		t.Fatalf("wallpaper target = %q, want the explicitly selected display one", wallpaperTarget)
	}
}

func TestMosaicTarget_RefreshRequiresNewChoiceAfterRemoval(t *testing.T) {
	topology := displays.Snapshot{
		Displays: []displays.Display{
			{ID: "one", Name: "One", Bounds: image.Rect(0, 0, 1920, 1080)},
			{ID: "two", Name: "Two", Bounds: image.Rect(1920, 0, 4480, 1440)},
		},
		Default: "one",
	}
	host := successfulHost(t)
	host.inspect = func() (displays.Snapshot, error) { return testTopology("one", 1920, 1080), nil }
	w := New(test.NewApp(), host)
	snapshot, err := NewSnapshot([]fyne.URI{uitest.TempJPEGURI(t, "source.jpg", 8, 6, color.White)}, SourceResult, topology)
	if err != nil {
		t.Fatal(err)
	}
	w.Show(snapshot)
	if !w.SelectTarget("two") {
		t.Fatal("could not select attached target two")
	}

	w.RefreshTargets()
	if w.Target() != "" || w.CanGenerate() {
		t.Fatalf("after removal target=%q canGenerate=%v, want no target and disabled", w.Target(), w.CanGenerate())
	}
	if !w.SelectTarget("one") || !w.CanGenerate() {
		t.Fatal("choosing an attached target did not re-enable generation")
	}
	w.Close()
}

func TestMosaicFailure_RefreshDisplaysLogsAndKeepsStatus(t *testing.T) {
	host := successfulHost(t)
	host.inspect = func() (displays.Snapshot, error) { return displays.Snapshot{}, errors.New("display service failed") }
	w := New(test.NewApp(), host)
	w.Show(mustSnapshot(t))

	logged := captureMosaicLog(t, w.RefreshTargets)
	for _, want := range []string{"failed to refresh mosaic displays", "display service failed"} {
		if !strings.Contains(logged, want) {
			t.Errorf("refresh log = %q, want it to contain %q", logged, want)
		}
	}
	if status := w.Status(); status != "Could not refresh displays: display service failed" {
		t.Fatalf("refresh status = %q", status)
	}
	w.Close()
}

func TestMosaicTarget_RefreshesDisplayNativePixelSizeAtGenerate(t *testing.T) {
	want := image.Pt(3456, 2234)
	generated := make(chan image.Point, 1)
	host := successfulHost(t)
	host.inspect = func() (displays.Snapshot, error) { return testTopology("retina", want.X, want.Y), nil }
	host.generate = func(_ context.Context, request mosaic.Request) (mosaic.Result, error) {
		generated <- request.TargetSize()
		return mosaic.Result{}, errors.New("stop after observing native target size")
	}
	w := New(test.NewApp(), host)
	w.SetUIQueue(&uitest.UIQueue{})
	snapshot, err := NewSnapshot(
		[]fyne.URI{uitest.TempJPEGURI(t, "source.jpg", 8, 6, color.White)},
		SourceResult,
		testTopology("retina", 1728, 1117),
	)
	if err != nil {
		t.Fatal(err)
	}
	w.Show(snapshot)
	w.Generate()
	settleWindow(t, w)
	if got := <-generated; got != want {
		t.Fatalf("generation target = %v, want native panel pixels %v", got, want)
	}
	w.Close()
}

func TestMosaicTarget_GenerateRejectsDisplayRemovedSinceOpen(t *testing.T) {
	topology := displays.Snapshot{
		Displays: []displays.Display{
			{ID: "one", Name: "One", Bounds: image.Rect(0, 0, 1920, 1080)},
			{ID: "two", Name: "Two", Bounds: image.Rect(0, 0, 2560, 1440)},
		},
		Default: "one",
	}
	generated := make(chan struct{}, 1)
	host := successfulHost(t)
	host.inspect = func() (displays.Snapshot, error) { return testTopology("one", 1920, 1080), nil }
	host.generate = func(_ context.Context, _ mosaic.Request) (mosaic.Result, error) {
		generated <- struct{}{}
		return mosaic.Result{}, errors.New("generation must not start")
	}
	w := New(test.NewApp(), host)
	w.SetUIQueue(&uitest.UIQueue{})
	snapshot, err := NewSnapshot(
		[]fyne.URI{uitest.TempJPEGURI(t, "source.jpg", 8, 6, color.White)},
		SourceResult,
		topology,
	)
	if err != nil {
		t.Fatal(err)
	}
	w.Show(snapshot)
	if !w.SelectTarget("two") {
		t.Fatal("could not select attached target two")
	}

	w.Generate()
	settleWindow(t, w)
	select {
	case <-generated:
		t.Fatal("generation started for a display removed before Generate")
	default:
	}
	if w.Target() != "" || w.CanGenerate() || w.Status() != "The selected display is no longer attached. Choose another display." {
		t.Fatalf("after generate-time removal target=%q canGenerate=%v status=%q", w.Target(), w.CanGenerate(), w.Status())
	}
	if got := w.Snapshot().Displays; len(got.Displays) != 1 || got.Displays[0].ID != "one" {
		t.Fatalf("refreshed topology = %+v, want only display one", got)
	}
	w.Close()
}

func TestMosaicTarget_CannotRetargetOrRefreshWhileGenerationIsBusy(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	inspections := 0
	topology := displays.Snapshot{
		Displays: []displays.Display{
			{ID: "one", Name: "One", Bounds: image.Rect(0, 0, 80, 50)},
			{ID: "two", Name: "Two", Bounds: image.Rect(80, 0, 180, 60)},
		},
		Default: "one",
	}
	host := successfulHost(t)
	host.generate = func(ctx context.Context, request mosaic.Request) (mosaic.Result, error) {
		close(started)
		<-release
		return mosaic.Generate(ctx, request)
	}
	host.inspect = func() (displays.Snapshot, error) {
		inspections++
		return topology, nil
	}
	snapshot, err := NewSnapshot(
		[]fyne.URI{uitest.TempJPEGURI(t, "source.jpg", 8, 6, color.White)}, SourceResult, topology,
	)
	if err != nil {
		t.Fatal(err)
	}
	w := New(test.NewApp(), host)
	w.SetUIQueue(&uitest.UIQueue{})
	w.Show(snapshot)
	w.Generate()
	<-started

	w.RefreshTargets()
	if w.SelectTarget("two") {
		t.Fatal("SelectTarget changed the target during generation")
	}
	if inspections != 1 || w.Target() != "one" {
		t.Fatalf("busy refresh inspections=%d target=%q, want only Generate revalidation and target one", inspections, w.Target())
	}
	if !w.displaySelect.Disabled() || !w.minimum.Disabled() || !w.frameSelect.Disabled() || !w.advancedButton.Disabled() || !w.dropShadow.Disabled() {
		t.Fatal("configuration controls stayed enabled during generation")
	}

	close(release)
	settleWindow(t, w)
	w.Close()
}

func TestMosaicGeometry_RestoresAndOutlivesClose(t *testing.T) {
	w := New(test.NewApp(), successfulHost(t))
	w.RestoreGeometry(widgets.Geometry{X: 12, Y: 34, PositionSet: true, Size: fyne.NewSize(820, 660)})
	w.Show(mustSnapshot(t))
	if got := w.Window().Canvas().Size(); got != fyne.NewSize(820, 660) {
		t.Fatalf("restored mosaic size = %v", got)
	}
	w.Window().Resize(fyne.NewSize(860, 700))
	w.Close()
	geometry := w.Geometry()
	if !geometry.PositionSet || geometry.X != 12 || geometry.Y != 34 || geometry.Size != fyne.NewSize(860, 700) {
		t.Fatalf("mosaic geometry after close = %+v", geometry)
	}
}

func TestMosaicGenerate_DoesNotBlockAndPublishesPreview(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	host := successfulHost(t)
	host.generate = func(_ context.Context, request mosaic.Request) (mosaic.Result, error) {
		close(started)
		<-release
		return mosaic.Generate(context.Background(), request)
	}
	w := New(test.NewApp(), host)
	w.SetUIQueue(&uitest.UIQueue{})
	w.Show(mustSnapshot(t))

	w.Generate()
	<-started
	if !w.Busy() || !w.Opened() || !w.loading.Visible() {
		t.Fatal("generation did not leave a responsive busy window")
	}
	close(release)
	settleWindow(t, w)
	if _, ok := w.Result(); !ok || w.Busy() || w.loading.Visible() || !w.PreviewActionsEnabled() {
		t.Fatalf("after generation result=%v busy=%v actions=%v", ok, w.Busy(), w.PreviewActionsEnabled())
	}
	w.Close()
}

func TestMosaicStartOver_ReturnsToSettingsForAnotherDisplay(t *testing.T) {
	application := test.NewApp()
	topology := displays.Snapshot{
		Displays: []displays.Display{
			{ID: "one", Name: "Display one", Bounds: image.Rect(0, 0, 80, 50)},
			{ID: "two", Name: "Display two", Bounds: image.Rect(0, 0, 120, 70)},
		},
		Default: "one",
	}
	snapshot, err := NewSnapshot(
		[]fyne.URI{uitest.TempJPEGURI(t, "source.jpg", 8, 6, color.White)},
		SourceResult,
		topology,
	)
	if err != nil {
		t.Fatal(err)
	}
	settings := mosaic.DefaultSettings()
	settings.Overlap = 0.03
	settings.Frame = mosaic.FramePolaroid
	generations := 0
	host := successfulHost(t)
	host.inspect = func() (displays.Snapshot, error) { return topology, nil }
	host.generate = func(ctx context.Context, request mosaic.Request) (mosaic.Result, error) {
		generations++
		return mosaic.Generate(ctx, request)
	}
	w := New(application, host)
	w.SetUIQueue(&uitest.UIQueue{})
	if err := w.SetSettings(settings); err != nil {
		t.Fatal(err)
	}
	w.Show(snapshot)
	w.advancedButton.OnTapped()
	test.Tap(w.generateButton)
	settleWindow(t, w)
	w.formatSelect.SetSelected("JPEG")
	_ = w.Window().Canvas().Capture()

	startOverObject, startOver := mosaicActionByLabel(t, w.root, "Start Over")
	previewPosition := application.Driver().AbsolutePositionForObject(w.previewPanel)
	buttonPosition := application.Driver().AbsolutePositionForObject(startOverObject)
	if buttonPosition.X > previewPosition.X+theme.Padding() || buttonPosition.Y > previewPosition.Y+theme.Padding() {
		t.Fatalf("Start Over position = %v, preview starts at %v; want top-left", buttonPosition, previewPosition)
	}
	test.Tap(startOver)

	if !w.config.Visible() || w.previewPanel.Visible() {
		t.Fatalf("after Start Over config visible=%v preview visible=%v", w.config.Visible(), w.previewPanel.Visible())
	}
	if _, ok := w.Result(); ok || w.Status() != "" {
		t.Fatalf("after Start Over result=%v status=%q, want neither", ok, w.Status())
	}
	if generations != 1 {
		t.Fatalf("Start Over caused %d total generations, want 1", generations)
	}
	if w.Target() != "one" || w.Settings() != settings || !sameURIs(w.Snapshot().Sources, snapshot.Sources) {
		t.Fatalf("Start Over changed target, settings, or sources: target=%q settings=%+v", w.Target(), w.Settings())
	}
	if !w.advancedControls.Visible() {
		t.Fatal("Start Over collapsed the retained advanced settings")
	}
	if focused := w.Window().Canvas().Focused(); focused != w.displaySelect {
		t.Fatalf("focus after Start Over = %T, want target display", focused)
	}

	if !w.SelectTarget("two") {
		t.Fatal("could not choose the second display after Start Over")
	}
	test.Tap(w.generateButton)
	settleWindow(t, w)
	result, ok := w.Result()
	if !ok || result.Bounds() != image.Rect(0, 0, 120, 70) || generations != 2 {
		t.Fatalf("second display result=%v bounds=%v generations=%d", ok, result.Bounds(), generations)
	}
	if w.formatSelect.Selected != "JPEG" {
		t.Fatalf("second preview format = %q, want JPEG", w.formatSelect.Selected)
	}
	w.Close()
}

func TestMosaicGenerate_StatusDoesNotOverlapSourceDescription(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	host := successfulHost(t)
	host.generate = func(_ context.Context, request mosaic.Request) (mosaic.Result, error) {
		close(started)
		<-release
		return mosaic.Generate(context.Background(), request)
	}
	application := test.NewApp()
	w := New(application, host)
	w.SetUIQueue(&uitest.UIQueue{})
	w.Show(mustSnapshot(t))

	w.Generate()
	<-started
	_ = w.Window().Canvas().Capture()
	sourcePosition := application.Driver().AbsolutePositionForObject(w.sourceLabel)
	statusPosition := application.Driver().AbsolutePositionForObject(w.status)
	sourceSize := w.sourceLabel.Size()
	statusSize := w.status.Size()
	overlaps := sourcePosition.X < statusPosition.X+statusSize.Width &&
		statusPosition.X < sourcePosition.X+sourceSize.Width &&
		sourcePosition.Y < statusPosition.Y+statusSize.Height &&
		statusPosition.Y < sourcePosition.Y+sourceSize.Height

	close(release)
	settleWindow(t, w)
	w.Close()
	if overlaps {
		t.Fatalf("generation status at %v size %v overlaps source description at %v size %v",
			statusPosition, statusSize, sourcePosition, sourceSize)
	}
}

func TestMosaicPreview_WindowResizeKeepsTargetPixelsAndDoesNotRegenerate(t *testing.T) {
	host := successfulHost(t)
	generations := 0
	host.generate = func(ctx context.Context, request mosaic.Request) (mosaic.Result, error) {
		generations++
		return mosaic.Generate(ctx, request)
	}
	w := New(test.NewApp(), host)
	w.SetUIQueue(&uitest.UIQueue{})
	w.Show(mustSnapshot(t))
	w.Generate()
	settleWindow(t, w)
	before, _ := w.Result()

	w.Window().Resize(fyne.NewSize(1000, 800))
	after, ok := w.Result()
	if !ok || after.Bounds() != image.Rect(0, 0, 80, 50) || !samePixels(before, after) {
		t.Fatal("window resize replaced or resampled the retained target-sized result")
	}
	if generations != 1 {
		t.Fatalf("window resize caused %d generations, want 1", generations)
	}
	w.Close()
}

func TestMosaicRegenerate_PreservesRequestAndChangesSeed(t *testing.T) {
	var requests []mosaic.Request
	host := successfulHost(t)
	host.generate = func(ctx context.Context, request mosaic.Request) (mosaic.Result, error) {
		requests = append(requests, request)
		return mosaic.Generate(ctx, request)
	}
	w := New(test.NewApp(), host)
	w.SetUIQueue(&uitest.UIQueue{})
	seed := int64(40)
	w.SetSeedSource(func() int64 { seed++; return seed })
	w.Show(mustSnapshot(t))
	w.Generate()
	settleWindow(t, w)
	w.Regenerate()
	settleWindow(t, w)

	if len(requests) != 2 {
		t.Fatalf("generation requests = %d, want 2", len(requests))
	}
	first, second := requests[0], requests[1]
	if first.Seed() == second.Seed() {
		t.Fatal("regeneration reused its seed")
	}
	if first.TargetSize() != second.TargetSize() || first.Settings() != second.Settings() ||
		!sameURIs(first.Sources(), second.Sources()) {
		t.Fatal("regeneration changed sources, target, or visual settings")
	}
	w.Close()
}

func TestMosaicStartOver_IsDisabledDuringRegeneration(t *testing.T) {
	release := make(chan struct{})
	started := make(chan struct{})
	calls, wallpaperCalls := 0, 0
	host := successfulHost(t)
	host.generate = func(ctx context.Context, request mosaic.Request) (mosaic.Result, error) {
		calls++
		if calls == 2 {
			close(started)
			<-release
		}
		return mosaic.Generate(ctx, request)
	}
	host.wallpaper = func(context.Context, mosaic.Result, displays.ID) error {
		wallpaperCalls++
		return nil
	}
	w := New(test.NewApp(), host)
	w.SetUIQueue(&uitest.UIQueue{})
	w.Show(mustSnapshot(t))
	w.Generate()
	settleWindow(t, w)
	exports := 0
	w.SetExporter(func(fyne.URI, image.Image, fyne.URI) error { exports++; return nil })

	w.Regenerate()
	<-started
	w.SaveImage()
	w.SetWallpaper()
	assertStartOverBlocked(t, w)
	if exports != 0 || wallpaperCalls != 0 {
		t.Fatalf("busy action gate enabled=%v exports=%d wallpaper=%d", w.PreviewActionsEnabled(), exports, wallpaperCalls)
	}
	close(release)
	settleWindow(t, w)
	assertStartOverRestored(t, w)
	w.Close()
}

func TestMosaicStartOver_IsDisabledDuringExport(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	uitest.StubSaveChooser(t, func(string) ([]byte, error) {
		close(started)
		<-release
		return nil, nil
	})
	w := generatedWindowFromConfig(t)
	w.Generate()
	settleWindow(t, w)

	w.SaveImage()
	<-started
	assertStartOverBlocked(t, w)
	close(release)
	settleWindow(t, w)
	assertStartOverRestored(t, w)
	w.Close()
}

func TestMosaicStartOver_IsDisabledDuringWallpaperChange(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	host := successfulHost(t)
	host.wallpaper = func(context.Context, mosaic.Result, displays.ID) error {
		close(started)
		<-release
		return nil
	}
	w := New(test.NewApp(), host)
	w.SetUIQueue(&uitest.UIQueue{})
	w.Show(mustSnapshot(t))
	w.Generate()
	settleWindow(t, w)

	w.SetWallpaper()
	<-started
	assertStartOverBlocked(t, w)
	close(release)
	settleWindow(t, w)
	assertStartOverRestored(t, w)
	w.Close()
}

func assertStartOverBlocked(t *testing.T, w *Window) {
	t.Helper()
	w.StartOver()
	_, hasResult := w.Result()
	if !w.Busy() || w.PreviewActionsEnabled() || !w.startOverButton.Disabled() ||
		!w.previewPanel.Visible() || w.config.Visible() || !hasResult {
		t.Fatalf("Start Over changed busy preview: busy=%v enabled=%v preview=%v config=%v result=%v",
			w.Busy(), w.PreviewActionsEnabled(), w.previewPanel.Visible(), w.config.Visible(), hasResult)
	}
}

func assertStartOverRestored(t *testing.T, w *Window) {
	t.Helper()
	if !w.PreviewActionsEnabled() || w.startOverButton.Disabled() {
		t.Fatal("completed preview work did not restore Start Over")
	}
}

func TestMosaicSettle_WaitsForWorkStartedByDrainedCompletion(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	host := successfulHost(t)
	host.generate = func(ctx context.Context, request mosaic.Request) (mosaic.Result, error) {
		close(started)
		<-release
		return mosaic.Generate(ctx, request)
	}
	queue := &uitest.UIQueue{}
	w := New(test.NewApp(), host)
	w.SetUIQueue(queue)
	w.Show(mustSnapshot(t))
	queue.Do(w.Generate)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	done := make(chan error, 1)
	go func() { done <- w.Settle(ctx) }()
	<-started
	select {
	case err := <-done:
		t.Fatalf("Settle returned before drained work completed: %v", err)
	default:
	}
	close(release)
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if _, ok := w.Result(); !ok {
		t.Fatal("Settle returned before applying the generated result")
	}
	w.Close()
}

func TestMosaicSupersede_ReverseCompletionPublishesOnlyNewest(t *testing.T) {
	var mu sync.Mutex
	var requests []mosaic.Request
	releases := []chan struct{}{make(chan struct{}), make(chan struct{})}
	started := make(chan int, 2)
	host := successfulHost(t)
	host.generate = func(_ context.Context, request mosaic.Request) (mosaic.Result, error) {
		mu.Lock()
		index := len(requests)
		requests = append(requests, request)
		mu.Unlock()
		started <- index
		<-releases[index]
		return mosaic.Generate(context.Background(), request)
	}
	w := New(test.NewApp(), host)
	w.SetUIQueue(&uitest.UIQueue{})
	seed := int64(10)
	w.SetSeedSource(func() int64 { seed++; return seed })
	w.Show(mustSnapshot(t))

	w.Generate()
	<-started
	w.Regenerate()
	<-started
	close(releases[1])
	close(releases[0])
	settleWindow(t, w)

	mu.Lock()
	newest := requests[1]
	mu.Unlock()
	want, err := mosaic.Generate(context.Background(), newest)
	if err != nil {
		t.Fatal(err)
	}
	got, ok := w.Result()
	if !ok || !samePixels(got, want) {
		t.Fatal("reverse completion did not retain the newest generation")
	}
	w.Close()
}

func TestMosaicFailureKeepsPreview(t *testing.T) {
	calls := 0
	host := successfulHost(t)
	host.generate = func(ctx context.Context, request mosaic.Request) (mosaic.Result, error) {
		calls++
		if calls == 2 {
			return mosaic.Result{}, errors.New("render failed")
		}
		return mosaic.Generate(ctx, request)
	}
	w := New(test.NewApp(), host)
	w.SetUIQueue(&uitest.UIQueue{})
	w.Show(mustSnapshot(t))
	w.Generate()
	settleWindow(t, w)
	before, _ := w.Result()

	logged := captureMosaicLog(t, func() {
		w.Regenerate()
		settleWindow(t, w)
	})
	for _, want := range []string{"failed to generate mosaic", "render failed"} {
		if !strings.Contains(logged, want) {
			t.Errorf("generation log = %q, want it to contain %q", logged, want)
		}
	}
	after, ok := w.Result()
	if !ok || !samePixels(before, after) || !w.PreviewActionsEnabled() {
		t.Fatal("failed regeneration did not preserve the actionable preview")
	}
	w.Close()
}

func TestMosaicWallpaper_PassesLatestResultAndTarget(t *testing.T) {
	var got mosaic.Result
	var target displays.ID
	host := successfulHost(t)
	host.wallpaper = func(_ context.Context, result mosaic.Result, display displays.ID) error {
		got, target = result, display
		return nil
	}
	w := New(test.NewApp(), host)
	w.SetUIQueue(&uitest.UIQueue{})
	w.Show(mustSnapshot(t))
	w.Generate()
	settleWindow(t, w)
	want, _ := w.Result()

	w.SetWallpaper()
	settleWindow(t, w)
	if target != "one" || !samePixels(got, want) {
		t.Fatalf("wallpaper target=%q or pixels did not match the latest result", target)
	}
	if !w.PreviewActionsEnabled() {
		t.Fatal("wallpaper completion did not re-enable preview actions")
	}
	w.Close()
}

func TestMosaicWallpaper_TargetUnsupportedKeepsPreviewAndExplainsSave(t *testing.T) {
	host := successfulHost(t)
	host.wallpaper = func(context.Context, mosaic.Result, displays.ID) error {
		return &wallpaper.TargetUnsupportedError{Platform: "Linux"}
	}
	w := New(test.NewApp(), host)
	w.SetUIQueue(&uitest.UIQueue{})
	w.Show(mustSnapshot(t))
	w.Generate()
	settleWindow(t, w)
	before, _ := w.Result()

	logged := captureMosaicLog(t, func() {
		w.SetWallpaper()
		settleWindow(t, w)
	})
	for _, want := range []string{"failed to set mosaic wallpaper", "Linux"} {
		if !strings.Contains(logged, want) {
			t.Errorf("wallpaper log = %q, want it to contain %q", logged, want)
		}
	}
	after, ok := w.Result()
	if !ok || !samePixels(before, after) || !w.PreviewActionsEnabled() {
		t.Fatal("unsupported target replaced the preview or left its actions disabled")
	}
	if status := w.Status(); !strings.Contains(status, "Save Image") {
		t.Fatalf("unsupported status = %q, want a Save Image alternative", status)
	}
	w.Close()
}

func TestMosaicClose_CancelsAndSettlesWithoutLateMutation(t *testing.T) {
	started := make(chan struct{})
	cancelled := make(chan struct{})
	host := successfulHost(t)
	host.generate = func(ctx context.Context, _ mosaic.Request) (mosaic.Result, error) {
		close(started)
		<-ctx.Done()
		close(cancelled)
		return mosaic.Result{}, ctx.Err()
	}
	w := New(test.NewApp(), host)
	w.SetUIQueue(&uitest.UIQueue{})
	w.Show(mustSnapshot(t))
	w.Generate()
	<-started

	w.Close()
	<-cancelled
	settleWindow(t, w)
	if w.Opened() || w.Busy() {
		t.Fatal("closed mosaic retained open/busy state")
	}
	if got := w.Snapshot(); len(got.Sources) != 0 {
		t.Fatalf("closed mosaic retained %d sources", len(got.Sources))
	}
}

func successfulHost(t *testing.T) *fakeHost {
	t.Helper()
	return &fakeHost{
		generate:  mosaic.Generate,
		inspect:   func() (displays.Snapshot, error) { return testTopology("one", 80, 50), nil },
		wallpaper: func(context.Context, mosaic.Result, displays.ID) error { return nil },
	}
}

func generatedWindowFromConfig(t *testing.T) *Window {
	t.Helper()
	w := New(test.NewApp(), successfulHost(t))
	w.SetUIQueue(&uitest.UIQueue{})
	w.Show(mustSnapshot(t))
	return w
}

func assertFocusCycle(t *testing.T, canvas fyne.Canvas, order []fyne.Focusable) {
	t.Helper()
	canvas.Focus(order[0])
	for index, want := range order {
		if got := canvas.Focused(); got != want {
			gotLabel, wantLabel := "", ""
			if accessible, ok := got.(fyne.Accessible); ok {
				gotLabel = accessible.AccessibilityLabel()
			}
			if accessible, ok := want.(fyne.Accessible); ok {
				wantLabel = accessible.AccessibilityLabel()
			}
			t.Fatalf("focus %d = %T %q, want %T %q", index, got, gotLabel, want, wantLabel)
		}
		canvas.FocusNext()
	}
	if got := canvas.Focused(); got != order[0] {
		t.Fatalf("wrapped focus = %T, want %T", got, order[0])
	}
	canvas.FocusPrevious()
	if got := canvas.Focused(); got != order[len(order)-1] {
		t.Fatalf("reverse focus = %T, want %T", got, order[len(order)-1])
	}
}

func containsMosaicObject(root *fyne.Container, target fyne.CanvasObject) bool {
	for _, object := range root.Objects {
		if object == target {
			return true
		}
		if child, ok := object.(*fyne.Container); ok && containsMosaicObject(child, target) {
			return true
		}
	}

	return false
}

func mosaicActionByLabel(t *testing.T, root fyne.CanvasObject, label string) (fyne.CanvasObject, fyne.Tappable) {
	t.Helper()
	var object fyne.CanvasObject
	var action fyne.Tappable
	var visit func(fyne.CanvasObject)
	visit = func(candidate fyne.CanvasObject) {
		if object != nil || !candidate.Visible() {
			return
		}
		accessible, accessibleOK := candidate.(fyne.Accessible)
		tappable, tappableOK := candidate.(fyne.Tappable)
		if accessibleOK && tappableOK && accessible.AccessibilityLabel() == label {
			object, action = candidate, tappable
			return
		}
		if container, ok := candidate.(*fyne.Container); ok {
			for _, child := range container.Objects {
				visit(child)
			}
		}
	}
	visit(root)
	if object == nil {
		t.Fatalf("visible mosaic UI has no accessible %q action", label)
	}

	return object, action
}

func mustSnapshot(t *testing.T) Snapshot {
	t.Helper()
	snapshot, err := NewSnapshot(
		[]fyne.URI{uitest.TempJPEGURI(t, "source.jpg", 8, 6, color.White)},
		SourceResult,
		testTopology("one", 80, 50),
	)
	if err != nil {
		t.Fatal(err)
	}

	return snapshot
}

func testTopology(id string, width, height int) displays.Snapshot {
	return displays.Snapshot{
		Displays: []displays.Display{{ID: displays.ID(id), Name: "Display " + id, Bounds: image.Rect(0, 0, width, height)}},
		Default:  displays.ID(id),
	}
}

func settleWindow(t *testing.T, window *Window) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := window.Settle(ctx); err != nil {
		t.Fatal(err)
	}
}

func captureMosaicLog(t *testing.T, action func()) string {
	t.Helper()
	var output bytes.Buffer
	restore := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(restore) })

	action()

	log.SetOutput(restore)
	return output.String()
}

func samePixels(a, b mosaic.Result) bool {
	aImage := a.Image().(*image.NRGBA)
	bImage := b.Image().(*image.NRGBA)
	if aImage.Bounds() != bImage.Bounds() {
		return false
	}
	for index := range aImage.Pix {
		if aImage.Pix[index] != bImage.Pix[index] {
			return false
		}
	}

	return true
}

func sameURIs(a, b []fyne.URI) bool {
	if len(a) != len(b) {
		return false
	}
	for index := range a {
		if a[index].String() != b[index].String() {
			return false
		}
	}
	return true
}
