package compare

import (
	"context"
	"image"
	"maps"
	"strconv"
	"strings"
	"testing"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/canvas"
	fynetest "fyne.io/fyne/v2/test"
)

func TestTileShaderSources_AgreeAndDeclareFixedPortableContract(t *testing.T) {
	const marker = "uniform vec2 frame;"
	desktop := strings.Index(tileShaderSourceDesktop, marker)
	es := strings.Index(tileShaderSourceES, marker)
	if desktop < 0 || es < 0 {
		t.Fatalf("shader sources must both declare the built-in frame uniform")
	}
	if tileShaderSourceDesktop[desktop:] != tileShaderSourceES[es:] {
		t.Fatal("desktop and GLES tile shader bodies differ below their preambles")
	}

	shared := tileShaderSourceDesktop[desktop:]
	if got := strings.Count(shared, "uniform sampler2D "); got != 1+detailSamplerCount {
		t.Errorf("sampler declarations = %d, want overview plus %d details", got, detailSamplerCount)
	}
	for _, declaration := range []string{
		"uniform sampler2D overview;",
		"texture2D(overview, local)",
		"vec2 pixel = vec2(",
		"tileContains(pixel, tile0MinX",
		"tileCoordinate(pixel, tile0MinX",
		"return texel / vec2(width, height);",
		"float bestStep = 0.0;",
		"min(tile0StepX, tile0StepY) > 0.0",
		"bestStep == 0.0",
		"color.rgb /= color.a;",
	} {
		if !strings.Contains(shared, declaration) {
			t.Errorf("tile shader missing %q", declaration)
		}
	}
	for _, inverted := range []string{"1.0 - local.y", "1.0 - texel.y / height"} {
		if strings.Contains(shared, inverted) {
			t.Errorf("tile shader vertically inverts Go image textures with %q", inverted)
		}
	}
	for _, normalizedDetail := range []string{
		"tileContains(local",
		"tileCoordinate(local",
		"min(maximum.x, 1.0)",
		"min(maximum.y, 1.0)",
	} {
		if strings.Contains(shared, normalizedDetail) {
			t.Errorf("tile shader still looks up detail in source-normalized coordinates with %q", normalizedDetail)
		}
	}
	for slot := range detailSamplerCount {
		slotName := strconv.Itoa(slot)
		for _, declaration := range []string{
			"uniform sampler2D detail" + slotName + ";",
			"uniform float tile" + slotName + "StepX;",
			"uniform float tile" + slotName + "StepY;",
			"uniform float tile" + slotName + "Width;",
			"uniform float tile" + slotName + "Height;",
		} {
			if !strings.Contains(shared, declaration) {
				t.Errorf("tile shader missing %q", declaration)
			}
		}
		for _, redundant := range []string{"Active", "MaxX", "MaxY", "Scale"} {
			declaration := "uniform float tile" + slotName + redundant + ";"
			if strings.Contains(shared, declaration) {
				t.Errorf("tile shader wastes portable fragment-uniform budget on %q", declaration)
			}
		}
	}
	for _, rawSourceUniform := range []string{"uniform float sourceWidth;", "uniform float sourceHeight;", "1.0e20"} {
		if strings.Contains(shared, rawSourceUniform) {
			t.Errorf("tile shader uses non-portable raw source coordinate contract %q", rawSourceUniform)
		}
	}
}

