# Spec: measured CI test sharding

Status: complete

## Problem Statement

PicFetch's required GitHub Actions CI gate takes roughly 21-24 minutes even
though much of its validation, build, Windows, non-UI test, and UI test work is
independent. The current Linux job serializes validation and build steps before
one race-enabled test command, and the exact `internal/ui` package dominates
that command.

Maintainers and contributors need faster feedback without trading away test
quality. Every currently covered Linux and Windows package must remain covered,
the Linux race detector and 30-minute package timeout must remain enabled, and
failures must continue to block reusable CI and tag releases. The change must
be justified by repeatable CI measurements rather than by one unusually fast
or slow runner.

## Solution

Measure the existing monolithic Linux race suite three times at one commit,
then run independent validation/build, Linux race, and Windows test work at the
same time. Keep non-UI Linux packages in one race entry and distribute the
exact `internal/ui` top-level tests across three isolated hosted runners using
a deterministic, duration-balanced manifest.

Add a guard that compares the checked-in assignment with the Linux-selected
test inventory and fails the required CI gate when a test is missing, repeated,
stale, malformed, or assigned to an invalid or empty shard. Use a small Go
helper to summarize machine-readable test output, build and validate the
assignment, emit safe exact-match filters, and retain concise failure output
alongside raw timing artifacts.

Measure the sharded result three times at one later commit. Aim for an 8-12
minute median reusable-CI gate, but treat that range as a goal rather than a
hard condition. Complete, reliable testing wins over elapsed time. Report both
the critical path and the increase in concurrent runner capacity and total
runner minutes.

## User Stories

1. As a contributor, I want required CI feedback sooner, so that I can respond
   to defects without waiting through avoidable serialization.
2. As a maintainer, I want performance changes based on three comparable CI
   attempts, so that runner variance is not mistaken for an improvement.
3. As a maintainer, I want all baseline attempts to use one commit, so that
   source changes do not contaminate the comparison.
4. As a maintainer, I want all post-change attempts to use one later commit, so
   that the final median is internally comparable.
5. As a maintainer, I want test-result caching disabled for measurements, so
   that every attempt executes the test bodies being timed.
6. As a reviewer, I want workflow creation, job execution, setup, package, and
   top-level test durations reported separately, so that I can identify the
   real critical path.
7. As a reviewer, I want initial queue delay separated from execution time, so
   that runner availability is not confused with code or workflow performance.
8. As a project owner, I want validation and build work to start independently
   from race tests, so that neither group waits for work it does not consume.
9. As a project owner, I want formatting, trust-root validation, vetting,
   normal builds, Windows cross-builds, and Windows vetting preserved, so that
   faster CI does not weaken non-test validation.
10. As a Windows user, I want the existing native Windows package tests to
    remain independent and unchanged, so that release confidence on Windows is
    not traded for Linux speed.
11. As a release operator, I want every validation and test entry to remain a
    mandatory part of reusable CI, so that a tag release cannot proceed after
    any failure.
12. As a maintainer, I want the Linux race detector retained for every Linux
    package, so that concurrency regressions remain visible.
13. As a maintainer, I want the 30-minute package timeout retained in every
    Linux race invocation, so that long UI packages do not fall back to Go's
    shorter default timeout.
14. As a maintainer, I want the explicit English locale retained, so that Fyne
    does not emit irrelevant locale parsing failures.
15. As a reviewer, I want Linux jobs pinned to one named runner image, so that
    moving aliases do not undermine baseline/post-change comparability.
16. As a test author, I want all non-UI module packages included automatically,
    so that root, tooling, and UI feature subpackages are not accidentally
    omitted.
17. As a test author, I want only the exact main UI package removed from the
    non-UI set, so that similarly prefixed UI subpackages still run with the
    race detector.
18. As a test author, I want each top-level UI test assigned exactly once, so
    that sharding neither loses coverage nor repeats work.
19. As a test author, I want subtests to remain with their top-level parent, so
    that existing setup, cleanup, and process-state assumptions remain intact.
20. As a test author, I want a newly added top-level test to fail the guard
    until it receives an explicit assignment, so that coverage cannot silently
    decay.
