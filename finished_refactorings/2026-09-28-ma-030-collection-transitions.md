# MA-030: authoritative collection identity and committed transitions

Status: all nine tickets complete and CI-qualified on 2722b1d. PR 69 records the
continuing latest-head code/security reviews, dispositions and checks. No merge/release authorized.
Baseline: `1f92396367acc41c663710ee4b4b981c43d1c184`.
Branch: `feature/ma-030-collection-transitions`.
Authority: [accepted design](../docs/collection-transitions.md),
[ADR 0005](../docs/adr/0005-collection-transition-ownership.md), and local
`.scratch/ma-030/spec.md` with its nine approved implementation tickets.

## Frame and settled scope

Deep SDD: deepen `appState` into the sole collection authority and migrate root
effects into explicit committed-transition phases. Implement D1-D11 and AC1-AC18;
preserve MA-028 command admission and MA-029 browsing/failed-load policy. No new
dependencies, native changes, storage schema, UUIDs, feature-state store or event
bus. Existing dependency/license closure is unchanged.

The user authorized implementation, a commit per ticket, pushes, a draft PR,
GitHub CI, ready-for-review after all tickets, and the Codex review loop. No merge
or release. Existing tracked edits to the three MA-030 wayfinding documents are
the accepted publication record; retain them when updating implementation status.

Test seams were confirmed in the accepted specification: production collection
operations/observations without a viewer; production actions through the existing
viewer harness with offline/OS stubs for cross-feature effects. Use the TDD skill
one behavior at a time. Missing symbols are not behavioral red evidence.

Honest limits remain those in the specification: unavailable/unreadable/truncated
saved entries, explicit HEIC readmission, containment-checked Favorite association,
and already-retired surfaces after canceled open admission.

## Contracts and migration inventory

Keep the collection model in `internal/ui`; it contains values, not features or
callbacks. UI owns writes; atomically published immutable observations are safe
for workers. Navigation replaces only a small selection observation, sharing
immutable membership/order/index data and keeping the collection generation.

Ticket 01 contracts: `appState.Observe() collectionSnapshot`,
`appState.Select(int) bool`; snapshot `Generation`, `Count`, `FileAt`,
`SourceFiles`, `Retained`, `Current`, `Bookmark`, `Resolve` and `FileSet` methods.
`collectionBookmark` binds `fileidentity.Occurrence` to its collection generation.
`FileSet` is the explicit URI-key adapter for duplicate workers; bookmarks use
paths. Return cloned slices from exported observation methods. Replacements
expire previous bindings; retained immutable observations remain usable.

Ticket 02 extends the same model with `Replace(collectionInput) collectionChange`
and `Clear() collectionChange`. Input contains source/display orders, retained
unavailable order, requested index and candidate Favorite. Change contains before
and after observations plus the shared occurrence survivors. Later operations:
`Merge`, `Reorder`, `Remove`, `RemoveTargets`, and `MarkUnavailable`, each returning
the same change. Concrete arguments are finalized before their first red test.

Root capture/retirement precedes model commit. Publication/invalidation precedes
browsing and feature rebinding; display handoff precedes final presentation.
Extend `sourcechange.go`'s existing capture/finish helpers with committed results;
retain named replacement/merge/reorder/removal/unavailable/write/policy effects.

| Authority/caller today | Destination | Ticket |
| --- | --- | --- |
| `state.go` writable file/source/index fields and duplicate snapshot | Model operations and coherent immutable observation | 01-07, audit 09 |
| `state.go` retained order, root `heic.go` gap reconstruction | Model retained membership and capture | 01, 02, 04, 06, 07 |
| `explorerInput.favoriteDir` | Committed model association; feature gets a captured value | 02 |
| `drop.go` scan/open commit sequence | Shared explicit root commit phases | 02, 03 |
| Favorite/session discovery input | Occurrence-preserving replay through common workflow | 04 |
| `sort.go` start-time chosen occurrence | Latest model selection at reorder commit | 05 |
| `sourcechange.go` per-item removal and survivor calculation | One model batch result and shared survivors | 06 |
| `load.go` retained-order repair | Model unavailable/removal operation; display keeps retries | 07 |
| `filework.go` current-member capture/revalidation | Coherent snapshot, tracked alias reads, shared root effects | 08 |
| analysis/duplicate policy adapters | Named effects, no membership mutation or blanket purge | 08 |
| clear/reset/shutdown and remaining adapters | Authoritative clear and existing stop/join barriers | 09 |

Favorite association moved from `explorerInput` into atomic collection
observations in 02. Model-local mutable fields
may remain behind operations during migration; consumers must not acquire a new
writable mirror. Each ticket records the authority/ordering obligation retired.

## Task graph and routing

`01 -> 02 -> 03 -> 04`; `02 -> 05`; `02 -> 06 -> 07`; `02 -> 08`;
`04 + 05 + 07 + 08 -> 09`. Execute numbered tickets serially because root files
overlap. All tickets are T0-owned as approved. Use the active capable lead while
context is hot; the ticket model recommendations do not justify cold handoffs.
Read-only scouts and bounded single-package implementation tasks may assist;
never delegate design-bearing tests, architecture, review, or post-review fixes.

### 01 — coherent observations

Owner: T0 inline; read-only consumer scout in parallel.
Files: state/collection observations, browsing/load/viewer adapters, model and
compatibility tests, affected fixture setup, architecture, test inventories.
Depends: none. Contract: observation and selection above.
Test/verify: `TestCollectionModel/snapshots`,
`TestCollectionModel/operations/selection`,
`TestCollectionCompatibility/navigation_snapshots`, plus
`TestGenerationTracksFileSetIdentityNotNavigation` and
`TestFileSnapshot_KeysAndGenerationMoveTogether`.
Budget: <= 2 spawns, <= 2 planned review rounds, full suite no.

### 02 — atomic replacement

Owner: T0 inline. Files: model, drop, viewer, explorer, sourcechange, tests/docs.
Depends: 01. Contract: Replace/Clear and committed change above.
Test/verify: model snapshots and operations replacement/clear;
`TestCollectionAdmission/{replacement,empty_input}`;
`TestCollectionFavoriteAssociation/replacement`;
`TestCollectionUnavailable/favorite_open`;
`TestCollectionReconciliation/replacement` plus Explorer/command/browsing regressions.
Budget: <= 1 spawn, <= 2 planned review rounds, full suite no.

### 03 — retained merge

Owner: T0 inline. Files: model, drop, HEIC/root tests.
Depends: 02. Contract: Merge extends full retained membership atomically;
no-op admission preserves association and generation. `Merge(collectionInput)`
takes source/retained additions plus the prepared full display order and chosen
index. A retained-only addition preserves existing display order/selection;
the model, not the caller, appends committed source/retained membership.
Test/verify: `TestCollectionMerge`, model operations/merge,
FavoriteAssociation/merge, Admission/merge, Reconciliation/merge;
`TestHEICUnavailableFiles` and `TestBrowsingCollectionChanges`.
Budget: <= 1 spawn, <= 2 planned review rounds, full suite no.

### 04 — saved replay and capture

Owner: T0 inline; bounded filescan implementation eligible after exact contract.
Files: filescan, drop/session/favorites adapters, model capture, related tests.
Depends: 03. Contract: explicit replay input uses recorded occurrences, no sibling
expansion; ordinary discovery unchanged; current separate admission limits.
`filescan.ReplayWithAdmission` shares ImagesWithAdmission's signature/policy but
skips directories and keeps every admitted occurrence. `openCollection` receives
an explicit discovery/replay kind; Favorite and session choose replay, including
pending-capability re-admission. `collectionSnapshot.Capture(collectionOrder)`
owns source/display saved order plus retained gaps; ranked captures keep their
explicit index scope. Storage formats and their owners remain unchanged.
Test/verify: Replay (Favorite/session repeats, singleton, chosen sort, missing,
repeated limits, separate retention, capability available/unavailable/canceled,
truncation), Admission/replay, Capture (source/display/ranked/selected order),
FavoriteAssociation/saving, existing ranked capture and filescan/favstore/session.
Budget: <= 1 spawn, <= 2 planned review rounds, full suite no.

