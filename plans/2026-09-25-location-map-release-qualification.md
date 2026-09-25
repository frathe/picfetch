# Location Map release qualification

Ronin approved closing the release gaps, with Pico launching a probeable client
and Ronin loading the images and performing the interactions. This is a Deep
continuation of the existing Location Map plan, not authorization to commit,
push, open a PR, change OS permissions, or release. The subsequent authorization
below supersedes that initial publishing restriction, not the release restriction.

## 2026-09-25 maintainer acceptance and PR review loop

Ronin explicitly closed the performance gate: "3 mark as done ... it is running
butter smooth!". Performance qualification is **done by maintainer acceptance**
of the recorded 50,672-image run and later 441-image updated-client smoke test.
Exact 10k latency/30k-count protocols, input-to-render samples and repeated
warm/reopen measurements are waived for this release, not represented as run or
passed. The formal runner/checker stays strict; no fabricated report is created.

Ronin assigned complete verification to the GitHub PR and authorized push,
PR creation and the GitHub Codex review loop. Preserve his existing `d337e10`
commit; commit the acceptance record, push `feature/image-map`, create a PR against
the repository default branch, and follow code/security reviews plus CI,
Qodana post-suppression SARIF and CodeQL. T0 validates/fixes findings with focused
regressions and GoLand inspection, replies/resolves threads, then obtains a fresh
clean review on the latest commit. No full local race rerun, merge or release.
Existing composite coverage gaps are not silently waived by performance approval.

Review skill adaptation: use standards/spec axes, but the explicit repository
workflow overrides that skill's parallel-reviewer default and fixed-point prompt.
The PR merge-base is the comparison point; all assessment and fixes remain T0.
One optional T3 Luna/medium scout may inspect three read-only historical PR API
resources (summary/reviews, comments, checks) to identify Codex code/security
completion formats and review trigger. G1 <=25-line prompt; G2 exact `gh pr view`
and `gh api` locators independently verifiable; G3 no writes and three resources;
G4 independent API evidence while T0 publishes; G5 remote review-report format is
new context. S/W: bounded report-format interpretation, no code or verdicts.
Budget: at most one scout, no delegated reviews/fixes, no local full-suite run.

### Qodana subscription ended

Ronin authorized disabling Qodana CI after the trial subscription expired.
Preserve its workflow/configuration for later restoration; do not weaken CodeQL,
tests or local GoLand inspection requirements. The Qodana gate is explicitly
waived while disabled, not reported as a passing scan. Verify the live workflow
state with `gh workflow view qodana_code_quality.yml` and retain the decision here.

One additional T3 Luna/medium scout researches only official GoLand documentation
for local GUI/command-line inspection and profile support. G1 <=25 lines;
G2 primary-source URLs and quoted supported commands independently checked by T0;
G3 read-only, no local files; G4/G5 unfamiliar local inspection capabilities while
T0 handles CI; S/W requires documentation interpretation, no delegated review.
Budget increase: one scout for the newly requested licensing/local-tool question.
The research skill's writing step stays with T0 to preserve T3 read-only routing.

### PR 58, first qualification round

Opened [PR 58](https://github.com/frathe/picfetch/pull/58) from `feature/image-map`
at `f2a2e02`; merge base `48ec832a6c9c1e351981b0f3797eee56bc873203`.
CI run `36152139675` passed non-UI/ui-3 races, Windows tests and Linux/macOS
native guards. ui-1/ui-2 exposed old seven-entry Window-menu assertions and a
Save Changes test that joined map invalidation outside its synctest bubble.
Both reproduced locally; update the expected map entry/shortcut and settle the
newly admitted worker inside the bubble. No production save behavior changed.

Codex security completed without findings on that head. Code review produced:

- Update isolation: confirmed. Automatic stale-stage removal, preference changes,
  manual check/apply and shutdown apply must all reject Location Map trials.
  `TestLocationMap/native_trial_update_isolation` failed for all those effects
  with stubbed update I/O; the guard fix passes under the race detector.
- Progress totals: rejected for the reported new-session scenario. `restartWork`
  already allocates a new hash engine on close/reopen and generation changes.
  `TestDuplicatePreparationProgressResetsAcrossSessions` proves fresh accounting;
  deliberately carrying the old total into the new engine made it fail at 2/2
  instead of 0/0. Restoring production code passes; no counter reset added.
- Map preloads: confirmed. The cluster navigation test now uses nonadjacent
  collection indexes and failed when an unlocated neighbor was preloaded.
  Preload candidates now use the same cohort/location order as navigation,
  retaining the active-empty-search behavior.

The changed Favorite version test closes ticket 08's first criterion: a new
viewer rejects the old no-GPS disk record after the image gains coordinates,
without changing source bytes. The version-check mutation failed as expected;
restored race test passes. Ticket audit is now 50/64 (46 verified, 4 accepted).

Ronin requested end-user release prose and Trane artwork. Updated `todos.md`,
canonical and bundled release notes; the latter two match byte-for-byte.
The commit-pinned GitHub raw artwork URL returned HTTP 200, `image/png`, 589369
bytes. Release-note/manual guard tests pass. No artwork was generated or edited.

GoLand inspected all seven changed Go files, including weak warnings. Only two
pre-existing duplicate test-scaffolding fragments remain in `menu_test.go`
(534/610), already covered by that file's exact `qodana.yaml` exclusion. Other
changed files are clear. Qodana workflow state is verified `disabled_manually`
after Ronin's explicit instruction; CodeQL passed with no open PR alerts.
Fresh latest-head review/CI remains required after pushing these fixes.

Focused native race run covering all `TestLocationMap`, menu, Save Changes
responsiveness, Find More Like This and automatic/manual update regressions
passed (`internal/ui`, 90.354s). Grid session-counter regression passed (1.310s),
including its deliberate negative verification. `make fmt-check`, exact Qodana
test-exclusion validation and release-note synchronization checks passed.

### Agent workflow documentation follow-up

Ronin requested the new local Qodana usage in `AGENTS.md`'s GitHub review loop.
The Writing for Agents skill keeps one local-analysis procedure there, with the
existing research guide as its setup/troubleshooting reference. The procedure
prefers IDE-local Qodana with existing configuration and uploads off; when that
cannot run, it accepts documented GoLand inspections of every changed code file,
including weak warnings, with no claim of identical profile coverage.

This session used that fallback: desktop-control startup failed before any
GoLand dialog interaction, while GoLand's inspection tools completed all seven
changed Go files listed in the preceding round. The only remaining findings
were the two previously excluded test-duplication fragments in `menu_test.go`.
Tool: GoLand `get_file_problems(errorsOnly=false)`, no timeouts; the tool does
not expose the active profile's name, so no `qodana.starter` parity is claimed.
File scope: `autoupdate.go`, `run.go`, `load.go`, `locationmap_test.go`,
`menu_test.go`, `savework_test.go` under `internal/ui`, plus
`internal/ui/grid/dupes_test.go`.
The inspected code is pushed as `fe04238459eea9c2df965773d1c086e8eb353d8e`;
this follow-up changes documentation only. `make vet` and `make build` also
passed before that push. All three review threads have evidence-backed replies
and are resolved; fresh reviews started automatically on that head.

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
demanded. Close the client to finalize source-free observations. At this handoff,
formal latency, native Linux/amd64 verification and composite coverage stayed open;
the subsequent maintainer decision above closes performance by acceptance and
assigns complete verification to the PR. Composite coverage is not waived.
