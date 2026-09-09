# Grid scroll follows duplicate merges

Route: Standard SDD. Implement the approved
[spec](../.scratch/grid-scroll-follow-duplicate-merges/spec.md) inside
`internal/ui/grid`, preserving the viewport and keeping the ring visible.

The spec's decisions and non-goals are accepted, including native scrollbar
tracking only on the next reflow and minimum reveal when no full row fits.
The approved test seams are real Fyne canvas input on the mounted Overview,
its highlight/selection/host notifications, and normal duplicate completion
through the existing UIQueue and Settle. No additional API or worker is needed.

## Tasks

1. **Scroll follower** — T0 inline. Modify `grid.go`, create `scrollfollow.go`,
   extend `marquee_test.go`. Private `scrollFollower` implements only
   `fyne.Scrollable`; shared visible-edge reconciliation uses GridWrap's
   geometry and public offset API. Red/green: real canvas scroll, same-column
   edge rows, partial/short rows, unchanged selection/current image, and input
   routing guards. Verify AC1/2 with
   `go test ./internal/ui/grid -run '^(TestScrollFollower_|TestMarquee)' -count=1`.
   Budget: one read-only Scout, two review rounds, no full suite.
2. **Live reflow** — T0 inline, depends on 1. Modify `search.go` and
   `dupes_test.go`. Retain the remapped host only within the viewport; otherwise
   use visible-edge reconciliation. Red/green: duplicate removal above a
   scrolled viewport, native scrollbar offset, retained visible host, explicit
   reset behavior, unchanged selection/image. Verify AC3/4 with
   `go test ./internal/ui/grid -run 'Test(HandleKey|HighlightChanged|SetHideDuplicates|RebuildFilter)' -count=1`
   and `go test ./internal/ui/grid -count=1`.
   Budget: zero spawns, two review rounds, no full suite.
3. **Final gate and records** — T0, depends on 2. Verify AC5 with `make verify`;
   inspect the diff and record outcomes in this plan, tickets and `todos.md`.
   Update the grid locator in `ARCHITECTURE.md`. Budget: zero spawns, one full
   suite. Do not commit.

Graph: routing Scout runs alongside T0 geometry recon; 1 -> 2 -> 3.

## Delegation gate

Routing Scout: G1 yes (bounded routing question); G2 yes (local source locations
can be verified with targeted reads); G3 yes (read-only); G4 yes (driver/test
routing sweep independent of geometry); G5 yes (routing context not yet held).
S: routing semantics need comprehension, not a transform. W: no implementation
delegated. All design, tests, implementation, review and fixes stay with T0.

## Checklist

- [x] Frame, approved spec and acceptance commands
- [x] Recon and plan; delegation gate recorded
- [x] Red/green scroll follower
- [x] Red/green live reflow
- [x] T0 review and negative guard checks
- [x] Final verification
- [x] Records and cost ledger

## Evidence and ledger

- Scroll tracer red: canvas offset advanced to 124 while the ring stayed at
  cell 1 instead of cell 4. Green after installing the scroll-only sibling.
- Edge-row reds: short viewports chose cells 4/10 instead of 1/7; a stale ring
  after native scrollbar movement chose the wrong directional edge. Green
  after intersecting-row fallback and direction-aware edge selection.
- Reflow red: thirty extras disappearing above row ten reset offset 1240 to
  zero and kept host 31. Green preserves offset 1240 and rings display cell 31
  (host 61), leaving the explicit selection and displayed image unchanged.
- Negative input guards: temporarily adding tap/drag/hover/mouse/focus methods
  failed every interface guard and every canvas pointer-routing case. Removed
  the mutation and reran green.
- Negative restoration guards: temporarily restoring offscreen hosts and
  bypassing explicit resets failed the reflow/native-scrollbar cases and all
  five explicit-reset cases. Restored the implementation and reran green.
- AC1/2 targeted suite: `ok github.com/frathe/picfetch/internal/ui/grid 0.872s`.
- AC3 focused regression: `ok github.com/frathe/picfetch/internal/ui/grid 0.529s`.
- AC4 targeted suite: `ok github.com/frathe/picfetch/internal/ui/grid 0.922s`.
- Final native grid package: `ok github.com/frathe/picfetch/internal/ui/grid 1.142s`.
- `git diff --check`: clean. `make verify` completed on 2026-09-09 with exit 0:
  formatting, TUF, Qodana exclusions, vet, build, exact shard manifest and all
  four Linux/amd64 race partitions passed. Grid package: 4.426s; main UI
  shards: ui-1 302.692s, ui-2 287.939s, ui-3 277.613s.
- Full log and all four raw streams are retained under
  `.scratch/grid-scroll-follow-duplicate-merges/verification/`. No skipped gate,
  no full-suite retry, no test/shard exclusions added, and no commit created.
- The approved limitation remains: native scrollbar movement reconciles the
  ring on the next filter reflow, and the proxy may not trigger the macOS
  transient scrollbar indicator. No new convention or separate memory needed.

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
|---|---|---|---|---|
| Scroll follower | 1/1 | 2 | no | Read-only routing Scout; code and review T0 |
| Live reflow | 0/0 | 2 | no | T0 red/green and negative verification |
| Final gate | 0/0 | 1 | once, passed | `make verify` |

