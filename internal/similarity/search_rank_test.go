package similarity

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand/v2"
	"reflect"
	"slices"
	"sort"
	"testing"
	"time"
)

func TestSearchRankKnownCosines(t *testing.T) {
	reference := Item{Path: "reference", Embedding: searchRankVector(3, 4)}
	candidates := []Item{
		{Path: "opposite", Embedding: searchRankVector(-3, -4)},
		{Path: "orthogonal", Embedding: searchRankVector(4, -3)},
		{Path: "z-identical", Embedding: searchRankVector(3, 4)},
		{Path: "near", Embedding: searchRankVector(1, 0)},
		{Path: "a-scaled", Embedding: searchRankVector(6, 8)},
	}
	want := []Match{{"a-scaled", 1}, {"z-identical", 1}, {"near", .6}, {"orthogonal", 0}, {"opposite", -1}}
	got, err := RankSimilar(context.Background(), reference, candidates, 30)
	if err != nil {
		t.Fatal(err)
	}
	assertSearchRankMatches(t, got, want)
}

func TestSearchRankTopKOracle(t *testing.T) {
	for _, ties := range []int{0, 40} {
		reference, candidates := searchRankCorpus(257)
		// Equal-score paths straddle the boundary in reverse path order.
		for i := range ties {
			candidates[i].Embedding = slices.Clone(reference.Embedding)
		}
		for _, limit := range []int{-1, 0, 1, 7, 30, 300, math.MaxInt} {
			t.Run(fmt.Sprintf("ties_%d/limit_%d", ties, limit), func(t *testing.T) {
				want := searchRankOracle(reference, candidates, limit)
				for range 2 {
					got, err := RankSimilar(context.Background(), reference, candidates, limit)
					if err != nil {
						t.Fatal(err)
					}
					assertSearchRankMatches(t, got, want)
					slices.Reverse(candidates)
				}
			})
		}
	}
}

func TestSearchRankInvalidReference(t *testing.T) {
	for name, reference := range map[string]Item{
		"missing":           {Path: "reference"},
		"short":             {Path: "reference", Embedding: []float32{1}},
		"long":              {Path: "reference", Embedding: append(searchRankVector(1), 0)},
		"zero":              {Path: "reference", Embedding: searchRankVector()},
		"nan":               {Path: "reference", Embedding: searchRankVector(1, float32(math.NaN()))},
		"positive_infinity": {Path: "reference", Embedding: searchRankVector(1, float32(math.Inf(1)))},
		"negative_infinity": {Path: "reference", Embedding: searchRankVector(1, float32(math.Inf(-1)))},
		"failed":            {Path: "reference", Embedding: searchRankVector(1), Error: "decode failed"},
	} {
		t.Run(name, func(t *testing.T) {
			for _, limit := range []int{0, 30} {
				got, err := RankSimilar(context.Background(), reference, nil, limit)
				if err == nil || len(got) != 0 {
					t.Fatalf("invalid reference at limit %d: matches %v, error %v", limit, got, err)
				}
			}
		})
	}
}

func TestSearchRankSkipsInvalidCandidates(t *testing.T) {
	reference := Item{Path: "reference", Embedding: searchRankVector(1)}
	for name, invalid := range map[string]Item{
		"missing":           {Path: "invalid"},
		"short":             {Path: "invalid", Embedding: []float32{1}},
		"long":              {Path: "invalid", Embedding: append(searchRankVector(1), 0)},
		"zero":              {Path: "invalid", Embedding: searchRankVector()},
		"nan":               {Path: "invalid", Embedding: searchRankVector(1, float32(math.NaN()))},
		"positive_infinity": {Path: "invalid", Embedding: searchRankVector(1, float32(math.Inf(1)))},
		"negative_infinity": {Path: "invalid", Embedding: searchRankVector(1, float32(math.Inf(-1)))},
		"failed":            {Path: "invalid", Embedding: searchRankVector(1), Error: "decode failed"},
	} {
		t.Run(name, func(t *testing.T) {
			candidates := []Item{invalid, {Path: "usable", Embedding: searchRankVector(-1)}}
			got, err := RankSimilar(context.Background(), reference, candidates, 30)
			if err != nil {
				t.Fatal(err)
			}
			assertSearchRankMatches(t, got, []Match{{"usable", -1}})
		})
	}
}

