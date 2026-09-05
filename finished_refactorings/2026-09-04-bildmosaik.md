# Implementation Plan: Generate Image Mosaic

Status: ready-for-human
Route: Deep
Spec: `.scratch/bildmosaik/spec.md`

## Frame

Deliver a cross-platform PicFetch workflow that snapshots Grid View sources,
renders a fully covered display-sized mosaic in a deep `internal/mosaic` module,
previews it in a secondary window, exports the exact result, and routes it
through the existing wallpaper integration.

This plan does not implement mobile support, manual card editing, mixed frame
styles, cloud processing, or universal per-monitor wallpaper support for every
Linux desktop.

## Route Rationale

This is a Deep change: it introduces two packages, adds a secondary-window
feature, changes Grid and menu contracts, persists settings, and extends
platform-specific Windows, macOS, and Linux wallpaper behavior. The plan keeps
the complex placement and rendering behavior behind one small interface and
keeps OS variation behind existing or new native seams.

## Decisions - Do Not Relitigate

| Decision | Resolution |
| --- | --- |
| Source with Grid selection | Explicit selection only |
| Source without selection | Entire current filtered Grid result |
| Layout | Adaptive jittered placement with guaranteed coverage |
| Source shape | Preserve aspect ratio; no card-internal crop or stretch |
| Too many sources | Stop after coverage and discard the shuffled tail |
| Too few sources | Repeat only from the same pool |
| Editing | No manual per-card editing in version 1 |
| Window shape | Dedicated secondary mosaic window |
| Rendering seam | `internal/mosaic` owns load, layout, render, and aggregation |
| Display seam | `internal/displays` owns platform display inspection |
| Export | Reuse `internal/filepicker` and `imaging.Export` |
| Wallpaper | Extend existing `internal/wallpaper`; do not fork it |
| Linux target limitation | Typed, honest fallback when per-display apply is unavailable |
| Processing | Local only; source files remain unchanged |

## Task Graph

```text
T1 Core contracts and Grid result seam
  -> T2 Mosaic generator
  -> T5 UI composition

T3 Display inspection
  -> T4 Mosaic window
  -> T5 UI composition

T2 Mosaic generator
  -> T4 Mosaic window

T4 Mosaic window + T5 UI composition
  -> T6 Export
  -> T7 Wallpaper seam

T7 Wallpaper seam
  -> T8 Native wallpaper adapters

T5 through T8
  -> T9 Docs, accessibility, tests, and final gate
```

## Tasks

### Task 1 - Establish mosaic contracts and the Grid result seam

Owner: T0 inline
Files: create `internal/mosaic/mosaic.go`; modify `internal/ui/grid/selection.go`; create/modify their tests; update `qodana.yaml`
Depends: none
Contract: `mosaic.Settings`, `mosaic.DefaultSettings`, `mosaic.Request`, `mosaic.Result`, and `(*grid.Overview).Result() []int`; Result returns defensive host indices, while selection remains a separate fact
Test: defaults and bounds validate; Grid Result reports the active filtered host indices independently of explicit selection and viewport
Verify: `go test ./internal/mosaic/... ./internal/ui/grid/... -run 'Test(Settings|Result)'`
Budget: 0 spawns; 1 review round; full suite: no

### Task 2 - Build the deep mosaic generator

Owner: T0 inline
Files: create `internal/mosaic/generator.go`, `layout.go`, `render.go`, and focused tests/testdata; update `qodana.yaml`
Depends: T1
Contract: `mosaic.New() *Generator` and `(*Generator).Generate(context.Context, Request) (Result, error)` are the external interface; decode, first-frame policy, SVG/RAW handling, seed, retries, occupancy, frames, and sRGB rendering stay behind it
Test: fixed seeds are deterministic; configured bounds hold; coverage has no empty pixel; excess sources are not loaded; exhausted pools repeat; corrupt sources skip; all-corrupt fails; source checksums stay unchanged
Verify: `go test ./internal/mosaic/... -run 'TestGenerate_'`
Budget: 0 spawns; 2 review rounds; full suite: no

### Task 3 - Add native display inspection

Owner: T0 inline
Files: create `internal/displays/displays.go`, build-tagged OS adapters, non-target stubs, and tests; add stubs to `internal/uitest`; update `qodana.yaml`
Depends: T1
Contract: `displays.Inspect(fyne.Window) (Snapshot, error)` returns ordered `Display` values with opaque ID, name, native pixel bounds, and the current-window display ID; callers never parse IDs
Test: adapter fixtures preserve native pixels and IDs; current-window choice, explicit refresh, display removal, HiDPI values, and unsupported inspection are deterministic through stubs
Verify: `go test ./internal/displays/... ./internal/uitest/... -run 'Test(Display|StubDisplay)'`
Budget: 0 spawns; 2 review rounds; full suite: no

