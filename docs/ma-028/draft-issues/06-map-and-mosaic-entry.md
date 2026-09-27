# 06: Admit map and mosaic visits across preparation

**What to build:** Explorer, Location Map and mosaic enter consistently from
menus, keys and direct callbacks, reject obsolete preparation, and preserve the
restrictions and source identity of retained visits.

**Blocked by:** 01: Establish shared admission through Save Changes.

**Status:** draft (publish as ready-for-agent after breakdown approval).

**Agent/model:** T0 Codex Lead, GPT-6 Astra, extra-high reasoning. Preparation,
retained visits and feature-owned lifetimes require cross-feature judgment.

**Context:** [MA-028 spec](../spec.md), AC2/AC6/AC8/AC9; follow the
[execution plan](../../../plans/2026-09-27-ma-028-command-admission.md). Browsing
ownership refactoring MA-029 and collection-transition MA-030 are out of scope.

- [ ] Migrate Explorer entry/retry, Location Map entry/return and mosaic entry
  through applicable menus, Shift+S/L/M, bare methods and Host callbacks. Keep
  Grid search's ownership of shifted text. Menu presentation and invocation use
  shared decisions without moving state to a central controller. Verify V06.
- [ ] Distinguish visible maps from retained Explorer cohorts and Location
  image/cluster-Grid visits. Record map-image, map-cluster and cohort/unassigned
  callbacks as new entries or feature-owned transitions as appropriate; retain
  their existing admission and ordered return behavior. Verify V06.
- [ ] Hold setup, source validation or duplicate preparation, change live
  admission, then deliver. Reject disallowed or stale continuation without
  replay while retaining each feature's tokens and established source semantics.
  Explorer still uses its existing re-entry semantics. Verify V06.
- [ ] Preserve mosaic's defensive source-pool snapshot, exact finished result
  and target display. Ordinary feature controls and committed notifications
  retain their owners. New-entry refusal cannot discard a committed effect.
  Verify V06 and the focused existing mosaic regressions.
- [ ] Unavailable commands preserve idle region selection; required yields
  precede effects; busy/modal menu state and refusal feedback remain consistent.
  Remove only migrated cross-feature checks. Record route/async cases,
  red/green, inspections and manifests; retain queue/shutdown settlement.

**Verification V06:**

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestCommandAdmissionVisits|TestCommandAdmissionAsync|TestVisualSimilarityExplorer|TestLocationMap|TestMosaic.*)$'`

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestCommandAdmissionRoutes$/^maps$'`

Logical independence from ticket 05 does not permit simultaneous edits to shared
input/menu files; follow the plan's serial schedule.
