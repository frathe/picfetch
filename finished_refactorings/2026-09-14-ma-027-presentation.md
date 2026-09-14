# MA-027 — Single-image presentation ownership

Route: Deep. Status: complete; qualified at e6024dc. Owner: Pico / T0.
Baseline: `f54998c`. Branch: `docs/ui-refactoring-backlog`.
Source: [accepted spec](../.scratch/ma-027/spec.md), [seven approved tickets](../.scratch/ma-027/issues/README.md), [execution gates](../.scratch/ma-027/ticket-execution.md).

Deliver one display owner for surface publication, captures, rotation/fades,
GIF playback, SVG sharpening, requested-image loading and bounded preloads.
Root retains collection/cohort choice, image-action admission and file work,
window/zoom policy, chrome and shared cache construction/invalidation.
The eight interview decisions are settled; the existing public display seam
and root integration harness are the approved TDD seams.
No dependency, user-visible string, codec, comparison architecture or command
admission redesign is planned. Existing dependency/license closure is unchanged.

## Interface and ownership

`internal/ui/display.Feature` is constructed with `New(Config) *Feature`.
`Surface() *canvas.Image` gives zoom its existing geometry contract; display
alone assigns pixels, visibility and translucency. Root may retain the surface
reference for composition and observation during migration, never publication.
`Snapshot() Snapshot` exposes requested/displayed `Identity` (source and revision),
loading, oriented logical size, vector/animation facts, duration, file facts and
rotation, with no mutable frame slice. `RotateBy(int)` / `ResetRotation() bool`,
`FadeTo(float32, float32, time.Duration)` / `ResetFade()`, `Clear()` and `Stop()`
are presentation intents. Clear permits reopening; Stop closes admission.

`Capture() (Capture, bool)` retains the published raster, identity, rotation and
request revision, without copying raster storage. `ReconcileSaved(Capture) bool`
updates the baseline only for the same presentation and request revision; it
does not replace published pixels. `CaptureStable() (Capture, func(), bool)`
adds a generation/acquisition-bound idempotent release. Capture also retains
logical vector content and its rasterizer for root's Copy Selection adapter.

`Load(Request)` owns one navigation, including internal cache retries and
caller-selected broken-source retries. Request supplies `Source fyne.URI` and
`Transition bool`. `Config.Callbacks` supplies `Probed(image.Rectangle)`,
`Presented(Snapshot) []fyne.URI`, `Failed(fyne.URI, error) fyne.URI`,
`AnimationTruncated(fyne.URI)`, `Requested(Identity)` and `Repaint()`. Display installs coherent
state/pixels, calls Presented synchronously on UI, rechecks validity, then admits
animation/preloads and completes the request. Nil failure replacement exhausts
the request. Root callbacks may clear or reenter Load; no old activation follows.

Config injects the root-owned `*imaging.ByteCache[*imaging.LoadedImage]`, one
`UIQueue` (`Do(func())`, `Drain() bool`), and instance timing/rasterizer seams.
`RequestVectorRender(scale float32, toPixels func(fyne.Position) (int, int))`
records layout intent and schedules work without synchronous widget mutations.
Load, animation and vector work retain distinct internal cancellation scopes.
`LoadDone() completion.Handle`, `AnimationDone() completion.Handle` and
`AppliedFrames() uint64` distinguish completion/application. `Settle()` joins
finite load/vector/preload work and drains delivery until stable; it never
waits for active continuous playback. `Wait()` joins all generations after
Clear/Stop, without requiring a UI callback to release workers. Tests stop
continuous playback before teardown settlement.

## Task graph and file map

01 -> 02 -> {03,04} -> 05 -> 06 -> 07. The siblings remain logically independent;
the lead executes shared root composition sequentially to avoid overlapping edits.
Each task follows one failing behavior test, minimal implementation, focused
preservation checks, then lead review. Exact commands are in the linked execution
gates; new child names below are required, not vacuous regex selections.

