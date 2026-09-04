# Implementation Plan: Smooth mosaic card edges

Status: ready-for-human
Route: Standard
Spec: `.scratch/bildmosaik/spec.md` AC12
Issue: `.scratch/bildmosaik/issues/18-use-area-coverage-for-card-edges.md`

## Frame

Deliver area-covered rotated card, frame, and photo boundaries while preserving
the exact target-sized renderer result. Layout changes, new controls, and a
GPU/window dependency are out of scope.

The route is Standard because the production change stays in one package, but
the rendering contract, deterministic references, and visual proof all change.

## Decisions - Do Not Relitigate

| Decision | Resolution |
| --- | --- |
| Quality mechanism | Vector area masks in destination space plus a tiny mask-only Gaussian filter |
| Photo sharpness | Filter only geometric coverage; do not blur interiors |
| GPU | Do not couple generation to a live GL canvas or readback |
| Test oracle | Independent polygon clipping with 16/255 mean and 48/255 maximum error ceilings, not the production rasterizer |
| Cancellation | Retain bounded row-band checkpoints |

## Task Graph

```text
T1 Area-coverage guard and renderer
  -> T2 Deterministic references and visual inspection
  -> T3 Documentation and final gate
```

## Tasks

### Task 1 - Render exact geometric coverage

Owner: T0 inline
Files: `internal/mosaic/render.go`, `internal/mosaic/generator_test.go`
Depends: none
Contract: shadow, body, and photo rectangles are rasterized in destination
space; source sampling remains interpolated; cancellation is observable
Test: independent polygon clipping rejects binary or inaccurate 12-degree edge pixels
Verify: `go test ./internal/mosaic -run 'TestRenderPlacement_(AreaCoverage|Interpolates|StopsBetweenTransformBands)' -count=1`
Budget: 0 implementation spawns; 2 review rounds; full suite: no
Result: complete; a minimized original-renderer oracle failed with 124/216
geometric partial pixels rendered binary and 65/255 maximum alpha error.
Destination-space masks plus the tiny mask filter pass the expanded thin-frame
oracle with 0/471 binary partials, 12.2/255 mean error, and 45/255 maximum error.

### Task 2 - Refresh deterministic proof

Owner: T0 inline
Files: `internal/mosaic/generator_test.go`
Depends: T1
Contract: frame hashes describe the inspected renderer; existing public generation behaviors remain unchanged
Test: focused generation suite and representative native-scale PNG inspection
Verify: `go test ./internal/mosaic -run 'TestGenerate_(DeterministicAndCoverage|FrameStyles|DropShadow|SourceFidelity)' -count=1`
Budget: 0 implementation spawns; 2 review rounds; full suite: no
Result: complete; a representative 1200 x 750 mosaic and 2x nearest-neighbor
detail were inspected. Native macOS/arm64 and canonical Linux/amd64 produced
identical updated frame hashes.

### Task 3 - Record and verify

Owner: T0 inline
Files: spec, issue 18, `todos.md`, this plan
Depends: T1, T2
Contract: docs state the actual mechanism and honest GPU/raster limit
Test: AC commands, negative guard verification, repository final gate
Verify: `make verify`
Budget: 0 implementation spawns; 2 review rounds; full suite: yes, once
Result: complete; `make verify` passed, including the canonical Linux/amd64
race-test partitions.

## Delegation Gate

Two bounded read-only Scouts cover facts not answered by a single repository
search: the graphics-context/export constraint and an independent edge-quality
metric. Implementation and visual judgment remain T0 work because G2 and G5 do
not permit delegating them.

## Cost Ledger

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
| --- | --- | ---: | --- | --- |
| Recon | 2 / 2 | - | no | reused bounded Scouts |
| T1 | 0 / 0 | 2 | no | complete; guard red, then green |
| T2 | 0 / 0 | 2 | no | complete; native and Linux visual/reference checks |
| T3 | 0 / 0 | 1 | yes | complete; final repository gate passed |