### Task 4 - Build the secondary mosaic window and lifecycle

Owner: T0 inline
Files: create `internal/ui/mosaicwin/` window, state, controls, preview, lifecycle, and tests; modify `internal/ui/widgets` only if an existing primitive cannot express a required control; update `qodana.yaml`
Depends: T1, T2, T3
Contract: `mosaicwin.New(fyne.App, Host) *Window`, `Show(Snapshot)`, `Opened() bool`, `Geometry()`, and `Settle(context.Context) error`; Host contains only generate, export, wallpaper, and preference effects
Test: defaults, advanced controls, display refresh, busy state, regeneration seed, stale-result rejection, exact preview result, closing, focus order, and settled cleanup
Verify: `go test ./internal/ui/mosaicwin/... -run 'Test(Mosaic|Generate|Regenerate|Target|Accessibility)'`
Budget: 0 spawns; 2 review rounds; full suite: no

### Task 5 - Compose Grid, menu, preferences, and mosaic window

Owner: T0 inline
Files: modify `internal/ui/menu.go`, `internal/ui/menus/`, `internal/ui/features.go`, `internal/ui/viewer.go`, `internal/ui/run.go`, `internal/ui/harness_test.go`, `internal/preferences/preferences.go`, their tests, all `translations/*.json`, and `qodana.yaml`
Depends: T1, T3, T4
Contract: internal/ui alone chooses `grid.Selection()` when non-empty and otherwise `grid.Result()`, snapshots URIs, then calls `mosaicwin.Show`; preferences round-trip only visual settings and secondary-window geometry
Test: menu availability and order, selection exclusivity, full filtered result, snapshot stability, preference defaults/round-trip, translation parity, and cleanup drain
Verify: `go test ./internal/preferences/... ./internal/ui/menus/... ./internal/ui/... -run 'Test(Mosaic|Translations)'`
Budget: 0 spawns; 2 review rounds; full suite: no

### Task 6 - Reuse the native file picker for exact-result export

Owner: T0 inline
Files: add mosaic export glue in `internal/ui/mosaicwin/` or its Host adapter in `internal/ui`; reuse `internal/filepicker` and `internal/imaging`; add focused tests and `qodana.yaml` entries
Depends: T4, T5
Contract: export accepts the current immutable `mosaic.Result` and requested `.png` or `.jpg`; it never reads viewer display state or triggers generation
Test: suggested timestamp name, extension handling, PNG/JPEG output bounds, current-result identity, cancel, chooser error, and write error
Verify: `go test ./internal/ui/mosaicwin/... ./internal/ui/... -run 'TestMosaicExport'`
Budget: 0 spawns; 1 review round; full suite: no

### Task 7 - Extend the wallpaper seam with a display target

Owner: T0 inline
Files: modify `internal/wallpaper/wallpaper.go`, `internal/ui/wallpaper.go`, `internal/uitest`, and their tests; add mosaic Host glue; update `qodana.yaml`
Depends: T3, T4, T5
Contract: replace the path-only dispatch with a request carrying persistent path plus optional opaque display ID; return a typed target-unsupported result distinct from an execution error; existing single-image wallpaper passes no target and preserves current behavior
Test: legacy single-image flow is unchanged; mosaic passes the exact preview file and selected ID; persistent-copy cleanup never removes the active file; limitation/error retains preview and export
Verify: `go test ./internal/wallpaper/... ./internal/ui/... -run 'Test(SetTarget|MosaicWallpaper|Wallpaper)'`
Budget: 0 spawns; 2 review rounds; full suite: no

### Task 8 - Implement and verify native wallpaper adapters

Owner: T0 inline
Files: modify `internal/wallpaper/darwin.go`, `windows.go`/`notwindows.go` as needed, `wallpaper.go`, and platform-selected tests; update `qodana.yaml`
Depends: T7
Contract: macOS maps the opaque display ID to one `NSScreen`; Windows uses a monitor-aware native path when available; GNOME/KDE return the typed limitation before any global apply when a specific display cannot be honored
Test: command/native seams receive escaped paths and selected IDs; unknown IDs fail closed; no-target preserves the existing all-desktop action; unsupported targeted Linux never reports false success
Verify: `go test ./internal/wallpaper/... -run 'Test(SetDarwin|SetWindows|SetLinux|Target)' && GOOS=windows GOARCH=amd64 go vet ./internal/wallpaper/...`
Budget: 0 spawns; 2 review rounds; full suite: no; native smoke test: required

