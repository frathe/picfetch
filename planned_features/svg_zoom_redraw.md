# SVG zoom re-render Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use
> `superpowers:subagent-driven-development` (recommended) to implement this
> plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.
> Parent session reviews every task **and fixes the code** before
> dispatching the next. Do not start Task N+1 until Task N is reviewed and
> fixed. **Do not run `git commit`.**

**Goal:** Zooming in on an SVG must produce a **new, sharper raster that
actually paints**, matching the manual: the image “re-renders sharp at
every zoom level rather than scaling up.”

**Architecture:** Keep the existing pipeline: `zoom.onScaleChanged` →
`requestVectorRender` → debounced `rasterizeVector` → `fyne.Do` →
`redrawRotatedFrame`. Change only the **target pixel size** (canvas
points × zoom is not device pixels on macOS Retina) and **force a
content-tree repaint** when the new raster lands (zoom already sized the
`canvas.Image`; a later `Resize` is a no-op once `syncWindowToZoom` grew
the window to match, so GPU texture rebuild cannot hang off Resize).

**Tech Stack:** Go 1.26, Fyne v2.8, existing `internal/ui/vector.go` +
`internal/ui/zoom` + `internal/imaging.ClampVectorRaster` / `RasterAt`.
No new dependencies. No imaging API change.

**Spec:** `todos.md` TODO “Bug: when zooming in on a SVG file the image
does not get wedrawn”.

**Precedent:** `internal/ui/spiral/spiral.go` (`updateFollowMode`) and
Fyne’s own `canvas.Image.renderSVG` already convert with
`Canvas.PixelCoordinateForPosition`. Both comments exist because
**`canvas.Scale()` is always 1 on macOS**; Retina/HiDPI is a private
`texScale` folded into that conversion (`scale * texScale` in
`fyne.io/fyne/v2/internal/driver/glfw/canvas.go`).

**What is already done (do not re-implement):**

- Debounced SVG re-render, hysteresis (`vectorSharpenRatio` 1.05 /
  `vectorReleaseRatio` 0.5), `requestLifecycle`, `vector.pending`.
- Logical size vs raster size; `SetLogicalSize`; unrotated `RasterAt`
  target (do not swap axes on `R`).
- `redrawRotatedFrame` already assigns `v.img.Image` and calls
  `v.img.Refresh()`.
- Tests that the **Go `image.Image` bounds** grow on `zoom.In()`
  (`TestSVGReRendersAtHigherDensityOnZoom`,
  `TestZoomKeysDriveVectorRerenders`). Those pass on the Fyne **test**
  driver (1× pixels). They do not lock device pixels or a GL repaint.

---

## Open questions (proposed defaults — confirm before dispatch)

Implementers treat the **Proposed** column as spec unless Florian
overrides it before Task 1 starts.

| # | Question | Proposed |
|---|----------|----------|
| 1 | Is this TODO the SVG zoom bug (not `exifwin.warmDone`)? | **Yes.** `warmDone` stays a later TODO. Uncommitted EXIF-button layout work is out of scope. |
| 2 | Symptom to fix? | **Both:** (a) raster target must be **device pixels** so Retina zoom is not a stretched 1× bitmap; (b) after a new raster lands, the window content must **repaint** (`ForceRepaint`), because window-follows-zoom makes `canvas.Image.Resize` a no-op. |
| 3 | Use `canvas.Scale()`? | **No.** Use `PixelCoordinateForPosition` only (spiral / Fyne SVG). Multiplying by `Scale()` as well would 4× on a driver where both `scale` and `texScale` are 2. |
| 4 | Sharp **first** paint on Retina (fit / 100%)? | **Yes, via the existing path.** `finishLoad` still decodes at logical size; `ResetToFit` already fires `onScaleChanged`. Device-pixel targeting makes that request 2× and the 90 ms debounce lands a sharp raster. Do **not** teach `internal/imaging` about DPI. Do **not** zero debounce for the first frame. |
| 5 | Keys vs scroll vs window-resize-while-fitting? | **All three.** They already share `onScaleChanged` → `requestVectorRender`. Do not add a second zoom path. |
| 6 | New `vectorView` test seam (`toPixels`)? | **No.** Pure helper + production calls `v.win.Canvas().PixelCoordinateForPosition`. Unit-test the helper with a fake 2× converter. Do not add a write-once field. |
| 7 | Golden screenshots / `make golden`? | **No.** The test driver is 1× and software-rendered. Lock math + `ForceRepaint` on the land path. Manual check on macOS after Task 2. |
| 8 | Migrate `warmDone` in this plan? | **No.** |

