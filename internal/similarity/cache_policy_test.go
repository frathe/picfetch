package similarity

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"testing/iotest"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/favstore"
)

func TestAnalysisCachePolicyProducerScope(t *testing.T) {
	for _, scope := range []cacheWriteScope{writeEnabledStores, writeFavoritesOnly} {
		for _, favoriteEnabled := range []bool{false, true} {
			for _, looseEnabled := range []bool{false, true} {
				t.Run(fmt.Sprintf("scope_%d/favorites_%t/loose_%t", scope, favoriteEnabled, looseEnabled), func(t *testing.T) {
					policy := cacheTestPolicy(t)
					member := cacheFixtureItem(t, "new-favorite.jpg")
					retained := cacheFixtureItem(t, "retained-loose.jpg")
					fresh := cacheFixtureItem(t, "fresh-loose.jpg")
					seed := cacheTestStore(t, policy)
					for _, item := range []Item{member, retained} {
						if err := seed.write(context.Background(), item); err != nil {
							t.Fatal(err)
						}
					}
					cacheTestFavorite(t, policy.Roots, member)
					policy.FavoriteEnabled, policy.LooseEnabled = favoriteEnabled, looseEnabled
					store, err := openRepresentationStore(context.Background(), policy, scope)
					if err != nil {
						t.Fatal(err)
					}
					t.Cleanup(store.close)
					if _, hit := cacheTestRead(t, store, context.Background(), retained); hit != looseEnabled {
						t.Fatalf("producer read scope changed: hit=%t", hit)
					}
					if _, hit := cacheTestRead(t, store, context.Background(), member); hit != (favoriteEnabled && looseEnabled) {
						t.Fatalf("Favorite promotion/opt-out: hit=%t", hit)
					}
					for _, item := range []Item{member, fresh} {
						if err := store.write(context.Background(), item); err != nil {
							t.Fatal(err)
						}
					}
					if _, err := os.Stat(cacheTestGeneralPath(policy.Roots, fresh)); (err == nil) != (looseEnabled && scope == writeEnabledStores) {
						t.Fatalf("producer persisted a loose miss outside its write scope: %v", err)
					}
					usage, err := (CacheManager{}).Inspect(context.Background(), policy.Roots, nil)
					if err != nil || (usage.Favorite.Records == 1) != favoriteEnabled {
						t.Fatalf("producer Favorite effects: %+v, %v", usage, err)
					}
				})
			}
		}
	}
}