### 05 — sort handoff

Owner: T0 inline. Files: model, sort/sourcechange, held-reader tests.
Depends: 02. Contract: Reorder preserves latest requested occurrence at commit;
explicit origin restoration wins. Obsolete display work cannot publish.
Test/verify: model operations/reorder, SortHandoff (held sort/load, later navigation,
repetition, cancellation), Reconciliation/reorder, ChangeKinds/reorder;
BrowsingCollectionChanges, FindMoreLikeThisSourceAndSortRetirement, display's
TestPresentationContract.
Budget: <= 1 scout, <= 2 planned review rounds, full suite no.

### 06 — batch removals

Owner: T0 inline. Files: model, sourcechange/viewer, browsing tests.
Depends: 02. Contract: Remove indexes versus RemoveTargets exact URI keys;
one publication, one survivor map, all matching unavailable targets included.
Confirmed exhausted-Trash red requires final presentation at root as well:
deletion Host narrows to `ReconcileDeletedFiles(uris, msg)` plus CurrentFile,
ShowToast and ForceRepaint; the feature no longer makes a second load/empty-state
decision. Existing payload tests retain a collection-only fake root; real scope
and one-load behavior are verified in root integration cases.
Test/verify: Removal (partial success, stale completion, unavailable/repeated,
single publication, symlink), model occurrences (earlier/later repeats/gaps),
Capture/after_removal, Reconciliation/removal, ChangeKinds/removal;
BrowsingCollectionChanges and deletion's symlink target test.
Budget: <= 1 scout, <= 2 planned review rounds, full suite no.

### 07 — unavailable recovery

Owner: T0 inline. Files: model, load/heic/sourcechange, tests.
Depends: 06. Contract: MarkUnavailable retains source occurrence and association;
ordinary Remove uses one existing display retry chain within captured scope.
Test/verify: model operations/unavailability, Unavailable/runtime_loss,
Reconciliation/load_failure, Lifecycle/load_recovery; HEICBackendLossPreservesSession,
HEICUnavailableGuide/Files, BrowsingLoadRecovery/VisitLifecycle, PresentationContract.
Budget: <= 1 scout, <= 2 planned review rounds, full suite no.

### 08 — writes and policies

Owner: T0 inline. Files: filework/sourcechange and policy adapters/tests.
Depends: 02. Contract: authoritative WriteResult survives cancellation; current
snapshot revalidated on queued delivery, aliases resolved by tracked workers.
Test/verify: CommittedEffects, ChangeKinds/content_and_policy,
Lifecycle/committed_writes; all six existing root write/alias cases named in the
parent spec, EXIF/mosaic committed-cancellation tests and source retirement regressions.
Budget: <= 1 scout, <= 2 planned review rounds, full suite no.

### 09 — converge and qualify

Owner: T0 inline. Files: remaining adapters, lifecycle/harness, tests/docs.
Depends: 04,05,07,08. Contract: all operations and shared phases complete; no
obsolete writable authority or duplicated survivor algorithm.
Test/verify: all finalized Collection suites and parent AC1-AC18, existing
MA-028/029 regression matrix, lead ownership diff review, required inspections,
and `make verify` (GitHub native amd64 race suite may supply broad race evidence
under the explicitly invoked review workflow). Latest-commit CI, CodeQL, Qodana
post-suppression SARIF, fresh Codex code/security review required before handoff.
Budget: <= 1 scout, <= 2 planned local review rounds; full suite once via CI;
remote review rounds continue until clean as authorized.

## Verification and evidence

All test names above abbreviate the `TestCollection` prefix. Each test selection
uses `go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '<anchored selection>'`.
Enumerate `-list '^TestCollection'` and inspect actual subcase output; no missing
selection or skipped required case is a pass. Every new top-level root test gets
an exact shard row; every new test file gets an exact Qodana exclusion. Run both
Make inventory checks per changed inventory. Record red/green and deliberate
negative guards. Run focused regressions per ticket, `make fmt-check`, targeted
vet, and GoLand changed-file inspection including weak warnings. Use current IDE
inspection fallback when IDE-local Qodana cannot be automated; CI Qodana remains
a separate gate. No speculative dependency or native runtime changes.

Initial environment: focused existing state/generation tests pass using installed
Go outside the filesystem sandbox (Snap launcher requires confinement setup).
GitHub read succeeds with network escalation; no branch PR exists yet.

### Delegation gate and ledger

01 read-only scout: G1 bounded consumer/synchronization question; G2 source
locations checked by `rg`/targeted reads; G3 no writes; G4 sweep spans reader and
worker routes; G5 lead has not traced those routes. Rule S cannot replace tracing;
Rule W prompt has no implementation. This is reconnaissance, not delegated review.

| Ticket | Spawns budget/actual | Lead reviews | Full suite | Evidence/status |
| --- | --- | --- | --- | --- |
| 01 | 2/1 | 1 | no | Complete; evidence below |
| 02 | 1/0 | 1 | no | Complete; evidence below |
| 03 | 1/0 | 2 | no | Complete; merge and CI race evidence below |
| 04 | 1/1 | 1 | no | Complete; replay/capture evidence below |
| 05 | 1/1 | 1 | no | Complete; sort handoff evidence below |
| 06 | 1/1 | 2 | no | Complete; batch/removal handoff evidence below |
| 07 | 1/1 | 1 | no | Complete; unavailable recovery evidence below |
| 08 | 1/1 | 1 | no | Complete; committed-write/policy evidence below |
| 09 | 1/1 | 1 | CI passed | Complete; both migrations and final qualification below |

### Progress

- [x] Frame, settled specification and agreed test seams read.
- [x] Deep plan, task graph, file map, routing, contracts and budgets recorded.
- [x] 01 coherent snapshots and navigation.
- [x] 02 atomic replacement and Favorite association.
- [x] 03 retained merge.
- [x] 04 saved replay and capture.
- [x] 05 latest-choice sort handoff.
- [x] 06 batch removals and shared survivor result.
- [x] 07 unavailable retention and scoped recovery.
- [x] 08 committed writes and policy distinctions.
- [x] 09 convergence and complete qualification; draft removed after archival push.
- [x] Start fresh latest-head Codex/security review loop; PR 69 records its current outcome and required CI.

09 scout gate: G1 bounded Close Files/reset/scan/replay delivery test inventory;
G2 concrete locations and barriers checked by targeted reads; G3 read-only;
G4 native chooser, scanning, reset and shutdown tests span separate lifetimes;
G5 lead has not traced those concrete tests. Shell inventory cannot explain
barriers and assertions. This is reconnaissance, not delegated review or design.

### Ticket 01 evidence — 2026-09-28

All production selection writes now use `Select`. Host collection readers,
browsing scopes/occurrences, preloads and failed-search worker validation consume
the observation. Duplicate workers receive its URI-key adapter. Selection shares
immutable membership. Fixtures use model operations instead of bypassing publication.
Association remains in `explorerInput` until 02; retained-order observation refresh
keeps the file-set generation until 02/07 retire its two-step callers.

Behavioral red: snapshots failed on caller mutation of unavailable membership;
selection failed because replacement published index 2 instead of 0; navigation
failed because the observation still selected index 0 instead of 1. All went
green. Negative guard: removing Resolve's generation check made selection fail
with `old bookmark resolved into unrelated replacement`; restored, reran green.
Initial missing-interface compilation is not counted as red. Pending-display
coverage waits preloads and purges cache to avoid a synchronous cache-hit assumption.

Uncached verbose Collection cases pass (`internal/ui`, 0.147s). Enumeration found
both TestCollectionModel and TestCollectionCompatibility. Affected regressions:
`go test -tags no_emoji,nodynamic -count=1 ./internal/ui -run
'Test.*(Collection|AppState|Generation|FileSnapshot|Browsing|FindMoreLikeThis|Sort|Drop|Reveal|Clipboard|Cache|HEIC|Delete)'`
passed (`internal/ui`, 17.583s). This retains restricted scopes, duplicate browsing
and repeated-origin cases in the existing Browsing/FindMoreLikeThis suites.
Focused vet, `make fmt`, and
`make fmt-check check-test-shards check-qodana-test-exclusions` passed; the Docker
shard check counted 729 runnables across three shards. `git diff --check` passed.

