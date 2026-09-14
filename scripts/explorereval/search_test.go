package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"io"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/uitest"
)

func TestSearchEvaluationRequiresReferenceCorpus(t *testing.T) {
	library := t.TempDir()
	if err := os.WriteFile(filepath.Join(library, "search-corpus.json"), []byte(`{"version":1,"sources":[],"queries":[]}`), 0600); err != nil {
		t.Fatal(err)
	}
	err := run(context.Background(), []string{"-search-evaluate", "-library", library, "-assets", filepath.Join(library, "no-assets")}, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "at least 20 distinct content references") {
		t.Fatalf("empty search corpus: %v; want reference-corpus rejection before asset setup", err)
	}
}

func TestSearchEvaluationRejectsConflictingCommands(t *testing.T) {
	for _, extra := range [][]string{{"-install"}, {"-probe"}, {"-trial", "throughput"}, {"-automatic"}, {"-provider", "coreml"}, {"-native", "some-app"}} {
		err := run(context.Background(), append([]string{"-search-evaluate"}, extra...), io.Discard)
		if err == nil || !strings.HasPrefix(err.Error(), "search evaluation cannot") {
			t.Errorf("conflicting search arguments %v: %v", extra, err)
		}
	}
}

func TestSearchEvaluationRejectsTrailingManifest(t *testing.T) {
	library := writeSearchFixtureCorpus(t, searchFixtureCorpus())
	path := filepath.Join(library, "search-corpus.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(data, []byte(`{}`)...), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readSearchCorpus(library); err == nil || !strings.Contains(err.Error(), "trailing") {
		t.Fatalf("trailing JSON was ignored: %v", err)
	}
}

func TestSearchEvaluationCancellationPreservesEvidence(t *testing.T) {
	corpus := searchFixtureCorpus()
	library := writeSearchFixtureCorpus(t, corpus)
	for _, source := range corpus.Sources {
		if err := os.WriteFile(filepath.Join(library, source.Path), uitest.EncodePNG(t, 3, 2, color.White), 0600); err != nil {
			t.Fatal(err)
		}
	}
	encoder := &searchFixtureEncoder{}
	native := searchRuntime{
		NewEncoder:    func() (searchEncoder, error) { return encoder, nil },
		VerifyOffline: func(_ context.Context) error { return nil },
		PeakRSS:       func() (int64, error) { return 4096, nil },
	}
	out := filepath.Join(t.TempDir(), "partial")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	err := evaluateSearch(ctx, configuration{Library: library, Out: out}, cancelSearchWriter{cancel: cancel}, native)
	if !errors.Is(err, context.Canceled) || encoder.calls != 1 || !encoder.closed {
		t.Fatalf("canceled source work did not stop: err=%v calls=%d closed=%v", err, encoder.calls, encoder.closed)
	}
	if _, err := os.Stat(filepath.Join(out, "search-result.json")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancellation published a complete result: %v", err)
	}
	before, err := os.ReadFile(filepath.Join(out, "search-corpus.json"))
	if err != nil {
		t.Fatal(err)
	}
	err = evaluateSearch(context.Background(), configuration{Library: library, Out: out}, io.Discard, native)
	if err == nil || !strings.Contains(err.Error(), "must be new") || encoder.calls != 1 {
		t.Fatalf("retry overwrote retained partial evidence: %v", err)
	}
	after, err := os.ReadFile(filepath.Join(out, "search-corpus.json"))
	if err != nil || string(before) != string(after) {
		t.Fatal("refused retry changed the captured corpus")
	}
}

type cancelSearchWriter struct{ cancel context.CancelFunc }

func (w cancelSearchWriter) Write(data []byte) (int, error) {
	w.cancel()
	return len(data), nil
}

