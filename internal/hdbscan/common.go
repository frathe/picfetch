package hdbscan

import (
	"errors"
	"math"
)

// dataDims returns the number of dimensions shared by all data points, and reports
// errRaggedData when they differ so that every point is indexed with its own width.
func dataDims(data [][]float64) (int, error) {
	if len(data) == 0 {
		return 0, errEmptySet
	}

	dims := len(data[0])

	for i := 1; i < len(data); i++ {
		if len(data[i]) != dims {
			return 0, errRaggedData
		}
	}

	return dims, nil
}

type DistFunc func([]float64, []float64) float64

// EuclideanDist measures the ordinary Euclidean distance between two vectors.
var EuclideanDist = func(a, b []float64) float64 {

	if len(a) != len(b) {
		return math.NaN()
	}

	var (
		s, t float64
	)

	for i := range a {
		t = a[i] - b[i]
		s += t * t
	}

	return math.Sqrt(s)
}

var errEmptySet = errors.New("empty training set")

var errRaggedData = errors.New("data points must have the same number of dimensions")

var errZeroMinpts = errors.New("minpts cannot be 0")

var errZeroWorkers = errors.New("number of workers cannot be less than 0")
