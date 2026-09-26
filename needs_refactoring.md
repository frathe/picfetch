# PicFetch — Open Refactoring Backlog

Updated 2026-09-26 after a cross-feature architecture assessment of PR review
history and the current implementation.

This file contains proposed refactorings and accepted dependency watches.
Completed findings have been removed; their history remains in Git and the
[maintainability implementation plan](finished_refactorings/2026-09-06-maintainability-plan.md).
Existing MA identifiers and accepted decisions are preserved: MA-024 verifier
size refactoring remains declined, and MA-025 records acceptance of the decoded
map-cache limitation previously recorded under MA-015 and in [todos.md](todos.md).
MA-026 is complete in `28d65ef`, with full CI qualification recorded in its
[archived plan](finished_refactorings/2026-09-14-explorer-feature.md).
MA-027 is complete in `e6024dc`, with native CI and clean code/security reviews
recorded in its [archived plan](finished_refactorings/2026-09-14-ma-027-presentation.md).
MA-028 remains the recommended first task. MA-029 through MA-033 below are
proposals, not accepted implementation plans or unresolved PR defects.

Historical inspection baseline: `main` at `54fd7c3` (v1.1.2). At that revision,
the root `internal/ui` package contained 55 production Go files, 10,835 non-test
lines including comments, and 275 methods on `viewer`. The refactoring goal is to move feature state and
background-work ownership behind small module interfaces while retaining
explicit cross-feature composition in `internal/ui`.

