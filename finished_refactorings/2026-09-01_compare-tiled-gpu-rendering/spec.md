# Spec: tiled GPU comparison rendering

Status: approved

## Problem

Side-by-side and swipe pan/zoom recursively refresh the viewer canvas. Fyne then
performs smooth full-image resampling and texture recreation on the interactive
path. Native profiling shows that work, rather than transform arithmetic, is the
dominant CPU and memory cost. The running macOS app also lacks the physical
Ctrl+D comparison shortcut because Fyne's default shortcut modifier is Command.

## User-visible contract

1. Command+D and physical Ctrl+D both open comparison on macOS; platforms where
   the default shortcut is already Control register the command once.
2. Side-by-side and swipe retain their current layout, clipping, divider,
   linking, temporary-Control unlink, zoom, pan, swap, vector, and lifecycle
   behavior.
3. Pan and zoom remain responsive under sustained input and do not briefly show
   blank source regions while detail imagery is prepared.
4. Comparison opens only after both decoded sources have display-ready overview
   imagery; the existing spinners communicate that wait.
5. Bilinear filtering is used at every scale.

## Internal contract

- A private scene-based pane renderer separates transform policy from Fyne's
  canvas implementation. Production uses a tiled shader renderer; tests can use
  a reference canvas renderer through the unexported constructor.
- Stable shaders receive scalar viewport/image/display uniforms, one overview,
  and up to seven guttered detail textures. Interaction changes uniforms without
  refreshing the viewer root.
- Immutable render sources own a bounded 64 MiB tile cache. Tile choice is a
  deterministic pure plan based on visible source bounds and physical pixels.
- One cancellable worker per pane generates missing tiles. Tokens and queued UI
  completions prevent stale source/view publication. `Settle` observes all work.
- Shader programs exist in equivalent GLSL 110 and GLSL 100 forms and correctly
  handle transparent premultiplied source pixels.

## Acceptance

- Focused red/green tests for every ticket in `issues/`.
- `go test ./internal/ui/compare -count=1` and affected assembled UI tests pass.
- `make test-native` and documentation guards pass.
- Final `make verify` passes once.
- User-driven native profiles satisfy the thresholds in
  `plans/2026-09-02-tiled-gpu-comparison-renderer.md`.

