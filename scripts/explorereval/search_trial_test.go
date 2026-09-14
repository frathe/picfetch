//go:build explorertrial && darwin

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image/color"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestProductionSearchEvaluation(t *testing.T) {
	t.Setenv("PICFETCH_EXPLORER_TEST_PROCESS", "1")
	corpus := searchFixtureCorpus()
	library := writeSearchFixtureCorpus(t, corpus)
	for i, source := range corpus.Sources {
		pixels := uitest.EncodePNG(t, 80, 60, color.NRGBA{R: uint8(i * 11), G: uint8(240 - i*7), B: 70, A: 255})
		if err := os.WriteFile(filepath.Join(library, source.Path), pixels, 0600); err != nil {
			t.Fatal(err)
		}
	}
	assets, err := filepath.Abs("../../.scratch/visual-similarity-explorer/assets")
	if err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "review")
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	output := searchCorpusChangingWriter{path: filepath.Join(library, "search-corpus.json")}
	if err := run(ctx, []string{"-search-evaluate", "-assets", assets, "-library", library, "-out", out}, &output); err != nil {
		t.Fatalf("real search command: %v\n%s", err, &output)
	}
	if !output.changed {
		t.Fatal("fixture did not change the manifest after its captured scan")
	}
	data, err := os.ReadFile(filepath.Join(out, "search-result.json"))
	if err != nil {
		t.Fatalf("search command did not produce ranking evidence: %v", err)
	}
	var report searchEvaluation
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	if report.Successful != 21 || report.Failed != 0 || len(report.Queries) != 20 || !report.OfflineVerified || report.PeakRSSBytes <= 0 || report.EncodeSeconds <= 0 {
		t.Fatalf("incomplete native search evidence: %+v", report)
	}
	for _, query := range report.Queries {
		if query.Error != "" || len(query.Matches) != 20 {
			t.Fatalf("native reference failed: %+v", query)
		}
		for _, match := range query.Matches {
			if match.ID == query.ReferenceID {
				t.Fatalf("reference matched itself: %s", match.ID)
			}
		}
	}
	data, err = os.ReadFile(filepath.Join(out, "favorite-profile.json"))
	if err != nil {
		t.Fatalf("search evaluation lacks cross-session Favorite reuse evidence: %v", err)
	}
	var reuse struct {
		InputSHA256 string
		Passes      []profilePass
	}
	if err := json.Unmarshal(data, &reuse); err != nil {
		t.Fatal(err)
	}
	if len(reuse.Passes) != 2 || reuse.Passes[0].Measurements.InferenceAttempts != 21 || reuse.Passes[1].Measurements.InferenceAttempts != 0 || reuse.Passes[1].Reused != 21 {
		t.Fatalf("cold/warm Favorite measurements did not prove reuse: %+v", reuse)
	}
	if reuse.InputSHA256 != report.SourceSHA256 {
		t.Fatal("unchanged ranking and Favorite baseline describe different sources")
	}
	t.Run("source changes after ranking", func(t *testing.T) {
		baselineOut := t.TempDir()
		for _, name := range []string{"search-corpus.json", "search-result.json"} {
			captured, err := os.ReadFile(filepath.Join(out, name))
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(baselineOut, name), captured, 0600); err != nil {
				t.Fatal(err)
			}
		}
		changed := uitest.EncodePNG(t, 80, 60, color.NRGBA{R: 255, A: 255})
		if err := os.WriteFile(filepath.Join(library, corpus.Sources[0].Path), changed, 0600); err != nil {
			t.Fatal(err)
		}
		err := profileSearchFavorites(ctx, configuration{Assets: assets, Library: library, Out: baselineOut}, io.Discard)
		if err == nil || !strings.Contains(err.Error(), "sources changed since ranking") {
			t.Fatalf("changed-source Favorite baseline succeeded or failed for another reason: %v", err)
		}
		if _, err := os.Stat(filepath.Join(baselineOut, "favorite-profile.json")); !errors.Is(err, os.ErrNotExist) {
			t.Fatalf("changed-source baseline retained a completed evidence file: %v", err)
		}
	})
}

// Command output observes the captured scan, then simulates a user editing the
// input manifest before the parent starts the separate Favorite baseline.
type searchCorpusChangingWriter struct {
	buffer  bytes.Buffer
	path    string
	changed bool
}

func (w *searchCorpusChangingWriter) Write(data []byte) (int, error) {
	n, err := w.buffer.Write(data)
	if err == nil && !w.changed && strings.Contains(w.String(), "search processed 21/21:") {
		w.changed = true
		err = os.WriteFile(w.path, []byte(`{"version":999}`), 0600)
	}
	return n, err
}

func (w *searchCorpusChangingWriter) String() string { return w.buffer.String() }
