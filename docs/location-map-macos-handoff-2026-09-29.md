# Location Map: return to macOS

Archived outcome, 2026-09-29: application fixes are complete, Ronin's manual zoom
check passed, and automated zoom timing is parked at his request. The
[closure record](../finished_refactorings/2026-09-29-location-map-gesture-timing.md)
and [PR 73](https://github.com/frathe/picfetch/pull/73) retain verification and
final-head review gates. The handoff instructions below are historical, not an
active request to resume measurement.

Continue on `feature/location-map-gesture-timing`. Ronin requested this handoff,
commit and push before switching computers. No merge or release is authorized.
Read `AGENTS.md`, the SDD/TDD working agreement and `ARCHITECTURE.md` before
implementation. Preserve any local edits when fetching and updating the branch.

## macOS continuation update

The window fix is now pushed as `f1d5e6a`: entering/returning to Location Map
maximizes the work area, with fixed qualification geometry preserved. The
reported first-map redraw symptom came from an old local app bundle; rebuilding
the branch bundle restored its existing repaint fix. Focused race tests, GoLand
Go inspections, build/shard checks and native entry/return checks passed. The
new code still needs full native-amd64 verification; ordinary feature pushes
do not start CI without a PR.

The resumed matcher investigation found a premature coarse pan-candidate
rejection and added a retained native-frame regression. A separate diagnostic
fixture now captures the first changed zoom frame: old tiles scale, photo cards
do not. The current matcher still rejects this mixed-scale content under the
unchanged coverage rule. Do not interpret the new fixture as a passing zoom or
latency sample. One clean run lost foreground before gesture 0; a dedicated
retry passed cold/warm scans, return and pan (73.617292 ms), then rejected the
first zoom. Both failures are retained. See the plan's "Matcher continuation after the
window fix" for exact evidence and remaining work. Earlier sections below are
the original Linux-to-Mac handoff, not newer verification claims.

## Completed

The application fixes are complete at `33cd7af`: fixed trial geometry before the
baseline, first-map root repaint and layout-independent synthetic keypad zoom.
Native compilation, focused regressions, permissions and visible zoom delivery
passed on the Mac. No Go or Swift source changed during the Linux continuation.

Fresh native Linux/amd64 `make verify` passed for `d0998a9`, including all race
partitions. Root UI outcomes were 738 passes and two existing conditional skips;
the runner and container both exited 0 with no OOM events. Focused qualifier and
recorder race tests, both portable Swift suites, and complete GoLand inspections
of all ten changed Go files also passed with no actionable findings. The
[plan](../finished_refactorings/2026-09-29-location-map-gesture-timing.md) records exact commands,
scope, timings, exclusions and inspection limitations. Documentation-only
handoff changes carry that unchanged-code evidence; do not repeat the full suite
under ARM-hosted amd64 emulation.

## Remaining issue

The native **measurement** remains incomplete. The last uninterrupted Mac run
completed cold/warm scans, pixel-verified viewer return and one identified pan
(74.764 ms), then stopped at `gesture 1: no identified native response within 3s`.
The map visibly zoomed, but registration rejected the frame. Permission, input
delivery and foreground focus were not the blocker in that run.

The unmodified before/timeout PNG pair is tracked in
[`scripts/locationmapqualify/testdata/native-zoom-20260929/`](../scripts/locationmapqualify/testdata/native-zoom-20260929/README.md).
A temporary Linux replay reproduced rejection, but did not establish its cause
or the first response time. Its temporary sources/data were removed. The final
frame cannot establish what happened in intermediate frames or supply latency.
Retiling and fixed-size photo cards remain hypotheses, not confirmed causes.

## Resume here

1. Fetch and update this same branch. Read this handoff and the plan's
   "Remaining live qualification" section. The existing application fixes do
   not need to be reimplemented.
2. Reproduce and diagnose registration at
   `scripts/locationmapqualify/native/transform.swift` (`identifyTransform`) and
   `native/capture.swift` (`Observer.visualFrame` and pending-frame processing).
   The [Linux handoff](location-map-linux-handoff-2026-09-29.md) records exact
   replay conversion. If the retained pair is insufficient, collect narrowly
   scoped intermediate-frame/candidate diagnostics on macOS. Establish a failing
   reproduction before changing the matcher; never force an unsupported final
   frame to pass.
3. Preserve independent pixel identification, scale/direction checks,
   stationary/ambiguous/repaint rejection, closed-viewer matching and every
   failed sample. Hash changes alone never qualify gestures. Keep capture
   timestamps independent of application camera state and worker completion.
4. After any repair, run `make location-map-qualification-test` and
   `make location-map-capture-test SWIFTC='xcrun swiftc'`, inspect changed files,
   and rebuild the native helper/application through `make location-map-qualify`.
   Swift semantic IDE inspection remains unavailable in the recorded GoLand
   setup; compiler success is separate evidence. Further source changes require
   fresh affected verification; the Linux pass covers the existing source only.
5. Run a fresh native trial with an unlocked desktop and PicFetch kept in the
   foreground. Existing helper authorization persists; preflight the existing
   permissions without granting or changing them automatically. On the original
   Mac, the synthetic corpus was `.scratch/location-map-smoke-20260929/images`.
   Check that it still exists and use a new evidence directory:

   ```sh
   make location-map-qualify \
     LOCATION_MAP_IMAGES=.scratch/location-map-smoke-20260929/images \
     LOCATION_MAP_EVIDENCE=.scratch/location-map-native-after-linux
   ```

   The example evidence path must not already exist. Do not overwrite an old run.
6. Require all 40 gestures plus cancellation/closed-viewer checks. Inspect the
   retained whole-window images against their witnesses and first verified frame
   times. Check smoke evidence against its actual 24-image count and confirm it
   cannot qualify the 10k/30k protocols. Real-collection qualification and Ronin's
   30k verdict remain separate. Update `todos.md` and the plan with actual results;
   keep the todo open until its remaining gates are resolved.

## What transfers

Git carries the source, documentation and diagnostic PNG pair. Original native
binaries, the generated 24-image corpus and raw Mac run logs remain on the Mac;
the failed run was `.scratch/location-map-native-20260929-complete-local/`.
Linux race artifacts remain on Linux at
`.scratch/race-runs/20260929T095440Z-KtahR5/`; their results are recorded in the
tracked plan. No replay loader or temporary instrumentation is being handed off.

No PR was opened. A feature-branch push alone does not establish fresh CI,
CodeQL, Qodana SARIF or Codex review results.
