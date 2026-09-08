package mosaic

import (
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math/rand/v2"
	"strings"
	"time"

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/imaging"
)

type loadedSource struct {
	pixels image.Image
	vector *imaging.Vector
	bounds image.Rectangle
}

type sourceLoader func(context.Context, fyne.URI) (*loadedSource, error)

const defaultRepeatCacheBytes = 64 << 20

// Generator owns the generation dependencies for one caller. New returns the
// production generator; the package-level Generate is the usual entry point.
type Generator struct {
	load          sourceLoader
	cacheBytes    int64
	beforePrepare func(preparationPlan) error
	previewClock  func() time.Time
}

// New creates a mosaic generator backed by PicFetch's canonical image loader.
func New() *Generator {
	return &Generator{load: loadCanonicalSource, cacheBytes: defaultRepeatCacheBytes, previewClock: time.Now}
}

// Generate renders one validated request with a fresh production generator.
func Generate(ctx context.Context, request Request) (Result, error) {
	return New().Generate(ctx, request)
}

// Progress measures canvas pixels covered by completed placements. It measures
// image coverage, not elapsed time; completion is reported after composition.
type Progress struct {
	CoveredPixels int
	TotalPixels   int
	// Preview is an optional independent canvas snapshot, at most 960 pixels
	// on its longest edge. The generator never mutates a published preview.
	// Nil means keep the preceding preview. Final pixels belong to Result.
	Preview image.Image
}

// GenerateWithProgress reports progress synchronously during generation.
// The callback must return promptly and may cancel ctx. Nil disables reporting.
func GenerateWithProgress(ctx context.Context, request Request, report func(Progress)) (Result, error) {
	return New().GenerateWithProgress(ctx, request, report)
}

// NoReadableSourcesError reports the sources that failed before any usable
// image could complete the mosaic.
type NoReadableSourcesError struct {
	Attempts []string
}

func (e *NoReadableSourcesError) Error() string {
	return "no readable mosaic source: " + strings.Join(e.Attempts, "; ")
}

// Generate lazily loads sources in deterministic shuffled order. Random mode
// renders while it plans; Shelf mode settles its layout before rendering.
func (g *Generator) Generate(ctx context.Context, request Request) (Result, error) {
	return g.GenerateWithProgress(ctx, request, nil)
}

