# PR 25: cache ownership and browsing transitions

Status: implementation and local verification complete. Final remote acceptance
is recorded on [PR 25](https://github.com/frathe/picfetch/pull/25) for its latest commit. Route: Deep (cross-package
refactoring). Evidence: [all 28 Codex findings](../docs/find-more-like-this/pr25-architecture-review.md).

Deliverable: put Favorite ownership, cache operation lifetime, and ranked browsing
decisions behind their owning modules while preserving the existing fixes.

## Decisions and limits

| Decision | Contract |
| --- | --- |
| One Favorite inventory | Retain directory handles and captured list versions; return healthy entries alongside errors. Unknown membership forbids general fallback, but remains inspectable/clearable. |
| Store owns routing | An internal write scope selects Favorite-only (Explorer) or all enabled stores (search); both consumers use store read/write methods. No dependency or wire-policy change. |
| Maintenance owns intent | Inspection is replaceable; eviction and policy retirement survive Settings close; cleanup and explicit retuning belong to their view. Coalesce automatic reserve requests. |
| Outcomes describe effects | Keep committed removals, cancellation, per-store observation completeness, and applied limit independent. A closed view cannot persist a late explicit retune. |
| Root owns browsing | Preserve separate live ranking, frozen image order, and saved Grid visits. Share per-operation context/capabilities at root; no global controller or event bus. |
| Existing safety boundaries | Preserve leases/epochs, staging-byte budgets, unavailable-source retention, source versions, cancellation/barrier joins and bounded transport/ranking. |

No new decoder, dependency, cache database, license resolution, release, or merge.
This refactor cannot guarantee absence of bugs or qualify outstanding shipped
dependency notices. Existing release qualification remains open.

## Tasks and acceptance

### 1. Favorite ownership and producer routing

Owner: T0 inline. Files: `internal/similarity/cache*.go`, `analyze.go`,
`search_worker.go` and existing cache tests.
Contract: `openFavoriteInventory(ctx, dir)` owns enumeration and root cleanup;
`openRepresentationStore(ctx, policy, scope)` owns every read/write decision.
Test: healthy/broken peers, empty/unknown membership, opt-outs, promotion,
Favorite-only misses, version changes, linear accounting and staging pressure.
Verify: `GOCACHE=/private/tmp/picfetch-h265-audit-buildcache go test -race -tags no_emoji -count=1 ./internal/similarity -run 'TestAnalysisCache|TestSearch'`.
Budget: no implementation spawns; one lead review plus necessary fixes.

### 2. Maintenance intent and transaction outcome

Owner: T0 inline. Depends: 1. Files: similarity maintenance and analysiscache UI
feature/work files, existing lifecycle/policy/feature tests.
Contract: explicit private operation intents own admission, lifetime and provider
dispatch; a maintenance transaction owns observations, committed removal accounting
and final report. Preserve public report/provider compatibility where sufficient.
Test: cancellations before/after lock, quiescence, removals and reconciliation;
close/reopen while cleanup, retune, policy retirement or eviction runs; unrelated
Favorite failures must not undo committed general policy.
Verify: `GOCACHE=/private/tmp/picfetch-h265-audit-buildcache go test -race -tags no_emoji -count=1 ./internal/similarity ./internal/ui/analysiscache -run TestAnalysisCache`.
Budget: no spawns; one lead review plus necessary fixes.

### 3. Browsing context and transitions

Owner: T0 inline. Independent of 1/2 until final gate. Files: root visualsearch,
navigation/action/menu adapters and existing root/feature/Grid tests as needed.
Contract: root captures current browsing order/capabilities once per operation;
search enter/back/exit/source replacement preserve complete surface state and
independent live progress. Search feature retains query/history ownership.
Test: cohort -> partial search -> opened image -> final publication -> overlay
dismissal -> Back -> Exit; duplicate identity, action targets and preload neighbors.
Verify: `GOCACHE=/private/tmp/picfetch-h265-audit-buildcache go test -race -tags no_emoji -count=1 ./internal/ui ./internal/ui/visualsearch ./internal/ui/grid ./internal/ui/menus -run 'TestFindMoreLikeThis|TestVisualSearch|TestRankedVisit|TestApply|TestActionsMenu|TestMosaicSources'`.
Budget: one read-only scout for command/menu/source ownership; implementation and
review remain lead-owned. Scout gate: short factual prompt, file/line oracle, zero
edits, independent cold consumer context, shell located but did not trace callers.

