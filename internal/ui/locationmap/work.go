package locationmap

import (
	"context"
	"errors"
	"fmt"
	"image"
	"maps"
	"slices"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/favthumbs"
	"github.com/frathe/picfetch/internal/imaging"
)

// UIQueue marshals worker delivery; the harness drains it off the workers.
type UIQueue interface {
	Do(func())
	Drain() bool
}
type fyneQueue struct{}

func (fyneQueue) Do(fn func()) { fyne.Do(fn) }
func (fyneQueue) Drain() bool  { return false }

// SetUIQueue configures delivery before starting work.
func (f *Feature) SetUIQueue(queue UIQueue) {
	if queue == nil {
		queue = fyneQueue{}
	}
	f.ui = queue
}

// Settle joins finite work and drains its delivery until no more work starts.
func (f *Feature) Settle() {
	for {
		f.workers.Wait()
		if !f.Visible() {
			f.retryWorkers.Wait()
		}
		if !f.ui.Drain() {
			return
		}
	}
}

func (f *Feature) run(ctx context.Context, work func()) {
	f.workers.Go(func() {
		select {
		case f.slots <- struct{}{}:
		case <-ctx.Done():
			return
		}
		defer func() { <-f.slots }()
		if ctx.Err() == nil {
			work()
		}
	})
}

func (f *Feature) scan(sources []Source) {
	ctx, generation, queue := f.ctx, f.generation, f.ui
	favoriteRoot := f.favoriteRoot.Load().(string)
	allSources := append([]fyne.URI(nil), f.sources...)
	bypass := maps.Clone(f.diskBypass)
	var pending struct {
		sync.Mutex
		points []Point
		counts Counts
		queued bool
	}
	publish := func(point *Point, counts Counts) {
		pending.Lock()
		if point != nil {
			pending.points = append(pending.points, *point)
		}
		pending.counts = counts
		if pending.queued {
			pending.Unlock()
			return
		}
		pending.queued = true
		pending.Unlock()
		queue.Do(func() {
			pending.Lock()
			points, counts := pending.points, pending.counts
			pending.points = nil
			pending.queued = false
			pending.Unlock()
			if ctx.Err() != nil || generation != f.generation {
				return
			}
			f.points = append(f.points, points...)
			f.counts = counts
			f.status.SetText(fmt.Sprintf(lang.L("%d of %d processed, %d located, %d without GPS, %d unreadable, %d conflicts"), counts.Completed, counts.Total, counts.Located, counts.Unlocated, counts.Failed, counts.Conflicts))
			if counts.Complete {
				if counts.Total == 0 {
					f.status.SetText(lang.L("Open images to browse their locations."))
				} else if counts.Located == 0 {
					f.status.SetText(fmt.Sprintf(lang.L("No usable image locations. %d without GPS, %d unreadable, %d conflicts"), counts.Unlocated, counts.Failed, counts.Conflicts))
				}
			}
			if !f.surface.manual {
				f.surface.fit()
			} else {
				f.surface.arrange()
			}
			f.host.LocationMapChanged()
		})
	}
	f.run(ctx, func() {
		versions := sourceVersions(ctx, allSources)
		queue.Do(func() {
			if ctx.Err() == nil && generation == f.generation {
				f.versions = versions
			}
		})
		owners, cacheErr := openFavoriteFacts(ctx, favoriteRoot)
		defer owners.close()
		f.cacheFailure(queue, ctx, generation, cacheErr)
		counts := Counts{Total: len(sources)}
		for _, source := range sources {
			if ctx.Err() != nil {
				return
			}
			metadata, version, err := f.readSource(ctx, source.URI, owners, queue, generation, bypass[source.URI.String()])
			representative := LocationCandidate{Metadata: metadata, ReadError: err != nil}
			var donors []LocationCandidate
			if err == nil && !validLocation(metadata) {
				for i, donor := range source.Donors {
					if ctx.Err() != nil {
						return
					}
					fact, _, readErr := f.readSource(ctx, donor.URI, owners, queue, generation, bypass[donor.URI.String()])
					donors = append(donors, LocationCandidate{Index: i, PixelCount: donor.Pixels, Metadata: fact, ReadError: readErr != nil})
				}
			}
			resolved := ResolveLocation(ctx, representative, donors)
			if ctx.Err() != nil {
				return
			}
			counts.Completed++
			var point *Point
			switch resolved.Outcome {
			case LocationUnreadable:
				counts.Failed++
			case LocationUnlocated:
				counts.Unlocated++
			case LocationConflict:
				counts.Conflicts++
			default:
				counts.Located++
				point = &Point{Source: source, Metadata: resolved.Metadata, Version: version}
				if resolved.DonorIndex >= 0 {
					point.Donor = source.Donors[resolved.DonorIndex]
				}
			}
			publish(point, counts)
		}
		counts.Complete = true
		publish(nil, counts)
		if latest := f.favoriteRoot.Load().(string); latest != "" {
			f.persistKnown(queue, ctx, latest, allSources, generation)
		}
	})
}

