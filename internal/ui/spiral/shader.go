package spiral

import (
	"fmt"
	"image"

	"fyne.io/fyne/v2/canvas"
)

// Both backends compile one shared body; only version and precision differ.
const shaderSourceDesktop = "#version 110\n" + shaderBody
const shaderSourceES = `#version 100
#ifdef GL_ES
# ifdef GL_FRAGMENT_PRECISION_HIGH
precision highp float;
# else
precision mediump float;
# endif
precision mediump int;
#endif
` + shaderBody

const shaderBody = `uniform vec2 frame;
uniform vec4 bounds;
uniform float time;
uniform float arms;
uniform float twistBase;
uniform float speed;
uniform float hueSpeed;
uniform float centerOffsetX;
uniform float centerOffsetY;
uniform float density;
uniform float preset;
uniform float tunnelTime;
uniform float imageOpacityMin;
uniform float imageOpacityMax;
uniform sampler2D traveller0;
uniform float traveller0Active;
uniform float traveller0Born;
uniform float traveller0Duration;
uniform float traveller0Angle;
uniform float traveller0Curve;
uniform float traveller0Margin;
uniform float traveller0Aspect;
uniform float traveller0Size;
uniform sampler2D traveller1;
uniform float traveller1Active;
uniform float traveller1Born;
uniform float traveller1Duration;
uniform float traveller1Angle;
uniform float traveller1Curve;
uniform float traveller1Margin;
uniform float traveller1Aspect;
uniform float traveller1Size;
uniform sampler2D traveller2;
uniform float traveller2Active;
uniform float traveller2Born;
uniform float traveller2Duration;
uniform float traveller2Angle;
uniform float traveller2Curve;
uniform float traveller2Margin;
uniform float traveller2Aspect;
uniform float traveller2Size;

const float TAU = 6.28318530718;
const int NUM_LAYERS = 4;

vec3 hsv2rgb(vec3 c) {
    vec4 K = vec4(1.0, 2.0/3.0, 1.0/3.0, 3.0);
    vec3 p = abs(fract(c.xxx + K.xyz) * 6.0 - K.www);
    return c.z * mix(K.xxx, clamp(p - K.xxx, 0.0, 1.0), c.y);
}

vec3 rippleSpiral(vec2 frag, vec2 center) {
    vec2 d = frag - center;
    float radius = length(d) / twistBase;
    float angle = atan(d.y, d.x) * arms;
    float phase = angle + radius - time * speed;
    float v = sin(phase);
    float hue = fract((v + 1.0) * 0.5 + time * hueSpeed);
    float val = 0.5 + 0.5 * v;
    return hsv2rgb(vec3(hue, 0.9, val));
}

// nautilusSpiral draws several counter-winding, power-law spiral arms
// (radius = u^(1/curve), so they bunch tightly near the centre and fan out
// toward the edge like a nautilus shell) and combines them with a lighten
// blend (componentwise max), echoing the layered, screen-blended look of a
// reference multi-layer canvas spiral this preset is modelled after.
vec3 nautilusSpiral(vec2 frag, vec2 center) {
    vec2 d = frag - center;
    float r = length(d);
    float theta = atan(d.y, d.x);
    float rMax = max(length(frame) * 0.5, 1.0);
    float u = clamp(r / rMax, 0.0, 1.0);
    float turns = twistBase / 10.0;

    vec3 result = vec3(0.02, 0.01, 0.04);
    for (int i = 0; i < NUM_LAYERS; i++) {
        float layerT = float(i) / float(NUM_LAYERS - 1);
        float curve = 2.0 + layerT * 1.2;
        float spiralAngle = pow(u, 1.0 / curve) * (turns + layerT) * TAU;
        // Outer layers wind slightly slower, and the outermost reverses
        // direction, giving the layers visible relative motion.
        float layerSpeed = speed * (0.75 - layerT);
        float phase = theta - spiralAngle - time * layerSpeed;
        float armPos = fract(phase * arms / TAU);

        float duty = 0.32;
        float edge = 0.05;
        float band = smoothstep(0.0, edge, armPos) - smoothstep(duty, duty + edge, armPos);
        band = clamp(band, 0.0, 1.0);

        float hue = fract(u * 0.5 + layerT * 0.18 + time * hueSpeed);
        vec3 layerColor = hsv2rgb(vec3(hue, 0.85, 1.0)) * band;
        result = max(result, layerColor);
    }
    return result;
}


vec2 photoSize(float aspect, float edge) {
    return aspect >= 1.0 ? vec2(edge, edge / aspect) : vec2(edge * aspect, edge);
}

float rayExit(vec2 center, float angle, vec2 halfSize) {
    vec2 d = vec2(cos(angle), sin(angle));
    vec2 distance = vec2(1.0e10);
    if (d.x > 0.00000001) distance.x = (frame.x + halfSize.x - center.x) / d.x;
    if (d.x < -0.00000001) distance.x = (-halfSize.x - center.x) / d.x;
    if (d.y > 0.00000001) distance.y = (frame.y + halfSize.y - center.y) / d.y;
    if (d.y < -0.00000001) distance.y = (-halfSize.y - center.y) / d.y;
    return max(0.0, min(distance.x, distance.y));
}

float travelDepth(float born, float duration) {
    float progress = clamp((tunnelTime - born) / max(duration, 0.001), 0.0, 1.0);
    return 0.2 * progress + 0.8 * progress * progress;
}

vec4 traveller(sampler2D photo, float active, float born, float duration,
               float angle, float curve, float margin, float aspect, float sizeScale, vec2 center) {
    if (active < 0.5 || aspect <= 0.0 || tunnelTime < born || tunnelTime >= born + duration) return vec4(0.0);
    float unit = min(frame.x, frame.y);
    float depth = travelDepth(born, duration);
    vec2 initial = photoSize(aspect, 0.10 * unit * sizeScale);
    vec2 finalSize = photoSize(aspect, 0.25 * unit * sizeScale);
    float start = 0.036 * unit + length(initial) * 0.5;
    float finish = max(rayExit(center, angle + curve, finalSize * 0.5) + 0.035 * unit * margin,
                       0.10 * unit + length(finalSize) * 0.5);
    float bearing = angle + curve * depth;
    float radius = mix(start, finish, depth);
    vec2 position = center + radius * vec2(cos(bearing), sin(bearing));
    vec2 size = photoSize(aspect, (0.10 + 0.15 * depth) * unit * sizeScale);
    vec2 local = (gl_FragCoord.xy - position) / size + vec2(0.5);
    if (local.x < 0.0 || local.y < 0.0 || local.x > 1.0 || local.y > 1.0) return vec4(0.0);
    float edge = rayExit(center, bearing, vec2(0.0));
    float alpha = mix(imageOpacityMin, imageOpacityMax, clamp((radius-start) / max(edge-start, 0.001*unit), 0.0, 1.0));
    alpha *= smoothstep(0.0, 0.75, tunnelTime - born);
    vec2 border = min(local, vec2(1.0)-local) * size;
    float feather = smoothstep(0.0, 0.04 * min(size.x, size.y), min(border.x, border.y));
    // Fyne uploads premultiplied pixels; retain their source alpha and detail.
    return texture2D(photo, vec2(local.x, 1.0-local.y)) * (alpha * feather);
}

// Sort samples, not sampler bindings: texture identities stay fixed on the GPU.
void depthOrder(inout vec4 a, inout float da, inout float ba,
                inout vec4 b, inout float db, inout float bb) {
    if (da > db || (da == db && ba > bb)) {
        vec4 c = a; a = b; b = c;
        float d = da; da = db; db = d;
        float birth = ba; ba = bb; bb = birth;
    }
}

vec4 tunnelImages(vec2 center) {
    vec4 c0 = traveller(traveller0, traveller0Active, traveller0Born, traveller0Duration,
        traveller0Angle, traveller0Curve, traveller0Margin, traveller0Aspect, traveller0Size, center);
    float d0 = travelDepth(traveller0Born, traveller0Duration);
    float b0 = traveller0Born;
    vec4 c1 = traveller(traveller1, traveller1Active, traveller1Born, traveller1Duration,
        traveller1Angle, traveller1Curve, traveller1Margin, traveller1Aspect, traveller1Size, center);
    float d1 = travelDepth(traveller1Born, traveller1Duration);
    float b1 = traveller1Born;
    vec4 c2 = traveller(traveller2, traveller2Active, traveller2Born, traveller2Duration,
        traveller2Angle, traveller2Curve, traveller2Margin, traveller2Aspect, traveller2Size, center);
    float d2 = travelDepth(traveller2Born, traveller2Duration);
    float b2 = traveller2Born;
    depthOrder(c0, d0, b0, c1, d1, b1);
    depthOrder(c1, d1, b1, c2, d2, b2);
    depthOrder(c0, d0, b0, c1, d1, b1);
    vec4 result = c1 + c0 * (1.0-c1.a);
    result = c2 + result * (1.0-c2.a);
    if (result.a > imageOpacityMax) result *= imageOpacityMax / result.a;
    return result;
}

void main() {
    vec2 frag = gl_FragCoord.xy;
    // density is in [0.25, 1.0]; map it to a pixel block size from 1px
    // (native, density = 1.0) up to 30px (heavily pixelated, density = 0.25)
    // so the slider's effect is clearly visible rather than a 1-4px change
    // that would be imperceptible on a high-DPI display.
    float blockSize = max(1.0, 40.0 * (1.0 - density));
    frag = (floor(frag / blockSize) + 0.5) * blockSize;
    vec2 center = vec2(frame.x * 0.5, frame.y * 0.5) + vec2(centerOffsetX, centerOffsetY);

    vec3 rgb;
    if (preset < 0.5) {
        rgb = rippleSpiral(frag, center);
    } else {
        rgb = nautilusSpiral(frag, center);
    }
    vec4 photos = tunnelImages(center);
    gl_FragColor = vec4(photos.rgb + rgb * (1.0 - photos.a), 1.0);
}
`

