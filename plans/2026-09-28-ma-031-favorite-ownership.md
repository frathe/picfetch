# MA-031 implementation and evidence

Status: tickets 01-08 complete; qualifying convergence/native behavior in 09. Baseline: `f09af93` (planning branch), code
baseline `7e52ea5b5fe95ae7d0995e88fa765ab57d549b86`.

Deliver shared validated Favorite membership and captured ownership to every
consumer, with tracked asynchronous Favorite UI storage. Route: **Deep SDD**.
The approved [spec](../.scratch/ma-031/spec.md),
[tickets](../.scratch/ma-031/issues/README.md),
[design](../docs/favorite-ownership.md) and ADR 0006 govern D1-D12 and AC1-AC22.
The user's 2026-09-28 implementation request authorizes ticket commits, pushes,
a draft PR, CI, readiness after all tickets, and the GitHub Codex review loop.
It does not authorize merging or releasing.

## Agreements and interface

The approved specification's three test seams are confirmed by the implementation
request: real temporary `favstore` operations; consumer policy operations; actual
Favorites/root UI actions with tracked queues. Tests use offline/OS fixtures.
Each behavior is developed as a failing test, its minimal implementation, then
focused verification. The lead owns architecture, all review and review fixes.
No new dependencies or shipped assets: existing notices and closure are unchanged.

`favstore.Open(ctx, directory)` returns a fully validated definition and an
immutable owner observation with no open handles. Ordered stored paths remain
distinct from normalized source membership. Relative paths share the working
directory captured at operation entry. `Owner.Acquire(ctx)` opens and validates
bounded directory-relative access; the caller closes it. `Access.Current(ctx)`
revalidates at publication/cleanup boundaries, without claiming atomic external
transactions. An observed retirement is permanent for that observation.
Inventory reads 64-entry batches, validates one entire definition at a time,
and retains only scope intersections (nil scope requests full membership).
It reports healthy owners, unknown owners and enumeration completeness separately.
Unknown owners provide directory-only access for explicit maintenance, never
validated membership or ordinary cache-write authority. List content/version
fingerprints are values; consumer record formats, namespaces and leases stay local.

The read/write definition limit is exactly `64 * 1024 * 1024` encoded bytes.
Identity, size and modification time detect normal changes and replacements;
attribute-preserving in-place edits can evade checks until reopen. Matching owner
associations consume memory. Active OS calls may finish after cancellation.
No watcher, registry, generic worker framework, cache merger, format migration,
source existence check, case folding or symlink resolution is in scope.

## Task graph and routing

`01 -> 02 -> {03, 04, 05}; 05 -> 06 -> {07, 08}; {03, 04, 07, 08} -> 09`.
Tickets commit separately and update this record, `todos.md` and local checkboxes.
Independent tickets may overlap only after contracts settle and with disjoint
files. At most two subagents run concurrently. The available model family differs
from the working agreement's historical tier names; roles still govern routing.

### 01 — Shared storage and Location Map

Owner: T0; one read-only T3 consumer scout. Files: `internal/favstore`
membership/owner/inventory code and behavior tests, `internal/ui/locationmap/favorites*`,
root GPS tests as needed, architecture and exact Qodana exclusions.
Contract: Open/Owner/Access plus full or scoped inventory as above.
Test: full validity/path/limit matrix; stable reads and permanent retirement;
cancellation; healthy/unknown inventory and resource growth; Map persistence and
retired cleanup with no idle roots. Verify: ticket 01 commands, enumerated first.
Budget: 2 spawns maximum; 2 planned review rounds; no full suite.

### 02 — Similarity and retained search

Owner: T0 (cross-package producer lifetime). Files: `internal/similarity/cache*`,
`analyze.go`, `search*` and focused existing tests.
Depends: 01. Contract: shared observations retain the complete original input
scope independently of prepared/ranked items and producer write permission.
Test: all owners; partial inventory/opt-outs; preparation-prefix save refresh;
move/replacement retirement; bounded handles; retained query and write scope.
Verify: ticket 02's analysis-cache, ownership, search-save and resource selections.
Budget: 1 bounded scout if necessary; 2 review rounds; no full suite.

### 03 — Maintenance

Owner: T0. Files: similarity cache inventory/records/management/transaction and
their tests. Depends: 02. Contract: deferred maintenance keeps value observations,
reacquiring bounded access; unknown membership protects automatic cleanup.
Test: explicit clear, replacement/fresh-record survival, lease revocation,
quiescence, committed accounting and handle growth. Verify: ticket 03 commands.
Budget: 0 spawns; 2 review rounds; no full suite.

