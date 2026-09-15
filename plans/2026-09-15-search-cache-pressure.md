# Continue search when the general cache fills

Status: implemented and locally verified; branch acceptance pending. Route: Deep, because the behavior crosses
the similarity worker, visual-search feature and root maintenance adapter.
The user selected the recommended continue-in-memory policy. Local commits are
authorized; this task does not invoke the GitHub review workflow.

## Contract and decisions

General-cache capacity must not interrupt preparation or discard its in-memory
vectors. On the first capacity refusal, the store stops general writes for that
producer. Reads and enabled Favorite writes remain available. A committed
Favorite save refreshes ownership without reopening general writes. A fresh
producer can write again under its captured policy. Oversized records continue
to be skipped individually; genuine storage errors retain warning behavior.

The worker reports capacity pressure on session readiness, after finite
preparation. The feature requests automatic eviction once, only after readiness
acknowledges any currently pending Favorite save. Readiness belongs to the
session even if its reference failed or Back abandoned the query. Cleanup retains
the prepared worker under the existing revoked-lease protocol. Explicit cleanup,
policy changes and cancellation keep their existing retirement/barrier behavior;
a save admitted after maintenance begins still follows that protocol.

No new UI preference or paused notice. Existing vector/IPC limits, cache record
format, dependencies, model assets and platform implementation stay unchanged.
This improves continuity, but analysis omitted from disk may need recomputing in
a later session. Canceling before readiness does not request automatic eviction;
a later completed search or explicit cache maintenance can reclaim space.

## Acceptance and task graph

1. **Store policy (lead):** adjust `cache_store.go` and existing
   `cache_policy_test.go`. Real temporary stores must prove the first refusal,
   subsequent write skipping even after space becomes available, continued
   general reads/Favorite writes, unchanged opt-outs, ownership refresh retaining
   the restriction, and fresh-producer admission. Existing budget/accounting and
   lease tests remain applicable.
   Verify: `go test -race -tags no_emoji -count=1 ./internal/similarity -run '^TestAnalysisCache'`.
2. **Worker and UI handoff (lead):** update `search_worker.go`,
   `search_pipeline_test.go`, `visualsearch/session.go`, its `feature_test.go`,
   root `visualsearch.go` and `visualsearch_test.go`. Characterize readiness with
   valid and failed references through real warm preparation; pressure must not
   appear on partial/final publications. Feature tests cover stale-query readiness,
   pending Favorite acknowledgement, one maintenance request and retained queries.
   Root regression uses real eviction and verifies unchanged explicit cleanup.
   Verify: focused `TestSearch.*Pipeline|TestSearchSession`,
   `TestVisualSearchCachePressure`, and `TestFindMoreLikeThisInitialAdmission`
   race tests in their owning packages.
3. **Records and final gate (lead):** update `spec.md`, architecture and todos;
   remove the obsolete paused string from both translation bundles. Existing
   test files and root test names need no new exclusions or shard rows.
   Inspect every changed Go file in GoLand including weak warnings. Negatively
   verify write suppression, readiness delivery and pending-save guards with
   temporary overlays. Run one `make verify`, then commit only this scope.

Graph: `1 -> 2 -> 3`; lead owns all design, tests, implementation, review and fixes.
Baseline: clean working tree at `6225603`, whose complete `make verify` passed.
No unchanged baseline suite is repeated.

## Delegation and budget

One read-only scout traces readiness ordering/stale delivery and existing tests
while the lead examines store admission. G1: bounded factual question; G2: named
functions/tests with line references verified by the lead; G3: no writes;
G4/G5: separate event-ordering sweep. No implementation or review delegation.
Budget: one scout, two focused implementation gates, one lead final gate and one
full suite. Actuals and red/green evidence are recorded below as work completes.

## Evidence

