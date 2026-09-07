# Command admission matrix

Captured from 524fcf8 before any consolidation, 2026-09-07. Ticket 30
selects coverage of the existing command boundary. Feature methods retain their
state ownership; `RunCommand` is the direct command entry used by favorites.
Low-level `grid.Toggle` and slideshow methods are composition primitives, not
independent application command routers.

| State | Keyboard | Menu / direct command | Escape |
|---|---|---|---|
| Comparison over a selected grid | Comparison transforms only; F1 opens Help; Open shortcuts reach explicit refusal | Ordinary callbacks blocked; Help opens; Open is disabled, and an invoked/stale callback still refuses before choosing | Close comparison, preserve covered grid and selection |
| Copy Selection idle | Escape cancels; Enter copies rectangle; navigation remains owned; 1/+/- and modifier-only keys retain mode; 0 and other commands yield | Ordinary commands yield first; zoom and Copy commands own their exceptions | Cancel selection, preserve files and rotation |
| Copy Selection busy | Feature swallows keys | Ordinary yield refuses while copy pending | Busy copy keeps its ownership; closing window cancels work |
| Grid | Grid owns keys; G closes only without selection; V closes; P swallowed | Show Grid is idempotent; Show Viewer closes; Show Picture-frame can switch out of ordinary grid | Cancel drag, selection, search, variants, hide-duplicates, then close, in that order |
| Inspect | G/Escape reopen variants; V preserves inspect; picture-frame refused | Show Grid reopens variants; Show Viewer preserves inspect; picture-frame refused | Reopen variants before reset |
| Picture-frame | P toggles off; G refused; V exits; arrows retain slideshow behavior | Show Picture-frame is idempotent; Show Grid refused; Show Viewer exits | Exit frame before cancelling a scan/sort or resetting |
| Ordinary viewer | G/P enter their modes; normal navigation/actions | Equivalent admitted commands share viewer behavior | Cancel scan, then sort, then inspect; reset a loaded session; close an empty window |

Dispatch precedence above the normal viewer: Fyne canvas dialog, delete card,
export prompt, comparison, grid, Copy Selection. The ordered Escape regression
uses real reachable transitions through inspect, comparison over selected grid,
selection, variants, hide-duplicates, grid close, Copy Selection and frame exit
before resetting. Existing causal scan/sort and dialog tests cover their own
priority branches without fabricating overlapping worker state.

Coverage locators (repository relative):

- `internal/ui/keys.go`: handleKeyEvent; `keys_test.go`: TestEscapeUnwindsModesBeforeReset and canvas overlay guards.
- `internal/ui/windowmenu_test.go`: TestWindowCommandAdmissionMatrix (36 route/state cases), existing empty-file and menu-enabled-state guards.
- `internal/ui/compare_test.go`: TestCompareHelp_F1OpensManualWithoutLeavingComparison now exercises keyboard/menu/direct Help; TestCompareCommandEntryPoints_MenuAndFeatureCallbacksAreIgnored; TestCompareOpenRefusal_DropDialogShortcutAndOpenWithAreDiscarded; comparison shortcut, isolation and restoration guards.
- `internal/ui/openfiles_test.go`: TestOpenFileDialog admission cases exercise direct/menu/shortcut before native chooser.
- `internal/ui/copyselection_lifecycle_test.go`: TestCopySelectionCancelsBeforeOtherCommands and TestCopySelectionBusyBlocksOtherCommands; `menu_yield_test.go`: TestYieldingMenuCallbacksWrapsEveryField.
- `internal/ui/grid/nav.go`: escape; feature nav/selection/dupes tests cover search, selection and filter ordering.
- `internal/ui/shortcuts.go`: wireGlobalShortcuts / yieldingShortcuts; built-in clipboard and SelectAll shortcut types retain their existing production registrations.

Decision: retain the current composition and wrappers. They already centralize
ordinary menu/shortcut yield decisions; the remaining checks distinguish
keyboard ownership, explicit show versus toggle, direct feature admission and
stale callbacks. A universal mode predicate would obscure these differences.
No chooser-thread change, application API or user-visible string is needed.

Verification: named inventory is in [phase6-inventory.txt](phase6-inventory.txt).
The expanded command/hide selection passes UI 5.864s and grid 1.149s. The full
native package command passes menus 0.524s and fails UI only on the previously
recorded macOS Copy Selection golden mismatch; [raw output](phase6-native-packages.txt)
is retained. [Negative overlays](phase6-negative.txt) reject blocking menu Help,
admitting picture-frame during inspect and losing selection on comparison
Escape. The shared canonical Linux make verify gate passes. No production
command-admission source changed for this matrix capture.

Final common gate: [make verify PASS](phase6-verify.txt), including all 667 root UI tests, canonical Linux/amd64 goldens and the full race suite. The native-only Copy Selection golden mismatch remains documented in [the macOS package output](phase6-native-packages.txt).