### 04 — Cohorts

Owner: T1 candidate after 02, otherwise T0. Files: `internal/favstore/cohorts.go`
and existing `favstore_test.go` (2 files); root/Explorer review remains T0.
Depends: 02. Contract: CohortStore retains shared Owner and complete membership;
legacy/current documents and containment stay unchanged.
Test: strict definition; stale/fresh owners; legacy/rollback and association.
Verify: ticket 04 commands. Budget: 1 spawn; 2 review rounds; no full suite.

### 05 — UI reads and lifecycle

Owner: T0. Files: `internal/ui/favorites`, root composition/shutdown/harness,
root ownership integration tests, manifests.
Depends: 02. Contract: feature-owned worker lifetime/UIQueue; Close/Stop/Wait/Settle;
complete opening carries captured owner and current admission into existing replay.
Test: held refresh/open, coalescing, root/session replacement, admission and
shutdown including existing removal. Verify: ticket 05 selections.
Budget: 1 bounded test-fixture task if worthwhile; 2 review rounds; no full suite.

### 06 — Saves

Owner: T0. Files: favstore mutation code/tests, Favorites save/lifecycle and root
integration. Depends: 05. Contract: captured target, conflict/cancel/committed
outcomes; one instance mutation lane shared with removal; committed Owner handoff.
Test: exact write limit, pre/post-publication cancellation, occupied/replaced
target, held serialized mutations, failed refresh and current/retired search.
Verify: ticket 06 selections. Budget: 0 spawns; 2 review rounds; no full suite.

### 07 — Removal

Owner: T0. Files: favstore removal/tests, Favorites manage/confirmation/lifecycle
and root integration. Depends: 06. Contract: confirmed captured owner, same mutation
lane, native-call completion separate from cancellation and queued presentation.
Test: replacement conflict, unstarted cancellation, held native call, post-commit
reconciliation and shutdown. Verify: ticket 07 selections.
Budget: 0 spawns; 2 review rounds; no full suite.

### 08 — Previews

Owner: T0; T1 only for a fixed disjoint adapter if justified at its frontier.
Files: `internal/favthumbs`, root `favthumbs.go`, Favorites handoff and tests.
Depends: 06. Contract: opening/committed save carry original captured owner to
every read/write/sweep; complete membership remains independent of decode prefix.
Test: held reads/publication/cleanup, moved/removed/identical replacement,
fresh owner, cached/offline tail and source-version guards.
Verify: ticket 08 selections. Budget: 1 spawn; 2 review rounds; no full suite.

### 09 — Convergence and qualification

Owner: T0. Files: remaining adapters, native guard runner/workflow if needed,
architecture and evidence. Depends: 03/04/07/08.
Contract: no duplicate membership/ownership authority or temporary bridge remains.
Test: all parent ACs; native Windows idle removal/move, active release and file
URIs; native macOS same filesystem/URI/lifecycle cases. Extend native runner with
exact mandatory names, rejecting skipped/missing cases. CI supplies native hosts.
Verify: parent AC map, manifests, `make verify`/equivalent full CI; GoLand every
changed Go file including weak warnings; fresh Qodana post-suppression SARIF.
Budget: 1 mechanical native-inventory task; 2 review rounds plus bot findings;
full suite once via CI (review-loop rule avoids duplicate broad local race suite).

Native qualification scout (09 reconnaissance overlaps 06): G1 yes, bounded
runner/workflow inventory; G2 yes, cited paths and exact commands checked by lead;
G3 yes, read-only; G4 yes, native CI spans independent workflow/runner families;
G5 yes, unrelated to lead's hot save/lifecycle context. No implementation or
review delegated. T3 uses available gpt-6-luna. One spawn within ticket-09 budget.

Native frontier from scout, checked against `scripts/nativeguards/main.go`:
both Windows and macOS suites already enforce exact parent/subtest run/pass
events and reject missing/skipped descendants. Register shared ownership
move/idle-removal plus a new explicit `active_release` case, cancelled read and
save-publication cases, both `TestAnalysisCacheFileURIPathsReopen` cases,
Favorite storage lifecycle/native-call completion and root ownership integration.
Add preview ownership cases after 08 names settle. Existing CI runs Windows and
both macOS architectures and retains raw event artifacts; no new runner is needed.

## Delegation decisions