GoLand fallback: current IDE profile, `get_file_problems`, `errorsOnly=false`,
baseline plus ticket 01 working diff. All 17 changed code files inspected without
timeouts/skips: collection.go, collection_test.go, state.go, browsing.go,
filework.go, load.go, sourcechange.go, viewer.go, clipboard_test.go, delete_test.go,
drop_test.go, filestate_test.go, heic_test.go, imgcache_test.go, reveal_test.go,
sort_test.go, visualsearch_test.go. No errors or ordinary warnings. Five weak
test-duplicate fragments in delete/filestate/imgcache are independent fixtures
covered by existing exact Qodana exclusions. Four unchanged viewer duplicate
fragments concern title presentation, reset/shutdown effects and Trash target
enumeration; their distinct callers are intentional here. Reset/removal migration
remains assigned to 06/09. No broad suppressions or cosmetic duplicate extraction.
collection.go reinspected clean after negative mutation. IDE-local Qodana has no
callable automation; CI post-suppression SARIF is still a separate required gate.

Lead review: no source I/O in observations, no new lifecycle, dependency, string,
platform behavior or writable feature mirror. Fixture breadth follows from live
reader migration; assertions preserved. Atomic association/batches remain open.

### Ticket 02 evidence

Retired `explorerInput.favoriteDir`: the model now publishes Favorite association,
orders, unavailable membership and requested index together through Replace/Clear.
Root open commit retires search/Grid, publishes, invalidates stale duplicates,
rebinds browsing/map, hands off display and presents within the browsing update.
Closed Grid rebuilds on next entry; calling FilesChanged here would unnecessarily
restart hashing behind the closed surface. Empty-state presentation no longer
clears an unavailable-only Favorite's committed facts. Legacy merge/unavailable,
sort and removal adapters remain explicitly assigned to 03/05/06/07/09.

AC mapping: proposed Admission/replacement is covered by the extended existing
`TestVisualSimilarityExplorer/favorite_identity_cancel/{scan,sort}` and
`TestStaleFileStateCompletionsDoNotOverwriteNewerState`: generation and association
now join membership/progress assertions. Run those with the new Admission/empty_input
case; there is deliberately no empty proposed replacement test. Existing Explorer
Favorite containment tests remain unchanged and run in the whole Explorer family.

Behavioral red: unavailable-only Favorite lost its candidate association; model
replacement with no retained members kept an association. Both pass after their
respective changes. Negative publication guard: deliberately publishing twice
failed model replacement/clear, both ordinary/Favorite replacement cases and root
reconciliation with their expected single-publication messages; restored and reran.
Mounted-surface assertions traverse widget render trees with the existing harness
walker. A mistaken container-only assertion was corrected, not counted as product red.

Focused uncached suite passed (`internal/ui`, 15.635s):
`go test -tags no_emoji,nodynamic -count=1 ./internal/ui -run
'^(TestCollection.*|TestVisualSimilarityExplorer|TestCommandAdmissionVisits|TestBrowsingCollectionChanges|TestStaleFileStateCompletionsDoNotOverwriteNewerState|TestHandleDrop_SupersededScanGoroutineExits)$'`.
An earlier affected Drop/Sort/HEIC/Generation/AppState regression run passed
(21.582s). `go vet -tags no_emoji,nodynamic ./internal/ui`, `make fmt`, and
`make fmt-check check-test-shards check-qodana-test-exclusions` passed; 733 runnables.
One redirected test invocation exited without output and is not used as evidence;
the direct focused invocation above succeeded.

GoLand fallback: current IDE profile, errorsOnly=false, revision d76e8da plus
ticket 02 working diff. All 11 changed code files inspected: collection.go,
collection_test.go, drop.go, explorer.go, explorer_test.go, filestate_test.go,
heic.go, menu_test.go, sourcechange.go, state.go and viewer.go. No timeouts,
errors or ordinary warnings. Two weak duplicate fragments each in filestate and
menu tests are independent fixtures under existing exact exclusions. Viewer's
same four intentional title/reset/Trash fragments remain; no blanket suppression.
collection.go and collection_test.go reinspected clean after their final changes.
CI's post-suppression Qodana result remains a separate final gate.

Lead review confirms no new workers, I/O in model, cache authority, dependency or
UI string. All collection effects remain on root composition; the model has no
feature callbacks added. Existing onRemove callback is still scheduled for 06.

04 scout gate: G1 bounded saved-replay/admission question; G2 returned source
locations checked against targeted reads; G3 read-only; G4 traversal, HEIC,
persistence and capture span multiple packages; G5 not yet traced by lead.
Rule S cannot replace the flow tracing and Rule W did not specify implementation.
The reused scout found both path and URI deduplication layers and the pending
capability replay pass; ordinary discovery semantics must remain unchanged.

05 scout gate: G1 bounded held-sort/load/origin test inventory; G2 source locations
checked by targeted reads; G3 no writes; G4 interaction tests span root browsing,
sorting and display packages; G5 the lead has not traced those test barriers.
S/W do not apply to flow tracing. Reuse the read-only scout while lead implements 03.

06 scout gate: G1 bounded removal/completion/retained-visit test inventory; G2
source locations verifiable by targeted reads; G3 no writes; G4 deletion and
root browsing, URI identity and cache effects span several test families; G5
lead has not traced those barriers. S/W do not replace flow tracing. Reuse the
scout while the lead owns ticket 05 design, implementation and review.

07 scout gate: G1 bounded runtime decoder-loss and retry-chain test inventory;
G2 concrete test locations and barriers verified by targeted reads; G3 read-only;
G4 HEIC capability, root recovery, retained scopes and display span separate test
families; G5 lead has not traced the concrete fixture lifetimes. S/W do not
replace flow tracing; reuse the scout during lead-owned ticket 06 work.

08 scout gate: G1 bounded committed-write, alias and policy test inventory; G2
source locations and deterministic barriers checked by targeted reads; G3 no
writes; G4 save/export/metadata/mosaic workers and root policy tests span several
families; G5 lead has not traced those concrete test lifetimes. S/W do not replace
flow tracing; reuse the scout while lead verifies ticket 06.

### Ticket 03 evidence

Merge now appends full retained membership in the model, including repeated
occurrences and unavailable gaps. Retained-only additions preserve the requested
browsable occurrence while committing incoming association and one generation.
No admitted additions is a true model/root no-op. The scan merge gate observes
retained membership, suppressing replacement-style siblings for unavailable-only
collections. Removed the obsolete retainUnavailableHEIC bridge. Preparation
continues using its existing token, per-input scan cap and separate retention cap.

New cases: Model/operations/merge; Merge/{unavailable_existing,per_input_limit};
Admission/merge/{browsable,unavailable}; FavoriteAssociation/merge's eight-way
existing/addition/association matrix and containment/{mixed,repeated,subset};
Reconciliation/merge shares the mounted-display/search-retirement guard.
Equivalent cancellation AC: existing
`TestVisualSimilarityExplorer/favorite_identity_cancel/{merge_scan,merge_sort}`
extended alongside the replacement cases; stale replacement/merge scan admission
and common sort callback rejection are covered by
`TestStaleFileStateCompletionsDoNotOverwriteNewerState`. Existing HEICUnavailableFiles
supplies repeated gap order and separate retention/capability regressions.

Behavioral reds: unavailable-existing merge expanded siblings; model Merge lost a
repeated source; retained-only Merge lost selection; four unavailable-addition
association cases failed atomic incoming identity. Each passed after its slice.
Negative guards: forced replacement admission failed per-input collection size,
unavailable membership, both no-op admission cases, all eight merge-order cases
and root reconciliation; disabling the model no-op guard failed its generation
assertion. Separately disabling Explorer's containment check failed mixed-source
cohort ownership while repeated/subset cases passed. Restored every mutation.
The containment fixture initially expanded siblings because it used ordinary
input; corrected to explicit Favorite input, not counted as product red.