### 4. Verification and GitHub review loop

Owner: T0. Depends: 1/2/3. Files: architecture, todos, this evidence record,
Qodana exclusions/UI shards only if tests added. Run focused regressions and
negative verification of new guards, `make verify-build`, `make check-test-shards`,
and GoLand inspections including weak warnings on changed code. Push authorized
fix commits; inspect full native CI, post-suppression Qodana, CodeQL, security and
a fresh clean Codex code review of the final commit. No broad local race duplicate.

Task graph: `1 -> 2 -> 4`, `3 -> 4`.

## Evidence and ledger

Budget: one scout, zero delegated implementations/reviews; focused local suites
and complete native GitHub CI. Actual: one read-only command ownership scout
completed; all design, implementation and review lead-owned. Local verification results are recorded below; remote results are linked from PR 25.

### Local implementation evidence

- Baseline focused cache/maintenance/search tests passed.
- Producer-scope matrix failed before enforcing Favorite-only write scope, then
  passed with all enabled/disabled combinations. Shared inventory preserves
  healthy peers beside damaged definitions; failed leases permit reads but deny writes.
- Inspection/mutation guard failed for cleanup and explicit retune under the old
  generic cancellation rule; explicit intent scheduling makes both pass.
- Direct ranked duplicate command and repeated preload snapshot guards failed
  before the shared browsing restriction/order changes, then passed.
- Combined cohort/search/opened-image/final/overlay/Back/Exit regression passes;
  existing identity, source replacement, action capture and live-progress tests remain.
- Public cache reports remain compatible; an internal transaction now owns every
  observation, committed removal and final cancellation/applied-limit outcome.
- One scout completed factual command/source mapping. No delegated fixes or review.

### Local gate results

- Cache/search/maintenance/Grid/menu focused race suites passed (similarity 9.5 s;
  analysiscache 2.4 s; visualsearch 1.7 s; Grid 3.0 s; menus 2.3 s).
- Expanded root regressions including Explorer passed in 129.3 s. After tightening
  the shared keyboard command entry, Find-more-like-this/Actions passed again in
  24.4 s. The keyboard regression was observed failing before that final change.
- Real production worker test `TestVisualSimilarityExplorerLocal/new_search_favorite_reuses_general_analysis`
  passed: both records promoted/reused; Favorite-only reopen made zero inferences.
- Negative overlays removed the lease guard and restored Back beneath a modal;
  each dedicated regression failed for its intended effect. Repository files
  were never replaced by those deliberately broken overlay variants.
- `make verify-build` passed; `make check-test-shards` passed for 686 runnable
  root tests over three shards; Qodana test exclusions remain complete. No new
  root test function or test file was needed (new cases extend existing files).
- All 28 changed Go files inspected in GoLand, including weak warnings. Struct
  packing warnings fixed. One narrowly scoped `GoDfaErrorMayBeNotNil` suppression
  documents the inventory's always-owned partial-result contract; reinspection clear.
- Restricted tool-sandbox attempts could not bind an unrelated update-test server,
  access Docker or launch nested macOS sandbox workers. Appropriate native checks
  were rerun with those host capabilities, without weakening worker policy.
- No new dependency or shipped closure change. Full Linux race/native Windows and
  Intel/ARM macOS CI, Qodana/CodeQL and fresh bot reviews remain the remote gate.

- Final timing guard exposed stale progress when preparation completed after Back
  was deferred behind a popup. Presentation now carries only the frozen visit and
  reads live progress on application; the new regression was observed failing first.

## Fresh review follow-up (12d8c51)

The first complete remote round passed all 13 CI/analysis checks and security
review, but its code review added three confirmed P2 findings. These extend the
historical 28-finding assessment; they are not a clean final round.

