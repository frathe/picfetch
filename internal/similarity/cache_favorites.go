package similarity

import (
	"context"
	"errors"

	"github.com/frathe/picfetch/internal/favstore"
)

type favoriteAnalysis struct {
	lease   *cacheLease
	owner   *favstore.Owner
	members map[string]bool
}

type favoriteInventory struct {
	favorites []*favoriteAnalysis
	lease     *cacheLease
	members   map[string][]*favoriteAnalysis
}

// Maintenance requests complete membership; producer admission below supplies
// its original input scope. Unknown membership never becomes an empty owner.
func openFavoriteInventory(ctx context.Context, dir string) (*favoriteInventory, error) {
	return scopedFavoriteInventory(ctx, dir, nil)
}

func scopedFavoriteInventory(ctx context.Context, dir string, scope []string) (*favoriteInventory, error) {
	cache := &favoriteInventory{members: map[string][]*favoriteAnalysis{}}
	inventory, err := favstore.Inventory(ctx, dir, scope)
	// The result is a value; per-owner failures preserve healthy observations.
	//goland:noinspection GoDfaErrorMayBeNotNil
	for _, entry := range inventory.Favorites {
		if entry.Owner == nil {
			continue
		}
		favorite := &favoriteAnalysis{owner: entry.Owner, members: entry.Members}
		cache.favorites = append(cache.favorites, favorite)
		for path := range favorite.members {
			cache.members[path] = append(cache.members[path], favorite)
		}
	}
	return cache, err
}

func (c *favoriteInventory) close() {
	if c != nil {
		c.lease.close()
	}
}

// Producer leases remain separate from the shared membership observation.
func openAnalysisCache(ctx context.Context, dir string) (*favoriteInventory, error) {
	return openScopedAnalysisCache(ctx, dir, nil)
}

func openScopedAnalysisCache(ctx context.Context, dir string, scope []string) (*favoriteInventory, error) {
	cache, inventoryErr := scopedFavoriteInventory(ctx, dir, scope)
	// A partial inventory is always returned and remains owned by the caller.
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

// Explicit save refresh completes only new/replaced owners, without changing
// producer write policy or losing unprepared members of its original scope.
func (c *favoriteInventory) changedMembers(previous *favoriteInventory) map[string][]*favoriteAnalysis {
	versions := make(map[string]*favstore.Owner, len(previous.favorites))
	for _, favorite := range previous.favorites {
		versions[favorite.owner.Path()] = favorite.owner
	}
	changed := map[string][]*favoriteAnalysis{}
	for _, favorite := range c.favorites {
		if favorite.owner.Same(versions[favorite.owner.Path()]) {
			continue
		}
		for path := range favorite.members {
			changed[path] = append(changed[path], favorite)
		}
	}
	return changed
}
