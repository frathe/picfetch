package similarity

import (
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

type warmSearchCorpus struct {
	policy CachePolicy
	paths  []string
	items  []Item
}

func (c warmSearchCorpus) matches(reference int) []Match {
	ref := c.items[reference]
	candidates := make([]Item, 0, len(c.items)-1)
	for _, item := range c.items {
		if item.Path != ref.Path {
			candidates = append(candidates, item)
		}
	}
	return searchRankOracle(ref, candidates, 30)
}

func newWarmSearchCorpus(t testing.TB, count int) warmSearchCorpus {
	t.Helper()
	var preview bytes.Buffer
	if err := jpeg.Encode(&preview, image.NewRGBA(image.Rect(0, 0, 32, 24)), nil); err != nil {
		t.Fatal(err)
	}
	data := preview.Bytes()
	_, items := searchRankCorpus(count)
	corpus := warmSearchCorpus{
		policy: CachePolicy{Roots: CacheRoots{GeneralDir: t.TempDir(), FavoritesDir: t.TempDir()}, LooseEnabled: true},
		paths:  make([]string, count),
		items:  items,
	}
	store, err := openRepresentationStore(context.Background(), corpus.policy, writeEnabledStores)
	if store != nil {
		defer store.close()
	}
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	for i := range corpus.items {
		item := &corpus.items[i]
		item.Path = filepath.Join(dir, fmt.Sprintf("%05d.jpg", i))
		if err := os.WriteFile(item.Path, data, 0600); err != nil {
			t.Fatal(err)
		}
		info, err := os.Stat(item.Path)
		if err != nil {
			t.Fatal(err)
		}
		item.Size, item.ModifiedNS = info.Size(), info.ModTime().UnixNano()
		item.SHA256 = fmt.Sprintf("%x", sha256.Sum256(data))
		item.Preview = data
		item.Facts = ImageFacts{Version: FactsVersion, Width: 32, Height: 24}
		if err := store.write(context.Background(), *item); err != nil {
			t.Fatal(err)
		}
		corpus.paths[i] = item.Path
	}
	return corpus
}

func TestSearchWarmPipeline(t *testing.T) {
	for _, changeSource := range []bool{false, true} {
		t.Run(fmt.Sprintf("changed-source=%v", changeSource), func(t *testing.T) {
			corpus := newWarmSearchCorpus(t, 201)
			ctx := context.Background()
			store := cacheTestStore(t, corpus.policy)
			p := searchPreparer{cache: store, versions: map[string]os.FileInfo{}}
			t.Cleanup(func() {
				if p.encoder != nil {
					p.encoder.Close()
				}
			})
			references := []int{200, 17, 42}
			queries := make(chan SearchQuery, 1)
			queries <- SearchQuery{ID: 1, ReferencePath: corpus.paths[references[0]]}
			var prepared []string
			var finals []uint64
			var removed string
			err := runSearchSession(ctx, SearchRequest{Paths: corpus.paths}, queries, func(ctx context.Context, path string) (Item, bool, error) {
				prepared = append(prepared, path)
				return p.prepare(ctx, path)
			}, p.validate, p.refreshFavorites, func(event SearchEvent) error {
				if event.Kind == SearchQueryFailure {
					return fmt.Errorf("warm reference preparation failed: %s", event.Error)
				}
				if event.Kind != SearchFinal {
					return nil
				}
				finals = append(finals, event.QueryID)
				if event.Processed != len(corpus.paths) || event.Reused != len(corpus.paths) || event.Failed != 0 {
					t.Fatalf("warm pipeline lost cached preparation: processed=%d reused=%d failed=%d", event.Processed, event.Reused, event.Failed)
				}
				if event.QueryID < 1 || event.QueryID > uint64(len(references)) {
					t.Fatalf("unexpected reference query: %d", event.QueryID)
				}
				want := corpus.matches(references[event.QueryID-1])
				assertSearchRankMatches(t, event.Matches, want)
				if event.QueryID == 2 && changeSource {
					next := corpus.items[references[2]]
					matches := corpus.matches(references[2])
					for _, path := range corpus.paths {
						if path != next.Path && !slices.ContainsFunc(matches, func(match Match) bool { return match.Path == path }) {
							removed = path
							break
						}
					}
					if err := os.Remove(removed); err != nil {
						t.Fatalf("could not change the unranked source: %v", err)
					}
				}
				if event.QueryID < uint64(len(references)) {
					queries <- SearchQuery{ID: event.QueryID + 1, ReferencePath: corpus.paths[references[event.QueryID]]}
				} else {
					close(queries)
				}
				return nil
			})
			wantFinals := []uint64{1, 2, 3}
			if changeSource {
				wantFinals = wantFinals[:2]
			}
			if (err != nil) != changeSource || !slices.Equal(finals, wantFinals) {
				t.Fatalf("source-validation sequence: final queries=%v error=%v", finals, err)
			}
			if changeSource && !strings.Contains(err.Error(), removed) {
				t.Fatalf("failure did not identify the changed unranked source: %v", err)
			}
			if len(prepared) != len(corpus.paths) || prepared[0] != corpus.paths[references[0]] || p.encoder != nil || p.warning != "" {
				t.Fatalf("warm references repeated preparation or opened inference: prepared=%d encoder=%v warning=%s", len(prepared), p.encoder != nil, p.warning)
			}
		})
	}
}

// BenchmarkSearchWarmPipeline includes reopening cache leases, representation
// reads, source validation, progressive ranking and publication. Initial cache
// population, model startup, subprocess transport and UI delivery are excluded.
func BenchmarkSearchWarmPipeline(b *testing.B) {
	for _, count := range []int{1000, 10000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			corpus := newWarmSearchCorpus(b, count)
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				ctx := context.Background()
				store, err := openRepresentationStore(ctx, corpus.policy, writeEnabledStores)
				if err != nil {
					store.close()
					b.Fatal(err)
				}
				p := searchPreparer{cache: store, versions: map[string]os.FileInfo{}}
				queries := make(chan SearchQuery, 1)
				queries <- SearchQuery{ID: 1, ReferencePath: corpus.paths[0]}
				close(queries)
				var final SearchEvent
				err = runSearchSession(ctx, SearchRequest{Paths: corpus.paths}, queries, p.prepare, p.validate, p.refreshFavorites, func(event SearchEvent) error {
					if event.Kind == SearchFinal {
						final = event
					}
					return nil
				})
				store.close()
				openedEncoder := p.encoder != nil
				if openedEncoder {
					p.encoder.Close()
				}
				if err != nil || final.Processed != count || final.Reused != count || final.Failed != 0 || openedEncoder || p.warning != "" {
					b.Fatalf("warm pipeline: error=%v processed=%d reused=%d failed=%d encoder=%v warning=%s", err, final.Processed, final.Reused, final.Failed, openedEncoder, p.warning)
				}
			}
		})
	}
}
