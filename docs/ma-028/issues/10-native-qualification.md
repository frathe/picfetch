# 10: Qualify native input and accept the integrated migration

**What to build:** Establish actual Linux, Windows and macOS desktop evidence
that editor shortcuts reach text, menus reflect current admission, and native
dispatch cannot bypass the integrated policy. Close MA-028 only with complete
deterministic and native evidence.

**Blocked by:** None; prerequisite 09 is resolved. Native desktop access,
physical-input observations and completed remote checks are still required.
The user authorized push and PR creation on 2026-09-27.

**Status:** ready-for-human (Linux native reset defect repaired and locally
verified; physical-input, Windows/macOS and CI qualification remain open).

## Linux qualification results (2026-09-27)

Requested by the user on this host. T0 owns native execution and all assessment.
One bounded read-only scout assignment locates safe launch isolation and existing
clipboard hold/observation seams while T0 checks desktop access. G1-G5: fewer than
25 prompt lines, source citations verified by targeted reads, zero edited files,
small launch/clipboard scope, distinct from T0's native input/display setup.
No implementation or review was delegated; that initial Linux qualification
did not include authorization for PR/CI action.

T0 ran the real Make-built native application at clean revision `6db8d73`
(unchanged implementation `9dc3a81`) on Ubuntu 24.04.5/GNOME Wayland with XWayland,
using window-targeted XTEST input. The four native scenario groups passed;
see [retained Linux evidence](../../command-admission-linux-qualification-2026-09-27.md)
for exact launch/harness, screenshots, payload comparisons and limitations.
Both app sessions exited normally and held clipboard launchers were released.
The initial qualification changed no production code; the subsequent reset
repair is recorded below. The spec's physical-input and literal `make run`
procedure is not silently replaced: this used injected OS events and a directly
launched `make build` binary. Windows/macOS remain untested.

### Completed Linux runtime checks (OS-injected input)

These scoped observations remain valid. The user-confirmed maximized-window
Escape defect is now repaired and verified below; literal physical-input and
the other platform/external gates still prevent complete acceptance.

- [x] Record clean revision, native build, OS/architecture, Fyne version, isolated
  profile, actual input route and retained artifacts.
- [x] Positive editor Ctrl+A/C in image and Grid naming dialogs: correct text
  payload/full field selection; underlying Grid screenshot unchanged.
- [x] Native Ctrl+C and actual Actions -> Copy image produce the fixture pixels;
  real Favorite modal blocks unrelated shortcuts/menu entry and its Add control
  saves only the isolated fixture list.
- [x] Reproducibly hold real region clipboard completion with a process-local
  launcher/FIFO. Verify disabled menus, Save refusal feedback, preserved PNG,
  and enabled menu/Copy Path recovery after release.
- [x] Exercise real delete/export owner cancel/focus controls and refusal of
  conflicting shortcuts; native close exits while busy and while modal.
- [x] Recheck retained payloads and fixture hashes; close both test windows and
  release held launchers. Carry forward unchanged-code deterministic/IDE evidence.

Physical hardware input remains unobserved. The hold is a child-scoped external
launcher, not the instance encoder seam mentioned in the runbook. No policy or
feature behavior was altered for testing. The local ext4 filesystem rejected
casefold on a new empty test directory (`Operation not supported`), so the
case-insensitive export regression initially required another filesystem; the
later FAT16 qualification below now closes that gap.

Implementation evidence: [verification record](../../command-admission-verification-2026-09-27.md).
Integrated commit: `9dc3a81` on `feature/ma-028-command-admission`.
Linux reset repair commit: `75fd69e`, with tracked evidence and local artifacts.
Case-insensitive qualification and native CI gap record: `574cf82`.
Focused native CI guards, verification and portability audit: `ba43d1b`.
The final local `make verify` and all 68 changed-file GoLand inspections passed.
Those local evidence commits preceded publication authorization. The user has
since approved moving this tracker to docs, committing, pushing and opening a
PR; remote checks remain unverified until their actual results are inspected.
The Linux focused run explicitly skipped
`TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset` on its case-sensitive
filesystem. That skip was not a passed AC9 case; the later actual FAT16 run at
`75fd69e` passed, including five race repetitions, and supplies the missing
filesystem observation without claiming Windows/macOS native qualification.

**Agent/model:** A native desktop operator on each OS, supported by the T0 Codex
Lead on GPT-6 Astra, high reasoning. A model cannot substitute for an unavailable
OS or physical/native accelerator evidence. T0 owns all diagnosis/fixes/review.

**Context:** [MA-028 spec](../spec.md), AC10 and its four-step native procedure;
the [execution plan](../../../plans/2026-09-27-ma-028-command-admission.md)
requires an early feasibility observation during 02 and final evidence here.

## Completed preparation

- [x] Integrate tickets 01-09 in commit `9dc3a81`; retain the completed
  command/route inventory, lead structural review and red/green evidence.
- [x] Pass the final local `make verify`, including all native Linux/amd64
  Docker race partitions, and record explicit environment/platform skips.
