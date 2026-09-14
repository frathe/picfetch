# Find more like this: implementation continuation

Status: implemented; current review/CI gate is recorded in [PR #25](https://github.com/frathe/picfetch/pull/25). Ronin approved proceeding
from his overall review on September 14, 2026, waiving exhaustive per-item judgments.
Branch: `feature/find-more-like-this`
Baseline: `1decc92ac85beec184bf8adc97d16504c108d1d2`

## Scope and authorization

Implement the approved [specification](../docs/find-more-like-this/spec.md) and
MVP [ticket sequence](../docs/find-more-like-this/tickets/README.md), then open
a GitHub PR and run the repository's GitHub Codex review loop. The request
authorizes commits, pushes, PR creation and review-loop interactions. The
positive/negative feedback extension remains deferred; merging and release are
outside the request.

Deep SDD and TDD at the already approved seams. The
[execution map](../docs/find-more-like-this/ticket-execution.md) supplies the
per-ticket contracts, files, dependencies and executable gates. Root owns
design, implementation, both Standards and Spec review axes, fixes and final
verification, following the repository's review-ownership rule. Focused local
checks accompany each slice; native Linux/amd64 CI supplies the complete race
suite during the authorized GitHub review loop. Inspect every changed code file
with GoLand and retain unavailable qualification explicitly as unverified.

## Quality frontier

Ronin reviewed the report and concluded that the results “all in all look quite
good” and “we can go that route.” This supersedes the earlier choice to complete
all per-item judgments first. Proceed with the accepted content-ranking approach;
individual labels, median relevance precision and exhaustive semantic case
coverage remain unmeasured. Do not claim the initial 0.6 P@10 target passed.
FML-002 is admitted by this explicit product decision. No new model or assets
are authorized or needed. See the [evaluation record](../docs/find-more-like-this/evaluation.md).

## Reconnaissance requiring reconciliation

MA-026's extraction and its CI qualification are complete, but that does not
establish every capability assumed by the search execution map. The read-only
source sweep located `explorer.Feature.EnsureReady(func()) bool`, `State`,
`Cohort`, `Sources`, `Close`, `Stop`, `Wait` and `Settle`. It found no frozen visit
capture/restoration or browsing-preserving suspension operation. `Close` retires
and clears the analysis/map state. Verify and reconcile this prerequisite
contract before the FML-002 integration; do not assume the proposed APIs exist.

Grid currently exposes host-index selection/results and display-index highlight,
query and scroll observations. `OpenSubset` preserves root collection order;
it does not support caller-ranked order or aggregate visit restoration. The
planned ranked-visit seam is therefore still new implementation work.

## Progress and cost

- [x] Locate the accepted specification, tickets and pre-agreed seams.
- [x] Create the requested feature branch from a clean main checkout.
- [x] Fetch and confirm the remote baseline; authenticated HTTPS works after SSH failed.
- [x] Open the existing local judgment report and verify visible images load.
- [x] Record Ronin’s qualitative proceed decision and the unmeasured limits.
- [x] Reconcile the actual Explorer prerequisite interfaces.
- [x] Implement and verify the approved MVP ticket sequence.
- [x] Complete local Standards/Spec review and GoLand inspections.
- [x] Open the PR and record review-loop fixes with verification evidence.

The latest pushed commit's final review gate is tracked live in
[PR #25](https://github.com/frathe/picfetch/pull/25): a fresh clean Codex code
review after the last finding disposition, completed security review without
actionable findings, clear Qodana/CodeQL results, and passing required CI.
This linked gate remains required; historical checks below do not substitute
for the latest commit's results.

One bounded read-only Scout was used while root read the full specification and
prepared the quality review. This is one recon spawn beyond the execution map's
zero implementation-spawn budget: it returned existing symbols and missing APIs
across Explorer/Grid/root, with no writes or delegated design/review. At that initial frontier no implementation agents or feature tests had been used.
The bounded implementation tasks and later verification are recorded below.


## Bounded ranker task

Owner: one implementer; root retains review and fixes. Files: new
`internal/similarity/search_rank.go` and `search_rank_test.go` only. Contract:
`RankSimilar(context.Context, Item, []Item, int) ([]Match, error)` where Match
holds Path and Score. AC-02/V02 specifies exact 768-dimensional cosine, immutable
inputs, bounded top-k, valid reference, skipped invalid candidates, deterministic
path ties, deduplication and cancellation. Verify with
`go test ./internal/similarity -run '^TestSearchRank' -count=1` and its benchmark.
Budget: one spawn, two lead review rounds, no full suite.

Delegation gate: G1 yes (bounded explicit contract); G2 yes (rank fixtures and
independent full-sort oracle); G3 yes (two exclusive new files); G4 yes (Item
and ranking contract only); G5 yes (root is working on worker/visit integration).
Rules S/W: an algorithm and behavioral tests cannot be generated by a text
transform, and this plan contains no implementation. This exceeds the earlier
zero implementation-spawn assumption by one independent task.

## Bounded visit/history task

Owner: one implementer. Exclusive files: new visualsearch feature and tests,
maximum three files in `internal/ui/visualsearch`. Contract: feature-owned
immutable visits/history, SearchProvider lifetime, latest-query delivery, and
queued host callbacks, with no displayed strings. Root supplies Grid/origin
capture/presentation and external error rendering. AC-05/V05 and AC-07/V07
provide the test seam. Verify `go test -race ./internal/ui/visualsearch -count=1`.
Budget: one spawn, two lead reviews, no full suite. G1–G5 pass: fixed API prompt,
observable host/provider tests, exclusive package/files, compact independent
state-machine context, and root working on native transport/cache composition.
Rules S/W pass: behavioral state machine, no scripted transform or code in plan.

## Implementation evidence so far

- Session tracer red: reference-first distinct preparation was empty under the
  stub. Green: `go test ./internal/similarity -run '^TestSearchSession' -count=1`.
  The 99/100/101/200 publication boundaries pass with cached candidates.
- Grid tracer red: ranked opening stayed hidden and returned root order.
  Green: `go test ./internal/ui/grid -run '^TestRankedVisitInitial' -count=1`.
- Explorer suspension red: the session remained current (the first draft also
  exposed an unjoined worker timeout). Green: focused Suspend/OpenCopies/
  CloseReopen race tests. Suspension retains the live map/camera/cohort; its
  analysisDone signal observes native return, and successor analyses serialize.
- Actual menu admission red: Find more like this was missing from Actions.
  Green: highlighted-reference admission, multi-selection refusal and both
  actual menu/registered-shortcut round trips. Ranked opening/stepping,
  image Escape, final Back and original highlight restoration pass.
- Favorite capture red: explicit-list naming did not open. Green: a caller
  mutates the provided slice and removes the host list while naming; saved
  membership remains the originally captured source.
- The independent ranker passed its known-value and full-sort oracle tests;
  root reran `TestSearchRank`. Its 10,000-vector benchmark measured 13.03 ms/op,
  768 B/op and one allocation on this Apple M5 Max. These are ranking-only
  measurements, not full search latency or semantic quality scores.
- The isolated visualsearch feature passed `go test -race
  ./internal/ui/visualsearch -count=1`. Its agent negatively verified the query
  identity and writer-completion guards through temporary Go overlays. Root
  retains final review and independent verification.

No MVP ticket is marked complete until its remaining acceptance cases and
integration checks pass. At that stage persistent cache, management UI, action/lifecycle coverage, native
qualification, inspections and the GitHub loop remained open.


## Final implementation and verification record

The implementation now composes ranked Grid browsing, reference history, captured
Favorite actions, persistent general/Favorite analysis and Cache settings. The
existing file-write reconciliation retires search on affected committed writes,
including aliases/stale writes, while unrelated exports retain it. Sort/source
replacement and shutdown invalidate search; native successor barriers serialize
Explorer and search without blocking UI. Cache maintenance closes root epochs
and joins current producers before removing managed records.

Additional bounded tasks used the already active ranker implementer for the two
cache regression files and one native transport test file, while the visit
implementer built only the new analysiscache package. These were implementation
and test tasks, not delegated reviews; root retained all assessment and fixes.
Their exclusive file scopes avoided shared production edits. The original
zero-spawn execution estimate was exceeded to parallelize independent work.

Observed additional red/green evidence:

- Cache usage/clear and persistent reopen failed their initial behavioral stubs,
  then passed. Eleven cache regressions cover old writers, held producer joins,
  cancellation, partial results, confinement, temporary bytes, conservative
  stale cleanup, LRU, Favorite priority/promotion and both cache preferences.
  Cancellation while waiting for a writer exposed a missing Canceled flag;
  root fixed all maintenance exits. Four temporary overlays proved the epoch,
  managed-name, Favorite-priority and disconnected-source guards.
- Cache controls had four substantive reds: missing measured totals, accepted
  limits never persisting, missing Clear, and automatic pressure canceling user
  cleanup. All are green. Automatic reservations now coalesce behind explicit
  maintenance; production completion prunes retired jobs. Eight package tests
  cover queued retirement, two held writer barriers, stale view delivery,
  confirmation, cancellation, limits and usage.
- Restored cohort Back failed because reconstruction cleared its callback.
  Grid visits now retain their cohort commands. Existing Grid viewport tests
  caught four extra pixels from an empty top container; its visibility now
  follows actual content. Menu composition tests include the new action.
- Five root integration tests cover menu/shortcut admission, round trips,
  progressive image-order retention, copy/Trash/comparison targets under result
  updates, and source/sort retirement. Grid progress is checked in the actual
  visible container tree, with filtered/evicted selections matched by identity.
- Four native transport tests verify control EOF, invalid/truncated streams,
  session/revision validation, readiness and canceled process/control joins.
  Five temporary overlays failed when those guards were removed.
- Real macOS native search: five synthetic images, two references per worker;
  cold 0.506 s with 0/5 reuse, reopened general cache 0.181 s with 5/5 reuse.
  Offline enforcement was true and each reference returned four other files.
  The earlier 446-image semantic report remains the qualitative evidence;
  these synthetic checks do not establish relevance precision.

Focused race regressions passed for similarity, Grid, menus, Favorites, Explorer,
visualsearch and analysiscache. Root FindMoreLikeThis integration passed in
9.319 s. The full preferences run exposed one expected round-trip fixture update
for the new defaulted limit; the corrected full preferences and Settings suites passed. Locale/manual checks passed.
`make verify-build` passed format/generated/notices checks, vet and build.
`make check-test-shards` passed with 686 UI runnables on three shards. The complete
Linux/amd64 race suite belongs to GitHub CI for this authorized review loop.

GoLand inspected all 68 changed Go files with errorsOnly:false, including weak
warnings. Root corrected the reported nil-test premise, comment ownership,
partial-result handling and padding. Final incremental checks are recorded in
the PR loop. The two intentional partial-result warnings in cache tests have
narrow documented suppressions. New tests have exact Qodana exclusions.

See [dependency qualification](../docs/find-more-like-this/dependency-qualification.md)
for unchanged pins, shipped notice paths and the existing HEIC/AVIF/font notice
issues. Those remain release qualification work; no dependency or asset was
substituted. Quantitative relevance remains unmeasured by Ronin's explicit
qualitative proceed decision. Live native acceptance outside macOS and final
GitHub review/CI remain to be recorded.

Final lead review found and fixed two further boundary defects: a canceled search
successor now retains the external Explorer retirement barrier, and the ordinary
Favorite-list adapter uses the current filename-filtered ranked Grid. Both new
behavioral regressions failed before the fixes and pass under the race detector.

Visual QA exposed an existing thumbnail-wrapper incompatibility with the pinned
x/image v0.46.0 scaler: its RGBA64 destination branch skipped generic Preview
sources. A temporary painter probe isolated transparent scaling output before
composition. Preview now implements the standard RGBA64Image interface, preserving
source-version metadata. A scaling regression failed before the fix; the ranked
Grid test now checks captured colored pixels instead of detached cell fields.
The isolated shader-free capture then displayed both ranked thumbnails. No
upstream dependency or renderer workaround is shipped.

Final incremental focused race checks passed for wrapper scaling, ranked Grid,
all five root search integration tests (including filtered Favorites), and the
search/Explorer predecessor barrier. Locale/manual checks and verify-build passed;
final inspections include the added thumbnail-wrapper and Settings test changes.
The PR loop will append the live GitHub results below.

The final GoLand pass reported one existing 16-line test-setup duplicate in
internal/favthumbs/store_test.go; its exact file already belongs to the repository
DuplicatedCode test exclusion. No new actionable inspection remains.

## GitHub review evidence

Implementation committed as `9f34b35a8afb164e7cc3ab6b01a0dd5248363064` on
`feature/find-more-like-this`; [PR #25](https://github.com/frathe/picfetch/pull/25)
is open. The configured SSH signing agent refused operation, so the authorized
feature commit is unsigned. Generated local Fyne metadata is excluded.

The initial Qodana run `34872016750` passed; the downloaded post-suppression
`qodana.sarif.json` has zero results. CodeQL actions/Go completed successfully.
Alert 7 claimed an unchecked uint64-to-int conversion, but the existing
`value > uint64(^uint(0)>>1)` guard rejects values above the architecture's
MaxInt before the cast. The lead documented that proof, dismissed the false
positive, and resolved its thread without changing correct production code.
This disposition requires a fresh Codex review after the current round.

Initial CI validation, Windows guards, macOS arm64 guards and macOS amd64
qualification passed. Linux race and the initial Codex code/security reviews
were still running at this record's update. The final results and any follow-up
fixes remain discoverable in the linked PR; this archive records completed
implementation, while the active GitHub review task remains in todos.md.

The initial security review completed without findings. Linux ui-2 exposed one
missed menu-structure fixture (`Actions menu items = 19, want 18`), not a data
race. Its expectation now includes Find more like this and verifies the action
starts disabled without a reference. Focused menu and search integration race
checks pass after the update. GoLand found only two existing test-setup duplicate
fragments in menu_test.go, already covered by that exact Qodana test exclusion;
the changed expectation has no actionable inspections. The other initial CI
jobs are recorded in the PR's check history.

## Interrupted review recovery

Resumed the existing branch and PR, retaining the unfinished filesystem changes.
The follow-up places a purple-outlined reference before up to 30 other matches,
hides completed progress, applies cache limits only on Enter or explicit Apply,
restores the Explorer cohort's retained map, delivers pending final rankings when
generic overlays close, and preloads from the captured search order. The overlay
observer has cancellation, acknowledged UI delivery and tracked shutdown/harness
completion. ARCHITECTURE.md and the specification describe these final behaviors.

The general-cache writer inventories once, updates committed byte totals, and
invalidates other writers through a lease-protected revision. A 100/1,000-record
benchmark measured 38.6/373.4 ms; a temporary overlay of the previous writer
measured 43.2/2,055.6 ms on this Mac. These are bounded local measurements.
The replacement-pressure finding is rejected: specification contract 14 and
FML-008 explicitly count managed temporary bytes, so the old record and its
staged replacement must both fit until rename. The existing write-budget test
verifies pressure preserves the old record; crediting it before rename would
violate that accepted contract. A source comment now explains this distinction.

CI run 34873011910's cancelled ui-3 stream stopped at
TestVisualSimilarityExplorer/replacement_discards_late_map. Its test awaited the
successor before releasing the predecessor, contradicting the new native-worker
exclusivity barrier. The recovered test releases the predecessor first. The full
TestVisualSimilarityExplorer now passes locally with -race in 106.247 s.

Focused -race checks passed for analysis-cache policy/management, ranked Grid,
duplicate badge layering, and all FindMoreLikeThis integration cases. Seven
temporary overlays independently failed on missing overlay return, cohort return,
ranked preloads, submitted limits, cross-writer accounting, reference-first order
and completed-progress hiding. No overlay changes entered production files.
Translation/manual checks, make verify-build and make check-test-shards pass
(686 runnables, three shards). The Grid screenshot was inspected after the same
whole-content refresh used by root; the reference precedes 30 matches and the
header/progress/action layout is readable. The render test now mirrors that refresh.

GoLand inspected all 15 changed Go files with errorsOnly:false. Its only remaining
weak warnings are existing duplicate setup fragments in grid/dupes_test.go,
already in the exact Qodana exclusion. The extra duplicate warning in
visualsearch.go was removed by sharing its overlay-open observation.

One bounded read-only Scout collected cancelled-job logs, post-suppression SARIF
and CodeQL metadata while root validated the fixes. G1-G5 pass: compact factual
artifact task, gh/SARIF output oracle, no source writes, small independent scope,
and no duplicated root context. No review or fix was delegated. Qodana run
34873011952 has zero post-suppression results; CodeQL alert 7 remains dismissed
with the previously recorded checked-conversion proof. Fresh review and complete
native CI for the follow-up commit are still required; the live PR records them.

Follow-up `f6fd776` passed Validation, Windows/macOS guards and both CodeQL
analyses; Qodana run 34878068256 again has zero post-suppression results.
Linux ui-2 exposed a cleanup-only nil dereference: startup/shutdown fixtures
build the production viewer without installing the optional test overlay queue.
TestShutdownStopsClipboardAdmission reproduced it locally before the fix.
The common drain now checks for that optional queue after stopping/joining its
worker; production behavior is unchanged. Focused race coverage includes every
direct startup-fixture drain caller and all search integration tests. GoLand
reports no findings in the corrected harness. Fresh CI/reviews remain the gate.

## Second review round

All native CI checks passed on `d4fe05a` (CI run 34878638889); Qodana run
34878638872 has zero post-suppression results, CodeQL has no actionable alerts,
and the security review completed. The fresh code review reported three further
cache-workflow findings. Root confirmed and fixed them at the existing seams:

- Settings usage inspection queues behind an active automatic eviction and
  coalesces across reopening; closing the tab drops its queued inspection.
  TestAnalysisCacheManagementLimitAutomaticReserveSurvivesViewOpen failed with
  cancellation before the fix; the complete analysiscache race suite passes.
- Inventory retains the same Favorite directory handle used to read membership,
  and stale decisions revalidate its exact file-list version, including an
  immediate check before unlink. An unknown/replaced list is conservatively
  skipped. TestAnalysisCacheStaleRevalidatesFavoriteAfterInventory failed by
  deleting a member restored through a concurrent Save after inventory; the
  complete analysis-cache regression group now passes with -race.
- Explorer receives the enabled general-cache directory and uses the shared
  representation reader. Compatible general hits are promoted when that newly
  saved Favorite is first analyzed, avoiding a competing post-save producer.
  Analyzer misses retain the existing Favorite-only writes. The native
  TestVisualSimilarityExplorerLocal/new_search_favorite_reuses_general_analysis
  reproduced repeated inference before the fix. It now verifies two reused
  sources and a later Favorite-only worker with zero inference attempts. This
  case plus favorite_cache and favorite_cache_ui passes with -race using the
  installed model and existing OS network denial (25.682 s on this Mac).

The prior Scout collected only the existing save/open/cache call paths while
root implemented the other two fixes. This was a bounded factual recon follow-up,
not delegated assessment or implementation. Root owns every review and fix.
No dependency, model asset, new worker or new top-level UI runnable was added.
Complete focused -race package suites pass for similarity, analysiscache,
Explorer and visualsearch, together with root search/lifecycle integration.
make verify-build passes. GoLand inspected all 13 changed Go files including
weak warnings; explicit partial-store checks and Boolean field grouping resolve
its new warnings. Existing duplicates in explorer_local_test.go remain covered
by that exact test-file exclusion. The native regressions were observed failing
before their fixes; fresh CI and code/security review are still required.

Ronin also requested inspection of fyne_metadata_init.go. Its ID, version,
build and migration flag match FyneApp.toml. Its 379,395-byte icon exactly
matches the pinned Fyne v1.7.2 512px Lanczos conversion of assets/appIcon.png.
The generator's source creates this transient file and normally removes it
after packaging. It has no unique feature changes and remains local, outside
the PR.


## Third review round

All CI checks passed on `ba86788` (CI run 34881372543). Its security review
completed, CodeQL has no actionable alerts, and Qodana run 34881372650 contains
zero post-suppression SARIF results. The fresh code review reported three
additional issues; root reproduced and fixed each:

- Intermediate source validation checks only the reference and up to 30 ranked
  matches. Each final publication still validates the entire prepared scope.
  TestSearchSessionBoundsPartialValidationAndChecksFinalScope uses 301 real
  source paths: a changed non-result survives intermediate batches but rejects
  final publication, while changed reference/result paths reject the next batch.
  Its unranked case failed with the previous growing full sweep.
- Cache maintenance enumerates Favorite directories independently, retaining
  partial errors while inspecting and cleaning healthy peers. Missing definitions
  are skipped; unreadable membership is retained as unknown for stale cleanup.
  TestAnalysisCacheMaintenanceUnreadableFavoriteKeepsHealthyPeers failed when
  one self-referencing file-list symlink hid the healthy Favorite from inventory.
  Both Clear and Remove stale now complete the healthy removal and report the
  inaccessible peer. The fixture also works under root-run Linux CI.
- History retains visits without obsolete preparation snapshots. Back preserves
  the session's current progress, and preparation/readiness from its retained
  worker can still arrive after the reference was abandoned, without publishing
  that query's ranking. Both completion-before-Back and completion-after-Back
  regressions failed before their fixes and now pass.

Complete focused -race suites pass for similarity (6.818 s), visualsearch
(1.694 s) and analysiscache (2.431 s); root FindMoreLikeThis integration passes
in 11.252 s. All eight changed Go files have zero GoLand findings, including
weak warnings. make verify-build passes. No new test files or top-level root UI
runnables were added.
The existing Scout retrieved only connector completion metadata and clean-result
artifact locations while root implemented fixes; no assessment was delegated.
Fresh code/security review and CI are required on the next pushed commit.


## Fourth review round

All CI passed on `8e8832f` (run 34883680012). CodeQL has no actionable PR alerts;
Qodana run 34883679946 has zero post-suppression SARIF results. Security review
completed at 19:03:50 UTC; code review completed at 19:04:36 UTC with three new
findings. Root confirmed and fixed all three at the existing cache seams:

- Producer inventory now enumerates Favorite directories independently through
  a retained parent handle and preserves healthy records alongside per-entry
  errors. Incomplete membership disables general reads/writes for that producer
  so an unknown owner cannot be treated as loose. Both unreadable and corrupt
  definitions reproduced lost reuse or a missing warning before this change.
- Favorite opt-out rejects a member's retained general record. Explorer supplies
  membership separately from its cache opt-out to the worker, and analyzer writes
  respect that opt-out. Unit coverage verifies retained bytes survive disabling
  and work again after re-enabling. The native new-search-Favorite regression
  also reproduced general reuse after opting out; it now proves fresh native
  preparation through the production settings/Explorer/worker path.
- Clear and retiring retunes create the configured owned roots before acquiring
  their shared leases. A second opener cannot create an uncoordinated cache
  during the maintenance pass. The regression uses a single nonblocking native
  lock attempt with cancellation on contention, covering both roots and both
  maintenance operations without timing guesses; both modes failed before.

The complete similarity, analysiscache, Explorer and visualsearch package suites
pass with -race (7.261/2.093/5.167/2.109 s). Native general-to-Favorite reuse,
opt-out and the existing favorite_cache_ui case pass in 14.568 s. The native
opt-out additionally verifies Favorite record identities/mtimes remain untouched;
a temporary overlay removing only the analyzer write guard failed this assertion,
and the restored production path passes in 7.651 s. No overlay changed source.
make
verify-build passes. GoLand inspected all nine changed code files including weak
warnings; only the existing exact-excluded explorer_local_test.go setup duplicates
remain. No new test files, root UI runnables, dependencies or worker lifetimes.
All implementation and assessment stayed with root. Fresh code/security review
and full CI remain the next pushed commit's gate.


## Fifth review round

All CI passed on `b3c1d66` (run 34885906209), its security review completed,
CodeQL has no actionable PR alerts, and Qodana run 34885906217 has zero
post-suppression SARIF results. The next code review reported four findings.
Root validated and fixed them; the required review loop supersedes the original
two-round estimate because each fresh review exposed additional confirmed work.

- A fixed reference now ranks only newly prepared items plus its retained top-30;
  a reference change resets ranking to one full prepared-scope pass. The 3,001-
  source regression exceeded its linear context-checkpoint budget before the
  fix, and now matches a full ranking at every publication, including a reference
  change at source 350. Three-iteration synthetic benchmarks on this M5 Max
  measured 1,000 vectors at 9.966 ms before / 3.480 ms after, and 10,000 at
  682.308 ms before / 27.208 ms after. These session/ranking measurements exclude
  decoding, model inference and cache I/O; they are not end-to-end search claims.
- Favorite command admission captures CurrentFiles once. Ranked source lookup
  uses one collection pass and a small requested-path map. The real Add Current
  List regression observed 49,176 Path calls for a 4,100-source opened visit
  before the fix; it now stays within a generous two-pass bound. Existing tests
  retain filtering, invocation-time files and overwrite behavior.
- The loose-persistence toggle saves its preference immediately and retires
  active producers independently of inspection/eviction success. Its tracked
  completion waits both producer barriers even when Settings closes. The
  incomplete-inspection regression failed before the fix; a second temporary
  overlay dropping only those barriers also failed after closing Settings.
  Production passes the same guard (2.024 s with -race).
- AGENTS.md now lists all eleven feature UIQueue exceptions and documents their
  captured delivery, visual-search readiness, analysis-cache quiescence and
  root settlement order. The writing-for-agents guidance was applied; no
  additional approval was inferred or requested.

Complete focused -race suites pass for similarity (7.170 s), analysiscache
(2.387 s), favorites (5.940 s), and visualsearch (2.641 s). Root FindMoreLikeThis
integration passes in 12.130 s. make verify-build and Docker shard validation
pass (686 root UI runnables, three shards). GoLand inspected all
nine changed code files, including weak warnings; the two new error-string
warnings were fixed and rechecked. Only previously excluded duplicate setup in
favorites/favorites_test.go remains. No new test file, root UI runnable,
dependency or untracked worker was added. Root owns all reviews and fixes.
The next pushed commit still requires fresh code/security review and CI.


## Duplicate CodeQL alert and current gate

On `f02fb65`, CodeQL recreated dismissed alert 7 as alert 8, at the unchanged
parseLimit conversion. The explicit `value > uint64(^uint(0)>>1)` guard rejects
values above the current architecture's signed-int maximum before narrowing;
`value > ^uint64(0)/mebibyte` separately protects conversion to bytes. A direct
comparison confirmed this function is byte-for-byte unchanged from `b3c1d66`.
Its invalid/overflow/valid-input regression passed again (0.711 s). Alert 8 was
dismissed with this proof, its thread resolved, and the CodeQL check returned
to green. Qodana run 34889164772 has zero post-suppression SARIF findings.
A fresh code review after this disposition remains part of the live PR gate.
No application code or broad suppression was changed for the duplicate report.


## Recovery round 6: visit identity, partial maintenance and worker boundaries

The fresh code review of `f02fb65` completed with six findings. The lead
confirmed and fixed each one:

- Captured Grid visits retain duplicate occurrence ordinals for selection and
  highlight. The first and second occurrence regressions both failed on the old
  path-only restore (`[0 2]` selected and highlight 2); each now restores its
  original single occurrence. Ranked menu state disables both duplicate commands.
- Cancellation no longer launches an uninterruptible final inventory. A progress
  callback cancels after one unlink and adds a record outside that captured
  inventory: the old implementation rescanned it; the fixed report retains its
  known two-record remainder. Cancellation during a second inventory also keeps
  the prior measured usage, covered by a separate red/green regression.
- General and Favorite inventory completeness are tracked independently.
  AppliedLimit is published only after successful general maintenance and accepted
  by the Cache tab even when Favorite inspection returns a partial error. A real
  unreadable Favorite definition plus an oversized general record verifies both
  eviction/persistence and the retained incomplete-status notice. Failed general
  retunes still do not persist a limit.
- Search admission rejects requests above the existing 64 MiB JSON input bound
  before starting a subprocess. This remains a separate transport limit from
  the 256 MiB vector budget and is now explicit in the spec. The decoder renews
  its allowance per message, accounting for prefetched bytes; a pair of large
  valid messages followed by a reference query reproduces the old lifetime-cap
  failure and passes with the fix. The search writer reuses its checked payload.
- Explorer installs its canceled-trial exit observation before predecessor waits,
  and emits it before its completion channel closes. A canceled-before-admission
  regression verifies the trial summary records cancellation without inference.

Focused race suites pass: similarity 9.097 s, analysiscache 2.545 s, Grid
6.356 s, menus 1.720 s, root FindMoreLikeThis 12.785 s. The final cancellation
adjustment passes analysis-cache regressions (1.671 s); Explorer feature
regressions pass (4.206 s). GoLand inspected every changed code file including
weak warnings, with no findings. Additional menu cases and final repository
checks are recorded below when complete. No new test file or root runnable was
added; existing exclusions and the 686-test shard inventory still apply.

The existing Scout was used only to locate protocol limits and framing tests;
all six assessments, regression design and fixes remained with the lead. The
extra review round was necessary because a review with findings cannot satisfy
the final gate. Concurrent HEIC implementation/dependency/doc edits and the local
metadata ignore rule are preserved and excluded from this recovery commit.

The exact staged PR tree was also checked in an isolated worktree, retaining the
PR's existing dependency versions. All verify-build steps pass. Focused -race
checks there pass for similarity (8.394 s), analysiscache (2.241 s), Grid
(4.532 s), menus (2.427 s), Explorer (2.481 s), and root UI (11.998 s).
GoLand's additional menu-test inspection is clear, for sixteen changed code files
in this round. Host-native shard discovery includes macOS-only runnables; the
prescribed Docker Linux/amd64 command supplies the authoritative shard check.


## Recovery round 7: partial inventory and source admission

All CI passed for `2d9a7cb` (run 34892960877); Qodana run 34892960799 has
zero post-suppression SARIF findings, CodeQL has no open PR alerts, and security
review completed at 20:36:06 UTC without actionable findings posted. Its code
review completed at 20:38:54 UTC with three additional findings, all confirmed
and fixed by the lead:

- A canceled inventory under the lease now publishes its partial observation
  and marks it incomplete. Cancellation before such an inventory is complete
  also marks preliminary counts incomplete. A writer added during the earlier
  inspection reproduces why its old snapshot cannot be presented as current.
  Completed inventories still retain accurate tracked remainders after unlinks.
  The Cache tab reports both cancellation and incompleteness when applicable.
  The previous implementation failed the report and rendered-status regressions;
  the old-management overlay also checks cancellation while waiting for a writer.
- Stale cleanup no longer uses an existing parent directory to establish source
  availability. A renamed source directory followed by an empty replacement
  models an unmounted volume with a surviving mount point and reproduced the
  erroneous unlink. The current payload has no volume identity, so all missing
  paths are conservatively reported unavailable. Permanently deleted sources
  can therefore retain analysis until full cleanup or general-cache LRU eviction;
  this limit is explicit in the spec. Changed source versions, invalid payloads
  and obsolete Favorite memberships remain removable. The healthy-peer fixture
  now uses a changed source version instead of ambiguous absence.
- First-use setup compares the current admissible reference with the captured
  reference as well as checking collection generation. Integration cases retain
  the same generation while navigating between two images during setup: an
  unchanged reference starts normally, a changed reference does not start the old
  search, and a new explicit request admits the newly displayed image.

Focused race checks pass: complete similarity suite 9.166 s, analysiscache
2.262 s, and root FindMoreLikeThis regressions 13.090 s. All six changed code
files have clear GoLand inspections, including weak warnings. New assertions
remain in existing test files/subtests; no root runnable, dependency, worker,
package or translation key was added. The implemented feature is filed under
Done in todos.md; its live review gate remains on PR #25. The new commit still
requires a fresh clean code review, security completion and complete CI.

The exact PR snapshot also passes make verify-build (format, generated assets,
notices, Qodana exclusions, vet and build) in the owned temporary worktree.
