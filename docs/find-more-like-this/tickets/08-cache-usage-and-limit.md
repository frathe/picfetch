# 08: Inspect cache usage and adjust its limit

Ticket: FML-011
Status: implemented; current review/CI gate is tracked in [PR #25](https://github.com/frathe/picfetch/pull/25).
Approval: /implement use tdd and sdd, 2026-09-14. Dependencies still gate admission.
Owner: T0 lead; design, review and fixes remain lead-owned.
Budget: zero spawns; at most two lead review rounds; focused checks.

**What to build:** Show general, Favorite and combined analysis usage in MB in Settings Cache, and let the user persistently change the general cache limit from its initial 2 GB value.

**Blocked by:** [FML-010](07-persistent-analysis-cache.md).

**Specification:** AC-09, AC-10 in the [canonical specification](../spec.md#acceptance-criteria-and-verification).

## Acceptance criteria

- [ ] The actual Cache-tab content shows measured general/Favorite/combined MB and a general limit initially 2048 MB. Inspect stored records even when their persistence toggle is off, count managed serialized temporary files, and label incomplete measurements honestly. ([V11](../ticket-execution.md#v11))
- [ ] Validate a positive, overflow-safe limit; invalid edits retain the previous valid value. Apply valid edits live and persist them across launches with existing Settings snapshot/patch behavior. ([V11](../ticket-execution.md#v11))
- [ ] If a lower limit requires eviction, close affected writer admission, cancel/join affected producers off UI and evict general records by least recent use on tracked workers. Preserve Favorite analysis and usable browsing; a later explicit query can start a fresh producer. ([V11](../ticket-execution.md#v11))
- [ ] Opening the tab inspects only and leaves producers active. Increases that need no eviction neither delete records nor invalidate shared writer leases. A valid changed limit retires and joins local Explorer/search producers before the initial maintenance inventory while retaining browsing. Persist only successful current maintenance; cancellation or failure keeps the prior accepted limit without restarting producers. The next explicit query uses the accepted limit. An unchanged policy keeps the worker alive. Report applied limit and measured remaining usage without treating the general limit as a combined-cache or process-memory cap. ([V11](../ticket-execution.md#v11))
- [ ] The owned Cache-tab feature tracks inspection/retuning and a per-instance queue. Close does not block UI, Stop ends admission and Settle observes worker completion and queued delivery. Settings close/reopen and application shutdown cannot install old results. ([V11](../ticket-execution.md#v11))

**Demo / completion evidence:** Open Cache, observe separate totals, lower the limit below current general usage, and see background LRU removal plus the persisted new limit.

**Execution:** [FML-011 file map, contracts and verification](../ticket-execution.md#fml-011).

Implementation and executable evidence: [continuation record](../../../finished_refactorings/2026-09-14-find-more-like-this.md).
