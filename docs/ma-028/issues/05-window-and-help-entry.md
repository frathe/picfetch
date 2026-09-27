# 05: Unify ordinary window and Help entry

**What to build:** Viewer, Grid, Picture-frame mode, comparison, EXIF, Settings
and Help entry respect shared admission without losing deliberate show/toggle
differences or comparison's Help access.

**Blocked by:** 01: Establish shared admission through Save Changes.

**Status:** resolved (done; implementation and local verification).

## Answer

Implemented in commit `9dc3a81` using vertical TDD slices under the accepted SDD plan. T0 reviewed
and fixed the integrated change. The final `make verify` passed, including all
native Linux/amd64 Docker race partitions. All 68 changed Go files returned no
GoLand inspection findings, including requested weak warnings.
See the [command/route and verification record](../../command-admission-verification-2026-09-27.md)
for observed red/green results, exact inspection scope/hash and retained logs.
This resolves the implementation slice. Linux and Windows qualification are
complete; only macOS desktop acceptance remains under
[ticket 10](10-native-qualification.md), with fresh remote gates required on
each final PR head. Its later record
includes the completed Linux reset repair, case-insensitive export qualification
and focused native CI selection. Checked items
record implementation and the available deterministic verification, not native
runtime qualification. References to native close/accelerators below retain
that separate acceptance gate.

**Agent/model:** T0 Codex Lead, GPT-6 Astra, high reasoning. Feature-created
menus and links require root composition decisions and lead-owned review.

**Context:** [MA-028 spec](../spec.md), window family and AC2/AC3/AC5/AC8; follow
the [execution plan](../../../plans/2026-09-27-ma-028-command-admission.md).
Explorer/Location Map/mosaic entry belongs to ticket 06.

- [x] Migrate actual Window/File/Help callbacks, accelerators, plain keys,
  comparison shortcuts, info links and bare user entries for the named surfaces.
  Add true bare-handler coverage to the existing window admission matrix.
  Verify V05.
- [x] Preserve G/P toggles versus idempotent menu Show and Grid's P ownership.
  Keep comparison target capture and covered-grid isolation. Help remains
  available during comparison, subject to an actual modal owner. Verify V05.
- [x] Cover Help's feature-created Manual, Release Notes, Licenses, About and
  Discussions entries and applicable direct links; classify Finis and both
  Hypno Spiral entry routes. Preserve feature-local controls and automatic
  notifications as such rather than treating every method as a new main-window
  command. Verify V05 with explicit cases/reasons in the route inventory.
- [x] Unavailable entries preserve idle Copy Selection; admitted yielding
  entries cancel it before opening. Busy/modal availability reaches all relevant
  menus, including feature-created Help menus. Native close remains available.
  Querying menus must not open windows, start work or show refusal feedback.
  Verify V05.
- [x] Remove this slice's duplicated cross-feature guards while keeping narrow
  adapters and explicit construction/overlay/shutdown order. Record actual
  callback/shortcut/bare cases, red/green and all changed-code inspections.

**Verification V05:**

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestWindowCommandAdmissionMatrix|TestWindowMenu.*|TestCompare.*|TestCommandAdmissionModalOwnership|TestCommandAdmissionBusyCopy)$'`

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestCommandAdmissionRoutes$/^windows$'`

Do not infer mounted UI solely from an object's Visible flag. Newly discovered
behavior differences outside D4-D7 require a concrete D1 scenario decision.

## Comments

2026-09-27: Reconciled the checklist with commit `9dc3a81` and its retained
verification record. Bare Window/Help entries, Show-versus-toggle semantics and modal/busy regressions passed.
