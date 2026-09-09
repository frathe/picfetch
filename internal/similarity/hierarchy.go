package similarity

import (
	"context"
	"math"
	"sort"
)

// cohortHierarchy cuts a minimum spanning tree of cohort centroids in the same
// 15D space used for clustering. Its ordered edges form nested broader groups;
// display coordinates and Unassigned never influence those joins.
func cohortHierarchy(ctx context.Context, centroids map[string][]float64) ([]CohortMerge, error) {
	keys := make([]string, 0, len(centroids))
	for key := range centroids {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	if len(keys) < 2 {
		return nil, ctx.Err()
	}
	used := make([]bool, len(keys))
	distances := make([]float64, len(keys))
	parents := make([]int, len(keys))
	for i := range distances {
		distances[i], parents[i] = math.Inf(1), -1
	}
	distances[0] = 0
	type edge struct {
		CohortMerge
		distance float64
	}
	edges := make([]edge, 0, len(keys)-1)
	for range keys {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		next := -1
		for i := range keys {
			if !used[i] && (next < 0 || distances[i] < distances[next]) {
				next = i
			}
		}
		used[next] = true
		if parent := parents[next]; parent >= 0 {
			left, right := keys[parent], keys[next]
			if right < left {
				left, right = right, left
			}
			edges = append(edges, edge{CohortMerge{Left: left, Right: right}, distances[next]})
		}
		for i, key := range keys {
			if used[i] {
				continue
			}
			var distance float64
			for axis, value := range centroids[key] {
				delta := value - centroids[keys[next]][axis]
				distance += delta * delta
			}
			if distance < distances[i] {
				distances[i], parents[i] = distance, next
			}
		}
	}
	sort.Slice(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		if a.distance != b.distance {
			return a.distance < b.distance
		}
		if a.Left != b.Left {
			return a.Left < b.Left
		}
		return a.Right < b.Right
	})
	merges := make([]CohortMerge, len(edges))
	for i, edge := range edges {
		merges[i] = edge.CohortMerge
	}
	return merges, nil
}