### 01 — Surface, rotation and fades
Owner: T0 inline. Depends: none. Budget: 0 implementation spawns, 2 review rounds.
Create display `feature.go`, `contract_test.go`; modify display `display.go`,
root `build.go`, `viewer.go`, `load.go`, `rotate.go`, plus Qodana exclusions.
Contract: New, Surface, Snapshot, RotateBy, ResetRotation, FadeTo, Clear/Stop.
Temporary `Present(fyne.URI, *imaging.LoadedImage, bool)` installs decoded
records until 05 makes loading internal. Temporary State/frame adapters serve
root GIF/SVG paths until 03/04; a publication adapter disappears with those paths.
Test `TestPresentationContract/surface`: surface is in the composed tree,
pixels/orientation and clear visibility; immutable requested/displayed identities
including reopen; fade reset restores opacity. Existing zoom/window/fade root
checks retain composition behavior. Verify: T01 execution gate.

### 02 — Captures and Save
Owner: T0 inline. Depends: 01. Budget: 0 spawns, 2 review rounds.
Create display `capture.go`; modify `contract_test.go`, root `clipboard.go`,
`export.go`, `save.go`, `filework.go`, `exportwork_test.go`, UI shard manifest.
Contract: Capture and ReconcileSaved; temporary root request revision bridge
is removed by 05. Stable capture's vector/animation paths remain adapters until
03/04, with operation-entry raster capture already fixed here.
Test `TestPresentationContract/captures`: captured content remains bound across
rotation/new requests/reopen, save baseline preserves later turns/reset and
does not republish pixels. Add `TestExportCapturesPixelsBeforeChooserReturns`
at root with a held native chooser, validating pixels captured before rotation.
Existing Save/clipboard/file-write regressions retain disk and alias semantics.
Verify: T02 execution gate.

### 03 — GIF and stable capture
Owner: T0 inline. Depends: 02. Budget: 0 spawns, 2 review rounds.
Move root animation worker and pause into display `animation.go` / `pause.go`;
modify display feature/capture/contracts, root `load.go`, `copyselection.go`,
`viewer.go`, `build.go`, `harness_test.go`, `animate_test.go` and affected
Copy Selection tests. Migrate local worker tests to display; retain root input,
navigation and mode integration tests. Remove root pause/frame publication adapters.
Test `TestPresentationContract/animation`: one outstanding callback; buffered
ack before next delay; cancel without UI drain; stale frame rejection; rotation
on later frames; stable capture failed/double/stale release; original loop resumes
with fresh delay. Observe all superseded generations, separately from frame apply.
Verify: T03 execution gate.

### 04 — SVG
Owner: T0 inline. Depends: 02. Budget: 0 spawns, 2 review rounds.
Move `vector.go` worker/policy to display; root retains only zoom/density forwarding.
Modify display feature/capture/contracts, root rotation/load/copy-selection glue,
harness and vector tests. Remove root vector state and frame replacement adapter.
Test `TestPresentationContract/vector`: logical/aspect/device-pixel/cap behavior,
debounce/hysteresis, immutable cache frame slice, stale and failed rendering,
logical region capture versus published-raster Export; layout delivery is async.
Verify: T04 execution gate.

### 05 — Loading and recovery
Owner: T0 inline. Depends: 03,04. Budget: 0 spawns, 2 review rounds.
Create display `load.go` and internal lifecycle support; modify root `load.go`,
`viewer.go`, `build.go`, `filework.go`, `save.go`, `run.go`, harness/load tests.
Root retains ShowImage admission and callback adapters, early/final/rotation/zoom
window distinctions, cohort removal/replacement, title/info/chrome/EXIF behavior.
Remove root load lifecycle, public Present and request-revision bridge.
Test `TestPresentationContract/load`: synchronous cache hits; outgoing identity;
slow/stale and cache-invalidated decode; retry chain single terminal completion;
callback reentry on success/failure; probe validity; no descendants before handoff.
Root tests retain broken cohorts, errors/warnings, Save cancellation and resize.
Verify: T05 execution gate.

### 06 — Preloads
Owner: T0 inline. Depends: 05. Budget: 0 spawns, 2 review rounds.
Move pool/claims/worker into display `preload.go`; root `load.go` only selects
neighbor URIs. Modify construction/viewer/harness/cache tests. Remove last preload
adapter, keep shared cache wiring and comparison's independent loader.
Test `TestPresentationContract/preloads`: two-worker bound, claims, one budget
for half-budget probe/decode, no promotion/eviction, canceled/invalidated writers,
foreground oversized image admission and complete shared records. LoadDone means
preloads admitted; finite settlement means their work completed.
Verify: T06 execution gate.

