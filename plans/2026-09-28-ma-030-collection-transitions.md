# MA-030: authoritative collection identity and committed transitions

Status: implementation in progress; tickets 01-02 complete, ticket 03 next.
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
no-op admission preserves association and generation.
Test/verify: `TestCollectionMerge`, model operations/merge,
FavoriteAssociation/merge, Admission/merge, Reconciliation/merge;
`TestHEICUnavailableFiles` and `TestBrowsingCollectionChanges`.
Budget: <= 1 spawn, <= 2 planned review rounds, full suite no.

### 04 — saved replay and capture

Owner: T0 inline; bounded filescan implementation eligible after exact contract.
Files: filescan, drop/session/favorites adapters, model capture, related tests.
Depends: 03. Contract: explicit replay input uses recorded occurrences, no sibling
expansion; ordinary discovery unchanged; current separate admission limits.
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
| 03 | 1/0 | 0 | no | Pending |
| 04 | 1/1 | 0 | no | Read-only replay scout complete |
| 05-08 | 1 each/0 | 0 | no | Pending |
| 09 | 1/0 | 0 | CI | Pending |

### Progress

- [x] Frame, settled specification and agreed test seams read.
- [x] Deep plan, task graph, file map, routing, contracts and budgets recorded.
- [x] 01 coherent snapshots and navigation.
- [x] 02 atomic replacement and Favorite association.
- [ ] 03 retained merge.
- [ ] 04 saved replay and capture.
- [ ] 05 latest-choice sort handoff.
- [ ] 06 batch removals and shared survivor result.
- [ ] 07 unavailable retention and scoped recovery.
- [ ] 08 committed writes and policy distinctions.
- [ ] 09 convergence, complete qualification and draft removal.
- [ ] Fresh clean latest-commit Codex/security reviews and required CI.

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
