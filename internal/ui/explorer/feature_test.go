package explorer_test

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/test"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/explorerpresets"
	"github.com/frathe/picfetch/internal/explorertrial"
	"github.com/frathe/picfetch/internal/similarity"
	"github.com/frathe/picfetch/internal/ui/explorer"
	"github.com/frathe/picfetch/internal/uitest"
)

func TestFeaturePresetSavePreviewAndApply(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	host := &featureHost{win: app.NewWindow("Explorer")}
	preview := uitest.EncodeJPEG(t, 32, 24, color.White)
	provider := func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
		var items []similarity.Item
		for _, path := range paths {
			items = append(items, similarity.Item{Path: path, Cohort: "unassigned", Preview: preview, Facts: similarity.ImageFacts{Version: 1, Make: "Canon"}})
		}
		emit(similarity.Event{Total: len(paths), Successful: len(paths), Complete: true, Items: items})
		return nil
	}
	f := explorer.NewFeature(host, explorer.Options{App: app, Queue: &uitest.UIQueue{}, Analyze: provider, Presets: &explorerpresets.Store{Dir: t.TempDir()}})
	t.Cleanup(func() { f.Stop(); f.Settle() })
	host.win.SetContent(f.Surface().Overlay())
	f.Open(explorer.OpenRequest{Sources: []string{"/a.jpg", "/b.jpg"}})
	f.Settle()
	f.ShowSimilarityPresets()
	f.Settle()
	test.Tap(featureButton(t, host, "New preset"))
	featureEntry(t, host, "Preset name", "Saved cameras")
	featureEntry(t, host, "Camera make", "Canon")
	test.Tap(featureButton(t, host, "Save preset"))
	f.Settle()
	test.Tap(featureButton(t, host, "Saved cameras"))
	test.Tap(featureButton(t, host, "Preview preset"))
	f.Settle()
	if featureButton(t, host, "Apply preset").Disabled() {
		t.Fatal("reviewed matching preset could not be applied")
	}
	test.Tap(featureButton(t, host, "Apply preset"))
	f.Settle()
	if host.win.Canvas().Overlays().Top() != nil {
		t.Fatal("applied preset left its review dialog open")
	}
	if len(f.Surface().Cohorts().Groups) != 1 {
		t.Fatal("applying the reviewed preset did not create one named cohort")
	}
}

func featureButton(t *testing.T, host *featureHost, label string) *widget.Button {
	t.Helper()
	var found *widget.Button
	walkFeature(host.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
		if b, ok := o.(*widget.Button); ok && b.Text == lang.L(label) {
			found = b
		}
	})
	if found == nil {
		t.Fatalf("missing dialog button %q", label)
	}
	return found
}
func featureEntry(t *testing.T, host *featureHost, placeholder, value string) {
	t.Helper()
	var found *widget.Entry
	walkFeature(host.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
		if e, ok := o.(*widget.Entry); ok && e.PlaceHolder == lang.L(placeholder) {
			found = e
		}
	})
	if found == nil {
		t.Fatalf("missing dialog entry %q", placeholder)
	}
	found.SetText(value)
}

func TestFeatureSetupRequiresAcceptance(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	host := &featureHost{win: app.NewWindow("Explorer")}
	f := explorer.NewFeature(host, explorer.Options{App: app, Queue: &uitest.UIQueue{}, Supported: true, AssetsReady: true})
	t.Cleanup(func() { f.Stop(); f.Settle() })
	accepted := false
	if f.EnsureReady(func() { accepted = true }) {
		t.Fatal("first-use setup skipped acceptance")
	}
	if accepted || f.Settings().IntroSeen {
		t.Fatal("showing setup acknowledged it")
	}
	var next *widget.Button
	downloadShown := false
	downloadText := fmt.Sprintf(lang.L("One-time download: about %.0f MB. After that, everything works offline."), float64(similarity.AssetDownloadBytes())/1e6)
	walkFeature(host.win.Canvas().Overlays().Top(), func(o fyne.CanvasObject) {
		if label, ok := o.(*widget.Label); ok && label.Text == downloadText {
			downloadShown = true
		}
		if b, ok := o.(*widget.Button); ok && b.Text == lang.L("Continue") {
			next = b
		}
	})
	if next == nil || !downloadShown {
		t.Fatal("setup did not present its explanation and Continue")
	}
	test.Tap(next)
	f.Settle()
	if !accepted || !f.Settings().IntroSeen || host.win.Canvas().Overlays().Top() != nil {
		t.Fatal("acceptance did not retire setup and release its caller")
	}
}

