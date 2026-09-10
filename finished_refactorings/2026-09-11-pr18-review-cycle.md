# PR 18 review fixes

Deliverable: fix the initial Codex/Qodana findings with regression evidence and
document the reusable GitHub Cortex review loop. The implementation is complete;
subsequent review/CI outcomes are recorded in the [PR conversation](https://github.com/frathe/picfetch/pull/18).

Route: Deep for this batch across the viewer, worker, and platform tooling.
Existing architecture and behavior remain the contract. No merge or release.
The user's session authorization permits commits, pushes, and review replies.

## Task 1 — Viewer navigation and collection transactions

Owner: T0 inline.
Files: internal/ui/{drop,explorer,shortcuts}.go, grid/nav.go, menus/menus.go;
existing UI/menu tests and any affected direct scan callers.
Contract: trial exit displays a current image; active analysis is not restarted
from its menu; cohort Escape preserves standing duplicate hiding; Favorites
shortcuts follow their enabled menu; explicit favorite membership suppresses
sibling expansion; favorite identity lands with a successful file-set reorder.
Verify: `go test ./internal/ui ./internal/ui/menus ./internal/ui/grid -run 'TestVisualSimilarityExplorer|TestApply|TestLaunch|TestFileState|TestGrid'`.
Budget: one read-only test-location Scout; all decisions/fixes remain T0.

## Task 2 — Worker completion and cancellation

Owner: T0 inline. Depends: none.
Files: internal/similarity/{analyze,assets}.go and package tests.
Contract: zero successful sources cannot publish completed success; asset hash
reads observe cancellation between bounded reads.
Verify: `go test ./internal/similarity` with regressions observed red first.
Budget: zero additional spawns; one final full suite for the batch.

## Task 3 — Profiling platform admission

Owner: T0 inline. Depends: none.
Files: scripts/explorereval/main.go and existing tests.
Contract: production throughput is admitted on supported Linux amd64 as well
as Apple Silicon; smoke/library retain their native platform restrictions.
Verify: `go test ./scripts/explorereval` with admission regression red first.
Budget: zero additional spawns.

## Task 4 — Documentation and review disposition

Owner: T0 inline. Depends: 1–3.
Files: AGENTS.md, ARCHITECTURE.md if bootstrap responsibilities change, todos.md.
Contract: document Explorer queue admission/cancellation/settlement; assess the
entry-point concern against pre-app identity/offline requirements; each thread
gets evidence supporting its fix or rejection.
Verify: focused changed tests locally, `git diff --check`, and GitHub full-suite
CI/review results for the pushed commit, per the user's updated instruction.

Independent tasks: 1, 2, 3. Root implements each; the Scout only maps existing
tests across packages while root validates findings. Delegation gate: bounded
read-only question, shell-verifiable file:line/test names, no writes or shared
edits, broad unfamiliar test inventory, no duplicated review context.

## Evidence and cost ledger

Initial review: all 11 unresolved Codex findings confirmed, with a narrower
menu fix that retains retry after analysis failure. The entry-point fix moves
trial validation/identity into `launch.Options.ApplicationID`. Qodana's sole
warning is the runtime value used before checking its error; the read is now
guarded. Current-head CI and CodeQL were green, with no open security alerts.

Regression evidence:
- Viewer tests failed on all reported behaviors before the fixes: active menu,
  cohort Escape, three Favorites shortcuts, cancelled scan/sort identity,
  unloaded trial exit, and single-file favorite sibling expansion.
- Asset checksum cancellation, all-failed map publication, and Linux throughput
  admission each failed for the reported cause, then passed.
- `go test ./internal/similarity ./scripts/explorereval ./internal/launch -count=1`
  passed all three packages.
- Final changed UI regression selection passed (`internal/ui`, 2.798s), including
  map-return recording and direct scan/sort completion callers.
- Expanded Explorer tests exposed duplicate map-return recording in the first
  Escape fix; corrected it to close the subset through the existing key path.
- `go test -tags explorertrial ./scripts/explorereval -run '^TestProductionProfile$'
  passed on Linux amd64 with real local assets (5.592s), including cold/warm
  reuse, cancellation, and the bounded 512-failure case with no success summary.
- The user requested changed tests locally and full-suite verification in CI.
  Stopped the broad local race run accordingly; it had reported no failures.
  Formatting, TUF/Qodana inventory, vet, build and shard inventory had passed.
  The interrupted local race evidence is under
  `.scratch/race-runs/20260910T221634Z-2Esloi` and `/tmp/pr18-verify.log`.

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|---|---|---|---|---|
| Test inventory | 1 / 1 | 0 | no | Read-only Scout; root validated findings |
| External reports | 0 / 1 | 0 | no | Scope expanded by user to Qodana/CodeQL/CI; reused Scout |
| Fixes | 0 / 0 | 1 | no | All fixes and regression design stayed with root |
| Gate | 0 / 0 | 1 | CI | Local broad run stopped on user request; inspect GitHub CI |

## External follow-up

- `4ab72aa` contains the initial fixes. All 11 original Codex threads received
  individual commit-linked explanations and were resolved.
- `55dbf16` documents the reusable GitHub Cortex review loop in `AGENTS.md`,
  including its explicit commit/push/comment authorization and focused local tests.
- Codex code and security reviews completed on `55dbf16` with no new findings.
  Validation, Windows, macOS, and non-UI race checks passed; UI race jobs were
  still running when the Qodana follow-up was prepared.
- The full Qodana report for run `34537395806` confirmed the original unchecked
  runtime warning was gone. It reported one new `GoImportUsedAsName` warning:
  local `favorites` shadowed the import. Renamed it to `favoriteBindings` and
  reran `TestVisualSimilarityExplorer/favorite_shortcuts` successfully (0.466s).
- The user requested that this workflow remain repeatable in `AGENTS.md`.
  Current-commit CI, Qodana, CodeQL and Codex results remain the external exit
  gate; the agent continues monitoring and correcting them in the PR loop.

The additional report fetch reused the same read-only Scout; root assessed and
fixed its finding. No implementation or review was delegated.
