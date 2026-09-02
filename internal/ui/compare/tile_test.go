package compare

import (
	"context"
	"image"
	"image/color"
	"reflect"
	"testing"

	"fyne.io/fyne/v2"
)

func plannerScene(width, height int, viewport fyne.Size, position fyne.Position, display fyne.Size, pixels image.Point) paneScene {
	frame := image.NewRGBA(image.Rect(0, 0, width, height))
	overview := image.Image(frame)
	if width > overviewMaxDimension || height > overviewMaxDimension {
		scale := float64(overviewMaxDimension) / float64(max(width, height))
		overview = image.NewRGBA(image.Rect(0, 0,
			max(1, int(float64(width)*scale+0.5)),
			max(1, int(float64(height)*scale+0.5)),
		))
	}
	return paneScene{
		source:        &renderSource{frame: frame, overview: overview},
		viewport:      viewport,
		imagePosition: position,
		imageSize:     display,
		displaySize:   pixels,
	}
}

func visibleRequests(plan tilePlan) []tileRequest {
	requests := make([]tileRequest, 0, len(plan.requests))
	for _, request := range plan.requests {
		if request.visible {
			requests = append(requests, request)
		}
	}
	return requests
}

func TestTilePlanner_SelectsDensityAndCoarsensToSamplerBudget(t *testing.T) {
	tests := []struct {
		name        string
		scene       paneScene
		wantLevel   int
		wantVisible int
	}{
		{
			name:        "overview already meets display density",
			scene:       plannerScene(4096, 2048, fyne.NewSize(800, 400), fyne.Position{}, fyne.NewSize(800, 400), image.Pt(800, 400)),
			wantLevel:   0,
			wantVisible: 0,
		},
		{
			name:        "two-x display retains another mip",
			scene:       plannerScene(4096, 2048, fyne.NewSize(800, 400), fyne.Position{}, fyne.NewSize(800, 400), image.Pt(1600, 800)),
			wantLevel:   1,
			wantVisible: 6,
		},
		{
			name:        "sampler budget forces coarser level",
			scene:       plannerScene(10000, 10000, fyne.NewSize(1000, 1000), fyne.Position{}, fyne.NewSize(1000, 1000), image.Pt(10000, 10000)),
			wantLevel:   3,
			wantVisible: 4,
		},
		{
			name:        "zoomed viewport keeps level zero",
			scene:       plannerScene(10000, 10000, fyne.NewSize(1000, 1000), fyne.NewPos(-4500, -4500), fyne.NewSize(10000, 10000), image.Pt(10000, 10000)),
			wantLevel:   0,
			wantVisible: 4,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			plan := planTiles(tc.scene)
			if plan.level != tc.wantLevel {
				t.Errorf("level = %d, want %d", plan.level, tc.wantLevel)
			}
			if got := len(visibleRequests(plan)); got != tc.wantVisible {
				t.Errorf("visible requests = %d, want %d (%v)", got, tc.wantVisible, plan.requests)
			}
			if len(plan.requests) > detailSamplerCount {
				t.Errorf("requests = %d, exceeds %d detail samplers", len(plan.requests), detailSamplerCount)
			}
		})
	}
}

func TestTilePlanner_SkipsDuplicateDetailsForOriginalSizeOverview(t *testing.T) {
	scene := plannerScene(
		800, 400,
		fyne.NewSize(400, 200),
		fyne.NewPos(-200, -100),
		fyne.NewSize(1600, 800),
		image.Pt(3200, 1600),
	)
	if plan := planTiles(scene); len(plan.requests) != 0 {
		t.Fatalf("original-size overview planned duplicate details: %#v", plan.requests)
	}
}

func TestTilePlanner_UsesSwipeRevealInsteadOfHiddenFullPane(t *testing.T) {
	scene := plannerScene(
		10000, 10000,
		fyne.NewSize(4000, 4000),
		fyne.Position{},
		fyne.NewSize(10000, 10000),
		image.Pt(10000, 10000),
	)
	full := planTiles(scene)
	if full.level == 0 {
		t.Fatal("full pane fixture did not require a coarser mip")
	}

	scene.revealSet = true
	scene.revealPosition = fyne.Position{}
	scene.revealSize = fyne.NewSize(500, 500)
	revealed := planTiles(scene)
	if revealed.level != 0 {
		t.Errorf("500px swipe reveal level = %d, want sharp level 0", revealed.level)
	}
	if got := len(visibleRequests(revealed)); got != 1 {
		t.Errorf("visible requests through swipe reveal = %d, want 1", got)
	}
}

func TestTilePlanner_PrefetchIsBoundedNearestAndDeterministic(t *testing.T) {
	scene := plannerScene(
		10000, 10000,
		fyne.NewSize(500, 500),
		fyne.NewPos(-4750, -4750),
		fyne.NewSize(10000, 10000),
		image.Pt(10000, 10000),
	)
	first := planTiles(scene)
	second := planTiles(scene)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("identical scenes produced different plans\nfirst:  %#v\nsecond: %#v", first, second)
	}
	if len(first.requests) != detailSamplerCount {
		t.Fatalf("requests = %d, want all %d slots filled", len(first.requests), detailSamplerCount)
	}
	seen := make(map[tileKey]bool)
	for _, request := range first.requests {
		if seen[request.key] {
			t.Errorf("duplicate request for tile %+v", request.key)
		}
		seen[request.key] = true
		if request.key.level != 0 {
			t.Errorf("prefetch changed level to %d, want 0", request.key.level)
		}
	}
	if got := len(visibleRequests(first)); got != 4 {
		t.Errorf("visible requests = %d, want 4", got)
	}
}

