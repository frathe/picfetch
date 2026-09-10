// Command explorertags regenerates the small embedded semantic tag vectors.
// It is development tooling; the viewer never loads or executes a text model.
package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"math"
	"os"

	ort "github.com/yalue/onnxruntime_go"
)

func main() {
	if err := run(); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	model := flag.String("model", "", "pinned text_model.onnx")
	runtime := flag.String("runtime", "", "local ONNX Runtime library")
	tokens := flag.String("tokens", "", "prepare_tokens.py output")
	catalogPath := flag.String("catalog", "internal/similarity/tag-catalog.json", "prompt catalogue with tag identities")
	out := flag.String("out", "", "output JSON float32 vectors keyed by tag identity")
	flag.Parse()
	file, err := os.Open(*model)
	if err != nil {
		return err
	}
	hash := sha256.New()
	_, err = io.Copy(hash, file)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	if fmt.Sprintf("%x", hash.Sum(nil)) != "baf12d941beabafafb14f7b4adb38dc15be18681b964a84410ec53d9d65e6293" {
		return fmt.Errorf("text model checksum mismatch")
	}
	data, err := os.ReadFile(*tokens)
	if err != nil {
		return err
	}
	var prompts [][]int64
	if err := json.Unmarshal(data, &prompts); err != nil {
		return err
	}
	if len(prompts) == 0 {
		return fmt.Errorf("no prompts")
	}
	data, err = os.ReadFile(*catalogPath)
	if err != nil {
		return err
	}
	var catalog struct{ Tags []struct{ ID string } }
	if err := json.Unmarshal(data, &catalog); err != nil {
		return err
	}
	if len(catalog.Tags) != len(prompts) {
		return fmt.Errorf("tag catalogue and token counts differ")
	}
	seen := map[string]bool{}
	for _, tag := range catalog.Tags {
		if tag.ID == "" || seen[tag.ID] {
			return fmt.Errorf("invalid tag identity")
		}
		seen[tag.ID] = true
	}
	ort.SetSharedLibraryPath(*runtime)
	if err := ort.InitializeEnvironment(); err != nil {
		return err
	}
	defer func() { _ = ort.DestroyEnvironment() }()
	input, err := ort.NewEmptyTensor[int64](ort.NewShape(1, 64))
	if err != nil {
		return err
	}
	defer func() { _ = input.Destroy() }()
	output, err := ort.NewEmptyTensor[float32](ort.NewShape(1, 768))
	if err != nil {
		return err
	}
	defer func() { _ = output.Destroy() }()
	options, err := ort.NewSessionOptions()
	if err != nil {
		return err
	}
	defer func() { _ = options.Destroy() }()
	if err := options.SetIntraOpNumThreads(6); err != nil {
		return err
	}
	session, err := ort.NewAdvancedSession(*model, []string{"input_ids"}, []string{"pooler_output"}, []ort.Value{input}, []ort.Value{output}, options)
	if err != nil {
		return err
	}
	defer func() { _ = session.Destroy() }()
	vectors := make(map[string][]float32, len(prompts))
	var canonical []byte
	for i, prompt := range prompts {
		if len(prompt) != 64 {
			return fmt.Errorf("prompt must contain 64 token IDs")
		}
		copy(input.GetData(), prompt)
		if err := session.Run(); err != nil {
			return err
		}
		norm := 0.0
		for _, value := range output.GetData() {
			norm += float64(value) * float64(value)
		}
		if norm == 0 || math.IsNaN(norm) || math.IsInf(norm, 0) {
			return fmt.Errorf("invalid text vector")
		}
		vector := make([]float32, len(output.GetData()))
		for j, value := range output.GetData() {
			vector[j] = value / float32(math.Sqrt(norm))
			canonical = binary.LittleEndian.AppendUint32(canonical, math.Float32bits(vector[j]))
		}
		vectors[catalog.Tags[i].ID] = vector
	}
	data, err = json.MarshalIndent(vectors, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(*out, append(data, '\n'), 0644); err != nil {
		return err
	}
	_, _ = fmt.Printf("vectorsSHA256: %x\nwrote %s\n", sha256.Sum256(canonical), *out)
	return nil
}
