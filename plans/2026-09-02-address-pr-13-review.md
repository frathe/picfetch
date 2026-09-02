# Address PR 13 review findings

Route: Standard, promoted from Thin when the review crossed from
`internal/ui/compare` into the assembled viewer. The diff exceeds the usual
eight-file guideline because each of the four independent review findings has
its own regression/configuration coverage; it adds no package, public API, or
architectural seam.

Deliverable: Close all four Codex review findings on PR 13 and the six Qodana
conversion notices without reintroducing comparison repaints.

## Spec

### Problem

- Swipe divider changes update Fyne clips but do not publish new reveal bounds
  to the detail-tile planner.
- Source-normalized detail steps can become unusable in a GLES fragment
  shader's `mediump` fallback for very large sources.
- The physical `Ctrl+L` hook can mutate comparison while a Fyne overlay owns
  keyboard input.
- The branch added six test files without exact `DuplicatedCode` exclusions;
  Qodana also reports six redundant rune conversions in `shader_test.go`.

### Decisions

| Question | Decision |
|---|---|
| Divider update seam | Store each pane's last complete scene and republish it with current reveal bounds and physical pane origin. Do not repaint the owner or recompute transforms. |
| GLES precision | Bind detail tiles in physical pane-pixel coordinates. Keep normalized coordinates only for the overview fallback. |
| Modal ownership | Apply the normal dispatcher's `Overlays().Top()` guard to the low-level physical-key hook. |
| Qodana | Synchronize every test path mechanically and use decimal slot names instead of rune conversions. |

### Acceptance criteria

1. Divider keys and drags republish reveal-aware scenes while the owner repaint
   count and image transform stay unchanged.
   Verify: `go test ./internal/ui/compare -run 'TestPaneRendererScene_Divider|TestCompareSwipePointer_DividerDragDoesNotRefreshStaticContent' -count=1`
2. A visible level-zero tile from a 32768-pixel source binds a representable
   pane-pixel step, retained tiles update their geometry, and both shader
   variants share the same contract.
   Verify: `go test ./internal/ui/compare -run 'TestTileShaderSources|TestShaderPaneRenderer_LargeSource|TestShaderPaneRenderer_MapsScene' -count=1`
3. Exact physical `Ctrl+L` is inert while a canvas overlay is present and works
   after the overlay is removed.
   Verify: `go test ./internal/ui -run 'TestCompareLinkToggle_.*Overlay' -count=1`
4. Every current `_test.go` has one exact duplication exclusion, and the six
   reported rune conversions are absent.
   Verify: `make check-qodana-test-exclusions` and
   `! rg "string\\(rune" internal/ui/compare/shader_test.go`

### Non-goals and honest limit

- No renderer replacement, tile-cache redesign, or runtime precision query.
- Fyne's software test driver does not execute the native GLES shader. The
  source/uniform contract and large-source arithmetic are covered locally;
  a fresh hosted Qodana result requires pushing the local commit.

## Task graph

Tasks 1 through 4 are independent and all feed the final gate. Delegation is
disabled for this run, so every task remains T0 inline.

### Task 1 - Refresh divider scenes

Owner: T0 inline
Files: modify/test `internal/ui/compare/{compare,transform,swipe,renderer_test}.go`
Depends: none
Contract: `pane.present(paneScene)` records the latest snapshot;
`Feature.applyReveal()` republishes current reveal geometry.
Test: A divider move adds one scene per pane without changing image geometry.
Verify: Acceptance criterion 1.
Budget: <= 0 spawns, <= 1 review round, full suite: no.

### Task 2 - Make detail coordinates mediump-safe

Owner: T0 inline
Files: modify/test `internal/ui/compare/{renderer,shader,shader_test,transform,vector}.go`
Depends: none
Contract: `paneScene.panePosition` is physical; detail `Min` and `Step`
uniforms are pane pixels; overview lookup remains normalized.
Test: A 32768-pixel source exposes a one-pixel detail step and shader source
contains no normalized detail lookup or unit clamp.
Verify: Acceptance criterion 2.
Budget: <= 0 spawns, <= 2 review rounds, full suite: no.

### Task 3 - Isolate the physical link hook

Owner: T0 inline
Files: modify/test `internal/ui/{keys.go,compare_test.go}`
Depends: none
Contract: `wireComparisonLinkToggleHook` calls `ToggleLink` only with no top
canvas overlay.
Test: An overlay blocks physical `Ctrl+L`; removing it restores the shortcut.
Verify: Acceptance criterion 3.
Budget: <= 0 spawns, <= 1 review round, full suite: no.

### Task 4 - Close Qodana findings and bookkeeping

Owner: T0 inline
Files: modify `qodana.yaml`, `Makefile`, `todos.md`, and this plan; modify/test
`internal/ui/compare/shader_test.go`.
Depends: none
Contract: `make check-qodana-test-exclusions` rejects missing/stale exact test
paths; shader slot names use `strconv.Itoa`.
Test: The exclusion check passes and the focused shader contract test passes.
Verify: Acceptance criterion 4.
Budget: <= 0 spawns, <= 1 review round, full suite: no.

### Final gate

Owner: T0 inline
Files: all changed files
Depends: tasks 1, 2, 3, and 4
Contract: repository verification remains green.
Test: All repository tests under the race detector.
Verify: `make verify`
Budget: <= 0 spawns, <= 1 review round, full suite: yes.

## Outcome and evidence

- Each behavioral regression was observed red for the review's stated reason
  before its production fix. A later source-contract guard also caught and
  removed the stale normalized `1.0` tile-maximum clamp.
- All four acceptance commands pass.
- `go test ./internal/ui/compare -count=1` and
  `go test ./internal/ui -run 'Compare|Comparison' -count=1` pass.
- `make verify` passes, including formatting, TUF validation, Qodana exclusion
  consistency, vet, build, and the Linux/amd64 race suite.
- The implementation was committed concurrently as local `8c83586`; the agent
  did not create that commit. Concurrent release-signing work is outside this
  plan and remains untouched.

## Cost ledger

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|---|---:|---:|---:|---|
| T1 | 0 / 0 | 1 | no | Inline hot context |
| T2 | 0 / 0 | 2 | no | Second pass caught stale unit clamp |
| T3 | 0 / 0 | 1 | no | Inline hot context |
| T4 | 0 / 0 | 1 | no | Config edits arrived concurrently and were validated |
| gate | 0 / 0 | 1 | yes | Passed once |
