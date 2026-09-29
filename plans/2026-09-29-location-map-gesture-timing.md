# Native Location Map gesture timing

Latest continuation: [return-to-macOS handoff](../docs/location-map-macos-handoff-2026-09-29.md)
after completed Linux verification, with transferable failed zoom screenshots
and ordered next checks. Ronin requested committing and pushing this handoff
before switching computers. Historical Linux/macOS evidence below retains the
revision and limits of each run.

Deliverable: identify requested pan/zoom transforms from captured pixels before
accepting native latency samples. Route: Deep, because the capture adapter is
macOS-specific. Branch: `feature/location-map-gesture-timing`.

## Linux verification continuation — 2026-09-29

Ronin confirmed the scope as Linux finalization after the macOS fixes. Local and
remote branch tips agree at `d0998a9ab2cce4c4a7c1ac20b9cc125f1e767f52`.
No application, Swift helper, matcher, acceptance rule or dependency changes
were made in this continuation. The macOS fixes and focused tests are complete;
the recorded native 40-gesture measurement is still incomplete. Linux test
results cannot fill that separate gate or establish a latency timestamp.

Verified on native Linux x86_64, Go 1.27.1:

- `make check-test-platform`: PASS against the native Linux/amd64 Docker daemon.
- `PATH=/snap/go/current/bin:$PATH make location-map-qualification-test`: PASS,
  race packages `scripts/locationmapqualify` 3.236s and
  `internal/locationtrial` 1.033s.
- `make location-map-capture-test`: PASS in the offline Swift 6.2.4 container,
  using the same read-only repository/make mounts and pinned image digest
  recorded below. Both portable transform and response-frame suites passed;
  this does not compile or qualify the macOS adapter.
- Fresh GoLand `get_file_problems(errorsOnly=false)` inspections completed on
  all ten branch-changed Go files: root `load.go`, `locationmap.go`,
  `locationmap_test.go`, `locationtrial.go`; qualifier `evidence.go`,
  `evidence_test.go`, `protocol.go`, `runner.go`, `runner_test.go`; and
  `scripts/testshards/main_test.go`. The first nine had no findings. The last
  retained the four existing weak duplicate-fragment warnings at 90/174 and
  225/275, in unchanged intentional fixture/assertion sequences covered by its
  exact Qodana exclusion. No actionable findings or incomplete scans. This is
  the documented IDE inspection fallback, not a Qodana SARIF result.
- `PATH=/snap/go/current/bin:$PATH make verify`: PASS, exit 0. Format,
  TUF/assets/notices, exact test exclusions, vet, build and the 740-runnable shard
  inventory passed. The complete non-UI partition and all three UI race shards
  passed (611.365s / 402.412s / 445.751s). Root UI outcomes: 738 passed and two
  existing conditional skips: case-alias export requires a case-insensitive
  filesystem; native HEIC image operations require explicit provider
  qualification. No tests, exclusions or worker isolation policies were changed.
  Runner/container exits are both 0; OOM counters/events are zero. Artifacts:
  `.scratch/race-runs/20260929T095440Z-KtahR5/`, including raw JSON streams,
  console, container state and memory evidence.

The Linux final gate is complete for `d0998a9`. These documentation-only changes
carry that unchanged-code evidence. No PR was opened and no new CI, CodeQL,
Qodana SARIF or bot-review result is claimed. The todo remains open for the
native measurement and previously recorded Swift inspection gap; the accepted
maintainer performance verdict remains separate.

An initial temporary replay investigation reproduced rejection of the retained
final zoom image; it did not establish the first changed frame or its timing.
All temporary replay sources/data were removed when the Linux-only scope was
clarified, with tracked source and the original PNG pair unchanged. One initial
verification attempt stopped at formatting those temporary files before any
race suite ran; the complete run above began after their removal.

Continuation budget/actual: one read-only scout, no delegated implementation or
review, one full race suite. Scout G1: bounded capture/render coordinate question;
G2: independently checked source locators; G3: no writes; G4/G5: unfamiliar
capture-to-render relationship; S/W: relationship search rather than an edit.
The lead owns verification and the evidence-only documentation changes.

## Scope and decisions

### macOS continuation after Linux — 2026-09-29

Resume at `5a95ea6`, clean and equal to the fetched feature branch. Deliverable:
diagnose the rejected native zoom, repair a demonstrated matcher defect if one
is established, and collect fresh native smoke evidence. The existing Deep
route and acceptance criteria apply; no release or real-collection claim follows
from the synthetic corpus. The Mac-local 24-image corpus and failed run remain
available. Preserve the complete Linux verification for unchanged Go source.

Budget: one read-only scout for capture/render coordinate relationships, no
delegated fixes or review, no emulated full suite. Delegation gate: G1 bounded
input/viewport/scene relationship; G2 lead checks returned source locators; G3 no
writes; G4/G5 the render path is not yet held by the lead; S/W this is a relation
across code boundaries, not an exact transform. Lead owns the replay, diagnosis,
tests, repair and final evidence.