- [Favorite saved during retained preparation](https://github.com/frathe/picfetch/pull/25#discussion_r4010389773):
  an explicit committed-save observer now sends a cache revision on the bounded
  query lane. Replacing a queued query retains the latest save revision. The
  worker acknowledges persistence, refreshes future routing, and reuses retained
  vectors for newly owned records, regenerating only previews. Opt-outs refresh
  ownership without persisting Favorite data. This is explicit new admission,
  not an automatic resurrection of a retired cache writer.
  Versioned inventory comparison limits completion to changed lists. An
  overlapping-Favorites regression proves reusable records reach the new list
  while the existing Favorite's record remains unchanged; missing-preview
  completion runs only for records unavailable in either enabled store.
- [Per-source search progress](https://github.com/frathe/picfetch/pull/25#discussion_r4010389781):
  transient counts use a 100 ms cadence; ranked/failure/final delivery stays exact.
  Virtual-time coverage failed with 1,001 progress messages, then passed with the
  bounded cadence and exact final counters. Reference-change and source-validation
  tests now trigger from actual preparation, independently of transient counts.
- [Cache inspection progress backlog](https://github.com/frathe/picfetch/pull/25#discussion_r4010389787):
  each work item owns one queued progress slot; a mutex protects the latest value
  across worker/UI. The 20,000-record test failed with 20,000 callbacks, then passed
  with one queued callback displaying the latest count and normal final completion.

Lead owns assessment and all fixes; no additional delegation. Real native UI save
and reopening tests pass with loose caching both off and on; the off case persists
two records and reopening makes zero inference attempts. Focused race suites pass
for similarity, cache UI, visualsearch, Favorites and root search integration.
Negative overlays verify save notification, query/save coalescing and refreshed
opt-outs. Existing duplicate-test fragments in explorer_local_test.go and Favorites
are intentional independent UI assertions and remain covered by the existing exact
Qodana test-file exclusions; the new duplicated trial setup was factored out.
Final focused race runs passed (similarity 9.2 s, cache UI 1.8 s, visualsearch
1.9 s, Favorites 2.2 s); both actual UI/native-worker save cases passed again
(2.5 s). GoLand reinspection of the final cache changes is clear.
Root browsing regressions passed under race (25.1 s); `make verify-build` and
`make check-test-shards` passed, including all 686 root-test assignments.
Fresh code/security reviews and CI on the follow-up commit remain the final gate.

### Review-loop continuation (1626f6e)

The current branch already contained the three fixes above. The lead checked
all 33 review threads, validated the three unresolved findings against the
current implementation, posted commit-specific verification evidence, and
resolved those threads. No additional application change was needed.

Focused regressions rerun on native Linux/amd64 with `go test -race -tags
no_emoji -count=1` pass: similarity (1.292 s), analysiscache (1.361 s),
visualsearch (1.041 s), and Favorites (1.070 s). The selected tests cover
explicit Favorite-save reuse, refreshed opt-outs, retained preparation,
progress cadence, inspection progress coalescing, bounded query/save delivery,
and the committed-save notification. The earlier red/green and native-model
evidence above remains applicable; no new code file requires inspection.

Qodana run 34909424665 has zero post-suppression SARIF results, successful
execution and exit code 0. The PR ref has zero open code-scanning alerts.
Validation, Windows and both macOS architectures passed on 1626f6e; Linux race
jobs and Codex reviews were still running when this record was updated.
Final acceptance of the latest pushed commit is recorded on
[PR #25](https://github.com/frathe/picfetch/pull/25), including fresh reviews
after these dispositions. No broad local race suite was duplicated. This
continuation used zero subagents; all assessment stayed with the lead.

### Review follow-up (044ed15): inspection/save and limit increases

The fresh review reported two P2 defects after all 13 CI/analysis checks passed.
Route: Standard; lead owns both fixes, with zero subagents. No new dependency,
worker, interface or package is planned. Existing test files will hold the guards.

- AC1: a committed ranked Favorite save reaches an active producer while a
  read-only cache inspection is running. A retired producer stays retired.
  Verify: `go test -race -tags no_emoji -count=1 ./internal/ui -run
  '^TestFindMoreLikeThisActionsCaptureRankedSources$/favorite-save-during-inspection'`.
- AC2: changing the captured cache limit in either direction retires the old
  producer, retains browsing, and gives the next explicit query the accepted
  policy. An unchanged policy preserves the active producer.
  Verify: `go test -race -tags no_emoji -count=1 ./internal/ui/visualsearch -run
  '^TestVisualSearchCacheLimitChangesRetireCapturedPolicy$'`.

Files: root `analysiscache.go` / `visualsearch_test.go`, visualsearch
`feature.go` / `feature_test.go`, this record and todos. The two code changes are
independent; both precede focused regression checks, `make verify-build`, GoLand
inspection, push, thread dispositions and another fresh review/CI round.
The accepted limit follows the existing explicit-restart behavior of persistence
changes; automatically restarting analysis or changing the worker protocol is
outside this fix. The prior broad-suite evidence is superseded by the next CI run.

Both guards failed for their intended behavior before the fixes: the real
Favorite naming/save path dropped its committed-save notification during a held
read-only inspection, and increasing the limit left `Preparing` true on the
old producer. The fixed guards pass under race (root 3.214 s; visualsearch
1.037 s). The save guard also checks that a suspended producer stays retired;
the policy matrix covers decreases, increases, unchanged limits, stale results,
preserved browsing and the policy captured by the next explicit query.

The root adapter now always forwards a committed save to the feature, whose own
producer admission ignores saves after retirement. Read-only inspection needs
no separate root admission rule. `SetCachePolicy` compares the complete captured
policy, retiring on any change instead of only a decreased byte limit.

Focused race regressions pass for all root FindMoreLikeThis cases (36.246 s),
visualsearch (1.076 s), and analysiscache (1.610 s). `make verify-build` passes.
GoLand's configured MCP endpoint was unavailable; its offline Project Default
inspection instead analyzed all four changed Go files in both baseline and
modified stages and reported no new findings. Reports and logs are retained
under `/tmp/picfetch-goland-review` and `/tmp/picfetch-goland-inspection` for this
session. No new root test runnable or test file requires a shard/exclusion update.
No package map, translation, dependency or production worker lifetime changed.
Lead review and implementation used zero subagents. The next pushed commit
requires fresh code/security review and complete native CI.

The canonical specification and FML-011 ticket also contained an earlier
no-retirement promise for limit increases. The immutable worker policy made that
promise incompatible with adopting the accepted higher limit. This follow-up
explicitly replaces it with the same local search retirement used for other
policy changes, retaining the displayed results and requiring an explicit query
to restart. Increasing a limit without eviction still preserves shared leases
and stored records. Live policy changes inside a running worker remain outside
this bounded fix. The canonical spec, ticket and V11 verification now state the
implemented behavior and reference the increase/decrease/unchanged regression.

### Review follow-up (25573b0): early visits, source failures and Grid work

The next review reported three further P2 findings; all 13 checks and security
review passed on this head. Route: Deep because the fixes cross the search,
Grid and root file-reconciliation lifetimes. Lead owns all implementation and
review; zero subagents. These fixes preserve the canonical spec's visit/source
contracts and bounded ranked display.

1. Preserve a defensively copied Grid anchor captured before the first search
   publication, attach it only when a successful visit is committed, and discard
   it when starting another reference/session. Files: visualsearch feature/session
   and existing feature/root tests. Verify `go test -race -tags no_emoji
   ./internal/ui/visualsearch ./internal/ui -run
   'TestVisualSearchCaptureGridBeforePublication|TestFindMoreLikeThisProgressiveForegroundIdentity'`.
2. Reconcile the captured collection off UI after terminal search failure, omitting
   confirmed missing sources and invalidating derived content before restoring
   the origin. Reuse the tracked file-work queue and stop context; session and
   collection identities reject stale delivery. Treat terminal failures
   conservatively rather than parsing worker error strings. Files: root
   visualsearch/filework, visualsearch State and existing root regressions.
   Verify `go test -race -tags no_emoji ./internal/ui -run
   '^TestFindMoreLikeThisSourceAndSortRetirement$'`.
3. Keep one generation-bound source-path index for ranked filtering, capture and
   restoration. Updates touch only the bounded result identities; collection
   replacement invalidates the mapping. Files: Grid ranked/search/state and
   existing ranked tests. Verify `go test -race -tags no_emoji
   ./internal/ui/grid -run '^TestRankedVisit'`.

Tasks 1 and 3 are independent; task 2 uses the search lifecycle observations.
All precede focused regressions, Make build verification, GoLand inspection,
commit/push and a fresh clean remote review. No new dependency, package, model,
worker lane, broad local race suite or delegated review is planned.

All three guards failed for their intended behavior before implementation:
the initial Grid anchor was empty, source failures retained stale entries/cache
writers, and 32 three-result revisions over 4,099 sources performed 525,440 path
lookups. The fixed Grid guard observes 768 lookups, preserves filtered selection,
and rebuilds its index after a generation-changing reorder. The feature guard
checks defensive copies, first-publication retention and new-session reset.

Source reconciliation waits for the retired producer and stats the captured
collection on tracked file workers. Only confirmed missing files are removed;
other stat errors retain their entries. Delivery verifies its request, search
session and collection generation, invalidates image/thumbnail writers, and uses
the existing source-change path to restore the origin. The displayed image is
also reloaded: an additional red guard demonstrated that cache invalidation
alone left replaced pixels behind the restored Grid. Coverage includes deleted,
replaced and all-deleted sources, plus stale delivery after a new collection,
query or explicit exit. Shutdown and new queries cancel the same tracked request.

Focused race checks passed for root FindMoreLikeThis and file-write regressions
(31.993 s), visualsearch (1.040 s), and Grid ranked visits (9.492 s).
The final initial-Grid/source-reconciliation rerun passed in 34.908 s.
`make verify-build` passed, including vet/build and repository generated checks.
No new root runnable or test file requires a shard/exclusion update.

GoLand's offline changed-files runner failed to restore part of its temporary
shelf, so that report was discarded. The working edits and evidence were
recovered and tested again; an isolated source copy was then inspected without
shelving. Its final ten Go files match the working files byte for byte. Project
Default reported no code errors, warnings or weak warnings in those files.
The remaining 35 advisory findings were reviewed: existing comment spelling and
grammar/style suggestions and optional syntax updates, outside the new code.
They do not establish defects in this change; proper package/API names account
for the spelling findings. No production suppression was added to hide them.
The complete report, changed-file findings and logs are retained under
`/tmp/picfetch-goland-review/round3-restored` and `/tmp/picfetch-pr25-*`.

Lead implementation/review used zero subagents. Architecture and todos reflect
the new ownership. Dependencies, translations and package boundaries are unchanged.
The next pushed head still requires fresh code/security review, post-suppression
Qodana inspection and complete native CI; final acceptance is recorded on PR #25.

### Review follow-up (41544f0): complete removals before search restoration

Fresh code review reported one P2 after all 13 checks passed, Qodana reported
zero post-suppression findings and security review completed. Route: Standard;
lead owns this bounded root fix with zero subagents. The batch-removal loop must
finish mutating the collection and cohort before one source-change notification
restores the search origin. A restoration must never load an image that another
index in the same batch will remove. Single-file removals retain immediate
notification. No new worker, dependency, field or package is needed.

Files: `internal/ui/viewer.go`, existing `visualsearch_test.go`, this evidence
record and todos. Add failing-first image-origin regressions for ordinary batch
removal, completed Trash reconciliation and terminal source reconciliation.
Assert the requested/restored image belongs to the final surviving collection
and that settling loads cannot remove a healthy survivor. Verify with
`go test -race -tags no_emoji ./internal/ui -run
'^TestFindMoreLikeThisSourceAndSortRetirement$/image-origin-batch'`, then focused
removal/search/deletion regressions, `make verify-build`, isolated GoLand
inspection, push, thread disposition and a fresh review/CI round.

All three new subtests failed before the fix: the display requested removed
`a.jpg` instead of surviving `b.jpg` after removing indexes 2 and 0. They pass
after separating the private collection/cohort mutation from source-change
notification (4.660 s under race). Public single removal still notifies once;
the batch path notifies once after its final valid index and then reconciles
Grid indexes. No extra mutable batch flag or worker was introduced.

The focused race suite for FindMoreLikeThis, RemoveFile(s), BatchDelete,
Deletion, PerformDelete, appState removal and VisualSimilarityExplorer passes
(224.668 s). `make verify-build` passes. New coverage uses subtests in the
existing file, so the shard manifest and Qodana test exclusions remain current.

GoLand inspection remains unverified for this final change. The isolated runner
first reported disabled project indexing and an unused root viewer; supplying
an explicit module root removed those symptoms but produced false build-constraint
errors for ordinary Fyne dependency files. Build/vet and the focused race suite
compile and exercise those same files successfully. No source suppression was
added for an incomplete project model. Both attempts and their complete findings
are retained in `/tmp/picfetch-goland-review/round4*`. This also limits confidence
in the preceding isolated inspection's project resolution; its lack of code
warnings should not be treated as a complete GoLand gate. The configured remote
Qodana report and fresh code/security reviews remain required after this push.

### Review follow-up (da1b014): restore after Grid and comparison transitions

The fresh code review reported two P2 findings after security, all 13 checks and
zero-result Qodana completed. Route: Standard; lead owns both root fixes, zero
subagents. Complete Grid index reconciliation before restoring an origin visit,
so its surviving selection and highlight are not cleared afterward. Terminal
source reconciliation must close an active comparison before origin restoration
admits an image reload. Use the existing comparison Close and source-change paths;
no new deferral state, worker or package is needed.

Files: `viewer.go`, `filework.go`, existing `visualsearch_test.go`, todos and this
record. Add failing-first Grid-origin removal guards and comparison/source-change
guards for deleted/replaced sources from both image and Grid origins. Verify with
`go test -race -tags no_emoji ./internal/ui -run
'^TestFindMoreLikeThisSourceAndSortRetirement$/(grid-origin-batch|comparison-source)'`,
then focused search/removal/comparison regressions and `make verify-build`.
Offline GoLand project resolution remains unavailable as documented above;
remote Qodana and fresh code/security review remain mandatory after push.

The Grid-origin guards failed with empty selection and reset highlight after
each ordinary/committed-Trash/source-failure batch. Comparison guards failed
because the comparison remained active for both deleted and replaced sources
from image and Grid origins. After moving Grid reconciliation before notification
and closing comparison inside current source-recovery delivery, all source
transition cases pass under race (21.824 s).

The Grid guards were strengthened to use 40 images and a real nonzero scroll
offset, checking the surviving selection, highlight, filename filter and scroll.
Focused FindMoreLikeThis, removal/deletion, comparison restoration/cancellation/
exit and command-entry regressions pass under race (67.521 s).
`make verify-build` passes. No new top-level runnable, test file, production
worker, dependency, translation or package was added. GoLand remains unverified
because of the documented offline project-resolution failure. The next pushed
head requires fresh remote review and analysis; final acceptance belongs on PR #25.

### Review follow-up (9d34a0c): retire captured limits before inspection

A later manual review found a P2 admission gap: an explicit limit increase did
not suspend local producers until asynchronous Retune completed. CacheManager
always inventories first, even when RetireWriters is true, so changing that flag
cannot close the gap. Route: Standard, lead-owned with zero subagents.

AC1: a valid changed limit suspends local Explorer/search producers on UI and
joins both existing completion barriers before calling the maintenance provider.
An unchanged limit and read-only inspection keep producers alive. Preserve the
existing shared-lease policy: only decreases force RetireWriters; an increase
that needs no eviction does not invalidate shared leases or delete records.
Verify: `go test -race -tags no_emoji ./internal/ui/analysiscache -run
'TestAnalysisCacheManagementLimit|TestAnalysisCacheManagementWriters'`.

AC2: while an increase scan is held, an active search has retired its captured
limit and retained its results, and standing policy remains unchanged until the
successful current operation applies. Cancellation keeps the prior policy and
joins retirement. Verify: `go test -race -tags no_emoji ./internal/ui -run
'^TestFindMoreLikeThisInitialAdmission$/limit-increase-before-inspection'`.

Files: analysiscache operation and existing feature tests, root visualsearch
tests, canonical spec/ticket/V11, todos and this evidence record. Add failing-first
barrier and root integration guards, then change admission using the existing
Host.Quiesce and operation predecessor joins. No new worker, interface, mutable
batch flag, dependency, package or root runnable is planned. Focused regressions,
Make build checks, available GoLand inspection and fresh remote code/security/
Qodana/CI follow. Offline GoLand's existing project-resolution limitation remains
explicitly unverified if its configured tooling is still unavailable.

Both new guards failed before implementation: changed limits reported zero
immediate retirements, and the root increase path still had a preparing search
when inspection began. Admission now captures Host.Quiesce barriers for every
valid changed limit; the existing operation worker joins them before provider
dispatch. RetireWriters remains reserved for decreases, preserving no-eviction
increase behavior for shared leases. A late provider quiescence alone would not
fix the gap because its first inventory has already run.

The feature matrix verifies increases, decreases, unchanged limits, both producer
barriers, unchanged shared-lease intent, success-only preference application,
and cancellation before/during inventory. Closing Settings before retirement
finishes keeps the operation tracked, joins both producers, and skips inventory
and preference application. The root guard holds an increase scan and verifies
retired preparation, preserved results, unchanged standing policy until success,
one maintenance operation and no automatic search restart.

Focused race checks pass: analysiscache 1.324 s, visualsearch 1.049 s, similarity
maintenance 1.155 s and root admission/Favorite-save tests 12.878 s. The final
cancellation/barrier matrix passes in 1.258 s. `make verify-build` passes. All
test additions use existing files; the root addition is an existing runnable's
subtest, so shard assignments and Qodana exclusions remain valid. The canonical
spec, FML-011 ticket/V11 and todos now place local retirement before inventory.
Available-tool discovery still exposes no GoLand inspection endpoint; the prior
offline project-resolution failures remain unverified. No source suppression
was added, and configured remote Qodana plus fresh code/security/CI are required
on the pushed fix. Lead review and implementation used zero subagents.

### Review follow-up (216364a): promotion warnings and temporary eviction

Fresh code review reported two confirmed P2 defects: general-hit promotion
discarded Favorite write errors, and retune sorted recent orphan temporary files
after reusable records. CI passed all 13 checks; the post-suppression Qodana SARIF
contained zero results and CodeQL had no open alerts. Security review completed.

Route: Standard, lead-owned with zero subagents. Two independent tasks within
similarity; the file count exceeds the usual eight only because an internal read
result change requires updating its existing consumers and test call sites.

- AC1: a general hit remains usable with no inference when Favorite promotion
  fails, and the search producer retains a bounded cache warning. Cover a blocked
  analysis directory and an independently retired Favorite lease; both producer
  write scopes keep the same read/promotion contract. Successful promotion stays
  warning-free. Verify: `go test -race -tags no_emoji ./internal/similarity -run
  '^TestAnalysisCachePolicyPromotionWarnings$' -count=1`.
- AC2: an over-budget retune removes managed general temporaries before older
  reusable records, continues with ordinary LRU if still over budget, and keeps
  exact remaining/removed bytes and records. Verify: `go test -race -tags no_emoji
  ./internal/similarity -run '^TestAnalysisCacheLimitRetune' -count=1`.

Files: store and its analyzer/search consumers, maintenance ordering, existing
cache policy/store/lifecycle tests, canonical spec, todos and this record. Tests
precede implementation. Keep cache misses nonfatal, reuse the existing warning
delivery, preserve root-scoped exclusion and Favorite protection, and introduce
no package, dependency, worker, test file or root UI runnable. Focused cache/search
race regressions and `make verify-build` precede the next push and fresh remote
reviews/CI. GoLand inspection remains unverified because no inspection endpoint
is available and the previous offline runner failed project resolution.

Both guards failed before implementation for the stated reasons: all four
promotion-failure cases retained an empty warning, and both temporary-eviction
cases removed reusable records while leaving the orphan. The internal store read
now returns promotion failure independently from its hit, and both analyzer and
search retain it in their existing bounded warning field. Explicit Favorite
refresh also preserves that error. Existing cache-read test helpers assert that
healthy reads and expected misses remain warning-free. Retune sorts temporary
records first under its existing exclusive lease, then applies the same LRU
ordering and byte target.

All `TestAnalysisCache` race tests pass (1.629 s). Focused `TestSearch`, cache
warning/policy UI, and maintenance UI race regressions pass in similarity
(11.016 s), visualsearch (1.041 s), and analysiscache (1.345 s).
`make verify-build` and `git diff --check` pass. No new test files or root UI
runnables were added, so Qodana exclusions and shard assignments remain current.
The red/green logs are `/tmp/picfetch-pr25-promotion-temp-{red,green}.log`;
regression and build logs use the same prefix. Lead review checked the acceptance
criteria and diff; zero subagents, one local review/fix round, and the full race
suite is delegated to the required GitHub CI workflow. GoLand remains unverified;
the pushed commit still requires fresh code/security reviews and Qodana/CodeQL/CI.

### Review follow-up (cc95ac7): image occurrences and completed cache producers

A later manual review found two confirmed P2 defects: image-origin restoration
uses the first matching path even when merge mode started on a later occurrence;
final cache pressure suspends the completed retained producer, and automatic
maintenance would suspend it again through the root Host. The current head has
13 passing CI checks and the previously inspected Qodana report has zero results.

Route: Deep for the existing cross-feature maintenance lifetime contract; extend
this accepted implementation record. Lead owns both tasks, zero subagents. Task
1 and task 2 are logically independent but share root search tests, so execute
them sequentially before one push and fresh review/CI round.

| Task | Files and contract | Acceptance command |
| --- | --- | --- |
| 1 | Root visualsearch and Visit capture, existing root round-trip tests. Save image path plus occurrence ordinal, preserve it through copied visits, and resolve that occurrence on Exit/Back/source reconciliation. If absent, use a surviving matching path, then the first available image. | `go test -race -tags no_emoji ./internal/ui -run '^TestFindMoreLikeThisInitialRoundTrip$/image-origin-occurrence' -count=1` |
| 2 | Visualsearch session, analysiscache Host/work/operation and root adapter, existing feature and root tests. Automatic eviction preserves fully prepared producers without pending Favorite writes; partial preparation and pending persistence are canceled/joined. Explicit cleanup, policy retirement and changed limits retain their existing retirement contract. Report pressure once per producer and show the paused toast only for unfinished preparation. | `go test -race -tags no_emoji ./internal/ui/visualsearch ./internal/ui/analysiscache ./internal/ui -run 'TestVisualSearchCachePressure|TestAnalysisCacheManagement|TestFindMoreLikeThisInitialAdmission/cache-pressure' -count=1` |

Task graph: task 1 -> task 2 -> focused race/build checks -> push -> fresh remote
reviews and CI. For task 2, shared lease invalidation still precedes quiescence;
the retained completed producer has no pending write and cannot reuse its old
lease after eviction. Explicit Favorite saves remain separate new admission.
No new process, goroutine, package, dependency, test file or root runnable is
planned. Canonical spec, applicable execution contract, todos and this evidence
record accompany the fix. Budget: zero spawns, one lead review per task plus
necessary fixes; one remote full suite per pushed head, no broad local race run.
GoLand remains unverified unless an inspection endpoint becomes available.

Task 1's initial three cases failed at the wrong restored index: Exit/Back
selected the first duplicate, and removing an earlier unrelated source did the
same after indexes shifted. Path plus zero-based occurrence now survives visit
capture/copy and restores the correct position; a fourth case verifies fallback
to the remaining matching path after an occurrence disappears. Initial green:
5.865 s. No absolute saved index or replacement collection snapshot was added.

Task 2's feature guard failed because the next reference started session 2, and
the real-root eviction guard observed the completed provider being canceled.
Automatic eviction now passes its retention intent through the existing Host;
the search feature retains only producers without preparation or pending Favorite
persistence. The same exclusive maintenance lease still invalidates old writes
before UI quiescence. Explicit cleanup and policy/limit changes pass full-retire
intent. Pressure notification is consumed once per producer, so a later warm
query does not repeat automatic eviction; completed preparation has no paused
toast. The root guard confirms actual removal, same-session query reuse, and
explicit cleanup cancellation. Initial feature/maintenance/root greens were
1.030 s / 1.299 s / 3.078 s.

An additional preparing/ready/pending-Favorite matrix checks writer quiescence.
Removing the pending-write protection in a temporary Go overlay made the
pending-Favorite case fail (retained=true), confirming the guard without changing
the worktree. Focused race regressions pass in visualsearch (1.029 s), analysiscache
(1.328 s), similarity cache (1.651 s), and root admission/round-trip/source
reconciliation (42.332 s). `make verify-build` and `git diff --check` pass. Logs
use `/tmp/picfetch-pr25-occurrence-*`, `/tmp/picfetch-pr25-final-pressure-*`, and
`/tmp/picfetch-pr25-pending-writer-red.log`.

Canonical spec/execution contracts, architecture map and todos are current. Test
additions use existing files and root subtests, preserving exclusions/shards.
Tool discovery exposes no GoLand inspection endpoint, so local IDE inspection
remains unverified; remote Qodana and fresh code/security/CI remain the pushed-head
gate. Both lead-owned tasks used zero subagents and one local review/fix round.
The separate user edit in AGENTS.md is preserved and excluded from the fix.

Qodana on 1c30a02 completed successfully but its post-suppression SARIF contained
one GoCommentStart weak warning: the new Host.Quiesce comment did not start with
the method name. Confirmed and corrected inline; no behavior changed or test was
added for comment wording. CodeQL has zero open PR-ref alerts. Code review
completed cleanly at 08:16:31 UTC and security review at 08:20:50 UTC on September
15; the connector posted its clean-result thumbs-up at 08:20:53 UTC. All 46
threads are resolved and all CI test jobs pass. This comment correction still
requires fresh reviews and Qodana/CI on its own pushed head. Formatting and
`git diff --check` pass; the behavior regression/build evidence above remains
applicable because only the comment and this evidence record changed.