---

## Dispatch order and models

Parent: this session. **One implementer at a time.** After each task:
parent reviews the diff, **fixes if needed**, then dispatches the next.

Available Cursor Task models for this repo: `composer-2.5-fast` (cheap
mechanical), `claude-sonnet-5-thinking-high` (standard Go/Fyne),
`claude-opus-5-thinking-high` (only if a task cannot be split).

| Task | What | Implementer | Reviewer |
|------|------|-------------|----------|
| 1 | `vectorRasterTarget` + wire `requestVectorRender` to `PixelCoordinateForPosition` + unit tests | `go-expert` · `claude-sonnet-5-thinking-high` | parent (this session) |
| 2 | `ForceRepaint` when a vector raster lands; keep existing SVG zoom tests green | `go-expert` · `composer-2.5-fast` | parent |
| 3 | `ARCHITECTURE.md`, manuals (one sentence), `todos.md` | `generalPurpose` · `composer-2.5-fast` | parent |
| Final | Whole-branch review after Task 3 | — | parent; escalate to `go-expert` · `claude-opus-5-thinking-high` only if Task 1’s pixel math is still wrong on a 2× fake |

Task 1 is policy + Fyne DPI (easy to use `Scale()` or double-clamp). Do
not downgrade it to the cheap model. Task 2 is transcription once Task 1
lands pixels. Task 3 is docs.

Do **not** use Opus for Tasks 1–3. The work splits. The parent, not a
subagent, owns cross-task review and any fix-up.

---

## Global Constraints

Copied from `AGENTS.md`; every task’s requirements implicitly include these.

- **Do not run `git commit`.** Each task ends with a *suggested* commit
  message. The parent does not commit either unless Florian asks.
- Do not add `TODO`/`FIXME` comments to source. Open work belongs in
  `todos.md`.
- Update `ARCHITECTURE.md` in the same change when the SVG zoom raster
  story changes (Task 3).
- Every user-visible string is `lang.L("English text")` with that exact
  key in every `translations/*.json` bundle. This plan adds **no** new
  strings.
- No new dependencies. No mutable **package-level** test seams. Existing
  write-once `vectorView` fields (`debounce` / `rasterize` / `after`) stay
  as they are; do not add `toPixels`.
- Do not migrate `exifwin.warmDone`. Do not touch `internal/ui/exifwin`.
- Do not change hysteresis ratios, `defaultVectorDebounce` (90 ms), or
  `ClampVectorRaster` policy.
- Do not swap raster-target axes on rotation (keep the comment in
  `requestVectorRender`).
- `requestVectorRender` still must not touch widgets synchronously (it
  runs inside zoom `Layout`). `ForceRepaint` belongs in
  `rasterizeVector`’s `fyne.Do` callback only.
- Mark intentionally ignored errors explicitly (`_ =` or `_, _ =`).
- Verification per task, from the repository root, after the task’s own
  focused tests pass: `gofmt -l .` (must print nothing), `go vet ./...`,
  `go build ./...`, then the focused tests named in the task. The parent
  runs `go test -race ./...` after Task 2.

---

## File map

| File | Role |
|------|------|
| `internal/ui/vector.go` | `vectorRasterTarget`; `requestVectorRender` uses it; Task 2 `ForceRepaint` in the `fyne.Do` land path |
| `internal/ui/vector_test.go` | Helper table tests; existing zoom/SVG tests must still pass |
| `ARCHITECTURE.md` | “How does an SVG stay sharp when I zoom?” — device pixels + repaint |
| `internal/ui/help/manual.md`, `manual_de.md` | One sentence: re-render is in screen pixels (Retina included) |
| `todos.md` | Move the SVG zoom bug under Done |

No `internal/imaging` API change. No `internal/ui/zoom` API change.
`zoom` already reports the unrounded `Scale()`; it stays window- and
DPI-ignorant.

---

## Why the current tests still pass while the bug is real

