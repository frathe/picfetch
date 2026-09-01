package compare

import (
	"context"
	"errors"
	"image"
	"slices"
	"strconv"
	"sync"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
)

const tileShaderSourceDesktop = `#version 110
uniform vec2 frame;
uniform vec4 bounds;
uniform sampler2D overview;
uniform sampler2D detail0;
uniform sampler2D detail1;
uniform sampler2D detail2;
uniform sampler2D detail3;
uniform sampler2D detail4;
uniform sampler2D detail5;
uniform sampler2D detail6;
uniform float tile0MinX;
uniform float tile0MinY;
uniform float tile0StepX;
uniform float tile0StepY;
uniform float tile0Width;
uniform float tile0Height;
uniform float tile1MinX;
uniform float tile1MinY;
uniform float tile1StepX;
uniform float tile1StepY;
uniform float tile1Width;
uniform float tile1Height;
uniform float tile2MinX;
uniform float tile2MinY;
uniform float tile2StepX;
uniform float tile2StepY;
uniform float tile2Width;
uniform float tile2Height;
uniform float tile3MinX;
uniform float tile3MinY;
uniform float tile3StepX;
uniform float tile3StepY;
uniform float tile3Width;
uniform float tile3Height;
uniform float tile4MinX;
uniform float tile4MinY;
uniform float tile4StepX;
uniform float tile4StepY;
uniform float tile4Width;
uniform float tile4Height;
uniform float tile5MinX;
uniform float tile5MinY;
uniform float tile5StepX;
uniform float tile5StepY;
uniform float tile5Width;
uniform float tile5Height;
uniform float tile6MinX;
uniform float tile6MinY;
uniform float tile6StepX;
uniform float tile6StepY;
uniform float tile6Width;
uniform float tile6Height;

bool tileContains(vec2 position, float minX, float minY, float stepX, float stepY, float width, float height) {
    vec2 minimum = vec2(minX, minY);
    vec2 maximum = minimum + max(vec2(width, height) - vec2(2.0), vec2(0.0)) * vec2(stepX, stepY);
    return position.x >= minimum.x && position.x < min(maximum.x, 1.0) &&
        position.y >= minimum.y && position.y < min(maximum.y, 1.0);
}

vec2 tileCoordinate(vec2 position, float minX, float minY, float stepX, float stepY, float width, float height) {
    vec2 texel = (position - vec2(minX, minY)) / vec2(stepX, stepY) + vec2(1.0);
    return texel / vec2(width, height);
}

void main() {
    vec2 objectSize = max(bounds.zw - bounds.xy, vec2(1.0));
    vec2 local = vec2(
        (gl_FragCoord.x - bounds.x) / objectSize.x,
        ((frame.y - gl_FragCoord.y) - bounds.y) / objectSize.y
    );
    if (local.x < 0.0 || local.x > 1.0 || local.y < 0.0 || local.y > 1.0) {
        discard;
    }

    vec4 color = texture2D(overview, local);
    float bestStep = 0.0;

    if (min(tile0StepX, tile0StepY) > 0.0 && (bestStep == 0.0 || max(tile0StepX, tile0StepY) < bestStep) && tileContains(local, tile0MinX, tile0MinY, tile0StepX, tile0StepY, tile0Width, tile0Height)) {
        color = texture2D(detail0, tileCoordinate(local, tile0MinX, tile0MinY, tile0StepX, tile0StepY, tile0Width, tile0Height));
        bestStep = max(tile0StepX, tile0StepY);
    }
    if (min(tile1StepX, tile1StepY) > 0.0 && (bestStep == 0.0 || max(tile1StepX, tile1StepY) < bestStep) && tileContains(local, tile1MinX, tile1MinY, tile1StepX, tile1StepY, tile1Width, tile1Height)) {
        color = texture2D(detail1, tileCoordinate(local, tile1MinX, tile1MinY, tile1StepX, tile1StepY, tile1Width, tile1Height));
        bestStep = max(tile1StepX, tile1StepY);
    }
    if (min(tile2StepX, tile2StepY) > 0.0 && (bestStep == 0.0 || max(tile2StepX, tile2StepY) < bestStep) && tileContains(local, tile2MinX, tile2MinY, tile2StepX, tile2StepY, tile2Width, tile2Height)) {
        color = texture2D(detail2, tileCoordinate(local, tile2MinX, tile2MinY, tile2StepX, tile2StepY, tile2Width, tile2Height));
        bestStep = max(tile2StepX, tile2StepY);
    }
    if (min(tile3StepX, tile3StepY) > 0.0 && (bestStep == 0.0 || max(tile3StepX, tile3StepY) < bestStep) && tileContains(local, tile3MinX, tile3MinY, tile3StepX, tile3StepY, tile3Width, tile3Height)) {
        color = texture2D(detail3, tileCoordinate(local, tile3MinX, tile3MinY, tile3StepX, tile3StepY, tile3Width, tile3Height));
        bestStep = max(tile3StepX, tile3StepY);
    }
    if (min(tile4StepX, tile4StepY) > 0.0 && (bestStep == 0.0 || max(tile4StepX, tile4StepY) < bestStep) && tileContains(local, tile4MinX, tile4MinY, tile4StepX, tile4StepY, tile4Width, tile4Height)) {
        color = texture2D(detail4, tileCoordinate(local, tile4MinX, tile4MinY, tile4StepX, tile4StepY, tile4Width, tile4Height));
        bestStep = max(tile4StepX, tile4StepY);
    }
    if (min(tile5StepX, tile5StepY) > 0.0 && (bestStep == 0.0 || max(tile5StepX, tile5StepY) < bestStep) && tileContains(local, tile5MinX, tile5MinY, tile5StepX, tile5StepY, tile5Width, tile5Height)) {
        color = texture2D(detail5, tileCoordinate(local, tile5MinX, tile5MinY, tile5StepX, tile5StepY, tile5Width, tile5Height));
        bestStep = max(tile5StepX, tile5StepY);
    }
    if (min(tile6StepX, tile6StepY) > 0.0 && (bestStep == 0.0 || max(tile6StepX, tile6StepY) < bestStep) && tileContains(local, tile6MinX, tile6MinY, tile6StepX, tile6StepY, tile6Width, tile6Height)) {
        color = texture2D(detail6, tileCoordinate(local, tile6MinX, tile6MinY, tile6StepX, tile6StepY, tile6Width, tile6Height));
        bestStep = max(tile6StepX, tile6StepY);
    }

    if (color.a <= 0.0) {
        discard;
    }
    color.rgb /= color.a;
    gl_FragColor = color;
}
`

