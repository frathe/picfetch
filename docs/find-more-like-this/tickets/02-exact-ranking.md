# FML-002 — Implement exact image ranking

Status: planned.
Type: task.
Owner: T0 lead.
Depends on: FML-001 recorded quality decision.
Acceptance: AC-02 in the [plan](../plan.md).
Budget: zero spawns; at most two lead review rounds; focused checks only.

## Files

- Add `internal/similarity/search_rank.go` and `search_rank_test.go`.
- Move/reuse the evaluator's trial ranking through this interface.
- Update exact Qodana exclusions and `ARCHITECTURE.md`.

## Contract and work

`RankSimilar(ctx context.Context, reference Item, candidates []Item, limit int)
([]Match, error)` returns immutable path/score values. `Match` contains `Path`
and `Score`; no UI, source I/O, projections, or native runtime is involved.

Use cosine similarity over finite, nonzero 768-element vectors, with float64
accumulation. Normalize without changing caller buffers. A bad reference or a
limit outside 1–30 returns an error. Exclude its path, collapse repeated candidate
paths, and skip failed/malformed candidates. Different files with identical
embeddings remain distinct. Sort descending by score and ascending by path on
ties. Return an empty successful list when there are no usable matches.

Use a bounded top-k selection rather than an all-pairs matrix. Check cancellation
during candidate scanning; cancellation returns an error, not partial success.
Resource accounting and preparation belong to the worker ticket. Keep the
algorithm independent of map clustering and tag labels.

## Acceptance and verification

- Hand-constructed vectors prove cosine order, normalization, stable ties,
  self-exclusion, identical embeddings/different paths, deduplication, zero/fewer
  than 30 results, NaN/Inf/dimension rejection, and input immutability.
- Compare top-k with a simple full-sort test oracle for generated fixtures.
- Controlled cancellation ends ranking without partial publication.
- Benchmark 1,000/10,000 candidates with allocation reporting; compare with
  FML-001's recorded target without a flaky wall-clock unit assertion.

```sh
go test ./internal/similarity -run '^TestSearchRank' -count=1
go test ./internal/similarity -run '^$' -bench '^BenchmarkSearchRank' -benchmem
```

Done when the exact ranking contract passes through this interface and the
evaluator uses it instead of retaining a second ranking implementation.