// newShader builds the shader canvas object and seeds its Uniforms from st's
// current values (rather than the package defaults), so a state that has
// already been adjusted - e.g. settings restored from a previous run -
// opens the spiral at its current values instead of snapping back to
// defaults.
//
// "time" is deliberately absent from the seeded map: canvas.NewShaderAnimation
// writes that entry itself on every animated frame, so setting it here would
// just be immediately overwritten.
func newShader(st *state) *canvas.Shader {
	sh := canvas.NewShader("hypno-spiral-tunnel-v1", []byte(shaderSourceDesktop), []byte(shaderSourceES))

	centerOffsetX, centerOffsetY := st.centerOffset()
	opacityMin, opacityMax := st.imageOpacityRange()
	preset := float32(0)
	if st.preset() {
		preset = 1
	}

	sh.Uniforms = map[string]float32{
		"arms":            float32(st.arms),
		"twistBase":       float32(st.twist),
		"speed":           float32(st.speed()),
		"hueSpeed":        float32(st.hueSpeed()),
		"centerOffsetX":   float32(centerOffsetX),
		"centerOffsetY":   float32(centerOffsetY),
		"density":         float32(st.density),
		"preset":          preset,
		"imageOpacityMin": float32(opacityMin),
		"imageOpacityMax": float32(opacityMax),
	}

	sh.Textures = make(map[string]image.Image, 3)
	placeholder := image.NewRGBA(image.Rect(0, 0, 1, 1))
	sh.Uniforms["tunnelTime"] = 0
	for i := range 3 {
		name := fmt.Sprintf("traveller%d", i)
		sh.Textures[name] = placeholder
		for _, key := range []string{"Active", "Born", "Duration", "Angle", "Curve", "Margin", "Aspect", "Size"} {
			sh.Uniforms[name+key] = 0
		}
	}
	return sh
}