func TestFeatureSourceReplacementRetiresPresetDialog(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	host := &featureHost{win: app.NewWindow("Explorer")}
	provider := func(_ context.Context, _ []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
		emit(similarity.Event{Complete: true})
		return nil
	}
	f := explorer.NewFeature(host, explorer.Options{App: app, Queue: &uitest.UIQueue{}, Analyze: provider, Presets: &explorerpresets.Store{Dir: t.TempDir()}})
	t.Cleanup(func() { f.Stop(); f.Settle() })
	f.Open(explorer.OpenRequest{Sources: []string{"/old.jpg"}})
	f.Settle()
	f.ShowSimilarityPresets()
	f.Wait() // The old library read has queued its UI delivery.
	if !f.State().DialogOpen {
		t.Fatal("fixture did not open the preset library")
	}
	f.Open(explorer.OpenRequest{Sources: []string{"/new.jpg"}})
	f.Settle()
	if f.State().DialogOpen || host.win.Canvas().Overlays().Top() != nil {
		t.Fatal("source replacement retained the old preset dialog or its queued delivery")
	}
}

func walkFeature(o fyne.CanvasObject, visit func(fyne.CanvasObject)) {
	if o == nil || !o.Visible() {
		return
	}
	visit(o)
	switch o := o.(type) {
	case *fyne.Container:
		for _, child := range o.Objects {
			walkFeature(child, visit)
		}
	case fyne.Widget:
		for _, child := range test.WidgetRenderer(o).Objects() {
			walkFeature(child, visit)
		}
	}
}

func TestFeatureOpenCopiesSourcesAndRejectsClosedDelivery(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	host := &featureHost{win: app.NewWindow("Explorer")}
	queue := &uitest.UIQueue{}
	started, release := make(chan struct{}), make(chan struct{})
	var captured []string
	provider := func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
		close(started)
		<-release
		captured = slices.Clone(paths)
		emit(similarity.Event{Complete: true, Successful: 1, Total: 1, Items: []similarity.Item{{Path: paths[0], Cohort: "a", Position: []float32{0, 0}}}})
		return nil
	}
	f := explorer.NewFeature(host, explorer.Options{Analyze: provider, Queue: queue})
	t.Cleanup(func() { f.Stop(); f.Settle() })
	paths := []string{"/original.jpg"}
	if !f.Open(explorer.OpenRequest{Sources: paths}) {
		t.Fatal("opening a prepared collection did not admit analysis")
	}
	<-started
	paths[0] = "/replacement.jpg"
	close(release)
	f.Wait()
	f.Close()
	f.Settle()
	if !slices.Equal(captured, []string{"/original.jpg"}) {
		t.Fatalf("analysis source snapshot changed with its caller: %v", captured)
	}
	if state := f.State(); state.Complete || state.HasMap {
		t.Fatalf("closed analysis installed its queued result: %+v", state)
	}
}

type featureHost struct {
	win    fyne.Window
	grid   bool
	toasts []string
}

func (h *featureHost) Window() fyne.Window             { return h.win }
func (h *featureHost) Changed()                        {}
func (h *featureHost) BrowseCohort(_ []string, _ bool) { h.grid = true }
func (h *featureHost) LeaveExplorer()                  {}
func (h *featureHost) ReturnToMap()                    { h.grid = false }
func (h *featureHost) Presentation() explorer.Presentation {
	surface := "map"
	if h.grid {
		surface = "grid"
	}
	return explorer.Presentation{Surface: surface, GridVisible: h.grid}
}
func (h *featureHost) ShowToast(message string)    { h.toasts = append(h.toasts, message) }
func (h *featureHost) Unfocus()                    { h.win.Canvas().Unfocus() }
func (h *featureHost) Modifiers() fyne.KeyModifier { return 0 }

func (h *featureHost) Repaint() {}

func TestFeatureLimitFailures(t *testing.T) {
	for _, limitErr := range []error{similarity.ErrAnalysisItemLimit, similarity.ErrAnalysisMemoryLimit} {
		for _, stale := range []bool{false, true} {
			t.Run(fmt.Sprintf("%v/stale=%v", limitErr, stale), func(t *testing.T) {
				app := test.NewApp()
				t.Cleanup(app.Quit)
				host := &featureHost{win: app.NewWindow("Explorer")}
				f := explorer.NewFeature(host, explorer.Options{App: app, Queue: &uitest.UIQueue{}, AssetsReady: true,
					Analyze: func(_ context.Context, _ []string, _ <-chan similarity.Control, _ func(similarity.Event)) error {
						return fmt.Errorf("analysis failed: %w", limitErr)
					}})
				t.Cleanup(func() { f.Stop(); f.Settle() })
				f.Open(explorer.OpenRequest{Sources: []string{"/a.jpg"}})
				f.Wait()
				if stale {
					f.Close()
				}
				f.Settle()
				if stale {
					if len(host.toasts) != 0 {
						t.Fatalf("stale failure showed toast: %v", host.toasts)
					}
					return
				}
				want := lang.L("Item limit exceeded. Adjust in Settings -> Limits -> Similarity Explorer.")
				if errors.Is(limitErr, similarity.ErrAnalysisMemoryLimit) {
					want = lang.L("Memory limit exceeded. Adjust in Settings -> Limits -> Similarity Explorer.")
				}
				if !slices.Equal(host.toasts, []string{want}) {
					t.Fatalf("limit failure toast = %v, want %q", host.toasts, want)
				}
				if state := f.State(); !state.CanRetry || !state.AssetsReady || state.Complete {
					t.Fatalf("limit failure left invalid retry state: %+v", state)
				}
			})
		}
	}
}

