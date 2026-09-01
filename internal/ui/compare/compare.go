// Package compare owns the transient main-window surface for comparing two
// files. It deliberately knows nothing about the grid or viewer state: its
// caller resolves the selected URIs, while this package owns presentation,
// concurrent loading, cancellation, and stale-result rejection.
package compare

import (
	"context"
	"errors"
	"image"
	"image/color"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"

	"github.com/frathe/picfetch/internal/completion"
	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/ui/widgets"
)

const comparisonChromeAlpha = 220

// Loader reads one comparison source. Implementations must observe ctx before
// expensive work and while doing cancellable I/O.
type Loader func(ctx context.Context, uri fyne.URI) (*imaging.LoadedImage, error)

// Callbacks are the only interactions a comparison has with its owner.
type Callbacks struct {
	Repaint      func()
	Opened       func()
	Closed       func()
	Failed       func(uri fyne.URI, err error)
	OrderChanged func(left, right string)
	Modifiers    func() fyne.KeyModifier
}

type pane struct {
	root    *fyne.Container
	image   *canvas.Image
	input   *paneInput
	spinner *widget.ProgressBarInfinite
}

func newPaneImage() *canvas.Image {
	img := canvas.NewImageFromImage(nil)
	img.FillMode = canvas.ImageFillContain
	img.ScaleMode = canvas.ImageScaleSmooth
	img.Hide()
	return img
}

func newPaneSpinner() *widget.ProgressBarInfinite {
	spinner := widget.NewProgressBarInfinite()
	spinner.Hide()
	return spinner
}

// Feature is one comparison session surface. Its mutable widget state is
// applied on Fyne's UI path; lifecycle and completion state are safe for the
// worker goroutines and test waiters that read them concurrently.
type Feature struct {
	loader    Loader
	callbacks Callbacks

	overlay      *fyne.Container
	content      *fyne.Container
	panes        [2]pane
	reveals      [2]paneReveal
	divider      *swipeDivider
	layoutToggle *widget.Button
	swap         *widget.Button
	badges       [2]*widget.Label

	sources    [2]fyne.URI
	identities [2]string
	loaded     [2]*imaging.LoadedImage
	rendered   [2]image.Image
	vectors    [2]vectorRasterState

	active bool
	ready  bool

	transform  linkedTransform
	viewports  [2]fyne.Size
	layoutMode comparisonLayout
	dividerAt  float32

	lifecycle requestLifecycle
	done      completion.Signal
	workers   sync.WaitGroup
	uiPending sync.WaitGroup
	uiCount   atomic.Int64
	ui        UIQueue

	vectorDebounce  time.Duration
	vectorAfter     func(time.Duration) <-chan time.Time
	vectorRasterize func(vector *imaging.Vector, width, height int) (image.Image, error)
	vectorPixels    func(fyne.CanvasObject, fyne.Size) (int, int)
}

