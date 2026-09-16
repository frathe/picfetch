package ui

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"slices"
	"strings"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	"fyne.io/fyne/v2/lang"
	"fyne.io/fyne/v2/storage"
	fynetest "fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

type observedCompareLoad struct {
	uri    fyne.URI
	loaded *imaging.LoadedImage
	err    error
}

func prepareRealComparison(t *testing.T, uris ...fyne.URI) *viewer {
	t.Helper()

	v := newTestViewer(t)
	dropAndWait(t, v, uris...)
	warmThumbs(t, v)
	v.grid.Toggle()
	v.grid.SelectAll()
	if got := v.grid.Selection(); len(got) != 2 {
		t.Fatalf("setup comparison selection = %v, want exactly two files", got)
	}
	return v
}

func observeRealCompareLoads(v *viewer) <-chan observedCompareLoad {
	observed := make(chan observedCompareLoad, 2)
	v.compareLoad = func(ctx context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		loaded, err := v.loadComparedImage(ctx, uri)
		observed <- observedCompareLoad{uri: uri, loaded: loaded, err: err}
		return loaded, err
	}
	return observed
}

func collectCompareLoads(t *testing.T, observed <-chan observedCompareLoad) map[string]observedCompareLoad {
	t.Helper()

	loads := make(map[string]observedCompareLoad, 2)
	for range 2 {
		select {
		case result := <-observed:
			loads[result.uri.Name()] = result
		case <-time.After(testTimeout):
			t.Fatal("timed out collecting comparison loader results")
		}
	}
	return loads
}

func requireSuccessfulCompareLoad(t *testing.T, loads map[string]observedCompareLoad, name string) *imaging.LoadedImage {
	t.Helper()

	result, ok := loads[name]
	if !ok {
		t.Fatalf("comparison loader did not receive %q", name)
	}
	if result.err != nil {
		t.Fatalf("comparison loader for %q: %v", name, result.err)
	}
	if result.loaded == nil {
		t.Fatalf("comparison loader for %q returned nil", name)
	}
	return result.loaded
}

func comparisonImageHolding(t *testing.T, v *viewer, frame image.Image) *canvas.Shader {
	t.Helper()

	for _, candidate := range comparisonShaders(v.compare.Overlay()) {
		if candidate.Textures["overview"] == frame {
			return candidate
		}
	}
	t.Fatal("comparison no longer displays the expected decoded frame")
	return nil
}

func TestCompareRAW_UsesEmbeddedPreviewFromCanonicalLoader(t *testing.T) {
	reference := storage.NewFileURI(uitest.WriteTempFile(t, "a-reference.png",
		uitest.EncodePNG(t, 19, 13, color.White)))
	raw := storage.NewFileURI(uitest.WriteTempFile(t, "b-photo.cr2",
		uitest.EncodeRAWPreview(t, uitest.RAWPreview{
			Width: 37, Height: 23, Color: color.RGBA{G: 255, A: 255},
		})))
	v := prepareRealComparison(t, reference, raw)
	v.imgCache.Purge()
	observed := observeRealCompareLoads(v)

	fireCompareShortcut(v)
	waitForCompare(t, v)
	loads := collectCompareLoads(t, observed)
	loaded := requireSuccessfulCompareLoad(t, loads, "b-photo.cr2")
	if !loaded.Preview {
		t.Fatal("RAW comparison load did not report an extracted embedded preview")
	}
	frame := loaded.Frames[0]
	if got, want := frame.Bounds(), image.Rect(0, 0, 37, 23); got != want {
		t.Fatalf("RAW comparison frame bounds = %v, want embedded preview bounds %v", got, want)
	}
	r, g, b, _ := frame.At(18, 11).RGBA()
	if g <= r || g <= b {
		t.Errorf("RAW preview center = R:%d G:%d B:%d, want green preview pixels", r, g, b)
	}
	comparisonImageHolding(t, v, frame)

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})
	fynetest.Tap(comparisonButton(t, v.compare.Overlay(), lang.L("Swipe")))
	fynetest.Tap(comparisonButton(t, v.compare.Overlay(), lang.L("Swap")))
	comparisonImageHolding(t, v, frame)
}

func TestCompareAnimated_FreezesFirstDecodedFrameForEntireSession(t *testing.T) {
	reference := storage.NewFileURI(uitest.WriteTempFile(t, "a-reference.png",
		uitest.EncodePNG(t, 18, 12, color.White)))
	animated := storage.NewFileURI(uitest.WriteTempFile(t, "b-motion.gif",
		uitest.EncodeAnimatedGIF(t, 24, 16,
			[]color.Color{
				color.RGBA{R: 255, A: 255},
				color.RGBA{B: 255, A: 255},
			},
			[]int{1, 1})))
	v := prepareRealComparison(t, reference, animated)
	v.imgCache.Purge()
	observed := observeRealCompareLoads(v)

	fireCompareShortcut(v)
	waitForCompare(t, v)
	loads := collectCompareLoads(t, observed)
	loaded := requireSuccessfulCompareLoad(t, loads, "b-motion.gif")
	if got := len(loaded.Frames); got != 1 {
		t.Fatalf("animated comparison decoded frames = %d, want only the displayed first frame", got)
	}
	first := loaded.Frames[0]
	r, _, b, _ := first.At(12, 8).RGBA()
	if r <= b {
		t.Fatalf("first animation frame = R:%d B:%d, want red", r, b)
	}

	comparisonImageHolding(t, v, first)
	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})
	comparisonImageHolding(t, v, first)
	fynetest.Tap(comparisonButton(t, v.compare.Overlay(), lang.L("Swipe")))
	v.win.Resize(fyne.NewSize(900, 620))
	comparisonImageHolding(t, v, first)
	fynetest.Tap(comparisonButton(t, v.compare.Overlay(), lang.L("Swap")))
	comparisonImageHolding(t, v, first)
}