const tileShaderSourceES = `#version 100

#ifdef GL_ES
# ifdef GL_FRAGMENT_PRECISION_HIGH
precision highp float;
# else
precision mediump float;
#endif
precision mediump int;
#endif

uniform vec2 frame;
uniform vec4 bounds;
uniform sampler2D overview;
uniform sampler2D detail0;
uniform sampler2D detail1;
uniform sampler2D detail2;
uniform sampler2D detail3;
uniform sampler2D detail4;
uniform sampler2D detail5;
uniform sampler2D detail6;
uniform float tile0MinX;
uniform float tile0MinY;
uniform float tile0StepX;
uniform float tile0StepY;
uniform float tile0Width;
uniform float tile0Height;
uniform float tile1MinX;
uniform float tile1MinY;
uniform float tile1StepX;
uniform float tile1StepY;
uniform float tile1Width;
uniform float tile1Height;
uniform float tile2MinX;
uniform float tile2MinY;
uniform float tile2StepX;
uniform float tile2StepY;
uniform float tile2Width;
uniform float tile2Height;
uniform float tile3MinX;
uniform float tile3MinY;
uniform float tile3StepX;
uniform float tile3StepY;
uniform float tile3Width;
uniform float tile3Height;
uniform float tile4MinX;
uniform float tile4MinY;
uniform float tile4StepX;
uniform float tile4StepY;
uniform float tile4Width;
uniform float tile4Height;
uniform float tile5MinX;
uniform float tile5MinY;
uniform float tile5StepX;
uniform float tile5StepY;
uniform float tile5Width;
uniform float tile5Height;
uniform float tile6MinX;
uniform float tile6MinY;
uniform float tile6StepX;
uniform float tile6StepY;
uniform float tile6Width;
uniform float tile6Height;

bool tileContains(vec2 position, float minX, float minY, float stepX, float stepY, float width, float height) {
    vec2 minimum = vec2(minX, minY);
    vec2 maximum = minimum + max(vec2(width, height) - vec2(2.0), vec2(0.0)) * vec2(stepX, stepY);
    return position.x >= minimum.x && position.x < min(maximum.x, 1.0) &&
        position.y >= minimum.y && position.y < min(maximum.y, 1.0);
}

vec2 tileCoordinate(vec2 position, float minX, float minY, float stepX, float stepY, float width, float height) {
    vec2 texel = (position - vec2(minX, minY)) / vec2(stepX, stepY) + vec2(1.0);
    return texel / vec2(width, height);
}

void main() {
    vec2 objectSize = max(bounds.zw - bounds.xy, vec2(1.0));
    vec2 local = vec2(
        (gl_FragCoord.x - bounds.x) / objectSize.x,
        ((frame.y - gl_FragCoord.y) - bounds.y) / objectSize.y
    );
    if (local.x < 0.0 || local.x > 1.0 || local.y < 0.0 || local.y > 1.0) {
        discard;
    }

    vec4 color = texture2D(overview, local);
    float bestStep = 0.0;

    if (min(tile0StepX, tile0StepY) > 0.0 && (bestStep == 0.0 || max(tile0StepX, tile0StepY) < bestStep) && tileContains(local, tile0MinX, tile0MinY, tile0StepX, tile0StepY, tile0Width, tile0Height)) {
        color = texture2D(detail0, tileCoordinate(local, tile0MinX, tile0MinY, tile0StepX, tile0StepY, tile0Width, tile0Height));
        bestStep = max(tile0StepX, tile0StepY);
    }
    if (min(tile1StepX, tile1StepY) > 0.0 && (bestStep == 0.0 || max(tile1StepX, tile1StepY) < bestStep) && tileContains(local, tile1MinX, tile1MinY, tile1StepX, tile1StepY, tile1Width, tile1Height)) {
        color = texture2D(detail1, tileCoordinate(local, tile1MinX, tile1MinY, tile1StepX, tile1StepY, tile1Width, tile1Height));
        bestStep = max(tile1StepX, tile1StepY);
    }
    if (min(tile2StepX, tile2StepY) > 0.0 && (bestStep == 0.0 || max(tile2StepX, tile2StepY) < bestStep) && tileContains(local, tile2MinX, tile2MinY, tile2StepX, tile2StepY, tile2Width, tile2Height)) {
        color = texture2D(detail2, tileCoordinate(local, tile2MinX, tile2MinY, tile2StepX, tile2StepY, tile2Width, tile2Height));
        bestStep = max(tile2StepX, tile2StepY);
    }
    if (min(tile3StepX, tile3StepY) > 0.0 && (bestStep == 0.0 || max(tile3StepX, tile3StepY) < bestStep) && tileContains(local, tile3MinX, tile3MinY, tile3StepX, tile3StepY, tile3Width, tile3Height)) {
        color = texture2D(detail3, tileCoordinate(local, tile3MinX, tile3MinY, tile3StepX, tile3StepY, tile3Width, tile3Height));
        bestStep = max(tile3StepX, tile3StepY);
    }
    if (min(tile4StepX, tile4StepY) > 0.0 && (bestStep == 0.0 || max(tile4StepX, tile4StepY) < bestStep) && tileContains(local, tile4MinX, tile4MinY, tile4StepX, tile4StepY, tile4Width, tile4Height)) {
        color = texture2D(detail4, tileCoordinate(local, tile4MinX, tile4MinY, tile4StepX, tile4StepY, tile4Width, tile4Height));
        bestStep = max(tile4StepX, tile4StepY);
    }
    if (min(tile5StepX, tile5StepY) > 0.0 && (bestStep == 0.0 || max(tile5StepX, tile5StepY) < bestStep) && tileContains(local, tile5MinX, tile5MinY, tile5StepX, tile5StepY, tile5Width, tile5Height)) {
        color = texture2D(detail5, tileCoordinate(local, tile5MinX, tile5MinY, tile5StepX, tile5StepY, tile5Width, tile5Height));
        bestStep = max(tile5StepX, tile5StepY);
    }
    if (min(tile6StepX, tile6StepY) > 0.0 && (bestStep == 0.0 || max(tile6StepX, tile6StepY) < bestStep) && tileContains(local, tile6MinX, tile6MinY, tile6StepX, tile6StepY, tile6Width, tile6Height)) {
        color = texture2D(detail6, tileCoordinate(local, tile6MinX, tile6MinY, tile6StepX, tile6StepY, tile6Width, tile6Height));
        bestStep = max(tile6StepX, tile6StepY);
    }

    if (color.a <= 0.0) {
        discard;
    }
    color.rgb /= color.a;
    gl_FragColor = color;
}
`