// New constructs one initially hidden comparison surface.
func New(loader Loader, callbacks Callbacks) *Feature {
	f := &Feature{
		loader:         loader,
		callbacks:      callbacks,
		transform:      defaultLinkedTransform(),
		dividerAt:      defaultDivider,
		layoutMode:     sideBySide,
		vectorDebounce: defaultCompareVectorDebounce,
		vectorAfter:    time.After,
		vectorRasterize: func(vector *imaging.Vector, width, height int) (image.Image, error) {
			return vector.RasterAt(width, height)
		},
		vectorPixels: displayPixelSize,
		ui:           fyneQueue{},
	}
	f.panes = [2]pane{newPane(f, 0), newPane(f, 1)}
	f.reveals = [2]paneReveal{newPaneReveal(f.panes[0].root), newPaneReveal(f.panes[1].root)}
	f.divider = newSwipeDivider(f)
	f.divider.Hide()
	f.content = container.New(
		comparisonContentLayout{feature: f},
		f.reveals[0].clip,
		f.reveals[1].clip,
		f.divider,
	)

	f.layoutToggle = widget.NewButton(lang.L("Swipe"), f.toggleLayout)
	f.layoutToggle.Disable()
	f.swap = widget.NewButton(lang.L("Swap"), f.swapSides)
	f.swap.Disable()
	back := widget.NewButton(lang.L("Back to Grid"), f.Close)
	toolbarCard := newChromeCard(container.NewHBox(f.layoutToggle, f.swap, back))
	toolbar := container.NewHBox(layout.NewSpacer(), toolbarCard)
	var badgeCards [2]*fyne.Container
	for i := range f.badges {
		f.badges[i] = widget.NewLabel("")
		badgeCards[i] = newChromeCard(f.badges[i])
	}
	badges := container.NewHBox(badgeCards[0], layout.NewSpacer(), badgeCards[1])
	chrome := container.NewBorder(toolbar, badges, nil, nil)
	backdrop := canvas.NewRectangle(theme.Color(theme.ColorNameBackground))
	// The comparison surface sits in the main window's root stack even while
	// hidden. Keep its minimum size at zero so merely registering the feature
	// cannot resize or relayout the ordinary image/grid surfaces underneath.
	f.overlay = container.New(overlayLayout{}, backdrop, newInputShield(), f.content, chrome)
	f.overlay.Hide()
	return f
}

func newChromeCard(content fyne.CanvasObject) *fyne.Container {
	backgroundColor := color.NRGBAModel.Convert(theme.Color(theme.ColorNameOverlayBackground)).(color.NRGBA)
	backgroundColor.A = comparisonChromeAlpha
	background := canvas.NewRectangle(backgroundColor)
	background.CornerRadius = widgets.CardRadius
	return container.NewStack(background, container.NewPadded(content))
}

func sourceIdentities(sources [2]fyne.URI) [2]string {
	identities := [2]string{sources[0].Name(), sources[1].Name()}
	if identities[0] != identities[1] {
		return identities
	}

	parts := [2][]string{uriPathParts(sources[0]), uriPathParts(sources[1])}
	for depth := 2; depth <= max(len(parts[0]), len(parts[1])); depth++ {
		candidates := [2]string{pathSuffix(parts[0], depth), pathSuffix(parts[1], depth)}
		identities = candidates
		if candidates[0] != candidates[1] {
			return candidates
		}
	}
	return identities
}

func uriPathParts(uri fyne.URI) []string {
	normalized := strings.ReplaceAll(uri.Path(), "\\", "/")
	raw := strings.Split(strings.Trim(normalized, "/"), "/")
	parts := make([]string, 0, len(raw)+1)
	if authority := uri.Authority(); authority != "" {
		parts = append(parts, authority)
	} else if strings.HasPrefix(normalized, "/") {
		// Preserve the filesystem root as the otherwise unnamed directory
		// component. It matters only when a root-level filename collides with
		// the same basename below a named folder: "/same.jpg" is then the
		// shortest directory/file identity available for that side.
		parts = append(parts, "")
	}
	for _, part := range raw {
		if part != "" && part != "." {
			parts = append(parts, part)
		}
	}
	if len(parts) == 0 {
		return []string{uri.Name()}
	}
	parts[len(parts)-1] = uri.Name()
	return parts
}

func pathSuffix(parts []string, depth int) string {
	if depth > len(parts) {
		depth = len(parts)
	}
	return strings.Join(parts[len(parts)-depth:], "/")
}

// Overlay is the opaque full-window canvas object internal/ui composes above
// the still-open grid.
func (f *Feature) Overlay() fyne.CanvasObject { return f.overlay }

// Visible reports whether a comparison session owns the main-window surface.
func (f *Feature) Visible() bool { return f.active }

// Ready reports whether both panes hold a decoded first frame.
func (f *Feature) Ready() bool { return f.ready }

// Done returns the replaceable completion signal for the latest Open call.
func (f *Feature) Done() *completion.Signal { return &f.done }

