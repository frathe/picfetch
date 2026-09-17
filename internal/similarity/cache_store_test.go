package similarity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/uitest"
)

func cacheFixtureItem(t *testing.T, name string) Item {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	data := []byte("source")
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	vector := make([]float32, 768)
	vector[0] = 1
	return Item{Path: path, Size: info.Size(), ModifiedNS: info.ModTime().UnixNano(), SHA256: fmt.Sprintf("%x", sha256.Sum256(data)), Embedding: vector, Preview: uitest.EncodeJPEG(t, 32, 24, color.White), Facts: ImageFacts{Version: FactsVersion, Width: 32, Height: 24}}
}
func TestAnalysisCacheGeneralReopensWithoutPreparation(t *testing.T) {
	item := cacheFixtureItem(t, "a.jpg")
	policy := CachePolicy{Roots: CacheRoots{GeneralDir: t.TempDir()}, LooseEnabled: true, GeneralLimitBytes: 1024 * 1024}
	store, err := openRepresentationStore(context.Background(), policy, writeEnabledStores)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.write(context.Background(), item); err != nil {
		store.close()
		t.Fatal(err)
	}
	store.close()
	reopened, err := openRepresentationStore(context.Background(), policy, writeEnabledStores)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	got, ok := cacheTestRead(t, reopened, context.Background(), item)
	if !ok || got.Path != item.Path || len(got.Embedding) != 768 || got.Embedding[0] != 1 {
		t.Fatal("reopened general record was not reusable")
	}
}

func TestAnalysisCacheFileURIPathsReopen(t *testing.T) {
	for _, favorite := range []bool{false, true} {
		name := "general"
		if favorite {
			name = "favorite"
		}
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			item := cacheFixtureItem(t, "source image.jpg")
			// Explorer and search receive URI paths, which use forward slashes
			// on Windows even though filepath.Clean uses backslashes.
			item.Path = storage.NewFileURI(item.Path).Path()
			policy := cacheTestPolicy(t)
			if favorite {
				cacheTestFavorite(t, policy.Roots, item)
				policy.LooseEnabled = false
			}
			store := cacheTestStore(t, policy)
			if err := store.write(ctx, item); err != nil {
				t.Fatalf("persist UI source: %v", err)
			}
			store.close()
			reopened := cacheTestStore(t, policy)
			got, hit := cacheTestRead(t, reopened, ctx, item)
			if !hit || got.Path != item.Path || !slices.Equal(got.Embedding, item.Embedding) || !bytes.Equal(got.Preview, item.Preview) {
				t.Fatal("reopened UI source lost its cached representation or identity")
			}
			usage, err := (CacheManager{}).Inspect(ctx, policy.Roots, nil)
			if err != nil || usage.General.Records+usage.Favorite.Records != 1 {
				t.Fatalf("persisted UI source missing from cache usage: %+v, %v", usage, err)
			}
			report, err := (CacheManager{}).Clean(ctx, CacheCleanRequest{Roots: policy.Roots, Mode: RemoveStale}, nil)
			if err != nil || report.RemovedRecords != 0 {
				t.Fatalf("valid UI source treated as stale: %+v, %v", report, err)
			}
		})
	}
}

func TestAnalysisCachePayloadPathValidation(t *testing.T) {
	item := cacheFixtureItem(t, "source.jpg")
	dir, base := filepath.Dir(item.Path), filepath.Base(item.Path)
	for _, tc := range []struct {
		name, path string
		valid      bool
	}{
		{"native", item.Path, true},
		{"file_uri", storage.NewFileURI(item.Path).Path(), true},
		{"empty", "", false},
		{"relative", base, false},
		{"dot", dir + "/./" + base, false},
		{"parent", dir + "/child/../" + base, false},
		{"repeated_separator", dir + "//" + base, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			candidate := item
			candidate.Path = tc.path
			got, err := decodeRepresentation(bytes.NewReader(cacheTestPayload(t, candidate)))
			if (err == nil) != tc.valid {
				t.Fatalf("path %q: valid=%t, error=%v", tc.path, tc.valid, err)
			}
			if err == nil && got.Path != tc.path {
				t.Fatal("cache validation changed the source identity")
			}
		})
	}
}

func TestAnalysisCacheAccountingObservesOtherWriters(t *testing.T) {
	ctx := context.Background()
	item := cacheFixtureItem(t, "a.jpg")
	policy := cacheTestPolicy(t)
	policy.GeneralLimitBytes = 2 * uint64(len(cacheTestPayload(t, item)))
	first, second := cacheTestStore(t, policy), cacheTestStore(t, policy)
	if err := first.write(ctx, item); err != nil {
		t.Fatal(err)
	}
	item.Path = filepath.Join(filepath.Dir(item.Path), "b.jpg")
	if err := second.write(ctx, item); err != nil {
		t.Fatal(err)
	}
	item.Path = filepath.Join(filepath.Dir(item.Path), "c.jpg")
	var pressure CachePressureError
	if err := first.write(ctx, item); !errors.As(err, &pressure) {
		t.Fatalf("another writer's bytes were ignored: %v", err)
	}
}

func BenchmarkAnalysisCachePopulation(b *testing.B) {
	for _, count := range []int{100, 1000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			var preview bytes.Buffer
			if err := jpeg.Encode(&preview, image.NewRGBA(image.Rect(0, 0, 16, 16)), nil); err != nil {
				b.Fatal(err)
			}
			for range b.N {
				root := b.TempDir()
				store, err := openRepresentationStore(context.Background(), CachePolicy{Roots: CacheRoots{GeneralDir: root}, LooseEnabled: true}, writeEnabledStores)
				if err != nil {
					b.Fatal(err)
				}
				item := Item{Embedding: make([]float32, 768), SHA256: fmt.Sprintf("%064d", 0), Preview: preview.Bytes()}
				item.Embedding[0] = 1
				for i := range count {
					item.Path = filepath.Join(root, fmt.Sprintf("source-%04d.jpg", i))
					if err := store.write(context.Background(), item); err != nil {
						store.close()
						b.Fatal(err)
					}
				}
				store.close()
			}
		})
	}
}
