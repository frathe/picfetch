# 02: Preserve text editing and all copy targets

**What to build:** Copy and Select All reach the focused text editor through
real application input paths. Outside an editor/modal owner, Copy keeps the
image-region, Grid-file, displayed-image priority. Menus accurately reflect
busy region copying and recover when it ends.

**Blocked by:** 01: Establish shared admission through Save Changes.

**Status:** resolved (done; implementation and local verification).

## Answer

Implemented in commit `9dc3a81` using vertical TDD slices under the accepted SDD plan. T0 reviewed
and fixed the integrated change. The final `make verify` passed, including all
native Linux/amd64 Docker race partitions. All 68 changed Go files returned no
GoLand inspection findings, including requested weak warnings.
See the [command/route and verification record](../../command-admission-verification-2026-09-27.md)
for observed red/green results, exact inspection scope/hash and retained logs.
This resolves the implementation slice. All-platform native acceptance is
now complete in [ticket 10](10-native-qualification.md), including the Linux
reset repair, case-insensitive export, focused native CI guards and physical
editor-input evidence. The clean `e2d3b30` review round permits final acceptance;
PR 66 records fresh gates for the documentation-only closure head. Checked
items below retain their original implementation/deterministic scope; the
separate platform records establish native runtime qualification.

**Agent/model:** T0 Codex Lead, GPT-6 Astra, extra-high reasoning. Native menu
interception, intent and clipboard lifetimes make this the highest-risk input slice.

**Context:** [MA-028 spec](../spec.md), especially D5-D7 and AC4-AC7; shared
interfaces and completion rules are in the
[execution plan](../../../finished_refactorings/2026-09-27-ma-028-command-admission.md).

- [x] Migrate Copy, explicit Copy image, Copy Path, Copy Selection activation
  and Select All through each actual menu/accelerator/registered-shortcut/bare
  entry. Record intent, target and N/A routes. Retain explicit image intent when
  focus alone changes, and modal refusal when a prompt owns input. Verify V02.
- [x] Demonstrate positive text Copy/Select All delivery and unchanged underlying
  image/Grid state. Test production adapter dispatch, not only direct invocation
  of an Entry's shortcut method. Account for main-menu accelerators running before
  focused-widget dispatch and without consulting Disabled. Verify V02; collect an
  early native feasibility observation where available, with missing platforms
  explicitly reserved for ticket 10.
- [x] Preserve region-before-Grid-before-displayed-pixels targeting, including
  requested-versus-displayed identity, uncommitted rectangles, repeated region
  activation and region-owned confirm/cancel. Do not move geometry into policy.
  Verify V02.
- [x] Busy region copy is a separate fact from ordinary clipboard serialization.
  Refresh policy-derived menu availability at busy entry/exit; refused attempts
  retain feedback and native close remains possible. Preserve clipboard worker
  admission, cancellation, delivery and completion ownership. Verify V02.
- [x] Recognized unavailable clipboard commands preserve idle selection. An
  admitted command performs only its required yield before effects. Retain local
  zoom/pan behavior. Remove this family's duplicated cross-feature checks and
  retain feature-local validity. Verify V02 and record inspections/shard exclusions.

**Verification V02:**

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestCommandAdmissionTextEditing|TestCommandAdmissionBusyCopy|TestCopySelection.*|TestClipboardBusyPreservesActiveOperationAcrossCopyRoutes)$'`

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestCommandAdmissionRoutes$/^clipboard$'`

Use held work and observable completion, never sleeps. Software-driver results
do not establish native accelerator qualification; no dependency upgrade is
pre-authorized as a way to solve dispatch.

## Comments

2026-09-27: Reconciled the checklist with commit `9dc3a81` and its retained
verification record. Focused-editor routing, copy-target priority and busy completion/cancellation regressions passed. Native accelerator observations remain ticket 10.
