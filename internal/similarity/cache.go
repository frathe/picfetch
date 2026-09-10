package similarity

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"image/jpeg"
	"io"
	"math"
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

type favoriteAnalysis struct {
	root    *os.Root
	list    os.FileInfo
	members map[string]bool
}

// A directory handle follows a favorite moved to Trash without recreating its
// old pathname. Replacing the file list invalidates this run's write admission.
type analysisCache map[string][]*favoriteAnalysis

func openAnalysisCache(ctx context.Context, dir string) (analysisCache, error) {
	cache := analysisCache{}
	if dir == "" {
		return cache, nil
	}
	names, err := favstore.List(dir)
	if err != nil {
		return cache, err
	}
	for _, name := range names {
		if err := ctx.Err(); err != nil {
			return cache, err
		}
		root, err := os.OpenRoot(favstore.Dir(dir, name))
		if err != nil {
			continue
		}
		before, err := root.Stat("file-list.json")
		if err != nil {
			_ = root.Close()
			continue
		}
		files, err := favstore.Load(dir, name)
		after, statErr := root.Stat("file-list.json")
		if err != nil || statErr != nil || !sameVersion(before, after) {
			_ = root.Close()
			continue
		}
		favorite := &favoriteAnalysis{root: root, list: before, members: map[string]bool{}}
		for _, file := range files {
			path := filepath.Clean(file.Path())
			if !favorite.members[path] {
				favorite.members[path] = true
				cache[path] = append(cache[path], favorite)
			}
		}
		if len(favorite.members) == 0 {
			_ = root.Close()
		}
	}
	return cache, nil
}

func (c analysisCache) close() {
	closed := map[*favoriteAnalysis]bool{}
	for _, favorites := range c {
		for _, favorite := range favorites {
			if !closed[favorite] {
				_ = favorite.root.Close()
				closed[favorite] = true
			}
		}
	}
}

func sameVersion(a, b os.FileInfo) bool {
	return os.SameFile(a, b) && a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}

func (f *favoriteAnalysis) current() bool {
	now, err := f.root.Stat("file-list.json")
	return err == nil && sameVersion(f.list, now)
}

func analysisName(path string) string {
	return fmt.Sprintf("analysis/%x.json", sha256.Sum256([]byte(path)))
}

func (c analysisCache) read(source Item) (Item, bool) {
	for _, favorite := range c[filepath.Clean(source.Path)] {
		if !favorite.current() {
			continue
		}
		file, err := favorite.root.Open(analysisName(source.Path))
		if err != nil {
			continue
		}
		var entry cachedRepresentation
		err = json.NewDecoder(io.LimitReader(file, 1<<20)).Decode(&entry)
		_ = file.Close()
		item := entry.Item
		if err != nil || entry.Version != RepresentationVersion || item.Path != source.Path || item.Size != source.Size || item.ModifiedNS != source.ModifiedNS || item.Error != "" || len(item.Embedding) != 768 {
			continue
		}
		hash, err := hex.DecodeString(item.SHA256)
		if err != nil || len(hash) != sha256.Size {
			continue
		}
		config, err := jpeg.DecodeConfig(bytes.NewReader(item.Preview))
		if err != nil || config.Width <= 0 || config.Height <= 0 || config.Width > 160 || config.Height > 160 {
			continue
		}
		norm := 0.0
		for _, value := range item.Embedding {
			norm += float64(value) * float64(value)
		}
		if norm == 0 || math.IsNaN(norm) || math.IsInf(norm, 0) {
			continue
		}
		item.Cohort, item.Position, item.Thumbnail = "", nil, ""
		item.Tags = nil
		return item, true
	}
	return Item{}, false
}

func (c analysisCache) write(ctx context.Context, item Item) error {
	for _, favorite := range c[filepath.Clean(item.Path)] {
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