Consumer scout: G1 yes (bounded ownership/held-test search); G2 yes (verify cited
locations using `rg`/targeted reads); G3 yes (no edits); G4 yes (three independent
consumer families); G5 yes (lead holds Map/shared storage context). S does not
replace cross-file lifetime tracing; W satisfied (no implementation handoff).
Other delegations are candidates only; record G1-G5 before assigning them.
The user requested the best suited agents per ticket, not a separate implementer
for every cross-cutting slice. Keep hot context and design-bearing work with T0.

Preview 08 fixture adapter: G1 yes, existing tests receive the fixed owner-valued
API; G2 yes, complete `internal/favthumbs` test command after lead implementation;
G3 deviation: four existing test files exceeded the three-file gate; the fourth
was the benchmark's mechanical call-site adaptation and should have stayed
inline. All four remained disjoint and were independently verified. G4 yes, fixtures are disjoint from
lead production/root and new ownership tests; G5 yes, lead holds production
context but has not loaded their full fixture corpus. T1 gpt-6-sol adapts fixtures
without new test design or review; lead owns new ownership guards and all fixes.
S alone cannot preserve held-operation fixture setup. W: signatures and fixture
contracts only, no implementation delegated.

Cohort ticket 04: G1 yes (two-file adapter and specified retirement matrix);
G2 yes (`go test ... ./internal/favstore -run '^TestCohorts$'`); G3 yes
(`cohorts.go` and `favstore_test.go`, disjoint from lead maintenance); G4 yes
(one consumer and the settled shared interface); G5 yes (lead has not built
cohort implementation context). S cannot implement semantic lifetime changes;
W satisfied (contract, not code). T1 maps to available `gpt-6-sol`, high effort.
Lead owns review, Explorer/root qualification, fixes and the ticket commit.

## Evidence and completion

- [x] Approved D1-D12 and test seams read; clean branch confirmed.
- [x] Deep plan and task graph recorded; no PR existed at start.
- [x] 01 shared ownership/Map
- [x] 02 similarity/search
- [x] 03 maintenance
- [x] 04 cohorts
- [x] 05 UI reads/lifetime
- [x] 06 saves
- [x] 07 removal
- [ ] 08 previews
- [ ] 09 complete qualification
- [ ] Ready PR; latest-head clean Codex code/security, required CI and CodeQL

Raw focused output will live under `.scratch/ma-031/evidence/`; this tracked
record retains commands, required cases, outcomes and revision references.
Local Go uses `/snap/go/current/bin/go` because the sandbox cannot launch snap's
wrapper. This is the installed Go 1.27.1 toolchain, not a dependency change.
GoLand inspection tools are available; use `get_file_problems(errorsOnly=false)`
to include weak warnings rather than warning-threshold batch filtering.

| Ticket | Spawns budget/actual | Review rounds | Full suite | State |
| --- | --- | --- | --- | --- |
| 01 | 2/1 | 2 | no | complete; native qualification remains 09 |
| 02 | 1/1 | 2 | no | complete |
| 03 | 0/0 | 2 | no | complete |
| 04 | 1/1 | 1 | no | complete; lead independently verified |
| 05 | 1/0 | lead gates below | no | complete |
| 06 | 0/0 | lead gates below | no | complete |
| 07 | 0/0 | lead gates below | no | complete |
| 08 | 1/1 | 2 | no | complete; lead independently verified |
| 09 | 1/1 | 0 | CI | claimed; native scout completed during 06 |

### Ticket 01 evidence

Implemented Open/Owner/Access/Observe and scoped Inventory. Definition.Files
captures relative interpretation and preserves Fyne URI conventions. Load/Count
already delegate; their UI scheduling remains 05. Location Map retains GPS
format, complete-byte/modification-time namespace algorithm, source-version
policy, serialized invalidation and 1,024-entry cleanup budget. Roots are bounded
per-operation acquisitions. No temporary decoder or ownership authority added.

Observed red -> green: duplicate/escaped keys, numeric aliases, null/empty/NUL
paths; oversized input; replaced owners; delayed relative interpretation;
healthy/unknown inventory; Map's two idle roots. Cancellation and unstable-read
guards also failed under deliberate temporary context-detachment and omitted
final-validation mutations, then passed after restoration. Growth requires the
size-limit error rather than an incidental later identity error.

Enumerated all five shared suites, seven Map Favorite tests and root LocationMap.
These uncached selections and checks passed:

```
go test -tags no_emoji,nodynamic -count=1 -v ./internal/favstore
go test -tags no_emoji,nodynamic -count=1 -v ./internal/favstore ./internal/ui/locationmap -run '^TestFavorite'
go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestLocationMap$/^gps_cache'
make fmt-check check-qodana-test-exclusions check-test-shards-direct
go vet -tags no_emoji,nodynamic ./internal/favstore ./internal/ui/locationmap ./internal/ui
```

