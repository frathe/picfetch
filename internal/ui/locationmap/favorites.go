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
	"strconv"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/lang"

	"github.com/frathe/picfetch/internal/favstore"
	"github.com/frathe/picfetch/internal/favthumbs"
	"github.com/frathe/picfetch/internal/imaging"
)

const favoriteGPSDirectory = ".location-map"
const maxGPSRecordBytes = 64 * 1024

var errFavoriteRetired = errors.New("location Favorite owner retired")

type favoriteOwner struct {
	dir             string
	root, records   *os.Root
	directory, list os.FileInfo
	members         map[string]bool
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
	for _, owner := range c.owners {
		_ = owner.records.Close()
		_ = owner.root.Close()
	}
}

func openFavoriteFacts(ctx context.Context, dir string) (*favoriteFacts, error) {
	c := &favoriteFacts{members: map[string][]*favoriteOwner{}}
	if dir == "" {
		return c, nil
	}
	names, err := favstore.List(dir)
	if err != nil {
		return c, err
	}
	var failures []error
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return c, err
		}
		owner, err := openFavoriteOwner(favstore.Dir(dir, name))
		if err != nil {
			if !errors.Is(err, errFavoriteRetired) {
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

func sameFavoriteVersion(a, b os.FileInfo) bool {
	return os.SameFile(a, b) && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}

func openFavoriteOwner(dir string) (owner *favoriteOwner, err error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			_ = root.Close()
		}
	}()
	list, err := root.Stat("file-list.json")
	if err != nil {
		return nil, err
	}
	file, err := root.Open("file-list.json")
	if err != nil {
		return nil, err
	}
	data, err := io.ReadAll(io.LimitReader(file, 16*1024*1024+1))
	_ = file.Close()
	if err != nil {
		return nil, err
	}
	if len(data) > 16*1024*1024 {
		return nil, errors.New("favorite membership exceeds location cache limit")
	}
	var members map[string]string
	if err := json.Unmarshal(data, &members); err != nil {
		return nil, err
	}
	directory, err := root.Stat(".")
	if err != nil {
		return nil, err
	}
	owner = &favoriteOwner{dir: dir, root: root, directory: directory, list: list, members: map[string]bool{}}
	for index, path := range members {
		i, e := strconv.Atoi(index)
		if e != nil || i < 0 {
			return nil, errors.New("invalid Favorite member index")
		}
		owner.members[filepath.Clean(path)] = true
	}
	if !owner.current() {
		return nil, errFavoriteRetired
	}
	// Separate immutable membership namespaces keep an old producer's atomic
	// rename out of a replacement Favorite even across a last-moment list change.
	digest := sha256.Sum256(append(data, []byte(list.ModTime().UTC().Format("20060102T150405.000000000"))...))
	namespace := hex.EncodeToString(digest[:])
	if err := root.Mkdir(favoriteGPSDirectory, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, err
	}
	cache, err := root.OpenRoot(favoriteGPSDirectory)
	if err != nil {
		return nil, err
	}
	defer func() { _ = cache.Close() }()
	if err := cache.Mkdir(namespace, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return nil, err
	}
	records, err := cache.OpenRoot(namespace)
	if err != nil {
		return nil, err
	}
	owner.records = records
	if !owner.current() {
		_ = records.Close()
		return nil, errFavoriteRetired
	}
	// Old handles cannot recreate these retired directories. Only our exact
	// hexadecimal membership namespaces are eligible for disposable cleanup.
	if folder, e := cache.Open("."); e == nil {
		entries, e := folder.ReadDir(-1)
		_ = folder.Close()
		if e == nil {
			for _, entry := range entries {
				if !owner.current() {
					break
				}
				if entry.IsDir() && entry.Name() != namespace && len(entry.Name()) == 64 {
					if _, e := hex.DecodeString(entry.Name()); e == nil {
						_ = cache.RemoveAll(entry.Name())
					}
				}
			}
		}
	}
	return owner, nil
}

func (o *favoriteOwner) current() bool {
	directory, err := os.Stat(o.dir)
	if err != nil || !os.SameFile(directory, o.directory) {
		return false
	}
	list, err := o.root.Stat("file-list.json")
	return err == nil && sameFavoriteVersion(list, o.list)
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
		if !owner.current() {
			continue
		}
		file, err := owner.records.Open(gpsRecordName(source.Path()))
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
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
			continue
		}
		if ctx.Err() == nil && owner.current() {
			return record.Metadata, true, nil
		}
	}
	return imaging.Metadata{}, false, nil
}

func (c *favoriteFacts) store(ctx context.Context, source fyne.URI, fact Fact) error {
	if fact.Version == "" {
		return nil
	}
	version, ok := favthumbs.EntryName(source)
	if !ok || version != fact.Version {
		return nil
	}
	data, err := json.Marshal(gpsRecord{Schema: 1, Path: filepath.Clean(source.Path()), Version: version, Metadata: fact.Metadata})
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
		if !owner.current() {
			continue
		}
		if err := owner.publish(ctx, gpsRecordName(source.Path()), data); err != nil && !errors.Is(err, errFavoriteRetired) {
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
	ctx := context.Background()
	owners, err := openFavoriteFacts(ctx, dir)
	// Partial inventory is non-nil even when some Favorite owners are unavailable.
	//goland:noinspection GoDfaErrorMayBeNotNil
	defer owners.close()
	f.cacheFailure(queue, f.lifetime, 0, err)
	for _, source := range sources {
		for _, owner := range owners.members[filepath.Clean(source.Path())] {
			if !owner.current() {
				continue
			}
			err := owner.records.Remove(gpsRecordName(source.Path()))
			if err != nil && !errors.Is(err, os.ErrNotExist) && owner.current() {
				f.cacheFailure(queue, f.lifetime, 0, err)
			}
		}
	}
}

func (o *favoriteOwner) publish(ctx context.Context, name string, data []byte) error {
	temporary := "." + rand.Text() + ".tmp"
	file, err := o.records.OpenFile(temporary, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		if !o.current() {
			return errFavoriteRetired
		}
		return err
	}
	defer func() { _ = o.records.Remove(temporary) }()
	_, writeErr := file.Write(data)
	syncErr := file.Sync()
	closeErr := file.Close()
	if err := errors.Join(writeErr, syncErr, closeErr); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if !o.current() {
		return errFavoriteRetired
	}
	return o.records.Rename(temporary, name)
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
	owners, err := openFavoriteFacts(ctx, dir)
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
	if err == nil || errors.Is(err, context.Canceled) || errors.Is(err, errFavoriteRetired) {
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