Ronin then reported a stale first-map surface and requested maximizing when
switching to Map. The running local app bundle was from `0ce849c` (September 26),
before this branch's explicit root repaint. Repackaging `5a95ea6` made ordinary
image -> Shift+L draw the map without resizing; the old bundle is preserved at
`.scratch/PicFetch-before-redraw-check.app`. The installed `/Applications` copy
was not replaced. This is artifact provenance, not a new rendering fix.

Added scope: reuse native work-area maximization on initial Map entry and return
from its image/Grid visits, without entering full-screen. Retain maximization
until normal image sizing/reset consumes it, like Grid/Explorer. Qualification
sessions retain fixed 1200x800 geometry. T0 owns this small root-UI change and
tests; no new dependencies, strings, packages or platform implementation.
Verification: focused `TestLocationMap` and reset regressions, GoLand inspection
of changed Go files, native packaged-app entry/return check. The existing command
harness and per-viewer native-window boundary are the test seams.
No new top-level tests or test files are planned. Native amd64 full verification
will remain pending for new Go source; the earlier Linux result is not relabeled.

Scout budget actual: one scout reused for a second bounded read-only question
after the redraw report (Fyne Show/Hide/invalidation semantics). No additional
spawn or delegated fix/review. Lead checked the reported library source locators.

Map window change verification (working tree based on `5a95ea6`):

- Red/green: `TestLocationMap/maximizes_window` separately failed on initial
  entry, return from a manually restored image window, and failure to restore
  native state for ordinary image sizing; each then passed. Both dynamic and
  static window preferences are covered. The fixed-trial guard was deliberately
  removed: `native_trial_observations` failed on the forbidden native maximize;
  restoring the guard made both focused subtests pass (0.666s).
- `go test -race -tags no_emoji,nodynamic ./internal/ui -run
  '^(TestLocationMap|TestEscapeResetRestoresNativeWindow|TestViewerReset|TestBatchDelete_LeavesTheWindowMaximized.*)$'
  -count=1`: PASS (79.861s).
- `make verify-build check-test-shards`: PASS, including vet, native build,
  notices/assets, exact Qodana exclusions and all 740 runnable shard assignments.
  The first attempt stopped on an unformatted temporary replay utility; it was
  formatted before the successful rerun. No complete race suite ran under ARM
  emulation. Native Linux/amd64 CI remains the full-suite gate for this new code.
- GoLand `get_file_problems(errorsOnly=false)` completed on `build.go`,
  `viewer.go`, `locationmap.go`, `locationmap_test.go`: no findings, including
  weak warnings. Rechecked `locationmap.go` after the negative guard experiment.
- `make package-mac`: PASS. Native image -> Shift+L drew and maximized the map;
  manually restored map -> opened photo -> Escape drew and maximized it again.
  Screenshots: `.scratch/location-map-window-20260929/{entry,return}.jpeg`.
  App binary SHA-256:
  `f4770bc6ca738f508658ba136d330552538da8ba0bde942086831e92de3ef59a`.
  The local bundle's generated build number is 477; the tracked packaging-only
  increment was restored. `/Applications/PicFetch.app` remains unchanged.

Ronin explicitly authorized committing and pushing the window fix on the current
branch, then continuing the existing timing work. The timing matcher changes
remain separate; no fresh CI, CodeQL or Qodana SARIF pass is claimed here.

Window fix committed/pushed as `f1d5e6a`. The native manual guard suite also
passed (`go test -tags no_emoji,nodynamic ./internal/ui/help -run TestManual
-count=1`, 0.449s). No CI run started: this feature branch has no PR and CI is
triggered by `main` pushes or PRs targeting `main`, not ordinary feature pushes.

### Matcher continuation after the window fix

Retained pan repair: the coarse search's 45% coverage gate discarded a real
subpixel peak before refinement. The retained first-frame pair has only 14/48
coarse matches at dx 56, but 33/48 refined matches at dx 58, dy 0, scale 1, with
the required spatial spread and separation. Keep the eight-match seed floor and
twelve-peak bound; apply the existing final 65% coverage, ambiguity, direction
and scale gates unchanged. The new native pan fixture pins this at the portable
observer seam. Restoring the premature gate made the test fail with
`Requested pan 123 was not identified`; removing it made both Swift suites pass.

The first changed zoom frame is now retained as a diagnostic luminance pair,
not merely the previous final timeout PNG. Its old map tiles visibly scale 2x
while photo cards retain their size. The matcher finds the correct transform
at dx -574, dy -394 but only 15/31 eligible high-contrast patches match.
Lowering the contrast floor in scratch replay increases this to 24/40 (60%);
reversing the original registration gives 34/84. Neither passes the unchanged
65% rule. Denser sampling and contrast-relative error limits also failed. All
such experiments remain under `.scratch/location-map-zoom-replay-20260929/`;
none changed production matching or invented an accepted timestamp. Recognizing
this mixed-scale scene requires separating map background from fixed-size cards
or another independently validated registration model. This remains open.

Verification and native evidence:

- `make location-map-capture-test SWIFTC='xcrun swiftc -module-cache-path
  /private/tmp/picfetch-swift-module-cache'`: both suites PASS, including the
  negatively verified retained pan pair and existing rejection cases.