Root cases included held removal/replacement, corrupt records, write failure,
live ownership, save promotion, fresh opening and cancelled live reuse. Raw
output: `.scratch/ma-031/evidence/01-storage-map.txt` and `01-root-gps.txt`.
All 741 existing root shard assignments match; three new test files have exact
Qodana exclusions. No new root top-level test was needed.

Resource measurement at 32/130 Favorites: unrelated retained owners 0/0,
matching owners 32/130, idle Favorite handles 0/0, peak Favorite handles 3/3.
GC heap observations: unrelated -14,512/-1,768 bytes (runtime noise), matching
+38,736/+126,296 bytes. Matching associations grow; encoded bytes do not bound
total process memory. One complete definition is processed per inventory step.
Map cleanup has constant root/enumerator depth independent of Favorite count.

GoLand `get_file_problems(errorsOnly=false)` inspected all nine changed Go files
on the ticket-01 working tree based on f09af93. Fixed one package-name-shadowing
warning; reinspection is clean. No timeouts/skips. This is the documented IDE
fallback, not Qodana-profile equivalence. Native idle removal/move has a guard
without platform skips; Windows/macOS execution, full CI and fresh Qodana SARIF
remain 09/PR gates.

### Ticket 02 evidence

Similarity and retained search now use the same shared owner contract as Map.
Each producer clones its complete original input scope independently of its
prepared items and write scope; explicit-save refresh preserves it. All matching
owners remain indexed. Producer reads/writes acquire and close owner-relative
access, checking retirement before accepting records and before publication.
Healthy partial inventory, unknown-member fallback protection, opt-outs,
Favorite-only pressure policy and producer leases remain consumer-owned.
`Owner.Same` compares captured values for explicit-refresh change detection;
it does not replace a fresh currentness check.

Observed red -> green: moved owner remained current; move-following reads were
accepted; producer retained unrelated membership and refresh did not retain its
explicit complete scope. Added identical-content replacement and fresh-owner
cases, plus original-scope refresh with zero, one and two reversed prepared
items. Existing retained-search save does not restart preparation or query.

Enumerated all 30 selected top-level suites. Uncached verbose output shows no
skips/failures for analysis-cache policy/maintenance, ownership and search-save:

```
go test -tags no_emoji,nodynamic -count=1 -v ./internal/similarity -run '^(TestAnalysisCache.*|TestFavoriteAnalysisOwnership|TestSearchSessionFavoriteSaveRetainsQueryAndPreparation)$'
go test -tags no_emoji,nodynamic -count=1 -v ./internal/favstore -run '^TestFavoriteInventory$/^resources$'
go vet -tags no_emoji,nodynamic ./internal/similarity ./internal/favstore
make fmt-check check-qodana-test-exclusions check-test-shards-direct
```

Raw evidence: `02-similarity.txt` and `02-shared-resources.txt` in the local
evidence directory. 32/130-owner retained producers held one source and every
matching owner, zero idle Favorite handles, and allowed every native directory
move. GC heap deltas were +1,512/+123,936 bytes including runtime noise; matching
associations grow while unrelated members are not retained. Shared inventory
also repeats its peak/idle measurements. Native Windows/macOS remains 09.

GoLand inspected all eleven changed Go files on the ticket-02 working tree
based on 8cd3616, including weak warnings. Two partial-result warnings were
confirmed false positives (always-returned inventory values), explained and
narrowly suppressed; affected files re-inspected clean without timeout/skip.
No dependencies, new test files or root test names changed.

Temporary maintenance bridge: `cache_records.go` acquires a shared owner to
open its analysis directory, but the old maintenance inventory still retains
analysis roots. Ticket 03 replaces those deferred handles with values and
record revalidation; there is no remaining parallel membership decoder there.

### Ticket 03 evidence

Removed the maintenance bridge: inventories now retain owner, cache-directory
and record identity values only. `cache_access.go` owns bounded reacquisition,
classification and final owner/directory/record revalidation. Enumeration reads
64-entry batches without a count cap. Every acquisition closes before moving
to the next Favorite or record. The existing exclusive lease, quiescence,
revocation, temporary-file treatment, payload format and transaction ledger stay
in similarity. Unknown membership blocks stale classification; explicit clear
may still remove recognized records in the captured directory.

