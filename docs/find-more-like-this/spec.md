# Find more like this

Status: MVP implemented; qualitative proceed approved September 14, 2026
Date: 2026-09-14
Source: /grill-with-docs, Ronin's accepted defaults and amendments, /to-spec
Delivery: FML-001's technical evaluator is implemented and the 446-image native
experiment is recorded. Ronin approved proceeding from his overall result review,
waiving exhaustive per-item judging; quantitative relevance remains unmeasured.
Search and cache implementation is undergoing final verification and PR review.
Owner: Pico / T0 lead for specification, architecture, review, and fixes

This is the canonical local issue-tracker specification. The feature's agreed
directory exception applies here; the [implementation plan](plan.md) and
[ticket index](tickets/README.md) hold execution details. This specification
owns observable behavior and acceptance. Ticket readiness remains subject to
their dependencies and the initial ranking-quality decision.

## Problem Statement

A PicFetch user can recognize an interesting image without knowing which other
images in the loaded collection resemble its content. Filename filtering and
the Explorer map do not provide a reference-driven, ranked browsing path with
a reliable way back to the original list.

Preparing a large collection may take time. The user needs useful intermediate
results, visible progress, ordinary image actions, and protection against an
update changing the image an action targets. Reopening an unchanged collection
should reuse previous analysis, including analysis already saved for Favorites.
Loose lists need the same reuse without requiring the user to save a Favorite.

Persistent analysis consumes disk space. The user needs to see that usage,
configure a sensible limit, and remove derived records without losing images,
Favorite definitions, or expensive installed model assets.

## Solution

Add Find more like this to Actions and the platform-default Cmd/Ctrl+Shift+L
shortcut. A displayed image or one Grid reference produces up to 30 images
ranked by content similarity. Grid shows the reference first with a purple outline,
followed by up to 30 other matches. This displayed list becomes active for browsing and
file actions. Choosing another reference searches the original collection again.
Back/Esc restores earlier visits and eventually the original list; Exit returns
there directly.

Show a determinate progress bar above Grid View, with processed/total and failed
counts. Refresh the ranked result after every 100 distinct processed images and
at completion, then hide the completed progress bar while keeping the counts and
results. Cached and failed images count toward progress. Users can select,
open, compare, copy, delete, generate mosaics, and save Favorites while analysis
continues; every admitted action keeps its captured source identities.

Reuse compatible per-image analysis from Favorites and a general disk cache.
Settings gains a Cache tab with general/Favorite analysis sizes and their total
in MB, Clear analysis cache and Remove stale records, a loose-list cache toggle,
and a configurable general-cache limit initially set to 2 GB. General records
are evicted by least recent use; Favorite analysis is outside that size limit.

### Settled decisions — do not relitigate

| Topic | Contract |
| --- | --- |
| Useful similarity | Depicted content across styles/media is sufficient for the first version. Appearance quality is measured separately. |
| Search scope | All distinct absolute paths captured from the original loaded collection, including merged folders. Filename filters, cohorts, and result revisions never narrow later-reference queries. |
| Active list and return | The search result drives browsing/actions. Retain the search origin separately and restore its list and visit on final Back or Exit. |
| Ranking | Exact cosine in the original 768-dimensional image representation; at most 30 nearest usable candidates, with deterministic path ties. No confidence percentage or relevance cutoff. |
| Progressive results | Every 100 distinct processed images plus final completion, with a continuously advancing top progress bar. |
| Interaction | Results remain usable; source identity, not a changing cell position, determines action targets. |
| History | Latest 20 successful reference visits plus a separate origin; Back is local and history is not persisted across launches. |
| Persistent reuse | Reuse Favorite analysis first, with general per-file persistence for loose lists. Persist image analysis rather than ranked query answers. |
| Key identity | SHA-256 of normalized absolute path, with source size/mtime and representation-version validation. A moved/copied source is prepared again. |
| Cleanup | The new Cache tab manages general and Favorite analysis only. Keep inaccessible-source records until their status can be checked. |
| Disk limit | Initially 2 GB for general analysis, configurable and persisted in the Cache tab; least-recently-used general eviction. |
| Measurement | Evaluate real usefulness before the full interface. Thirty seconds is an initial first-search usability target, not measured performance. |

