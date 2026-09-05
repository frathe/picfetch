# Mosaic review fixes

Status: implemented and verified; retained here until branch acceptance
Route: Deep (cross-package fixes and Windows adapter behavior)

## Scope and decisions

Fix the six findings in the current branch review: untranslated display names,
detached Windows displays accepted as attached, colliding display labels,
unbounded repeated-source retention, partial GIF first frames, and masks that
fade at the canvas boundary. The user authorized all six fixes. Existing
review reproductions and the existing feature/decoder/adapter boundaries are
the regression seams; no additional product decisions are needed.

- Keep display identity in opaque IDs. Disambiguate only colliding labels and
  translate generated fallback names through the shared snapshot path.
- Interpret `GetMonitorRECT` separately from generic HRESULT success:
  `S_OK` is attached, `S_FALSE` is detached, and failures remain errors.
  Enumeration skips detached entries; wallpaper preflight rejects them.
- Bound the generation-local repeat cache to 64 MiB using `imaging.ByteCache`
  and its `AddIfFits` policy. An oversized source may render but is not retained.
- Preserve a GIF's logical first canvas in the canonical frozen decode path,
  without decoding subsequent frames or changing the animation budget policy.
- Rasterize the coverage mask with a one-pixel filter margin before clipping.
  Keep the existing area-coverage algorithm and photo interpolation.
- No source modification, real wallpaper changes, new background work, or
  unrelated cleanup. The user's existing `.gitignore` edit is outside scope.

## Tasks and acceptance commands

All tasks are T0 inline. Each slice runs red, then green, at its existing
boundary. The review already established the causes, so new hypothesis search
and instrumentation are unnecessary. No agent delegation is appropriate for
post-review fixes or platform and localization changes.

### T1 - Display choice and localization

Files: `internal/ui/mosaicwin/window.go`, existing window tests,
`internal/displays/{displays,windows,darwin}.go`, `translations/{en,de}.json`.
Contract: target IDs survive duplicate labels, choice, refresh, and generation;
fallback display names use one localized English key.
Test: two identical displays retain the initial default and can both be chosen;
reordered refresh preserves the chosen ID.
Verify: `go test ./internal/ui/mosaicwin -run TestMosaicTarget -count=1 && go test . -run TestTranslations -count=1`
Budget: 0 spawns; up to 2 review rounds; no full suite.

### T2 - Windows attachment status

Files: `internal/wincom/monitor.go`, its new protocol tests,
`internal/wincom/desktopwallpaper_windows.go`, `internal/displays/windows.go`,
`internal/wallpaper/target_windows.go`, existing adapter tests, `qodana.yaml`.
Contract: shared `MonitorAttached` interprets native rectangle-query results;
native calls and COM lifetime remain in the existing Windows files.
Test: native success, detached status, and failure are distinct; detached
targets cannot pass wallpaper preflight.
Verify: `go test ./internal/wincom ./internal/displays ./internal/wallpaper -count=1 && GOOS=windows GOARCH=amd64 go vet ./internal/...`
Budget: 0 spawns; up to 2 review rounds; no full suite.

### T3 - Bound repeated-source memory

Files: `internal/mosaic/generator.go`, existing generator tests,
`internal/imaging/bytecache.go`.
Contract: reuse the canonical decoded-byte estimate; a per-generator byte
budget permits deterministic small-budget tests without global test seams.
Test: over-budget sources are reloaded rather than retained, evicted sources
remain usable, and cache size does not change rendered pixels or source order.
Verify: `go test ./internal/mosaic -run 'TestGenerate_(RepeatCache|LazyPool|UsesDistinct|Deterministic)' -count=1 && go test ./internal/imaging -run 'Test(ImageBytes|LoadedImageBytes|ByteCache)' -count=1`
Budget: 0 spawns; up to 2 review rounds; no full suite.

### T4 - Frozen GIF canvas fidelity

Files: `internal/imaging/{loader,gif}.go`, existing GIF tests,
`internal/mosaic/generator_test.go`, `internal/uitest/uitest.go` (shared
partial-frame fixture).
Contract: `DecodeLoaded` retains the logical canvas and frame offset when
animation is disabled or over budget; later frames remain undecoded.
Test: a partial first frame agrees with the first fully composited frame and
with a matching PNG during mosaic generation.
Verify: `go test ./internal/imaging -run 'TestDecodeLoaded_.*(GIF|Animation|StaticFrame)' -count=1 && go test ./internal/mosaic -run TestGenerate_SourceFidelity -count=1`
Budget: 0 spawns; up to 2 review rounds; no full suite.

### T5 - Canvas-boundary masks

