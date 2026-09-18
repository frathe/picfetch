# Bound Grid decode queue growth

Route: Standard. The lead owns the queue design, regression tests, implementation
and review. Two read-only scouts collected profile and lifecycle evidence; no
implementation is delegated.

The user's 22k-item Grid capture completed normally after the toast fix, but it
peaked at 21,781 goroutines. The goroutine profile shows thousands of
`decodepool.Pool.Go` waiters blocked before the four-slot decode semaphore.
That queue shape can consume substantial memory and scheduling time before a
thumbnail decode starts.

## Decisions

| Decision | Result |
|---|---|
| Admission shape | A Pool owns at most its configured number of decode workers; pending jobs are data, not goroutines. |
| Cancellation | One work-session cancellation path removes its pending jobs and calls their callbacks with `acquired == false`, so Grid claims and hash accounting are released while reads still occupy slots. |
| Priority | Visible Grid thumbnails run ahead of background duplicate hashing. |
| Scope | Retain the Cache-tab Favorite preview limit and its default of 1000. Do not retune image or thumbnail cache budgets in this fix. |

## Acceptance criteria

1. A blocked four-slot pool can accept a 22,000-job Grid-shaped backlog without
   creating a goroutine per queued job.
   Verify: `go test -tags no_emoji,nodynamic ./internal/decodepool -run '^TestGo_QueueDoesNotCreateWaiterPerJob$'`.
2. A cancelled Grid work session releases a queued thumbnail claim while all
   decode slots remain occupied, and cancelled hash jobs clear their accounting.
   Verify: `go test -tags no_emoji,nodynamic ./internal/ui/grid -run '^(TestRequestThumbnail_CancelledSlotWaiterReleasesClaim|TestHashRemaining_)$'`.
3. A visible thumbnail request is admitted before a queued duplicate-hash job.
   Verify: focused `internal/ui/grid` priority regression plus the package test
   command from criterion 2.
4. Existing decode-pool concurrency and preload cancellation contracts remain
   intact.
   Verify: `go test -race -tags no_emoji,nodynamic ./internal/decodepool ./internal/ui/grid ./internal/ui/display`.
5. The package map and active work record describe the bounded priority queue,
   and GoLand reports no findings on changed Go files.
   Verify: `make fmt-check`, `make vet`, `make build`, and GoLand inspection.

The byte-bounded thumbnail cache still permits transient allocations while a
source is decoded; this change removes queued-worker growth, not all image
memory use. The user-requested 1000-preview default remains configurable in
Cache settings.

## Tasks

### Task 1 — Red queue regression

- Owner: T0 inline
- Files: modify `internal/decodepool/decodepool_test.go`
- Depends: none
- Contract: a four-slot pool exposes no more than a small fixed goroutine delta
  while 22,000 jobs wait behind occupied slots.
- Test: use `testing/synctest` to hold workers and prove the old semaphore-waiter
  implementation red.
- Verify: criterion 1 command.
- Budget: 0 spawns · 1 review round · full suite: no.

### Task 2 — Bounded priority dispatcher

- Owner: T0 inline
- Files: modify `internal/decodepool/decodepool.go`,
  `internal/decodepool/decodepool_test.go`
- Depends: Task 1
- Contract: `Pool.Go` remains nonblocking and invokes its callback on cancelled
  queued work; a lifecycle-bound queue accepts high and low priority jobs without
  per-job waiting goroutines. `Wait` covers callbacks.
- Test: preserve the existing cancellation and concurrency cases, with a grouped
  cancellation regression where needed.
- Verify: criterion 1 and `go test -tags no_emoji,nodynamic ./internal/decodepool`.
- Budget: 0 spawns · 1 review round · full suite: no.

### Task 3 — Route Grid work through the lifecycle queue

- Owner: T0 inline
- Files: modify `internal/ui/grid/{work,hashengine,thumbs}.go` and existing Grid
  tests/harness as needed
- Depends: Task 2
- Contract: each Grid work session owns one queue; thumbnails submit high priority
  and duplicate hashes submit low priority. Session cancellation resolves queued
  callbacks before any replacement session uses the shared pool.
- Test: cancellation and priority behavior at Grid's real request paths.
- Verify: criteria 2–4 commands.
- Budget: 0 spawns · 1 review round · full suite: no.

### Task 4 — Record and land

- Owner: T0 inline
- Files: modify `ARCHITECTURE.md`, `todos.md`, this plan
- Depends: Tasks 1–3
- Contract: documentation names bounded queued workers, session cancellation and
  Grid priority.
- Test: inspect diff and documentation locators.
- Verify: criterion 5 commands, then push and run the authorized GitHub Codex
  review loop on the new commit.
- Budget: 0 spawns · fresh hosted review · full suite: GitHub CI.

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|---|---:|---:|---:|---|
| Recon | 2 / 2 | 0 | no | Profile and caller scouts; both read-only. |
| T1–T4 | 0 / 0 | 1 each | CI | Lead implementation and review. |

## Verification

- The initial tight regression ran against the semaphore implementation and
  failed as intended: queuing 22,000 jobs behind four occupied slots added
  22,000 goroutines. It directly matches the captured Grid queue shape.
- The bounded-worker and grouped-cancellation regressions now pass. They cover
  both a 22,000-job pending queue and a 22,000-job cancellation while a decode
  slot remains occupied, so neither normal queueing nor cancellation creates a
  goroutine per job.
- Grid regressions prove a current thumbnail begins before an already queued
  hash job, and that cancellation clears queued hash accounting without
  releasing an occupied source read. Existing thumbnail claim/reopen,
  duplicate-hash, and display preload contracts remain green.
- `go test -race -tags no_emoji,nodynamic ./internal/decodepool ./internal/ui/grid ./internal/ui/display -count=1`
  passed. `go test -race -tags no_emoji,nodynamic ./internal/ui -count=1`
  passed through the production viewer wiring.
- `make fmt-check`, `make vet`, and `make build` passed. GoLand inspected every
  changed Go file with no remaining errors or warnings; two pre-existing
  duplicate-fixture warnings in `grid/dupes_test.go` now have narrow,
  rationale-bearing suppressions.
- The user's five-minute bounded retest completed normally. Its font-cache
  allocation stayed near 12 MB, down from the pre-fix capture's 940 MB, while
  the goroutine snapshot identified the independent queue defect fixed here.
  Raw captures remain local because they contain user environment data.
- Fresh hosted CI, security analysis, and Codex review are still required after
  this commit before PR #45 can be marked ready.
