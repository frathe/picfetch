# PR 57 Close Files review

## Frame and spec

Route: Standard. The deliverable is a corrected PR 57 with the Close Files menu
accurate through initial scan/sort cancellation, a fresh clean Codex review,
security review, and required CI. The PR description and
`finished_refactorings/2026-09-24-close-files-active-work.md` are the originating
spec; the Codex P2 thread identifies the cancellation gap.

Problem: Escape clears `scanOp.active` or `sortOp.active` during the first load,
but does not publish the resulting menu state. The stale worker completion
returns without publishing it either. Close Files can therefore remain enabled
in an idle, empty viewer. The PR's sort completion also refreshes the menu
after starting an image load; under Fyne's inline test driver that races with
the load callback.

Decisions:

| Decision | Reason |
| --- | --- |
| Test the Escape dispatch and the observable File menu item in existing UI tests. | This is the user's actual cancellation path and avoids new shard assignments. |
| Synchronize the menu after successful scan and sort cancellation. | The cancellation paths own the transition from active to idle. |
| Refresh the sort-completion menu before invoking its callback. | The callback can start an image load; loading publishes its own final menu state. |
| Retain the existing Close Files behavior during active work. | The PR's intended way to cancel remains available. |

Acceptance criteria and evidence commands:

1. Close Files is enabled during a first scan/sort and disabled after Escape
   cancels it, without closing the window or installing a stale result.
   `go test -tags no_emoji,nodynamic ./internal/ui -run '^(TestCancelScan_CancelsInFlightScanWithNoFilesYet|TestHandleKeyEvent_EscapeDuringFirstDropReorderDoesNotCloseWindow)$' -count=1`
2. The PR's idle, active, loaded, and other File menu matrix cases still pass.
   `go test -tags no_emoji,nodynamic ./internal/ui ./internal/ui/menus -run '^(TestCloseFilesItem_.*|TestApply_FileItems)$' -count=1`
3. Sort completion does not race with image presentation under the Fyne test
   driver. `go test -race -tags no_emoji,nodynamic ./internal/ui -run '^TestShutdownClosesComparisonWithoutRefreshingRetiredUI$' -count=1`
4. The latest pushed commit has a fresh Codex review without findings, a
   completed security review without actionable findings, no actionable
   post-suppression Qodana/CodeQL findings, and required CI green.
   Evidence: current-head GitHub reviews, review threads, checks, and SARIF.

This does not change the meaning of Escape, sorting, scan limits, or file
admission. Automated tests cover the item state; native menu rendering still
needs a manual platform check.

## Tasks and delegation

Dependency: inspect PR -> regression red -> minimal fix -> focused verification
-> lead review -> commit/push -> fresh GitHub review and CI loop.

| Task | Owner | Files | Verification | Budget |
| --- | --- | --- | --- | --- |
| Collect GitHub threads, reports, and checks | Read-only scout | None | Lead checks IDs and current-head status | 1 spawn |
| Correct cancellation state and race | Lead | `internal/ui/drop.go`, `sort.go`, `drop_test.go`, `sort_test.go`, `menu_test.go` | AC1-3, red/green and focused race | 0 spawns, 1 review |
| Finish evidence and loop | Lead | This plan, `todos.md`, originating spec | AC4, changed-file inspections, focused tests | CI owns full race suite |

The GitHub inventory is an independent, read-only task with a bounded factual
result and no shared files (delegation gate G1-G5). The Lead reviews findings,
edits code, and owns the final gate. The PR loop's local-test and commit rules
in `AGENTS.md` take precedence over the general SDD handoff gate.

## Evidence and ledger

| Task | Spawns budget/actual | Review rounds | Full local suite | Notes |
| --- | --- | --- | --- | --- |
| GitHub inventory | 1 / 1 | Lead assessed | No | Codex P2 on Escape cancellation; initial post-suppression Qodana and CodeQL clear; security review clear. |
| Fix | 0 / 0 | 1 lead review | No | Cancellation red/green and focused race correction complete. |
| GitHub loop | 0 / 0 | See PR review history | CI owns broad race suite | GitHub records the latest-head reviews and checks. |

Local evidence:

- On the original PR head, the Escape regression assertions failed in both
  scan and sort cases because Close Files remained enabled. Both passed after
  the cancellation paths synchronized menus.
- The original PR head failed all three Linux UI race shards. A focused race
  run reproduced the new `finishSort` menu refresh racing with image
  presentation. Moving the refresh before `onDone` made all eleven tests that
  had failed across the three shards pass locally under `-race`.
- Focused Close Files and menu-matrix tests pass. No new top-level UI test or
  test file was added, so shard and Qodana exclusion manifests are unchanged.
- Removing the scan-start menu refresh made the held real-scan guard fail at
  its expected assertion; restoring the refresh made it pass again.
- `make check-test-shards verify-build` passed: 704 runnable UI tests in three
  shards, format/notices checks, vet, and build. GoLand reported no warnings
  in the eight changed Go files. `git diff --check` and `gofmt -l` were clean.