const (
	leftShaderPaneName  = "picfetch-compare-tiled-v1-left"
	rightShaderPaneName = "picfetch-compare-tiled-v1-right"
)

type shaderPaneRenderer struct {
	shader      *canvas.Shader
	placeholder *image.RGBA
	source      *renderSource
	scene       paneScene
	bound       [detailSamplerCount]*renderTile

	queueUI      func(func())
	generateTile func(context.Context, *renderSource, tileKey) (*renderTile, error)

	workerMu          sync.Mutex
	workerRevision    uint64
	workerSource      *renderSource
	workerRequests    []tileRequest
	workerContext     context.Context
	workerCancel      context.CancelFunc
	workerRunning     bool
	workerPending     workTracker
	publicationQueued bool
}

func newShaderPaneRenderer(index int) paneRenderer {
	name := leftShaderPaneName
	if index == 1 {
		name = rightShaderPaneName
	}
	placeholder := image.NewRGBA(image.Rect(0, 0, 1, 1))
	shader := canvas.NewShader(name, []byte(tileShaderSourceDesktop), []byte(tileShaderSourceES))
	shader.Textures = make(map[string]image.Image, 1+detailSamplerCount)
	shader.Uniforms = make(map[string]float32, detailSamplerCount*6)
	shader.Textures["overview"] = placeholder
	for slot := range detailSamplerCount {
		shader.Textures[detailTextureName(slot)] = placeholder
		for _, suffix := range []string{"MinX", "MinY", "StepX", "StepY", "Width", "Height"} {
			shader.Uniforms[detailUniform(slot, suffix)] = 0
		}
	}
	shader.Hide()
	return &shaderPaneRenderer{
		shader:       shader,
		placeholder:  placeholder,
		queueUI:      func(apply func()) { fyne.Do(apply) },
		generateTile: generateRenderTile,
	}
}

