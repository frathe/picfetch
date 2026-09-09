package similarity

import (
	"context"
	"crypto/sha256"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/alDuncanson/latent/projection"
	"github.com/nozzle/umap"
)

func Group(ctx context.Context, items []Item, durations map[string]float64) error {
	if err := ctx.Err(); err != nil {
		return err
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
		return fmt.Errorf("no images were represented successfully")
	}
	config := umap.DefaultConfig()
	config.Metric, config.Init, config.Seed, config.NumWorkers = "cosine", "random", 42, 1
	config.NNeighbors, config.NEpochs = 15, 300
	config.NComponents = 15
	start := time.Now()
	reduced := umap.New(config).FitTransform(vectors)
	durations["reduction_seconds"] = time.Since(start).Seconds()
	if err := validateCoordinates(reduced, len(vectors), 15); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	start = time.Now()
	groups := projection.Cluster(reduced, projection.HDBSCANConfig{MinClusterSize: 4, MinSamples: 2})
	durations["hdbscan_seconds"] = time.Since(start).Seconds()
	if len(groups.Labels) != len(vectors) {
		return fmt.Errorf("clustering lost input identities")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	config.NComponents = 2
	start = time.Now()
	positions := umap.New(config).FitTransform(vectors)
	durations["projection_seconds"] = time.Since(start).Seconds()
	if err := validateCoordinates(positions, len(vectors), 2); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	members := map[int][]string{}
	for i, label := range groups.Labels {
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
		items[index].Cohort = ids[groups.Labels[i]]
		items[index].Position = positions[i]
	}
	return nil
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
