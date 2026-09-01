package compare

import (
	"context"
	"image"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/storage"
	fynetest "fyne.io/fyne/v2/test"

	"github.com/frathe/picfetch/internal/imaging"
	"github.com/frathe/picfetch/internal/uitest"
)

func asyncTileScene(source *renderSource, x float32) paneScene {
	return paneScene{
		source:        source,
		viewport:      fyne.NewSize(400, 400),
		imagePosition: fyne.NewPos(x, -824),
		imageSize:     fyne.NewSize(2048, 2048),
		displaySize:   image.Pt(2048, 2048),
	}
}

func waitForTileStart(t *testing.T, started <-chan struct{}) {
	t.Helper()
	select {
	case <-started:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for tile worker")
	}
}

func waitForRenderer(t *testing.T, renderer paneRenderer) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := renderer.Wait(ctx); err != nil {
		t.Fatalf("renderer Wait: %v", err)
	}
}

func TestShaderPaneRenderer_RapidScenesUseAtMostOneTileWorker(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	source, err := prepareRenderSource(context.Background(), image.NewRGBA(image.Rect(0, 0, 2048, 2048)))
	if err != nil {
		t.Fatalf("prepareRenderSource: %v", err)
	}
	renderer := newShaderPaneRenderer(0).(*shaderPaneRenderer)
	queue := &uitest.UIQueue{}
	renderer.queueUI = queue.Do
	started := make(chan struct{}, 1)
	release := make(chan struct{})
	var active atomic.Int64
	var maximum atomic.Int64
	var once sync.Once
	renderer.generateTile = func(ctx context.Context, source *renderSource, key tileKey) (*renderTile, error) {
		current := active.Add(1)
		defer active.Add(-1)
		for {
			seen := maximum.Load()
			if current <= seen || maximum.CompareAndSwap(seen, current) {
				break
			}
		}
		once.Do(func() { started <- struct{}{}; <-release })
		return generateRenderTile(ctx, source, key)
	}

	renderer.Present(asyncTileScene(source, -824))
	waitForTileStart(t, started)
	latest := asyncTileScene(source, -824)
	for i := range 100 {
		latest = asyncTileScene(source, -float32((i*173)%1600))
		renderer.Present(latest)
	}
	close(release)
	waitForRenderer(t, renderer)
	for queue.Drain() {
	}
	if got := maximum.Load(); got != 1 {
		t.Errorf("concurrent tile generators = %d, want exactly 1", got)
	}
	if got := active.Load(); got != 0 {
		t.Errorf("active tile generators after Wait = %d, want 0", got)
	}
	want := make(map[tileKey]bool)
	for _, request := range planTiles(latest).requests {
		want[request.key] = true
	}
	for slot, tile := range renderer.bound {
		if tile != nil && !want[tile.key] {
			t.Errorf("slot %d retained stale-view tile %+v; latest plan is %+v", slot, tile.key, want)
		}
	}
}

