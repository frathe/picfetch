# MA-031: shared Favorite membership and ownership

Status: accepted design; all twelve decisions resolved; implementation pending.
Date: 2026-09-28
Source: `/grill-with-docs MA-031`
Inspected revision: `7e52ea5b5fe95ae7d0995e88fa765ab57d549b86`.

This is the accepted design record for
[MA-031](../finished_refactorings/2026-09-29-needs-refactoring.md#ma-031). The interview is complete. The
[local specification](../.scratch/ma-031/spec.md) is published as
`ready-for-agent`, with 70 user stories, test boundaries and 22 acceptance
criteria covering both migration stages. Implementation planning remains next;
this record does not authorize implementation or commits. The local tracker is
gitignored under the repository's normal publication convention.

## Accepted decisions

The user answered "go with defaults" to Q1-Q3 and "go with defults" to Q4-Q8.
The user answered "go with defaults" to Q9-Q12 after the final round explicitly
stated that accepting those defaults also confirms the shared design. Every
branch of the interview is resolved.

| Decision | Contract |
| --- | --- |
| D1 / Q1: completion | Deliver incrementally, starting with similarity and Location Map. Before closing MA-031, assess Favorite opening/counting, cohorts and preview ownership and migrate their remaining duplicated membership/ownership checks. The first two consumers are a milestone, not completion. Thumbnail encoding, GPS records, analysis records and their budgets stay with their current owners. |
| D2 / Q2: ownership boundary | Shared `favstore` operations own validated membership, captured directory/list identity, currentness checks and resource release. Old work cannot acquire a replacement Favorite merely because its pathname matches. Analysis leases and each consumer's publication protocol remain with that consumer: a successful identity check alone does not authorize a later write atomically. |
| D3 / Q3: compatibility | Establish consistent validation and explicit resource bounds, permitting deliberate rejection of malformed or oversized lists. Preserve the existing disk format, saved order and repeated occurrences agreed in MA-030. D5-D6 settle exact limits and compatibility; saving must not produce a list the shared reader refuses. |
| D4 / Q4: retirement | Retire captured ownership when its directory moves away from the captured pathname, is removed/replaced, or its list is replaced/changed. Identical-content replacement also requires fresh ownership. Stop admitting further disk writes through the retired owner; useful in-memory work may continue under its feature's lifetime. This deliberately changes similarity's move-following behavior. |
| D5 / Q5: definition limit | All membership readers and Save share a 64 MiB encoded-list limit. Oversized saves fail without replacing the previous Favorite, and membership is never truncated. Some previously accepted lists will be rejected. Consumer work/cache budgets remain separate. |
| D6 / Q6: document compatibility | Preserve sparse/noncanonical nonnegative numeric indexes with distinct numeric positions, repeated image paths, original path spelling and offline members. Reject duplicate JSON keys, numeric aliases identifying one position, invalid indexes, non-object documents and non-string/empty/NUL-containing paths. Interpret legacy relative paths consistently against one captured working directory; membership checks do not resolve symlinks, fold case or require source existence. |
| D7 / Q7: partial inventory | Return healthy owners with explicit failures; unknown membership is never empty membership. Preserve healthy Favorite use and similarity's refusal of general-cache fallback while membership is incomplete. Automatic cleanup cannot infer stale membership from unknown data; explicit clearing remains available through safely captured owners. |
| D8 / Q8: freshness | Retain operation-bound snapshots and checks at read, publication and cleanup boundaries, plus existing explicit refresh after Favorite saves. Add no watcher, global registry or whole-list rehash before every record operation. Identity/size/modification-time checks can miss an in-place edit preserving all attributes until reopening; document that limit. |
| D9 / Q9: resource retention | Enumerate in cancellable batches of 64 and decode one definition at a time per inventory operation. Analysis/search and Location Map retain membership intersecting their complete source scope while validating full definitions and preserving every matching owner. Full membership remains available for opening, cohorts and cleanup. Reopen/revalidate directory handles for bounded work instead of pinning one per Favorite indefinitely; no fixed Favorite-count cap or silent omission. Retained memory grows with requested membership and matching owners. |
| D10 / Q10: UI storage work | Move Favorite storage reads, existence checks and saves onto feature-owned tracked workers and bring removal under the same lifecycle. Queue generation-checked results; preserve naming/confirmation flows, menu ordering, shortcuts and count-error fallback. Retain the previous complete menu until replacement is ready, recheck admission for asynchronous opens, serialize saves/removals and coalesce refreshes. |
| D11 / Q11: cancellation and commits | Cancellation observed before save publication preserves the prior list; cancelled reads never open partial membership. A completed save or Trash move remains authoritative after cancellation. Reconcile current menus/cache ownership without stale dialogs or collection replacement; later menu/preview failures cannot undo or suppress committed-save effects. Stop admission and suppress late UI delivery at shutdown, then join tracked work and release resources off UI. Active filesystem/native calls may need to return. |
| D12 / Q12: confirmation conflicts | Bind overwrite/removal confirmation to captured directory/list identity and recheck immediately before mutation. A detected change requires fresh confirmation. A newly occupied name returns through the overwrite decision. This detects conflicts without promising atomic transactions against arbitrary external edits after the final check. |

The [ownership ADR](adr/0006-shared-favorite-ownership.md) records D2's
architectural boundary. The glossary distinguishes complete Favorite membership
from its ordered saved list and from the loaded collection's Favorite association.

## Source observations at the inspected revision

These are code and test-source observations, not fresh runtime verification.

- [favstore.go](../internal/favstore/favstore.go), `readList`, reads the whole
  membership file without a byte bound. `Load` and `Count` both require indexes
  accepted by `strconv.Atoi` and nonnegative. `Load` sorts numerically and
  preserves repeated paths. The comment claiming different Load/Count index
  policies is stale. `Save` atomically replaces the list without a size bound.
- [cohorts.go](../internal/favstore/cohorts.go), `OpenCohorts`, decodes the same
  JSON map without validating its index keys, retaining exact path strings as
  a membership set. It closes its opening handle; subsequent saves reopen the
  pathname and validate the observed list identity, size and modification time.
- [cache_favorites.go](../internal/similarity/cache_favorites.go) reads all
  directory entries at once and each complete membership file without a byte
  bound. It retains cleaned, unique paths for every healthy Favorite and a
  reverse index supporting several Favorite owners per source. Unknown
  membership does not discard healthy peers; maintenance retains failed owners
  for inspection and explicit clearing.
- [cache.go](../internal/similarity/cache.go), `favoriteAnalysis.current`, checks
  list identity through its retained directory handle. That handle follows a
  moved directory. `TestFavoriteAnalysisFollowsOpenedDirectory` deliberately
  covers moves and same-name replacements; it also records that a retained
  handle can prevent moving the directory on Windows.
- [cache_store.go](../internal/similarity/cache_store.go) suppresses general-cache
  reads and writes if Favorite inventory or producer-lease admission is
  incomplete, preserving healthy Favorite records without allowing unknown
  membership to bypass Favorite preferences. Explicit Favorite-save refresh
  opens fresh ownership and preserves the producer's existing write scope.
- [Location Map favorites.go](../internal/ui/locationmap/favorites.go) reads
  directory entries in batches of 64 and bounds each complete list to 16 MiB.
  It validates all index keys, then retains only saved members intersecting the
  operation's live sources. An unrelated Favorite retains no owner handle or
  GPS cache directory. Multiple owners per source are supported.
- Location Map's `favoriteOwner.current` requires both the original pathname's
  directory identity and the retained list version to match. Moves therefore
  retire its owner, unlike similarity. GPS namespaces derive from complete
  list bytes and modification time; a scoped member set is not that identity.
  Namespace cleanup separately examines at most 1,024 entries per opening.
- [favthumbs Sync](../internal/favthumbs/sync.go) limits original decoding to a
  configured prefix of unique paths. Its sweep uses complete Favorite membership
  to preserve previews outside that prefix and for temporarily offline sources.
- [filescan.go](../internal/filescan/filescan.go) defaults to 200,000 admitted
  images per operation. The setting has no application ceiling, and merging can
  exceed any individual scan limit. Favorite saving is not capped by scan,
  analysis or preview limits; opening decodes the complete list before replay
  applies its admission limit. A common 16 MiB definition limit would therefore
  reject some lists the application can currently save through ordinary use.
- Saved URI paths preserve their original spelling, including forward-slash
  Windows paths. The installed Fyne loader resolves relative strings against
  the working directory; similarity and Location Map currently only clean their
  membership strings. A shared decoder must not silently introduce filesystem
  existence checks or discard temporarily unavailable members.
- [Favorite UI](../internal/ui/favorites/favorites.go) currently calls `List`,
  `Count`, `Load`, `Exists` and `Save` inline. A successful save sends `onSaved`
  and preview preparation before attempting menu refresh; menu failure must not
  lose those committed-save effects. [Manage](../internal/ui/favorites/manage.go)
  removes on a worker but rebuilds labels on UI. Directory-handle lifetime also
  matters for Windows removal, as the similarity move test records above.
- Favorite removal currently has only a private waitgroup, completed inside its
  `fyne.Do` callback. The feature has no Stop/Wait/Settle or request generation;
  root shutdown/harness cleanup do not join that group. Dialog-identity guards
  protect teardown but not storage-result delivery. Favorite-preview lifetimes
  are separate. Existence/confirmation and later Save/Remove also do not bind
  the mutation to the directory/list identity observed before confirmation.
- [Preview storage](../internal/favthumbs/store.go) currently uses pathname-based
  `MkdirAll`, temporary-file creation and rename. Its cancellation/source-version
  checks do not supply Favorite directory/list ownership. Migrating preview
  persistence therefore includes owner-bound I/O and sweep admission, not merely
  sharing the decoder.

## Resolved decision tree

D1-D3 settled completion, the shared boundary and permission to resolve input
inconsistencies. D4-D8 settled retirement, size limits, document compatibility,
partial inventories and freshness. D9-D12 settled scoped retention and handle
lifetimes, UI storage workers, cancellation/committed effects and confirmation
conflicts. The final acceptance also confirmed shared understanding. No design
question remains open. The published specification carries executable acceptance
criteria; Go signatures, task decomposition and final test-name mapping belong
to the subsequent implementation plan.

## Round 2 accepted details

These defaults were accepted by the user's second response:

- **Q4: retirement.** Retire captured ownership when its directory leaves the
  captured pathname, its directory is replaced, or its list is replaced/changed.
  Identical-content replacement still requires fresh ownership. Continue useful
  in-memory work where its feature permits, but stop admitting disk publication
  through the retired owner. Reacquire explicitly. This deliberately changes
  similarity's current move-following behavior; retained-handle confinement and
  consumer publication protocols still bound effects racing the final check.
- **Q5: bounds.** Use a common 64 MiB encoded membership-file limit for reading
  and saving. This is an accepted engineering choice, not a preexisting limit or a
  guarantee that every previously saved list fits. For scale, 200,000 paths of
  100 bytes already exceed 16 MiB before JSON overhead. Reject an oversized save
  without replacing its prior Favorite; never truncate membership to fit.
  Keep definition admission distinct from consumer work/cache budgets.
- **Q6: document compatibility.** Preserve sparse/noncanonical nonnegative
  numeric indexes when they identify distinct numeric positions; reject
  duplicate JSON keys, numeric aliases identifying the same position, invalid
  index keys, non-string/empty/NUL-containing paths and non-object documents.
  Preserve repeated image paths, path spelling and unavailable members. Retain
  legacy relative-path loading with one consistently captured working-directory
  interpretation, rather than giving each consumer a different source identity.
  Membership comparison must not resolve symlinks or fold path case.
- **Q7: partial inventory.** Return healthy owners with explicit per-Favorite
  failures and an incomplete result for enumeration failure/cancellation.
  Unknown membership is never empty membership. Preserve similarity's refusal
  of general-cache fallback while Favorite ownership is incomplete, healthy
  Favorite use, conservative stale cleanup and maintenance's explicit-clear
  access to safely captured unknown owners.
- **Q8: freshness.** Capture ownership for each operation and recheck at the
  consumer's existing read/publication/cleanup boundaries; use existing explicit
  Favorite-save refresh where available. Detect ordinary replacement and
  identity/size/modification-time changes, without a new watcher, global registry
  or whole-list rehash before each record operation. An in-place edit preserving
  all observed version attributes can evade currentness checks until reopening;
  a captured content digest does not itself detect later changes.

## Final-round accepted details

The user accepted these defaults and confirmed the shared design:

- **Q9: resource retention.** Enumerate Favorites in cancellable batches of 64
  and decode at most one complete definition at a time per inventory operation.
  Analysis/search and Location Map retain only membership intersecting their
  captured complete source scope, while still validating every scanned definition
  and retaining its full-list identity. A preview preparation prefix, top-k result
  or visible subset never substitutes for that scope or complete saved membership.
  Preserve every matching Favorite owner, independent of collection association.
  Count/open/cohorts/cleanup request the complete view they need. Retain ownership
  values between operations; open/revalidate directory handles for bounded record
  work instead of pinning one handle per Favorite for a producer's lifetime.
  Keep analysis leases separate and retain confinement throughout an actual I/O
  operation. Do not introduce a fixed Favorite-count cap or silently skip owners.
  The 64 MiB encoded limit is not a 64 MiB process-memory guarantee: parser
  overhead and requested outputs add memory, and retained matching ownership
  still grows with source-to-Favorite associations. Final implementation must
  measure that growth and bound simultaneously open handles independently of
  total Favorite count.
- **Q10: UI storage work.** Move Favorite enumeration/counting, existence checks,
  loading and saving to feature-owned tracked workers; bring existing removal
  delivery under the same lifecycle. Capture requested names/lists on UI and
  preserve current naming/overwrite/confirmation flows, alphabetical menu order,
  shortcut assignment and bare-name fallback for an unreadable count. Return
  immutable results through a per-instance queue with request/lifetime checks
  and recheck command admission before applying an asynchronous open. Keep the
  previous complete menu until a current replacement is ready. Read failures do
  not replace the loaded collection. Same-instance saves/removals serialize,
  while refresh requests coalesce. No dialog may reappear through stale delivery.
- **Q11: cancellation and committed effects.** Cancellation is observed between
  filesystem operations, during bounded input/output work and before publication;
  an active filesystem/native Trash call may need to return. Cancelling a read
  never opens a partial Favorite; cancellation before a save's publication leaves
  the prior saved list intact. Atomic publication or a successful Trash move is
  an authoritative disk effect even if cancellation follows. Within a live app,
  reconcile current menus and dependent cache ownership after committed changes
  without restoring obsolete dialogs, opening a superseded collection or
  bypassing current cache opt-outs. Menu failure cannot suppress committed-save
  notifications. Close/Stop do not wait on UI; tracked completion joins all work
  off UI and releases handles. Shutdown suppresses UI delivery while preserving
  committed disk effects. Successful operations are not rolled back because a
  later preview/cache/refresh step failed.
- **Q12: confirmation conflicts.** Capture the target directory/list identity
  used for an overwrite or removal confirmation, then revalidate it immediately
  before mutation. A detected replacement/change fails that operation and
  requires a fresh confirmation for the current Favorite; do not automatically
  retry against the replacement. For a new Favorite, detect a newly occupied name
  before publishing and return through the normal overwrite decision. This is a
  detected-conflict policy, not a promise of atomic compare-and-replace against
  arbitrary filesystem edits after the final check. Preserve the D2 limitation;
  do not add a cross-process Favorite transaction service in this refactor.

## Scope and limits

Completion requires the initial similarity/Location Map migration and the
remaining shared membership/ownership work in Favorite opening/counting, cohorts
and previews, including Favorite UI worker lifetimes. The first two consumers
alone do not complete MA-031. Retain the existing Favorite and cache formats;
no cache merger, global Favorite registry, filesystem watcher or cross-process
Favorite transaction service is part of this design. MA-030's collection
association and saved occurrence semantics remain authoritative.

The design deliberately rejects some previously accepted definitions, including
ones larger than 64 MiB. That encoded limit does not cap total process memory.
Identity/size/time checks do not detect every external edit, and a currentness
check does not make a later mutation atomic against another process. Active
filesystem/native operations may outlive cancellation until they return.
These limits must remain explicit in the specification and verification record.

## Required verification for subsequent implementation

- Shared storage tests: numeric order and repeated paths; every D6 rejection;
  exact size boundary and oversized-save preservation; mutation while opening;
  removal/move/same-name replacement; partial inventory and cancellation;
  explicit close/handle accounting and large unrelated inventories.
- Consumer tests: all matching owners and opt-out behavior; obsolete publication
  refusal and useful fresh records after invalidation; incomplete ownership
  blocking general fallback; complete membership versus eager-work prefixes;
  maintenance's conservative stale cleanup and explicit clear.
- UI/lifetime tests: captured save list, asynchronous open admission, superseded
  read/menu delivery, committed-save effects after cancellation or refresh failure,
  confirmation conflicts, worker shutdown and drainable delivery. Native Windows
  tests must verify that idle retained ownership does not block ordinary removal;
  an operation's active OS call is not promised to be interruptible.
- Retain existing cache formats, analysis leases, GPS namespace rules, source
  version checks and source-local suppressions. Complete GoLand inspections and
  repository verification are implementation gates, not claims of this interview.

## Session evidence

- Two read-only scouts traced similarity and Location Map independently; the
  lead inspected the relevant source before forming recommendations. Bounded
  follow-ups checked collection limits, saved-path compatibility and Favorite UI
  lifetime seams.
- No production code changed and no runtime tests were run for this interview.
- At interview completion, documentation checks found no whitespace errors or
  stale interview-status wording; all local links in this record and the ADR
  resolved.
- This record, the ADR, glossary and backlog pointers are the documentation
  deliverables for the accepted rounds. No commit was requested.
- `/to-spec MA-031` published the local specification with all twelve accepted
  decisions. Its acceptance commands are future implementation gates, not
  evidence of runtime qualification during documentation work.
