package locationmap

import (
	"math"
	"slices"
)

const mercatorLatitudeLimit = 85.05112878

// WorldPoint is a normalized Web Mercator location. X wraps at one world.
type WorldPoint struct {
	X, Y float64
}

// Project converts a valid geographic coordinate to normalized Web Mercator.
func Project(lat, lon float64) (WorldPoint, bool) {
	if !finite(lat) || !finite(lon) || lat < -90 || lat > 90 || lon < -180 || lon > 180 {
		return WorldPoint{}, false
	}
	lat = math.Max(-mercatorLatitudeLimit, math.Min(mercatorLatitudeLimit, lat))
	x := (lon + 180) / 360
	if x == 1 {
		x = 0
	}
	radians := lat * math.Pi / 180
	y := (1 - math.Asinh(math.Tan(radians))/math.Pi) / 2
	y = math.Max(0, math.Min(1, y))
	return WorldPoint{X: x, Y: y}, true
}

func finite(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

// Camera holds the world point at the viewport center and pixels per world.
type Camera struct {
	X, Y, Scale float64
}

// FitCamera fits valid points into the padded viewport on the shortest world arc.
func FitCamera(points []WorldPoint, width, height, padding float64) Camera {
	defaultCamera := Camera{X: .5, Y: .5, Scale: 256}
	if !finite(width) || !finite(height) || !finite(padding) || width <= 0 || height <= 0 || padding < 0 {
		return defaultCamera
	}

	xs := make([]float64, 0, len(points))
	minY, maxY := 1., 0.
	for _, point := range points {
		if !validWorldPoint(point) {
			continue
		}
		xs = append(xs, point.X)
		minY = math.Min(minY, point.Y)
		maxY = math.Max(maxY, point.Y)
	}
	if len(xs) == 0 {
		return defaultCamera
	}
	slices.Sort(xs)
	largestGap, start := -1., 0
	for i, x := range xs {
		next := xs[(i+1)%len(xs)]
		if i == len(xs)-1 {
			next++
		}
		if gap := next - x; gap > largestGap {
			largestGap, start = gap, (i+1)%len(xs)
		}
	}
	spanX := 1 - largestGap
	centerX := math.Mod(xs[start]+spanX/2, 1)
	spanY := maxY - minY
	scale := 256. * 16384
	if spanX > 0 {
		scale = math.Min(scale, math.Max(0, width-2*padding)/spanX)
	}
	if spanY > 0 {
		scale = math.Min(scale, math.Max(0, height-2*padding)/spanY)
	}
	return Camera{X: centerX, Y: (minY + maxY) / 2, Scale: math.Max(256, scale)}
}

func validWorldPoint(point WorldPoint) bool {
	return finite(point.X) && finite(point.Y) && point.X >= 0 && point.X < 1 && point.Y >= 0 && point.Y <= 1
}

// Cluster is one leader's screen position and the input indexes it represents.
type Cluster struct {
	Members []int
	X, Y    float64
}

type clusterCell struct {
	X, Y float64
}

// Clusters groups visible points around stable leaders in input order.
func Clusters(points []WorldPoint, camera Camera, width, height, diameter float64) []Cluster {
	return groupClusters(points, camera, width, height, diameter, true)
}

// NavigationClusters groups all projected points, including those outside the viewport.
func NavigationClusters(points []WorldPoint, camera Camera, width, height, diameter float64) []Cluster {
	return groupClusters(points, camera, width, height, diameter, false)
}

func groupClusters(points []WorldPoint, camera Camera, width, height, diameter float64, visibleOnly bool) []Cluster {
	if !finite(camera.X) || !finite(camera.Y) || !finite(camera.Scale) || camera.Scale <= 0 ||
		!finite(width) || !finite(height) || width <= 0 || height <= 0 ||
		!finite(diameter) || diameter <= 0 {
		return nil
	}
	centerX := math.Mod(camera.X, 1)
	if centerX < 0 {
		centerX++
	}
	leaders := make(map[clusterCell]int)
	var clusters []Cluster
	for index, point := range points {
		if !validWorldPoint(point) {
			continue
		}
		deltaX := math.Remainder(point.X-centerX, 1)
		x := width/2 + deltaX*camera.Scale
		y := height/2 + (point.Y-camera.Y)*camera.Scale
		if !finite(x) || !finite(y) || (visibleOnly &&
			(x < -diameter || x > width+diameter || y < -diameter || y > height+diameter)) {
			continue
		}
		cell := clusterCell{X: clusterCellCoordinate(x, diameter), Y: clusterCellCoordinate(y, diameter)}
		leader := -1
		for offsetX := -1; offsetX <= 1; offsetX++ {
			for offsetY := -1; offsetY <= 1; offsetY++ {
				candidate, exists := leaders[clusterCell{cell.X + float64(offsetX), cell.Y + float64(offsetY)}]
				if !exists || (leader >= 0 && candidate >= leader) {
					continue
				}
				if math.Abs(x-clusters[candidate].X) < diameter && math.Abs(y-clusters[candidate].Y) < diameter {
					leader = candidate
				}
			}
		}
		if leader >= 0 {
			clusters[leader].Members = append(clusters[leader].Members, index)
			continue
		}
		leaders[cell] = len(clusters)
		clusters = append(clusters, Cluster{Members: []int{index}, X: x, Y: y})
	}
	return clusters
}

func clusterCellCoordinate(value, diameter float64) float64 {
	cell := math.Floor(value / diameter)
	if !finite(cell) {
		return value
	}
	return cell
}

// NearestCluster returns the closest cluster in a cardinal screen direction.
func NearestCluster(clusters []Cluster, selected int, width, height float64, dx, dy int) int {
	validSelection := selected >= 0 && selected < len(clusters) && validCluster(clusters[selected])
	if !finite(width) || !finite(height) || width <= 0 || height <= 0 ||
		!((dx == -1 || dx == 1) && dy == 0 || (dy == -1 || dy == 1) && dx == 0) {
		if validSelection {
			return selected
		}
		return -1
	}
	if !validSelection {
		best := -1
		bestDistance := math.Inf(1)
		for index, cluster := range clusters {
			if !validCluster(cluster) {
				continue
			}
			if distance := math.Hypot(cluster.X-width/2, cluster.Y-height/2); best == -1 || distance < bestDistance {
				best, bestDistance = index, distance
			}
		}
		return best
	}
	best := selected
	bestDistance := math.Inf(1)
	current := clusters[selected]
	for index, cluster := range clusters {
		if index == selected || !validCluster(cluster) {
			continue
		}
		offsetX := cluster.X - current.X
		offsetY := cluster.Y - current.Y
		if dx != 0 && float64(dx)*offsetX <= 0 || dy != 0 && float64(dy)*offsetY <= 0 {
			continue
		}
		if distance := math.Hypot(offsetX, offsetY); best == selected || distance < bestDistance {
			best, bestDistance = index, distance
		}
	}
	return best
}

func validCluster(cluster Cluster) bool {
	return len(cluster.Members) > 0 && finite(cluster.X) && finite(cluster.Y)
}
