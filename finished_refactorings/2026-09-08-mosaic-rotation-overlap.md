# Mosaic layouts, rotation, and visible overlap

Status: two-mode implementation selected; focused checks and final repository
verification are tracked below.
Route: Standard.

Deliverable: retain PicFetch's original varied Random mosaic as the default,
offer an ordered axis-aligned Shelf mode, and retain the 0 to 90 degree
rotation path in Random.

## Selected contract

- **Random** remains the original seeded free-form arrangement and is the
  default for a new or existing installation. It is the only mode with
  **Maximum rotation**, from 0 to 90 degrees.
- **Shelf** is a selectable ordered, horizontal-shelf arrangement. It uses the
  same source snapshot and the same frame, size, overlap, and shadow settings
  as Random. It is deliberately axis-aligned, so its rotation control is
  hidden while Overlap remains available. On an extremely thin output, its
  calculated short edge floors at one physical pixel so every card can make
  useful raster progress.
- The saved value is locale-independent. Missing, empty, or unknown values map
  to Random so existing preferences retain the original behaviour.
- Both modes preserve source aspect ratios, output coverage, cancellation, and
  the existing request/result API.

## Validation scope

Shelf owns the new strict final-image checks: every retained occurrence must be
visible, the least-visible occurrence must retain at least 45 percent of its
in-canvas photo area, and Overlap must produce a measured response. The matrix
covers frame styles, shadows, small and large source pools, and axis-aligned
Shelf scenes. Random separately retains its 0 to 90 degree rotation checks.

Random retains its established primary-card visibility guard and its varied
seeded placement. The strict every-occurrence Shelf condition is deliberately
not imposed on Random, because the original algorithm puts gap-repair cards
below primary cards. Treating that as a universal contract would replace the
product's preferred default layout rather than repair a regression.

## Remaining limitation

Random gap-repair cards can be completely covered by later cards. This is an
explicit limitation of the preserved varied mode, not a guarantee provided by
its quality checks. A future Random-specific search can address it only if it
keeps the varied appearance and is evaluated independently from Shelf.

## Work and verification

- [x] Preserve the original Random planner and render path.
- [x] Add a Shelf planner and Shelf render path without changing the source
  snapshot or visual-setting contract.
- [x] Add layout selection, locale strings, preference persistence, and safe
  fallback to Random.
- [x] Keep the 0 to 90 degree rotation control and checks in Random while
  hiding it for axis-aligned Shelf.
- [x] Restrict strict final-pixel visibility and overlap-response checks to
  Shelf; retain the Random primary-card guard.
- [x] Inspect paired Random and Shelf renders generated from the same sources.
- [x] Run focused package checks.
- [x] Run the required Makefile verification against the final implementation.

## Verification commands

- `go test ./internal/mosaic -count=1`
- `go test ./internal/preferences ./internal/ui/mosaicwin -count=1`
- `go test . ./internal/ui ./internal/ui/help -run '^(TestTranslations_EveryLocaleCoversEnglish|TestTranslations_EnglishMapsEachKeyToItself|TestTranslations_NoArrowFollowedByASpace|TestTranslationsHaveNoUnicodeArrows|TestManualDocumentsMosaicLayoutRotationAndOverlap|TestManualHasNoUnicodeArrows)$' -count=1`
- `make verify`

## Evidence to retain

The original Random arrangement deterministically fails the new
every-occurrence condition because some repair cards are hidden underneath
primary cards. The final-pixel oracle independently matches rendered pixels;
the failure is evidence for the mode boundary above, not a reason to make the
default layout uniform. Paired artifacts must identify their source set, seed,
settings, and output path so the visual comparison remains reproducible.

The user approved the current visual result. Earlier apparent gap-filling or
maze-like output is historical behavior, not a new regression from this work.

Paired visual evidence uses the same six bundled sources, seed `2468`, and a
1600 by 900 target: `/private/tmp/picfetch-mosaic-mode-comparison-20260908/README.md`.
The Random and Shelf images confirm the intended distinct appearances;
`shelf-overlap-0.png` and `shelf-overlap-20.png` confirm that Overlap has a
visible Shelf effect and should remain exposed.

Shelf also has a regression test for 1 by 1000 and 1000 by 1 targets. The
one-pixel short-edge floor avoids the confirmed subpixel-card explosion while
leaving normal-size output unchanged; post-plan coverage validation observes
cancellation during both marking and validation.
