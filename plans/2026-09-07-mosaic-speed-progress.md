# Mosaic generation speed and progress

Deliverable: restore practical mosaic generation speed without changing image
quality, layout, settings, formats, memory limits, or cancellation, and show
measured canvas coverage in a determinate progress bar.

Route: promoted to Deep because the renderer, public progress contract and UI
span three packages plus their tests, documentation and locale bundles.
Lead owns design, implementation, review and fixes. Task graph: renderer (1) ->
progress (2) -> window/adapter (3) -> final gate (4). No commit is requested for
this new feature task. Renderer changes precede progress wiring.

## Evidence and decisions

Baseline `bc916fb`: deterministic 6000x4000 NRGBA source, default settings/seed 42,
1920x1080 generation with source decoding excluded: 231.763 s, 740,297,112 allocated
bytes, 42,381,963 allocations. One rotated 320x220 Polaroid placement: 2.660 s;
87.45% of sampled CPU is in Catmull-Rom Transform. Profiles/temporary benchmark:
`/tmp/picfetch-mosaic-profile/`. This isolates the branch's tiled resampling cost
from file I/O. Suspects ranked before further optimization: repeated 2-D kernel
work, repeated opacity scans, then layout/GC. The CPU profile supports the first.

Keep full-precision separable Catmull-Rom weights and premultiplied intermediates.
Bound temporary intermediates to small row bands; retain original virtual sample
coordinates, masks, frame/shadow treatment and the 64 MiB preparation budget.
Do not reduce output/source resolution or substitute a cheaper filter.

Progress measures completed canvas coverage, not elapsed-time prediction.
Callbacks run synchronously within generation; UI delivery coalesces updates on
the existing tracked worker/UIQueue and rejects cancelled/superseded sessions.

## Tasks and acceptance

1. T0: optimize bounded preparation. Files: `internal/mosaic/render.go`,
   `preparation.go`, new `resample.go`, existing `generator_test.go`.
   Existing Generate/render boundaries remain; preserve source fidelity and
   pixel equivalence to full preparation. Add deterministic benchmarks and a
   source-sample-count regression through rendered output (not a wall-clock test).
   Verify: `go test -race ./internal/mosaic`; benchmark before/after with identical
   fixture/seed and `-benchtime=1x`; full/tiled, transparency, panorama, SVG,
   cancellation and frame fingerprint tests remain unchanged.
2. T0: measured generation progress. Files: `generator.go`, `layout.go`,
   existing generator tests. Add `Progress{CoveredPixels, TotalPixels int}` and
   `GenerateWithProgress(ctx, request, func(Progress))`, retaining existing Generate.
   Verify public progress starts at zero, increases monotonically within bounds,
   finishes only on success, cancels promptly, and leaves output pixels identical:
   `go test -race ./internal/mosaic -run 'Progress|Cancel|Deterministic'`.
3. T0: determinate UI. Files: `internal/ui/mosaicwin/window.go`, new `progress.go`,
   existing mosaicwin tests, `internal/ui/mosaic.go`, translations/en.json and
   de.json. Extend existing Host generator callback argument. Preserve widget
   membership, cancellation, failure/previous preview and close/reopen behavior;
   coalesce held-queue progress without adding untracked work.
   Verify: `go test -race ./internal/ui/mosaicwin`; focused root mosaic tests;
   locale parity and rendered progress screenshot inspection.
4. T0: docs/final gate. Update ARCHITECTURE.md for new file responsibilities and
   todos.md. `make verify` once; if concurrent Docker partitions exhaust memory,
   retain logs and rerun only terminated partitions alone. Final lead diff review.

## Delegation and budget

One read-only UI-lifetime scout while T0 profiles rendering. G1: bounded mapping;
G2: rg-verifiable file:line results; G3: no writes; G4: UI-only context; G5: lead
had not read that UI. Shell found the relevant files before delegation. No review,
fix, interface decision or user-visible string is delegated.

| Task | Spawns budget/actual | Review rounds | Full suite |
| --- | --- | --- | --- |
| UI recon | 1 / 1 | lead-owned | no |
| Renderer | 0 / 0 | 2 | no |
| Progress/API/UI | 0 / 0 | 2 | no |
| Final gate | 0 / 0 | 1 | once; UI-3-only retry |

## Implementation evidence

- The sampling guard failed before optimization: 60,960,900 source reads versus
  its 30,720,000 limit. Separable filtering passed, then source-row conversion
  reuse reduced the single-photo benchmark further: 2.660 s -> 0.274 s.
- Same 24 MP synthetic fixture, decoding excluded: 1080p 231.763 s -> 27.837 s
  (8.3x); 4K after 35.472 s. These are measured scenarios, not a claim about every
  source set. Full/tiled comparisons, existing frame fingerprints and canonical
  formats pass. New up/downscale and clipped translucent 16-bit checks match
  the full CatmullRom.Scale result byte-for-byte. A box-filter mutation fails.
- Progress tests first rejected zero callbacks, missing cancellation and missing
  UI delivery. Complete mosaic/window race suites pass (20.596 s / 6.678 s), as do
  root mosaic integration tests (14.301 s). Progress lifetime tests pass 10 race
  repetitions (5.407 s). Removing stale-revision protection exposed a weak final
  assertion; the tightened test checks old delivery before the new update and
  rejects that mutation for Cancel, Regenerate and Close/reopen.
- A 720x500 rendered window at 75% coverage was visually inspected; progress is
  in the visible surface, with readable percentage text and an available Cancel
  button. Artifact: `/tmp/picfetch-mosaic-profile/progress.png`.
- Translation parity, formatting and exact Qodana exclusions pass. No new root UI
  runnable or test file was added, so no shard/exclusion manifest edit is needed.
- Full gate uses the usual Ubuntu 24.04 Linux/amd64 test image with only a Go
  `GOMEMLIMIT=1GiB` target added in an attempt to avoid the already-confirmed concurrent-shard
  Docker OOM. This is test-process configuration, not a product memory change.
  Command: `make verify TEST_IMAGE=picfetch-mosaic-verify:local`, against an
  isolated copy of the working diff. Formatting/TUF/Qodana, vet and build passed;
  non-UI and UI-1/UI-2 race partitions passed. Docker still emitted an OOM event
  for container `78413c82aaf2` and terminated UI-3 without an individual test
  failure. A UI-3-only retry in the same image passed all 230 tests (302.173 s).
  Logs: `/tmp/picfetch-mosaic-profile/verify.log`, `ui3-retry.log`, and the retained
  complete JSON stream `race-events/ui-3.json`. All source hashes match the tested
  isolated copy; only final documentation evidence was updated afterward.
- Lead review checked resampling normalization/alpha, workspace accounting,
  unchanged geometry/feature contracts, callback ownership and source diff scope.
  No unresolved product finding remains. The concurrent Docker memory issue
  predates these changes and remains an infrastructure follow-up in todos.md.

Windows/amd64 `go vet ./internal/...` also passed.
