# FML-005 — Own search state and Back history

Status: planned.
Type: task.
Owner: T0 lead.
Depends on: FML-003, FML-004, and [MA-026](../../../needs_refactoring.md#ma-026).
Acceptance: AC-05 in the [plan](../plan.md).
Budget: zero spawns; at most two lead review rounds; focused checks only.

## Files

- Add `internal/ui/visualsearch/feature.go`, `history.go`, `uiqueue.go`, and tests.
- Reconcile Explorer setup/browse interface names after MA-026; no root viewer
  worker ownership is introduced here.
- Update `ARCHITECTURE.md` and exact Qodana exclusions.

## Contract and work

`Feature` owns a defensive source snapshot, the active request/session identity,
monotonic query IDs, bounded successful history, progress, and search controls.
Provide Start, Explore, Back, Close, terminal Stop, State, and Settle operations.
Inject `similarity.SearchProvider` and a per-instance UI queue; narrow callbacks
present a ranked visit, restore the origin, and notify state changes.

Starting captures origin separately from the 20-visit history. Commit a new
visit only after successful matching; replace queued references while indexing.
The pending state preserves the last successful view. Newer queries reject older
deliveries after the UI queue hop. A failed query leaves that view/history intact.

Before leaving a visit, capture its current filename filter, selected/highlighted
paths, and viewport anchor. Back while a request is pending first abandons it;
otherwise restore the previous successful visit. With no committed visit yet,
Back restores origin. Back at the earliest retained visit and Exit also restore
origin. A fresh query after Back discards the abandoned
forward branch. No Back operation calls the provider or retains image buffers.

Close cancels the session and suppresses late UI delivery without blocking UI;
Stop also ends future admission. Settle joins all work, drains completions, and
repeats when a delivered action starts work. State has enough information for
menus without exposing mutable history, worker groups, or the search index.

## Acceptance and verification

Use a controlled fake provider and held `uitest.UIQueue` to test out-of-order
delivery, busy Back, query failure, branching, 20-visit eviction with retained
origin, filter/selection/scroll restoration, reopening, and terminal shutdown.
Check provider call counts to prove Back and repeated UI restoration do no
inference. Exercise behavior through Feature, not private fields.

```sh
go test -race ./internal/ui/visualsearch -run '^TestVisualSearch' -count=1
```

Done when feature behavior is testable without constructing unrelated viewer
features and all worker/history ownership stays inside the module.