## User Stories

1. As an image browser, I want to use the displayed image as a reference, so that I can find related content without knowing filenames.
2. As a Grid user, I want to use exactly one explicitly selected image, so that my reference is unambiguous.
3. As a Grid user without a selection, I want the highlighted image to be the reference, so that keyboard browsing can start a search directly.
4. As a Grid user with several selected images, I want the single-reference command disabled, so that it cannot silently choose one.
5. As a keyboard user, I want the menu and shortcut to behave identically, so that either entry starts the same operation.
6. As a collection browser, I want merged folders included in the search scope, so that useful matches can cross folder boundaries.
7. As an Explorer user, I want searching from a cohort visit to examine the original collection, so that the cohort does not trap later discovery.
8. As an explorer, I want each new reference to search that same original scope, so that a 30-image result does not progressively shrink my search universe.
9. As an image browser, I want matches ordered by content similarity, so that promising images appear first.
10. As an image browser, I want photographs and illustrations of related subjects to match, so that differences in medium do not preclude discovery.
11. As an image browser, I want the closest available results even when similarity is weak, so that I can judge their usefulness myself.
12. As an image browser, I want up to 30 other files ranked after the separately marked reference, so that I can compare them with the starting image.
13. As a collection browser, I want repeated occurrences of the same path collapsed, so that they do not consume result slots.
14. As a collection browser, I want distinct files with identical pixels to remain eligible, so that I can discover alternate copies.
15. As a user with a small collection, I want fewer than 30 results to work normally, so that a small scope is still useful.
16. As a user with unreadable candidates, I want those files skipped and counted, so that one bad file does not prevent useful matches.
17. As a user with a failed reference, I want a clear recoverable error, so that unrelated results are not presented as a valid search.
18. As a user waiting for preparation, I want a processed/total progress bar at the top of Grid View, so that I can see how much work remains.
19. As a user waiting for preparation, I want new results every 100 processed images, so that I can explore before the full collection is ready.
20. As a user with cached or failed files, I want them included in progress accounting, so that the denominator reflects the actual scope.
21. As a user with fewer than 100 images remaining, I want a final result update, so that the last batch is not omitted.
22. As an explorer, I want to keep selecting and opening images during preparation, so that useful intermediate results are actionable.
23. As an explorer, I want an action to keep the files I chose when a result refreshes, so that reordering cannot redirect a copy, comparison, or deletion.
24. As an image viewer, I want an opened image to stay stable during background updates, so that reading a picture is not interrupted by a rank change.
25. As an explorer, I want a promising result to become the next reference, so that I can follow relationships through the collection.
26. As an explorer changing references quickly, I want the newest request to win, so that an older completion cannot replace my choice.
27. As an explorer, I want ordinary click or Enter to open an image, so that a reference change remains a deliberate separate action.
28. As an image viewer, I want stepping to follow the captured result order, so that navigation follows the search I opened.
29. As a Grid user, I want filename filtering within ranked results, so that I can narrow browsing while keeping similarity order.
30. As an explorer, I want Back to restore the previous reference, list, filter, selection, and position, so that I can retrace a branch.
31. As an explorer, I want the original list retained independently of history limits, so that I always have a route back.
32. As an explorer, I want Exit to restore the origin directly, so that I can leave a long exploration quickly.
33. As an explorer with a pending query, I want Back to abandon it safely, so that late results cannot reopen that branch.
34. As an explorer, I want failed queries and batch refreshes to avoid extra history entries, so that Back reflects reference choices.
35. As an image viewer, I want Esc to return from an opened result to its Grid before moving backward, so that existing navigation remains predictable.
36. As a user of dialogs and comparison, I want those surfaces to retain their normal input ownership, so that search does not steal keys or actions.
37. As an Explorer user, I want the original frozen cohort and map camera restored on return, so that reference searching does not lose my place.
38. As a user comparing images, I want the chosen pair captured from the active result, so that refreshes cannot change the comparison.
39. As a user copying files, I want the captured selection or result scope used, so that unrelated original-collection files are not copied.
40. As a mosaic creator, I want its input captured from the active result, so that later analysis cannot change an already-started mosaic.
41. As a Favorite user, I want to save selected matches or the complete filtered Grid result, so that I can retain a useful discovery.
42. As a Favorite user, I want naming and overwrite dialogs to retain the invocation-time list, so that background refreshes cannot change what is saved.
43. As a Favorite user, I want saving a search result to leave the origin Favorite's membership alone unless I explicitly overwrite it, so that exploration does not silently edit my saved collection.
44. As an explorer, I want an export to an unrelated destination to preserve history, so that producing an output does not end browsing.
45. As a user changing or deleting a searched source, I want obsolete analysis rejected and a valid origin restored, so that stale results do not refer to the old file state.
46. As a user returning to an origin, I want deleted files omitted, so that Back does not imply it undid a disk operation.
47. As a Favorite user, I want compatible existing analysis reused, so that an already-prepared collection opens faster.
48. As a loose-list user, I want per-file analysis retained in a general cache, so that saving a Favorite is not required for later reuse.
49. As a user opening overlapping lists, I want shared files to reuse analysis, so that each list does not require a separate full preparation.
50. As a user saving a loose result as a Favorite, I want compatible analysis reused there, so that promotion does not require inference again.
51. As a user editing or replacing an image, I want source-version checks to invalidate its record, so that cached results do not silently describe an earlier image.
52. As a user moving files, I want moved paths treated as new cache identities, so that inexpensive cache lookup remains predictable.
53. As a privacy-conscious user, I want the existing Favorite-cache preference respected and a loose-list cache toggle, so that persistence follows my settings.
54. As a user managing storage, I want a dedicated Cache tab showing general, Favorite, and combined analysis usage in MB, so that I can understand where space is used.
55. As a user managing storage, I want the general cache to start with a 2 GB limit that I can change, so that storage use fits my machine.
56. As a user changing the cache limit, I want the choice saved across launches and excess general records evicted in the background, so that the limit is effective without freezing the app.
57. As a user managing storage, I want least-recently-used general records evicted first, so that recently useful analysis is favored.
58. As a user cleaning storage, I want full analysis cleanup and stale-only cleanup, so that I can choose how much reusable work to discard.
59. As a user cleaning storage, I want images, Favorite definitions, thumbnails, and installed models outside those actions, so that analysis cleanup has a clear scope.
60. As a user with disconnected drives, I want unavailable files' records retained during stale cleanup, so that temporary disconnection does not discard useful analysis.
61. As a user cleaning storage, I want progress, cancellation, and accurate partial-failure reporting, so that I know what was removed and what remains.
62. As a user cleaning storage, I want old background writers excluded from completed cleanup, so that removed records do not immediately reappear.
63. As a user closing Settings or PicFetch, I want pending work canceled and late delivery ignored, so that a closed surface cannot reopen unexpectedly.
64. As an offline user, I want local analysis and reuse after explicit asset setup, so that image searching does not depend on a network service.
65. As a first-time user, I want explicit setup, cancellation, and retry, so that a missing asset never causes a hidden download.
66. As a user on a supported desktop platform, I want truthful platform and isolation status, so that the interface does not promise protections it cannot enforce.
67. As a user of English or German, I want clear labels and keyboard-accessible controls, so that the entire search and cache workflow is usable in either language.
68. As a feature evaluator, I want real-image relevance and cold/warm resource measurements, so that release decisions are based on observed usefulness.