1. **`requestVectorRender` targets canvas points**, not framebuffer
   pixels:
   `int(logical.Width*scale + 0.5)`. On this machine (darwin), GLFW’s
   `canvas.Scale()` is 1 and Retina is `texScale` ≈ 2.
   `PixelCoordinateForPosition` uses `scale * texScale`. A 340-point
   SVG at 125% zoom requests **425** pixels; the GL painter then asks
   for a texture of **~850** (`textureScale` / `pixScale`) and
   `painter.scaleImage` uploads `min(dest, source)` = **425** texels
   stretched over a Retina framebuffer. The picture **grows with the
   window** (`syncWindowToZoom`) but stays a stretched 1× bitmap — it
   does not look redrawn.

2. **`canvas.Image.Resize` no-ops when size is unchanged**
   (`fyne.io/fyne/v2/canvas/image.go`). Sequence on `+`: `apply`
   resizes the image to `logical*scale`; `onChanged` →
   `syncWindowToZoom` grows the window to that size; the next `Layout`
   sees the same size and **does not** `Refresh`. The new denser Go
   image later assigned in `rasterizeVector` therefore cannot rely on
   Resize to rebuild the GPU texture. `img.Refresh()` is queued;
   `finishLoad` additionally `ForceRepaint`s the registered content
   tree for the same class of “Refresh on a nested object didn’t
   paint” trap (`viewer.ForceRepaint` doc). The vector land path
   currently does not.

3. **Fyne test driver is 1×** (`PixelCoordinateForPosition` ≈ identity).
   `TestSVGReRendersAtHigherDensityOnZoom` only checks
   `v.img.Image.Bounds()`, which already updates. It cannot fail this
   bug.

---

## Assumptions (locked for implementers)

1. **Conversion:** `vectorRasterTarget(logical, zoomScale, toPixels)`
   builds `fyne.NewPos(logical.Width*zoomScale, logical.Height*zoomScale)`
   and passes that to `toPixels`. Production `toPixels` is
   `v.win.Canvas().PixelCoordinateForPosition`. Nil canvas / nil
   `toPixels` rounds the point size with `+ 0.5` (same as today).
2. **Clamp once in the helper** via `imaging.ClampVectorRaster` before
   `vectorNeedsRender`, same as today’s clamp-before-compare rule.
3. **Hysteresis unchanged** and still compares **pixel** sizes (`have`
   is `v.vector.raster`, `want` is the helper’s result). After a 2×
   fit raster lands, further zoom compares against that denser `have`.
4. **Rotation:** still unrotated logical size × scale, then
   `redrawRotatedFrame`. Do not swap `w`/`h` here.
5. **`ForceRepaint` only inside `fyne.Do`**, after
   `redrawRotatedFrame`, still under the token/`svg`/`displayFrames`
   guards.
6. **Existing SVG zoom tests stay passing** at 1× (test canvas).

---

## Task 1: Device-pixel raster target

**Files:**
- Modify: `internal/ui/vector.go` (`requestVectorRender`; add
  `vectorRasterTarget` next to it)
- Modify: `internal/ui/vector_test.go`

**Interfaces:**
- Consumes: `v.vector.logical`, `scale` from zoom, `v.win.Canvas()`,
  `imaging.ClampVectorRaster`, existing `vectorNeedsRender`.
- Produces:
  - `func vectorRasterTarget(logical fyne.Size, scale float32, toPixels func(fyne.Position) (int, int)) (w, h int)`
  - `requestVectorRender` uses that helper instead of inlined
    `logical*scale` rounding.

**Do not** call `ForceRepaint` in this task. Task 2 does paint.

- [ ] **Step 1: Write the failing tests** in `vector_test.go`

Append after `TestVectorNeedsRender` (the table that already lives in
this file).