### Task 9 - Document, harden, and run the final gate

Owner: T0 inline
Files: modify `ARCHITECTURE.md`, `internal/ui/help/manual.md`, `internal/ui/help/manual_de.md`, `todos.md`, translations, end-to-end/golden tests, and `qodana.yaml`; move this plan to `finished_refactorings/` only after branch acceptance
Depends: T2 through T8
Contract: architecture records the new modules and seams; manuals describe selection fallback, regeneration, export, and wallpaper limits; the release matrix records native smoke results
Test: all spec AC commands, no Unicode arrows in manuals/translations, golden preview, 10,000-source lazy-load behavior, rapid regeneration, source immutability, and native Windows/macOS/Linux smoke matrix
Verify: `make fmt-check && go vet ./... && go build ./... && go test -timeout 30m -race ./... && GOOS=windows GOARCH=amd64 go vet ./internal/...`
Budget: 0 spawns; 2 review rounds; full suite: yes, once

## File Map

| Area | Planned change |
| --- | --- |
| `internal/mosaic/` | New deep generation module and deterministic tests |
| `internal/displays/` | New platform display-inspection seam and adapters |
| `internal/ui/mosaicwin/` | New secondary-window feature and lifecycle |
| `internal/ui/grid/selection.go` | Expose defensive current-result host indices |
| `internal/ui/menus/`, `internal/ui/menu.go` | New action and state matrix |
| `internal/ui/features.go`, `viewer.go`, `run.go` | Construct, adapt, persist, and settle the feature |
| `internal/preferences/` | Persist visual settings and window geometry only |
| `internal/ui/wallpaper.go`, `internal/wallpaper/` | Share persistent copy and add target-aware routing |
| `internal/filepicker`, `internal/imaging` | Reused unchanged unless a proven narrow extension is required |
| `translations/`, manuals | Localized user-facing workflow and help |
| `ARCHITECTURE.md` | Add new module ownership and data flow |
| `qodana.yaml` | List every new test path exactly |

## Delegation Gate

No task is delegated in this plan. Architecture, user-visible strings,
cross-feature composition, and platform behavior are explicitly Lead-owned by
the repository agreement. The layout module is algorithmically dense and its
acceptance depends on visual and coverage judgment, so cold delegation would
not reduce context cost. Mechanical translation-catalogue updates should be
scripted after the English keys are final rather than delegated.

## Cost Ledger

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
| --- | --- | ---: | --- | --- |
| T1 | 0 / 0 | 1 | no | Contracts and Grid seam |
| T2 | 0 / 0 | 2 | no | Dense layout/render module |
| T3 | 0 / 0 | 2 | no | Platform seam |
| T4 | 0 / 0 | 2 | no | Secondary-window lifecycle |
| T5 | 0 / 0 | 2 | no | User-visible cross-feature composition |
| T6 | 0 / 0 | 1 | no | Reuse existing export path |
| T7 | 0 / 0 | 2 | no | Existing wallpaper migration |
| T8 | 0 / 0 | 2 | no | Native behavior; smoke tests required |
| T9 | 0 / 0 | 2 | yes | Automated gate green; physical smoke pending |

## Native Smoke-Test Matrix

| Platform | Required checks |
| --- | --- |
| Windows | One and two displays; select each target; persist after app exit; legacy single-image wallpaper unchanged |
| macOS | Internal display plus external display; select each target; preserve existing desktop options; persist after app exit |
| Linux GNOME | Generate for detected display; targeted apply either works as documented or reports limitation before global change |
| Linux KDE Plasma | Same as GNOME, including unknown/unsupported version fallback |
| Unsupported Linux desktop | Generation and export work; targeted wallpaper reports unsupported and leaves preview open |

## Land Checklist

- Run every acceptance command from `.scratch/bildmosaik/spec.md`.
- Negatively verify each new guard test once, restore, then rerun it green.
- Run the final gate exactly once after focused work is green.
- Update `todos.md` with proven remaining work and remove the open feature entry
  only when every ticket is resolved.
- Update `ARCHITECTURE.md` in the package-creation change.
- Record native smoke-test results; state unverified platforms honestly.
- Move this plan and the local tracker directory to `finished_refactorings/`
  only after the branch is accepted.
