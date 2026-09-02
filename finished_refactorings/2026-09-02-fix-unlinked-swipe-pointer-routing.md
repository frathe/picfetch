# Fix unlinked Swipe pointer routing

Status: complete

Route: Standard. The diagnosed bug is confined to comparison input geometry,
but the fix spans the comparison package's behavior tests, implementation, and
standing documentation. It does not change render geometry, reveal clipping,
divider semantics, exported APIs, preferences, translations, or manuals.

Deliverable: in Swipe layout, an unlinked pointer gesture targets the photo
revealed beneath the pointer, including after divider movement, while right-pane
wheel zoom remains anchored beneath the cursor in the full render viewport.

## Task graph

`01 -> 02 -> 03`

### Task 01 - Route Swipe input by reveal

Owner: T0 inline
Files: modify `internal/ui/compare/compare_test.go`,
`internal/ui/compare/transform.go`, and `internal/ui/compare/swipe.go`
Depends: none
Contract: add private `layoutPaneInput(index, input)` geometry driven by
`paneVisibleArea`; exercise it during pane and reveal layout without resizing
the renderer viewport.
Test: actual canvas hover, drag, wheel, and transform-key routing follows the
revealed pane at the default divider, after movement, and at both extremes.
Verify: `go test ./internal/ui/compare -run '^TestCompareSwipeUnlinkedCanvasRoutesPointerByReveal$' -count=1`
Budget: 0 spawns; 1 review round; full suite: no

### Task 02 - Preserve the right wheel anchor

Owner: T0 inline
Files: modify `internal/ui/compare/compare_test.go` and
`internal/ui/compare/input.go`
Depends: 01
Contract: copy each non-nil scroll event and translate its reveal-local
position by the pane input origin before forwarding it; never mutate the
caller-owned event.
Test: the normalized right-photo point beneath a reveal-local cursor is stable
through wheel zoom; the event is unchanged and nil is inert.
Verify: `go test ./internal/ui/compare -run '^TestCompareSwipeUnlinkedRightWheelPreservesViewportAnchor$' -count=1`
Budget: 0 spawns; 1 review round; full suite: no

### Task 03 - Document, review, and verify

Owner: T0 inline
Files: modify `CONTEXT.md`, `ARCHITECTURE.md`, `todos.md`, the local spec and
tickets, and this plan
Depends: 01, 02
Contract: record canonical terminology and the reveal/input invariant; run all
acceptance commands, negatively verify both guards, and finish with one
`make verify` invocation.
Test: package and assembled comparison regression suites plus both intentional
guard violations.
Verify: `make verify`
Budget: 0 spawns; 1 review round; full suite: yes, once

## Delegation gate and cost ledger

All tasks stay inline: the diagnosis, test seam, and implementation contract
are already hot context (G5), and every task touches files shared with the next.
Rule S has no useful mechanical transform here; Rule W favors applying the
already-specified small change directly.

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|---|---:|---:|---:|---|
| 01 | 0 / 0 | 1 | no | guard failed on original Right-over-left symptom, then passed |
| 02 | 0 / 0 | 1 | no | guard failed on anchor drift, then passed |
| 03 | 0 / 0 | 1 | yes | `make verify` passed on its only invocation |

## Outcome

The two TDD slices passed their focused and package-level commands, both guards
were negatively verified and restored, and the assembled comparison tests
remained green. `make verify` passed formatting, the offline TUF-root check,
vet, build, and the complete Linux/amd64 race suite. No subagents were used.
