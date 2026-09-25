# Location Map release qualification

Ronin approved closing the release gaps, with Pico launching a probeable client
and Ronin loading the images and performing the interactions. This is a Deep
continuation of the existing Location Map plan, not authorization to commit,
push, open a PR, change OS permissions, or release.

## Contract and seams

Retain the approved viewer action/render, controlled source I/O, tile HTTP,
Favorite storage, and native trial seams. Add missing composite acceptance
coverage through those existing interfaces, one vertical red/green slice at a
time. Tests of already-working behavior must be negatively verified. Do not
infer native paint latency from UI callbacks, automated approval from a human
trial, or complete acceptance from passing parent test names.

Ronin controls the chosen image collection and clicks. Launch an isolated trial
with source-free app state and local process observations; do not automate input
or screen capture via the existing CGEvent helper without explicit approval.
Manual observations supplement rather than silently replace the agreed 10k
latency measurements. Private library artifacts stay outside the repository.

## Tasks and verification

1. T0: Fill composite viewer regression gaps in `internal/ui/locationmap_test.go`:
   duplicate representative/progress, frozen Grid visits and hidden tile work,
   partial scan/cache reuse, repeated occurrences and source-version races,
   external hidden-donor validation, read-only/input limits and final empty state.
   Command: `go test -race -tags no_emoji,nodynamic -count=1 -run
   '^TestLocationMap$' ./internal/ui`; also affected feature packages.
2. T0: Prepare the existing native trial launch for human-controlled loading and
   clicking, bind observations to the binary digest, and retain local state/RSS
   evidence. Verify admitted count, completed stages, return/reopen behavior and
   graceful recorder shutdown. Preserve gaps honestly until measured.
3. T0: Update release notes and supported ticket checkboxes; inspect every changed
   Go file, run format/vet/build, manifests, qualification tool tests and one full
   `make verify` attempt. Native Linux/amd64 remains an external gate if absent.
4. Ronin: Exercise the launched native client, report visual/interaction results
   and final 30k verdict. No inferred approval, no premature plan archival.

## Routing and delegation gate

Use `.scratch/location-map/model-routing.md`; T0 retains all review and fixes.
One T3 scout (Luna/medium), read-only: trace native runner launch/report contracts
in `scripts/locationmapqualify/{main,runner,protocol}.go`. Shell located the
entry points but does not establish their coupled command/cleanup protocol.
G1: standalone prompt <=25 lines; G2: factual claims verified against named
locators and `go test ./scripts/locationmapqualify`; G3: three files, no writes;
G4/G5: runner context independent from lead's viewer tests. Rule S: control-flow
facts require comprehension; W: no implementation is prewritten or delegated.
Budget: one scout, zero implementation spawns unless a new bounded contract is
recorded; no delegated reviews; one final full-gate attempt.

### Human-controlled observer increment

Recon establishes that `run` requires helper-generated input and screen evidence;
relabeling it as a manual run would be misleading. Add a separate `manual` CLI
under the existing tool, accepting `-binary FILE -evidence NEW_DIR [-timeout 30m]`.
It launches only `--location-map-trial NEW_DIR/native --max-files=100000` (no image
paths), samples process RSS and source-free state, and waits for user exit or
deadline. Retain binary SHA-256, PID, timestamped RSS/state observations, console,
final state and errors in private files. Use `manual-report.json`, never the
formal `report.json`; no latency claim or human verdict is generated. Preserve
failed/interrupted runs and reject existing evidence directories. No screen,
input helper, network endpoint or new application-side goroutine.

Owner T1 Sol/high, only `scripts/locationmapqualify/main.go`, new `manual.go` and
new `manual_test.go`; root owns documentation/Qodana exclusion/integration and
all review/fixes. Existing process start/stop and memory sampler may be reused.
Test the CLI/runner boundary using a controlled subprocess/per-call process
factory, never the desktop: missing/bad flags, isolated no-path argv, graceful
exit with retained source-free state/RSS, interruption/deadline, failed launch,
existing directory refusal, and rejection by the formal checker.
Oracle: `go test -race -count=1 ./scripts/locationmapqualify`.
G1 <=25-line standalone prompt; G2 exact command; G3 three disjoint files;
G4 independent tooling implementation while T0 owns viewer tests; G5 new
human-control requirement, not a review finding or prewritten implementation.
S/W: process lifecycle requires implementation, not a mechanical transform.
Budget increased explicitly to one bounded implementation plus one scout.

## Evidence

Baseline worktree clean at `9521edb`. No commit/push/release performed.

## Added release blocker: Grid light-startup to dark switch

Ronin reports that Grid retains light regions after starting in Light and
switching to Dark. T0 adds a rendered viewer regression at the existing approved
startup/Settings/Grid/canvas seam, including both initial appearances, live
switches, search chrome and reopening. Diagnose the reproducible pixel failure
before changing production code. Shell points to Grid's opaque backdrop; no
additional scout is justified for that single located constructor. Fix only
the confirmed owning surface, inspect changed files and rerun shared theme tests.

