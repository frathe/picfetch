# Find more like this — feature and implementation plan

Status: vertical breakdown approved; FML-001 tooling implemented and local measurements recorded; human relevance decision pending.
Date: 2026-09-13.
Design interview updated: 2026-09-14.
Baseline: `a4c8863`, based on PicFetch v1.1.2 (`54fd7c3`).
Route: Deep for implementation; documentation-only for this planning change.
Owner: T0 lead; architecture, review, and fixes remain lead-owned.

At the user's request, this directory is the canonical home for the
[specification](spec.md), this implementation plan, and [tickets](tickets/README.md),
overriding the usual `plans/` and `.scratch/` locations. The specification owns
observable behavior and acceptance; this plan summarizes decisions and delivery.
Track status in [todos.md](../../todos.md). No duplicate tracker set is required.

## September 14 design decisions

Ronin accepted the interview defaults and added persistent caching:

- The newly filtered search result becomes the active list and the source of
  truth for browsing and file actions. Retain a separate copy of the initial
  list, the search origin, so returning restores it as the active list.
- Every new reference searches the original collection, even while a result
  supplies the active browsing/action list. Back/Escape steps through visits;
  from the earliest result it restores the origin. Exit restores it directly.
  Use Esc for the earlier `EKP` spelling under the accepted defaults. Restoring
  a list does not undo disk changes.
- Refresh the ranked result after every 100 scans during preparation. Display
  an actual progress bar at the top of Grid View, with processed/total status.
  Count each distinct image processed, including cache hits and failures, and
  show failures separately. Progress advances between result publications.
  Publish the remaining results at completion, including collections smaller
  than 100. Results remain interactive; selections follow source identity and
  actions capture their targets when invoked.
- Retain 30 seconds as an initial first-search usability target to evaluate,
  not a measured performance claim. Measure first partial presentation and
  complete preparation separately.
- Content similarity is sufficient for the first version. Show the closest
  usable results even when resemblance is weak; ranking is relative to this
  collection, with no confidence percentage or unvalidated relevance cutoff.
- Reuse compatible per-file analysis from existing Favorite caches. Persist
  analysis for loose lists in a general application cache directory using
  hashed per-file keys. Add a dedicated Cache tab in Settings showing current
  general/Favorite analysis usage separately and combined in MB, with full or
  stale-only analysis cleanup. The tab also holds the configurable general-cache
  limit, initially 2 GB. Model assets, thumbnails, and Favorite lists/cohorts
  remain outside these cleanup actions.

The cache decisions, including the accepted Q9/Q11 defaults, are recorded under
Persistent analysis cache. This interview defines the product; implementation
and real-model evaluation remain the work of the tickets below.

## Goal and first version

Let someone use an image to find related pictures in their loaded collection,
follow a promising result as a new reference, and retrace those choices.
The first version uses the existing SigLIP 2 image model and ordinary Grid
actions. Quality must be evaluated before investing in the complete interface.

1. Invoke **Find more like this** from the Actions menu or `Cmd/Ctrl+Shift+L`.
2. Capture the reference and search origin; show cancellable preparation with
   a progress bar at the top of Grid View if representations are not ready.
3. Show the reference, a scope/count label, and up to 30 matches in descending
   similarity order, refreshing after every 100 scans and at completion.
   The result becomes the active list. Ordinary click/Enter still opens an image.
4. Invoke the same action on a match to use it as the next reference.
5. Use the visible Back action to restore the previous visit. The earliest Back
   restores the originating image/Grid visit; Exit returns there directly.
6. Compare selected matches, copy them, generate a mosaic, or save the selected
   matches/all current results as a named Favorite through existing workflows.
7. Reopening an unchanged collection reuses persistent per-file analysis. The
   Settings Cache tab reports disk usage, provides full/stale cleanup, and sets
   the general-cache limit, initially 2 GB.

## Product decisions

These are the accepted product decisions; none claims measured model quality.