// GenerateWithProgress is Generate with a synchronous coverage callback.
func (g *Generator) GenerateWithProgress(ctx context.Context, request Request, report func(Progress)) (Result, error) {
	if err := validateStoredRequest(request); err != nil {
		return Result{}, err
	}
	if g == nil || g.load == nil {
		return Result{}, fmt.Errorf("mosaic generator has no source loader")
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	total := request.target.X * request.target.Y
	if report != nil {
		report(Progress{TotalPixels: total})
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if request.settings.Layout == LayoutShelf {
		return g.generateShelfWithProgress(ctx, request, report)
	}

	pool := newSourcePool(request.sources, request.seed, g.load)
	if g.cacheBytes > 0 {
		pool.cache.SetBudget(g.cacheBytes)
	}
	active := make(map[int]*loadedSource)
	canvas := image.NewNRGBA(image.Rectangle{Max: request.target})
	fillNRGBA(canvas, color.NRGBA{R: 28, G: 30, B: 34, A: 255})
	primaryLayer := image.NewNRGBA(canvas.Bounds())
	previews := previewSnapshots{clock: g.previewClock}
	reportCoverage := func(covered int) {
		if report == nil || covered >= total || ctx.Err() != nil {
			return
		}
		preview := previews.next(canvas, primaryLayer)
		if ctx.Err() == nil {
			report(Progress{CoveredPixels: covered, TotalPixels: total, Preview: preview})
		}
	}

	next := func() (candidate, error) {
		entry, source, err := pool.next(ctx)
		if err != nil {
			return candidate{}, err
		}
		active[entry.id] = source

		return candidate{
			id:     entry.id,
			aspect: float64(source.bounds.Dx()) / float64(source.bounds.Dy()),
		}, nil
	}
	onPlacement := func(placement placement) error {
		source := active[placement.candidateID]
		delete(active, placement.candidateID)
		if source == nil {
			return fmt.Errorf("mosaic source %d was not loaded", placement.candidateID)
		}

		destination := primaryLayer
		if placement.repair {
			destination = canvas
		}

		return renderPlacementWithBudget(ctx, destination, source, placement, maxPreparationBytes, g.beforePrepare)
	}

	_, err := walkLayout(ctx, request.target, request.settings, request.seed, next, onPlacement, reportCoverage)
	if err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	draw.Draw(canvas, canvas.Bounds(), primaryLayer, primaryLayer.Bounds().Min, draw.Over)
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if report != nil {
		report(Progress{CoveredPixels: total, TotalPixels: total})
	}

	// canvas was allocated for this result and was never shared with a caller,
	// so ownership can transfer without a second target-sized copy.
	return Result{pixels: canvas}, nil
}

// generateShelfWithProgress renders a settled Shelf-mode plan. Its repair
// choices depend on the completed layout, so it must not stream placements.
func (g *Generator) generateShelfWithProgress(ctx context.Context, request Request, report func(Progress)) (Result, error) {
	total := request.target.X * request.target.Y
	pool := newSourcePool(request.sources, request.seed, g.load)
	pool.cacheInitial = true
	if g.cacheBytes > 0 {
		pool.cache.SetBudget(g.cacheBytes)
	}
	next := func() (candidate, error) {
		return pool.nextCandidate(ctx)
	}
	plan, err := planShelfLayout(ctx, request.target, request.settings, request.seed, next)
	if err != nil {
		return Result{}, err
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}

	canvas := image.NewNRGBA(image.Rectangle{Max: request.target})
	fillNRGBA(canvas, color.NRGBA{R: 28, G: 30, B: 34, A: 255})
	primaryLayer := image.NewNRGBA(canvas.Bounds())
	previews := previewSnapshots{clock: g.previewClock}
	reportCoverage := func(covered int) {
		if report == nil || covered >= total || ctx.Err() != nil {
			return
		}
		preview := previews.next(canvas, primaryLayer)
		if ctx.Err() == nil {
			report(Progress{CoveredPixels: covered, TotalPixels: total, Preview: preview})
		}
	}

	renderedCovered := make([]bool, total)
	covered := 0
	for _, placement := range plan.placements {
		source, err := pool.source(ctx, placement.candidateID)
		if err != nil {
			return Result{}, err
		}
		destination := primaryLayer
		if placement.repair {
			destination = canvas
		}
		if err := renderPlacementWithBudget(ctx, destination, source, placement, maxPreparationBytes, g.beforePrepare); err != nil {
			return Result{}, err
		}
		if added := markCovered(renderedCovered, request.target, placement); added > 0 {
			covered += added
			reportCoverage(covered)
		}
	}
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	draw.Draw(canvas, canvas.Bounds(), primaryLayer, primaryLayer.Bounds().Min, draw.Over)
	if err := ctx.Err(); err != nil {
		return Result{}, err
	}
	if report != nil {
		report(Progress{CoveredPixels: total, TotalPixels: total})
	}

	return Result{pixels: canvas}, nil
}

func validateStoredRequest(request Request) error {
	if len(request.sources) == 0 {
		return &ValidationError{Field: "sources", Err: fmt.Errorf("must not be empty")}
	}
	if request.target.X <= 0 || request.target.Y <= 0 {
		return &ValidationError{Field: "target", Err: fmt.Errorf("must be positive")}
	}

	return request.settings.Validate()
}

func loadCanonicalSource(ctx context.Context, uri fyne.URI) (*loadedSource, error) {
	data, bounds, err := imaging.ReadAndProbe(ctx, uri)
	if err != nil {
		return nil, err
	}
	decoded, err := imaging.DecodeLoaded(ctx, data, 0)
	if err != nil {
		return nil, err
	}
	if len(decoded.Frames) == 0 || decoded.Frames[0] == nil {
		return nil, fmt.Errorf("decoded image has no frame")
	}
	sourceBounds := decoded.Frames[0].Bounds()
	if decoded.Vector != nil {
		sourceBounds = bounds
	}

	return &loadedSource{
		pixels: decoded.Frames[0],
		vector: decoded.Vector,
		bounds: sourceBounds,
	}, nil
}

type sourceEntry struct {
	id  int
	uri fyne.URI
}

type sourcePool struct {
	entries      []sourceEntry
	cursor       int
	cycle        int
	readable     []sourceEntry
	cache        *imaging.ByteCache[*loadedSource]
	cacheInitial bool
	// layoutBounds holds only source geometry learned while Shelf plans. It
	// avoids repeatedly decoding an oversized source merely to reuse its aspect.
	layoutBounds map[int]image.Rectangle
	attempts     []string
	load         sourceLoader
}

func newSourcePool(sources []fyne.URI, seed int64, load sourceLoader) *sourcePool {
	entries := make([]sourceEntry, 0, len(sources))
	seen := make(map[string]struct{}, len(sources))
	for _, uri := range sources {
		key := uri.String()
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		entries = append(entries, sourceEntry{id: len(entries), uri: uri})
	}
	random := rand.New(rand.NewPCG(uint64(seed)^0x243f6a8885a308d3, uint64(seed)^0x13198a2e03707344))
	random.Shuffle(len(entries), func(i, j int) {
		entries[i], entries[j] = entries[j], entries[i]
	})

	cache := imaging.NewByteCache(defaultRepeatCacheBytes, func(source *loadedSource) int64 {
		loaded := imaging.LoadedImage{Frames: []image.Image{source.pixels}, Vector: source.vector}
		return loaded.DecodedBytes()
	})
	return &sourcePool{entries: entries, cache: cache, layoutBounds: make(map[int]image.Rectangle), load: load}
}

func (p *sourcePool) next(ctx context.Context) (sourceEntry, *loadedSource, error) {
	for p.cursor < len(p.entries) {
		entry := p.entries[p.cursor]
		p.cursor++
		source, err := p.load(ctx, entry.uri)
		if err == nil {
			p.remember(entry, source)
			return entry, source, nil
		}
		if contextError(ctx, err) != nil {
			return sourceEntry{}, nil, contextError(ctx, err)
		}
		p.record(entry, err)
	}

	if len(p.readable) == 0 {
		return sourceEntry{}, nil, &NoReadableSourcesError{Attempts: append([]string(nil), p.attempts...)}
	}
	// Once the original shuffled pool is exhausted, cycle only through the
	// entries already proven readable. Repeats share a byte-budgeted cache;
	// oversized sources are released after each placement instead of retained.
	for tried := 0; tried < len(p.readable); tried++ {
		entry := p.readable[p.cycle%len(p.readable)]
		p.cycle++
		key := entry.uri.String()
		if source, ok := p.cache.Get(key); ok {
			return entry, source, nil
		}
		source, err := p.load(ctx, entry.uri)
		if err == nil {
			_ = p.cache.AddIfFits(key, source)
			return entry, source, nil
		}
		if contextError(ctx, err) != nil {
			return sourceEntry{}, nil, contextError(ctx, err)
		}
		p.record(entry, err)
	}

	return sourceEntry{}, nil, &NoReadableSourcesError{Attempts: append([]string(nil), p.attempts...)}
}

// nextCandidate returns source geometry for the settled Shelf plan. Repeated
// candidates reuse their measured bounds, while rendering still observes the
// normal byte-bounded source cache.
func (p *sourcePool) nextCandidate(ctx context.Context) (candidate, error) {
	for p.cursor < len(p.entries) {
		entry := p.entries[p.cursor]
		p.cursor++
		source, err := p.load(ctx, entry.uri)
		if err == nil {
			p.remember(entry, source)
			return candidateForSource(entry, source), nil
		}
		if cause := contextError(ctx, err); cause != nil {
			return candidate{}, cause
		}
		p.record(entry, err)
	}

	if len(p.readable) == 0 {
		return candidate{}, &NoReadableSourcesError{Attempts: append([]string(nil), p.attempts...)}
	}
	entry := p.readable[p.cycle%len(p.readable)]
	p.cycle++
	if bounds, ok := p.layoutBounds[entry.id]; ok {
		return candidateForBounds(entry, bounds), nil
	}

	// All readable entries record bounds before joining this cycle. Keep this
	// fallback defensive in case a future caller builds a pool differently.
	source, err := p.load(ctx, entry.uri)
	if err == nil {
		p.layoutBounds[entry.id] = source.bounds
		return candidateForSource(entry, source), nil
	}
	if cause := contextError(ctx, err); cause != nil {
		return candidate{}, cause
	}
	p.record(entry, err)

	return candidate{}, &NoReadableSourcesError{Attempts: append([]string(nil), p.attempts...)}
}

func (p *sourcePool) remember(entry sourceEntry, source *loadedSource) {
	p.readable = append(p.readable, entry)
	p.layoutBounds[entry.id] = source.bounds
	if p.cacheInitial {
		_ = p.cache.AddIfFits(entry.uri.String(), source)
	}
}

func candidateForSource(entry sourceEntry, source *loadedSource) candidate {
	return candidateForBounds(entry, source.bounds)
}

func candidateForBounds(entry sourceEntry, bounds image.Rectangle) candidate {
	return candidate{
		id:     entry.id,
		aspect: float64(bounds.Dx()) / float64(bounds.Dy()),
	}
}

func (p *sourcePool) source(ctx context.Context, id int) (*loadedSource, error) {
	for _, entry := range p.entries {
		if entry.id != id {
			continue
		}
		key := entry.uri.String()
		if source, ok := p.cache.Get(key); ok {
			return source, nil
		}
		source, err := p.load(ctx, entry.uri)
		if err == nil {
			_ = p.cache.AddIfFits(key, source)
			return source, nil
		}
		if cause := contextError(ctx, err); cause != nil {
			return nil, cause
		}
		return nil, fmt.Errorf("mosaic source %q could not be reloaded: %w", entry.uri.Name(), err)
	}

	return nil, fmt.Errorf("mosaic source %d was not part of the layout", id)
}

func (p *sourcePool) record(entry sourceEntry, err error) {
	name := fmt.Sprintf("source %d", entry.id)
	if entry.uri != nil {
		name = entry.uri.Name()
	}
	p.attempts = append(p.attempts, fmt.Sprintf("%s: %v", name, err))
}

func contextError(ctx context.Context, err error) error {
	if cause := ctx.Err(); cause != nil {
		return cause
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return err
	}

	return nil
}

func fillNRGBA(destination *image.NRGBA, value color.NRGBA) {
	for y := destination.Bounds().Min.Y; y < destination.Bounds().Max.Y; y++ {
		for x := destination.Bounds().Min.X; x < destination.Bounds().Max.X; x++ {
			destination.SetNRGBA(x, y, value)
		}
	}
}
