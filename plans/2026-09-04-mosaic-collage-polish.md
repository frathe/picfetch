# Implementation Plan: Mosaic collage polish

Status: ready-for-human
Route: Standard
Spec: `.scratch/bildmosaik/spec.md`
Issue: `.scratch/bildmosaik/issues/17-polish-card-rendering-and-controls.md`

## Frame

Deliver a smoother, lightly layered mosaic renderer and a compact configuration
window. Source selection, export, wallpaper routing, and manual card editing are
out of scope.

The route is Standard because this extends one existing feature without new
package or platform seams. It exceeds eight physical files only through the two
translation catalogues and two manuals; production behavior remains confined to
the existing mosaic, mosaic-window, and preference flow.

## Decisions - Do Not Relitigate

| Decision | Resolution |
| --- | --- |
| Overlap meaning | Intended neighboring-card inset, plus a 45% minimum retained area for primary interior cards |
| Coverage repairs | Render underneath primary cards |
| Shadow coverage | Never counts as opaque coverage |
| Shadow default | On, independently persisted |
| Smoothness | Anti-aliased geometric edges plus interpolated rotated image samples |
| Collapsed UI | Target display is the only visible visual setting |
| Visual target | Match the supplied layered-photo character, not its exact pixels |

## Task Graph

```text
T1 Domain setting and placement guard
  -> T2 Smooth layered renderer
  -> T3 Advanced UI and persistence
  -> T4 Docs, visual check, and final gate
```

## Tasks

### Task 1 - Define shadow and overlap behavior

Owner: T0 inline
Files: `internal/mosaic/mosaic.go`, `layout.go`, `mosaic_test.go`, `layout_test.go`
Depends: none
Contract: `mosaic.Settings.DropShadow bool`; deterministic layout keeps coverage repairs below primary placements and excludes shadow from opaque coverage
Test: reported settings retain at least 45% of each primary interior card; shadow on/off changes layout bounds without changing validation
Verify: `go test ./internal/mosaic -run 'Test(Settings|Layout_)'`
Budget: 0 implementation spawns; 1 review round; full suite: no
Result: complete; the reported case moved from 0% retained visibility to the
45% floor, and both the threshold and shadow-coverage guards were negatively
verified

### Task 2 - Render smooth layered cards

Owner: T0 inline
Files: `internal/mosaic/generator.go`, `render.go`, `generator_test.go`
Depends: T1
Contract: repairs composite below primary cards; body/frame/image/shadow edges use fractional coverage; rotated source sampling is interpolated
Test: deterministic transition pixels prove anti-aliasing; shadow-off emits no shadow and coverage stays opaque
Verify: `go test ./internal/mosaic -run 'TestGenerate_'`
Budget: 0 implementation spawns; 2 review rounds; full suite: no
Result: complete; layered repair rendering, 2x card surfaces, bilinear affine
edges, interpolated source pixels, and banded cancellation are green and were
visually inspected at 1000 x 625

### Task 3 - Compact Advanced UI and persist shadow

Owner: two bounded implementer turns for UI and preferences; T0 strings and review
Files: `internal/ui/mosaicwin/window.go`, `mosaicwin_test.go`, `internal/preferences/preferences.go`, `preferences_test.go`, `internal/ui/preferences_wiring_test.go`, `translations/{en,de}.json`
Depends: T1
Contract: collapsed configuration exposes only target display; expanded Advanced owns all visual controls including `Drop shadow`; both boolean values round-trip
Test: canvas visibility, pointer expansion, focus/accessibility order, settings snapshot, and fresh/round-trip preferences
Verify: `go test ./internal/preferences ./internal/ui/mosaicwin ./internal/ui -run 'TestMosaic(Controls|Configuration|Keyboard|Preferences)' && go test . -run TestTranslations`
Budget: 2 implementation turns; 2 review rounds; full suite: no
Result: complete; both preference values round-trip, the rendered collapsed and
expanded states were inspected, and pointer, focus, busy-state, ownership, and
enabled-render guards were negatively verified

### Task 4 - Document and verify the visual result

Owner: T0 inline
Files: `.scratch/bildmosaik/spec.md`, issue 17, `internal/ui/help/manual.md`, `manual_de.md`, `todos.md`, this plan
Depends: T2, T3
Contract: documentation matches the compact UI and optional shadow; a deterministic representative render is inspected before the final gate
Test: manual/translation guards plus focused AC commands
Verify: `make fmt-check && go vet ./... && go build ./... && go test -timeout 30m -race ./...`
Budget: 0 implementation spawns; 2 review rounds; full suite: yes, once
Result: complete; manuals, catalogues, spec, issue, and completed-work notes
match the implementation; `make verify` passed once

## Delegation Gate

Two read-only Scouts were used during recon: one for layout/render causality and
one for the complete persistence/UI path. After the `DropShadow` contract was
fixed, the existing agents each took one independent, mechanically testable
slice: three preference/wiring files and two mosaic-window files. Their prompts
fit G1, each had a focused command for G2, the file sets were disjoint for G3,
and T0 continued the layout/render work in parallel for G4/G5. T0 retained the
algorithm, visual judgment, translations, docs, integration review, every
post-review fix, negative verification, and the final gate. The two extra
implementation turns exceeded the recon-only starting budget to honor the
active parallel-work instruction; the overage is recorded below.

## Cost Ledger

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
| --- | --- | ---: | --- | --- |
| Recon | 2 / 2 | - | no | layout and settings Scouts |
| T1 | 0 / 0 | 1 | no | complete; T0 layout guards |
| T2 | 0 / 0 | 2 | no | complete; T0 renderer and visual review |
| T3 | 0 / 2 | 2 | no | complete; bounded preference and UI turns, T0 review/fixes |
| T4 | 0 / 0 | 2 | yes | complete; `make verify` passed |
