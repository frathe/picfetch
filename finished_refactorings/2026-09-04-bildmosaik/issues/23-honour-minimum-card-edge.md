# Honour the minimum shorter card edge

Type: task
Status: resolved
Priority: P2
Blocked by: 04

## Goal

Make the "Minimum shorter card edge" setting behave as a floor, so no card is
laid out smaller than the value the user chose.

## Evidence

- The spec's settings table reads "| Minimum shorter card edge | 18% of the
  target display's shorter edge |", and states "The minimum size is measured on
  the unrotated card before clipping at the outer canvas."
- `internal/mosaic/layout.go:82` computes `baseShort` from `MinimumShortEdge`,
  and `:111-113` uses it as a midpoint: `shorter := baseShort * (1 + variation)`
  where `variation` is symmetric over plus/minus `SizeVariation`.
- At the defaults (18% edge, 12% variation) the smallest card is 15.84% of the
  shorter edge, below the stated minimum. At the low end of both ranges it falls
  under the documented 10% floor.
- `internal/mosaic/layout_test.go:38` asserts `base*(1-variation)`, so the
  current behaviour is pinned by a test and will not regress on its own.

## Decisions

- Treat the setting as a floor: derive the base from it so the smallest possible
  card equals the configured value, i.e. `base = minimum / (1 - SizeVariation)`,
  and keep the variation symmetric around that base.
- Guard the division for a variation of 1.0.
- Update `internal/mosaic/layout_test.go:38` and any golden expectations that
  encode the old midpoint behaviour.
- If instead the current distribution is preferred as-is, rename the setting to
  "Typical shorter card edge" in the spec, both manuals, and both translation
  catalogues rather than changing the maths. Pick one; do not ship the mismatch.

## Acceptance Criteria

- With any valid combination of minimum edge and size variation, no placed card's
  unrotated shorter edge is below the configured minimum.
- Card sizes still vary across a generation; the variation is not collapsed.
- Coverage of the canvas is unchanged and generation still terminates.

```sh
go test ./internal/mosaic -run 'TestLayout' -count=1
```

## Non-Goals

- Changing the default values in the settings table
- Changing rotation, overlap, or jitter behaviour
- Adding a separate maximum-size setting

## Comments

Found by a spec-axis review of the branch on 2026-09-04. Either the code or the
label is wrong; the ticket is closed by making them agree.

## Answer

The configured minimum is now the lower end of the symmetric size distribution:
the midpoint is derived by dividing the floor by `1 - SizeVariation`. Valid-range
tests cover zero, default, and maximum variation, assert the floor and continued
variation, and the division guard at 1.0 was negatively verified. Coverage still
terminates, the full mosaic package passes, and the four deterministic frame
hashes were updated for the intentional larger-card output.