```go
func TestVectorRasterTarget(t *testing.T) {
	times2 := func(p fyne.Position) (int, int) {
		return int(p.X*2 + 0.5), int(p.Y*2 + 0.5)
	}

	for _, tc := range []struct {
		name     string
		logical  fyne.Size
		scale    float32
		toPixels func(fyne.Position) (int, int)
		wantW    int
		wantH    int
	}{
		{"1x fit", fyne.NewSize(340, 340), 1, nil, 340, 340},
		{"1x one zoom step", fyne.NewSize(340, 340), 1.25, nil, 425, 425},
		{"2x fit (Retina)", fyne.NewSize(340, 340), 1, times2, 680, 680},
		{"2x one zoom step", fyne.NewSize(340, 340), 1.25, times2, 850, 850},
		{"wide 2x", fyne.NewSize(520, 260), 1, times2, 1040, 520},
		{"non-positive scale is zero", fyne.NewSize(340, 340), 0, times2, 0, 0},
		{"empty logical is zero", fyne.Size{}, 1.25, times2, 0, 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			w, h := vectorRasterTarget(tc.logical, tc.scale, tc.toPixels)
			if w != tc.wantW || h != tc.wantH {
				t.Fatalf("vectorRasterTarget = %dx%d, want %dx%d", w, h, tc.wantW, tc.wantH)
			}
		})
	}
}
```

`times2` is what `PixelCoordinateForPosition` does on a 2× framebuffer
for a position at the origin (glfw: `round(x * scale * texScale)`).
Do not import glfw.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test -count=1 ./internal/ui/ -run TestVectorRasterTarget -v`

Expected: FAIL compile or “undefined: vectorRasterTarget”.

- [ ] **Step 3: Implement `vectorRasterTarget` and wire it**

In `internal/ui/vector.go`, above `requestVectorRender`, add:

```go
// vectorRasterTarget is the pixel size requestVectorRender asks RasterAt
// for: logical size × zoom scale, converted from canvas points to device
// pixels. On macOS fyne.Canvas.Scale() is always 1 and Retina lives in a
// private texScale; PixelCoordinateForPosition is the conversion
// Fyne's own SVG rasterizer and internal/ui/spiral already use. A nil
// toPixels (no canvas yet) rounds the point size, matching the old
// logical*scale arithmetic.
func vectorRasterTarget(logical fyne.Size, scale float32, toPixels func(fyne.Position) (int, int)) (w, h int) {
	if scale <= 0 || logical.Width <= 0 || logical.Height <= 0 {
		return 0, 0
	}

	pos := fyne.NewPos(logical.Width*scale, logical.Height*scale)
	if toPixels == nil {
		w, h = int(pos.X+0.5), int(pos.Y+0.5)
	} else {
		w, h = toPixels(pos)
	}
	if w <= 0 || h <= 0 {
		return 0, 0
	}

	return imaging.ClampVectorRaster(w, h)
}
```

Replace the body of `requestVectorRender` so the size arithmetic becomes:

```go
func (v *viewer) requestVectorRender(scale float32) {
	if v.vector.svg == nil || scale <= 0 || v.vector.logical.Width <= 0 {
		return
	}

	var toPixels func(fyne.Position) (int, int)
	if v.win != nil {
		if c := v.win.Canvas(); c != nil {
			toPixels = c.PixelCoordinateForPosition
		}
	}
	w, h := vectorRasterTarget(v.vector.logical, scale, toPixels)
	if w <= 0 || h <= 0 {
		return
	}

	// No rotation adjustment here on purpose. The raster is produced in
	// unrotated space and redrawRotatedFrame turns it afterwards, and a
	// quarter turn preserves pixel count - so a raster of the unrotated
	// logical size times the scale rotates to exactly the size zoom lays
	// out. Swapping the axes would instead stretch the drawing, since
	// oksvg's SetTarget scales each axis independently.

	if !vectorNeedsRender(v.vector.raster, image.Pt(w, h)) {
		return
	}

	token := v.vector.lifecycle.begin()
	v.vector.pending.Add(1)

	go v.rasterizeVector(v.vector.svg, w, h, token)
}
```

Keep the “Clamp before comparing” idea: the helper clamps. Do **not**
call `ClampVectorRaster` a second time in `requestVectorRender`.
Do **not** use `v.win.Canvas().Scale()`.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -count=1 ./internal/ui/ -run 'TestVectorRasterTarget|TestSVGReRenders|TestZoomKeysDriveVector|TestVectorNeedsRender|TestRasterizeVector|TestRotatedNonSquareSVG|TestInfoOverlayReportsLogicalSize' -v`

Expected: PASS. Existing zoom tests stay at 1× (test canvas).

Then from the repository root:

```
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./internal/ui/ -run 'TestSVG|TestVector|TestZoomKeysDriveVector|TestRasterizeVector|TestRotatedNonSquare|TestInfoOverlayReportsLogicalSize|TestRotatingAZoomedSVG'
```