func TestShaderPaneRenderer_SameSourceViewChangeDoesNotCancelAllocatedTile(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	source, err := prepareRenderSource(context.Background(), image.NewRGBA(image.Rect(0, 0, 4096, 2048)))
	if err != nil {
		t.Fatalf("prepare source: %v", err)
	}
	renderer := newShaderPaneRenderer(0).(*shaderPaneRenderer)
	queue := &uitest.UIQueue{}
	renderer.queueUI = queue.Do
	first := asyncTileScene(source, 0)
	latest := asyncTileScene(source, -1600)
	latestPlan := planTiles(latest)
	if sameTileRequests(planTiles(first).requests, latestPlan.requests) {
		t.Fatal("fixture view changes produced identical tile requests")
	}

	type allocatedWork struct {
		ctx  context.Context
		key  tileKey
		tile *renderTile
		err  error
	}
	allocated := make(chan allocatedWork, 1)
	release := make(chan struct{})
	var once sync.Once
	renderer.generateTile = func(ctx context.Context, source *renderSource, key tileKey) (*renderTile, error) {
		blocked := false
		var tile *renderTile
		var generateErr error
		once.Do(func() {
			blocked = true
			tile, generateErr = generateRenderTile(context.Background(), source, key)
			allocated <- allocatedWork{ctx: ctx, key: key, tile: tile, err: generateErr}
		})
		if blocked {
			<-release
			return tile, generateErr
		}
		return generateRenderTile(ctx, source, key)
	}

	renderer.Present(first)
	var work allocatedWork
	select {
	case work = <-allocated:
	case <-time.After(3 * time.Second):
		t.Fatal("timed out waiting for allocated tile work")
	}
	if work.err != nil || work.tile == nil {
		close(release)
		waitForRenderer(t, renderer)
		t.Fatalf("allocate first tile: tile=%v err=%v", work.tile, work.err)
	}
	renderer.Present(latest)
	contextErr := work.ctx.Err()
	close(release)
	waitForRenderer(t, renderer)
	for queue.Drain() {
	}
	if contextErr != nil {
		t.Errorf("same-source view change canceled allocated tile work: %v", contextErr)
	}
	if !source.tiles.Contains(work.key.cacheKey()) {
		t.Error("allocated same-source tile was discarded instead of cached")
	}
	wanted := make(map[tileKey]bool, len(latestPlan.requests))
	for _, request := range latestPlan.requests {
		wanted[request.key] = true
		if !source.tiles.Contains(request.key.cacheKey()) {
			t.Errorf("latest-view tile %+v was not completed", request.key)
		}
	}
	bound := 0
	for slot, tile := range renderer.bound {
		if tile == nil {
			continue
		}
		bound++
		if !wanted[tile.key] {
			t.Errorf("slot %d retained tile %+v outside latest plan", slot, tile.key)
		}
	}
	if bound == 0 {
		t.Error("latest view completed without binding a detail tile")
	}
}

func TestShaderPaneRenderer_CoalescesTilePublicationWhileUIQueueIsHeld(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	source, err := prepareRenderSource(context.Background(), image.NewRGBA(image.Rect(0, 0, 4096, 4096)))
	if err != nil {
		t.Fatalf("prepare source: %v", err)
	}
	scene := paneScene{
		source:      source,
		viewport:    fyne.NewSize(1000, 1000),
		imageSize:   fyne.NewSize(4096, 4096),
		displaySize: image.Pt(4096, 4096),
	}
	if got := len(planTiles(scene).requests); got <= 1 {
		t.Fatalf("publication fixture requests %d tiles, want several", got)
	}
	renderer := newShaderPaneRenderer(0).(*shaderPaneRenderer)
	queue := &uitest.UIQueue{}
	renderer.queueUI = queue.Do
	renderer.Present(scene)
	waitForRenderer(t, renderer)
	if got := queue.Len(); got != 1 {
		t.Errorf("queued tile publications = %d, want one coalesced callback", got)
	}
	for queue.Drain() {
	}
}

func TestShaderPaneRenderer_ClearCancelsActiveTileGeneration(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	source, err := prepareRenderSource(context.Background(), image.NewRGBA(image.Rect(0, 0, 2048, 2048)))
	if err != nil {
		t.Fatalf("prepare source: %v", err)
	}
	renderer := newShaderPaneRenderer(0).(*shaderPaneRenderer)
	started := make(chan struct{})
	canceled := make(chan struct{})
	var once sync.Once
	renderer.generateTile = func(ctx context.Context, _ *renderSource, _ tileKey) (*renderTile, error) {
		once.Do(func() { close(started) })
		<-ctx.Done()
		close(canceled)
		return nil, ctx.Err()
	}

	renderer.Present(asyncTileScene(source, -824))
	waitForTileStart(t, started)
	renderer.Present(paneScene{})
	select {
	case <-canceled:
	case <-time.After(time.Second):
		t.Fatal("clearing the pane did not cancel active tile generation")
	}
	waitForRenderer(t, renderer)
	if renderer.shader.Visible() {
		t.Fatal("cleared shader remained visible")
	}
}

