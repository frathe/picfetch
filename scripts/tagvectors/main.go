// Command tagvectors converts the authoritative JSON vectors into exact
// little-endian float32 data for embedding, without loading a model or runtime.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"math"
	"os"
)

func main() {
	catalog := flag.String("catalog", "internal/similarity/tag-catalog.json", "source catalogue")
	source := flag.String("source", "internal/similarity/tag-vectors.json", "source JSON float32 vectors")
	out := flag.String("out", "internal/similarity/tag-vectors.bin", "generated binary vectors")
	check := flag.Bool("check", false, "fail if generated output is missing or stale")
	flag.Parse()
	if err := generate(*catalog, *source, *out, *check); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(catalogPath, sourcePath, outPath string, check bool) error {
	catalog, err := os.ReadFile(catalogPath)
	if err != nil {
		return err
	}
	source, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	encoded, err := encodeVectors(catalog, source)
	if err != nil {
		return err
	}
	existing, err := os.ReadFile(outPath)
	if err == nil && bytes.Equal(existing, encoded) {
		return nil
	}
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if check {
		return fmt.Errorf("%s is missing or stale; run make generate-tag-vectors", outPath)
	}
	return os.WriteFile(outPath, encoded, 0644)
}

func encodeVectors(catalogData, source []byte) ([]byte, error) {
	var catalog struct {
		VectorsSHA256 string
		Tags          []struct{ ID string }
	}
	if err := json.Unmarshal(catalogData, &catalog); err != nil {
		return nil, err
	}
	var vectors map[string][]float32
	if err := json.Unmarshal(source, &vectors); err != nil {
		return nil, err
	}
	if len(catalog.Tags) == 0 || len(vectors) != len(catalog.Tags) {
		return nil, fmt.Errorf("vector and catalogue counts differ or are empty")
	}
	seen := make(map[string]bool, len(catalog.Tags))
	encoded := make([]byte, 0, len(catalog.Tags)*768*4)
	for _, tag := range catalog.Tags {
		if tag.ID == "" || seen[tag.ID] {
			return nil, fmt.Errorf("invalid or duplicate tag identity %q", tag.ID)
		}
		seen[tag.ID] = true
		vector := vectors[tag.ID]
		if len(vector) != 768 {
			return nil, fmt.Errorf("tag %q must have 768 values", tag.ID)
		}
		var norm float64
		for _, value := range vector {
			norm += float64(value) * float64(value)
			encoded = binary.LittleEndian.AppendUint32(encoded, math.Float32bits(value))
		}
		if math.IsNaN(norm) || math.Abs(norm-1) > .0001 {
			return nil, fmt.Errorf("tag %q is not a finite unit vector", tag.ID)
		}
	}
	if fmt.Sprintf("%x", sha256.Sum256(encoded)) != catalog.VectorsSHA256 {
		return nil, fmt.Errorf("vector checksum differs from catalogue")
	}
	return encoded, nil
}
