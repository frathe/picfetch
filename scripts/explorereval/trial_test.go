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
	"runtime"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/uitest"
)

// The command itself starts the production worker inside OS network denial.
// Run this test outside the outer sandbox used by the older evaluator tests.
func TestProductionProfile(t *testing.T) {
	library := t.TempDir()
	for name, pixels := range map[string][]byte{
		"red.jpg":    uitest.EncodeJPEG(t, 300, 180, color.NRGBA{R: 255, A: 255}),
		"blue.jpg":   uitest.EncodeJPEG(t, 300, 180, color.NRGBA{B: 255, A: 255}),
		"broken.jpg": []byte("broken JPEG"),
	} {
		if err := os.WriteFile(filepath.Join(library, name), pixels, 0600); err != nil {
			t.Fatal(err)
		}
	}
	assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "profile")
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := run(ctx, []string{"-trial", "throughput", "-assets", assets, "-library", library, "-out", out}, io.Discard); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "profile.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report struct {
		Available, Sampled            int
		ExecutableSHA256, InputSHA256 string
		Passes                        []struct {
			Successful, Failed, Reused           int
			OfflineVerified                      bool
			WorkerExitedSeconds, FirstMapSeconds float64
			Measurements                         struct {
				InferenceAttempts, Publications                int
				ElapsedSeconds, EncodeSeconds, GroupingSeconds float64
			}
		}
	}
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Available != 3 || report.Sampled != 3 || len(report.Passes) != 2 || len(report.ExecutableSHA256) != 64 || len(report.InputSHA256) != 64 {
		t.Fatalf("profile lost its corpus, executable identity or cold/warm passes: %s", data)
	}
	for i, pass := range report.Passes {
		if pass.Successful != 2 || pass.Failed != 1 || pass.Reused != i*2 || !pass.OfflineVerified || pass.Measurements.InferenceAttempts != (1-i)*2 || pass.Measurements.Publications != 1 || pass.Measurements.GroupingSeconds <= 0 || pass.FirstMapSeconds <= 0 || pass.WorkerExitedSeconds < pass.FirstMapSeconds || pass.WorkerExitedSeconds < pass.Measurements.ElapsedSeconds {
			t.Fatalf("pass %d lost failed-source, reuse, timing or exit evidence: %+v", i, pass)
		}
	}
	if report.Passes[0].Measurements.EncodeSeconds <= 0 || report.Passes[1].Measurements.EncodeSeconds != 0 {
		t.Fatal("profile attributed cache reuse to inference throughput")
	}
	events, err := os.ReadFile(filepath.Join(out, "events.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	for _, contents := range [][]byte{data, events} {
		for _, forbidden := range []string{library, "red.jpg", "blue.jpg", "broken.jpg", `"Embedding":`, `"Preview":`, `"Items":`} {
			if bytes.Contains(contents, []byte(forbidden)) {
				t.Fatalf("metadata-only profile contains %q", forbidden)
			}
		}
	}
	if err := run(ctx, []string{"-trial", "throughput", "-assets", assets, "-library", library, "-out", out}, io.Discard); err == nil {
		t.Fatal("profile overwrote existing evidence")
	}
	retained, err := os.ReadFile(filepath.Join(out, "profile.json"))
	if err != nil || !bytes.Equal(data, retained) {
		t.Fatal("refused rerun altered completed evidence")
	}
	canceledOut := filepath.Join(t.TempDir(), "canceled")
	cancelCtx, cancelRun := context.WithCancel(ctx)
	defer cancelRun()
	err = run(cancelCtx, []string{"-trial", "throughput", "-assets", assets, "-library", library, "-out", canceledOut}, cancelOnProgress{cancel: cancelRun})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("profiling cancellation after actual progress returned %v", err)
	}
	if _, err := os.Stat(filepath.Join(canceledOut, "profile.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("canceled profiling retained a completed summary: %v", err)
	}
	partial, err := os.ReadFile(filepath.Join(canceledOut, "events.jsonl"))
	if err != nil || len(partial) == 0 {
		t.Fatal("canceled profiling lost its completed progress evidence")
	}
	t.Run("bounded", func(t *testing.T) {
		large := t.TempDir()
		for i := range 513 {
			if err := os.WriteFile(filepath.Join(large, fmt.Sprintf("%04d.jpg", i)), []byte("broken JPEG"), 0600); err != nil {
				t.Fatal(err)
			}
		}
		bounded := filepath.Join(t.TempDir(), "bounded")
		if err := run(ctx, []string{"-trial", "throughput", "-assets", assets, "-library", large, "-out", bounded}, io.Discard); err != nil {
			t.Fatal(err)
		}
		data, err := os.ReadFile(filepath.Join(bounded, "profile.json"))
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal(data, &report); err != nil {
			t.Fatal(err)
		}
		if report.Available != 513 || report.Sampled != 512 || len(report.Passes) != 2 {
			t.Fatal("bounded profile silently admitted the full oversized collection")
		}
		for _, pass := range report.Passes {
			if pass.Successful != 0 || pass.Failed != 512 || pass.Reused != 0 || pass.Measurements.InferenceAttempts != 0 {
				t.Fatal("failed-source profile fabricated inference or lost sampled sources")
			}
		}
	})
}

