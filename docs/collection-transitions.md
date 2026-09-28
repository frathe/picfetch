# MA-030: collection identity and committed transitions

Status: both slices implemented; all nine tickets complete and CI-qualified.
PR 69 records latest-head code/security reviews, dispositions and checks.
Date: 2026-09-28
Source: `/grill-with-docs MA-030`
Inspected revision: `eb2ea0b2b3072f6baada09d7da81b68f577f863d`.

Strengthen [MA-030](../needs_refactoring.md#ma-030)'s collection model and
root reconciliation so callers no longer coordinate collection facts and
feature update ordering independently. Both proposed slices are in scope.
This document is the accepted design record. The
[local specification](../.scratch/ma-030/spec.md) is published as
`ready-for-agent`, with 56 user stories, test seams and 18 command-backed
acceptance criteria. The approved [nine implementation tickets](../.scratch/ma-030/issues/README.md)
are published with blockers and per-ticket model assignments. The tracker is
gitignored by repository convention. The
[archived Deep SDD record](../finished_refactorings/2026-09-28-ma-030-collection-transitions.md)
records all nine completed tickets, the full AC1-18 mapping, local inspections,
and full CI/SARIF qualification. [PR 69](https://github.com/frathe/picfetch/pull/69)
tracks the final latest-head review loop.

## Accepted decisions

The user answered "Q2 we should do both slices. use defaults" to Q1-Q3.
The user then answered "go with defaults" to Q4-Q8.
The user answered "use defaults" to Q9-Q11. That final round explicitly
offered acceptance as settlement of the remaining choices and confirmation of
the shared design. All branches of the interview are resolved.

| Decision | Contract |
| --- | --- |
| D1 / Q1: behavior | Preserve intended interactions, including MA-029's accepted restoration and failed-load rules. Present newly discovered inconsistencies as concrete scenarios before changing their behavior. |
| D2 / Q2: completion | Deliver both slices incrementally: authoritative collection model, then migration of relevant collection changes through explicit root reconciliation phases. The first slice is a milestone, not completion. Favorite storage redesign remains MA-031. |
| D3 / Q3: ownership | The collection model owns membership, source/display order, unavailable entries, the chosen occurrence, Favorite association and survivor mappings. Root UI owns ordered feature updates. Browsing visits remain with MA-029, Grid interaction remains with Grid, filesystem work remains with existing owners, and display retains pixels and load retries. |
| D4 / Q4: identity | Retain path-plus-ordinal bookmarks, collection bindings and explicit survivor mappings. Removing an earlier occurrence remaps later survivors. Do not introduce UUIDs or identity surviving application restarts. |
| D5 / Q5: deletion | A completed source deletion removes every matching collection occurrence, including retained unavailable occurrences. Decoder unavailability retains the affected occurrence for persistence. Removing unavailable matches corrects the current visible-only Trash reconciliation. |
| D6 / Q6: Favorite association | Association selects the candidate Favorite for Explorer cohort persistence, subject to the existing analyzed-source containment check. Successful opening/merging of Favorite B chooses B; ordinary open/merge clears association; saving a Favorite does not rebind it. Association does not assert exact collection equality or per-file cache ownership. |
| D7 / Q7: no browsable members | A completed Favorite open with only unavailable occurrences commits those entries and its association together. Losing the last browsable image to decoder unavailability retains the association. Keep existing empty/error presentation, HEIC guidance and explicit reopening for readmission. Cancellation preserves the previous committed collection. This corrects current association loss. |
| D8 / Q8: partial batch success | Publish successful removals delivered together as one collection change with one survivor mapping; reconcile and notify against the final state. Failed targets remain. Completed disk effects remain authoritative. |
| D9 / Q9: unavailable-only merge | Retained unavailable occurrences make a collection nonempty for Merge. Newly admitted unavailable entries count as additions and follow D6's association rules. A merge admitting nothing preserves committed membership and association. |
| D10 / Q10: saved-list replay | Favorite/session reopening replays recorded membership, including repetitions, without adding neighboring files. Rebuild fresh occurrence bindings, apply the chosen sort and existing availability rules, and count each admitted occurrence toward its applicable resource bound. Ordinary folder/drop discovery keeps deduplication; persistent identities are not introduced. |
| D11 / Q11: sort and navigation | At ordinary sort commit, preserve the latest chosen occurrence, including an image still loading. MA-029 origin restoration retains its explicit priority. Obsolete loads cannot publish after the handoff. |

The [ownership ADR](adr/0005-collection-transition-ownership.md) records D3's
tradeoff. The specification carries the executable acceptance map. Exact Go
types, migration tasks and finalized test selections belong to the subsequent
implementation plan; they must preserve the contracts below.

## Current source observations

These are code and test-source observations, not fresh runtime verification.

- MA-029 already moved occurrence-survivor mapping into
  [sourcechange.go](../internal/ui/sourcechange.go),
  `captureBrowsingReconciliation`, and shares it with retained visits and the
  detached search origin. [sort.go](../internal/ui/sort.go), `SetSortMode`,
  already uses the same capture/finish helpers. The original MA-030 proposal's
  feature-specific location of this mapping is historical.
- [state.go](../internal/ui/state.go) holds displayed order, source order and
  retained unavailable order. Its published snapshot pairs displayed URI keys
  with a generation; unavailable membership, chosen occurrence and Favorite
  association are not all part of that publication. Removal publishes once
  per removed occurrence and calls an eviction hook after each publication.
- [fileidentity/occurrence.go](../internal/fileidentity/occurrence.go) identifies
  an occurrence by path and ordinal within a captured list. It deliberately
  does not identify filesystem content versions. State's retained-order
  matching uses URI-string ordinals; these identity domains need explicit
  adapters, not accidental interchangeability.
- [heic.go](../internal/ui/heic.go), `persistedFiles`, reconstructs unavailable
  entries around surviving occurrences. Session saving uses source order;
  ordinary Favorite capture uses display order. Ranked Favorite capture in
  [visualsearch.go](../internal/ui/visualsearch.go) instead captures the ranked
  browsing scope without inserting unavailable members.
- [drop.go](../internal/ui/drop.go), `applyScannedCollection`, commits Favorite
  association through `explorerInput.favoriteDir` alongside successful sorted
  scan results. Existing scan/sort cancellation tests preserve the prior list
  and association. D6-D7 settle the ordinary association and all-unavailable
  Favorite policies; D9 extends them to unavailable-only merges.
- [viewer.go](../internal/ui/viewer.go), `ReconcileDeletedFiles`, reconciles
  completed Trash moves against all matching visible occurrences. Its current
  enumeration does not include retained unavailable occurrences.

## Resolved decision tree

The first round settled behavior, completion and ownership (D1-D3). Those
decisions enabled identity, deletion, Favorite association, unavailable-only
Favorite retention and batch publication (D4-D8). The final round settled
unavailable-only merges, saved-list repetition and navigation during sorting
(D9-D11). Existing reconciliation, asynchronous-delivery and verification
constraints are carried forward below.

## Round 2: accepted details

- **Q4: occurrence identity lifetime.** Retain path-plus-ordinal
  bookmarks with a collection binding and explicit survivor mapping on changes.
  Removing the first A in `[A, B, A]` remaps the surviving second A to its new
  ordinal. Do not introduce UUIDs or identity across restart in MA-030.
- **Q5: committed deletion versus temporary unavailability.** Completed Trash
  moves remove all matching collection occurrences, including
  retained unavailable ones. A decoder-unavailable result instead retains the
  affected occurrence for persistence; other failed-load behavior stays as
  accepted in MA-029. Removing retained unavailable matches after Trash is an
  accepted correction to the current visible-only enumeration; it is not an
  existing verified behavior.
- **Q6: Favorite association meaning.** Treat the association as
  the candidate Favorite for Explorer cohort restoration/persistence, subject
  to the existing analyzed-source membership check. Preserve current successful
  nonempty-open behavior: opening/merging Favorite B chooses B; an ordinary
  open/merge clears association; saving a whole list or subset does not rebind
  it. It is not an assertion that the collection exactly equals the Favorite,
  nor ownership of all per-file caches.
- **Q7: a collection with no browsable members.** A completed Favorite open
  containing only temporarily unavailable entries still commits
  those entries and its Favorite association together. Likewise, losing the
  last browsable member to decoder unavailability retains that
  association. Keep existing empty/error and HEIC guidance, and require the
  existing reopen workflow to readmit members after support becomes available.
  Cancellation preserves the previous committed collection and association.
  This intentionally corrects association loss in the current empty paths;
  an input containing no retained or admitted members is a separate case.
- **Q8: batch publication.** One collection publication and one
  survivor mapping for all successful removals delivered together, followed by
  feature reconciliation and notifications against the final state. Partial
  OS success removes only successful targets; a committed disk effect is not
  rolled back because another target failed.

Favorite semantics were traced through
[Explorer workflow](../internal/ui/explorer/workflow.go), which checks
`CohortStore.Contains` over its analyzed sources before restoring saved cohorts.
The check tests path containment, not order or duplicate counts. Favorite
saving in [favorites.go](../internal/ui/favorites/favorites.go) notifies Location
Map and visual search through root's `favoriteSaved`; it does not change the
Explorer association. The zero-browsable scan and last-load-failure paths in
[drop.go](../internal/ui/drop.go) and [load.go](../internal/ui/load.go) currently
clear association through `ShowEmptyStateError` before restoring unavailable
entries. These are source observations; dedicated runtime tests were not run.

## Round 3: accepted details

- **Q9: merge with unavailable membership.** Treat a collection
  containing only unavailable occurrences as nonempty for Merge. Add to its
  retained membership and suppress replacement-style sibling expansion.
  Admitting only new unavailable occurrences is still a collection commit and
  follows D6: an incoming Favorite supplies the candidate association, while
  ordinary input clears it. If no browsable or retained occurrences are
  admitted, keep the previous collection and association. This corrects the
  current `len(state.files) > 0` merge gate in `handleCollectionDrop` and its
  inconsistent unavailable-only result handling.
- **Q10: saved-list occurrence round trips.** Preserve repeated
  recorded entries on Favorite/session reopening, rebuilding fresh occurrence
  bindings within the new collection. Retain recorded sequence as source
  order and apply the chosen sort normally. Keep ordinary directory/drop
  discovery deduplication, source admission, cancellation and configured
  bounds; each admitted saved occurrence counts toward its applicable bound.
  Keep the existing Favorite index-to-path payload and ordered session URI
  payload, with no persistent UUIDs or stored visit identity. This corrects
  a current loss: both reopening
  routes use `ImagesWithAdmission`, whose per-scan `seenFiles` collapses repeats.
  Session restore currently also shares ordinary single-file sibling expansion;
  replaying an explicit saved list uses its recorded membership, as
  Favorite open already does.
- **Q11: navigation during sorting.** Preserve the latest chosen
  occurrence at ordinary sort commit, including a requested image whose pixels
  are still loading. A slower sort must not navigate back to the occurrence
  captured when sorting started. Established MA-029 origin restoration and
  exhausted-scope rules retain their explicit priority; superseded display
  work remains unable to publish. This is a deliberate correction to
  `SetSortMode`'s start-time capture when the user navigates during its worker.

## Intended behavior corrections and limits

D5, D7 and D9-D11 deliberately change current behavior: remove unavailable
occurrences of completed Trash targets; retain Favorite association when all
members are unavailable; merge into unavailable-only collections; replay saved
occurrences without deduplication or sibling expansion; and retain newer
navigation across a slow sort. These are explicit exceptions to D1's general
compatibility rule, not accidental consequences of moving code.

Unavailable entries are still not automatically readmitted after a support
check. Reopening applies current availability, failed-load and resource rules,
so saved-list replay does not guarantee every recorded entry becomes browsable.
An occurrence binding remains local to a collection lifetime even when a saved
list records the same source more than once. Favorite association remains a
candidate for cohort persistence; it is not persisted as session identity and
does not replace Favorite store ownership checks.

This work adds no global store, event dispatcher, UUID system, persisted visit
history, new automatic analysis restart, generic worker-lifetime abstraction,
Favorite storage redesign or alternate decoder/load pipeline. Source reads,
alias resolution and worker ownership remain with existing modules. Both
collection-model and root-reconciliation migrations must finish before MA-030
is complete; keeping duplicate writable authorities is not completion.

## Constraints carried forward

These follow the accepted scope, existing repository conventions and MA-029;
they are not requests to reopen already accepted behavior.

- Keep explicit root phases: capture retained visits and retire competing
  delivery; publish the complete collection change and invalidate affected
  derived facts; reconcile feature bindings; restore/admit display through its
  existing path; publish coherent notifications. The model contains values,
  not feature callbacks. Consumers use one committed result and its survivor
  mapping instead of reconstructing removal mappings themselves.
- Immutable snapshots bind their order and lookup to the same collection
  generation. Navigation does not change collection generation. Membership,
  order, unavailable membership and association cannot be read as accidental
  combinations from separate commits. Selection capture does not force full
  collection rebuilding or cancel unrelated scan/sort work on every image.
- Reorder, removal, content write and policy change retain named, distinct
  effects. Keep existing cache/version invalidation and generation admission;
  do not turn every transition into a full purge or analysis restart. A
  collection commit does not wait on the UI thread for image decoding,
  thumbnails, maps or other worker completion.
- `imaging.WriteResult.Committed` remains authoritative after request
  cancellation or navigation. Resolve current source aliases on tracked
  workers; re-evaluate a changed collection before delivering reconciliation.
  Update affected current facts/pixels without reviving an obsolete request,
  unrelated image, retired browsing visit or stale toast. Terminal shutdown
  retains existing stop/settlement contracts. Export to an unrelated path
  does not insert it into the collection.
- Preserve Trash's captured URI-target semantics. Moving a symlink removes
  that target's occurrences, not distinct paths that still reference its
  surviving destination. Worker-side alias resolution for content writes
  remains a different operation from identifying completed Trash targets.
- Preserve the distinction between the requested image and displayed pixels,
  and keep display's single retry chain. MA-029 origin restoration, restricted
  scope exhaustion, duplicate-inspection exceptions and failed-load recovery
  remain authoritative. No new automatic command replay or analysis restart.
- Literal empty input remains a no-op. A completed nonempty replacement scan
  with no admitted or retained entries uses the existing empty/error behavior;
  a corresponding merge keeps the collection. This does not promise to retain
  browsing surfaces already retired by open admission. Explicit Close Files
  clears all membership and association. Cancellation preserves committed
  membership and association without promising restoration of every prior
  feature surface.
- Preserve saved-list capture distinctions: session captures source order;
  ordinary Favorite capture uses display order with unavailable entries; a
  ranked/selected Favorite captures its established explicit result scope.
  Keep the existing unavailable-gap placement and admission/resource limits.
  Persisting a session does not acquire a saved Favorite association.

## Verification required for implementation

New collection tests must exercise the production interface without a desktop
harness: immutable snapshots, generation/lookup coherence, replacement/merge,
stable repeated occurrences, retained unavailable order, association,
cancellation and one-publication batch removal. Root integration tests must
cover the accepted behavior corrections, retained search/map/Grid origins,
partial Trash success, sort during held loads, and stale committed file work.

Existing source anchors include `TestHEICUnavailableFiles`,
`TestFindMoreLikeThisSourceAndSortRetirement`,
`TestGenerationTracksFileSetIdentityNotNavigation`,
`TestFileSnapshot_KeysAndGenerationMoveTogether`,
`TestSaveChangesCancellationKeepsCommittedDiskEffectsAndCurrentView`,
`TestExportCommittedAliasRefreshesCurrentPixelsAfterDelivery`,
`TestFileMutationInvalidationRechecksLoadedSetBeforeDelivery`,
`TestFileMutationReconciliationSurvivesUnrelatedCommit` and
`TestMetadataRemovalRefreshesCurrentAliasWithoutLosingRotation`.
Preserve `TestPerformDelete_MovesSymlinkWithoutFollowingIt` in
[deletion_test.go](../internal/ui/deletion/deletion_test.go) and the existing
serialization contracts covered by `TestSaveLoadRoundTripPreservesNumericOrder`
in [favstore_test.go](../internal/favstore/favstore_test.go) and
`TestSaveSession_RoundTrip` in
[session_test.go](../internal/session/session_test.go).
The scout found no dedicated root test holding an image load across sort
completion or covering merge into an unavailable-only collection; the
implementation must add deliberate synchronization for those scenarios.

The local specification binds every acceptance criterion to a command and
retains the repository's required focused TDD, test/shard/exclusion upkeep,
changed-code GoLand inspection and `make verify` gates. Proposed tests must
actually exist and run before a command can establish acceptance. Documentation
whitespace and local links are checked separately; implementation gates have
not run and are not claimed to pass.

## Interview work

One read-only scout traced collection identity, retained order and existing
tests, followed by bounded Favorite-consumer and completion-edge-case traces.
The lead owns decisions, documentation and assessment. No production code
changed and no runtime tests were run for this interview.

Documentation verification: tracked diff and both new documents are free of
trailing whitespace; all 18 local links in the new documents and seven added
local links in existing documents resolve. All nine listed root regression
names resolve to current declarations. A broader link scan encountered the
pre-existing `.scratch/ma-027/spec.md` reference in `needs_refactoring.md`, whose
gitignored target is absent from this checkout; that unrelated link is unchanged.