- Native helper compilation with `-parse-as-library -warnings-as-errors -O`:
  PASS. All diagnostic instrumentation was removed from shipped Swift sources.
- `make location-map-qualification-test`: race packages PASS, qualifier 3.497s
  and recorder 1.318s. No Go qualifier/recorder source changed.
- `GOOS=windows GOARCH=amd64 go vet -tags no_emoji,nodynamic ./internal/...`:
  PASS for the shared window-policy change. This is not a native Windows run.
- GoLand `lint_files` on both changed Swift files returned no analyzed items;
  Swift semantic inspection remains unavailable, not a clean inspection gate.
- Diagnostic runs remain at `.scratch/location-map-native-after-linux-diagnostic`,
  `...-after-linux-frames`, and `...-after-linux-zoom-frames`. Instrumentation
  changes workload: their intermediate frame timestamps are not qualification.
- A clean helper/app run at `.scratch/location-map-native-after-window-fix`
  completed cold/warm scans and pixel-verified close, then failed at gesture 0
  because PicFetch lost foreground (`EOF` from the helper). No valid gesture
  latency follows. The failed report and images are retained. App SHA-256
  `57a464dbd8c2886f6817f6bfe598343b1fbbb45fc7a5a5388aba887ee4246c5f`;
  helper `1a3f9f80b31228f020c1ce5e8d94220e216acaf8ceddee864bf2f7660d411603`.
  Raw native logs also repeat the Fyne threading warnings already present in
  prior runs; this continuation does not certify those warnings as resolved.
- After Ronin dedicated the desktop, the clean retry at
  `.scratch/location-map-native-after-window-fix-retry1` passed cold/warm scans,
  pixel-verified close and pan: first identified captured response 73.617292 ms,
  scale 1, dx 58, dy 0, 106/122 patches. Lead inspected the retained pan pair
  against that witness. Foreground and input delivery were no longer blockers.
  The first zoom visibly changed the map but still failed registration after
  three seconds, with `visible_ns=0`; its retained final PNG is not a timestamp.
  This confirms the pan path, not complete native performance qualification.

Lead reviewed the bounded seed-search change; no delegated implementation or
review and no new dependencies. The complete 40-gesture/cancellation measurement,
Swift semantic inspection and real-collection qualification remain unverified.

The current helper refuses gestures because body hashes cannot distinguish input
responses from tile delivery. Keep hashes for stable-frame admission and the
existing closed-viewer exit boundary. Use independent image registration for
gestures, without reading application camera state or using worker timestamps.
No application behavior, dependencies, performance promises or existing maintainer
acceptance changes. Ronin subsequently authorized commits for the macOS handoff;
the feature branch will be pushed for that handoff. No merge/release or native
input automation has been requested.

Use the existing frame-observation and Go protocol/evidence seams documented in
the prior PR58 plan. A portable Swift matcher consumes sampled luminance frames;
the macOS adapter alone owns capture/input. Match distributed textured patches
under translation and the keyboard's 2x/0.5x scales. Reject insufficient texture,
stationary content, wrong direction, ambiguous registration and unrelated repaint.
Preserve the first verified frame's WindowServer timestamp and retained PNG pair.
Persist a versioned transform witness; old boolean-only reports cannot qualify.

Honest limit: textureless/ambiguous maps, changed content with no remaining visual
correspondence, clamped zoom and unstable baselines fail closed. Synthetic matcher
tests do not qualify native performance. This Linux host has no macOS SDK or
native capture permissions; any unavailable native gate remains unverified.

## Acceptance and tasks

1. T0: pure matcher and synthetic frame tests, existing capture policy tests,
   Make/CI wiring. Identify both pan directions and zoom directions; reject
   stationary, repainted, ambiguous, wrong-direction and insufficient frames.
   Verify: `make location-map-capture-test` (portable Swift toolchain); native
   adapter typecheck remains in both macOS CI jobs.
2. T0, depends on 1: connect capture to matcher, retain transform witness and
   first qualifying frame time, preserve stable admission/exit and failed samples.
   Update Go protocol and report validation at their existing seams.
   Verify: `make location-map-qualification-test`; synthetic Swift policy tests.
3. T0, depends on 2: review complete diff, update docs/todos and record evidence.
   Verify: changed-file GoLand inspections (including weak warnings),
   `make verify`, plus the Swift command above. Native SDK compilation and live
   screen/input checks must be reported separately if unavailable.

Graph: 1 -> 2 -> 3. Budget: one read-only scout, no delegated implementation or
review; one final complete suite. Lead owns design, tests, changes and review.
Scout G1: bounded CI/native-host reconnaissance; G2: verify returned locators;
G3: no writes; G4/G5: unread build/native-evidence breadth; S/W: relationship search,
not a deterministic edit or delegated implementation. Scout found both macOS
CI architectures, no local Swift and no retained Location Map screen fixtures.

## Checklist and evidence

