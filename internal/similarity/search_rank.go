package similarity

import (
	"cmp"
	"context"
	"fmt"
	"math"
	"slices"
)

// Match identifies a source and its cosine similarity to the reference image.
type Match struct {
	Path  string
	Score float64
}

// RankSimilar returns at most limit matches ordered by descending exact cosine,
// with ascending paths breaking ties. Inputs are never changed. Embeddings must
// contain 768 finite values and have a nonzero norm; an invalid or failed reference
// is an error, while invalid or failed candidates are skipped. The reference path
// is excluded, and repeated candidate paths retain their highest valid score.
// After checking cancellation and the reference, a nonpositive limit returns an
// empty result. Cancellation returns no partial matches.
func RankSimilar(ctx context.Context, reference Item, candidates []Item, limit int) ([]Match, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if reference.Error != "" {
		return nil, fmt.Errorf("search reference failed: %s", reference.Error)
	}
	referenceNorm, valid := searchEmbeddingNorm(reference.Embedding)
	if !valid {
		return nil, fmt.Errorf("search reference requires a finite nonzero 768-dimensional embedding")
	}
	if limit <= 0 {
		return nil, nil
	}
	limit = min(limit, len(candidates))
	// Both ordering and duplicate tracking use only the retained top-k. An evicted
	// path can qualify again only if a later representation improves its score.
	matches := make([]Match, 0, limit)
	for _, candidate := range candidates {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if candidate.Path == reference.Path || candidate.Error != "" {
			continue
		}
		norm, usable := searchEmbeddingNorm(candidate.Embedding)
		if !usable {
			continue
		}
		var dot float64
		for i, value := range candidate.Embedding {
			dot += float64(value) * float64(reference.Embedding[i])
		}
		score := dot / (referenceNorm * norm)
		match := Match{Path: candidate.Path, Score: score}
		duplicate := slices.IndexFunc(matches, func(prior Match) bool { return prior.Path == match.Path })
		if duplicate >= 0 {
			if matches[duplicate].Score >= score {
				continue
			}
			matches = slices.Delete(matches, duplicate, duplicate+1)
		}
		at, _ := slices.BinarySearchFunc(matches, match, compareSearchMatches)
		if at >= limit {
			continue
		}
		if len(matches) < limit {
			matches = append(matches, Match{})
		}
		copy(matches[at+1:], matches[at:])
		matches[at] = match
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return matches, nil
}

func searchEmbeddingNorm(vector []float32) (float64, bool) {
	if len(vector) != 768 {
		return 0, false
	}
	var squared float64
	for _, value := range vector {
		x := float64(value)
		if math.IsNaN(x) || math.IsInf(x, 0) {
			return 0, false
		}
		squared += x * x
	}
	return math.Sqrt(squared), squared > 0
}

func compareSearchMatches(a, b Match) int {
	if scoreOrder := cmp.Compare(b.Score, a.Score); scoreOrder != 0 {
		return scoreOrder
	}
	return cmp.Compare(a.Path, b.Path)
}