## Implementation Decisions

1. **Module ownership.** The viewer-independent similarity module owns exact
   ranking, source preparation, persistent representations, and subprocess
   transport. The visualsearch feature owns scope, origin, current visit,
   history, request identities, and queued presentation. Grid owns ordered visits
   and rendering. Favorites owns captured-list saving. A dedicated analysiscache
   UI feature owns cache inspection/removal work and supplies the Cache-tab
   surface to settingswin. Root UI composes these modules and command admission;
   it does not own new search or maintenance worker groups.

2. **Existing prerequisite.** Complete the separately tracked MA-026 Explorer
   extraction before search integration. Shared setup must have a cancellable
   continuation for its requesting feature, and Explorer must expose frozen
   visit capture and observable analysis shutdown. MA-027 and MA-028 are not
   prerequisites. Feature construction and overlay composition remain explicit.

3. **Reference admission.** In image view use the displayed source. In Grid use
   one explicit selection, otherwise the highlighted source; several selections
   disable admission. Use the same decision for menu, registered shortcut, and
   direct entry. Preserve dialog, comparison, focused search, Copy Selection,
   scan/sort, and picture-frame ownership. A shared asset setup completion must
   recheck the captured scope and start only its requesting feature.

4. **Scope versus active list.** Capture the original collection as immutable
   distinct absolute paths. Keep the origin's ordered list and browsing state
   separately. Ranked results are the active list exposed to navigation/actions;
   retaining original root indexes internally is permitted only as an identity
   lookup. Every reference query keeps the original scope. A ranked refresh does
   not constitute collection replacement or restart indexing.