func TestGenerateRenderTile_LevelZeroHasExactPixelsAndRealGutters(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 1024, 2))
	for y := range 2 {
		for x := range 1024 {
			frame.SetRGBA(x, y, color.RGBA{R: uint8(x), G: uint8(x >> 8), B: uint8(y), A: 255})
		}
	}
	source, err := prepareRenderSource(context.Background(), frame)
	if err != nil {
		t.Fatalf("prepareRenderSource: %v", err)
	}

	left, err := generateRenderTile(context.Background(), source, tileKey{level: 0, x: 0, y: 0})
	if err != nil {
		t.Fatalf("generate left tile: %v", err)
	}
	if got := left.texture.Bounds().Size(); got != image.Pt(1024, 4) {
		t.Fatalf("left texture size = %v, want 1024x4", got)
	}
	assertPixel := func(label string, got color.Color, x, y int) {
		t.Helper()
		if gotRGBA, want := color.RGBAModel.Convert(got), color.RGBAModel.Convert(frame.At(x, y)); gotRGBA != want {
			t.Errorf("%s = %v, want source(%d,%d) %v", label, gotRGBA, x, y, want)
		}
	}
	assertPixel("left outer gutter", left.texture.At(0, 1), 0, 0)
	assertPixel("left first interior", left.texture.At(1, 1), 0, 0)
	assertPixel("left last interior", left.texture.At(1022, 1), 1021, 0)
	assertPixel("left neighboring gutter", left.texture.At(1023, 1), 1022, 0)

	right, err := generateRenderTile(context.Background(), source, tileKey{level: 0, x: 1, y: 0})
	if err != nil {
		t.Fatalf("generate right tile: %v", err)
	}
	if got := right.texture.Bounds().Size(); got != image.Pt(4, 4) {
		t.Fatalf("right texture size = %v, want 4x4", got)
	}
	assertPixel("right neighboring gutter", right.texture.At(0, 1), 1021, 0)
	assertPixel("right first interior", right.texture.At(1, 1), 1022, 0)
	assertPixel("right last interior", right.texture.At(2, 1), 1023, 0)
	assertPixel("right outer gutter", right.texture.At(3, 1), 1023, 0)
}

func TestGenerateRenderTile_OddCoarseEdgeKeepsNeighborGuttersContinuous(t *testing.T) {
	frame := image.NewRGBA(image.Rect(0, 0, 2045, 3))
	for y := range 3 {
		for x := range 2045 {
			frame.SetRGBA(x, y, color.RGBA{
				R: uint8((x * 37) % 251),
				G: uint8((x*x + y*19) % 253),
				B: uint8((x*11 + y*47) % 255),
				A: 255,
			})
		}
	}
	source, err := prepareRenderSource(context.Background(), frame)
	if err != nil {
		t.Fatalf("prepareRenderSource: %v", err)
	}

	left, err := generateRenderTile(context.Background(), source, tileKey{level: 1, x: 0, y: 0})
	if err != nil {
		t.Fatalf("generate left coarse tile: %v", err)
	}
	right, err := generateRenderTile(context.Background(), source, tileKey{level: 1, x: 1, y: 0})
	if err != nil {
		t.Fatalf("generate right coarse tile: %v", err)
	}
	for y := 1; y < left.texture.Bounds().Dy()-1; y++ {
		if got, want := left.texture.RGBAAt(left.texture.Bounds().Dx()-2, y), right.texture.RGBAAt(0, y); got != want {
			t.Errorf("left interior/right gutter mismatch at row %d: %v != %v", y, got, want)
		}
		if got, want := left.texture.RGBAAt(left.texture.Bounds().Dx()-1, y), right.texture.RGBAAt(1, y); got != want {
			t.Errorf("left gutter/right interior mismatch at row %d: %v != %v", y, got, want)
		}
	}
}

func TestRenderSourceTileCache_IsByteBoundedAndReturnsHits(t *testing.T) {
	source, err := prepareRenderSource(context.Background(), image.NewRGBA(image.Rect(0, 0, 1, 1)))
	if err != nil {
		t.Fatalf("prepareRenderSource: %v", err)
	}
	first := &renderTile{key: tileKey{}, texture: image.NewRGBA(image.Rect(0, 0, 1024, 1024))}
	source.tiles.Add(first.key.cacheKey(), first)
	if got, ok := source.tiles.Get(first.key.cacheKey()); !ok || got != first {
		t.Fatal("tile cache did not return the exact cached tile")
	}
	for i := 1; i < 17; i++ {
		key := tileKey{x: i}
		source.tiles.Add(key.cacheKey(), &renderTile{key: key, texture: image.NewRGBA(image.Rect(0, 0, 1024, 1024))})
	}
	if got := source.tiles.Bytes(); got > tileCacheBudgetBytes {
		t.Errorf("tile cache bytes = %d, exceeds budget %d", got, tileCacheBudgetBytes)
	}
	if got := source.tiles.Len(); got != 16 {
		t.Errorf("tile cache entries = %d, want 16 full-size textures", got)
	}
}