func sourceVersions(ctx context.Context, sources []fyne.URI) map[string]string {
	versions := make(map[string]string, len(sources))
	for _, source := range sources {
		if ctx.Err() != nil {
			break
		}
		versions[source.String()], _ = favthumbs.EntryName(source)
	}
	return versions
}

// ValidateSources runs entry/return stats on a tracked worker. Completion belongs
// to this generation and cannot reopen a retired collection or browsing session.
func (f *Feature) ValidateSources(done func(bool)) {
	if f.stopped {
		return
	}
	if len(f.versions) == 0 {
		done(false)
		return
	}
	if f.validationCancel != nil {
		f.validationCancel()
	}
	ctx, cancel := context.WithCancel(f.lifetime)
	f.validationCancel = cancel
	f.validationRevision++
	revision, generation, queue := f.validationRevision, f.generation, f.ui
	sources, before := slices.Clone(f.sources), maps.Clone(f.versions)
	f.workers.Go(func() {
		after := sourceVersions(ctx, sources)
		changed := !maps.Equal(before, after)
		for _, version := range after {
			// Two unknown versions cannot prove unchanged provider content.
			if version == "" {
				changed = true
				break
			}
		}
		queue.Do(func() {
			if ctx.Err() != nil || f.stopped || generation != f.generation || revision != f.validationRevision {
				return
			}
			cancel()
			f.validationCancel = nil
			f.versions = after
			done(changed)
		})
	})
}

func (f *Feature) readSource(ctx context.Context, uri fyne.URI, owners *favoriteFacts, queue UIQueue, generation uint64, bypass bool) (imaging.Metadata, string, error) {
	if err := ctx.Err(); err != nil {
		return imaging.Metadata{}, "", err
	}
	version, known := favthumbs.EntryName(uri)
	if fact, ok := f.facts.Get(uri.String(), version); ok {
		return fact, version, nil
	}
	writer := f.facts.Capture(uri.String(), version)
	if !bypass {
		if fact, ok, err := owners.load(ctx, uri, version); ok {
			after, afterKnown := favthumbs.EntryName(uri)
			if known != afterKnown || version != after {
				return imaging.Metadata{}, version, errors.New("location source changed during cache read")
			}
			_ = writer.Store(ctx, fact)
			return fact, version, ctx.Err()
		} else {
			f.cacheFailure(queue, ctx, generation, err)
		}
	}
	metadata, err := imaging.ReadMetadataURIContext(ctx, uri)
	if err != nil {
		return imaging.Metadata{}, version, err
	}
	metadata = locationMetadata(metadata)
	after, afterKnown := favthumbs.EntryName(uri)
	if known != afterKnown || version != after {
		return imaging.Metadata{}, version, errors.New("location source changed during metadata read")
	}
	if writer.Store(ctx, metadata) {
		f.cacheFailure(queue, ctx, generation, f.persistFact(ctx, owners, uri, Fact{Version: version, Metadata: metadata}))
	}
	return metadata, version, ctx.Err()
}

func (f *Feature) previews(points []Point, revision uint64) {
	images := make([]image.Image, len(points))
	missing := false
	for i, img := range f.surface.images {
		images[i] = img.Image
		missing = missing || img.Image == nil
	}
	if !missing {
		return
	}
	ctx, cancel := context.WithCancel(f.ctx)
	f.previewCancel = cancel
	generation, queue, writer := f.generation, f.ui, f.thumbs
	f.run(ctx, func() {
		for i, point := range points {
			if ctx.Err() != nil {
				return
			}
			if images[i] != nil {
				continue
			}
			u := point.Source.URI
			version, versioned := favthumbs.EntryName(u)
			if version != point.Version {
				continue
			}
			if cached, ok := writer.Peek(u.String()); ok {
				if preview, ok := cached.(*favthumbs.Preview); ok && versioned && preview.SourceVersion == version {
					images[i] = preview
					continue
				}
			}
			pixels, err := imaging.LoadThumbnailContext(ctx, u)
			if err != nil || ctx.Err() != nil {
				continue
			}
			after, _ := favthumbs.EntryName(u)
			if after != version {
				continue
			}
			preview := &favthumbs.Preview{Image: pixels, SourceVersion: version}
			images[i] = preview
			if versioned {
				_ = writer.AddIfFits(u.String(), preview)
			}
		}
		queue.Do(func() {
			if ctx.Err() != nil || generation != f.generation || revision != f.surface.revision {
				return
			}
			f.surface.setPreviews(images)
		})
	})
}
