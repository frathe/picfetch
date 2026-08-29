package imaging

import (
	"image"
	"runtime"
	"testing"
)

func BenchmarkOrientationTransforms(b *testing.B) {
	src := markedImage(640, 480)
	tests := []struct {
		name      string
		transform func(image.Image) image.Image
	}{
		{"flip-horizontal", flipH},
		{"flip-vertical", flipV},
		{"rotate-180", rotate180},
		{"rotate-90-clockwise", rotate90CW},
		{"rotate-270-clockwise", rotate270CW},
	}

	for _, test := range tests {
		b.Run(test.name, func(b *testing.B) {
			b.ReportAllocs()
			var got image.Image
			for b.Loop() {
				got = test.transform(src)
			}
			runtime.KeepAlive(got)
		})
	}
}