func TestAnalysisCachePolicyExplicitFavoriteSaveReusesPreparedItems(t *testing.T) {
	policy := cacheTestPolicy(t)
	policy.LooseEnabled = false
	prepared := cacheFixtureItem(t, "already-prepared.jpg")
	later := cacheFixtureItem(t, "not-yet-prepared.jpg")
	cached := cacheFixtureItem(t, "already-cached.jpg")
	if err := favstore.Save(policy.Roots.FavoritesDir, "Existing", []fyne.URI{storage.NewFileURI(cached.Path)}); err != nil {
		t.Fatal(err)
	}
	store := cacheTestStore(t, policy)
	if err := store.write(context.Background(), cached); err != nil {
		t.Fatal(err)
	}
	existingPath := filepath.Join(favstore.Dir(policy.Roots.FavoritesDir, "Existing"), analysisName(cached.Path))
	existing, err := os.Stat(existingPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.write(context.Background(), prepared); err != nil {
		t.Fatal(err)
	}
	if err := favstore.Save(policy.Roots.FavoritesDir, "Trip", []fyne.URI{storage.NewFileURI(prepared.Path), storage.NewFileURI(later.Path), storage.NewFileURI(cached.Path)}); err != nil {
		t.Fatal(err)
	}
	retained := prepared
	retained.Preview = nil
	completed := 0
	if err := store.refreshFavorites(context.Background(), []Item{retained, cached}, func(_ context.Context, item Item) (Item, error) {
		completed++
		if !slices.Equal(item.Embedding, prepared.Embedding) {
			t.Fatal("prepared vector was lost")
		}
		item.Preview = prepared.Preview
		return item, nil
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.write(context.Background(), later); err != nil {
		t.Fatal(err)
	}
	if completed != 1 {
		t.Fatalf("prepared record completions: %d", completed)
	}
	after, err := os.Stat(existingPath)
	if err != nil || !sameVersion(existing, after) {
		t.Fatal("saving one Favorite rewrote an unrelated compatible record")
	}
	reopened := cacheTestStore(t, policy)
	for _, item := range []Item{prepared, later, cached} {
		if _, hit := cacheTestRead(t, reopened, context.Background(), item); !hit {
			t.Fatalf("new Favorite missed %s", item.Path)
		}
	}
	usage, err := (CacheManager{}).Inspect(context.Background(), policy.Roots, nil)
	if err != nil || usage.General.Records != 0 || usage.Favorite.Records != 4 {
		t.Fatalf("save effects: %+v, %v", usage, err)
	}
}

func TestAnalysisCachePolicyWriteBudget(t *testing.T) {
	item := cacheFixtureItem(t, "source.jpg")
	data := cacheTestPayload(t, item)
	t.Run("oversized", func(t *testing.T) {
		policy := cacheTestPolicy(t)
		policy.GeneralLimitBytes = uint64(len(data) - 1)
		store := cacheTestStore(t, policy)
		if err := store.write(context.Background(), item); err != nil {
			t.Fatal(err)
		}
		if _, ok := cacheTestRead(t, store, context.Background(), item); ok {
			t.Fatal("record larger than the whole budget was persisted")
		}
	})
	t.Run("replacement_pressure_preserves_old_record", func(t *testing.T) {
		policy := cacheTestPolicy(t)
		policy.GeneralLimitBytes = uint64(len(data))
		store := cacheTestStore(t, policy)
		if err := store.write(context.Background(), item); err != nil {
			t.Fatal(err)
		}
		var pressure CachePressureError
		if err := store.write(context.Background(), item); !errors.As(err, &pressure) {
			t.Fatalf("replacement staging did not respect the byte budget: %v", err)
		}
		usage, err := (CacheManager{}).Inspect(context.Background(), policy.Roots, nil)
		if err != nil || usage.General.Bytes != uint64(len(data)) || usage.General.Records != 1 {
			t.Fatalf("replacement usage: %+v, %v", usage, err)
		}
		other := item
		other.Path = filepath.Join(filepath.Dir(item.Path), "second.jpg")
		if err := store.write(context.Background(), other); err != nil {
			t.Fatalf("producer retried general persistence after pressure: %v", err)
		}
		if _, ok := cacheTestRead(t, store, context.Background(), item); !ok {
			t.Fatal("writer silently evicted an existing record under pressure")
		}
	})
}

func TestAnalysisCachePolicyPressurePreservesFavorites(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		t.Run(fmt.Sprintf("favorites_%t", enabled), func(t *testing.T) {
			ctx := context.Background()
			policy := cacheTestPolicy(t)
			policy.FavoriteEnabled = enabled
			retained := cacheFixtureItem(t, "retained.jpg")
			blocked := cacheFixtureItem(t, "blocked.jpg")
			member := cacheFixtureItem(t, "member.jpg")
			laterMember := cacheFixtureItem(t, "later-member.jpg")
			loose := cacheFixtureItem(t, "loose.jpg")
			policy.GeneralLimitBytes = 0
			for _, item := range []Item{retained, blocked, member, laterMember, loose} {
				policy.GeneralLimitBytes = max(policy.GeneralLimitBytes, uint64(len(cacheTestPayload(t, item))))
			}
			cacheTestFavorite(t, policy.Roots, member)
			store := cacheTestStore(t, policy)
			if err := store.write(ctx, retained); err != nil {
				t.Fatal(err)
			}
			var pressure CachePressureError
			if err := store.write(ctx, blocked); !errors.As(err, &pressure) || pressure.NeedBytes == 0 {
				t.Fatalf("first capacity refusal: %v", err)
			}
			if _, hit := cacheTestRead(t, store, ctx, retained); !hit {
				t.Fatal("capacity pressure disabled general reads")
			}
			if err := store.write(ctx, blocked); err != nil {
				t.Fatalf("later general write repeated pressure: %v", err)
			}
			if err := store.write(ctx, member); err != nil {
				t.Fatal(err)
			}
			if _, hit := cacheTestRead(t, store, ctx, member); hit != enabled {
				t.Fatal("capacity pressure changed existing Favorite persistence")
			}
			// Space becoming available does not re-admit this producer's writes.
			if err := os.Remove(cacheTestGeneralPath(policy.Roots, retained)); err != nil {
				t.Fatal(err)
			}
			if err := store.write(ctx, blocked); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(cacheTestGeneralPath(policy.Roots, blocked)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("general persistence resumed after space was freed: %v", err)
			}
			// Explicit save admits Favorite persistence, including future members.
			cacheTestFavorite(t, policy.Roots, member, blocked, laterMember)
			if err := store.refreshFavorites(ctx, []Item{blocked}, func(_ context.Context, item Item) (Item, error) {
				return item, nil
			}); err != nil {
				t.Fatal(err)
			}
			if err := store.write(ctx, laterMember); err != nil {
				t.Fatal(err)
			}
			for _, item := range []Item{blocked, laterMember} {
				if _, hit := cacheTestRead(t, store, ctx, item); hit != enabled {
					t.Fatalf("refreshed Favorite persistence changed for %s", item.Path)
				}
			}
			if err := store.write(ctx, loose); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(cacheTestGeneralPath(policy.Roots, loose)); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("Favorite refresh reopened general writes: %v", err)
			}
			fresh := cacheTestStore(t, policy)
			if err := fresh.write(ctx, loose); err != nil {
				t.Fatal(err)
			}
			if _, hit := cacheTestRead(t, fresh, ctx, loose); !hit {
				t.Fatal("fresh producer could not persist after space was freed")
			}
		})
	}
}

func TestAnalysisCachePolicyFavoriteFirstAndPromotion(t *testing.T) {
	for _, existingFavorite := range []bool{false, true} {
		name := "promote_general"
		if existingFavorite {
			name = "prefer_favorite"
		}
		t.Run(name, func(t *testing.T) {
			policy := cacheTestPolicy(t)
			item := cacheFixtureItem(t, "source.jpg")
			general := cacheTestStore(t, policy)
			if err := general.write(context.Background(), item); err != nil {
				t.Fatal(err)
			}
			cacheTestFavorite(t, policy.Roots, item)
			store := cacheTestStore(t, policy)
			want := item.Embedding
			if existingFavorite {
				favorite := item
				favorite.Embedding = make([]float32, 768)
				favorite.Embedding[1] = 1
				if err := store.write(context.Background(), favorite); err != nil {
					t.Fatal(err)
				}
				want = favorite.Embedding
			}
			got, ok := cacheTestRead(t, store, context.Background(), item)
			if !ok || !slices.Equal(got.Embedding, want) {
				t.Fatal("Favorite priority or general fallback returned the wrong representation")
			}
			policy.LooseEnabled = false
			onlyFavorite := cacheTestStore(t, policy)
			got, ok = cacheTestRead(t, onlyFavorite, context.Background(), item)
			if !ok || !slices.Equal(got.Embedding, want) {
				t.Fatal("Favorite-only reopen could not reuse its existing or promoted representation")
			}
			usage, err := (CacheManager{}).Inspect(context.Background(), policy.Roots, nil)
			if err != nil || usage.General.Records != 1 || usage.Favorite.Records != 1 {
				t.Fatalf("promotion/priority inventory: %+v, %v", usage, err)
			}
		})
	}
}

func TestAnalysisCachePolicyPromotionWarnings(t *testing.T) {
	for _, scope := range []cacheWriteScope{writeEnabledStores, writeFavoritesOnly} {
		for _, failure := range []string{"none", "blocked_directory", "retired_lease"} {
			t.Run(fmt.Sprintf("scope_%d/%s", scope, failure), func(t *testing.T) {
				ctx := context.Background()
				policy := cacheTestPolicy(t)
				item := cacheFixtureItem(t, "source.jpg")
				seed := cacheTestStore(t, policy)
				if err := seed.write(ctx, item); err != nil {
					t.Fatal(err)
				}
				cacheTestFavorite(t, policy.Roots, item)
				store, err := openRepresentationStore(ctx, policy, scope)
				if err != nil {
					t.Fatal(err)
				}
				t.Cleanup(store.close)
				switch failure {
				case "blocked_directory":
					if err := os.WriteFile(filepath.Join(favstore.Dir(policy.Roots.FavoritesDir, "Trip"), "analysis"), []byte("blocked"), 0600); err != nil {
						t.Fatal(err)
					}
				case "retired_lease":
					if _, err := (CacheManager{}).Clean(ctx, CacheCleanRequest{Roots: CacheRoots{FavoritesDir: policy.Roots.FavoritesDir}, Mode: ClearAll}, nil); err != nil {
						t.Fatal(err)
					}
				}
				preparer := searchPreparer{cache: store, versions: map[string]os.FileInfo{}}
				got, reused, err := preparer.prepare(ctx, item.Path)
				if err != nil || !reused || !slices.Equal(got.Embedding, item.Embedding) || preparer.encoder != nil {
					t.Fatalf("promotion failure lost usable analysis or repeated inference: reused=%t, err=%v", reused, err)
				}
				if (preparer.warning != "") != (failure != "none") {
					t.Fatalf("promotion warning: %q, failure=%s", preparer.warning, failure)
				}
				if _, hit := store.favorites.read(item); hit != (failure == "none") {
					t.Fatalf("unexpected Favorite persistence: hit=%t, failure=%s", hit, failure)
				}
			})
		}
	}
}

func TestAnalysisCachePolicyDisabledStores(t *testing.T) {
	for _, favoriteEnabled := range []bool{false, true} {
		for _, looseEnabled := range []bool{false, true} {
			name := "favorite_off"
			if favoriteEnabled {
				name = "favorite_on"
			}
			if looseEnabled {
				name += "/loose_on"
			} else {
				name += "/loose_off"
			}
			t.Run(name, func(t *testing.T) {
				policy := cacheTestPolicy(t)
				policy.FavoriteEnabled, policy.LooseEnabled = favoriteEnabled, looseEnabled
				member := cacheFixtureItem(t, "member.jpg")
				loose := cacheFixtureItem(t, "loose.jpg")
				cacheTestFavorite(t, policy.Roots, member)
				store := cacheTestStore(t, policy)
				for _, item := range []Item{member, loose} {
					if err := store.write(context.Background(), item); err != nil {
						t.Fatal(err)
					}
				}
				if _, ok := cacheTestRead(t, store, context.Background(), member); ok != favoriteEnabled {
					t.Fatalf("Favorite cache preference ignored: hit=%v", ok)
				}
				if _, ok := cacheTestRead(t, store, context.Background(), loose); ok != looseEnabled {
					t.Fatalf("loose cache preference ignored: hit=%v", ok)
				}
				if _, err := os.Stat(cacheTestGeneralPath(policy.Roots, member)); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("Favorite analysis was redirected or duplicated into general storage: %v", err)
				}
				// Retained disk usage is independent of subsequent read/write opt-outs.
				policy.FavoriteEnabled, policy.LooseEnabled = false, false
				disabled := cacheTestStore(t, policy)
				for _, item := range []Item{member, loose} {
					if _, ok := cacheTestRead(t, disabled, context.Background(), item); ok {
						t.Fatal("disabled store reused a retained record")
					}
				}
				usage, err := (CacheManager{}).Inspect(context.Background(), policy.Roots, nil)
				if err != nil || (usage.General.Records == 1) != looseEnabled || (usage.Favorite.Records == 1) != favoriteEnabled {
					t.Fatalf("disabled-store inventory: %+v, %v", usage, err)
				}
			})
		}
	}
}