func TestShaderPaneRenderer_StaleSourceCannotBindAndLatestSourceCompletes(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	oldSource, err := prepareRenderSource(context.Background(), image.NewRGBA(image.Rect(0, 0, 2048, 2048)))
	if err != nil {
		t.Fatalf("prepare old source: %v", err)
	}
	newSource, err := prepareRenderSource(context.Background(), image.NewRGBA(image.Rect(0, 0, 2048, 2048)))
	if err != nil {
		t.Fatalf("prepare new source: %v", err)
	}
	renderer := newShaderPaneRenderer(0).(*shaderPaneRenderer)
	queue := &uitest.UIQueue{}
	renderer.queueUI = queue.Do
	oldStarted := make(chan struct{}, 1)
	releaseOld := make(chan struct{})
	var oldTile *renderTile
	var once sync.Once
	renderer.generateTile = func(ctx context.Context, source *renderSource, key tileKey) (*renderTile, error) {
		if source == oldSource {
			once.Do(func() { oldStarted <- struct{}{}; <-releaseOld })
			tile, generateErr := generateRenderTile(context.Background(), source, key)
			oldTile = tile
			return tile, generateErr
		}
		return generateRenderTile(ctx, source, key)
	}

	renderer.Present(asyncTileScene(oldSource, -824))
	waitForTileStart(t, oldStarted)
	renderer.Present(paneScene{})
	if renderer.shader.Visible() {
		t.Fatal("renderer remained visible between old and reopened sources")
	}
	renderer.Present(asyncTileScene(newSource, -824))
	close(releaseOld)
	waitForRenderer(t, renderer)
	for queue.Drain() {
	}
	if renderer.shader.Textures["overview"] != newSource.overview {
		t.Fatal("stale worker replaced the latest overview")
	}
	for slot, tile := range renderer.bound {
		if tile == oldTile && oldTile != nil {
			t.Errorf("stale old-source tile bound in slot %d", slot)
		}
	}
	if newSource.tiles.Len() == 0 {
		t.Fatal("latest source produced no cached detail tiles")
	}
}

func TestShaderPaneRenderer_ClearCancelsWorkAndCachedSourceIsReusable(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	source, err := prepareRenderSource(context.Background(), image.NewRGBA(image.Rect(0, 0, 2048, 2048)))
	if err != nil {
		t.Fatalf("prepare source: %v", err)
	}
	renderer := newShaderPaneRenderer(0).(*shaderPaneRenderer)
	queue := &uitest.UIQueue{}
	renderer.queueUI = queue.Do
	var generated atomic.Int64
	renderer.generateTile = func(ctx context.Context, source *renderSource, key tileKey) (*renderTile, error) {
		generated.Add(1)
		return generateRenderTile(ctx, source, key)
	}
	scene := asyncTileScene(source, -824)
	renderer.Present(scene)
	waitForRenderer(t, renderer)
	for queue.Drain() {
	}
	firstCount := generated.Load()
	if firstCount == 0 {
		t.Fatal("initial scene generated no detail tiles")
	}

	renderer.Present(paneScene{})
	waitForRenderer(t, renderer)
	if renderer.shader.Visible() {
		t.Fatal("clear left shader visible")
	}
	renderer.Present(scene)
	waitForRenderer(t, renderer)
	for queue.Drain() {
	}
	if got := generated.Load(); got != firstCount {
		t.Errorf("reopening cached source generated %d tiles total, want unchanged %d", got, firstCount)
	}
	if renderer.shader.Textures["overview"] != source.overview {
		t.Fatal("reopened source did not restore its overview")
	}
}

