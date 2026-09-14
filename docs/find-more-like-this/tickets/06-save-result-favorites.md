# 06: Save a captured result as a Favorite

Ticket: FML-006
Status: ready-for-agent
Approval: /implement use tdd and sdd, 2026-09-14. Dependencies still gate admission.
Owner: T0 lead; design, review and fixes remain lead-owned.
Budget: zero spawns; at most two lead review rounds; focused checks.

**What to build:** Save selected matches or the complete filtered result as a named Favorite, and make Add Current List capture the active ranked list during exploration.

**Blocked by:** [FML-004](04-result-actions.md).

**Specification:** AC-06 in the [canonical specification](../spec.md#acceptance-criteria-and-verification).

## Acceptance criteria

- [ ] Save to Favorites snapshots selected members of the current filtered Grid result, or every filtered result when none are selected, including offscreen files. Hidden selected files outside that filtered result are not silently added. ([V06](../ticket-execution.md#v06))
- [ ] Add Current List snapshots the complete active ranked list during search and the ordinary collection outside search. Capture at invocation, before naming or overwrite dialogs, and retain a defensive copy through later result changes. ([V06](../ticket-execution.md#v06))
- [ ] Reuse naming, overwrite, error and preview behavior. Saving a new result does not alter the origin Favorite unless the user explicitly chooses to overwrite it. ([V06](../ticket-execution.md#v06))
- [ ] Tests invoke actual entry points and hold naming/overwrite completion across ranked revisions and navigation. The saved Favorite membership is exactly the invocation-time scope. ([V06](../ticket-execution.md#v06))
- [ ] This slice does not depend on a general cache. FML-010 later proves that compatible loose-list analysis can warm these newly saved Favorites without inference. ([V06](../ticket-execution.md#v06))

**Demo / completion evidence:** Open a naming dialog, allow another ranked batch to arrive, then save and confirm that Favorite membership still matches the invocation-time list.

**Execution:** [FML-006 file map, contracts and verification](../ticket-execution.md#fml-006).
