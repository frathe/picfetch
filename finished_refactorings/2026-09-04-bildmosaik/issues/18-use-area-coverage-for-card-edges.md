# Use area coverage for mosaic card edges

Type: task
Status: resolved
Priority: P1
Blocked by: 17

## Goal

Remove the remaining choppy staircase from rotated mosaic borders and corners
without softening the source photographs.

## Evidence

The native result reported after ticket 17 still shows visibly stepped white
card borders. The existing guard only requires one partially transparent pixel,
so its 2x point-sampled affine transform passes without proving accurate
destination-pixel coverage.

## Decisions

- Keep `internal/mosaic` independent of Fyne windows and graphics drivers; the
  returned CPU image remains the single preview/export/wallpaper result.
- Rasterize exact rotated rectangles into area-coverage masks, then apply a
  center-heavy 3x3 Gaussian filter to the mask only.
- Use those masks for the shadow, body/frame, and photo rectangle. Do not blur
  photo interiors.
- Preserve cancellation checks between bounded row bands.

## Acceptance Criteria

- A 12-degree rotated card edge matches an independent polygon/pixel clipping
  oracle without materially partial pixels becoming binary; mean error stays
  within 16/255 and maximum error within 48/255 after the tiny mask filter.
- The card body and photo rectangle share exact unquantized geometry.
- Existing source interpolation, frame, shadow, deterministic coverage, and
  cancellation behavior remain green.
- A representative generated mosaic is visually inspected at native scale.

```sh
go test ./internal/mosaic -run 'TestRenderPlacement_(AreaCoverage|Interpolates|StopsBetweenTransformBands)' -count=1 &&
go test ./internal/mosaic -run 'TestGenerate_(DeterministicAndCoverage|FrameStyles|DropShadow|SourceFidelity)' -count=1
```

## Non-Goals

- GPU-dependent export or wallpaper generation
- Applying Gaussian blur to source-image pixels
- Changing layout, overlap, controls, or persisted settings

## Comments

Reported from a native mosaic where thin white borders remained visibly jagged
after the first interpolation pass.

## Answer

Implemented on 2026-09-04. Rotated shadow, card-body/frame, and photo rectangles
now rasterize directly into destination-space area masks. A separable 3x3
Gaussian with 14:1:1 center-heavy weights filters only those masks; source
photograph pixels retain the existing Catmull-Rom preparation and interpolated
rotation.

The minimized original-renderer oracle exposed 124/216 partial pixels rendered
binary and a 65/255 maximum alpha error. The expanded thin-frame oracle now has
zero binary pixels across 471 partials, 12.2/255 mean error, and 45/255 maximum
error. The representative 1200 x 750 result and a 2x nearest-neighbor crop were
inspected, and native plus Linux/amd64 deterministic frame references agree.
The final `make verify` repository gate also passed, including all canonical
Linux/amd64 race-test partitions.
