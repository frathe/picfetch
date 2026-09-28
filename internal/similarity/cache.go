package similarity

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/frathe/picfetch/internal/favstore"
)

// RepresentationVersion identifies the cached representation format. Change it
// when the representation or oriented pixel preprocessing changes.
const RepresentationVersion = ModelRevision + "/oriented-bilinear-224-v1"

type cachedRepresentation struct {
	Version string
	Item    Item
}

func sameVersion(a, b os.FileInfo) bool {
	return os.SameFile(a, b) && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}

func (f *favoriteAnalysis) current() bool {
	if f.members == nil || f.owner == nil {
		return false
	}
	access, err := f.owner.Acquire(context.Background())
	if err != nil {
		return false
	}
	_ = access.Close()
	return true
}

func analysisName(path string) string {
	return fmt.Sprintf("analysis/%x.json", sha256.Sum256([]byte(path)))
}

func (c *favoriteInventory) read(ctx context.Context, source Item) (Item, bool) {
	for _, favorite := range c.members[filepath.Clean(source.Path)] {
		if item, hit := favorite.read(ctx, source); hit {
			return item, true
		}
	}
	return Item{}, false
}

func (f *favoriteAnalysis) read(ctx context.Context, source Item) (Item, bool) {
	if f.members == nil {
		return Item{}, false
	}
	access, err := f.owner.Acquire(ctx)
	if err != nil {
		return Item{}, false
	}
	defer func() { _ = access.Close() }()
	file, err := access.Root.Open(analysisName(source.Path))
	if err != nil {
		return Item{}, false
	}
	info, err := file.Stat()
	if err != nil || info.Size() > maximumAnalysisRecordBytes {
		_ = file.Close()
		return Item{}, false
	}
	item, err := decodeRepresentation(file)
	_ = file.Close()
	if err != nil || item.Path != source.Path || item.Size != source.Size || item.ModifiedNS != source.ModifiedNS || access.Current(ctx) != nil {
		return Item{}, false
	}
	return item, true
}

func (c *favoriteInventory) write(ctx context.Context, item Item) error {
	for _, favorite := range c.members[filepath.Clean(item.Path)] {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !favorite.current() {
			continue
		}
		if err := favorite.write(ctx, item); err != nil {
			return err
		}
	}
	return nil
}

func (f *favoriteAnalysis) write(ctx context.Context, item Item) error {
	if f.lease == nil {
		return ErrCacheRetired
	}
	return f.lease.write(ctx, func() error { return f.writeRecord(ctx, item) })
}
func (f *favoriteAnalysis) writeRecord(ctx context.Context, item Item) error {
	access, err := f.owner.Acquire(ctx)
	if errors.Is(err, favstore.ErrRetired) {
		return nil
	}
	if err != nil {
		return err
	}
	defer func() { _ = access.Close() }()
	root := access.Root
	if err := root.Mkdir("analysis", 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	name := "analysis/." + rand.Text() + ".tmp"
	file, err := root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = root.Remove(name) }()
	item.Cohort, item.Position, item.Thumbnail = "", nil, ""
	item.Tags = nil
	err = json.NewEncoder(file).Encode(cachedRepresentation{Version: RepresentationVersion, Item: item})
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := access.Current(ctx); err != nil {
		if errors.Is(err, favstore.ErrRetired) {
			return nil
		}
		return err
	}
	return root.Rename(name, analysisName(item.Path))
}
