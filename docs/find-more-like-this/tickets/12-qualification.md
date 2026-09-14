# 12: Qualify the complete feature on supported desktops

Ticket: FML-008
Status: ready-for-agent
Approval: /implement use tdd and sdd, 2026-09-14. Dependencies still gate admission.
Owner: T0 lead; design, review and fixes remain lead-owned.
Budget: zero spawns; at most two lead review rounds; full final gate.

**What to build:** Deliver recorded English/German, keyboard, native-session, cache and resource evidence for the complete feature, with repository checks and truthful platform limitations.

**Blocked by:** [FML-007](11-lifecycle-regressions.md).

**Specification:** AC-01, AC-08 in the [canonical specification](../spec.md#acceptance-criteria-and-verification).

## Acceptance criteria

- [ ] Exercise real cold/warm sessions, first partial and full preparation, repeated references, cross-launch Favorite/general reuse, settings persistence and both cleanup commands on each qualified supported native target with installed assets. ([V08](../ticket-execution.md#v08))
- [ ] Rerun the labeled evaluation through production ranking/session/cache behavior. Complete the deferred general-cache and final-session measurements, compare targets, record machine/model/corpus identities and publish measured limitations rather than inferred performance. ([V08](../ticket-execution.md#v08))
- [ ] Inspect actual English/German controls and keyboard ownership. Strings were translated in their introducing slices; this gate checks the combined experience and all locale/manual guards. ([V08](../ticket-execution.md#v08))
- [ ] Record existing dependency/model/runtime source versions, shipped closure, license obligations and notice delivery. Preserve current macOS/Linux denial and Windows unverified network-isolation status; no new model, dependency or implicit download is authorized. ([V08](../ticket-execution.md#v08))
- [ ] Complete the native Linux/amd64 verification path, exact UI shard assignments, test inspection exclusions, architecture updates and GoLand inspections for changed code. Skipped native tests, empty selections or unavailable inspections are explicitly unverified and cannot satisfy acceptance. ([V08](../ticket-execution.md#v08))

**Demo / completion evidence:** Recorded native runs and repository checks establish what is supported, measured and still unverified before feature acceptance.

**Execution:** [FML-008 file map, contracts and verification](../ticket-execution.md#fml-008).
