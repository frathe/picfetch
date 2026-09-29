# Location Map: Linux handoff

Archived outcome, 2026-09-29: application fixes are complete and Ronin accepted
manual zoom. Automated native timing is parked at his request; the
[closure record](../finished_refactorings/2026-09-29-location-map-gesture-timing.md)
and [PR 73](https://github.com/frathe/picfetch/pull/73) retain the final gates.
The Linux results and [return-to-macOS handoff](location-map-macos-handoff-2026-09-29.md)
below are historical, not instructions to resume the parked measurement.

Continue on `feature/location-map-gesture-timing`. **Linux verification is
complete at `d0998a9`; native timing measurement remains incomplete.** The macOS
fixes and focused tests passed, but the recorded native run stops at its first
zoom even though the screen visibly zooms. No application or Swift source changed
during Linux verification. Ronin authorized committing and pushing the original
handoff; merge/release is not authorized. The complete history is in the
[implementation plan](../finished_refactorings/2026-09-29-location-map-gesture-timing.md).

## Linux completion — 2026-09-29

Local and remote branch tips were confirmed as `d0998a9`. The native Linux/amd64
prerequisite, focused qualifier/recorder race tests, portable Swift suites and
full `make verify` passed. All ten changed Go files received complete GoLand
inspections including weak warnings, with no actionable findings; existing
test-duplication warnings retain their exact exclusions. This is IDE fallback
evidence, not a fresh Qodana SARIF result.

The full race run is retained at `.scratch/race-runs/20260929T095440Z-KtahR5/`.
All partitions passed; runner/container exits are 0 and no OOM events occurred.
The 740-entry root UI inventory produced 738 passes and its two existing
conditional skips (case-insensitive filesystem and explicit native HEIC provider
qualification). The plan records exact scope, timings and limitations. These
local results do not establish a successful native macOS timing run or new CI.

## Original Linux handoff steps

1. Fetch and check out this branch, preserving any local edits. Read `AGENTS.md`,
   its SDD/TDD working agreement and `ARCHITECTURE.md` before implementation.
2. Check `make check-test-platform`. Full verification needs a **native
   Linux/amd64** Docker daemon. The Mac's ARM daemon was correctly rejected.
3. Run the focused baselines, then diagnose the retained zoom frames:

   ```sh
   make location-map-qualification-test
   make location-map-capture-test
   ```

   The Swift target requires `swiftc`; the plan records the previously working
   Linux Swift 6.2.4 container fallback. Linux checks the portable matcher/policy,
   not the AppKit/ScreenCaptureKit adapter.
4. Linux inspections and `make verify` now cover the newer Go changes at
   `d0998a9`, as recorded above. Repeat affected checks if source changes.

## What is already fixed

`33cd7af` contains three fixes and their failing-then-passing regressions:

- Establish 1200x800 isolated-trial geometry before the closed-viewer baseline;
  suppress trial auto-resizing without changing the saved static-size setting.
- Request the mounted root repaint on first Location Map entry. Fyne's hidden
  container could report visible without publishing new native pixels.
- Use synthetic macOS keypad plus/minus codes (0x45/0x4e) for trial zoom. ANSI
  punctuation codes translated to acute-accent/eszett on the active layout.
  No physical keypad or system keyboard-layout change is needed. Ordinary app
  shortcuts are unchanged; the latest screen capture confirms zoom input works.

`ed44d6a` and `c69a4c0` record the macOS preparation and subsequent native results.
All temporary debug logging was removed. Code remains unchanged since `33cd7af`.

## Reproduce the remaining measurement failure

The transferable, unmodified 1200x800 PNG pair is committed under
[`scripts/locationmapqualify/testdata/native-zoom-20260929/`](../scripts/locationmapqualify/testdata/native-zoom-20260929/README.md).
It shows only generated synthetic photos at synthetic Berlin coordinates, not a
private photo collection. Its README records provenance and interpretation limits.
Other `.scratch` directories, native binaries and raw logs remain Mac-local;
Git will not transfer them.

The latest native run admitted 24 JPEGs, completed cold/warm scans and the initial
pixel-verified return, then identified a pan at 74.764 ms. The first zoom ended
with `gesture 1: no identified native response within 3s`; its timestamp remains
zero. The actual-count evidence checker rejected the incomplete report.
Sleep, permissions and foreground focus were not the blocker in this run.

Start at `native/transform.swift` (`identifyTransform`) and
`native/capture.swift` (`Observer.visualFrame` and pending-frame processing),
under `scripts/locationmapqualify`. A Linux replay should reproduce the capture
conversion: top-down 4x4 blocks, integer average of `29*B + 150*G + 77*R` divided
by `16*256`, producing a 300x200 luminance frame from each PNG. Keep raw PNGs
intact. A small standard-library Go PNG reader can feed the portable Swift seam;
no replay loader has been implemented yet.

Establish a failing replay before changing registration. Check candidate
coverage, ambiguity and sampled content. Retiling and constant-size thumbnail
cards are hypotheses, not confirmed causes. The saved after-image is the last
frame at timeout, **not** the first changed frame; it cannot establish a missing
early response or supply a latency timestamp. If that pair cannot distinguish
the cause, collect narrowly scoped candidate/frame diagnostics on macOS.
Do not require an unsupported final-frame transform to pass merely to clear a test.

Preserve independent transform identification, direction/scale checks,
stationary/ambiguous/repaint rejection, every failed sample and closed-viewer
matching. Hash changes alone never qualify gestures.

## Verification already available / still required

On the changed code: focused root Location Map/static-size/launch-policy race
tests passed (74.245s); qualifier and recorder race suites passed (3.647s/1.448s);
portable Swift and native adapter compilation passed with warnings as errors;
`make verify-build` passed. All seven changed Go files had complete clean GoLand
inspections, including weak warnings. Exact commands and scope are in the plan.
Swift semantic IDE inspections remain unavailable, not passed.

The fresh full Linux/amd64 gate is complete. Still required: a complete
**macOS** native 40-gesture plus cancellation run after measurement repair.
Linux unit tests or offline replay cannot replace WindowServer timing. Check smoke evidence against
its actual count and confirm it cannot qualify 10k/30k. Real-collection performance
qualification and Ronin's 30k verdict remain separate.

No PR was opened during this handoff. Feature-branch push alone does not trigger
this repository's main/PR CI; do not infer CI, CodeQL, Qodana or bot-review passes
from the push. Keep the todo open until its remaining gates are actually resolved.