func TestCompareSettle_DrainsShaderTilesAndSwapReusesSourceCaches(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	var renderers [2]*shaderPaneRenderer
	feature := newFeature(func(_ context.Context, _ fyne.URI) (*imaging.LoadedImage, error) {
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, 2048, 2048))}}, nil
	}, Callbacks{}, func(index int) paneRenderer {
		renderer := newShaderPaneRenderer(index).(*shaderPaneRenderer)
		renderers[index] = renderer
		return renderer
	})
	var generated atomic.Int64
	for _, renderer := range renderers {
		renderer.generateTile = func(ctx context.Context, source *renderSource, key tileKey) (*renderTile, error) {
			generated.Add(1)
			return generateRenderTile(ctx, source, key)
		}
	}
	queue := &uitest.UIQueue{}
	feature.SetUIQueue(queue)
	feature.vectorPixels = func(fyne.CanvasObject, fyne.Size) (int, int) { return 2048, 2048 }
	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := feature.Settle(ctx); err != nil {
		t.Fatalf("Settle after Open: %v", err)
	}
	if queue.Len() != 0 {
		t.Fatalf("UI queue after Settle = %d, want empty", queue.Len())
	}
	firstCount := generated.Load()
	if firstCount == 0 {
		t.Fatal("shader feature generated no tiles")
	}

	feature.swapSides()
	if err := feature.Settle(ctx); err != nil {
		t.Fatalf("Settle after Swap: %v", err)
	}
	if got := generated.Load(); got != firstCount {
		t.Errorf("Swap generated %d tiles total, want cached count %d", got, firstCount)
	}
}

func TestCompareSwap_DuringTileWorkReusesCompletedSourceCache(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	var renderers [2]*shaderPaneRenderer
	feature := newFeature(func(_ context.Context, uri fyne.URI) (*imaging.LoadedImage, error) {
		width := 2048
		if uri.Name() == "right.png" {
			width = 3072
		}
		return &imaging.LoadedImage{Frames: []image.Image{image.NewRGBA(image.Rect(0, 0, width, 2048))}}, nil
	}, Callbacks{}, func(index int) paneRenderer {
		renderer := newShaderPaneRenderer(index).(*shaderPaneRenderer)
		renderers[index] = renderer
		return renderer
	})
	feature.vectorPixels = func(fyne.CanvasObject, fyne.Size) (int, int) { return 3072, 2048 }
	queue := &uitest.UIQueue{}
	feature.SetUIQueue(queue)

	leftStarted := make(chan struct{})
	var leftOnce sync.Once
	renderers[0].generateTile = func(ctx context.Context, source *renderSource, key tileKey) (*renderTile, error) {
		if source.frame.Bounds().Dx() == 2048 {
			leftOnce.Do(func() { close(leftStarted) })
			<-ctx.Done()
			return nil, ctx.Err()
		}
		return generateRenderTile(ctx, source, key)
	}
	var rightSourceGenerations atomic.Int64
	renderers[1].generateTile = func(ctx context.Context, source *renderSource, key tileKey) (*renderTile, error) {
		if source.frame.Bounds().Dx() == 3072 {
			rightSourceGenerations.Add(1)
		}
		return generateRenderTile(ctx, source, key)
	}

	feature.Overlay().Resize(fyne.NewSize(800, 400))
	feature.Open([2]fyne.URI{
		storage.NewFileURI("left.png"),
		storage.NewFileURI("right.png"),
	})
	feature.workers.Wait()
	if !queue.Drain() {
		t.Fatal("comparison load did not queue a UI completion")
	}
	waitForTileStart(t, leftStarted)
	waitForRenderer(t, renderers[1])
	beforeSwap := rightSourceGenerations.Load()
	if beforeSwap == 0 {
		t.Fatal("right source cache was not prepared before Swap")
	}

	feature.swapSides()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := feature.Settle(ctx); err != nil {
		t.Fatalf("Settle after active Swap: %v", err)
	}
	if got := rightSourceGenerations.Load(); got != beforeSwap {
		t.Errorf("completed right source regenerated %d tiles across Swap, want cached count %d", got, beforeSwap)
	}
	if renderers[0].shader.Textures["overview"] != feature.renderSources[0].overview {
		t.Fatal("left renderer did not publish the swapped source")
	}
}
