# Find more like this — feature and implementation plan

Status: planned; no feature code or evaluation results yet.
Date: 2026-09-13.
Baseline: `a4c8863`, based on PicFetch v1.1.2 (`54fd7c3`).
Route: Deep for implementation; documentation-only for this planning change.
Owner: T0 lead; architecture, review, and fixes remain lead-owned.

At the user's request, this directory is the canonical home for this feature's
plan and [tickets](tickets/README.md), overriding the usual `plans/` and
`.scratch/` locations for these documents. Track its status in
[todos.md](../../todos.md). No duplicate spec or ticket set is required.

## Goal and first version

Let someone use an image to find related pictures in their loaded collection,
follow a promising result as a new reference, and retrace those choices.
The first version uses the existing SigLIP 2 image model and ordinary Grid
actions. Quality must be evaluated before investing in the complete interface.

1. Invoke **Find more like this** from the Actions menu or `Cmd/Ctrl+Shift+L`.
2. Capture the reference and loaded collection; show cancellable preparation
   progress if representations are not ready.
3. Show the reference, a scope/count label, and up to 30 matches in descending
   similarity order. Ordinary click/Enter still opens an image.
4. Invoke the same action on a match to use it as the next reference.
5. Use the visible Back action to restore the previous visit. The earliest Back
   restores the originating image/Grid visit; Exit returns there directly.
6. Compare selected matches, copy them, generate a mosaic, or save the selected
   matches/all current results as a named Favorite through existing workflows.

## Product decisions

These are implementation defaults for the agreed first version, not claims
about measured model quality. Changes should be recorded here before coding.

| Topic | Decision |
| --- | --- |
| Reference | Displayed image, or exactly one explicit Grid selection. With no Grid selection use its highlighted file. More than one selected file disables the single-reference action. |
| Search scope | Freeze all distinct absolute source paths in the loaded collection, including files from different folders. Filename filters and Explorer cohorts do not narrow the search pool. No filesystem-wide indexing. |
| Duplicate behavior | Exclude the reference path and repeated occurrences of an identical path. Distinct files with similar/identical pixels may appear. Ranked visits do not reapply hide-duplicates to the fixed result; the prior setting is retained for return. |
| Ranking | Exact cosine similarity in the original 768-dimensional image embedding space, never distance on the 2D Explorer map or the 15D grouping fit. Descending score; ties use ascending captured path for deterministic results. |
| Count | At most 30 successful distinct matches; fewer is a valid result. A failed reference is an error. Invalid/unreadable candidates are omitted and counted. No percentage-confidence claim. |
| Preparation | Rank once the initial source snapshot has been processed. Progress is visible; results do not continually reshuffle during indexing. A newer reference replaces a queued query without restarting indexing. |
| Navigation | Search results keep their own rank order. Image stepping within the visit follows that order. The root collection order and sort preference remain unchanged. |
| History | Retain the latest 20 successful reference visits plus one origin snapshot. Save reference, match paths/order, query, selected/highlighted paths, and viewport anchor. Keep no decoded images or embeddings in history. No cross-launch persistence. |
| Back while busy | Cancel/supersede the pending query and restore the last committed visit before popping older history; return to origin if none exists. Failed queries do not push history. Back does not run inference. |
| Grid actions | Reuse source-identity mapping for opening, comparing, copying, deleting, and mosaic generation. A new search-specific Save to Favorites action uses selected visible matches, or all visible matches when none are selected, in rank order. |
| Source changes | Added/removed/replaced sources, collection reorder/generation change, or committed file mutations invalidate the search session and history. Do not apply obsolete results. Return to a valid ordinary view and offer an explicit new search. |
| First use | Share Explorer's existing verified asset setup and platform admission. Search never downloads implicitly. Preserve the existing Favorite analysis-cache preference. |
| Modes | Fyne dialogs, comparison, and pending Copy Selection retain ownership. Idle Copy Selection yields. Admit new sessions after scan/sort and picture-frame activity ends. Explorer map typing remains map-owned; invoke search from an image or Grid visit. |
| Escape | Existing dialogs/cards/comparison get first handling; ranked Grid retains marquee, selection, and filename-search unwinding. Only then does Escape perform Back. From an opened result image, Escape first returns to its ranked Grid. |

When entering from an Explorer cohort, preserve its frozen browsing state and
camera in the origin visit. Stop/join any active Explorer analysis off the UI
thread before starting a native search worker. Returning restores the frozen
visit without implicitly resuming inference. Own at most one native analysis
worker for the main window at a time. These transitions belong to root UI
composition and the extracted Explorer interface, not to Grid or the ranker.

