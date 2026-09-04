# Mosaic domain model and validation

Type: task
Status: resolved
Priority: P0
Blocked by: none

## Goal

Define the small, platform-independent interface to the deep mosaic module:
validated settings and requests in, a read-only rendered result out.

## Existing code anchors

- `internal/imaging.LoadedImage` and `imaging.Export` already use
  `image.Image`; the mosaic module must remain compatible with that path.
- `internal/imaging` also uses `fyne.URI` without depending on `internal/ui`,
  so source URIs do not require a new path abstraction.
- Numeric preference fallback is handled explicitly in
  `internal/ui/startup.go`; zero is a valid value for three mosaic controls
  and cannot double as an "unset" sentinel later.

## Scope

- Create `internal/mosaic/mosaic.go` and its tests.
- Define `FrameStyle`, `Settings`, `DefaultSettings()`, `Request`, and
  `Result`. Use ratios for percentage settings (`0.18`, `0.12`, `0.08`) and
  degrees for rotation so UI labels do not leak into the module.
- Give frame styles stable, locale-independent preference values. Unknown
  values must normalize to `None`; display labels belong in `mosaicwin`.
- Make request construction defensively copy the source slice. Validate nil
  URIs, an empty pool, non-positive target axes, integer-overflowing target
  area, and every configured range.
- Return typed validation errors that identify the invalid field. Context
  cancellation and the later "no readable source" outcome are not validation
  errors.
- Keep layout placements, decoded sources, and mutable pixel buffers out of
  the external interface. `Result` may expose bounds and an image accessor,
  but callers must not be able to mutate the pixels retained by the result.
- Add the new package to `ARCHITECTURE.md` in this change and add every new
  `_test.go` path to Qodana's exact exclusion list.

## Acceptance Criteria

- Defaults are 18% minimum shorter edge, 12% size variation, 8% overlap, and
  7 degrees maximum rotation.
- Allowed ranges are respectively 10-30%, 0-25%, 0-20%, and 0-12 degrees,
  inclusive.
- Explicit zero variation, overlap, and rotation remain valid and distinct
  from an absent preference.
- Requests and results retain no caller-owned mutable slices or pixel buffers.
- The external module interface contains no Fyne widgets, viewer state,
  display enumeration, file picker, wallpaper, or preference behavior.

```sh
go test ./internal/mosaic -run 'Test(Settings|FrameStyle|Request|Result)' -count=1 &&
make check-qodana-test-exclusions
```

## Non-Goals

- Layout or rendering implementation
- Localized frame labels
- Preference I/O

## Comments

Refined against `internal/imaging`, `internal/preferences`, and the repository's
deep-module conventions on 2026-09-04.

Implemented and verified on 2026-09-04: validated immutable requests/results,
stable frame preferences, defaults, ranges, and defensive ownership are green.