5. **Ranking contract.** RankSimilar compares finite, nonzero 768-element image
   vectors using exact cosine, without mutating caller buffers. Return descending
   score and ascending captured path on ties, excluding the reference path and
   repeated paths. Distinct files with identical vectors remain distinct. Skip
   invalid candidates with accounting; reject an invalid reference. Use bounded
   top-k storage, cancellation checks, and no clustering or projected distances.

6. **Search-session protocol.** Client.Search starts a cancellable worker with
   frozen scope, limit, roots, and persistence policy. SearchQuery identifies a
   reference and monotonically increasing query. SearchEvent distinguishes
   progress, partial/final results, readiness, and failure, carrying immutable
   values plus session/query/publication identities. Completion of a query leaves
   the worker available for later queries. Provider return observes subprocess
   exit and joined control work. Preserve ordinary Analyze semantics.

7. **Preparation and publication.** Prepare the reference first. Each distinct
   input contributes once to processed/total, including the reference, cache
   hits, and failures. Initial results publish every 100 processed inputs and at
   completion; an exact final boundary publishes one final result. Update
   progress between result publications. A new reference uses already-prepared
   vectors and can publish their ranking once its own representation is ready,
   without restarting preparation or changing the automatic batch count. The
   latest query wins. Explorer's separate 30-image setting is unaffected.

8. **Foreground interaction.** Apply batch revisions to a foreground ranked Grid
   without reopening it. Preserve selection for surviving result identities,
   including selections merely hidden by filename filtering; drop identities
   evicted from the ranked result without substituting another file. Retain a
   usable source-bound highlight/viewport. Capture source targets when an action
   is admitted. While an image, comparison, or modal action owns the surface,
   retain its captured view/input and defer Grid presentation to the newest
   eligible revision on return. Image stepping follows its captured result order.

