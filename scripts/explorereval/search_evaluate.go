package main

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"time"

	"fyne.io/fyne/v2/storage"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/similarity"
)

type searchEncoder interface {
	Encode(context.Context, image.Image) ([]float32, error)
	Close()
}

// The evaluation boundary substitutes only native inference and OS observations.
type searchRuntime struct {
	NewEncoder    func() (searchEncoder, error)
	VerifyOffline func(context.Context) error
	PeakRSS       func() (int64, error)
}

type searchMatch struct {
	ID, Path, Thumbnail string
	Score               float64
	Relevant            bool
}

type searchQueryResult struct {
	ReferenceID, Intent, Thumbnail, Error string
	Matches                               []searchMatch
	PrecisionAt10                         *float64
	RankSeconds                           float64
}

type searchBenchmark struct {
	Candidates, Queries    int
	VectorBytes            int64
	P50Seconds, P95Seconds float64
}

type searchEvaluation struct {
	Config                                     configuration
	ModelRevision, RepresentationVersion       string
	GOOS, GOARCH, GoVersion, CorpusSHA256      string
	SourceSHA256                               string
	OfflineVerified                            bool
	Successful, Failed, ContentJudged          int
	AppearanceJudged                           int
	MedianContentP10, MedianAppearanceP10      *float64
	Decision                                   string
	SetupSeconds, DecodeSeconds, EncodeSeconds float64
	PreparationSeconds, FirstPartialSeconds    float64
	WarmP50Seconds, WarmP95Seconds             float64
	PeakRSSBytes, RetainedVectorBytes          int64
	Items                                      []item
	Queries                                    []searchQueryResult
	Benchmark                                  searchBenchmark
}