All seven Collection top-level families ran uncached and verbose, 1.511s; no
required subcases skipped. Final affected suite passed (`internal/ui`, 20.026s):
`go test -tags no_emoji,nodynamic -count=1 ./internal/ui -run
'^(TestCollection.*|TestHEICUnavailableFiles|TestBrowsingCollectionChanges|TestStaleFileStateCompletionsDoNotOverwriteNewerState|TestVisualSimilarityExplorer|Test.*Merge.*)$'`.
Focused vet, make fmt/fmt-check, shard/exclusion checks and diff check passed.
The inventory now contains 734 runnables. No new files or dependencies.

GoLand fallback: revision 3582c7f plus ticket 03 diff, current IDE profile with
errorsOnly=false. All seven changed code files inspected: collection.go,
collection_test.go, drop.go, explorer_test.go, filestate_test.go, heic.go and
sourcechange.go. No errors, ordinary warnings, timeouts or skips. Only two old
17-line fixture-duplication weak warnings in filestate_test.go, covered by its
existing exact exclusion. Mutated files reinspected after restoration/fixes;
Explorer workflow also reinspected clean and has no final diff.

CI follow-up / diagnosing-bugs: ticket 02's run 36395246262 exposed races in all
three UI shards. The single existing session-restore test reproduced them under
`go test -race -tags no_emoji,nodynamic -count=2 ./internal/ui -run
'^TestRestoreSession_LoadsSavedFilesAndHidesLink$'` (race failure, 1.434s).
Ranked hypotheses were trailing repaint, deferred menu flush, then startup harness.
Moving only repaint removed that overlap but exposed the deferred menu race.
Finishing both synchronous effects before display admission restored the original
inline-driver contract; display's own request/presentation callbacks publish the
subsequent loading/pixel state. No test harness relaxation or broad queue change.
The unchanged repro passed three race runs (3.079s). All ten CI-failing tests and
the Collection families passed focused race checks, count=3, in two batches
(75.393s and 9.501s). Those existing tests provide the red-capable regression seam.
No instrumentation retained. This inline lead fix is included with 03 because it
corrects the shared replacement/merge boundary; second lead review recorded.

The macOS amd64 job failed during RealAssetInstall with a model-download
connection reset; it is unverified, not passed. The next pushed revision will run
that external gate again. Qodana/CodeQL job success on 3582c7f is not final SARIF or
latest-commit evidence. The PR remains draft until all nine tickets are complete.

### Ticket 04 evidence

Explicit discovery/replay input now selects one common bounded filescan admission
loop. Favorite menus and session restore keep recorded occurrences, skip directory
discovery, and use the same semantics after pending capability completion. Ordinary
drop sibling/recursive/dedup behavior remains. Model snapshot Capture owns saved
source/display order and unavailable gap attachment; shutdown and ordinary
Favorite capture use it. Ranked capture observes one snapshot and retains its
explicit scope. The old persistedFiles adapter is now test-only and forwards to
the model; retire it in 09. No serialization/schema, association-at-session-start,
native runtime, worker ownership, dependency or user-string changes.

Final inventory: Replay/{favorite,session}/{singleton,repeated}, unavailable_only,
discovery_deduplicates_aliases, missing_entry, trial_truncation, truncated_browsable,
unavailable_entries_and_limits/{gaps,separate_caps,floor}, pending_capability/
{available,available_at_cap,unavailable,unavailable_at_cap,canceled,superseded};
Capture/{orders,saving,ranked_selected/{all,selected}}. The Favorite cases reopen
stored numbered path maps and verify fresh bindings. Session cases save before
newTestViewer's actual startup load, then invoke production restore and verify
the consumed offer. Saving exercises actual Favorite serialization and shutdown
session persistence with repeated entries/unavailable gaps.

Equivalent AC mapping: proposed Admission/replay is Replay/pending_capability/
{canceled,superseded} plus existing Explorer/favorite_identity_cancel/{sort,merge_sort}
and CaptureSort superseded-completion tests; all remain on production admission.
The earlier held-directory scan cases now use ordinary OpenFiles: Favorite replay
does not traverse directories. New pending-check cases cover Favorite scan
retirement, rather than retaining an invalid folder-as-Favorite fixture.
FavoriteAssociation/saving is Capture/saving plus ranked_selected, asserting
unchanged association/generation after actual saves. Existing
FindMoreLikeThisActionsCaptureRankedSources, BrowsingActionTargets and the
favorites package's capture-before-naming tests retain broader scope coverage.

Reds: repeated Favorite/session entries collapsed; singleton session expanded
neighbors; model capture omitted unavailable gaps. Ordinary alias discovery also
revealed that pre-dedup admission bookkeeping retained a rejected alias. Filtering
that bookkeeping by actual admitted occurrence counts fixes model consistency
without changing scanner discovery policy. Each now passes.
Negative guards: forcing replay through discovery failed directory avoidance,
occurrence budgets, both saved-repeat entry points and capability re-admission;
reinstating URI dedup in root recording failed unavailable-only/gap/cap cases;
disabling selected-save filtering failed exactly the selected ranked capture.
All mutations restored and affected inspections rerun.

Two fixture errors are not counted as product red: a prior persisted HEIC
observation prevented the controlled pending check, fixed by clearing that
observation before installing its backend; HEIC guidance replaces the truncation
toast in mixed replay, so feedback is pinned with browsable-only replay instead.
Expected preview failures on deliberately unavailable/missing sources are logged
through the existing error boundary, not ignored test failures.

Uncached verbose Replay/Capture passed (internal/ui, 1.366s), all required cases
executed. Broad affected root selection
`Test.*(Collection|Drop|Session|HEIC|FindMoreLikeThisActionsCaptureRankedSources|BrowsingActionTargets|CaptureSort)`
passed (9.528s). Full Explorer plus ranked/action-target regression families passed
(15.345s). Focused race command `GORACE=halt_on_error=1 go test -race
-tags no_emoji,nodynamic -count=1 ./internal/ui -run
'^(TestCollection.*|TestHEICUnavailableFiles|TestFindMoreLikeThisActionsCaptureRankedSources|TestBrowsingActionTargets)$'`
passed (56.783s). All filescan/favstore/session/favorites tests passed, as did focused
vet, make fmt/fmt-check, shard/exclusion checks (736 runnables) and diff check.

GoLand fallback: current IDE profile, errorsOnly=false, revision 7580808 plus 04
diff; all 12 changed code files inspected with no skipped/timed-out files:
filescan.go, filescan_test.go and UI collection.go, collection_replay_test.go,
drop.go, explorer_test.go, heic.go, heic_test.go, run.go, session.go, viewer.go,
visualsearch.go. No errors or ordinary warnings. Only the same four intentional
viewer title/reset/Trash weak-duplicate fragments recorded in 01/02; no new
suppressions. The new test file has its exact Qodana exclusion and both new
families have sorted shard assignments.

Lead review confirms source I/O stays in preparation; collection observations
only manipulate captured values. Both pending-HEIC admission passes share the
same kind/budget. Retained capture clones its outputs. CI on ticket 03 revision
7580808 is now entirely green, including all race shards and both macOS native
guards; final SARIF and fresh review gates still await the final implementation.

### Ticket 05 evidence

`Reorder` captures the latest model selection at commit and publishes its mapped
path/ordinal with the prepared display order in one generation. Source order,
retained gaps and Favorite association survive unchanged. `SetSortMode` captures
source preparation only; its former parallel commit sequence now lives at the
root collection boundary. The old lowercase reorder mutator is removed from
production and fixtures. Root detaches search, publishes, rebinds visits and Grid,
rebuilds map facts, honors explicit origin restoration, then admits one display
request after synchronous menu/repaint work. Empty restricted scopes cancel an
obsolete request without loading outside their scope. No new workers or caches.

Acceptance mapping:

- Model/operations/reorder pins later repeated occurrence, one generation,
  association/source/retained preservation, anchored capture, input copying and
  immutable old snapshots/bookmarks.
- SortHandoff/{loaded_latest,pending_latest} holds sort metadata and image open
  independently. A later navigation selects the second occurrence while sorting
  waits; sort commits before the old held decode returns. The old LoadDone ends
  without releasing the reader, exactly one fresh request is admitted, and late
  old pixels cannot replace either display or cache. Navigation continues from
  the remapped occurrence. canceled/superseded reuse the established deterministic
  held-reader regressions, strengthened with no-publication assertions.
