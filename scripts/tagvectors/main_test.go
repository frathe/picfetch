package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"testing"
)

func vectorFixture(t *testing.T) ([]byte, map[string][]float32, []byte) {
	t.Helper()
	vectors := map[string][]float32{"first": make([]float32, 768), "second": make([]float32, 768)}
	vectors["first"][0] = 1
	vectors["second"][1] = -1
	want := make([]byte, 2*768*4)
	// The catalogue deliberately differs from alphabetical JSON key order.
	binary.LittleEndian.PutUint32(want[4:], math.Float32bits(-1))
	binary.LittleEndian.PutUint32(want[768*4:], math.Float32bits(1))
	digest := fmt.Sprintf("%x", sha256.Sum256(want))
	catalog := []byte(fmt.Sprintf(`{"vectorsSHA256":%q,"tags":[{"id":"second"},{"id":"first"}]}`, digest))
	return catalog, vectors, want
}

func TestEncodeVectorsPreservesBitsAndCatalogueOrder(t *testing.T) {
	catalog, vectors, want := vectorFixture(t)
	for _, indent := range []string{"", "  "} {
		source, err := json.MarshalIndent(vectors, "", indent)
		if err != nil {
			t.Fatal(err)
		}
		got, err := encodeVectors(catalog, source)
		if err != nil || !bytes.Equal(got, want) {
			t.Fatalf("conversion error = %v; exact bits match = %v", err, bytes.Equal(got, want))
		}
	}
}

func TestEncodeVectorsRejectsInvalidSources(t *testing.T) {
	for _, name := range []string{"missing", "extra", "dimension", "non-unit", "checksum", "duplicate", "invalid JSON"} {
		t.Run(name, func(t *testing.T) {
			catalog, vectors, _ := vectorFixture(t)
			switch name {
			case "missing":
				delete(vectors, "first")
			case "extra":
				vectors["third"] = vectors["first"]
			case "dimension":
				vectors["first"] = vectors["first"][:767]
			case "non-unit":
				vectors["first"][0] = 2
			case "checksum":
				vectors["first"][0] = -1
			case "duplicate":
				catalog = bytes.ReplaceAll(catalog, []byte(`"second"`), []byte(`"first"`))
			case "invalid JSON":
				catalog = []byte("{")
			}
			source, err := json.Marshal(vectors)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := encodeVectors(catalog, source); err == nil {
				t.Fatal("accepted invalid source assets")
			}
		})
	}
}

func TestGenerateChecksFreshnessAndPreservesOutputOnInvalidSource(t *testing.T) {
	catalog, vectors, want := vectorFixture(t)
	source, err := json.Marshal(vectors)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	catalogPath, sourcePath, outPath := filepath.Join(dir, "catalog.json"), filepath.Join(dir, "source.json"), filepath.Join(dir, "vectors.bin")
	for path, data := range map[string][]byte{catalogPath: catalog, sourcePath: source} {
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := generate(catalogPath, sourcePath, outPath, true); err == nil {
		t.Fatal("check accepted missing output")
	}
	if err := generate(catalogPath, sourcePath, outPath, false); err != nil {
		t.Fatal(err)
	}
	if err := generate(catalogPath, sourcePath, outPath, true); err != nil {
		t.Fatal(err)
	}
	stale := bytes.Clone(want)
	stale[0] ^= 1
	if err := os.WriteFile(outPath, stale, 0600); err != nil {
		t.Fatal(err)
	}
	if err := generate(catalogPath, sourcePath, outPath, true); err == nil {
		t.Fatal("check accepted stale output")
	}
	if err := os.WriteFile(sourcePath, []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := generate(catalogPath, sourcePath, outPath, false); err == nil {
		t.Fatal("generation accepted invalid source")
	}
	got, err := os.ReadFile(outPath)
	if err != nil || !bytes.Equal(got, stale) {
		t.Fatalf("failed generation changed existing output: %v", err)
	}
}
