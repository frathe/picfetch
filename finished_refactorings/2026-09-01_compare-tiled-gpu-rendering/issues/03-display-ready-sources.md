# 03 - Prepare immutable display-ready sources

Status: resolved

## Contract

Convert each decoded or rerasterized frame into an immutable render source with
an aspect-preserving overview whose long edge is at most 1024. Keep spinners
until both overviews are ready and never mutate canonical decoded pixels.

## Acceptance

Tests cover dimensions, fidelity, transparency, cancellation/staleness, raster
and vector readiness, and publication through the UI queue.

## Comments

The source tests failed first on the absent preparation function, overview,
and per-instance hook. They now prove the 1024-pixel overview bound, alpha and
canonical-frame preservation, cancellation, and spinner/readiness boundary.
The full compare suite passes.