- Reconciliation/reorder/{empty_scope,explicit_origin,notifications,cohort,cluster}
  pins restricted emptiness, an explicit repeated image origin over a newer
  ranked image, withheld menu publication, retained cohort selection/return, and
  a frozen repeated map cluster's exact selection/return.
- ChangeKinds/reorder pins ordinary Grid selection clearing, unchanged image
  cache and writer admission, retained duplicate facts, and rejection of old
  generation-bound duplicate producers. Display's whole PresentationContract
  verifies existing retry/cache/preload/animation/capture ownership.

TDD and negative evidence: model reorder initially selected index 1 instead of
the remapped second occurrence at 2. Both held root cases initially restored the
start-time/outgoing b.jpg at 0. Deliberately restoring start-of-sort selection
reproduced both failures. Discarding captured origin/Grid visits failed all four
corresponding reconciliation cases. A blanket image-cache purge failed the
change-kind guard. Skipping a pending load handoff and bypassing empty scope
eligibility failed their respective guards. Every mutation was restored before
final acceptance, race, vet and reinspection. An initial held-reader fixture
needed a cache entry under its test URI scheme to avoid blocking its own preload;
that harness mistake was corrected before the stated root red evidence.

Verification on 96d8ce1 + ticket 05 working tree:

- Final uncached verbose Model, SortHandoff, Reconciliation and ChangeKinds:
  PASS, 0.998s; all named subcases ran, no skips.
- Collection/sort/BrowsingCollectionChanges/FindMoreLikeThisSourceAndSortRetirement/
  deletion regressions: PASS, 7.747s. GridBrowseDuringAnalysis, BrowsingScope and
  BrowsingRoundTrips: PASS, 1.888s; exact BrowsingEmptyScope: PASS, 0.429s.
- Display PresentationContract, all subcases: PASS, 0.199s.
- Focused race across all Collection families, CaptureSort, BrowsingCollectionChanges
  and FindMoreLikeThisSourceAndSortRetirement: PASS, 90.722s.
- Focused UI vet, make fmt/fmt-check, shard inventory (738 runnables), exact Qodana
  test exclusions and git diff --check: PASS.

GoLand Inspect Code fallback covered all nine changed code files: collection.go,
state.go, sort.go, sourcechange.go, collection_test.go, collection_sort_test.go,
sort_test.go, grid_test.go and delete_test.go; weak warnings included, no skipped
files or timeouts. Seven files are clean. Grid's existing independent variants
title/hover fixtures (20 lines each) and shutdown setup (9 lines), plus deletion's
shutdown setup (13 lines), remain intentional test duplication covered by their
existing exact qodana.yaml exclusions. No new suppression or production finding.
Restored sort/sourcechange and final test refinements were reinspected clean.

Lead review confirms no source I/O in the model, no blanket purge, no bypass of
origin priority or duplicate/visit policies, and no dependency/UI-string changes.
Ticket 04's latest GitHub jobs are green on 96d8ce1; this is not a substitute for
the final revision's post-suppression SARIF and fresh Codex code/security reviews.

### Ticket 06 evidence

`Remove` applies browsable indexes against one observation; `RemoveTargets`
matches captured URI keys against both available and unavailable entries.
One publication returns before/after snapshots, one path-occurrence survivor map,
and unique removed URI effects. Source/retained filtering uses URI ordinals,
while browsing uses path ordinals; neither resolves filesystem aliases. A
surviving requested occurrence is remapped, removed selection clamps, and only
full membership exhaustion clears association. Unmatched/invalid removal is a
no-op. Root remaps captured visits from the model result, then evicts affected
cache keys, updates cohort/Grid/map facts and restores a valid scope. Removed the
per-item state mutator, onRemove callback and independently computed root mapping.

An additional behavioral red showed the deletion feature starting an unrelated
image after root returned an exhausted search/cohort to its map. Its Host now
delivers successful URIs plus the outcome message in one call; root owns the
sole scoped display/empty-state handoff. `sourcesTrashed` keeps that final load
separate from ordinary removal and display's failed-load retry. Model publication
is separate from committed-empty presentation, so last-browsable deletion no
longer issues another Clear or discards unavailable survivors. Explicit Close
still clears full membership. No new workers, dependency or translated string.

Acceptance mapping:

- Removal/{partial_batch,empty,unavailable_survivor,stale_delivery,symlink_target,
  unmatched_keeps_search}: actual controlled OS moves, partial failure, duplicate
  target deduplication, matching unavailable deletion, one generation, callback
  observation of complete facts, old snapshot immutability, queued completion
  after replacement, destination survival, and no unrelated search retirement.
- Model/occurrences/{earlier,later,targets,uri_keys_and_path_bookmarks}: exact
  source/gap removal, concrete shared map, repeated occurrence selection,
  distinction between full URI keys and same-path bookmarks, association clearing
  only after full exhaustion, and invalid/unmatched no-op.
- Capture/after_removal saves a real Favorite and shutdown session; neither can
  reinsert removed entries or scramble gaps after mixed ordinary/target removal.
- Reconciliation/removal/{repeated_image_origin,image_origin_single_load,
  partial_collection,partial_cohort,partial_cluster,exhausted_trash/false|true}:
  exact middle-of-three repeated origin, one display request, partial OS failure
  and unavailable survivors under retained search/cohort/map origins, preserved
  selection/return, and no widening of exhausted restricted scopes.
- ChangeKinds/removal verifies affected cache eviction without purging unrelated
  content/writers, preserved duplicate facts and rejection of old producers.
  The migrated model/cache test explicitly proves the model only reports effects.

TDD: initial root reds showed 2->5 generation for a three-occurrence batch,
retained deleted targets, 2->4 on last-browsable presentation, lost association,
and ignored queued unavailable-target completion. Model reds behind the old
per-item adapter showed absent survivor maps and incomplete target removal.
Exhausted-Trash integration then failed for both Grid and image origins, driving
the narrowed Host. The cluster fixture initially attempted a deliberately
forbidden search from Location Map; it now keeps the map origin directly and
waits its tracked return validation, preserving existing command admission.

Negative guards: disabling unavailable-target filtering failed model targets and
root retained/partial batches; dropping the shared map failed the middle repeated
origin; omitting cache eviction failed ChangeKinds/removal. Retaining removed
source-order entries failed persisted capture. Re-enabling a second Trash load
failed the exact-one-load guard, and bypassing scope eligibility failed both
exhausted origin cases. All mutations restored before final verification.

Verification on 2efaa44 + ticket 06 working tree:

- Final verbose Removal/Model/Capture/Reconciliation/ChangeKinds: PASS, 2.413s;
  every named case ran, symlink case passed without skipping.
- Collection/Browsing/RemoveFile/Deletion/AppState/HEIC/BatchDelete/search-source
  retirement regressions: PASS, 11.469s.
- Explorer source_changes (including held remove_after_reorder), Grid
  source_identity and Location Map automatic_rebuild: PASS, 1.123s, all subcases.
- Full deletion package: PASS, 0.047s. Explicit valid/broken symlink contract:
  PASS, 0.036s, no skips.
- Focused race across all Collection suites, browsing reconciliation/recovery/
  emptiness, search retirement, Deletion and BatchDelete: PASS, 115.807s.
- UI/deletion vet, make fmt/fmt-check, shard inventory (739 runnables), exact
  Qodana exclusions and git diff --check: PASS.

GoLand Inspect Code fallback covered all 15 changed code files, weak warnings
included, with no skips/timeouts: build, collection, collection_removal_test,
collection_replay_test, collection_sort_test, collection_test, deletion/deletion,
deletion/deletion_test, filestate_test, imgcache_test, sourcechange, state,
state_test, viewer and visualsearch_test (all .go under internal/ui). No errors
or ordinary warnings. Intentional weak duplicate fragments: paired file-state
transition sequences (17 lines each), root/model cache fixtures (6 each), and
Grid/image search retirement fixtures (14 each), all under existing exact test
exclusions. Viewer's existing title (20/16) and dropzone/HEIC presentation (26)
fragments retain their prior justified disposition; the old Trash duplicate
fragment disappeared. Restored collection/sourcechange/viewer reinspected with
the same findings; no new suppression.

