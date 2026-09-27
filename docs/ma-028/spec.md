# MA-028: shared command admission

Status: ready-for-human (Linux reset and Windows prompt-focus repaired; scoped Linux/Windows native checks passed; physical/native desktop acceptance open)
Date: 2026-09-27
Source: `/to-spec MA-028`, synthesizing the accepted D1-D7 interview decisions.
Inspected revision: `770052498a8f4fb3ca7bb8be02448287b40ccecb`.

## Completion status

- [x] Tickets 01-09: implementation and available local deterministic
  verification completed in `9dc3a81`; all 46 ticket checklist items are done.
- [x] Record final Make/race, Windows cross-vet and 68-file GoLand evidence in
  the [verification record](../command-admission-verification-2026-09-27.md).
- [x] Record the four Linux native scenario groups with OS-injected XTEST input,
  real menus/clipboard and busy/modal native close at `6db8d73` in the
  [Linux evidence](../command-admission-linux-qualification-2026-09-27.md).
  This is not physical-keyboard operator evidence or a waiver of AC10.
- [x] Fix the subsequently confirmed Linux native Escape/maximize reset defect.
  Actual native geometry/state checks pass repeatedly, including fixed-size
  controls, alongside a red/green boundary regression, three-file GoLand
  inspections and fresh full `make verify`. See ticket 10 and the Linux record.
- [x] Run the case-insensitive export regression on actual FAT16 storage at
  `75fd69e`: one ordinary pass and five race passes, no skips. Evidence and
  cleanup are retained in the Linux qualification record.
- [x] Add and locally verify focused native CI selection for the case-alias
  export and macOS Copy-menu guards. Command-boundary/workflow tests pass,
  deliberate omissions fail, and the real runner passes FAT16 while rejecting
  an ext4 skip. The isolated guards now also execute and pass in native Windows
  amd64 and macOS arm64/amd64 CI, including the AppKit assertion on both Macs.
- [x] Pass fresh CI, CodeQL and Codex code/security review at `4674cca`; inspect
  Qodana's post-suppression SARIF with zero results. Exact evidence and the
  final documentation-head gate are in the
  [review-loop record](../command-admission-pr-66-review-2026-09-27.md).
- [ ] Ticket 10: native Linux/Windows/macOS physical-input and full desktop
  qualification. Completed preparation is checked
  separately in [ticket 10](issues/10-native-qualification.md).

The problem statement and contract below describe the accepted pre-implementation
specification. MA-028 remains open until ticket 10 is accepted.

## Problem Statement

PicFetch users can reach the same action through a menu, a shortcut, a plain
key, a feature callback or an open/drop request. Those routes independently
decide whether the action can run. As new views are added, one route can remain
enabled or act on covered content while another correctly refuses it.

The accepted design identifies four corrections: ordinary commands must not
act beneath a modal prompt; editing shortcuts must reach the focused text
field; menus must reflect commands blocked by a pending image-region copy;
and an unavailable command must not discard an idle image-region selection.
For example, Save Changes currently can cancel Copy Selection before discovering
that there is no rotation to save.

Some differences are intentional. G toggles Grid View, while its menu action
shows it idempotently. Comparison disables Open in the menu but explains the
refusal when a user attempts an open through another route. Centralization
must preserve these distinctions and the subject each action operates on.

## Solution

Give every application-wide command one shared admission decision, based on
the current interaction and the action's actual intent. Menus present that
decision without changing the interaction; invocation checks it again before
performing any required yielding and then executing the existing action.

Preserve feature ownership, established refusal feedback, navigation and
asynchronous operation semantics. Migrate every applicable route in all six
command families. A pilot covering only some commands is not completion of
MA-028. The accepted behavior corrections are D4-D7 below; newly discovered
discrepancies require a concrete scenario and a separate user decision.

## User Stories

1. As a user, I want menus and shortcuts to respect the same interaction rules,
   so that changing how I invoke a command does not bypass a restriction.
2. As a user, I want unavailable commands disabled in menus, so that I can see
   which actions are possible now.
3. As a user, I want opening or refreshing a menu to preserve my work, so that
   inspecting availability never cancels a selection or starts an operation.
4. As a user, I want actions to check current availability when invoked, so that
   a menu drawn earlier cannot authorize an action in a changed view.