9. **History.** First successful partial/final publication commits one reference
   visit; further publications revise it without pushing entries. Before changing
   visits capture its filename query, selected/highlighted identities, and viewport
   anchor. Back abandons an uncommitted pending query first, otherwise restores
   the prior visit. Restored visits are frozen: reject later deliveries for the
   abandoned query. Exit and Back from the earliest retained visit restore origin.
   Image origins retain the path and its occurrence ordinal in merged lists.
   If that occurrence no longer exists, use a surviving occurrence of the same
   path, then the first available image.
   Evict older visits beyond 20 while retaining origin. A new query after Back
   discards the abandoned forward branch. Failed queries before first publication
   add no history; later failure leaves the last usable result with honest status.

10. **Action scope.** Opening, comparison, copying, deletion, and mosaics resolve
    from captured result identities using existing workflows. Save to Favorites
    captures selected members of the current filtered Grid result, or its whole
    result when none are selected, including offscreen files. Add Current List
    captures the active ranked list during exploration and the ordinary collection
    outside it. Naming/overwrite dialogs retain defensive copies. A new Favorite
    may reuse compatible analysis but does not change origin membership.

11. **Source changes and return.** An unrelated output preserves exploration.
    A changed/deleted search source, including a committed write whose initiating
    UI request became stale, retires the affected search session and restores a
    reconciled origin. Omit missing files and choose a valid fallback highlight;
    an empty origin returns to the ordinary empty view. An explicit collection
    replacement adopts the new collection instead. Reorder/generation changes
    invalidate unsafe mappings. External changes detected during worker validation
    reject stale work. Back does not undo disk mutations or trigger inference.

12. **Persistent representation and identity.** Reuse the existing successful
    representation payload: image vector, bounded preview, source facts/digest,
    and model/preprocessing identity. Omit query ranks, map positions, cohort
    assignments, and history. SHA-256 of normalized absolute path selects a
    record; validate its full path, size, nanosecond mtime, representation version,
    finite vector, and bounded payload before reuse. Hashing a key alone does not
    read image bytes. Preserve existing source-version checks before and after
    preparation. Same-size/same-time external rewrites remain an acknowledged
    metadata-validation limit; moves/copies miss under the new path.

13. **Cache ownership and preferences.** Check admitted Favorite records before
    general records. Favorite membership is source-based across saved Favorites,
    independent of active-list filtering. Persist loose-source analysis in an
    application-scoped general store; compatible records can warm a newly saved
    Favorite. Avoid a redundant general copy of every Favorite record. Preserve
    the existing Favorite cache toggle, add a default-enabled loose-list toggle,
    and do not bypass an opted-out Favorite write by redirecting it to general
    storage. Cache misses fall back to preparation. Failed promotion into a
    Favorite reports a bounded cache warning while retaining the usable general
    hit, without repeating inference. Tests and native trials inject isolated roots.

14. **Disk budget.** Default the persisted general limit to 2048 MB using the
    existing Settings byte convention. Validate positive input and overflow,
    retaining the last valid setting for invalid edits. Apply a valid limit on
    Enter or the Apply cache limit button; typing alone does not evict records.
    Shrinking below usage schedules worker-driven LRU eviction. Skip persistence
    for a record larger than the entire general budget while keeping its usable
    in-memory analysis. Count serialized bytes, including managed temporary
    files. When eviction is needed, remove managed general temporary files before
    applying access-time LRU to reusable records. Favorite analysis is outside
    this cap. Combined reported usage may therefore exceed the general limit.

15. **Cache-tab contract.** The analysiscache UI feature provides tab content,
    Inspect, Clean, Retune, Close, terminal Stop, and Settle behavior over an
    injected maintenance provider. Settings retains its value-snapshot/live-edit
    preference pattern. Inspection reports general, Favorite, and combined MB,
    including retained records whose cache toggle is off. Reading usage does not
    remove data. Cleanup reports removed and remaining bytes/records plus skips,
    failures, and cancellation. Incomplete inspection is visibly incomplete.