`TestGridThemeSwitch` reproduced the exact partial-color failure with Light
startup: empty body white `{255 255 255 255}` while the resolved Dark background
and edge regions were `{23 23 24 255}`. Dark startup reproduced the inverse.
An explicit overlay Refresh still failed, ruling out a missing refresh alone.
The backdrop snapshots `theme.Color` in its constructor. Reusing the existing
`widgets.NewThemedRectangle` fixes that owning rectangle; the test does not force
an extra Refresh. Both startup appearances, repeated toggles, search and reopen
are covered. No pixel/color cache, app-wide refresh hook or new worker is added.
Grid production inspection is clean. The existing Grid test file has only three
pre-existing duplicate-fixture weak warnings, outside the new test (current
lines 729/776/869); its exact Qodana test exclusion already applies.

## Added native finding: duplicate preparation feedback

The human-controlled client admitted 50,672 files (not an exact 10k/30k gate).
Ronin reports the long `Checking duplicate groups...` wait needs a progress bar.
T0 owns this coupled preparation/UI increment. Use existing viewer/controlled
read seams; require mounted progress during a held hash, actual checked-work
advancement, indeterminate final grouping, and removal on completion/cancel with
no late resurrection. Apply to both Location Map and Explorer's identical wait.

Grid will expose a UI-side `DuplicatePreparationProgress() (completed,total int)`
snapshot and a separate progress observer, delivered by existing throttled hash
completions (not native-menu rebuilds, polling, new workers or source scans).
Track admitted hash work per existing hash-engine session. Feature-owned ordinary
and indeterminate Fyne progress bars alternate: determinate for pending hashes,
indeterminate when only final grouping/validation is outstanding; both hide/stop
on cancellation or transition to the real feature. No new user-facing text.
Root alone joins Grid observations to whichever preparation continuation is live.
Tests: `TestLocationMap/preparation_and_counts/duplicate_preparation`, new held
progress subcase, `TestVisualSimilarityExplorer/pending_duplicates`, plus Grid's
controlled work tests. Lead retains all design, implementation, review and fixes.
No further delegation: the coupled lifecycle is already lead-held context.

Manual session launched before this finding: PID 37635, evidence
`/private/tmp/picfetch-location-manual.V4RjLP/session`, exec session 62103, one-hour
deadline. It contains the Grid theme fix but not the new progress UI. Preserve
that run separately and relaunch an updated client with Ronin when ready.

## Added interaction: directional photo/cluster selection

Ronin clarified that arrows should jump to the nearest image/cluster and draw
an Explorer-style border, not pan by fixed pixels. T0 owns viewer/render tests,
selection identity/lifetime, themed border, Enter opening the exact current
photo/cluster and camera exposure. Preserve +/- zoom, 0 Fit All, pointer drag,
ordinary image/Grid arrow navigation and Escape return. First arrow picks the
nearest cluster to the viewport center; later arrows pick the Euclidean-nearest
cluster in the requested open half-plane, with stable input-order ties. An edge
with no candidate retains selection. Include offscreen targets at current zoom;
move the camera only enough to expose the chosen card/count. Selection follows
an occurrence through reclustering and image/Grid return, but resets on close.
Use the already-approved viewer action/render and pure geography seams.
Ronin subsequently requested Shift+arrows for panning without changing the
current selection. T0 owns that small modifier-routing increment; the focused
`keyboard_navigation_shift_pan` guard first failed with no movement for
Shift+Right. Plain-arrow selection and Enter remain unchanged.
Oracle: `go test -race -tags no_emoji,nodynamic -count=1 -run
'^TestLocationMap$/keyboard_navigation' ./internal/ui`.

T1 Sol/high owns only `internal/ui/locationmap/geography.go` and its existing
`geography_test.go`: `NavigationClusters` shares Clusters' geometry/grouping but
does not viewport-cull; `NearestCluster(clusters []Cluster, selected int,
width, height float64, dx, dy int) int` returns the nearest index as above,
preserving an existing index at a directional edge and -1 for no candidates.
Only cardinal unit directions are admitted. Exclude nonfinite positions and
empty members. Test all directions, center initialization, ties, empty/invalid
inputs, offscreen groups and wrapped dateline geometry before implementation.
Oracle: `go test -tags no_emoji,nodynamic -count=1 -run
'Test(NavigationClusters|NearestCluster|Clusters)' ./internal/ui/locationmap`.
G1 <=25-line prompt; G2 exact command; G3 two disjoint files; G4 bounded geometry
while T0 integrates UI/lifetimes; G5 new implementation, not a review fix. S/W:
behavioral geometry work, no prewritten implementation. Budget increased by one
bounded implementer for this new user request; all review/fixes remain T0.

## Verification and handoff evidence

