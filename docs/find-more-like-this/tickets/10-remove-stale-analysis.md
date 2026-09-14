# 10: Remove stale analysis conservatively

Ticket: FML-013
Status: ready-for-agent
Approval: /implement use tdd and sdd, 2026-09-14. Dependencies still gate admission.
Owner: T0 lead; design, review and fixes remain lead-owned.
Budget: zero spawns; at most two lead review rounds; focused checks.

**What to build:** Use Remove stale records in the same Cache tab to reclaim invalid analysis while preserving records for temporarily disconnected or inaccessible sources.

**Blocked by:** [FML-012](09-clear-analysis-cache.md).

**Specification:** AC-09, AC-10 in the [canonical specification](../spec.md#acceptance-criteria-and-verification).

## Acceptance criteria

- [ ] Remove corrupt/incompatible records, outdated source versions and obsolete Favorite memberships through the same confined maintenance path as full cleanup. ([V13](../ticket-execution.md#v13))
- [ ] Keep records when source status cannot be established because of a disconnected drive or access error. Treat a missing source as stale only when its source location can be checked; ambiguous absence must not become a destructive conclusion. ([V13](../ticket-execution.md#v13))
- [ ] Preserve valid records and all excluded user/assets data. Report removed/remaining bytes and records plus unavailable/skipped/failed items, including honest partial or canceled cleanup. ([V13](../ticket-execution.md#v13))
- [ ] Exercise the actual stale-only Settings command with accessible-deleted, modified, inaccessible, disconnected, corrupt, incompatible and membership-changed fixtures. Writer exclusion, cancellation, close/reopen and shutdown retain the full-cleanup guarantees. ([V13](../ticket-execution.md#v13))

**Demo / completion evidence:** Clean a mixed cache and verify that modified/deleted accessible sources are removed while valid and disconnected-drive records remain.

**Execution:** [FML-013 file map, contracts and verification](../ticket-execution.md#fml-013).
