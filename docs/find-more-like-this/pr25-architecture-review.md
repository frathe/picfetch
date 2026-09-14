> Follow-up: Ronin authorized the proposed refactor. Implementation and verification
> are tracked in the [ownership plan](../../plans/2026-09-15-search-ownership.md).
> The assessment below records the code and reviews before that implementation.

# PR #25: architecture assessment from the complete Codex history

Date: 2026-09-15. Reviewed tree: `fd043bedaba24c72a71bbcc92edae4310785d8d6`.
Status: assessment complete; proposed refactoring has not been implemented.
Ronin requested this assessment before further finding-by-finding fixes.

## Conclusion

The history supports a focused architectural follow-up in cache policy and
maintenance, followed by a smaller review of browsing transitions. Several
rules require knowledge spread across consumers or operation phases. Successive
reviews found the same rule broken in a different consumer or at a different
point in its lifetime. That is stronger evidence than the raw number of findings.

This does not justify replacing PicFetch's overall feature architecture. The
viewer-independent worker, retained search feature, Grid, root composition and
per-instance UI queues serve distinct purposes. Several accepted fixes have
already improved locality. A new manager that only forwards their existing calls
would add interface complexity without removing any of the present obligations.

## Evidence and limits

The complete paginated GitHub thread response contains 30 threads, including
28 originating Codex findings. The recorded dispositions are 27 accepted/fixed
and one rejected against the explicit temporary-file budget. All 28 were read,
including replies and resolved/outdated threads. The classification below assigns
each finding exactly once. The full inventory links each original comment.

At inspection time, GitHub exposed no unresolved threads or newer findings on
PR #25. This assessment covers the accessible history; a separate unpublished
report would need to be added. Review comments' current `commit_id` can follow
updated diff positions; chronology here uses their creation time and recorded
fix replies, not that field as proof of the original reviewed tree.

Current implementation and relevant contracts/tests were inspected directly.
This is a design assessment, not a new security audit or a claim that every
potential defect has been excluded. Tests were not rerun for this documentation
change. The latest reviewed commit already has passing native Linux/Windows/macOS
CI, clean Codex code/security completion, zero post-suppression Qodana results,
and zero open code-scanning alerts, as recorded on the PR.

| Theme | Findings | Interpretation |
| --- | ---: | --- |
| Cache policy, ownership and partial failures | 7 | Strongest evidence of shared rules escaping their owning module. |
| Maintenance intent, lifetime and results | 6 | Cancellation, committed effects and presentation lifetime need a clearer operation model. |
| Browsing and session integration | 8 | Cross-feature transitions and identity need a consistently consumed contract. |
| Repeated work at scale | 4 | Complexity was hidden in repeated helper calls; substantial corrections already exist. |
| Transport admission | 1 | An independent request limit needed explicit admission. |
| Documentation | 1 | The exhaustive queue-owner list lagged implementation. |
| Intentional staging budget | 1 | Rejected finding; retain temporary-file headroom. |

## 1. Cache ownership and policy are not fully encapsulated

The clearest recurrence is unreadable Favorite membership. Maintenance first
omitted healthy peers; the producer then repeated the same all-or-nothing
inventory mistake. Later fixes had to preserve opt-out behavior when general
records still existed, make newly saved Favorites reuse compatible analysis,
and permit policy changes despite unrelated inspection failures.

The current tree has useful shared code: `loadFavoriteAnalysis` validates a
captured file-list version, and `representationStore` provides common reuse and
promotion. However:

- [cache.go](../../internal/similarity/cache.go), `openAnalysisCache` at line 38,
  and [cache_records.go](../../internal/similarity/cache_records.go), `scan` at
  line 33, independently enumerate Favorites and assign meaning to per-entry
  failures. Sharing the individual file parser does not share inventory semantics.
- [cache_store.go](../../internal/similarity/cache_store.go), `read` and `write`,
  interpret membership plus persistence flags. Explorer's
  [analyze.go](../../internal/similarity/analyze.go), lines 196–198, additionally
  checks `cache.policy.FavoriteEnabled` and directly calls `cache.favorites.write`.
  That preserves Explorer's intentional Favorite-only writes, but exposes the
  store's internal routing to a second consumer.