| Topic | Decision |
| --- | --- |
| Reference | Displayed image, or exactly one explicit Grid selection. With no Grid selection use its highlighted file. More than one selected file disables the single-reference action. |
| Search scope | Freeze all distinct absolute source paths in the original loaded collection, including different folders. Every reference searches that same scope; filename filters, Explorer cohorts, and progressive results do not reduce it. No filesystem-wide indexing. |
| Duplicate behavior | Exclude the reference path and repeated occurrences of an identical path. Distinct files with similar/identical pixels may appear. Ranked visits do not reapply hide-duplicates to result revisions; the prior setting is retained for return. |
| Ranking | Exact cosine similarity in the original 768-dimensional image embedding space, never distance on the 2D Explorer map or the 15D grouping fit. Descending score; ties use ascending captured path for deterministic results. |
| Count | At most 30 successful distinct matches; fewer is a valid result. Return nearest usable candidates even when similarity is weak, without a confidence claim or relevance cutoff. A failed reference is an error. Invalid/unreadable candidates are omitted and counted. |
| Preparation | Refresh every 100 distinct images processed and at completion. Cache hits and failures count. Show continuously advancing processed/total progress above Grid View, with failures separate. Prepare the reference before candidate publication. A newer reference replaces a queued query without restarting indexing. |
| Navigation | The ranked result is the active list for image stepping and ordinary file actions. The original collection remains the search scope and retained origin. Restoring a visit restores its active list. Root UI owns this distinction; Grid does not mutate appState. |
| Live interaction | Allow selection, opening, and actions during preparation. Track selection/highlight by source identity across ranks and capture action targets at invocation. A refresh cannot retarget an action to the file later occupying the same cell. Preserve a usable viewport. |
| History | Retain the latest 20 successful reference visits plus one origin snapshot. Save reference, match paths/order, query, selected/highlighted paths, and viewport anchor. Keep no decoded images or embeddings in history. No cross-launch persistence. |
| Back while busy | Cancel/supersede the pending query and restore the last committed visit before popping older history; return to origin if none exists. Failed queries do not push history. Back does not run inference. |
| Grid actions | The active search result supplies opening, comparing, copying, deleting, mosaic generation, and Favorite saving. Save to Favorites captures selected result members or the complete filtered Grid result when none are selected, including offscreen files. Add Current List uses the active result while exploring and the ordinary collection outside search. |
| Source changes | Unrelated output writes preserve exploration. A changed/deleted search source invalidates affected analysis and retires the search session to its retained origin, reconciling missing paths. Collection replacement supersedes the session. Restoring origin cannot resurrect deleted files. No automatic inference restart. |
| First use | Share Explorer's verified asset setup and platform admission. Search never downloads implicitly. Reuse compatible Favorite analysis and preserve its cache preference; loose lists receive persistent caching under the policy below. |
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
| [similarity/cache.go](../../internal/similarity/cache.go) stores versioned per-file embeddings and previews in Favorite-owned analysis directories. | Reuse compatible warm records and extend persistence to a separate general directory. Retain source/model validation and Favorite write admission. |
| [grid/subset.go](../../internal/ui/grid/subset.go) uses a membership map; [grid/search.go](../../internal/ui/grid/search.go) scans root file order. | Existing `OpenSubset` cannot preserve similarity ranking. Add an ordered visit while preserving root indexes for actions. |
| [favorites.go](../../internal/ui/favorites/favorites.go) saves `Host.FileCount/FileAt`. | Capture the active list at invocation so later refreshes or naming dialogs cannot change the saved scope. |
| [explorer/setup.go](../../internal/ui/explorer/setup.go) now owns setup and invokes a captured continuation through `EnsureReady`. | Reuse the MA-026 continuation for search; do not accidentally open a map when setup finishes for search. |

## Proposed modules and interfaces