func TestShaderPaneRenderer_LargeSourceUsesRepresentablePanePixelTileCoordinates(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	frame := image.NewRGBA(image.Rect(0, 0, 32768, 2))
	source := newPreparedRenderSource(frame, image.NewRGBA(image.Rect(0, 0, 1024, 1)))
	tile := &renderTile{
		key:      tileKey{level: 0, x: 16, y: 0},
		texture:  image.NewRGBA(image.Rect(0, 0, tileTextureDimension, 4)),
		interior: image.Rect(16*tileInterior, 0, 17*tileInterior, 2),
		scale:    1,
	}
	source.tiles.Add(tile.key.cacheKey(), tile)
	scene := paneScene{
		source:        source,
		viewport:      fyne.NewSize(1024, 2),
		imagePosition: fyne.NewPos(-16*tileInterior, 0),
		imageSize:     fyne.NewSize(32768, 2),
		displaySize:   image.Pt(32768, 2),
	}
	renderer := newShaderPaneRenderer(0).(*shaderPaneRenderer)
	renderer.bindAvailable(scene, tilePlan{requests: []tileRequest{{key: tile.key}}})

	if got := renderer.shader.Uniforms["tile0MinX"]; got != 0 {
		t.Errorf("visible detail tile pane X = %v, want 0", got)
	}
	if got := renderer.shader.Uniforms["tile0StepX"]; got != 1 {
		t.Errorf("large-source detail X step = %v, want one display pixel per texel", got)
	}
	if got := renderer.shader.Uniforms["tile0StepY"]; got != 1 {
		t.Errorf("large-source detail Y step = %v, want one display pixel per texel", got)
	}
}

func TestShaderPaneRenderer_HasStableUniqueNamesAndFixedTextureSlots(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	left := newShaderPaneRenderer(0).(*shaderPaneRenderer)
	right := newShaderPaneRenderer(1).(*shaderPaneRenderer)
	leftAgain := newShaderPaneRenderer(0).(*shaderPaneRenderer)
	if left.shader.Name == right.shader.Name {
		t.Fatalf("pane shader names collide at %q", left.shader.Name)
	}
	if left.shader.Name != leftAgain.shader.Name {
		t.Errorf("left shader name changed between instances: %q and %q", left.shader.Name, leftAgain.shader.Name)
	}
	if len(left.shader.Textures) != 1+detailSamplerCount {
		t.Errorf("shader textures = %d, want %d fixed slots", len(left.shader.Textures), 1+detailSamplerCount)
	}
	if len(left.shader.Uniforms) != 2+detailSamplerCount*6 {
		t.Errorf("shader scalar uniforms = %d, want %d", len(left.shader.Uniforms), 2+detailSamplerCount*6)
	}
}

func TestShaderPaneRenderer_PreservesSamplerSlotsWhenRequestPriorityChanges(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	source, err := prepareRenderSource(context.Background(), image.NewRGBA(image.Rect(0, 0, 4096, 1024)))
	if err != nil {
		t.Fatalf("prepare source: %v", err)
	}
	keys := []tileKey{
		{level: 0, x: 0, y: 0},
		{level: 0, x: 1, y: 0},
		{level: 0, x: 2, y: 0},
	}
	for _, key := range keys {
		source.tiles.Add(key.cacheKey(), &renderTile{
			key:      key,
			texture:  image.NewRGBA(image.Rect(0, 0, tileTextureDimension, tileTextureDimension)),
			interior: image.Rect(key.x*tileInterior, 0, (key.x+1)*tileInterior, tileInterior),
			scale:    1,
		})
	}

	renderer := newShaderPaneRenderer(0).(*shaderPaneRenderer)
	scene := paneScene{
		source:      source,
		viewport:    fyne.NewSize(4096, 1024),
		imageSize:   fyne.NewSize(4096, 1024),
		displaySize: image.Pt(4096, 1024),
	}
	renderer.bindAvailable(scene, tilePlan{requests: []tileRequest{
		{key: keys[0]}, {key: keys[1]}, {key: keys[2]},
	}})
	before := renderer.bound
	for slot := range len(keys) {
		if before[slot] == nil {
			t.Fatalf("initial sampler slot %d is unbound", slot)
		}
	}
	renderer.bindAvailable(scene, tilePlan{requests: []tileRequest{
		{key: keys[2]}, {key: keys[1]}, {key: keys[0]},
	}})
	for slot := range len(keys) {
		if renderer.bound[slot] != before[slot] {
			t.Errorf("request reprioritization replaced sampler slot %d", slot)
		}
	}
}