5. As a user confirming deletion, I want ordinary commands blocked underneath
   the confirmation, so that the pending choice remains the active interaction.
6. As a user choosing export options, I want the export prompt to own input,
   so that shortcuts cannot unexpectedly act on the image or Grid behind it.
7. As a user in a dialog, I want its controls and confirmation/cancel actions to
   work, so that command isolation does not prevent completing the interaction.
8. As a user, I want window close available during prompts and pending region
   copies, so that these interactions do not trap me in the application.
9. As a user selecting an application menu item, I want that item to work when
   otherwise available, so that the open menu is not mistaken for a dialog.
10. As a user editing text, I want Copy to copy the selected text, so that the
    clipboard does not unexpectedly receive an underlying image or file list.
11. As a user editing text, I want Select All to select the field's text, so that
    the underlying Grid selection remains unchanged.
12. As a desktop user, I want editing shortcuts to work through native menu
    accelerators, so that an application binding does not swallow text editing.
13. As a user explicitly choosing an image command, I want it to retain its
    image intent subject to admission, so that focus alone does not turn it
    into an unrelated text action.
14. As a user in Copy Selection mode, I want Copy to use my image-region
    selection first, so that it does not copy the entire displayed image.
15. As a Grid user, I want Copy to use Grid file targets when no image-region
    selection owns Copy, so that batch copy retains its existing meaning.
16. As a single-image viewer user, I want Copy to use the displayed image,
    so that it does not capture the requested image before its pixels arrive.
17. As a user with an idle image-region selection, I want an unavailable Save
    Changes command to leave it intact, so that a no-op does not destroy work.
18. As a user leaving Copy Selection through an available command, I want the
    selection cancelled before the command acts when required, so that the
    old interaction cannot overlap the new action.
19. As a user copying an image region, I want conflicting menu commands disabled
    while copying and restored afterwards, so that availability follows progress.
20. As a user attempting a blocked shortcut during region copying, I want the
    established refusal feedback, so that I understand why it did not run.
21. As a user copying an entire image or Grid files, I want the existing clipboard
    serialization without unrelated region-copy restrictions, so that a busy
    clipboard is not mistaken for a different interaction mode.
22. As a user adjusting an image-region selection, I want zoom, pan, repeated
    activation and selection-local keys to keep their established behavior,
    so that sharing command policy does not change selection geometry controls.
23. As a user navigating from the EXIF window, I want idle Copy Selection to
    yield before navigation and busy copying to refuse it, so that the existing
    distinction from selection-owned typed navigation remains useful.
24. As a keyboard user, I want G/P to remain toggles and menu Show actions to
    remain idempotent, so that each input retains its intended meaning.
25. As a Grid user, I want its ordinary key ownership preserved, so that P does
    not acquire the behavior of an explicit Picture-frame menu action.
26. As a comparison user, I want Help available and attempted opens explained
    and discarded, so that comparison preserves its deliberate exceptions.
27. As a user opening files or Favorites, I want menus, numbered shortcuts,
    direct callbacks, drag/drop and OS-open delivery to respect admission,
    so that no alternate entry bypasses the active interaction.
28. As a user saving, exporting, deleting, revealing or setting wallpaper, I want
    the existing captured subject retained, so that later navigation does not
    silently redirect an action to another file or image.
29. As a user entering Grid, Picture-frame mode, comparison, Explorer, Location
    Map, mosaic, EXIF, Settings or Help, I want consistent command admission,
    so that a direct callback cannot open a disallowed surface.
30. As a user visiting images or clusters from Location Map, I want retained
    visit restrictions to survive surface changes, so that a hidden map is not
    mistaken for the absence of a map visit.
31. As a user browsing cohorts, search results or duplicate variants, I want
    sort, duplicate and search restrictions preserved, so that alternate inputs
    cannot invalidate the visit's established rules.
32. As a user pressing Escape, I want the established return/cancel order,
    so that introducing shared admission does not change where I return.
33. As a user waiting for a chooser or feature setup, I want admission and
    request identity checked again at continuation, so that an obsolete result
    cannot start a now-disallowed action.
34. As a user whose file write has completed, I want the application to reconcile
    its effects even after navigation or a new prompt, so that caches and file
    information reflect the committed change.
35. As a maintainer adding a restricted visit, I want its command rules expressed
    through shared policy and context, so that each input adapter does not need
    another independent set of admission predicates.

