package favstore

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Favorite retains scoped membership, or explicit unknown membership, with its owner.
type Favorite struct {
	Owner   *Owner
	Members map[string]bool
	Err     error
}

// InventoryResult distinguishes completed enumeration from unknown definitions.
type InventoryResult struct {
	Favorites []Favorite
	Complete  bool
}

// Inventory validates all definitions and retains membership matching scope.
// A nil scope requests complete membership; an empty non-nil scope matches none.
func Inventory(ctx context.Context, dir string, scope []string) (InventoryResult, error) {
	return (&Store{}).Inventory(ctx, dir, scope)
}

// Inventory uses this store's filesystem operation seam.
func (s *Store) Inventory(ctx context.Context, dir string, scope []string) (InventoryResult, error) {
	var result InventoryResult
	if err := ctx.Err(); err != nil {
		return result, err
	}
	if dir == "" {
		result.Complete = true
		return result, nil
	}
	base, err := os.Getwd()
	if err != nil {
		return result, err
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(base, dir)
	}
	folder, err := os.Open(dir)
	if errors.Is(err, os.ErrNotExist) {
		result.Complete = true
		return result, nil
	}
	if err != nil {
		return result, err
	}
	defer func() { _ = folder.Close() }()
	var wanted map[string]bool
	if scope != nil {
		wanted = make(map[string]bool, len(scope))
		for _, path := range scope {
			wanted[sourceKey(base, path)] = true
		}
	}
	var failures []error
	for {
		if err := ctx.Err(); err != nil {
			return result, errors.Join(append(failures, err)...)
		}
		entries, readErr := folder.ReadDir(64)
		for _, entry := range entries {
			if err := ctx.Err(); err != nil {
				return result, errors.Join(append(failures, err)...)
			}
			if !entry.IsDir() || !ValidName(entry.Name()) {
				continue
			}
			path := filepath.Join(dir, entry.Name())
			definition, err := s.open(ctx, path, base)
			if errors.Is(err, os.ErrNotExist) && !errors.Is(err, ErrRetired) {
				continue
			}
			if err != nil {
				failures = append(failures, fmt.Errorf("favorite %q: %w", entry.Name(), err))
				if ctx.Err() != nil {
					return result, errors.Join(failures...)
				}
				owner, _ := Observe(ctx, path)
				result.Favorites = append(result.Favorites, Favorite{Owner: owner, Err: err})
				continue
			}
			members := make(map[string]bool)
			for _, path := range definition.Paths {
				key := sourceKey(base, path)
				if wanted == nil || wanted[key] {
					members[key] = true
				}
			}
			if wanted == nil || len(members) > 0 {
				result.Favorites = append(result.Favorites, Favorite{Owner: definition.Owner, Members: members})
			}
		}
		if readErr != nil {
			if errors.Is(readErr, io.EOF) {
				result.Complete = true
			} else {
				failures = append(failures, readErr)
			}
			return result, errors.Join(failures...)
		}
	}
}
