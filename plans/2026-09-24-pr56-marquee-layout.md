# PR 56: keep the grid's pointer frame stable during marquee selection

Route: Standard. The PR changes one Grid feature; review found two additional
top-bar layout paths in ranked progress and queued grouping. The lead owns the spec, review, fixes,
and final gate. No subagent edits. Budget: zero implementation spawns, one
focused test cycle per finding, CI supplies the complete race suite.

## Problem and decisions

The PR defers selection-bar visibility while a marquee is active. The ranked
progress bar can still appear or disappear during the same gesture, moving the
Grid body that supplies pointer coordinates. A queued duplicate-grouping result
can also call `syncTopBar` and show the selection bar during the drag. The
existing PR test checks bar visibility on an unmounted overlay, not the body's
actual position.

| Decision | Reason |
| --- | --- |
| Keep the mounted Grid catcher's screen position fixed until DragEnd or Escape | The driver sends drag positions against the mouse-down frame. |
| Apply the latest ranked progress when the gesture finishes | Progress remains accurate without moving the pointer frame. |
| Use Grid's existing fake host and Fyne test window | This observes layout and selection through the feature seam. |

## Acceptance criteria

1. Selection appearing or clearing during an ordinary marquee leaves the
   mounted catcher's screen position unchanged until `DragEnd`; the bar then
   reflects the final selection.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui/grid -run '^TestMarqueeDrag_DefersTopBarLayoutUntilDragEnd$' -count=1 -v`.
2. Ranked progress completing or restarting during a marquee leaves the
   mounted catcher's screen position unchanged until `DragEnd`; its latest
   state appears after release.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui/grid -run '^TestRankedProgress_DefersLayoutDuringMarquee$' -count=1 -v`.
3. Existing selection, Escape, Close, and ranked-result behavior remains valid.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui/grid -run '^(TestMarquee|TestEscape_CancelsAnInProgressMarquee|TestClose_DisarmsAnInProgressMarquee|TestRankedVisit)' -count=1`.
4. A duplicate-grouping completion delivered during a marquee does not show
   the selection bar or move the body until `DragEnd`.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui/grid -run '^TestMarqueeDrag_GroupingCompletionKeepsTopBarLayout$' -count=1 -v`.

The selection count and ranked progress text may update at mouse-up rather than
during the drag. A native desktop gesture and platform-specific Fyne layout
remain qualified by CI and manual use, respectively.

## Tasks

### Task 1 — mounted geometry guards

Owner: lead. Files: `internal/ui/grid/marquee_test.go`,
`internal/ui/grid/ranked_test.go`. Contract: no new production interface.
Test: catcher's screen position and visible bar/progress before, during, and
after drag. Verify: acceptance commands 1 and 2. Delegation gate: review and
test design need judgment, so G2 and G5 do not pass; keep inline.

### Task 2 — defer ranked progress presentation

Owner: lead. Files: `internal/ui/grid/grid.go`, `internal/ui/grid/ranked.go`.
Contract: `SetRankedProgress` retains the latest progress while dragging and
`flushRanked` applies it after any pending ranked visit. Clear that pending
state on Grid close. Verify: acceptance commands 2 and 3. Delegation gate:
cross-function state and review fix fail G3 and G5; keep inline.

### Task 3 — guard the shared top-bar path

Owner: lead. Files: `internal/ui/grid/search.go`,
`internal/ui/grid/marquee.go`, `internal/ui/grid/marquee_test.go`.
Contract: every `syncTopBar` caller defers layout during a live marquee;
Escape clears the drag state before synchronizing the restored selection.
Test: deliver a queued grouping result during a mounted drag. Verify:
acceptance commands 3 and 4. Delegation gate: cross-caller review fix and
current context fail G3 and G5; keep inline.

The ranked guard must fail on the original PR before Task 2, and the queued
grouping guard before Task 3. After focused tests and GoLand inspections,
review the pushed head through Codex, security review, Qodana, CodeQL, and
required CI before handoff.

## Evidence and cost ledger

| Task | Spawns (budget/actual) | Review rounds | Full local suite | Notes |
| --- | --- | --- | --- | --- |
| Recon | 1 / 1 | — | no | Read-only Scout on Grid event flow. |
| Task 1 | 0 / 0 | 1 | no | Mounted geometry guards pass; the ordinary guard failed on the deliberately restored original sync (39 px shift in both directions). |
| Task 2 | 0 / 0 | 1 | no | Ranked guard failed on the PR head (39 px shift), then passed after deferring progress. Grid package tests and GoLand inspections pass. |
| Task 3 | 0 / 0 | 1 | no | Grouping guard failed before the shared toolbar guard (39 px shift), then passed. |
| Gate | — | 1 clean remote round | no | `make verify-build` and Grid package tests passed locally; all required CI passed at `828370c`. |

At `828370c`, the fresh Codex code and security reviews completed without
findings; PR 56 had no review threads. CI run 36009340002 passed all platform
and race jobs. Qodana run 36009340031's post-suppression SARIF had zero results;
both analyses in CodeQL run 36009340010 had zero results. The PR's live checks
are the source for the latest head after this evidence record is committed.
