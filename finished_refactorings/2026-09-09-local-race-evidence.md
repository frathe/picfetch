# Preserve local race-test evidence

Route: Standard. The user accepted the defaults: treat the previous memory
mitigation and subsequent complete passes as sufficient, fix diagnostic
retention, and investigate races further only if a failure recurs.

## Problem and accepted decisions

The public Docker runner discards its four raw streams on container removal.
An attempted host `TEST_CAPTURE` override did not reach the inner Make process.
The September 9 browse verification retains that explicit limitation.

- Keep the 16 GiB limit, VM preflight, four concurrent partitions and race flags.
- Give every public race invocation a unique host directory under
  `.scratch/race-runs/`; `TEST_ARTIFACTS_DIR` changes the parent directory.
- Preserve raw streams, console output, run status, Docker state/OOM events,
  and available cgroup memory counters before removing the container.
- Record unavailable diagnostics explicitly. A failed test/container must
  remain a failure even if diagnostic collection or cleanup also fails.
- Direct partition targets retain `TEST_CAPTURE` for CI and prepared runners.
- No dependency upgrade, lower-memory mode, automatic retry or race-code fix.

These are reversible tooling choices, with no new domain terminology or
hard-to-reverse architectural trade-off; no glossary entry or ADR is needed.

## Historical boundary

- The September 7 Qodana gate's package-only ui-3 failure remains unexplained;
  its isolated retry passed. See [the original record](../finished_refactorings/2026-09-07-qodana-findings.md).
- Later runs established Docker memory exhaustion independently. A retained
  [packaging record](../finished_refactorings/2026-09-07-maintainability/evidence/27-reviewed-packaging-smoke.md)
  records `oom_kill=1`, peak 7,931,346,944 bytes and VM memory 8,319,213,568
  bytes, followed by a successful `GOGC=25` retry. The [release follow-up](../finished_refactorings/2026-09-08-release-review-followups.md)
  separately records an OOM run and its successful `GOFLAGS=-p=1` retry.
- Commit `6c35f38` added the 16 GiB limit and VM preflight on September 8.
  The September 9 [grid-scroll gate](../finished_refactorings/2026-09-08-grid-scroll-follow-duplicate-merges/verification/make-verify.log)
  has four retained raw streams: 53 non-UI package passes, one package skip,
  and three UI package passes, with no failure event. The later [grid-browse
  console](../.scratch/grid-duplicate-browse-during-scan/evidence/verify.log)
  has 56 package passes and no failure event, but lost its raw streams.
  Both runs used the normal four-partition commands and 16 GiB budget.

## Acceptance criteria and agreed test boundary

The existing Make/Docker CLI boundary is the default test seam. Tests use fake
Docker/system commands and temporary artifacts, never the real desktop.

1. Successful and failing public runs keep distinct host evidence directories
   after container cleanup, including all four raw streams and console output.
   Verify: `go test ./scripts/testshards -run TestMakeRaceArtifacts -count=1`.
2. OOM/exit evidence survives collection errors and interrupted/failed runs;
   available cgroup counters are retained, unavailable ones are identified.
   Verify: `go test ./scripts/testshards -run 'TestMakeRaceArtifacts|TestRaceContainerEvidence' -count=1`.
3. The canonical concurrency, flags, memory budget and direct CI capture
   interface remain intact. Verify: `go test ./scripts/testshards -count=1`.
4. One canonical `make verify` passes and its default host artifacts contain
   complete raw streams and memory/status diagnostics. Verify: `make verify`,
   then parse the four saved JSON streams and inspect diagnostic files.

## Tasks and ownership

1. Lead: add a failing artifact-retention regression to the existing
   `scripts/testshards/main_test.go`, then implement Make wiring and a
   `scripts/testshards/docker-race.sh` runner. Contract: public
   `TEST_ARTIFACTS_DIR` parent; direct `TEST_CAPTURE` unchanged. Verify AC1.
2. Lead, after 1: extend failure/diagnostic cases and collection. Verify AC2/3;
   negatively verify guards before final review.
3. Lead, after 2: README, architecture locator, TODO reconciliation and final
   verification/evidence record. Verify AC4, archive this accepted plan.

One reused read-only Scout checks primary Docker/cgroup documentation while
the lead owns the test and implementation. G1: bounded API facts; G2: official
sources; G3: no edits; G4: independent API lookup; G5: facts not yet held.
No implementation, design or review is delegated. Budget: one Scout, two
review rounds, one full gate unless an actual failure justifies a retry.

## Outcome

Artifact-retention tests failed against the original Make recipe because no
host evidence directory survived. After the runner change they passed for
successful and failed runs. Additional guards first failed on missing Docker
state/events and silent collection failures, then passed after collection was
implemented. Container snapshots cover cgroup v2, v1 and unavailable counters.
The interruption fixture waits for attachment, sends TERM, and observes worker
completion and container cleanup without running Docker.

Native `go test -race ./scripts/testshards -count=1` passed (8.446s). The first
attempt could not access a shared Go cache file inside the sandbox; the
authorized retry used that existing cache. Five temporary mutations separately
disabled the capture mount, failure propagation, Docker state collection,
final memory snapshot, and container cleanup. Every named guard failed for
its intended assertion; the runner was restored after each. Logs and the
mutation harness are retained under `.scratch/local-race-evidence/`.

The diagnostic limits follow the [Docker events reference](https://docs.docker.com/reference/cli/docker/system/events/)
(last 256 daemon events) and [Linux memory interfaces](https://www.kernel.org/doc/html/latest/admin-guide/cgroup-v2.html#memory-interface-files).
A killed container may never deliver its final live cgroup snapshot; its
retained Docker state and partial streams remain separately useful.

Final `make verify` passed in one run with the default 16 GiB budget and four
concurrent Linux/amd64 partitions. Host format/TUF/Qodana guards, vet and build
passed; the linker emitted its existing duplicate `-lobjc` warning. The Linux
runner tests passed under race detection (16.716s), all 53 non-UI test packages
passed (one additional package had no tests), and UI shards passed in
328.621s, 291.560s and 283.229s.

Default artifacts survive at
`.scratch/race-runs/20260909T075350Z-NV7m2s/`; the complete root gate log is
`.scratch/local-race-evidence/verify.log`. Every raw JSON line was parsed,
with no failure events or unfinished package/test lifecycles. Both saved exit
codes are zero, Docker state reports stopped with no OOM kill, and all before/
after cgroup OOM counters are zero. Peak memory was 12,653,977,600 bytes against
the 17,179,869,184-byte limit. No diagnostic was unavailable. The saved cleanup
result and a subsequent filtered Docker container listing confirm removal.
The machine-readable checks are in
`.scratch/local-race-evidence/artifact-summary.json`.

The TODO is closed under the accepted defaults. The original September 7 package-only
failure remains unexplained; later OOM events and successful mitigated runs
must not be retroactively presented as proof of its cause.

| Task | Spawns budget/actual | Review rounds | Full suite |
| --- | --- | --- | --- |
| API reconnaissance | 1/1 reused Scout | source facts checked | no |
| Runner, guards and failure cases | 0/0 | 2, lead-owned | no |
| Documentation and final gate | 0/0 | 1, lead-owned | 1/1, passed |

No commit was made. Suggested commit: `build: preserve local race-test diagnostics`.