Observed red -> green: clear followed moved owners and identical/replaced lists;
both modes deleted fresh records and followed replaced analysis directories;
deferred inventories retained one root per Favorite. New tests replace these
observations after the full inventory via the first committed removal callback.
Native unknown-membership clear/stale cases complement the existing Unix-only
unreadable-symlink fixture; they do not depend on a platform skip.

Enumerated 30 analysis-cache/ownership suites. Uncached output in
`03-maintenance.txt` and `03-shared-inventory.txt` confirms all selected cases
passed, without local skips. Commands:

```
go test -tags no_emoji,nodynamic -count=1 -v ./internal/similarity -run '^(TestAnalysisCache.*|TestFavoriteAnalysisOwnership)$'
go test -tags no_emoji,nodynamic -count=1 -v ./internal/favstore -run '^TestFavoriteInventory$/^(membership|resources)$'
go vet -tags no_emoji,nodynamic ./internal/similarity
make fmt-check check-qodana-test-exclusions check-test-shards-direct
```

This includes healthy peers, unknown owners, stale-list replacement, writer
retirement, quiescence, partial failures and exact cancelled-removal accounting.
32/130 deferred inventories retain all records, zero idle Favorite handles,
three peak handles, and permit every directory move. Measured GC heap growth
was +31,512/+233,984 bytes including runtime noise. Value associations grow;
handles do not. General-directory observations are local cache facts, not a
second Favorite ownership authority.

GoLand inspected all four changed/new maintenance Go files on the ticket-03
working tree based on 19d8786, including weak warnings: clean, no timeout/skip.
No new test files or root test names; manifests remain exact. Native qualification
is still ticket 09. Cohort changes concurrently present were not part of this
ticket's implementation or commit.

### Ticket 04 evidence

T1 changed only `cohorts.go` and its existing test file. Lead reviewed and
independently executed the acceptance commands. Cohorts now retain shared
Owner plus complete normalized membership, acquiring bounded access for reads
and atomic saves. Legacy arrays and version-2 documents, presets, group order,
unassigned membership, source spelling, containment and association semantics
remain consumer-owned and unchanged. No idle root or second definition decoder.

T1 observed strict-definition tests fail for duplicate numeric aliases, invalid
later paths and oversized lists, then pass after adoption. Extended cases prove
move/change retirement remains permanent after restoring the pathname/list,
identical list and directory replacements reject stale saves, fresh owners save,
relative membership is complete, and cancellation preserves the cohort document.

Enumerated required storage, Explorer and root suites and retained uncached output:

```
go test -tags no_emoji,nodynamic -count=1 -v ./internal/favstore -run '^(TestFavoriteMembership|TestFavoriteMembershipLimits|TestFavoriteOwnership|TestCohorts)$'
go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui/explorer -run '^(TestFeatureFavoriteSaveRollback|TestFeatureCohortRetainsCapturedMembership)$'
go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestVisualSimilarityExplorer$/^(create_cohort_favorite|favorite_identity_cancel)$'
go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestCollectionReplay|TestCollectionCapture)$'
go vet -tags no_emoji,nodynamic ./internal/favstore ./internal/ui/explorer ./internal/ui
```

All passed without skips: both cohort/preset rollback subcases; captured cohort
membership; Favorite creation; scan/sort/merge identity cancellation; replay and
capture association. Raw output: `04-cohorts-storage.txt`, `04-cohorts-explorer.txt`,
`04-cohorts-root.txt` and `04-collection.txt`. Formatting/manifests also passed
on the combined worktree. GoLand inspected both changed Go files including weak
warnings on the tree based on 19d8786/8144d9e: clean, no timeout/skip. No new files,
test exclusions, shard rows, formats or dependencies were needed.

### Ticket 05 evidence

Ordinary discovery/count/open now use shared validation on tracked workers.
Refresh retains the last complete menu, coalesces requests and discards retired
delivery. Open carries the captured owner plus complete occurrences through
existing replay, checking admission again. Favorites owns the fourteenth test
UIQueue; root shutdown and harness stop/join it before preview/collection cleanup.
Native removal is tracked through OS return, not an unobserved UI callback.

Observed red -> green: held reads blocked UI; queued callbacks delivered retired
opens; identical replacement passed the old path-based load; source replacement
failed to invalidate a held open. The final root guard compares durable scan
handles, not a transient active counter. Disabling CancelOpen made its source
case fail with "admitted another scan"; restoring it passed. Feature lifecycle
guards also fail under that mutation. Raw guard: `05-root-guard-red.txt`.

