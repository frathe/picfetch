package similarity

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

var ErrCacheRetired = errors.New("analysis cache writer retired")

type cacheLeaseRoot struct {
	path  string
	root  *os.Root
	lock  *os.File
	epoch string
}
type cacheLease struct {
	roots []*cacheLeaseRoot
	gate  chan struct{}
}

func openCacheLease(ctx context.Context, roots CacheRoots, create bool) (*cacheLease, error) {
	lease := &cacheLease{gate: make(chan struct{}, 1)}
	lease.gate <- struct{}{}
	paths := []string{roots.GeneralDir, roots.FavoritesDir}
	slices.Sort(paths)
	paths = slices.Compact(paths)
	for _, path := range paths {
		if path == "" {
			continue
		}
		if !filepath.IsAbs(path) {
			lease.close()
			return nil, fmt.Errorf("cache root must be absolute")
		}
		if create {
			if err := os.MkdirAll(path, 0700); err != nil {
				lease.close()
				return nil, err
			}
		}
		root, err := os.OpenRoot(path)
		if errors.Is(err, os.ErrNotExist) && !create {
			continue
		}
		if err != nil {
			lease.close()
			return nil, err
		}
		file, err := root.OpenFile(".analysis-lock", os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			_ = root.Close()
			lease.close()
			return nil, err
		}
		lease.roots = append(lease.roots, &cacheLeaseRoot{path: path, root: root, lock: file})
	}
	release, err := lease.lock(ctx)
	if err != nil {
		lease.close()
		return nil, err
	}
	defer release()
	for _, entry := range lease.roots {
		data, err := entry.root.ReadFile(".analysis-epoch")
		if errors.Is(err, os.ErrNotExist) {
			data = []byte(rand.Text())
			err = entry.root.WriteFile(".analysis-epoch", data, 0600)
		}
		if err != nil {
			lease.close()
			return nil, err
		}
		entry.epoch = string(data)
	}
	return lease, nil
}
func (l *cacheLease) close() {
	if l == nil {
		return
	}
	for _, entry := range l.roots {
		_ = entry.lock.Close()
		_ = entry.root.Close()
	}
}
func (l *cacheLease) lock(ctx context.Context) (func(), error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-l.gate:
	}
	locked := 0
	release := func() {
		for i := locked - 1; i >= 0; i-- {
			unlockCacheFile(l.roots[i].lock)
		}
		l.gate <- struct{}{}
	}
	for _, entry := range l.roots {
		if err := lockCacheFile(ctx, entry.lock); err != nil {
			release()
			return nil, err
		}
		locked++
	}
	return release, nil
}
func (l *cacheLease) current() bool {
	for _, entry := range l.roots {
		data, err := entry.root.ReadFile(".analysis-epoch")
		if err != nil || string(data) != entry.epoch {
			return false
		}
	}
	return true
}
func (l *cacheLease) write(ctx context.Context, write func() error) error {
	if l == nil {
		return write()
	}
	release, err := l.lock(ctx)
	if err != nil {
		return err
	}
	defer release()
	if err := ctx.Err(); err != nil {
		return err
	}
	if !l.current() {
		return ErrCacheRetired
	}
	return write()
}
func (l *cacheLease) invalidate() error {
	for _, entry := range l.roots {
		if err := entry.root.WriteFile(".analysis-epoch", []byte(rand.Text()), 0600); err != nil {
			return err
		}
	}
	return nil
}