func TestAnalysisCachePolicyFavoriteOptOutRejectsRetainedGeneralRecord(t *testing.T) {
	t.Run("active_save_refreshes_opt_out_without_persisting", func(t *testing.T) {
		policy := cacheTestPolicy(t)
		policy.FavoriteEnabled = false
		item := cacheFixtureItem(t, "new-member.jpg")
		store := cacheTestStore(t, policy)
		if err := store.write(context.Background(), item); err != nil {
			t.Fatal(err)
		}
		cacheTestFavorite(t, policy.Roots, item)
		if err := store.refreshFavorites(context.Background(), []Item{item}, func(_ context.Context, _ Item) (Item, error) {
			t.Fatal("Favorite opt-out admitted preview completion")
			return Item{}, nil
		}); err != nil {
			t.Fatal(err)
		}
		if _, hit := cacheTestRead(t, store, context.Background(), item); hit {
			t.Fatal("refreshed Favorite opt-out reused a general record")
		}
		if err := store.write(context.Background(), item); err != nil {
			t.Fatal(err)
		}
		usage, err := (CacheManager{}).Inspect(context.Background(), policy.Roots, nil)
		if err != nil || usage.General.Records != 1 || usage.Favorite.Records != 0 {
			t.Fatalf("opt-out changed retained bytes: %+v, %v", usage, err)
		}
	})
	policy := cacheTestPolicy(t)
	item := cacheFixtureItem(t, "former-loose.jpg")
	general := cacheTestStore(t, policy)
	if err := general.write(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	cacheTestFavorite(t, policy.Roots, item)
	policy.FavoriteEnabled = false
	disabled := cacheTestStore(t, policy)
	if _, ok := cacheTestRead(t, disabled, context.Background(), item); ok {
		t.Fatal("Favorite opt-out was bypassed through its retained general record")
	}
	if err := disabled.write(context.Background(), item); err != nil {
		t.Fatal(err)
	}
	usage, err := (CacheManager{}).Inspect(context.Background(), policy.Roots, nil)
	if err != nil || usage.General.Records != 1 || usage.Favorite.Records != 0 {
		t.Fatalf("opt-out should retain existing bytes without promotion: %+v, %v", usage, err)
	}
	policy.FavoriteEnabled = true
	enabled := cacheTestStore(t, policy)
	if _, ok := cacheTestRead(t, enabled, context.Background(), item); !ok {
		t.Fatal("re-enabled Favorite could not reuse the retained general record")
	}
}

func TestAnalysisCachePolicyIncompleteMembershipPreservesHealthyFavorites(t *testing.T) {
	t.Run("lease_failure_keeps_reads_but_blocks_writes", func(t *testing.T) {
		policy := cacheTestPolicy(t)
		item := cacheFixtureItem(t, "healthy.jpg")
		cacheTestFavorite(t, policy.Roots, item)
		seed := cacheTestStore(t, policy)
		if err := seed.write(context.Background(), item); err != nil {
			t.Fatal(err)
		}
		seed.close()
		lock := filepath.Join(policy.Roots.FavoritesDir, ".analysis-lock")
		if err := os.Remove(lock); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(lock, 0700); err != nil {
			t.Fatal(err)
		}
		store, err := openRepresentationStore(context.Background(), policy, writeEnabledStores)
		if store != nil {
			t.Cleanup(store.close)
		}
		if err == nil || store == nil {
			t.Fatal("broken lease did not report partial admission")
		}
		if _, hit := cacheTestRead(t, store, context.Background(), item); !hit {
			t.Fatal("lease failure discarded readable Favorite")
		}
		item.Embedding = make([]float32, 768)
		item.Embedding[1] = 1
		if err := store.write(context.Background(), item); !errors.Is(err, ErrCacheRetired) {
			t.Fatalf("write without a lease: %v", err)
		}
		got, hit := cacheTestRead(t, store, context.Background(), item)
		if !hit || got.Embedding[1] != 0 {
			t.Fatal("unleased producer changed the record")
		}
	})
	for _, corrupt := range []bool{false, true} {
		name := "unreadable"
		if corrupt {
			name = "corrupt"
		}
		t.Run(name, func(t *testing.T) {
			if !corrupt && runtime.GOOS == "windows" {
				t.Skip("creating the unreadable symlink fixture requires Unix")
			}
			policy := cacheTestPolicy(t)
			healthy := cacheFixtureItem(t, "healthy.jpg")
			unknown := cacheFixtureItem(t, "unknown-membership.jpg")
			cacheTestFavorite(t, policy.Roots, healthy)
			original := cacheTestStore(t, policy)
			if err := original.write(context.Background(), healthy); err != nil {
				t.Fatal(err)
			}
			if err := original.write(context.Background(), unknown); err != nil {
				t.Fatal(err)
			}
			blocked := filepath.Join(policy.Roots.FavoritesDir, "Blocked")
			if err := os.Mkdir(blocked, 0700); err != nil {
				t.Fatal(err)
			}
			if corrupt {
				cacheTestWriteFile(t, filepath.Join(blocked, "file-list.json"), []byte("invalid JSON"))
			} else if err := os.Symlink("file-list.json", filepath.Join(blocked, "file-list.json")); err != nil {
				t.Fatal(err)
			}
			partial, err := openRepresentationStore(context.Background(), policy, writeEnabledStores)
			if partial != nil {
				t.Cleanup(partial.close)
			}
			if err == nil || partial == nil {
				t.Fatalf("incomplete membership was not reported: %v", err)
			}
			if _, ok := cacheTestRead(t, partial, context.Background(), healthy); !ok {
				t.Fatal("unreadable Favorite prevented healthy Favorite reuse")
			}
			if _, ok := cacheTestRead(t, partial, context.Background(), unknown); ok {
				t.Fatal("incomplete membership admitted unknown general reuse")
			}
			for _, item := range []Item{healthy, cacheFixtureItem(t, "new-unknown.jpg")} {
				if err := partial.write(context.Background(), item); err != nil {
					t.Fatal(err)
				}
				if _, err := os.Stat(cacheTestGeneralPath(policy.Roots, item)); !errors.Is(err, os.ErrNotExist) {
					t.Fatalf("incomplete membership redirected a write to general storage: %v", err)
				}
			}
		})
	}
}

func TestAnalysisCachePayloadBounds(t *testing.T) {
	item := cacheFixtureItem(t, "source.jpg")
	valid := cacheTestPayload(t, item)
	for _, suffix := range []string{"", " \n\t", "{}", "null", "junk"} {
		t.Run(fmt.Sprintf("suffix_%q", suffix), func(t *testing.T) {
			payload := append(bytes.Clone(valid), suffix...)
			got, err := decodeRepresentation(bytes.NewReader(payload))
			wantValid := strings.TrimSpace(suffix) == ""
			if (err == nil) != wantValid || (wantValid && got.Path != item.Path) {
				t.Fatalf("single record with suffix %q: path=%q error=%v", suffix, got.Path, err)
			}
		})
	}
	for _, extra := range []int{0, 1, 1024} {
		t.Run(fmt.Sprintf("limit_plus_%d", extra), func(t *testing.T) {
			payload := append(bytes.Clone(valid), bytes.Repeat([]byte(" "), maximumAnalysisRecordBytes-len(valid)+extra)...)
			reader := bytes.NewReader(payload)
			got, err := decodeRepresentation(reader)
			if (err == nil) != (extra == 0) || (extra == 0 && got.Path != item.Path) {
				t.Fatalf("record limit plus %d: path=%q error=%v", extra, got.Path, err)
			}
			if read := len(payload) - reader.Len(); read > maximumAnalysisRecordBytes+1 {
				t.Fatalf("oversize detection read %d bytes", read)
			}
		})
	}
	for _, prefix := range [][]byte{nil, valid} {
		t.Run(fmt.Sprintf("reader_error_after_%d_bytes", len(prefix)), func(t *testing.T) {
			reader := io.MultiReader(bytes.NewReader(prefix), iotest.ErrReader(errors.New("cache read failed")))
			if _, err := decodeRepresentation(reader); err == nil {
				t.Fatal("reader failure was accepted as a complete record")
			}
		})
	}
}

func TestAnalysisCachePolicyRejectsInvalidRecords(t *testing.T) {
	for _, favorite := range []bool{false, true} {
		name := "general"
		if favorite {
			name = "favorite"
		}
		t.Run(name, func(t *testing.T) {
			policy := cacheTestPolicy(t)
			item := cacheFixtureItem(t, "source.jpg")
			if favorite {
				cacheTestFavorite(t, policy.Roots, item)
				policy.LooseEnabled = false
			}
			store := cacheTestStore(t, policy)
			path := cacheTestGeneralPath(policy.Roots, item)
			if favorite {
				path = filepath.Join(favstore.Dir(policy.Roots.FavoritesDir, "Trip"), analysisName(item.Path))
			}
			for _, kind := range []string{"path", "size", "mtime", "version", "vector_length", "zero_vector", "digest", "preview", "failed", "oversized", "trailing"} {
				t.Run(kind, func(t *testing.T) {
					bad := item
					switch kind {
					case "path":
						bad.Path += "-another"
					case "size":
						bad.Size++
					case "mtime":
						bad.ModifiedNS++
					case "vector_length":
						bad.Embedding = []float32{1}
					case "zero_vector":
						bad.Embedding = make([]float32, 768)
					case "digest":
						bad.SHA256 = "not a digest"
					case "preview":
						bad.Preview = []byte("not a JPEG")
					case "failed":
						bad.Error = "failed source"
					}
					data := cacheTestPayload(t, bad)
					switch kind {
					case "version":
						data = []byte(strings.Replace(string(data), RepresentationVersion, "different-model", 1))
					case "oversized":
						data = append(data, []byte(strings.Repeat(" ", 1024*1024))...)
					case "trailing":
						data = append(data, []byte("{}")...)
					}
					cacheTestWriteFile(t, path, data)
					if _, ok := cacheTestRead(t, store, context.Background(), item); ok {
						t.Fatal("invalid record was reused")
					}
					cacheTestWriteFile(t, path, cacheTestPayload(t, item))
					if _, ok := cacheTestRead(t, store, context.Background(), item); !ok {
						t.Fatal("valid record was not reusable after the invalid record was replaced")
					}
				})
			}
		})
	}
}

func TestAnalysisCacheLimitRetuneUsesGeneralLRU(t *testing.T) {
	policy := cacheTestPolicy(t)
	favorite := cacheFixtureItem(t, "favorite.jpg")
	cacheTestFavorite(t, policy.Roots, favorite)
	store := cacheTestStore(t, policy)
	if err := store.write(context.Background(), favorite); err != nil {
		t.Fatal(err)
	}
	items := []Item{cacheFixtureItem(t, "first.jpg"), cacheFixtureItem(t, "second.jpg"), cacheFixtureItem(t, "third.jpg")}
	var secondBytes uint64
	for i, item := range items {
		if err := store.write(context.Background(), item); err != nil {
			t.Fatal(err)
		}
		path := cacheTestGeneralPath(policy.Roots, item)
		when := time.Date(2020, 1, 1, 0, 0, i, 0, time.UTC)
		if err := os.Chtimes(path, when, when); err != nil {
			t.Fatal(err)
		}
		if i == 1 {
			info, err := os.Stat(path)
			if err != nil {
				t.Fatal(err)
			}
			secondBytes = uint64(info.Size())
		}
	}
	if _, ok := cacheTestRead(t, store, context.Background(), items[0]); !ok {
		t.Fatal("could not touch the oldest entry through a cache read")
	}
	quiesced := 0
	manager := CacheManager{Quiesce: func(_ context.Context, _ CacheRoots) error { quiesced++; return nil }}
	before, err := manager.Inspect(context.Background(), policy.Roots, nil)
	if err != nil || quiesced != 0 {
		t.Fatalf("inspection retired producers: %+v, %v", before, err)
	}
	increase, err := manager.Retune(context.Background(), CacheRetuneRequest{Roots: policy.Roots, LimitBytes: before.General.Bytes + 1}, nil)
	if err != nil || quiesced != 0 || increase.RemovedRecords != 0 {
		t.Fatalf("increased limit retired producers or evicted records: %+v, %v", increase, err)
	}
	limit := before.General.Bytes - secondBytes
	report, err := manager.Retune(context.Background(), CacheRetuneRequest{Roots: policy.Roots, LimitBytes: limit}, nil)
	if err != nil || quiesced != 1 || report.AppliedLimit != limit || report.RemovedRecords != 1 || report.RemovedBytes != secondBytes || report.Remaining.General.Bytes != limit || report.Remaining.Favorite != before.Favorite {
		t.Fatalf("LRU retune: %+v, quiesced %d, %v", report, quiesced, err)
	}
	fresh := cacheTestStore(t, policy)
	for i, item := range items {
		if _, ok := cacheTestRead(t, fresh, context.Background(), item); ok != (i != 1) {
			t.Fatalf("LRU retained wrong item %d: hit=%v", i, ok)
		}
	}
	if _, ok := cacheTestRead(t, fresh, context.Background(), favorite); !ok {
		t.Fatal("general retune evicted Favorite analysis")
	}
}

func TestAnalysisCacheLimitRetuneReclaimsTemporariesFirst(t *testing.T) {
	for _, evictRecord := range []bool{false, true} {
		t.Run(fmt.Sprintf("evict_record_%t", evictRecord), func(t *testing.T) {
			ctx := context.Background()
			policy := cacheTestPolicy(t)
			store := cacheTestStore(t, policy)
			items := []Item{cacheFixtureItem(t, "older.jpg"), cacheFixtureItem(t, "newer.jpg")}
			var recordBytes uint64
			for i, item := range items {
				if err := store.write(ctx, item); err != nil {
					t.Fatal(err)
				}
				recordBytes += uint64(len(cacheTestPayload(t, item)))
				when := time.Date(2020, 1, 1, 0, 0, i, 0, time.UTC)
				if err := os.Chtimes(cacheTestGeneralPath(policy.Roots, item), when, when); err != nil {
					t.Fatal(err)
				}
			}
			orphan := filepath.Join(policy.Roots.GeneralDir, "v1", "."+strings.Repeat("x", 20)+".tmp")
			if err := os.WriteFile(orphan, []byte("unfinished record"), 0600); err != nil {
				t.Fatal(err)
			}
			limit, removedRecords := recordBytes, 0
			if evictRecord {
				limit -= uint64(len(cacheTestPayload(t, items[0])))
				removedRecords = 1
			}
			report, err := (CacheManager{}).Retune(ctx, CacheRetuneRequest{Roots: policy.Roots, LimitBytes: limit}, nil)
			if err != nil || report.AppliedLimit != limit || report.Remaining.General.Bytes != limit || report.Remaining.General.Records != 2-removedRecords || report.RemovedRecords != removedRecords || report.RemovedBytes != recordBytes+uint64(len("unfinished record"))-limit {
				t.Fatalf("temporary-first retune: %+v, %v", report, err)
			}
			if _, err := os.Stat(orphan); !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("unusable temporary survived eviction: %v", err)
			}
			fresh := cacheTestStore(t, policy)
			for i, item := range items {
				if _, hit := cacheTestRead(t, fresh, ctx, item); hit != (!evictRecord || i == 1) {
					t.Fatalf("temporary displaced reusable record %d: hit=%t", i, hit)
				}
			}
		})
	}
}

