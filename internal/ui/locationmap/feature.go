// Package locationmap presents the geographic distribution of a collection.
package locationmap

import (
	"context"
	"image"
	"slices"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/fileidentity"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/widgets"
)

// Host owns transitions out of the geographic surface.
type Host interface {
	LeaveLocationMap()
	LocationMapChanged()
	OpenLocationImage(fileidentity.Occurrence)
	OpenLocationCluster([]fileidentity.Occurrence)
	ShowToast(string)
	Unfocus()
}

// Feature owns the map surface and its browsing session.
type Feature struct {
	host               Host
	overlay            *fyne.Container
	workers            sync.WaitGroup
	ui                 UIQueue
	options            Options
	ctx                context.Context
	cancel             context.CancelFunc
	previewCancel      context.CancelFunc
	generation         uint64
	stopped            bool
	active             bool
	points             []Point
	status             *widget.Label
	surface            *Surface
	counts             Counts
	slots              chan struct{}
	thumbs             imaging.CacheWriter[image.Image]
	displayed          fileidentity.Occurrence
	tiles              *TileStore
	tileCancel         context.CancelFunc
	tileView           tileView
	paintedTileView    tileView
	tileRevision       uint64
	tileWait           func(context.Context, time.Duration) error
	retryWorkers       sync.WaitGroup
	lastTileToast      time.Time
	tileSlots          chan struct{}
	facts              *FactCache
	favoriteRoot       atomic.Value
	persistence        sync.Mutex
	sources            []fyne.URI
	lifetime           context.Context
	stopLifetime       context.CancelFunc
	cacheWarned        bool
	versions           map[string]string
	validationCancel   context.CancelFunc
	validationRevision uint64
	diskBypass         map[string]bool
}

// SetDisplayed tracks published image content, independently of a pending request.
func (f *Feature) SetDisplayed(identity fileidentity.Occurrence) {
	if f.displayed == identity {
		return
	}
	f.displayed = identity
	if f.Visible() {
		f.surface.arrange()
	}
}

// Source retains a collection occurrence independently of root indexes.
type Source struct {
	URI      fyne.URI
	Identity fileidentity.Occurrence
	Pixels   int64
	Donors   []Source
}
type Point struct {
	Source   Source
	Metadata imaging.Metadata
	Version  string
	Donor    Source
}
type Counts struct {
	Total, Completed, Located, Unlocated, Failed, Conflicts int
	Complete                                                bool
}
type Options struct {
	Context    func(context.Context) context.Context
	Thumbnails func() imaging.CacheWriter[image.Image]
}

func New(host Host, options Options) *Feature {
	f := &Feature{host: host, ui: fyneQueue{}, options: options, slots: make(chan struct{}, 2)}
	f.tileSlots = make(chan struct{}, 4)
	f.facts = NewFactCache()
	f.diskBypass = map[string]bool{}
	f.favoriteRoot.Store("")
	f.lifetime, f.stopLifetime = context.WithCancel(context.Background())
	f.ConfigureTiles(TileOptions{}, nil)
	f.status = widget.NewLabel("")
	f.surface = newSurface(f)
	clipped := container.NewClip(f.surface)
	// Border may resize its center before asking for MinSize. Fyne 2.8's
	// Clip constructor defers BaseWidget initialization until that query.
	clipped.ExtendBaseWidget(clipped)
	content := container.NewBorder(container.NewHBox(widget.NewLabel(lang.L("Location Map")),
		widget.NewButton(lang.L("Fit All"), func() { f.surface.fit(); host.Unfocus() }),
		widget.NewButton(lang.L("Back to Viewer"), host.LeaveLocationMap)), container.NewVBox(f.status, widget.NewLabel(lang.L("© OpenStreetMap contributors"))), nil, nil, clipped)
	f.overlay = container.NewStack(widgets.NewThemedRectangle(theme.ColorNameBackground), content)
	f.overlay.Hide()
	return f
}

func (f *Feature) Overlay() fyne.CanvasObject { return f.overlay }
func (f *Feature) Visible() bool              { return f.overlay.Visible() }
func (f *Feature) Points() []Point            { return slices.Clone(f.points) }
func (f *Feature) Counts() Counts             { return f.counts }
func (f *Feature) Active() bool               { return f.active }

// Preparing exposes cancellation while root finishes the duplicate snapshot.
func (f *Feature) Preparing() {
	f.prepare(false)
}

