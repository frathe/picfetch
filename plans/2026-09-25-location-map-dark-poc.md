# Location Map local dark-filter POC

Status: Ronin accepted the POC's appearance; shared EXIF filtering and light-mode
panel fixes are implemented and locally verified. No commit or app relaunch.
Native/full-suite qualification limits remain recorded below.
Route: Standard. Deliverable: a testable local dark filter in the existing
Location Map, retaining the OSM provider, plus accurate OpenStreetMap privacy text.

## Scope and decisions

- Follow the app theme; light mode retains original map pixels. Filter only
  basemap tiles, never photo previews, metadata or image files.
- Keep standard OSM URLs, request policy, caches and stale-scene drag behavior.
  Theme switches must not require new network requests or lose painted tiles.
- No new provider, credentials, dependency, persistent preference or layout.
- Clearly name the filter as a POC; Ronin decides whether its colors/readability
  are good enough. The prototype skill's web route/variant bar is inapplicable
  to this native single-filter request; existing theme selection is the switch.
  Explicit TDD instruction takes precedence over the skill's no-tests shortcut.
- Reuse the previously approved viewer/render and HTTP seams documented in
  `2026-09-25-location-map.md`; no new test seam needs approval.

## Acceptance / verification

1. Light tiles remain original; dark tiles become dark with contrasting labels;
   photographs remain unchanged. Live theme switching restores original pixels
   without HTTP demand, including an in-flight replacement scene.
   `go test -tags no_emoji,nodynamic -count=1 -run
   '^TestLocationMap/dark_filter$' ./internal/ui`
2. Tile continuity and policy regressions remain green.
   `go test -race -tags no_emoji,nodynamic -count=1 -run
   '^TestLocationMap/(dark_filter|tile_recovery|tile_policy_and_bounds|photo_presentation)$'
   ./internal/ui`
3. Privacy explains both opt-in map entry points, information disclosed, local
   processing/caching and avoiding map requests. Verify against traced code:
   `git diff --check` plus lead source/paragraph assessment (not a legal audit).
4. Runnable native POC: `make build`; inspect rendered synthetic light/dark
   captures. Human cartographic quality and 30k performance remain unverified
   until Ronin tests. Full gate: one `make verify` attempt, and GoLand inspection
   of every changed Go file. No commits or app relaunch without instruction.

## Tasks and ownership

1. T0: one failing viewer/render test, minimal local filter and theme integration
   in `internal/ui/locationmap`, then next behavior slice. Existing root test file.
   Budget 0 spawns; no suite during red/green.
2. T3: read-only privacy fact trace across the collection map and EXIF map.
   T0 retains wording, review and all changes. Budget 1 scout, Luna/medium.
3. T0: update PRIVACY.md, package map if adding a file, todos, verification and
   evidence here. Native `make build` supplies `bin/picfetch` for testing.

Task 2 is independent of task 1; task 3 follows both.
Scout gate: shell finds separate tile implementations but cannot establish their
complete lifecycle/data disclosure from matches alone. G1: <=25-line standalone
prompt; G2: lead validates returned locators with `rg`/targeted reads; G3: no
writes; G4/G5: separate EXIF/lifecycle context not held by the lead. S: semantic
control-flow trace, not regex rewriting. W: no delegated implementation/review.

## Ledger and evidence

| Task | Spawns budget/actual | Reviews | Full suite |
| --- | --- | --- | --- |
| Privacy fact trace | 1/1 | lead validated source locators | no |
| POC + tests + docs | 0/0 | 2; shared fixture extracted after IDE warning | no |
| Final gate | 0/0 | lead | one attempt; Docker platform prerequisite blocked |

### Delivered and verification evidence

- The localized `dark_poc.go` filter reverses luminance, softens chroma and adds
  a charcoal tint. It wraps original immutable tile images, adding no second
  pixel cache, provider, dependency or background worker. Color conversion is
  performed when the renderer samples a tile; native drag cost is not measured.
  Live theme refresh updates only mounted map tiles. Completed replacement
  scenes recheck the theme, preserving the existing all-tiles-ready swap.
- Three red/green slices: original white map failed the dark-background check;
  theme switching failed to restore white; a partly loaded replacement retained
  old-theme black tiles instead of light labels. Each failure was observed and
  then passed after its respective implementation change. The theme test uses
  Fyne's adaptive theme, not the test driver's fixed dark theme.
- `go test -race -tags no_emoji,nodynamic -count=1 -run '^TestLocationMap$'
  ./internal/ui` passed in 42.339s. After the shared-fixture cleanup, focused race
  cases `dark_filter|tile_recovery|tile_policy_and_bounds|photo_presentation`
  passed in 10.259s. No live OSM calls, source images or private-library captures
  were needed. Synthetic light/dark captures were inspected at
  `/private/tmp/picfetch-dark-poc.30VdRa/{dark,light}-filter.png`: map contrast
  changes while the framed photo pixels are unchanged. These fixtures do not
  qualify real map label/road readability or the native GL renderer.