The store's existing write scope now narrows to Favorite-only on its first
capacity refusal. Favorite refresh already carries this scope into the replacement
store, so no additional session flag or reset logic was introduced.
`searchPreparer.run` extracts the existing session wrapper without changing its
initial behavior, allowing the real warm path to exercise pressure delivery
without native model assets. Pressure is then attached only to `SearchReady`.
Feature delivery requires readiness, the current Favorite acknowledgement and
the existing once-per-producer guard. The obsolete paused toast and both locale
entries were removed. All test files and root test names already have exclusions
and shard assignments.

### Red and green

The tests failed for their intended reasons before the relevant behavior changes:

- Store: subsequent general writes repeated `CachePressureError` instead of
  skipping persistence. Log: `/tmp/picfetch-cache-pressure-store-red.log`.
- Worker: a valid reference reported pressure on the 100-source partial result;
  an invalid reference completed with readiness but lost its pressure request.
  Log: `/tmp/picfetch-cache-pressure-worker-red.log`.
- Feature: partial pressure suspended preparation in all four scenarios.
  Log: `/tmp/picfetch-cache-pressure-feature-red.log`.
- Root: the real cache manager was admitted before preparation finished.
  Log: `/tmp/picfetch-cache-pressure-root-red.log`.

Focused race checks passed after implementation:

| Package and filter | Result | Log under `/tmp/` |
| --- | --- | --- |
| `./internal/similarity -run '^TestAnalysisCache'` | pass, 1.669 s | `picfetch-cache-pressure-store-green.log` |
| `./internal/similarity -run 'TestSearch.*Pipeline\|TestSearchSession'` | pass, 5.468 s | `picfetch-cache-pressure-worker-green.log` |
| `./internal/ui/visualsearch` (complete feature package) | pass, 1.029 s | `picfetch-cache-pressure-feature-green.log` |
| `./internal/ui -run '^TestFindMoreLikeThisInitialAdmission$'` | pass, 6.694 s | `picfetch-cache-pressure-root-green.log` |

Each command used `/snap/go/current/bin/go test -race -tags no_emoji -count=1`.
The worker fixture begins with a real store capacity refusal and uses 201 cached
sources, covering the production preparation/publication path with valid and
invalid references. It does not claim a native inference or subprocess trial.
The root regression observes queued partial delivery and readiness explicitly,
then waits actual automatic eviction and checks a retained query and explicit
cleanup retirement. No timing sleeps or new background work were added.

### Negative guards and review

Three independent temporary Go overlays proved further guards:

- Reopening general writes during Favorite refresh failed both preference cases
  with `Favorite refresh reopened general writes`.
- Removing the pending-save condition failed with `eviction interrupted pending
  Favorite persistence`.
- Removing the once-per-producer condition failed when the next reference
  repeated its cleanup request.

Logs: `/tmp/picfetch-cache-pressure-negative-{refresh,favorite-pending,once}.log`.
Overlays live in `/tmp/picfetch-cache-pressure-overlays/`; production sources
were never mutated by these checks. Initial red runs also negatively verified
general-write suppression and worker/UI readiness guards.

GoLand inspected all eight changed Go files with `errorsOnly:false`, including
weak warnings, and returned no findings. The lead reviewed the production diff
and test evidence against the contract. No dependencies, assets, platform code
or package boundaries changed. `git diff --check` passed.

### Final gate

`env PATH=/snap/go/current/bin:$PATH make verify` passed: formatting, TUF and
asset checks, vet, build, shard validation and the complete native Linux/amd64
Docker race suite. All 68 packages outside root UI completed successfully; the
three root-UI shards passed in 302.748 s, 315.783 s and 317.832 s. There were no
failed tests or race reports. The root pressure/admission regression passed in
6.570 s within that run.

Log: `/tmp/picfetch-cache-pressure-verify.log`.
Race artifacts: `.scratch/race-runs/20260915T101317Z-ZLzzmY/`.

Final scope is eight Go files, two translation bundles, architecture, spec,
todos and this implementation record. The working tree was clean at admission;
no unrelated user edits were included. Actuals: one read-only scout, zero
implementation agents, two focused implementation gates, one lead review and
one complete suite. No additional review/fix round was needed.
