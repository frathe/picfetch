package similarity

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/favstore"
)

func TestAnalysisCacheReleasesIdleOwners(t *testing.T) {
	dir := t.TempDir()
	shared := storage.NewFileURI(filepath.Join(dir, "shared.jpg"))
	other := storage.NewFileURI(filepath.Join(dir, "other.jpg"))
	for _, name := range []string{"First", "Second"} {
		if err := favstore.Save(dir, name, []fyne.URI{shared, other}); err != nil {
			t.Fatal(err)
		}
	}
	cache, err := openAnalysisCache(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	defer cache.close()
	for _, name := range []string{"First", "Second"} {
		if err := os.Rename(favstore.Dir(dir, name), filepath.Join(dir, name+"-moved")); err != nil {
			t.Fatalf("idle producer pinned Favorite: %v", err)
		}
	}
	for _, favorite := range cache.favorites {
		if favorite.current() {
			t.Fatal("moved owner remains current")
		}
	}
}

func TestFavoriteAnalysisOwnership(t *testing.T) {
	t.Run("identical_replacement", func(t *testing.T) {
		ctx := context.Background()
		policy := cacheTestPolicy(t)
		item := cacheFixtureItem(t, "member.jpg")
		cacheTestFavorite(t, policy.Roots, item)
		store := cacheTestStore(t, policy)
		if err := store.write(ctx, item); err != nil {
			t.Fatal(err)
		}
		cacheTestFavorite(t, policy.Roots, item)
		stale := item
		stale.Embedding = append([]float32(nil), item.Embedding...)
		stale.Embedding[0] = -1
		if err := store.write(ctx, stale); err != nil {
			t.Fatal(err)
		}
		if _, hit := cacheTestRead(t, store, ctx, item); hit {
			t.Fatal("old owner read through identical list replacement")
		}
		fresh := cacheTestStore(t, policy)
		got, hit := cacheTestRead(t, fresh, ctx, item)
		if !hit || got.Embedding[0] != 1 {
			t.Fatal("retired producer altered a replacement owner's record")
		}
	})
	t.Run("retained_resources", func(t *testing.T) {
		for _, count := range []int{32, 130} {
			t.Run(fmt.Sprint(count), func(t *testing.T) {
				dir := t.TempDir()
				source := storage.NewFileURI(filepath.Join(dir, "shared.jpg"))
				for i := range count {
					if err := favstore.Save(dir, fmt.Sprintf("%03d", i), []fyne.URI{source, storage.NewFileURI(filepath.Join(dir, fmt.Sprintf("unrelated-%d.jpg", i)))}); err != nil {
						t.Fatal(err)
					}
				}
				runtime.GC()
				var beforeMemory, afterMemory runtime.MemStats
				runtime.ReadMemStats(&beforeMemory)
				cache, err := openScopedAnalysisCache(context.Background(), dir, []string{source.Path()})
				if err != nil {
					t.Fatal(err)
				}
				defer cache.close()
				if len(cache.members) != 1 || len(cache.members[filepath.Clean(source.Path())]) != count {
					t.Fatal("producer retained unrelated membership or omitted matching owners")
				}
				runtime.GC()
				runtime.ReadMemStats(&afterMemory)
				entries, _ := os.ReadDir("/proc/self/fd")
				for _, entry := range entries {
					name, _ := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
					for _, favorite := range cache.favorites {
						if name == favorite.owner.Path() || strings.HasPrefix(name, favorite.owner.Path()+string(filepath.Separator)) {
							t.Fatalf("producer retained Favorite handle: %s", name)
						}
					}
				}
				for i := range count {
					if err := os.Rename(favstore.Dir(dir, fmt.Sprintf("%03d", i)), filepath.Join(dir, fmt.Sprintf("moved-%03d", i))); err != nil {
						t.Fatalf("idle owner retained native handle: %v", err)
					}
				}
				t.Logf("producer retained %d owner associations and 1 source; heap delta=%d bytes (includes runtime noise); every idle Favorite moved", count, int64(afterMemory.HeapAlloc)-int64(beforeMemory.HeapAlloc))
				runtime.KeepAlive(cache)
			})
		}
	})
	t.Run("original_scope", func(t *testing.T) {
		for _, preparedCount := range []int{0, 1, 2} {
			t.Run(fmt.Sprint(preparedCount), func(t *testing.T) {
				ctx := context.Background()
				policy := cacheTestPolicy(t)
				first, later, unrelated := cacheFixtureItem(t, "first.jpg"), cacheFixtureItem(t, "later.jpg"), cacheFixtureItem(t, "unrelated.jpg")
				cacheTestFavorite(t, policy.Roots, first, later, unrelated)
				req := request{Paths: []string{first.Path, later.Path}, FavoritesDir: policy.Roots.FavoritesDir}
				store, err := openAnalyzerRepresentationStore(ctx, req)
				if err != nil {
					t.Fatal(err)
				}
				defer store.close()
				if len(store.favorites.members) != 2 {
					t.Fatalf("producer retained unrelated membership: %d", len(store.favorites.members))
				}
				if err := favstore.Save(policy.Roots.FavoritesDir, "Second", []fyne.URI{storage.NewFileURI(first.Path), storage.NewFileURI(later.Path), storage.NewFileURI(unrelated.Path)}); err != nil {
					t.Fatal(err)
				}
				prepared := []Item{later, first}[:preparedCount]
				if err := store.refreshFavorites(ctx, prepared, func(_ context.Context, item Item) (Item, error) { return item, nil }); err != nil {
					t.Fatal(err)
				}
				if len(store.favorites.members) != 2 || len(store.favorites.members[first.Path]) != 2 || len(store.favorites.members[later.Path]) != 2 {
					t.Fatal("refresh lost original scope or a matching owner")
				}
				for _, item := range []Item{first, later} {
					if err := store.write(ctx, item); err != nil {
						t.Fatal(err)
					}
					for _, name := range []string{"Trip", "Second"} {
						if _, err := os.Stat(filepath.Join(favstore.Dir(policy.Roots.FavoritesDir, name), analysisName(item.Path))); err != nil {
							t.Fatalf("full-scope owner lost eventual write: %v", err)
						}
					}
				}
			})
		}
	})
	for _, replacement := range []bool{false, true} {
		name := "moved"
		if replacement {
			name = "replaced"
		}
		t.Run(name, func(t *testing.T) {
			dir := t.TempDir()
			original := filepath.Join(dir, "original.jpg")
			other := filepath.Join(dir, "other.jpg")
			if err := favstore.Save(dir, "Trip", []fyne.URI{storage.NewFileURI(original)}); err != nil {
				t.Fatal(err)
			}
			cache, err := openAnalysisCache(context.Background(), dir)
			if err != nil {
				t.Fatal(err)
			}
			defer cache.close()
			favorite := cache.favorites[0]
			moveErr := os.Rename(favstore.Dir(dir, "Trip"), filepath.Join(dir, "Moved"))
			if moveErr != nil {
				t.Fatalf("idle owner prevents Favorite move: %v", moveErr)
			}
			if replacement && moveErr == nil {
				if err := favstore.Save(dir, "Trip", []fyne.URI{storage.NewFileURI(other)}); err != nil {
					t.Fatal(err)
				}
			}
			if favorite.current() {
				t.Fatal("producer followed a moved Favorite")
			}
			if _, hit := favorite.read(context.Background(), Item{Path: original}); hit {
				t.Fatal("retired owner reused a record")
			}
			if !replacement {
				if _, err := os.Stat(favstore.Dir(dir, "Trip")); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("old owner recreated path: %v", err)
				}
			} else {
				fresh, err := openAnalysisCache(context.Background(), dir)
				if err != nil {
					t.Fatal(err)
				}
				defer fresh.close()
				if len(fresh.members[other]) != 1 || !fresh.members[other][0].current() {
					t.Fatal("fresh replacement owner unavailable")
				}
			}
		})
	}
}