// Open resets and reveals the overlay synchronously, then starts both source
// loads concurrently as one cancellable generation.
func (f *Feature) Open(sources [2]fyne.URI) {
	token := f.lifecycle.begin()
	done := f.done.Begin()
	f.clearVectorRasters()

	f.active = true
	f.ready = false
	f.transform = defaultLinkedTransform()
	f.resetLayout()
	f.layoutToggle.Disable()
	f.swap.Disable()
	f.sources = sources
	f.identities = sourceIdentities(sources)
	for i, identity := range f.identities {
		f.badges[i].SetText(identity)
	}
	f.notifyOrderChanged()
	for i := range f.panes {
		f.loaded[i] = nil
		f.panes[i].image.Image = nil
		f.panes[i].image.Hide()
		f.panes[i].spinner.Show()
	}
	f.overlay.Show()
	f.repaint()
	if f.callbacks.Opened != nil {
		f.callbacks.Opened()
	}

	f.workers.Go(func() {
		f.load(token, sources, done)
	})
}

// Close cancels pending work and removes only the comparison overlay.
func (f *Feature) Close() {
	if !f.active {
		return
	}
	f.lifecycle.invalidate()
	f.hide()
	if f.callbacks.Closed != nil {
		f.callbacks.Closed()
	}
}

// Settle waits until every comparison generation, including a superseded
// one, has stopped. Production does not block on it; deterministic test
// cleanup uses it after Close has cancelled the current generation.
func (f *Feature) Settle(ctx context.Context) error {
	for {
		// Every worker queues its UI completion before marking itself done.
		// A drained load completion can start vector work, so repeat until
		// both worker sets and the queue are empty at the same boundary.
		if err := waitWithContext(ctx, f.workers.Wait); err != nil {
			return err
		}
		for i := range f.vectors {
			if err := waitWithContext(ctx, f.vectors[i].pending.Wait); err != nil {
				return err
			}
		}
		if f.ui.Drain() {
			continue
		}
		if f.uiCount.Load() == 0 {
			return nil
		}
		if err := waitWithContext(ctx, f.uiPending.Wait); err != nil {
			return err
		}
	}
}