func TestCompareOrientation_UsesCanonicalEXIFPixelsAndIgnoresViewerRotation(t *testing.T) {
	oriented := storage.NewFileURI(uitest.WriteTempFile(t, "a-oriented.jpg",
		uitest.EncodeOrientedJPEG(t, 40, 20, 6)))
	reference := storage.NewFileURI(uitest.WriteTempFile(t, "b-reference.png",
		uitest.EncodePNG(t, 18, 18, color.White)))
	v := prepareRealComparison(t, oriented, reference)
	if got := v.FileAt(v.CurrentIndex()).Name(); got != "a-oriented.jpg" {
		t.Fatalf("setup current file = %q, want oriented source", got)
	}
	v.rotateBy(1)
	if got := v.display.Rotation(); got != 1 {
		t.Fatalf("setup viewer rotation = %d, want one view-only turn", got)
	}
	if got, want := v.img.Image.Bounds(), image.Rect(0, 0, 40, 20); got != want {
		t.Fatalf("setup viewer-only rotated bounds = %v, want %v", got, want)
	}
	v.imgCache.Purge()
	observed := observeRealCompareLoads(v)

	fireCompareShortcut(v)
	waitForCompare(t, v)
	loads := collectCompareLoads(t, observed)
	loaded := requireSuccessfulCompareLoad(t, loads, "a-oriented.jpg")
	frame := loaded.Frames[0]
	if got, want := frame.Bounds(), image.Rect(0, 0, 20, 40); got != want {
		t.Fatalf("canonical EXIF-corrected bounds = %v, want %v", got, want)
	}
	topR, _, topB, _ := frame.At(10, 10).RGBA()
	bottomR, _, bottomB, _ := frame.At(10, 30).RGBA()
	if topR <= topB || bottomB <= bottomR {
		t.Errorf("canonical pixels top R:B=%d:%d bottom R:B=%d:%d, want red above blue", topR, topB, bottomR, bottomB)
	}
	comparisonImageHolding(t, v, frame)
	if got := v.display.Rotation(); got != 1 {
		t.Errorf("comparison changed covered viewer rotation to %d, want 1", got)
	}

	v.handleKeyEvent(&fyne.KeyEvent{Name: fyne.KeyPlus})
	fynetest.Tap(comparisonButton(t, v.compare.Overlay(), lang.L("Swap")))
	comparisonImageHolding(t, v, frame)
}

func TestCompareMemory_RejectsSourcesWhoseCombinedEstimateExceedsBudget(t *testing.T) {
	left := storage.NewFileURI(uitest.WriteTempFile(t, "a-wide.png",
		uitest.EncodePNG(t, 401, 211, color.RGBA{R: 255, A: 255})))
	right := storage.NewFileURI(uitest.WriteTempFile(t, "b-tall.png",
		uitest.EncodePNG(t, 233, 377, color.RGBA{B: 255, A: 255})))
	v := prepareRealComparison(t, left, right)
	v.imgCache.SetBudget(1)
	v.imgCache.Purge()

	observed := observeRealCompareLoads(v)

	fireCompareShortcut(v)
	waitForCompare(t, v)
	loads := collectCompareLoads(t, observed)
	budgetRefusals := 0
	for name, result := range loads {
		if result.loaded != nil || result.err == nil {
			t.Errorf("comparison loader for %q = (%v, %v), want memory-budget refusal", name, result.loaded, result.err)
			continue
		}
		if errors.Is(result.err, errComparisonMemoryBudget) {
			budgetRefusals++
		} else if !errors.Is(result.err, context.Canceled) {
			t.Errorf("comparison loader for %q error = %v, want memory-budget refusal or peer cancellation", name, result.err)
		}
	}
	if budgetRefusals == 0 {
		t.Fatal("comparison did not reject either oversized source against its shared budget")
	}
	if got := v.imgCache.Len(); got != 0 {
		t.Errorf("refused comparison retained %d cache entries, want 0", got)
	}
	if got, want := v.toast.text.Text, lang.L("Selected images exceed the comparison memory budget"); got != want {
		t.Errorf("comparison refusal toast = %q, want %q", got, want)
	}
	settleToast(t, v)
}