## Implementation Decisions

### Settled decisions

These decisions are accepted, not questions to reopen during ticket planning.

| Decision | Contract |
| --- | --- |
| D1: intended behavior | Preserve deliberate route differences and existing feedback. Present newly discovered inconsistencies as concrete scenarios before changing their user-visible behavior. |
| D2: complete migration | Cover every application-wide family across its applicable menus, shortcuts, plain keys, direct user-action entries and open/drop delivery. Migrate family by family. |
| D3: ownership | A pure module private to root UI decides admission, refusal, required yielding and target kind from captured facts. Handlers capture payloads and execute effects; features retain their state and validity checks. |
| D4: modal ownership | Delete confirmation, export prompts and actual dialogs block unrelated main-window commands. Their own controls and window close remain usable. An open application menu is a different input owner. |
| D5: editing ownership | Copy and Select All editing shortcuts reach the focused text field, including accelerator delivery, without image/Grid effects. Explicit image commands keep their image intent. |
| D6: busy presentation | Commands refused during a pending image-region copy are disabled in menus until available. Preserve shortcut/direct refusal feedback and window close; ordinary clipboard work is a separate fact. |
| D7: availability before yielding | A recognized unavailable command leaves idle Copy Selection intact. An admitted command requiring a yield cancels the mode before performing its action. |

### Policy and execution boundary

The immutable context is captured on UI from existing owners. It distinguishes
the visible surface, retained browsing visit, input owner, command intent and
route, operation state, substantive capabilities and prospective target kind.
Examples of distinct capabilities include whether Save Changes has savable
content and whether export can start. Do not encode all of these observations
as a single active-mode flag or confuse requested and displayed images.

The decision must distinguish refusal that preserves the interaction from an
allowed action requiring a yield, and identify the relevant target kind and
refusal reason. A boolean alone is insufficient. The pure module reads no
widgets, owns no mutable feature state, captures no pixels or file payloads,
performs no I/O, starts no work, cancels nothing and produces no feedback.

Root adapters perform observation, interpret input intent, apply required yields
and execute existing handlers. Every user-action entry consults current policy;
a previously queried decision or disabled menu item is not an execution guard.
Admission precedes yield effects, payload capture that starts an operation and
the operation itself. Feature-local validity, cancellation and identity checks
remain at their execution seams without becoming a second cross-feature policy.

Menus consume policy-derived availability in their existing value snapshot.
Labels, checked states, construction and native presentation remain menu
responsibilities. Favorites and Help menu entry points are part of the route
inventory, not exemptions merely because they are constructed by feature owners.
Availability updates must reach the native menu when relevant facts change,
including entry into and exit from busy region copying.

### Migration inventory

| Family | Commands and routes to enumerate before migration is accepted |
| --- | --- |
| Open, Close Files and Favorites | Open dialog/dropzone, drag/drop, OS-open delivery, Close Files, Favorite open/add/manage and numbered shortcuts; applicable menu and direct host entries. Retain chooser, scan, storage and confirmation lifetimes. |
| Clipboard, Copy Selection and Grid selection | Copy, Copy Path, region activation and Select All; menu actions, editing/native accelerators, registered shortcuts and bare user-action handlers. Retain target priority, region geometry and clipboard serialization. |
| File and image actions | Save Changes, export, Trash, reveal and wallpaper through menus, shortcuts and root-facing user callbacks. Prompt-owned controls and committed completions retain their distinct roles. |
| Window and feature entry | Viewer, Grid, Picture-frame mode, comparison, Explorer, Location Map, mosaic, EXIF, Settings, Help and other root-owned window entries; include applicable direct links and callbacks. Feature-local controls retain their owners. |
| Sort, duplicates and search | Sort selection/cycling, duplicate hiding/browsing and Find more like this, across applicable menu, plain/modified-key and direct routes. Preserve restricted visits and preparation lifetimes. |
| Navigation and presentation | Root-owned image navigation, rotation/reset, zoom, merge/info and Picture-frame settings, including applicable secondary-window callbacks. Feature-local interpretation remains with the feature. |

Planning must record each actual command, its intent, applicable entry routes,
observation/execution owners and proving tests. Mark non-applicable routes with
a reason. A family is complete only when every applicable route uses shared
admission and its duplicated cross-feature predicates have been removed.
The inventory is a migration/evidence artifact, not a runtime command catalogue.