func TestSearchEvaluationRejectsUnboundedOrMisspelledCorpus(t *testing.T) {
	corpus := searchFixtureCorpus()
	library := writeSearchFixtureCorpus(t, corpus)
	path := filepath.Join(library, "search-corpus.json")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `"relevant_ids"`, `"relevent_ids"`, 1))
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readSearchCorpus(library); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("misspelled judgments silently became unreviewed: %v", err)
	}
	if err := os.Truncate(path, 16*1024*1024+1); err != nil {
		t.Fatal(err)
	}
	if _, err := readSearchCorpus(library); err == nil || !strings.Contains(err.Error(), "16 MiB") {
		t.Fatalf("oversized corpus not bounded: %v", err)
	}
	corpus.Sources = make([]searchSource, (256*1024*1024)/(768*4)+1)
	library = writeSearchFixtureCorpus(t, corpus)
	if _, err := readSearchCorpus(library); err == nil || !strings.Contains(err.Error(), "256 MiB") {
		t.Fatalf("vector admission did not precede allocation/validation: %v", err)
	}
}

func TestSearchEvaluationRejectsAmbiguousCorpus(t *testing.T) {
	for _, tc := range []struct {
		name, want string
		change     func(*searchCorpus)
	}{
		{"version", "version", func(c *searchCorpus) { c.Version = 2 }},
		{"duplicate identity", "duplicate source", func(c *searchCorpus) { c.Sources[1].ID = c.Sources[0].ID }},
		{"duplicate path", "duplicate source path", func(c *searchCorpus) { c.Sources[1].Path = c.Sources[0].Path }},
		{"escaping path", "relative source path", func(c *searchCorpus) { c.Sources[0].Path = "../outside.jpg" }},
		{"absolute path", "relative source path", func(c *searchCorpus) { c.Sources[0].Path = "/outside.jpg" }},
		{"unknown reference", "unknown reference", func(c *searchCorpus) { c.Queries[0].ReferenceID = "absent" }},
		{"unknown relevance", "unknown relevant", func(c *searchCorpus) { c.Queries[0].RelevantIDs = []string{"absent"} }},
		{"self relevance", "reference cannot be relevant", func(c *searchCorpus) { c.Queries[0].RelevantIDs = []string{c.Queries[0].ReferenceID} }},
		{"repeated relevance", "duplicate relevant", func(c *searchCorpus) { c.Queries[0].RelevantIDs = []string{"s-01", "s-01"} }},
		{"repeated query", "duplicate query", func(c *searchCorpus) { c.Queries = append(c.Queries, c.Queries[0]) }},
		{"unknown intent", "intent", func(c *searchCorpus) { c.Queries[0].Intent = "confidence" }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			corpus := searchFixtureCorpus()
			tc.change(&corpus)
			library := writeSearchFixtureCorpus(t, corpus)
			err := run(context.Background(), []string{"-search-evaluate", "-library", library, "-assets", filepath.Join(library, "missing")}, io.Discard)
			if err == nil || !strings.HasPrefix(err.Error(), "search corpus") || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("invalid corpus returned %v; want %q", err, tc.want)
			}
		})
	}
}

func searchFixtureCorpus() searchCorpus {
	corpus := searchCorpus{Version: 1}
	for i := range 21 {
		id := fmt.Sprintf("s-%02d", i)
		corpus.Sources = append(corpus.Sources, searchSource{ID: id, Path: id + ".jpg"})
		if i < 20 {
			corpus.Queries = append(corpus.Queries, searchQuery{ReferenceID: id, Intent: "content", RelevantIDs: []string{"s-20"}})
		}
	}
	return corpus
}

func writeSearchFixtureCorpus(t *testing.T, corpus searchCorpus) string {
	t.Helper()
	library := t.TempDir()
	data, err := json.Marshal(corpus)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(library, "search-corpus.json"), data, 0600); err != nil {
		t.Fatal(err)
	}
	return library
}