func TestCompareMemory_LoadAdmission(t *testing.T) {
	frame := image.NewNRGBA64(image.Rect(0, 0, 3, 2))
	frame.SetNRGBA64(0, 0, color.NRGBA64{R: 0x1234, A: 0x8001})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, frame); err != nil {
		t.Fatal(err)
	}
	uri := storage.NewFileURI(uitest.WriteTempFile(t, "six-pixels.png", encoded.Bytes()))
	const frameBytes = 3 * 2 * 8
	for _, cached := range []bool{false, true} {
		for _, tc := range []struct {
			name   string
			budget int64
			fits   bool
		}{
			{"below_limit", 2*frameBytes - 1, false},
			{"at_limit", 2 * frameBytes, true},
		} {
			name := "decode/" + tc.name
			if cached {
				name = "cached/" + tc.name
			}
			t.Run(name, func(t *testing.T) {
				v := newTestViewer(t)
				v.imgCache.SetBudget(tc.budget)
				if cached {
					v.imgCache.Add(uri.String(), &imaging.LoadedImage{Frames: []image.Image{frame}})
				}
				loaded, err := v.loadComparedImage(context.Background(), uri)
				if !tc.fits {
					if loaded != nil || !errors.Is(err, errComparisonMemoryBudget) {
						t.Fatalf("load = (%v, %v), want refusal", loaded, err)
					}
					if !cached && v.imgCache.Len() != 0 {
						t.Fatal("refused decode populated the cache")
					}
					return
				}
				if err != nil {
					t.Fatal(err)
				}
				if got := loaded.DecodedBytes(); got != frameBytes {
					t.Fatalf("retained bytes = %d, want %d for 16-bit pixels", got, frameBytes)
				}
			})
		}
	}

	t.Run("cached_animation_charges_only_first_frame", func(t *testing.T) {
		v := newTestViewer(t)
		first := image.NewRGBA(image.Rect(0, 0, 3, 2))
		complete := &imaging.LoadedImage{
			Frames: []image.Image{first, image.NewRGBA(first.Bounds()), image.NewRGBA(first.Bounds())},
			Delays: []time.Duration{time.Second, time.Second, time.Second},
		}
		v.imgCache.SetBudget(2 * int64(len(first.Pix)))
		v.imgCache.Add(uri.String(), complete)
		loaded, err := v.loadComparedImage(context.Background(), uri)
		if err != nil {
			t.Fatal(err)
		}
		if loaded == complete || len(loaded.Frames) != 1 || loaded.Frames[0] != first || len(loaded.Delays) != 0 {
			t.Fatal("comparison must retain a detached first-frame record")
		}
		if record, ok := v.imgCache.Get(uri.String()); !ok || record != complete || len(record.Frames) != 3 || len(record.Delays) != 3 {
			t.Fatal("comparison changed the complete cached animation")
		}
	})
}

func TestCompareInputLimit_FailsWithoutRemovingEitherSelectedSource(t *testing.T) {
	smallData := uitest.EncodePNG(t, 4, 4, color.White)
	largeData := uitest.EncodePNG(t, 256, 256, color.RGBA{R: 73, G: 121, B: 199, A: 255})
	if len(largeData) <= len(smallData) {
		t.Fatalf("fixture encoded sizes small=%d large=%d, want large source bigger", len(smallData), len(largeData))
	}
	small := storage.NewFileURI(uitest.WriteTempFile(t, "a-small.png", smallData))
	large := storage.NewFileURI(uitest.WriteTempFile(t, "b-large.png", largeData))
	v := prepareRealComparison(t, small, large)
	beforeSelection := append([]int(nil), v.grid.Selection()...)
	v.imgCache.Purge()
	limit := int64((len(smallData) + len(largeData)) / 2)
	imaging.SetMaxEncodedBytes(limit)
	t.Cleanup(func() { imaging.SetMaxEncodedBytes(0) })
	observed := observeRealCompareLoads(v)

	fireCompareShortcut(v)
	waitForCompare(t, v)
	loads := collectCompareLoads(t, observed)
	largeResult, ok := loads["b-large.png"]
	if !ok {
		t.Fatal("comparison loader did not receive the over-limit source")
	}
	if _, ok := errors.AsType[*imaging.InputTooLargeError](largeResult.err); !ok {
		t.Fatalf("large comparison load error = %v, want *imaging.InputTooLargeError", largeResult.err)
	}
	if v.compare.Visible() || !v.grid.Visible() {
		t.Fatal("input-limit failure should close comparison and reveal Grid View")
	}
	if got := v.grid.Selection(); !slices.Equal(got, beforeSelection) {
		t.Errorf("selection after input-limit failure = %v, want %v", got, beforeSelection)
	}
	if v.FileCount() != 2 || v.FileAt(0).Name() != "a-small.png" || v.FileAt(1).Name() != "b-large.png" {
		t.Errorf("file set after input-limit failure = %v, want both original files", v.state.files)
	}
	if !strings.Contains(v.toast.text.Text, "input limit") {
		t.Errorf("input-limit failure toast = %q, want encoded-size reason", v.toast.text.Text)
	}
	settleToast(t, v)
}
