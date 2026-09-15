package imaging

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestParseVectorLogicalSize(t *testing.T) {
	v, err := ParseVector(svgDoc(`viewBox="0 0 24 24"`))
	if err != nil {
		t.Fatalf("ParseVector: %v", err)
	}
	if got := v.Logical(); got.Dx() != 340 || got.Dy() != 340 {
		t.Fatalf("Logical = %dx%d, want 340x340", got.Dx(), got.Dy())
	}
}

func TestParseVectorRejectsNestedDefinitionReuse(t *testing.T) {
	// A small, unused definition: rejection does not require expanding it.
	data := []byte(`<svg viewBox="0 0 24 24"><defs><rect id="tile" width="1" height="1"/><use href="#tile"/></defs></svg>`)
	if _, err := ParseVector(data); err == nil {
		t.Fatal("definition containing use must be refused before SVG parsing")
	}
}

func TestSVGRejectsNonFiniteSizes(t *testing.T) {
	for _, value := range []string{"NaN", "+Inf", "-Inf"} {
		if _, ok := svgLength(value); ok {
			t.Fatalf("accepted non-finite length %q", value)
		}
		if _, _, _, _, ok := svgViewBox(value + " 0 24 24"); ok {
			t.Fatalf("accepted non-finite origin %q", value)
		}
	}
}

func TestParseVectorAllowsDirectDefinitionReuse(t *testing.T) {
	data := []byte(`<svg viewBox="0 0 24 24"><defs><rect id="tile" width="12" height="12" fill="red"/></defs><use href="#tile"/></svg>`)
	vector, err := ParseVectorContext(context.Background(), data)
	if err != nil {
		t.Fatal(err)
	}
	img, err := vector.RasterAt(24, 24)
	if err != nil {
		t.Fatal(err)
	}
	r, _, _, a := img.At(5, 5).RGBA()
	if r == 0 || a == 0 {
		t.Fatal("direct definition reuse lost its pixels")
	}
}

func TestSVGPreflightLimits(t *testing.T) {
	for name, data := range map[string]string{
		"nesting":      `<svg viewBox="0 0 24 24">` + strings.Repeat("<g>", maxSVGDepth) + strings.Repeat("</g>", maxSVGDepth) + `</svg>`,
		"reuse_work":   `<svg viewBox="0 0 24 24"><defs><rect id="tile" width="1" height="1"/></defs>` + strings.Repeat(`<use href="#tile"/>`, 400) + `</svg>`,
		"source_bytes": strings.Repeat(" ", maxSVGBytes+1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := validateSVG(context.Background(), []byte(data)); !errors.Is(err, errSVGComplexity) {
				t.Fatalf("preflight error = %v", err)
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ParseVectorContext(ctx, svgDoc(`viewBox="0 0 24 24"`)); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled SVG: %v", err)
	}
}

func TestParseVectorRecoversPercentageWidth(t *testing.T) {
	v, err := ParseVector(svgDoc(`width="100%" height="100%" viewBox="0 0 24 24"`))
	if err != nil {
		t.Fatalf("ParseVector: %v", err)
	}
	if got := v.Logical(); got.Dx() != 340 || got.Dy() != 340 {
		t.Fatalf("Logical = %dx%d, want 340x340", got.Dx(), got.Dy())
	}
}

func TestParseVectorRejectsNonSVG(t *testing.T) {
	if _, err := ParseVector([]byte(`{"hello":"world"}`)); !errors.Is(err, ErrNotSVG) {
		t.Fatalf("err = %v, want ErrNotSVG", err)
	}
}

func TestParseVectorRejectsSizelessSVG(t *testing.T) {
	if _, err := ParseVector(svgDoc("")); !errors.Is(err, ErrNoSVGSize) {
		t.Fatalf("err = %v, want ErrNoSVGSize", err)
	}
}

func TestParseVectorRejectsTruncatedSVG(t *testing.T) {
	if _, err := ParseVector([]byte(`<svg viewBox="0 0 24 24"><circle cx="12"`)); err == nil {
		t.Fatal("a truncated document must not parse")
	}
}

func TestRasterAtProducesRequestedSize(t *testing.T) {
	v, err := ParseVector(svgDoc(`viewBox="0 0 24 24"`))
	if err != nil {
		t.Fatalf("ParseVector: %v", err)
	}

	for _, size := range []int{64, 340, 1360} {
		img, err := v.RasterAt(size, size)
		if err != nil {
			t.Fatalf("RasterAt(%d): %v", size, err)
		}
		if b := img.Bounds(); b.Dx() != size || b.Dy() != size {
			t.Fatalf("RasterAt(%d) = %dx%d", size, b.Dx(), b.Dy())
		}
	}
}

func TestRasterAtActuallyDrawsSomething(t *testing.T) {
	v, err := ParseVector([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><rect width="24" height="24" fill="#ff0000"/></svg>`))
	if err != nil {
		t.Fatalf("ParseVector: %v", err)
	}

	img, err := v.RasterAt(64, 64)
	if err != nil {
		t.Fatalf("RasterAt: %v", err)
	}

	r, g, b, a := img.At(32, 32).RGBA()
	if a == 0 || r == 0 || g != 0 || b != 0 {
		t.Fatalf("centre pixel = (%d,%d,%d,%d), want opaque red", r, g, b, a)
	}
}

func TestRasterAtClampsToPixelCeiling(t *testing.T) {
	v, err := ParseVector(svgDoc(`viewBox="0 0 60000 60000"`))
	if err != nil {
		t.Fatalf("ParseVector: %v", err)
	}

	// Must neither panic nor allocate 14 GB - oksvg panics outright at this
	// size, which is why RasterAt clamps first and recovers second.
	img, err := v.RasterAt(60000, 60000)
	if err != nil {
		return // a contained failure is an acceptable outcome; a crash is not
	}
	if n := int64(img.Bounds().Dx()) * int64(img.Bounds().Dy()); n > MaxVectorRasterPixels() {
		t.Fatalf("raster is %d px, over the %d cap", n, MaxVectorRasterPixels())
	}
}

func TestRasterAtIsSafeForConcurrentUse(t *testing.T) {
	v, err := ParseVector(svgDoc(`viewBox="0 0 24 24"`))
	if err != nil {
		t.Fatalf("ParseVector: %v", err)
	}

	// Two internal/ui/vector.go rasterizeVector goroutines can land inside
	// RasterAt on the same *Vector at once: one that already passed its
	// staleness check keeps running while a fresher scale change spawns
	// another. SetTarget mutates state Draw reads; -race is the point of
	// this test.
	var wg sync.WaitGroup
	for i := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := v.RasterAt(32+i*8, 32+i*8); err != nil {
				t.Errorf("RasterAt: %v", err)
			}
		}()
	}
	wg.Wait()
}