16. **Stale cleanup and confinement.** Full cleanup covers managed general and
    Favorite analysis records. Stale cleanup removes corrupt/incompatible records,
    old source versions, and obsolete Favorite memberships. Preserve records
    whose source status cannot be established because a drive is disconnected
    or access fails. A missing path counts as stale only when its source location
    can be checked. The current record format does not retain volume identity,
    so missing paths are reported unavailable even when their parent directory
    exists; full cleanup and general-cache LRU can still reclaim those records.
    Constrain traversal/removal to managed record roots and
    formats; protect user images, Favorite lists/cohorts, thumbnails, model/runtime
    assets, settings, session data, and unrelated files.

17. **Maintenance versus writers.** Removal or eviction first closes write
    admission for the managed roots and cancels/joins affected Explorer/search
    producer work off UI, retaining their last usable browsing state. Exclude
    cooperating subprocess writers as well as in-process writers through the
    same root-scoped maintenance lease. Only then remove records or enforce a
    reduced limit. Old leases cannot publish after cleanup completion. Release
    maintenance admission on every exit; cancellation may leave a partially
    cleaned cache and must report that fact. Do not automatically resume inference
    solely because maintenance ended. A later explicit analysis/query can start
    a fresh producer while retaining the captured browsing context. Read-only
    inspection does not retire active producers. A limit increase that needs no
    eviction does not invalidate shared writer leases or delete records. A valid
    changed limit cancels local Explorer/search producers on UI and joins their
    completion off UI before the initial maintenance inventory, preserving their
    browsing state. Persist the limit only after successful current maintenance;
    failed or canceled maintenance retains the previous accepted limit without
    restarting producers. The next explicit reference query uses the accepted
    limit. An unchanged policy keeps the search worker alive.
    Automatic eviction preserves a fully prepared search producer with no
    pending Favorite persistence, so another reference ranks its retained vectors.
    Its old write lease stays revoked. Unfinished preparation and pending writes
    still cancel and join; explicit cleanup and policy changes still retire it.
    Report pressure once per producer and display the paused notice only when
    preparation was unfinished.

18. **Lifecycle and resources.** Each owning feature has a per-instance drainable
    UI queue, worker completion, cancellation, and session/request checks inside
    delivered callbacks. Close is nonblocking on UI; Stop ends future admission;
    Settle joins, drains, and repeats when delivery admits more work. Share at
    most one native analysis worker for the main window. Bound retained vectors
    to 256 MiB before allocation. The subprocess also bounds each JSON input
    message to 64 MiB; oversized search requests are rejected before launch,
    independently of the vector budget. Later reference queries receive fresh
    message budgets. The vector budget is separate from the 2 GB disk limit and
    is not a total RSS cap. Release source pixels and temporary previews between
    inputs; history holds no decoded images or vectors.

19. **Assets and platform behavior.** Reuse the pinned SigLIP 2 encoder, canonical
    orientation/decoding and input-size limits, verified runtime setup, telemetry
    policy, and current platform admission. Search never downloads implicitly.
    Preserve Windows' unverified OS network-isolation status and macOS/Linux
    denial behavior. Reuse the shipped dependency closure and notices; SHA-256
    adds no dependency. Localize every new string and preserve accessible input
    ownership and supported desktop behavior.

## Testing Decisions

Use the test boundaries already selected in the accepted plan. The highest
integration boundary is a real viewer constructed through its existing test
harness, driving actual menu items, registered shortcuts, Grid actions, and
Settings controls. Replace external analysis, filesystem roots, and desktop
effects with controlled providers. Assert displayed behavior, captured file
identities, persisted records, and observed completion rather than private fields.

Focused module tests supplement integration where they prove contracts more
directly: exact ranking fixtures, reusable worker sessions, Grid visits,
visualsearch history, cache-store behavior, and the owned Cache-tab lifecycle.
Use held queues/readers/process adapters to force adverse ordering. Never sleep
to guess completion; a cleared counter is not callback completion. Walk the
displayed container tree when asserting that a control appears on a surface.

