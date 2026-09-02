# Isolate comparison commands

Route: Standard. One bounded feature across `internal/ui` and its existing
`menus`, `favorites`, and `compare` feature packages; locale/manual edits are
mechanical follow-through. No new package, Host interface, or controller.

Deliverable: while the comparison surface is visible, every ordinary command
entry is inert, open-file inputs are refused, Help and comparison exit still
work, and pointer/typed input cannot reach the covered grid or viewer.

## Decisions

| Decision | Resolution |
|---|---|
| Exclusive-mode fact | `compare.Feature.Visible()`; `internal/ui` owns composition and policy. |
| Command seam | Guard the viewer's real menu, shortcut, keyboard, Host, and open-input adapters; do not teach feature packages about each other. |
| Menu state | Add comparison activity to the existing immutable `menus.State` snapshot; disable ordinary File/Actions/Window items and every Favorites item, leaving Help available. |
| Pointer input | A transparent comparison-owned input shield participates in Fyne hit-testing ahead of the covered grid/viewer. |
| Open input | One localized refusal helper is used by dialog, drop, and Open With paths; input is discarded, never queued. |

The TDD seams are the already-approved user-facing interfaces: production menu
items, shortcut handlers, canvas key/pointer dispatch, viewer Host methods, and
the three file-open adapters. Package-local widget fields are not a test seam.

## Acceptance criteria

1. Escape and F1 work during comparison; F1 leaves comparison active.
   Verify: `go test ./internal/ui/... -run 'Compare(AllowedCommands|Help)' -count=1`
2. Menus disable ordinary commands and every non-help command entry is inert.
   Verify: `go test ./internal/ui/... -run 'Compare(CommandIsolation|MenuState|CommandEntryPoints)' -count=1`
3. Typed and pointer input cannot reach covered surfaces.
   Verify: `go test ./internal/ui/... -run 'Compare(InputIsolation|GridLeak)' -count=1`
4. Dialog, drop, and native Open With input is discarded with the localized refusal.
   Verify: `go test ./internal/ui/... -run 'CompareOpenRefusal' -count=1`
5. Locale parity and both manuals describe the exclusive mode.
   Verify: `go test ./... -run 'Translations|Manual' -count=1`

Non-goals: linked zoom/pan, swipe, transition-state preservation, fidelity work,
secondary-window redesign, or replacing the existing feature package seams.

Honest limit: normal OS window closing remains outside app command dispatch; it
is allowed by leaving Fyne's close lifecycle untouched, not simulated here.

## Tasks

### Task 1 — Pin exclusive command and menu behavior
Owner: T0 inline
Status: completed
Files: modify `internal/ui/compare_test.go`, `internal/ui/menus/menus_test.go`, and focused existing tests only when their seam is the natural home
Depends: none
Contract: the acceptance test names above exercise production entry points
Test: one vertical slice at a time for allowed keys, menu state, blocked callbacks, input leakage, and open refusal
Verify: ticket commands 1–4 above
Budget: 0 spawns · 1 review round · full suite: no

### Task 2 — Implement the exclusive command seam
Owner: T0 inline
Status: completed
Files: modify existing `internal/ui` adapters plus `internal/ui/{menus,favorites,compare}` owners
Depends: Task 1's current failing slice
Contract: `compare.Visible()` remains the sole active-mode fact; no feature imports another feature or gains a broad Host
Test: make each failing vertical slice green before adding the next
Verify: ticket commands 1–4 above
Budget: 0 spawns · 1 review round · full suite: no

### Task 3 — Localize, document, and land
Owner: T0 inline (catalogue repetition scripted only if needed)
Status: completed
Files: `translations/*.json`, both manuals, `todos.md`, ticket, `ARCHITECTURE.md`, this plan
Depends: Task 2
Contract: exact key `Return to Grid View before opening files`; no Unicode arrows
Test: locale/manual guards
Verify: ticket command 5, then `make verify`
Budget: 0 spawns · 1 review round · full suite: yes, once

Task graph: `Task 1 -> Task 2 -> Task 3` (vertical red/green slices interleave
Tasks 1 and 2 as required by TDD).

## Delegation gate

No task is delegated: G4 and G5 fail because the lead already holds the
cross-entry command map and the implementation decisions. Catalogue repetition
is governed by Rule S (script before spawn). Budget: 0 spawns.

## Cost ledger

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|---|---:|---:|---|---|
| T1 | 0 / 0 | 1 | no | Completed as vertical red/green slices; every isolation boundary received a negative guard mutation. |
| T2 | 0 / 0 | 2 | no | Completed; the second review isolated and fixed activation/load ordering under the race detector. |
| T3 | 0 / 0 | 1 | yes / yes | Completed; all ticket commands and `make verify` passed (`internal/ui` race tests: 622.956s). |

## Verification record

- All six ticket acceptance commands passed on 2026-09-01.
- The focused failure-path race test passed 10 consecutive runs.
- `make verify` passed formatting/TUF checks, vet, build, and the complete
  Linux/amd64 Docker race suite.
