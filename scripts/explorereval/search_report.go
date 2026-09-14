package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
)

//go:embed search_review.html
var searchReviewHTML string

func writeSearchReview(report searchEvaluation, corpus searchCorpus) error {
	encoded, err := json.Marshal(corpus)
	if err != nil {
		return err
	}
	view := struct {
		searchEvaluation
		CorpusJSON string
	}{report, string(encoded)}
	page, err := template.New("search").Funcs(template.FuncMap{
		"precision": searchPrecision,
		"judged":    func(value *float64) bool { return value != nil },
	}).Parse(searchReviewHTML)
	if err != nil {
		return err
	}
	file, err := os.OpenFile(filepath.Join(report.Config.Out, "search-review.html"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	err = page.Execute(file, view)
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	text := fmt.Sprintf(`# Search evaluation

Decision: **%s**. Technical measurements do not supply a human quality verdict.

## Relevance

- Sources: %d represented, %d failed.
- Content: %d judged; median precision at ten %s. Initial target: 0.6 across at least 20 content references.
- Appearance: %d judged; separate median precision at ten %s.
- Unreviewed or failed references have no measured precision. Missing result slots count as non-relevant.

## Measurements

- Setup %.3f s; source read/decode %.3f s; inference %.3f s.
- First ranked result %.3f s; complete preparation %.3f s. These are worker computation boundaries, not Grid paint or interaction latency. The initial first-search target is 30 seconds.
- Warm exact queries: p50 %.6f s, p95 %.6f s on the prepared corpus, without new inference.
- Synthetic exact ranking: %d candidates, %d timed queries after warm-up, %d vector bytes; p50 %.6f s, p95 %.6f s. Initial p95 target: 0.200 s. Deterministic 768-dimensional vectors; these timings do not establish real-image relevance.
- Worker maximum RSS: %d bytes, measured after ranking and before report serialization; includes Go, native inference and synthetic benchmark storage, excludes the parent and separate Favorite baseline. Retained real-source vectors: %d bytes.

## Reproducibility

Model %s; representation %s. CPU provider, six inference threads, canonical full oriented decoding; exact cosine in the original 768 dimensions. This trial uses full-sort ranking, with path ties. FML-002 will move the evaluated algorithm behind the production bounded top-k interface.

Machine target %s/%s; %s. Corpus SHA-256: %s. Source identity/version/content SHA-256: %s.

[Local review](search-review.html) · [Initial ranked result](search-initial.json) · [Retained vectors and results](search-result.json).

[Favorite reuse baseline](favorite-profile.json) is produced by separate cold/warm production Explorer workers after this ranking worker exits. Those timings include map generation and are not search latency. The temporary Favorite is isolated from user Favorites and removed before its final profile is written. A failed/canceled command may retain partial artifacts; only a successful command with both reports supplies complete technical evidence.

General-cache and final production-search measurements remain pending until their implementation. The macOS experiment retains OS-enforced network denial and makes no other-platform or style/composition guarantee. Targets are local evaluation criteria, not universal performance claims.

Use the local review controls to mark relevance and download the updated corpus; that review uses the retained ranking and does not run inference. Save the judgments and record a human proceed/revise decision before FML-002.
`, report.Decision, report.Successful, report.Failed, report.ContentJudged, searchPrecision(report.MedianContentP10), report.AppearanceJudged, searchPrecision(report.MedianAppearanceP10), report.SetupSeconds, report.DecodeSeconds, report.EncodeSeconds, report.FirstPartialSeconds, report.PreparationSeconds, report.WarmP50Seconds, report.WarmP95Seconds, report.Benchmark.Candidates, report.Benchmark.Queries, report.Benchmark.VectorBytes, report.Benchmark.P50Seconds, report.Benchmark.P95Seconds, report.PeakRSSBytes, report.RetainedVectorBytes, report.ModelRevision, report.RepresentationVersion, report.GOOS, report.GOARCH, report.GoVersion, report.CorpusSHA256, report.SourceSHA256)
	return os.WriteFile(filepath.Join(report.Config.Out, "search-evaluation.md"), []byte(text), 0600)
}

func searchPrecision(value *float64) string {
	if value == nil {
		return "unreviewed"
	}
	return fmt.Sprintf("%.3f", *value)
}
