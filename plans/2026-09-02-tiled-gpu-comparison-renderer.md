# Tiled GPU comparison renderer

Status: complete

Route: Deep. Profiling found a cross-cutting rendering defect: every comparison
pan and zoom refreshes the viewer root, causing Fyne to discard and rebuild
smoothly scaled image textures on the UI/render thread. The change introduces a
private rendering seam, asynchronous source/tile preparation, bounded caches,
and portable shader contracts. No package, dependency, preference, or exported
API is added.

Deliverable: side-by-side and swipe comparison pan/zoom by changing stable
shader uniforms. Source imagery is uploaded through a bounded overview plus
visible detail tiles, and both Command+D and physical Ctrl+D open comparison on
macOS.

## Evidence and locked decisions

- A 10-second native sample of the exact feature-branch binary attributed the
  interaction stall to Fyne Catmull-Rom image scaling (`scaleY_RGBA64Image_Src`,
  `scaleX_YCbCr420`, and `drawNRGBAOver`), reached through
  `paneInput.Dragged -> panBy -> viewer.ForceRepaint -> Container.Refresh`.
- Comparison owns one renderer per pane. Existing Fyne clip/reveal geometry and
  input routing remain authoritative for side-by-side and swipe layouts.
- `compare.New` keeps its public signature. An unexported constructor accepts a
  renderer factory so package tests can use a deterministic canvas reference
  renderer while production uses shaders.
- Each production renderer has a stable pane-specific shader name, one overview
  sampler, and seven detail samplers. Desktop GLSL 110 and GLES GLSL 100 stay
  structurally equivalent. Tile coordinates are normalized and use six scalar
  values per detail, keeping the fragment-uniform contract below the GLES 2
  minimum while avoiding mediump overflow on raw source coordinates.
- Detail textures are at most 1024 by 1024 pixels: a 1022-pixel interior with a
  one-pixel sampling gutter. The overview's long edge is at most 1024 pixels.
- The immutable decoded frame remains canonical. Detail mips are generated once
  in cancellable background work and cached per source in a 64 MiB byte cache.
- The overview always covers the source. Missing or stale details therefore
  reduce sharpness temporarily but never create blank regions.
- The tile planner selects a power-of-two level from source pixels per physical
  display pixel, coarsening until the visible set fits seven samplers, then uses
  spare slots for nearest-neighbor prefetch.
- Initial readiness means decoded sources and their overview textures are ready.
  Existing spinners remain visible until then. SVG reraster publication follows
  the same display-ready-source rule.
- At most one tile worker runs per pane. Source/view tokens reject stale work;
  all worker completions enter through the feature's `UIQueue`; `Settle` waits
  for load, vector, and tile work.
- Pan, zoom, resize, and tile publication update only pane renderers. Owner
  repaint remains for comparison open/close and surrounding chrome lifecycle.
- Bilinear GPU sampling is always used. Shader output unpremultiplies nonzero
  RGBA samples to match Fyne's shader blending contract.
- Clearing a pane replaces all fixed texture slots with transparent one-pixel
  placeholders. Fyne may retain the most recent fixed GL textures until reuse or
  app exit, but the retained set is bounded.
- Real GPU acceptance is macOS-only. Portable tests cover planning, lifecycle,
  shader structure, and builds; Windows/Linux runtime GPU behavior is explicitly
  unverified in this change. Fyne's software test painter does not implement
  `canvas.Shader`; comparison therefore requires the native GL painter and
  deterministic software-renderer tests use the private reference adapter.

Non-goals: the normal single-image viewer, grid thumbnails, a user-configurable
tile budget, changing comparison command ownership, and GPU-specific runtime
verification on Windows or Linux.

## Bite-sized execution

| Ticket | Slice | Verification boundary |
|---|---|---|
| 01 | Shortcut parity and interaction repaint guard | UI shortcut tests plus 100 pan/zoom events |
| 02 | Renderer/scene seam | Reference renderer scene and geometry tests |
| 03 | Display-ready immutable sources | Overview/source preparation and readiness tests |
| 04 | Virtual tile planner and bounded cache | Pure planner/cache table tests |
| 05 | Desktop/GLES shader adapter | Structural shader and uniform/texture tests |
| 06 | Async tile delivery and lifecycle | cancellation, stale, settle, and swap tests |
| 07 | Production wiring and test migration | compare and assembled UI suites |
| 08 | Native profile, docs, and landing gate | native sample, memory check, `make verify` once |

Task graph: 01 -> 02 -> 03 -> 04 -> 05 -> 06 -> 07 -> 08. Every behavior
slice is first observed red against absent behavior or an intentional local
mutation, then made green and refactored before the next slice.

## Native performance acceptance

Build an unstripped binary from the final working tree and collect separate
10-second samples while the user continuously pans/zooms in side-by-side and
swipe mode with the same source images used for the baseline.

- No main-thread Catmull-Rom image scaling during steady-state interaction.
- No gesture-to-`viewer.ForceRepaint` stack during steady-state interaction.
- Texture uploads occur for source/mip/tile changes, not every pointer event.
- Input does not build a visible interaction backlog.
- Peak physical footprint is at most 1.2 GiB, versus the measured ~2.0 GiB
  baseline.

Accepted native result, 2026-09-02: the user reported both modes visually
smooth. The side-by-side sample was 95.2% main-thread idle at 1.0 GiB; the swipe
sample was 95.7% main-thread idle at 1.0 GiB, with a 1.1 GiB process peak. Both
were free of Catmull-Rom, `drawNRGBAOver`, and gesture-to-`ForceRepaint` stacks.
The live Go heap remained 267-271 MiB across repeated collections. Physical
`Ctrl+D` was exercised to open the comparison. Windows and Linux GPU runtime
behavior remains unverified as planned.

## Ownership and cost ledger

Implementation and integration ownership remains with the primary agent because
renderer state, Fyne lifecycle, test migration, and profiling share hot context.
At the user's request, three read-only subagents independently audited shader,
lifecycle, and integration concerns; their findings were reproduced red and
fixed by the primary agent. Review budget: one local review after each ticket,
the three parallel audits, and one final review. Full verification budget: one
`make verify`, in ticket 08 only.

| Work | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|---|---:|---:|---:|---|
| Tickets 01-07 | 3 / 3 | per ticket + 3 audits | no | user-requested shader, lifecycle, and integration audits |
| Ticket 08 | 0 / 0 | 1 | yes | one full-gate invocation; one focused Linux UI race rerun completed the portable test fix |