func TestShaderPaneRenderer_MapsSceneAndCachedTilesWithoutReplacingStableTextures(t *testing.T) {
	app := fynetest.NewApp()
	t.Cleanup(app.Quit)

	frame := image.NewRGBA(image.Rect(0, 0, 1024, 2))
	source := newPreparedRenderSource(frame, image.NewRGBA(image.Rect(0, 0, 512, 1)))
	for x := range 2 {
		key := tileKey{level: 0, x: x, y: 0}
		tile, err := generateRenderTile(context.Background(), source, key)
		if err != nil {
			t.Fatalf("generate tile %d: %v", x, err)
		}
		source.tiles.Add(key.cacheKey(), tile)
	}

	renderer := newShaderPaneRenderer(0).(*shaderPaneRenderer)
	object := renderer.Object()
	scene := paneScene{
		source:        source,
		viewport:      fyne.NewSize(512, 200),
		imagePosition: fyne.NewPos(-20, 99),
		imageSize:     fyne.NewSize(1024, 2),
		panePosition:  image.Pt(17, 23),
		displaySize:   image.Pt(1024, 2),
	}
	renderer.Present(scene)
	if renderer.Object() != object {
		t.Fatal("renderer replaced its canvas object while presenting a scene")
	}
	if got := renderer.shader.Position(); got != scene.imagePosition {
		t.Errorf("shader position = %v, want %v", got, scene.imagePosition)
	}
	if got := renderer.shader.Size(); got != scene.imageSize {
		t.Errorf("shader size = %v, want %v", got, scene.imageSize)
	}
	if !renderer.shader.Visible() {
		t.Fatal("shader stayed hidden after a valid scene")
	}
	if got := renderer.shader.Textures["overview"]; got != source.overview {
		t.Fatal("overview texture is not the prepared source overview")
	}
	if got := renderer.shader.Uniforms["paneX"]; got != 17 {
		t.Errorf("shader pane X = %v, want 17", got)
	}
	if got := renderer.shader.Uniforms["paneY"]; got != 23 {
		t.Errorf("shader pane Y = %v, want 23", got)
	}
	if got := renderer.shader.Uniforms["tile0MinX"]; got != -20 {
		t.Errorf("first detail pane X = %v, want -20", got)
	}
	if got := renderer.shader.Uniforms["tile0MinY"]; got != 99 {
		t.Errorf("first detail pane Y = %v, want 99", got)
	}
	if got := renderer.shader.Uniforms["tile0StepX"]; got != 1 {
		t.Errorf("first detail display X step = %v, want 1", got)
	}
	if got := renderer.shader.Uniforms["tile0StepY"]; got != 1 {
		t.Errorf("first detail display Y step = %v, want 1", got)
	}
	before := make(map[string]image.Image, len(renderer.shader.Textures))
	maps.Copy(before, renderer.shader.Textures)
	renderer.Present(scene)
	for name, texture := range before {
		if renderer.shader.Textures[name] != texture {
			t.Errorf("unchanged scene replaced texture %q", name)
		}
	}

	moved := scene
	moved.imagePosition.X = 30
	renderer.Present(moved)
	bound := 0
	for slot, tile := range renderer.bound {
		if tile == nil {
			continue
		}
		bound++
		wantX := float32(tile.interior.Min.X*tile.scale) + moved.imagePosition.X
		if got := renderer.shader.Uniforms[detailUniform(slot, "MinX")]; got != wantX {
			t.Errorf("moved detail slot %d pane X = %v, want %v", slot, got, wantX)
		}
	}
	if bound == 0 {
		t.Fatal("geometry update left every cached detail tile unbound")
	}

	renderer.Present(paneScene{})
	if renderer.shader.Visible() {
		t.Fatal("shader remained visible after clear")
	}
	for slot := range detailSamplerCount {
		if renderer.shader.Uniforms[detailUniform(slot, "StepX")] != 0 ||
			renderer.shader.Uniforms[detailUniform(slot, "StepY")] != 0 {
			t.Errorf("detail slot %d remained active after clear", slot)
		}
		if renderer.shader.Textures[detailTextureName(slot)] != renderer.placeholder {
			t.Errorf("detail slot %d retained an application texture after clear", slot)
		}
	}
	if renderer.shader.Textures["overview"] != renderer.placeholder {
		t.Fatal("clear retained the source overview")
	}
	if _, ok := renderer.Object().(*canvas.Shader); !ok {
		t.Fatalf("renderer object type = %T, want *canvas.Shader", renderer.Object())
	}
}