- [x] Frame/spec/recon/plan and delegation gate.
- [x] Red/green at the established observation and evidence seams.
- [x] Lead review, negative guards, focused verification and inspections.
- [x] Linux final gate and docs/todo handoff; native qualification pending below.

Initial Go protocol suite passes with `/snap/go/current/bin/go test -count=1
./scripts/locationmapqualify`. The `/snap/bin/go` launcher is unavailable inside
the sandbox; the installed toolchain binary works. Docker reports native
`linux/x86_64`. No shipped dependency or license closure changes are planned.

| Task | Spawns budget/actual | Review rounds | Full suite |
| --- | --- | --- | --- |
| Recon | 1 / 1 | Lead locator check | no |
| Implementation | 0 / 0 | 1 + prerequisite repair | no |
| Final gate | 0 / 0 | passed locally; macOS pending | 2 (first exposed missing Git) |

## Implementation and verification

`native/transform.swift` searches translations and 2x/0.5x scales independently
of the requested input. It requires at least eight textured patches, 65% match
coverage, distribution across both axes and separation from competing transforms.
Capture uses averaged 4x4 luminance blocks; stable admission and exit still use
the existing body hash. The first identified captured frame supplies its original
WindowServer timestamp. Timeouts retain the failed observation and PNG pair.
Schema 2 requires `patch-grid-v1` scale/translation/count witnesses. Collection
also checks the witness against the requested key and rejects wrong response kinds.

Red/green evidence:

- Swift's initial empty matcher failed `Requested pan 123 was not identified`.
  Positive pan/zoom fixtures subsequently passed in both directions. Negative
  fixtures cover stationary, wrong direction/scale, unrelated repaint, one
  arriving tile, repeated patterns, insufficient texture and malformed frames;
  partial tile occlusion still accepts coherent pan. Deliberately removing the
  pan direction/zero-motion guard failed `Accepted stationary as pan`; restoring
  it returned both portable suites to green.
- `TestCheckReportRejectsInvalidTransformEvidence` failed all nine invalid-witness
  cases before report validation; all passed afterwards. The focused `^TestCheck`
  suite passed. `TestNativeProtocolRejectsMismatchedTransform` failed all five
  cases before collection validation, then passed.
- `PATH=/snap/go/current/bin:$PATH make location-map-qualification-test`: PASS,
  `scripts/locationmapqualify` 3.262s, `internal/locationtrial` 1.038s, both race.
- Both Swift suites pass with warnings as errors, using the Make target's compile
  commands in an offline container (the official image lacks Make). Toolchain:
  Swift 6.2.4, x86_64 Linux; image `swift:6.2`, digest
  `sha256:9bea530093ffff8cf6c259991715ee843fe4d0f932e612f7f4b79cca6e00db87`.
  Output: `Native Location Map visual transform tests passed` and
  `Native Location Map response-frame policy passed`. The AppKit adapter is
  excluded on Linux and remains unverified against the macOS SDK.
- The actual `make location-map-capture-test` target also passed after mounting
  `/usr/bin/make` read-only at `/usr/local/bin/make` in that same offline Swift
  container, with the repository mounted read-only at `/work`.

No new shipped dependencies or assets. Swift/Git changes are development tooling;
the macOS helper uses the existing system frameworks and repository-owned matcher.

### Full-suite prerequisite repair

The first `make verify` passed format/assets/notices/vet/build and shard inventory,
then found the existing MA-033 provenance tests require Git in the Docker runner.
Evidence: `.scratch/race-runs/20260929T063542Z-MK5T29/console.log` and raw JSON.
`TestLaunchPolicyNativeSuite/evidence_metadata` reproduced in 0.007s with
`exec: "git": executable file not found in $PATH`. Installing Git exposed the
second environment boundary: Git rejected `/work` as differently owned.
Exact-path process-scoped Git configuration fixed that; the same regression
passed under race in 1.086s. No application/provenance test was weakened.

The diagnosis workflow used the explicit missing-executable error and tight
single-test reproduction; speculative hypothesis/instrumentation phases were
unnecessary for that direct environment failure. Existing Docker setup tests
were extended first and failed for missing Git. Race, ordinary test and coverage
containers now install Git and export only `safe.directory=/work` through Git's
process-scoped config. Host/global Git configuration is untouched.
This repair adds `scripts/testshards/docker-race.sh` and its existing test file
to scope. A second full verification is justified by this gate failure; it is
the only planned broad-suite repeat. Sandboxed host testshards tests separately
hit read-only Go cache/VCS access; container verification supplies that evidence.

Final `PATH=/snap/go/current/bin:$PATH make verify`: PASS, exit 0. Fresh artifacts:
`.scratch/race-runs/20260929T064642Z-tWQBoB/`. Both runner/container exit codes
are 0, with zero OOM events. All non-UI packages passed, including nativeguards,
testshards and locationmapqualify; UI shards passed in 637.873s, 428.694s and
473.562s. Formatting, TUF/assets/notices, exact Qodana exclusions, vet, build and
the complete 740-runnable UI shard inventory passed. `bash -n` and
`git diff --check` passed. The separate corrected-container testshards race
package passed in 3.500s. No test exclusion or worker isolation policy changed.