Enumerated and ran uncached shared membership/limits, complete Favorites feature,
root ownership/admission/replay/capture/shutdown, all Favorite-named root tests,
startup, Explorer, Map, browsing and modal/compare regressions.
Logs: `05-membership.txt`, `05-feature.txt`, `05-root-ownership.txt`,
`05-root-focused.txt`, `05-root-consumers.txt` in the local evidence directory.
All selected cases pass without skips. Focused Favorites race suite and vet for
favstore, Favorites and root also pass. Formatting and exact exclusions pass;
root shard inventory contains 742 tests, including the new integration suite.
Correction found in 06: the optional Explorer-local parent requires the
`explorertrial` tag and was not included in these untagged runs; it is not
claimed as ticket-05 runtime evidence. Ticket 06 explicitly qualifies the changed
native fixture with that tag and pinned assets.

GoLand inspected all 34 changed/new Go files, including weak warnings, on the
ticket-05 tree based on 013aa9f. Final changed guards re-inspected clean. No
timeout/skipped file. Intentional fixture duplicates in browsing, compare,
Explorer-local, Favorites, step and visual-search tests are covered by their
existing exact Qodana exclusions. Two viewer.go weak duplicate matches resolve
only to the ignored `.scratch/ma-028/linux-reset-repair-2026-09-27/mutant-viewer.go`
historical artifact, excluded from the production review scope (not shipped or
present in CI); no production suppression or unrelated artifact edit was added.

Temporary bridges remain explicit: synchronous Save/Exists move to the captured
mutation lane in 06; root's captured-open owner currently forwards to the legacy
preview API until 08. Neither bridge adds a definition decoder. AGENTS documents
queue/shutdown ordering using writing-for-agents guidance. No dependency changes.

### Ticket 06 evidence

`Target` captures existing directory/list identities and absent-name/root facts
without idle handles. Detected conflicts permanently retire it. Save validates
all occurrences and encoded bytes, checks cancellation while writing a private
temporary file and revalidates immediately before atomic publication. The result
explicitly retains Committed plus the published file's own owner; it never reopens
the name after publication to recover ownership. Legacy Save delegates here.

Favorites captures files before naming and root/name/provider before workers.
One feature-local admission-ordered queue serializes saves and existing native
removal, including cancelled jobs between live jobs. Occupied/changed targets
require a fresh Cancel-default confirmation; a vacated conflicted name returns
to naming. Committed effects notify current consumers and hand off the captured
owner even after Close/root replacement or failed menu refresh. Presentation
stays generation-bound; Stop suppresses delivery. Preview persistence itself
remains the documented 08 bridge; removal confirmation becomes identity-bound
in 07 on this same queue.

Observed red -> green: empty/NUL/oversized/escaped oversized saves replaced the
old list; stale and identical targets were overwritten; pre-publication cancel
committed and post-publication cancel lost ownership; queued saves delivered
inline; changed-then-vacated targets revived. Negative verification removed the
final publication check (both conflict and cancelled-save cases failed), and
suppressed post-close delivery (committed-effect guard failed). Restored code
passes. Raw red logs: `06-save-red`, `06-conflicts-red`, `06-ui-red`,
`06-target-retirement-red`, `06-publication-guard-red`, `06-committed-guard-red`.

Enumerated exact storage/feature/root parents. Uncached green evidence includes:

```
go test -tags no_emoji,nodynamic -count=1 -v ./internal/favstore -run '^(TestFavoriteConflicts|TestFavoriteCancellation|TestFavoriteMembershipLimits|TestSave.*)$'
go test -race -tags no_emoji,nodynamic -count=1 -v ./internal/ui/favorites
go test -tags no_emoji,nodynamic -count=1 -v ./internal/similarity -run '^TestSearchSessionFavoriteSaveRetainsQueryAndPreparation$'
go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestBrowsingActionTargets|TestFavoriteOwnershipIntegration|TestFindMoreLikeThisActionsCaptureRankedSources|TestCollectionCapture|TestCollectionReplay)$'
go test -race -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestHEICUnavailableFiles$'
go test -tags no_emoji,nodynamic,explorertrial -count=1 -v ./internal/ui -run '^TestVisualSimilarityExplorerLocal$'
```

All passed without skips. Logs: `06-storage.txt`, `06-feature.txt`, `06-search.txt`,
`06-root.txt`, `06-heic.txt`, `06-explorer-local.txt` (all 23 native child cases,
87.05 seconds). Exact root inventory remains 742. Focused vet and Make formatting,
exclusions and shard checks pass. New storage test has its exact Qodana exclusion.

