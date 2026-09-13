package similarity

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"testing"
)

func TestEmbeddedTagVectorsMatchJSON(t *testing.T) {
	source, err := os.ReadFile("tag-vectors.json")
	if err != nil {
		t.Fatal(err)
	}
	var vectors map[string][]float32
	if err := json.Unmarshal(source, &vectors); err != nil {
		t.Fatal(err)
	}
	ids := TagIDs()
	if got, want := len(tagVectors), len(ids)*768*4; got != want {
		t.Fatalf("embedded vector bytes = %d, want %d exact float32 bytes", got, want)
	}
	tagger, err := NewTagger()
	if err != nil {
		t.Fatal(err)
	}
	for i, id := range ids {
		want := vectors[id]
		if len(want) != 768 {
			t.Fatalf("JSON vector %s has %d values", id, len(want))
		}
		for j, value := range want {
			offset := (i*768 + j) * 4
			bits := math.Float32bits(value)
			if binary.LittleEndian.Uint32(tagVectors[offset:]) != bits || math.Float32bits(tagger.prototypes[i].vector[j]) != bits {
				t.Fatalf("tag %s value %d changed float32 bits", id, j)
			}
		}
		if tagger.prototypes[i].id != id {
			t.Fatalf("prototype %d has identity %q, want %q", i, tagger.prototypes[i].id, id)
		}
	}
}

func TestTaggerRejectsInvalidAssets(t *testing.T) {
	for _, name := range []string{"truncated", "extended", "corrupt", "wrong model", "empty ID", "duplicate ID", "non-unit", "NaN", "infinite"} {
		t.Run(name, func(t *testing.T) {
			data := bytes.Clone(tagVectors)
			var catalog map[string]any
			if err := json.Unmarshal(tagCatalog, &catalog); err != nil {
				t.Fatal(err)
			}
			switch name {
			case "truncated":
				data = data[:len(data)-1]
			case "extended":
				data = append(data, 0)
			case "corrupt":
				data[0] ^= 1
			case "wrong model":
				catalog["model"] = "different model"
			case "empty ID":
				catalog["tags"].([]any)[0].(map[string]any)["id"] = ""
			case "duplicate ID":
				tags := catalog["tags"].([]any)
				tags[1].(map[string]any)["id"] = tags[0].(map[string]any)["id"]
			case "non-unit", "NaN", "infinite":
				value := float32(2)
				if name == "NaN" {
					value = float32(math.NaN())
				} else if name == "infinite" {
					value = float32(math.Inf(1))
				}
				binary.LittleEndian.PutUint32(data, math.Float32bits(value))
				// A matching hash must not bypass the numeric validation.
				catalog["vectorsSHA256"] = fmt.Sprintf("%x", sha256.Sum256(data))
			}
			catalogData, err := json.Marshal(catalog)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := newTagger(catalogData, data); err == nil {
				t.Fatal("accepted invalid tag assets")
			}
		})
	}
}

func BenchmarkNewTagger(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		if _, err := NewTagger(); err != nil {
			b.Fatal(err)
		}
	}
}