### 07 — Final qualification
Owner: T0 inline. Depends: 06. Budget: 0 spawns, 2 review rounds, one full suite.
Remove all remaining scaffolding; update `ARCHITECTURE.md`, `AGENTS.md` ownership
and queue rules, `todos.md`, `needs_refactoring.md`, tracker/evidence, Qodana exact
test exclusions and UI shard assignments. Retain test mappings after moves.
Review all 37 ticket criteria / 68 stories against current command evidence and
structural ownership, using code-review's Standards and Spec axes inline (the
repo's explicit lead-only review rule overrides skill delegation). Baseline is
the pinned starting commit, including the working diff before the authorized commit.
Verify T07 gate: `make check-test-shards`, `make verify`, GoLand inspection of every
changed code file including weak warnings. No clean gate claim without execution.
Move this plan to finished_refactorings only after acceptance gates are complete.

## Environment and execution ledger

2026-09-14: host Go1.27.1 darwin/arm64 with cgo/clang. Docker desktop/default both
linux/aarch64; no native amd64 context is configured. Full `make verify` cannot
pass its architecture prerequisite here. Run focused host race tests and
`make verify-build`; retain final gate failure as unverified, never weaken isolation.
GoLand connected; use build_project and get_file_problems(errorsOnly:false),
check timeouts, fix confirmed issues and rerun. Focused application race tests have run; results follow.

| Task | Spawns budget/actual | Review rounds | Full suite | Evidence |
|------|----------------------|---------------|------------|----------|
| recon | 1 / 1 | — | no | Independent read-only environment/IDE discovery; no code delegation |
| 01–06 | 0 / 0 each | 1 per slice, then final pass | no | focused gates passed |
| 07 | 0 / 0 | 2 | attempted | prerequisite rejected ARM daemon; native suite unverified |

Lead checklist: framing through implementation/review complete. Native final gate remains open; the implementation handoff records that limit.
User invoked implement, which explicitly authorizes committing this scope on the
current branch. Preserve unrelated `.codex/config.toml`; no push requested.

### Evidence — 01 / start of 02

Surface red: contract failed with "presented content has no source identity";
second identity red: "pending navigation lost the distinction". Both now pass.
`go test -race ./internal/ui/display ./internal/ui/zoom` passed (display 1.868s,
zoom cached). T01 focused root command passed (`internal/ui`, 6.546s).
Lead reviewed ownership: root frame adapters remain named for 03/04; surface
assignment/visibility/fades moved. BeginRequest/CancelRequest are explicit
temporary request bridges removed by 05. Captures red failed in both later-turn
and later-reset cases with "capture did not retain the published content".

### Evidence — 02 / 03 / start of 04

T02 display contract passed (`internal/ui/display`, 1.847s); focused Save,
clipboard, Export and held-chooser tests passed (`internal/ui`, 10.616s).
T03 red: "playback did not schedule its next delay". Public animation contract
now passes (`internal/ui/display`, 1.578s), and GIF/navigation/Copy Selection/fade
integrations passed (`internal/ui`, 11.902s). Root no longer owns animation
workers, frame timing/delivery, counters or pause. Completion uses existing
`completion.Handle` rather than exposing raw channels; the harness distinguishes
AnimationDone, AppliedFrames and Wait. PauseObserved preserves the existing
worker-observation barrier through the public interface.

Moved coverage: root TestAnimationQueuedFramePacing,
TestAnimationCancellationDiscardsQueuedFrame and TestAnimationQueuedPauseKeepsCapturedFrame
are superseded by TestPresentationContract/animation (held queue, acknowledgement,
rotation, cancellation, stale callback and acquisition release). The root
private-state failed-capture test is moving to the public capture contract.
T04 red: device-pixel rotated raster remained (260,520), expected (1040,2080).
The new contract now passes (`internal/ui/display`, 2.480s). Root integration
migration is in progress. All new test files have exact Qodana exclusions.

### Final contract and design corrections

All temporary production adapters are removed: there is no display.State,
public frame setter, Present, BeginRequest, root playback/pause/SVG worker or
root preload pool. The external test fixture's Present helper seeds the injected
cache and calls the final Load interface; it is not a production adapter.
CaptureStable and its observations live with Capture; Wait/Settle live with the
feature lifecycle. Root's img reference is composition/observation only.

