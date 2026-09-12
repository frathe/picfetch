# Mosaic wallpaper on Linux with multiple displays

## Problem and scope

Ronin reports Ubuntu 24.04 ARM64 can set a viewer image as wallpaper but the
mosaic reports that this desktop cannot set wallpaper for one display. The ARM64
machine has two displays; the working x64 machine has one. The existing Linux
adapter deliberately rejects a nonempty target unless `Solo` is true. Both
architectures share this code and release build path.

Route: Deep (existing platform behavior, two UI packages). Deliver an explicit
global wallpaper alternative after the desktop reports targeted wallpaper is
unsupported. Keep the generated pixels and selected output dimensions intact.
No new dependencies or changes to platform adapters, display detection, native
per-display support, or packaging. No commits or publishing are authorized.

## Decisions and acceptance

- First try the selected display as today. A typed target-unsupported result
  relabels the existing preview button to `Set on All Displays`, explains the
  alternative, and preserves Save Image. No global change happens until the
  user clicks the relabelled button. Other errors must not offer this fallback.
- That explicit click sends the same immutable mosaic with an empty target and
  `Solo=false` through the existing global wallpaper write/set/cleanup path.
  Never turn a missing selected target into an implicit global request.
- Scope the fallback to the open mosaic session; closing resets it. A stale
  worker completion cannot relabel a reopened window or change its status.
- Preserve targeted and single-display requests, busy/error handling, and
  English/German translation parity.

Acceptance commands:

1. UI behavior and lifecycle:
   `go test ./internal/ui/mosaicwin -run TestMosaicWallpaper -count=1`.
2. Captured immutable PNG, explicit global dispatch, existing target semantics:
   `go test ./internal/ui -run TestMosaicWallpaper -count=1`.
3. Linux rejection/solo/global backend coverage:
   `go test ./internal/wallpaper -count=1`.
4. Locale parity: `go test . -run TestTranslations -count=1`.
5. Final gate: GoLand inspections of all changed Go files, then `make verify`.

Native wallpaper application on Ronin's two-display Ubuntu desktop requires a
manual check; tests replace OS effects and cannot establish that outcome.
Ronin confirmed that native outcome on September 12, 2026 (see acceptance below).

## Tasks and routing

Task graph: build-path scout alongside lead reproduction; then tests -> fix ->
focused checks -> lead review and final gate -> evidence/todos.

| Task | Owner | Files | Verification | Budget |
| --- | --- | --- | --- | --- |
| Build-path evidence | Read-only scout | No edits; Makefile, release workflow, historical ARM plan | Lead checks reported build paths and OS-only adapter dispatch | 1 spawn, no review delegation |
| Regression and fix | Lead | `internal/ui/mosaicwin/window.go`, existing mosaic and viewer wallpaper test files, `internal/ui/wallpaper.go`, both locale bundles | AC1-4, red then green | 0 spawns, 1 review plus fixes |
| Handoff evidence | Lead | English/German manuals, this plan, `todos.md` | Inspections, `make verify`, final diff | Full suite once |

Scout gate: bounded read-only question (G1), exact build-path lines checked with
shell (G2), no writes (G3), independent packaging context (G4/G5). It requires
following build and historical evidence across files rather than a mechanical
transform (S); lead retains all design, implementation and review (W).

## Evidence

- Existing tests reproduce the deliberate restriction: the unsupported mosaic
  case and `TestLinuxTarget` pass with the screenshot's refusal semantics.
- Red: the new mosaic regression failed with the exact reported status and
  unchanged `Set as Wallpaper` button. Explicit global host coverage failed
  with `mosaic wallpaper requires a target display`. Missing-target coverage
  caught an empty target reaching the host, which must remain blocked after
  the host starts accepting explicit global requests.
- Green: focused `TestMosaicWallpaper`, `TestLinuxTarget`, and `TestTranslations`
  pass in all four affected test packages on the host.
- Mutation checks fail as intended when ordinary errors enable global scope,
  stale callbacks lose their revision guard, or Close retains global scope.
  All deliberate mutations were restored before verification.
- Full host package checks pass for `internal/ui/mosaicwin`, `internal/wallpaper`
  and `internal/ui/help`; focused viewer wallpaper race tests also pass.
- GoLand inspected all four changed Go files including weak warnings. No new
  findings. The mosaic test file retains two pre-existing duplicate fragments
  at lines 113 and 158, outside this change and already excluded by the exact
  `DuplicatedCode` path in `qodana.yaml:262`; no broader suppression was added.
- Visual captures at 760x620 reviewed for English and German. The German
  explanation initially clipped; the preview status now wraps by words and
  both the full message and all actions fit. Captures and an isolated Go overlay
  harness are in ignored `.scratch/mosaic-wallpaper/`; no golden files changed.
- `make verify-build` passes after the final wrapping change (format, TUF,
  Qodana test exclusions, vet, build). `make check-test-shards` confirms all 682
  root UI tests have assignments; no root top-level test or test file was added.
- Full Docker race gate completed. All three UI shards pass: 681 top-level
  tests passed and one skipped, covering all 682 assigned tests. The affected
  mosaic window, wallpaper and help packages pass with `-race`. Non-UI results
  are 1,922 top-level passes, four skips and two failures. The only failures
  are the already-recorded local amd64 seccomp failures in
  `TestLinuxWorkerIsolation` and `TestAssetInstall/worker_reaches_asset_check_and_exits`,
  both reporting `offline worker seccomp: invalid argument`.
  `make verify` therefore exits 2; a clean complete gate is not claimed.
  Raw evidence: `.scratch/race-runs/20260912T180206Z-tmKJNs/`.
- Final implementation diff and whitespace review complete. The agent performed
  no native desktop effect, commit, push or release during implementation.

## Native acceptance — September 12, 2026

Ronin confirmed the global mosaic wallpaper action on native Ubuntu 24.04
hardware with two displays. This closes the outstanding native confirmation
item in `todos.md`; the accepted plan is archived in `finished_refactorings/`.
The previously recorded automated verification results and local seccomp
failures remain unchanged.

## Ledger

| Task | Spawns budget/actual | Review rounds | Full suite |
| --- | --- | --- | --- |
| Scout | 1 / 1 | Lead checks citations | No |
| Fix | 0 / 0 | 1 plus visual wrapping fix | No |
| Final gate | 0 / 0 | 1 | Once; only known seccomp failures |