func waitWithContext(ctx context.Context, wait func()) error {
	done := make(chan struct{})
	go func() {
		wait()
		close(done)
	}()
	select {
	case <-done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (f *Feature) hide() {
	f.clearVectorRasters()
	f.active = false
	f.ready = false
	f.layoutToggle.Disable()
	f.swap.Disable()
	f.sources = [2]fyne.URI{}
	f.identities = [2]string{}
	f.loaded = [2]*imaging.LoadedImage{}
	f.rendered = [2]image.Image{}
	for i := range f.panes {
		f.badges[i].SetText("")
		f.panes[i].image.Image = nil
		f.panes[i].image.Hide()
		f.panes[i].spinner.Hide()
	}
	f.overlay.Hide()
	f.repaint()
}

type loadResult struct {
	index  int
	uri    fyne.URI
	loaded *imaging.LoadedImage
	err    error
}

func (f *Feature) load(token requestToken, sources [2]fyne.URI, done func()) {
	results := make(chan loadResult, len(sources))
	for i, uri := range sources {
		go func() {
			loaded, err := f.loadOne(token.context(), uri)
			results <- loadResult{index: i, uri: uri, loaded: loaded, err: err}
		}()
	}

	var loaded [2]loadResult
	var failure *loadResult
	for range sources {
		result := <-results
		loaded[result.index] = result
		if result.err != nil && failure == nil {
			failed := result
			failure = &failed
			// Stop cancellable work on the other side. latest below deliberately
			// ignores this context cancellation so this generation can still
			// publish its own failure; Close advances the revision as well.
			token.cancelContext()
		}
	}

	f.queueUI(func() {
		defer done()
		if !token.latest() {
			return
		}
		if failure != nil {
			f.fail(failure.uri, failure.err)
			return
		}

		for _, result := range loaded {
			if err := validateLoaded(result.loaded); err != nil {
				f.fail(result.uri, err)
				return
			}
		}

		for i, result := range loaded {
			f.loaded[i] = result.loaded
			f.rendered[i] = result.loaded.Frames[0]
			f.vectors[i].setRaster(result.loaded.Frames[0])
			f.panes[i].image.Image = f.rendered[i]
			f.panes[i].spinner.Hide()
			f.panes[i].image.Show()
			f.panes[i].image.Refresh()
		}
		f.ready = true
		f.applyTransform()
		f.layoutToggle.Enable()
		f.swap.Enable()
		f.repaint()
	})
}

func (f *Feature) swapSides() {
	if !f.active || !f.ready {
		return
	}
	f.sources[0], f.sources[1] = f.sources[1], f.sources[0]
	f.identities[0], f.identities[1] = f.identities[1], f.identities[0]
	f.clearVectorRequests()
	f.loaded[0], f.loaded[1] = f.loaded[1], f.loaded[0]
	f.rendered[0], f.rendered[1] = f.rendered[1], f.rendered[0]
	for i := range f.panes {
		f.badges[i].SetText(f.identities[i])
		f.vectors[i].setRaster(f.rendered[i])
		f.panes[i].image.Image = f.rendered[i]
		f.panes[i].image.Refresh()
	}
	f.applyTransform()
	f.notifyOrderChanged()
	f.repaint()
}

func (f *Feature) notifyOrderChanged() {
	if f.callbacks.OrderChanged != nil {
		f.callbacks.OrderChanged(f.identities[0], f.identities[1])
	}
}

func (f *Feature) loadOne(ctx context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
	if f.loader == nil {
		return nil, errors.New("comparison loader is not configured")
	}
	return f.loader(ctx, uri)
}

func validateLoaded(loaded *imaging.LoadedImage) error {
	if loaded == nil || len(loaded.Frames) == 0 || loaded.Frames[0] == nil {
		return errors.New("decoded image has no frame")
	}
	bounds := loaded.Frames[0].Bounds()
	if bounds.Dx() <= 0 || bounds.Dy() <= 0 {
		return errors.New("decoded image has invalid dimensions")
	}
	return nil
}

func (f *Feature) fail(uri fyne.URI, err error) {
	f.lifecycle.invalidate()
	f.hide()
	if f.callbacks.Closed != nil {
		f.callbacks.Closed()
	}
	if f.callbacks.Failed != nil {
		f.callbacks.Failed(uri, err)
	}
}

func (f *Feature) repaint() {
	if f.callbacks.Repaint != nil {
		f.callbacks.Repaint()
	}
}

// overlayLayout fills every layer without contributing a minimum size to the
// root stack that owns this transient surface.
type overlayLayout struct{}

func (overlayLayout) Layout(objects []fyne.CanvasObject, size fyne.Size) {
	for _, object := range objects {
		object.Move(fyne.NewPos(0, 0))
		object.Resize(size)
	}
}

func (overlayLayout) MinSize([]fyne.CanvasObject) fyne.Size { return fyne.Size{} }

type requestLifecycle struct {
	revision atomic.Uint64
	mu       sync.Mutex
	cancel   context.CancelFunc
}

func (l *requestLifecycle) begin() requestToken {
	ctx, cancel := context.WithCancel(context.Background())
	l.mu.Lock()
	if l.cancel != nil {
		l.cancel()
	}
	revision := l.revision.Add(1)
	l.cancel = cancel
	l.mu.Unlock()
	return requestToken{ctx: ctx, cancel: cancel, lifecycle: l, revision: revision}
}

func (l *requestLifecycle) invalidate() {
	l.mu.Lock()
	l.revision.Add(1)
	if l.cancel != nil {
		l.cancel()
		l.cancel = nil
	}
	l.mu.Unlock()
}

type requestToken struct {
	ctx       context.Context
	cancel    context.CancelFunc
	lifecycle *requestLifecycle
	revision  uint64
}

func (t requestToken) context() context.Context { return t.ctx }

func (t requestToken) latest() bool {
	return t.lifecycle != nil && t.lifecycle.revision.Load() == t.revision
}

func (t requestToken) cancelContext() {
	if t.cancel != nil {
		t.cancel()
	}
}