func TestFeatureCloseReopenAndStop(t *testing.T) {
	for _, action := range []string{"close", "reopen", "stop"} {
		t.Run(action, func(t *testing.T) {
			app := test.NewApp()
			t.Cleanup(app.Quit)
			host := &featureHost{win: app.NewWindow("Explorer")}
			oldQueue := &uitest.UIQueue{}
			queued := make(chan struct{})
			provider := func(ctx context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
				emit(similarity.Event{Complete: true, Total: 1, Successful: 1, Items: []similarity.Item{{Path: paths[0], Cohort: "unassigned"}}})
				close(queued)
				<-ctx.Done()
				return ctx.Err()
			}
			f := explorer.NewFeature(host, explorer.Options{Analyze: provider, Queue: oldQueue})
			t.Cleanup(func() { f.Stop(); f.Settle() })
			f.Open(explorer.OpenRequest{Sources: []string{"/old.jpg"}})
			<-queued
			if action == "stop" {
				f.Stop()
			} else {
				f.Close()
			}
			f.Wait()
			options := f.Options()
			options.Queue = &uitest.UIQueue{}
			options.Analyze = func(_ context.Context, paths []string, _ <-chan similarity.Control, emit func(similarity.Event)) error {
				emit(similarity.Event{Complete: true, Total: 1, Successful: 1, Items: []similarity.Item{{Path: paths[0], Cohort: "unassigned"}}})
				return nil
			}
			f.Configure(options)
			if action != "close" {
				admitted := f.Open(explorer.OpenRequest{Sources: []string{"/new.jpg"}})
				if admitted != (action == "reopen") {
					t.Fatalf("wrong admission after %s", action)
				}
			}
			f.Settle()
			oldQueue.Drain()
			if action == "reopen" {
				if !f.State().Complete || !slices.Equal(f.Sources(), []string{"/new.jpg"}) {
					t.Fatal("old queued delivery replaced the new workflow")
				}
				candidates := f.Surface().PreparePreset(explorerpresets.Preset{}).Candidates
				if len(candidates) != 1 || candidates[0].Path != "/new.jpg" {
					t.Fatal("old queued delivery replaced the reopened map's contents")
				}
			} else if f.State().HasMap || f.Surface().Visible() {
				t.Fatal("retired delivery reopened the feature")
			}
		})
	}
}

func TestFeatureCanceledBarrierRecordsTrialExit(t *testing.T) {
	app := test.NewApp()
	t.Cleanup(app.Quit)
	dir := filepath.Join(t.TempDir(), "trial")
	trial, err := explorertrial.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	host := &featureHost{win: app.NewWindow("Explorer")}
	analyzed := false
	f := explorer.NewFeature(host, explorer.Options{App: app, Queue: &uitest.UIQueue{}, Trial: trial, Analyze: func(_ context.Context, _ []string, _ <-chan similarity.Control, _ func(similarity.Event)) error {
		analyzed = true
		return nil
	}})
	barrier := make(chan struct{})
	f.WaitBefore(barrier)
	f.Open(explorer.OpenRequest{Sources: []string{"/a.jpg"}})
	f.Close()
	close(barrier)
	f.Settle()
	f.Stop()
	_ = trial.Close() // A canceled-only trial intentionally remains uncollected.
	data, err := os.ReadFile(filepath.Join(dir, "session.json"))
	if err != nil {
		t.Fatal(err)
	}
	var summary explorertrial.Summary
	if err := json.Unmarshal(data, &summary); err != nil {
		t.Fatal(err)
	}
	if analyzed || len(summary.Analyses) != 1 || summary.Analyses[0].Outcome != "canceled" {
		t.Fatalf("canceled predecessor wait lost exit observation: analyzed=%v summary=%+v", analyzed, summary)
	}
}
