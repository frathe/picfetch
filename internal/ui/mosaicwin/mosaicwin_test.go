package mosaicwin

import (
	"context"
	"errors"
	"image"
	"image/color"
	"strings"
	"sync"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/test"

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

func TestMosaicAccessibility_InteractiveControlsHaveMeaningfulNames(t *testing.T) {
	w := New(test.NewApp(), successfulHost(t))
	w.Show(mustSnapshot(t))
	controls := []fyne.CanvasObject{
		w.displaySelect, w.refreshButton, w.minimum, w.frameSelect, w.advancedButton,
		w.variation, w.overlap, w.rotation, w.generateButton, w.cancelButton,
		w.formatSelect, w.regenerateButton, w.wallpaperButton, w.saveButton, w.closeButton,
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
		w.displaySelect, w.refreshButton, w.minimum, w.frameSelect, w.advancedButton, w.generateButton, w.cancelButton,
	})

	w.advancedButton.OnTapped()
	assertFocusCycle(t, canvas, []fyne.Focusable{
		w.displaySelect, w.refreshButton, w.minimum, w.frameSelect, w.advancedButton,
		w.variation, w.overlap, w.rotation, w.generateButton, w.cancelButton,
	})

	w.Generate()
	settleWindow(t, w)
	assertFocusCycle(t, canvas, []fyne.Focusable{
		w.formatSelect, w.regenerateButton, w.wallpaperButton, w.saveButton, w.closeButton,
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
	handler(&fyne.KeyEvent{Name: fyne.KeyEscape})
	settleWindow(t, w)
	if !w.Opened() || w.Busy() {
		t.Fatalf("first Escape open=%v busy=%v, want cancelled and open", w.Opened(), w.Busy())
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

func TestMosaicTarget_CannotRetargetOrRefreshWhileGenerationIsBusy(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	inspections := 0
	host := successfulHost(t)
	host.generate = func(ctx context.Context, request mosaic.Request) (mosaic.Result, error) {
		close(started)
		<-release
		return mosaic.Generate(ctx, request)
	}
	host.inspect = func() (displays.Snapshot, error) {
		inspections++
		return testTopology("two", 100, 60), nil
	}
	topology := displays.Snapshot{
		Displays: []displays.Display{
			{ID: "one", Name: "One", Bounds: image.Rect(0, 0, 80, 50)},
			{ID: "two", Name: "Two", Bounds: image.Rect(80, 0, 180, 60)},
		},
		Default: "one",
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
	if inspections != 0 || w.Target() != "one" {
		t.Fatalf("busy refresh inspections=%d target=%q, want no refresh and target one", inspections, w.Target())
	}
	if !w.displaySelect.Disabled() || !w.minimum.Disabled() || !w.frameSelect.Disabled() || !w.advancedButton.Disabled() {
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

func TestMosaicActionGate_DisablesPreviewEffectsDuringRegeneration(t *testing.T) {
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
	if w.PreviewActionsEnabled() || exports != 0 || wallpaperCalls != 0 {
		t.Fatalf("busy action gate enabled=%v exports=%d wallpaper=%d", w.PreviewActionsEnabled(), exports, wallpaperCalls)
	}
	close(release)
	settleWindow(t, w)
	if !w.PreviewActionsEnabled() {
		t.Fatal("completed regeneration did not restore preview actions")
	}
	w.Close()
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

	w.Regenerate()
	settleWindow(t, w)
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

	w.SetWallpaper()
	settleWindow(t, w)
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
		inspect:   func() (displays.Snapshot, error) { return testTopology("one", 1920, 1080), nil },
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
