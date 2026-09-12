# Timing-informed UI shard rebalance

Status: Complete. Hosted CI confirms the improvement on the rebalance commit
and a September 12 follow-up, with every assigned test accounted for. Measured
balance and runner overhead are recorded below. The historical full local gate
retains the two pre-existing amd64-container seccomp failures.
Route: Standard, confined to the manifest and its evidence.

## Problem and measured evidence

The [Release v1.1.0 run](https://github.com/frathe/picfetch/actions/runs/34627413846),
attempt 1 at `b82dfc86a7f8272234ee714405567f804f27df14`, completed all four Linux
race jobs successfully. This worktree starts at that same commit. GitHub job
timestamps and the complete uploaded `go test -json` streams give:

| Partition | Job wall time | Race step | UI package elapsed | Top-level test sum |
|---|---:|---:|---:|---:|
| non-ui | 3m39s | 2m47s | n/a | n/a (packages overlap) |
| ui-1 | 14m18s | 13m42s | 814.532s | 813.360s |
| ui-2 | 7m16s | 6m32s | 384.802s | 383.560s |
| ui-3 | 5m43s | 4m16s | 250.250s | 249.000s |

The old manifest still described September 3 median weights of about 327s per
shard, followed by 114 additions. The new observation covers all 681 assigned
top-level tests exactly once: 680 pass and the existing filesystem-dependent
`TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset` skips on Linux. Its
assignment and skip behavior are preserved.

`TestVisualSimilarityExplorer` contributes 374.060s to ui-1. Its 87 direct
subtests sum to 374.040s; the longest is `close_files_memory` at 21.020s.
The current manifest accepts only top-level runnable names, and its exact
anchored filters run every child with its parent. Explorer is therefore
indivisible under the existing contract. However, its weight is below the
three-shard mean of 481.973s, so splitting test code is unnecessary. The test's
shared preference reset and cleanup, and the `explorer-ui-test` name filter,
remain intact.

## Decision and projection

Move Explorer from ui-1 to ui-3 and move 26 existing ui-3 tests to ui-1/ui-2.
Preserve all other assignments, three process-isolated UI shards, serial tests
within each process, and the existing race/locale/timeout/uncached-run flags.
No workflow, release, production code, test body, dependency, or asset changes.

The selection uses integer-millisecond weights from this one complete CI attempt.
Repeatedly choose the heaviest and lightest shard, then transfer the test that
minimizes the resulting load range (ties: larger test, then lexical name).
Stop when the maximum projected load is at most 105% of the mean. The tolerance
avoids reshuffling small tests to fit noise in a single observation.

| Shard | Old count | New count | Measured old test sum | Projected new test sum |
|---|---:|---:|---:|---:|
| ui-1 | 224 | 235 | 813.360s | 470.520s |
| ui-2 | 227 | 241 | 383.560s | 470.530s |
| ui-3 | 230 | 205 | 249.000s | 504.870s |

The projected maximum falls by 308.490s (37.9%), to 8m25s of test bodies.
A complete longest-processing-time reassignment would move 482 tests and
project 481.960s / 481.930s / 482.030s, saving only another 22.840s.
The chosen change moves 27 of 681 tests and leaves 654 assignments alone.

These are single-sample projections, not medians or a new CI result. Runner
speed, compilation, setup, `TestMain`, cleanup, and execution-order effects can
change the realized result. Job wall time is not the sum in the final column.
Explorer imposes a 374.060s observed floor on any whole-test allocation; the
total-work floor of 481.973s is larger here. Fresh CI must confirm the improvement.

## Evidence provenance and replay

Download the raw artifacts outside the repository:

```sh
gh run download 34627413846 --pattern 'linux-race-*' --dir /private/tmp/picfetch-shard-34627413846
gh run view 34627413846 --json jobs,headSha,status,conclusion,url
```

Each artifact is `linux-race-ui-N-34627413846-attempt-1`, containing
`go-test-linux-race-ui-N.json`:

| Shard | Artifact ID | SHA-256 of extracted JSON |
|---|---:|---|
| ui-1 | 10275586632 | `ee792d1e6ba496d8cd65099daafe32ab8f3ee1560482905fe2c0fc0a416add3e` |
| ui-2 | 10275376040 | `778e32b9511374c49de95b30704c08c83ff09aed860586845db04fd985dc89f7` |
| ui-3 | 10274907129 | `821be497725db38319abdc6a60b39e09a28364dcb073cfe061293f915b551344` |

Recompute coverage and projections from the raw streams and rebalance manifest:

```sh
python3 - <<'PY'
import json
import subprocess
from pathlib import Path

root = Path('/private/tmp/picfetch-shard-34627413846')
manifest = subprocess.check_output(
    ['git', 'show', '7fa25009fc8bec2faa47552178026daa1c131b40:.github/testshards/internal-ui.tsv'],
    text=True)
rows = [s.split() for s in manifest.splitlines()
        if s and not s.startswith('#')]
assignment = dict(rows)
assert len(rows) == len(assignment)
weights = {}
for file in sorted(root.glob('linux-race-ui-*/*.json')):
    for line in file.read_text().splitlines():
        event = json.loads(line)
        name = event.get('Test', '')
        if name and '/' not in name and event['Action'] in ('pass', 'skip', 'fail'):
            assert event['Action'] in ('pass', 'skip')
            assert name not in weights, name
            weights[name] = round(event['Elapsed'] * 1000)
assert assignment.keys() == weights.keys()
loads = {s: sum(weights[n] for n in assignment if assignment[n] == s)
         for s in ('ui-1', 'ui-2', 'ui-3')}
print(len(weights), {s: ms / 1000 for s, ms in loads.items()})
assert max(loads.values()) * 300 <= sum(weights.values()) * 105
PY
```

## Acceptance, ownership, and validation

Lead owns the manifest, evidence, review, and final verification. One read-only
Scout inspected Explorer fixture lifetimes and name-sensitive references while
the lead obtained timing artifacts. Delegation was bounded to that independent
question, no writes or shared files; `rg`/line reads verify its references.
No implementation or review was delegated.

- AC1: Exact same 681 runnables, no duplicate or omitted assignment, sorted rows
  and accurate counts: baseline/current inventory comparison, replay above,
  generated regex selection audit, and `make check-test-shards`.
- AC2: Slowest projected load within 5% of the mean: replay above. Before the
  edit the audit failed with `813.36 / 383.56 / 249.0`; after the edit it must pass.
- AC3: Existing guard behavior and race/Make contracts: `go test ./scripts/testshards`.
- AC4: Repository final gate: `make verify`, retaining any environment limitations
  separately from the balance estimate. No changed Go files require IDE inspection.
- AC5: Scope: `git diff --check`, `git diff --stat`, and lead diff review.

Budget: one Scout (actual one), one lead review, one full local gate.

### Completed validation

- Baseline replay failed before the edit on the slowest-load assertion, then
  passed with `470.52 / 470.53 / 504.87` after it.
- Compared `HEAD` and current assignments: identical 681-name inventories,
  exactly 27 moves, no duplicates, sorted blocks, matching header counts.
- Audited the actual filters emitted by `testshards regex`: every recorded
  runnable and subtest belongs to exactly one shard; prefix/suffix near-misses
  do not match. `testshards summarize` accepted all three complete baseline
  streams and all three new local UI streams.
- `go test ./scripts/testshards` passed (`6.977s`). Its existing tests cover
  invalid/duplicate/missing assignments, anchored filters, parallel-call refusal,
  and the unchanged Make race contracts.
- `make verify` passed formatting, TUF expiry, Qodana exclusions, vet, native
  build, and the canonical Docker Linux/amd64 inventory check:
  `checked package=github.com/frathe/picfetch/internal/ui runnables=681 shards=3`.
- Every UI partition passed under the existing `-race -count=1 -timeout 30m`
  contract. The final raw-event audit verified all 681 top-level tests executed
  exactly once (680 pass, the same filesystem-dependent test skips) and all
  87 direct Explorer subtests passed on ui-3.

| Local Docker partition | Assigned tests | Outcome | Actual package time |
|---|---:|---|---:|
| ui-1 | 235 | 235 pass | 390.289s |
| ui-2 | 241 | 240 pass, 1 existing skip | 418.436s |
| ui-3 | 205 | 205 pass | 438.937s |

These are actual local correctness-run timings, not a hosted before/after
comparison: all four partitions share the local Docker container, while GitHub
uses separate runners. No new CI run was dispatched and no release was modified.

The complete `make verify` command exited 2 because these unchanged non-UI tests
reproduced the previously documented local `offline worker seccomp: invalid argument`:

- `internal/similarity`: `TestLinuxWorkerIsolation`.
- `scripts/explorereval`: `TestAssetInstall/worker_reaches_asset_check_and_exits`
  (and its parent).

Those same tests passed in the baseline native Linux CI job at this source
commit. No other failing tests were present in the local raw streams, and Docker
reported `OOMKilled: false`. The existing local worker-isolation TODO remains
open; this rebalance does not change or suppress those tests. A clean full local
gate is therefore **not** claimed.

Console log: `/private/tmp/picfetch-shard-34627413846/verify.log`.
Raw local streams, container exit, and memory diagnostics:
`.scratch/race-runs/20260911T174325Z-L93xOa/` (ignored, not committed).

Lead review and `git diff --check` passed. Scope is exactly the manifest, this
record, and `todos.md`; no code files, test bodies, workflows, or dependencies
changed. No GoLand code inspections apply to this documentation/configuration
change. Actual budget: one read-only Scout, one lead review, one full local gate.

## Hosted CI confirmation — September 12, 2026

The outstanding CI confirmation is complete. Two successful attempt-1 runs
were audited against the original Release baseline:

- [Rebalance CI 34632011148](https://github.com/frathe/picfetch/actions/runs/34632011148),
  at `7fa25009fc8bec2faa47552178026daa1c131b40`. Only the manifest and two
  documentation files differ from the baseline, so this is the direct
  implementation comparison with the same 681 tests and unchanged test code.
- [Mosaic CI 34710982261](https://github.com/frathe/picfetch/actions/runs/34710982261),
  at `371a44c9796e5bd4a375beb0d621113b9383b8e9`. This follow-up includes subsequent
  feature and dependency changes and one added root test, `TestHypnoTunnel`,
  on ui-1. All 681 existing assignments are unchanged. It confirms continued
  improvement with 682 tests, rather than isolating the rebalance's effect.

| Run | Shard | Tests | Job wall time | Race step | UI package elapsed | Top-level test sum |
|---|---|---:|---:|---:|---:|---:|
| Rebalance | ui-1 | 235 | 9m18s | 8m46s | 516.516s | 515.250s |
| Rebalance | ui-2 | 241 | 9m21s | 8m41s | 512.983s | 511.710s |
| Rebalance | ui-3 | 205 | 10m55s | 10m19s | 612.286s | 611.050s |
| Mosaic | ui-1 | 236 | 8m36s | 7m48s | 418.239s | 417.000s |
| Mosaic | ui-2 | 241 | 10m42s | 9m48s | 520.285s | 519.010s |
| Mosaic | ui-3 | 205 | 9m44s | 8m57s | 483.428s | 482.320s |

The maximum UI job wall time falls from the baseline's 14m18s to 10m55s
(23.7% shorter) in the direct comparison, then 10m42s (25.2% shorter) in the
follow-up. Maximum test-body time falls from 813.360s to 611.050s (24.9%) and
519.010s (36.2%), respectively. These are observed UI job and test durations,
not overall workflow or release durations.

The maximum test sum is 11.9% above the three-shard mean on the rebalance
commit and 9.8% above it in the follow-up, versus 68.8% in the baseline.
Thus the original projected within-5% balance is not an observed guarantee.
The slowest shard changes from ui-3 to ui-2 between the two new samples.
Job setup and other steps add 32–54 seconds beyond the race step in these
runs; the race step also includes work outside the reported package duration.
Runner and setup variation remain visible. The evidence supports closing the
confirmation item and retaining the assignments; two observations do not
establish long-term medians or justify chasing the changing slowest runner.

### Artifact provenance and verification

All nine raw UI streams, including the original baseline, passed the repository's
`testshards summarize` validation. An independent raw-event audit matched each
stream's top-level terminal events to its run commit's manifest, rejected
duplicates and failures, and verified that the three shards cover the manifest
exactly once. The rebalance run has 680 passes and one existing skip; the mosaic
run has 681 passes and the same skip,
`TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset`, on ui-2. Every UI
package and all four Linux race jobs pass in both hosted runs; overall CI also
passes. The original baseline JSON hashes match the provenance table above.

| Run | Shard | Artifact ID | SHA-256 of extracted JSON |
|---|---|---:|---|
| 34632011148 | ui-1 | 10277795138 | `093cba88c2b8d84b9d7152aa8954d286f86e30d0b5c2927306c695d985816257` |
| 34632011148 | ui-2 | 10276662220 | `cec79c4e6d69e0502b86abae9d22a8691c60bc4fdd8af76f23b9f1a6f1cf4653` |
| 34632011148 | ui-3 | 10277815403 | `64b9446adfb75646e46156c8c01470479f686d79ffce0194d6fd5fcd953e2a07` |
| 34710982261 | ui-1 | 10303685616 | `b4fdbe0c8946280df350b5ad3225ac3a92417c66ed6b201de3c8ca119c24d68c` |
| 34710982261 | ui-2 | 10303622100 | `5093d6f4695a4b011e74b5787d02b8874f650d0bd7a876b72ba1f0ed2ac7f9d2` |
| 34710982261 | ui-3 | 10303083254 | `40bbc90febdb23566037272abdf0ed6720e0411a67dd575d2d52bfeaf1341ec1` |

Downloaded evidence is in `/tmp/picfetch-shard-ci-confirmation/`. Replay downloads
while the workflow's 14-day artifact retention permits:

```sh
root=/tmp/picfetch-shard-ci-replay
mkdir -p "$root"
for run in 34627413846 34632011148 34710982261; do
    gh run download "$run" --repo frathe/picfetch --pattern 'linux-race-ui-*' --dir "$root/$run"
    gh run view "$run" --repo frathe/picfetch --json jobs,headSha,status,conclusion,url > "$root/run-$run.json"
    gh api "repos/frathe/picfetch/actions/runs/$run/jobs" --paginate > "$root/jobs-$run.json"
done
for stream in "$root"/*/linux-race-ui-*/*.json; do
    go run ./scripts/testshards summarize -json "$stream" > "$stream.summary.tsv"
done
```

Use each run's `headSha` with `git show SHA:.github/testshards/internal-ui.tsv`
for its historical assignment inventory. Sum `Elapsed` for terminal `pass` or
`skip` events with a nonempty `Test` containing no `/`; require each assigned
test exactly once in its shard. Package `pass` events have no `Test`. Job and
race-step durations come from `completed_at - started_at` in the jobs API.
The original projection replay above now pins the rebalance manifest so later
test additions cannot invalidate its historical 681-test comparison.

This confirmation changes only documentation and the manifest's evidence link.
No CI run was dispatched, no assignments changed, and no tests were rerun.
The installed Go snap launcher cannot run inside the sandbox; its underlying
Go 1.27.1 binary built the existing standard-library-only summarizer with a
temporary build cache, and all nine stream validations passed. Link, manifest
preservation, and whitespace checks passed. No changed Go files require GoLand
inspection. Confirmation budget: zero delegates, one lead review, no full suite.