- A missing source cannot presently be distinguished reliably from a disconnected
  volume. The conservative retention rule is a deliberate domain limitation;
  extracting a module cannot manufacture the missing volume identity.

Recommended first change: give shared Favorite inventory and access policy one
owner. Its interface should carry known membership, incomplete/unknown ownership,
and captured versions explicitly. Configure each producer's permitted reads and
writes when it opens the store, so Search and Explorer use the same record
operations while preserving their different write policies. Keep directory
handles, source/version validation and conservative unknown-ownership behavior
inside this implementation. This can begin within `internal/similarity`; a package
move alone is not the improvement.

## 2. Maintenance combines operation state with Settings presentation

The repeated failures involved distinct concepts that had been treated alike:
editing versus accepting a limit, observing usage versus performing eviction,
closing Settings versus retiring writers, and an error versus a successful
general-cache effect accompanied by incomplete Favorite information.

[analysiscache/feature.go](../../internal/ui/analysiscache/feature.go) now correctly
separates draft text, accepted policy, pending inspection and automatic eviction.
[work.go](../../internal/ui/analysiscache/work.go), line 31, still runs these
different intents through `start(viewBound, run, apply, after...)`, which cancels
the current operation. Callers must arrange exceptions and pending work before
entering that shared mechanism. `setEnabled` even uses it to join retirement
barriers independently of a maintenance provider call. These are observed
interface obligations, not a newly reproduced failure.

[cache_management.go](../../internal/similarity/cache_management.go) must separately
track preliminary inventory, inventory under the lease, committed removals,
reconciliation, cancellation and whether a general limit was applied. The later
post-lock cancellation finding followed the earlier cancellation fix because
another return path still exposed the wrong observation. The current
`AppliedLimit` and per-store completeness fields are real improvements; retain
their semantics.

Recommended change: model operation intent and lifetime explicitly within the
existing owner. Usage refresh may be superseded; automatic eviction survives
Settings closure; accepting a preference has a different commit point from
deleting cache records. Have the maintenance implementation construct one outcome
that states committed effects, observation completeness, cancellation and applied
policy independently. The Settings view renders that outcome rather than
reconstructing success from combinations of error and status fields. Preserve
root-scoped cross-process leases, epochs, admission closure and producer joins.

## 3. Browsing transitions require a clearer shared contract

The eight findings here span restored Explorer surfaces, deferred overlay
delivery, preload order, Back progress, duplicate occurrence identity, trial
completion, menu capability and delayed setup admission. They share a theme:
one transition changed part of the application while another consumer still
used a different idea of the active view, identity or lifetime.

The present separation is partly intentional and should remain:

- [visualsearch/feature.go](../../internal/ui/visualsearch/feature.go) owns source
  scope, query identity, visits/history and live session progress.
- Root [visualsearch.go](../../internal/ui/visualsearch.go) owns surface
  composition, pending presentation and the frozen order used while an image is
  open. Grid owns gesture/selection/scroll state and duplicate occurrences.
- The newest result, an opened image's captured order and a saved history visit
  are different values. Combining them into one mutable result list would
  reintroduce target changes under the user.

There is already one `activeSearchIndexes` resolver. Navigation reaches it through
`cohortIndexes`/`nextVisibleIndex`; preload now follows that path too; Favorites
captures one `CurrentFiles` snapshot. Back now retains live session progress.
Those fixes should be preserved rather than replaced with another forwarding layer.

Recommended next step: write and test the root browsing transition contract, then
consolidate its remaining decisions. Capture an immutable browsing context for
each user operation: identity/occurrence, applicable order, origin and available
commands. Menu enablement, command admission, preload and action capture should
consume the same decisions. Keep those cross-feature decisions in `internal/ui`,
as the repository conventions require. Treat Enter, Back, Exit, source replacement,
setup completion and producer retirement as complete transitions with defined
effects. Do not build a general application controller or move widgets into the
search worker. Keep Fyne's missing generic overlay-close notification behind its
existing small adapter.

