package similarity

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/frathe/picfetch/internal/heic"
	"github.com/frathe/picfetch/internal/imaging"
)

// Finite and retained native scenarios intentionally own independent lifetimes.
//
//goland:noinspection DuplicatedCode
func TestHEICAnalysisWorker(t *testing.T) {
	t.Run("native_finite", func(t *testing.T) {
		if os.Getenv("PICFETCH_HEIC_NATIVE_TEST") != "1" {
			t.Skip("native qualification requires PICFETCH_HEIC_NATIVE_TEST=1 and installed similarity assets")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()
		backend := heic.NewClient("")
		defer func() { backend.Stop(); backend.Wait() }()
		ctx = heic.WithSnapshot(ctx, heic.Snapshot{Backend: backend, Available: true, Generation: 17})
		names := []string{"probe8.heic", "probe10.heic", "nonfirst-primary.heic", "container-rotate.heic", "exif-rotate.heic", "container-and-exif.heic"}
		paths := make([]string, len(names))
		for i, name := range names {
			path, err := filepath.Abs(filepath.Join("..", "heic", "testdata", name))
			if err != nil {
				t.Fatal(err)
			}
			paths[i] = path
		}
		metadataSource, err := os.ReadFile(paths[4])
		if err != nil {
			t.Fatal(err)
		}
		// Replace the authored fixture's one inline orientation tag with an
		// equally sized, independently expected ASCII camera-make tag.
		tiff := bytes.Index(metadataSource, []byte{'I', 'I', 42, 0})
		if tiff < 0 || binary.LittleEndian.Uint16(metadataSource[tiff+10:]) != 0x112 {
			t.Fatal("authored EXIF orientation fixture changed")
		}
		copy(metadataSource[tiff+10:tiff+22], []byte{0x0f, 0x01, 2, 0, 4, 0, 0, 0, 'M', 'I', 'T', 0})
		metadataPath := filepath.Join(t.TempDir(), "camera-make.heic")
		if err := os.WriteFile(metadataPath, metadataSource, 0600); err != nil {
			t.Fatal(err)
		}
		names, paths = append(names, "camera-make.heic"), append(paths, metadataPath)
		unassociated := bytes.Clone(metadataSource)
		reference := bytes.Index(unassociated, []byte("cdsc"))
		if reference < 0 {
			t.Fatal("authored metadata association changed")
		}
		copy(unassociated[reference:reference+4], "free")
		unassociatedPath := filepath.Join(t.TempDir(), "unassociated-make.heic")
		if err := os.WriteFile(unassociatedPath, unassociated, 0600); err != nil {
			t.Fatal(err)
		}
		names, paths = append(names, "unassociated-make.heic"), append(paths, unassociatedPath)
		broken := filepath.Join(t.TempDir(), "broken.heic")
		if err := os.WriteFile(broken, []byte("not an image"), 0600); err != nil {
			t.Fatal(err)
		}
		paths = append(paths, broken, filepath.Join(t.TempDir(), "missing.heic"))
		client := Client{GeneralAnalysisDir: t.TempDir()}
		var fresh Event
		if err := client.Analyze(ctx, paths, nil, func(event Event) {
			if event.Complete {
				fresh = event
			}
		}); err != nil {
			t.Fatal(err)
		}
		if fresh.Successful != len(names) || fresh.Failed != 2 || fresh.Reused != 0 || fresh.Measurements.InferenceAttempts != len(names) || len(fresh.Items) != len(paths) {
			t.Fatalf("native analysis did not produce all valid HEIC sources: successful=%d failed=%d reused=%d items=%+v", fresh.Successful, fresh.Failed, fresh.Reused, fresh.Items)
		}
		for i, name := range names {
			item := fresh.Items[i]
			if item.Error != "" || item.Facts.Version != FactsVersion || item.Facts.Width != 64 || item.Facts.Height != 64 || item.Facts.Format != "heic" {
				t.Fatalf("%s lost native pixels or facts: error=%s embedding=%d facts=%+v", name, item.Error, len(item.Embedding), item.Facts)
			}
			if name == "camera-make.heic" && item.Facts.Make != "MIT" {
				t.Fatalf("HEIC metadata facts were dropped: %+v", item.Facts)
			}
			if name == "unassociated-make.heic" && item.Facts.Make != "" {
				t.Fatalf("HEIC facts used metadata not associated with the primary: %+v", item.Facts)
			}
			preview, err := jpeg.Decode(bytes.NewReader(item.Preview))
			if err != nil {
				t.Fatal(err)
			}
			assertHEICPreview(t, name, preview)
		}
		if !fresh.OfflineVerified && EnforcesNetworkIsolation() {
			t.Fatal("native analysis bypassed worker isolation")
		}
		// A pre-fix macOS cache has valid pixels and facts version 1, but
		// empty camera metadata. Refresh its facts without repeating inference.
		legacyPath := filepath.Join(client.GeneralAnalysisDir, "v1", filepath.Base(analysisName(metadataPath)))
		legacyData, err := os.ReadFile(legacyPath)
		if err != nil {
			t.Fatal(err)
		}
		var legacy cachedRepresentation
		if err := json.Unmarshal(legacyData, &legacy); err != nil {
			t.Fatal(err)
		}
		legacy.Item.Facts.Version = 1
		legacy.Item.Facts.Make = ""
		legacyData, err = json.Marshal(legacy)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(legacyPath, legacyData, 0600); err != nil {
			t.Fatal(err)
		}
		var warm Event
		if err := client.Analyze(ctx, paths, nil, func(event Event) {
			if event.Complete {
				warm = event
			}
		}); err != nil {
			t.Fatal(err)
		}
		if warm.Successful != len(names) || warm.Failed != 2 || warm.Reused != len(names) || warm.Measurements.InferenceAttempts != 0 {
			t.Fatalf("native cache reuse: successful=%d failed=%d reused=%d attempts=%d", warm.Successful, warm.Failed, warm.Reused, warm.Measurements.InferenceAttempts)
		}
		for i := range names {
			if warm.Items[i].Facts != fresh.Items[i].Facts || !bytes.Equal(warm.Items[i].Preview, fresh.Items[i].Preview) {
				t.Fatalf("cache changed canonical HEIC facts or pixels for %s", names[i])
			}
		}
		repairedData, err := os.ReadFile(legacyPath)
		if err != nil {
			t.Fatal(err)
		}
		repaired, err := decodeRepresentation(bytes.NewReader(repairedData))
		if err != nil || repaired.Facts != fresh.Items[6].Facts {
			t.Fatalf("repaired metadata was not persisted: %+v, %v", repaired.Facts, err)
		}
	})
	t.Run("native_retained_search", testHEICRetainedSearch)
	t.Run("native_limits", testHEICAnalysisLimits)
}

func testHEICAnalysisLimits(t *testing.T) {
	if os.Getenv("PICFETCH_HEIC_NATIVE_TEST") != "1" {
		t.Skip("native qualification requires PICFETCH_HEIC_NATIVE_TEST=1 and installed similarity assets")
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	heicPath, err := filepath.Abs(filepath.Join("..", "heic", "testdata", "probe8.heic"))
	if err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(heicPath)
	if err != nil {
		t.Fatal(err)
	}
	var pixels bytes.Buffer
	if err := png.Encode(&pixels, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	pngPath := filepath.Join(t.TempDir(), "small.png")
	if err := os.WriteFile(pngPath, pixels.Bytes(), 0600); err != nil {
		t.Fatal(err)
	}
	for _, limit := range []struct {
		name          string
		bytes, pixels int64
		successful    int
	}{
		{"normal", imaging.MaxEncodedBytes(), 4096, 2},
		{"pixels", imaging.MaxEncodedBytes(), 4095, 1},
		{"bytes", info.Size() - 1, 4096, 1},
	} {
		t.Run(limit.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
			defer cancel()
			req := request{Assets: defaultAssets(executable), Paths: []string{heicPath, pngPath}, MaxEncodedBytes: limit.bytes, MaxPixels: limit.pixels, HEIC: workerHEIC{Available: true, Generation: 17}}
			cmd := workerCommand(ctx, executable)
			cmd.Env = append(os.Environ(), workerEnvironment+"=1")
			var result Event
			if err := analyzeCommand(ctx, cmd, req, nil, func(event Event) {
				if event.Complete {
					result = event
				}
			}); err != nil {
				t.Fatal(err)
			}
			if result.Successful != limit.successful || result.Failed != 2-limit.successful || len(result.Items) != 2 || result.Items[1].Error != "" {
				t.Fatalf("captured %s budget failed: successful=%d failed=%d items=%+v", limit.name, result.Successful, result.Failed, result.Items)
			}
			if limit.successful == 1 && result.Items[0].Error == "" {
				t.Fatal("HEIC escaped captured resource budget")
			}
		})
	}
}

//goland:noinspection DuplicatedCode
func testHEICRetainedSearch(t *testing.T) {
	if os.Getenv("PICFETCH_HEIC_NATIVE_TEST") != "1" {
		t.Skip("native qualification requires PICFETCH_HEIC_NATIVE_TEST=1 and installed similarity assets")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	backend := heic.NewClient("")
	defer func() { backend.Stop(); backend.Wait() }()
	ctx = heic.WithSnapshot(ctx, heic.Snapshot{Backend: backend, Available: true, Generation: 17})
	root := t.TempDir()
	names := []string{"probe8.heic", "probe8-copy.heic", "nonfirst-primary.heic", "broken.heic", "missing.heic"}
	paths := make([]string, len(names))
	for i, name := range names {
		paths[i] = filepath.Join(root, name)
		if name == "missing.heic" {
			continue
		}
		data := []byte("not an image")
		if name != "broken.heic" {
			fixture := name
			if name == "probe8-copy.heic" {
				fixture = "probe8.heic"
			}
			var err error
			data, err = os.ReadFile(filepath.Join("..", "heic", "testdata", fixture))
			if err != nil {
				t.Fatal(err)
			}
		}
		if err := os.WriteFile(paths[i], data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	cache := t.TempDir()
	policy := CachePolicy{Roots: CacheRoots{GeneralDir: cache}, LooseEnabled: true}
	for _, warm := range []bool{false, true} {
		queries := make(chan SearchQuery, 1)
		queries <- SearchQuery{ID: 1, ReferencePath: paths[0]}
		var finals, ready int
		err := (Client{}).Search(ctx, SearchRequest{SessionID: 17, Paths: paths, Cache: policy}, queries, func(event SearchEvent) {
			if !event.OfflineVerified && EnforcesNetworkIsolation() {
				t.Error("search bypassed native isolation")
			}
			if event.Kind == SearchQueryFailure || event.Kind == SearchFailure {
				t.Errorf("valid HEIC reference failed: %s", event.Error)
			}
			if event.Kind == SearchFinal {
				finals++
				wantReused := 0
				if warm {
					wantReused = 3
				}
				if event.Processed != 5 || event.Failed != 2 || event.Reused != wantReused || len(event.Matches) != 2 {
					t.Errorf("native search lost valid sources or reused poisoned entries: %+v", event)
				}
				if event.QueryID == 1 && (len(event.Matches) == 0 || event.Matches[0].Path != paths[1] || event.Matches[0].Score < 0.99999) {
					t.Errorf("identical native pixels did not rank as the exact match: %+v", event.Matches)
				}
				if event.QueryID == 2 {
					close(queries)
				}
			}
			if event.Kind == SearchReady {
				ready++
				if ready == 1 {
					queries <- SearchQuery{ID: 2, ReferencePath: paths[2]}
				}
			}
		})
		if err != nil || finals != 2 || ready < 1 {
			t.Fatalf("native retained search warm=%t: finals=%d ready=%d error=%v", warm, finals, ready, err)
		}
	}
	// The finite consumer reuses the representations written by search. Its
	// previews expose the canonical pixels that were actually sent to inference.
	var result Event
	if err := (Client{GeneralAnalysisDir: cache}).Analyze(ctx, paths[:3], nil, func(event Event) {
		if event.Complete {
			result = event
		}
	}); err != nil {
		t.Fatal(err)
	}
	if result.Reused != 3 || result.Successful != 3 || result.Measurements.InferenceAttempts != 0 {
		t.Fatalf("search representations were not reusable by analysis: %+v", result)
	}
	for i, item := range result.Items {
		if item.Facts.Width != 64 || item.Facts.Height != 64 || item.Facts.Format != "heic" {
			t.Fatalf("search lost facts: %+v", item.Facts)
		}
		preview, err := jpeg.Decode(bytes.NewReader(item.Preview))
		if err != nil {
			t.Fatal(err)
		}
		assertHEICPreview(t, names[i], preview)
	}
}

func assertHEICPreview(t *testing.T, name string, preview image.Image) {
	t.Helper()
	if preview.Bounds() != image.Rect(0, 0, 64, 64) {
		t.Fatalf("%s used a thumbnail: %v", name, preview.Bounds())
	}
	points := [2]image.Point{{8, 32}, {56, 32}}
	whiteFirst := false
	switch name {
	case "nonfirst-primary.heic":
		whiteFirst = true
	case "container-rotate.heic", "container-and-exif.heic":
		points, whiteFirst = [2]image.Point{{32, 8}, {32, 56}}, true
	case "exif-rotate.heic":
		points = [2]image.Point{{32, 8}, {32, 56}}
	}
	for i, point := range points {
		r, g, b, _ := preview.At(point.X, point.Y).RGBA()
		white := i == 0 && whiteFirst || i == 1 && !whiteFirst
		for _, component := range []uint32{r, g, b} {
			if white && component < 245*257 || !white && component > 10*257 {
				t.Fatalf("%s has incorrect primary/orientation at %v: %v,%v,%v", name, point, r/257, g/257, b/257)
			}
		}
	}
}
