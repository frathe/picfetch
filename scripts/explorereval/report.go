package main

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"

	"github.com/frathe/picfetch/internal/similarity"

	"fyne.io/fyne/v2/storage"
)

type reviewImage struct {
	Name, Thumbnail, Error string
	Source                 template.URL
}
type reviewCohort struct {
	ID, Name       string
	Images, Sample []reviewImage
	X, Y           float64
}
type reviewData struct {
	Total, Represented, Failed, Unassigned int
	Seconds, FirstMap, MemoryMiB           string
	Cohorts                                []reviewCohort
	Failures                               []reviewImage
	MapJSON                                template.JS
}

func writeReview(result evaluation, initial []item) error {
	bindingVersion, nativeVersion := reportRuntimeVersions(runtime.GOOS, runtime.GOARCH)
	view := reviewData{Total: len(result.Items), Seconds: fmt.Sprintf("%.2f", result.ElapsedSeconds), FirstMap: fmt.Sprintf("%.2f", result.FirstMapSeconds), MemoryMiB: fmt.Sprintf("%.1f", float64(result.PeakRSSBytes)/(1<<20))}
	groups := map[string]*reviewCohort{}
	for _, entry := range result.Items {
		picture := reviewImage{Name: filepath.Base(entry.Path), Thumbnail: entry.Thumbnail, Error: entry.Error, Source: template.URL(storage.NewFileURI(entry.Path).String())}
		if entry.Error != "" {
			view.Failures = append(view.Failures, picture)
			view.Failed++
			continue
		}
		view.Represented++
		if entry.Cohort == "unassigned" {
			view.Unassigned++
		}
		group := groups[entry.Cohort]
		if group == nil {
			group = &reviewCohort{ID: entry.Cohort}
			groups[entry.Cohort] = group
		}
		group.Images = append(group.Images, picture)
		group.X += float64(entry.Position[0])
		group.Y += float64(entry.Position[1])
	}
	keys := make([]string, 0, len(groups))
	for key := range groups {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		group := groups[key]
		group.Name = fmt.Sprintf("Cohort %d", len(view.Cohorts)+1)
		if key == "unassigned" {
			group.Name = "Unassigned"
		}
		group.X /= float64(len(group.Images))
		group.Y /= float64(len(group.Images))
		group.Sample = group.Images[:min(15, len(group.Images))]
		view.Cohorts = append(view.Cohorts, *group)
	}
	mapData, err := json.Marshal(view.Cohorts)
	if err != nil {
		return err
	}
	view.MapJSON = template.JS(mapData) // json.Marshal escapes HTML delimiters in source names.
	t, err := template.New("review").Parse(reviewHTML)
	if err != nil {
		return err
	}
	f, err := os.OpenFile(filepath.Join(result.Config.Out, "review.html"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	err = t.Execute(f, view)
	closeErr := f.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	metrics := compareMaps(initial, result.Items)
	report := fmt.Sprintf(`# Local pipeline evaluation

Technical execution complete. Semantic verdict: **pending user review**.
Open [the local cohort report](review.html). No source image was inspected by a remote model.

## Measured run

- Input: %d available, %d selected, %d represented, %d failed.
- Final non-noise cohorts: %d; unassigned: %d (%.1f%% of represented images).
- First map: %.3f seconds; measured elapsed before report generation: %.3f seconds.
- Runtime/model setup and scan: %.3f seconds; image read/decode: %.3f seconds; inference: %.3f seconds.
- Worker peak RSS: %d bytes (%.1f MiB). %s. OS high-water mark, no sampling interval.
- Initial stages: %v.
- Final stages: %v.
- Initial-map inputs: %d. Mean best Jaccard overlap of initial non-noise cohorts with final cohorts: %s.
- Common represented inputs: %d. Standardized position RMS change after fitting rotation/reflection: %s.
- Manifest digest: %s.

## Reproducibility

Go ONNX Runtime binding %s; native ONNX Runtime %s; requested provider %s.
CPU uses six intra-op threads. CoreML, if selected, may fall back per operator;
the requested provider is not evidence of GPU/ANE execution.
SigLIP 2 base patch16 224 vision export revision %s, float32 pooler_output.
Full oriented first-frame RGB pixels, white alpha background, Go bilinear resize
to 224x224, rescale/normalize to [-1,1], L2-normalize 768 outputs.
The pinned processor specifies bilinear interpolation; byte-for-byte parity
with Pillow preprocessing has not been established.
Model, processor and runtime hashes are pinned in internal/similarity/assets.sha256.

Grouping: nozzle/umap f6085fb2514d, 15 dimensions, cosine metric, 15 neighbors,
300 epochs, random initialization, seed 42, one worker, other defaults.
HDBSCAN: the MIT-licensed PhotoPrism pkg/vector/alg subset at commit
c48d23f6b03c25fc19d376d789fac56c32a26fdb, retained in internal/hdbscan,
minimum cohort 4, minPts=3 (self plus two neighbors), one worker, Euclidean
on the 15D representation. The adapter retains the root as one cohort when no
smaller cluster qualifies and at least four distinct sources remain.
Map: a separate fresh 2D UMAP fit on the original
768D representations with the same remaining parameters. Every publication
refits all admitted inputs; it does not use the broken Transform API.
Membership-derived cohort identifiers remove arbitrary numeric label changes.
Inputs use canonical PicFetch scanning and decoding. The fixed seed and
folder/format round-robin sample are recorded by manifest order and source hashes.
TCP connect and empty UDP send both require OS EPERM/EACCES before processing.
The worker and descendants run under macOS sandbox-exec deny network*.

## Limits and remaining acceptance

This bounded smoke corpus does not qualify the intended 50,000-image library.
The current HDBSCAN implementation is quadratic; it is not a selected full-library
engine. Library reference parity, useful depicted-content neighborhoods,
cross-media correspondence and grouping granularity need evaluation. No semantic
success is inferred from finite coordinates or the number of cohorts.
The HTML is a local review artifact; native PicFetch map/Grid integration,
progressive visual continuity, cache reuse and interaction latency remain pending.
Inspect the initial/final JSON and cohort report, identify useful and misleading
groups, and provide a quality verdict before ticket 03 is marked ready.
`, result.Available, view.Total, view.Represented, view.Failed, len(groups)-boolInt(view.Unassigned > 0), view.Unassigned, 100*float64(view.Unassigned)/float64(view.Represented),
		result.FirstMapSeconds, result.ElapsedSeconds, result.SetupSeconds, result.DecodeSeconds, result.EncodeSeconds, result.PeakRSSBytes, float64(result.PeakRSSBytes)/(1<<20), result.MemoryScope, result.InitialStages, result.FinalStages, len(initial), metrics.Jaccard, metrics.Common, metrics.Movement, result.ManifestSHA256, bindingVersion, nativeVersion, result.Config.Provider, similarity.ModelRevision)
	return os.WriteFile(filepath.Join(result.Config.Out, "pipeline-evaluation.md"), []byte(report), 0o600)
}

func reportRuntimeVersions(goos, goarch string) (string, string) {
	if goos == "darwin" && goarch == "amd64" {
		return "v1.25.0", "1.23.2"
	}
	return "v1.36.0", "1.29.0"
}

func boolInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

type mapComparison struct {
	Common            int
	Jaccard, Movement string
}

func compareMaps(initial, final []item) mapComparison {
	before, after := map[string]map[string]bool{}, map[string]map[string]bool{}
	finalByPath := map[string]item{}
	for _, entry := range final {
		if entry.Error == "" {
			finalByPath[entry.Path] = entry
			if entry.Cohort != "unassigned" {
				if after[entry.Cohort] == nil {
					after[entry.Cohort] = map[string]bool{}
				}
				after[entry.Cohort][entry.Path] = true
			}
		}
	}
	var a, b [][2]float64
	for _, entry := range initial {
		if entry.Error != "" {
			continue
		}
		if entry.Cohort != "unassigned" {
			if before[entry.Cohort] == nil {
				before[entry.Cohort] = map[string]bool{}
			}
			before[entry.Cohort][entry.Path] = true
		}
		if newer, ok := finalByPath[entry.Path]; ok {
			a = append(a, [2]float64{float64(entry.Position[0]), float64(entry.Position[1])})
			b = append(b, [2]float64{float64(newer.Position[0]), float64(newer.Position[1])})
		}
	}
	comparison := mapComparison{Common: len(a), Jaccard: "not available (no initial non-noise cohorts)", Movement: "not available"}
	var sum float64
	for _, old := range before {
		best := 0.0
		for _, newer := range after {
			intersection := 0
			for path := range old {
				if newer[path] {
					intersection++
				}
			}
			score := float64(intersection) / float64(len(old)+len(newer)-intersection)
			best = math.Max(best, score)
		}
		sum += best
	}
	if len(before) > 0 {
		comparison.Jaccard = fmt.Sprintf("%.4f", sum/float64(len(before)))
	}
	if len(a) < 2 {
		return comparison
	}
	for _, points := range [][][2]float64{a, b} {
		var x, y float64
		for _, p := range points {
			x += p[0]
			y += p[1]
		}
		x /= float64(len(points))
		y /= float64(len(points))
		var norm float64
		for i := range points {
			points[i][0] -= x
			points[i][1] -= y
			norm += points[i][0]*points[i][0] + points[i][1]*points[i][1]
		}
		norm = math.Sqrt(norm / float64(len(points)))
		if norm == 0 {
			return comparison
		}
		for i := range points {
			points[i][0] /= norm
			points[i][1] /= norm
		}
	}
	best := math.Inf(1)
	for _, reflection := range []float64{1, -1} {
		var dot, cross float64
		for i, p := range a {
			x, y := b[i][0], b[i][1]*reflection
			dot += x*p[0] + y*p[1]
			cross += x*p[1] - y*p[0]
		}
		angle := math.Atan2(cross, dot)
		c, s := math.Cos(angle), math.Sin(angle)
		var loss float64
		for i, p := range a {
			x, y := b[i][0], b[i][1]*reflection
			dx, dy := c*x-s*y-p[0], s*x+c*y-p[1]
			loss += dx*dx + dy*dy
		}
		best = math.Min(best, math.Sqrt(loss/float64(len(a))))
	}
	comparison.Movement = fmt.Sprintf("%.4f", best)
	return comparison
}

//go:embed review.html
var reviewHTML string
