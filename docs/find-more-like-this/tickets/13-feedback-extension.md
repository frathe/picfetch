# 13: Explore positive and negative examples

Ticket: FML-009
Status: needs-triage
Disposition: deferred; not an MVP implementation ticket.
Owner: T0 lead; design, review and fixes remain lead-owned.
Budget when selected: zero spawns; one experiment and at most two lead review rounds; focused checks.

**What to build:** If separately selected after the first version is evaluated, investigate whether explicit positive/negative examples improve useful discovery without replacing the single-reference workflow.

**Blocked by:** [FML-008](12-qualification.md); separate selection and an accepted follow-up specification.

**Specification:** Out of scope for MVP in the [canonical specification](../spec.md#acceptance-criteria-and-verification).

## Acceptance criteria

- [ ] Keep this follow-up deferred and outside MVP blocking edges. FML-008 completion alone does not authorize implementation. ([V09](../ticket-execution.md#v09))
- [ ] Before coding, define feedback meaning, reset/undo behavior, original-scope semantics, explanation to the user and evaluation against the single-reference baseline. ([V09](../ticket-execution.md#v09))
- [ ] Preserve the candidate experiment: compare a normalized positive centroid with a bounded negative-centroid penalty. Evaluate weighting, example limits and content/appearance intent before choosing defaults; handle contradictory examples and invalid or degenerate representations explicitly. ([V09](../ticket-execution.md#v09))
- [ ] Examples stay within the prepared session and are excluded from matches. Removal/reset and Back restore captured example sets and results without inference; supplying no feedback preserves single-reference results. Real labeled judgments establish benefit, with deterministic fixtures supporting the selected rule. ([V09](../ticket-execution.md#v09))
- [ ] Record a separately accepted experiment and falsifiable verification command; do not infer training, persistent preferences or a new model from this placeholder. ([V09](../ticket-execution.md#v09))

**Demo / completion evidence:** No implementation is scheduled. A separately selected experiment must establish an approved, testable scope first.

**Execution:** [FML-009 file map, contracts and verification](../ticket-execution.md#fml-009).
