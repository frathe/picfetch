package consolehelp

import (
	"math"
	"strings"
)

type vector struct{ x, y, z float64 }
type ellipsoid struct {
	center, radius vector
	albedo         float64
}

// trane is an original analytic sculpture based on the retained mascot artwork.
// All details are surfaces in the same 3D space, including the eyes and tongue.
func trane() []ellipsoid {
	return []ellipsoid{
		{vector{0, 0.15, 0}, vector{0.92, 1.02, 0.78}, 0.88},     // skull
		{vector{0, -0.65, -0.12}, vector{0.64, 0.58, 0.57}, 0.8}, // neck ruff
		{vector{0, -0.4, 0.55}, vector{0.68, 0.67, 0.5}, 0.95},   // jaw
		{vector{0, 0.13, 0.86}, vector{0.32, 0.30, 0.36}, 0.9},   // nose bridge
		{vector{0, -0.45, 0.99}, vector{0.44, 0.46, 0.13}, 0.1},  // open smile
		{vector{0, -0.69, 1.12}, vector{0.24, 0.3, 0.1}, 0.58},   // tongue
		{vector{0, -0.67, 1.219}, vector{0.018, 0.17, 0.007}, 0.22},
		{vector{0, 0.04, 1.26}, vector{0.3, 0.2, 0.2}, 0.12},
		{vector{-0.09, 0.12, 1.43}, vector{0.085, 0.035, 0.014}, 0.92},
		{vector{-0.19, 0.99, -0.05}, vector{0.38, 0.23, 0.38}, 0.9},
		{vector{-0.91, -0.05, -0.04}, vector{0.4, 0.89, 0.4}, 0.68},
		{vector{-1.05, -0.49, 0.02}, vector{0.34, 0.46, 0.34}, 0.72},
		{vector{-0.42, 0.43, 0.64}, vector{0.3, 0.37, 0.22}, 0.54},
		{vector{-0.42, 0.43, 0.78}, vector{0.235, 0.29, 0.12}, 1.0},
		{vector{-0.4, 0.42, 0.875}, vector{0.145, 0.2, 0.067}, 0.13},
		{vector{-0.445, 0.51, 0.933}, vector{0.052, 0.066, 0.02}, 1.0},
		{vector{-0.42, 0.76, 0.66}, vector{0.29, 0.1, 0.16}, 0.76},
		{vector{-0.27, -0.17, 0.95}, vector{0.39, 0.3, 0.34}, 1.0},
		{vector{0.91, -0.05, -0.04}, vector{0.4, 0.89, 0.4}, 0.68},
		{vector{1.05, -0.49, 0.02}, vector{0.34, 0.46, 0.34}, 0.72},
		{vector{0.42, 0.43, 0.64}, vector{0.3, 0.37, 0.22}, 0.54},
		{vector{0.42, 0.43, 0.78}, vector{0.235, 0.29, 0.12}, 1.0},
		{vector{0.4, 0.42, 0.875}, vector{0.145, 0.2, 0.067}, 0.13},
		{vector{0.355, 0.51, 0.933}, vector{0.052, 0.066, 0.02}, 1.0},
		{vector{0.42, 0.76, 0.66}, vector{0.29, 0.1, 0.16}, 0.76},
		{vector{0.27, -0.17, 0.95}, vector{0.39, 0.3, 0.34}, 1.0},
	}
}

// render uses orthographic ray/ellipsoid intersections and a depth buffer.
// Terminal cells are assumed to be twice as tall as they are wide.
func render(degrees float64, width, height int) string {
	const ramp = " .,:;irsXA253hMHGS#9B&@"
	angle := math.Mod(degrees, 360) * math.Pi / 180
	sine, cosine := math.Sincos(angle)
	scale := math.Min(float64(width)/3.5, float64(height)*2/2.9)
	pixels := []byte(strings.Repeat(" ", width*height))
	depths := make([]float64, len(pixels))
	for i := range depths {
		depths[i] = math.Inf(-1)
	}
	for _, shape := range trane() {
		cx, cy, cz := shape.center.x, shape.center.y, shape.center.z
		rx, ry, rz := shape.radius.x, shape.radius.y, shape.radius.z
		centerX, centerZ := cosine*cx+sine*cz, -sine*cx+cosine*cz
		extentX := math.Hypot(cosine*rx, sine*rz)
		left := max(0, int(math.Floor(float64(width)/2+(centerX-extentX)*scale)))
		right := min(width, int(math.Ceil(float64(width)/2+(centerX+extentX)*scale)))
		top := max(0, int(math.Floor(float64(height)/2-(cy+ry)*scale/2)))
		bottom := min(height, int(math.Ceil(float64(height)/2-(cy-ry)*scale/2)))
		ix, iy, iz := 1/(rx*rx), 1/(ry*ry), 1/(rz*rz)
		quadratic := sine*sine*ix + cosine*cosine*iz
		for row := top; row < bottom; row++ {
			y := (float64(height)/2-float64(row)-0.5)*2/scale - cy
			for col := left; col < right; col++ {
				x := (float64(col)+0.5-float64(width)/2)/scale - centerX
				localX, localZ := cosine*x, sine*x
				linear := -sine*localX*ix + cosine*localZ*iz
				constant := localX*localX*ix + y*y*iy + localZ*localZ*iz - 1
				discriminant := linear*linear - quadratic*constant
				if discriminant < 0 {
					continue
				}
				z := (-linear + math.Sqrt(discriminant)) / quadratic
				index := row*width + col
				if z+centerZ <= depths[index] {
					continue
				}
				depths[index] = z + centerZ
				localX -= sine * z
				localZ += cosine * z
				nx, ny, nz := localX*ix, y*iy, localZ*iz
				worldX, worldZ := cosine*nx+sine*nz, -sine*nx+cosine*nz
				length := math.Sqrt(nx*nx + ny*ny + nz*nz)
				light := max(0, (-0.45*worldX+0.65*ny+0.61*worldZ)/length)
				brightness := shape.albedo * (0.24 + 0.76*light)
				shade := max(1, min(len(ramp)-1, int(math.Round(brightness*float64(len(ramp)-1)))))
				pixels[index] = ramp[shade]
			}
		}
	}
	rows := make([]string, height)
	for row := range rows {
		rows[row] = string(pixels[row*width : (row+1)*width])
	}
	return strings.Join(rows, "\n")
}
