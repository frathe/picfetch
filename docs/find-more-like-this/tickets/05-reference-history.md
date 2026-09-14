# 05: Follow new references and retrace the search

Ticket: FML-005
Status: implemented; current review/CI gate is tracked in [PR #25](https://github.com/frathe/picfetch/pull/25).
Approval: /implement use tdd and sdd, 2026-09-14. Dependencies still gate admission.
Owner: T0 lead; design, review and fixes remain lead-owned.
Budget: zero spawns; at most two lead review rounds; focused checks.

**What to build:** Choose a match as the next reference, search the original collection again using prepared vectors, and use Back/Esc or Exit to recover earlier visits and the initial list.

**Blocked by:** [FML-003](03-progressive-search.md).

**Specification:** AC-03, AC-05, AC-06, AC-07 in the [canonical specification](../spec.md#acceptance-criteria-and-verification).

## Acceptance criteria

- [ ] A new reference always searches the initial captured scope. It supersedes older queries, reuses prepared vectors as soon as its own representation is ready and never restarts preparation or its automatic 100-source batch counter. ([V05](../ticket-execution.md#v05))
- [ ] Ordinary click/Enter still opens an image. A deliberate reference command uses the same menu/shortcut admission rule. Warm queries require no repeated decode or inference, and superseded deliveries cannot replace the current reference. ([V05](../ticket-execution.md#v05))
- [ ] Commit one visit per reference on its first successful publication and revise it on later batches. Capture result order, filename query, selected/highlighted identities and viewport before leaving; store no decoded images or vectors in history. ([V05](../ticket-execution.md#v05))
- [ ] Back first abandons an uncommitted pending query; otherwise it restores the previous successful visit locally without inference. Restored visits stay frozen against late batches. Failed first publications add no entry; later failure preserves the usable visit. ([V05](../ticket-execution.md#v05))
- [ ] Retain the latest 20 successful reference visits plus a separate non-evicted origin. A new query after Back discards the forward branch; Back from the earliest retained visit or Exit restores origin, including a frozen Explorer cohort and camera. ([V05](../ticket-execution.md#v05))
- [ ] Esc respects dialogs/cards/comparison, then Grid marquee/selection/filename search, before Back. From an opened result it first restores ranked Grid. Adverse-order tests observe shutdown and suppression after Back/Exit. ([V05](../ticket-execution.md#v05))

**Demo / completion evidence:** Follow several references, go Back with the same filter and selection, branch again, and Exit to the initial collection without inference.

**Execution:** [FML-005 file map, contracts and verification](../ticket-execution.md#fml-005).

Implementation and executable evidence: [continuation record](../../../finished_refactorings/2026-09-14-find-more-like-this.md).
