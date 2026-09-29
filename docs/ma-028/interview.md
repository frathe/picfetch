# MA-028: shared command admission

Status: resolved
Date: 2026-09-26
Resolved: 2026-09-27
Source: `/grill-with-docs MA-028`
Inspected revision: `1adc477` (working tree initially clean).
Closing revision check: `6c0db30`; both commits have tree
`de996854edc661ba403fae8486d6d7680d662f90`, verified with `git rev-parse` and an
empty `git diff --name-status 1adc477 6c0db30`.

## Purpose

Stress-test [MA-028](../../finished_refactorings/2026-09-29-needs-refactoring.md#ma-028), establish its scope
and behavior contract, and retain the decisions before implementation planning.
The user accepted all seven recommendations across two rounds. The tracked
[design record](../command-admission.md) consolidates those decisions.
This file retains interview evidence; implementation has not started.

## Established facts

- The backlog proposes a pure decision module, initially private to root UI,
  taking a command and a captured context. Effects, feature-owned state,
  native input decoding and asynchronous operation-token checks stay with
  their existing owners. MA-029/030 are separate proposals.
- `internal/ui/menu.go:75` wraps menu callbacks in several admission/yield
  policies; `internal/ui/shortcuts.go:99` separately gates registered shortcuts.
  `internal/ui/keys.go:107` has its own ordered input dispatch.
- `internal/ui/menus/menus.go:451` independently computes disabled states.
  Comparison disables the Open menu item, while its shortcut is admitted
  far enough for `openFileDialog` to show the comparison refusal. Equal policy
  does not require identical presentation or effects on every input route.
- `internal/ui/batch.go:79` chooses the copy subject in this order:
  image-region selection, Grid selection, displayed image. Clipboard commands
  bypass the ordinary shortcut wrapper intentionally.
- `internal/ui/copyselection.go:97` cancels idle Copy Selection before another
  command proceeds, but refuses with a toast while a region copy is busy.
  The pure menu query must never perform that cancellation or show the toast.
- `internal/ui/windowmenu_test.go:20` explicitly distinguishes Grid/Picture-frame
  key toggles from idempotent menu commands. Grid also consumes the P key while
  the menu may switch from Grid to Picture-frame mode.
- `internal/ui/copyselection_yield_test.go:17` asserts that direct navigation
  from the EXIF window cancels idle Copy Selection and advances; its busy case
  at line 43 asserts refusal with feedback. These route distinctions must be
  represented explicitly if retained.
- `internal/ui/menu.go:171` captures visible Location Map and retained visits
  as different facts, despite the current `LocationMapActive`/`CohortActive`
  names. `internal/ui/browsing.go:12` also restricts duplicate operations for
  retained Location Map, Explorer cohort and search visits.
- `internal/ui/visualsearch.go:53` rechecks collection generation and reference
  after asynchronous setup. `internal/ui/openfiles.go:33` rechecks the chooser
  token and comparison before delivery. An admission snapshot cannot replace
  those checks.

These are code observations, not runtime verification. No tests have been run
for this design interview.

## Existing regression anchors

The scout identified these additional assertions for later scenario rounds:

- `TestYieldingMenuCallbacksWrapsEveryField` (`menu_yield_test.go:16`) covers
  idle yielding through substituted callbacks, with deliberate copy/zoom
  exemptions; it does not establish complete busy/comparison behavior.
- `TestCopySelectionKeyboard` (`copyselection_lifecycle_test.go:19`) suppresses
  typed image navigation during Copy Selection. Busy-region-copy coverage is
  in `TestCopySelectionBusyBlocksOtherCommands` (line 385); actual window close
  remains available. Shared clipboard admission is independently covered by
  `TestClipboardBusyPreservesActiveOperationAcrossCopyRoutes`
  (`clipboardwork_test.go:97`).
- `TestEscapeUnwindsModesBeforeReset` (`keys_test.go:13`) protects the layered
  return order. Fyne dialog key/rune isolation has separate guards in that file.
- `TestCompareOpenRefusal_DropDialogShortcutAndOpenWithAreDiscarded`
  (`compare_test.go:1154`) covers the explicit Open refusal. Comparison menu,
  shortcut and covered-input isolation have separate tests in the same file.
- `TestVisualSimilarityExplorer/keyboard_entry` and `/replacement_admission`
  (`explorer_test.go:468`, line 513) distinguish Grid search ownership and
  admission during replacement scan/sort.
- `TestFindMoreLikeThisInitialAdmission/location-map`
  (`visualsearch_test.go:47`) blocks search throughout retained Location Map
  visits. `TestLocationMap/ranked_search_entry` (`locationmap_test.go:171`)
  permits the opposite transition, retiring search and using the collection.
- `TestLocationMap` separately covers Copy Selection/navigation (line 67),
  duplicate shortcuts (line 130), clipboard shortcuts (line 203), and direct
  map-image visit admission (line 756).

Paths above are relative to `internal/ui`. These source assertions are useful
anchors, not evidence that the suite currently passes or exhaustively covers
every command, route and simultaneous state. In the window matrix, the route
named "direct command" uses `RunCommand`; it is not a bare handler call.

## Accepted decisions

The user answered "go with defaults" to Q1-Q3 on 2026-09-26.

- **D1 / Q1: Behavior contract.** Preserve intended interactions and deliberate
  route differences. Present each suspected inconsistency as a concrete scenario
  for an explicit decision before changing it; existing predicates/tests alone
  do not make accidental behavior a requirement.
- **D2 / Q2: Completion scope.** Migrate all application-wide command families
  across their applicable menu, shortcut, plain-key and direct-entry routes,
  family by family. Include applicable drop/open delivery. Feature-local key
  interpretation and Escape's existing return order stay with their owners;
  browsing-state ownership remains MA-029.
- **D3 / Q3: Policy responsibility.** Use a pure decision module private to root
  UI. Captured facts produce admission, refusal, required yielding and target
  kind. Feature adapters supply capabilities; handlers capture actual files or
  pixels and execute effects. Menus query without effects and retain labels and
  checked state. Feature-local validity checks, operation tokens and freshness
  checks remain necessary.
- **D4 / Q4: Modal ownership (2026-09-27).** Ordinary main-window commands are
  blocked consistently across routes while a delete confirmation or export
  prompt owns interaction. The prompt's own controls and window close remain
  available; genuine dialogs retain their input ownership.
- **D5 / Q5: Text editing (2026-09-27).** Copy and Select All editing shortcuts
  reach the focused text field without affecting an underlying image or Grid,
  including when native/menu accelerator dispatch would otherwise intercept
  them. Explicit image commands retain their image intent.
- **D6 / Q6: Busy presentation (2026-09-27).** Menu commands refused during a
  pending region copy are disabled until available again. Existing refusal
  feedback for attempted shortcut/direct invocation and window close remain.
- **D7 / Q7: Availability before yielding (2026-09-27).** A recognized command
  that cannot run leaves idle Copy Selection intact. An admitted command that
  needs it to yield cancels the mode before performing its action.

The user answered "go with defaults" to Q4-Q7 on 2026-09-27. All presented
decisions are accepted and the design frontier is empty. Code signatures,
migration tasks and new test names belong to implementation planning, within
this contract. Any newly discovered user-visible discrepancy still follows D1.
No application code or dependencies were changed and no commit was requested.

## Round one (resolved)

The three root recommendations below were accepted together. Their dependent
questions were addressed in round two and consolidated in the tracked design.

### Q1: Behavior contract

Should the refactor preserve every current route difference, or preserve the
intended interactions while correcting individually identified inconsistencies?

Recommendation: preserve intended behavior and known deliberate exceptions;
bring each suspected inconsistency back as a concrete scenario for an explicit
decision before changing it. Neither a test nor an existing predicate alone
establishes that an accidental behavior is desired.

Depends-on-Q1 frontier: modal/focused-input behavior, refusal feedback,
busy-copy menu presentation, and any newly established route discrepancies.

### Q2: Completion scope

Does MA-028 finish with a bounded pilot, or after all application-wide command
families use shared admission across their applicable routes?

Recommendation: complete all application-wide families, migrated in small
slices: Open/Close/Favorites; file/image actions and clipboard; window/feature
entry; sort/duplicates/search. Include menus, registered shortcuts, plain-key
commands, direct user-action handlers and applicable drop/open delivery.
Feature-local input interpretation and the existing Escape return order stay
with their owners; browsing-state ownership remains MA-029.

Depends-on-Q2 frontier: exact command inventory, exceptional entry routes,
migration slices and observable completion criteria.

### Q3: Policy responsibility

Should the shared module only answer whether a command is allowed, also choose
its subject and required preparation, or own executing transitions as well?

Recommendation: a pure decision over captured facts that identifies admission,
refusal, required yielding and the target kind. Feature adapters supply facts
such as export capability; handlers capture actual files/pixels and perform
effects. Menu availability consumes the decision without running effects;
checked state and labels remain menu presentation. Feature-local validity
checks and operation tokens remain necessary.

Depends-on-Q3 frontier: precise command/context/decision vocabulary, query
versus invocation semantics, direct-entry coverage, and revalidation after
asynchronous preparation.

## Round two (resolved)

Q4-Q7 depend on D1-D3 and were accepted together on 2026-09-27.

### Q4: Modal command ownership

While the delete confirmation or export prompt is visible, should ordinary
main-window actions such as Open, Rotate, Copy and changing views be blocked
consistently across menus, shortcuts and direct user-action handlers?

Recommendation: yes. The prompt owns interaction until it is completed or
dismissed; its controls and window close remain usable. Apply the same
principle to actual Fyne dialogs. Merely opening an application menu is not
a reason to reject the command selected from that menu. Comparison remains
distinct from a modal prompt and retains its explicit Help/Open exceptions.

Evidence: `internal/ui/keys.go:127` establishes overlay/deletion/export priority
for ordinary keys, but `shortcuts.go:99` and `menu.go:106` do not uniformly apply
those facts. `export.go:96` and `batch.go:36` separately prevent the two cards
from opening over one another. These are source observations; general modal
command isolation is not established by the existing tests.

### Q5: Focused text editing

When a text entry owns focus, should Copy and Select All operate on that text,
even if the corresponding image/Grid command could otherwise run?

Recommendation: yes. Editing shortcuts belong to the focused text control;
they must not copy an underlying photo or select the underlying Grid. Explicit
image commands remain image commands and are subject to Q4 when a dialog is
open. Input adapters must preserve this distinction, including menu accelerator
delivery. Blocking an image action without delivering the text shortcut is
insufficient.

Evidence: the pinned local Fyne v2.8.0 source in
`$GOMODCACHE/fyne.io/fyne/v2@v2.8.0/internal/driver/glfw/window.go:861`
tries matching main-menu accelerators before focused-widget/canvas shortcuts.
Its `triggerMenuShortcut` at line 886 invokes matching actions without consulting
the item's Disabled flag. This establishes a path that must be accounted for,
not a native-platform runtime reproduction. Existing bare ShortcutHandler tests
do not exercise that driver ordering. Native menu handling and focused text must
be verified separately from policy-unit tests.

### Q6: Busy region-copy menu presentation

While an image-region copy is running, should commands that will refuse to run
also appear disabled in the menus?

Recommendation: yes. Show their current unavailability, preserve existing
refusal feedback for attempted shortcut/direct invocation, and keep window
close available. Do not conflate a busy region copy with an ordinary clipboard
worker, or disable unrelated commands just because any worker is active.

Evidence: `internal/ui/copyselection.go:97` refuses yielding while Busy, but
`menu.go:171` and `menus/menus.go:451` do not pass/apply a busy-region-copy fact
to the ordinary menu availability matrix. Querying that matrix must not itself
yield or display a refusal.

### Q7: Unavailable commands and idle Copy Selection

If an idle image-region selection exists and Save Changes has nothing to save,
should pressing its shortcut leave the selection intact?

Recommendation: yes. Check a recognized command's substantive availability
before yielding; an unavailable command should not discard the selection.
Commands that can run after yielding still cancel it before executing. This
does not change feature-local handling of unowned keys or the established
typed-navigation versus EXIF-navigation distinction.

Evidence: `internal/ui/shortcuts.go:99` yields before invoking the Save binding
at line 235; `internal/ui/save.go:47` then returns when there is no savable
rotation. The menu wrapper at `menu.go:113` has the same ordering. This makes
selection cancellation possible even though no save follows. This is a source
trace, not a newly executed regression.

### Consequences already constrained by D1-D3

- Retain operation-specific asynchronous semantics. Search binds the original
  generation/reference (`visualsearch.go:53`), while Explorer's setup callback
  re-enters its existing entry point (`explorer.go:19`). A shared policy is not
  permission to replace these with one generic queue or retargeting rule.
- Recheck the live policy at asynchronous admission points alongside existing
  token/identity checks. Completion of already committed disk effects still
  reconciles through existing file-work ownership; it is not a new user command.
- The migration inventory must distinguish actual invocation intent (toggle,
  show, text edit, file/image action), route, visible surface and retained visit.
  Avoid deriving all of them from a single active-mode flag.
- Integration coverage must exercise actual menu callbacks, registered
  shortcuts and unwrapped user-action entry points. The existing matrix's
  `RunCommand` route alone does not prove bare handler admission.

## Reconnaissance and delegation

The lead read the architecture map, working agreement, glossary and relevant
production adapters. One read-only scout checks existing admission tests in
parallel; architecture and specification decisions remain with the lead/user.
Scout scope is factual test assertions with file/line evidence, with no writes,
review or implementation. Round-one budget: one scout; actual: one. Round two
reuses that scout for the bounded modal-input question while the lead inspects
asynchronous continuation semantics; no new agent is spawned.
The closing round reuses the same scout to check whether the move from
`1adc477` to `6c0db30` invalidated the recorded source evidence. No code review
or design decision is delegated.

The bounded question fits a cold-start prompt, touches no files, and explores
test context the lead has not already read. Its cited assertions are checkable
by targeted source reads. This is search fan-out, not delegated design.
