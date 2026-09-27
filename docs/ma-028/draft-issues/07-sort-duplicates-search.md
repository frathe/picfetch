# 07: Preserve restricted browsing across sort, duplicates and search

**What to build:** Sort, duplicate operations and Find more like this observe
the same restrictions from each route, including while a map visit is retained
behind an image or Grid and while search setup completes asynchronously.

**Blocked by:** 01: Establish shared admission through Save Changes.

**Status:** draft (publish as ready-for-agent after breakdown approval).

**Agent/model:** T0 Codex Lead, GPT-6 Astra, extra-high reasoning. Retained visit
restrictions and search generation/reference are the primary risks.

**Context:** [MA-028 spec](../spec.md), sort/duplicates/search family and
AC2/AC6/AC8/AC9; use the
[execution plan](../../../finished_refactorings/2026-09-27-ma-028-command-admission.md).

- [ ] Migrate selected/cycled sort, hide duplicates, browse variants and Find
  more like this through actual menus, S/D/Shift+D, the search shortcut and bare
  user entries. Inventory search Back/Exit/save-results and selection-analysis
  callbacks; preserve feature-owned controls instead of relocating them.
  Verify V07.
- [ ] Preserve restrictions for retained Location Map, Explorer cohorts,
  ranked search and duplicate-variant visits, not just visible surfaces.
  Retain Picture-frame/variant exclusions, source-group capability and no-op
  behavior. Reject unavailable commands before yielding idle selection.
  Verify V07.
- [ ] Search setup retains its captured collection generation and reference,
  and also reevaluates current admission before starting. Changed identity or
  newly blocked interaction cannot start/replay/retarget the query. Duplicate
  preparation retains its own request lifetime. Verify V07 with held work.
- [ ] Preserve existing search scope/origin restoration, pending delivery,
  cancellation and producer shutdown. A policy gate on a new command does not
  block already-owned completion or source reconciliation. Verify V07.
- [ ] Menus expose policy-derived availability and actual attempts preserve
  feedback. Remove duplicated cross-feature predicates, retaining source/feature
  validity. Record every route, local-control exclusion, red/green and inspections.

**Verification V07:**

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^(TestCommandAdmissionVisits|TestCommandAdmissionAsync|TestFindMoreLikeThis.*|TestSort.*|TestHideDuplicates.*|TestBrowseDuplicates.*)$'`

`go test -tags no_emoji,nodynamic -count=1 -v ./internal/ui -run '^TestCommandAdmissionRoutes$/^search$'`

Do not infer symmetric transitions: existing search-to-Location Map entry and
Location-visit-to-search refusal intentionally differ.