Names below summarize the proposed module boundaries; they do not exist at this
baseline. The [ticket execution map](ticket-execution.md#contracts-shared-by-slices)
pins the shared request, visit, cache-provider and lifecycle contracts. The
vertical breakdown and test seams were approved by `/implement use tdd and sdd`.

| Module | Interface and ownership |
| --- | --- |
| `internal/similarity` | `RankSimilar(ctx, reference Item, candidates []Item, limit int) ([]Match, error)`; `Match` carries path and cosine score. Pure ranking validates dimensions/norms and does not mutate inputs. |
| `internal/similarity` | `Client.Search(ctx, SearchRequest, <-chan SearchQuery, func(SearchEvent)) error`, mirrored by an injectable `SearchProvider`. Request freezes original paths/limit/cache policy; query carries monotonic ID and reference path. Events distinguish progress, partial/final matches, ready, and query failure, with session/query/publication identity and immutable values. Provider return observes worker exit. |
| Search worker implementation | Reuse canonical oriented decoding, source-version checks, model/runtime, telemetry/isolation policy, and Favorite cache admission. Extract shared representation preparation inside `similarity`; do not run UMAP/HDBSCAN for search. Retain normalized vectors and source facts, release decoded frames and temporary previews between sources. |
| `internal/ui/grid` | `OpenRanked(RankedVisit)`, `CaptureVisit() Visit`, and `RestoreVisit(Visit)` own ordered source-to-root-index mapping, progressive updates, and UI visit state. Root indexes are lookup details; `ResultIndexes()` supplies active display order. Normal subset behavior is unchanged. |
| `internal/ui/visualsearch` | `Feature` owns source snapshot, query generations, history, status/controls, worker tracking, and per-instance UI queue. External operations are Start, Explore, Back, Close, terminal Stop, State, and Settle. Production and fake `SearchProvider` adapters cross the same seam. Small presentation/return/state-changed callbacks connect to root UI. |
| `internal/ui/favorites` | `AddFiles([]fyne.URI)` captures a defensive copy and reuses naming/overwrite/error/preview behavior. Root resolves Add Current List against the active search result during exploration; ordinary behavior stays collection-wide. No temporary root-list swap to influence a save. |
| `internal/ui` | Select the reference, capture original scope/origin, expose the active search list to navigation/actions, mediate setup/modes, and reconcile affected-source changes. Keep `Run` the sole exported root entry. No new search worker fields on `viewer`. |
| Cache persistence and maintenance | Similarity owns a root-scoped maintenance provider for inspection, cleanup, and limit retuning, coordinated with cache writers. A new `internal/ui/analysiscache` feature owns the Cache-tab surface, Inspect/Clean/Retune/Close/Stop/Settle lifecycle, tracked workers, and UI queue. Settings hosts its surface; root UI coordinates Explorer/search stop and return callbacks. |

MA-026 is complete: shared setup, frozen Explorer visits and analysis
shutdown now have an owning feature interface for FML-002, with full native
CI evidence in the [archived plan](../../finished_refactorings/2026-09-14-explorer-feature.md).
FML-001 still needs the real relevance
decision before FML-002. MA-027 and MA-028 are not prerequisites; respect their
proposed seams without expanding this feature into those refactorings.

## Persistent analysis cache

Accepted scope: reuse Favorite analysis, add per-file general storage for loose
lists, and provide a dedicated Cache tab in Settings. It shows analysis size in
MB with full/stale cleanup and a configurable general-cache limit initially set
to 2 GB. Persist reusable image analysis; ranked query results and Back history
remain session-local because they depend on the exact collection and reference.

All four cache-policy decisions are accepted:

| Decision | Status | Policy |
| --- | --- | --- |
| Per-file key | Accepted; Q9 | Reuse SHA-256 over the normalized absolute path, validating path, size, nanosecond mtime, and model/preprocessing version inside the record. Hashing this small key needs no image read. Moved/copied sources are prepared again; cross-path content reuse is outside the first version. |
| Cleanup scope and location | Accepted; Q10 | A new Settings Cache tab reports general plus Favorite analysis, with separate totals and combined MB. Clear all managed analysis or remove stale records. Preserve Favorite lists/cohorts, thumbnails, installed model/runtime assets, settings, and session data. |
| Stale records | Accepted; Q11 | Remove incompatible/corrupt analysis, records for changed sources, and obsolete Favorite membership. Preserve records when a disconnected drive or access failure prevents source checks. Missing-source cleanup requires evidence that the source location is accessible. |
| General-cache growth | Accepted; Q12 | Initially cap general storage at 2 GB, configurable in the Cache tab, and evict least-recently-used general records. Favorite analysis remains until explicit cleanup. Persist the chosen limit across launches. |

Use the existing Settings byte convention: 2 GB is 2048 MB, or
`2 * 1024 * 1024 * 1024` bytes. Display usage in MB and label the configurable
limit as applying to general analysis; combined usage can exceed that limit
because it also includes Favorite analysis. Validate positive limits with
overflow checks and retain the last valid value on invalid input. Apply valid
edits through Settings' existing live-change convention; if usage exceeds a
lowered limit, schedule eviction on tracked workers. Include the default-enabled
loose-list cache toggle in the Cache tab, preserving the existing Favorite-cache
preference. Usage inspection includes stored records even when their persistence
toggle is off.

Warm reads try admitted Favorite records before general storage when policy
allows. A compatible general record can warm a newly saved Favorite without
inference. Ownership follows source membership across Favorites, not the active
result list; using a result must not overwrite original Favorite membership.
Avoid a second general copy of every Favorite-owned record. Cache errors degrade
to ordinary preparation with a bounded warning rather than failing search.

Use an application-scoped directory beneath the platform user-cache root,
separate from assets and Favorite data. Tests and trials inject isolated roots.
Retain representation validation and atomic writes. Cleanup closes writer
admission, cancels/joins affected Explorer/search producers off UI, and acquires
root-scoped maintenance ownership before removal. Keep the last browsing visit
usable. Old in-process/subprocess leases cannot publish after completed cleanup;
release admission on cancellation/error and restart inference only on a later
explicit analysis/query. Report measured
remaining/removed bytes and skipped/failed records, including partial completion.
Opening Settings performs inspection only; choosing cleanup starts removal.
FML-010 introduces persistent reuse, bounded writes and the Cache persistence
control; FML-011 adds usage and live limit editing. FML-012 and FML-013 add full
and stale-only cleanup through the same provider and analysiscache UI boundary.
Shared value types and writer-coordination contracts are in the execution map.

## Lifetime, resources, and failure behavior

- One search session owns an immutable collection snapshot and a worker-local
  index. Latest-query wins; callbacks recheck both session and query identity
  inside the UI queue. Close invalidates admission/delivery without blocking UI;
  Stop is terminal; Settle joins every worker and drains/repeats as required.
- Repeated reference changes reuse the index without decoding/inference again.
  Persistent per-file records also avoid inference after reopening an unchanged
  collection. Back restores captured results locally. Query channel closure and
  cancellation end the worker; malformed protocol or unexpected exit produces a recoverable
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

The [specification's acceptance criteria](spec.md#acceptance-criteria-and-verification)
are authoritative; the table below is the execution summary with matching IDs.
Commands are future implementation gates. Named tests must exist and
execute; an empty test selection or a skipped native test is not acceptance.
All non-native tests use controlled dependencies, not a real desktop or model.

| ID | Observable acceptance | Ticket / verification command |
| --- | --- | --- |
| AC-01 | A reproducible set of at least 20 varied references records relevance and cold/warm resource measurements; no style/composition promise is inferred from subject matches. | 01; `go test ./scripts/explorereval -run '^TestSearchEvaluation' -count=1`, plus its real-model experiment and recorded human judgments. |
| AC-02 | Rank is deterministic, reference-free, distinct, capped at 30, and correct for known cosine fixtures; malformed candidates cannot corrupt results. | 02; `go test ./internal/similarity -run '^TestSearchRank' -count=1`. |
| AC-03 | Repeated queries reuse representations; each 100 processed inputs plus completion publishes the latest query's partial/final result; stale work and cancellation/EOF cannot outlive their session. | FML-002/003/005/010/007; `go test -race ./internal/similarity -run '^TestSearch(Session|Protocol|Cache)' -count=1`. |
| AC-04 | Grid shows score order and a top progress bar; live updates preserve source-bound actions, filtering, and visit restoration. | FML-002/003/004; `go test -race ./internal/ui/grid -run '^TestRankedVisit' -count=1`. |
| AC-05 | Reference changes, failed queries, bounded history, Back, and Exit preserve the agreed state without inference on Back. | 05; `go test -race ./internal/ui/visualsearch -run '^TestVisualSearch' -count=1`. |
| AC-06 | Actual menu/shortcut/direct entries agree on admission; opening/stepping/comparison/copy/mosaic use the right sources; a result Favorite contains only the captured selection/result. | FML-002/004/005/006; `go test -race ./internal/ui ./internal/ui/favorites -run '^TestFindMoreLikeThis' -count=1`. |
| AC-07 | File mutations, source replacement, setup cancellation, and shutdown discard obsolete results and cannot reopen closed UI or leave a search worker. | 07; `go test -race ./internal/ui ./internal/ui/visualsearch -run '^TestVisualSearchLifecycle' -count=1`. |
| AC-08 | English/German behavior, keyboard navigation, current platform support, asset reuse, and repository verification are qualified. | 08; `go test . -run '^TestTranslations_' -count=1`, `make check-test-shards`, `make verify`, and the ticket's native search test. |
| AC-09 | Compatible Favorite/general records avoid repeated inference across sessions; invalid records miss safely; general eviction honors the configured limit, initially 2 GB. | FML-010/011/012/013; `go test -race ./internal/similarity -run '^TestAnalysisCache' -count=1`. |
| AC-10 | A dedicated Settings Cache tab reports analysis usage in MB, persists a configurable 2 GB initial general limit, and completes full/stale cleanup with scoped removal, honest partial results, and no late writes. | FML-010/011/012/013; `go test -race ./internal/ui/... -run '^TestAnalysisCacheManagement' -count=1`. |

## Execution and scope control

The [ticket index](tickets/README.md) proposes twelve vertical MVP slices and one
deferred follow-up. The [execution map](ticket-execution.md) carries file maps,
contracts and verification. First evaluate quality (FML-001), then deliver a
usable single search and origin return (FML-002), progressive browsing (003),
result actions (004), reference history (005) and Favorite snapshots (006).
Persistent reuse (010), storage controls (011), full cleanup (012) and stale-only
cleanup (013) each extend a working path. Recovery and final qualification remain
007/008. FML-007 validates defenses implemented from the beginning; it is not
permission to postpone cancellation. File prefixes reflect delivery order while
FML IDs remain stable. The breakdown was approved by `/implement use tdd and sdd`; MVP tickets are
ready-for-agent subject to their blockers, with FML-001 currently claimed.

Each implementation ticket uses TDD through its owning interface, at most two
lead review rounds before reassessment, and zero planned subagent spawns.
Use focused tests per ticket and one full `make verify` for the completed MVP
or each separately handed-off mergeable PR. Maintain exact Qodana exclusions,
UI shard assignments, and `ARCHITECTURE.md` as code moves. Inspect changed code
with GoLand at implementation closeout and record any unavailable native gates.
This planning commit needs document/link/dependency checks only.

Not in the MVP: text queries, external reference-image import, a persistent
collection/query database, learned preferences, score-axis sliders, automatic file
deletion, cross-launch history, or multi-example feedback. [FML-009](tickets/13-feedback-extension.md)
records the positive/negative-example extension separately, preserving the idea
without making it an MVP dependency.

September 13 planning record: zero spawns; one lead documentation review; no application,
real-model, or native tests run during planning. Ranking usefulness, cold-index
time, and practical RSS remain unmeasured until the implementation tickets run.

September 14 interview record: three bounded read-only tasks used one Scout to
establish implementation, progressive-publication, and cache facts; design and document
edits remain lead-owned. Documentation checks only; no feature implementation,
model evaluation, application tests, or commit is part of this interview.

Specification publication: `spec.md` is labeled `ready-for-agent` in the local
tracker. One further bounded read-only task reused the Scout to verify existing
test prior art; synthesis, seam decisions, and review stayed lead-owned. Checks
cover document structure, story numbering, links, AC coverage, and dependencies.

Ticket synthesis: twelve vertical MVP drafts and one deferred follow-up replace
the layer-based breakdown, with stable tracking IDs and no duplicate tracker home.
One bounded read-only task reused the existing Scout for committed-write and
Settings lifecycle facts; decomposition, interfaces and review remained lead-owned.
Document-only verification covers links, numbered files, all 68 user stories,
all ten acceptance gates and a minimal acyclic dependency graph. The canonical
specification is unchanged by ticket drafting. No implementation tests or commits
are part of this step.
