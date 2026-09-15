# Cache maintenance phases and producer handoff

Status: implemented and verified locally; branch acceptance pending. Route: Deep (analysis-cache, root composition and visual
search). Follow-up to the [complete PR #25 assessment](../docs/find-more-like-this/pr25-complete-architecture-review.md).
Ronin authorized continuing implementation with useful SDD/TDD and local commits.

## Problem and contract

`Host.Quiesce(bool)` hides why a producer is being retired and whether cache
write admission has already been revoked. Policy changes retire local producers
before the provider's initial inspection; record maintenance invalidates shared
write leases before requesting UI retirement and joins those workers before
removing records. Those are separate phases with different guarantees.

The current cancellation handoff also selects between a queued UI reply and
context cancellation. Cancellation may abandon the reply while the UI callback
is already retiring producers. Completion must include every retirement that
the callback actually started, while cancellation before the callback starts
must remain independent of UI delivery.

| Decision | Contract |
| --- | --- |
| Explicit reasons | Replace the preservation boolean with named policy-change, record-removal and automatic-eviction reasons. Root may preserve a fully prepared search producer only for automatic eviction after its writes were revoked. |
| Ordered phases | Keep local policy retirement before inspection, shared lease invalidation under exclusive locks, UI cancellation, worker join, record mutation and final UI delivery distinct. Name the queued handoff and move its ownership out of worker orchestration. |
| Claimed callback owns completion | A canceled, unclaimed callback does nothing and does not require UI drain. Once claimed, its reply and every nonnil producer barrier belong to maintenance completion even if cancellation happens meanwhile. |
| Preserve behavior | Partial preparation still pauses on cache pressure; fully prepared vectors may survive automatic eviction. Explicit Favorite writes join. Opt-out remains committed despite view closure or inspection errors; limits apply only after accepted maintenance. |
| Scope | No disk format, cross-process locking, cache budgets, dependency, platform policy or user-visible strings change. No publishing is included. |

The limit of this refactor: it makes phase and handoff ownership local; it does
not remove the need for both local cancellation and cross-process write leases.
Changing cache-full behavior requires a separate product decision and spec.

## Tasks and acceptance

1. **Characterize callback ownership** (lead). New
   `internal/ui/analysiscache/quiescence_test.go`, existing feature tests.
   Prove cancellation before UI claim does not need a drain; cancellation after
   claim waits for the callback reply and both producer barriers. Late queued
   callbacks cannot retire producers for an abandoned operation. Use controlled
   queues and `synctest`, no sleeps. Capture red behavioral evidence.
   Verify: `go test -race -tags no_emoji ./internal/ui/analysiscache -run 'TestJoinRevokedWriters|TestAnalysisCacheManagement' -count=1`.
2. **Make the phases explicit** (lead, depends on 1). New
   `internal/ui/analysiscache/quiescence.go`; existing analysis-cache
   `feature.go`, `operation.go`, `work.go`, `feature_test.go`; root
   `internal/ui/analysiscache.go`; visual-search `session.go`, `feature_test.go`.
   Use a claimed/canceled UI handoff and named quiescence reasons; give the
   prepared-worker notification a name that states write revocation. Preserve
   captured queues, view lifetime, predecessor joining and accepted outcomes.
   Verify: focused race tests over analysis-cache, visual-search, root UI and
   similarity using `TestJoinRevokedWriters|TestAnalysisCache|TestVisualSearchCache|TestVisualSearchLifecycle|TestFindMoreLikeThis`.
3. **Review and verify** (lead, depends on 2). Negatively verify the new guards;
   inspect every changed Go file with GoLand, including weak warnings. Update
   architecture, exact Qodana test exclusion, todos and this evidence record.
   Run `env PATH=/snap/go/current/bin:$PATH make verify` once at the final gate.
   Commit the verified scope, excluding the existing user edit in AGENTS.md.

Graph: `1 -> 2 -> 3`. No new root-UI top-level tests are planned.
Dependencies are unchanged; standing distribution qualification remains in
`docs/find-more-like-this/dependency-qualification.md`.

## Recon, routing and budget

Shell searches located phase and test boundaries. One read-only scout traced
the similarity manager's lock, lease and cancellation guarantees while the lead
traced UI admission/delivery and search-worker disposition. G1: bounded factual
question; G2: named source/tests verified by the lead; G3: no edits; G4/G5:
the lower-level lock/report path was outside the lead's UI focus. No design,
implementation, review or fixes are delegated.

| Work | Spawns budget/actual | Review | Full suite |
| --- | --- | --- | --- |
| Recon | 1 / 1 | lead verifies facts | no |
| 1-2 | 0 / 0 | lead regression gate | no |
| 3 | 0 / 0 | final lead gate | once |

Existing lower-level guards cover invalidated old writers, initially absent
roots, lock contention, joining a producer, cancellation after quiescence and
one removal, cancellation while waiting for a writer, partial inventory errors,
LRU and temporary-file priority. The new guard addresses the UI callback handoff
above those existing boundaries.

## Implementation and evidence

`QuiesceReason` now names `PolicyChange`, `RecordRemoval` and `AutomaticEviction`.
The policy path joins local producers before provider inspection. The manager's
post-revocation callback uses `joinRevokedWriters`; automatic eviction reaches
search through `CacheWritesRevoked`. The retained-preparation, pending-Favorite,
limit-acceptance and Settings lifetime rules are preserved.

The handoff has one atomic ownership decision: UI claims a queued request or
cancellation abandons it. A claimed callback delivers its producer barriers,
and maintenance joins every one before completing. An abandoned callback has
no later effects and does not require a UI drain to finish cancellation.

### TDD and negative guards

- Baseline focused race suite passed: analysis-cache 1.321 s, visual-search
  1.046 s, similarity 1.590 s, root UI 53.252 s. Log:
  `/tmp/picfetch-cache-phases-baseline.log`.
- Wrote `TestJoinRevokedWriters` and mechanically extracted the existing
  handoff without changing its behavior. The claimed-callback case failed:
  "maintenance completed while a claimed callback still owned producer retirement".
  Other cancellation/join scenarios passed. Log:
  `/tmp/picfetch-cache-phases-red.log`.
- Added ownership arbitration; all handoff and maintenance tests passed
  (analysis-cache 1.308 s). Log: `/tmp/picfetch-cache-phases-green.log`.
- The real-manager phase sequence covers usage inspection, unchanged/increased/
  decreased limits, clear, stale cleanup, automatic eviction and persistence
  opt-out. Existing integration tests retain prepared searches only for
  automatic eviction and join preparation or explicitly pending Favorite writes.
- Four temporary Go overlays failed for their intended behavioral reason:
  abandoning an already claimed reply, requiring UI delivery after unclaimed
  cancellation, skipping canceled producer joins, and classifying automatic
  eviction as explicit record removal. Logs:
  `/tmp/picfetch-cache-phases-negative-{reply,queued,join,reason}.log`.
  Production files were unaffected by these negative runs.

Focused integration command passed analysis-cache (1.310 s), visual-search
(1.037 s), similarity (1.607 s) and root UI (53.465 s):

```sh
/snap/go/current/bin/go test -race -tags no_emoji -count=1 \
  ./internal/ui/analysiscache ./internal/ui/visualsearch ./internal/similarity ./internal/ui \
  -run 'TestJoinRevokedWriters|TestAnalysisCache|TestVisualSearchCache|TestVisualSearchLifecycle|TestFindMoreLikeThis'
```

Log: `/tmp/picfetch-cache-phases-focused.log`. The phase-sequence test was then
flattened into a single ordered scenario, avoiding parent-test failure calls
from dependent subtests; the negative reason guard ran on that final form.

GoLand inspected all nine changed Go files with `errorsOnly:false`, including
weak warnings; every result was empty and complete. The new test has its exact
Qodana duplication exclusion. No root-UI runnable was added. `git diff --check`
is clean. The lead's final diff review preserves captured queues, stale-operation
checks and the independent local-retirement/shared-lease phases.

Final gate: `env PATH=/snap/go/current/bin:$PATH make verify` passed (exit 0).
Formatting, generated assets/TUF checks, Qodana exclusion validation, vet and
build passed. The native Linux/amd64 Docker race run passed all 68 non-root-UI
packages and all three root UI shards, including the final ordered phase test.
Log: `/tmp/picfetch-cache-phases-verify.log`; structured artifacts:
`.scratch/race-runs/20260915T091324Z-5hIXmZ/`.

One read-only scout and one full verification gate were used, within budget.
The lead owns all implementation, review and fixes. The existing AGENTS.md user
edit is excluded from the authorized local commit. Cache-pressure behavior is
unchanged; the optional product simplification remains in `todos.md`.
