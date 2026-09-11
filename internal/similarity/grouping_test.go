package similarity

import (
	"context"
	"errors"
	"math"
	"reflect"
	"testing"
)

func TestClusterCoordinates(t *testing.T) {
	point := func(x float32) []float32 {
		p := make([]float32, 15)
		p[0] = x
		return p
	}
	t.Run("single_group", func(t *testing.T) {
		for _, n := range []int{4, 6} {
			points := make([][]float32, n)
			for i := range points {
				points[i] = point(float32(i) * .01)
			}
			labels, err := clusterCoordinates(context.Background(), points)
			if err != nil || len(labels) != n {
				t.Fatalf("single group lost inputs: %v %v", labels, err)
			}
			for _, label := range labels {
				if label < 0 || label != labels[0] {
					t.Fatalf("single compact group was discarded: %v", labels)
				}
			}
		}
	})
	t.Run("separated_groups_and_noise", func(t *testing.T) {
		points := [][]float32{point(0), point(.01), point(.02), point(.03), point(10), point(10.01), point(10.02), point(10.03), point(1000)}
		labels, err := clusterCoordinates(context.Background(), points)
		if err != nil || len(labels) != len(points) {
			t.Fatalf("clustering lost inputs: %v %v", labels, err)
		}
		if labels[0] < 0 || labels[4] < 0 || labels[0] == labels[4] || labels[8] != -1 {
			t.Fatalf("expected two distinct groups and isolated noise: %v", labels)
		}
		for i := range 8 {
			if labels[i] != labels[(i/4)*4] {
				t.Fatalf("split a compact group: %v", labels)
			}
		}
		again, err := clusterCoordinates(context.Background(), points)
		if err != nil || !reflect.DeepEqual(labels, again) {
			t.Fatalf("nondeterministic groups: %v %v", again, err)
		}
	})
	t.Run("too_small", func(t *testing.T) {
		for n := 1; n < 4; n++ {
			points := make([][]float32, n)
			for i := range points {
				points[i] = point(float32(i))
			}
			labels, err := clusterCoordinates(context.Background(), points)
			if err != nil || len(labels) != n {
				t.Fatalf("small input: %v %v", labels, err)
			}
			for _, label := range labels {
				if label != -1 {
					t.Fatalf("undersized cohort: %v", labels)
				}
			}
		}
	})
	t.Run("identical_groups", func(t *testing.T) {
		points := [][]float32{point(0), point(0), point(0), point(0), point(10), point(10), point(10), point(10)}
		labels, err := clusterCoordinates(context.Background(), points)
		if err != nil || len(labels) != 8 || labels[0] < 0 || labels[4] < 0 || labels[0] == labels[4] {
			t.Fatalf("identical points lost their groups: %v %v", labels, err)
		}
		for i := range points {
			if labels[i] != labels[(i/4)*4] {
				t.Fatalf("split coincident points: %v", labels)
			}
		}
	})
	t.Run("invalid", func(t *testing.T) {
		for _, points := range [][][]float32{nil, {{}}, {point(0), {1}}, {point(float32(math.NaN()))}, {point(float32(math.Inf(1)))}} {
			if _, err := clusterCoordinates(context.Background(), points); err == nil {
				t.Fatal("invalid reduced coordinates accepted")
			}
		}
	})
	t.Run("cancelled", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel()
		if _, err := clusterCoordinates(ctx, [][]float32{point(0)}); !errors.Is(err, context.Canceled) {
			t.Fatalf("cancelled clustering: %v", err)
		}
	})
}
