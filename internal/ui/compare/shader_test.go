package compare

import (
	"context"
	"image"
	"maps"
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
	for slot := range detailSamplerCount {
		for _, declaration := range []string{
			"uniform sampler2D detail" + string(rune('0'+slot)) + ";",
			"uniform float tile" + string(rune('0'+slot)) + "StepX;",
			"uniform float tile" + string(rune('0'+slot)) + "StepY;",
			"uniform float tile" + string(rune('0'+slot)) + "Width;",
			"uniform float tile" + string(rune('0'+slot)) + "Height;",
		} {
			if !strings.Contains(shared, declaration) {
				t.Errorf("tile shader missing %q", declaration)
			}
		}
		for _, redundant := range []string{"Active", "MaxX", "MaxY", "Scale"} {
			declaration := "uniform float tile" + string(rune('0'+slot)) + redundant + ";"
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
	if len(left.shader.Uniforms) != detailSamplerCount*6 {
		t.Errorf("shader scalar uniforms = %d, want %d", len(left.shader.Uniforms), detailSamplerCount*6)
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
	scene := paneScene{source: source}
	renderer.bindAvailable(scene, tilePlan{requests: []tileRequest{
		{key: keys[0]}, {key: keys[1]}, {key: keys[2]},
	}})
	before := renderer.bound
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
	if got := renderer.shader.Uniforms["tile0StepX"]; got != 1.0/1024.0 {
		t.Errorf("first detail normalized X step = %v, want %v", got, float32(1.0/1024.0))
	}
	if got := renderer.shader.Uniforms["tile0StepY"]; got != 1.0/2.0 {
		t.Errorf("first detail normalized Y step = %v, want 0.5", got)
	}
	before := make(map[string]image.Image, len(renderer.shader.Textures))
	maps.Copy(before, renderer.shader.Textures)
	renderer.Present(scene)
	for name, texture := range before {
		if renderer.shader.Textures[name] != texture {
			t.Errorf("unchanged scene replaced texture %q", name)
		}
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