- [x] Inspect all 68 changed Go files with GoLand; record the exact scope/hash
  and no findings. Preserve the distinction from Qodana CI/SARIF.
- [x] Trace pinned Fyne accelerator dispatch and pass Windows internal-package
  cross-vet. Neither establishes physical/native desktop behavior.
- [x] Record the native runbook, missing platforms and case-insensitive export
  regression in the verification record.

## Remaining acceptance checks

- [x] Repair Linux Escape/reset from ordinary WM maximization. Dynamic reset now
  requests native restore before resize, without changing fixed-size or
  Grid/Explorer-owned behavior. New boundary regression seen red and green;
  intentional omission through a Go overlay fails Escape and Close Files.
  Five ordinary native repeats, Grid, original multi-image/Grid/rotation,
  fixed ordinary maximize and fixed Grid controls pass. All three changed Go
  files inspect cleanly; Windows cross-vet and fresh `make verify` pass.
  Retain exact source hashes, screenshots, logs and
  `.scratch/race-runs/20260927T103730Z-3Tubvu` in the Linux repair record.
- [ ] On each native Linux/Windows/macOS desktop, run the integrated revision
  with `make run`; record OS/architecture, revision, Fyne version, input route,
  reproducible steps and actual visible/payload outcomes. Launch alone does
  not establish any scenario below. Verify V10.
- [ ] With loaded images/Grid and a focused naming/editing field, select
  distinctive text and use physical platform Copy and Select All. Confirm text
  payload/selection and unchanged underlying image/Grid, including the route
  that would otherwise match an application menu accelerator. Verify V10.
- [ ] Separately invoke explicit image menu actions in an admitted context and
  under a real dialog. Confirm image intent when admitted, modal refusal when
  blocked, and positive prompt control/confirmation/cancel behavior. Verify V10.
- [ ] Exercise actual menu selection and native availability through modal and
  held region-copy entry/exit. Disabled items cannot bypass live admission;
  legitimate menu selections still work, refusal feedback is preserved and
  native close remains usable. Hold copy reproducibly through an instance-owned
  seam if needed; do not rely on clipboard speed. Verify V10.
- [ ] Reconcile results with ticket 09 and all spec ACs. T0 fixes confirmed
  defects inline and repeats affected automated checks, inspections and native
  scenarios on the changed revision. A newly discovered behavior choice outside
  D4-D7 requires its concrete D1 scenario decision. An unavailable platform stays
  unverified and this ticket stays open.
- [x] Run `TestExportCommittedCaseAliasKeepsWrittenPixelsOnReset` on a
  case-insensitive filesystem: at `75fd69e`, actual Linux `vfat`/FAT16 storage
  passed once normally and five times with race instrumentation, with no skips.
  The test verifies case aliases identify the same file and reset preserves
  exported pixels. The temporary mount and owned loop device were removed;
  logs/hashes/setup history remain under `.scratch/ma-028/case-insensitive-2026-09-27`.
- [x] Add a focused native CI suite for the case-alias test and macOS Copy-menu
  assertion. Both jobs now invoke `command-admission` and retain its JSON;
  runner/workflow tests observed red then green reject absent/skipped guards
  and HEIC exemptions, without Linux-only goldens. Deliberate inventory/filter
  omissions fail. The actual new runner passes on FAT16 and rejects an ext4
  skip. GoLand found no issues in the two Go files and workflow; Windows
  runner cross-vet and fresh full `make verify` passed. The subsequent CI
  execution below remains distinct from physical-input/full-desktop acceptance.
  Current raw evidence: `.scratch/ma-028/native-ci-2026-09-27` and
  `.scratch/race-runs/20260927T113110Z-p0dfVe`; the tracked Linux record summarizes
  outcomes and exact inspection hashes for cross-desktop handoff.
- [x] Execute the focused guards in native Windows amd64 and macOS arm64/amd64
  CI. Run `36317733211` at `1bfca4a` passed case-alias export on all three and
  the native Copy-menu assertion on both macOS runners, with zero skips.
  T0 inspected the retained JSON events. See the
  [review-loop record](../../command-admission-pr-66-review-2026-09-27.md).
- [ ] Obtain fresh required CI and CodeQL results and inspect Qodana's
  post-suppression SARIF for the integrated revision. Local GoLand inspection
  evidence does not establish those external gates. Initial CI/CodeQL and Codex
  code/security reviews passed at `1bfca4a`; Qodana completed with eight findings,
  assessed in the review record. The user has invoked the review loop; fresh
  latest-commit results are required after the static-analysis cleanup.
- [ ] Record the final evidence and remaining limitations. Only after all
  required gates pass may the implementation be proposed as accepted and the
  plan archived. No merge, release, commit or push is implied.

**Verification V10:** `make run` on each native platform followed by the four
scenario groups above and retained observations. Test-driver dispatch, source
traces, unrelated native suites and a blank runbook cannot pass this ticket.
