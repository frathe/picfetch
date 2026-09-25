# Location Map: 30k feedback polish

Status: implemented and locally verified; native/full-suite qualification remains
open as detailed below. Awaiting Ronin's acceptance; prior worktree was clean.
Route: Standard, extending the existing Location Map implementation plan.
Authority: Ronin's follow-up after the 30k trial. This supersedes the earlier
filename-caption/sidebar interaction; it is not a new performance verdict.

## Scope and acceptance

Use the already-approved viewer action/render and controlled HTTP seams.
No new dependency, provider, credentials or source-image mutation is assumed.

1. Dragging retains the prior painted tile scene until every replacement tile
   is ready. Partial results and failures do not expose a checkerboard in place
   of an already loaded map. Preserve request cancellation, retry and bounds.
   Ronin's follow-up explicitly requires moving that retained scene with the
   camera, not freezing its screen coordinates; painted photos also survive
   repeated drag events without an asynchronous reload gap.
   Verify: `go test -tags no_emoji,nodynamic -run
   '^TestLocationMap/(tile_recovery|photo_presentation)$' ./internal/ui`.
2. Singleton photos have a slight frame/shadow, no filename caption, a filename
   hover tooltip, and open their image directly. Remove the right sidebar.
3. Clusters show one representative preview above the count bubble and retain
   exact frozen cluster/Grid navigation. Preview work uses the shared pipeline.
   Both the preview and count open the exact cluster in Grid (Ronin's final
   clarification); only singleton photos open their image directly.
   Verify 2-3: `go test -tags no_emoji,nodynamic -run
   '^TestLocationMap/(photo_presentation|photo_direct_open|cluster_visit|direct_image_visit|camera_retention|selection_feedback)$' ./internal/ui`.
4. Verify whether the current OSM raster service has an official dark variant.
   Use it if supported; do not silently adopt a third-party provider or fake a
   provider feature. Record the verified limit and seek direction if necessary.

Tasks 1-3: T0 inline, existing locationmap surface/viewport/feature/work files,
root locationmap acceptance tests and manuals; fixtures/strings only as needed.
Red-green vertically, then lead review/fixes. Verify focused `TestLocationMap`
cases and the feature package with required build tags; race after integration.
One final `make verify` attempt, GoLand on changed code, locale/manual guards,
shard/exclusion checks if applicable. No commit/push authorized.

## Read-only OSM scout admission

T3 Luna/medium, fresh bounded context; one scout for this new feedback phase.
The repository only records the standard raster endpoint and policy; dark-style
availability requires primary API/policy reading, not repository text search.
G1: <=25-line standalone factual prompt. G2: lead opens cited official sources
and verifies endpoint/style and policy claims. G3: zero writes. G4/G5: external
service capability is independent context not held by the lead. S: semantic
service/policy facts, not a deterministic transform. W: no implementation given.
Research skill guides primary sourcing; routing's read-only scout rule takes
precedence, so lead records returned findings here. No UI/review delegated.

## Ledger

| Task | Spawns budget/actual | Reviews | Full suite |
| --- | --- | --- | --- |
| OSM capability scout | 1/1 | lead verified official policy | no |
| Tile presentation and photo/cluster interaction | 0/0 | lead, including follow-up and rendered-tooltip fix | no |
| Final gate | 0/0 | lead | one attempt; platform prerequisite blocked |

## Evidence

Red/green: the original drag test saw 6 painted tiles become 0; direct click
opened the obsolete sidebar. New photo tests failed for missing hover/cluster
preview. Ronin's follow-up then exposed stationary retained tiles and discarded
photo pixels on each drag; tests reproduced both before the correction.
Atomic retry and obsolete-delivery tests both fail when their guards are
deliberately disabled, and pass restored. No sleeps or live OSM calls in tests.
Diagnosis stopped after the reproducible eager-clear cause; no profiler or
history bisection was needed for this rendering-state defect.

Existing source-version-matched photo pixels now transfer to rebuilt cards on
UI; fully warm rearrangements start no preview worker. Tile geometry follows
the same camera transform while a detached replacement scene completes.
Each current painted/replacement scene is capped at 128 references; retired
requests are cancelled and their callbacks reject obsolete revisions. Encoded/
decoded store budgets are unchanged. Newly exposed regions without previously loaded
tiles can still show the offline background. Close/hide releases painted state.

