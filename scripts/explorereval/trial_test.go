//go:build explorertrial

package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image/color"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/uitest"
)

// This test must be run inside the same denied-network boundary as the trial.
// It requires the real pinned assets; missing assets fail instead of skipping.
func TestRealEvaluation(t *testing.T) {
	library := t.TempDir()
	red := uitest.EncodeJPEG(t, 300, 180, color.NRGBA{R: 255, A: 255})
	blue := uitest.EncodeJPEG(t, 300, 180, color.NRGBA{B: 255, A: 255})
	for name, data := range map[string][]byte{"red.jpg": red, "red-copy.jpg": red, "blue.jpg": blue, "broken.jpg": []byte("broken JPEG")} {
		if err := os.WriteFile(filepath.Join(library, name), data, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "run")
	if err := run(context.Background(), []string{"-worker", "-assets", assets, "-library", library, "-out", out}, io.Discard); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "result.json"))
	if err != nil {
		t.Fatalf("successful real evaluation must deliver result.json: %v", err)
	}
	var result struct {
		OfflineVerified bool
		FirstMapSeconds float64
		PeakRSSBytes    int64
		Items           []struct {
			Path      string
			Error     string
			Embedding []float32
			Cohort    string
			Position  []float32
		}
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if !result.OfflineVerified || len(result.Items) != 4 {
		t.Fatalf("want offline verification and all four inputs, got offline=%v items=%d", result.OfflineVerified, len(result.Items))
	}
	seen := map[string]bool{}
	vectors := map[string][]float32{}
	for _, item := range result.Items {
		name := filepath.Base(item.Path)
		if seen[name] {
			t.Fatalf("repeated input %s", name)
		}
		seen[name] = true
		if name == "broken.jpg" {
			if item.Error == "" || len(item.Embedding) != 0 || item.Cohort != "" {
				t.Fatalf("broken input reported as represented: %+v", item)
			}
			continue
		}
		if item.Error != "" || len(item.Embedding) != 768 || item.Cohort == "" || len(item.Position) != 2 {
			t.Fatalf("incomplete representation for %s", name)
		}
		var norm float64
		for _, v := range item.Embedding {
			norm += float64(v) * float64(v)
		}
		if math.IsNaN(norm) || math.Abs(norm-1) > 1e-5 {
			t.Fatalf("invalid normalized representation: %v", norm)
		}
		for _, v := range item.Position {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				t.Fatal("non-finite map")
			}
		}
		vectors[name] = item.Embedding
	}
	cosine := func(a, b []float32) float64 {
		var dot float64
		for i, v := range a {
			dot += float64(v) * float64(b[i])
		}
		return dot
	}
	if cosine(vectors["red.jpg"], vectors["red-copy.jpg"]) < 0.99999 {
		t.Fatal("identical input did not preserve its representation")
	}
	if cosine(vectors["red.jpg"], vectors["blue.jpg"]) > 0.99 {
		t.Fatal("encoder did not distinguish different source pixels")
	}
	if result.FirstMapSeconds <= 0 || result.PeakRSSBytes <= 0 {
		t.Fatal("missing first-map timing or native peak-memory measurement")
	}
	initial, err := os.ReadFile(filepath.Join(out, "initial.json"))
	if err != nil {
		t.Fatalf("missing initial map before full regrouping: %v", err)
	}
	var snapshot struct{ Items []struct{ Path string } }
	if err := json.Unmarshal(initial, &snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Items) != 2 {
		t.Fatalf("initial map must account for the first half, got %d", len(snapshot.Items))
	}
	review, err := os.ReadFile(filepath.Join(out, "review.html"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(review), "red.jpg") || !strings.Contains(string(review), "blue.jpg") || !strings.Contains(string(review), "broken.jpg") {
		t.Fatal("review must expose successful sources and failed inputs")
	}
}

func TestRealEvaluationCancelsAfterProgress(t *testing.T) {
	library := t.TempDir()
	if err := os.WriteFile(filepath.Join(library, "photo.jpg"), uitest.EncodeJPEG(t, 40, 30, color.White), 0o600); err != nil {
		t.Fatal(err)
	}
	assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "canceled")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err = run(ctx, []string{"-worker", "-assets", assets, "-library", library, "-out", out}, cancelOnProgress{cancel: cancel})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation after real processing returned %v", err)
	}
	if _, err := os.Stat(filepath.Join(out, "result.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled work published a completed result: %v", err)
	}
}

type cancelOnProgress struct{ cancel context.CancelFunc }

func (w cancelOnProgress) Write(p []byte) (int, error) {
	if strings.Contains(string(p), "processed ") {
		w.cancel()
	}
	return len(p), nil
}

func TestRealCommandHandlesTermination(t *testing.T) {
	library := t.TempDir()
	pixels := uitest.EncodeJPEG(t, 40, 30, color.White)
	for i := range 64 {
		if err := os.WriteFile(filepath.Join(library, fmt.Sprintf("photo-%03d.jpg", i)), pixels, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "terminated")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	command := exec.CommandContext(ctx, os.Args[0], "-worker", "-assets", assets, "-library", library, "-out", out)
	command.Env = append(os.Environ(), "PICFETCH_EXPLORER_TEST_PROCESS=1")
	var stderr bytes.Buffer
	command.Stderr = &stderr
	stdout, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if command.ProcessState == nil {
			_ = command.Process.Kill()
			_ = command.Wait()
		}
	})
	scanner := bufio.NewScanner(stdout)
	if !scanner.Scan() || !strings.HasPrefix(scanner.Text(), "processed ") {
		t.Fatalf("worker did not report actual processing: %q", scanner.Text())
	}
	if err := command.Process.Signal(syscall.SIGTERM); err != nil {
		t.Fatal(err)
	}
	for scanner.Scan() {
	}
	err = command.Wait()
	if err == nil || command.ProcessState.ExitCode() != 1 || !strings.Contains(stderr.String(), "context canceled") {
		t.Fatalf("termination must follow the cancellation/cleanup path: %v, stderr %s", err, stderr.String())
	}
	if _, err := os.Stat(filepath.Join(out, "result.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("terminated worker published completion: %v", err)
	}
}
