# 03: Admit Open, Close Files and Favorites consistently

**What to build:** Opening or closing a collection, or acting on Favorites,
respects the current interaction through every entry. Incoming files cannot
replace the collection or dismiss an active confirmation behind the user's back.

**Blocked by:** 01: Establish shared admission through Save Changes.

**Status:** draft (publish as ready-for-agent after breakdown approval).

**Agent/model:** T0 Codex Lead, GPT-6 Astra, high reasoning. Root/Favorite
coordination and chooser delivery exceed a single-package delegated task.

**Context:** [MA-028 spec](../spec.md), D2-D4/D6-D7 and AC2/AC3/AC9; follow the
[execution plan](../../../plans/2026-09-27-ma-028-command-admission.md).

- [ ] Inventory/migrate Open menu, both open shortcuts, dropzone, drop, OS-open,
  restore-session user link, bare Open/Close Files and Favorite Host entries;
  include Favorite menu/index shortcuts, add/manage and dialog entry routes.
  Keep startup and already-owned dialog continuations explicitly classified.
  Verify V03 with real callbacks, bindings and bare entries.
- [ ] Replace old drop/OS-open tests that expect a pending delete confirmation
  to be cancelled. Refusal now preserves that prompt and collection; ordinary
  admitted open still works. Prompt controls and native close remain usable.
  Comparison still discards attempted opens with established feedback while its
  Open menu remains disabled. Verify V03.
- [ ] Menu queries do not yield or perform Favorite storage I/O. Dynamic Favorite
  rebuilds and native menu refresh retain the latest command availability.
  Numbered shortcuts resolve the current slot, then recheck admission. An
  unavailable command leaves idle Copy Selection intact. Verify V03.
- [ ] Recheck current admission when a held native chooser delivers, alongside
  its request token. Preserve cancellation after newer opens, drops, reset and
  shutdown; do not replay refused OS/chooser deliveries later. Keep admitted scan
  work and existing Favorite capture/storage semantics with their owners.
  Verify V03 using held and out-of-order completions.
- [ ] Remove superseded cross-feature checks only for migrated entries; classify
  Favorite naming/overwrite/removal controls as owned interactions. Record all
  route evidence, observed red/green, inspections and manifest updates.

**Verification V03:**

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestCommandAdmissionModalOwnership|TestCommandAdmissionAsync|TestOpenChooser.*|TestOpenFileDialog.*|TestOpenWith.*|TestHandleDrop.*|TestFavorite.*|TestCloseFiles.*|TestCompareOpenRefusal_DropDialogShortcutAndOpenWithAreDiscarded)$'`

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestCommandAdmissionRoutes$/^open$'`

Every new case must actually run; a generic runner alone is not entry coverage.
