# Measured CI test sharding

Status: Complete. The controlled monolithic and sharded checkpoints each have
three accepted same-commit attempts; the final topology passes its coverage,
failure-gating, balance, elapsed-time, and runner-cost assessment.

Route: Deep

Prerequisite: satisfied. The first signed Windows release completed in the
[`v0.2.17` release run](https://github.com/frathe/picfetch/actions/runs/33668786016),
including the Windows signing gate. This work remains a separate pull request.

## Deliverable

Reduce the elapsed time of the reusable GitHub Actions CI gate by running its
independent work concurrently and by splitting the exact `internal/ui` package
across three isolated Linux race-test runners. Preserve every existing test,
package, race-detector invocation, timeout, locale, Windows check, and release
gate. Base shard assignment and the final assessment on measured data rather
than an assumed 8-12 minute result.

The pull request has two mandatory CI checkpoints:

1. an instrumented but still monolithic Linux race run, repeated three times at
   one commit; and
2. the final sharded workflow, repeated three times at one later commit.

The agent does not create either commit. At each checkpoint the user commits
and pushes the prepared tree, then reruns the same workflow attempt twice so
all three samples use the same SHA.

## Scope

- Instrument the current CI well enough to separate workflow/job setup,
  validation/build work, non-UI packages, and exact `internal/ui` tests.
- Add a small standard-library Go command at `scripts/testshards` for duration
  summaries, deterministic assignment, manifest validation, anchored shard
  regexes, and concise JSON capture.
- Check in an exact Linux `top-level test name -> shard` manifest for three UI
  shards.
- Make local `make verify` exercise the same four Linux race commands
  sequentially in one Linux/amd64 Docker container.
- Split CI into one validation/build job, a four-entry Linux race matrix, and
  the existing independent Windows test job.
- Record both elapsed-time improvement and the runner-capacity trade-off.

## Non-goals

- Removing `-race`, changing the 30-minute package timeout, skipping tests,
  narrowing package coverage, making jobs optional, or using
  `continue-on-error`.
- Broadly adding `t.Parallel()` to UI tests. The exact `internal/ui` package
  intentionally depends on process isolation for Fyne and other global state.
- Adding a fourth UI shard automatically. A result above 12 minutes is a
  measurement and review point, not permission to weaken the suite or consume
  more runners.
- Refactoring application packages or changing product behavior.
- Changing the Windows test command or adding a Windows race toolchain.
- Changing release packaging/signing, `.github/workflows/release.yml`, branch
  protection, or repository rulesets.
- Committing raw JSON artifacts or keeping local/CI timing files in the source
  tree.
- Creating `CONTEXT.md` or an ADR. This is a reversible CI implementation and
  introduces no new domain terminology or durable architectural policy.

## Evidence before implementation

### Existing CI observations

The current `build-and-test` job serializes validation/build work before one
`go test -race ./...` command. The exact `internal/ui` package dominates that
command. These three recent current-code executions are useful coarse evidence,
but they are not the controlled baseline required by this plan because one is
a reusable release invocation and the attempts do not share one SHA.

| Execution | Critical job | Before race step | Race step | Exact `internal/ui` |
|---|---:|---:|---:|---:|
| [Current `main`](https://github.com/frathe/picfetch/actions/runs/33673120188) | 22:00 | 2:23 | 19:36 | 16:19 |
| [Release commit on `main`](https://github.com/frathe/picfetch/actions/runs/33668780732) | 23:52 | 4:15 | 19:34 | 16:16 |
| [Reusable CI in the release](https://github.com/frathe/picfetch/actions/runs/33668786016) | 21:42 | 2:29 | 19:10 | 16:03 |
| Median | **22:00** | **2:29** | **19:34** | **16:16** |

The durations are not additive: Go can run packages concurrently within the
single race command. Phase 1 must replace this coarse comparison with three
instrumented attempts and explicit package/test data.

### Fresh local Linux/amd64 baseline

Docker was restarted and the canonical Make target was run once at clean HEAD
`6af8c51f9bdfb0b36907be8b5c8241586ec3a602`:

```sh
/usr/bin/time -p make --silent test-race \
  TEST_RACE='-race -count=1 -json' \
  > /private/tmp/picfetch-baseline.dKv4tT/go-test-linux-amd64.json
```

| Observation | Duration/result |
|---|---:|
| Docker wall time, including image setup and locale installation | 13:22.04 |
| First-to-last JSON event | 11:13.53 |
| Setup/compile difference | 2:08.51 |
| Exact `internal/ui` package | 11:12.513 |
| UI top-level terminal events | 567/567 passed |
| Sum of UI top-level elapsed values | 11:11.36 |
| Last non-UI package completion after the first timed event | 0:27.92 |

The raw file has 11,401 lines, is 2,480,273 bytes, and has SHA-256
`e728a20d192448490691b87cb11ed7ed21ed4fda64f06b5c21669d2222debe5c`.
It is temporary evidence, not a repository artifact.

The five slowest UI tests in that run were:

| Test | Elapsed |
|---|---:|
| `TestCompareCommandEntryPoints_MenuAndFeatureCallbacksAreIgnored` | 29.05s |
| `TestCopySelectionCancelsBeforeOtherCommands` | 26.19s |
| `TestCompareOpenRefusal_DropDialogShortcutAndOpenWithAreDiscarded` | 9.85s |
| `TestCopySelectionAvailability` | 8.33s |
| `TestCompareHelp_F1OpensManualWithoutLeavingComparison` | 7.03s |

A deterministic longest-processing-time allocation of this single run would
produce theoretical test-only totals of 223.72s, 223.81s, and 223.83s, with
211, 177, and 179 top-level tests. Those figures are only a feasibility check:
the checked-in manifest must be generated from the three fresh CI baseline
attempts, and real shards also repeat compilation and `TestMain` setup.

The large local/CI difference confirms that local data is suitable for test
weights and correctness checks, while CI data is authoritative for CI elapsed
time and acceptance.

## Locked decisions

| Area | Decision |
|---|---|
| Primary metric | Reusable CI gate wall time, from workflow creation until every required called-workflow job completes. |
| Secondary metrics | Queue-excluded execution window, each job/step duration, shard/package/test medians, and summed runner minutes. |
| Target | Aim for an 8-12 minute median. It is not a hard gate; test quality and complete coverage take priority. |
| Comparable samples | Three attempts at one monolithic checkpoint SHA and three attempts at one sharded checkpoint SHA. Each Linux race command uses `-count=1`; rerun request time is recorded for the queue-inclusive metric. |
| Linux runner | Pin all Linux jobs to `ubuntu-24.04`; do not use the moving `ubuntu-latest` label. |
| Job topology | One validation/build job, one four-entry `linux-race` matrix (`non-ui`, `ui-1`, `ui-2`, `ui-3`), and the existing Windows test job: six concurrent job slots in total. |
| Dependencies | The six jobs have no `needs` edges between them. Completion of the reusable workflow remains the release gate, so every job is mandatory. |
| Matrix failure policy | `fail-fast: false`, with the workflow's existing `cancel-in-progress: true`; no optional jobs and no hidden failures. |
| UI shard count | Exactly three initially. Do not add a fourth without a new measured decision. |
| Assignment | A checked-in exact mapping from each Linux-selected top-level `Test`, `Fuzz`, or `Example` in package `internal/ui` to exactly one of `ui-1..ui-3`. Subtests remain with their parent. |
| Initial balance | Median each top-level test's duration across the three baseline JSON files, then use deterministic longest-processing-time assignment. Sort equal durations by test name; break equal shard loads by shard number. |
| Guard | Fail on an unassigned inventory name, duplicate assignment, stale name, malformed row, unknown shard, empty shard, or newly introduced `t.Parallel()` call in the exact UI package. |
| Non-UI scope | `go list ./...` minus only the exact import path returned by `go list ./internal/ui`; keep the root package, scripts, and all `internal/ui/...` subpackages. |
| Linux safety flags | Every final Linux test invocation retains `LANG=en_US.UTF-8`, `-race`, `-count=1`, and `-timeout 30m`. |
| Windows scope | Preserve `go test ./internal/update/... ./internal/ui/autoupdate/...` as an independent `windows-latest` job. |
| Logs/artifacts | Stream concise failure/test summaries, upload raw `go test -json` for every Linux matrix entry with `if: always()`, `if-no-files-found: error`, and 14-day retention. Include matrix shard and run attempt in names. |
| Local parity | Public `make check-test-shards` is Linux/amd64-Docker canonical. `make verify` checks the manifest and runs `non-ui`, then `ui-1..ui-3`, sequentially in one such container with the same flags and package selections as CI. |
| Rebalancing | No timing-based CI guard. Rebalance within three shards only after three post-change attempts if slowest/fastest shard medians differ by more than 20%, or the gate misses the target because of shard imbalance. Any rebalance creates a new SHA and restarts the three-attempt comparison. |
| Runner cost | Report it. More than 2x the baseline runner-minute median triggers review, never automatic test weakening or rejection by itself. |
| Over-target result | If three safe, reasonably balanced shards still have a median above 12 minutes, stop before final handoff and present the measured critical path. Do not silently add capacity or weaken checks. |
| Release gate | Leave `release.yml` unchanged. A failed job in the called reusable workflow already fails its caller and blocks every release job that needs `test`. |

## Honest limit

After sharding, no required job runs all exact `internal/ui` top-level tests in
one process. This can change behavior if two tests accidentally depend on
process order or on state left by a test assigned to another shard. That loss
of whole-package process-order coverage is accepted because the package has no
current `t.Parallel()` calls, its harness drains and resets owned background
state, and the requested isolation boundary is separate hosted runners.

The manifest guard prevents coverage gaps and overlap; it cannot prove that
tests are independent. The new `t.Parallel()` guard forces an explicit safety
review before this concurrency model can change. A future ordering-dependent
failure must be fixed as a test-isolation defect or motivate a separately
approved monolithic check; it must not be hidden by retries or optional jobs.

## Planned file changes

| File | Purpose |
|---|---|
| `.github/workflows/ci.yml` | First add monolithic measurement capture; later replace the serialized job with the validation job and four-entry race matrix while preserving Windows and reusable-workflow gating. |
| `.github/testshards/internal-ui.tsv` | Checked-in exact `test-name<TAB>ui-N` assignment plus provenance comments for the baseline SHA/attempts. |
| `scripts/testshards/main.go` | Standard-library CLI for JSON capture/summaries, deterministic planning, inventory validation, parallel-call guard, and anchored regex output. |
| `scripts/testshards/main_test.go` | Parser, balancing, manifest, regex, inventory, capture, and negative guard tests. |
| `Makefile` | Shared direct race commands and one-container sequential local orchestration; `make verify` parity. |
| `qodana.yaml` | Exact exclusion for `scripts/testshards/main_test.go`. |
| `ARCHITECTURE.md` | Add the new repository tooling package and its responsibility. |
| `todos.md` | Move this item to Done only after measured acceptance. |
| This plan | Record both checkpoints, outcome, exceptions, and the final cost ledger; move it to `finished_refactorings/` only when complete. |

`.github/workflows/release.yml` is intentionally absent from the change set.

## Helper contract

The CLI stays small, deterministic, and standard-library-only. Exact flag
spelling may be adjusted during the red-green cycle, but these behaviors are
the contract:

```text
testshards summarize  -json RUN1 -json RUN2 -json RUN3
testshards plan       -package ./internal/ui -shards 3 \
                      -json RUN1 -json RUN2 -json RUN3
testshards check      -package ./internal/ui \
                      -manifest .github/testshards/internal-ui.tsv
testshards regex      -manifest .github/testshards/internal-ui.tsv \
                      -shard ui-1
testshards capture    -out RAW.json
```

- `summarize` reports package durations and terminal top-level test results for
  one file, and medians for multiple files. It rejects malformed/truncated JSON
  and reports failed or missing terminal events rather than inventing weights.
- `plan` requires every current inventory item to have a terminal result in
  every supplied baseline. It outputs all names exactly once, ordered by shard
  and then name, using the locked median/LPT/tie-break rules.
- `check` uses `go list -json` and `go test -list` in the current build context.
  The checked-in manifest is accepted only through the Linux/amd64 container
  target or on a Linux CI runner, never from a Darwin inventory. It compares
  exact sets, requires all three shards to be non-empty, and scans selected
  exact-package test files for `.Parallel()` calls.
- `regex` parses and structurally validates the entire manifest before emitting
  one Go-compatible, `regexp.QuoteMeta`-escaped expression of the form
  `^(TestA|TestB)$`. It never emits a match-all or empty expression.
- `capture` reads newline-delimited `go test -json` from stdin, preserves it in
  `RAW.json`, and prints compact package/test/failure lines. The shell pipeline
  uses `pipefail`, so either a failed `go test` or failed capture step fails the
  job. Raw artifacts remain available even on failure.

`TestMain` is package harness code, not an independently runnable test, and is
therefore not a manifest entry. A top-level parent owns all names below its
slash-separated subtest path.

## Acceptance criteria

| ID | Criterion | Verification |
|---|---|---|
| AC1 | Three instrumented monolithic attempts exist at one SHA, each with readable raw JSON and timing metadata. | For each N in 1-3, run `gh run view <baseline-run-id> --attempt N --json attempt,headSha,status,conclusion,createdAt,startedAt,updatedAt,jobs`; download and hash that attempt's artifacts before starting N+1. |
| AC2 | The checked-in manifest is deterministic and covers the exact Linux UI inventory once, with no stale entries and no empty shard. | `make check-test-shards` twice, plus `git diff --exit-code -- .github/testshards/internal-ui.tsv` after regenerating from the same inputs. |
| AC3 | Deliberately missing, duplicate, stale, malformed, unknown-shard, empty-shard, and `.Parallel()` fixtures are rejected with actionable names. | `go test ./scripts/testshards -run 'TestCheckRejects|TestParallelGuardRejects' -count=1 -v` |
| AC4 | Each generated expression is anchored, non-empty, and matches exactly its own manifest names. | `go test ./scripts/testshards -run 'TestRegex' -count=1 -v` |
| AC5 | The non-UI set is exactly all module packages except `github.com/frathe/picfetch/internal/ui`; UI subpackages, root, and scripts remain included. | `go test ./scripts/testshards -run 'TestPackagePartition' -count=1 -v`, then inspect the `non-ui` CI summary against `go list ./...`. |
| AC6 | Every Linux matrix entry uses the explicit locale, race detector, fresh execution, and 30-minute package timeout; all four pass. | Inspect the command lines in `gh run view <sharded-run-id> --log`; each must contain `LANG=en_US.UTF-8`, `-race`, `-count=1`, and `-timeout 30m`. |
| AC7 | A test or capture failure keeps the exact shard/test visible and fails the job while still uploading raw JSON. | `go test ./scripts/testshards -run 'TestCapture' -count=1 -v`, plus one temporary negative workflow/helper verification before restoring the tree. |
| AC8 | Formatting/TUF/vet/build/Windows cross-build are independent from Linux race tests, and Windows tests remain independent and unchanged. | `gh run view <sharded-run-id> --json jobs`; confirm six job executions overlap where runners permit and compare the Windows command with the base revision. |
| AC9 | A failure in any validation, matrix, or Windows job fails the reusable CI caller; no shard is optional. | Inspect the completed workflow graph and `.github/workflows/ci.yml`; there must be no `continue-on-error`, and matrix `fail-fast` must be `false`. |
| AC10 | Release gating is unchanged. | `git diff --exit-code <base-sha> -- .github/workflows/release.yml` |
| AC11 | Local verification runs the same package partition and shard expressions sequentially in one Linux/amd64 container. | `make verify` |
| AC12 | Three final attempts share one SHA and include gate, queue-excluded, validation/setup, shard, package/test, and runner-minute medians. No coverage or failure is lost. | For each N in 1-3, run `gh run view <sharded-run-id> --attempt N --json attempt,headSha,status,conclusion,createdAt,startedAt,updatedAt,jobs`; download and hash that attempt's artifacts before starting N+1. |
| AC13 | Median gate time is preferably 8-12 minutes; any accepted exception identifies the measured critical path. A >20% shard imbalance is addressed before acceptance. | Run `testshards summarize` over all three final-attempt artifacts and calculate the workflow/job medians from the Actions API record. |
| AC14 | The concurrency trade-off is explicit. | Record pre/post median runner minutes and peak six-job topology in the Outcome and `todos.md` Done entry. |

## Task graph

```text
T1 measurement-only workflow
  -> C1 user commit/push + three baseline attempts
    -> T2 baseline analysis
      -> T3 helper and manifest (test-first)
        -> T4 shared Makefile commands
          -> T5 final parallel workflow
            -> T6 local and negative safety verification
              -> C2 user commit/push + three sharded attempts
                -> T7 measured comparison/rebalance decision
                  -> T8 documentation and final handoff
```

C1 and C2 are real external checkpoints. Work must not skip them or substitute
different SHAs for convenience.

## Tasks

### T1 — Add the measurement-only checkpoint

- **Owner:** Inline implementation agent.
- **Files:** `.github/workflows/ci.yml` only.
- **Depends:** None.
- **Contract:** Keep the current job topology and commands, pin its one Linux
  job to `ubuntu-24.04`, add `-count=1 -json` to the monolithic Linux race
  command, preserve its exit status through a `pipefail` capture, upload the raw
  stream with `if: always()` and 14-day retention, and expose enough step timing
  to separate setup/validation from the race command. Include
  `github.run_attempt` in the artifact name. Do not introduce shards in this
  diff.
- **Test:** Review the workflow diff for unchanged validation, package scope,
  test flags other than the additive `-count=1 -json`, Windows job,
  concurrency, and `workflow_call` trigger. The explicit Linux runner pin is
  the only other execution change.
- **Verify:** `git diff --check`; `make fmt-check`; GitHub validates and runs
  the workflow at C1.
- **Budget:** One focused edit/review pass; no full local race rerun and no
  delegated work.

### C1 — Capture three fresh monolithic CI attempts

- **Owner:** User for commit/push; GitHub Actions for execution; implementation
  agent for read-only collection.
- **Files:** No further source edit until all attempts are available.
- **Depends:** T1.
- **Contract:** The user commits and pushes the measurement-only tree. Let the
  first PR workflow finish, rerun that same run twice, and retain attempts 1-3.
  [GitHub reruns retain the original SHA and ref](https://docs.github.com/en/actions/how-tos/manage-workflow-runs/re-run-workflows-and-jobs).
  Immediately after each attempt, use `gh run view --attempt N` and download
  its uniquely named artifact to an off-tree directory before requesting the
  next rerun; do not depend on an old attempt's artifacts remaining
  downloadable after a rerun. Record UTC just before each
  `gh run rerun <run-id>` request. For attempt 1, use the Actions `createdAt`;
  for attempts 2 and 3, use that recorded request time as the queue-inclusive
  origin because the run's original `createdAt` is unchanged.
  Record run ID, attempt, head SHA, runner image, Go version/cache outcome,
  creation/start/completion timestamps, every step duration, total runner
  minutes, package durations, top-level UI durations, failures, and artifact
  names. All attempts must use the same SHA and `-count=1`; do not substitute
  older `main` runs.
- **Test:** Each JSON stream parses to a terminal package result and complete UI
  inventory. Any infrastructure-failed attempt is reported and repeated rather
  than included silently.
- **Verify:** For N in 1-3, `gh run view <run-id> --attempt N --json
  attempt,headSha,status,conclusion,createdAt,startedAt,updatedAt,jobs`, followed
  immediately by `gh run download <run-id> -D <off-tree-attempt-N-dir>` and a
  SHA-256 inventory.
- **Budget:** Exactly three valid attempts; reruns, not empty commits.

### T2 — Freeze the measured baseline and proposed allocation

- **Owner:** Inline implementation agent.
- **Files:** This plan's evidence/outcome sections; raw data stays outside the
  repository.
- **Depends:** C1.
- **Contract:** Calculate per-attempt and median primary/secondary metrics.
  Explain queue time separately. Produce the three-shard LPT proposal from
  median top-level CI durations and verify it contains the Linux inventory
  exactly once. Record any test whose outcomes/durations were not comparable.
- **Test:** Independently total manifest membership and compare package sets;
  do not treat summed per-test elapsed values as workflow wall time.
- **Verify:** Re-run the summary/allocation command over the same inputs and
  require byte-identical normalized output.
- **Budget:** One analysis pass and one arithmetic review; no workflow changes.

### T3 — Build the helper and manifest test-first

- **Owner:** Inline implementation agent. The parser, inventory, and workflow
  contract are tightly coupled, so this is not a safe delegation seam.
- **Files:** `scripts/testshards/main.go`,
  `scripts/testshards/main_test.go`, `.github/testshards/internal-ui.tsv`,
  `qodana.yaml`, `ARCHITECTURE.md`.
- **Depends:** T2.
- **Contract:** Implement the helper contract above with pure internal
  functions around a thin CLI. Write failing tests first for JSON truncation,
  missing terminal results, deterministic median/LPT ties, every manifest
  rejection mode, exact package partition, Go-compatible quoting/anchoring,
  empty regex refusal, parallel-call detection, concise failures, and raw JSON
  preservation. Generate the initial manifest only from the three accepted C1
  files. Add the exact new `_test.go` path to Qodana exclusions and document
  the tooling package in the architecture map.
- **Execution note:** Run initial planning/validation in an explicit
  Linux/amd64 container even though the reusable Make target is added in T4;
  never generate the checked-in Linux manifest from the host's Darwin test
  inventory.
- **Test:** `go test ./scripts/testshards -count=1`; each negative fixture must
  first demonstrate the intended red result before implementation makes the
  suite green.
- **Verify:** `go test ./scripts/testshards -count=1 -v`;
  `make check-qodana-test-exclusions`; deterministic regenerate-and-diff over
  the accepted C1 Linux inputs; `git diff --check`. The canonical live Linux
  inventory check begins in T4 after its Docker target exists.
- **Budget:** One red-green-refactor cycle plus one review pass; no full suite.

### T4 — Give Makefile and CI one race-test contract

- **Owner:** Inline implementation agent.
- **Files:** `Makefile`.
- **Depends:** T3.
- **Contract:** Keep `make test` as the complete non-race Linux/amd64 Docker
  suite. Make `make test-race` and `make verify` validate the manifest and run
  the four final commands sequentially inside one Docker container so package
  compilation/cache and apt setup are reused. Provide direct in-container
  targets used by Actions for the guard, `non-ui`, and one selected UI shard.
  Public `make check-test-shards` must enter Linux/amd64 Docker; the direct
  target is explicitly internal/CI-only so a macOS inventory cannot be mistaken
  for the canonical one. Derive the non-UI list from `go list ./...` minus only
  `go list ./internal/ui`; reject an empty set. Centralize
  `-race -count=1 -timeout 30m` and the explicit locale so local and CI
  invocations cannot drift.
- **Test:** Exercise helper unit tests and dry-inspect all four expanded command
  lines before the expensive run. Confirm the union is all packages and the
  intersection between exact UI and non-UI is empty.
- **Verify:** `make check-test-shards`; focused direct command checks inside the
  Linux container; the single final `make verify` is reserved for T6.
- **Budget:** One Makefile refactor pass; at most one focused Docker setup.

### T5 — Split the reusable CI workflow

- **Owner:** Inline implementation agent.
- **Files:** `.github/workflows/ci.yml`.
- **Depends:** T4.
- **Contract:** Replace `build-and-test` with:

  1. `validation`, pinned to `ubuntu-24.04`, containing checkout/setup,
     formatting, TUF-root validation, Linux GUI dependencies, manifest guard,
     vet, normal build, both Windows cross-builds, and Windows vet;
  2. `linux-race`, pinned to `ubuntu-24.04`, with a four-value matrix and
     `fail-fast: false`; every entry performs its own checkout/setup/dependency
     installation and invokes the matching shared Make target; and
  3. the existing `windows-test`, still independent and command-identical.

  No job has `needs` on another. Retain `workflow_call`, read-only permissions,
  the current cancellation group/behavior, and required failure semantics.
  Each race entry captures concise output plus a raw JSON artifact on success
  or failure. Artifact and visible job names include `non-ui` or the exact UI
  shard, plus the workflow attempt where appropriate. Do not edit
  `release.yml`.
- **Test:** Review all old steps against the new job map one by one. Search for
  forbidden `continue-on-error`, unintended `needs`, moving Linux labels, or a
  matrix-level fail-fast default.
- **Verify:** `git diff --check`; `git diff --exit-code <base-sha> --
  .github/workflows/release.yml`; actual GitHub validation at C2.
- **Budget:** One workflow edit and one independent contract review; no local
  duplicate full-suite run.

### T6 — Verify safety locally and with deliberate negative inputs

- **Owner:** Inline implementation agent.
- **Files:** Only fixes revealed by verification.
- **Depends:** T5.
- **Contract:** Prove each guard fails for a temporary missing entry, duplicate
  entry, stale entry, malformed row, unknown shard, empty shard, and UI
  `.Parallel()` call, then restore the valid inputs. Prove capture reports an
  exact failed test while retaining its raw event stream and nonzero pipeline
  status. Run the complete sequential Linux race suite once through the final
  Makefile path. Do not regenerate golden files.
- **Test:** Focused helper negative tests, followed by the full repository gate.
- **Verify:** `go test ./scripts/testshards -count=1 -v`;
  `make check-test-shards`; `make verify`; `git diff --check`;
  `git status --short` with no failed golden renders or timing artifacts.
- **Budget:** One full local `make verify`; one fix-and-rerun is allowed only if
  the first run finds a real defect.

### C2 — Capture three fresh sharded CI attempts

- **Owner:** User for commit/push; GitHub Actions for execution; implementation
  agent for read-only collection.
- **Files:** No source edit until the three attempts are classified.
- **Depends:** T6.
- **Contract:** The user commits and pushes the sharded tree. Run the initial
  attempt and rerun the same workflow twice. Collect the same fields as C1,
  plus each shard's test count, median/maximum test duration, compile/setup
  overhead, and raw artifact. Confirm all six job executions are mandatory and
  that all four Linux package/test sets passed. As at C1, download and hash each
  attempt before triggering the next rerun, and record the rerun request time
  used by the queue-inclusive metric.
- **Test:** All attempts share one SHA and produce a complete exact-set union.
  Infrastructure failures are labeled and replaced, not averaged into product
  performance.
- **Verify:** For N in 1-3, `gh run view <run-id> --attempt N --json
  attempt,headSha,status,conclusion,createdAt,startedAt,updatedAt,jobs`, followed
  immediately by `gh run download <run-id> -D <off-tree-attempt-N-dir>` and a
  SHA-256 inventory.
- **Budget:** Exactly three valid attempts at the accepted sharded SHA.

### T7 — Compare, rebalance only if justified, and apply the stop rule

- **Owner:** Inline implementation agent.
- **Files:** Manifest and this plan only if measurement justifies a rebalance;
  workflow/Makefile only for a demonstrated defect.
- **Depends:** C2.
- **Contract:** Compare medians for gate wall time, queue-excluded window,
  critical job, validation/setup, race execution, UI shards, non-UI, and total
  runner minutes. Confirm zero missing/duplicate package or test names.

  - If slowest/fastest UI shard median is at most 1.20, do not churn the
    manifest merely to make totals prettier.
  - If it exceeds 1.20 and that imbalance affects the critical path, recompute
    within the same three shards from C2 data, rerun T6, create a new user
    checkpoint, and collect three new attempts before comparison.
  - Allow at most one measured rebalancing round in this pull request.
  - If safe, balanced CI still has a median above 12 minutes, stop and present
    the critical path to the user. Do not add a fourth shard or weaken tests.
  - If total runner minutes exceed 2x baseline, flag and explain the source;
    this is a review signal, not a reason to remove coverage.

- **Test:** Recompute summaries from raw data; reconcile Actions job timing
  with JSON package/test timing without adding overlapping durations.
- **Verify:** Deterministic helper summary plus a checked arithmetic table in
  this plan.
- **Budget:** One comparison pass; at most one rebalance-and-three-run loop.

### T8 — Close documentation and hand off

- **Owner:** Inline implementation agent.
- **Files:** This plan, `todos.md`; move the plan to `finished_refactorings/`
  only after every accepted criterion passes.
- **Depends:** T7 and any required user decision for an over-target result.
- **Contract:** Record run IDs, SHAs, attempt table, before/after medians,
  critical path, runner minutes, shard membership counts, safety verification,
  and any accepted target exception. Move the TODO to Done with the explicit
  trade-off that lower elapsed time uses more concurrent capacity and may use
  more total runner minutes. Report the honest whole-process limit. Leave no
  raw timing artifacts in the repository.
- **Test:** Read the finished record against AC1-AC14 and inspect the final
  source diff for unrelated changes.
- **Verify:** `make verify`; `git diff --check`; `git status --short`; final
  successful GitHub run links.
- **Budget:** Documentation-only pass after CI acceptance; do not repeat the
  full suite unless code changed after T6.

## Delegation and verification budget

Reconnaissance used three bounded, read-only scouts for current workflow,
measurement, and UI-test inventory facts. Implementation stays inline because
the helper, manifest, Makefile commands, and workflow form one safety contract
and touch shared files. No implementation subagents are budgeted.

- Maximum implementation review loops: two (contract review, then final diff
  review).
- Expensive local full suites: one at T6, plus at most one defect-driven rerun.
- CI executions: three valid C1 attempts and three valid C2 attempts, plus at
  most three after one justified rebalancing checkpoint.
- Golden regeneration: zero.
- Workflow/release mutations outside the listed files: zero.

## Outcome

C1 and T2 are complete. The measurement-only workflow was committed as
`fcd11103969ac8d1b5a1a220270c0730cb5e1913` and exercised in three successful
attempts of [Actions run 33734286005](https://github.com/frathe/picfetch/actions/runs/33734286005)
for [draft PR 14](https://github.com/frathe/picfetch/pull/14). All attempts used
the same pull-request ref, Ubuntu 24.04, Go 1.27.1, a primary setup-cache hit,
and the unchanged package/test commands. There were no infrastructure or test
failures to discard.

### Controlled monolithic baseline

The primary gate origin is the workflow creation time for attempt 1 and the
recorded UTC rerun-request time for attempts 2 and 3. Completion is the later
of the two required workflow jobs. `Qodana for Go` is a separately injected,
zero-second check rather than a job declared by `ci.yml`; it is excluded from
the workflow execution and runner-minute totals.

| Attempt | Origin (UTC) | Queue | Gate | Queue-excluded | Linux job | Windows job | Runner minutes |
|---:|---|---:|---:|---:|---:|---:|---:|
| 1 | 08:36:41 (created) | 0:02 | 23:12 | 23:10 | 23:10 | 1:13 | 24:23 |
| 2 | 09:01:13 (rerun request) | 0:07 | 22:17 | 22:10 | 22:10 | 1:16 | 23:26 |
| 3 | 09:24:20 (rerun request) | 0:05 | 22:07 | 22:02 | 22:02 | 1:20 | 23:22 |
| Median | - | **0:05** | **22:17** | **22:10** | **22:10** | **1:16** | **23:26** |

The required Linux job's step timings were:

| Step | Attempt 1 | Attempt 2 | Attempt 3 | Median |
|---|---:|---:|---:|---:|
| Set up job | 0:01 | 0:01 | 0:01 | 0:01 |
| Checkout | 0:01 | 0:03 | 0:03 | 0:03 |
| Set up Go | 0:12 | 0:17 | 0:16 | 0:16 |
| Formatting | 0:05 | 0:04 | 0:06 | 0:05 |
| TUF-root validation | 0:01 | 0:01 | 0:01 | 0:01 |
| Linux GUI dependencies | 0:13 | 0:24 | 0:21 | 0:21 |
| Vet | 0:10 | 0:10 | 0:11 | 0:10 |
| Normal build | 0:04 | 0:04 | 0:04 | 0:04 |
| Windows cross-builds | 1:10 | 1:15 | 1:13 | 1:13 |
| Windows vet | 0:09 | 0:10 | 0:10 | 0:10 |
| Linux race test | 21:01 | 19:35 | 19:26 | **19:35** |
| Raw artifact upload | 0:01 | 0:02 | 0:06 | 0:02 |

From Linux-job start to the race step was 2:07, 2:30, and 2:27, for a
**2:27 median**. This is the serial validation/build work that the final
topology can overlap with testing.

The machine-readable race streams contained the same 44 package terminal
events and the same 567 exact-`internal/ui` top-level tests in every attempt.
All 567 tests passed in every attempt with no top-level skips. The only package
skip was the consistent no-test-files `internal/ui/assets` package.

| Attempt | Compile/start gap | JSON event span | Non-UI tail from first event | Exact `internal/ui` | Sum of UI top-level elapsed values |
|---:|---:|---:|---:|---:|---:|
| 1 | 3:15.022 | 17:45.933 | 1:00.635 | 17:41.764 | 17:40.420 |
| 2 | 3:17.373 | 16:18.196 | 1:02.854 | 16:14.021 | 16:12.450 |
| 3 | 3:10.968 | 16:15.323 | 1:01.564 | 16:11.149 | 16:09.690 |
| Median | **3:15.022** | **16:18.196** | **1:01.564** | **16:14.021** | **16:12.450** |

The non-UI tail is measured from the first test event, after the shared compile
gap. Measured from race-step start, the last non-UI package completed at a
4:15.657 median. Package elapsed values overlap and are not summed into a
wall-clock figure. The slowest package medians were:

| Package | Attempt 1 | Attempt 2 | Attempt 3 | Median |
|---|---:|---:|---:|---:|
| `internal/ui` | 1061.764s | 974.021s | 971.149s | **974.021s** |
| `internal/ui/compare` | 55.462s | 57.668s | 56.376s | 56.376s |
| `internal/imaging` | 36.054s | 37.578s | 36.704s | 36.704s |
| `internal/ui/help` | 14.419s | 14.876s | 15.397s | 14.876s |
| `internal/ui/favorites` | 9.447s | 10.644s | 9.867s | 9.867s |
| `internal/ui/exifwin` | 6.780s | 7.058s | 6.616s | 6.780s |
| `internal/ui/copyselection` | 4.382s | 5.201s | 4.977s | 4.977s |
| `internal/ui/settingswin` | 4.208s | 4.226s | 4.057s | 4.208s |
| `internal/update` | 3.815s | 3.856s | 3.876s | 3.856s |
| `internal/ui/grid` | 3.483s | 3.674s | 3.622s | 3.622s |

Every other package had a median at or below 2.149s. The complete 44-package
table and all 567 per-attempt top-level test durations are retained in the
off-tree normalized record described below. The slowest UI-test medians were:

| Test | Median |
|---|---:|
| `TestCompareCommandEntryPoints_MenuAndFeatureCallbacksAreIgnored` | 40.31s |
| `TestCopySelectionCancelsBeforeOtherCommands` | 35.99s |
| `TestCopySelectionAvailability` | 18.83s |
| `TestCompareOpenRefusal_DropDialogShortcutAndOpenWithAreDiscarded` | 13.19s |
| `TestCompareHelp_F1OpensManualWithoutLeavingComparison` | 10.05s |
| `TestCompareLinkToggle_RequiresExactPhysicalControlL` | 9.33s |
| `TestYieldingMenuCallbacksWrapsEveryField` | 8.82s |
| `TestBuildMainMenu_ManualOpenedObserverSyncsWindowHelp` | 8.43s |
| `TestAdvance_ShuffleOnNeverRepeatsCurrentIndex` | 7.46s |
| `TestCopySelectionKeyboard` | 6.02s |

### Baseline artifacts and reproducibility

Raw streams were downloaded and hashed outside the repository before each
subsequent rerun:

| Attempt | Artifact | Lines | Bytes | SHA-256 |
|---:|---|---:|---:|---|
| 1 | `linux-race-baseline-33734286005-attempt-1` | 11,402 | 2,481,918 | `c31eea242ae6a6a63cd8a64e93a04416939a55a6b63083b1fdc798503803b07e` |
| 2 | `linux-race-baseline-33734286005-attempt-2` | 11,402 | 2,481,964 | `852ba59fe2cef71ae76d7cd17d144ff42c83e709d36f8f123038297e436beb61` |
| 3 | `linux-race-baseline-33734286005-attempt-3` | 11,402 | 2,482,006 | `5e1da150e8524848b5b682e16f57199875e095ecb234d59c3492ca7665f8671d` |

The off-tree directory is
`/private/tmp/picfetch-ci-baseline.XeJlI7`. It contains the raw streams,
required-job logs, Actions metadata, analysis scripts, and two independently
generated copies of each normalized report. The 1,191-line test/package/LPT
report reproduced byte-for-byte at SHA-256
`bb61c7b69d41e98b22ef15575a7614b13363b14d029a4cd9b96be6d236454803`;
the Actions timing report reproduced at
`39f50e66026a211121e7dc3fc15dd3b4b1e69a51542dc50434a9db27b4723522`.
The 18-file checksum inventory itself hashes to
`6aefc52fcdfcacff0ae5c8fb96ea767d4d9086fb209639f0be869c85af4d4b3d`.
No raw timing or log file is in the repository.

The raw artifacts were protected at each rerun boundary as planned. Full
required-job log files were materialized and hashed after attempt 3 rather
than before attempts 2 and 3 were requested. GitHub's attempt-addressable logs
remained intact, and their timing metadata had already been inspected, so no
evidence was lost. This is a documented collection-order deviation from
ticket 01, not a substituted or incomparable sample.

### Proposed three-shard allocation

Deterministic longest-processing-time allocation using each test's median over
the three CI streams produced this exact-set proposal:

| Shard | Tests | Sum of median weights |
|---|---:|---:|
| `ui-1` | 177 | 326.75s |
| `ui-2` | 178 | 327.07s |
| `ui-3` | 212 | 326.74s |

The 567-name union is exact and contains no duplicate assignment or empty
shard. Repeated Linux/amd64 generation produced the checked-in manifest
byte-for-byte at SHA-256
`6468a50ce0925ade39d253be7e7b0e728aa3d38fdc9f32a6b5547347155c5a18`.
These are theoretical test-only weights, not predicted job wall times;
each real runner will repeat checkout, toolchain/dependency setup, package
compilation, and package harness work. The measured critical path nevertheless
supports the planned boundary: exact `internal/ui` dominates the race command,
the non-UI set finishes early, and the serial validation/build phase can run
independently. Ticket 03 now supplies the live Linux inventory guard and exact
filter generation around this assignment.

### Live assignment safety

Ticket 03 is complete. The helper now checks the entire manifest against test
files and top-level `Test`, `Fuzz`, and `Example` names selected by the current
Go build, ignores `TestMain` and subtest paths, rejects `.Parallel()` calls in
selected exact-package test files, and emits escaped exact-match filters only
after full structural validation. A public `make check-test-shards` runs that
acceptance boundary in Linux/amd64 Docker; the direct target refuses Darwin or
a mismatched cross-build context as canonical.

Two independent Docker checks each reported 567 runnable names across three
non-empty shards without modifying `.github/testshards/internal-ui.tsv`; its
SHA-256 remains
`6468a50ce0925ade39d253be7e7b0e728aa3d38fdc9f32a6b5547347155c5a18`.
Focused command tests cover every required rejection and all three generated
filters.

### Shared local race contract

Ticket 04 is complete. `make test` remains the complete unsharded non-race
Linux/amd64 suite. `make test-race` and `make verify` enter one Linux/amd64
container, run the canonical manifest guard, then execute the exact non-UI
package complement and `ui-1` through `ui-3` sequentially. The prepared-runner
targets expose the same guard, non-UI, and selected-UI commands for the final CI
matrix without nesting Docker. Each race command visibly uses
`LANG=en_US.UTF-8`, `-race`, `-count=1`, `-timeout 30m`, JSON capture, and Bash
`pipefail`.

The helper's `partition` path resolved the repository to 45 packages: 44
non-UI packages plus exact `github.com/frathe/picfetch/internal/ui`. The
complement retains the root, every tooling package, and every UI feature
subpackage, and fails closed if the selected package is absent or leaves an
empty complement. Its `capture` path preserved an existing 2,481,918-byte CI
stream byte-for-byte while reducing live output to partitioned package/test
terminals and failure details.

A deliberate failing test stream returned pipeline status 1 and named the
partition, package, and exact test; a deliberate capture-open failure returned
capture status 1. The final post-review `make verify` run checked all 567 UI
runnables, passed the non-UI complement, then passed `ui-1` in 229.604s,
`ui-2` in 228.490s, and `ui-3` in 211.661s. No golden was regenerated and no
raw event stream was left in the repository. The initial full run also passed;
the plan's one allowed defect-driven rerun covered a review fix that made the
direct command lines visible for later CI log inspection.

### Parallel reusable CI topology

Ticket 05 is complete. The reusable workflow now exposes one Ubuntu 24.04
validation job, one fail-fast-disabled Ubuntu 24.04 race matrix with exact
`non-ui`, `ui-1`, `ui-2`, and `ui-3` entries, and the existing independent
native Windows job. None has a `needs` edge. Validation retains formatting,
TUF-root checking, Linux GUI dependencies, live manifest validation, vet,
normal build, both Windows cross-builds, and Windows vet. Each race entry owns
checkout, Go setup, Linux GUI dependencies, the matching direct Make contract,
an attempt-qualified summary, and an always-run raw artifact upload that fails
closed when the stream is absent and retains it for 14 days.

A parsed workflow contract confirmed the exact topology, names, matrix,
failure policy, triggers, permissions, cancellation group, artifact settings,
and unchanged Windows command. A separate parsed comparison against the
pre-ticket `dd43c77` revision showed that every pre-existing validation/build
step and the native Windows job body survived unchanged; `release.yml` has no
source diff from `main`.
`make fmt-check`, `go vet ./...`, `go test ./scripts/testshards -count=1`, all
four direct-target dry runs, and `git diff --check` pass. The full local race
evidence remains ticket 04's final `make verify`, since this ticket changes
only Actions YAML. The C2 evidence below now confirms the hosted six-slot
execution, required results, and aggregate workflow success.

### Sharded checkpoint validity

The user committed and pushed the final topology as
`b91d56ce67b991b595484a50d9b65c52f1a09dba`. Three successful attempts of
[Actions run 33758913251](https://github.com/frathe/picfetch/actions/runs/33758913251)
on `feature/CI-test-performance-improvements` used that exact SHA and ref. The
initial workflow completed before attempt 2 was requested, and attempt 2's raw
artifacts, required-job logs, metadata, and checksums were secured before
attempt 3 was requested. No test or infrastructure failure required a
replacement attempt.

The queue-inclusive origin is the workflow creation time for attempt 1 and the
UTC timestamp recorded immediately before each rerun request for attempts 2
and 3. Completion is the last required job completion. The separately injected
neutral `Qodana for Go` check declares no workflow job or runner time and is
excluded exactly as it was from the baseline.

| Attempt | Origin (UTC) | First required start | Required completion | Queue | Gate | Queue-excluded | Critical entry | Runner minutes | Result |
|---:|---|---|---|---:|---:|---:|---|---:|---|
| 1 | 13:04:11 (created) | 13:04:13 | 13:10:26 | 0:02 | 6:15 | 6:13 | `ui-3` | 24:50 | Pass |
| 2 | 13:13:19 (rerun request) | 13:13:35 | 13:21:11 | 0:16 | 7:52 | 7:36 | `ui-1` | 31:01 | Pass |
| 3 | 13:22:53 (rerun request) | 13:23:16 | 13:30:52 | 0:23 | 7:59 | 7:36 | `ui-3` | 30:43 | Pass |
| Median | - | - | - | **0:16** | **7:52** | **7:36** | UI race | **30:43** | **3/3 pass** |

All six required job intervals overlapped in every attempt: validation,
Windows, `non-ui`, and all three UI shards. This observed peak of six proves
that the workflow has no scheduling dependency between them when hosted-runner
capacity is available.

### Required jobs, validation, and setup

| Required job | Attempt 1 | Attempt 2 | Attempt 3 | Median |
|---|---:|---:|---:|---:|
| Validation | 2:32 | 2:42 | 2:24 | **2:32** |
| Windows tests | 1:12 | 1:36 | 1:14 | **1:14** |
| Linux race `non-ui` | 4:22 | 5:03 | 5:34 | **5:03** |
| Linux race `ui-1` | 4:29 | 7:30 | 7:07 | **7:07** |
| Linux race `ui-2` | 6:03 | 7:02 | 6:48 | **6:48** |
| Linux race `ui-3` | 6:12 | 7:08 | 7:36 | **7:08** |

The independent validation job retained every pre-sharding check. Its
step-level medians were:

| Validation step | Median |
|---|---:|
| Job setup | 0:01 |
| Checkout | 0:02 |
| Go setup | 0:13 |
| Formatting | 0:07 |
| TUF-root validation | 0:01 |
| Linux GUI dependencies | 0:19 |
| UI manifest validation | 0:11 |
| Vet | 0:10 |
| Normal build | 0:03 |
| Windows cross-builds | 1:13 |
| Windows vet | 0:10 |

The Windows test step itself had a 0:06 median. All three attempts ran the
unchanged `go test ./internal/update/... ./internal/ui/autoupdate/...` command
successfully.

### Linux race partitions, coverage, and balance

The runner-setup column spans job start through the race-contract step. The
compile/helper gap is inside that contract, from its start to the first Go JSON
event. `non-ui` has many concurrently executed packages, so its final column is
the first-to-last JSON span; each UI value is the exact package terminal.

| Entry | Inventory per attempt | Job median | Runner setup | Race contract | Compile/helper gap | JSON/package execution | Result |
|---|---|---:|---:|---:|---:|---:|---|
| `non-ui` | 44 packages; 1,486 top-level tests | 5:03 | 1:21 | 3:33 | 2:50.331 | 0:42.891 span | Pass |
| `ui-1` | 177 tests | 7:07 | 0:38 | 6:24 | 0:58.913 | 5:21.919 | Pass |
| `ui-2` | 178 tests | 6:48 | 0:36 | 5:57 | 0:54.565 | 5:02.435 | Pass |
| `ui-3` | 212 tests | 7:08 | 0:48 | 6:15 | 0:57.727 | 5:10.880 | Pass |

The UI top-level elapsed-sum medians were 5:20.640, 5:01.300, and 5:09.550.
The slowest/fastest exact-package median ratio is
`321.919 / 302.435 = 1.064`; the full-job ratio is
`428 / 408 = 1.049`. Both are safely below the 1.20 review threshold, so the
manifest receives no cosmetic rebalance. The apparently fast `ui-1` first
attempt is ordinary runner variance and does not override the three-run
median.

Each attempt's `non-ui` package terminals equal the current 44-package module
complement after subtracting only exact `internal/ui`. The three UI result sets
equal the manifest's 567-name union, with exactly 177, 178, and 212 names and
no missing, stale, or repeated test. All 567 UI tests passed in every attempt.
The non-UI stream consistently had 1,485 passing top-level tests and one
expected zero-second skip,
`TestCheckRejectsDarwinDerivedInventoryAsCanonical`; the only skipped package
was the no-test-files `internal/ui/assets`. Thus the final union is all 45
current module packages plus every exact UI top-level runnable.

All 12 Linux job logs show the literal `LANG=en_US.UTF-8`, `-race`,
`-count=1`, and `-timeout 30m` command. Each job uploaded its required raw
stream after the test step. The workflow contains no `needs` or
`continue-on-error`, keeps matrix `fail-fast: false`, and the overall run was
successful only after all six required results succeeded. Earlier deliberate
helper and pipeline failures returned nonzero while retaining and naming the
raw failing test. `release.yml` remains unchanged and still makes every
release job depend on the reusable CI caller.

### Final package and test durations

The slowest non-UI package terminal medians were:

| Package | Attempt 1 | Attempt 2 | Attempt 3 | Median |
|---|---:|---:|---:|---:|
| `internal/ui/compare` | 37.588s | 37.812s | 48.611s | **37.812s** |
| `internal/imaging` | 27.352s | 27.840s | 34.887s | 27.840s |
| `internal/ui/help` | 11.137s | 11.357s | 15.939s | 11.357s |
| `internal/ui/favorites` | 8.891s | 10.199s | 11.469s | 10.199s |
| `internal/ui/exifwin` | 5.907s | 5.249s | 7.335s | 5.907s |
| `internal/ui/copyselection` | 3.383s | 4.125s | 5.245s | 4.125s |
| `internal/ui/grid` | 3.170s | 3.985s | 3.928s | 3.928s |
| `internal/update` | 3.827s | 3.696s | 3.733s | 3.733s |
| `internal/ui/settingswin` | 3.472s | 3.402s | 4.669s | 3.472s |
| `scripts/testshards` | 2.820s | 2.859s | 3.314s | 2.859s |

Every other non-UI package had a median at or below 2.006s. The slowest exact
UI top-level test medians were:

| Test | Shard | Attempt 1 | Attempt 2 | Attempt 3 | Median |
|---|---|---:|---:|---:|---:|
| `TestCompareCommandEntryPoints_MenuAndFeatureCallbacksAreIgnored` | `ui-1` | 23.80s | 44.04s | 39.86s | **39.86s** |
| `TestCopySelectionCancelsBeforeOtherCommands` | `ui-2` | 30.69s | 35.91s | 33.48s | 33.48s |
| `TestCompareOpenRefusal_DropDialogShortcutAndOpenWithAreDiscarded` | `ui-3` | 11.63s | 13.04s | 14.60s | 13.04s |
| `TestCopySelectionAvailability` | `ui-3` | 10.00s | 11.31s | 12.57s | 11.31s |
| `TestCompareHelp_F1OpensManualWithoutLeavingComparison` | `ui-3` | 8.68s | 9.89s | 10.83s | 9.89s |
| `TestYieldingMenuCallbacksWrapsEveryField` | `ui-1` | 5.26s | 9.70s | 8.79s | 8.79s |
| `TestCompareLinkToggle_RequiresExactPhysicalControlL` | `ui-2` | 8.03s | 9.25s | 8.76s | 8.76s |
| `TestBuildMainMenu_ManualOpenedObserverSyncsWindowHelp` | `ui-3` | 7.09s | 8.12s | 8.88s | 8.12s |
| `TestAdvance_ShuffleOnNeverRepeatsCurrentIndex` | `ui-2` | 6.42s | 7.37s | 7.04s | 7.04s |
| `TestCopySelectionKeyboard` | `ui-1` | 3.60s | 6.61s | 6.08s | 6.08s |

The complete 44-package, 1,486 non-UI top-level-test, and 567 UI-test tables
are retained in the reproducible off-tree summaries below; package elapsed
values overlap and are never summed into wall-clock figures.

### Final assessment

| Metric | Monolithic median | Sharded median | Change |
|---|---:|---:|---:|
| Queue-inclusive required gate | 22:17 | **7:52** | -14:25 (-64.7%) |
| Queue-excluded execution | 22:10 | **7:36** | -14:34 (-65.7%) |
| Initial queue | 0:05 | 0:16 | +0:11 |
| Validation/build preparation | 2:27 serial before race | 2:32 independent job | +0:05, now overlapped |
| Linux race command / slowest entry | 19:35 | **6:24** | -13:11 (-67.3%) |
| Exact UI package / slowest shard | 16:14.021 | **5:21.919** | -10:52.102 (-67.0%) |
| Required runner minutes | 23:26 | **30:43** | +7:17 (+31.1%) |
| Peak required job slots | 2 | **6** | +4 |

The 7:52 gate slightly surpasses the 8-12 minute aim. That range was never a
correctness threshold: complete race coverage, native Windows tests,
validation, raw failure evidence, and release gating were retained first. The
critical path is now the UI race group; the per-attempt last entries were
`ui-3`, `ui-1`, and `ui-3`, and the slowest median UI job is `ui-3` at 7:08.

The 1.064 package-work ratio does not permit a rebalance, so there is no second
checkpoint and no fourth shard. The result is below 12 minutes, so the
over-target stop rule does not trigger. Median required runner time is 1.31x
baseline and remains below the two-times threshold of 46:52; nevertheless, the
accepted trade-off is explicit: about 65% less elapsed feedback uses four more
concurrent runner slots and about 31% more summed runner time. No target or
cost exception was accepted.

The honest whole-process limit remains: no required process executes all 567
exact UI tests together. The manifest proves exact coverage and non-overlap,
but cannot prove the absence of dependencies on package-wide test order or
state left by a test assigned to another shard. Separate hosted runners,
existing harness cleanup, and the guard against unreviewed `.Parallel()` calls
are the accepted safety boundary.

### Sharded artifacts and reproducibility

Every raw stream was downloaded and hashed outside the repository before the
next rerun request:

| Attempt | Entry | Lines | Bytes | SHA-256 |
|---:|---|---:|---:|---|
| 1 | `non-ui` | 8,653 | 1,870,485 | `8b299c13be26f4e602409c6e72a41c4b2306c38b1480900f1e4fbd143ea7545d` |
| 1 | `ui-1` | 868 | 192,319 | `a9c53dc5b6ebbee68b823707696c9804c5d5e89118b5261679cec15b0c61e61c` |
| 1 | `ui-2` | 905 | 199,951 | `5be8edd5f1d4fcbce34080c0cae7263a4e8b3dc53f2035ccf9aa5a082c030e4c` |
| 1 | `ui-3` | 1,177 | 264,482 | `13f0802971148a8f4aeedbe1d7e617c655352ce0de7a9d70e7dcd577601573ca` |
| 2 | `non-ui` | 8,653 | 1,870,592 | `7001bd134f748e60df6faa9af420f1fd384630d60c1c1ec8d3d75089856126f2` |
| 2 | `ui-1` | 868 | 192,289 | `74cb55a3752a0f9fd8330d801aaf70e625051822d896c32699cd8a4316858dae` |
| 2 | `ui-2` | 905 | 199,931 | `eb6e0e9187e48849f43d87ba098d77d41fe5341f87089b3873399dcf5a0d09bd` |
| 2 | `ui-3` | 1,177 | 264,450 | `188d53f18c32cac9202be9c0872a861964b87387e81237e189d14e3f59b9a39e` |
| 3 | `non-ui` | 8,653 | 1,870,675 | `54dbc33d0689d714172fcd0d8ae7c284e6fe00c101a6e66f8e21d0983de373d3` |
| 3 | `ui-1` | 868 | 192,335 | `2f45b71e98203dbcd8d70f21ba66db3b17027990bb9292a63f6014145af61e33` |
| 3 | `ui-2` | 905 | 199,946 | `2af5a53867f75cdae217431db576d97cf58e76ab05aaa07ac4b2dfb82938dc48` |
| 3 | `ui-3` | 1,177 | 264,459 | `616b18dcf62f6281173c80d78b6720079a1fe28e5d4a6d1c56e23915f41c6f5b` |

The off-tree record is `/private/tmp/picfetch-ci-sharded.EK5bnR`. It contains
each attempt's API metadata, artifact listing, four raw streams, separate logs
for all six required jobs, per-file checksums, the recorded origins, and the
analysis scripts. GitHub's aggregate-log command encountered the injected
Qodana check's absent log on attempt 1, so all required logs were fetched by
job ID instead; the partial aggregate output is retained and hashed rather than
mistaken for a complete log.

Two independent raw-event analyses reproduced byte-for-byte at SHA-256
`d791bfecdd6bfc6aee10be8ced9bd702af178d7925364354bd89704b02aac159`;
the two Actions timing analyses reproduced at
`d80437d9c4667d043bd8752e5fb36a674b2bd356eca60ebd1296eccacc350265`.
Independent three-run helper summaries reproduced at
`75792ee459e8c2a0b38446a0ce9f73038e94894d0742ebe44d7fa5b622f00e20`
for `non-ui`, `7a4c5bd62efd4abff3e5afb6ad5df153bf20552d867c273bea9c15e0fbbf2f9a`
for `ui-1`, `43726218909254d4757b487f236babc04b898138f3f45d39ab5249bef28a6114`
for `ui-2`, and
`b1605d31f39d81c93dab680f8e6eef1a2d73fdd4ff1c70738335ef4fbe81da53`
for `ui-3`. The verified 65-file checksum inventory hashes to
`290ceefb16f4f3bd94bb00c83d79f5e580c894992665741dae7743390c6a6072`.
No raw CI, log, or local timing artifact is tracked by Git.

### Final verification

After the assessment and documentation-only edits, `make fmt-check`,
`go vet ./...`, `go build ./...`, and
`go test ./scripts/testshards -count=1` passed. The canonical
`make check-test-shards` Linux/amd64 run reported 567 runnables across three
shards. Static checks reconfirmed the unchanged Windows test command, no
`needs` or `continue-on-error` in `ci.yml`, exactly one `fail-fast: false`, and
no source diff in `release.yml` from the branch base. All 12 downloaded race
logs contained the required locale, race, fresh-count, and timeout flags, and
the off-tree checksum inventory verified in full.

The expensive `make verify` race suite was not repeated after these
documentation-only edits, per T8's budget. The identical code and workflow
checkpoint had already passed T6's final local `make verify`, then all four
hosted race entries in each of the three C2 attempts. Final `git diff --check`
passed, the open TODO moved to Done, this plan moved to
`finished_refactorings`, and status showed only those intended documentation
changes and the plan move.

### Measurement record

| Checkpoint | SHA | Run ID | Attempts | Gate median | Queue-excluded median | Runner-minute median |
|---|---|---|---:|---:|---:|---:|
| Monolithic C1 | `fcd11103969ac8d1b5a1a220270c0730cb5e1913` | [33734286005](https://github.com/frathe/picfetch/actions/runs/33734286005) | 3/3 | 22:17 | 22:10 | 23:26 |
| Sharded C2 | `b91d56ce67b991b595484a50d9b65c52f1a09dba` | [33758913251](https://github.com/frathe/picfetch/actions/runs/33758913251) | 3/3 | **7:52** | **7:36** | **30:43** |

### Shard record

| Entry | Tests/packages | Median race contract | Median setup | Result |
|---|---:|---:|---:|---|
| `non-ui` | 44 packages / 1,486 tests | 3:33 | 1:21 | 3/3 pass |
| `ui-1` | 177 tests | 6:24 | 0:38 | 3/3 pass |
| `ui-2` | 178 tests | 5:57 | 0:36 | 3/3 pass |
| `ui-3` | 212 tests | 6:15 | 0:48 | 3/3 pass |

## Cost ledger

| Item | Planned | Actual |
|---|---:|---:|
| Read-only reconnaissance scouts | 3 | 3 |
| Implementation subagents | 0 | 0 |
| Local full race/verify runs after planning | 1, plus at most 1 defect rerun | 2; initial pass plus allowed log-visibility rerun |
| Fresh monolithic CI attempts | 3 | 3 |
| Fresh sharded CI attempts | 3 | 3 |
| Optional rebalance CI attempts | 0 or 3 | 0; 1.064 ratio did not trigger it |
| Peak required job slots | 6 | 6 configured and observed |
| Median runner minutes, before/after | report both | 23:26 / 30:43 (+31.1%) |
