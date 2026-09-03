# 01: Capture a controlled monolithic CI baseline

**What to build:** Give maintainers a trustworthy, reproducible picture of the
current required CI critical path before changing its topology. The existing
Linux race suite remains monolithic while producing machine-readable evidence
from three attempts at one commit, allowing setup, validation/build, non-UI,
and exact UI time to be distinguished without changing coverage.

**Blocked by:** None (can start immediately).

**Status:** resolved

- [x] The monolithic Linux job is pinned to Ubuntu 24.04 and retains the
      English UTF-8 locale, race detector, 30-minute package timeout, and full
      package scope.
- [x] Test-result caching is disabled and machine-readable Go test events are
      captured while the real test exit status remains authoritative.
- [x] Raw output is uploaded on success or failure under an attempt-specific
      name, fails visibly when absent, and uses 14-day retention.
- [x] Formatting, trust-root validation, vetting, normal build, Windows
      cross-build, Windows vet, native Windows tests, cancellation, and
      reusable-workflow release gating retain their existing behavior.
- [x] The user commits and pushes the measurement-only checkpoint, and the
      initial run is rerun twice only after each preceding attempt completes.
- [x] All three valid attempts have the same Actions SHA and ref. The initial
      creation time and both rerun request times are recorded so queue-inclusive
      durations remain meaningful.
- [x] Each attempt's raw artifact is downloaded and hashed outside the
      repository before the next rerun begins. Required-job logs are also
      downloaded and hashed; the documented comment below records that the
      full log files were materialized after attempt 3.
- [x] Infrastructure-failed or incomplete attempts are identified and replaced
      rather than included in the baseline median.
- [x] The baseline record includes per-attempt and median gate time,
      queue-excluded execution, step timings, total runner minutes, package
      durations, and exact UI top-level test durations.
- [x] No raw timing artifact is added to source control.

## Comments

- 2026-09-03: Prepared the measurement-only checkpoint. YAML parsing,
  `git diff --check`, and `make fmt-check` pass. The workflow diff changes only
  the explicit Linux runner pin, race-test measurement flags/capture, and raw
  artifact upload. Waiting for the user-owned commit and push before collecting
  the three same-SHA Actions attempts; repository policy leaves commits to the
  user.
- 2026-09-03: Resolved from three successful attempts of
  [run 33734286005](https://github.com/frathe/picfetch/actions/runs/33734286005)
  at exact SHA `fcd11103969ac8d1b5a1a220270c0730cb5e1913`. Median required-gate
  wall time is 22:17, queue-excluded execution is 22:10, the Linux race step is
  19:35, exact `internal/ui` is 16:14.021, and total required runner time is
  23:26. All attempts contain the same 44 package terminals and the same 567
  passing top-level UI tests with no failures or top-level skips.
- 2026-09-03: Raw attempt artifacts were downloaded and hashed before each
  subsequent rerun. Full required-job logs were downloaded and hashed after the
  third attempt; GitHub retained the attempt-addressable logs and their timing
  metadata had already been inspected, so this collection-order deviation lost
  no evidence. The complete off-tree record is under
  `/private/tmp/picfetch-ci-baseline.XeJlI7`; its deterministic 1,191-line
  package/test/allocation report hashes to
  `bb61c7b69d41e98b22ef15575a7614b13363b14d029a4cd9b96be6d236454803`.
  No raw data is tracked by Git.
