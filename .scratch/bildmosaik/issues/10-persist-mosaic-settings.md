# Persist mosaic visual settings and window geometry

Type: task
Status: resolved
Priority: P1
Blocked by: 01, 07

## Goal

Restore the last valid visual configuration and mosaic-window geometry through
the existing standing-preferences flow, never through session/cache storage.

## Existing code anchors

- `preferences.State`, `Save`, and `Load` are the only standing preference
  store. `internal/session` is reserved for the last-open file set.
- `widgets.Singleton.Remember/Geometry/StopTracking` plus
  `preferences.WindowGeometry` are already used by Settings and EXIF windows.
- Startup is `loadStartupState -> buildViewer -> restoreStartupGeometry`;
  shutdown stops secondary-window pollers before `currentPreferences()` and
  `preferences.Save` run.

## Scope

- Add mosaic settings and `MosaicWindow preferences.WindowGeometry` to
  `preferences.State`, with stable, explicit key names for each field.
- Load numeric values with per-key fallbacks rather than a zero-is-unset rule:
  zero variation, overlap, and rotation are valid user choices. Normalize each
  corrupt/out-of-range field independently through ticket 01's domain rules so
  one bad key does not discard all good settings.
- Persist the frame by its locale-independent `FrameStyle` preference value;
  unknown strings fall back to None.
- Add `RestoreSettings`, `Settings`, `RestoreGeometry`, `Geometry`, and
  `StopTracking` behavior to `mosaicwin` as needed. A window never opened this
  run must preserve seeded geometry exactly like Settings/EXIF.
- Wire restoration in `startup.go`, construction seeding in `features.go`,
  shutdown poller stop in `run.go`, and settings/geometry capture in
  `currentPreferences()`.
- Never persist seed, display ID, source URI, host index, source-kind flag,
  preview pixels, or in-flight lifecycle state.
- Update existing preference and wiring tests; add new test paths to Qodana.

## Acceptance Criteria

- Minimum size plus explicit zero/non-zero variation, overlap, rotation, and
  every frame style round-trip across a fresh app instance.
- A fresh preference store receives ticket 01 defaults. NaN, infinity,
  out-of-range numbers, and unknown frame strings normalize per field.
- Mosaic window position/size round-trip, including origin `(0,0)`, while an
  unopened run does not overwrite earlier good geometry.
- Search of the preference store and serialized values finds no source path,
  URI, Grid index, seed, display ID, or preview data.
- Existing Settings/EXIF geometry and all prior preferences retain their tests.

```sh
go test ./internal/preferences -run 'Test.*Mosaic' &&
go test ./internal/ui/mosaicwin -run 'TestMosaic(Settings|Geometry)' &&
go test ./internal/ui -run 'TestMosaicPreferences' &&
make check-qodana-test-exclusions
```

## Non-Goals

- Persisting the target display across restarts
- Adding mosaic controls to the general Settings window

## Comments

The original ticket missed all four existing wiring sites and would have lost
geometry or clobbered explicit zero settings if implemented like older caps.

Implemented and verified on 2026-09-04: normalized visual settings, valid zeros,
geometry, startup restore, and shutdown snapshot round trips are green.