// PreparingRebuild retires old work without changing the browsing parent/camera.
func (f *Feature) PreparingRebuild() { f.prepare(true) }

func (f *Feature) prepare(retain bool) {
	if f.stopped {
		return
	}
	visible := !retain || f.Visible()
	f.Close()
	f.active = true
	f.counts = Counts{}
	f.status.SetText(lang.L("Checking duplicate groups..."))
	if visible {
		f.overlay.Show()
	}
	f.host.LocationMapChanged()
}

func (f *Feature) Open(sources []Source) {
	f.open(sources, false)
}

// Rebuild replaces derived positions while retaining camera and hidden visits.
func (f *Feature) Rebuild(sources []Source) { f.open(sources, true) }

func (f *Feature) open(sources []Source, retain bool) {
	if f.stopped {
		return
	}
	visible := !retain || f.Visible()
	f.Close()
	f.active = true
	if !retain {
		f.surface.manual = false
	}
	ctx := context.Background()
	if f.options.Context != nil {
		ctx = f.options.Context(ctx)
	}
	f.ctx, f.cancel = context.WithCancel(ctx)
	if f.options.Thumbnails != nil {
		f.thumbs = f.options.Thumbnails()
	}
	f.counts = Counts{Total: len(sources)}
	var members []fyne.URI
	for _, source := range sources {
		members = append(members, source.URI)
		for _, donor := range source.Donors {
			members = append(members, donor.URI)
		}
	}
	f.SetSources(members)
	f.status.SetText(lang.L("Reading image locations..."))
	if visible {
		f.overlay.Show()
	}
	f.scan(slices.Clone(sources))
	f.host.LocationMapChanged()
}

func (f *Feature) Close() {
	if f.validationCancel != nil {
		f.validationCancel()
		f.validationCancel = nil
	}
	f.validationRevision++
	f.generation++
	f.suspendTiles()
	if f.cancel != nil {
		f.cancel()
		f.cancel = nil
	}
	f.active = false
	f.points = nil
	f.surface.clear()
	f.overlay.Hide()
	f.host.LocationMapChanged()
}
func (f *Feature) Stop() { f.stopped = true; f.stopLifetime(); f.Close() }
func (f *Feature) Wait() { f.workers.Wait(); f.retryWorkers.Wait() }

// SetSources bounds raw facts to the current loaded collection, even while hidden.
func (f *Feature) SetSources(sources []fyne.URI) {
	f.sources = slices.Clone(sources)
	keys := make([]string, 0, len(sources))
	allowed := make(map[string]bool, len(sources))
	for _, source := range sources {
		keys = append(keys, source.String())
		allowed[source.String()] = true
	}
	f.facts.Keep(keys)
	keep := make(map[string]string, len(keys))
	for _, key := range keys {
		if version, ok := f.versions[key]; ok {
			keep[key] = version
		}
	}
	f.versions = keep
	for key := range f.diskBypass {
		if !allowed[key] {
			delete(f.diskBypass, key)
		}
	}
}

// InvalidateSources observes a committed write, including resolved aliases.
// Version equality alone must not reuse a fact after an explicit mutation.
func (f *Feature) InvalidateSources(sources []fyne.URI) {
	keys := make([]string, 0, len(sources))
	for _, source := range sources {
		keys = append(keys, source.String())
		f.diskBypass[source.String()] = true
	}
	f.facts.Invalidate(keys)
	if !f.stopped && len(sources) > 0 {
		// A committed disk effect survives closing the map and application Stop.
		// The tracked finite cleanup is joined off UI; no new source read is needed.
		dir, queue := f.favoriteRoot.Load().(string), f.ui
		sources = slices.Clone(sources)
		f.workers.Go(func() { f.invalidatePersistentFacts(queue, dir, sources) })
	}
}

func (f *Feature) RawFacts() map[string]Fact { return f.facts.Snapshot() }

// HideForImage preserves the scan and camera while leaving the map surface.
func (f *Feature) HideForImage() {
	f.suspendTiles()
	f.overlay.Hide()
	f.surface.clear()
	f.host.LocationMapChanged()
}
func (f *Feature) Return() {
	if !f.active {
		return
	}
	f.overlay.Show()
	f.surface.arrange()
	f.host.Unfocus()
	f.host.LocationMapChanged()
}
