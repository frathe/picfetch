# 04: Use ordinary actions on the active result

Ticket: FML-004
Status: implemented; current review/CI gate is tracked in [PR #25](https://github.com/frathe/picfetch/pull/25).
Approval: /implement use tdd and sdd, 2026-09-14. Dependencies still gate admission.
Owner: T0 lead; design, review and fixes remain lead-owned.
Budget: zero spawns; at most two lead review rounds; focused checks.

**What to build:** Filter a ranked result by filename and use comparison, copy, deletion, mosaics and image navigation with the files actually chosen from that result, even while rankings refresh.

**Blocked by:** [FML-003](03-progressive-search.md).

**Specification:** AC-04, AC-06, AC-07 in the [canonical specification](../spec.md#acceptance-criteria-and-verification).

## Acceptance criteria

- [ ] Filename filtering preserves similarity order and does not shrink the original search scope. Save and restore source-bound filter, highlight, selection and viewport facts through Grid visits; hidden selected members remain selected. ([V04](../ticket-execution.md#v04))
- [ ] Comparison captures exactly the selected pair; copy, deletion and mosaics capture their existing command-defined selected/result scope rather than the original collection. Offscreen result members remain eligible where the command includes the full result. ([V04](../ticket-execution.md#v04))
- [ ] Opening and stepping retain the captured result order while background revisions are deferred. Comparison, dialogs and focused controls keep normal input ownership; returning installs only the newest eligible revision. ([V04](../ticket-execution.md#v04))
- [ ] Delayed actions and committed writes cannot be retargeted by reordering or a stale initiating request. Changed/deleted searched sources retire to a reconciled origin, including source aliases, original-scope members outside the current result, and a fully deleted origin; unrelated output leaves exploration usable. ([V04](../ticket-execution.md#v04))
- [ ] Exercise real controls with held deliveries and desktop-effect stubs. The observable chosen files, rendered Grid membership and reconciled origin prove correctness rather than a private index or a visibility flag alone. ([V04](../ticket-execution.md#v04))

**Demo / completion evidence:** Filter a partial result, compare or copy selected images while another batch arrives, and verify the original selected paths reach the existing action.

**Execution:** [FML-004 file map, contracts and verification](../ticket-execution.md#fml-004).

Implementation and executable evidence: [continuation record](../../../finished_refactorings/2026-09-14-find-more-like-this.md).