## 4. Scaling and verification contributed independently

Four accepted findings were repeated-work problems: inventory on every cache
write, source validation over every growing prefix, repeated ranked-path lookup
while saving, and full-prefix reranking at each publication. A helper can look
bounded locally while its repeated caller makes total work quadratic.

The fixes now include revision-aware accounting, bounded intermediate validation,
one captured file snapshot, and incremental top-k ranking. The recorded ranking
benchmark improved 10,000 prepared vectors from about 682 ms to 27 ms; this
excludes decoding, inference and cache I/O. The 446-image human relevance exercise
answered a different question and could not establish large-collection behavior.

Retain deterministic work-count tests at N and 2N alongside relevant benchmarks.
Do not use a broad refactor to replace already-correct incremental work. Preserve
the explicit per-message transport budget and the intentional temporary-file
disk budget. Neither issue calls for a viewer-wide redesign.

## Proposed order and proof

1. Shared cache inventory/access policy. Exercise both real producer paths against
   the same policy cases: Favorite/loose enabled combinations, complete/incomplete
   ownership, healthy/broken peers, general-to-Favorite promotion and version
   changes. No caller should need to reach into store internals to uphold policy.
2. Explicit maintenance intent and outcomes. Check cancellation before the lease,
   during quiescence, during inventory, after committed removal and during final
   reconciliation. Combine these with Settings close/reopen and unrelated Favorite
   errors. Observe committed files and policy, not just a returned error.
3. Browsing transition coverage and a narrowly scoped consolidation in root.
   Exercise cohort -> search -> partial result -> opened image -> final result ->
   overlay dismissal -> Back -> Exit, plus source changes and duplicate occurrences.
   Assert displayed identity, selection, all action targets, preload neighbors,
   independent session progress and one terminal observation per admitted trial.
4. Preserve linear-work guards and rerun native platform checks and the full GitHub
   review loop after implementation. Existing meaningful regressions stay until
   replacement coverage proves the same behavior through the resulting interface.

These are proposed refactoring acceptance criteria, not claims of work completed.
The first two changes address the strongest recurrence. I recommend settling their
design before further local patches, then doing the browsing work as a separate
change. A completely new caching engine, database, global event bus or generic
controller is not justified by this evidence.

## Assessment process

Lead read all findings/dispositions, assessed current code and contracts, and
owns every recommendation. One read-only scout mapped search state and consumers;
the lead verified its key navigation/preload/restoration references in the source.
No fixes or reviews were delegated. Scout gate: bounded factual question, no writes,
file/line oracle, cold UI ownership context, no implementation prescribed; shell
search identified the modules but did not trace their relationships. Budget/actual:
one scout/one scout. No source edits, new regression tests, commits, pushes or bot
review requests were made for this assessment.

## Complete finding inventory

Numbers follow the original thread inventory; gaps are non-Codex threads.
All entries below are resolved on GitHub. "Fixed" records the existing disposition,
not a new fix made during this assessment.

