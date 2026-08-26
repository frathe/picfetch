//go:build heicleak

package imaging

import (
	"os"
	"runtime"
	"runtime/debug"
	"testing"

	"fyne.io/fyne/v2/storage"
)

func TestHEICDecode_DoesNotGrowRSSUnbounded(t *testing.T) {
	if os.Getenv("PICFETCH_HEIC_LEAK_TEST") == "" {
		t.Skip("set PICFETCH_HEIC_LEAK_TEST=1 to run native RSS check")
	}

	data, err := os.ReadFile("testdata/test_exif.heic")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	path := writeTempFile(t, "leak.heic", data)

	rssBefore, ok := readRSS()
	if !ok {
		t.Skip("RSS measurement unavailable on this platform")
	}

	const iterations = 40
	for i := 0; i < iterations; i++ {
		debug.FreeOSMemory()
		runtime.GC()
		_, err := LoadImage(storage.NewFileURI(path), DefaultImgCacheBytes)
		if err != nil {
			t.Fatalf("decode %d: %v", i, err)
		}
	}
	debug.FreeOSMemory()
	runtime.GC()

	rssAfter, ok := readRSS()
	if !ok {
		t.Fatal("RSS measurement failed after decode loop")
	}

	const maxGrowthMB = 150 // libheif one-time init + headroom; leaky build adds hundreds of MB
	if growth := rssAfter - rssBefore; growth > maxGrowthMB*1024*1024 {
		t.Fatalf("RSS grew %d bytes over %d decodes; want <= %d MB (leak suspected)", growth, iterations, maxGrowthMB)
	}
}
