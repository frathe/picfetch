# 04: Protect file actions without losing committed effects

**What to build:** Export, Trash, reveal and wallpaper consistently respect
admission while retaining their captured subjects. Prompt controls still work,
and completed writes still update caches and file information after the view or
input owner changes.

**Blocked by:** 01: Establish shared admission through Save Changes.

**Status:** resolved (done; implementation and local verification).

## Answer

Implemented in commit `9dc3a81` using vertical TDD slices under the accepted SDD plan. T0 reviewed
and fixed the integrated change. The final `make verify` passed, including all
native Linux/amd64 Docker race partitions. All 68 changed Go files returned no
GoLand inspection findings, including requested weak warnings.
See the [command/route and verification record](../../command-admission-verification-2026-09-27.md)
for observed red/green results, exact inspection scope/hash and retained logs.
This resolves the implementation slice, not MA-028's final acceptance: native
physical-input, Windows/macOS execution and fresh CI/Qodana/CodeQL evidence
remain open under [ticket 10](10-native-qualification.md). Its later record
includes the completed Linux reset repair, case-insensitive export qualification
and focused native CI selection. Checked items
record implementation and the available deterministic verification, not native
runtime qualification. References to native close/accelerators below retain
that separate acceptance gate.

**Agent/model:** T0 Codex Lead, GPT-6 Astra, high reasoning; T0 owns the distinction
between new user entry and operation-owned/committed continuations.

**Context:** [MA-028 spec](../spec.md), file/image family and AC3/AC6/AC9; use the
[execution plan](../../../plans/2026-09-27-ma-028-command-admission.md).

- [x] Migrate export, Trash, reveal and ordinary wallpaper across actual menu,
  registered shortcut, info link and bare user entries where applicable. Include
  root-facing mosaic wallpaper entry in the inventory without retargeting its
  exact result. Check capability before yielding or operation-starting capture.
  Verify V04.
- [x] Delete/export/real-dialog ownership blocks unrelated commands. Their own
  choices, keyboard controls, options and cancellation remain usable, as does
  native close. A legitimate menu selection is not mistaken for a dialog.
  Verify V04 through actual prompts and entry routes.
- [x] Preserve captured files/pixels, format/options and wallpaper target through
  later navigation. Shared decisions choose a target kind, never fresh payloads
  after a worker has begun. Retain cancellation and worker/queue settlement.
  Verify V04.
- [x] Live checks at genuinely new async admission points coexist with tokens
  and identity. Already committed Save/export/metadata-removal/mosaic writes
  reconcile under existing file-work ownership after navigation or modal entry;
  they must not be discarded by the fresh-command policy. Verify V04 with held
  completion and observable cache/info effects.
- [x] Menus reflect this family's modal/busy/capability decisions, attempts keep
  established feedback, and unavailable actions preserve idle selection. Remove
  duplicated cross-feature admission while retaining feature-local validity.
  Record route cases, red/green, inspections and exact shard/test exclusions.

**Verification V04:**

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestCommandAdmissionModalOwnership|TestCommandAdmissionAsync|TestSaveChanges.*|TestExport.*|TestFileMutation.*|TestReveal.*|TestWallpaper.*|TestMosaicWallpaper.*|TestDelete.*|TestRequestDelete.*)$'`

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestCommandAdmissionRoutes$/^files$'`

Use OS stubs and temporary fixtures. This ticket extends the Save slice; it does
not repeat its policy implementation or redesign file transactions.

## Comments

2026-09-27: Reconciled the checklist with commit `9dc3a81` and its retained
verification record. File-action, modal ownership and committed-effect regressions passed on the available filesystem. The skipped case-insensitive export case remains ticket 10.