func evaluateSearch(ctx context.Context, config configuration, output io.Writer, native searchRuntime) error {
	start := time.Now()
	if err := ctx.Err(); err != nil {
		return err
	}
	corpus, err := readSearchCorpus(config.Library)
	if err != nil {
		return err
	}
	if err := native.VerifyOffline(ctx); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(config.Out), 0700); err != nil {
		return err
	}
	if err := os.Mkdir(config.Out, 0700); err != nil {
		return fmt.Errorf("evidence directory must be new: %w", err)
	}
	if err := os.Mkdir(filepath.Join(config.Out, "images"), 0700); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(config.Out, "search-corpus.json"), corpus); err != nil {
		return err
	}
	registration := similarity.RegisterLocalFiles()
	if err := registration(); err != nil {
		return err
	}
	encoder, err := native.NewEncoder()
	if err != nil {
		return err
	}
	defer encoder.Close()
	report := searchEvaluation{
		Config: config, ModelRevision: similarity.ModelRevision, RepresentationVersion: similarity.RepresentationVersion,
		GOOS: runtime.GOOS, GOARCH: runtime.GOARCH, GoVersion: runtime.Version(),
		OfflineVerified: true, Decision: "pending human decision", SetupSeconds: time.Since(start).Seconds(),
	}
	report.Items = make([]item, len(corpus.Sources))
	versions := make([]os.FileInfo, len(corpus.Sources))
	manifest, err := os.ReadFile(filepath.Join(config.Out, "search-corpus.json"))
	if err != nil {
		return err
	}
	report.CorpusSHA256 = fmt.Sprintf("%x", sha256.Sum256(manifest))
	var accounting evaluation
	order := make([]int, 0, len(corpus.Sources))
	for i, source := range corpus.Sources {
		if source.ID == corpus.Queries[0].ReferenceID {
			order = append(order, i)
		}
	}
	for i := range corpus.Sources {
		if i != order[0] {
			order = append(order, i)
		}
	}
	for processed, i := range order {
		source := corpus.Sources[i]
		if err := ctx.Err(); err != nil {
			return err
		}
		entry := item{Path: filepath.Join(config.Library, filepath.FromSlash(source.Path))}
		before, sourceErr := os.Stat(entry.Path)
		versions[i] = before
		if sourceErr == nil {
			entry.Size, entry.ModifiedNS = before.Size(), before.ModTime().UnixNano()
			loaded, decodeErr := accounting.decode(ctx, storage.NewFileURI(entry.Path), &entry)
			sourceErr = decodeErr
			if sourceErr == nil {
				encodeStart := time.Now()
				entry.Embedding, sourceErr = encoder.Encode(ctx, loaded.Frames[0])
				report.EncodeSeconds += time.Since(encodeStart).Seconds()
				if sourceErr == nil && searchVectorNorm(entry.Embedding) == 0 {
					sourceErr = fmt.Errorf("invalid image representation")
				}
				if sourceErr == nil {
					entry.Thumbnail = fmt.Sprintf("images/%04d.jpg", i)
					if err := writeSearchPreview(config.Out, entry.Thumbnail, loaded.Frames[0]); err != nil {
						return err
					}
				}
			}
			if sourceErr == nil {
				after, statErr := os.Stat(entry.Path)
				if statErr != nil || !os.SameFile(before, after) || after.Size() != entry.Size || after.ModTime().UnixNano() != entry.ModifiedNS {
					sourceErr = fmt.Errorf("source changed during analysis")
				}
			}
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		if sourceErr != nil {
			entry.Error, entry.Embedding, entry.Thumbnail = sourceErr.Error(), nil, ""
			report.Failed++
		} else {
			report.Successful++
			report.RetainedVectorBytes += int64(len(entry.Embedding)) * 4
		}
		report.Items[i] = entry
		if _, err := fmt.Fprintf(output, "search processed %d/%d: ready=%d failed=%d\n", processed+1, len(corpus.Sources), report.Successful, report.Failed); err != nil {
			return err
		}
		if processed+1 == min(100, len(corpus.Sources)) {
			if err := revalidateSearchSources(ctx, &report, versions); err != nil {
				return err
			}
			first, err := rankSearchQuery(ctx, corpus, report.Items, corpus.Queries[0])
			if err != nil {
				return err
			}
			if first.Error == "" {
				report.FirstPartialSeconds = time.Since(start).Seconds()
			}
			if err := writeJSON(filepath.Join(config.Out, "search-initial.json"), struct {
				Processed int
				Result    searchQueryResult
			}{processed + 1, first}); err != nil {
				return err
			}
		}
	}
	if err := revalidateSearchSources(ctx, &report, versions); err != nil {
		return err
	}
	report.DecodeSeconds = accounting.DecodeSeconds
	report.PreparationSeconds = time.Since(start).Seconds()
	digest := sha256.New()
	identities := json.NewEncoder(digest)
	for _, entry := range report.Items {
		if err := identities.Encode(struct {
			Path, SHA256, Error string
			Size, ModifiedNS    int64
		}{entry.Path, entry.SHA256, entry.Error, entry.Size, entry.ModifiedNS}); err != nil {
			return err
		}
	}
	report.SourceSHA256 = fmt.Sprintf("%x", digest.Sum(nil))
	var content, appearance, timings []float64
	for _, query := range corpus.Queries {
		if err := ctx.Err(); err != nil {
			return err
		}
		result, err := rankSearchQuery(ctx, corpus, report.Items, query)
		if err != nil {
			return err
		}
		report.Queries = append(report.Queries, result)
		if result.Error == "" {
			timings = append(timings, result.RankSeconds)
		}
		if result.PrecisionAt10 != nil {
			if query.Intent == "content" {
				content = append(content, *result.PrecisionAt10)
			} else {
				appearance = append(appearance, *result.PrecisionAt10)
			}
		}
	}
	report.ContentJudged, report.AppearanceJudged = len(content), len(appearance)
	report.MedianContentP10, report.MedianAppearanceP10 = searchMedian(content), searchMedian(appearance)
	report.WarmP50Seconds, report.WarmP95Seconds = searchPercentile(timings, 0.5), searchPercentile(timings, 0.95)
	report.Benchmark, err = benchmarkSearchRanking(ctx)
	if err != nil {
		return err
	}
	report.PeakRSSBytes, err = native.PeakRSS()
	if err != nil {
		return err
	}
	if err := writeSearchReview(report, corpus); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	return writeJSON(filepath.Join(config.Out, "search-result.json"), report)
}

// A source prepared early can change while later inputs are still being decoded.
// Recheck the captured version at publication; the report keeps its failure as
// evidence, but never ranks that representation against a newer source file.
func revalidateSearchSources(ctx context.Context, report *searchEvaluation, versions []os.FileInfo) error {
	for i := range report.Items {
		if err := ctx.Err(); err != nil {
			return err
		}
		entry := &report.Items[i]
		if entry.Error != "" || len(entry.Embedding) == 0 {
			continue
		}
		now, err := os.Stat(entry.Path)
		if err == nil && os.SameFile(versions[i], now) && now.Size() == entry.Size && now.ModTime().UnixNano() == entry.ModifiedNS {
			continue
		}
		report.Successful--
		report.Failed++
		report.RetainedVectorBytes -= int64(len(entry.Embedding)) * 4
		entry.Error, entry.Embedding, entry.Thumbnail = "source changed before publication", nil, ""
	}
	return nil
}

