# Copy Selection viewer, entry points, and pixel copy

Route: Deep

Deliverable: resolve Copy Selection tickets 03, 04, and 05 in dependency
order: compose the existing feature into the real viewer, expose it through
the Actions menu and `Alt+Shift+C`, then copy the stable selected image pixels
through the existing clipboard path.

Non-goals: the cross-feature cancellation matrix and keyboard-ownership work
owned by ticket 06; manuals, golden acceptance, and final feature-wide
documentation owned by ticket 07; source-file modification; a new clipboard
backend; completing unrelated unfinished parts of ticket 01.

Honest limit: ticket 01 is still marked claimed and its committed tracer lacks
several promised editing/visual behaviors. Ticket 05 directly requires its
missing `copyselection.PNG` contract, so that isolated prerequisite is included
here. The remaining ticket-01 gap is not silently treated as part of tickets
03-05.

## Decisions

| Decision | Resolution |
|---|---|
| Viewer selection coordinates | Build one `copyselection.View` from `zoom.Geometry()` and rotation-aware `displayedDimensions()`. |
| Geometry delivery | Defer `ViewChanged` through a per-viewer `fyne.Do` seam because zoom reports from layout. |
| Source stability | Capture the visible oriented raster frame, or the SVG vector/logical-size/rotation tuple, when mode starts. |
| Animation | Pause frame advancement through a viewer-owned gate while preserving the visible frame and resume when mode ends. |
| Worker lifecycle | Use a dedicated request lifecycle for staleness plus the existing `viewer.clipboard` completion signal; complete UI state through `fyne.DoAndWait`. |
| Availability | Build one positive menu-state value from decoded/not-loading/normal-viewer/no-modal facts; the action repeats the same guard. |
| Existing copy shortcuts | Preserve `ShortcutCopy` and default-modifier `Shift+C`; add only `Alt+Shift+C`. |

These decisions are fixed for this implementation.

## Acceptance criteria

1. Direct viewer activation starts only for a decoded image, places the
   feature at the specified paint depth, forwards zoom geometry/scroll, and
   restores the information overlay on end.
   Verify: `go test -race ./internal/ui/... -run 'TestCopySelection(Activation|InfoOverlay|ZoomPanResize|Cancel)$'`
2. Actions contains Copy image, Copy selection, Copy image path in that order;
   state, callback, and real `Alt+Shift+C` wiring agree without changing the
   existing C shortcuts.
   Verify: `go test -race ./internal/ui/menus ./internal/ui -run 'Test(ActionsMenu_CopySelection|CopySelectionAvailability|WireCopySelectionShortcut)$'`
3. `Copy selection` and `Copy to clipboard` exist in every locale and preserve
   English identity/arrow guards.
   Verify: `go test . -run 'TestTranslations_(EveryLocaleCoversEnglish|EnglishMapsEachKeyToItself|NoArrowFollowedByASpace)$' && go test ./internal/ui -run TestTranslationsHaveNoUnicodeArrows`
4. Literal raster/alpha/rotation/animation/SVG/RAW inputs yield the selected
   content pixels, independent of viewport geometry, and failure retains an
   editable selection while success ends it.
   Verify: `go test -race ./internal/ui/copyselection ./internal/ui -run 'TestCopySelection(Pixels|Transparency|Rotation|AnimatedFrame|SVG|RAWPreview|Busy|Success|EncodeFailure|ClipboardFailure)$'`
5. Each issue is marked resolved with its focused command output recorded.
6. The repository final gate passes once.
   Verify: `make fmt-check && go vet ./... && go build ./... && go test -timeout 20m -race ./...`

## Task graph

`T0 prerequisite -> T1 viewer composition -> T2 entry points -> T3 pixel worker -> T4 review/gate`

### T0 - Restore the promised PNG seam

Owner: T1 implementer subagent

Files: create `internal/ui/copyselection/png.go` and `png_test.go` only.

Contract: `PNG(image.Image, image.Rectangle) ([]byte, error)` returns a
zero-origin pixel-exact PNG and rejects nil/empty/out-of-source bounds without
adding a crop limit.

Test: literal colors and alpha over non-zero source bounds plus invalid cases.

Verify: `go test -race ./internal/ui/copyselection/...`