## Existing code that shapes the work

| Evidence | Consequence |
| --- | --- |
| [similarity/client.go](../../internal/similarity/client.go) runs a cancellable subprocess and expects one completed map before process exit. | Add an explicit search-session protocol; an ordinary `Analyze` call cannot serve repeated warm queries after completion. |
| [similarity/analyze.go](../../internal/similarity/analyze.go) removes `Embedding` before emitting map items. | Keep the search index in the worker. Send match paths/scores and small progress facts to the UI. |
| [similarity/cache.go](../../internal/similarity/cache.go) versions representations and admits writes against existing Favorites. | Reuse this cache policy; ordinary collections get session memory, not a new persistent database. |
| [grid/subset.go](../../internal/ui/grid/subset.go) uses a membership map; [grid/search.go](../../internal/ui/grid/search.go) scans root file order. | Existing `OpenSubset` cannot preserve similarity ranking. Add an ordered visit while preserving root indexes for actions. |
| [favorites.go](../../internal/ui/favorites/favorites.go) saves `Host.FileCount/FileAt`. | Add a captured-list save entry; calling Add Current List would save the wrong scope. |
| [explorersetup.go](../../internal/ui/explorersetup.go) completes by calling `showExplorer`. | After MA-026, expose a cancellable setup continuation usable by either feature; do not accidentally open a map when setup finishes for search. |

## Proposed modules and interfaces

Names below are contracts for the implementation tickets. New types/functions
are proposed; they do not exist at this baseline.

| Module | Interface and ownership |
| --- | --- |
| `internal/similarity` | `RankSimilar(ctx, reference Item, candidates []Item, limit int) ([]Match, error)`; `Match` carries path and cosine score. Pure ranking validates dimensions/norms and does not mutate inputs. |
| `internal/similarity` | `Client.Search(ctx, SearchRequest, <-chan SearchQuery, func(SearchEvent)) error`, mirrored by an injectable `SearchProvider`. Request freezes paths/limit; query carries monotonic ID and reference path. Events distinguish indexing, ready, matches, and query failure, carrying query identity and immutable values. Provider return observes worker exit. |
| Search worker implementation | Reuse canonical oriented decoding, source-version checks, model/runtime, telemetry/isolation policy, and Favorite cache admission. Extract shared representation preparation inside `similarity`; do not run UMAP/HDBSCAN for search. Retain normalized vectors and source facts, release decoded frames and temporary previews between sources. |
| `internal/ui/grid` | `OpenRanked(RankedVisit)`, `CaptureVisit() Visit`, and `RestoreVisit(Visit)` own ordered source-to-root-index mapping and UI visit state. `ResultIndexes()` remains display order; normal subset behavior is unchanged. |
| `internal/ui/visualsearch` | `Feature` owns source snapshot, query generations, history, status/controls, worker tracking, and per-instance UI queue. External operations are Start, Explore, Back, Close, terminal Stop, State, and Settle. Production and fake `SearchProvider` adapters cross the same seam. Small presentation/return/state-changed callbacks connect to root UI. |
| `internal/ui/favorites` | `AddFiles([]fyne.URI)` captures a defensive copy and reuses naming/overwrite/error/preview behavior. Existing `AddCurrentList` retains its original scope. No temporary root-list replacement. |
| `internal/ui` | Select the reference, build snapshots, mediate setup and mode transitions, bridge ranked image navigation and actions, and invalidate on source changes. Keep `Run` the sole exported root entry. No new search worker fields on `viewer`. |

Implement MA-026 before tickets 05 and 06 so shared setup, frozen Explorer
visits, and analysis shutdown have an owning feature interface. Tickets 01–04
can proceed independently. MA-027 and MA-028 are not prerequisites; respect
their proposed seams without expanding this feature into those refactorings.

## Lifetime, resources, and failure behavior

- One search session owns an immutable collection snapshot and a worker-local
  index. Latest-query wins; callbacks recheck both session and query identity
  inside the UI queue. Close invalidates admission/delivery without blocking UI;
  Stop is terminal; Settle joins every worker and drains/repeats as required.
- Repeated reference changes reuse the index without decoding/inference again.
  Back restores captured results locally. Query channel closure and cancellation
  end the worker; malformed protocol or unexpected exit produces a recoverable
  failure, with all pipes/control writers joined. Normal `Analyze` semantics
  and isolation must continue to work.