Existing prior art includes comparison's explicit-selection menu/shortcut
admission tests; Grid's selection-survives-filter and defensive ResultIndexes
tests; Explorer lifecycle tests with held provider delivery; Settings tab-content
and previous-snapshot tests; and native Favorite cache reuse/membership/lifetime
tests. Existing Favorite-directory move coverage does not prove concurrent wipe
safety; maintenance requires its own observable writer-exclusion regressions.

### Acceptance criteria and verification

These are implementation gates, not checks already passed by this document.
Most named search/cache tests do not yet exist. The existing
TestAnalysisCacheClosesRoots alone does not satisfy AC-09. A command selecting no tests or
skipping a required native case is not acceptance. Existing translation guards
remain real but do not establish feature implementation.

| ID | Observable acceptance | Required verification |
| --- | --- | --- |
| AC-01 | At least 20 varied references have labeled relevance, separate content/appearance reporting, cold preparation/first-partial timings, warm query p50/p95, reopened Favorite/general reuse, vector storage and native RSS measurements, and a recorded proceed/revise decision. | `go test ./scripts/explorereval -run '^TestSearchEvaluation' -count=1`, followed by the real evaluation command below and recorded human relevance judgments. |
| AC-02 | Known cosine fixtures and a full-sort oracle prove correct deterministic top-30 ranking, reference/path exclusion, distinct-file eligibility, malformed-input handling, cancellation, and input immutability. | `go test ./internal/similarity -run '^TestSearchRank' -count=1`; `go test ./internal/similarity -run '^$' -bench '^BenchmarkSearchRank' -benchmem`. |
| AC-03 | Independent sessions and repeated queries reuse eligible analysis; held worker/IPC tests prove latest-query wins, exact 99/100/101/200-input publication accounting, final flush, source-version rejection, and observed EOF/cancel/exit. | `go test -race ./internal/similarity -run '^TestSearch(Session|Protocol|Cache)' -count=1`. |
| AC-04 | Real Grid content includes the top progress bar and correct ranked identities; filtering, reordering, source disappearance, delayed gestures/actions, and deferred foreground delivery preserve the defined targets and visit state. | `go test -race ./internal/ui/grid -run '^TestRankedVisit' -count=1`. |
| AC-05 | Feature tests prove one visit per reference, progressive revisions, pending/committed Back, branching, 20-visit eviction with retained origin, failed-query behavior, no inference on Back, and terminal shutdown. | `go test -race ./internal/ui/visualsearch -run '^TestVisualSearch' -count=1`. |
| AC-06 | Actual commands share admission and complete search, opening/stepping, comparison, copy, mosaic, and both Favorite-save scopes using captured identities; tests include setup retry/cancel and Explorer origin restoration. | `go test -race ./internal/ui ./internal/ui/favorites -run '^TestFindMoreLikeThis' -count=1`; `make check-test-shards`. |
| AC-07 | Source mutation, unrelated output, replacement, cleanup during analysis, queued batches after Back/Exit, and shutdown preserve valid return state and cannot apply stale work or leave a producer writing after retirement. | `go test -race ./internal/ui ./internal/ui/visualsearch -run '^TestVisualSearchLifecycle' -count=1`, plus AC-03 and AC-10. |
| AC-08 | English/German controls, keyboard behavior, cache-limit persistence, real cold/warm sessions, cleanup, current supported platforms, isolation status, and repository checks have recorded evidence. | `go test . -run '^TestTranslations_' -count=1`; `make check-test-shards`; `make verify`; the native command below on each qualified target. |
| AC-09 | Temporary roots and counted fake inference prove cross-session Favorite/general reuse, disabled-cache behavior, source/model/version misses, promotion reuse, bounded payload validation, exact byte accounting, conservative stale removal, configurable LRU, and root-scoped writer exclusion. | `go test -race ./internal/similarity -run '^TestAnalysisCache' -count=1`. |
| AC-10 | Actual Cache-tab controls show separate/combined MB and the persisted 2048 MB initial limit; invalid input, lowered limits, both cleanup actions, partial errors, cancellation, late delivery, and writer coordination have observed results while excluded data and browsing remain intact. | `go test -race ./internal/ui/... -run '^TestAnalysisCacheManagement' -count=1`; `make check-test-shards`. |

