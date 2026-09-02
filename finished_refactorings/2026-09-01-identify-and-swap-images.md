# Identify and swap compared images

Status: complete

Route: Standard. This ticket extends the existing comparison feature and its
viewer adapter, then adds localized UI/manual wording. It introduces no new
package, interface to grid state, or platform behavior.

Deliverable: An active comparison permanently identifies both sides, shows a
top-right Swap/Back toolbar, and can exchange left and right without decoding
again or changing the covered grid.

Source ticket:
`.scratch/compare-two-grid-selected-images/issues/02-identify-and-swap-images.md`

## Locked decisions

| Decision | Contract |
|---|---|
| Test seam | Exercise `internal/ui/compare.Feature` through `Open`, `Overlay`, `Ready`, `Done`, and real button taps; exercise title/restoration and grid preservation through the assembled viewer. |
| Feature ownership | `compare.Feature` owns ordered sources, displayed frames, badge identities, toolbar enablement, and Swap. It still never reads grid/viewer state. |
| Title callback | `Callbacks.OrderChanged(left, right)` reports the same identities painted in the badges. `internal/ui` writes the exact comparison title directly and `Callbacks.Closed` reapplies the covered grid title. |
| Identity rule | Different basenames stay basenames. Equal basenames expand both URI paths from two trailing components outward until the suffixes differ. |
| Presentation | One rounded translucent card anchors at each bottom corner; one rounded translucent toolbar card anchors at the top right. Back remains enabled; Swap enables only when both loads validate. |
| Swap | A ready-only UI operation exchanges sources, decoded frames, badges, and the reported order in place. It starts no loader and leaves fitted side-by-side geometry unchanged. |

## Acceptance commands

1. Toolbar: `go test ./internal/ui/... -run 'CompareToolbar' -count=1`
2. Identities: `go test ./internal/ui/... -run 'CompareIdentity' -count=1`
3. Title and restoration: `go test ./internal/ui/... -run 'CompareTitle' -count=1`
4. Swap roles/no reload: `go test ./internal/ui/... -run 'CompareSwap' -count=1`
5. Covered grid preservation: `go test ./internal/ui/... -run 'CompareSwapPreservesGrid' -count=1`
6. Locales/manuals: `go test ./... -run 'Translations|Manual' -count=1`

Non-goals: Swipe, linked zoom/pan, resize/transition preservation beyond the
currently fitted side-by-side layout, command isolation, and source-fidelity
work assigned to tickets 03 through 07.

Honest limit: Equal source paths have no distinguishing suffix; in that
degenerate case both badges remain the same full available suffix because the
comparison has no distinct path identity to display.

## Tasks

### T1 - Toolbar, badges, and identity expansion

Owner: T0 inline

Files: modify `internal/ui/compare/compare.go` and
`internal/ui/compare/compare_test.go`.

Contract: construct the three translucent corner cards; keep Back enabled and
Swap disabled until ready; compute and paint the ordered identities.

Test: loading/ready toolbar state, corner placement/translucency, distinct
basenames, and shortest distinguishing equal-basename suffixes.

Verify: acceptance commands 1 and 2.

Budget: 0 spawns; at most 2 review rounds; no full suite.

### T2 - Swap and viewer title lifecycle

Owner: T0 inline

Files: modify `internal/ui/compare/compare.go`,
`internal/ui/compare/compare_test.go`, `internal/ui/compare.go`,
`internal/ui/features.go`, `internal/ui/viewer.go`, and
`internal/ui/compare_test.go`.

Contract: `Callbacks.OrderChanged(left, right)` fires on Open and Swap; Swap
exchanges ready roles without a loader call; viewer title is exact while
active and the ordinary grid title is reapplied by `Callbacks.Closed`.

Test: visible frames/badges/title exchange, load count remains two, title
restores on Back/Escape/failure, and grid files/selection/filter/highlight/
scroll plus the selection anchor remain unchanged.

Verify: acceptance commands 3 through 5.

Budget: 0 spawns; at most 3 review rounds; no full suite.

### T3 - Localization, manuals, and landing records

Owner: T0 inline

Files: modify both `translations/*.json`, both manuals and their guard test,
`ARCHITECTURE.md`, `todos.md`, this ticket, and this plan.

Contract: localize Swap and the title format; document identity, readiness,
Swap, and unchanged-grid return in both manuals; mark only ticket 02 complete.

Test: locale parity, English identity, bilingual manual guard, and the existing
Unicode-arrow guards.

Verify: acceptance command 6.

Budget: 0 spawns; at most 2 review rounds; no full suite.

## Delegation gate

No task is delegated. T1 fixes presentation semantics, T2 holds the hot
cross-package contract, and T3 contains user-visible strings; G4/G5 and the
workflow's never-delegate rules keep all three inline.

## Final gate

Run once after the focused acceptance commands and negative guard checks:
`make verify`.

## Cost ledger

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
|---|---:|---:|---|---|
| T1 | 0 / 0 | 2 | no | Added root-directory identity case; repaired the old backdrop probe for multiple rectangles. |
| T2 | 0 / 0 | 2 | no | Exact title ownership and selection-anchor preservation verified. |
| T3 | 0 / 0 | 1 | no | Locale parity and bilingual manual guard verified. |
| gate | - | - | - | yes | `make verify` passed; Linux/amd64 `internal/ui` race suite 576.016s. |
