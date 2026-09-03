# 06: Measure, assess, and close the optimization

**What to build:** Give maintainers an evidence-backed decision on the final
CI topology. Three same-commit sharded attempts must prove coverage, failure
gating, balance, elapsed time, and runner cost before the optimization is
closed or an over-target critical path is presented for explicit review.

**Blocked by:** 05: Run all required CI work concurrently.

**Status:** resolved

- [x] The user commits and pushes the prepared sharded checkpoint, and the
      initial workflow is rerun twice only after each preceding attempt
      completes.
- [x] All three valid attempts share one Actions SHA and ref; creation/rerun
      request times, logs, and artifacts are recorded separately.
- [x] Each attempt's raw artifacts are downloaded and hashed outside the
      repository before the next rerun starts, and infrastructure-failed
      attempts are identified and replaced.
- [x] Every attempt proves that validation, all four Linux race entries, and
      native Windows tests are mandatory, independently scheduled where runner
      capacity permits, and successful.
- [x] The union of final Linux results equals the complete module package and
      exact UI top-level test inventories with no missing or repeated entry.
- [x] The comparison reports pre/post medians for queue-inclusive reusable-CI
      gate time, queue-excluded execution, validation/setup, every race entry,
      package/test durations, the critical path, and summed runner minutes.
- [x] The target assessment treats an 8-12 minute median as an aim rather than
      a correctness gate and gives test quality priority.
- [x] A slowest/fastest UI-shard median ratio at or below 1.20 causes no
      cosmetic manifest churn.
- [x] A ratio above 1.20 that affects the critical path permits at most one
      measured rebalance within the same three shards, followed by local
      verification and three new same-commit CI attempts.
- [x] A safe, reasonably balanced result above 12 minutes pauses closure and
      presents the measured critical path for explicit maintainer disposition;
      it does not automatically add a fourth shard or weaken testing.
- [x] A post-change runner-minute median above twice baseline is highlighted
      and explained without removing coverage or hiding failures.
- [x] The final work record documents the whole-process ordering limitation,
      run IDs, SHAs, attempt results, shard counts, accepted exception if any,
      and the increased concurrent-capacity/runner-minute trade-off.
- [x] The open work item moves to Done and the implementation plan moves to the
      completed-work record only after every accepted safety and measurement
      criterion passes.
- [x] No raw CI or local timing artifact remains in source control, and final
      formatting/status checks show only intended changes.

## Comments

- 2026-09-03: Resolved from three successful attempts of
  [run 33758913251](https://github.com/frathe/picfetch/actions/runs/33758913251)
  at exact SHA `b91d56ce67b991b595484a50d9b65c52f1a09dba` on
  `feature/CI-test-performance-improvements`. Attempt 1 used the workflow
  creation time; the attempt 2 and 3 origins were recorded immediately before
  their rerun requests. Each preceding attempt's four raw streams and six
  required-job logs were downloaded, parsed, and hashed off-tree before the
  next request. No infrastructure attempt needed replacement.
- 2026-09-03: The median required gate fell from 22:17 to 7:52 and the
  queue-excluded window from 22:10 to 7:36. The exact final inventory is 44
  non-UI packages plus 567 UI top-level tests split 177/178/212; every UI test
  passed in every attempt with no missing or duplicate assignment. The UI
  package medians have a 1.064 slowest/fastest ratio, so the 1.20 rebalance
  condition did not trigger and the manifest was left unchanged.
- 2026-09-03: Median required runner time rose from 23:26 to 30:43 (+31.1%)
  while peak required capacity rose from two jobs to six. This remains below
  the 46:52 two-times review threshold. The completed implementation record
  contains the full timing, coverage, artifact, failure-gating, critical-path,
  and whole-process ordering assessment. No target exception or rebalance was
  accepted.