### Deliberate differences and lifetimes

- Preserve G/P toggles versus idempotent menu Show actions, including Grid's
  ownership of the P key. Preserve comparison Help and Open refusal behavior.
- Outside modal/editor ownership, Copy selects an image-region selection,
  then Grid file targets, then displayed-image pixels. Repeated activation and
  an uncommitted rectangle retain their current feature behavior.
- Typed navigation in Copy Selection stays with that feature. EXIF-host
  navigation yields an idle selection and refuses while busy. Key 0 yields
  because it also resets rotation; zoom/pan exceptions and feature-local
  treatment of unowned keys remain. D7 applies to recognized commands.
- Visible Location Map and retained map visits are different facts. Preserve
  restrictions through map/image/cluster-Grid transitions and existing Explorer,
  search and duplicate-variant visits. Escape keeps its ordered return/cancel
  behavior, owned by the existing dispatchers/features.
- Recheck live admission at asynchronous preparation/delivery admission points
  alongside operation tokens and source identity. Search retains its captured
  collection generation and reference after setup; Explorer retains its entry
  revalidation semantics. No generic queue, replay or retargeting rule is added.
- Already committed disk effects still reconcile through existing file-work
  ownership. They are not new user commands and must not be discarded because
  a prompt or another view now blocks fresh entry. Preserve established pending
  delivery, cancellation, worker completion and shutdown behavior.

Exact Go types, signatures, file changes, migration slices and final test names
belong to implementation planning. Preserve explicit feature construction,
overlay composition and shutdown order. No new dependency, persistent schema
or public application API is selected by this specification.

## Testing Decisions

### Test boundaries

Use the existing root viewer harness as the primary integration boundary:
construct the real viewer, invoke actual menu callbacks, registered shortcuts,
plain-key dispatch and bare user-action entries, then observe outcomes and
settle work through the established queues/signals. Include drop, OS-open and
secondary-window host routes where applicable. A route through a generic
wrapper does not establish coverage of a bare handler.

Add table-driven tests at the pure decision boundary, covering independent
surface/visit/input-owner/busy/capability/intent/target distinctions and their
meaningful combinations. Menus retain presentation tests. These complement
integration tests; do not mock the policy in a test intended to prove that an
entry route enforces it, or use its output as the test's expected result.

The test boundaries above were presented as an optional expectations check
during synthesis; they are the selected default, consistent with the accepted
design's verification requirements. Native accelerator/focus qualification
remains separate from both the pure tests and Fyne's test-driver dispatch.

Assert effects and their absence: selected text, clipboard payload/type, Grid
targets, displayed-image identity, active selection, menu availability, prompt
ownership, refusal feedback and admitted work. For mounted UI, inspect the
actual container tree when visibility alone cannot establish presence. Use
existing OS stubs and temporary fixtures; wait for completion rather than sleep.

### Prior art and gaps to close

- Extend `TestWindowCommandAdmissionMatrix`. Its existing "direct command"
  route calls `RunCommand`; add actual bare-handler coverage and the broader
  family/route matrix rather than relying on that label.
- Retain window/action menu, comparison isolation and Open-refusal tests.
  `TestYieldingMenuCallbacksWrapsEveryField` substitutes callbacks; it does
  not prove actual command capability checks or native dispatch.
- Extend Copy Selection keyboard, repeated-activation, yield and busy-copy
  guards. Keep `TestClipboardBusyPreservesActiveOperationAcrossCopyRoutes`
  as the separate ordinary-clipboard serialization contract.
- Existing dialog key/rune guards and selected Favorites shortcut guards are
  useful examples, not proof that all commands respect modal ownership.
- `TestHandleDrop_CancelsAnyPendingDeleteConfirmation` and
  `TestOpenWithHandler_DeliveryClosesTheGridAndCancelsAPendingDelete` currently
  expect incoming files to cancel a delete prompt. D2's open/drop scope and
  D4's modal ownership supersede that expectation: cover refusal with the
  confirmation and collection intact, and retain ordinary admitted open/drop
  coverage separately. Do not carry those old assertions forward as invariants.
- Preserve Explorer, Find more like this, Location Map and Escape transition
  guards, chooser stale-result tests and committed-write reconciliation tests.
