package locationmap

import (
	"math"
	"reflect"
	"testing"
)

func TestProjectCoordinates(t *testing.T) {
	for _, tc := range []struct {
		name     string
		lat, lon float64
		want     WorldPoint
	}{
		{"origin", 0, 0, WorldPoint{.5, .5}},
		{"east boundary", 0, 180, WorldPoint{0, .5}},
		{"west boundary", 0, -180, WorldPoint{0, .5}},
		{"north pole", 90, 0, WorldPoint{.5, 0}},
		{"south pole", -90, 0, WorldPoint{.5, 1}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := Project(tc.lat, tc.lon)
			if !ok || math.Abs(got.X-tc.want.X) > 1e-9 || math.Abs(got.Y-tc.want.Y) > 1e-8 {
				t.Fatalf("Project(%v, %v) = (%+v, %v), want %+v", tc.lat, tc.lon, got, ok, tc.want)
			}
		})
	}
	for _, coords := range [][2]float64{
		{91, 0}, {-91, 0}, {0, 181}, {0, -181},
		{math.NaN(), 0}, {0, math.NaN()}, {math.Inf(1), 0}, {0, math.Inf(-1)},
	} {
		if _, ok := Project(coords[0], coords[1]); ok {
			t.Errorf("Project(%v, %v) accepted invalid coordinates", coords[0], coords[1])
		}
	}
	for _, lat := range []float64{-90, 90} {
		point, _ := Project(lat, 0)
		if point.Y < 0 || point.Y > 1 {
			t.Errorf("projected pole is not a valid world point: %+v", point)
		}
	}
}

func TestFitCameraUsesShortDatelineArc(t *testing.T) {
	west, _ := Project(0, -179)
	east, _ := Project(0, 179)
	points := []WorldPoint{west, east}
	before := append([]WorldPoint(nil), points...)
	camera := FitCamera(points, 1000, 600, 100)
	if math.Min(camera.X, 1-camera.X) > 1e-9 || math.Abs(camera.Y-.5) > 1e-9 {
		t.Fatalf("unexpected dateline camera center: %+v", camera)
	}
	if math.Abs(camera.Scale-144000) > 1e-6 {
		t.Fatalf("scale = %v, want 144000 for a two-degree arc", camera.Scale)
	}
	if !reflect.DeepEqual(points, before) {
		t.Fatal("FitCamera modified its points")
	}
}

func TestFitCameraEmptyAndInvalidInputs(t *testing.T) {
	want := Camera{X: .5, Y: .5, Scale: 256}
	for _, tc := range []struct {
		points        []WorldPoint
		width, height float64
		padding       float64
	}{
		{nil, 800, 600, 20},
		{[]WorldPoint{{math.NaN(), .5}}, 800, 600, 20},
		{[]WorldPoint{{.5, .5}}, math.NaN(), 600, 20},
		{[]WorldPoint{{.5, .5}}, 800, 0, 20},
		{[]WorldPoint{{.5, .5}}, 800, 600, math.Inf(1)},
	} {
		if got := FitCamera(tc.points, tc.width, tc.height, tc.padding); got != want {
			t.Errorf("FitCamera(%v, %v, %v, %v) = %+v, want %+v", tc.points, tc.width, tc.height, tc.padding, got, want)
		}
	}
	point := []WorldPoint{{.25, .75}, {2, .5}, {.5, -1}}
	if got := FitCamera(point, 800, 600, 20); got.X != .25 || got.Y != .75 || got.Scale != 256*16384 {
		t.Errorf("invalid world points changed fit: %+v", got)
	}
}