The diagnosing-bugs loop reproduced the ticket-05 CI ui-3 failure: the HEIC
Favorite fixture waited for scan before draining Favorite delivery. Its one-line
settlement fix passes locally under race. The optional Explorer-local suite had
been omitted by the untagged selection (corrected above). Installed the existing
checksum-pinned Linux ONNX 1.29.0/model assets via the repository installer,
retaining its license/notices, no dependency/source/version change. Real-worker
qualification found an obsolete move-following cache expectation and a preexisting
literal facts-version-1 expectation. Tests now require frozen records at move,
fresh-owner completion, and the existing `similarity.FactsVersion` constant.
Focused reproductions and the complete tagged parent pass. No runtime policy
was weakened to make these tests pass.

GoLand inspected all 22 changed/new Go files including weak warnings on the tree
based on 7a9d174; final edited files re-inspected. No timeout/skips or actionable
production warnings. Remaining test duplicate fragments in browsing, Explorer-local,
Favorites and visual-search tests retain their existing exact exclusions. Agent
guide records the mutation/commit rules using writing-for-agents guidance. T0
owned this cross-package implementation; independent T3 native-runner discovery
for 09 ran alongside it. Complete latest-commit CI remains the later PR gate.

### Ticket 07 evidence

Removal now captures a shared Target on the same serialized queue as Save,
before presenting confirmation. Its storage operation revalidates directory/list
identity, closes all acquired handles before native Trash, and reports Committed
separately from cancellation. Legacy Remove delegates to that operation. A
conflict recaptures only to ask a fresh question, never to retry automatically.
Manage close retires its capture/prompt/unstarted work; active native calls stay
tracked until return. Committed completion refreshes current menus after Close or
root changes, while stale errors/toasts/dialogs and collection replay stay blocked.
Cached owners retain the existing next-access retirement policy; preview disk
ownership is still the explicitly scheduled 08 migration.

T0 owned the slice; no second scheduler or extra worker lifetime. Observed
red -> green: actual Manage confirmation trashed an identical replacement.
Disabling Target.current negatively verified all three storage move/directory/
identical-list guards: the native stub was called and replacements were removed.
Restoring the gate passed. Logs: `07-removal-red.txt`, `07-owner-guard-red.txt`.

Enumerated exact suites and retained uncached output:

```
go test -tags no_emoji,nodynamic -count=1 -v ./internal/favstore -run '^(TestFavoriteOwnership|TestFavoriteConflicts|TestRemove.*)$'
go test -race -tags no_emoji,nodynamic -count=1 -v ./internal/ui/favorites
go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestFavoriteOwnershipIntegration$'
go vet -tags no_emoji,nodynamic ./internal/favstore ./internal/ui/favorites ./internal/ui
make fmt-check check-qodana-test-exclusions check-test-shards-direct
```

All passed without skips (`07-storage.txt`, `07-feature.txt`, `07-root.txt`).
Cases include malformed/missing lists, pre-submission cancellation, cancellation
inside a committed native move, held capture retired by Manage/root/Stop,
native success/failure after close, root replacement and production shutdown;
existing focus/superseded-dialog/removal regressions remain green. Root uses real
Manage input and temporary native-move stubs, preserving its collection and scan
handle. No test source or top-level root name added; exact manifest remains 742.

GoLand inspected all twelve changed/new Go files including weak warnings on the
tree based on c5e5a83: clean, no timeouts/skips. Architecture and agent guide record
the captured-removal and Manage-close lifetimes. No dependency/format changes.
CI on ticket 06's pushed commit is fully green; post-suppression SARIF inspection
and latest-commit native/full verification remain 09's gates, not implied here.

### Ticket 08 evidence

All preview disk APIs now consume the Owner captured by complete Favorite Open
or committed Save. Root's pathname bridge is removed. Each bounded read, write
and sweep reacquires that observation without rediscovery; operations are
directory-relative, publication/corrupt cleanup/sibling cleanup check currentness,
and sweep additionally rechecks the inventoried record before unlink. Only the
thumbs child can be created. Original decoding retains no Favorite access.
Existing JPEG/PNG formats, quality, names, source versions, decode prefix,
cached tails, memory admission and full-membership/offline cleanup stay intact.

Observed red -> green: all four moved/removed/identical-list/replaced-directory
held cache hits created preview storage. Deliberately reopening owners or removing
final guards then failed held original, publication, corrupt cleanup, sweep and
fresh-record tests. Root's temporary fresh-name lookup failed both queued actual
open and save cases. Restored implementations pass. Raw negatives:
`08-owner-red.txt`, `08-preview-guard-red.txt`, `08-root-guard-red.txt`.

Uncached verified commands (Go toolchain path as above):