func benchmarkSearchRanking(ctx context.Context) (searchBenchmark, error) {
	result := searchBenchmark{Candidates: 10000, Queries: 30, VectorBytes: 10000 * 768 * 4}
	corpus := searchCorpus{Version: 1, Sources: make([]searchSource, result.Candidates)}
	items := make([]item, result.Candidates)
	for i := range items {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		id := fmt.Sprintf("benchmark-%05d", i)
		corpus.Sources[i] = searchSource{ID: id, Path: id}
		items[i] = item{Path: id, Embedding: make([]float32, 768)}
		for j := range items[i].Embedding {
			items[i].Embedding[j] = float32((i+j*31)%257 - 128)
		}
	}
	query := searchQuery{ReferenceID: corpus.Sources[0].ID, Intent: "content"}
	if _, err := rankSearchQuery(ctx, corpus, items, query); err != nil {
		return result, err
	}
	timings := make([]float64, 0, result.Queries)
	for range result.Queries {
		ranked, err := rankSearchQuery(ctx, corpus, items, query)
		if err != nil {
			return result, err
		}
		timings = append(timings, ranked.RankSeconds)
	}
	result.P50Seconds, result.P95Seconds = searchPercentile(timings, .5), searchPercentile(timings, .95)
	return result, nil
}

func writeSearchPreview(directory, name string, source image.Image) error {
	f, err := os.OpenFile(filepath.Join(directory, name), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	err = jpeg.Encode(f, imaging.ScaleForExport(source, 240), &jpeg.Options{Quality: 85})
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}

func rankSearchQuery(ctx context.Context, corpus searchCorpus, items []item, query searchQuery) (searchQueryResult, error) {
	start := time.Now()
	result := searchQueryResult{ReferenceID: query.ReferenceID, Intent: query.Intent}
	ref := -1
	for i, source := range corpus.Sources {
		if source.ID == query.ReferenceID {
			ref = i
			break
		}
	}
	if ref < 0 || ref >= len(items) || items[ref].Error != "" || searchVectorNorm(items[ref].Embedding) == 0 {
		result.Error = "reference could not be represented"
		return result, nil
	}
	reference := items[ref]
	result.Thumbnail = reference.Thumbnail
	refNorm := searchVectorNorm(reference.Embedding)
	for i, candidate := range items {
		if err := ctx.Err(); err != nil {
			return result, err
		}
		if candidate.Path == reference.Path || candidate.Error != "" {
			continue
		}
		norm := searchVectorNorm(candidate.Embedding)
		if norm == 0 {
			continue
		}
		var dot float64
		for j, value := range reference.Embedding {
			dot += float64(value) * float64(candidate.Embedding[j])
		}
		id := corpus.Sources[i].ID
		result.Matches = append(result.Matches, searchMatch{ID: id, Path: candidate.Path, Thumbnail: candidate.Thumbnail, Score: dot / (refNorm * norm), Relevant: slices.Contains(query.RelevantIDs, id)})
	}
	slices.SortFunc(result.Matches, func(a, b searchMatch) int {
		if a.Score > b.Score {
			return -1
		}
		if a.Score < b.Score {
			return 1
		}
		if a.Path < b.Path {
			return -1
		}
		if a.Path > b.Path {
			return 1
		}
		return 0
	})
	result.Matches = result.Matches[:min(30, len(result.Matches))]
	result.RankSeconds = time.Since(start).Seconds()
	if query.RelevantIDs != nil {
		relevant := 0
		for _, match := range result.Matches[:min(10, len(result.Matches))] {
			if slices.Contains(query.RelevantIDs, match.ID) {
				relevant++
			}
		}
		precision := float64(relevant) / 10
		result.PrecisionAt10 = &precision
	}
	return result, nil
}

func searchVectorNorm(vector []float32) float64 {
	if len(vector) != 768 {
		return 0
	}
	var squared float64
	for _, value := range vector {
		if math.IsNaN(float64(value)) || math.IsInf(float64(value), 0) {
			return 0
		}
		squared += float64(value) * float64(value)
	}
	return math.Sqrt(squared)
}

func searchMedian(values []float64) *float64 {
	if len(values) == 0 {
		return nil
	}
	ordered := slices.Sorted(slices.Values(values))
	value := ordered[len(ordered)/2]
	if len(ordered)%2 == 0 {
		value = (value + ordered[len(ordered)/2-1]) / 2
	}
	return &value
}

func searchPercentile(values []float64, percentile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	ordered := slices.Sorted(slices.Values(values))
	return ordered[max(0, int(math.Ceil(float64(len(ordered))*percentile))-1)]
}