func TestClustersStableGreedyMembership(t *testing.T) {
	points := []WorldPoint{
		{.4, .5}, {.48, .5}, {.44, .5}, {.401, .501},
		{.26, .5}, {.24, .5}, {math.NaN(), .5}, {1, .5},
	}
	before := append([]WorldPoint(nil), points...)
	got := Clusters(points, Camera{X: .5, Y: .5, Scale: 1000}, 400, 300, 50)
	want := []Cluster{
		{Members: []int{0, 2, 3}, X: 100, Y: 150},
		{Members: []int{1}, X: 180, Y: 150},
		{Members: []int{4}, X: -40, Y: 150},
	}
	if len(got) != len(want) {
		t.Fatalf("Clusters() = %+v, want %+v", got, want)
	}
	for i := range want {
		if !reflect.DeepEqual(got[i].Members, want[i].Members) || math.Abs(got[i].X-want[i].X) > 1e-8 || math.Abs(got[i].Y-want[i].Y) > 1e-8 {
			t.Fatalf("cluster %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	for i := range before {
		if math.Float64bits(points[i].X) != math.Float64bits(before[i].X) || math.Float64bits(points[i].Y) != math.Float64bits(before[i].Y) {
			t.Fatal("Clusters modified its points")
		}
	}
}

func TestClustersWrapAndZoom(t *testing.T) {
	points := []WorldPoint{{.99, .5}, {.01, .5}}
	near := Clusters(points, Camera{X: 0, Y: .5, Scale: 1000}, 400, 300, 50)
	if len(near) != 1 || !reflect.DeepEqual(near[0].Members, []int{0, 1}) || math.Abs(near[0].X-190) > 1e-8 {
		t.Fatalf("near dateline clusters = %+v", near)
	}
	far := Clusters(points, Camera{X: 0, Y: .5, Scale: 4000}, 400, 300, 50)
	if len(far) != 2 || !reflect.DeepEqual(far[0].Members, []int{0}) || !reflect.DeepEqual(far[1].Members, []int{1}) {
		t.Fatalf("zoomed dateline clusters = %+v", far)
	}
}

func TestClustersStrictDiameterBoundary(t *testing.T) {
	points := []WorldPoint{{.5, .5}, {.5625, .5}, {.5, .5625}}
	got := Clusters(points, Camera{X: .5, Y: .5, Scale: 1024}, 400, 300, 64)
	if len(got) != 3 {
		t.Fatalf("points exactly one diameter apart clustered: %+v", got)
	}
	for i, cluster := range got {
		if !reflect.DeepEqual(cluster.Members, []int{i}) {
			t.Fatalf("cluster %d members = %v", i, cluster.Members)
		}
	}
}

func TestClustersCoincidentThirtyThousand(t *testing.T) {
	points := make([]WorldPoint, 30000)
	for i := range points {
		points[i] = WorldPoint{.5, .5}
	}
	got := Clusters(points, Camera{X: .5, Y: .5, Scale: 1000}, 400, 300, 50)
	if len(got) != 1 || len(got[0].Members) != len(points) || got[0].X != 200 || got[0].Y != 150 {
		t.Fatalf("coincident points formed %+v clusters", len(got))
	}
	for i, member := range got[0].Members {
		if member != i {
			t.Fatalf("member %d = %d", i, member)
		}
	}
}

func TestClustersRejectsInvalidViewport(t *testing.T) {
	point := []WorldPoint{{.5, .5}}
	for _, tc := range []struct {
		camera                  Camera
		width, height, diameter float64
	}{
		{Camera{.5, .5, 0}, 400, 300, 50},
		{Camera{.5, .5, 1000}, math.NaN(), 300, 50},
		{Camera{.5, .5, 1000}, 400, 300, 0},
	} {
		if got := Clusters(point, tc.camera, tc.width, tc.height, tc.diameter); len(got) != 0 {
			t.Errorf("invalid viewport produced clusters: %+v", got)
		}
	}
}

func TestNavigationClustersIncludesOffscreenPointsAndWrappedMembership(t *testing.T) {
	points := []WorldPoint{
		{.99, .5}, {.01, .5}, // One pin across the dateline.
		{.3, .5}, {.301, .5}, // A second pin beyond the viewport.
		{math.NaN(), .5},
	}
	camera := Camera{X: 0, Y: .5, Scale: 1000}
	visible := Clusters(points, camera, 400, 300, 50)
	if len(visible) != 1 || !reflect.DeepEqual(visible[0].Members, []int{0, 1}) {
		t.Fatalf("visible clusters = %+v, want dateline pair only", visible)
	}
	got := NavigationClusters(points, camera, 400, 300, 50)
	if len(got) != 2 || !reflect.DeepEqual(got[0].Members, []int{0, 1}) || !reflect.DeepEqual(got[1].Members, []int{2, 3}) ||
		math.Abs(got[0].X-190) > 1e-8 || math.Abs(got[1].X-500) > 1e-8 {
		t.Fatalf("navigation clusters = %+v, want wrapped pair and offscreen pair", got)
	}
}

func TestNavigationClustersKeepsDistantLeadersSeparate(t *testing.T) {
	points := []WorldPoint{{.2, .5}, {.3, .5}, {.2, .5}}
	got := NavigationClusters(points, Camera{X: .5, Y: .5, Scale: 1e20}, 400, 300, 1)
	if len(got) != 2 || !reflect.DeepEqual(got[0].Members, []int{0, 2}) || !reflect.DeepEqual(got[1].Members, []int{1}) {
		t.Fatalf("distant navigation clusters = %+v, want first and third grouped", got)
	}
}

func TestNearestClusterChoosesClosestInEachDirection(t *testing.T) {
	clusters := []Cluster{
		{Members: []int{0}, X: 100, Y: 100},
		{Members: []int{1}, X: 110, Y: 100},
		{Members: []int{2}, X: 90, Y: 100},
		{Members: []int{3}, X: 100, Y: 90},
		{Members: []int{4}, X: 100, Y: 110},
		{Members: []int{5}, X: 130, Y: 100},
	}
	for _, tc := range []struct {
		name   string
		dx, dy int
		want   int
	}{
		{"right", 1, 0, 1},
		{"left", -1, 0, 2},
		{"up", 0, -1, 3},
		{"down", 0, 1, 4},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := NearestCluster(clusters, 0, 200, 200, tc.dx, tc.dy); got != tc.want {
				t.Fatalf("NearestCluster(%s) = %d, want %d", tc.name, got, tc.want)
			}
		})
	}
}

