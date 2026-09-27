# MA-029: explicit browsing-visit ownership

Status: accepted design; implementation in progress (tickets 01-04 complete).
Date: 2026-09-27
Source: `/grill-with docs MA-029`
Inspected revision: `e7d04561dd7f1565480ed8a46adf76cc327f0e73`.
Branch: `feature/ma-029-browsing-visits`.

Give [MA-029](../needs_refactoring.md#ma-029) one authoritative owner for the
active browsing visit, its scope and return transitions. This replaces the
need for unrelated navigation callers to infer ownership from several features.
The [ownership ADR](adr/0004-browsing-visit-ownership.md) records the trade-off.

The [local specification](../.scratch/ma-029/spec.md), marked `ready-for-agent`,
captures user stories, testing seams and the executable acceptance map. It is
published to the repository's gitignored Markdown tracker; this design remains
the tracked record of accepted decisions.

All ten interview decisions are accepted. The authorized implementation follows
the [Deep SDD plan](../plans/2026-09-27-ma-029-browsing-visits.md), with finalized
scope contracts, per-ticket TDD/inspection evidence and incremental commits.
The [nine local tickets](../.scratch/ma-029/issues/README.md) track remaining
migrations; scope centralization alone is not completion of MA-029.

## Accepted decisions

The user answered "go with defaults" to Q1-Q3 and then Q4-Q7 on 2026-09-27.
After discussing the UX difference between skipping a failed search result and
restoring the original visit, the user answered "use defaults" to Q8-Q10.
The final round explicitly offered acceptance as settlement of the remaining
design choices; all branches of the interview are now resolved.

| Decision | Contract |
| --- | --- |
| D1 / Q1: behavior | Preserve intended interactions, including deliberate differences between Escape, G, Back and Exit. Present any suspected inconsistency as a concrete scenario before changing its behavior. |
| D2 / Q2: completion | Migrate Explorer, ranked search and Location Map incrementally. Centralizing scope resolution is the first milestone, not completion. Remove each migrated ownership flag or derive it from the new owner; do not keep two writable authorities. |
| D3 / Q3: authority | A module private to root UI owns the active browsing visit, ordered scope, return destination and transitions. Root executes the resulting feature calls. Grid retains selection/filter/scroll, search retains query history, and maps retain cameras. MA-028 retains command admission. |
| D4 / Q4: collection seam | Deliver MA-029 independently using existing occurrence identities, collection generations and reconciliation ordering. Include necessary visit remapping; leave the broader collection-model redesign to MA-030. |
| D5 / Q5: empty scope | Never implicitly widen an empty Explorer, ranked search or Location Map visit to the collection. Filtered Grid can remain empty with its filter and return controls. Exhaustion by removal has an explicit return or empty-state transition; incomplete discovery is distinct from completed exhaustion. D10 retains the older duplicate-inspection exception. |
| D6 / Q6: scope lifetime | Preserve frozen search image order and captured Explorer cohort/Location Map cluster membership. Ranked Grid can show newer results on return; direct Location Map image visits can include new discoveries. Each action captures a consistent scope while retaining its own target-selection rules. |
| D7 / Q7: delayed delivery | Reject obsolete visit/collection returns and recheck admission before revealing a surface. Preserve relevant deferred search presentation and busy-copy refusal without automatic command replay. Confirmed source changes reconcile even if a visible return is blocked. |
| D8 / Q8: missing targets | Preserve surviving exact occurrence bookmarks and drop missing Grid selections. Image restoration prefers the original occurrence, then the same source in the restored scope, then its first eligible image. Exhausted Explorer and Location Map visits return to their respective parent surfaces; an empty collection uses the existing empty-viewer/error presentation. Do not restart analysis automatically. |
| D9 / Q9: failed loads | Preserve recovery as a separate decision after source reconciliation. An explicitly restored image origin takes priority; otherwise use the existing successor rule within the resulting scope. Keep one display retry chain and the unavailable-HEIC stop/guidance path. |
| D10 / Q10: compatibility | Preserve duplicate grouping/inspection, including its live membership, collection-based Home/End and singleton/empty navigation fallback, through the baseline adapter. Keep Picture-frame timing/shuffle with its feature and shared navigation seam. |

The new module must reduce the ordering rules callers need to remember.
A bag of fields with public setters would leave those rules spread across the
existing callers. Feature analysis, storage and worker lifetimes stay with their
current owners. Exact Go types and migration tasks belong to the later plan.

## Ownership and transition contract

| Concern | Owner after migration |
| --- | --- |
| Active browsing visit, foreground visit, ordered scope and return destination | Private root-UI browsing module |
| Whether an action is admitted, refused or must yield Copy Selection | Existing MA-028 command policy over current facts |
| Feature-call ordering, source reconciliation, display handoff and menu publication | Root UI composition |
| Selection, filename filtering, highlight, scroll and their captured bookmarks | Grid |
| Analysis/cohort data and map camera | Explorer |
| Ranked queries, result production and query history | Visual search |
| GPS facts, geographic clustering, validation and map camera | Location Map |
| Collection identities/generations, storage and committed file effects | Existing collection and file-work owners; broader redesign remains MA-030 |
| Decoding, retries, pixels, animation and preloading workers | Display and imaging through their existing contracts |

Store the return destination and scope explicitly. Keep a retained origin, the
foreground visit and an interaction covering it distinguishable: comparison
can cover a ranked Grid, and Copy Selection can own input inside an image visit.
Neither feature visibility nor a single application-wide mode enum can replace
those distinctions. Search Back remains query-history navigation; image return
and Exit search remain separate transitions.

Grid bookmarks preserve interaction state, not authority to revive a retired
visit. The root browsing owner decides whether and where restoration is valid;
root supplies current bindings and performs the ordered Grid/feature calls.
The plan must remove the existing need for callers to independently remember
which subset/ranked setup must precede restoration. Feature-local state stays
with its owner; migration removes old cross-feature ownership flags or makes
them read-only projections of the new authority.

Capture an immutable scope for each action or preload pair. Freezing search
image order and cluster/cohort membership does not freeze every feature's
collection ordering, source validity or interaction state. Navigation, Favorite
capture and batch actions retain their different target-selection rules.
The baseline adapter preserves ordinary and duplicate browsing semantics.

Feature-owned setup, validation and workers keep their existing cancellation
and completion contracts. A delayed return must still belong to the current
visit and collection when delivered, and satisfy current command admission.
Root applies committed source effects even when that visible return is refused.
Present coherent menus and surfaces after the ordered transition; do not expose
half-restored state through intermediate callbacks.

## Explicit behavior decisions and limits

D8 makes last-member exhaustion explicit. If an Explorer cohort disappears
while other collection images remain, return to Explorer with its existing
source-changed/reanalysis guidance instead of allowing unrelated collection
images to become the implicit browsing scope. Location Map exhaustion returns
to its map; completed empty scopes remain distinct from discovery in progress.
An empty filename filter alone does not exit its Grid visit.

D9 deliberately retains search retirement on a source-load failure. For
example, search started from image A can return to A when result B fails to
decode. It does not simply advance to result C while keeping search active.
For a Grid origin, restore that Grid and retain the existing background-image
successor behavior. A future change to keep searching after source removal is
outside this design. Restoration and display retry must not start competing
loads.

D10 deliberately leaves duplicate inspection's older navigation behavior in
place. Its singleton/empty fallback and collection-based Home/End are explicit
compatibility rules, not grounds to weaken D5 for the migrated visit kinds.

This design adds no persisted visit history, new commands, feature registry,
global mutable store, generic event bus, automatic command replay or analysis
restart. MA-030 collection redesign and feature-owned cameras, analysis data,
workers and storage are outside the migration. Source I/O does not move into
the browsing module. Any newly discovered user-visible discrepancy follows D1.

## Established facts

These are source observations, not runtime verification.

- [explorer.go](../internal/ui/explorer.go), `cohortIndexes`, selects Location
  Map, search or Explorer order through a precedence chain. Search order has
  a separate active flag; several navigation helpers distinguish restriction
  only by a nonempty index list in
  [visibility.go](../internal/ui/visibility.go). An empty list alone therefore
  cannot express the proposed scope contract.
- [browsing.go](../internal/ui/browsing.go), `captureSearchOrder`, uses current
  Grid results while Grid is visible and a captured image order otherwise.
  `TestFindMoreLikeThisProgressiveForegroundIdentity` and
  `TestFindMoreLikeThisInitialRoundTrip` retain assertions for frozen image
  navigation and return to newer ranked results.
- [locationmap.go](../internal/ui/locationmap.go), `locationIndexes`, preserves
  captured exact cluster occurrences but refreshes a direct map-image visit
  from current map points, ordered by the collection. The fallback during an
  incomplete scan uses the captured order. `TestLocationMap` has guards for
  frozen cluster membership and discovery continuing during a direct image
  visit. The cited direct-visit test checks discovery and return, not whether
  image navigation expands to the new members; that is a code observation.
- [visualsearch.go](../internal/ui/visualsearch.go), `searchKey`, returns an
  image to its search Grid without popping search history. Search's
  [Feature.Back and Feature.Exit](../internal/ui/visualsearch/feature.go)
  respectively restore history and retire the session to its original visit.
  Comparison covers a Grid; Copy Selection can own input within an image
  visit. Input ownership and a retained browsing visit are different facts.
- [grid/ranked.go](../internal/ui/grid/ranked.go), `Visit`, currently combines
  selection/filter/highlight/scroll bookmarks, scope information and subset
  callbacks. Restoring ranked interaction requires `OpenRanked` to install
  live callbacks first. These responsibilities need an explicit relationship
  to D3's transition owner; the interview has not selected Go interfaces.
- [sourcechange.go](../internal/ui/sourcechange.go), `reconcileSources`, detaches
  search before callbacks, applies collection changes, retires analysis,
  reconciles Grid and Location Map, then restores the search origin. A failed
  image load receives the restored index through display's existing retry
  chain, without starting a competing load.
- [state.go](../internal/ui/state.go) already publishes collection keys and
  generation together. Existing `fileidentity.Occurrence`,
  `grid.Visit.RemapOccurrences` and Location Map's survivor mapping provide
  identity mechanisms before MA-030. [sort.go](../internal/ui/sort.go) exits
  search before sorting and rejects stale sort requests separately from
  publishing a new collection generation.
- [locationmap.go](../internal/ui/locationmap.go), `returnLocationMap`, keeps
  the current visit visible during validation and rechecks admission before
  revealing the map. Confirmed source changes still reconcile when a busy
  region copy prevents the visible return. `TestLocationMap` retains guards
  for the validation barrier and delivery after close/collection replacement.

## Round two: resolved decisions

The user accepted all four recommendations below.

### Q4: relationship to MA-030

Should MA-029 proceed using existing collection identities and reconciliation,
or include MA-030's collection-model redesign as a prerequisite?

Recommendation: proceed independently using the current occurrence/generation
and reconciliation mechanisms. Include narrowly necessary visit-remapping
changes, but keep collection storage, Favorite association and the broader
committed-transition redesign in MA-030. Preserve reconciliation ordering.

### Q5: empty restricted scopes

If a visit has no eligible images, may navigation or preload silently fall back
to the full collection?

Recommendation: never widen a restricted visit implicitly. A filtered Grid may
remain empty with its filter and return controls available. Exhaustion by
removal follows an explicit owner-specific return or empty-state transition;
the last Location Map cluster member already returns to its map. Keep an
incomplete scope distinct from a completed scope with no surviving members.
Any new choice of return destination remains subject to D1.

### Q6: frozen and live scopes

Should all visits freeze the same way, or preserve their existing differences?

Recommendation: preserve the differences. Search image order and captured
cohort/cluster membership stay frozen; a return to ranked Grid can use newer
results. A direct Location Map image visit may incorporate newly mapped
members. Frozen membership does not imply every feature ignores collection
sorting or removals. Capture one immutable scope per action and preserve each
consumer's deliberate target rules, including Favorite and batch selection.

### Q7: delayed returns

What should happen when validation or setup completes after the originating
visit has ended, its collection has been replaced, or another interaction
currently prevents the return?

Recommendation: reject obsolete visit/collection deliveries and recheck current
command admission before revealing a surface. Preserve existing deferred
search presentation while it remains relevant and existing busy-copy refusal;
do not introduce an automatic replay queue. Completed source changes still
reconcile independently of permission to change the visible surface.

## Additional source observations

- `restoreSearchOrigin` currently resolves an image origin by exact occurrence,
  then first occurrence of the same path, then collection index zero. Ordinary
  Grid bookmarks restore surviving occurrence identities. Selecting a fallback
  image and retaining an explicit Grid selection are different contracts.
- Explorer's `HasCohort` currently tests nonempty membership; removing its final
  source erases that active-cohort observation. Source reconciliation retires
  the analysis and gives the Explorer surface its existing source-changed
  guidance. This establishes a possible return surface, not proof that today's
  last-member flow already returns there. D8 accepts that explicit transition.
- `imageLoadFailed` gives an explicitly restored image origin priority over its
  post-removal successor choice, and passes the chosen source back to display's
  current retry chain. Unavailable HEIC support stops that attempt and shows
  guidance rather than advancing. `TestFindMoreLikeThisSourceAndSortRetirement`
  has separate image/Grid-origin load-failure assertions, including a guard
  against a competing display request. Explorer's missing-cohort-member case
  checks that surviving cohort membership still bounds recovery.
- Duplicate Grid browsing and image inspection have separate existing owners:
  Grid retains its browse source/group; `dupes.Model` retains the inspected
  source key and resolves its current group. `Model.NextVisible` uses the
  inspection ring only with at least two members; otherwise it uses ordinary
  collection visibility. Home/End also use ordinary visibility. Root's
  `reopenVariantGrid` clears inspection and re-enters duplicate Grid browsing.
  See [actionmenu.go](../internal/ui/actionmenu.go),
  [grid/dupes.go](../internal/ui/grid/dupes.go) and
  [dupes/visible.go](../internal/dupes/visible.go). Q10 makes the migration's
  relationship to those older rules explicit.

## Round three: resolved decisions

The user accepted all three recommendations after the failed-load UX
clarification. The recommendations below are the accepted contract.

### Q8: missing return targets and exhausted visits

When the saved image or the last restricted member disappears, what should
replace the visit?

Recommendation: restore surviving exact occurrence bookmarks. Missing Grid
selection members are dropped rather than reassigned to another occurrence.
For an image origin, prefer the original surviving occurrence, then the same
source within the restored scope, then its first eligible image. An ordinary
image origin may fall back to the first collection image. An exhausted Explorer
cohort returns to Explorer, retaining its existing source-changed/reanalysis
guidance when analysis was retired; an exhausted Location Map visit returns to
its map. If no collection files remain, use the existing empty-viewer/error
presentation. Do not automatically restart analysis or silently escape a
restricted scope. Parent-return controls must remain usable when the map has
no eligible images.

### Q9: failed loads with surviving candidates

Should recovery behave exactly like pressing Next, or retain its distinct
successor policy after source reconciliation?

Recommendation: preserve recovery as a separate decision. An explicitly
restored image origin has priority; otherwise preserve the existing successor
choice within the reconciled visit's scope. Display's current retry chain alone
performs the load. Keep the unavailable-HEIC guidance path that stops instead
of advancing. This question concerns surviving candidates; Q8 separately covers
exhausted visits.

### Q10: older duplicate-inspection rules

Does MA-029 also redesign duplicate-group browsing and inspection navigation?

Recommendation: preserve their existing rules through the baseline scope
adapter. The new owner composes with the duplicate model; it does not redesign
grouping, inspection membership, Home/End or singleton/empty fallback behavior.
D5's strict no-widening guarantee applies to the migrated Explorer, ranked
search and Location Map visits. Record duplicate inspection's existing
collection fallback as an explicit compatibility rule rather than silently
changing it under this refactor. Picture-frame timing/shuffle implementation
also remains with its feature and uses the shared navigation seam.

## Implementation planning and required evidence

Start by naming the scope resolver and distinguishing unrestricted, empty
restricted and incomplete scopes. Then migrate Explorer/search image and Grid
returns, followed by Location Map. Preserve explicit root composition and
MA-028 command admission. MA-030 is not a prerequisite: use the existing
occurrence and generation mechanisms and preserve source-reconciliation order.

The Deep SDD plan must map each acceptance criterion to an executable command
and pin the module interface, affected files and incremental migration gates.
It must cover:

1. Visit transitions and scope snapshots through the same interface callers
   use, including empty restrictions, frozen/live scope differences, surviving
   occurrence identity and obsolete deliveries.
2. Collection -> Explorer cohort -> search -> image -> Grid/history/origin
   round trips, preserving Unassigned behavior and Grid bookmarks.
3. Comparison over ranked results, Copy Selection during delayed return,
   preparation progress and refusal without stale surface revelation.
4. Sort/removal during a pending return, source writes, collection replacement,
   last-member exhaustion and restoration to an unavailable origin image.
5. Separate failure recovery with one display retry chain, HEIC guidance and
   the preserved duplicate/Picture-frame behavior.
6. Navigation and preloads sharing each captured scope while Favorite and
   batch actions retain their own target rules. Include a direct Location Map
   image-navigation assertion for new discoveries; its current discovery-only
   regression does not establish that behavior.

Existing integration anchors include
`TestFindMoreLikeThisInitialRoundTrip`,
`TestFindMoreLikeThisProgressiveForegroundIdentity`,
`TestFindMoreLikeThisActionsCaptureRankedSources`,
`TestFindMoreLikeThisSourceAndSortRetirement`,
`TestVisualSimilarityExplorer`, `TestLocationMap` and
`TestEscapeUnwindsModesBeforeReset`. Extend these and add focused module tests
as the interface is specified; their present existence is not proof of the new
contract. Retain the repository's focused-test, negative-guard, GoLand inspection
and final `make verify` requirements, including exact test exclusions/shards.

Structural completion also requires removal/projection of the old migrated
ownership flags and no feature-precedence checks scattered through navigation
consumers. A subsequent browsing feature should provide its scope and
transitions without editing unrelated features' navigation helpers. The lead
owns this review alongside behavioral evidence.

## Evidence and delegation

Two bounded read-only scouts collected return-state and collection-change
facts, each limited to three production files. A third-round follow-up on the
first scout collected the older duplicate-navigation rules from three files.
The lead inspected the cited flows and owns all design decisions; no review or
implementation was delegated.
No application tests, native trials or static inspections were run for this
documentation-only interview. Existing test names above are regression anchors,
not claims that those tests were executed.