- `go test -race -tags no_emoji,nodynamic -count=1 -run
  '^(TestLocationMap|TestGridThemeSwitch|TestVisualSimilarityExplorer)$'
  ./internal/ui`: PASS, 186.302s. Later additional keyboard reclustering test:
  focused keyboard race pass, 4.052s.
- `go test -race -tags no_emoji,nodynamic -count=1
  ./internal/ui/locationmap ./internal/ui/widgets ./internal/ui/grid
  ./internal/ui/explorer ./scripts/locationmapqualify`: all PASS
  (1.641s / 5.052s / 11.876s / 4.877s / 3.354s respectively).
- `make fmt-check check-test-shards check-qodana-test-exclusions`: PASS;
  Linux/amd64 shard inventory validates all 706 root UI runnables in three shards.
- `make vet build`: PASS. Native linker's duplicate `-lobjc` warning only.
- `go test -tags no_emoji,nodynamic -count=1 -run
  TestManualHasNoUnicodeArrows ./internal/ui/help`: PASS, 0.382s.
- `make verify`: BLOCKED before the full suite by `linux/aarch64` Docker daemon;
  repository requires native Linux/amd64. No isolation-test bypass, remote CI
  launch, commit or push was performed. Earlier IDE build-tag limitation on
  unchanged imaging/AVIF policy remains distinct from these changed-file checks.
- GoLand all-findings inspection: all 22 changed/new Go files inspected. Only
  three pre-existing Grid test duplicate-fragment weak warnings (729/776/869),
  outside this change and covered by that test file's existing exact Qodana
  exclusion; no new diagnostics. Inspect final edits again at handoff.

Confirmed red guards include: initial Light/Dark Grid pixels; mounted preparation
and incremental progress; preparation animation retirement; directional keyboard
border/Enter; offscreen camera exposure; selection loss after zoom/reclustering;
cluster-return camera retention; cancellation/completed-fact reuse; hidden donor
return validation; old-version metadata and preview publication; displayed hidden
duplicate provenance; all fallback accounting classes; erroneous failure caching.
Each deliberate violation was restored. See test names above and the owning
ticket commands; parent-test success alone is not treated as absent coverage.

Ticket audit advanced 37/64 to 45/64: two duplicate-fallback points, two cluster
visit/lifecycle points, three live-cache/source-version points, one external
return point. Remaining composite gaps are still unchecked: repeated-occurrence
raw-fact identity; metadata limits/XMP/read-only composite; unchanged-entry/return
metadata reuse composite; rebuilt-thumbnail/camera composite; competing committed
writes with replacement; mutation-to-final-empty/read-only composite. These are
verification gaps, not newly observed production failures.

Ronin's practical performance verdict and source-free observations are retained
in ticket 13, bound to SHA-256
`6c8ddb19810b8ad08f403bc25503dea4cfe710ca84576cb89f98a2d6308fe626`.
50,672 images; cold preparation 403.190224s, GPS 140.132114s; 2,792 RSS samples,
peak 4,164,026,368 bytes; clean observer exit. Concurrent developer tests mean
this is not an isolated benchmark. Exact 10k latency, repeated warm/reopen and
storage qualification remain unperformed. New keyboard/progress controls were
added after that run and must not inherit its human verdict retroactively.

| Task | Spawns budget/actual | Lead review | Full suite |
| --- | --- | --- | --- |
| Native runner facts | 1/1 T3 Luna medium | Locator verification | No |
| Manual runner increment | 1/1 T1 Sol high | Tests and inline lifecycle/test fixes | No |
| Keyboard geometry increment | 1/1 T1 Sol high | Independent package/race output, code inspection | No |
| Viewer integration/progress/themes | 0/0 T0 | Inline red/green and negative guards | Focused only |
| Final gate | 0/0 T0 | Format, shard, exclusions, vet/build, GoLand | Attempt blocked by daemon architecture |

Final Shift+arrow increment: full `TestLocationMap` race run PASS, 50.076s;
all keyboard subcases PASS with race, 4.448s. `make fmt-check vet build` passed
again. Final edited viewer/action/surface/tests and manual-runner files inspected
again with GoLand, all findings included, no diagnostics. Desktop Fyne source
confirms bare Shift+arrows use typed-key dispatch (not CustomShortcut).

Updated isolated native client launched for Ronin: PID 72953, runner exec session
1010, evidence `/private/tmp/picfetch-location-keyboard.JLGkqj/session`, one-hour
timeout, no input paths or automated interactions. SHA-256:
`fc6784bde0591cb01e4949b2b588e451f8d34285b6aa23476c8df82300c9e797`.
This build adds keyboard selection/Shift-pan and duplicate progress. Ronin's
native smoke verdict: "looks good". The observer reports 441 admitted images;
record this as acceptance of the updated client, not another large-scale run or
an inferred checklist of individual actions. No repeat large benchmark is
demanded. Close the client to finalize source-free observations. Formal latency,
native Linux/amd64 verification and the listed composite coverage gaps stay open.