21. As a test author, I want future tests, fuzz targets, and examples included
    in the Linux inventory, so that supported top-level test forms cannot evade
    the guard.
22. As a maintainer, I want duplicate and stale manifest names rejected, so
    that the assignment remains an exact description of the current suite.
23. As a maintainer, I want malformed rows, unknown shard names, and empty
    shards rejected, so that invalid configuration cannot degrade into a
    successful no-op test command.
24. As a maintainer, I want every shard filter anchored and non-empty, so that
    similarly named tests or accidental match-all expressions cannot alter
    coverage.
25. As a reviewer, I want shard generation deterministic, so that identical
    timing inputs produce an identical reviewable manifest.
26. As a reviewer, I want initial balancing based on median CI durations, so
    that one outlier does not dominate assignment.
27. As a reviewer, I want deterministic tie-breaking during balancing, so that
    manifest changes do not depend on map or filesystem iteration order.
28. As a maintainer, I want UI shards isolated on separate hosted runners, so
    that Fyne process-global state is not made concurrent inside one process.
29. As a maintainer, I want new `t.Parallel()` use in the exact UI test package
    to force an explicit safety review, so that in-process concurrency is not
    introduced casually.
30. As a contributor, I want a failed job name to identify `non-ui` or the
    exact UI shard, so that the failing area is immediately visible.
31. As a contributor, I want failure output to identify the exact package and
    test while remaining concise, so that diagnosis does not require scanning
    an entire raw event stream.
32. As a CI investigator, I want raw machine-readable test events retained on
    both success and failure, so that timings and failures can be audited after
    the run.
33. As a CI investigator, I want artifact names to include the matrix entry and
    workflow attempt, so that repeated same-commit measurements cannot be
    confused.
34. As a project owner, I want all matrix entries to finish after one entry
    fails, so that one run reveals every failing shard rather than only the
    first one.
35. As a project owner, I want a newer superseding workflow to cancel the older
    one, so that obsolete work does not consume runners unnecessarily.
36. As a local contributor, I want one canonical verification command to run
    the same race partitions sequentially in Linux/amd64 Docker, so that local
    verification matches CI without needing six hosted runners.
37. As a macOS contributor, I want manifest validation to use a Linux build
    inventory, so that Darwin-only tests are not mistaken for Linux coverage
    gaps.
38. As a maintainer, I want the three UI shards rebalanced only after measured
    post-change evidence, so that routine test additions do not create noisy
    global reshuffles.
39. As a maintainer, I want a greater-than-20-percent shard imbalance reviewed,
    so that one poorly balanced shard does not become the avoidable critical
    path.
40. As a project owner, I want a result above 12 minutes explained rather than
    hidden, so that performance goals never override test quality.
41. As a project owner, I want a fourth shard to require a separate measured
    decision, so that concurrent runner consumption does not grow
    automatically.
42. As a project owner, I want pre/post total runner minutes reported, so that
    lower elapsed time is considered alongside capacity and resource cost.
43. As a project owner, I want a more-than-twofold runner-minute increase
    highlighted for review, so that unusually expensive parallelization is
    visible without turning cost into a reason to weaken tests.
44. As a release operator, I want the established reusable-workflow release
    gate left intact, so that this optimization cannot bypass signed-release
    safety.
45. As a maintainer, I want raw measurement data kept outside source control,
    so that the repository contains decisions and summaries rather than
    transient CI artifacts.
46. As a reviewer, I want the concurrency trade-off and whole-process test
    limitation documented, so that acceptance is based on an honest safety
    model.

## Implementation Decisions

- Use two explicit implementation checkpoints. The first changes only
  measurement behavior and pins the Linux runner; the second introduces the
  final parallel topology and shard execution.
- The user creates and pushes each checkpoint commit. Each checkpoint is run
  once and rerun twice so its three samples retain the same Actions SHA and
  ref.
- For rerun attempts, record the rerun request time before submission and
  download/hash each attempt's artifact before starting the next attempt.
  This keeps the queue-inclusive origin and per-attempt evidence unambiguous.
- Pin all Linux work to Ubuntu 24.04.
- Use one validation/build job, one four-entry Linux race matrix, and the
  existing independent Windows test job. This produces six required job slots:
  validation, non-UI race tests, three UI race shards, and Windows tests.
