# Similarity protocol helper shutdown

Status: implemented and locally verified on
`fix/similarity-protocol-race-timeout`; GitHub review loop requested.
Route: Standard; one test-harness correction in `internal/similarity`.

## Problem and evidence

The next `todos.md` item records PR 61's failed complete Docker race gate:
the analysis protocol helper delivered its complete event, then its 20-second
`testing.M` alarm fired inside `os.Exit(0)` / `runtime_beforeExit`.
The original PR 61 raw directory is no longer present. Its retained record is
`plans/2026-09-26-pr61-license-notices.md:195`. An older identical 10-second
failure remains in `.scratch/os-heic/evidence/make-verify.log:1435`.

Baseline focused race test passed (package 4.360s; complete subtest 3.10s).
The original symptom reproduces deterministically with:

```sh
GORACE=atexit_sleep_ms=21000 go test -tags no_emoji,nodynamic -race -count=1 -timeout=90s -run '^TestAnalysisProtocolPreservesLimitErrorsAndConfiguration$/complete$' -v ./internal/similarity
```

Before the fix: FAIL after 20.140s, `panic: test timed out after 20s`,
`os.runtime_beforeExit(0x0)`, `os.Exit(0x0)`, and `events=1`.
This isolates the competing test alarm from streaming and pipe cleanup.
It does not establish how much PR 61's individual phases were slowed by load.

## Decisions and scope

| Decision | Contract |
| --- | --- |
| Helper dispatch | Use the existing `TestMain` worker-dispatch pattern before `m.Run`; protocol stdout contains only protocol frames. |
| Deadline ownership | Parent command context bounds startup, streaming and race shutdown to one minute; cancellation still kills and joins the child. |
| Regression boundary | Existing real subprocess / `analyzeCommand` integration seam; retain 50,655-item/configuration and resource-error checks. |
| Race behavior | Preserve race instrumentation and its normal exit handling. |

No application behavior, wire protocol, dependency, license closure, worker
isolation, translation, or unrelated search helper changes. The separate
architecture and native gesture todos are outside this change. Extremely slow
or stuck helpers still fail the parent deadline; this is an intentional bound.

## Acceptance criteria

- AC1: The configured 50,655-item completed result and typed memory-limit error
  survive real subprocess transport, with child exit observed. A 1ns child
  test alarm must not run in worker mode.
  `go test -tags no_emoji,nodynamic -race -count=3 -run '^TestAnalysisProtocolPreservesLimitErrorsAndConfiguration$' ./internal/similarity`
- AC2: Parent cancellation after completed delivery still terminates and joins
  a helper that remains alive. Same command as AC1.
- AC3: The original extended race-shutdown reproduction passes after the fix.
  Run the reproduction command above.
- AC4: Repository verification passes: `make verify`; changed Go files receive
  complete GoLand inspections including weak warnings. Unavailable gates are
  recorded as unverified.

## Tasks and ownership

1. T0: strengthen the existing subprocess test in `analyze_test.go`; observe
   RED with a tiny child test deadline. Dispatch the helper in `main_test.go`,
   apply parent deadline ownership, and add cancellation coverage. Verify AC1–3.
   Budget: zero implementation spawns, two review rounds, no broad suite.
2. T0: inspect/review, run `make verify`, update this evidence and `todos.md`.
   Depends on task 1. Budget: zero spawns, one final broad suite.

Independent recon scout: retained logs and Docker concurrency/environment only.
G1: bounded read-only question; G2: reported paths/lines checked with shell;
G3: no writes; G4: artifact sweep avoids copying full logs; G5: lead has not read
those artifacts. Script search preceded delegation; no implementation or review
delegated. Available inherited scout model used; repository's named tier models
are not available in this harness.

Task graph: recon -> test/fix -> final gate -> evidence/todo handoff.

## Verification and ledger

Base revision: `0ce849cf7491a3cadcdf77cacca1268b96bd0994` plus this branch's
changes. After local verification, Ronin authorized commit, push, PR creation
and the GitHub Codex review loop. The loop includes finding dispositions and
fresh code/security reviews, CodeQL, Qodana SARIF assessment and required CI on
the latest pushed commit. Merge and release remain outside this request.

- RED: the 1ns helper-alarm regression failed with `panic: test timed out
  after 1ns` before dispatch changed (package 0.128s).
- GREEN: the same real subprocess test passed after dispatch changed
  (package 4.341s). The guard was observed failing before the fix; no deliberate
  violation remains.
- AC1/AC2: three focused race repetitions passed after adding the cancellation
  case: `ok github.com/frathe/picfetch/internal/similarity 17.388s`.
  Each run preserved the resource-error case and complete 50,655-item snapshot;
  cancellation returned `context.Canceled` with a joined child.
- AC3: the extended shutdown command passed:
  `ok github.com/frathe/picfetch/internal/similarity 46.330s`.
  Complete delivery plus helper exit took 23.15s, beyond the old 20-second alarm.
  The subtest regexp also selected `cancel-after-complete` (2.15s); the parent
  race process's own injected 21-second exit sleep accounts for the remaining
  package time. No race-detector options are changed in committed test code.
- GoLand fallback inspections: IDE-local Qodana execution is not exposed by
  the available tools. `get_file_problems(errorsOnly=false)` inspected both
  changed Go files, requesting errors and all warnings, including weak warnings.
  Both returned empty findings without timeouts. This is the current GoLand
  profile, not an equivalent `qodana.starter`/post-suppression SARIF scan.
- Inspected file SHA-256:
  `internal/similarity/analyze_test.go` =
  `2472ef0a1b9f0d1f53683fa78ded7cf81c82371803219280e6d910f05b999111`;
  `internal/similarity/main_test.go` =
  `2d15352d3869712c6cf0a3a7ce7f81740037203eef8e6ed8f797eb890c00a8ff`.
- Lead review: only existing test files changed; existing Qodana exact exclusions
  apply. No top-level UI test or package/file movement, so shard assignments and
  architecture map need no changes. No new dependencies or notice obligations.
- `git diff --check` passed. `make check-test-platform` passed for the native
  Linux/amd64 Docker daemon.
- AC4: `make verify` passed with exit 0: formatting, TUF root, exact Qodana
  exclusions, generated assets, dependency notices, vet, build, shard inventory
  and the complete concurrent Docker race suite. Artifacts:
  `.scratch/race-runs/20260926T212204Z-L8Ptud`; both host and container exit 0.
  Similarity package: 71.272s; its protocol test: 21.09s across all three cases
  (limit 0.59s, complete 10.55s, cancel-after-complete 9.95s).
  All UI shards passed: ui-1 470.317s, ui-2 360.024s, ui-3 374.336s.
  No reported test failures or race warnings. This full run covers the
  concurrency conditions of the original failed gate.
- PR 61's original full verification remains a failed historical gate; this
  branch's new run is recorded separately. The PR review threads/checks and
  final loop comment will retain remote review and CI evidence for the pushed
  revision. Local inspection evidence carries forward only while the inspected
  code hashes above remain unchanged.

| Task | Spawns budget/actual | Review rounds | Full suite |
| --- | --- | --- | --- |
| Recon | 1 / 1 | n/a | no |
| Test/fix | 0 / 0 | 1 | no |
| Final gate | 0 / 0 | 1 | one, passed |
