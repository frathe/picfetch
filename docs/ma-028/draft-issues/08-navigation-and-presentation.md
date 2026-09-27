# 08: Share navigation and presentation admission

**What to build:** Root-owned navigation, rotation, zoom, merge/info and
Picture-frame settings obey shared admission while feature-local keys,
automatic playback and Escape retain their established meaning.

**Blocked by:** 01: Establish shared admission through Save Changes.

**Status:** draft (publish as ready-for-agent after breakdown approval).

**Agent/model:** T0 Codex Lead, GPT-6 Astra, high reasoning; input ownership and
the difference between user navigation and background display work stay lead-owned.

**Context:** [MA-028 spec](../spec.md), navigation/presentation family and
AC2/AC3/AC6/AC8; use the
[execution plan](../../../plans/2026-09-27-ma-028-command-admission.md).

- [ ] Migrate next/previous/first/last user navigation, rotation both ways,
  reset+fit, actual size, zoom, merge/info, shuffle and interval actions through
  applicable keys, menus, bare entries and secondary-window Host callbacks.
  Classify settings-owned setters, Grid opens and timed display advances
  explicitly. Verify V08.
- [ ] Typed navigation during Copy Selection stays feature-owned; EXIF-host
  navigation yields an idle selection and refuses with feedback when busy.
  Reset key 0 yields because it also resets rotation. Zoom/pan exceptions,
  repeated activation and treatment of unowned keys remain intact. Verify V08.
- [ ] Recognized unavailable commands do not cancel idle selection. Allowed
  commands perform required yields before effects and use the displayed/requested
  subject distinction correctly. Modal/busy queries are pure and invocation
  reevaluates current facts. Verify V08.
- [ ] Preserve comparison/Grid/map/search local key interpretation and Escape's
  ordered return/cancel behavior. Prompt controls and native close remain
  available. Do not move Escape into a command registry. Verify V08.
- [ ] Do not gate generic image loading, committed reconciliation or timed
  slideshow callbacks indiscriminately as new user commands. Preserve their
  lifetimes, manual countdown reset and feature cancellation rules. Verify V08
  plus focused slideshow/display regressions chosen from the actual changed paths.
- [ ] Record all route cases and N/A reasons; remove migrated cross-feature
  predicates while preserving local validity. Record red/green, changed-file
  inspections, new-test exclusions and shard assignments.

**Verification V08:**

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestCommandAdmissionYieldOrdering|TestCommandAdmissionVisits|TestStepImage.*|TestCopySelection.*|TestEscapeUnwindsModesBeforeReset)$'`

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestCommandAdmissionRoutes$/^navigation$'`

Use existing completion observations, not in-flight counters or sleep-based waits.
