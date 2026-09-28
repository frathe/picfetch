package locationmap

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/favthumbs"
	"github.com/frathe/picfetch/internal/imaging"
)

const favoriteGPSDirectory = ".location-map"
const maxGPSRecordBytes = 64 * 1024
const maxFavoriteCleanupEntries = 1024

type favoriteOwner struct {
	owner     *favstore.Owner
	namespace string
	members   map[string]bool
}
type favoriteFacts struct {
	owners  []*favoriteOwner
	members map[string][]*favoriteOwner
}
type gpsRecord struct {
	Schema        int
	Path, Version string
	Metadata      imaging.Metadata
}

func (c *favoriteFacts) close() {
	c.owners = nil
	c.members = nil
}

func openFavoriteFacts(ctx context.Context, dir string, sources []fyne.URI) (*favoriteFacts, error) {
	c := &favoriteFacts{members: map[string][]*favoriteOwner{}}
	if err := ctx.Err(); err != nil {
		return c, err
	}
	if dir == "" || len(sources) == 0 {
		return c, nil
	}
	paths := make([]string, len(sources))
	for i, source := range sources {
		paths[i] = source.Path()
	}
	inventory, inventoryErr := favstore.Inventory(ctx, dir, paths)
	var failures []error
	if inventoryErr != nil {
		failures = append(failures, inventoryErr)
	}
	for _, favorite := range inventory.Favorites {
		if err := ctx.Err(); err != nil {
			return c, errors.Join(append(failures, err)...)
		}
		if favorite.Err != nil {
			continue
		}
		owner := &favoriteOwner{owner: favorite.Owner, namespace: favorite.Owner.Version(), members: favorite.Members}
		if err := owner.prepare(ctx); err != nil {
			if !errors.Is(err, favstore.ErrRetired) {
				failures = append(failures, err)
			}
			continue
		}
		c.owners = append(c.owners, owner)
		for path := range owner.members {
			c.members[path] = append(c.members[path], owner)
		}
	}
	return c, errors.Join(failures...)
}

func (o *favoriteOwner) prepare(ctx context.Context) error {
	access, err := o.owner.Acquire(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = access.Close() }()
	if err := access.Root.Mkdir(favoriteGPSDirectory, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	cache, err := access.Root.OpenRoot(favoriteGPSDirectory)
	if err != nil {
		return err
	}
	defer func() { _ = cache.Close() }()
	if err := access.Current(ctx); err != nil {
		return err
	}
	if err := cache.Mkdir(o.namespace, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	return o.cleanupRetired(ctx, access, cache, o.namespace)
}

func (o *favoriteOwner) records(ctx context.Context) (*favstore.Access, *os.Root, error) {
	access, err := o.owner.Acquire(ctx)
	if err != nil {
		return nil, nil, err
	}
	records, err := access.Root.OpenRoot(filepath.Join(favoriteGPSDirectory, o.namespace))
	if err != nil {
		_ = access.Close()
		return nil, nil, err
	}
	return access, records, nil
}

// Cleanup is best-effort and bounded across directory and record entries.
// Unexpected nested trees are never traversed. A later visit may reclaim more;
// namespace isolation does not depend on completing this maintenance.
func (o *favoriteOwner) cleanupRetired(ctx context.Context, access *favstore.Access, cache *os.Root, namespace string) error {
	folder, err := cache.Open(".")
	if err != nil {
		return ctx.Err()
	}
	defer func() { _ = folder.Close() }()
	remaining := maxFavoriteCleanupEntries
	for remaining > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries, readErr := folder.ReadDir(min(64, remaining))
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if remaining == 0 || access.Current(ctx) != nil {
				return nil
			}
			remaining--
			name := entry.Name()
			if !entry.IsDir() || name == namespace || len(name) != 64 {
				continue
			}
			if _, err := hex.DecodeString(name); err != nil {
				continue
			}
			records, err := cache.OpenRoot(name)
			if err != nil {
				continue
			}
			err = o.cleanRetiredRecords(ctx, access, records, &remaining)
			_ = records.Close()
			if err != nil {
				return err
			}
			if err := ctx.Err(); err != nil {
				return err
			}
			if access.Current(ctx) == nil {
				_ = cache.Remove(name) // Only succeeds for an empty namespace.
			}
		}
		if readErr != nil {
			break
		}
	}
	return ctx.Err()
}

