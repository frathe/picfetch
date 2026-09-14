package similarity

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSearchSessionInitialReferenceFirstAndFinal(t *testing.T) {
	dir := t.TempDir()
	a, b := filepath.Join(dir, "a.jpg"), filepath.Join(dir, "b.jpg")
	queries := make(chan SearchQuery, 1)
	queries <- SearchQuery{ID: 1, ReferencePath: b}
	close(queries)
	var prepared []string
	var events []SearchEvent
	err := runSearchSession(context.Background(), SearchRequest{SessionID: 7, Paths: []string{a, b, a}, Limit: 30}, queries,
		func(_ context.Context, path string) (Item, bool, error) {
			prepared = append(prepared, path)
			vector := make([]float32, 768)
			vector[0] = 1
			return Item{Path: path, Embedding: vector}, false, nil
		}, nil, func(e SearchEvent) error { events = append(events, e); return nil })
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(prepared, []string{b, a}) {
		t.Fatalf("reference-first distinct preparation: %v", prepared)
	}
	var finals []SearchEvent
	for _, e := range events {
		if e.Kind == SearchFinal {
			finals = append(finals, e)
		}
	}
	if len(finals) != 1 || finals[0].Processed != 2 || finals[0].Total != 2 || len(finals[0].Matches) != 1 || finals[0].Matches[0].Path != a {
		t.Fatalf("final result: %+v", finals)
	}
}

func TestSearchSessionPublicationBoundaries(t *testing.T) {
	for _, n := range []int{99, 100, 101, 200} {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			dir := t.TempDir()
			paths := make([]string, n)
			for i := range paths {
				paths[i] = filepath.Join(dir, fmt.Sprintf("%03d.jpg", i))
			}
			queries := make(chan SearchQuery, 1)
			queries <- SearchQuery{ID: 1, ReferencePath: paths[0]}
			close(queries)
			var publications []int
			var kinds []SearchKind
			err := runSearchSession(context.Background(), SearchRequest{Paths: paths, Limit: 30}, queries, func(_ context.Context, path string) (Item, bool, error) {
				vector := make([]float32, 768)
				vector[0] = 1
				return Item{Path: path, Embedding: vector}, true, nil
			}, nil, func(e SearchEvent) error {
				if e.Kind == SearchPartial || e.Kind == SearchFinal {
					publications = append(publications, e.Processed)
					kinds = append(kinds, e.Kind)
				}
				return nil
			})
			want := []int{n}
			if n > 100 {
				want = []int{100, n}
			}
			if err != nil || !slices.Equal(publications, want) || len(kinds) == 0 || kinds[len(kinds)-1] != SearchFinal {
				t.Fatalf("publications %v %v, want %v: %v", publications, kinds, want, err)
			}
		})
	}
}

