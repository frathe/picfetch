# Open and close a fitted side-by-side comparison

Status: complete

Route: Deep. This ticket establishes a new comparison feature package, crosses
the grid/menu/viewer composition seams, adds cancellable concurrent work, and
introduces localized UI and manual text.

Deliverable: From an open Grid View, exactly two explicitly selected host files
can be opened in an immediately visible, fitted 50/50 comparison overlay and
closed back to the untouched grid.

Source spec:
`.scratch/compare-two-grid-selected-images/issues/01-open-side-by-side-comparison.md`

## Locked decisions

| Decision | Contract |
|---|---|
| Feature seam | `internal/ui/compare.Feature` owns presentation, ready/loading state, cancellation, staleness, and completion. It never sees `appState` or `grid.Overview`. |
| Caller seam | `internal/ui` snapshots `grid.Selection()` and resolves the two ascending host indices to URIs. It reports invalid entry and load failure through the existing toast. |
| Load seam | A per-viewer `compare.Loader` is injected. Production uses the existing `imaging.ReadAndProbe` plus `DecodeLoaded` path and image cache; tests can block or fail it without package globals. |
| Composition | The opaque comparison overlay is stacked immediately above the grid and below transient confirmation/toast layers. The grid remains open underneath. |
| Exit | Back to Grid and Escape call only `Feature.Close`; they do not call `grid.Close`, `ShowImage`, or mutate the title. |
| Initial rendering | Two no-gap equal-width panes each contain a fitted `canvas.Image` and their own centered spinner. Every open resets both panes. |

## Acceptance commands

1. Entry and enablement:
   `go test ./internal/ui/... -run 'CompareEntry' -count=1`
2. Host-index selection and ordering:
   `go test ./internal/ui/... -run 'CompareSelection' -count=1`
3. Overlay and fitted 50/50 presentation:
   `go test ./internal/ui/... -run 'Compare(Overlay|SideBySide)' -count=1`
4. Concurrent loading and observable completion:
   `go test ./internal/ui/... -run 'CompareLoading' -count=1`
5. Restoration, cancellation, and exit:
   `go test ./internal/ui/... -run 'Compare(Restoration|Cancel|Exit)' -count=1`
6. Failure and stale completion:
   `go test ./internal/ui/... -run 'Compare(Failure|Stale)' -count=1`
7. Locales and manuals:
   `go test ./... -run 'Translations|Manual' -count=1`

Non-goals for this ticket: identity badges, comparison title, Swap, command
isolation beyond Escape, linked zoom/pan, swipe mode, transition preservation,
and SVG rerasterization. Those remain in tickets 02 through 07.

Honest limit: comparison retains two full decoded sources at once. It preserves
existing input limits and reports failure rather than downsampling.

## Task graph

`T1 menu/selection entry` and `T2 comparison module` are contract-independent;
`T3 viewer composition` depends on both. `T4 documentation/landing` depends on
T3.

### T1 - Selection observer and menu/shortcut entry

Owner: T0 inline

Files: modify `internal/ui/grid/grid.go`, `internal/ui/grid/selection.go`,
`internal/ui/grid/marquee.go`, grid tests, `internal/ui/menus/menus.go`,
`internal/ui/menus/menus_test.go`, `internal/ui/menu.go`,
`internal/ui/shortcuts.go`, and their focused tests.

Contract: `grid.SetOnSelectionChanged(func())`; `menus.Callbacks.Compare`;
`menus.State.CanCompare`; `ActionItems.Compare()`; production shortcut
`CustomShortcut{KeyD, KeyModifierShortcutDefault}`.

Test: selection changes resync menu state; the item label/shortcut/state and
the real shortcut wiring match the issue; invalid entry never uses
`grid.Targets()` fallback.

Verify: `go test ./internal/ui/grid ./internal/ui/menus ./internal/ui -run 'CompareEntry|SelectionChanged' -count=1`

Budget: 0 spawns; at most 2 review rounds; no full suite.

### T2 - Deep comparison module

Owner: T0 inline

Files: create `internal/ui/compare/compare.go`,
`internal/ui/compare/compare_test.go`.

Contract: `Loader`, `Callbacks`, `New`, `Overlay`, `Open`, `Close`, `Visible`,
`Ready`, and `Done`. Every `Open` begins one cancellable generation and one
completion generation; both loaders start concurrently; only the latest
generation may paint.

Test: immediate opaque overlay, exact equal-pane layout, fitted/centered first
frames, one spinner per pane, Back to Grid during load, concurrent starts,
cancel, failure-close, and stale-result refusal.

Verify: `go test ./internal/ui/compare -run 'Compare(Overlay|SideBySide|Loading|Cancel|Failure|Stale|Exit)' -count=1`

Budget: 0 spawns; at most 2 review rounds; no full suite.

### T3 - Viewer composition and grid-preserving workflow

Owner: T0 inline

Files: create `internal/ui/compare.go`, `internal/ui/compare_test.go`; modify
`internal/ui/viewer.go`, `internal/ui/features.go`, `internal/ui/build.go`,
`internal/ui/keys.go`, `internal/ui/harness_test.go`, and focused menu/shortcut
tests as learned by T1.

Contract: `viewer.compareSelected()` uses exactly `grid.Selection()`; the
feature overlay is above the grid; Escape closes comparison before the grid;
cleanup closes and waits out comparison work; decode uses the normal imaging
pipeline and `imgCache` without removing failed files.

Test: hidden selected host indices remain eligible and ordered; grid selection,
query/hide state, highlight, scroll offset, title, and file set survive success,
cancel, exit, and either-side failure.

Verify: run acceptance commands 1 through 6.

Budget: 0 spawns; at most 3 review rounds; no full suite.

### T4 - Locales, manuals, architecture, and landing records

Owner: T0 inline

Files: modify `translations/en.json`, `translations/de.json`,
`internal/ui/help/manual.md`, `internal/ui/help/manual_de.md`,
`ARCHITECTURE.md`, `todos.md`, and the issue status/checklist if all gates pass.

Contract: localize `Compare selected images`, `Select exactly 2 images to
compare`, and `Back to Grid`; document only ticket 01 behavior; record the new
package and overlay order.

Test: locale parity, manual guards, and no Unicode arrows in app-drawn content.

Verify: acceptance command 7.

Budget: 0 spawns; at most 2 review rounds; no full suite.

## Delegation gate

No task is delegated. T1 and T4 contain user-visible strings (never delegated),
T2 establishes architecture (never delegated), and T3 crosses packages and
holds the hot integration context (G3 and G5 fail). Rule S applies only to
catalogue key insertion if it can be performed deterministically; edits still
remain T0-owned and reviewed.

## Final gate

Run once after all acceptance commands and negative guard checks:

`make verify`

## Cost ledger

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
|---|---:|---:|---|---|
| T1 | 0 / 0 | 2 | no | Selection observer, menu state, and production shortcut path green. |
| T2 | 0 / 0 | 3 | no | Failure attribution, inert hidden surface, and opacity guards green. |
| T3 | 0 / 0 | 3 | no | Filter/order, cancellation/failure, and restoration integrations green. |
| T4 | 0 / 0 | 2 | no | Locale parity and bilingual manual guard green. |
| gate | - | - | yes | `make verify` passed; Linux/amd64 UI race suite 566.852s. |
