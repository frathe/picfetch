package explorer

import (
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/explorerpresets"
	"github.com/frathe/picfetch/internal/explorertrial"
	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/similarity"
)

// WorkflowHost owns collection and cross-feature navigation decisions.
type WorkflowHost interface {
	Window() fyne.Window
	Changed()
	Repaint()
	BrowseCohort([]string, bool)
	LeaveExplorer()
	ReturnToMap()
	Presentation() Presentation
	ShowToast(string)
	Unfocus()
	Modifiers() fyne.KeyModifier
}

// Presentation describes the root-owned browsing surface for trial records.
type Presentation struct {
	GridVisible, ComparisonActive bool
	Surface                       string
	Paths                         []string
}

// UIQueue marshals worker deliveries and permits deterministic test settlement.
type UIQueue interface {
	Do(func())
	Drain() bool
}
type featureQueue struct{}

func (featureQueue) Do(f func()) { fyne.Do(f) }
func (featureQueue) Drain() bool { return false }

// Settings holds standing Explorer preferences; it contains no live workflow state.
type Settings struct{ CacheFavorites, AutoFit, Automatic, IntroSeen bool }

// Options supplies the existing native and UI adapters. Configure on UI before
// admission; workers capture their provider/client/store when they start.
type Options struct {
	App                    fyne.App
	Discussions            func()
	Client                 similarity.Client
	Analyze                similarity.Provider
	Queue                  UIQueue
	Presets                *explorerpresets.Store
	Supported, AssetsReady bool
	Settings               Settings
	Trial                  *explorertrial.Session
}

// OpenRequest captures a prepared collection and its Favorite ownership.
type OpenRequest struct {
	Sources                   []string
	FavoriteDir, FavoritesDir string
	GeneralAnalysisDir        string
}

// State is a value observation; Revision identifies the current source workflow.
type State struct {
	DialogOpen                                           bool
	Complete, HasMap, HasSources, CanRetry               bool
	SetupOpen, AssetsReady, SessionCurrent, CohortSaving bool
	Available                                            int
	Revision                                             uint64
}

// Feature owns Explorer's workflow independently of the viewer.
type Feature struct {
	analysisDone            chan struct{}
	analysisBefore          <-chan struct{}
	preparing               bool
	setup                   *explorerSetup
	host                    WorkflowHost
	app                     fyne.App
	discussions             func()
	win                     fyne.Window
	analyze                 similarity.Provider
	stopping                bool
	introSeen, assetsReady  bool
	client                  similarity.Client
	supported               bool
	trial                   *explorertrial.Session
	trialRun, trialEvent    int
	cacheFavorites, autoFit bool
	lifecycle               requestLifecycle
	token                   requestToken
	workers, presetWorkers  sync.WaitGroup
	ui                      UIQueue
	surface                 *Map
	sources, cohort         []string
	unassignedCohort        bool
	cohortDialog            dialog.Dialog
	favoriteDir             string
	cohortStore             *favstore.CohortStore
	cohortLoadErr           error
	cohortSaving            bool
	presets                 *explorerpresets.Store
	presetOp                requestLifecycle
	complete, hasMap        bool
	controls                chan similarity.Control
	available, mapped       int
	building, automatic     bool
}

func NewFeature(host WorkflowHost, options Options) *Feature {
	f := &Feature{host: host, win: host.Window()}
	f.surface = New(f)
	f.Configure(options)
	return f
}

// Configure replaces runtime adapters on UI. Existing workers retain captured
// providers and stores. Replace Queue only after Wait has joined those workers.
func (f *Feature) Configure(options Options) {
	f.app, f.discussions = options.App, options.Discussions
	f.client, f.analyze = options.Client, options.Analyze
	f.ui = options.Queue
	if f.ui == nil {
		f.ui = featureQueue{}
	}
	f.presets = options.Presets
	if f.presets == nil {
		f.presets = &explorerpresets.Store{Dir: explorerpresets.DefaultDir()}
	}
	f.supported, f.assetsReady, f.trial = options.Supported, options.AssetsReady, options.Trial
	f.ApplySettings(options.Settings)
}
func (f *Feature) Options() Options {
	return Options{
		App: f.app, Discussions: f.discussions,
		Client: f.client, Analyze: f.analyze, Queue: f.ui, Presets: f.presets,
		Supported: f.supported, AssetsReady: f.assetsReady,
		Settings: f.Settings(), Trial: f.trial,
	}
}
func (f *Feature) Settings() Settings {
	return Settings{CacheFavorites: f.cacheFavorites, AutoFit: f.autoFit, Automatic: f.automatic, IntroSeen: f.introSeen}
}
func (f *Feature) ApplySettings(settings Settings) {
	f.cacheFavorites, f.autoFit, f.introSeen = settings.CacheFavorites, settings.AutoFit, settings.IntroSeen
	if f.automatic != settings.Automatic {
		f.SetSimilarityAutoUpdate(settings.Automatic)
	}
}
func (f *Feature) State() State {
	return State{
		DialogOpen: f.cohortDialog != nil,
		Complete:   f.complete, HasMap: f.hasMap, HasSources: len(f.sources) > 0,
		CanRetry:  !f.complete && f.controls == nil && f.setup == nil && !f.preparing,
		SetupOpen: f.setup != nil, AssetsReady: f.assetsReady,
		SessionCurrent: f.token.current(), CohortSaving: f.cohortSaving,
		Available: f.available, Revision: f.lifecycle.currentRevision(),
	}
}
func (f *Feature) Surface() *Map            { return f.surface }
func (f *Feature) Cohort() ([]string, bool) { return slices.Clone(f.cohort), f.unassignedCohort }
func (f *Feature) HasCohort() bool          { return len(f.cohort) > 0 }
func (f *Feature) RemoveCohortSource(path string) {
	f.cohort = slices.DeleteFunc(f.cohort, func(p string) bool { return p == path })
}

