package similarity

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/nozzle/umap"

	"github.com/frathe/picfetch/internal/hdbscan"
)

func Group(ctx context.Context, items []Item, durations map[string]float64) ([]CohortMerge, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var vectors [][]float32
	var indexes []int
	byPath := map[string]int{}
	for i, entry := range items {
		if entry.Error == "" {
			if _, exists := byPath[entry.Path]; exists {
				continue
			}
			byPath[entry.Path] = len(vectors)
			vectors = append(vectors, entry.Embedding)
			indexes = append(indexes, i)
		}
	}
	if len(vectors) == 0 {
		return nil, fmt.Errorf("no images were represented successfully")
	}
	config := umap.DefaultConfig()
	config.Metric, config.Init, config.Seed, config.NumWorkers = "cosine", "random", 42, 1
	config.NNeighbors, config.NEpochs = 15, 300
	config.NComponents = 15
	start := time.Now()
	reduced := umap.New(config).FitTransform(vectors)
	durations["reduction_seconds"] = time.Since(start).Seconds()
	if err := validateCoordinates(reduced, len(vectors), 15); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	start = time.Now()
	groups, err := clusterCoordinates(ctx, reduced)
	durations["hdbscan_seconds"] = time.Since(start).Seconds()
	if err != nil {
		return nil, err
	}
	if len(groups) != len(vectors) {
		return nil, fmt.Errorf("clustering lost input identities")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	config.NComponents = 2
	start = time.Now()
	positions := umap.New(config).FitTransform(vectors)
	durations["projection_seconds"] = time.Since(start).Seconds()
	if err := validateCoordinates(positions, len(vectors), 2); err != nil {
		return nil, err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	members := map[int][]string{}
	for i, label := range groups {
		members[label] = append(members[label], items[indexes[i]].Path)
	}
	ids := map[int]string{-1: "unassigned"}
	for label, paths := range members {
		if label == -1 {
			continue
		}
		sort.Strings(paths)
		ids[label] = fmt.Sprintf("cohort-%x", sha256.Sum256([]byte(strings.Join(paths, "\n"))))[:19]
	}
	for index, entry := range items {
		if entry.Error != "" {
			continue
		}
		i := byPath[entry.Path]
		items[index].Cohort = ids[groups[i]]
		items[index].Position = positions[i]
	}
	centroids := map[string][]float64{}
	for i, label := range groups {
		if label == -1 {
			continue
		}
		key := ids[label]
		if centroids[key] == nil {
			centroids[key] = make([]float64, 15)
		}
		for axis, value := range reduced[i] {
			centroids[key][axis] += float64(value) / float64(len(members[label]))
		}
	}
	start = time.Now()
	merges, err := cohortHierarchy(ctx, centroids)
	durations["hierarchy_seconds"] = time.Since(start).Seconds()
	return merges, err
}

func clusterCoordinates(ctx context.Context, points [][]float32) ([]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if len(points) == 0 {
		return nil, fmt.Errorf("clustering requires reduced coordinates")
	}
	if err := validateCoordinates(points, len(points), 15); err != nil {
		return nil, err
	}
	precise := make([][]float64, len(points))
	for i, point := range points {
		precise[i] = make([]float64, len(point))
		for d, value := range point {
			precise[i][d] = float64(value)
		}
	}
	// Density uses two other neighbors. This implementation counts the point
	// itself, so minPts=3 preserves that setting. Four points form a cohort.
	// One worker keeps ordering deterministic; the owning subprocess provides
	// cancellation while the synchronous clustering pass is running.
	const minimumCohortSize = 4
	hierarchy, err := hdbscan.HDBSCAN(precise, 3, minimumCohortSize, 1, hdbscan.EuclideanDist)
	if err != nil {
		return nil, fmt.Errorf("cluster reduced coordinates: %w", err)
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	labels := hierarchy.Labels()
	// Upstream excludes the root cluster. Preserve Explorer's single-cohort
	// behavior when no smaller cluster qualifies and the root is large enough.
	// Keep noise assignments when the hierarchy does select smaller clusters.
	if labels.Count() == 0 && len(labels) >= minimumCohortSize {
		for i := range labels {
			labels[i] = 1
		}
	}
	return labels, nil
}

func validateCoordinates(points [][]float32, count, dimensions int) error {
	if len(points) != count {
		return fmt.Errorf("projection lost inputs: %d of %d", len(points), count)
	}
	for _, point := range points {
		if len(point) != dimensions {
			return fmt.Errorf("projection dimension %d, want %d", len(point), dimensions)
		}
		for _, v := range point {
			if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
				return fmt.Errorf("non-finite projection")
			}
		}
	}
	return nil
}