Files: `internal/mosaic/render.go`, existing generator/render tests and their
deterministic frame references if the verified boundary correction changes them.
Contract: cropping a render is equivalent to rendering the same card into a
larger canvas and then cropping; off-canvas card coverage does not fade.
Test: a rotated card extending beyond the canvas stays fully opaque; edge
coverage and transform-band cancellation guards remain valid.
Verify: `go test ./internal/mosaic -run 'TestRenderPlacement_|TestGenerate_(FrameStyles|Deterministic|Coverage|SourceFidelity)' -count=1`
Budget: 0 spawns; up to 2 review rounds; no full suite.

### T6 - Review, documentation, and final gate

Files: `ARCHITECTURE.md`, `todos.md`, this plan, and any affected test metadata.
Depends: T1-T5 (otherwise independent).
Verify: `make verify && GOOS=windows GOARCH=amd64 go vet ./internal/... && go test ./internal/displays ./internal/wallpaper -count=1`
Budget: 0 spawns; one final full suite after focused checks pass.

## Honest verification limit

Synthetic fixtures and native-result protocol tests never touch the desktop.
Windows cross-vet verifies compilation and build-selected code, not physical
display behavior. The existing supervised multi-display smoke matrix remains
open in the tracker.

## Cost and results

| Task | Spawns budget/actual | Review rounds | Full suite | Evidence |
| --- | --- | --- | --- | --- |
| T1 | 0 / 0 | 1 of 2 | no | Target identity regression red, then green; display and translation checks pass. |
| T2 | 0 / 0 | 1 of 2 | no | Detached native status regression red, then green; adapter tests and Windows cross-vet pass. |
| T3 | 0 / 0 | 2 of 2 | no | Repeat retention regression red, then green; byte accounting checks pass. Distinct source colors strengthen the pixel comparison. |
| T4 | 0 / 0 | 1 of 2 | no | Decoder and generator canvas regressions red, then green; existing GIF and source-fidelity checks pass. |
| T5 | 0 / 0 | 1 of 2 | no | Clipping regression red, then green; frame references updated after visual and pixel comparison. |
| T6 | 0 / 0 | 1 | one, passed | `make verify`, Windows cross-vet, native macOS adapter tests, and diff whitespace checks pass. |

Observed negative verification:

- T1 initially chose ID `two` although the snapshot default was `one`. The
  passing test now covers both choices, reordered refresh, and the wallpaper ID.
- T2 initially classified `S_FALSE` as attached. The shared protocol test also
  covers native success and documented failures. Enumeration skips the detached
  result; the wallpaper preflight returns an error before the setter runs.
  Native status semantics follow Microsoft's
  [GetMonitorRECT documentation](https://learn.microsoft.com/en-us/windows/win32/api/shobjidl_core/nf-shobjidl_core-idesktopwallpaper-getmonitorrect).
- T3 initially loaded each source exactly twice and then retained it indefinitely.
  Both oversized-entry refusal and combined-budget eviction now force reloads,
  while distinct source images produce the same pixels with either cache budget.
- T4 initially returned a 10x20 first-frame rectangle instead of its 80x40 logical
  canvas, both with animation disabled and over budget. The matching-PNG mosaic
  comparison also failed before the fix.
- T5 initially reduced a fully covering card to alpha 224 at the clipped canvas
  corner. Rendering into a larger canvas exposed the same clipping discrepancy.
  Fully covering cards now remain opaque in either direction of rotation;
  cropped and larger-canvas renders agree within one alpha level.

Temporary before/after artifacts were generated outside the repository and
visually inspected. The four frame fingerprints change at canvas boundaries;
six interior pixels across all four fixtures differ by only one or two channel
levels from rasterization rounding. The layout and source pixels are unchanged.
Existing area-coverage, cancellation, source-fidelity, and determinism tests
pass. No UI golden screenshot was regenerated.

Final gate completed on 2026-09-05: `make verify` passed formatting, TUF root,
Qodana exclusions, vet, build, shard inventory, and the complete Linux/amd64
Docker race suite. All three main UI shards and 49 other tested packages passed;
the assets package had no tests. There were no failure or data-race markers.
Windows/amd64 `go vet ./internal/...` and native macOS display/wallpaper tests
also passed. The full suite ran once; no final-review code changes were needed.

`ARCHITECTURE.md` records the shared Windows result handling, display labels,
decoded-byte accounting, repeat cache, and GIF/mask behavior. `todos.md` records
the fixes as done and retains the physical multi-display smoke task with its
current tracker path. The user's pre-existing `.gitignore` change is untouched.
No commit was created.