func (o *favoriteOwner) cleanRetiredRecords(ctx context.Context, access *favstore.Access, records *os.Root, remaining *int) error {
	folder, err := records.Open(".")
	if err != nil {
		return ctx.Err()
	}
	defer func() { _ = folder.Close() }()
	for *remaining > 0 {
		if err := ctx.Err(); err != nil {
			return err
		}
		entries, readErr := folder.ReadDir(min(64, *remaining))
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return err
			}
			if access.Current(ctx) != nil {
				return nil
			}
			*remaining -= 1
			name := entry.Name()
			if entry.IsDir() {
				continue
			}
			if strings.HasPrefix(name, ".") && strings.HasSuffix(name, ".tmp") {
				_ = records.Remove(name)
			} else if len(name) == 69 && strings.HasSuffix(name, ".json") {
				if _, err := hex.DecodeString(name[:64]); err == nil {
					_ = records.Remove(name)
				}
			}
		}
		if readErr != nil {
			break
		}
	}
	return ctx.Err()
}

func gpsRecordName(path string) string {
	// One record per member bounds disk residency across source revisions and
	// makes explicit committed-write invalidation independent of old versions.
	// The JSON record still validates the precise source version on every hit.
	sum := sha256.Sum256([]byte(filepath.Clean(path)))
	return hex.EncodeToString(sum[:]) + ".json"
}

func (c *favoriteFacts) load(ctx context.Context, source fyne.URI, version string) (imaging.Metadata, bool, error) {
	if version == "" {
		return imaging.Metadata{}, false, nil
	}
	for _, owner := range c.members[filepath.Clean(source.Path())] {
		if ctx.Err() != nil {
			return imaging.Metadata{}, false, ctx.Err()
		}
		metadata, hit, err := owner.load(ctx, source, version)
		if errors.Is(err, os.ErrNotExist) || errors.Is(err, favstore.ErrRetired) {
			continue
		}
		if err != nil || hit {
			return metadata, hit, err
		}
	}
	return imaging.Metadata{}, false, nil
}

func (o *favoriteOwner) load(ctx context.Context, source fyne.URI, version string) (imaging.Metadata, bool, error) {
	access, records, err := o.records(ctx)
	if err != nil {
		return imaging.Metadata{}, false, err
	}
	defer func() { _ = records.Close(); _ = access.Close() }()
	file, err := records.Open(gpsRecordName(source.Path()))
	if err != nil {
		return imaging.Metadata{}, false, err
	}
	data, err := io.ReadAll(io.LimitReader(file, maxGPSRecordBytes+1))
	_ = file.Close()
	if err != nil {
		return imaging.Metadata{}, false, err
	}
	var record gpsRecord
	if len(data) > maxGPSRecordBytes || json.Unmarshal(data, &record) != nil || record.Schema != 1 || record.Path != filepath.Clean(source.Path()) || record.Version != version || record.Metadata.HasGPS && !validLocation(record.Metadata) {
		return imaging.Metadata{}, false, nil
	}
	if err := access.Current(ctx); err != nil {
		return imaging.Metadata{}, false, err
	}
	return locationMetadata(record.Metadata), true, nil
}

