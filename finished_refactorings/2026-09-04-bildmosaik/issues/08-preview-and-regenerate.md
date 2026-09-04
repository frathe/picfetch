# Preview and regenerate the exact mosaic result

Type: task
Status: resolved
Priority: P0
Blocked by: 06, 07

## Goal

Present the latest generated pixels and request fresh seeded variants without a
second rendering path or a mutable result identity.

## Existing code anchors

- PicFetch previews ordinary pixels with `canvas.Image` using
  `ImageFillContain` and `ImageScaleSmooth`; the mosaic preview should follow
  that established behavior.
- A Go `image.Image` interface does not provide true immutability or useful
  identity. Ticket 01 therefore protects retained pixels, and consumers prove
  exactness by generation/result and pixels rather than pointer equality.
- Ticket 06's lifecycle, queue, worker tracker, and `Settle` are the only
  generation synchronization path.

## Scope

- Add a proportionally contained `canvas.Image` preview sourced from the
  current `mosaic.Result`. Resizing the window may rescale only the preview
  widget; it must not call `mosaic.Generate` or resample the retained result.
- Replace the initial Generate/Cancel action row after success with Regenerate,
  Set as Wallpaper, Save Image, and Close. Wire Save and Wallpaper to narrow
  callbacks that capture the current result before any background work.
- Keep the source snapshot, target ID, and every visual setting across
  Regenerate. Inject a per-window seed source for tests; production seeds must
  change on each successful request without mutable package-level seams.
- While replacement generation is active, keep the previous preview visible
  but disable Save and Wallpaper so they cannot act on a superseded result.
  Regenerate is also disabled until the active generation settles.
- If regeneration fails, retain the previous successful preview and re-enable
  its actions after presenting the error. A successful newer generation
  atomically replaces it.
- Close invalidates generation, clears transient sources/result widgets, and
  retains only the remembered visual settings/geometry.
- Add and translate new labels/errors in both bundles in this ticket, and keep
  Qodana exclusions synchronized.

## Acceptance Criteria

- Preview aspect ratio and source pixels remain unchanged across window resize;
  target-sized result pixels are not replaced by preview-sized pixels.
- Regenerate sends the same sources, target, and settings with a distinct seed.
- Reverse-order completions cannot re-enable actions or replace the current
  result.
- Save and Wallpaper callbacks receive pixels from the latest completed
  generation and are unreachable while it is superseded.
- A failed regeneration leaves the last good preview actionable; initial
  generation failure leaves no preview actions.
- Close followed by `Settle` leaves no sources, result, worker, or queued UI
  mutation alive.

```sh
go test ./internal/ui/mosaicwin -run 'TestMosaic(Preview|Regenerate|ActionGate|FailureKeepsPreview|Close)' -count=1 &&
go test . -run 'TestTranslations_' -count=1 &&
make check-qodana-test-exclusions
```

## Non-Goals

- Pointer-identity assertions on `image.Image`
- Manual card movement, removal, resize, or rotation
- Preview-specific mosaic generation

## Comments

The previous requirement for the "exact result instance" was replaced with the
observable contract that matters: the latest generation's exact pixels.

Implemented and verified on 2026-09-04: retained target pixels, resize fidelity,
fresh-seed regeneration, action gating, failure retention, and close cleanup are green.