- `make fmt-check vet build` passed and rebuilt `bin/picfetch` for this POC.
  All four changed Go files were inspected with GoLand, including weak warnings.
  The duplicate retry-fixture finding was fixed inline; reinspection was clean.
  No top-level UI test or new test file was added, so shard and exact Qodana
  exclusion entries did not change.
- `make verify` was attempted once and stopped at its platform prerequisite:
  the selected Docker daemon is `linux/aarch64`, not native `linux/amd64`.
  No isolation tests were skipped and no worker policy was weakened.
- Lead validated the scout's request/cancellation/storage facts against
  `locationmap.go`, `locationmap/{feature,tiles,favorites}.go` and
  `exifwin/{exifwin,tiles}.go`. PRIVACY.md now covers both explicit map entry
  points, tile-coordinate/area disclosure, local Favorite metadata records,
  the local filter, and how to avoid/cancel map requests. This is implementation
  documentation, not a legal compliance audit.

### Try it

Run `./bin/picfetch` (or `make run`), load GPS-tagged photos, choose Dark in
Settings' appearance selector, then open Window -> Location Map. Switch to
Light to compare, and pan/zoom while judging road/label readability and drag
smoothness. Photos should retain their normal colors. EXIF's mini-map is outside
this POC. Keep this experiment uncommitted pending Ronin's verdict; any later
prototype archival/commit requires authorization.

## Accepted filter / EXIF and light-theme follow-up

Ronin tested the POC and said it looks great, accepted keeping it, and requested
the same filter in EXIF. A supplied native screenshot shows dark toolbar/footer
surfaces under light-mode controls in Location Map; he reports the same issue
on Similarity Explorer panels. This supersedes the EXIF non-goal above.

Route promoted to Deep for shared presentation work across feature packages.
No provider, network policy, image editing or new UI controls are authorized.
Use the existing viewer action/render/HTTP and EXIF window seams. T0 owns design,
all implementation and review; one additional T3 Luna/medium scout is allowed
only for Explorer's themed-surface control-flow facts. Shell found constructor
and refresh matches but not all panel ownership/refresh paths. G1 <=25-line
prompt; G2 lead verifies locators and adds rendered regression tests; G3 read-only;
G4/G5 separate Explorer presentation context; no review or implementation sent.

Tasks: reproduce light-mode failure in the viewer's rendered header/footer;
trace hypotheses, then replace stale theme snapshots at owning presentation
seams; reuse the accepted immutable map-pixel transform in both map surfaces;
test EXIF live theme changes with offline tiles; update package map/privacy/todos
and run focused race tests, all-file GoLand inspections and one final gate.
Contract/file map and exact tests will be recorded after the reproduction and
EXIF adapter reconnaissance. Native screenshot/30k input is not copied into the
repository. Prototype archival/commits still require separate authorization.

### Follow-up contracts and file map

1. **Theme surfaces, T0.** `widgets.NewThemedRectangle(fyne.ThemeColorName)`
   returns a rectangle whose fill resolves at paint time, so normal Fyne theme
   refresh cannot leave an old color snapshot. Own the helper in existing
   `widgets/style.go`; use it for Location Map's outer background/tooltip and
   Explorer's outer background. Tests stay in existing viewer test files and
   sample the real header/footer/sidebar after Dark -> Light -> Dark switches.
   Verify `go test -tags no_emoji,nodynamic -run
   '^Test(LocationMap|VisualSimilarityExplorer)/(dark_filter|theme_switch)$'
   ./internal/ui`. No new strings/controls. Fixed high-contrast photo frames and
   white-on-dark pile captions are intentional and not the panel bug.
2. **Shared map pixels and EXIF adapter, T0.** Move the accepted filter from
   `locationmap/dark_poc.go` to `mapstyle/pixels.go`, exported only as
   `mapstyle.ForTheme(image.Image) image.Image`. Collection call sites retain
   their timing. `exifwin/map.go` extends the pinned Fyne-X Map, wrapping only
   the map raster generator with the shared filter, not map controls/markers.
   Keep its existing transport, caches, controls, navigation and UI queue.
   Test through EXIF's real expanded window, synthetic local HTTP tiles and
   captured pixels in existing `exifwin_test.go`. Verify
   `go test -tags no_emoji,nodynamic -run '^TestLocationMapTheme$'
   ./internal/ui/exifwin`, plus the complete EXIF package's race regressions.
