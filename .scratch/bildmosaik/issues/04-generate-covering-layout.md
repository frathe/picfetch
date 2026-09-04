# Generate a bounded, fully covering mosaic layout

Type: task
Status: resolved
Priority: P0
Blocked by: 01

## Goal

Plan deterministic, organic-looking card placement while guaranteeing geometric
coverage and respecting every configured bound.

## Existing code anchors

- No current package provides mosaic placement; this logic belongs behind the
  new `internal/mosaic` interface rather than in `internal/ui`.
- `internal/imaging` exposes oriented bounds through `ReadAndProbe`, but layout
  must remain pure: decoding/probing and unreadable-source policy belong to the
  generator in ticket 05.
- The repository tests deep modules at their interface. Internal plan types may
  be tested from package-local tests, but must not be exported only for tests.

## Scope

- Add package-private candidate and placement types in
  `internal/mosaic/layout.go`. Candidates contain only stable source identity
  and oriented aspect ratio; placements contain the geometry needed by the
  renderer.
- Use a request-owned seeded PRNG. Never use package-global `math/rand` state or
  wall-clock time in layout.
- Shuffle candidate order once per generation, consume it lazily, and cycle
  only through the same readable candidates when the pool is exhausted.
- Measure the minimum shorter edge on the unrotated image rectangle before
  outer-canvas clipping. Size variation and rotation are symmetric bounded
  offsets around their base values.
- Include the selected frame border, Polaroid footer, and shadow footprint in
  placement and occupancy bounds.
- Track output-pixel coverage (a bitset/mask or an equivalently exact model),
  continue until every target pixel is geometrically covered, and return a
  typed layout error if a safety bound is exhausted. Nominal summed card area
  is not proof.
- Stop asking the candidate iterator for another source as soon as coverage is
  complete. Keep all planning details behind `mosaic.Generate`.

## Acceptance Criteria

- Identical target, settings, candidate metadata, and seed produce identical
  placements and draw order.
- Every unrotated image rectangle respects the minimum-short-edge and size
  variation bounds; every angle respects the configured maximum.
- Coverage tests inspect every target pixel for 16:9, 16:10, 21:9, 4:3,
  portrait, tiny, and non-round target sizes.
- One readable source can repeat to coverage; a longer pool has no candidate
  request after the placement that completes coverage.
- Frame and shadow extents affect both coverage and overlap calculations.
- Cancellation can be checked between placement attempts without changing the
  deterministic result of a non-cancelled request.

```sh
go test ./internal/mosaic -run 'TestLayout_(Deterministic|Bounds|Coverage|Pool|Cancel)'
```

## Non-Goals

- File I/O or decoding
- Pixel rendering
- Exposing placements in `mosaic.Result`

## Comments

The dependency on ticket 03 was removed: Grid source resolution does not affect
the pure layout implementation.

Implemented and verified on 2026-09-04: seeded bounded layout, exact pixel
coverage, light overlap, frame/shadow occupancy, pool stopping, and cancellation are green.
