# Linux worker isolation in the local amd64 test container

Implemented and verified September 12, 2026.

Route: Standard; test tooling and documentation only. Lead owns implementation
and review, with no delegation. No shipped dependency or license changes.

## Problem and evidence

The September 11 Apple Silicon verification record reports seccomp `EINVAL`
under Linux/amd64 emulation, while the same race tests pass in a native ARM64
container. The original logs are absent from this checkout; the retained record
is [the Windows setup plan](../plans/2026-09-11-explorer-windows-setup.md#confirmed-svg-source-map-crash-and-fix).
This session's Docker daemon reports Linux/x86_64 with a native x86_64 kernel.
Docker documents that amd64 containers on Apple Silicon use emulation and may
not work: <https://docs.docker.com/desktop/troubleshoot-and-support/troubleshoot/known-issues/>.
This supports an execution-environment limitation, not a change to worker policy.
The exact Apple Silicon emulator/version remains unverified here.

## Decisions

| Decision | Rationale |
| --- | --- |
| Require a native Linux amd64 Docker daemon for complete local suites | The gate must exercise real amd64 seccomp enforcement and canonical golden pixels together. |
| Inspect the selected Docker daemon, not the client host | An ARM client can use a suitable daemon; client `uname` is insufficient. |
| Fail before container creation/setup | Avoid discovering an unsupported environment after the long race run. |
| Keep golden generation and shard inventory available under emulation | Neither claims worker isolation qualification. |
| Retain native amd64 CI and all worker tests unchanged | No skips, seccomp relaxation, or architecture substitution. |

Non-goals: split-architecture suites, production sandbox changes, emulator setup,
CI workflow changes. Honest limit: Apple Silicon's ARM Docker VM cannot run the
complete gate; run it from a checkout on native Linux amd64 or use existing CI.
A native ARM64 focused check is supplementary evidence only.

## Acceptance criteria

1. Complete public targets (`test`, `coverage`, `test-race`, `verify`) reject ARM,
   unknown/non-Linux daemons and Docker query failures before setup; both Docker
   amd64 architecture spellings pass. Client architecture is irrelevant.
   Command: `go test ./scripts/testshards -run TestMakeTestPlatform -count=1`.
2. Golden/shard commands remain usable; complete unsharded test/coverage and
   concurrent race contracts remain intact, and native CI retains worker tests.
   Command: `go test ./scripts/testshards -count=1`.
3. The actual worker isolates existing/new threads and reaches the asset check
   on native Linux amd64 with the race detector.
   Command: `go test -race -count=1 -run '^(TestLinuxWorkerIsolation|TestAssetInstall)$' ./internal/similarity ./scripts/explorereval`.
4. Final repository gate passes: `make verify`; GoLand inspects the changed test
   file including weak warnings. Unavailable checks are recorded explicitly.

## Tasks

### Task 1 — Fail early for incompatible Docker daemons
Owner: T0 inline.
Files: Makefile, scripts/testshards/main_test.go.
Contract: `check-test-platform` validates the selected daemon; complete public
targets depend on it, while golden and shard inventory do not.
Test: command fixtures cover architecture admission, Docker errors, early stop,
and preserved suite/CI contracts.
Verify: acceptance criteria 1 and 2.
Budget: 0 spawns; 1 review round; no full suite during iteration.

### Task 2 — Record supported workflows and verify
Owner: T0 inline. Depends: Task 1.
Files: README.md, AGENTS.md, todos.md, this plan.
Contract: documentation distinguishes canonical gate from supplementary native
ARM64 evidence and explains how to inspect the daemon.
Verify: acceptance criteria 3 and 4, plus diff review.
Budget: 0 spawns; 1 final review; full suite once.

Task graph: Task 1 -> Task 2. All work remains inline.

## Evidence and cost ledger

- Red: the new command fixture failed because `check-test-platform` and its
  target dependencies were absent.
- Green: `go test ./scripts/testshards -run '^TestMakeTestPlatform$' -count=1`
  passes (0.246s). Deliberately admitting `linux/arm64` made the regression fail
  for accepting that daemon; restoring the gate returned it to green.
- Complete tooling package: `go test ./scripts/testshards -count=1` passes
  (3.126s). The initial sandboxed run failed existing Git/VCS fixture queries;
  the same command outside that sandbox passes without code changes.
- Native Linux amd64 focused race checks: similarity 2.050s, explorereval 1.098s,
  both pass. The sandbox initially blocked the parent loopback socket; running
  outside the agent sandbox confirmed the unchanged parent/worker distinction.
- GoLand inspected Makefile (no findings) and the complete changed test file.
  Four weak duplicate-fragment warnings at lines 89, 173, 224 and 274 cover
  unchanged fixtures. These are intentional test repetition covered by the
  existing exact `scripts/testshards/main_test.go` exclusion in qodana.yaml.
  Reinspection after review reports the same four warnings, no new findings.
- Diff review confirms no changes to production worker code, its tests, the
  native Linux amd64 CI runner or direct race partition commands. The partition
  still includes similarity and explorereval with no worker-test exclusions.
- The final `make verify` has passed formatting, TUF/exclusion checks, vet,
  build and the 682-entry UI shard inventory. Native amd64 Docker has passed
  `TestLinuxWorkerIsolation` (1.030s), the worker asset-check subtest (0.280s),
  and `TestMakeTestPlatform` (0.680s). Their complete race packages pass:
  similarity 2.642s, explorereval 1.569s, testshards 13.059s.
- `make verify` completed with exit 0 on native Linux amd64. All three UI race
  partitions pass: ui-1 1227.821s, ui-2 1253.371s, ui-3 1337.801s. Across the UI
  inventory, 681 top-level tests pass and the existing case-alias test skips on
  the case-sensitive filesystem. No worker guard skips. Full log:
  `.scratch/linux-worker-isolation/verify.log`; raw events and container evidence:
  `.scratch/race-runs/20260912T194211Z-fp5wGI/`.
- Host/container exit codes are both 0. Docker reports `OOMKilled=false`; cgroup
  OOM counters are zero, peak memory is 7,637,266,432 bytes within the 16 GiB
  limit, and no diagnostic-collection errors were recorded. Non-UI results:
  1,925 top-level passes, four existing platform/permission skips, zero failures.

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
| --- | --- | --- | --- | --- |
| 1 | 0/0 | 1 | no | Command fixtures and negative guard check pass. |
| 2 | 0/0 | 1 | yes, once | Complete native Linux amd64 gate passes. |
