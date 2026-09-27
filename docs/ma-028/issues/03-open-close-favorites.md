# 03: Admit Open, Close Files and Favorites consistently

**What to build:** Opening or closing a collection, or acting on Favorites,
respects the current interaction through every entry. Incoming files cannot
replace the collection or dismiss an active confirmation behind the user's back.

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

**Agent/model:** T0 Codex Lead, GPT-6 Astra, high reasoning. Root/Favorite
coordination and chooser delivery exceed a single-package delegated task.

**Context:** [MA-028 spec](../spec.md), D2-D4/D6-D7 and AC2/AC3/AC9; follow the
[execution plan](../../../finished_refactorings/2026-09-27-ma-028-command-admission.md).

- [x] Inventory/migrate Open menu, both open shortcuts, dropzone, drop, OS-open,
  restore-session user link, bare Open/Close Files and Favorite Host entries;
  include Favorite menu/index shortcuts, add/manage and dialog entry routes.
  Keep startup and already-owned dialog continuations explicitly classified.
  Verify V03 with real callbacks, bindings and bare entries.
- [x] Replace old drop/OS-open tests that expect a pending delete confirmation
  to be cancelled. Refusal now preserves that prompt and collection; ordinary
  admitted open still works. Prompt controls and native close remain usable.
  Comparison still discards attempted opens with established feedback while its
  Open menu remains disabled. Verify V03.
- [x] Menu queries do not yield or perform Favorite storage I/O. Dynamic Favorite
  rebuilds and native menu refresh retain the latest command availability.
  Numbered shortcuts resolve the current slot, then recheck admission. An
  unavailable command leaves idle Copy Selection intact. Verify V03.
- [x] Recheck current admission when a held native chooser delivers, alongside
  its request token. Preserve cancellation after newer opens, drops, reset and
  shutdown; do not replay refused OS/chooser deliveries later. Keep admitted scan
  work and existing Favorite capture/storage semantics with their owners.
  Verify V03 using held and out-of-order completions.
- [x] Remove superseded cross-feature checks only for migrated entries; classify
  Favorite naming/overwrite/removal controls as owned interactions. Record all
  route evidence, observed red/green, inspections and manifest updates.

**Verification V03:**

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestCommandAdmissionModalOwnership|TestCommandAdmissionAsync|TestOpenChooser.*|TestOpenFileDialog.*|TestOpenWith.*|TestHandleDrop.*|TestFavorite.*|TestCloseFiles.*|TestCompareOpenRefusal_DropDialogShortcutAndOpenWithAreDiscarded)$'`

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestCommandAdmissionRoutes$/^open$'`

Every new case must actually run; a generic runner alone is not entry coverage.

## Comments

2026-09-27: Reconciled the checklist with commit `9dc3a81` and its retained
verification record. Open/drop/OS-open/Favorites routes and held chooser delivery regressions passed.