func TestSearchEvaluationReportsRankingAndPendingJudgments(t *testing.T) {
	corpus := searchFixtureCorpus()
	corpus.Queries[0].RelevantIDs = []string{"s-01", "s-02"}
	corpus.Queries[1].RelevantIDs = nil
	corpus.Queries = append(corpus.Queries, searchQuery{ReferenceID: "s-00", Intent: "appearance"})
	library := writeSearchFixtureCorpus(t, corpus)
	for _, source := range corpus.Sources {
		if err := os.WriteFile(filepath.Join(library, source.Path), uitest.EncodePNG(t, 3, 2, color.White), 0600); err != nil {
			t.Fatal(err)
		}
	}
	encoder := &searchFixtureEncoder{}
	report := runSearchFixture(t, library, encoder)
	out := report.Config.Out
	if report.Successful != 21 || report.Failed != 0 || report.ContentJudged != 19 || report.Decision != "pending human decision" {
		t.Fatalf("incorrect quality accounting: %+v", report)
	}
	if len(report.Queries) != 21 || len(report.Queries[0].Matches) != 20 || report.Queries[0].Matches[0].ID != "s-01" {
		t.Fatalf("rank order, count or self-exclusion incorrect: %+v", report.Queries)
	}
	if p := report.Queries[0].PrecisionAt10; p == nil || *p != 0.2 {
		t.Fatalf("precision at ten = %v; want 0.2", p)
	}
	if report.Queries[1].PrecisionAt10 != nil || report.Queries[20].PrecisionAt10 != nil {
		t.Fatal("unreviewed content/appearance was reported as measured precision")
	}
	if encoder.calls != 21 || !encoder.closed {
		t.Fatalf("native inference lifecycle: calls=%d closed=%v", encoder.calls, encoder.closed)
	}
	for _, name := range []string{"search-review.html", "search-evaluation.md", "search-corpus.json"} {
		if _, err := os.Stat(filepath.Join(out, name)); err != nil {
			t.Fatal(err)
		}
	}
	page, err := os.ReadFile(filepath.Join(out, "search-review.html"))
	if err != nil {
		t.Fatal(err)
	}
	for _, label := range []string{"Download reviewed corpus", "Mark this reference reviewed", "P@10", "connect-src 'none'"} {
		if !strings.Contains(string(page), label) {
			t.Errorf("local review does not provide %q", label)
		}
	}
}

func TestSearchEvaluationRecordsReproducibleFirstResult(t *testing.T) {
	corpus := searchFixtureCorpus()
	// The reference is deliberately not the first source in manifest order.
	corpus.Queries[0], corpus.Queries[19] = corpus.Queries[19], corpus.Queries[0]
	library := writeSearchFixtureCorpus(t, corpus)
	for _, source := range corpus.Sources {
		if err := os.WriteFile(filepath.Join(library, source.Path), uitest.EncodePNG(t, 3, 2, color.White), 0600); err != nil {
			t.Fatal(err)
		}
	}
	report := runSearchFixture(t, library, &searchFixtureEncoder{})
	if len(report.SourceSHA256) != 64 || report.FirstPartialSeconds <= 0 || report.FirstPartialSeconds > report.PreparationSeconds {
		t.Fatalf("missing source identity or first-result measurement: digest=%q first=%v complete=%v", report.SourceSHA256, report.FirstPartialSeconds, report.PreparationSeconds)
	}
	data, err := os.ReadFile(filepath.Join(report.Config.Out, "search-initial.json"))
	if err != nil {
		t.Fatal(err)
	}
	var first struct {
		Processed int
		Result    searchQueryResult
	}
	if err := json.Unmarshal(data, &first); err != nil {
		t.Fatal(err)
	}
	if first.Processed != 21 || first.Result.ReferenceID != "s-19" || len(first.Result.Matches) != 20 {
		t.Fatalf("wrong final flush for a scope below 100: %+v", first)
	}
	data, err = os.ReadFile(filepath.Join(report.Config.Out, "search-result.json"))
	if err != nil {
		t.Fatal(err)
	}
	var measured struct {
		Benchmark struct {
			Candidates, Queries    int
			VectorBytes            int64
			P50Seconds, P95Seconds float64
		}
	}
	if err := json.Unmarshal(data, &measured); err != nil {
		t.Fatal(err)
	}
	if measured.Benchmark.Candidates != 10000 || measured.Benchmark.Queries < 20 || measured.Benchmark.VectorBytes != 10000*768*4 || measured.Benchmark.P50Seconds <= 0 || measured.Benchmark.P95Seconds < measured.Benchmark.P50Seconds {
		t.Fatalf("missing reproducible 10,000-vector measurements: %+v", measured.Benchmark)
	}
}