func cacheTestRead(t *testing.T, store *representationStore, ctx context.Context, source Item) (Item, bool) {
	t.Helper()
	item, hit, err := store.read(ctx, source)
	if err != nil {
		t.Fatal(err)
	}
	return item, hit
}

func cacheTestPayload(t *testing.T, item Item) []byte {
	t.Helper()
	data, err := json.Marshal(cachedRepresentation{Version: RepresentationVersion, Item: item})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func cacheTestGeneralPath(roots CacheRoots, item Item) string {
	return filepath.Join(roots.GeneralDir, "v1", filepath.Base(analysisName(item.Path)))
}

func cacheTestPolicy(t *testing.T) CachePolicy {
	t.Helper()
	return CachePolicy{
		Roots:           CacheRoots{GeneralDir: t.TempDir(), FavoritesDir: t.TempDir()},
		FavoriteEnabled: true, LooseEnabled: true, GeneralLimitBytes: 1024 * 1024,
	}
}

func cacheTestStore(t *testing.T, policy CachePolicy) *representationStore {
	t.Helper()
	store, err := openRepresentationStore(context.Background(), policy, writeEnabledStores)
	if store != nil {
		t.Cleanup(store.close)
	}
	if err != nil {
		t.Fatal(err)
	}
	return store
}

func cacheTestFavorite(t *testing.T, roots CacheRoots, items ...Item) {
	t.Helper()
	files := make([]fyne.URI, len(items))
	for i, item := range items {
		files[i] = storage.NewFileURI(item.Path)
	}
	if err := favstore.Save(roots.FavoritesDir, "Trip", files); err != nil {
		t.Fatal(err)
	}
}