3. **Docs/gate, T0.** Update ARCHITECTURE.md for the shared package/adapter,
   PRIVACY.md for both local filters, todos and this evidence. Repeat changed
   tests and focused feature regressions, all changed Go files' IDE inspections,
   format/vet/build and one verification-gate attempt for this follow-up.

Dependencies: theme surfaces and shared-map adapter can be implemented
sequentially; docs/gate follows both. One read-only scout budget/actual 1/1;
implementation spawns 0/0 because the lead owns the coupled renderer context.
No new dependency: Fyne and Fyne-X versions/licenses/notices remain unchanged.

Diagnosis evidence: a light header rendered `{23 23 24 255}` rather than
`{255 255 255 255}`, including after an explicit overlay Refresh. The current
theme lookup was correct. Constructor-time `theme.Color` snapshots in ordinary
canvas rectangles, not missing theme selection or missing refresh delivery,
explain the failure. No profiler or history bisection is needed for this
deterministic color-state defect.

### Follow-up evidence / handoff

- Location Map toolbar/footer and Explorer toolbar/sidebar failures reproduced
  in captured pixels at the approved viewer seam, then passed after the shared
  themed rectangle. Deliberately restoring the old constructor-time color
  snapshot made all four rendered checks fail again; restoring the fix returned
  them to green. The new EXIF test failed on unfiltered original `{1 2 3 255}`
  pixels in dark mode, then passed with the shared transform and repeated
  Light/Dark changes without new tile downloads.
- Correct focused selector:
  `go test -tags no_emoji,nodynamic -count=1 -run
  '^Test(LocationMap|VisualSimilarityExplorer)/(dark_filter|theme_switch)$'
  ./internal/ui` passed in 1.123s. An earlier malformed combined selector ran
  no tests; its output was rejected, the selector corrected and tests rerun.
- Viewer integration race run passed in 164.130s:
  `go test -race -tags no_emoji,nodynamic -count=1 -run
  '^Test(LocationMap|VisualSimilarityExplorer|ShowExifWindow.*|ExifWindow.*|ExifNavigation.*|HandleKeyEvent_EOpensExifWindow|ExifLink_.*|ShutdownStopsExifAdmission)$'
  ./internal/ui`.
- Complete affected-package race runs passed: EXIF 8.073s, Location Map 1.575s,
  Explorer 4.356s, shared widgets 4.867s. `make fmt-check vet build` passed and
  rebuilt `bin/picfetch`. `git diff --check` passed. No new root top-level test
  or `_test.go` file was added; EXIF's new test stays in its existing package file.
  `make check-test-shards check-qodana-test-exclusions` passed: 705 UI runnables,
  three shards, existing exact test-file exclusions valid.
- All 11 changed Go files were inspected by GoLand with weak warnings included.
  New/changed code was clean. Six pre-existing duplicate-setup warnings remain
  in unchanged EXIF tests at current lines 678/718/742, 1044/1079 and 1368.
  `git diff --unified=0` confirms none belongs to the new test. These independent
  fixture setups use the repository's existing exact-file DuplicatedCode
  exclusion in `qodana.yaml:253`; no broad suppression or unrelated refactor was
  added. Do not describe the entire EXIF file as warning-free. The final restored
  themed helper was reinspected and clean.
- Inspected the regenerated synthetic Location Map light screenshot:
  `/private/tmp/picfetch-dark-poc.30VdRa/light-filter.png`. Header/footer now
  match light-mode controls and labels; photo pixels remain unchanged. EXIF and
  Explorer captured-pixel checks are software-renderer evidence, not a new
  native performance or GL qualification claim. Ronin's supplied private-library
  screenshot was inspected only, not copied into the repo.
- Follow-up `make verify` again stopped at the native Linux/amd64 prerequisite
  because the selected Docker daemon is Linux/aarch64. No tests skipped to
  manufacture a clean full gate. Native app has not been relaunched by Pico.
- Fyne v2.8.0 and Fyne-X
  v0.0.0-20260712112324-6989f2f174fb remain pinned and unchanged. Both BSD-3-Clause
  notices are already shipped in THIRD-PARTY-NOTICES.md. The adapter retains OSM
  attribution, request identity and cache behavior; no new dependency or asset.
- Follow-up scout budget/actual 1/1, read-only Luna/medium. Lead independently
  checked `explorer/map.go` constructor/refresh paths and its viewer fixture,
  owned the rendering decisions, implementation, review and fixes.

To test this follow-up, restart `./bin/picfetch`, expand EXIF's Location section,
and switch Settings appearance between Light and Dark. Check both Location Map
and Similarity Explorer chrome. The shared filter is now accepted code in
`internal/ui/mapstyle/pixels.go`, not a second experimental provider or a second
copy of the formula.