func TestSearchEvaluationRejectsChangedPreparedReference(t *testing.T) {
	corpus := searchFixtureCorpus()
	library := writeSearchFixtureCorpus(t, corpus)
	for _, source := range corpus.Sources {
		if err := os.WriteFile(filepath.Join(library, source.Path), uitest.EncodePNG(t, 3, 2, color.White), 0600); err != nil {
			t.Fatal(err)
		}
	}
	reference := filepath.Join(library, corpus.Sources[0].Path)
	info, err := os.Stat(reference)
	if err != nil {
		t.Fatal(err)
	}
	encoder := &searchFixtureEncoder{onEncode: func(call int) {
		if call == 2 {
			changed := info.ModTime().Add(time.Second)
			if err := os.Chtimes(reference, changed, changed); err != nil {
				t.Fatal(err)
			}
		}
	}}
	report := runSearchFixture(t, library, encoder)
	if report.Successful != 20 || report.Failed != 1 || report.Queries[0].Error == "" || report.Queries[0].PrecisionAt10 != nil || report.RetainedVectorBytes != 20*768*4 {
		t.Fatalf("changed prepared reference was published: successful=%d failed=%d bytes=%d query=%+v", report.Successful, report.Failed, report.RetainedVectorBytes, report.Queries[0])
	}
}

func TestSearchEvaluationReportsSparseAndInvalidRepresentations(t *testing.T) {
	corpus := searchFixtureCorpus()
	corpus.Queries[0].RelevantIDs = []string{"s-18"}
	library := writeSearchFixtureCorpus(t, corpus)
	for _, source := range corpus.Sources[:20] { // Last source is intentionally unreadable.
		if err := os.WriteFile(filepath.Join(library, source.Path), uitest.EncodePNG(t, 3, 2, color.White), 0600); err != nil {
			t.Fatal(err)
		}
	}
	encoder := &searchFixtureEncoder{represent: func(call int) ([]float32, error) {
		vector := make([]float32, 768)
		switch call {
		case 1, 19:
			vector[0] = 1 // Distinct paths with identical pixels remain eligible.
		case 2:
			return []float32{1}, nil
		case 3: // A zero vector is invalid too.
		case 4:
			vector[0] = float32(math.NaN())
		case 5:
			vector[0] = float32(math.Inf(1))
		case 20:
			vector[0] = -1 // A weak match is retained without a score cutoff.
		default:
			return nil, errors.New("fixture inference failure")
		}
		return vector, nil
	}}
	report := runSearchFixture(t, library, encoder)
	query := report.Queries[0]
	if report.Successful != 3 || report.Failed != 18 || len(query.Matches) != 2 || query.Matches[0].ID != "s-18" || query.Matches[1].ID != "s-19" || query.Matches[1].Score != -1 {
		t.Fatalf("invalid or weak source accounting: successful=%d failed=%d query=%+v", report.Successful, report.Failed, query)
	}
	if query.PrecisionAt10 == nil || *query.PrecisionAt10 != .1 || report.Queries[1].PrecisionAt10 != nil || report.Queries[1].Error == "" {
		t.Fatal("missing slots or failed references distorted precision")
	}
}

func runSearchFixture(t *testing.T, library string, encoder searchEncoder) searchEvaluation {
	t.Helper()
	out := filepath.Join(t.TempDir(), "review")
	native := searchRuntime{
		NewEncoder:    func() (searchEncoder, error) { return encoder, nil },
		VerifyOffline: func(_ context.Context) error { return nil },
		PeakRSS:       func() (int64, error) { return 4096, nil },
	}
	if err := evaluateSearch(context.Background(), configuration{Library: library, Out: out}, io.Discard, native); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(out, "search-result.json"))
	if err != nil {
		t.Fatal(err)
	}
	var report searchEvaluation
	if err := json.Unmarshal(data, &report); err != nil {
		t.Fatal(err)
	}
	return report
}

type searchFixtureEncoder struct {
	calls     int
	closed    bool
	onEncode  func(int)
	represent func(int) ([]float32, error)
}

func (e *searchFixtureEncoder) Encode(_ context.Context, _ image.Image) ([]float32, error) {
	e.calls++
	if e.onEncode != nil {
		e.onEncode(e.calls)
	}
	if e.represent != nil {
		return e.represent(e.calls)
	}
	vector := make([]float32, 768)
	vector[0] = 1
	return vector, nil
}

func (e *searchFixtureEncoder) Close() { e.closed = true }