Run the real evaluation with prepared local corpus, assets, and evidence roots:

```sh
go run ./scripts/explorereval -search-evaluate -library "$PICFETCH_SEARCH_CORPUS" -assets "$PICFETCH_SEARCH_ASSETS" -out "$PICFETCH_SEARCH_EVIDENCE"
```

Use at least 20 varied, labeled content-reference queries, including duplicates,
hard negatives, unreadable inputs, and sparse/weak-match collections. Record
median precision at 10 with missing slots counted as non-relevant; the initial
content-usefulness target is 0.6. Report appearance judgments separately. The
initial warm-ranking target is p95 below 200 ms for 10,000 prepared vectors on
the recorded development machine. Record first partial and complete preparation
separately against the initial 30-second usability target. Targets are evaluation
criteria, not universal shipping claims. Failure requires a recorded scope or
algorithm decision before proceeding; it does not authorize a new model.

Qualify the real session on each supported native target with installed assets:

```sh
go test -tags=explorertrial ./scripts/explorereval -run '^TestRealSearchSession$' -count=1 -v
```

Use focused tests while implementing each ticket, then the full native
Linux/amd64 Docker verification path for the final gate. Negatively verify the
stale-delivery, target-identity, and cleanup-writer guards by deliberately removing
their protection in temporary overlays and observing the expected failure.
Inspect changed code with GoLand, maintain exact Qodana exclusions and root UI
shard assignments, and record unavailable native checks as unverified. Native
results cannot be inferred from compilation, software screenshots, or fake models.

## Out of Scope

- Text queries, a new model/text encoder/runtime, cloud processing, automatic
  asset downloads, learned preferences, and model training.
- Positive/negative-example feedback; its separately deferred ticket remains
  available after the first version is evaluated.
- A filesystem-wide image index, persistent collection/query database,
  cross-launch search history, external reference import, or cross-path
  content-addressed reuse after moving/copying a file.
- A style/composition guarantee, similarity confidence percentages, an
  unvalidated relevance threshold, or per-axis similarity sliders.
- Automatic image deletion, disk-mutation undo through Back, removing model
  assets/thumbnails/user data through analysis cleanup, or a Favorite-analysis
  size cap.
- Implementing MA-027/MA-028, changing the application's identity, or claiming
  platform behavior or real-model performance without evidence.

## Further Notes

The origin is a return snapshot, the scope is the original comparison universe,
and a search result is the active ordered list. Keep these glossary concepts
distinct even when internal root-index mappings are retained for existing actions.

The general disk limit uses the repository's established MB conversion: 2048 MB
is 2 GiB of bytes and is presented as the requested initial 2 GB setting. Cache
usage means measured managed-file sizes, with incomplete measurements labeled;
the report does not claim to measure total application RSS or filesystem block
allocation. The separate 256 MiB vector budget is not a total process-memory cap.

The interfaces and test names above are design contracts, not verification
results. MVP implementation and its measured scope are recorded in the
[implementation evidence](implementation.md) and
[continuation record](../../finished_refactorings/2026-09-14-find-more-like-this.md).
The latest pushed commit's review/CI gate is tracked in
[PR #25](https://github.com/frathe/picfetch/pull/25). Quantitative relevance and
unrun native acceptance remain unverified; repository merge/release authorization
rules still apply.

The existing feature directory remains the single tracker home. No duplicate
specification or ticket set is created elsewhere. Documentation checks for this
publication cover required sections, story numbering, local links, criterion
coverage, and ticket dependencies; application/model tests run during delivery.