### Inspections

Analyzed working tree based on `1c08654`, carried into the handoff commit without
further Go changes: GoLand `get_file_problems(errorsOnly=false)` on all five
changed Location Map Go files returned no findings. The changed testshards Go
file has four weak duplicate-fragment warnings at lines 90/174 and 225/275,
all in unchanged intentional fixture/assertion sequences covered by its existing
exact `qodana.yaml` exclusion. No duplication refactoring is warranted.
The shell file, Makefile and CI YAML returned no diagnostics. All four Swift files returned no
diagnostics through the same API, but batch lint returned an empty item list;
do not count that as Swift language/SDK inspection evidence. Portable Swift
warnings-as-errors compilation is the verified check; macOS SDK/IDE inspection
is still required. This is the documented IDE fallback, not a Qodana SARIF pass.
No PR was opened; this repository's CI triggers on main pushes and PRs, so the
feature-branch push alone does not provide fresh CI, CodeQL or Qodana evidence.

## macOS handoff

Stay on `feature/location-map-gesture-timing` and read this plan before continuing.
Commit/push authorization persists for the handoff; no merge/release authorization.
The todo remains open until native verification resolves the remaining work.

### macOS preparation completed — 2026-09-29

Verified the clean handoff revision `232781f6207c8376f2a5f7cde846c3bad17b6a85`
on macOS 27.0 (26A428), arm64, with Apple Swift 6.4
(`swiftlang-6.4.0.34.1`), macOS SDK 27.0 and Go 1.27.1. No source changes were
needed.

- `make location-map-capture-test SWIFTC='xcrun swiftc'`: PASS; both visual
  transform and response-frame policy suites passed with warnings as errors.
  On this host the latter also compiles the AppKit capture adapter.
- Production `xcrun swiftc -parse-as-library -warnings-as-errors -typecheck
  scripts/locationmapqualify/native/capture.swift
  scripts/locationmapqualify/native/transform.swift`: PASS.
- `make location-map-qualification-test`: PASS under race;
  `scripts/locationmapqualify` 3.460s and `internal/locationtrial` 1.176s.
- `make build`: PASS; the linker reported only the duplicate `-lobjc` warning.
  Optimized production capture-helper compilation with `xcrun swiftc
  -parse-as-library -warnings-as-errors -O`: PASS.
- Initial sandboxed checks could not write the normal Swift/Go caches. The Swift
  diagnostic also reported an SDK/compiler mismatch after failing to build
  SwiftShims; the same commands passed with approved cache access, without a
  toolchain or SDK change.
- GoLand batch inspection of all four Swift files returned no items and
  `more=true`; the individual `capture.swift` request timed out. These are
  incomplete inspection results, not clean IDE gates. Swift SDK compilation
  with warnings as errors is verified separately. No SwiftLint is installed.

Prepared binaries (local, not committed):

| Binary | SHA-256 |
| --- | --- |
| `bin/picfetch` | `d65b039fe5bfd208a66aa4df66391d3caca2ba05e3fcefba997575f55bb59106` |
| `bin/location-map-capture` | `8812471c1a5b9836f6add30c4e2ef559cbb2449004b9a94bdf496f6684afd69a` |

The live trial has not started. The collection path and explicit foreground
CGEvent authorization were requested from Ronin. Use a new evidence directory
under `.scratch` once those are supplied; retain failures and inspect the PNGs
and transform witnesses before claiming native qualification. Existing records
identify earlier evidence directories but do not choose a collection for this
run. Screen/input permissions have not been requested or changed.

Native continuation cost: one read-only scout for prior collection/evidence and
protocol locators, no delegated review or implementation. Its bounded search,
read-only scope, independently checked locators and previously unread evidence
meet the existing G1–G5 scout gate. Carry forward the complete native Linux/amd64
verification above: this continuation changes documentation only and does not
run an emulated full suite. No new CI, CodeQL or Qodana result is claimed.

### Continuation readiness check — 2026-09-29

The next-todo continuation rechecked HEAD at `232781f` and both prepared binary
SHA-256 values; they match the macOS preparation above exactly. Only this plan
and `todos.md` had existing working-tree edits. No source or binary changes were
needed, so the recorded native and Linux test evidence carries forward without
repeating the suites.

A fresh GoLand `get_file_problems(errorsOnly=false, timeout=10000)` request for
`native/transform.swift` returned `errors=[]` with `timedOut=true`: incomplete,
not a pass. Installed GoLand 2026.2.2.1 has no Swift/AppCode/SourceKit inspection
plugin. Its bundled `textmate-plugin/lib/bundles/swift/package.json` identifies
Swift Language Basics and describes only snippets, syntax highlighting and
bracket matching. Swift semantic IDE inspections, including weak warnings, are
therefore unavailable in this configuration. Command-line SourceKit-LSP and
swift-format exist, but their availability does not establish inspection coverage;
the prior warnings-as-errors compiler evidence remains a separate check.

