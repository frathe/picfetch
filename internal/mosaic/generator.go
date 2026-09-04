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

	"fyne.io/fyne/v2"

	"github.com/frathe/picfetch/internal/imaging"
)

type loadedSource struct {
	pixels image.Image
	vector *imaging.Vector
	bounds image.Rectangle
}

type sourceLoader func(context.Context, fyne.URI) (*loadedSource, error)

// Generator owns the generation dependencies for one caller. New returns the
// production generator; the package-level Generate is the usual entry point.
type Generator struct {
	load sourceLoader
}

// New creates a mosaic generator backed by PicFetch's canonical image loader.
func New() *Generator {
	return &Generator{load: loadCanonicalSource}
}

// Generate renders one validated request with a fresh production generator.
func Generate(ctx context.Context, request Request) (Result, error) {
	return New().Generate(ctx, request)
}

// NoReadableSourcesError reports the sources that failed before any usable
// image could complete the mosaic.
type NoReadableSourcesError struct {
	Attempts []string
}

func (e *NoReadableSourcesError) Error() string {
	return "no readable mosaic source: " + strings.Join(e.Attempts, "; ")
}

// Generate lazily loads sources in deterministic shuffled order and renders
// placements as the covering layout requests them.
func (g *Generator) Generate(ctx context.Context, request Request) (Result, error) {
	if err := validateStoredRequest(request); err != nil {
		return Result{}, err
	}
	if g == nil || g.load == nil {
		return Result{}, fmt.Errorf("mosaic generator has no source loader")
	}

	pool := newSourcePool(request.sources, request.seed, g.load)
	active := make(map[int]*loadedSource)
	canvas := image.NewNRGBA(image.Rectangle{Max: request.target})
	fillNRGBA(canvas, color.NRGBA{R: 28, G: 30, B: 34, A: 255})
	primaryLayer := image.NewNRGBA(canvas.Bounds())

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

		return renderPlacement(ctx, destination, source, placement)
	}

	_, err := walkLayout(ctx, request.target, request.settings, request.seed, next, onPlacement)
	if err != nil {
		return Result{}, err
	}
	draw.Draw(canvas, canvas.Bounds(), primaryLayer, primaryLayer.Bounds().Min, draw.Over)

	// canvas was allocated for this result and was never shared with a caller,
	// so ownership can transfer without a second target-sized copy.
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
	entries  []sourceEntry
	cursor   int
	cycle    int
	readable []sourceEntry
	cache    map[int]*loadedSource
	attempts []string
	load     sourceLoader
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

	return &sourcePool{entries: entries, cache: make(map[int]*loadedSource), load: load}
}

func (p *sourcePool) next(ctx context.Context) (sourceEntry, *loadedSource, error) {
	for p.cursor < len(p.entries) {
		entry := p.entries[p.cursor]
		p.cursor++
		source, err := p.load(ctx, entry.uri)
		if err == nil {
			p.readable = append(p.readable, entry)
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
	// entries already proven readable. Cache begins at the first repeat; unique
	// one-use sources were released immediately after their placement.
	for tried := 0; tried < len(p.readable); tried++ {
		entry := p.readable[p.cycle%len(p.readable)]
		p.cycle++
		if source := p.cache[entry.id]; source != nil {
			return entry, source, nil
		}
		source, err := p.load(ctx, entry.uri)
		if err == nil {
			p.cache[entry.id] = source
			return entry, source, nil
		}
		if contextError(ctx, err) != nil {
			return sourceEntry{}, nil, contextError(ctx, err)
		}
		p.record(entry, err)
	}

	return sourceEntry{}, nil, &NoReadableSourcesError{Attempts: append([]string(nil), p.attempts...)}
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