| Thread | Original Codex finding | Theme | Existing disposition |
| --- | --- | --- | --- |
| 2 | [Avoid rescanning the whole cache for every record](https://github.com/frathe/picfetch/pull/25#discussion_r4007684414) | Scaling | Fixed |
| 3 | [Restore the Explorer surface with a cohort origin](https://github.com/frathe/picfetch/pull/25#discussion_r4007684420) | Browsing / session | Fixed |
| 4 | [Exclude the replaced record from the limit check](https://github.com/frathe/picfetch/pull/25#discussion_r4007684427) | Staging budget | Rejected: temporary bytes count toward the limit |
| 5 | [Commit cache limits only after editing finishes](https://github.com/frathe/picfetch/pull/25#discussion_r4007684434) | Maintenance lifecycle | Fixed |
| 6 | [Flush the final result after generic overlays close](https://github.com/frathe/picfetch/pull/25#discussion_r4007684440) | Browsing / session | Fixed |
| 7 | [Preload neighbors from the ranked search order](https://github.com/frathe/picfetch/pull/25#discussion_r4007684445) | Browsing / session | Fixed |
| 8 | [Promote cached analysis when saving a new Favorite](https://github.com/frathe/picfetch/pull/25#discussion_r4008194374) | Cache policy / ownership | Fixed |
| 9 | [Preserve automatic eviction when Settings opens](https://github.com/frathe/picfetch/pull/25#discussion_r4008194381) | Maintenance lifecycle | Fixed |
| 10 | [Revalidate Favorite membership before stale deletion](https://github.com/frathe/picfetch/pull/25#discussion_r4008194389) | Maintenance lifecycle | Fixed |
| 11 | [Avoid restatting every prior source for each batch](https://github.com/frathe/picfetch/pull/25#discussion_r4008395315) | Scaling | Fixed |
| 12 | [Continue past unreadable Favorite definitions](https://github.com/frathe/picfetch/pull/25#discussion_r4008395320) | Cache policy / ownership | Fixed |
| 13 | [Keep completed preparation status when navigating Back](https://github.com/frathe/picfetch/pull/25#discussion_r4008395326) | Browsing / session | Fixed |
| 14 | [Preserve healthy Favorite membership after inventory errors](https://github.com/frathe/picfetch/pull/25#discussion_r4008572101) | Cache policy / ownership | Fixed |
| 15 | [Respect the Favorite opt-out before general-cache fallback](https://github.com/frathe/picfetch/pull/25#discussion_r4008572107) | Cache policy / ownership | Fixed |
| 16 | [Lock cache roots that are absent when maintenance starts](https://github.com/frathe/picfetch/pull/25#discussion_r4008572116) | Maintenance lifecycle | Fixed |
| 17 | [Apply the persistence toggle despite inspection failures](https://github.com/frathe/picfetch/pull/25#discussion_r4008793451) | Cache policy / ownership | Fixed |
| 18 | [Reuse one ranked-index snapshot when saving Favorites](https://github.com/frathe/picfetch/pull/25#discussion_r4008793460) | Scaling | Fixed |
| 19 | [Update the exhaustive UIQueue exception list](https://github.com/frathe/picfetch/pull/25#discussion_r4008793472) | Documentation | Fixed |
| 20 | [Stop reranking the full prefix at every publication](https://github.com/frathe/picfetch/pull/25#discussion_r4008793476) | Scaling | Fixed |
| 22 | [Preserve duplicate occurrence identity when restoring visits](https://github.com/frathe/picfetch/pull/25#discussion_r4009091797) | Browsing / session | Fixed |
| 23 | [Keep cancellation active during the final inventory](https://github.com/frathe/picfetch/pull/25#discussion_r4009091806) | Maintenance lifecycle | Fixed |
| 24 | [Accept every vector-budget-valid search request](https://github.com/frathe/picfetch/pull/25#discussion_r4009091815) | Transport | Fixed |
| 25 | [Apply general-cache limits despite unrelated Favorite errors](https://github.com/frathe/picfetch/pull/25#discussion_r4009091827) | Cache policy / ownership | Fixed |
| 26 | [Record canceled Explorer trials before barrier exit](https://github.com/frathe/picfetch/pull/25#discussion_r4009091837) | Browsing / session | Fixed |
| 27 | [Disable duplicate commands during ranked search](https://github.com/frathe/picfetch/pull/25#discussion_r4009091850) | Browsing / session | Fixed |
| 28 | [Preserve the post-lock inventory when cancellation lands](https://github.com/frathe/picfetch/pull/25#discussion_r4009379209) | Maintenance lifecycle | Fixed |
| 29 | [Do not treat a surviving mount point as source availability](https://github.com/frathe/picfetch/pull/25#discussion_r4009379214) | Cache policy / ownership | Fixed |
| 30 | [Revalidate the captured reference after asset setup](https://github.com/frathe/picfetch/pull/25#discussion_r4009379226) | Browsing / session | Fixed |
