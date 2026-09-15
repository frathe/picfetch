# Warm search pipeline verification

Status: implemented, measured and locally verified; branch acceptance pending. Route: Standard (one package, test/benchmark and records).
This implements the remaining measurement recommendation from the
[44-finding assessment](../docs/find-more-like-this/pr25-complete-architecture-review.md).
The optional cache-full behavior decision is pending; this work preserves the
current behavior. Local commits remain authorized.

## Contract

Measure the real warm worker path: reopen the representation store, read cached
records, validate source versions, prepare and rank a full scope, and publish
results. Keep this distinct from the existing benchmark that supplies prepared
vectors directly. Exclude initial cache population, native model startup,
subprocess transport and UI paint; report that scope explicitly.

A deterministic regression sequence must use the same real warm path, prepare
every source once without opening the encoder, rank later references from the
retained scope, and reject source changes before publishing invalid results.
No timing threshold belongs in the test suite. Benchmarks report wall time and
allocations and do not establish real-image relevance or cold inference speed.

## Tasks and verification

1. Lead: add `internal/similarity/search_pipeline_test.go` with a temporary
   source/cache fixture, a production-path characterization and a warm pipeline
   benchmark at 1,000 and 10,000 sources. Reuse existing deterministic vectors
   and canonical cache admission. Source files and caches remain in test temp
   directories. Verify with focused race tests and negatively verify the reuse
   and source-validation guards through temporary overlays.
2. Lead: measure the new benchmark and existing progressive-session benchmark
   together using `go test -run '^$' -bench 'BenchmarkSearch(WarmPipeline|SessionProgressive)$' -benchtime=3x -count=3 -benchmem ./internal/similarity`.
   Record the raw command, environment, measurements and interpretation here.
   Do not claim subprocess/UI end-to-end latency from worker-only measurements.
3. Lead: add the exact Qodana test-file exclusion, update todos and this record,
   inspect changed Go files with GoLand including weak warnings, and run one
   `make verify` final gate before committing. Existing AGENTS.md edits remain
   separate. No packages, production interfaces, dependencies or strings change.

Graph: `1 -> 2 -> 3`. New tests are outside root UI and need no shard rows.
All review and any discovered fixes stay with the lead. Baseline is the passing
`make verify` on `905be55`; no unchanged baseline suite is repeated.

## Routing and budget

One read-only scout traced pressure/store replacement and lease ownership while
the lead traced worker preparation/publication. G1: bounded factual task;
G2: named store/test locations verified by the lead; G3: no writes; G4/G5:
separate store lifetime focus. Its pressure facts also inform the pending product
choice. No implementation or review is delegated. Budget/actual: one scout/one,
zero implementation agents, one final lead gate and one full suite.

## Characterization evidence

`TestSearchWarmPipeline` runs 201 real temporary JPEG sources through a reopened
general representation store and the production `searchPreparer`. It chooses
three reference images, checks complete reuse and independently ranked results,
and observes one preparation per source with no encoder opened. The second
sequence removes a source outside the next query's top 30 and requires complete-
scope validation to reject publication. The benchmark uses the same fixture
and production path, without the test's independent ranking oracle.

The initial test oracle included the reference itself among candidates; that
fixture error was corrected to match the documented self-exclusion contract.
It was not a production defect. Focused race characterizations then passed in
4.128 s: `go test -race -tags no_emoji -count=1 ./internal/similarity -run
'TestSearchWarmPipeline|TestSearchSession'`. Log:
`/tmp/picfetch-warm-search-focused.log`.

Negative overlays then proved the guards: forcing reused results to report
misses failed both sequences; checking only the reference/top matches instead
of the complete prepared scope incorrectly published query three and failed
the changed-source sequence. Logs:
`/tmp/picfetch-warm-search-negative-{reuse,validation}.log`. Production files were
never mutated. These are characterizations and negative verification of existing
behavior, not a claimed production red/green bug fix.

An exploratory benchmark overlapped a focused test rerun. The recorded benchmark
is a separate repetition after tests and negative checks finished; no other
test or build is launched by this task during that measurement.

## Recorded measurements

Environment: Go 1.27.1, native Linux/amd64, Intel Core i7-1360P, benchmark CPU
suffix 16. Synthetic deterministic 768-dimensional vectors are stored beside
small temporary JPEG sources; the general cache is fully populated and the
Favorite inventory is empty. No encoder or model assets are opened.

```sh
/snap/go/current/bin/go test -tags no_emoji -run '^$' \
  -bench 'BenchmarkSearch(WarmPipeline|SessionProgressive)$' \
  -benchtime=3x -count=3 -benchmem ./internal/similarity
```

Command passed in 39.312 s. Raw output:
`/tmp/picfetch-warm-search-bench-final.log`. Medians of three runs, each reporting
the average of three iterations:

| Path | Sources | Time/op | Allocated MiB/op | Allocations/op |
| --- | ---: | ---: | ---: | ---: |
| Warm store/preparation/validation/ranking | 1,000 | 159.549 ms | 57.15 | 66,840 |
| Warm store/preparation/validation/ranking | 10,000 | 1,480.090 ms | 570.49 | 667,107 |
| Session supplied with prepared vectors | 1,000 | 4.643 ms | 1.19 | 1,074 |
| Session supplied with prepared vectors | 10,000 | 34.224 ms | 11.75 | 10,521 |

Allocated MiB counts all allocation traffic during an operation, not retained
memory or peak RSS. Cache population, native startup, IPC and UI paint are outside
the timed region. These are warm filesystem-cache observations on this machine,
not a product startup SLA or a relevance measurement.

The complete warm worker path took about 43 times the prepared-vector path at
10,000 sources. Its time grew about 9.3 times for ten times the input size, and
allocation traffic grew about ten times. This supports preserving the current
bounded ranking work. If warm startup needs improvement, profile the combined
cache lookup, payload decoding, source validation and filesystem stages first;
the benchmark does not attribute that cost to one of those stages individually.
There is no evidence here for another broad ownership refactor or a cache-format
change without further profiling and an explicit performance target.

## Final gate

GoLand inspected the changed Go file with `errorsOnly:false`, including weak
warnings, and returned no findings. The exact Qodana duplication exclusion is
present; no root-UI test shard changes are needed. Architecture and todos point
to this coverage and measurement scope. `git diff --check` passes.

`env PATH=/snap/go/current/bin:$PATH make verify` passed: formatting/TUF checks,
vet, build, shard validation and the complete native Linux/amd64 Docker race
suite. All 68 packages outside root UI completed, and all three root-UI shards
passed, with no failed tests or race reports. The new warm-pipeline regression
passed in that run in 6.750 s. Log: `/tmp/picfetch-warm-search-verify.log`.
Race artifacts: `.scratch/race-runs/20260915T093948Z-sOFyYA/`.

The lead checked the final scope against the contract and evidence. Only the
test/benchmark, its Qodana exclusion and architecture/plan/todo records are in
this change. The pending cache-full product question does not change this
completed characterization/benchmark deliverable.
