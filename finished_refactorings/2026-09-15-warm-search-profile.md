# Profile warm search preparation

Status: profiled, optimized and locally verified; branch acceptance pending. Route: Standard, one subsystem and bounded evidence.
The original PR #25 recommendations are implemented through `312d17b`.
The warm-worker benchmark measured about 1.48 s and 570 MiB of cumulative
allocation traffic at 10,000 cached sources, compared with 34 ms for supplied
vectors. This follow-up identifies the work before proposing further changes.

## Contract

Collect CPU and allocation profiles of `BenchmarkSearchWarmPipeline/10000` with
the existing synthetic cached corpus. Attribute warm work through the
`runSearchSession` call tree, keeping fixture population outside that analysis.
Do not infer retained memory from allocation traffic or application startup from
this worker benchmark. Native inference, subprocess transport and UI paint
remain outside its scope.

Any proposed simplification must preserve cached-record validity checks, source
version validation, lease/opt-out behavior and ranking results. Use TDD for a
confirmed defect or new behavioral guard. A production change needs a measured
benefit and focused regression evidence, changed-file GoLand inspections and
one final `make verify`. If no small change is justified, finish with measured
evidence and advice; documentation-only work does not rerun an unchanged suite.

## Tasks

1. Lead: capture profiles with `/snap/go/current/bin/go test -tags no_emoji
   -run '^$' -bench '^BenchmarkSearchWarmPipeline$/^10000$' -benchtime=10x
   -count=1 -cpuprofile /tmp/picfetch-warm-profile-cpu.pprof -memprofile
   /tmp/picfetch-warm-profile-alloc.pprof -o /tmp/picfetch-warm-profile.test
   ./internal/similarity`. Inspect `go tool pprof -top` and `-list`, using
   `-focus runSearchSession` and allocation-space/count sample indexes.
2. Lead: verify the dominant stacks against source and rejection tests, select
   a bounded next action based on evidence, and record its contract before any
   production edit. Benchmark comparisons run without concurrent task tests.
3. Lead: update this record and todos; review scope, run applicable checks and
   commit the result locally. Baseline is the passing complete `make verify` on
   `312d17b`; no unchanged baseline suite is repeated.

Graph: `1 -> 2 -> 3`. No new dependencies or model assets. Lead owns all design,
implementation, review and fixes. One read-only scout maps cache record validation
and rejection tests while the lead profiles: G1 bounded question; G2 verified
file/test references; G3 no writes; G4/G5 independent validation sweep. Budget:
one scout, zero implementation agents, one lead review and at most one full
suite if production or test code changes. Evidence and actuals follow below.

## Evidence

The baseline profile passed in 21.858 s (10 measured iterations: 1.483 s/op,
598,194,428 B/op, 667,083 allocations/op). Raw log:
`/tmp/picfetch-warm-profile.log`. CPU/heap profiles and the exact binary use the
paths from task 1. The focused profile excludes fixture construction and temp
cleanup; allocation samples include both benchmark calibration and measured
iterations and are used as proportions, not divided into a misleading per-op
retained-memory figure.

Within `runSearchSession`, `decodeRepresentation` accounts for 83.24% of CPU
samples and `json.Decoder.Decode` for 81.22%. JSON input-buffer growth accounts
for 55.25% of allocation space, JPEG configuration decoding for 22.48%, and
embedding slice growth for 12.58%. These cumulative CPU percentages overlap;
they must not be summed. Filesystem syscall samples account for 12.09% flat.

The lead checked `/snap/go/current/src/encoding/json/v2_stream.go:74`: this
installed Go 1.27.1's compatibility Decoder first calls `ReadValue`, then
unmarshals that complete value. The cache format is one bounded JSON document.
This supports testing a bounded read followed by `json.Unmarshal`, using the
same standard-library compatibility API and retaining every semantic check.

## Candidate contract and TDD

Files: `internal/similarity/cache_payload.go` and the existing
`cache_policy_test.go`. Read at most the 1 MiB record limit plus one byte for
oversize detection. Accept one complete JSON document with optional trailing
whitespace, reject extra JSON/trailing junk, oversize payloads and reader errors,
and preserve all version/path/vector/digest/preview checks. No cache format,
dependencies, writer or source-version protocol changes.

First add direct decoder boundary tests (exact-limit whitespace, overflow,
trailing documents/junk and read failure), alongside the existing both-store
invalid-record table. Run them and inspect any failure before implementation.
The current helper's limited stream can hide bytes beyond its artificial EOF;
the new boundary guard must reject such input itself even though store callers
already precheck file sizes.

