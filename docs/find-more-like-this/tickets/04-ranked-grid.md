# FML-004 — Add ranked Grid visits

Status: planned.
Type: task.
Owner: T0 lead.
Depends on: FML-002 match contract.
Acceptance: AC-04 in the [plan](../plan.md).
Budget: zero spawns; at most two lead review rounds; focused checks only.

## Files

- Add `internal/ui/grid/ranked.go`, `visit.go`, and focused tests.
- Modify `grid.go`, `search.go`, `nav.go`, and `selection.go` as required.
- Update `ARCHITECTURE.md` and exact Qodana exclusions.

## Contract and work

Add `OpenRanked(RankedVisit)`, `CaptureVisit() Visit`, and `RestoreVisit(Visit)`.
`RankedVisit` carries ordered source paths, reference/title facts, and Back/Exit
callbacks. `Visit` retains mode, ordered paths/subset, filename query,
selected/highlighted source paths, and viewport anchor; it holds no decoded
images. Values are copied and tied to the host collection generation.

Build rank-order-to-root-index mapping without assigning `appState.files` or
changing sort preferences. Collapse repeated paths; reject stale generation
restoration and drop missing identities safely. `ResultIndexes()` remains the
display order; selection/copy/delete/comparison keep root file identities.
Filename search narrows the ranked order without re-sorting it. Suppress
duplicate hiding/browsing changes for the fixed ranked list while remembering
the origin setting. Keep ordinary and Explorer subset behavior unchanged.

A zero-match visit must still render its reference/status/Back controls, even
though ordinary Grid opening may require files. Preserve marquee, selection,
and search Escape stages before invoking Back. Single click/Enter still opens
the selected file; a separate command changes the reference.

## Acceptance and verification

- A deliberately shuffled rank list appears in rank order with correct root
  indexes for opening, selection, filtering, and action targets.
- Restoring an ordinary, subset, and ranked visit restores highlight, selection,
  query, and viewport anchor; recycled/offscreen cells behave the same.
- Empty results and changed/missing source identities are handled safely.
- Existing Grid filtering, duplicate, selection, and keyboard tests pass.

```sh
go test -race ./internal/ui/grid -run '^TestRankedVisit' -count=1
go test -race ./internal/ui/grid -count=1
```

Done when a ranked view is a native Grid visit and no caller needs to mutate
the collection's file order to display search results.