Repository inspection disproved one premise in the accepted cache decision:
AddIfFits rejects an individually oversized value but can evict existing values.
The explicit non-eviction requirement takes precedence for display speculation.
A structured clarification timed out without an answer; no approval was inferred.
The implementation assumption was stated to Ronin: add CacheWriter.AddIfRoom,
with generation, existing-key and remaining-budget checks under one cache lock.
Other AddIfFits callers retain their existing behavior. Foreground Add, shared
cache construction, budgets, half-budget probe gate and two-worker bound remain.
This adds internal/imaging/bytecache.go to the file map, without dependencies.

The final review also corrected two observable identity/timing edges. Root's
published-file lookup and Save admission now use display's published source,
including after a pending different source is cancelled. EXIF continues rejecting
requests during loading. The EXIF held-reader fixture now publishes its controlled
URI through a cached Load rather than silently replacing collection identity.
Stable capture notifies the existing GIF loop on pause/release, retiring its timer;
queued callbacks validate the acquisition as well as the animation generation.
A brief capture therefore resumes with a fresh delay even before the loop observes
its pause, and an already queued frame cannot escape after release.

### Evidence — remaining slices and review fixes

- T04 root SVG/zoom/Copy Selection checks passed (12.066s). Device-pixel target,
  hysteresis tables, debounce/cancellation, soft failure, stale delivery and cache
  frame-slice isolation now live in display. Root retains composition/geometry.
- T05 red: cache hit did not finish synchronously through handoff. Green: display
  1.570s, expanded root 16.049s. Cache-generation/probe, retired native reader and
  failure-reentry contract passed (1.660s). Broken Explorer cohort recovery passed
  (3.575s). One LoadDone handle covers all retries and root handoff/work admission.
- T06 red: neighbor preparation was not admitted. First green: root 8.430s,
  display 3.656s. Review then found Wait omitted retired preloads; the regression
  failed with "Wait forgot a retired speculative worker" and passed after Wait
  joined the pool. The non-eviction contract failed against AddIfFits and passed
  with AddIfRoom. Existing cache-writer tests also pass.
- Terminal admission red: a stopped owner admitted a fade. FadeTo now rejects it.
- Published-source red: outgoing pixels were labeled with the requested source.
  The root source-identity regression passes after reading Snapshot.Displayed.
- Fresh-delay red: a brief capture resumed the old deadline. The acquisition
  change notification and queued-delivery guard pass the deterministic test and
  the root animated Copy Selection test (display 2.850s; root 2.755s).
- Negative verification of held-chooser Export: temporarily using live pixels at
  write time failed with exported (16,8), expected captured 8x16. Restored.
- Negative verification of callback reentry: temporarily removing the post-handoff
  validity check failed because the obsolete handoff restarted the newer GIF.
  Restored. The public contract now observes that playback generation's survival.

### Final available qualification — 2026-09-14

- Final focused root race command passed (80.554s): navigation/load/retry/cache,
  published identity, Save eligibility and writes, Export/held chooser, clipboard,
  Copy Selection, EXIF, SVG, rotation, zoom/window and fade integration families.
  Exact command is retained in the local ticket execution record.
- Final display/imaging/zoom race command passed (3.325s / 1.724s / 2.503s):
  TestPresentationContract, ByteCache/CacheWriter, vector, rotation/reset and
  zoom/geometry families. All six display contract children are present.
- make verify-build passed: formatting and generated TUF/vector/app-asset/notice
  checks, root go vet and root go build with the Makefile's no_emoji tag.
- make check-test-shards passed: 681 root runnables across three shards. Exact
  Qodana test exclusions also passed their Makefile validator.
- make verify attempted once at the final gate and rejected linux/aarch64 at
  check-test-platform. The full Linux/amd64 race and matching golden suite did
  not run. No isolation policy, golden pixels or platform gate was relaxed.
- GoLand inspected all 52 changed code files with errorsOnly:false. Fixed import
  formatting, misplaced package comments, struct padding, an unused test helper
  and a builtin-shadowing test variable; all corrected files were reinspected.
  Remaining duplicate test-scaffolding findings were reviewed in copyselection,
  deletion, grid, imgcache, openfiles, slideshow and step tests. Each file already
  has an exact DuplicatedCode exclusion in qodana.yaml, as required by the repo's
  readable-test convention. IDE and CI profiles differ; these are documented
  exclusions, not unexplained ignored findings. No inspection timed out.
- GoLand build reported success with limited build-diagnostic collection;
  actual compilation is also established by Makefile build and race tests.

### Lead review against f54998c