Delegation gate: G1 yes (one fixed function); G2 yes (one package command); G3
yes (two new files no Lead overlap); G4 yes (the package contract is smaller
than the viewer context); G5 yes (only the missing symbol was discovered, its
implementation context was not built). Rule S does not apply; Rule W is
respected.

Budget: 1 existing-agent follow-up; 1 Lead review round; full suite no.

### T1 - Compose the feature into the viewer

Owner: T0 inline

Files: `internal/ui/viewer.go`, `features.go`, `build.go`, create
`copyselection.go`, and focused `internal/ui` tests.

Contract: `viewer.regionCopy`, `startRegionCopy`, `finishRegionCopy`,
`cancelRegionCopy`, and one zoom/display-to-View adapter; overlay between info
and grid; layout callbacks deferred through a per-viewer seam.

Test: activation/no-op states, info restoration, overlay order, geometry
survival, scroll forwarding, and cancellation.

Verify: acceptance criterion 1, then `go test ./internal/ui`.

Budget: 1 existing-agent follow-up for the exhaustive menu test/translation
surface; 1 Lead review round; full suite no.

### T2 - Add menu and shortcut entry points

Owner: T0 inline (user-visible strings and cross-package state are
non-delegable under the repository workflow).

Files: `internal/ui/menus/menus.go`, its tests, `menu.go`, `actionmenu.go`,
`shortcuts.go`, focused viewer tests, and `translations/*.json`.

Contract: `menus.Callbacks.CopySelection`, one `ActionItems` accessor, one
positive `menus.State` value built in `viewer.menuState`, viewer action
`copyActionsSelection`, and `wireCopySelectionShortcut` with Alt+Shift+C.

Test: composition/order/accelerator/callback, availability matrix, menu and
real shortcut reach the same action, existing C bindings remain unchanged.

Verify: acceptance criteria 2 and 3.

Budget: 1 existing-agent follow-up for isolated source-format tests; 1 Lead
review round; full suite no.

### T3 - Copy the stable selected pixels

Owner: T0 inline

Files: viewer-side Copy Selection glue, animation/display glue only as needed,
`internal/ui/harness_test.go`, and focused pixel/lifecycle tests.

Contract: source captured at entry; raster/RAW use the oriented displayed
frame; SVG rasterizes at logical dimensions under the existing cap; animation
pauses/resumes; worker uses `copyselection.PNG`, `clipboard.CopyImage`,
staleness, `viewer.clipboard`, and a final Fyne UI update.

Test: literal pixels/alpha/rotation, frozen animation frame, SVG/RAW, busy,
success, and dispatch failure/retry.

Verify: acceptance criterion 4.

Budget: 0 implementation spawns; 1 Lead review round; full suite no.

### T4 - Review, issue records, and final gate

Owner: T0 inline; review and final verification are never delegated.

Files: issue 03/04/05 status and Answer sections, this ledger, and `todos.md`
only if the work proves an open item belongs there.

Test: run mechanical checks, all acceptance commands, negatively verify the
shortcut guard and one pixel-fidelity guard, restore both, inspect the diff,
then run the repository gate once.

Verify: acceptance criteria 1-6.

Budget: 0 spawns; up to 2 Lead fix/review rounds; full suite yes, once.

## Cost ledger

| Task | Spawns (budget/actual) | Review rounds | Full suite | Notes |
|---|---:|---:|---:|---|
| Recon | 2 / 2 | - | no | Read-only viewer/menu and pixel-worker scouts, explicitly requested by user. |
| T0 | 1 follow-up / 1 | 1 | no | Lead-reviewed PNG seam; focused race-enabled package green. |
| T1 | 0 / 0 | 1 | no | Focused race slice and complete `internal/ui` package green. |
| T2 | 1 follow-up / 1 | 1 | no | Menu specialist extended exhaustive tests/translations; focused race and locale gates green, including restored negative shortcut guard. |
| T3 | 1 follow-up / 1 | 1 | no | Stable-source worker, animation pause, and source-format tests green; restored one-pixel fidelity mutation failed as intended. |
| T4 | 0 / 0 | 2 | yes | Format/TUF/vet/build green. First race gate exposed one stale Actions count in `menu_test.go`; repaired focused, then the complete race gate passed (`internal/ui` 357.789s). |
