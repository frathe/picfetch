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
	"testing"

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
	store, err := openRepresentationStore(context.Background(), policy)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.write(context.Background(), item); err != nil {
		store.close()
		t.Fatal(err)
	}
	store.close()
	reopened, err := openRepresentationStore(context.Background(), policy)
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.close()
	got, ok := reopened.read(context.Background(), item)
	if !ok || got.Path != item.Path || len(got.Embedding) != 768 || got.Embedding[0] != 1 {
		t.Fatal("reopened general record was not reusable")
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
				store, err := openRepresentationStore(context.Background(), CachePolicy{Roots: CacheRoots{GeneralDir: root}, LooseEnabled: true})
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