Standards: ownership and construction order are explicit; the shared cache stays
root-owned; platform seams and root action admission are preserved. Display's
queue is installed in the production-equivalent harness, with cancellation before
Wait/Settle. No source TODOs, new dependencies or user-visible strings were added.
Qodana exact test exclusions and root shard inventory accompany the moved tests.

Spec: the six public contract children and retained root integrations cover the
seven ticket slices. Captures retain immutable published rasters without another
full image copy; saved-baseline reconciliation does not republish. Cache hits
remain synchronous; callbacks recheck validity before descendants. Load, GIF and
SVG scopes are distinct, while teardown joins all retired generations. Root
retains broken-file/cohort/neighbor choice, zoom/window/chrome and file workers.
No confirmed code finding remained after the fixes above. At this local stage,
AC9/T07 stayed open solely for native Linux/amd64 qualification. The GitHub
review-loop evidence below closes that gate before this plan is archived.

### Reproduce focused qualification

```sh
go test -race -timeout 60s ./internal/ui/display ./internal/imaging ./internal/ui/zoom -run '^(TestPresentationContract|TestByteCache|TestCacheWriter|TestVector|TestRotate|TestReset|TestZoom|TestGeometry)'
go test -race -timeout 5m ./internal/ui -run '^(TestViewerShow_|TestShowImage_|TestDisplayedFileUsesPublishedSource$|TestAttemptLoad_|TestInvalidateLoad_|TestFinishLoad_|TestPreloadOne_|TestImageCacheWriters_|TestCompareFirstNavigation_|TestCanSaveRotation_|TestSaveChanges|TestExport|TestClipboardCaptures|TestClipboardQueued|TestCopySelection|TestExif|TestEXIF|TestSVG|TestVector|TestRotatedNonSquareSVG|TestRotatingAZoomedSVG|TestInfoOverlayReportsLogical|TestCloseFilesClearsVector|TestRotateBy_|TestResetRotation_|TestShow_ResetsZoom|TestStaticWindowSize_Load|TestSyncWindowToZoom_GridVisible|TestTogglePictureFrameMode_ExitResetsFade|TestHandleKeyEvent_EscapeResetsFade|TestReset_ResetsFade|TestRemoveFile_PurgesCache|TestAppState_RemoveFileEvicts|TestClearToDropzone_Purges)'
make verify-build
make check-qodana-test-exclusions
make check-test-shards
make verify
```

Final documentation check resolved all 142 local file links in changed standing,
plan and tracker records. Root production code has no surface pixel/opacity writes
or obsolete frame/timer/SVG/preload adapters; retained image reads serve composition
and existing command availability. Local issue records remain under the repository's
ignored .scratch tracker; this commit includes the standing docs and active plan.
The unrelated .codex/config.toml change is excluded from the implementation commit.

## GitHub Codex review loop — 2026-09-14

