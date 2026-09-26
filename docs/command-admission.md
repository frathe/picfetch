# MA-028: shared command admission

Status: accepted design; implementation planning and verification pending.
Date: 2026-09-27

The `/grill-with-docs MA-028` interview established one shared decision module
for application-wide commands. The user accepted Q1-Q3 on 2026-09-26 and Q4-Q7
on 2026-09-27. All presented design decisions are settled. The
[interview record](../.scratch/ma-028/interview.md) retains source evidence and
the [ADR](adr/0003-shared-command-admission.md) records the ownership trade-off.
This document describes the intended change; it does not claim implementation
or passing tests.

## Accepted decisions

| Decision | Contract |
| --- | --- |
| D1: behavior | Preserve intended interactions and deliberate route differences. Bring newly discovered user-visible inconsistencies back as concrete scenarios before changing them. D4-D7 are the explicitly accepted corrections. |
| D2: scope | Migrate all application-wide command families across applicable menus, keys, shortcuts and direct user-action entries, family by family. Include drop and OS-open delivery. |
| D3: ownership | A pure module private to root UI decides admission, refusal, required yielding and target kind from captured facts. Existing handlers capture payloads and execute effects; features retain their state and validity checks. |
| D4: modal input | Delete confirmation, export prompts and actual dialogs own interaction. Ordinary main-window commands cannot act underneath them. The owner's controls and window close remain usable. |
| D5: text editing | Copy and Select All editing shortcuts reach the focused text field without affecting the underlying image or Grid, including through menu accelerator delivery. Explicit image commands retain their image intent. |
| D6: busy region copy | Commands refused during a pending region copy appear disabled in menus. Preserve existing refusal feedback for attempted shortcut/direct invocation and allow window close. |
| D7: yield ordering | Check a recognized command's availability before cancelling idle Copy Selection. An unavailable command preserves the selection; an admitted command cancels it when required before acting. |

## Responsibility and scope

Root UI remains the composition owner. The decision module reads an immutable
context captured on UI; it neither reads widgets nor performs I/O, starts work,
cancels a feature or shows feedback. Its facts distinguish:

- The visible surface from any retained browsing visit.
- The owner of input, including a modal prompt or focused text field.
- The command's intent and entry route, including show versus toggle.
- Relevant operation state and capabilities, including busy region copy versus
  ordinary clipboard work, and whether a save/export can actually start.
- The kind of subject the command would use.

Adapters gather those observations from their existing owners. Execution
captures actual file identities, selections or image pixels through the existing
paths. The module's decision must express both an allowed action requiring a
yield and a refusal that must leave the current interaction intact; a single
boolean cannot carry the contract.

Every user-action entry rechecks the shared decision. A prior menu query or a
disabled menu item is not an execution guard. Menus consume policy decisions
for availability without cancelling Copy Selection or producing feedback;
labels, checked states, item construction and native presentation remain menu
responsibilities. Feature-specific validity checks remain at their execution
seams and must not become an independent copy of the cross-feature policy.

The following inventory bounds implementation planning; the plan must enumerate
the actual commands and routes before declaring each family migrated.

| Family | Applicable routes and retained owners |
| --- | --- |
| Open, Close Files and Favorites | Menu items, numbered Favorite shortcuts, dropzone/open shortcuts, drag/drop, OS-open delivery, direct host entries; retain chooser/scan lifecycles and Favorite storage. |
| Clipboard, Copy Selection and Grid selection | Copy/Copy Path/region activation/Select All; include focused-editor dispatch and native accelerators; retain region geometry and clipboard workers. |
| Save, export, Trash, reveal and wallpaper | Menus, shortcuts, direct user-action callbacks; retain captured payloads, confirmation controls, worker tokens and committed-write reconciliation. |
| Window and feature entry | Viewer, Grid, Picture-frame, comparison, Explorer, Location Map, mosaic, EXIF, Settings, Help and other root-owned window entry; retain each feature's local controls and lifecycle. |
| Sort, duplicates and search | Menu actions, plain/modified keys and direct entry; retain restricted-visit rules, target selection and asynchronous setup/preparation. |
| Navigation and presentation | Root-owned image navigation, rotation/reset, zoom, merge/info and Picture-frame settings; feature-local key meaning stays with its owner. |

