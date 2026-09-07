# Live mosaic canvas preview

Deliverable: show completed photo placements during generation, with bounded
preview memory and refresh rate, while preserving final pixels and all actions.

Route: Standard. Two feature packages; the existing Host/progress path carries
preview snapshots without an adapter change. Lead owns design, code and review.
The user accepted the generator/window testing and performance-check scope.

## Decisions

- Extend `mosaic.Progress` with an optional independent `image.Image` preview.
  Reporting stays synchronous on the existing generation worker; Generate with
  no callback incurs no preview work. Preview images are never mutated again by
  the producer. Callers may retain them.
- Publish after the first completed placement, then at most every 250 ms.
  Preserve aspect ratio, cap the longest edge at 960 pixels, and combine the
  existing background/repair and primary layers at preview resolution. Final
  composition/filtering is unchanged. This is a preview, not another render.
- Coalesce coverage and latest preview into one pending UI delivery. Reject
  retired generations, release discarded snapshots, and keep Cancel visible.
- Keep the previous finished image/result separately during regeneration;
  cancellation/failure restores it, or returns to configuration on a first run.
  Save/Wallpaper remain disabled until successful completion.
- No new workers, dependencies, strings, settings, or platform behavior.

## Tasks and acceptance

1. Lead: generator snapshots. Modify generator.go, existing generator_test.go;
   add preview.go. Verify early/immutable snapshots, bounded dimensions/cadence,
   both-layer composition and identical final pixels with preview on/off.
   Command: `go test -race ./internal/mosaic`.
2. Lead: window delivery/lifetime. Modify mosaicwin/window.go, progress.go,
   existing mosaicwin_test.go. Verify preview is in the visible surface, callbacks
   coalesce on UI, old sessions cannot paint, Cancel remains reachable, and
   cancellation/failure restores finished output or configuration.
   Command: `go test -race ./internal/ui/mosaicwin`.
3. Lead: benchmark preview on/off at 1080p and 4K using the existing 24 MP fixture;
   inspect a rendered window. Target a small measured overhead; revise throttling
   if previews materially erode the speed improvement.
   Command: `go test ./internal/mosaic -run '^$' -bench BenchmarkMosaicGeneration -benchtime=1x`.
4. Lead: final review, negative guard checks, architecture/todos update and
   `make verify` once. If Docker memory pressure kills a partition, record the
   evidence and rerun only affected partitions alone. No commit requested.

Task graph: 1 -> 2 -> 3 -> 4. Baseline benchmark runs alongside task 1.

## Delegation and budget

One read-only execution task: run the existing baseline benchmark and record
raw output while Lead implements. G1 bounded command; G2 process exit/output;
G3 no source edits; G4 no code context required; G5 baseline execution is cold.
No delegated design, review or fixes. Budget: one spawn, one final full gate;
targeted tests/benchmarks as evidence requires.

| Task | Spawns budget/actual | Review rounds | Full suite | Evidence |
|------|----------------------|---------------|------------|----------|
| baseline | 1 / 1 | 1 | no | Two baseline and paired on/off runs; same execution agent reran after resizer fix. |
| generator | 0 / 0 | 2 | no | Immutable/bounded snapshots, exact output, cadence and allocation guards pass. |
| window | 0 / 0 | 2 | no | Coalescing, stale frames, restore, layout and visible Cancel guards pass. |
| final gate | 0 / 0 | 1 | yes | All checks/partitions passed, with isolated UI-3 OOM retry. |

## Evidence

- Initial core test failed because no live snapshots were reported; the window
  test failed because no preview reached the visible surface. A layout guard
  then reproduced overlapping/unallocated controls. Explicit refreshes now
  arrange the progress bar and newly visible Cancel before painting.
- The final complete native race suites passed: internal/mosaic 23.175 s and
  internal/ui/mosaicwin 8.346 s. No top-level root UI test or test file was added,
  so neither shard assignments nor Qodana test exclusions require changes.
- Temporary mutation overlays were rejected for six faults: no throttling,
  aliasing live canvas pixels, dropping the primary layer, per-pixel resampler
  allocations, accepting retired progress, and failing to restore finished
  previews. Logs are under `/tmp/picfetch-live-preview/*-negative.txt`.
- Captured and inspected the real generator's partial preview at 760x620 and
  580x620. The progress bar, canvas, status and accessible Cancel are visible
  without overlap. Artifacts: `/tmp/picfetch-live-preview/live-760.png` and
  `live-580.png`. Screenshot code exists only in a temporary Go overlay.
- Baseline/paired benchmarks use the existing deterministic 6000x4000 decoded
  NRGBA source, default settings/seed 42, one generation per run, two runs per
  case. Decoding and UI paint are excluded. Initial NRGBA preview destinations
  used generic per-pixel resampling and added 8.6–10.8% time. RGBA snapshots
  select specialized loops and remove that allocation regression.

| Target | Preview off mean | Preview on mean | Overhead | Snapshots |
|--------|------------------|-----------------|----------|-----------|
| 1080p | 25.773 s | 26.496 s | 2.80% | 93 |
| 4K | 35.429 s | 36.155 s | 2.05% | 102 |

Each 16:9 snapshot is 960x540 (about 2 MiB); the producer, queued latest image,
current preview and previously finished output have distinct lifetimes. The
queue retains at most one pending snapshot per generation. Cumulative allocated
bytes increase by 193.5 MB / 212.2 MB over the entire benchmark generation;
these are successive snapshots, not retained buffers. Final composition,
source resolution and the 64 MiB preparation limit are unchanged.

Full gate: `make verify TEST_IMAGE=picfetch-mosaic-verify:local` against the
isolated final-source snapshot `/tmp/picfetch-live-preview/verify`:
format/TUF/Qodana checks, native vet/build, shard validation, non-UI race packages
and UI-1/UI-2 passed. UI-3 was terminated without an individual test failure;
Docker confirmed OOM at epoch 1788810518 for container
`54e27a18ac3357bd039d43e8025677d2df03c18bf377332c6ad43052d57514ae`.
The same UI-3 partition passed all 230 tests when rerun alone on that exact
snapshot/image (296.174 s, exit 0), with raw events retained at
`/tmp/picfetch-live-preview/ui-3-retry.json`. UI-1 passed in 363.082 s and UI-2
in 357.174 s. Thus all verification checks and race partitions passed; the
initial concurrent `make verify` itself exited 2 because of the OOM.
No source edits followed the verified snapshot. Only this evidence was added.
Windows cross-platform vet also passed:
`GOOS=windows GOARCH=amd64 go vet ./internal/...`.
