# FML-003 — Add a reusable search worker session

Status: planned.
Type: task.
Owner: T0 lead.
Depends on: FML-002.
Acceptance: AC-03 in the [plan](../plan.md).
Budget: zero spawns; at most two lead review rounds; focused checks only.

## Files

- Add `internal/similarity/search.go`, `search_worker.go`, and their tests.
- Modify `client.go`, `analyze.go`, and internal representation preparation;
  reuse `cache.go`, existing worker launchers, and build-tagged encoder adapters.
- Update `ARCHITECTURE.md` and exact Qodana exclusions.

## Contract and work

Implement `Client.Search(ctx, SearchRequest, <-chan SearchQuery,
func(SearchEvent)) error` and its injectable `SearchProvider` function type.
Request owns a copied source-path snapshot and limit. A query has monotonic ID
and reference path. Events distinguish indexing/ready/matches/query error;
carry reference/query identity, counters, and immutable match values. A
completed query does not terminate the session or reuse map `Event.Complete`.

Add an explicit private request operation while preserving the existing Analyze
operation and parser limits. Extract shared versioned representation preparation
inside `similarity`: both routes use the same orientation, input-size limits,
encoder, telemetry policy, platform isolation, and optional Favorite cache.
Search performs no grouping/projection. Keep vectors and source facts in worker
memory, disposing of frames/previews after each source and compatible cache write.

Enforce the plan's checked embedding-cap admission. Index each unique source
once; a warm reference change performs no source decode or inference. Check
source versions on workers before admitting a query result; a changed snapshot
requires explicit restart. During preparation retain only the latest pending
query. After readiness serialize queries, checking staleness/cancellation, and
emit matches for the matching ID. A bad reference is recoverable per query.

Context cancellation, closed input, malformed protocol, emitter/pipe failure,
and unexpected worker exit must terminate and join worker/control-writer work.
Provider return includes observed worker exit. Do not move native inference
into the Fyne process or leak vectors through event payloads.

## Acceptance and verification

Use a fake representation producer/process adapter to cover latest-query wins,
zero warm inference, cache hit/miss/version invalidation, removed Favorites,
input limits, capacity rejection before allocation, corrupt inputs, partial
candidate failure, duplicate paths, worker EOF, cancel, and protocol rejection.
Existing Analyze completion and isolation tests must still pass.

```sh
go test -race ./internal/similarity -run '^TestSearch(Session|Protocol|Cache)' -count=1
go test -race ./internal/similarity -count=1
```

Done when session tests observe actual stop/exit, no callback retains mutable
worker buffers, and repeated searches reuse one prepared index.