| ID | Priority | Remaining work | Status |
| --- | --- | --- | --- |
| [MA-028](#ma-028) | P1 | Share command policy across all entry routes | Recommended first; medium scope |
| [MA-029](#ma-029) | P1 | Give browsing visits one explicit state owner | Recommended; large, incremental |
| [MA-030](#ma-030) | P1 | Deepen collection identity and committed transitions | Recommended; large, incremental |
| [MA-031](#ma-031) | P2 | Share Favorite membership and ownership primitives | Recommended; medium scope |
| [MA-032](#ma-032) | P2 | Consolidate proven worker-lifetime mechanics | Conditional extraction; medium scope |
| [MA-033](#ma-033) | P2 | Capture launch side-effect policy once | Recommended independent small task |

Priorities express architectural value, not the severity of a currently open bug.

## 2026-09-26 assessment: the application, not just Location Map

**Recommendation:** introduce focused state management for browsing and collection
transitions, plus one shared command-decision module. Keep feature-owned state
inside the features. A global mutable application store, service locator, or
automatic feature registry would increase the number of modules that can depend
on each other's details without resolving the observed ordering problems.

The repeated weakness is that an interaction's complete contract spans several
owners: which surface owns input, which visit owns navigation, which collection
revision a result belongs to, and when a committed change may become visible.
Much of this is currently correct, but extending it requires remembering a long
set of coordinated edits. This is primarily coupling through shared rules and
call ordering; the evidence does not establish a package-import-cycle problem.

### Evidence and limits

Code examination began on `feature/image-map` at `7a161a2`. Concurrent work
committed the existing Location Map edits as
`9b7ceaada465ebc8890cde6b94f953f8567c85c5` during the assessment; the final PR head
and dispositions were checked again at that revision. This assessment did not
make those fixes. Code references below identify files and symbols in that
checkout; historical review links describe their reviewed revisions.

The review-thread inventory was surveyed across the repository, then the
following PRs were selected for relevant, repeated patterns. Counts include
non-Codex threads where present and are **not confirmed-defect counts**.

| PR | Review threads | Relevant evidence examined |
| --- | ---: | --- |
| [17: maintainability audit](https://github.com/frathe/picfetch/pull/17) | 9 | Sort generations, variant identity, committed writes and cache invalidation |
| [18: Similarity Explorer](https://github.com/frathe/picfetch/pull/18) | 33 | Menu/shortcut parity, cohort return, collection replacement, Favorite ownership |
| [25: visual search and cache controls](https://github.com/frathe/picfetch/pull/25) | 47 | Ranked visits, occurrence restoration, reconciliation ordering, producer lifetimes |
| [37: JPEG metadata removal](https://github.com/frathe/picfetch/pull/37) | 17 | Cancellation, memory admission, format-specific interpretation contracts |
| [45: Favorite preview work](https://github.com/frathe/picfetch/pull/45) | 8 | Cache freshness, work scope versus membership, complete worker barriers |
| [50: system HEIC](https://github.com/frathe/picfetch/pull/50) | 21 | Unavailable collection members, repeated occurrences, shutdown persistence |
| [58: Location Map](https://github.com/frathe/picfetch/pull/58) | 27 | Recurrence of command, visit, source-version and lifetime problems |

High-comment PRs were a discovery aid, not a representative defect-rate sample.
Packaging, format correctness and CI findings do not automatically imply UI
architecture problems. Relevant review reports were traced into current code;
this is an architectural assessment, not an exhaustive re-review of every PR.

Two PR 58 reports must specifically **not** be used as justification:

- [Duplicate progress totals](https://github.com/frathe/picfetch/pull/58#discussion_r4106078866)
  were rejected: `grid.restartWork` already replaces the hash engine, and the
  disposition records a regression plus a failing deliberate mutation.
- [Initial painted-tile baseline](https://github.com/frathe/picfetch/pull/58#discussion_r4111279295)
  was rejected: `movePaintedTiles` already establishes the baseline before
  mounting. The unchanged first-pan/zoom guard was negatively verified.

The latest confirmed cleanup-cancellation and auto-fit-demand findings were
[fixed](https://github.com/frathe/picfetch/pull/58#discussion_r4111279161)
[in 9b7ceaa](https://github.com/frathe/picfetch/pull/58#discussion_r4111279404).
All 27 inline threads were resolved at the final query. That says nothing about
a fresh review or CI result on the new head; neither gate is certified here.
The Copy Selection priority fix in `7a161a2` was a
[lead finding](https://github.com/frathe/picfetch/pull/58#issuecomment-5845857728),
not another Codex inline finding.

### Existing architecture worth preserving

[PR 24](https://github.com/frathe/picfetch/pull/24) completed Explorer and display
ownership. [PR 26](https://github.com/frathe/picfetch/pull/26) already centralized
source reconciliation, occurrence identity and cache-maintenance handoffs after
PR 25. These are foundations for the proposals, not unfinished extractions.

- `display.Feature` owns pixels, captures, load retries and speculation;
  `zoom` owns geometry. Selected and displayed sources remain distinct.
- Consumer-owned narrow Hosts and immutable menu/settings snapshots keep
  features independent of `viewer` and `appState`.
- `fileidentity`, `dupes`, `imaging`, `favstore` and the OS adapters already
  provide useful seams. Prefer deepening them to adding parallel infrastructure.
- [sourcechange.go](internal/ui/sourcechange.go), `reconcileSources`, explicitly
  orders search detachment, collection changes, feature retirement, Grid
  reconciliation and origin restoration. Preserve that central ordering.
- [analysiscache/operation.go](internal/ui/analysiscache/operation.go) distinguishes
  view-bound inspection from committed policy/eviction work;
  [quiescence.go](internal/ui/analysiscache/quiescence.go) joins claimed handoffs.
  [ByteCache](internal/imaging/bytecache.go) deliberately distinguishes foreground,
  speculative and warming admission. Those differences should remain explicit.

### Target ownership

| Concern | Proposed authoritative owner | What stays elsewhere |
| --- | --- | --- |
| Collection membership, occurrences, order, Favorite association | Deepened `appState`/collection module in root UI | Scanning, decoding and persistence I/O |
| Active browsing owner, foreground visit and return destination | Root browsing-session module | Feature analysis, search history, map camera and Grid interaction |
| Whether a command runs and what must yield first | Pure command policy over a captured context | Fyne event decoding and action execution |
| Applying a committed source/collection change | Explicit root reconciliation transaction | Each feature's invalidation and worker implementation |
| Favorite membership identity and owned directory access | `favstore` storage primitives | Representation formats, budgets and analysis write leases |
| Request cancellation and terminal worker tracking | Small instance-owned mechanics, where proven reusable | Feature-specific settlement and persistence policy |

Root UI remains the composition owner. A state module earns its place by making
an invariant true through its interface; moving the same booleans and callbacks
into a differently named struct would not reduce coupling.

### Registry decision

Keep [features.go](internal/ui/features.go) and [build.go](internal/ui/build.go)
explicit. Construction order, overlay order and lifecycle order are different:
Grid must precede its consumers, comparison covers an existing Grid, and cache
maintenance may need UI delivery while joining producers. One registration order
cannot express all three correctly.

A small, static **command catalogue** may eventually share command IDs, labels
and accelerator metadata after MA-028. It is not a feature registry and is
optional. Do not introduce string-keyed feature lookup, self-registration,
reflection, a global event bus, or an all-features `Controller` interface.
An unordered `SourceChanged` broadcast would lose the causal guarantees already
captured in `reconcileSources`.

<a id="ma-026"></a>

## MA-026 — Make Explorer a complete feature module

**Complete (`28d65ef`), 2026-09-14.** Explorer owns its workflow, dialogs,
cancellation and workers; root retains source preparation and navigation.
[Native CI](https://github.com/frathe/picfetch/actions/runs/34829485376) passed
all four Linux/amd64 race partitions and every platform job. Qodana and CodeQL
are clear. The [archived plan](finished_refactorings/2026-09-14-explorer-feature.md)
holds the implementation and verification evidence. This anchor remains for
existing dependency links; MA-026 is no longer open refactoring work.

<a id="ma-027"></a>

## MA-027 — Give single-image presentation ownership of its lifecycle

**Complete (`e6024dc`), 2026-09-14.** The
[archived Deep SDD plan](finished_refactorings/2026-09-14-ma-027-presentation.md)
records the accepted contract, seven slices, red/green evidence and qualification.

[display.Feature](internal/ui/display/feature.go) now owns the surface, source
identities, captures, rotation/fades and load/GIF/SVG/preload workers. Root
[load.go](internal/ui/load.go) chooses requested/retry/neighbor sources and
composes display's synchronous handoff with zoom, window policy, title and EXIF.
[vector.go](internal/ui/vector.go) only forwards layout density. Root no longer
owns mutable frames, animation pauses, timers or SVG raster workers.

The public display contract covers stale delivery, coherent handoff, same-source
reopening, saved baselines, fresh-delay capture release and all-generation
teardown. Root tests retain actual input, file actions, cache sharing and window
composition. Foreground cache admission and budgets remain; display speculation
uses a separate atomic non-evicting operation. This corrects the design's mistaken
assumption that existing AddIfFits never evicts; other callers keep its semantics.

[Native CI](https://github.com/frathe/picfetch/actions/runs/34851844070) passed
all four Linux/amd64 race partitions, validation and Windows/macOS guards.
Qodana and CodeQL have zero findings; fresh Codex code/security reviews are
clean. The unchanged golden masters pass. All seven tickets are resolved;
this anchor remains for existing dependency links.

See the [accepted specification](.scratch/ma-027/spec.md),
[verification map](.scratch/ma-027/verification.md), and
[surface-ownership ADR](docs/adr/0001-single-image-presentation-ownership.md).

<a id="ma-028"></a>

## MA-028 — Centralize shared command-admission decisions

**P1; high confidence.** The best first change because it reduces the cost of
every later mode addition without moving feature state.

**Recurring evidence:** Explorer's
[enabled menu but blocked Favorites shortcuts](https://github.com/frathe/picfetch/pull/18#discussion_r3983220054),
search's [duplicate commands leaking through menus](https://github.com/frathe/picfetch/pull/25#discussion_r4009091850),
and Location Map's [clipboard bypass](https://github.com/frathe/picfetch/pull/58#discussion_r4106674537)
and [duplicate-key bypass](https://github.com/frathe/picfetch/pull/58#discussion_r4107351569)
are the same class of coordination failure across different features.

**Current seam:** `handleKeyEvent` in [keys.go](internal/ui/keys.go),
`yieldingShortcuts.AddShortcut` in [shortcuts.go](internal/ui/shortcuts.go),
`yieldingMenuCallbacks`/`menuState` in [menu.go](internal/ui/menu.go),
[menus.State and Apply](internal/ui/menus/menus.go), and direct entries such as
`searchReference`, clipboard and Copy Selection each encode policy. Clipboard
intentionally bypasses the ordinary shortcut wrapper because it has its own
target priority. `menuState.LocationMapActive` means visible surface, while
`CohortActive` also includes a retained map visit: names conceal different facts.

**Refactor:** one pure decision module takes a command and an immutable context
captured on UI. Represent visible surface, retained visit, modal input owner,
busy operation and command target separately. Return allow, block or a named
precondition/refusal, including yielding idle Copy Selection. The executor
performs effects and rechecks admission after asynchronous preparation.
Menu enablement consumes the same policy; it must not execute yield effects.

Start as private root UI code with existing state adapters. Migrate one command
family at a time and remove its duplicated policy only after every route uses
the shared decision. MA-029 can later replace the snapshot's input source.
Keep native shortcut handling and feature-local key interpretation separate.

**Preserve:** route-specific behavior, such as Open's comparison refusal and
Help availability, is explicit policy, not forced menu/keyboard equivalence.
Keep direct-entry guards and operation-token checks. Keep Escape precedence,
copy target priority and busy-copy blocking; a single `CanExecute bool` is
insufficient to express these transitions.

**Verification/done:** table-driven policy tests plus actual menu callbacks,
registered shortcuts and direct entries demonstrate the same shared rules.
Extend `TestWindowCommandAdmissionMatrix` and the existing comparison,
Copy Selection, Explorer and search guards. A new restricted visit must not
require hand-editing independent admission predicates in all input adapters.

<a id="ma-029"></a>

## MA-029 — Give browsing visits one explicit state owner

**P1; high confidence in the need, medium confidence in the final interface.**
This is the useful state-management module suggested by the review history.

**Recurring evidence:** search
[failed to restore its Explorer origin](https://github.com/frathe/picfetch/pull/25#discussion_r4007684420);
Explorer's [menu return lost Unassigned behavior](https://github.com/frathe/picfetch/pull/18#discussion_r3986368996);
Location Map [overlapped with ranked search ownership](https://github.com/frathe/picfetch/pull/58#discussion_r4106350273)
and [opened behind the retained ranked Grid](https://github.com/frathe/picfetch/pull/58#discussion_r4107097075).
Preload order diverged from navigation in
[search](https://github.com/frathe/picfetch/pull/25#discussion_r4007684445) and
[Location Map](https://github.com/frathe/picfetch/pull/58#discussion_r4105976290).

**Current seam:** [browsing.go](internal/ui/browsing.go) already shares a
restriction snapshot and ranked order. However, `cohortIndexes` in
[explorer.go](internal/ui/explorer.go) also selects Location Map and search order;
`searchPresentation` in [visualsearch.go](internal/ui/visualsearch.go),
`locationInput` in [locationmap.go](internal/ui/locationmap.go), feature state and
[grid.Visit](internal/ui/grid/ranked.go) collectively determine foreground,
history and return behavior. `grid.Visit` mixes a presentation bookmark with
subset callbacks; ranked restoration requires a preceding `OpenRanked` call.
`explorerGridChanged` also dispatches Location Map changes.

Navigation and preloads **already share** `cohortIndexes` after the PR 58 fix.
The remaining opportunity is to make the owning visit explicit, rather than
discover it through a precedence chain. Failure recovery in
[load.go](internal/ui/load.go), `imageLoadFailed`, has its own successor policy;
it needs an explicit relationship to the visit, not a blind replacement with
ordinary Next behavior.

**Refactor:** a root browsing-session module owns the active browsing owner,
foreground visit, return destination and ordered scope. A retained Explorer
cohort, ranked search and exact occurrence cluster are explicit variants.
Distinguish an empty restricted scope from an ordinary collection. Capture one
immutable scope per action for navigation, preloads, Favorite capture and batch
targets; preserve deliberate differences in their target selection.

Transitions such as open image, return to Grid, return to origin and leave
session update this state in one place. Root executes the ordered feature calls
and publishes menu/presentation changes after the transition. Async validation
or setup must revalidate the transition before revealing its surface.

Keep analysis results/cohort data in Explorer, ranked query history in search,
camera in each map and selection/filter/scroll in Grid. The new module owns
cross-feature navigation, not copies of all those fields. During migration,
replace each old ownership flag with a projection or remove it; do not maintain
two writable versions. A single flat enum for the whole app is inadequate:
comparison may cover a Grid, and Copy Selection may own input within an image
visit whose originating feature is still alive.

**First slice:** name and centralize the existing browse-scope resolver and its
empty-scope behavior. Then migrate shared image/Grid return transitions for
Explorer and search before adapting Location Map. Use MA-030's occurrence
remapping when collection members disappear; no filesystem I/O in this module.

**Verification/done:** exercise collection -> Explorer cohort -> search -> image
-> Back/Exit; comparison over ranked results; Copy Selection during return;
sort/removal while a return is pending; and replacement of the original
collection. Preserve Grid interaction state and frozen image-order rules.
Extend `TestFindMoreLikeThisInitialRoundTrip`, progressive-visit tests, Explorer
return tests and `TestLocationMap`. The next browsing feature supplies its scope
and transitions without adding itself to unrelated features' navigation helpers.

<a id="ma-030"></a>

## MA-030 — Deepen collection identity and committed transitions

**P1; high confidence.** Strengthen the existing collection model and root
reconciliation seam; a new generic state store is unnecessary.

**Recurring evidence:** [sort generation/grouping](https://github.com/frathe/picfetch/pull/17#discussion_r3950334572),
[premature Favorite identity](https://github.com/frathe/picfetch/pull/18#discussion_r3983220066),
[Grid occurrence restoration](https://github.com/frathe/picfetch/pull/25#discussion_r4009091797),
[image occurrence restoration](https://github.com/frathe/picfetch/pull/25#discussion_r4013315670),
and [HEIC retained-order removal](https://github.com/frathe/picfetch/pull/50#discussion_r4077282457)
all depended on agreement about which collection member a value identified.
PR 25 also exposed
[restoration during a partial batch](https://github.com/frathe/picfetch/pull/25#discussion_r4010968614)
and [restoration before Grid reconciliation](https://github.com/frathe/picfetch/pull/25#discussion_r4011126519).

**Current seam:** [state.go](internal/ui/state.go) correctly publishes keys and
generation atomically. [fileidentity](internal/fileidentity/occurrence.go)
correctly separates occurrences from filesystem versions. Yet selected/display
order, unavailable-member order and Favorite association are coordinated across
`appState`, [heic.go](internal/ui/heic.go), `explorerInput.favoriteDir` and
`applyScannedCollection` in [drop.go](internal/ui/drop.go). Sort commits in
[sort.go](internal/ui/sort.go) have a separate reconciliation sequence.
`captureLocationReconciliation` builds an occurrence-survivor map in a
feature-specific adapter even though deletion remapping is a collection fact.

**Refactor in two slices:**

1. Deepen `appState` into the authoritative collection model: membership and
   displayed/source order, unavailable entries, selected occurrence, Favorite
   association and a generation-bound lookup. Expose immutable snapshots and
   commit operations for replacement, merge, reorder and batch removal. Produce
   occurrence-survivor mappings once when membership changes. Reuse
   `fileidentity`; do not replace path/ordinal bookmarks with a UUID system
   without evidence that the current contract cannot work.
2. Extend `reconcileSources` into explicit root commit phases shared by the
   relevant collection changes: capture visits/retire competing delivery,
   commit collection and invalidate affected derived state, reconcile feature
   views, then restore/admit display and publish notifications. Keep differences
   between reorder, removal, content write and policy change named. Do not make
   every change purge every cache or restart every feature.

The collection module does not know feature pointers or Fyne widgets. The
reconciler knows the concrete participants because their ordering matters.
Keep `imaging.WriteResult.Committed` authoritative even after a request becomes
stale, and retain display's existing retry chain. Preserve alias resolution on
workers in [filework.go](internal/ui/filework.go), `writtenFileSources`.
The PR 26 transaction is already useful; migrate the remaining paths into its
contract instead of layering an event dispatcher over it.

**Verification/done:** one member-removal operation provides the same remapping
to Grid, image visits and retained unavailable-file persistence. Replacement
commits files and Favorite identity together; cancellation preserves both.
Test repeated URIs around unavailable entries, sort during load, batch removal
with a retained search origin, and stale committed Save/Export/Strip callbacks.
`TestHEICUnavailableFiles`, `TestFindMoreLikeThisSourceAndSortRetirement`,
`TestSaveChangesCancellationKeepsCommittedDiskEffectsAndCurrentView` and
`TestExportCommittedAliasRefreshesCurrentPixelsAfterDelivery` are existing
integration anchors. New collection tests should not need a desktop harness.

<a id="ma-031"></a>

## MA-031 — Share Favorite membership and ownership primitives

**P2; high confidence in duplicated responsibility.** This is storage ownership
shared by several features, not a request to merge their caches.

**Evidence:** Explorer had
[pathname reads against a retained directory owner](https://github.com/frathe/picfetch/pull/18#discussion_r3987044215);
analysis maintenance needed
[membership revalidation before deletion](https://github.com/frathe/picfetch/pull/25#discussion_r4008194389),
and producers needed [ownership refresh after a save](https://github.com/frathe/picfetch/pull/25#discussion_r4010389773).
Location Map separately acquired inventory, membership, retirement and cleanup
rules, including [live-scope bounds](https://github.com/frathe/picfetch/pull/58#discussion_r4107582881).
The preview finding in [PR 45](https://github.com/frathe/picfetch/pull/45#discussion_r4047794447)
also demonstrates that a bounded work prefix is not the complete membership.

**Current seam:** `favstore.readList`/`Load`,
[OpenCohorts](internal/favstore/cohorts.go),
[similarity.loadFavoriteAnalysis](internal/similarity/cache_favorites.go), and
[locationmap.openFavoriteOwner](internal/ui/locationmap/favorites.go) separately
decode membership and implement different ownership/version checks. Some use
whole-file reads; Location Map applies a 16 MiB bound and scoped membership.
The inconsistency is observable in code; this assessment does not claim a new
exploitable defect in each reader.

**Refactor:** put bounded membership decoding, ordered entries, directory/list
identity and explicit partial-inventory results behind `favstore` operations.
Allow callers to request a live-source intersection while retaining the complete
membership identity needed for validation. Current ownership and resource
release must be part of the interface, including replacement/move behavior.
Start with two real consumers, similarity and Location Map, then assess cohorts
and previews against the same contract.

Keep analysis cross-process leases, GPS namespaces, thumbnail encoding and each
cache's budget/admission policy in their existing owners. A membership snapshot
alone is not an atomic write authorization: adapters must preserve their
publication/lease protocol. Do not introduce a process-global Favorite registry
that can become stale after another process replaces a list.

**Verification/done:** shared storage tests cover malformed/oversized membership,
directory move and same-name replacement, partial inventory errors, cancellation
and bounded enumeration. Consumer tests retain opt-out behavior, obsolete-writer
rejection, valid fresh records after invalidation, and the full-membership versus
eager-work distinction. No disk-format migration is needed for the first slice.

<a id="ma-032"></a>

## MA-032 — Consolidate proven worker-lifetime mechanics

**P2; medium confidence.** The need for consistent contracts is strong; a
universal task manager is not justified. Pilot a small extraction before any
broad migration.

**Evidence:** PR 45 found
[unjoined pool dispatchers](https://github.com/frathe/picfetch/pull/45#discussion_r4049599084)
and [unjoined cancellation callbacks](https://github.com/frathe/picfetch/pull/45#discussion_r4049782614).
HEIC needed [production shutdown joining](https://github.com/frathe/picfetch/pull/50#discussion_r4076485060)
and [persistence despite discarded UI delivery](https://github.com/frathe/picfetch/pull/50#discussion_r4079976936).
Location Map's [committed cleanup lifetime](https://github.com/frathe/picfetch/pull/58#discussion_r4111247911)
required surviving ordinary close while still responding to terminal Stop.

**Current seam:** request-token mechanics repeat in
[root](internal/ui/lifecycle.go), [display](internal/ui/display/lifecycle.go) and
[Explorer](internal/ui/explorer/lifecycle.go); root/display are especially close.
Worker admission, retired work and queues are independently composed across
features. [run.go](internal/ui/run.go) and
[harness_test.go](internal/ui/harness_test.go) enumerate different shutdown
obligations. Some differences are required, not omissions.

**Refactor:** first establish reusable instance-owned request cancellation and
revision tokens for two matching consumers, including parent-context binding
and explicit release. Extract worker tracking only where it actually hides
admission/retirement mechanics; avoid a helper that still makes every caller
reimplement completion. Name view-bound work, session work and work retained
after a committed effect. Feature code chooses the lifetime and owns durable
reconciliation.

Preserve the distinct observations: worker exited, result delivered, finite
operation settled, continuous producer stopped, and disk effect committed.
For example, display's `Settle` excludes continuous animation, search retains
a ready producer, and analysis-cache settlement drains UI while joining because
workers may await `Host.Quiesce`. A universal `WaitAll; Drain` loop can deadlock
or wait forever. Cancellation also cannot interrupt an already-blocked native
filesystem call merely because its context is canceled.

Keep stop/close calls on UI and joins off UI. Keep explicit ordered shutdown;
do not flatten it into a registry loop. Retain production-specific exceptions
such as Spiral's uninterruptible source-read handling.

**Verification/done:** deterministic held-worker/queued-callback tests prove
supersession, close/reopen, Stop admission, all retired generations, callback
completion and late-result rejection for both migrated consumers. Include an
actual production shutdown path as well as the stronger test harness. If the
pilot cannot remove caller obligations without special-case flags, retain the
local implementations and share contract tests/documentation instead.

<a id="ma-033"></a>

## MA-033 — Capture launch side-effect policy once

**P2; high confidence, small independent task.** Runtime policy is distinct from
interactive command policy and must exist before feature construction.

**Evidence:** trial isolation first missed
[manual update/apply paths](https://github.com/frathe/picfetch/pull/58#discussion_r4105976269),
then [pre-app predecessor cleanup](https://github.com/frathe/picfetch/pull/58#discussion_r4106350305).
Current guards repeat Store-managed/Explorer-trial/Location-Map-trial decisions
in [main.go](main.go), [run.go](internal/ui/run.go) and
[autoupdate.go](internal/ui/autoupdate.go). The UI guards also derive launch
policy from live feature objects (`explorer.Trial`, `locationTrial`).

**Refactor:** derive one immutable runtime policy from validated
[launch.Options](internal/launch/launch.go) and distribution mode before normal
startup side effects. Pass the relevant value to root composition and updater
admission. It decides normal-install cleanup, update check/download/apply and
storage isolation; it does not hold feature objects or perform those effects.
Reuse `Options.ApplicationID`'s existing validation and keep `main` thin.

**Verification/done:** a portable/Store/Explorer-trial/Location-Map-trial matrix
covers automatic and manual update paths, staged apply and both startup cleanup
paths. Use temporary sentinel artifacts and injected updater operations. Adding
a new isolated launch mode changes one policy decision and its tests, not a
search for scattered feature-name predicates.

## Smaller opportunities and deliberate deferrals

- **Progress and expensive effects:** PR 25's
  [search progress](https://github.com/frathe/picfetch/pull/25#discussion_r4010389781)
  and [inspection progress](https://github.com/frathe/picfetch/pull/25#discussion_r4010389787)
  findings, followed by PR 58's automatic camera/tile demand, show that producer
  updates must not dictate expensive UI/network work one-for-one. Existing fixes
  coalesce different things. Reuse a small latest-value delivery primitive only
  where semantics match; progress may be lossy, final results and committed
  effects may not. This is not justification for a global event bus.
- **Source version vocabulary:** Grid and Location Map use
  `favthumbs.EntryName` as a version token outside Favorite-preview storage.
  A neutral typed source-version value, with explicit unknown status and
  worker-only capture, would improve that dependency direction. Keep collection
  occurrence, content version, request revision and Favorite ownership distinct.
  Preserve existing cache filenames and their limitations; mtime/size is not a
  content hash. Consider this after MA-030/031, not as a new global source registry.
- **Resource and format contracts:** PR 37's
  [extra full decode](https://github.com/frathe/picfetch/pull/37#discussion_r4036338483)
  and HEIC's [probe decode](https://github.com/frathe/picfetch/pull/50#discussion_r4076749418)
  support explicit probe/decode/inspection cost contracts in imaging. JPEG color
  interpretation, redirect restrictions, tile-scene geometry and evidence-timing
  correctness stay with their owning modules. An application state rewrite
  would not have prevented those findings.
- **No blanket cache unification:** a foreground image may exceed its byte
  budget; speculative preloads must not evict; Favorite warming must not promote
  unrelated entries. Preserve these policies and existing accepted limitations.
  A single cache manager would hide meaningful differences.

## Suggested sequence and verification approach

1. Implement MA-028 against current feature observations. This gives subsequent
   refactors one place to preserve command policy.
2. Implement MA-030's collection snapshot/remapping slice and MA-029's shared
   browse-scope slice; then migrate browsing ownership one flow at a time.
3. Extend the existing source transaction to the remaining collection commits.
   Keep behavior changes separate from ownership moves, especially failure
   successor selection and empty-scope handling.
4. MA-031 and MA-033 can proceed independently once their contracts are agreed.
   Attempt MA-032 as a two-consumer pilot, not a prerequisite for all other work.

For each implementation, first capture externally observable behavior at the
module's interface and keep representative real UI route tests. Test sequences,
not merely individual flags: pending setup -> reference change -> completion;
search -> compare -> committed write -> origin return; sort/delete with repeated
sources; close -> reopen -> old callback; policy change during preparation.
Assert actual mounted content and displayed identity, not only `Visible()`.

Focused existing verification entry points, to be narrowed to the changed
scenarios during implementation:

```sh
go test -tags no_emoji,nodynamic ./internal/ui/menus ./internal/fileidentity
go test -tags no_emoji,nodynamic ./internal/ui -run '^(TestWindowCommandAdmissionMatrix|TestFindMoreLikeThisInitialRoundTrip|TestFindMoreLikeThisSourceAndSortRetirement|TestHEICUnavailableFiles)$'
go test -tags no_emoji,nodynamic ./internal/favstore ./internal/favthumbs
```

These commands are future verification aids, **not tests run by this assessment**
or complete acceptance criteria for the proposed refactors. Each accepted task
needs its own SDD/TDD plan, focused regressions, changed-file GoLand inspections
and the repository's final verification gate. No dependencies or runtime code
were changed for this report; no native behavior or performance improvement is
claimed.

Assessment method: lead-owned architecture judgment and code reading, supported
by two bounded read-only scouts for browse-order and lifecycle call paths.
Their symbol locations were checked against the code. GitHub access was
read-only. Documentation validation passed: `git diff --check`, 41 new local
links/anchors resolved, and all nine explicit backlog anchors were unique.
The production test suite was not run for this documentation-only change.

<a id="ma-023"></a>

## MA-023 — Retire the HEIC fork

**Closed by removal, 2026-09-14.** Ronin requested that HEIC support and the
decoder dependency be removed pending distribution qualification. The fork
upgrade watch and native leak-test helpers are retired. See the
[removal record](finished_refactorings/2026-09-14-remove-heic-decoder.md); reconsidering HEIC
support is separate work tracked in [todos.md](todos.md).