- Cap retained embedding storage at `256 * 1024 * 1024` bytes, checked before
  allocation from distinct path count and `768 * 4` bytes per vector. Reject a
  larger session clearly rather than silently truncating its scope. Count and
  measure metadata/IPC/native overhead separately; this is not a total RSS cap.
  Retain neither collection-wide previews nor decoded frames in the index.
- Use existing encoded-file limits. Recheck source versions before accepting
  preparation and before publishing a query; externally changed sources make
  the snapshot stale. This retains the existing metadata-based version model;
  undetectable same-size/same-time external rewrites are not newly solved here.
- Cancel/retry must leave the last committed visit usable. Reference failure,
  zero matches, skipped candidates, missing assets, platform rejection, worker
  failure, and capacity rejection each get clear localized status.
- All analysis runs locally under the current platform policy. Reuse installed
  assets and notices; no new model, text encoder, dependency, installer, or cloud
  service is selected. Windows must continue to report `OfflineVerified=false`.

## Acceptance and evidence

Commands below are future implementation gates. Named tests must exist and
execute; an empty test selection or a skipped native test is not acceptance.
All non-native tests use controlled dependencies, not a real desktop or model.

| ID | Observable acceptance | Ticket / verification command |
| --- | --- | --- |
| AC-01 | A reproducible set of at least 20 varied references records relevance and cold/warm resource measurements; no style/composition promise is inferred from subject matches. | 01; `go test ./scripts/explorereval -run '^TestSearchEvaluation' -count=1`, plus its real-model experiment and recorded human judgments. |
| AC-02 | Rank is deterministic, reference-free, distinct, capped at 30, and correct for known cosine fixtures; malformed candidates cannot corrupt results. | 02; `go test ./internal/similarity -run '^TestSearchRank' -count=1`. |
| AC-03 | Repeated queries reuse representations, source changes reject stale work, and cancellation/EOF observes worker exit without late UI effects. | 03/07; `go test -race ./internal/similarity -run '^TestSearch(Session|Protocol|Cache)' -count=1`. |
| AC-04 | Grid shows score order while selection/actions resolve the original files; filename filtering and visit restoration preserve that order. | 04; `go test -race ./internal/ui/grid -run '^TestRankedVisit' -count=1`. |
| AC-05 | Reference changes, failed queries, bounded history, Back, and Exit preserve the agreed state without inference on Back. | 05; `go test -race ./internal/ui/visualsearch -run '^TestVisualSearch' -count=1`. |
| AC-06 | Actual menu/shortcut/direct entries agree on admission; opening/stepping/comparison/copy/mosaic use the right sources; a result Favorite contains only the captured selection/result. | 06; `go test -race ./internal/ui ./internal/ui/favorites -run '^TestFindMoreLikeThis' -count=1`. |
| AC-07 | File mutations, source replacement, setup cancellation, and shutdown discard obsolete results and cannot reopen closed UI or leave a search worker. | 07; `go test -race ./internal/ui ./internal/ui/visualsearch -run '^TestVisualSearchLifecycle' -count=1`. |
| AC-08 | English/German behavior, keyboard navigation, current platform support, asset reuse, and repository verification are qualified. | 08; `go test . -run '^TestTranslations_' -count=1`, `make check-test-shards`, `make verify`, and the ticket's native search test. |

## Execution and scope control

The [ticket index](tickets/README.md) contains dependencies, file maps, concrete
contracts, tests, and completion criteria. First assess quality (01), then build
ranking/session/Grid foundations (02–04), feature/history and integration
(05–06), and lifecycle/release qualification (07–08). Ticket 07 validates defenses
implemented from the beginning; it is not permission to postpone cancellation.

Each implementation ticket uses TDD through its owning interface, at most two
lead review rounds before reassessment, and zero planned subagent spawns.
Use focused tests per ticket and one full `make verify` for the completed MVP
or each separately handed-off mergeable PR. Maintain exact Qodana exclusions,
UI shard assignments, and `ARCHITECTURE.md` as code moves. Inspect changed code
with GoLand at implementation closeout and record any unavailable native gates.
This planning commit needs document/link/dependency checks only.

Not in the MVP: text queries, external reference-image import, a persistent
collection database, learned preferences, score-axis sliders, automatic file
deletion, cross-launch history, or multi-example feedback. [Ticket 09](tickets/09-feedback-extension.md)
records the positive/negative-example extension separately, preserving the idea
without making it an MVP dependency.

Planning record: zero spawns; one lead documentation review; no application,
real-model, or native tests run during planning. Ranking usefulness, cold-index
time, and practical RSS remain unmeasured until the implementation tickets run.