Lead review confirms the concrete survivor map is constructed only in the model,
root effects run after complete publication, stale committed URI effects remain
authoritative and deletion Close/Stop/Settle semantics are unchanged. Ticket 05
GitHub jobs are green on 2efaa44; final SARIF and fresh reviews remain open.

### Ticket 07 evidence — 2026-09-28

MarkUnavailable now commits browsable removal, exact retained occurrence status,
source placement, association and the shared survivor map together. It reuses
the model removal walk without source I/O or callbacks. Root names decoder loss
separately, applies the same remapping/derived effects, then stops at the existing
guide. Ordinary failures continue in display's original retry chain. Removed
fileOccurrence, retainOrder and retainedOrder; no post-publication repair remains
in load.go. No package moves, workers, dependencies or user-visible strings.

Acceptance evidence:

- Model/operations/unavailability: middle-of-three repeated occurrence, source
  and retained placement, survivor ordinals, unique cache effect, immutable old
  snapshot, final browsable exhaustion with retained association and no Current,
  and invalid-index no-ops.
- Unavailable/runtime_loss/{false,true}: actual HEIC worker failure after cached
  capability admission, one publication, complete facts at sibling Grid callback,
  last-member and sibling guide stops, retained Favorite capture, real mixed
  Favorite save, support recheck without reinsertion, and explicit reopen.
  The empty Grid does not emit a result-change callback; final observation and
  mounted guidance cover that case. Existing MA-028 disables Add Current List
  with no browsable image; capture remains complete, and strengthened
  HEICBackendLossPreservesSession proves actual shutdown persistence plus binding.
- Unavailable/runtime_loss/retained_search_origin restores a nested Explorer
  origin while stopping at guidance rather than admitting another image.
- Reconciliation/load_failure/scoped_retry runs all four existing browsing
  recovery cases; repeated_origin preserves the exact last repeated origin over
  an ordinary successor, with one generation and one display request chain.
- Lifecycle/load_recovery: all 12 combinations of ordinary/unavailable failure,
  held/queued delivery and replacement/close-reopen/production Stop. Old LoadDone
  completes before releasing held work, current retained facts stay intact, no
  stale guide/retry appears, and workers settle through existing barriers.

TDD: real last-member/sibling decoder failures initially exposed association
loss and incomplete callback observations. The model adapter initially failed
the exact retained-occurrence assertion. Negative guards independently failed
last-member retention when Replace cleared association on browsable exhaustion,
and explicit-origin tests when recovery ignored the restored origin. The latter
also failed the strengthened repeated-origin test. All mutations restored.
Reader-open count is not retry admission (other image consumers may read); the
test pins display RequestRevision and the original LoadDone handle instead.

Verification on 2b08c3e + ticket 07 working tree:

- Final verbose Model/Unavailable/Lifecycle/Reconciliation: PASS, 3.801s, every
  required subcase ran. Broader HEIC/browsing acceptance run: PASS, 4.844s.
- Complete display PresentationContract: PASS, 0.250s.
- Focused race across all Collection and HEIC families, BrowsingLoadRecovery,
  BrowsingVisitLifecycle and search/source retirement: PASS, 152.582s.
- UI/display vet, make fmt/fmt-check, exact Qodana exclusions, Docker shard check
  (740 runnables) and git diff --check: PASS.
- GoLand Inspect Code fallback, current IDE profile, errorsOnly=false: all eight
  changed code files (collection, collection_test, collection_recovery_test,
  heic, heic_test, load, sourcechange, state under internal/ui) completed with
  no findings, including weak warnings; no skips/timeouts or suppressions.

Lead review confirms one authoritative publication, unchanged command admission,
separate decoder-guide versus ordinary-retry policy, and existing cancellation/
queue ownership. Ticket 06's GitHub jobs are all green on 2b08c3e; post-suppression
SARIF and final fresh reviews remain required at the final gate.

### Ticket 08 evidence — 2026-09-28

Committed-source resolution now consumes the coherent collection observation,
including its actual URI values, rather than reparsing the duplicate-key adapter.
Both affected-source and current-image decisions are bound to a captured
collection generation. A newly exposed race allowed a queued current-image
decision to load an obsolete index after an earlier member was removed: removal
moves selection without necessarily issuing a display request. Refresh now
re-reads the current binding when collection generation or cache generation
changes; changed display requests still suppress obsolete presentation. Alias
resolution, metadata reads and completion stay in the tracked file-work lane.

Caller inventory: save.go/saveRotation and export.go/exportAs preserve committed
results despite stale action tokens and deliver afterFileWrite; viewer.go's
AfterMetadataRemoved and mosaic.go's AfterFileExported deliver their existing
Host notifications there too. Their existing commit-time image-cache purges are
unchanged. Root sourceWritten invalidates derived content/map/analysis state,
preserves Grid indexes and refreshes comparison. sourcesRevalidated additionally
purges image content and rebuilds indexes, using the model's Remove result when
missing sources exist. SetMaxFileSizeMB and pushDuplicateDistance retain named
policy effects and no-op checks; analysis-cache maintenance keeps its separate
quiescence/lifecycle owner. No new dispatcher, workers, packages or dependencies.

Acceptance mapping:

- CommittedEffects/callers/{save,export,metadata,mosaic}: actual disk commits,
  held stale save/export results and delivered metadata/mosaic Host results;
  replacement Favorite has a noncurrent symlink alias plus an unavailable entry.
  All four invalidate affected derived producers without changing binding,
  membership, chosen occurrence, unrelated pixels/rotation or stale toast text.
  Required alias cases ran without skipping.
- CommittedEffects/chosen_index_revalidation holds the second UI delivery,
  removes an earlier member through root, and proves refreshed pixels still
  belong to the chosen source at its new index, with no membership publication.
- Lifecycle/committed_writes covers both queued decisions across same-membership
  rebinding, unrelated replacement, close/reopen and terminal file-work admission
  closure, for committed and uncommitted results (16 cases). Completion occurs
  exactly once and includes required new work; disk truth survives cancellation.
- ChangeKinds/content_and_policy covers content, validation, analysis, duplicate
  threshold, and both unchanged-policy cases. It pins separate image/thumbnail/
  Favorite/duplicate producer invalidation, generation, browsing rebinding and
  Grid selection behavior. The encoded-limit fixture restores the global limit.

TDD: chosen_index_revalidation failed against the old implementation because a
queued export refresh selected the following image. The generation-bound refresh
made it pass. Negative guards: skipping affected-source generation revalidation
failed queued replacement; disabling alias matching failed all four caller cases;
adding an analysis-policy blanket image-cache purge failed its distinct-effects
case. All mutations restored before final verification.

Verification on 3887956 + ticket 08 working tree:

- Final verbose CommittedEffects/Lifecycle/ChangeKinds: PASS, 2.605s; all required
  subcases ran. Location Map competing writes/replacement: all four cases PASS,
  0.434s, preserving source versions and current-map facts.
- Existing Save cancellation, Export current/noncurrent alias, unrelated export,
  loaded-set revalidation, competing commit, metadata rotation, complete Explorer,
  browsing reconciliation and search/source-retirement regressions: PASS, 17.822s.
- Metadata-removal committed/uncommitted cancellation matrix: PASS, 0.184s;
  mosaic committed/uncommitted cancellation matrix: PASS, 0.109s.
- Focused race across new families plus existing save/export/metadata/alias,
  browsing and search-retirement contracts: PASS, 75.545s.
- UI vet, make fmt/fmt-check, exact Qodana exclusions, Docker shard inventory
  (741 runnables) and git diff --check: PASS.
- GoLand Inspect Code fallback, current IDE profile, errorsOnly=false: filework,
  collection_effects_test, collection_recovery_test and collection_sort_test
  completed with no findings (including weak warnings), no skips/timeouts.
  Restored sourcechange also reinspected clean. No suppressions added.

Lead review confirms all four write routes share the ordered root boundary,
source I/O remains off UI, stale-request suppression stays separate from disk
truth, and policy differences remain explicit. Ticket 07's GitHub jobs are all
green on 3887956; final SARIF/review gates remain pending ticket 09.

