# Polish mosaic card rendering and configuration

Type: task
Status: resolved
Priority: P1
Blocked by: 04, 05, 07, 10

## Goal

Make generated mosaics read as a layered photo collage: smooth rotated borders,
controlled primary-card overlap, optional subtle shadows, and a compact
configuration view whose visual controls live behind Advanced.

## Scope

- Prevent tiny uncovered rotation holes from causing later cards to bury the
  primary collage. Coverage-repair placements stay behind primary cards.
- Count only opaque card bodies toward coverage; shadows remain visual extents.
- Anti-alias rotated body, frame, image, and shadow edges and interpolate source
  sampling at rotated coordinates.
- Add a default-on, persisted `DropShadow` setting and an Advanced checkbox.
- Keep only target-display selection outside the full-width Advanced toggle.
- Update the standing spec, manuals, translations, and completed-work notes.

## Acceptance Criteria

- The reported 16% size / 18% variation / 7% overlap / 12-degree scenario leaves
  every substantially in-canvas primary card at least 45% visible.
- Coverage remains exact and deterministic; repair placements cannot paint over
  primary cards.
- A rotated border has blended edge pixels rather than a binary staircase.
- Shadow-on and shadow-off renders differ deterministically; shadow-off has no
  shadow footprint, and both preference values round-trip.
- Collapsed Advanced hides minimum size, frame, variation, overlap, rotation,
  and shadow while leaving target display available. Expanded focus order
  reaches all of them.

```sh
go test ./internal/mosaic/... -run 'Test(Layout_(ConfiguredOverlap|ShadowDoesNotCount)|Generate_(RepairCards|DropShadow|Coverage|Deterministic)|RenderPlacement_(Antialiases|Interpolates))' -count=1 &&
go test ./internal/preferences/... -run 'TestMosaicPreferences' -count=1 &&
go test ./internal/ui/mosaicwin/... ./internal/ui/... -run 'TestMosaic(Controls|Configuration|Keyboard|Preferences)' -count=1 &&
go test . ./internal/ui/help -run 'Test(Translations|Manual)' -count=1
```

## Non-Goals

- Manual card editing or per-card shadow/frame choices
- Exact pixel imitation of the supplied reference photograph
- Changing source selection, export, or wallpaper behavior

## Comments

Reported with a native macOS result where 7% overlap still produced completely
buried interior cards and aliased white borders.

## Answer

Implemented on 2026-09-04. Tiny-hole placements are now coverage repairs drawn
under the primary collage, and the planner preserves at least 45% of every
substantially in-canvas primary card. Shadows no longer count toward opaque
coverage. Cards render through padded 2x layers with interpolated affine
sampling, producing smooth rotated edges while retaining cancellable work
bands.

`DropShadow` defaults on, persists both choices, and is exposed with every
visual option behind the existing Advanced button; target display remains
outside it. Representative mosaic output and both window states were visually
inspected. All new guards were negatively verified, and `make verify` passed.
