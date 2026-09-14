# 09: Clear analysis without losing user data

Ticket: FML-012
Status: implemented; current review/CI gate is tracked in [PR #25](https://github.com/frathe/picfetch/pull/25).
Approval: /implement use tdd and sdd, 2026-09-14. Dependencies still gate admission.
Owner: T0 lead; design, review and fixes remain lead-owned.
Budget: zero spawns; at most two lead review rounds; focused checks.

**What to build:** Use Clear analysis cache in Settings to remove managed general and Favorite analysis, with cancellation, accurate partial results and a usable browsing state during cleanup.

**Blocked by:** [FML-011](08-cache-usage-and-limit.md), [FML-005](05-reference-history.md).

**Specification:** AC-07, AC-09, AC-10 in the [canonical specification](../spec.md#acceptance-criteria-and-verification).

## Acceptance criteria

- [ ] The actual command clears managed general/Favorite analysis records, including recognized managed temporary records. Confine traversal and deletion to owned record roots/formats; preserve source images, Favorite lists/cohorts, thumbnails, installed model/runtime assets, preferences, sessions and unrelated siblings. ([V12](../ticket-execution.md#v12))
- [ ] Before removal, close admission for affected roots, cancel/join affected Explorer/search producers off UI and acquire maintenance ownership shared with cooperating in-process and subprocess writers. A retired writer cannot recreate records after cleanup reports completion. ([V12](../ticket-execution.md#v12))
- [ ] Keep the last usable search/Explorer visit and Back history. Completion does not automatically resume inference; a later explicit analysis/query creates a fresh producer with retained browsing context. Read-only inspection never performs this retirement. ([V12](../ticket-execution.md#v12))
- [ ] Show progress and cancellation. Return removed and remaining record/byte counts with skipped/failed items and visibly partial or canceled outcomes. Release maintenance admission on every success/error/cancel path. ([V12](../ticket-execution.md#v12))
- [ ] Hold a writer immediately before commit and deliver queued callbacks after cleanup, close/reopen and shutdown. Tests observe no late publication, intact excluded data, honest partial outcomes and a nonblocking UI. ([V12](../ticket-execution.md#v12))

**Demo / completion evidence:** Start search preparation, clear analysis while a write is held, then release it; browsing remains usable and the completed cleanup cannot be undone by the old writer.

**Execution:** [FML-012 file map, contracts and verification](../ticket-execution.md#fml-012).

Implementation and executable evidence: [continuation record](../../../finished_refactorings/2026-09-14-find-more-like-this.md).
