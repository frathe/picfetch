package similarity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/frathe/picfetch/internal/favstore"
)

func TestAnalysisCacheUsageAndClearConfinement(t *testing.T) {
	roots := CacheRoots{GeneralDir: t.TempDir(), FavoritesDir: t.TempDir()}
	records := filepath.Join(roots.GeneralDir, "v1")
	if err := os.Mkdir(records, 0700); err != nil {
		t.Fatal(err)
	}
	managed := filepath.Join(records, strings.Repeat("a", 64)+".json")
	untouched := filepath.Join(roots.GeneralDir, "original.jpg")
	for _, p := range []string{managed, untouched} {
		if err := os.WriteFile(p, []byte("12345"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	quiesced := false
	manager := CacheManager{Quiesce: func(_ context.Context, _ CacheRoots) error { quiesced = true; return nil }}
	usage, err := manager.Inspect(context.Background(), roots, nil)
	if err != nil || usage.General.Bytes != 5 || usage.General.Records != 1 || quiesced {
		t.Fatalf("usage %+v quiesced=%v: %v", usage, quiesced, err)
	}
	report, err := manager.Clean(context.Background(), CacheCleanRequest{Roots: roots, Mode: ClearAll}, nil)
	if err != nil || !quiesced || report.RemovedBytes != 5 || report.RemovedRecords != 1 || report.Remaining.General.Bytes != 0 {
		t.Fatalf("cleanup %+v quiesced=%v: %v", report, quiesced, err)
	}
	if _, err := os.Stat(untouched); err != nil {
		t.Fatalf("cleanup touched original: %v", err)
	}
}

func TestAnalysisCacheMaintenanceOwnership(t *testing.T) {
	t.Run("unknown_membership", func(t *testing.T) {
		for _, mode := range []CacheCleanMode{ClearAll, RemoveStale} {
			t.Run(fmt.Sprint(mode), func(t *testing.T) {
				roots := CacheRoots{FavoritesDir: t.TempDir()}
				dir := favstore.Dir(roots.FavoritesDir, "Unknown")
				cacheTestWriteFile(t, filepath.Join(dir, "file-list.json"), []byte(`{"0":"/member.jpg","1":""}`))
				record := filepath.Join(dir, "analysis", strings.Repeat("a", 64)+".json")
				cacheTestWriteFile(t, record, []byte("corrupt"))
				report, err := (CacheManager{}).Clean(context.Background(), CacheCleanRequest{Roots: roots, Mode: mode}, nil)
				// The incomplete inventory accompanies a useful effect ledger.
				//goland:noinspection GoDfaErrorMayBeNotNil
				if err == nil || !report.Remaining.Incomplete {
					t.Fatal("unknown membership was reported as complete")
				}
				if mode == ClearAll {
					if report.RemovedRecords != 1 || report.Remaining.Favorite.Records != 0 {
						t.Fatalf("explicit unknown-owner clear: %+v", report)
					}
				} else if report.RemovedRecords != 0 || report.Remaining.Favorite.Records != 1 || report.Unavailable != 1 {
					t.Fatalf("stale cleanup treated unknown membership as empty: %+v", report)
				}
			})
		}
	})
	for _, mode := range []CacheCleanMode{ClearAll, RemoveStale} {
		for _, change := range []string{"identical_list", "moved", "replacement", "fresh_record", "fresh_directory"} {
			t.Run(fmt.Sprintf("%d/%s", mode, change), func(t *testing.T) {
				policy := cacheTestPolicy(t)
				item := cacheFixtureItem(t, "member.jpg")
				cacheTestFavorite(t, policy.Roots, item)
				dir := favstore.Dir(policy.Roots.FavoritesDir, "Trip")
				record := filepath.Join(dir, analysisName(item.Path))
				cacheTestWriteFile(t, record, []byte("old corrupt record"))
				marker := filepath.Join(policy.Roots.GeneralDir, "v1", strings.Repeat("a", 64)+".json")
				cacheTestWriteFile(t, marker, []byte("corrupt marker"))
				changed := false
				preserved := record
				report, err := (CacheManager{}).Clean(context.Background(), CacheCleanRequest{Roots: policy.Roots, Mode: mode}, func(progress CacheProgress) {
					if progress.Phase != "remove" || changed {
						return
					}
					changed = true
					switch change {
					case "identical_list":
						cacheTestFavorite(t, policy.Roots, item)
					case "moved", "replacement":
						moved := filepath.Join(t.TempDir(), "moved")
						if err := os.Rename(dir, moved); err != nil {
							t.Fatalf("idle maintenance pins Favorite: %v", err)
						}
						preserved = filepath.Join(moved, analysisName(item.Path))
						if change == "replacement" {
							cacheTestFavorite(t, policy.Roots, item)
							cacheTestWriteFile(t, record, []byte("replacement record"))
						}
					case "fresh_record":
						staged := filepath.Join(dir, "fresh.json")
						cacheTestWriteFile(t, staged, []byte("new corrupt record"))
						if err := os.Rename(staged, record); err != nil {
							t.Fatal(err)
						}
					case "fresh_directory":
						moved := filepath.Join(t.TempDir(), "old-analysis")
						if err := os.Rename(filepath.Join(dir, "analysis"), moved); err != nil {
							t.Fatalf("idle maintenance pins analysis directory: %v", err)
						}
						preserved = filepath.Join(moved, filepath.Base(record))
						cacheTestWriteFile(t, record, []byte("replacement record"))
					}
				})
				if err != nil {
					t.Fatal(err)
				}
				if !changed || report.RemovedRecords != 1 || report.Skipped != 1 || report.Unavailable != 1 {
					t.Fatalf("obsolete cleanup not retired: %+v", report)
				}
				if _, err := os.Stat(preserved); err != nil {
					t.Fatalf("old inventory removed changed owner's record: %v", err)
				}
				if change == "replacement" || change == "fresh_directory" {
					if _, err := os.Stat(record); err != nil {
						t.Fatalf("replacement record lost: %v", err)
					}
				}
			})
		}
	}
	t.Run("resources", func(t *testing.T) {
		for _, count := range []int{32, 130} {
			t.Run(fmt.Sprint(count), func(t *testing.T) {
				roots := CacheRoots{FavoritesDir: t.TempDir()}
				for i := range count {
					dir := favstore.Dir(roots.FavoritesDir, fmt.Sprintf("%03d", i))
					cacheTestWriteFile(t, filepath.Join(dir, "file-list.json"), []byte(`{"0":"/member.jpg"}`))
					cacheTestWriteFile(t, filepath.Join(dir, "analysis", strings.Repeat("a", 64)+".json"), []byte("record"))
				}
				inventory := &analysisInventory{}
				defer inventory.close()
				handles := func() int {
					entries, _ := os.ReadDir("/proc/self/fd")
					count := 0
					for _, entry := range entries {
						name, _ := os.Readlink(filepath.Join("/proc/self/fd", entry.Name()))
						if strings.HasPrefix(name, roots.FavoritesDir+string(filepath.Separator)) {
							count++
						}
					}
					return count
				}
				peak := 0
				runtime.GC()
				var beforeMemory, afterMemory runtime.MemStats
				runtime.ReadMemStats(&beforeMemory)
				if err := inventory.scan(context.Background(), roots, func(_ CacheProgress) { peak = max(peak, handles()) }); err != nil {
					t.Fatal(err)
				}
				if len(inventory.records) != count {
					t.Fatal("maintenance inventory truncated")
				}
				runtime.GC()
				runtime.ReadMemStats(&afterMemory)
				if idle := handles(); idle != 0 || peak > 3 {
					t.Fatalf("maintenance handles: idle=%d peak=%d", idle, peak)
				}
				for i := range count {
					if err := os.Rename(favstore.Dir(roots.FavoritesDir, fmt.Sprintf("%03d", i)), filepath.Join(roots.FavoritesDir, fmt.Sprintf("moved-%03d", i))); err != nil {
						t.Fatalf("idle maintenance prevents native move: %v", err)
					}
				}
				t.Logf("maintenance retained %d value records, zero idle Favorite handles, peak=%d; heap delta=%d bytes including runtime noise", count, peak, int64(afterMemory.HeapAlloc)-int64(beforeMemory.HeapAlloc))
				runtime.KeepAlive(inventory)
			})
		}
	})
}
