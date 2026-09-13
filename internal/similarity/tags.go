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

//go:generate go run ../../scripts/tagvectors -catalog tag-catalog.json -source tag-vectors.json -out tag-vectors.bin
//go:embed tag-vectors.bin
var tagVectors []byte

// TagCatalogueVersion binds preset rules to the prompt vectors and thresholds.
func TagCatalogueVersion() string { return fmt.Sprintf("%x", sha256.Sum256(tagCatalog)) }

// TagIDs returns a fresh list of the fixed catalogue's source-independent IDs.
func TagIDs() []string {
	var catalog struct{ Tags []struct{ ID string } }
	if err := json.Unmarshal(tagCatalog, &catalog); err != nil {
		return nil
	}
	ids := make([]string, 0, len(catalog.Tags))
	for _, tag := range catalog.Tags {
		ids = append(ids, tag.ID)
	}
	return ids
}

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
	return newTagger(tagCatalog, tagVectors)
}

func newTagger(catalogData, vectorData []byte) (*Tagger, error) {
	var catalog struct {
		Model, VectorsSHA256           string
		Scale, Bias, MinimumScore      float64
		MinimumShare, MinimumBestShare float64
		Tags                           []struct{ ID string }
	}
	if err := json.Unmarshal(catalogData, &catalog); err != nil {
		return nil, err
	}
	if catalog.Model != ModelRevision || len(catalog.Tags) == 0 || len(vectorData) != len(catalog.Tags)*768*4 || catalog.Scale <= 0 || catalog.MinimumScore <= 0 || catalog.MinimumScore >= 1 || catalog.MinimumShare <= 0 || catalog.MinimumBestShare < catalog.MinimumShare || catalog.MinimumBestShare >= 1 {
		return nil, fmt.Errorf("invalid semantic tag assets")
	}
	// The generator stores canonical float32 bits in catalogue order. The
	// digest therefore identifies exactly the same values as the source JSON.
	if fmt.Sprintf("%x", sha256.Sum256(vectorData)) != catalog.VectorsSHA256 {
		return nil, fmt.Errorf("semantic tag vector checksum mismatch")
	}
	tagger := &Tagger{scale: catalog.Scale, bias: catalog.Bias, minimumScore: catalog.MinimumScore,
		minimumShare: catalog.MinimumShare, bestShare: catalog.MinimumBestShare}
	seen := map[string]bool{}
	for i, tag := range catalog.Tags {
		if tag.ID == "" || seen[tag.ID] {
			return nil, fmt.Errorf("invalid semantic tag identity")
		}
		seen[tag.ID] = true
		prototype := tagPrototype{id: tag.ID, vector: make([]float32, 768)}
		var norm float64
		for j := range prototype.vector {
			offset := (i*768 + j) * 4
			value := math.Float32frombits(binary.LittleEndian.Uint32(vectorData[offset:]))
			prototype.vector[j] = value
			norm += float64(value) * float64(value)
		}
		if math.IsNaN(norm) || math.Abs(norm-1) > .0001 {
			return nil, fmt.Errorf("invalid semantic tag vector")
		}
		tagger.prototypes = append(tagger.prototypes, prototype)
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
