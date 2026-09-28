# 09: Qualify the integrated implementation

Status: ready-for-agent
Parent: [MA-033 specification](../spec.md)
**Blocked by:** [08: Complete policy adoption and native guard coverage](08-convergence-and-native-guards.md).

**What to build:** a trustworthy handoff for the integrated implementation, with
evidence for every accepted behavior and every required native host/build.
Unavailable or incomplete gates remain explicit rather than being called passed.

## Acceptance criteria

- [ ] Reconcile all 22 parent acceptance criteria against actual command output,
  required child inventory, behavioral red/green evidence and tested revisions.
  Run all parent AC1-AC19 commands on their applicable hosts/builds; include the
  lead's completed AC21 caller assessment. Later changes invalidate affected
  earlier evidence until rerun. Verify: V1.
- [ ] Focused launch-policy qualification passes on native Linux, native Windows
  and both supported macOS architectures; the Store suite passes on native
  Windows with Store tags. Retain independent per-platform/revision captures.
  Actual offline prerequisites, Windows Explorer refusal, Open With ordering
  and predecessor behavior are observed without claiming new platform support.
  Verify: V2.
- [ ] Native trial-tool compatibility is observed on its required host; a
  controlled child is not real-model qualification, and reservation remains
  distinct from ready evidence. Verify: V3 and the applicable parent AC18 fixtures.
- [ ] Formatting/generated inputs/notices, vet, build, exact test exclusions,
  root-UI shards and the full race suite pass through the repository's supported
  Linux/amd64 Docker gate or qualifying native-amd64 CI. Never weaken isolation
  to accommodate emulation. Verify: V4.
- [ ] Every changed code file has complete GoLand inspection evidence including
  weak warnings, tool/profile, analyzed revision and disposition. Reinspect after
  fixes. Review fresh post-suppression Qodana SARIF when CI runs; licensing errors,
  timeouts, skipped files and unavailable hosts remain unverified. Verify: V5.
- [ ] The lead records the final scope, remaining limits and verification status;
  update the working evidence/todos and package map where ownership changed.
  Do not mark implementation accepted or archive an active plan prematurely.
  Verify: V1, V4, V5.

## Verification

Parent scope: AC18's native portion, AC20 and AC22; integrated evidence for all
AC1-AC22. The parent supplies the exact commands referenced by V1 and V5.

- V1: execute every command in the parent's **AC1-AC19 verification map** with
  its required tags/host and complete the **AC21 lead-assessment command** and
  caller dispositions. Check the [ownership map](README.md#acceptance-criterion-ownership)
  against recorded outputs; do not substitute the existence of ticket checkmarks.
- V2 (new runner suites from 08):
  `go run ./scripts/nativeguards -suite launch-policy -capture .scratch/ma-033/evidence/native-launch-policy.json`;
  on native Windows, `go run ./scripts/nativeguards -suite launch-policy-store -capture .scratch/ma-033/evidence/native-launch-policy-store.json`.
  Create the evidence directory first and use separate per-host/revision captures
  so one run never overwrites another host's evidence. A Store-tagged test does
  not establish installed Store-package behavior.
- V3 (existing, native macOS): `go test -tags no_emoji,nodynamic,explorertrial -count=1 -v ./scripts/explorereval -run '^TestNativeLibraryRunner$'`.
- V4: `make check-test-shards`; `make check-qodana-test-exclusions`; `make verify`.
- V5: run the parent's **AC22 inspection workflow** on the integrated revision
  and retain its required file-scope/profile/findings record. There is no shell
  command that turns missing IDE or SARIF evidence into a passing inspection.

Follow the [shared execution rules](README.md#execution-and-evidence-rules).
The final gate and any fixes are lead-owned. This ticket does not authorize a
push, GitHub review loop, merge or release. Network/filesystem sandboxing, model
quality and Location Map latency requalification remain outside MA-033.