One additional read-only scout checked installed inspection/tool metadata while
the lead checked binary identity and trial setup. The bounded question,
read-only scope, independently checked metadata, and previously unread tooling
meet G1–G5; no review or implementation was delegated. This adds one scout for
the newly investigated inspection limitation, beyond the prior continuation's
recorded collection/evidence search.

The chosen collection path and explicit CGEvent-helper authorization remain
pending. No live trial, screen capture, input injection, permission change,
commit or push occurred in this readiness check. The repository's Computer Use
workflow requires a specific request before CGEvent synthesis; the next-todo
request did not identify the collection or explicitly request that mechanism.

### Unlocked-desktop continuation — 2026-09-29

Ronin's repeated request to continue after unlocking authorizes the previously
described CGEvent capture/input helper. That authorization is no longer pending.
Computer Use reached PicFetch successfully; it showed the empty welcome window,
so there was no currently loaded collection to reuse.

The existing helper's permission-only probe (`bin/location-map-capture 0
.scratch`, using a PID that cannot identify an application window) failed with
`native capture/input permission is absent`. The result was identical outside
the sandbox. A separate native CoreGraphics preflight reported:

```text
Screen Recording: true
Post Event: false
```

Computer Use opened System Settings -> Privacy & Security -> Device Control
and Data Access. GoLand's switch was off, while the separate Codex Computer Use
app's switch was on. Ronin was asked to enable GoLand for the helper and reply
when ready. No permission was requested through an API or changed automatically;
no native trial or gesture measurement was started.

Prepared a disposable fallback corpus in
`.scratch/location-map-smoke-20260929/images`: 24 generated patterned JPEGs with
synthetic GPS coordinates, independently read back with ExifTool (24 files, all
with latitude/longitude). The local `generate.go` records deterministic creation
and refuses an existing output directory. `bin/location-map-qualify` was built
successfully. These inputs can check real native capture/input, but cannot
qualify the 10k/30k performance promise or replace Ronin's chosen real collection.
No private library was searched. A read-only fixture scout located the existing
GPS test helpers and established that the checked fixture directories had no
ready GPS corpus; its bounded search and independently checked locators follow
the existing G1–G5 scout gate. No review or implementation was delegated.

Remaining immediate dependency: enable the native helper's input-posting access,
then repeat its preflight and run a new evidence directory. Keep the synthetic
smoke explicitly separate from real-collection qualification. Source and shipped
dependencies remain unchanged; prior compiler/test evidence still applies.

### Native retry and geometry regression — 2026-09-29

Both macOS permission preflights now pass. Two real runs on the prepared binary
admitted all 24 synthetic GPS images. Their evidence is retained locally:

- `.scratch/location-map-native-20260929T084115Z`: foreground focus was lost
  after the first map entry; the helper refused to continue.
- `.scratch/location-map-native-20260929T084400Z`: initial entry passed, but
  `close-00` timed out. App state confirmed map retirement; retained PNGs show
  different window geometry during entry and after return to the photo.

Ranked causes: the trial's map-entry resize changes the closed-viewer baseline;
initial display work could publish an early baseline; ScreenCaptureKit could
deliver a stale frame across the resize. The existing `beginLocationTrial`
resizes only after the helper captures its baseline. A focused root regression
will first require the fixed 1200x800 geometry before initial map entry, then
across entry/exit. This uses the established native-trial observation test seam.

Task (T0 inline): update `internal/ui/locationtrial.go`, `load.go` and the existing
`TestLocationMap/native_trial_observations` in `locationmap_test.go`; fix isolated
trial geometry before loading, leaving ordinary launch behavior unchanged.
Acceptance: `go test -tags no_emoji,nodynamic -count=1 ./internal/ui -run
'^TestLocationMap$/^native_trial_observations$'`, focused native-trial/static-size
regressions, changed-file GoLand inspections, native build and a new capture run.
Budget: no delegated implementation/review; reuse the retained native Linux
full-suite evidence only for unchanged code. New code requires fresh CI/native
amd64 final verification; no emulated complete-suite pass may be claimed.

The geometry-only native retries (`geometry-fixed`, `geometry-fixed-02` under
the same evidence prefix) exposed a second defect: the app completed scanning
and reported the map active/visible, but complete, newer WindowServer frames
retained the photo body unchanged. A diagnostic helper recorded frame hashes
and timestamps; the separate `location-map-paint-debug-20260929` app was kept
alive to establish active/visible state before shutdown. `Container.Show` in the
pinned Fyne only clears `Hidden`, and the existing root `ForceRepaint` contract
explains why never-painted hidden children cannot invalidate the native canvas.
The old entry resize had supplied that missing invalidation.

Scope adds `internal/ui/locationmap.go`: request the existing root repaint after
first entry. The same native-trial regression observes a transparent mounted
root child's `Refresh` boundary; a software screenshot alone would repaint and
miss this bug. It failed with `map entry did not request a repaint through the
mounted window root` before the fix. Temporary Swift diagnostic changes were
removed after collecting the retained debug run; the production helper remains
unchanged. The isolated diagnostic app was stopped after state collection.

Red/green and native evidence for the final implementation:

- The geometry guard failed with `{520 340}` before the fix. A first attempt
  using the persisted static-size setting failed the broader startup-default
  regression; a new preference guard also failed before correction. The final
  rule belongs to trial auto-sizing and leaves the saved preference unchanged.
- The native-trial regression passed after both geometry and root repaint
  changes. No new top-level test or test file was added, so the existing shard
  assignment and exact Qodana exclusion continue to cover it.
- `.scratch/location-map-native-20260929-repaint-fixed` completed cold map
  entry, pixel-verified close, and warm entry on all 24 admitted JPEGs. The
  inspected before/after close PNGs show the map and the same-size photo. It
  stopped on foreground refusal before the first pan; no gesture timing is
  qualified. The checker with `-images 24` correctly rejected the incomplete
  report. Application SHA-256:
  `5d0bb063fa88bf58a4b2a7c792b43de9956ea3a022f23bb2f51d0eda54f7d2a7`.
- The next retry (`repaint-fixed-02`) also refused foreground admission. A
  bounded diagnostic retry (`focus-debug`) identified the foreground process
  as `loginwindow`: the desktop had locked again. Ronin was asked to unlock
  it; capture/input authorization and both permission flags are already valid.
  The diagnostic log change was removed; production Swift has no diff.
- `make verify-build`: PASS for the final implementation, including format,
  TUF, generated assets/notices, exclusions, vet and build. The native linker
  retains its existing duplicate `-lobjc` warning.
- Final focused `go test -race -tags no_emoji,nodynamic -count=1 ./internal/ui`
  run: PASS in 74.245s. Its anchored selection covered `TestLocationMap`,
  `TestLaunchPolicyIntegration`, all `TestStaticWindowSize_*` and
  `TestSetStaticWindowSize_*` cases, plus the static-size zoom and rotation
  regressions. Geometry, preference preservation and repaint guards were each
  observed failing before their respective corrections.
- GoLand `get_file_problems(errorsOnly=false)` completed with no findings on
  `load.go`, `locationtrial.go`, `locationmap.go` and `locationmap_test.go`,
  analyzed as the working tree based on `ed44d6a`. This is the documented IDE
  fallback, not a fresh Qodana SARIF result.
- `make check-test-platform` outside the sandbox rejects the local
  `linux/aarch64` daemon. Fresh full Linux/amd64 verification is unverified for
  these Go changes; the earlier full-suite result covers the prior code only.
- Native logs retain Fyne's existing threading-migration and shutdown warnings;
  this continuation makes no claim that those diagnostics have been resolved.

No shipped dependency, matcher acceptance rule, timing threshold or foreground
guard changed. All failed runs are retained, and the task remains open for the
complete 40-gesture native trial and fresh full verification.

### Unlocked retry and keyboard-layout regression — 2026-09-29

Ronin unlocked the Mac and requested another run. The unchanged helper in
`.scratch/location-map-native-20260929-awake-retry` completed cold/warm entry,
pixel-verified close and one identified pan (58 captured pixels right, 79/94
matching patches, 99.383 ms). The first zoom timed out with identical retained
before/after pixels. This is partial smoke evidence, not a qualified report.

The diagnosis loop used that native command and its retained PNGs. Ranked
predictions were layout-dependent key translation, routing/focus, then missing
repaint. A bounded key-logging retry (`key-diagnostic`) stopped earlier on an
unidentified pan and required forced app termination; it did not establish zoom
delivery. All temporary key logging was removed. The read-only Carbon probe in
`.scratch/location-map-key-translation.swift` then reproduced the actual layout
boundary: ANSI 0x18/0x1b translate to acute-accent/eszett, while keypad 0x45/0x4e
translate to plus/minus. No keyboard settings were changed.

A read-only native-key scout located the installed GLFW/Fyne path: Cocoa first
maps physical key positions, but Fyne's punctuation lookup calls `GetKeyName`,
which translates through the current layout. The lead independently checked the
locators and owns the fix/review. G1: bounded native key-translation question;
G2: verify local source locators; G3: no writes; G4/G5: unfamiliar dependency
boundary; S/W: cross-module relationship search. This adds one scout to the
continuation budget, no delegated implementation or review.

Task (T0 inline): use keypad plus/minus in the existing Go command protocol and
Swift registration contract; preserve transform acceptance, failed samples,
permissions and foreground checks. The existing `runner_test.go` protocol seam
now checks all 20 zoom inputs; `native/transform_test.swift` tests the new keys
with the same independent synthetic frames and rejection cases. `screen-v4`
records the actual keypad input in the evidence protocol. No app shortcut,
system setting, dependency or distribution obligation changed.

Verification:

- Go regression first failed `zoom 0 input = 0x18, shift=false; want keypad
  0x45`, then passed after the protocol fix. Swift first failed `Requested zoom
  69 was not identified`, then passed after updating its command mapping.
- `make location-map-qualification-test`: PASS; race packages
  `scripts/locationmapqualify` 3.647s and `internal/locationtrial` 1.448s.
- `make location-map-capture-test SWIFTC='xcrun swiftc'`: PASS for visual
  transforms and response-frame policy, with warnings as errors. The production
  Swift helper and native app also built successfully.
- Final `make verify-build`: PASS again after the protocol changes (format,
  TUF/assets/notices, exact exclusions, vet and build).
- GoLand inspection including weak warnings: complete, no findings for
  `protocol.go`, `runner.go`, `runner_test.go`, working tree based on `ed44d6a`.
  The four previously changed UI files retain their recorded clean inspection
  and focused race evidence. Swift semantic IDE support remains unavailable;
  compiler checks do not replace it.
- `.scratch/location-map-native-20260929-keypad` completed entry/close/warm
  entry, then refused foreground admission before gesture 0 and required forced
  app termination. It does not validate the corrected zoom on screen. All failed
  runs remain retained; no sample or acceptance threshold was relaxed.
  App SHA-256 remains `5d0bb063fa88bf58a4b2a7c792b43de9956ea3a022f23bb2f51d0eda54f7d2a7`;
  corrected helper SHA-256 is
  `b6be126d47a5bc8cf2a5a90d642dce4b8827103e4fb0ac1343887c5d9f877c66`.

The remaining native gate needs an uninterrupted foreground session; Ronin was
asked to avoid switching applications during the trial. Fresh full Linux/amd64
verification is still required for the changed Go code.

### Complete local native attempt — 2026-09-29

Ronin requested the complete local run after commit `33cd7af`. The existing
application/helper binaries above were run with process-scoped idle-sleep
prevention; no permanent power or security setting changed:

```sh
/usr/bin/caffeinate -di ./bin/location-map-qualify run \
  -images .scratch/location-map-smoke-20260929/images \
  -evidence .scratch/location-map-native-20260929-complete-local \
  -binary ./bin/picfetch -helper ./bin/location-map-capture -timeout 5m
