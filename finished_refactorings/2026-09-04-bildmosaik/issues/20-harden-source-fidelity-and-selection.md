# Harden mosaic source fidelity and selection

Type: task
Status: resolved
Priority: P1
Blocked by: 03, 04, 05

## Goal

Make mosaic generation preserve the orientation and identity guarantees users
already see in Grid View: oriented images retain their display geometry, a
randomized pass does not reuse an image while unused choices remain, and hidden
duplicates contribute only their highest-resolution representative.

## Evidence

- The shared orientation reader handles JPEG APP1 and TIFF IFD0 metadata but
  ignores the standard PNG `eXIf` and WebP `EXIF` chunks. Those files reach
  mosaic layout and rendering with both unrotated pixels and dimensions.
- The source pool shuffles slice entries rather than URI identities. Repeated
  occurrences of one URI receive separate identities and can be selected before
  other distinct sources.
- Explicit Grid selections survive later filtering. If duplicate hiding is
  enabled after selecting a lower-resolution member, the hidden member remains
  in the mosaic snapshot instead of resolving to the visible representative.

## Decisions

- Extend the shared orientation reader to PNG `eXIf` and WebP `EXIF`, so
  probing and canonical decoding apply the same transform. Raster card
  geometry comes from the decoded first frame; vector geometry retains its
  probed logical bounds.
- Collapse exact URI repetitions before shuffling. Distinct URIs remain
  distinct unless Grid duplicate hiding is active.
- When duplicates are actually hidden, map explicit selections to the duplicate
  feature's highest-resolution representative and collapse repeated mappings.
  Preserve current explicit-selection behavior while duplicate variants are
  being browsed.

## Acceptance Criteria

- A source carrying orientation metadata renders identically to a lossless copy
  of its display-ready pixels under the same request and seed.
- Successful source loads do not repeat a URI while another distinct URI in the
  randomized pool remains unused.
- A lower-resolution duplicate selected before duplicate hiding contributes its
  highest-resolution representative, exactly once, to the mosaic window.

```sh
go test ./internal/mosaic -run 'TestGenerate_(RespectsDecodedOrientation|UsesDistinctSourceURIsBeforeReuse)' -count=1 &&
go test ./internal/ui -run 'TestMosaicSources_HiddenDuplicatesUseHighestResolution' -count=1
```

## Non-Goals

- Treating different URIs as duplicates when Grid duplicate hiding is off
- Replacing PicFetch's existing orientation metadata support wholesale
- Changing the shuffle seed contract, layout algorithm, or Grid selection model

## Comments

Reported after native macOS and Windows validation of the mosaic workflow.

## Answer

Implemented on 2026-09-04. The canonical imaging path now recognizes
orientation tags in PNG `eXIf` and WebP `EXIF` containers in addition to its
existing JPEG, TIFF/RAW, and HEIC behavior. Mosaic raster geometry comes
from the display-ready decoded frame, so layout and pixels cannot disagree
about the oriented aspect ratio.

The generator collapses exact URI repetitions before shuffling and therefore
uses each distinct readable URI before cycling. At command entry, a selected
duplicate hidden by Grid View resolves to the existing highest-resolution
representative; multiple selected members collapse to one, while browsing
variants retains the exact explicit selection.

All three guards were observed failing for their reported behavior before the
fixes. Focused acceptance tests, package tests, formatting, Qodana exclusions,
vet, build, and the canonical Linux/amd64 race partitions pass through
`make verify`.