func (r *shaderPaneRenderer) Object() fyne.CanvasObject { return r.shader }

func (r *shaderPaneRenderer) Present(scene paneScene) {
	if scene.source == nil || scene.source.frame == nil || scene.source.overview == nil {
		r.clear()
		return
	}

	if r.source != scene.source {
		r.cancelTileRequest()
		r.source = scene.source
		r.shader.Textures["overview"] = scene.source.overview
		for slot := range detailSamplerCount {
			r.clearDetail(slot)
		}
	}
	r.scene = scene

	plan := planTiles(scene)
	r.bindAvailable(scene, plan)

	r.shader.Resize(scene.imageSize)
	r.shader.Move(scene.imagePosition)
	r.shader.Show()
	r.shader.Refresh()
	r.requestTiles(scene.source, plan.requests)
}

func (r *shaderPaneRenderer) bindAvailable(scene paneScene, plan tilePlan) {
	if scene.source == nil || scene.source.tiles == nil || len(plan.requests) == 0 {
		for slot := range detailSamplerCount {
			r.clearDetail(slot)
		}
		return
	}

	ready := make(map[tileKey]*renderTile, detailSamplerCount)
	allDesiredReady := true
	// Touch distant prefetch entries first so the nearest visible tiles finish
	// as the cache's most-recently-used entries.
	for _, request := range slices.Backward(plan.requests) {

		tile, ok := scene.source.tiles.Get(request.key.cacheKey())
		if !ok {
			allDesiredReady = false
			continue
		}
		ready[request.key] = tile
	}

	assigned := make(map[tileKey]bool, detailSamplerCount)
	for slot, tile := range r.bound {
		if tile == nil {
			continue
		}
		if _, wanted := ready[tile.key]; wanted {
			assigned[tile.key] = true
			continue
		}
		if allDesiredReady {
			r.clearDetail(slot)
		}
	}

	for _, request := range plan.requests {
		tile := ready[request.key]
		if tile == nil || assigned[request.key] {
			continue
		}
		slot := r.availableDetailSlot(ready)
		if slot < 0 {
			break
		}
		r.bindDetail(slot, tile, scene.source.frame.Bounds().Size())
		assigned[request.key] = true
	}
}