func TestSearchRankDistinctPaths(t *testing.T) {
	reference := Item{Path: "reference", Embedding: searchRankVector(1)}
	candidates := []Item{
		reference,
		{Path: "repeated", Embedding: searchRankVector(0, 1)},
		{Path: "z-copy", Embedding: slices.Clone(reference.Embedding)},
		{Path: "a-copy", Embedding: slices.Clone(reference.Embedding)},
		{Path: "opposite", Embedding: searchRankVector(-1)},
		{Path: "weak", Embedding: searchRankVector(0, 1)},
		{Path: "repeated", Embedding: searchRankVector(1)},
		{Path: "z-copy", Embedding: slices.Clone(reference.Embedding)},
		{Path: "repeated", Embedding: searchRankVector(-1)},
	}
	want := []Match{{"a-copy", 1}, {"repeated", 1}, {"z-copy", 1}, {"weak", 0}, {"opposite", -1}}
	for _, limit := range []int{1, 2, 3, 30} {
		t.Run(fmt.Sprint(limit), func(t *testing.T) {
			for range 2 {
				got, err := RankSimilar(context.Background(), reference, candidates, limit)
				if err != nil {
					t.Fatal(err)
				}
				assertSearchRankMatches(t, got, want[:min(limit, len(want))])
				slices.Reverse(candidates)
			}
		})
	}
}

func TestSearchRankCancellation(t *testing.T) {
	reference, candidates := searchRankCorpus(128)
	for _, after := range []int{1, 17} {
		t.Run(fmt.Sprintf("check_%d", after), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			checking := &searchRankCancelContext{Context: ctx, cancel: cancel, after: after}
			got, err := RankSimilar(checking, reference, candidates, 30)
			if !errors.Is(err, context.Canceled) || len(got) != 0 {
				t.Fatalf("cancelled search: matches %v, error %v", got, err)
			}
		})
	}
	t.Run("empty_cancelled_query", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		got, err := RankSimilar(ctx, reference, nil, 0)
		if !errors.Is(err, context.Canceled) || len(got) != 0 {
			t.Fatalf("cancelled empty search: matches %v, error %v", got, err)
		}
	})
	t.Run("deadline", func(t *testing.T) {
		ctx, cancel := context.WithDeadline(context.Background(), time.Unix(0, 0))
		defer cancel()
		got, err := RankSimilar(ctx, reference, candidates, 30)
		if !errors.Is(err, context.DeadlineExceeded) || len(got) != 0 {
			t.Fatalf("expired search: matches %v, error %v", got, err)
		}
	})
}

func TestSearchRankImmutableInputs(t *testing.T) {
	_, candidates := searchRankCorpus(64)
	reference := candidates[5]
	candidates[7].Embedding = reference.Embedding
	candidates[0].Tags = []string{"tag"}
	candidates[0].Position = []float32{1, 2}
	candidates[0].Preview = []byte{1, 2, 3}
	before := make([]Item, len(candidates))
	for i, candidate := range candidates {
		before[i] = candidate
		before[i].Embedding = slices.Clone(candidate.Embedding)
		before[i].Tags = slices.Clone(candidate.Tags)
		before[i].Position = slices.Clone(candidate.Position)
		before[i].Preview = slices.Clone(candidate.Preview)
	}
	got, err := RankSimilar(context.Background(), reference, candidates, 30)
	if err != nil || len(got) != 30 {
		t.Fatalf("search: match count %d, error %v", len(got), err)
	}
	if !reflect.DeepEqual(candidates, before) || !reflect.DeepEqual(reference, before[5]) {
		t.Fatal("ranking changed caller-owned items or shared vector buffers")
	}
	got[0].Path = "caller-owned result"
	got[0].Score = -100
	again, err := RankSimilar(context.Background(), reference, candidates, 30)
	if err != nil || len(again) != 30 || again[0].Path == got[0].Path {
		t.Fatalf("result mutation leaked into another query: match count %d, error %v", len(again), err)
	}
}