Escape keeps its existing ordered return/cancel behavior. Feature-local key
interpretation, native shortcut decoding, analysis workers, cameras and source
collection state remain with their existing owners. This effort does not
implement MA-029 browsing ownership, MA-030 collection transitions, MA-033
launch policy, a command catalogue, an automatic feature registry or a global
application store.

## Deliberate interaction differences

Shared admission preserves the actual intent of each route:

- G/P toggle Grid/Picture-frame; menu Show actions are idempotent. Grid's
  ordinary key ownership may consume P while the explicit menu action switches
  to Picture-frame mode.
- Comparison retains Help availability and its Open refusal. Disabling Open
  in the menu does not remove the refusal shown for an attempted shortcut,
  drop or direct open.
- Outside a focused editor or modal owner, Copy selects an image-region
  selection, then Grid file targets, then the displayed image, in that order.
  Repeated Copy Selection activation and an uncommitted rectangle retain their
  existing feature behavior.
- Typed navigation during Copy Selection stays with that feature. EXIF-host
  navigation yields an idle selection before advancing and refuses while busy.
  Key 0 yields because it also resets rotation; the existing zoom/pan exceptions
  and feature-local treatment of unowned keys remain.
- A visible Location Map and its retained image/cluster visits are different
  facts. Existing visit restrictions, including search and duplicate-command
  restrictions, survive map-to-image and map-to-Grid transitions.

The prompt's own confirmation/cancel actions are distinct from new unrelated
commands. A menu being open is likewise distinct from a dialog owning input;
selecting a legitimate command from a menu must still work. Preserve existing
refusal messages and feedback rules outside the accepted D4-D7 corrections.

## Admission across asynchronous work

Recheck current admission at asynchronous preparation/delivery points alongside
existing request tokens and identity checks. A captured decision is not lasting
permission to apply an obsolete result. Preserve each operation's established
target semantics: search checks the captured collection generation and reference
after setup, while Explorer re-enters its existing admission path.

This change introduces no generic command queue or automatic retargeting rule.
Already committed disk effects still reconcile through the existing file-work
path, even if the current view changes or a prompt opens. Those completions are
not new user commands; applying the new entry gate to them would lose effects
the application must account for.

## Migration and evidence required

Start with private root UI policy and adapters over current feature observations.
Migrate one family at a time. Remove duplicated admission predicates only when
all of that family's applicable routes use the shared decision. Keep construction,
overlay and shutdown composition explicit. The eventual implementation plan
chooses exact Go types, file changes, test names and migration slices.

Verification must establish:

1. Pure decision tests cover surface, retained visit, input owner, relevant busy
   state, capability, intent and target distinctions, including refusal without
   effects and admission requiring a yield.
2. Actual menu callbacks, registered shortcuts and unwrapped user-action entries
   enforce the same shared rules and their deliberate differences. Extend
   `TestWindowCommandAdmissionMatrix`; its current `RunCommand` route alone does
   not cover bare handlers.
3. Concrete regressions cover D4-D7: commands cannot act below a prompt; editor
   Copy/Select All reach the editor; busy-region menu availability changes and
   recovers; unavailable Save leaves idle Copy Selection intact. Repeated menu
   queries cause no cancellation, clipboard operation, worker or toast.
4. Existing comparison, Copy Selection, Explorer, search, Location Map and
   Escape guards remain meaningful. Asynchronous preparation retains stale-result
   rejection, captured subjects and committed-effect reconciliation.
5. A newly restricted visit needs policy/context work rather than independent
   admission predicates in each input adapter. This is a structural review
   criterion in addition to the behavior tests.

The pinned Fyne v2.8.0 GLFW driver checks matching main-menu shortcuts before
focused-widget and canvas-shortcut dispatch (`internal/driver/glfw/window.go`,
`triggersShortcut`). Its `triggerMenuShortcut` calls a matching action without
checking Disabled. Therefore a disabled item or a bare `fyne.ShortcutHandler`
test does not prove text-editing priority. Preserve the distinction between
explicit menu actions and editing shortcuts and verify the actual desktop
accelerator/focus routes. Do not solve this by silently dropping the editor's
shortcut. Platform-specific native menu behavior needs separate evidence;
source inspection is not a native runtime qualification.

Implementation must follow the repository's focused-regression, shard/exclusion,
GoLand inspection and final verification requirements. No application tests or
native experiments were run for this documentation-only interview. The inspected
commits `1adc477` and `6c0db30` have the same tree, so the intervening history
change did not invalidate the source evidence.