Ronin authorized pushes, fix commits, review replies/resolution and fresh Codex
reviews on existing draft PR [24](https://github.com/frathe/picfetch/pull/24).
The complete suite runs in GitHub's native workers; local checks remain focused.
One additional read-only scout inventoried workflow/report locations. The lead
owns all finding validation and fixes. No merge/release is authorized.

Initial pushed head: 7273deb. [CI run 34849048536](https://github.com/frathe/picfetch/actions/runs/34849048536)
passed validation, Windows, both macOS guards, non-UI race and UI shard 2.
Shard 1 found a migrated test waiting on a deliberately held preload; shards 1/3
also reported three golden mismatches. These are qualification findings, not a
clean native pass. Qodana's downloaded post-suppression qodana.sarif.json contains
zero results; CodeQL Go/Actions analyses for merge 65bd7a3 contain zero results.

The unresolved older Codex thread on search_cache.go is confirmed: a changed
source between ranking and cold Favorite profiling could produce a successful
mixed-snapshot evidence set. The real-worker regression in
TestProductionSearchEvaluation/source_changes_after_ranking failed before the
fix, then passed when cold InputSHA256 was checked against ranking SourceSHA256.
The same test proves an unchanged snapshot still produces both passes and a
changed snapshot leaves no completed Favorite profile. Commands/results:

- go test -race -tags explorertrial ./scripts/explorereval -run '^TestProductionSearchEvaluation$' -count=1: red 13.048s; green 12.629s.
- go test -race ./scripts/explorereval -run '^TestSearch': green 9.823s.
- GoLand inspected both changed evaluator files, including weak warnings: clean.

TestGridBrowseDuringAnalysis/open_variant reproduced the CI deadlock locally.
Its cached load completes synchronously; only LoadDone should be observed while
the fixture deliberately holds a neighbor reader for progressive Grid analysis.
The migration incorrectly substituted full waitUntilLoaded/Settle. Restoring
that foreground-only observation passes the complete TestGridBrowseDuringAnalysis
family under -race (9.277s), retaining the held-source scenario. GoLand findings
are the previously documented duplicate fixtures with exact Qodana exclusions.

Golden qualification then ran through make golden (Linux/amd64 rendering).
The differences were inspected and explained below; no masters were changed.

### Rendering diagnosis and round 1 outcome

Head 1fb550c fixed the older source-snapshot finding and Grid wait. Native CI
34850281070 completed all tests with only the same three screenshot failures.
Codex code review reported no major issues and security review reported no
security issues for that head; all threads were resolved. Qodana and CodeQL
reports again contained zero results (CodeQL merge 84e0d08).

make golden reproduced the three mismatches under Linux/amd64. Pixel comparison
found one changed pixel in dropzone_hover and six in each bad-drop image, each
at a maximum difference of one 8-bit color level, confined to artwork. Text,
geometry and border positions matched. An isolated worktree at the previous PR
head 84650fb passed its original masters in the same rendering environment.

The intervening mascot pointer overlay painted a transparent rectangle above
that artwork. Software compositing still rounded a few underlying colors.
Replacing that renderer with an empty container preserves pointer hit testing
without painting. The unchanged golden suite then passed via make golden; no
master images were replaced. Pointer, mascot-circle and Restore-link race tests
passed (6.103s). GoLand inspection of trane.go is clean. This is a one-file
rendering correction to an earlier change on this PR, not a display API change.

The rendering fix required fresh native CI and code/security review on its own
pushed head before final qualification. The following evidence covers that head.

### Native qualification of e6024dc

[CI run 34851844070](https://github.com/frathe/picfetch/actions/runs/34851844070)
passed all four Linux/amd64 race partitions, validation and both macOS native
guard jobs. This includes the original golden masters and held-source Grid
regression. Windows attempt 1 failed the unchanged picker transport guard at
20.02s, consistent with its 20-second PowerShell deadline. The single-job retry
passed the same guard in 6.83s and the complete Windows job. No timeout or guard
was relaxed. The workflow's final conclusion is success.

[Qodana run 34851844147](https://github.com/frathe/picfetch/actions/runs/34851844147)
has zero results in its downloaded post-suppression qodana.sarif.json.
[CodeQL run 34851844195](https://github.com/frathe/picfetch/actions/runs/34851844195)
has zero Go/Actions results and no analysis errors or warnings. Its merge
b5fad75 has e6024dc as the PR parent and 54fd7c3 as the base; GitHub regenerated
the merge preview as c560920 with the same parents during qualification.

The additional manual review of 1fb550c also finished with no code/security
findings. A single fresh [review request for e6024dc](https://github.com/frathe/picfetch/pull/24#issuecomment-5665198356)
was posted after that round finished. Its [code review](https://github.com/frathe/picfetch/pull/24#issuecomment-5665235067)
and [security review](https://github.com/frathe/picfetch/pull/24#issuecomment-5665305648)
completed without findings on e6024dc. Every review thread is resolved, including
the older evaluator finding with its [fix and verification reply](https://github.com/frathe/picfetch/pull/24#discussion_r4005739823).
The isolated baseline worktree was removed after the rendering comparison;
golden masters remain unchanged.

### Acceptance and handoff

MA-027 is accepted at e6024dc: AC1–AC9 and all seven tickets are complete.
The plan is archived, the local tracker is resolved and the open-work records
point to this evidence. The implementation commit is 7273deb; review fixes are
1fb550c (ranking/Favorite snapshot and Grid observation) and e6024dc (non-painting
pointer overlay). No dependencies or license obligations changed.

The documentation-only acceptance commit must receive its own fresh CI and
Codex code/security reviews before the PR loop finishes. Final head results
remain on [PR 24](https://github.com/frathe/picfetch/pull/24); this record pins
the code qualification above without claiming future checks. No merge or
release is part of this handoff. The unrelated local Codex configuration is
excluded from every task commit.