`gofmt -l .` must print nothing. If it names a file, run `gofmt -w` on it
and re-check.

- [ ] **Step 5: Suggested commit** (do not run `git commit`)

```
fix: rasterize zoomed SVGs at device pixels

Canvas points × zoom left Retina on a stretched 1× bitmap because
macOS reports Scale() as 1. PixelCoordinateForPosition matches Fyne's
own SVG path and the spiral easter egg.
```

---

## Task 2: Repaint when the new raster lands

**Files:**
- Modify: `internal/ui/vector.go` (`rasterizeVector`’s `fyne.Do` callback
  only)

**Interfaces:**
- Consumes: Task 1’s land path, `viewer.redrawRotatedFrame`,
  `viewer.ForceRepaint`.
- Produces: after a current token writes `displayFrames` / `vector.raster`
  and `redrawRotatedFrame`, `v.ForceRepaint()` runs on the same
  `fyne.Do`.

**Do not** call `ForceRepaint` from `requestVectorRender` (Layout).
**Do not** change `internal/ui/zoom`.

- [ ] **Step 1: Write the failing test**

There is no honest GPU-texture assertion under `fyne/test`. Do **not**
invent a sleep, a screenshot, or a `ForceRepaint` counter on `viewer`.

Instead, extend the land-path comment in the test that already waits out
a zoomed SVG, so a future deletion of `ForceRepaint` is a review issue
rather than a green test that lies:

In `TestSVGReRendersAtHigherDensityOnZoom`, after the existing bounds
assertions, add this comment only (no new assertion):

```go
	// Production also ForceRepaints here (rasterizeVector): after
	// syncWindowToZoom the canvas.Image size already matches the zoomed
	// window, so Resize is a no-op and cannot rebuild the GL texture.
	// The test driver has no GPU cache; this comment is the lock.
```

The **failing** part of this task is production behaviour on a real
window, which the parent verifies on macOS after the step-3 tests pass.
If you need a compile-time lock that `ForceRepaint` is referenced from
`vector.go`, that is already implied by calling it.

- [ ] **Step 2: Run the existing zoom test (still passes)**

Run: `go test -count=1 ./internal/ui/ -run TestSVGReRendersAtHigherDensityOnZoom -v`

Expected: PASS (comment-only).

- [ ] **Step 3: Call `ForceRepaint` after `redrawRotatedFrame`**

In `rasterizeVector`’s `fyne.Do` callback, the end of the success path
is currently:

```go
		v.displayFrames[0] = frame
		v.vector.raster = image.Pt(b.Dx(), b.Dy())

		// The one place that writes v.img.Image, which is what makes the
		// re-render compose with a pending rotation for free.
		v.redrawRotatedFrame()
```

Replace the last two lines (keep the `displayFrames` / `raster`
assignments) with:

```go
		v.displayFrames[0] = frame
		v.vector.raster = image.Pt(b.Dx(), b.Dy())

		// The one place that writes v.img.Image, which is what makes the
		// re-render compose with a pending rotation for free.
		v.redrawRotatedFrame()

		// finishLoad ForceRepaints after putting pixels on screen for the
		// same reason: Refresh on a nested canvas.Image can miss the
		// registered content tree. Zoom already sized the image; once
		// syncWindowToZoom grows the window to match, Resize is a no-op
		// and will not rebuild the GL texture for the denser raster.
		v.ForceRepaint()
```

Do not move this outside the token / `v.vector.svg != vec` /
`len(v.displayFrames) == 0` guards.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test -count=1 ./internal/ui/ -run 'TestSVG|TestVector|TestZoomKeysDriveVector|TestRasterizeVector|TestRotatedNonSquare|TestInfoOverlayReportsLogicalSize|TestRotatingAZoomedSVG|TestCloseFilesClearsVector' -v`

Expected: PASS.

Then from the repository root:

```
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./internal/ui/
```

Parent after this task: `go test -race ./...` from the repository root.

- [ ] **Step 5: Suggested commit** (do not run `git commit`)

```
fix: repaint the window when an SVG re-render lands

