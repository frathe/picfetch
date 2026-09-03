# 02: Generate the deterministic shard manifest

**What to build:** Let maintainers turn the three accepted baseline event
streams into a reproducible, reviewable three-shard assignment. The same inputs
must always produce the same summary and assignment, and incomplete timing data
must fail rather than silently inventing weights.

**Blocked by:** 01: Capture a controlled monolithic CI baseline.

**Status:** resolved

- [x] A standard-library Go command reads interleaved machine-readable test
      streams and reports package durations plus terminal top-level test
      outcomes.
- [x] Malformed or truncated input, duplicate terminal outcomes, missing
      terminal outcomes, and tests absent from any required baseline fail with
      actionable diagnostics.
- [x] Three-run summaries use the median duration for each top-level test and
      never treat summed test durations as workflow wall time.
- [x] Planning assigns every observed Linux top-level UI test, fuzz target, and
      example to exactly one of three shards while keeping subtests with their
      parent.
- [x] Planning uses deterministic longest-processing-time balancing with
      test-name ordering for equal durations and shard-number ordering for equal
      loads.
- [x] Repeating summary and planning with identical inputs produces
      byte-identical normalized output.
- [x] The checked-in manifest records baseline provenance and the complete
      initial name-to-shard assignment generated from the three accepted CI
      attempts, not from the one local feasibility run.
- [x] Command-level tests cover interleaved packages, pass/fail/skip terminals,
      subtests, median calculation, incomplete baselines, and every balancing
      tie-break.
- [x] The new tooling package is recorded in the architecture map and its test
      file is included in the repository's exact Qodana test exclusions.
- [x] Focused tooling tests and the Qodana exclusion check pass without running
      the full race suite.

## Comments

- 2026-09-03: Claimed after ticket 01 produced three accepted same-SHA CI
  streams. The implementation seam is the `scripts/testshards` command's
  observable files/stdout/stderr/exit status; live Linux inventory validation
  and filter generation remain ticket 03.
- 2026-09-03: Resolved with standard-library `summarize` and `plan` command
  paths plus command-boundary tests. The real three-run summary contains 44
  package rows and 2,027 top-level runnable rows and reproduced byte-for-byte
  at SHA-256
  `933691863a4ed010c18beaea88d5b2b3420fc892b157e34d1968a19ca041fefd`.
- 2026-09-03: The checked-in manifest was generated from Actions run
  `33734286005`, attempts 1-3, at baseline SHA
  `fcd11103969ac8d1b5a1a220270c0730cb5e1913`. Host and repeated Linux/amd64
  Docker generation were byte-identical at SHA-256
  `6468a50ce0925ade39d253be7e7b0e728aa3d38fdc9f32a6b5547347155c5a18`.
  Its 567 unique entries are split 177/178/212 with median-weight sums
  326.750s/327.070s/326.740s; no subtest or `TestMain` is assigned.
- 2026-09-03: `go test ./scripts/testshards -count=1 -v`, `make vet`,
  `make fmt-check`, `make check-qodana-test-exclusions`, and
  `git diff --check` pass. Per the ticket budget, no full race suite ran.