### Ticket 09 qualification candidate — 2026-09-28

Both migrations are now implemented. The raw files/unsortedFiles/index/
unavailableOrder/favoriteDir fields and setFiles/replaceFiles/clearFiles/
publish/publishGeneration/snapshot bridges are removed. Immutable collectionData
is the only membership/order/association storage. Replace, Reorder and Select
are the only atomic stores; Remove, RemoveTargets, MarkUnavailable, Merge and
Clear compose those operations. Selection shares immutable data; stable reorder
shares source/retained projections and builds the occurrence index only once.
Root readers and fixtures use observations/operations, never writable mirrors.
Map and browsing restoration reuse the model's occurrence index. No new worker,
package, dependency, native behavior, schema or user-visible string was added.

Lead ownership/ordering assessment against baseline 1f923963:

| Previous obligation | Final authority and caller disposition |
| --- | --- |
| Mutable appState fields plus separately published duplicate keys | collection.go owns all facts and one coherent publication; state.go retains only standing sort/merge preferences and the atomic observation |
| Root HEIC retained-order repair and persistedFiles adapter | Model MarkUnavailable/Capture own exact occurrences and gaps; heic.go only prepares admitted retained inputs and presents support guidance |
| explorerInput Favorite association | Removed; model observation supplies the candidate to Explorer. Explorer's own captured favoriteDir belongs to its analysis session and containment-checked cohort persistence, not a writable loaded-collection mirror |
| Scan/open/merge partial fields and duplicated completion ordering | drop.go prepares complete input; commitOpenedCollection publishes, invalidates/rebinds features and flushes notifications before final display admission |
| Favorite/session discovery semantics | OpenFavorite and startup session explicitly choose replay through the common scanner/admission/sort/commit path; storage owners/formats stay unchanged |
| Start-time chosen image during sort | commitCollectionReorder reads the latest model selection, preserves explicit browsing-origin priority, rebinds before one load and invalidates an ineligible obsolete load |
| Per-item removal and consumer-computed survivors | One model walk returns exact-URI effects and path/ordinal survivors. Root passes that same map to Grid bookmarks, retained visits, image origins and frozen scopes; no consumer reconstructs it |
| Deletion feature's second load/empty decision | ReconcileDeletedFiles owns outcome presentation and one final load; deletion keeps OS moves and its cancellation/completion lifetime |
| Load-failure retained repair/retry | Model commits once; ordinary failure returns a scoped successor to the original display retry chain; unavailable failure retains membership and stops at guidance |
| Committed write/source validation | fileWork captures the observation, performs alias/source I/O on tracked workers, revalidates both queued decisions and calls named sourceWritten/sourcesRevalidated effects; removals use the shared batch result |
| Policy updates | Existing analysis/duplicate adapters retain separate effects and no-op checks; no membership publication, universal purge or automatic analysis restart |
| Close Files/reset/shutdown | Clear after surface retirement, then empty presentation. Committed empty presentation does not clear unavailable membership. Shutdown preserves session capture and cancels existing admissions; harness joins/drains chooser, fileWork, HEIC, scan/sort and feature lanes in their established order |
| Derived compatibility adapters | dupeFileSet forwards immutable URI keys/generation; Host Count/FileAt/Current project an observation. Feature-local Grid indexes, display request generations, source/cache versions and browsing visit bindings retain their separate domains |

The inventory is an ownership assessment, not only an old-symbol search. No
source I/O moved onto UI, model methods call no features, and feature workers,
caches, query history, comparison, map cameras and Picture-frame timing remain
with their existing owners. The initial fixture migration was mechanical, then
hot reader loops were changed to capture one observation without slice cloning.
All previous test assertions remain, using production observations instead of
private mirrored fields. Old method names in test titles remain descriptive
compatibility names, not retained setters.

New lifecycle evidence: Close Files' registered menu action and reset clear
every fact in one publication while mounting the welcome surface. Twelve held
scan/replay/sort cases cover cancel, replacement, close/reopen and production
shutdown, including repeated members, retained HEIC and fresh Explorer cohorts.
Queued native chooser results cannot reopen a closed/replaced collection or
cross terminal chooser admission. Validation-removal deletes multiple repeated
sources in one commit, preserves the unavailable gap/Favorite, retires search and
restores the surviving image origin before exactly one load.

TDD/convergence: existing behavior contracts were green before bridge removal;
they were rerun after the structural change. The new close/reset guard failed
both cases when authoritative Clear was deliberately omitted, then passed after
restoration. Earlier tickets record independent intended-violation failures for
immutable input, association publication, replay deduplication, latest choice,
shared survivor remapping, unavailable retention and committed-write revalidation.
The new production-shutdown fixture initially leaked capture-date preferences
into the shared test app; its cleanup now restores preferences and session data.
A validation test initially supplied an out-of-range rank; correcting that
fixture is not claimed as behavioral red evidence.

#### Final AC mapping

All names below have the TestCollection prefix unless another exact family is
shown. Every named Collection case ran uncached without skipping. Full outputs
and exact case inventory are retained locally in `.scratch/ma-030/evidence/09-*.log`.

| AC | Concrete final evidence |
| --- | --- |
| 1 | Model/snapshots, operations/selection; Compatibility/navigation_snapshots; TestGenerationTracksFileSetIdentityNotNavigation and TestFileSnapshot_KeysAndGenerationMoveTogether |
| 2 | Model/operations/{replacement,merge,reorder,unavailability,clear,selection} |
| 3 | Model/occurrences/{earlier,later,targets,uri_keys_and_path_bookmarks}; Capture/after_removal; repeated image/load origins |
| 4 | Admission/{empty_input,merge}; Lifecycle/preparation 12 cases; Replay/pending_capability/{canceled,superseded}; TestVisualSimilarityExplorer/favorite_identity_cancel; TestStaleFileStateCompletions and existing TestHandleDrop cases |
| 5 | Removal/{partial_batch,empty,unavailable_survivor,stale_delivery,symlink_target,unmatched_keeps_search}; Reconciliation/removal; deletion/TestPerformDelete_MovesSymlinkWithoutFollowingIt (ran, no skip) |
| 6 | Unavailable/favorite_open and runtime_loss/{false,true,retained_search_origin}; TestHEICBackendLossPreservesSession, TestHEICUnavailableFiles and guidance cases |
| 7 | FavoriteAssociation/replacement and all merge/containment cases; Capture/{saving,ranked_selected}; existing Explorer Favorite identity/cohort cases |
| 8 | Merge/{unavailable_existing,per_input_limit}; FavoriteAssociation/merge unavailable matrix; Admission/merge no-ops |
| 9 | Replay: real Favorite/session singleton/repeated, missing, limits/floor/truncation, pending capability and discovery aliases; complete filescan/favstore/session suites |
| 10 | Capture/{orders,saving,ranked_selected,after_removal}; TestFindMoreLikeThisActionsCaptureRankedSources |
| 11 | SortHandoff/{loaded_latest,pending_latest,canceled,superseded}; Reconciliation/reorder/{explicit_origin,empty_scope}; display/TestPresentationContract |
| 12 | Reconciliation/{replacement,merge,reorder,removal,validation_removal,load_failure}; TestBrowsingCollectionChanges, TestFindMoreLikeThisSourceAndSortRetirement and TestLocationMap |
| 13 | CommittedEffects/{callers,chosen_index_revalidation}; Lifecycle/committed_writes; existing SaveChanges cancellation, Export alias/unrelated, FileMutation and MetadataRemoval alias cases; EXIF/mosaic committed cancellation matrices |
| 14 | ChangeKinds/{reorder,removal,content_and_policy}, including unchanged-policy no-ops and cache/producer distinctions |
| 15 | Lifecycle/{close_facts,preparation,queued_chooser,load_recovery,committed_writes}; TestBrowsingVisitLifecycle, TestOpenChooser and TestEscapeUnwindsModesBeforeReset |
| 16 | Compatibility/navigation_snapshots plus existing TestCommandAdmission*, TestWindowCommandAdmissionMatrix, TestCompare*, TestCopySelection*, TestBrowsing* (live/frozen/duplicate scopes and returns), TestFindMoreLikeThis* (query history), TestAdvance*, TestSlide* and Picture-frame keyboard/shutdown cases. These existing tests, not duplicate wrappers, are the finalized compatibility gate |
| 17 | Lead ownership/ordering inventory above plus all AC1-16 evidence |
| 18 | Local checks/inspections below pass; complete native-amd64 CI race and fresh post-suppression Qodana SARIF remain pending this candidate push |

