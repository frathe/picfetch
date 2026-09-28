package favstore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestFavoriteInventory(t *testing.T) {
	t.Run("membership", func(t *testing.T) {
		dir := t.TempDir()
		for name, data := range map[string]string{
			"A":         `{"0":"/match.jpg","1":"/other.jpg"}`,
			"B":         `{"2":"/match.jpg","8":"/match.jpg"}`,
			"Unrelated": `{"0":"/unrelated.jpg"}`,
			"Unknown":   `{"0":"/match.jpg","99":""}`,
		} {
			if err := os.Mkdir(Dir(dir, name), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(Dir(dir, name), fileListName), []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
		}
		inventory, err := Inventory(context.Background(), dir, []string{"/match.jpg"})
		if err == nil || !inventory.Complete || len(inventory.Favorites) != 3 {
			t.Fatalf("partial inventory lost healthy/unknown owners: %+v, %v", inventory, err)
		}
		healthy, unknown := 0, 0
		for _, favorite := range inventory.Favorites {
			if favorite.Err != nil {
				unknown++
				if favorite.Members != nil || favorite.Owner == nil {
					t.Fatal("unknown owner became membership or lost safe directory")
				}
				continue
			}
			healthy++
			if len(favorite.Members) != 1 || !favorite.Members[filepath.Clean("/match.jpg")] {
				t.Fatalf("wrong scoped membership: %v", favorite.Members)
			}
		}
		if healthy != 2 || unknown != 1 {
			t.Fatalf("healthy=%d unknown=%d", healthy, unknown)
		}
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		cancelled, err := Inventory(ctx, dir, nil)
		if err == nil || cancelled.Complete {
			t.Fatal("cancelled enumeration reported complete inventory")
		}
	})
	t.Run("resources", func(t *testing.T) {
		for _, count := range []int{32, 130} {
			t.Run(fmt.Sprint(count), func(t *testing.T) {
				dir := t.TempDir()
				for i := range count {
					path := filepath.Join(dir, fmt.Sprintf("%03d", i))
					if err := os.Mkdir(path, 0o700); err != nil {
						t.Fatal(err)
					}
					data, _ := json.Marshal(map[string]string{"0": fmt.Sprintf("/unrelated-%d.jpg", i), "1": "/shared.jpg"})
					if err := os.WriteFile(filepath.Join(path, fileListName), data, 0o600); err != nil {
						t.Fatal(err)
					}
				}
				before := favoriteHandles(dir)
				peak := before
				store := Store{readFile: func(ctx context.Context, f *os.File) ([]byte, error) {
					peak = max(peak, favoriteHandles(dir))
					return readDefinition(ctx, f)
				}}
				runtime.GC()
				var startMemory, unrelatedMemory, matchingMemory runtime.MemStats
				runtime.ReadMemStats(&startMemory)
				unrelated, err := store.Inventory(context.Background(), dir, []string{"/absent.jpg"})
				if err != nil || len(unrelated.Favorites) != 0 || !unrelated.Complete {
					t.Fatalf("unrelated retention: %+v, %v", unrelated, err)
				}
				runtime.GC()
				runtime.ReadMemStats(&unrelatedMemory)
				matching, err := store.Inventory(context.Background(), dir, []string{"/shared.jpg"})
				if err != nil || !matching.Complete || len(matching.Favorites) != count {
					t.Fatalf("inventory truncated owners: %d, %v", len(matching.Favorites), err)
				}
				for _, f := range matching.Favorites {
					if len(f.Members) != 1 {
						t.Fatal("retained unrequested full membership")
					}
				}
				after := favoriteHandles(dir)
				if runtime.GOOS == "linux" && (after > before || peak-before > 3) {
					t.Fatalf("unbounded/leaked handles: before=%d peak=%d after=%d", before, peak, after)
				}
				t.Logf("Favorites=%d retained unrelated=%d matching=%d idle handle delta=%d peak delta=%d", count, len(unrelated.Favorites), len(matching.Favorites), after-before, peak-before)
				runtime.GC()
				runtime.ReadMemStats(&matchingMemory)
				t.Logf("retained heap observations (include runtime noise): unrelated delta=%d bytes, matching delta=%d bytes", int64(unrelatedMemory.HeapAlloc)-int64(startMemory.HeapAlloc), int64(matchingMemory.HeapAlloc)-int64(unrelatedMemory.HeapAlloc))
				runtime.KeepAlive(matching)
				runtime.KeepAlive(unrelated)
			})
		}
	})
}

func favoriteHandles(dir string) int {
	entries, _ := os.ReadDir("/proc/self/fd")
	count := 0
	for _, entry := range entries {
		path, _ := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
		if path == dir || strings.HasPrefix(path, dir+string(filepath.Separator)) {
			count++
		}
	}
	return count
}
