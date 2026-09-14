# 07: Reuse analysis across reopened loose lists

Ticket: FML-010
Status: implemented; current review/CI gate is tracked in [PR #25](https://github.com/frathe/picfetch/pull/25).
Approval: /implement use tdd and sdd, 2026-09-14. Dependencies still gate admission.
Owner: T0 lead; design, review and fixes remain lead-owned.
Budget: zero spawns; at most two lead review rounds; focused checks.

**What to build:** Reopen an unchanged loose or overlapping list without repeating eligible inference, reuse Favorite analysis first, and control loose-list persistence from a new Cache settings tab.

**Blocked by:** [FML-006](06-save-result-favorites.md).

**Specification:** AC-03, AC-09, AC-10 in the [canonical specification](../spec.md#acceptance-criteria-and-verification).

## Acceptance criteria

- [ ] Search and Explorer share compatible per-file preparation. Read admitted Favorite analysis before general records regardless of how the source was opened; a compatible general record can warm a newly saved Favorite. Avoid redundant general writes for Favorite-owned sources. ([V10](../ticket-execution.md#v10))
- [ ] Use SHA-256 of normalized absolute path as the key, with full path, size, nanosecond mtime, model/preprocessing version and bounded finite payload validation. Key creation reads no image bytes; moved/copied paths miss, and source versions are checked around preparation. ([V10](../ticket-execution.md#v10))
- [ ] Persist reusable vectors, source facts/digest and bounded preview only. Do not persist query ranks, map placement or history. Corrupt/incompatible/unusable entries fall back to preparation with bounded warnings; acknowledge the existing same-size/same-time rewrite limitation. ([V10](../ticket-execution.md#v10))
- [ ] Add the Cache tab and default-enabled loose-list persistence control using existing live preference/persistence behavior. Preserve the Favorite toggle; an opted-out Favorite write must not be redirected to general storage. Stored records remain available for later inspection when toggles are off. ([V10](../ticket-execution.md#v10))
- [ ] Bound general storage to the initial 2048 MB budget with serialized-byte accounting and LRU admission, including managed temporary files. Skip persistence for an oversized record without losing its in-memory result; Favorite analysis is outside the cap. User retuning and detailed usage arrive in FML-011. ([V10](../ticket-execution.md#v10))
- [ ] Writes and automatic eviction use root-scoped admission shared with cooperating subprocess writers. Old writer leases cannot publish after eviction; cancel/join affected producers off UI before removal while retaining usable browsing. Explicit later queries can restart; eviction does not restart inference itself. ([V10](../ticket-execution.md#v10))
- [ ] Temporary-root tests count encoder calls across independent sessions, overlapping lists and Favorite promotion, exercise both preference combinations and stale/corrupt records, and prove bounded writes plus writer exclusion under a scaled budget. ([V10](../ticket-execution.md#v10))

**Demo / completion evidence:** Prepare a loose list, close and reopen it, then save a result as a Favorite and observe compatible reuse with no repeated inference.

**Execution:** [FML-010 file map, contracts and verification](../ticket-execution.md#fml-010).

Implementation and executable evidence: [continuation record](../../../finished_refactorings/2026-09-14-find-more-like-this.md).
