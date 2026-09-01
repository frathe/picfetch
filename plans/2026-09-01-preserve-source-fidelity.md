# Preserve comparison source fidelity

Status: complete

Route: Standard. This ticket completes the comparison feature through the
confirmed `compare.Feature` / `Overlay` seam and the assembled viewer's real
imaging adapter. It adds no package, dependency, preference, or runtime string.

Deliverable: comparison keeps canonical full-fidelity sources across every
transform and rerasterizes SVGs with cancellable, stale-safe background work.

## Locked decisions

| Decision | Contract |
|---|---|
| Test seam | `compare.Feature` / `Overlay` proves rendering and lifecycle behavior; the assembled viewer proves the real decode/cache path. |
| Raster source | Canvas geometry may scale the original decoded first frame, but comparison never substitutes or mutates it. |
| Vector source | A pane-local frame rerasterizes to its current device-pixel display target, clamped by the existing vector ceiling; cached `LoadedImage` remains immutable. |
| Readiness | `Ready` continues to mean both sources decoded; an initial valid SVG frame remains interactive while a sharper raster is pending. |
| Failure | Initial load failure closes comparison; a later SVG reraster failure retains the last valid frame, matching the normal viewer. |
| Cleanup | `Settle` covers superseded load/vector work through its queued UI completion. |

Non-goals: new formats, thumbnails, animation playback, applying viewer-only
rotation, changing the cache budget, and the separate comparison additions in
`todos.md`.

## Tasks

### Task 1 - Source and vector fidelity

Owner: T0 inline

Status: completed

Files: `internal/ui/compare` and its tests.

Contract: retain raster frames; add per-pane debounced SVG rerasterization for
zoom, resize, layout, and Swap; cancel stale work on every source/target change.

Test: `CompareRasterFidelity`, `CompareVector`, `CompareCancel`,
`CompareSettle`, and `CompareStale` as vertical red/green slices.

Verify: the first, second, and sixth ticket acceptance commands.

Budget: 0 spawns; at most 3 review rounds; no full suite.

### Task 2 - Canonical loader guarantees

Owner: T0 inline

Status: completed

Files: assembled comparison tests; production adapter only if a guard exposes a
real mismatch.

Contract: real RAW, GIF, oriented, over-budget, and oversized inputs preserve
the normal imaging path's behavior without mutating the file set.

Test: `CompareRAW`, `CompareAnimated`, `CompareOrientation`, `CompareMemory`,
and `CompareInputLimit`.

Verify: the third through fifth ticket acceptance commands.

Budget: 0 spawns; at most 2 review rounds; no full suite.

### Task 3 - Documentation and landing

Owner: T0 inline

Status: completed

Files: both manuals and their guard, architecture/TODO/ticket records, and this
plan.

Contract: document the completed fidelity behavior, retain unrelated TODO
additions, and record only negatively verified and green claims.

Verify: the seventh ticket acceptance command, all focused commands, then
`make verify` once.

Budget: 0 spawns; at most 1 review round; full suite once.

Task graph: Task 1 -> Task 2 -> Task 3. Each new guard is observed failing
against missing behavior or a deliberate mutation before final acceptance.

## Delegation gate

No task is delegated. The state, concurrency, and source-format decisions are
hot cross-file context (G4/G5); manuals and final review cannot be delegated.
Budget: 0 spawns.

## Cost ledger

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
|---|---:|---:|---|---|
| T1 | 0 / 0 | 3 | no | Raster retention, SVG target/clamp, stale/cancel, and settle guards all mutation-checked and race-clean. |
| T2 | 0 / 0 | 2 | no | Real loader guards cover RAW, GIF, EXIF/view rotation, cache pressure, and encoded-input refusal. |
| T3 | 0 / 0 | 1 | no | Bilingual manual guard went red/green; architecture, agent invariant, TODO, and ticket records updated. |
| gate | - | 1 | yes | All seven acceptance commands, full comparison suite, and `make verify` passed. |
