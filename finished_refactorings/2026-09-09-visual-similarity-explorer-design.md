# Visual similarity explorer: design interview

Date: 2026-09-09
Status: Q1-Q5 defaults accepted; Q6 clarified; remaining design and evaluation choices open.
This is not an approved specification or implementation plan.

The [specification](../.scratch/visual-similarity-explorer/spec.md) synthesizes
this record and includes the accepted answers below.

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
- The trial must evaluate real content grouping and map interaction together.
- The first trial runs on this Mac. Broader platform support remains open.
- The explorer takes its input from PicFetch's currently opened file set,
  including existing merged-folder and Favorites workflows.
- Each image belongs to one cohort at a time; cohorts do not overlap in the
  first trial. Presentation of algorithmically unassigned images still needs
  a decision.
- An opened cohort keeps its membership fixed while it is browsed in Grid
  View. Updated grouping is used after returning to the map and opening a
  pile again. This does not freeze all background analysis or map updates.

The user accepted Q1-Q5 with “all defaults” and clarified Q6: the earlier
discussion concerned promising local models and analysis tools. SigLIP 2,
HDBSCAN, and UMAP are candidates for evaluation, not a selected stack or a
requirement to use UMAP.

## Implementation candidate

- Evaluate SigLIP 2 for image representations, HDBSCAN for cohort grouping,
  and UMAP for dimensionality reduction supporting clustering and/or map
  layout. Their documented roles and progressive-update limitations are
  recorded in the [candidate research](2026-09-09-visual-similarity-explorer-candidates.md).
  This is an evaluation direction, not an approved dependency or pipeline.
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

Read-only Grid View inspection during the resumed interview establishes:

- `internal/ui/grid/search.go:119` builds a subset of host file indices for
  filename search, duplicate hiding, and duplicate browsing without replacing
  the main file set. It has no external arbitrary-cohort filter input.
- `internal/ui/grid/grid.go:583` initializes the highlight from the displayed
  image on opening; `grid.go:613` and `grid.go:633` close the overlay and clear
  selection, search, and duplicate browsing. Ordinary close/reopen therefore
  does not restore a saved grid browsing state.
- `internal/ui/compare_test.go:1575` contains a regression test asserting that
  returning from comparison preserves nonzero grid scroll, highlight, and
  selection. Comparison keeps the grid open underneath. This is an existing
  return pattern, not proof that an explorer round trip is already supported;
  the test was inspected, not rerun during this documentation session.

The selected Mac reports Apple M5 Max, 18 CPU cores, and 51,539,607,552 bytes
(48 GiB) of memory via `sysctl machdep.cpu.brand_string hw.memsize hw.ncpu`.
These are observed hardware facts, not throughput or memory-use measurements
for any candidate pipeline. No model assets were downloaded or executed.

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
- How is cohort membership determined, what similarity thresholds (if any)
  apply, and how are unassigned images presented while keeping cohorts
  non-overlapping?
- How should a cohort with fewer than ten images appear, and what sample count
  should the pile representation use?
- How stable should random samples remain across interaction and reopening?
- What caching, if any, should be used?
- What performance is feasible on the selected Apple M5 Max Mac with 48 GiB
  memory, and which candidate runtime/model variant should be evaluated?
- What exact completion does the rough initial-analysis expectation refer to?
- What makes an image count as “processed”, and does the count cover all
  analysis stages?
- How should gradual visual updates be realized, and how often should the
  current map state refresh during processing?
- How should cohort identity and map placement evolve after reclustering,
  while preserving the fixed membership of an already opened cohort?
- What anchoring, if any, applies to the pile currently being explored, and
  how should scroll position and selection be preserved while results change?
- How responsive should the explorer be while processing? No performance
  measurements or speed guarantees exist.
- What is the eventual scope of metadata-based grouping, including geolocation?
- Which image encoder, clustering, and projection algorithms should be used
  after evaluating the clarified SigLIP 2/HDBSCAN/UMAP candidates?

No hard-to-reverse architecture trade-off has been settled; no ADR is warranted
yet. Further answers belong in this evolving record and its specification
before an implementation plan is approved.

## Resumed interview: accepted first round

The user accepted the defaults for Q1-Q5 and corrected the premise of Q6.
The earlier settled decisions remain in force.

| Question | Accepted answer |
|----------|-----------------|
| Q1 | Judge real grouping and map interaction together on the intended library. |
| Q2 | Start with this Mac; eventual platform support remains open. |
| Q3 | Use the opened file set, including existing merge and Favorites workflows. |
| Q4 | One cohort per image for the first trial. |
| Q5 | Freeze the opened cohort; use newer grouping after returning to the map and opening a pile again. |
| Q6 | The earlier discussion was about promising local models/tools: SigLIP 2, HDBSCAN, and UMAP. These are evaluation candidates; no algorithm is mandated. |

The open questions above remain tracked. No model variant, dependency,
grouping/projection pipeline, or implementation plan is selected.

## Next decision frontier

Q7-Q12 are pending proposals, not part of the accepted first-round defaults.

| Question | Decision requested | Recommendation |
|----------|--------------------|----------------|
| Q7 | Where should the map and cohort grid appear? | Use the main window; opening a pile covers the map with Grid View. Return restores the map, with existing grid Escape precedence respected. |
| Q8 | Is Go with a native local inference runtime acceptable for this proof of concept? | Evaluate Go with native inference first, following the user's Go-feasibility question and the candidate research. Packaging for broader distribution remains separate. |
| Q9 | How should images without a confident cluster be presented? | A clearly marked Unassigned collection that opens in Grid View, so no images disappear and no similarity is implied. |
| Q10 | Should analysis results survive application restarts? | Cache image representations locally, reuse unchanged images with the same model/preprocessing, and recompute changed/new inputs. Exact storage and limits remain planning work. |
| Q11 | What pile sample size and stability should the trial use? | Up to fifteen distinct images, showing all when fewer exist; keep the sample stable while cohort membership is unchanged. |
| Q12 | What does the progress counter mean? | Show images with completed representations separately from skipped/failed images and grouping/layout status; the trial is ready only when its final grouping and map are delivered. |

Q8 asks about the permitted shape of the trial, not permission to install or
download a selected runtime/model. None has been selected or installed.
