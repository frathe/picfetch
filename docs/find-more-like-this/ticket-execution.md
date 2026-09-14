# Find more like this — proposed ticket execution map

Status: approved for implementation by /implement use tdd and sdd, 2026-09-14.
Date: 2026-09-14
Owner: Pico / T0 lead

The [specification](spec.md) owns behavior; the [ticket index](tickets/README.md)
owns delivery order and blocking edges. This companion keeps file maps, proposed
interfaces and executable verification out of the behavior-focused tickets.
Nothing here claims implemented APIs or passing feature tests.

## Routing and readiness

This remains a Deep implementation effort. Each ticket is one bounded behavior
through its necessary layers, owned inline by T0 with zero planned spawns and at
most two lead review rounds before reassessing scope. Focused checks accompany
each slice; FML-008 owns the complete final gate. A separately handed-off PR also
requires its repository gate. Design, review and fixes are never delegated.

The user approved the breakdown and its test seams with `/implement use tdd and
sdd`. FML-001 is claimed; other MVP tickets are `ready-for-agent`, subject to
completed blockers. FML-009 remains deferred.
FML-001 needs its real corpus/assets and recorded relevance decision to finish;
fixture-only success does not unblock FML-002.

[MA-026](../../needs_refactoring.md#ma-026) is complete, including full native CI
qualification; its gate before FML-002's UI integration is satisfied.
MA-027/028 are not prerequisites. Keep MA-026's evidence in its existing tracker.
Its extracted feature must supply frozen cohort/camera capture and restoration,
observable analysis shutdown, and shared setup with a cancellable continuation.
Copy its actual names into the root adapters before beginning FML-002; these
capabilities are a dependency gate, not work postponed into the search feature.

The file numbers express delivery order; FML tracking IDs remain stable. FML-004
and FML-005 have independent behavior gates after FML-003, but their root/Grid
files overlap: implement them serially rather than assigning simultaneous edits.
The general cache is an enhancement to a working search, not an artificial
prerequisite for the first result. Full cleanup and stale cleanup are separate
complete Settings actions so each fits one fresh implementation context.

## Contracts shared by slices

These signatures are proposed implementation seams, not existing source. Keep
implementations inside the owning modules; root UI supplies composition and
source identities. A layer needed by one slice lands with that slice's working
consumer and tests. Later tickets extend behavior without making an earlier
slice depend on their unfinished code.

### Ranking and search transport

```go
RankSimilar(ctx context.Context, reference Item, candidates []Item, limit int) ([]Match, error)
(*Client).Search(ctx context.Context, request SearchRequest, queries <-chan SearchQuery, emit func(SearchEvent)) error
type SearchProvider func(context.Context, SearchRequest, <-chan SearchQuery, func(SearchEvent)) error
```

`Match` carries `Path string` and `Score float64`. `SearchRequest` freezes a
`SessionID uint64`, distinct `Paths []string`, `Limit int` and `Cache CachePolicy`;
existing client configuration supplies verified runtime/assets and admission.
`SearchQuery` carries `ID uint64` and `ReferencePath string`. `SearchEvent` carries
`SessionID`, `QueryID` and `Revision` as uint64, a typed kind, processed/total/failed
counts, immutable matches, and typed failure/status information. Kinds distinguish
progress, partial result, final result, ready, query failure and terminal failure.
Do not send embeddings, mutable buffers, translated UI text or an unbounded preview
collection across this boundary. Provider return observes child exit and control
writer completion. Ordinary `Analyze` retains its existing finite-map contract.

FML-002 uses this operation for one query and one final ranked result; it does not
need chaining or scheduled partial publications to demonstrate the first path.
FML-003 adds the 100-source cadence and retained index; FML-005 supplies repeated
queries through the same operation. A reference/query failure must not become a
successful empty result. Candidate failures remain separately counted.

### Ordered visits and feature ownership

```go
(*Overview).OpenRanked(visit RankedVisit)
(*Overview).CaptureVisit() Visit
(*Overview).RestoreVisit(visit Visit)
(*Feature).Start(request StartRequest) bool
(*Feature).Explore(referencePath string) bool
(*Feature).Back() bool
(*Feature).Exit()
(*Feature).State() State
(*Feature).Suspend() <-chan struct{}
(*Feature).Close()
(*Feature).Stop()
(*Feature).Settle()
```

The Overview operations belong to `internal/ui/grid`; the Feature operations here
belong to `internal/ui/visualsearch`. Favorites separately adds
`(*Feature).AddFiles(files []fyne.URI)` to `internal/ui/favorites`.
`RankedVisit` supplies ordered source identities, a
revision and small progress facts. `Visit` captures filename query, selected and
highlighted identities and viewport anchor. Root indexes are disposable lookups;
they never define the chosen file after a revision or collection replacement.

`StartRequest` contains copied original scope, origin browsing snapshot, reference
and cache policy. A narrow consumer-side Host presents ranked state, captures and
restores visits, and notifies root composition of state changes. Root alone exposes
the active ordered result to navigation/actions. The feature owns request tokens,
history and a per-instance UI queue; neither Grid nor viewer gains search workers.

Introduce Start/Exit/basic lifecycle with FML-002, progressive revisions with
FML-003, Explore/Back history with FML-005 and maintenance suspension with FML-010.
Suspend is requested on UI, invalidates producer delivery, retains browsing and
returns an observable completion signal. Wait on it off UI. A subsequent explicit
query can start a new producer; restoring a visit or finishing cleanup cannot.
Close invalidates without a UI-thread join, Stop ends admission, and Settle
joins/drains/repeats. AddFiles copies its argument before naming dialogs begin.

### Cache policy and maintenance

`CacheRoots` contains `GeneralDir string` and `FavoritesDir string`. Roots are
application-scoped in production and injected temporary directories in tests.
`CachePolicy` contains Roots, FavoriteEnabled, LooseEnabled and GeneralLimitBytes.
Membership discovery remains available for routing even when Favorite persistence
is disabled; do not silently classify those Favorite sources as loose writes.

The existing Favorite layout stays intact. General records use an isolated
`picfetch/analysis` directory beneath the platform user-cache root, separate from
assets, updates and wallpaper caches. Its owned record namespace is versioned;
record basenames use the existing lowercase SHA-256 path-key convention. Use
`2048 * 1024 * 1024` bytes initially, independent of the vector budget. Existing
Favorite records have no general-cache size cap.

```go
type CacheMaintenanceProvider interface {
    Inspect(context.Context, CacheRoots, func(CacheProgress)) (CacheUsage, error)
    Clean(context.Context, CacheCleanRequest, func(CacheProgress)) (CacheReport, error)
    Retune(context.Context, CacheRetuneRequest, func(CacheProgress)) (CacheReport, error)
}
```

This interface lives with the viewer-independent similarity backend. Concrete
methods name unused arguments `_` as required by repository conventions.
`CacheCleanRequest` contains Roots and Mode (`ClearAll` or `RemoveStale`).
`CacheRetuneRequest` contains Roots and LimitBytes. `CacheUsage` holds General and
Favorite byte/record totals plus an Incomplete flag; its total is their sum.
`CacheReport` carries observed Before/Remaining usage, RemovedBytes/RemovedRecords,
skipped/unavailable/failure counts, Canceled and applied limit when relevant.
`CacheProgress` supplies operation phase and observed counts/bytes, not a false
known total for an incomplete scan. Preserve partial values alongside errors.

The configured provider receives an injected
`Quiesce func(context.Context, CacheRoots) error`. Before destructive maintenance
it closes affected writer admission, asks root composition to suspend affected
Explorer/search producers on UI, joins them off UI, then enters removal. The same
root-scoped lease/epoch protocol gates atomic commits from parent and cooperating
subprocess writers. A normal persistence write that needs eviction requests parent
coordination and releases its writer claim before producer shutdown is joined;
it must never wait for maintenance while holding a claim that maintenance joins.
Ordinary reads/writes respect the current limit and source-version lease.

This lease is scoped to managed roots, not a mutable package-global test hook or
only a process-local mutex. Retired writers cannot recreate deleted directories or
publish under a newer epoch. Retain existing opened-root/moved-Favorite guarantees.
Rescan after exclusion before reporting removals, release admission on every exit,
and preserve browsing if eviction retires a currently preparing producer. Pure
inspection and no-eviction retuning do not call Quiesce. Automatic LRU uses this
same path; there is no second uncoordinated deletion implementation.
For automatic eviction only, quiescence may retain a completed search producer
with no pending Favorite persistence. Its old lease stays revoked, while later
reference queries rank the retained vectors. Preparing producers and pending
Favorite writes still cancel/join. Explicit cleanup and policy changes retain
their full producer-retirement contract.

```go
(*Feature).TabContent() fyne.CanvasObject
(*Feature).Inspect()
(*Feature).Clean(mode similarity.CacheCleanMode)
(*Feature).Retune(limitBytes uint64)
(*Feature).Close()
(*Feature).Stop()
(*Feature).Settle()
```

These operations belong to the new `internal/ui/analysiscache` feature with an
injected provider, tracked workers and its own drainable queue. FML-010 introduces
its tab/persistence control and necessary automatic eviction; FML-011 adds usage
and editable limit, FML-012 adds ClearAll, and FML-013 adds RemoveStale. Settings
hosts its content and keeps the existing three-method Host and preference
snapshot/live-patch pattern. Root composes the quiescence adapter and forwards
close/shutdown; it does not gain a maintenance worker group.

## Per-ticket file maps

Every entry below has Owner T0 inline, budget zero spawns / at most two lead
reviews, and the named verification gate. Paths are proposed where files do not
yet exist. Tests assert the ticket's observable criteria; no full test bodies or
implementation algorithm is prescribed here.

<a id="fml-001"></a>
### FML-001 — Evaluation

Files: create `scripts/explorereval/search.go`, `search_evaluate.go`,
`search_cache.go`, `search_report.go`, `search_review.html`, `search_test.go` and
`search_trial_test.go`; modify its `main.go`, `README.md`, `evaluate.sh`, the
root Makefile, Qodana exclusions and `ARCHITECTURE.md`. The report and separate
Favorite baseline are split to keep their measurement boundaries explicit. Produce
`docs/find-more-like-this/evaluation.md` only when real measurements exist.
Contract: the evaluation entry consumes a `search-corpus.json` manifest beneath
the selected library root, with stable source IDs/relative paths and labeled
content/appearance intent. Output identities, judgments, top-30 results, metrics
and machine information into the chosen evidence root. Keep private images there.
Test/verify: [V01](#v01); final production cache/session metrics are rerun in V08.

<a id="fml-002"></a>
### FML-002 — First completed search

Files: create `internal/similarity/search_rank.go`, `search.go`, `search_worker.go`
and focused tests; extend `client.go`, `analyze.go`, and shared preparation only as
needed for a single final query. Create `internal/ui/visualsearch/feature.go`,
`uiqueue.go` and initial tests; add `grid/ranked.go` and visit tests. Add the root
`internal/ui/visualsearch.go` adapter and `visualsearch_test.go`; update explicit
construction, menu/shortcut/input, active-list navigation, committed-file
reconciliation and shutdown/harness wiring. Reuse the extracted MA-026 setup and
origin interfaces. Reconcile exact Explorer filenames after that prerequisite.
Contract: the single-query subset of the shared transport/visit APIs, plus source
invalidation and exit. Preparation progress is displayed from the first slice.
Test/verify: [V02](#v02). No history branch, general store or partial scheduler is
needed to close this slice. It must not land as disconnected backend scaffolding.

<a id="fml-003"></a>
### FML-003 — Progressive browsing

Files: extend similarity search session/worker tests, visualsearch feature tests,
`grid/ranked.go`, `grid/selection.go`, `grid/marquee.go`, `grid/nav.go`, and root
search integration tests. Reuse the initial root adapter and active-list seam.
Contract: immutable revision delivery every 100 distinct inputs and final flush;
source-bound gestures and deferred presentation are required with the first live
revision. Completion keeps the worker index ready for later references.
Test/verify: [V03](#v03), including 99/100/101/200 boundaries and held surfaces.

<a id="fml-004"></a>
### FML-004 — Result actions

Files: extend `grid/search.go`, `grid/ranked.go` and tests; integrate root
`batch.go`, `compare.go`, `mosaic.go`, `filework.go`, `viewer.go`, and existing
navigation/action adapters. Add focused scenarios to root search integration
tests; reuse `internal/uitest` desktop seams.
Contract: actions receive captured URI identities, existing selected/result
semantics and invocation-time order. `afterFileWrite`/`writtenFileLoaded` include
committed stale writes and filesystem aliases; `ReconcileDeletedFiles` reports
actual moved identities. Keep existing file-work completion/reconciliation paths.
Test/verify: [V04](#v04); basic invalidation already lands in FML-002.

<a id="fml-005"></a>
### FML-005 — Reference history

Files: add `visualsearch/history.go` and tests; extend its feature, root adapter,
input routing and integration tests; extend similarity repeated-query tests.
Contract: Explore/Back and retained-origin state over the existing transport;
first successful result commits one visit, late batches cannot revise restored
visits, latest 20 entries do not evict origin. Grid captures/restores source-bound
visits; no second preparation or navigation path is introduced.
Test/verify: [V05](#v05). Independent of FML-004's action expansion after FML-003,
but shared root/Grid files mean these tickets are implemented serially.

<a id="fml-006"></a>
### FML-006 — Favorite snapshots

Files: extend `internal/ui/favorites/add.go`, `favorites.go`, existing tests,
root `shortcuts.go` (`showAddFavorites`), `features.go`, `menu.go`, and search
integration tests. The Favorites feature's AddFiles owns copying.
Contract: AddFiles and root invocation-time selection/filter scope. General
cache promotion is exercised when that store arrives in FML-010.
Test/verify: [V06](#v06).

<a id="fml-010"></a>
### FML-010 — Persistent reuse

Files: extend `similarity/cache.go`, `analyze.go`, shared preparation and search
requests; create `generalcache.go`, `cachemaintenance.go` and focused tests.
Create `internal/ui/analysiscache/feature.go`, `uiqueue.go` and tests; introduce
the Cache tab in `settingswin/settingswin.go` and tests. Extend preferences,
startup/current-preferences/ApplySettings wiring and create root
`internal/ui/analysiscache.go` coordination with harness/shutdown coverage.
Contract: CacheRoots/CachePolicy, compatibility routing, atomic root-scoped writer
admission, fixed initial LRU budget, automatic eviction and the persistence
control. Introduce only the maintenance primitives needed by that working path.
Test/verify: [V10](#v10), including isolated cross-session inference counts and
promotion into a saved Favorite. Never open real user cache roots in tests.

<a id="fml-011"></a>
### FML-011 — Usage and live budget

Files: extend similarity cache maintenance and tests, analysiscache feature and
tests, settingswin content tests, preferences serialization/validation and root
ApplySettings/startup/shutdown tests. No root-owned worker group is added.
Contract: Inspect/Retune through the injected provider, exact measured file bytes,
partial results, overflow-safe positive limits and existing live patch semantics.
Test/verify: [V11](#v11), including uninterrupted inspection, shared leases
preserved on increases without eviction, local producer retirement before inventory and
observed off-UI eviction when a lowered limit requires removal.

<a id="fml-012"></a>
### FML-012 — Full cleanup

Files: extend `similarity/cachemaintenance.go` and tests, analysiscache feature
controls/tests, root coordination and `analysiscache_test.go`, plus search/Explorer
lifecycle tests at their owning interfaces. Reuse FML-010's automatic-eviction
writer protocol rather than building another purge path.
Contract: Clean(ClearAll) with Quiesce, scoped root exclusion, measured partial
reports, cancellation and retained browsing/history. Settings close forwards
Close; shutdown forwards Stop; harness joins/drains both feature queues.
Test/verify: [V12](#v12), including held in-process and subprocess commits.

<a id="fml-013"></a>
### FML-013 — Stale-only cleanup

Files: extend the same maintenance implementation/tests and Cache-tab controls
and integration tests; retain existing Favorite membership/root semantics.
Contract: Clean(RemoveStale) distinguishes verified stale from unavailable and
reports that distinction through the same result/progress types. An inaccessible
drive is never inferred stale solely from a failed source stat.
Test/verify: [V13](#v13), with explicit unavailable/accessibility fixtures.

<a id="fml-007"></a>
### FML-007 — Cross-feature recovery

Files: add `internal/ui/visualsearch_lifecycle_test.go` and integration cases to
root/owned cache and visualsearch tests; fix only observed integration defects in
the participating owners. Keep deletion, save, export, EXIF and mosaic tests on
their existing committed-write notification paths.
Contract: no new service; exercise the complete accepted lifecycle across existing
seams. Negatively verify guards with temporary overlays and restore every overlay.
Test/verify: [V07](#v07); do not postpone foundational safety into this ticket.

<a id="fml-008"></a>
### FML-008 — Qualification

Files: extend tagged native trials in `scripts/explorereval`, record real evidence
and `evaluation.md`, update the feature evidence/status record, manuals and release
documentation as applicable. Inspect all changed code, locales, notice delivery,
Qodana exclusions and shard assignments before the final repository gate.
Contract: no new functionality assumed; finish real qualification and recorded
limitations for the implementation already delivered.
Test/verify: [V08](#v08), plus all canonical specification gates.

<a id="fml-009"></a>
### FML-009 — Deferred feedback

Candidate files: extend the local evaluation runner/report and fixtures first;
only a successful, separately accepted experiment expands similarity query/ranking
and visualsearch visits/history. The retained candidate compares a normalized
positive centroid with a bounded negative penalty; weights and example limits
remain experimental. This is not agent-grabbable implementation work.
Test/verify: [V09](#v09); it is not part of MVP coverage or the active frontier.

## Shared implementation checks

For each slice: write meaningful tests through its consumer interface first,
observe the stated failure, implement, and run the named focused gate. Assert
actual rendered-tree membership and captured file identities with held queues,
readers/process adapters and completion signals, never sleeps. Confirm every
required case executes; zero selected tests and skipped native trials are not
evidence. Existing `TestAnalysisCacheClosesRoots` alone does not prove the new cache.

Translate every introduced UI string immediately in all catalogs; keep exact
test-file exclusions in `qodana.yaml` and root UI tests in the shard manifest.
Update `ARCHITECTURE.md` when modules/files move, maintain `todos.md`, and inspect
changed code with GoLand. Any unverified platform/inspection result is recorded.
No dependency or model change is selected by these tickets. Existing shipped
licenses and notices still require recorded review at qualification.

Common incremental commands, in addition to each gate below:

```sh
go test . -run '^TestTranslations_' -count=1
make check-test-shards
git diff --check
```

Every ticket checkbox links to its V gate. Each V gate requires the indicated
observable cases and recorded command output, not merely a matching test prefix.
Future test names are reserved requirements for implementation, not existing tests.

<a id="v01"></a>
### V01 — Evaluation

Fixture tests validate manifests, report completeness, stable IDs/order, precision
denominators and failure accounting. The real run must record at least 20 human
relevance judgments and a proceed/revise decision; fixtures alone cannot do so.
Set the three variables to prepared local corpus, assets and evidence directories.

```sh
go test ./scripts/explorereval -run '^TestSearchEvaluation' -count=1
go run ./scripts/explorereval -search-evaluate -library "$PICFETCH_SEARCH_CORPUS" -assets "$PICFETCH_SEARCH_ASSETS" -out "$PICFETCH_SEARCH_EVIDENCE"
```

<a id="v02"></a>
### V02 — First search

Pin reference admission and actual menu/shortcut entry, scope/origin, explicit
setup/cancel/retry, initial top progress, final active-list browsing/stepping,
Exit/Esc, source mutation/replacement, capacity refusal and terminal shutdown.
Rank fixtures compare a full-sort oracle and cover immutable input, dimensions,
NaN/Inf/zero vectors, ties, deduplication, self-exclusion and cancellation.

```sh
go test -race ./internal/similarity -run '^TestSearch(Rank|ProtocolInitial|SessionInitial)' -count=1
go test ./internal/similarity -run '^$' -bench '^BenchmarkSearchRank' -benchmem
go test -race ./internal/ui/... -run '^(TestFindMoreLikeThisInitial|TestVisualSearchInitial|TestRankedVisitInitial)' -count=1
```

<a id="v03"></a>
### V03 — Progressive browsing

Observe 99/100/101/200 accounting with duplicate inputs, cache hits and failures;
exact final boundaries, reference-first preparation, retained-index readiness,
foreground identity preservation, deferred surfaces and cancel/EOF/exit. Include
held gestures and source removal between admission and delivery.

```sh
go test -race ./internal/similarity -run '^TestSearch(Session|Protocol)' -count=1
go test -race ./internal/ui/... -run '^(TestFindMoreLikeThisProgressive|TestVisualSearchProgressive|TestRankedVisitProgressive)' -count=1
```

<a id="v04"></a>
### V04 — Actions

Use actual Grid controls and OS-effect stubs to observe the chosen pair/files,
filtered order, offscreen result scope, captured stepping order and committed
mutation reconciliation. Include aliases, original-scope sources outside the
current result, unrelated exports and an empty origin.

```sh
go test -race ./internal/ui/grid -run '^TestRankedVisit' -count=1
go test -race ./internal/ui -run '^TestFindMoreLikeThisActions' -count=1
```

<a id="v05"></a>
### V05 — History

Observe newest-reference wins, zero repeated inference for warm queries, one visit
per successful reference, failures before/after first publication, pending-query
Back, frozen restoration, branching, 20-visit eviction and independent origin.
Actual root commands cover input ownership and frozen Explorer return.

```sh
go test -race ./internal/similarity -run '^TestSearch(Session|Protocol)' -count=1
go test -race ./internal/ui/visualsearch -run '^TestVisualSearch' -count=1
go test -race ./internal/ui -run '^TestFindMoreLikeThisHistory' -count=1
```

<a id="v06"></a>
### V06 — Favorite capture

Hold naming/overwrite callbacks across revisions and navigation. Read the saved
membership to prove selected-versus-filtered scope, offscreen inclusion, ordinary
Add Current List behavior and origin-Favorite preservation.

```sh
go test -race ./internal/ui ./internal/ui/favorites -run '^TestFindMoreLikeThisFavorites' -count=1
```

<a id="v10"></a>
### V10 — Persistent analysis

Count inference across independent sessions and overlapping lists, Favorite-first
reads and general-to-Favorite promotion. Exercise cache preference combinations,
source/version/payload misses, changed Favorite membership/root moves, oversized
records, scaled LRU budgets and held subprocess writers during automatic eviction.
Settings entry proves the new tab and persisted loose-list toggle are usable.

```sh
go test -race ./internal/similarity -run '^(TestAnalysisCache|TestSearchCache)' -count=1
go test -race ./internal/ui/... -run '^(TestFindMoreLikeThisPersistentCache|TestAnalysisCacheManagementPersistence)' -count=1
```

<a id="v11"></a>
### V11 — Usage and budget

Check actual tab membership, measured separate/combined MB (including temps and
disabled stores), partial inspection, initial 2048 MB, invalid/overflow edits,
preference round trip, live shrink/LRU, uninterrupted inspection and observed
close/reopen/shutdown. Increases without eviction preserve shared leases. A valid
changed limit retires and joins local producers before the initial inventory,
preserving browsing until the next explicit query captures the accepted policy.
Failed/canceled maintenance keeps the prior limit without restarting producers.
Unchanged policy keeps the worker alive. Use scaled byte budgets; do not write
2 GB for a unit test.

```sh
go test -race ./internal/similarity -run '^TestAnalysisCache(Usage|Limit|Maintenance)' -count=1
go test -race ./internal/preferences -run '^TestAnalysisCachePreferences' -count=1
go test -race ./internal/ui/... -run '^TestAnalysisCacheManagement(Usage|Limit|Lifecycle)' -count=1
go test -race ./internal/ui -run '^TestFindMoreLikeThisInitialAdmission$/limit-increase-before-inspection' -count=1
go test -race ./internal/ui/visualsearch -run '^TestVisualSearchCacheLimitChangesRetireCapturedPolicy$' -count=1
```

<a id="v12"></a>
### V12 — Full cleanup

Hold root leases and in-process/subprocess commits immediately before publication.
Observe quiescence, removed/remaining bytes and records, partial/canceled reports,
release on all exits, retained browsing/history and no old write after completion.
Exercise the actual Clear analysis cache control and assert excluded trees intact.

```sh
go test -race ./internal/similarity -run '^TestAnalysisCache(Clear|Maintenance|Confinement)' -count=1
go test -race ./internal/ui/... -run '^TestAnalysisCacheManagement(Clear|Writers|Lifecycle)' -count=1
```

<a id="v13"></a>
### V13 — Stale cleanup

Use accessible missing/modified sources, inaccessible/disconnected locations,
corrupt/versioned payloads, membership changes and valid records. The same worker
exclusion, confinement, partial reporting and cancellation guarantees apply.

```sh
go test -race ./internal/similarity -run '^TestAnalysisCache(Stale|Maintenance|Confinement)' -count=1
go test -race ./internal/ui/... -run '^TestAnalysisCacheManagement(Stale|Writers|Lifecycle)' -count=1
```

<a id="v07"></a>
### V07 — Recovery journeys

Run the complete held-worker journeys and existing transport/cache regressions.
Use `go test -overlay "$PICFETCH_GUARD_OVERLAY"` with each focused guard test to
deliberately disable its protection in a temporary overlay. Record the expected
failure and the restored passing command; never leave the violation in source.

```sh
go test -race ./internal/ui ./internal/ui/visualsearch -run '^TestVisualSearchLifecycle' -count=1
go test -race ./internal/ui ./internal/ui/favorites -run '^TestFindMoreLikeThis' -count=1
go test -race ./internal/ui/... -run '^TestAnalysisCacheManagement' -count=1
go test -race ./internal/similarity -count=1
```

<a id="v08"></a>
### V08 — Native qualification and final gate

Run every canonical acceptance gate against the final implementation. Reproduce
V01 with completed Favorite/general reuse and final session metrics. Qualify the
tagged test on each supported native target using isolated roots and real assets;
record actual machine/model/OS/isolation evidence. Inspect every changed code file
with GoLand and review exact licenses/notices. The complete suite uses a native
Linux/amd64 daemon or equivalent native CI; emulation is not the isolation gate.

```sh
go test . -run '^TestTranslations_' -count=1
make check-test-shards
make verify
go test -tags=explorertrial ./scripts/explorereval -run '^TestRealSearchSession$' -count=1 -v
```

<a id="v09"></a>
### V09 — Deferred follow-up gate

These prospective gates are retained from the deferred proposal; they are neither
implemented nor authorization to begin. Separate selection and an accepted
experiment must establish the chosen rule and record a proceed/reject decision.
Any implementation after a proceed decision needs its own accepted scope.

```sh
go test ./scripts/explorereval -run '^TestSearchFeedbackEvaluation' -count=1
go test -race ./internal/similarity ./internal/ui/visualsearch -run '^TestSearchFeedback' -count=1
```

## Coverage and planning evidence

| Spec gate | Slices responsible for delivery |
| --- | --- |
| AC-01 | FML-001 evaluates usefulness; FML-008 completes production cache/session measurements. |
| AC-02 | FML-002 ships the exact ranker with its first UI consumer. |
| AC-03 | FML-002 initial transport; FML-003 cadence/index; FML-005 requery; FML-010 persistent reuse; FML-007 combined recovery. |
| AC-04 | FML-002 initial visit/progress; FML-003 live revisions; FML-004 filtering/actions. |
| AC-05 | FML-002 origin/Exit; FML-003 progressive visit; FML-005 full history. |
| AC-06 | FML-002 entry/setup/return; FML-004 actions; FML-005 chaining; FML-006 Favorite capture. |
| AC-07 | Each introducing slice's lifecycle, FML-012 cleanup integration, FML-007 cross-feature negative guards. |
| AC-08 | Strings and local checks in each introducing slice; FML-008 final native qualification. |
| AC-09 | FML-010 reuse/admission; FML-011 usage/retuning; FML-012 full removal; FML-013 stale-only removal. |
| AC-10 | FML-010 persistence control; FML-011 usage/limit; FML-012/013 cleanup controls and lifecycle. |

| User stories | Primary delivering slice |
| --- | --- |
| 1–17, 31–32, 35–37, 45–46, 64–66 | FML-002, with full history/Explorer return exercised again in FML-005. |
| 18–24 | FML-003; top progress already appears in FML-002. |
| 25–27, 30, 33–34 | FML-005. |
| 28–29, 38–40, 44 | FML-004. |
| 41–43 | FML-006. |
| 47–53, 57 | FML-010. |
| 54–56, 63 | FML-011; lifecycle applies to every producing slice. |
| 58–59, 61–62 | FML-012, extended by FML-013 for stale-only cleanup. |
| 60 | FML-013. |
| 67 | Every slice introducing strings; FML-008 combined qualification. |
| 68 | FML-001 and final production measurements in FML-008. |

Ticket drafting reused one existing read-only Scout for a bounded sweep of
committed-write and Settings lifecycle call sites. The lead confirmed the reported
root reconciliation method locations; all decomposition, contracts, writing and
review stayed lead-owned. No new agent was spawned. This documentation change
uses link, coverage, numbering, dependency and whitespace checks only. No feature,
model or native implementation gate has run, and no commit is authorized here.