func (c *favoriteFacts) store(ctx context.Context, source fyne.URI, fact Fact) error {
	if fact.Version == "" {
		return nil
	}
	version, ok := favthumbs.EntryName(source)
	if !ok || version != fact.Version {
		return nil
	}
	data, err := json.Marshal(gpsRecord{Schema: 1, Path: filepath.Clean(source.Path()), Version: version, Metadata: locationMetadata(fact.Metadata)})
	if err != nil {
		return err
	}
	if len(data) > maxGPSRecordBytes {
		return errors.New("location metadata exceeds cache record limit")
	}
	var failures []error
	for _, owner := range c.members[filepath.Clean(source.Path())] {
		if err := ctx.Err(); err != nil {
			return err
		}
		if err := owner.publish(ctx, gpsRecordName(source.Path()), data); err != nil && !errors.Is(err, favstore.ErrRetired) {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

// persistFact serializes publication with committed-write cleanup. Revalidate
// live raw authority inside that boundary so a delayed promotion cannot restore
// a fact that was invalidated after it took its memory snapshot.
func (f *Feature) persistFact(ctx context.Context, owners *favoriteFacts, source fyne.URI, fact Fact) error {
	f.persistence.Lock()
	defer f.persistence.Unlock()
	current, ok := f.facts.Get(source.String(), fact.Version)
	if !ok || current != fact.Metadata || ctx.Err() != nil {
		return nil
	}
	return owners.store(ctx, source, fact)
}

func (f *Feature) invalidatePersistentFacts(queue UIQueue, dir string, sources []fyne.URI) {
	f.persistence.Lock()
	defer f.persistence.Unlock()
	ctx := f.lifetime
	owners, err := openFavoriteFacts(ctx, dir, sources)
	// Partial inventory is non-nil even when some Favorite owners are unavailable.
	//goland:noinspection GoDfaErrorMayBeNotNil
	defer owners.close()
	f.cacheFailure(queue, f.lifetime, 0, err)
	for _, source := range sources {
		if ctx.Err() != nil {
			return
		}
		for _, owner := range owners.members[filepath.Clean(source.Path())] {
			if ctx.Err() != nil {
				return
			}
			err := owner.remove(ctx, gpsRecordName(source.Path()))
			if err != nil && !errors.Is(err, os.ErrNotExist) && !errors.Is(err, favstore.ErrRetired) {
				f.cacheFailure(queue, f.lifetime, 0, err)
			}
		}
		// A replacement scan may have won the persistence mutex first. Restore
		// only its current, version-validated raw fact after removing stale disk
		// state, so delayed cleanup cannot erase the scan's valid publication.
		if ctx.Err() != nil {
			return
		}
		version, known := favthumbs.EntryName(source)
		if metadata, ok := f.facts.Get(source.String(), version); known && ok {
			f.cacheFailure(queue, f.lifetime, 0, owners.store(ctx, source, Fact{Version: version, Metadata: metadata}))
		}
	}
}

func (o *favoriteOwner) remove(ctx context.Context, name string) error {
	access, records, err := o.records(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = records.Close(); _ = access.Close() }()
	if err := access.Current(ctx); err != nil {
		return err
	}
	return records.Remove(name)
}

func (o *favoriteOwner) publish(ctx context.Context, name string, data []byte) error {
	access, records, err := o.records(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = records.Close(); _ = access.Close() }()
	temporary := "." + rand.Text() + ".tmp"
	file, err := records.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = records.Remove(temporary) }()
	_, writeErr := file.Write(data)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := access.Current(ctx); err != nil {
		return err
	}
	return records.Rename(temporary, name)
}

// SetFavoritesRoot captures root-owned configuration without filesystem work.
func (f *Feature) SetFavoritesRoot(dir string) { f.favoriteRoot.Store(dir) }

// FavoritesSaved promotes known raw facts without reopening the collection.
func (f *Feature) FavoritesSaved(dir string) {
	f.SetFavoritesRoot(dir)
	if f.stopped {
		return
	}
	ctx := f.lifetime
	queue := f.ui
	sources := append([]fyne.URI(nil), f.sources...)
	f.workers.Go(func() { f.persistKnown(queue, ctx, dir, sources, 0) })
}

func (f *Feature) persistKnown(queue UIQueue, ctx context.Context, dir string, sources []fyne.URI, generation uint64) {
	if ctx.Err() != nil {
		return
	}
	owners, err := openFavoriteFacts(ctx, dir, sources)
	// The inventory is always non-nil; errors describe partially unavailable owners.
	//goland:noinspection GoDfaErrorMayBeNotNil
	defer owners.close()
	f.cacheFailure(queue, ctx, generation, err)
	for _, source := range sources {
		if ctx.Err() != nil {
			return
		}
		version, _ := favthumbs.EntryName(source)
		if fact, ok := f.facts.Get(source.String(), version); ok {
			f.cacheFailure(queue, ctx, generation, f.persistFact(ctx, owners, source, Fact{Version: version, Metadata: fact}))
		}
	}
}

func (f *Feature) cacheFailure(queue UIQueue, ctx context.Context, generation uint64, err error) {
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, favstore.ErrRetired) {
		return
	}
	queue.Do(func() {
		if ctx.Err() != nil || f.stopped || generation != 0 && generation != f.generation || f.cacheWarned {
			return
		}
		f.cacheWarned = true
		fyne.LogError("could not cache Favorite locations", fmt.Errorf("location cache: %w", err))
		f.host.ShowToast(lang.L("Could not cache Favorite locations. Locations remain available in memory."))
	})
}