func TestSearchRankFiniteExtremes(t *testing.T) {
	for name, scale := range map[string]float32{
		"smallest": math.SmallestNonzeroFloat32,
		"largest":  math.MaxFloat32,
	} {
		t.Run(name, func(t *testing.T) {
			reference := Item{Path: "reference", Embedding: searchRankVector(scale)}
			candidates := []Item{
				{Path: "opposite", Embedding: searchRankVector(-scale)},
				{Path: "same", Embedding: searchRankVector(scale)},
				{Path: "orthogonal", Embedding: searchRankVector(0, scale)},
			}
			got, err := RankSimilar(context.Background(), reference, candidates, 30)
			if err != nil {
				t.Fatal(err)
			}
			assertSearchRankMatches(t, got, []Match{{"same", 1}, {"orthogonal", 0}, {"opposite", -1}})
		})
	}
}

func BenchmarkSearchRank(b *testing.B) {
	reference, candidates := searchRankCorpus(10000)
	b.ReportAllocs()
	for b.Loop() {
		matches, err := RankSimilar(context.Background(), reference, candidates, 30)
		if err != nil || len(matches) != 30 {
			b.Fatalf("search: match count %d, error %v", len(matches), err)
		}
	}
}

// Cancel at an observed context checkpoint, without timing assumptions or a
// mutable production hook, including while a candidate scan is in progress.
type searchRankCancelContext struct {
	context.Context
	cancel context.CancelFunc
	after  int
	checks int
}

func (c *searchRankCancelContext) Err() error {
	c.checks++
	if c.checks == c.after {
		c.cancel()
	}
	return c.Context.Err()
}

func searchRankCorpus(count int) (Item, []Item) {
	random := rand.New(rand.NewPCG(17, 29))
	vector := func() []float32 {
		values := make([]float32, 768)
		for i := range values {
			values[i] = 2*random.Float32() - 1
		}
		return values
	}
	reference := Item{Path: "reference", Embedding: vector()}
	candidates := make([]Item, count)
	for i := range candidates {
		candidates[i] = Item{Path: fmt.Sprintf("image-%05d", count-i), Embedding: vector()}
	}
	return reference, candidates
}

// The oracle normalizes copies first and fully sorts every candidate, separately
// from the production ranker's streaming bounded selection.
func searchRankOracle(reference Item, candidates []Item, limit int) []Match {
	if limit <= 0 {
		return nil
	}
	normalized := func(vector []float32) []float64 {
		values := make([]float64, len(vector))
		var length float64
		for _, value := range vector {
			length = math.Hypot(length, float64(value))
		}
		for i, value := range vector {
			values[i] = float64(value) / length
		}
		return values
	}
	ref := normalized(reference.Embedding)
	matches := make([]Match, 0, len(candidates))
	for _, candidate := range candidates {
		var score float64
		for i, value := range normalized(candidate.Embedding) {
			score += value * ref[i]
		}
		matches = append(matches, Match{candidate.Path, score})
	}
	sort.Slice(matches, func(i, j int) bool {
		if matches[i].Score == matches[j].Score {
			return matches[i].Path < matches[j].Path
		}
		return matches[i].Score > matches[j].Score
	})
	return matches[:min(limit, len(matches))]
}

func searchRankVector(values ...float32) []float32 {
	vector := make([]float32, 768)
	copy(vector, values)
	return vector
}

func assertSearchRankMatches(t *testing.T, got, want []Match) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("match count = %d; want %d", len(got), len(want))
	}
	for i := range want {
		if got[i].Path != want[i].Path || math.IsNaN(got[i].Score) || math.Abs(got[i].Score-want[i].Score) > 1e-12 {
			t.Errorf("match %d = %+v; want %+v", i, got[i], want[i])
		}
	}
}
