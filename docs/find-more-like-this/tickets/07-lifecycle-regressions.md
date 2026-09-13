# FML-007 — Verify source changes and complete shutdown

Status: planned.
Type: task.
Owner: T0 lead.
Depends on: FML-006.
Acceptance: AC-03 and AC-07 in the [plan](../plan.md).
Budget: zero spawns; at most two lead review rounds; focused checks only.

## Files

- Extend visualsearch/similarity tests and root `visualsearch_test.go`.
- Wire/fix lifecycle hooks in `drop.go`, `sort.go`, `filework.go`, `viewer.go`,
  `run.go`, `harness_test.go`, and the extracted Explorer setup as required.
- Update exact Qodana/shard entries and architecture/concurrency documentation.

## Contract and work

Cancellation and generation checks must already exist in earlier tickets. This
ticket proves their cross-feature behavior and closes integration gaps.

Invalidate on collection replacement/merge, sort generation change, removal,
and any committed Save/Export/Strip/mosaic write handled by filework. Include
commits whose UI request became stale and alias paths already reconciled by
the existing mutation path. Retire the old index/history and restore a valid
ordinary view; do not automatically launch inference or restore deleted sources.

An ordinary image navigation inside a valid ranked visit does not retire its
search session. Late setup, query, or worker-error callbacks cannot reopen a
closed feature or replace a newer reference. External version changes detected
on workers reject the snapshot; filesystem stats never enter paint/input paths.

At shutdown end admission on UI, cancel session/setup work, and join native
search/control writers off UI. Harness drain calls feature Stop/Settle in causal
order with Grid/Explorer/filework. Test completion includes actual callback
delivery or discard, not only a pending counter reaching zero.

## Acceptance and verification

Held readers/providers/queues cover source replacement during indexing, queued
results after close/reopen, mutation before/after commit, changed/deleted
reference, cancellation during setup, back navigation during a query, and
shutdown with blocked IPC. No sleeps or real desktop mutation.

Deliberately remove session/query checks and shutdown cancellation in temporary
overlays; require the matching tests to fail, then restore the code. Preserve
existing Explorer/clipboard/filework/comparison cancellation regressions.

```sh
go test -race ./internal/ui ./internal/ui/visualsearch -run '^TestVisualSearchLifecycle' -count=1
go test -race ./internal/similarity -run '^TestSearch(Session|Protocol|Cache)' -count=1
```

Done when every invalidation/stop route has observed completion and negative
guard evidence, with no late UI effects or surviving search worker.
