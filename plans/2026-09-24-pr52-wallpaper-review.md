# PR 52 mosaic wallpaper review

## Frame and spec

Route: Deep. This is a cross-package security correction at the UI/native
wallpaper boundary. The deliverable is a PR whose selected-display authorization
is checked after the PNG copy is written, with a clean fresh Codex review and CI.
PR 52's description and the unresolved Codex thread are the originating spec;
the established global fallback contract is in
`finished_refactorings/2026-09-12-mosaic-global-wallpaper.md`.

Problem: `mosaicwin.Window.SetWallpaper` refreshes the displays before queueing a
worker. The worker then encodes a PNG before the Linux wallpaper call. A second
display attached during that interval makes the previously calculated `Solo`
authorization stale and may change wallpaper on every display.

Decisions:

| Decision | Reason |
| --- | --- |
| Keep the window's click-time refresh. | It updates the target selector and gives immediate feedback when a display is gone. |
| Derive `Solo` from a new inspection in the viewer after the PNG write, just before platform dispatch. | The authorization must reflect the latest topology at the effect boundary. |
| Refuse dispatch if inspection fails or the selected target is gone; remove the unused copy. | Uncertain targeting must not create a global change or leave a cache file. |
| Keep an empty target as the existing explicitly chosen all-displays action. | It needs no single-display authorization. |

Acceptance criteria and evidence commands:

1. Click-time refresh uses current topology and performs the requested action
   when the target remains attached; failure or lost target performs no action.
   `go test -tags no_emoji,nodynamic ./internal/ui/mosaicwin -run 'TestMosaicWallpaper_' -count=1`
2. A topology change during PNG encoding cannot carry stale `Solo` to the OS;
   the copy exists before the final inspection. A lost target or inspection
   error prevents dispatch and removes the new copy.
   `go test -tags no_emoji,nodynamic ./internal/ui -run 'TestMosaicWallpaper_' -count=1`
3. A current single-display target still works and the explicit global action
   retains its existing behavior.
   `go test -tags no_emoji,nodynamic ./internal/ui/mosaicwin ./internal/ui ./internal/wallpaper -run 'TestMosaicWallpaper_|TestLinux' -count=1`
4. The latest pushed commit has a fresh clean Codex code and security review,
   no actionable post-suppression Qodana/CodeQL findings, and required CI green.
   Evidence: `gh pr view 52` and the current-head review threads/checks/SARIF.

The remaining limit is the unavoidable interval between the final OS topology
inspection and the desktop's wallpaper effect; the available APIs provide no
atomic topology-and-wallpaper operation. Native multi-display behavior still
requires a manual Linux check.

## Tasks

Dependency: evidence -> regression -> minimal fix -> focused verification ->
lead review -> commit/push -> fresh GitHub review and CI loop.

| Task | Owner | Files | Test and verification | Budget |
| --- | --- | --- | --- | --- |
| Collect PR threads, checks and SARIF | Read-only scouts | None | Lead validates current-head IDs/status | 2 spawns, no review delegation |
| Correct the regression and effect boundary | Lead | `internal/ui/mosaicwin/window.go`, `mosaicwin_test.go`, `internal/ui/wallpaper.go`, `wallpaper_test.go` | AC1-3; one red/green slice at a time | 0 spawns, 1 lead review |
| Finish record and GitHub loop | Lead | This plan, `todos.md` | AC4; changed-file GoLand inspections, focused tests, CI | No broad local race suite |

The delegated evidence collection was read-only, bounded, and independent of
the code changes (G1-G5). The Lead assesses all findings and applies every fix.
PR-loop verification and commit rules in `AGENTS.md` supersede the general
SDD/TDD handoff gate.

## Evidence and ledger

| Task | Spawns budget/actual | Review rounds | Full local suite | Notes |
| --- | --- | --- | --- | --- |
| Evidence | 2 / 2 | Lead assessed | No | One confirmed Codex P2 thread; initial Qodana/CodeQL clear. |
| Fix | 0 / 0 | 1 lead review | No | Red/green and negative guard checks complete. |
| GitHub loop | 0 / 0 | Pending | CI owns broad race suite | Commit, fresh review, and CI pending. |

Local evidence:

- The first new test failed on the PR head with zero final display inspections,
  then passed after the viewer rechecked after PNG encoding.
- The failure-path test failed for both lost-target and inspection-error cases
  when their guards were temporarily disabled. The window click-time test
  likewise failed both cases when its guard was disabled. All guards were
  restored, and the focused tests passed.
- Focused `-race` tests passed in `internal/ui/mosaicwin`, `internal/ui`, and
  `internal/wallpaper`; full `mosaicwin` and `wallpaper` package tests passed.
- `make check-test-shards` accepted 704 root UI tests in three shards.
  `make verify-build` passed format, notices, vet, and build checks.
- GoLand found no errors or warnings in the four changed Go files after the
  final test edit. The complete broad race suite remains GitHub CI's gate.