// This test must be run inside the same denied-network boundary as the trial.
// It requires the real pinned assets; missing assets fail instead of skipping.
func TestRealEvaluationOrientationAllocations(t *testing.T) {
	for orientation := uint16(2); orientation <= 8; orientation++ {
		t.Run(fmt.Sprintf("orientation_%d", orientation), func(t *testing.T) {
			library := t.TempDir()
			if err := os.WriteFile(filepath.Join(library, "oriented.jpg"), uitest.EncodeOrientedJPEG(t, 1024, 768, orientation), 0600); err != nil {
				t.Fatal(err)
			}
			assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
			if err != nil {
				t.Fatal(err)
			}
			out := filepath.Join(t.TempDir(), "run")
			var before, after runtime.MemStats
			runtime.ReadMemStats(&before)
			err = run(context.Background(), []string{"-worker", "-assets", assets, "-library", library, "-out", out}, io.Discard)
			runtime.ReadMemStats(&after)
			if err != nil {
				t.Fatal(err)
			}
			// This bounds the complete real command, including inference and layout.
			// The allowance leaves ample room for those stages, but no per-pixel
			// object allocation when correcting an ordinary camera JPEG's orientation.
			allocations := after.Mallocs - before.Mallocs
			if allocations > 100_000 {
				t.Fatalf("oriented source analysis allocated %d objects; want at most 100000 without per-pixel boxing", allocations)
			}
			t.Logf("complete oriented-source analysis: %d allocations", allocations)
		})
	}
}

func TestRealEvaluationPreviewWorkspace(t *testing.T) {
	library := t.TempDir()
	if err := os.WriteFile(filepath.Join(library, "portrait.jpg"), uitest.EncodeJPEG(t, 3072, 4096, color.NRGBA{R: 173, G: 81, B: 29, A: 255}), 0600); err != nil {
		t.Fatal(err)
	}
	assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "run")
	var before, after runtime.MemStats
	runtime.ReadMemStats(&before)
	err = run(context.Background(), []string{"-worker", "-assets", assets, "-library", library, "-out", out}, io.Discard)
	runtime.ReadMemStats(&after)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "result.json"))
	if err != nil {
		t.Fatal(err)
	}
	var result evaluation
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Items) != 1 || result.Items[0].Error != "" || len(result.Items[0].Embedding) != 768 || result.Items[0].Thumbnail == "" {
		t.Fatal("workspace measurement did not complete source inference and preview delivery")
	}
	// Includes full JPEG decoding, real inference and final report delivery.
	// A preview must not add another source-height floating-point image.
	allocated := after.TotalAlloc - before.TotalAlloc
	if allocated > 64<<20 {
		t.Fatalf("source analysis allocated %d bytes; want at most 64 MiB with bounded preview workspace", allocated)
	}
	t.Logf("complete source analysis: %d allocated bytes", allocated)
}

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
