# Comparison link button

Status: complete

Route: Standard. This adds one discoverable comparison control across the
existing compare feature, assembled-viewer test seam, localization, manuals,
and architecture/release notes. It adds no package, dependency, preference, or
exported API.

Deliverable: a ready-gated top-left Unlink/Link control shares the exact
`Ctrl+L` action and carries the existing target-aware Unlinked status beside
it, while the existing action toolbar stays at the top right.

## Locked decisions

- The button reads Unlink while linked and Link while unlinked.
- Button and physical `Ctrl+L` are inert until both sources are ready.
- The top-left card contains button then status; status remains hidden while
  linked.
- Open, failure/close, and Swap reset to linked. Existing transform behavior is
  unchanged.
- Tests use `compare.Feature` and the assembled viewer; no golden is added.

## Tasks

### Task 1 - Top-left chrome tracer

Owner: T2 mechanical for the exact red test; T0 for review and green.

Files: `internal/ui/compare/compare_test.go`, then
`internal/ui/compare/compare.go` and translation bundles.

Test: the permanent compact/translucent left card, button-first ordering,
loading disablement, readiness enablement, and unchanged right action card.

Verify: `go test ./internal/ui/compare -run '^TestCompareLinkControl_TopLeftCardAndReadyGate$' -count=1`

Budget: one spawn, one T0 review round, no full suite.

### Task 2 - Shared button and shortcut behavior

Owner: T2 mechanical for the exact red integration test; T0 for review and
green.

Files: `internal/ui/compare_test.go`, then link-state synchronization in
`internal/ui/compare/compare.go` and its package tests.

Depends: Task 1.

Test: loading-time `Ctrl+L` is inert; after readiness `Ctrl+L` reveals Link plus
Unlinked, and tapping Link restores Unlink plus hidden status. Open and Swap
reset the same state.

Verify: `go test ./internal/ui -run 'Compare(LinkControl|LinkToggle)' -count=1`

Budget: one spawn, one T0 review round, no full suite.

### Task 3 - Documentation and landing

Owner: T0 inline.

Files: both manuals and their guard, `ARCHITECTURE.md`, `todos.md`, issue/spec,
and this plan.

Depends: Tasks 1 and 2.

Contract: document and localize the finished pointer/keyboard workflow, run
focused regressions, negatively verify the new guards, and record evidence.

Verify: `go test . -run '^TestTranslations_' -count=1`,
`go test ./internal/ui/help -run '^TestManual' -count=1`, then `make verify`
once.

Budget: zero spawns, at most two T0 review rounds, one full suite.

Task graph: 1 -> 2 -> 3. Agent failures are fixed by T0 without respawning.

## Cost ledger

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
|---|---:|---:|---|---|
| T1 | 1 / 1 | 2 | no | Red tracer delegated; primary implemented and tightened the layout assertion. |
| T2 | 1 / 1 | 2 | no | Red tracer delegated; primary implemented and negatively verified the ready gate. |
| T3 | 0 / 0 | 2 | yes | Strings, manuals, architecture notes, focused regressions, and `make verify` passed. |

## Outcome

- Added a compact top-left card whose disabled-while-loading Unlink/Link button
  and adjacent target-aware status share `compare.Feature.ToggleLink` with
  physical `Ctrl+L`.
- Preserved the separate top-right layout, Swap, and Back to Grid action card.
- Retained linked resets for Open and Swap and documented the behavior in both
  locales.
- Captured two permanent red-to-green tracers. Negative mutations separately
  proved the readiness guard and button-before-status ordering.
- Final verification: focused comparison, translation, and manual suites
  passed; `make verify` passed formatting/TUF checks, vet, build, and the full
  Linux/amd64 race suite on 2026-09-02.