OSM capability: the lead verified the scout's primary sources on 2026-09-25.
The current [standard raster endpoint](https://operations.osmfoundation.org/policies/tiles/)
has no documented dark-style selector. The separate
[vector service](https://operations.osmfoundation.org/policies/vector/) requires
a renderer/style integration, not a raster URL toggle. Provider unchanged; no
third-party service, license or dependency added. A dark basemap is not claimed.
UI frame/tooltips still use app-theme colors where appropriate.

## Final local evidence

- Full `TestLocationMap` race run passed in 35.393s: 320 retained JSON events,
  78 test passes including all 18 original parent names, no failures/skips.
  `.scratch/location-map/evidence/14-polish-tests.json` is the final run after
  both drag-continuity and tooltip-layout corrections. This is headless host
  evidence, not native screen/performance qualification or proof of missing
  proposed child scenarios from the original tickets.
- `go test -race -tags no_emoji,nodynamic -count=1
  ./internal/ui/locationmap ./scripts/locationmapqualify ./internal/locationtrial`
  passed. `make location-map-qualification-test` also passed.
- `make fmt`, final `make fmt-check vet build`, `make check-test-shards` (705
  runnables/3 shards), Qodana exact exclusions, manual/translation tests and
  `git diff --check` passed. `bin/picfetch` rebuilt; no commit or app relaunch.
- GoLand inspected all five changed Go files with `errorsOnly:false`, no
  findings. Surface/test inspection was repeated after the screenshot-detected
  tooltip fix. Results: `.scratch/location-map/evidence/14-polish-goland.json`.
  Its build endpoint reports success but limited diagnostics; the tagged Make
  build is the actual build evidence.
- Inspected headless screenshots for framed singleton/cluster photos and the
  hover tooltip. The first tooltip screenshot showed only an ellipsis; added a
  failing measured-width assertion, fixed label measurement/wrapping, re-rendered
  and confirmed the full filename. Images are disposable synthetic fixtures,
  not private-library captures, under `/private/tmp/picfetch-map-polish.Vc00us`.
- `make verify` attempted once: stops because selected Docker daemon reports
  `linux/aarch64`, not required native `linux/amd64`. No worker-isolation tests
  skipped or policy weakened. Native smoke/10k evidence, complete original
  composite-scenario coverage and a build-bound post-fix 30k verdict remain open.
- Audited all 13 local tickets, checked 37/64 supported complete criteria,
  corrected completed items' obsolete test selectors, and explained missing
  portions of unchecked composite criteria. Updated README/spec addendum,
  original plan and todos. Ronin's informal 30k report is recorded verbatim;
  no formal performance result or acceptance verdict fabricated.

The routing agreement keeps these coupled rendering/review changes with T0;
the one permitted T3 OSM scout used Luna/medium. All six earlier planned T1
implementation delegates had already delivered. No additional implementation
delegation, peer review, dependency or provider change was introduced.

## Final cluster-click clarification

Ronin requests identical cluster-preview/count behavior: a preview above
"22 images" opens exactly those 22 in Grid, not its one representative image.
Use the already-approved viewer action/Grid seam; change only the surface
callback and its root acceptance tests, then synchronize current docs/ticket 05.
T0 inline, zero spawns: this two-code-file change uses hot context and the
routing plan retains cluster/UI integration with the lead.
Verify: `go test -tags no_emoji,nodynamic -count=1 -run
'^TestLocationMap/(cluster_visit|photo_presentation|photo_direct_open)$' ./internal/ui`.
Add a 22-member fixture plus an unrelated located photo, compare both click
targets' exact Grid membership, and retain singleton direct-open coverage.
No new strings, dependencies, test files or top-level tests.

Completed: preview and count share the same captured cluster-membership action;
singleton previews retain direct image opening. The 22-member-plus-outsider test
failed before the change with `cluster click did not open Grid (preview=true)`;
the adapted presentation test also failed as expected. Both pass after the fix.
Full `TestLocationMap` race regression passed in 35.271s. `make fmt-check vet
build`, manual tests and diff whitespace checks pass; `bin/picfetch` rebuilt.
GoLand reports no findings in the two changed code files (all severities), with
its limited-diagnostics build endpoint also successful. `make verify` attempted
once for this clarification and still stops at the Linux/amd64 prerequisite
because the selected Docker daemon is Linux/aarch64. Docs/spec/ticket 05 now
describe preview and count as the same Grid route. No extra acceptance point is
claimed complete; the broader qualification gaps remain unchanged.
The earlier 14-polish JSON is historical evidence for the pre-clarification
interaction; this rerun is the current interaction evidence.
