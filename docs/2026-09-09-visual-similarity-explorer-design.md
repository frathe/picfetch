# Visual similarity explorer: design interview

Date: 2026-09-09
Status: Evolving design record; interview paused after the voice session.
This is not an approved specification or implementation plan.

## Background

The earlier idea was a zoomable, pannable map of related images, with clicking
to highlight related pictures proposed as a possible interaction. That click
behavior remains unconfirmed. This note records the decisions settled in the
ongoing interview and leaves unanswered questions open.

## Settled decisions

- The proof of concept is intended to give a general feel for the explorer
  idea and its interaction, with scope kept small.
- After initial setup, image analysis and the explorer must run fully locally
  without requiring network access. The library contains sensitive information,
  so its contents must only be processed locally: this is a privacy constraint.
  Initial setup may prepare required assets but does not authorize uploading
  library images for processing. No specific model download, size, dependency,
  or architecture is selected.
- The intended trial dataset is the user's own image library, estimated at
  roughly 50,000 images. Knowing its contents will help the user judge whether
  the groups feel sensible and whether the feature is useful. This is the
  full library trial scale; the tentative ten to twenty images per pile are
  previews of a cohort, not the trial dataset.
- About one hour is only a rough initial expectation for analysis of the
  roughly 50,000-image library, not a hard maximum or cutoff. Processing
  transparency is important; the estimate remains unmeasured and unproven.
- The preferred proof-of-concept feedback includes both the current state of
  the map as analysis progresses and a progress indicator showing how many
  images have been processed out of the total.
- The explorer is an interactive, freeform map that can be panned and freely
  zoomed, with a deliberately scattered appearance.
- The first implementation groups by content similarity: a photograph of a
  building and a painting of a similar building should be near each other
  even when their styles or colors differ.
- Each image cohort is represented by a loose pile of randomly sampled images,
  with some scattered around it so the cohort's contents are visible. Roughly
  ten to twenty images is the tentative direction for this representation,
  not a fixed or validated count and not a limit on cohort size.
- Clicking a pile opens all images in that cohort using PicFetch's existing
  Grid View. Returning restores the same map position and zoom. Ordinary free
  panning and zooming remain available on the map.
- The explorer should feel alive while background processing continues:
  panning, zooming, opening a pile, browsing its cohort in the existing Grid
  View, and returning to the map all remain available during processing.
- Other piles may move as grouping improves, provided the display does not
  flicker. Updates should use subtle shifts and slowly appearing images;
  whole stacks should not move large distances over a short time. These are
  qualitative visual requirements.
- An option to group using metadata is desired; its eventual scope remains
  open.

## Implementation candidate

- Investigate borrowing a little code from the existing mosaic generator for
  the stack appearance: scatter sampled images around each cohort's
  concentrated location to form loose stacks. This is a reuse candidate;
  feasibility has not been established, and unchanged reuse is not assumed.

## Verified implementation context

Read-only inspection of `ARCHITECTURE.md` (`internal/mosaic`) and
`internal/mosaic/layout.go` confirms seeded rotation, overlap, frame/shadow
geometry, and rendered pixel output. The current `walkLayout` fills uncovered
target pixels across a canvas; it does not arrange separate piles around
cohort locations.

Reusing or adapting the visual treatment is therefore plausible, but localized
cohort placement would require different logic. Unchanged drop-in reuse has
not been demonstrated. This evidence records no architecture decision or
authorization to implement.

## Deferred capabilities

- Geolocation grouping remains desired but is deferred until after testing
  the first proof of concept, which focuses on content similarity.
- User-entered words such as “artwork” or “architecture” should eventually
  influence grouping by matching depicted content. This is explicitly beyond
  the first proof of concept.
- Progressively spreading piles out to reveal all images as the user zooms
  is outside the first proof of concept. The user rejected that proposed
  behavior because it sounded like too much implementation work.

## Open questions

- What remaining primary workflow details and exact presentation mechanics
  should the proof of concept use, including where the cohort grid appears?
- How much real content-similarity processing does the proof of concept need
  to judge the idea's general feel? A mock-only or randomly grouped prototype
  has not been approved.
- How is cohort membership determined, what similarity thresholds (if any)
  apply, and can cohorts overlap?
- How should a cohort with fewer than ten images appear, and what sample count
  should the pile representation use?
- How stable should random samples remain across interaction and reopening?
- What caching, if any, should be used?
- Which hardware will be used for the trial, and what performance is feasible
  on it?
- What exact completion does the rough initial-analysis expectation refer to?
- What makes an image count as “processed”, and does the count cover all
  analysis stages?
- How should gradual visual updates be realized, and how often should the
  current map state refresh during processing?
- How should changed clustering affect cohort membership and an open cohort
  grid?
- What anchoring, if any, applies to the pile currently being explored, and
  how should scroll position and selection be preserved while results change?
- How responsive should the explorer be while processing? No performance
  measurements or speed guarantees exist.
- What is the eventual scope of metadata-based grouping, including geolocation?
- Does the user's phrase “u-map” name the UMAP algorithm or the map concept?
  Which grouping and projection algorithms to use remains undecided.

No hard-to-reverse architecture trade-off has been settled; no ADR is warranted
yet. Further answers belong in this evolving record before a specification or
implementation plan is prepared.
