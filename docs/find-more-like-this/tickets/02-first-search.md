# 02: Find matches and return to the original list

Ticket: FML-002
Status: implemented; current review/CI gate is tracked in [PR #25](https://github.com/frathe/picfetch/pull/25).
Approval: /implement use tdd and sdd, 2026-09-14. Dependencies still gate admission.
Owner: T0 lead; design, review and fixes remain lead-owned.
Budget: zero spawns; at most two lead review rounds; focused checks.

**What to build:** From an image or Grid, invoke Find more like this, prepare the captured collection locally, browse one completed ranked result, and return to the original visit. This is the first usable search path; later slices extend it with live batches and reference history.

**Blocked by:** [FML-001](01-ranking-evaluation.md), including its recorded proceed decision.
The [MA-026 Explorer extraction](../../../needs_refactoring.md#ma-026) prerequisite is complete.

**Specification:** AC-02, AC-03, AC-04, AC-05, AC-06, AC-07 in the [canonical specification](../spec.md#acceptance-criteria-and-verification).

## Acceptance criteria

- [ ] Actions and Cmd/Ctrl+Shift+L use one admission decision: displayed image, exactly one explicit Grid selection, otherwise the highlighted source. Multiple selections disable the command; existing dialog, comparison, Copy Selection, scan/sort and picture-frame ownership remains intact. ([V02](../ticket-execution.md#v02))
- [ ] Freeze distinct absolute source paths from the original loaded collection, including merged folders. Preserve the original list and visit independently; filename filtering and Explorer cohorts do not narrow the search pool. ([V02](../ticket-execution.md#v02))
- [ ] Use explicit asset setup with cancellation/retry and current platform admission. A setup completion starts only its requesting feature after rechecking scope. Explorer analysis stops and joins off UI before search; the frozen cohort and camera remain returnable. ([V02](../ticket-execution.md#v02))
- [ ] Exact cosine ranks valid 768-dimensional vectors by descending score and ascending path ties, excludes the reference/repeated paths, retains distinct identical-pixel files, and returns up to 30 matches without a cutoff. Invalid candidates count as failures; a failed reference produces a recoverable error. ([V02](../ticket-execution.md#v02))
- [ ] The completed result becomes the active ordered list. Click/Enter opens an image and stepping uses its captured result order. Exit restores origin; Esc from an opened result first restores its Grid, then follows existing input unwinding before returning to origin. ([V02](../ticket-execution.md#v02))
- [ ] Use a cancellable search operation through the real worker transport and owned feature queue, preserving ordinary Explorer analysis. Show a top processed/total progress bar, reuse admitted Favorite analysis, check source versions and the separate 256 MiB vector cap before allocation, and release source pixels between inputs. ([V02](../ticket-execution.md#v02))
- [ ] Even this first slice rejects stale setup/result delivery and retires a changed/deleted search source to a reconciled origin. Unrelated outputs preserve browsing; collection replacement adopts the new collection. Close is nonblocking, shutdown is terminal and all workers are observed stopped. ([V02](../ticket-execution.md#v02))
- [ ] Actual menu/shortcut tests complete the path through an injected provider; ranking fixtures and transport tests prove its backend. New visible strings are translated when introduced. The evaluator uses the production ranker rather than retaining a second algorithm. ([V02](../ticket-execution.md#v02))

**Demo / completion evidence:** Search from a merged collection, open a match, and Exit back to the same original visit. Reference chaining and automatic partial-result publication arrive in later tickets.

**Execution:** [FML-002 file map, contracts and verification](../ticket-execution.md#fml-002).

Admission: Ronin’s September 14 qualitative proceed decision supersedes the
per-item judgment prerequisite; see the updated evaluation record.

Implementation and executable evidence: [continuation record](../../../finished_refactorings/2026-09-14-find-more-like-this.md).
