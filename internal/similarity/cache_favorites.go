package similarity

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/frathe/picfetch/internal/favstore"
)

type favoriteAnalysis struct {
	lease   *cacheLease
	root    *os.Root
	list    os.FileInfo
	members map[string]bool
}

// A directory handle follows a favorite moved to Trash without recreating its
// old pathname. Replacing the file list invalidates this run's write admission.
type favoriteInventory struct {
	favorites []*favoriteAnalysis
	lease     *cacheLease
	members   map[string][]*favoriteAnalysis
}

// openFavoriteInventory always returns an owned inventory, including on error.
// Its healthy entries remain usable while failed membership stays unknown.
func openFavoriteInventory(ctx context.Context, dir string) (*favoriteInventory, error) {
	cache := &favoriteInventory{members: map[string][]*favoriteAnalysis{}}
	if dir == "" {
		return cache, nil
	}
	parent, err := os.OpenRoot(dir)
	if errors.Is(err, os.ErrNotExist) {
		return cache, nil
	}
	if err != nil {
		return cache, err
	}
	defer func() { _ = parent.Close() }()
	directory, err := parent.Open(".")
	if err != nil {
		return cache, err
	}
	entries, err := directory.ReadDir(-1)
	_ = directory.Close()
	var failures []error
	if err != nil {
		failures = append(failures, err)
	}
	for _, entry := range entries {
		if !entry.IsDir() || !favstore.ValidName(entry.Name()) {
			continue
		}
		if err := ctx.Err(); err != nil {
			return cache, errors.Join(append(failures, err)...)
		}
		// Ownership transfers to the inventory even when membership is unknown:
		// maintenance may still inspect or explicitly clear these records.
		//goland:noinspection GoResourceLeak
		root, err := parent.OpenRoot(entry.Name())
		if err != nil {
			failures = append(failures, err)
			continue
		}
		favorite, err := loadFavoriteAnalysis(root)
		if errors.Is(err, os.ErrNotExist) {
			_ = root.Close()
			continue
		}
		if err != nil {
			failures = append(failures, err)
			favorite = &favoriteAnalysis{root: root}
		}
		cache.favorites = append(cache.favorites, favorite)
		for path := range favorite.members {
			cache.members[path] = append(cache.members[path], favorite)
		}
	}
	return cache, errors.Join(failures...)
}

func loadFavoriteAnalysis(root *os.Root) (*favoriteAnalysis, error) {
	before, err := root.Stat("file-list.json")
	if err != nil {
		return nil, err
	}
	data, err := root.ReadFile("file-list.json")
	if err != nil {
		return nil, err
	}
	var files map[string]string
	if err := json.Unmarshal(data, &files); err != nil {
		return nil, err
	}
	after, err := root.Stat("file-list.json")
	if err != nil {
		return nil, err
	}
	if !sameVersion(before, after) {
		return nil, fmt.Errorf("favorite membership changed during cache admission")
	}
	favorite := &favoriteAnalysis{root: root, list: before, members: map[string]bool{}}
	for key, path := range files {
		index, err := strconv.Atoi(key)
		if err != nil || index < 0 {
			return nil, fmt.Errorf("invalid file index %q", key)
		}
		favorite.members[filepath.Clean(path)] = true
	}
	return favorite, nil
}

func (c *favoriteInventory) close() {
	if c == nil {
		return
	}
	for _, favorite := range c.favorites {
		_ = favorite.root.Close()
	}
	c.lease.close()
}

// Producer leases are separate from inventory: maintenance already owns the
// exclusive lease when it inspects the same Favorite directories.
func openAnalysisCache(ctx context.Context, dir string) (*favoriteInventory, error) {
	cache, inventoryErr := openFavoriteInventory(ctx, dir)
	// The error describes partial membership; the inventory is always owned.
	//goland:noinspection GoDfaErrorMayBeNotNil
	if len(cache.members) == 0 {
		return cache, inventoryErr
	}
	var err error
	cache.lease, err = openCacheLease(ctx, CacheRoots{FavoritesDir: dir}, false)
	if err != nil {
		return cache, errors.Join(inventoryErr, err)
	}
	for _, favorite := range cache.favorites {
		favorite.lease = cache.lease
	}
	return cache, inventoryErr
}

// changedMembers limits explicit-save completion to newly created or replaced
// lists. Unchanged Favorites need no per-record I/O during this refresh.
func (c *favoriteInventory) changedMembers(previous *favoriteInventory) map[string][]*favoriteAnalysis {
	versions := make(map[string]os.FileInfo, len(previous.favorites))
	for _, favorite := range previous.favorites {
		versions[favorite.root.Name()] = favorite.list
	}
	changed := map[string][]*favoriteAnalysis{}
	for _, favorite := range c.favorites {
		before := versions[favorite.root.Name()]
		if before != nil && favorite.list != nil && sameVersion(before, favorite.list) {
			continue
		}
		for path := range favorite.members {
			changed[path] = append(changed[path], favorite)
		}
	}
	return changed
}