Verify: `go test -race -tags no_emoji -count=1 ./internal/similarity -run
'TestAnalysisCache|TestSearch.*Pipeline|TestSearchSession'`. Compare unchanged and
candidate `BenchmarkSearchWarmPipeline` at 1,000/10,000 inputs, 3 iterations and
3 repetitions, without overlapping tests. Keep a production change only if the
repeated 10,000-source median improves by at least 10% without increased
allocation volume. If the simple candidate does not meet that threshold, record
the result and retain the existing implementation. Any kept change receives
GoLand inspection, negative boundary verification and one `make verify` gate.

## Implementation and measurements

The candidate reads through `io.LimitReader` with `io.ReadAll`, rejects an extra
byte beyond the record limit, then uses `json.Unmarshal`. Every semantic check
after JSON decoding remains unchanged. The boundary test first failed for records
padded beyond the limit: the old helper accepted both limit-plus-one and
limit-plus-1,024-byte records. Existing store callers already rejected oversized
files before this helper, so this finding is not evidence that those cache reads
accepted oversized files. Red log: `/tmp/picfetch-warm-profile-boundary-red.log`.

Focused race regressions passed in 5.623 s with the task-2 filter, covering all
analysis-cache tests and retained search sessions. Log:
`/tmp/picfetch-warm-profile-focused.log`. GoLand inspected both changed Go files,
including weak warnings. The new `bytes` import revealed an existing local
variable with the same name; it was renamed to `recordBytes`, and the final
inspection returned no findings. An incomplete first rename prevented one
benchmark command from compiling; it produced no measurement and was corrected
before the recorded candidate run.

### Comparison

Same native Linux/amd64 host, Go 1.27.1 and Intel Core i7-1360P. Each run used:

```sh
/snap/go/current/bin/go test -tags no_emoji -run '^$' \
  -bench '^BenchmarkSearchWarmPipeline$' -benchtime=3x -count=3 \
  -benchmem ./internal/similarity
```

These timings have no CPU profiler enabled. Baseline and candidate runs were
sequential, with no other tests, builds or IDE inspections initiated by this
task during either measurement. Medians of three repetitions:

| Sources | Before time/op | After time/op | Before bytes/op | After bytes/op | Before allocations/op | After allocations/op |
| ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| 1,000 | 167.173 ms | 136.815 ms | 59,929,434 | 51,212,800 | 66,838 | 64,803 |
| 10,000 | 1,471.340 ms | 1,264.073 ms | 598,197,954 | 511,221,258 | 667,082 | 647,059 |

At 10,000 sources this is 14.1% less time and 14.5% less allocation volume,
meeting the candidate's acceptance threshold. The change is retained. Logs:
`/tmp/picfetch-warm-profile-before.log` (37.941 s) and
`/tmp/picfetch-warm-profile-after.log` (33.598 s). Allocation volume is cumulative
traffic, not retained memory or peak RSS; these synthetic warm-worker timings
do not include inference, subprocess transport or UI paint.

The profile supports this local parsing simplification. It does not justify a
new cache format, a custom JPEG validator, a different ranking algorithm or a
new persistence manager. Further performance work should begin with a user-visible
latency target and a representative corpus.

## Final gate

Three temporary overlays negatively verified the guards: removing the size
check accepted oversized records; removing the bounded reader consumed bytes
beyond limit-plus-one; switching back to a first-value-only streaming decode
accepted extra JSON and trailing junk. Each test failed for that exact reason.
Logs: `/tmp/picfetch-warm-profile-negative-{size,read,trailing}.log`; overlays:
`/tmp/picfetch-warm-profile-overlays/`. Production files were not mutated by
these negative checks. Both final GoLand inspections are clear, and
`git diff --check` passes.

`env PATH=/snap/go/current/bin:$PATH make verify` passed: formatting, TUF and
asset checks, vet, build, shard validation and the complete native Linux/amd64
Docker race suite. All 68 packages outside root UI completed successfully, and
all three root-UI shards passed (317.401 s, 331.042 s and 332.289 s). There were
no failed tests or race reports. The new payload-boundary test passed in 0.220 s
within the full run. Log: `/tmp/picfetch-warm-profile-verify.log`.
Race artifacts: `.scratch/race-runs/20260915T110024Z-HpLdQW/`.

No new test file or root UI test was added, so existing Qodana exclusions and
shard assignments remain sufficient. The lead's review found no further
production changes necessary. Actuals: one read-only scout, zero implementation
agents, one profile capture, two successful comparison runs, one focused race
gate, three negative overlays, one inspection-driven test-variable fix and one
complete verification suite. All production work remained in the shared decoder;
the other files contain coverage, navigation and evidence records.