- Do not place dependency edges between those jobs. The reusable workflow's
  aggregate result remains the dependency used by tag releases.
- Keep workflow-level cancellation of superseded work. Disable matrix
  fail-fast so every shard reports its result. Do not permit optional failures.
- Keep the exact current Windows test command and its existing lack of the race
  detector; Windows scope does not expand or contract in this work.
- Every Linux race invocation uses the English UTF-8 locale, the race detector,
  a fresh test execution count, and the 30-minute package timeout.
- Determine the non-UI package set from the module's complete package list by
  subtracting only the exact `internal/ui` import path. Refuse an empty or
  inconsistent partition.
- Use exactly three UI shards initially. Each top-level Linux-selected test,
  fuzz target, or example in the exact UI package belongs to exactly one shard;
  subtests remain attached to their parent.
- Store the assignment as a checked-in, reviewable name-to-shard manifest with
  baseline provenance. New tests require an explicit manifest update rather
  than an automatic broad reshuffle.
- Build the initial assignment from the median duration of each top-level test
  across the three controlled baseline attempts.
- Balance with deterministic longest-processing-time assignment: descending
  median duration, test-name ordering for equal durations, and shard-number
  ordering for equal shard loads.
- Add a standard-library Go helper with command-level behaviors for JSON
  capture, duration summaries, deterministic planning, live Linux inventory
  validation, safe shard-filter generation, and readable failure summaries.
- Validate the whole manifest before emitting a shard filter. Filters are
  anchored, escaped for Go's regular-expression engine, and never empty or
  match-all.
- The live guard rejects missing, duplicate, stale, malformed, unknown-shard,
  and empty-shard configurations. It also rejects newly introduced parallel
  calls in Linux-selected tests of the exact UI package pending explicit
  review.
- Run the canonical manifest check under Linux/amd64. A direct checker may run
  on CI Linux, but a public local command must not accept a Darwin-derived
  inventory as canonical.
- Keep the normal non-race Docker suite intact. Change the race and full
  verification paths to run the non-UI partition and three UI shards
  sequentially in one Linux/amd64 container, reusing setup and build caches.
- In CI, every race entry performs its own checkout, Go setup, and Linux GUI
  dependency setup so it is independently runnable.
- Stream compact package/test/failure information while persisting the raw Go
  JSON stream. Shell pipeline semantics must preserve a failure from either the
  test command or the capture helper.
- Upload one raw artifact per Linux matrix entry even after test failure. Treat
  a missing artifact as an error and retain routine artifacts for 14 days.
- Measure the reusable-CI gate from attempt submission/creation to the last
  required completion. Also report the execution window from first required
  job start to last completion, individual job/step medians, shard/package/test
  medians, and summed runner minutes.
- Do not add a timing-based CI pass/fail threshold. Timing determines review
  and rebalancing, not correctness.
- Rebalance within the same three shards only when three post-change attempts
  show a slowest/fastest median ratio above 1.20 that affects the critical path,
  or when imbalance explains a missed target. Permit at most one measured
  rebalancing round in this pull request.
- If a safe and reasonably balanced three-shard result remains above 12
  minutes, stop before final acceptance and present the measured critical path.
  Do not add a fourth shard, skip checks, retry failures, or weaken coverage
  automatically.
- Treat a total-runner-minute median above twice the monolithic baseline as a
  mandatory review note, not as an automatic rejection or permission to remove
  tests.
- Update the repository architecture map for the new tooling package and close
  the open work item only after the final three-run comparison is accepted.
- Do not create a domain ADR: the sharding topology is reversible operational
  configuration and does not change PicFetch's product domain model.

## Testing Decisions

- The highest useful seams are the test-sharding command, the canonical
  Make/Docker verification command, and completed GitHub Actions runs. No
  application-level seam or UI interaction is introduced for this work.
- Test the helper through CLI-equivalent inputs and observable outputs: raw Go
  JSON streams, manifest text, current inventory, emitted filters, summaries,
  diagnostics, and exit status. Avoid assertions on incidental parser data
  structures or algorithm implementation details.
