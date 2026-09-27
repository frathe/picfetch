# 02: Preserve text editing and all copy targets

**What to build:** Copy and Select All reach the focused text editor through
real application input paths. Outside an editor/modal owner, Copy keeps the
image-region, Grid-file, displayed-image priority. Menus accurately reflect
busy region copying and recover when it ends.

**Blocked by:** 01: Establish shared admission through Save Changes.

**Status:** draft (publish as ready-for-agent after breakdown approval).

**Agent/model:** T0 Codex Lead, GPT-6 Astra, extra-high reasoning. Native menu
interception, intent and clipboard lifetimes make this the highest-risk input slice.

**Context:** [MA-028 spec](../spec.md), especially D5-D7 and AC4-AC7; shared
interfaces and completion rules are in the
[execution plan](../../../finished_refactorings/2026-09-27-ma-028-command-admission.md).

- [ ] Migrate Copy, explicit Copy image, Copy Path, Copy Selection activation
  and Select All through each actual menu/accelerator/registered-shortcut/bare
  entry. Record intent, target and N/A routes. Retain explicit image intent when
  focus alone changes, and modal refusal when a prompt owns input. Verify V02.
- [ ] Demonstrate positive text Copy/Select All delivery and unchanged underlying
  image/Grid state. Test production adapter dispatch, not only direct invocation
  of an Entry's shortcut method. Account for main-menu accelerators running before
  focused-widget dispatch and without consulting Disabled. Verify V02; collect an
  early native feasibility observation where available, with missing platforms
  explicitly reserved for ticket 10.
- [ ] Preserve region-before-Grid-before-displayed-pixels targeting, including
  requested-versus-displayed identity, uncommitted rectangles, repeated region
  activation and region-owned confirm/cancel. Do not move geometry into policy.
  Verify V02.
- [ ] Busy region copy is a separate fact from ordinary clipboard serialization.
  Refresh policy-derived menu availability at busy entry/exit; refused attempts
  retain feedback and native close remains possible. Preserve clipboard worker
  admission, cancellation, delivery and completion ownership. Verify V02.
- [ ] Recognized unavailable clipboard commands preserve idle selection. An
  admitted command performs only its required yield before effects. Retain local
  zoom/pan behavior. Remove this family's duplicated cross-feature checks and
  retain feature-local validity. Verify V02 and record inspections/shard exclusions.

**Verification V02:**

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestCommandAdmissionTextEditing|TestCommandAdmissionBusyCopy|TestCopySelection.*|TestClipboardBusyPreservesActiveOperationAcrossCopyRoutes)$'`

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestCommandAdmissionRoutes$/^clipboard$'`

Use held work and observable completion, never sleeps. Software-driver results
do not establish native accelerator qualification; no dependency upgrade is
pre-authorized as a way to solve dispatch.