- Add focused-editor tests that verify positive text-shortcut delivery as well
  as absence of image/Grid effects. Calling an entry's shortcut method directly
  proves neither accelerator interception nor actual native menu behavior.

### Acceptance criteria and commands

The following are implementation acceptance gates, not results of this spec
task. The `TestCommandAdmission*` names are provisional verification targets;
planning may choose final names and update this map before implementation.
Every named criterion needs real assertions and observed intended failures
before its implementation is accepted. Missing tests, no matching tests and
skips cannot satisfy a gate. Use verbose output to retain the executed cases.

| ID | Observable criterion | Verification command |
| --- | --- | --- |
| AC1 | Pure decisions distinguish all context dimensions above, return refusal/yield/target outcomes and leave their input unchanged; identical input gives identical output without effects. | `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestCommandAdmissionPolicy$'` |
| AC2 | All six families enforce the shared contract through every applicable route, including actual callbacks, registered shortcuts, bare entries and open/drop delivery; deliberate route differences remain. | `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestCommandAdmissionRoutes|TestWindowCommandAdmissionMatrix)$'` |
| AC3 | Delete/export/Fyne-dialog ownership blocks unrelated underlying commands across routes, while owner controls and window close work; opening a menu does not block a legitimate selection from it. | `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestCommandAdmissionModalOwnership$'` |
| AC4 | Focused-editor Copy/Select All reach text without changing the underlying image/Grid; explicit image commands retain their intent subject to modal policy. | `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestCommandAdmissionTextEditing$'` |
| AC5 | Busy region copying disables exactly the refused menu commands and restores availability afterwards; shortcut/direct attempts retain feedback and close remains possible. Ordinary clipboard work retains its distinct admission. | `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestCommandAdmissionBusyCopy|TestClipboardBusyPreservesActiveOperationAcrossCopyRoutes)$'` |
| AC6 | An unavailable command, including Save Changes with no savable rotation, preserves idle selection. An admitted command requiring a yield cancels it before effects. Existing copy target, typed/EXIF navigation, reset and zoom/pan distinctions remain. | `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestCommandAdmissionYieldOrdering|TestCopySelection.*|TestStepImage.*)$'` |
| AC7 | Repeated availability queries never cancel selection, start workers/clipboard work or emit feedback; invocation reevaluates changed facts instead of trusting stale menu state. | `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestCommandAdmissionQueries$'` |
| AC8 | Comparison exceptions, retained browsing restrictions, key-versus-Show semantics and Escape return order survive the migration. | `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestCommandAdmissionVisits|TestWindowCommandAdmissionMatrix|TestCompare.*|TestVisualSimilarityExplorer|TestFindMoreLikeThis.*|TestLocationMap|TestEscapeUnwindsModesBeforeReset)$'` |
| AC9 | Admission changes during chooser/setup/preparation reject disallowed continuations without generic replay; stale identities remain rejected and committed writes still reconcile after navigation or modal entry. | `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestCommandAdmissionAsync|TestOpenChooser.*|TestOpenFileDialog.*|TestSaveChanges.*|TestExportCancellation.*|TestExportCommitted.*|TestFileMutationReconciliation.*)$'` |
| AC10 | Actual desktop accelerator dispatch satisfies AC4, including menu interception, on Linux, Windows and macOS; native menu enablement/activation reflects AC3/AC5/AC7. | `make run` on each native platform, followed by the recorded qualification procedure below; launch success alone is not a pass. |
| AC11 | Full migration has no independent cross-feature admission policy left in input adapters; a newly restricted visit requires shared context/policy changes rather than predicates in each adapter. | `git diff <implementation-base> -- internal/ui` for lead structural review against the complete command/route inventory, plus the AC2 command. Replace the base with the recorded starting commit. |
| AC12 | Formatting, generated inputs/notices, vet, build, exact test exclusions, shard assignments and the complete required suite pass. Changed code is inspected, including weak warnings, with findings disposed. | `make check-test-shards`, `make check-qodana-test-exclusions`, then `make verify`; retain GoLand inspection evidence separately. |

Before acceptance runs, list the new guards with
`go test -tags no_emoji,nodynamic ./internal/ui -list '^TestCommandAdmission'`
and compare the output with the finalized AC map. Record the executed cases
for every command/route inventory row. AC11 needs lead judgment; a text search
or green test command alone cannot prove architectural ownership.

