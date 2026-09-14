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
	if f.list == nil {
		return false
	}
	now, err := f.root.Stat("file-list.json")
	return err == nil && sameVersion(f.list, now)
}

func analysisName(path string) string {
	return fmt.Sprintf("analysis/%x.json", sha256.Sum256([]byte(path)))
}

func (c *favoriteInventory) read(source Item) (Item, bool) {
	for _, favorite := range c.members[filepath.Clean(source.Path)] {
		if item, hit := favorite.read(source); hit {
			return item, true
		}
	}
	return Item{}, false
}

func (f *favoriteAnalysis) read(source Item) (Item, bool) {
	if !f.current() {
		return Item{}, false
	}
	file, err := f.root.Open(analysisName(source.Path))
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
	if err != nil || item.Path != source.Path || item.Size != source.Size || item.ModifiedNS != source.ModifiedNS {
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
	if err := f.root.Mkdir("analysis", 0o755); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	name := "analysis/." + rand.Text() + ".tmp"
	file, err := f.root.OpenFile(name, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer func() { _ = f.root.Remove(name) }()
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
	if !f.current() {
		return nil
	}
	return f.root.Rename(name, analysisName(item.Path))
}