func (r *shaderPaneRenderer) availableDetailSlot(wanted map[tileKey]*renderTile) int {
	for slot, tile := range r.bound {
		if tile == nil {
			return slot
		}
	}
	for slot, tile := range r.bound {
		if _, keep := wanted[tile.key]; !keep {
			return slot
		}
	}
	return -1
}

func (r *shaderPaneRenderer) bindDetail(slot int, tile *renderTile, sourceSize image.Point) {
	if slot < 0 || slot >= detailSamplerCount || tile == nil || tile.texture == nil {
		return
	}
	r.bound[slot] = tile
	r.shader.Textures[detailTextureName(slot)] = tile.texture
	r.shader.Uniforms[detailUniform(slot, "MinX")] = float32(tile.interior.Min.X*tile.scale) / float32(sourceSize.X)
	r.shader.Uniforms[detailUniform(slot, "MinY")] = float32(tile.interior.Min.Y*tile.scale) / float32(sourceSize.Y)
	r.shader.Uniforms[detailUniform(slot, "StepX")] = float32(tile.scale) / float32(sourceSize.X)
	r.shader.Uniforms[detailUniform(slot, "StepY")] = float32(tile.scale) / float32(sourceSize.Y)
	r.shader.Uniforms[detailUniform(slot, "Width")] = float32(tile.texture.Bounds().Dx())
	r.shader.Uniforms[detailUniform(slot, "Height")] = float32(tile.texture.Bounds().Dy())
}

func (r *shaderPaneRenderer) clearDetail(slot int) {
	r.bound[slot] = nil
	r.shader.Textures[detailTextureName(slot)] = r.placeholder
	r.shader.Uniforms[detailUniform(slot, "StepX")] = 0
	r.shader.Uniforms[detailUniform(slot, "StepY")] = 0
}

func (r *shaderPaneRenderer) clear() {
	r.cancelTileRequest()
	r.source = nil
	r.scene = paneScene{}
	r.shader.Textures["overview"] = r.placeholder
	for slot := range detailSamplerCount {
		r.clearDetail(slot)
	}
	r.shader.Hide()
	r.shader.Refresh()
}

func (r *shaderPaneRenderer) setQueueUI(queue func(func())) {
	if queue == nil {
		r.queueUI = func(apply func()) { fyne.Do(apply) }
		return
	}
	r.queueUI = queue
}

