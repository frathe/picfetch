# 11: Prove recovery across actions and cleanup

Ticket: FML-007
Status: implemented; current review/CI gate is tracked in [PR #25](https://github.com/frathe/picfetch/pull/25).
Approval: /implement use tdd and sdd, 2026-09-14. Dependencies still gate admission.
Owner: T0 lead; design, review and fixes remain lead-owned.
Budget: zero spawns; at most two lead review rounds; focused checks.

**What to build:** Exercise the complete user journey through progressive search, reference history, source changes and cache cleanup, proving that stale work cannot replace the chosen visit or undo completed maintenance.

**Blocked by:** [FML-013](10-remove-stale-analysis.md).

**Specification:** AC-03, AC-04, AC-05, AC-06, AC-07, AC-09, AC-10 in the [canonical specification](../spec.md#acceptance-criteria-and-verification).

## Acceptance criteria

- [ ] Run combined scenarios through real menu, shortcut, Grid, action and Settings entry points: prepare, explore, filter, compare/copy/mosaic/save, Back/Exit, clean cache and resume only by explicit query. ([V07](../ticket-execution.md#v07))
- [ ] Committed Save/Export/EXIF/Mosaic writes reconcile even if their initiating request became stale. Changed/deleted searched sources and aliases retire to a reconciled origin, unrelated outputs preserve history and collection replacement adopts the new collection. ([V07](../ticket-execution.md#v07))
- [ ] Force query replacement between batches, late delivery after Back/Exit, cleanup during preparation, close/reopen, canceled setup and terminal shutdown. Observe worker/process exit and joined callbacks; retain at most one native analysis producer. ([V07](../ticket-execution.md#v07))
- [ ] Negatively verify stale-delivery, action-target and maintenance-writer guards using temporary overlays: each guard must fail for its stated violation and pass again after restoration. ([V07](../ticket-execution.md#v07))
- [ ] Defenses already belong to the slices that introduce the behavior. This ticket adds cross-feature journeys and repairs demonstrated interactions; it is not permission to defer cancellation, source identity or write confinement. ([V07](../ticket-execution.md#v07))

**Demo / completion evidence:** A deterministic held-worker scenario completes a search/action/history/cleanup journey and proves its protections by making each guard fail under deliberate violation.

**Execution:** [FML-007 file map, contracts and verification](../ticket-execution.md#fml-007).

Implementation and executable evidence: [continuation record](../../../finished_refactorings/2026-09-14-find-more-like-this.md).
