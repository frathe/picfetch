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

	// Warm libheif/wazero once so one-time init is not counted as per-decode growth.
	if _, err := LoadImage(storage.NewFileURI(path), DefaultImgCacheBytes); err != nil {
		t.Fatalf("warmup decode: %v", err)
	}
	settleRSS()

	const mid = 20
	for i := 0; i < mid; i++ {
		if _, err := LoadImage(storage.NewFileURI(path), DefaultImgCacheBytes); err != nil {
			t.Fatalf("decode %d: %v", i, err)
		}
	}
	settleRSS()
	rssMid, ok := readRSS()
	if !ok {
		t.Skip("RSS measurement unavailable on this platform")
	}

	const iterations = 20
	for i := 0; i < iterations; i++ {
		if _, err := LoadImage(storage.NewFileURI(path), DefaultImgCacheBytes); err != nil {
			t.Fatalf("decode %d: %v", mid+i, err)
		}
	}
	settleRSS()
	rssAfter, ok := readRSS()
	if !ok {
		t.Fatal("RSS measurement failed after decode loop")
	}

	// With the upstream leak, RSS climbs roughly linearly with decode count.
	// After PR #16 the second batch should not add much beyond wasm init noise.
	const maxGrowthMB = 80
	if growth := rssAfter - rssMid; growth > maxGrowthMB*1024*1024 {
		t.Fatalf("RSS grew %d bytes over %d decodes after warmup; want <= %d MB (leak suspected)", growth, iterations, maxGrowthMB)
	}
}

func settleRSS() {
	debug.FreeOSMemory()
	runtime.GC()
	debug.FreeOSMemory()
	runtime.GC()
}