func (r *shaderPaneRenderer) requestTiles(source *renderSource, requests []tileRequest) {
	if source == nil || source.tiles == nil {
		r.cancelTileRequest()
		return
	}
	missing := false
	for _, request := range requests {
		if !source.tiles.Contains(request.key.cacheKey()) {
			missing = true
			break
		}
	}
	r.workerMu.Lock()
	if r.workerRunning && r.workerSource == source {
		if sameTileRequests(r.workerRequests, requests) {
			r.workerMu.Unlock()
			return
		}
		// A pan or zoom replaces the desired plan, but does not cancel the
		// tile whose destination buffer is already allocated. Let that one
		// finish, cache it, then move directly to the latest plan. Repeated
		// cancellation here otherwise strands one full tile per input event
		// until the next GC cycle.
		r.workerRevision++
		r.workerRequests = append(r.workerRequests[:0], requests...)
		r.workerMu.Unlock()
		return
	}
	if !missing {
		r.workerMu.Unlock()
		// A worker may have populated the last missing cache entry between
		// Present's bind pass and this check. No active same-source worker
		// remains whose queued publication needs preserving.
		r.cancelTileRequest()
		return
	}
	r.workerRevision++
	if r.workerCancel != nil {
		r.workerCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	r.workerContext = ctx
	r.workerCancel = cancel
	r.workerSource = source
	r.workerRequests = append(r.workerRequests[:0], requests...)
	if r.workerRunning {
		r.workerMu.Unlock()
		return
	}
	r.workerRunning = true
	r.workerPending.Add(1)
	r.workerMu.Unlock()
	go r.runTileWorker()
}

func (r *shaderPaneRenderer) cancelTileRequest() {
	r.workerMu.Lock()
	r.workerRevision++
	if r.workerCancel != nil {
		r.workerCancel()
	}
	r.workerContext = nil
	r.workerCancel = nil
	r.workerSource = nil
	r.workerRequests = nil
	r.workerMu.Unlock()
}

func (r *shaderPaneRenderer) runTileWorker() {
	defer r.workerPending.Done()
	for {
		r.workerMu.Lock()
		if r.workerSource == nil || len(r.workerRequests) == 0 {
			r.workerRunning = false
			r.workerMu.Unlock()
			return
		}
		revision := r.workerRevision
		source := r.workerSource
		requests := append([]tileRequest(nil), r.workerRequests...)
		requestContext := r.workerContext
		r.workerMu.Unlock()

		for _, request := range requests {
			if requestContext.Err() != nil {
				break
			}
			if source.tiles.Contains(request.key.cacheKey()) {
				continue
			}
			tile, err := r.generateTile(requestContext, source, request.key)
			if err != nil {
				if !errors.Is(err, context.Canceled) {
					fyne.LogError("Failed to prepare comparison detail tile", err)
				}
				continue
			}
			if !r.cacheGeneratedTile(requestContext, source, tile) {
				break
			}
			r.queueTilePublication()
			if !r.latestTileRequest(revision, source) {
				break
			}
		}

		r.workerMu.Lock()
		if r.workerRevision != revision {
			r.workerMu.Unlock()
			continue
		}
		r.workerRunning = false
		r.workerCancel = nil
		r.workerMu.Unlock()
		return
	}
}

func (r *shaderPaneRenderer) cacheGeneratedTile(ctx context.Context, source *renderSource, tile *renderTile) bool {
	r.workerMu.Lock()
	defer r.workerMu.Unlock()
	if ctx.Err() != nil || r.workerContext != ctx || r.workerSource != source {
		return false
	}
	source.tiles.Add(tile.key.cacheKey(), tile)
	return true
}

func (r *shaderPaneRenderer) queueTilePublication() {
	r.workerMu.Lock()
	if r.publicationQueued {
		r.workerMu.Unlock()
		return
	}
	r.publicationQueued = true
	r.workerMu.Unlock()

	// Capture only the renderer. When the UI catches up, one callback binds
	// every currently available tile for the latest scene; stale generations
	// therefore cannot retain sources or build an interaction backlog.
	r.queueUI(func() {
		r.workerMu.Lock()
		r.publicationQueued = false
		r.workerMu.Unlock()
		if r.source == nil || r.scene.source != r.source {
			return
		}
		r.bindAvailable(r.scene, planTiles(r.scene))
		r.shader.Refresh()
	})
}

func (r *shaderPaneRenderer) latestTileRequest(revision uint64, source *renderSource) bool {
	r.workerMu.Lock()
	defer r.workerMu.Unlock()
	return r.workerRevision == revision && r.workerSource == source
}

func sameTileRequests(a, b []tileRequest) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func (r *shaderPaneRenderer) Wait(ctx context.Context) error {
	return r.workerPending.WaitContext(ctx)
}

func detailTextureName(slot int) string {
	return "detail" + strconv.Itoa(slot)
}

func detailUniform(slot int, suffix string) string {
	return "tile" + strconv.Itoa(slot) + suffix
}
