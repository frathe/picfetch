package similarity

import (
	"crypto/sha256"
	_ "embed"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
)

//go:embed tag-catalog.json
var tagCatalog []byte

//go:embed tag-vectors.json
var tagVectors []byte

// Tagger applies the pinned, fixed prompt catalogue to image representations.
// It owns only small immutable text vectors; it never opens images or a runtime.
type Tagger struct {
	prototypes                            []tagPrototype
	scale, bias                           float64
	minimumScore, minimumShare, bestShare float64
}

type tagPrototype struct {
	id     string
	vector []float32
}

func NewTagger() (*Tagger, error) {
	var catalog struct {
		Model, VectorsSHA256           string
		Scale, Bias, MinimumScore      float64
		MinimumShare, MinimumBestShare float64
		Tags                           []struct{ ID string }
	}
	if err := json.Unmarshal(tagCatalog, &catalog); err != nil {
		return nil, err
	}
	var vectors map[string][]float32
	if err := json.Unmarshal(tagVectors, &vectors); err != nil {
		return nil, err
	}
	if catalog.Model != ModelRevision || len(catalog.Tags) == 0 || len(vectors) != len(catalog.Tags) || catalog.Scale <= 0 || catalog.MinimumScore <= 0 || catalog.MinimumScore >= 1 || catalog.MinimumShare <= 0 || catalog.MinimumBestShare < catalog.MinimumShare || catalog.MinimumBestShare >= 1 {
		return nil, fmt.Errorf("invalid semantic tag assets")
	}
	tagger := &Tagger{scale: catalog.Scale, bias: catalog.Bias, minimumScore: catalog.MinimumScore,
		minimumShare: catalog.MinimumShare, bestShare: catalog.MinimumBestShare}
	seen := map[string]bool{}
	// Hash canonical float32 bits in catalogue order, independently of JSON
	// whitespace/key order. This keeps the original vector identity unchanged.
	canonical := make([]byte, 0, len(catalog.Tags)*768*4)
	for _, tag := range catalog.Tags {
		if tag.ID == "" || seen[tag.ID] {
			return nil, fmt.Errorf("invalid semantic tag identity")
		}
		seen[tag.ID] = true
		prototype := tagPrototype{id: tag.ID, vector: vectors[tag.ID]}
		if len(prototype.vector) != 768 {
			return nil, fmt.Errorf("invalid semantic tag vector dimensions")
		}
		var norm float64
		for _, value := range prototype.vector {
			canonical = binary.LittleEndian.AppendUint32(canonical, math.Float32bits(value))
			norm += float64(value) * float64(value)
		}
		if math.IsNaN(norm) || math.Abs(norm-1) > .0001 {
			return nil, fmt.Errorf("invalid semantic tag vector")
		}
		tagger.prototypes = append(tagger.prototypes, prototype)
	}
	if fmt.Sprintf("%x", sha256.Sum256(canonical)) != catalog.VectorsSHA256 {
		return nil, fmt.Errorf("semantic tag vector checksum mismatch")
	}
	return tagger, nil
}

// Tags returns every qualifying ID, never a forced highest-scoring label.
// Relative shares compare this fixed catalogue, not calibrated confidence.
func (t *Tagger) Tags(vector []float32) []string {
	if len(vector) != 768 {
		return nil
	}
	var norm float64
	for _, value := range vector {
		norm += float64(value) * float64(value)
	}
	if norm == 0 || math.IsNaN(norm) || math.IsInf(norm, 0) {
		return nil
	}
	norm = math.Sqrt(norm)
	scores := make([]float64, len(t.prototypes))
	var total, best float64
	for i, prototype := range t.prototypes {
		var dot float64
		for i, value := range vector {
			dot += float64(value) * float64(prototype.vector[i])
		}
		score := 1 / (1 + math.Exp(-(t.scale*dot/norm + t.bias)))
		scores[i] = score
		total += score
		best = max(best, score)
	}
	// Absolute caption scores vary widely across subjects. Reject weak or
	// ambiguous matches before selecting every substantial relative match.
	if best < t.minimumScore || total == 0 || best/total < t.bestShare {
		return nil
	}
	var tags []string
	for i, prototype := range t.prototypes {
		if scores[i]/total >= t.minimumShare {
			tags = append(tags, prototype.id)
		}
	}
	return tags
}