### Native qualification and final evidence

For AC10, launch the tested revision with `make run` on each platform and retain
OS/architecture, Fyne version, revision, input route and observed outcomes:

1. With images loaded, focus a naming/editing field and select distinctive text.
   Use physical platform Copy and Select All shortcuts. Verify the copied text
   and selected field content, with unchanged underlying Grid/image state.
   Repeat where application menu accelerators would otherwise intercept them.
2. Separately invoke explicit image menu actions in admitted contexts and while
   a dialog owns input. Verify their image intent and modal refusal respectively.
3. Exercise actual application-menu selection and native enablement across modal
   and busy-copy entry/exit. Use a reproducibly held copy through an instance-owned
   qualification seam if needed; do not rely on timing a fast real clipboard.
4. Verify disabled items cannot bypass invocation checks, legitimate menu actions
   work, and native window close remains usable. Retain reproducible input steps
   and observable results; a source trace or software test-driver pass is not
   platform runtime evidence.

The pinned Fyne v2.8.0 GLFW source tries matching main-menu shortcuts before
focused-widget and canvas shortcuts, and its matching menu-action path does
not inspect Disabled. This source observation explains the required coverage;
it does not qualify every native platform. A solution must actually deliver
editing shortcuts, not merely suppress the image command.

Plan native evidence collection before implementation. A platform unavailable
to the implementer remains explicitly unverified; it cannot be marked passed
by an unrelated native test suite. Record any test harness added for this
purpose and its exact invocation in the implementation evidence.

Run focused regressions during migration and the complete gate at handoff via
the Makefile's native Linux/amd64 Docker path. Add exact Qodana exclusions for
new test files and shard assignments for new root-UI top-level tests. Inspect
every changed code file with GoLand, including weak warnings; re-run affected
inspections after fixes and record revision, scope, findings and dispositions.
Incomplete inspections and native runs remain unverified. Apply the repository's
current Qodana CI requirements; do not restore the retired paused-CI waiver.

## Out of Scope

- MA-029 browsing-state ownership, MA-030 collection transitions and MA-033
  launch policy, or other adjacent architectural proposals.
- A runtime command catalogue, automatic feature registry, global application
  store, generic command queue or automatic retargeting mechanism.
- Moving feature state, cameras, workers, local key interpretation or Escape's
  ordered return behavior into the admission module.
- Rewriting decoder, clipboard, file-write, Favorite storage or worker lifetimes,
  or changing established subject capture and committed-effect reconciliation.
- Inventing new user-visible behavior beyond D4-D7 without the D1 scenario
  decision, replacing native shortcut decoding, or selecting a dependency upgrade.
- Implementing the refactor, creating implementation tickets, committing,
  pushing, running a PR review loop or publishing a release in this spec task.

## Further Notes

The accepted [design record](../command-admission.md),
[ownership ADR](../adr/0003-shared-command-admission.md) and
[interview](interview.md) are the source decisions. Use the
[domain vocabulary](../../CONTEXT.md), particularly Grid selection,
image-region selection, Copy Selection mode, requested image and displayed image.

The honest limit: shared admission removes duplicated cross-feature decisions;
it does not make action validity permanent or replace feature-owned state.
Different intents and routes can still have different allowed behavior.
Platform evidence remains necessary even after deterministic tests pass.

This request publishes a specification only. Implementation should take the
Deep route because the migration spans feature adapters and native input on
shipped platforms. The next artifact is a family-by-family implementation plan
or dependency-ordered tickets, with exact contracts, coverage and native evidence
tasks. The design frontier is settled; those implementation details are not
reasons to repeat the interview.

Reconnaissance used the existing design and targeted production/test reads.
The inspected revision changes only design/backlog documentation relative to
the interview's closing revision, so its application-source evidence remains
applicable. No application tests, native experiments or GoLand code inspections
were run for this documentation-only synthesis. Acceptance commands above are
future gates, not claimed results.

Delegation record: one read-only scout examined existing test assertions while
the lead checked production adapters and wrote the spec. G1-G5: a bounded
question, source-citation verification, no writes/shared-file conflict, smaller
test-only context and no duplicated lead test reading. Scripted name searches
preceded contextual reading; no spec decision, review or implementation was
delegated. Budget/actual: one scout / one scout; no implementation or full-suite
runs in this task.