#### Local qualification

Analyzed revision: 6a852718659fbf7302c86fe1edcb44483fb0d91e plus ticket 09 working
tree (candidate commit containing this record). Final comment-only clarification
in viewer.go was reinspected with identical findings. No code changed after the
functional/race run except that comment.

- All 14 Collection families enumerated, then `-count=1 -v -run '^TestCollection'`:
  PASS, 9.869s, no missing or skipped subcase.
- Compatibility selection (CommandAdmission*, WindowCommandAdmissionMatrix,
  Browsing*, FindMoreLikeThis*, CopySelection*, Compare*, Escape, LocationMap*,
  HEIC*, Advance*, Slide*, TogglePictureFrameMode*, Picture-frame keys/shutdown):
  PASS, 40.318s, 578 passing test/subtest entries. Optional
  TestHEICNativeImageOperations skipped because its native opt-in was not set;
  this is not native qualification. Required harness HEIC cases all ran.
- Complete TestVisualSimilarityExplorer: PASS, 14.239s. Existing open/chooser,
  reset, save/export, sort/selection, info/reveal reader regressions: PASS,
  12.162s. Metadata alias/rotation plus reveal cases: PASS, 0.158s. Exact
  noncurrent-alias/loaded-set/competing-commit FileMutation cases recorded in
  `09-mutation.log`.
- Complete uncached verbose filescan/favstore/session: PASS; display
  PresentationContract, Trash symlink and EXIF/mosaic committed-cancellation
  suites: PASS. Their actual case output is retained, not only parent results.
- Focused race: all Collection families, BrowsingVisitLifecycle, Escape,
  generation and file-snapshot contracts: PASS, 143.198s. Full broad race is
  assigned to GitHub's native-amd64 suite under the authorized review workflow.
- `make fmt`, `make check-test-platform verify-build check-test-shards` and
  `git diff --check`: PASS. Includes TUF/generated assets/notices, exact Qodana
  exclusions, all-package vet/build and Docker shard check: 741 runnables.
- GoLand Inspect Code fallback, IDE profile, errorsOnly=false: every one of the
  78 changed Go files since baseline completed. Eight initial timeouts were
  repeated successfully; no partial/missing final results. Exact file scope and
  28 weak duplicate fragments are in `09-inspections.json`; no errors or ordinary
  warnings. All test fragments are independent scenario fixtures covered by
  existing exact DuplicatedCode exclusions. Viewer applyTitle (20 lines) and
  presentDropzone (26) retain their previously assessed title/HEIC-presentation
  overlap: those different presentation conditions are intentional, not another
  collection authority or commit sequence. No broad suppression/extraction added.

Ticket 08's full CI, CodeQL and Qodana jobs passed on 6a852718. That older head
does not qualify this candidate. Keep PR 69 draft until ticket 09's complete CI
gate succeeds, then record completion, remove draft and begin the fresh Codex
code/security review loop. No merge or release is authorized.

### Ticket 09 completed qualification

Commit 2722b1d64f6b61e58ecd37c41f570864d8d92dcb passed the full
[CI run 36410134058](https://github.com/frathe/picfetch/actions/runs/36410134058):
validation, all four native Linux/amd64 race partitions (non-ui/ui-1/ui-2/ui-3),
Linux native, Windows, macOS arm64 and macOS amd64 guards. Raw race events are
retained under `.scratch/ma-030/evidence/ci-2722b1d`: zero failure events and 179
passing Collection test/subtest entries, no Collection skips. The explicit local native
HEIC opt-in skip is not counted as evidence; the configured platform CI guards
retain their established scopes and installed-codec limits.

[CodeQL 36410134005](https://github.com/frathe/picfetch/actions/runs/36410134005)
completed both Go and Actions analyses successfully; the PR ref has no open
code-scanning alerts. FOSSA dependency quality, license compliance and security
analysis all passed.

[Qodana 36410134060](https://github.com/frathe/picfetch/actions/runs/36410134060)
artifact 10964241508 was downloaded and its `/qodana.sarif.json` inspected:
QDGO 262.11335, revisionId exactly 2722b1d64f6b61e58ecd37c41f570864d8d92dcb,
executionSuccessful true, exitCode 0, **zero post-suppression results**. This is
the qodana.starter differential report, not pre-suppression CSV totals. No
licensing failure or incomplete scan was accepted. All AC1-18 gates are met.

The archival commit is documentation-only and carries forward all 78-file local
IDE inspection and code/test evidence from 2722b1d, including the recorded weak
warning dispositions. Final exact stale-completion/Viewer/duplicate-inspection
regressions also passed, 1.513s; the three FileMutation cases passed, 0.335s.
All nine scratch tickets/checklists are resolved; remaining work is the separately
authorized fresh review loop and latest-head checks. Earlier CI or a disposed
review containing findings cannot substitute for a fresh clean final round.

### PR 69 review round 1 — e5a4699

Fresh code review reported one confirmed P2: Close Files and Escape used the
browsable count to decide whether a collection existed. An unavailable-only
Favorite or loss of the final decoder therefore could not be cleared through
Close Files, and Escape closed the window while preserving retained session
membership. This is a missing consequence of D7/AC15, not permission to enable
image commands without an image. The concrete discrepancy and narrow correction
were reported before implementation. Security review completed without findings.
The fresh e5a4699 Qodana SARIF names that exact revision, executionSuccessful true,
exitCode 0 and zero results (run 36411410303, artifact 10964179398).

Lead-owned TDD fix: `collectionSnapshot.HasMembers()` observes full retained
membership without cloning it. Command context keeps `hasCollection` distinct
from browsable `hasFiles`; only Close Files uses the former. Escape resets a
retained-only collection before closing a truly empty viewer. Merge reuses the
same predicate with unchanged behavior. All modal/surface/Stop/Copy Selection
admission precedences remain intact; image-dependent commands remain refused.

Six real entry-point regressions in Lifecycle/retained_only_close cover
Favorite opening and actual HEIC decoder loss through menu, direct Close Files
and Escape. All six failed for the intended symptom before the fix (0.348s),
then passed (0.401s with the policy capability table). An earlier misspelled
fixture helper was a compile error, not behavioral red evidence. The cases also
pin modal ownership, one generation increment, cleared association/session
capture, image-command refusals and the second Escape closing the empty window.

Verification on e5a4699 plus this review fix:

- Uncached verbose Collection*, CommandAdmission*, WindowCommandAdmissionMatrix,
  Escape/Close Files, all HandleKeyEvent*, BrowsingVisitLifecycle and required
  HEIC families: PASS 19.038s, no skips.
- Focused race: CollectionLifecycle, CommandAdmissionVisits, Escape, Close Files
  and HEICBackendLossPreservesSession: PASS 51.704s.
- make fmt/fmt-check, exact Qodana exclusions, shard inventory (741), all-package
  vet and git diff --check: PASS. No top-level runnable/test-file additions.
- GoLand Inspect Code fallback, IDE profile, all severities: collection.go,
  collection_lifecycle_test.go, collection_recovery_test.go, commandadmission.go,
  commandadmission_test.go, commandpolicy.go, drop.go and keys.go all completed
  with zero findings, no timeouts. These supersede prior evidence for those files;
  unchanged files retain the 2722b1d inspection evidence. No native/dependency change.

Standards axis: no actionable violations; the explicit collection predicate
avoids spreading retained-list inspection into UI callers. Spec axis: the one
confirmed AC15 gap is fixed and regression-covered. Review and fixes stayed
with T0 under the repository agreement; no new review/fixer delegation.
Full logs are under `.scratch/ma-030/evidence/review1-*`. After the fix push,
reply/resolve the thread and require another fresh code/security review and
complete latest-head CI/SARIF. Round 1 is not a clean final review.
