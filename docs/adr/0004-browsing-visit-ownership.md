# Give browsing visits one root-UI owner

Status: accepted; implemented and CI-qualified

Use one private root-UI module to own the active browsing visit, its ordered
scope, foreground visit, return destination and valid transitions across
Explorer, ranked search and Location Map. Root executes ordered feature calls;
Grid retains interaction state, search retains query history, maps retain
cameras and data, and MA-028 retains command admission. This makes visit
invariants authoritative through a small interface while preserving explicit
composition, at the cost of maintaining adapters to feature-owned state.

A shared struct with independent setters would preserve the existing ordering
burden on callers. A global application store or flat mode enum would conflate
retained visits with foreground surfaces and input owners. Keep those concepts
distinct, remove or derive migrated ownership flags, and reuse existing
collection identities/reconciliation without making MA-030 a prerequisite.

The [accepted design](../browsing-visits.md) records all ten decisions,
including explicit empty-scope returns, separate failed-load recovery and the
older duplicate-inspection compatibility rules, plus the required verification.