```
go test -race -tags no_emoji,nodynamic -count=1 -v ./internal/favthumbs ./internal/ui/favorites
go test -race -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestFavoriteOwnershipIntegration|TestFavoritePreviewAfterCommitBeforeNotificationRejectsOldMemoryHit|TestShutdownCancelsFavoritePreviews)$'
go test -race -tags no_emoji,nodynamic -count=1 ./internal/ui -run 'Test(SyncFavoritePreviews|FavoritePreview|ShutdownCancelsFavoritePreviews|HEIC)'
go vet -tags no_emoji,nodynamic ./internal/favthumbs ./internal/ui/favorites ./internal/ui
make fmt-check check-qodana-test-exclusions check-test-shards-direct
```

All test selections passed without skips; full preview/feature logs are
`08-preview-feature.txt`, root ownership/shutdown output `08-root.txt`. Broader
root preview/HEIC regression selection passed in 20.041s. Source replacement,
committed-file notification, held-read/slot-waiter cancellation, cache budgets,
offline tail and fresh-owner completion remained green. Exact root inventory
remains 742; the new preview test file has its exact Qodana exclusion.

GoLand inspected all thirteen changed/new Go files including weak warnings on
the tree based on d7626ca. No timeouts/skips. The only two weak duplicate fragments
are the independent cancellation/retirement publication fixtures in store_test
and ownership_test, covered by exact test exclusions. Lead independently verified
the agent's four-file fixture adaptation and reviewed currentness boundaries.
Architecture and agent guide now document captured preview ownership, with the
writing-for-agents skill keeping this beside the existing source-version rule.
No dependencies, shipped assets or persistent formats changed.

### Ticket 09 native inventory (fixed before implementation)

Use a separate focused `favorite-ownership` native suite on Linux, Windows and
both existing macOS architectures. Adding root UI to the full native package
suite would also select Linux-only golden tests, so a parent-name filter is
required. No codec-exception flag applies. Retain raw JSON and fail missing,
skipped or failed registered parents/children using the existing event validator.

Required parents and children:
- favstore `TestFavoriteOwnership`: move, directory_replacement, list_replacement,
  list_change, idle_removal, new active_release (cancel, explicit close, then move).
- favstore `TestFavoriteCancellation`: cancelled, changed_while_reading, growth,
  save_before, save_after, save_after_replacement.
- favstore `TestFavoriteConflicts`: publication, retirement, occupied,
  identical_list, directory, base, missing_list; removal_identical_list,
  removal_directory, removal_move, removal_missing_list, removal_malformed,
  removal_cancelled, removal_committed_after_cancel.
- similarity `TestAnalysisCacheFileURIPathsReopen`: general, favorite.
- favthumbs `TestSyncFavoriteOwnership`: held_read, publication, cleanup, sweep,
  fresh_record, move, remove, identical_list, directory.
- favorites `TestFavoriteStorageLifecycle`: close, source, root, modal, stop,
  active_native_call; `TestFavoriteStorageCommittedEffects`: removal_close,
  removal_root, removal_stop, removal_failed_after_close, close, root, stop,
  replacement, refresh_failure.
- root `TestFavoriteOwnershipIntegration`: preview_queued_open, preview_queued_save,
  native_removal_close/root/shutdown, committed_save_close/opt_out/shutdown,
  queued_replay, modal, source, malformed, identical_replacement, shutdown.
- root `TestFavoritePreviewAfterCommitBeforeNotificationRejectsOldMemoryHit`
  and `TestShutdownCancelsFavoritePreviews`.

T0 implements this cross-package qualification; the bounded native scout was
already completed in 06. No further delegate can improve the hot final-gate work.

The qualification implementation is pushed before closing ticket 09 because
native CI must execute a real committed revision. Ticket 09 remains claimed;
its completion/evidence commit follows verified native/full results. Local
`nativeguards -suite favorite-ownership` passed all 76 exact requirements on
Linux/amd64, with raw JSON in `09-native-linux.json`. Runner tests reject every
required result when missing, skipped or failed. Initial unknown-suite and
missing-workflow reds are `09-native-inventory-red.txt` and
`09-native-workflow-red.txt`. Making Access.Close a no-op fails active_release
(`09-active-release-red.txt`); restored code passes. GoLand inspected all three
changed Go files including weak warnings on the c58a07f-based tree: clean,
no timeouts/skips. Focused vet, runner tests and Make formatting/exclusions/shards
pass; no new test file or root top-level name. Complete platform/CI acceptance
is still pending and is not inferred from local success.
