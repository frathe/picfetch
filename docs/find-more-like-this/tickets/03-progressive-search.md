# 03: Browse matches while preparation continues

Ticket: FML-003
Status: ready-for-agent
Approval: /implement use tdd and sdd, 2026-09-14. Dependencies still gate admission.
Owner: T0 lead; design, review and fixes remain lead-owned.
Budget: zero spawns; at most two lead review rounds; focused checks.

**What to build:** Make the first search useful before all images are prepared: publish interactive ranked batches every 100 distinct processed sources and show continuously advancing progress above Grid.

**Blocked by:** [FML-002](02-first-search.md).

**Specification:** AC-03, AC-04, AC-07 in the [canonical specification](../spec.md#acceptance-criteria-and-verification).

## Acceptance criteria

- [ ] Prepare the reference first. Count each distinct path once, including reference, cache hits and failures. Progress shows processed/total and failure counts between result publications; Explorer's independent 30-image setting is unchanged. ([V03](../ticket-execution.md#v03))
- [ ] Publish at every 100 processed sources and completion, including fewer than 100 and the final remainder. An exact final boundary emits one final result, not a duplicate partial; exercise 99/100/101/200-source scopes. ([V03](../ticket-execution.md#v03))
- [ ] Apply a foreground revision without reopening Grid. Surviving selected/highlighted identities and a usable viewport remain stable; disappearing result members drop without substitution. Filename-hidden selections remain selected. ([V03](../ticket-execution.md#v03))
- [ ] Every action captures source identities when admitted. Delayed gestures cannot act on the new occupant of an old cell. While an image, comparison or modal owns the surface, retain its captured view/input and apply only the newest eligible Grid revision on return. ([V03](../ticket-execution.md#v03))
- [ ] The first successful publication commits the visit; later batches revise it. Failure before any publication adds no successful visit; later failure retains the last usable result with honest status. Exit, source replacement and close suppress queued batches. ([V03](../ticket-execution.md#v03))
- [ ] The worker retains its prepared index after completion for later queries, with immutable session/query/publication-tagged deliveries and observed cancel/EOF/exit. No vectors or collection-wide previews travel to UI or history. ([V03](../ticket-execution.md#v03))

**Demo / completion evidence:** Hold a preparation worker between batches, open or select a result, then release the next batch without losing the chosen source.

**Execution:** [FML-003 file map, contracts and verification](../ticket-execution.md#fml-003).