// Preparing presents root-owned duplicate preparation without starting analysis.
func (f *Feature) Preparing() {
	if f.stopping {
		return
	}
	f.retireAnalysis()
	f.cohort = nil
	f.preparing = true
	f.surface.Show()
	f.surface.Status(lang.L("Checking duplicate groups..."))
	f.host.Repaint()
}
func (f *Feature) Close() {
	f.retireAnalysis()
	f.surface.Hide()
	f.cohort = nil
}
func (f *Feature) Stop() {
	f.stopping = true
	f.Close()
}
func (f *Feature) Wait() {
	f.workers.Wait()
	f.presetWorkers.Wait()
}

// Settle joins and delivers until idle, reporting whether it delivered UI work.
// Root uses that fact to repeat settlement of Grid work admitted by delivery.
func (f *Feature) Settle() bool {
	delivered := false
	for {
		f.Wait()
		if !f.ui.Drain() {
			return delivered
		}
		delivered = true
	}
}
func (f *Feature) SourcesChanged() {
	if len(f.sources) == 0 && f.setup == nil && !f.preparing {
		return
	}
	f.retireAnalysis()
	f.surface.Status(lang.L("Source files changed. Open the explorer to analyze again."))
	f.host.Changed()
}
func (f *Feature) retireAnalysis() {
	f.closeExplorerSetup()
	f.preparing = false
	if len(f.sources) > 0 {
		f.trial.Action(f.trialRun, "explorer-exit", len(f.sources))
	}
	f.presetOp.invalidate()
	if f.cohortDialog != nil {
		f.cohortDialog.Hide()
	}
	f.controls = nil
	f.lifecycle.invalidate()
	f.cohortStore, f.cohortLoadErr = nil, nil
	f.cohortSaving = false
	f.surface.SetResult(nil, nil)
	f.surface.UpdateState(false, false)
	f.sources = nil
	f.complete, f.hasMap = false, false
}
func (f *Feature) OpenSimilarityCohort(paths []string)     { f.openCohort(paths, false) }
func (f *Feature) OpenSimilarityUnassigned(paths []string) { f.openCohort(paths, true) }
func (f *Feature) openCohort(paths []string, unassigned bool) {
	if f.stopping || len(paths) == 0 {
		return
	}
	f.RecordView("map-departure")
	f.cohort, f.unassignedCohort = slices.Clone(paths), unassigned
	f.host.BrowseCohort(slices.Clone(paths), unassigned)
	f.host.Changed()
	f.RecordView("cohort-open")
}
func (f *Feature) LeaveSimilarityMap() { f.host.LeaveExplorer() }

func (f *Feature) Unfocus()                            { f.host.Unfocus() }
func (f *Feature) Modifiers() fyne.KeyModifier         { return f.host.Modifiers() }
func (f *Feature) Trial() *explorertrial.Session       { return f.trial }
func (f *Feature) RecordAction(kind string, count int) { f.trial.Action(f.trialRun, kind, count) }
func (f *Feature) RecordView(kind string) {
	if f.trial == nil {
		return
	}
	shown := f.host.Presentation()
	var view *explorertrial.MapView
	if shown.Surface == "map" {
		camera := f.surface.View()
		view = &explorertrial.MapView{Piles: camera.Piles, Zoom: camera.Zoom, MinimumZoom: camera.MinimumZoom, CenterX: camera.Center.X, CenterY: camera.Center.Y, Width: camera.Size.Width, Height: camera.Size.Height}
	}
	f.trial.Presented(f.trialRun, f.trialEvent, kind, shown.Surface, shown.Paths, view)
}

// ReturnToMap completes a feature-owned edit through the root navigation adapter.
func (f *Feature) ReturnToMap() {
	f.host.ReturnToMap()
	f.surface.Show()
	f.host.Repaint()
	f.host.Changed()
	f.RecordView("map-return")
}

// SettlePresets observes preset completion without waiting on a streaming analysis.
func (f *Feature) SettlePresets() {
	for {
		f.presetWorkers.Wait()
		if !f.ui.Drain() {
			return
		}
	}
}

func (f *Feature) Sources() []string { return slices.Clone(f.sources) }

// WaitBefore serializes the next analysis with another feature's native worker.
// The supplied completion includes all of that feature's retired generations.
func (f *Feature) WaitBefore(done <-chan struct{}) { f.analysisBefore = done }

// Suspend cancels worker delivery while retaining the last map, its camera and
// frozen cohort. The returned signal includes native worker exit; callers wait
// off UI before starting a different native analysis.
func (f *Feature) Suspend() <-chan struct{} {
	f.closeExplorerSetup()
	f.lifecycle.invalidate()
	f.controls = nil
	f.preparing = false
	f.surface.UpdateState(false, false)
	if f.analysisDone != nil {
		return f.analysisDone
	}
	done := make(chan struct{})
	close(done)
	return done
}