Window-follows-zoom leaves canvas.Image.Resize a no-op, so a denser
raster must ForceRepaint the registered content tree to upload GL.
```

---

## Task 3: Docs and todos

**Files:**
- Modify: `ARCHITECTURE.md` (vector.go row + “How does an SVG stay sharp
  when I zoom?”)
- Modify: `internal/ui/help/manual.md` (SVG bullet in supported formats)
- Modify: `internal/ui/help/manual_de.md` (matching SVG bullet)
- Modify: `todos.md`

**Interfaces:** none. No code.

- [ ] **Step 1: `ARCHITECTURE.md`**

In the `vector.go` table cell, after the sentence about
`requestVectorRender` / hysteresis, add that the raster **target is
device pixels** via `Canvas.PixelCoordinateForPosition` (not
`Scale()` — macOS Retina), and that `rasterizeVector` **ForceRepaints**
when a frame lands so a window that already grew with zoom still
uploads the new texture.

In the “How does an SVG stay sharp when I zoom?” index row, add the
same two facts in one short clause. Do not mention `warmDone`. Do not
rewrite the rest of either cell.

- [ ] **Step 2: Manuals**

English (`internal/ui/help/manual.md`), SVG bullet currently:

> **SVG** — `.svg` (vector; small icons open large enough to fill the window,
> and the image re-renders sharp at every zoom level rather than scaling up)

Keep that. Append one sentence in the same parenthetical or immediately
after:

> Re-rendering uses the screen’s pixels (including Retina), not just the
> window size in points.

German (`manual_de.md`): find the matching SVG sentence and add the same
fact (screen/Retina pixels, not only point size). No new `lang.L` keys.

- [ ] **Step 3: `todos.md`**

Move

```
## Bug: when zooming in on a SVG file the image does not get wedrawn
```

from TODO into Done, as a short past-tense note: zoomed SVGs now
rasterize at `PixelCoordinateForPosition` density and ForceRepaint when
the frame lands, so a Retina window that grew with zoom is not left on
a stretched 1× texture. Leave the `warmDone` item under TODO, untouched.

- [ ] **Step 4: Verify**

```
gofmt -l .
go vet ./...
go build ./...
go test -count=1 ./internal/ui/ ./internal/ui/help/
```

(help package if manuals are tested there; if `go test` on `help` is a
no-op besides compile, that is fine.)

- [ ] **Step 5: Suggested commit** (do not run `git commit`)

```
docs: SVG zoom re-renders in device pixels and repaints

Retina was stretching a 1× raster; the manual now matches the code.
```

---

## Self-review (plan vs spec)

| Spec / TODO | Task |
|-------------|------|
| Zooming an SVG must re-render, not only stretch | Task 1 device pixels; existing tests keep 1× behaviour |
| Image must actually paint after zoom | Task 2 `ForceRepaint` |
| Keys, scroll, fit-resize share one path | No new path; `onScaleChanged` unchanged |
| Do not use `canvas.Scale()` on macOS | Task 1 helper + Global Constraints |
| First Retina paint sharpens via ResetToFit | Task 1; no imaging DPI |
| `warmDone` untouched | Global constraint; Task 3 todos |
| ARCHITECTURE / manuals / todos | Task 3 |

No placeholders. `vectorRasterTarget` signature is used in Task 1 tests
and production wiring the same way.

---

## Parent review checklist (after every task)

1. Diff is only the files in that task’s File map.
2. No `git commit`.
3. Task 1: `requestVectorRender` does **not** call `Canvas.Scale()`;
   helper unit tests include the 2× cases; `TestSVGReRendersAtHigherDensityOnZoom`
   still passes.
4. Task 2: `ForceRepaint` is inside `fyne.Do`, after
   `redrawRotatedFrame`, behind the staleness guards; not in
   `requestVectorRender`.
5. Task 3: `warmDone` TODO still present; SVG zoom bug is under Done.
6. After Task 2, parent runs `go test -race ./...` and, if possible,
   opens an SVG on this Mac, presses `+` several times, and checks the
   drawing gets **sharper**, not only a larger window.

If Task 1’s 2× cases fail because `int(p.X*2 + 0.5)` disagrees with a
`float32` intermediate, keep the test’s `times2` and match it in the
helper’s `toPixels` path (the helper must not invent a third rounding
rule). Production rounding is whatever `PixelCoordinateForPosition`
returns; the helper must not re-round that result.
