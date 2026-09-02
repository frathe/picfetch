# 02 - Introduce the pane scene renderer seam

Status: resolved

## Contract

Keep `compare.New` unchanged and add an unexported constructor/factory for two
pane renderers. Transform application presents immutable `paneScene` values.
The test reference adapter preserves deterministic canvas geometry and pixels.

## Acceptance

Feature tests prove initial, transformed, swapped, cleared, and resized scenes,
and prove pane render objects stay stable across interaction.

## Comments

The seam test first failed to compile because `paneScene`, `paneRenderer`, and
the private constructor did not exist. After implementation it proved source,
logical/physical geometry, stable objects, Swap, clear, and Settle coverage.
The full compare suite exposed and then pinned source publication before a
viewport exists; the suite is green.