func TestSearchSessionProgressiveRankingHasLinearWork(t *testing.T) {
	const count = 3001
	dir := t.TempDir()
	paths := make([]string, count)
	for i := range paths {
		paths[i] = filepath.Join(dir, fmt.Sprintf("%04d.jpg", i))
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Each candidate evaluation has a cancellation checkpoint. This generous
	// linear budget includes preparation and a reference change, but rejects
	// rescanning the growing prefix at every publication without a timing limit.
	bounded := &searchRankCancelContext{Context: ctx, cancel: cancel, after: 8*count + 100}
	queries := make(chan SearchQuery, 1)
	query := SearchQuery{ID: 1, ReferencePath: paths[0]}
	queries <- query
	var prepared []Item
	byPath := map[string]Item{}
	publications := 0
	err := runSearchSession(bounded, SearchRequest{Paths: paths, Limit: 30}, queries, func(_ context.Context, path string) (Item, bool, error) {
		i := len(prepared)
		vector := make([]float32, 768)
		vector[0], vector[1] = float32(i%9+1), float32(i%13-6)
		item := Item{Path: path, Embedding: vector}
		if i > 0 && i%37 == 0 {
			item.Error = "unreadable image"
		}
		prepared = append(prepared, item)
		byPath[path] = item
		return item, true, nil
	}, nil, func(event SearchEvent) error {
		if event.Kind == SearchProgress && event.Processed == 350 {
			query = SearchQuery{ID: 2, ReferencePath: paths[150]}
			queries <- query
		}
		if event.Kind == SearchPartial || event.Kind == SearchFinal {
			want, err := RankSimilar(context.Background(), byPath[query.ReferencePath], prepared, 30)
			if err != nil || !slices.Equal(event.Matches, want) {
				t.Fatalf("incremental ranking differs at %d prepared sources: %v", event.Processed, err)
			}
			publications++
			if event.Kind == SearchFinal {
				close(queries)
			}
		}
		return nil
	})
	if err != nil || publications != 32 {
		t.Fatalf("progressive ranking exceeded linear work or lost publications: checks=%d publications=%d error=%v", bounded.checks, publications, err)
	}
}

func BenchmarkSearchSessionProgressive(b *testing.B) {
	for _, count := range []int{1000, 10000} {
		b.Run(fmt.Sprint(count), func(b *testing.B) {
			_, items := searchRankCorpus(count)
			paths := make([]string, len(items))
			byPath := make(map[string]Item, len(items))
			dir := b.TempDir()
			for i := range items {
				items[i].Path = filepath.Join(dir, items[i].Path)
				paths[i] = items[i].Path
				byPath[paths[i]] = items[i]
			}
			prepare := func(_ context.Context, path string) (Item, bool, error) { return byPath[path], true, nil }
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				queries := make(chan SearchQuery, 1)
				queries <- SearchQuery{ID: 1, ReferencePath: paths[0]}
				close(queries)
				if err := runSearchSession(context.Background(), SearchRequest{Paths: paths, Limit: 30}, queries, prepare, nil, func(_ SearchEvent) error { return nil }); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func TestSearchSessionBoundsPartialValidationAndChecksFinalScope(t *testing.T) {
	for name, changedIndex := range map[string]int{"reference": 0, "match": 1, "unranked": 50} {
		t.Run(name, func(t *testing.T) {
			paths := make([]string, 301)
			dir := t.TempDir()
			for i := range paths {
				paths[i] = filepath.Join(dir, fmt.Sprintf("%03d.jpg", i))
				if err := os.WriteFile(paths[i], []byte("source"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			p := searchPreparer{versions: map[string]os.FileInfo{}}
			queries := make(chan SearchQuery, 1)
			queries <- SearchQuery{ID: 1, ReferencePath: paths[0]}
			close(queries)
			prepare := func(_ context.Context, path string) (Item, bool, error) {
				info, err := os.Stat(path)
				if err != nil {
					return Item{}, false, err
				}
				p.versions[path] = info
				vector := make([]float32, 768)
				vector[0] = 1
				return Item{Path: path, Embedding: vector}, true, nil
			}
			var publications []int
			processed := 0
			err := runSearchSession(context.Background(), SearchRequest{Paths: paths, Limit: 30}, queries, prepare, p.validate, func(event SearchEvent) error {
				processed = event.Processed
				if event.Kind == SearchPartial || event.Kind == SearchFinal {
					publications = append(publications, event.Processed)
					if event.Processed == 100 {
						return os.Remove(paths[changedIndex])
					}
				}
				return nil
			})
			wantPublications, wantProcessed := []int{100}, 200
			if name == "unranked" {
				wantPublications, wantProcessed = []int{100, 200, 300}, len(paths)
			}
			if err == nil || !strings.Contains(err.Error(), paths[changedIndex]) || !slices.Equal(publications, wantPublications) || processed != wantProcessed {
				t.Fatalf("validation boundaries: publications=%v processed=%d error=%v", publications, processed, err)
			}
		})
	}
}