```

Result: exit 1, `gesture 1: no identified native response within 3s`.
Cold/warm scans, initial pixel-verified close and gesture 0 passed. The pan
witness is scale 1, dx 58, dy 0, 106/122 matches, 74.764 ms. The retained
`gesture-001-before.png` and `gesture-001-after.png` visibly show the map zooming
in, establishing that the synthetic keypad input reaches the native app without
a physical keypad. The observer did not identify a qualifying transform, so its
zoom timestamp correctly remains zero and the run is incomplete. This failure
is neither missing permission nor foreground refusal. The failed sample and
whole-window PNG pair remain intact; no timing or acceptance rule was relaxed.

The diagnosis workflow's bounded evidence comparison distinguishes successful
visible input from failed measurement identification. It does not yet establish
why registration rejected the real scene; that requires replay/instrumentation
against the retained frame pair before any matcher change. No source changes
were made in this verification continuation.

`location-map-qualify check` with the actual count of 24 returned exit 1,
`incomplete or non-native screen capture report`, as required. The complete
40-gesture/cancellation gate has not passed. The runner finished and its
temporary idle-sleep assertion ended.

Fresh `make check-test-platform` also failed: the local daemon reports
`linux/aarch64`, while the complete suite requires native Linux/amd64. No
emulated full suite was run, no isolation test was skipped, and no new full-suite
pass is claimed. Unchanged-source focused/compiler/IDE evidence above carries
forward from `33cd7af`; full native Linux/amd64 verification remains open.

### Remaining live qualification

1. Stay on this branch. Native compilation and focused tests are complete at the
   revision above; repeat them if source changes during the trial. Swift semantic
   inspection is unavailable in the installed GoLand configuration. Complete it
   with a suitable native tool if available, preserving the distinction between
   that gate and the passed compiler checks.
2. Resolve the real-frame zoom registration rejection recorded above before
   expecting a complete run; preserve the failed pair as its reproduction.
   CGEvent-helper approval, both native permissions and visible keypad zoom
   delivery are verified. Resume
   with an unlocked desktop and keep PicFetch foreground. Use the prepared synthetic
   corpus for a smoke check or Ronin's chosen real collection, always with a
   fresh evidence directory. The helper preflights existing Screen Recording
   and Accessibility permissions; do not grant them automatically. Run
   `make location-map-qualify` with explicit paths.
3. Inspect whole-window before/after PNGs against retained witnesses and the
   recorded direction/latency. Verify 40 pan/zoom samples, closure matching and
   preservation of any failed sample. Use actual admitted image count when
   checking smoke evidence, and demonstrate rejection under 10k/30k counts.
   If registration refuses real imagery, retain the failed evidence and improve
   the matcher against a reproducible fixture; never weaken it to hash changes.
4. A small native smoke validates the helper, not the 10k/30k performance promise.
   Use the documented exact-count protocols if formal qualification is requested.
   Ronin alone supplies the 30k verdict. Keep the maintainer acceptance separate.
5. Record native results, update this plan/todos and commit/push. On Apple Silicon,
   do not run or claim an emulated amd64 full-suite pass. The Linux continuation
   above now verifies the Go fixes through `d0998a9`; further source changes need
   fresh affected verification. Both macOS CI jobs now
   run the portable matcher/policy target and type-check the production helper.