- Use fixture event streams to cover interleaved packages, top-level tests and
  subtests, pass/fail/skip terminals, malformed and truncated JSON, missing
  terminal events, and repeated same-test observations across three runs.
- Prove deterministic planning by supplying equal durations and equal shard
  loads, rerunning with identical inputs, and requiring byte-identical
  normalized output.
- Prove the manifest safety boundary with deliberately invalid fixtures for an
  unassigned current test, duplicate assignment, stale name, malformed row,
  unknown shard, empty shard, and newly added parallel call. Each diagnostic
  names the offending test, row, or shard.
- Prove that each emitted filter is anchored, non-empty, valid under Go's
  regular-expression implementation, matches every assigned name, and matches
  no name owned by another shard.
- Prove package coverage by comparing the observed non-UI set plus the exact UI
  package with the full module package inventory. Root, tooling, and all UI
  feature subpackages must appear in the union.
- Prove capture behavior with a deliberately failed test stream: concise output
  names the package/test, the raw stream remains readable, and the overall
  pipeline remains failed.
- Existing Make targets are the prior art for canonical Linux/amd64 Docker
  execution. The final full local acceptance seam remains `make verify`.
- The existing UI harness and its explicit cleanup/drain behavior are prior art
  for process-state containment. No broad UI-test parallelism is added.
- The first workflow checkpoint must produce three valid monolithic attempts.
  An infrastructure failure is labeled and replaced rather than included in a
  performance median.
- The final checkpoint must produce three valid sharded attempts at one SHA.
  Every required job must pass and all raw artifacts must parse to a complete,
  non-overlapping package/test union.
- Inspect actual job logs to confirm the locale, race, count, and timeout flags;
  do not infer them solely from generated configuration.
- Inspect the completed workflow graph to confirm all six entries are required,
  independent where runners permit, and visible after a shard failure.
- Compare the Windows command against the base revision and verify the release
  workflow has no source diff.
- Run the complete local race/verification suite once after the final topology
  is assembled, with at most one defect-driven rerun. Do not regenerate golden
  screenshots.
- Record the final gate median, queue-excluded median, critical entry, shard
  balance, package/test coverage, and runner-minute median in the completed
  work record.

## Out of Scope

- Removing, skipping, retrying, or making any existing validation or test
  optional.
- Weakening Linux race coverage, the package timeout, locale handling, or
  package inventory.
- Broad use of in-process test parallelism.
- Automatically increasing the UI shard count beyond three.
- Changing product code, UI behavior, translations, golden screenshots, or
  application package boundaries.
- Expanding or shrinking Windows test coverage or installing a Windows race
  toolchain.
- Changing release packaging, signing, publication, or reusable-workflow
  dependencies.
- Changing repository rulesets, branch protection, or required-status-check
  administration.
- Committing raw CI JSON or local timing artifacts.
- Creating new domain terminology, `CONTEXT` entries, or an ADR.

## Further Notes

- The prerequisite first signed Windows release is complete; this work remains
  a separate pull request.
- Recent coarse CI evidence has a 22:00 median critical job, 2:29 median work
  before the race step, 19:34 median race step, and 16:16 median exact UI
  package. Controlled checkpoint data supersedes these observations.
- A fresh Linux/amd64 Docker run at clean commit
  `6af8c51f9bdfb0b36907be8b5c8241586ec3a602` completed in 13:22.04 wall time.
  Its JSON execution span was 11:13.53; the exact UI package took 11:12.513 and
  all 567 Linux top-level UI tests passed.
- A three-way deterministic allocation of that one local run produced
  theoretical test-only totals of 223.72s, 223.81s, and 223.83s. This proves
  feasibility only; the initial checked-in manifest uses the three fresh CI
  baselines.
- Lower elapsed time consumes up to six concurrent hosted runners and may use
  more total runner minutes than the monolithic job.
- Honest limit: after sharding, no required job executes every exact UI test in
  one process. The manifest proves coverage and non-overlap but cannot prove
  the absence of test-order dependencies. Separate runners, the existing UI
  harness cleanup, and the prohibition on unreviewed parallel calls are the
  accepted safety boundary.
- The companion Deep implementation plan contains the phased task graph,
  acceptance commands, measurement tables, stop rule, and cost ledger.