func TestNearestClusterFirstSelectionAndInvalidInputs(t *testing.T) {
	clusters := []Cluster{
		{Members: []int{0}, X: 95, Y: 100},
		{Members: []int{1}, X: 105, Y: 100},
		{Members: []int{2}, X: 100, Y: 120},
		{Members: nil, X: 100, Y: 100},
		{Members: []int{4}, X: math.NaN(), Y: 100},
	}
	for _, tc := range []struct {
		name          string
		selected      int
		width, height float64
		dx, dy        int
		want          int
	}{
		{"first ignores direction", -1, 200, 200, 0, -1, 0},
		{"out-of-range selection", 99, 200, 200, 1, 0, 0},
		{"empty selected cluster", 3, 200, 200, 1, 0, 0},
		{"nonfinite selected cluster", 4, 200, 200, 1, 0, 0},
		{"zero direction", 0, 200, 200, 0, 0, 0},
		{"diagonal direction", 0, 200, 200, 1, 1, 0},
		{"large direction", 0, 200, 200, 2, 0, 0},
		{"invalid direction and selection", -1, 200, 200, 0, 0, -1},
		{"zero width", 0, 0, 200, 1, 0, 0},
		{"nonfinite height", 0, 200, math.Inf(1), 1, 0, 0},
		{"invalid dimensions and selection", -1, math.NaN(), 200, 1, 0, -1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := NearestCluster(clusters, tc.selected, tc.width, tc.height, tc.dx, tc.dy); got != tc.want {
				t.Fatalf("NearestCluster() = %d, want %d", got, tc.want)
			}
		})
	}
	if got := NearestCluster(nil, -1, 200, 200, 1, 0); got != -1 {
		t.Errorf("empty clusters selected %d, want -1", got)
	}
	if got := NearestCluster(clusters[3:5], -1, 200, 200, 1, 0); got != -1 {
		t.Errorf("no valid clusters selected %d, want -1", got)
	}
}

func TestNearestClusterStrictHalfPlaneDistanceAndStableTie(t *testing.T) {
	clusters := []Cluster{
		{Members: []int{0}, X: 100, Y: 100},
		{Members: []int{1}, X: 100, Y: 101}, // On the boundary: excluded for Right.
		{Members: []int{2}, X: 115, Y: 100},
		{Members: []int{3}, X: 105, Y: 101},
		{Members: []int{4}, X: 105, Y: 99}, // Equal distance to 3: input order wins.
		{Members: nil, X: 101, Y: 100},
		{Members: []int{6}, X: math.Inf(1), Y: 100},
	}
	if got := NearestCluster(clusters, 0, 200, 200, 1, 0); got != 3 {
		t.Fatalf("nearest right cluster = %d, want 3", got)
	}
	if got := NearestCluster(clusters, 0, 200, 200, -1, 0); got != 0 {
		t.Fatalf("no left cluster returned %d, want selected 0", got)
	}
	extreme := []Cluster{
		{Members: []int{0}, X: 1e308, Y: 1e308},
		{Members: []int{1}, X: 0, Y: -1e308},
	}
	if got := NearestCluster(extreme, 0, 200, 200, 1, 0); got != 0 {
		t.Fatalf("extreme off-axis distance selected %d to the right, want 0", got)
	}
}
