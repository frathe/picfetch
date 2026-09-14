package similarity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image/jpeg"
	"os"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
)

type searchPreparer struct {
	assets   string
	cache    *representationStore
	encoder  *Encoder
	versions map[string]os.FileInfo
	warning  string
	pressure uint64
}

func (p *searchPreparer) prepare(ctx context.Context, path string) (Item, bool, error) {
	item := Item{Path: path}
	before, err := os.Stat(path)
	if err != nil {
		return item, false, err
	}
	item.Size, item.ModifiedNS = before.Size(), before.ModTime().UnixNano()
	if item.Size > imaging.MaxEncodedBytes() {
		return item, false, fmt.Errorf("source exceeds encoded image limit")
	}
	reused := false
	cached, hit, cacheErr := p.cache.read(ctx, item)
	if cacheErr != nil && p.warning == "" {
		p.warning = cacheErr.Error()
	}
	if hit {
		item = cached
		reused = true
	}
	if !reused {
		data, bounds, err := imaging.ReadAndProbe(ctx, storage.NewFileURI(path))
		if err != nil {
			return item, false, err
		}
		loaded, err := imaging.DecodeLoaded(ctx, data, 1)
		if err != nil {
			return item, false, err
		}
		if p.encoder == nil {
			p.encoder, err = NewEncoder(p.assets, "cpu")
			if err != nil {
				return item, false, err
			}
		}
		item.Embedding, err = p.encoder.Encode(ctx, loaded.Frames[0])
		if err != nil {
			return item, false, err
		}
		item.Facts = imageFacts(path, data, bounds)
		item.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
		var preview bytes.Buffer
		if err := jpeg.Encode(&preview, imaging.ScaleForExport(loaded.Frames[0], 160), &jpeg.Options{Quality: 80}); err != nil {
			return item, false, err
		}
		item.Preview = preview.Bytes()
	}
	after, err := os.Stat(path)
	if err != nil {
		return item, false, err
	}
	if !sameVersion(before, after) {
		return item, false, fmt.Errorf("source changed during search preparation")
	}
	if err := ctx.Err(); err != nil {
		return item, false, err
	}
	p.versions[path] = after
	if !reused {
		if err := p.cache.write(ctx, item); err != nil {
			var pressure CachePressureError
			if errors.As(err, &pressure) {
				p.pressure = max(p.pressure, pressure.NeedBytes)
			} else if p.warning == "" {
				p.warning = err.Error()
			}
		}
	}
	return item, reused, nil
}

func searchLocal(ctx context.Context, req request, queries <-chan SearchQuery, emit func(SearchEvent) error) error {
	if EnforcesNetworkIsolation() {
		if err := VerifyOffline(ctx); err != nil {
			return err
		}
	}
	if err := VerifyAssets(ctx, req.Assets); err != nil {
		return err
	}
	_ = RegisterLocalFiles()
	cache, cacheErr := openRepresentationStore(ctx, req.Search.Cache, writeEnabledStores)
	if cache != nil {
		defer cache.close()
	}
	p := searchPreparer{assets: req.Assets, cache: cache, versions: map[string]os.FileInfo{}}
	if cacheErr != nil {
		p.warning = cacheErr.Error()
	}
	defer func() {
		if p.encoder != nil {
			p.encoder.Close()
		}
	}()
	return runSearchSession(ctx, *req.Search, queries, p.prepare, p.validate, p.refreshFavorites, func(event SearchEvent) error {
		event.CacheWarning = p.warning
		if event.Kind == SearchPartial || event.Kind == SearchFinal {
			event.CachePressureBytes = p.pressure
		}
		return emit(event)
	})
}

// Intermediate publications validate only the reference and at most 30 visible
// matches. A final publication validates the whole prepared scope once, avoiding
// a growing filesystem sweep at every 100-source boundary.
func (p *searchPreparer) validate(ctx context.Context, reference Item, matches []Match, final bool) error {
	check := func(path string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		before := p.versions[path]
		after, err := os.Stat(path)
		if err != nil || before == nil || !sameVersion(before, after) {
			return fmt.Errorf("search source changed: %s", path)
		}
		return nil
	}
	if final {
		for path := range p.versions {
			if err := check(path); err != nil {
				return err
			}
		}
		return nil
	}
	if err := check(reference.Path); err != nil {
		return err
	}
	for _, match := range matches {
		if err := check(match.Path); err != nil {
			return err
		}
	}
	return nil
}

// A committed Favorite save admits a new persistence pass in this retained
// worker. Vectors stay in memory; only the small disk preview is regenerated.
func (p *searchPreparer) refreshFavorites(ctx context.Context, items []Item) {
	if err := p.cache.refreshFavorites(ctx, items, p.completePreview); err != nil && p.warning == "" {
		p.warning = err.Error()
	}
}

func (p *searchPreparer) completePreview(ctx context.Context, item Item) (Item, error) {
	data, _, err := imaging.ReadAndProbe(ctx, storage.NewFileURI(item.Path))
	if err != nil {
		return item, err
	}
	if fmt.Sprintf("%x", sha256.Sum256(data)) != item.SHA256 {
		return item, fmt.Errorf("source changed before Favorite persistence")
	}
	loaded, err := imaging.DecodeLoaded(ctx, data, 1)
	if err != nil {
		return item, err
	}
	if err := p.validate(ctx, item, nil, false); err != nil {
		return item, err
	}
	var preview bytes.Buffer
	if err := jpeg.Encode(&preview, imaging.ScaleForExport(loaded.Frames[0], 160), &jpeg.Options{Quality: 80}); err != nil {
		return item, err
	}
	item.Preview = preview.Bytes()
	return item, nil
}
