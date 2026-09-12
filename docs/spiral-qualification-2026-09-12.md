# Spiral native qualification, September 12, 2026

## Accepted result

Status: complete, accepted by Ronin on September 12, 2026.

After testing the updated Spiral for an hour, Ronin reported stable,
relatively smooth operation, FPS staying around 60, and smooth GIF playback.
He explicitly counted the result as a pass for a fun Easter egg. This is
Ronin's direct observation; Pico did not instrument or record that hour.

This acceptance, together with the existing automated and native checks below,
closes ticket 05 and the Hypno Spiral tunnel image stream TODO. It supersedes
the earlier requirement for an additional high-frame-rate capture and the
remaining qualification language in the historical sections below. The final
defaults retain the 0.75-second entrance fade, 3% protected centre disc and
one prepared preview. No preload expansion or further tuning is required.

The evidence limits remain explicit: the hour-long report does not establish
every native edge case, exact GPU timing/allocation, or other GL/GLES backends.
Real viewer-trigger/duplicate behavior and one/two/unreadable-source recovery
retain their automated coverage; no additional native demonstrations are
claimed. The earlier overlay variability was not diagnosed, but Ronin's
sustained run establishes an acceptable experience for this feature. The two
known local amd64 seccomp failures remain in their separate TODO.

Closure changes only work records and archives the accepted plans. Existing
test/build/GoLand results remain the verification evidence; no production code
changed and no broad test rerun was needed. Documentation links and
`git diff --check` were checked at closure. No commit was created by Pico.

## Qualification history

Pico resumed the first open TODO, ticket 05 of the Hypno Spiral tunnel, from
`324cec5`. Native QA found and fixed a fullscreen-restoration defect: Exit
Full Screen collapsed the canvas to 1x1. Spiral now establishes a 960x600
window before entering fullscreen. This record separates resource and
geometry evidence from motion that the capture cannot resolve.

## Reproduction and evidence

The isolated native harness and raw records are in
`.scratch/hypno-spiral-tunnel/qualification-20260912/`. Build and run it from
the repository root:

```sh
go build -o '.scratch/hypno-spiral-tunnel/qualification-20260912/PicFetch Spiral Qualification.app/Contents/MacOS/spiral-qualification' ./.scratch/hypno-spiral-tunnel/qualification-20260912/harness
'.scratch/hypno-spiral-tunnel/qualification-20260912/PicFetch Spiral Qualification.app/Contents/MacOS/spiral-qualification'
```

The app identity is `io.github.frathe.picfetch.spiralqualification20260912`.
Its controller reopens the same production Spiral instance. It loads four
existing CC0/public-domain Explorer fixtures (cat, astronaut, coffee and
train), PicFetch's transparent icon, the existing synthetic two-frame GIF,
and `assets/picfetch_functionality.gif`: seven sources, including two animated
previews. Fixture provenance remains in
`internal/ui/testdata/explorer/README.md`. No shipped dependency or asset was
added or changed.

The native backend is macOS 26.6.2, Apple M5 Max, Go 1.27.1 and Fyne 2.8.0
desktop OpenGL. The baseline canvas reports 1920x1200 with scale 1; the rebuilt
trial opens on the 3840x1600 external display. Default settings:
Ripple, rotation 2.2, hue speed 0.06, four arms, twist 30, density 1,
image speed/size 1x, image gap 2.50 seconds, randomness 35%, Main order and
15–85% transparency. Only help visibility and window focus were changed
during the default interval.

`baseline-native.jsonl` records one UI-delivered snapshot per second: shader uniforms,
bound texture dimensions, Go heap metrics and goroutine counts. `baseline-rss.jsonl`
records process RSS/CPU from `ps`. The sampler does not force GC during the
default run; it scavenges once after the first close to measure release.
`baseline-binary.sha256` records the original executable identity; the
retained app was rebuilt with the fix and has its own `binary.sha256`.
The rebuilt run uses `native.jsonl` and `rss.jsonl`. `analyze.py` reconstructs
births across the shader's
minute-boundary rebases. This observes the instrumented native app, not the
unmodified main executable or a GPU allocation profiler.

Reproduce the summaries after collection:

```sh
python3 .scratch/hypno-spiral-tunnel/qualification-20260912/analyze.py --prefix baseline-
python3 .scratch/hypno-spiral-tunnel/qualification-20260912/analyze.py --start 100 --end 710
```

## Sustained baseline and controls

The first 609.215 seconds contain 609 samples and 209 observed admissions
across 42 profile batches. All completed interior batches contain 3–7 images.
The seven successful sources and Main order imply 29 complete source cycles;
the sampler measures births, not source identities. Admission gaps range from
2.368 to 3.952 seconds. Angular quadrant counts are 60/52/44/53. The largest
interval between one-second UI samples was 1.017 seconds.

After the first 120 seconds, process RSS stays between 221.5 and 223.25 MiB.
Successive two-minute means are 222.581, 222.562, 222.519 and 222.563 MiB.
Live Go heap ranges from 19.30 to 36.37 MiB, with 69 completed GC cycles by
the end of the default interval. Goroutine samples are 12–13. There is no
continuing growth in these measurements. The trace observes no more than
three active texture slots, with at most 2.416 MiB of bound RGBA frame pixels
for these fixtures. This excludes retained off-frame GIF frames, transient
decodes and driver allocations; it is not a GPU memory limit.

The native control pass reached both presets/orders, zero/100% randomness,
0.35x/2x speed, 0.75/6-second gaps, 0.5x/2x image size, 15–15%/99–99%
transparency and negative rotation (-0.60). Dragging the pointer with Follow
enabled moved the centre near all four edges. Source snapshots and active
flights stayed available through control changes. Screenshots and actual
uniform samples retain the checks. One/two/all-failed/recovered source sets
remain covered by the existing behavioral tests; this run's native corpus
contains seven readable sources.

After closing Spiral, an explicit scavenging pass reduced live Go heap to
16.683 MiB; it remained there for the following 55 seconds with 11 goroutines.
The process RSS then measured 166.703 MiB. The controller was still open, so
this is retained process memory, not a fully unloaded app baseline. Normal
shutdown returned exit code 0 after 960.84 seconds.

## Fullscreen restore fix

The native Window > Exit Full Screen action produced a persistent 1x1
canvas, confirmed by successive telemetry samples. The existing
`TestShowOpensFullScreenShaderWindow` was extended at the Spiral/window
boundary and failed with `leaving fullscreen restores an unusable canvas:
{1 1}; want at least 640x480`. Initializing the window to 960x600 before
fullscreen makes the guard pass. The software driver verifies initialization;
the native repeat verifies actual restoration ordering.

The rebuilt native app restores to 960x600, then successfully resizes to
644x420 and returns to fullscreen at 3840x1600. Evidence:
`fixed-windowed.jpg`, `fixed-resized.jpg`, `fixed-fullscreen.jpg` and the
corresponding `native.jsonl` dimensions. The complete Spiral race package
passes in 2.860s. Both changed Go files have complete, empty GoLand results,
including warnings. No new test file, locale key or root UI runnable was added.

The fixed executable then ran for another 609.178 default seconds after
returning to fullscreen (process seconds 100–709.178). Its 610 samples show
197 admissions, 41 profile batches and no more than three active textures.
Completed interior batches remain 3–7 images, and admission gaps range from
2.527 to 4.113 seconds. Warm live Go heap is 19.17–36.36 MiB; goroutines remain
12–13. RSS ranges from 175.33 to 235.06 MiB and decreases during the run.
This run overlaps Docker verification and uses a larger display and a fresh
random seed, so it is not a matched performance comparison with the baseline.
Neither run shows continuing retained-heap growth.

After the final trial's first close and scavenging, live Go heap is
18.094 MiB with 11 goroutines. The native reopen retains the selected 1.95x
image speed and size, and new photos appear. The before/after screenshots
are `before-reopen.jpg` and `retained-controls-after-reopen.jpg`. Normal
shutdown returns exit code 0 after 976.71 seconds.

`final-native-frame-sequence.mp4` retains 25 native JPEG samples spanning
44.284 seconds from the fixed executable; the encoded clip is 46.12 seconds
with the final sample held. All 25 frames are 1844x768 captures of Spiral.
The earlier accepted baseline sequence contains 51 frames at 1229x768.
`capture-validation.json` records format/dimension checks using `sips` on
every frame. Both sequences retain their actual sample timing and have
insufficient temporal resolution for a smoothness verdict.

## Native capture and visual assessment

`verified-native-motion.mp4` preserves 51 timestamped native screenshots over
44.365 seconds. Every retained frame has the full scene dimensions; originals
and timestamps remain in `motion-frames/` and `motion-frames.json`. Frames
show the clear centre, upright photos, changing routes, transparent icon
edges, GIF frame changes and the spiral visible through the image layer.
The first attempt switched to the controller window and is rejected as a
motion qualification record.

Pico's bounded visual verdict: the centre remains dominant and the image
layer is restrained; the recognizable photographs move through varied
positions without obscuring the spiral. This supports retaining the defaults
Ronin already described as calming. The approximately one-frame-per-second
capture cannot establish smooth acceleration, absence of short stalls or
single-frame exit popping. It also does not measure rendered alpha or GPU
allocation counts. A full-rate native moving record remains required for
those parts of 05-B/05-C. QuickTime's recording menu and shortcut did not
produce an accessible recorder; launching the system capture UI failed.

## Verification

```sh
go test -race ./internal/imaging ./internal/ui/spiral ./internal/ui/help ./internal/ui -run '^Test(Tunnel.*|HypnoTunnel|AnimatedPreview.*|DecodeAnimatedGIF.*|OverlayToggleKeys|ManualHasNoUnicodeArrows|TranslationsHaveNoUnicodeArrows)$' -count=1
go test . -run '^TestTranslations' -count=1
```

All selected checks passed: imaging 1.703s, Spiral 2.717s, Help 2.311s,
UI 14.144s and root translations 0.528s. Existing guards cover the real
viewer entry adapters, frozen duplicate-aware input, GIF timing/budgets,
ordering, failed-source recovery, control extremes, clocks/resize and
stale/blocked teardown. Their intended RED failures are recorded in the
original and follow-up plans. The new fullscreen guard's RED/GREEN results
are in `fullscreen-red.log` and `fullscreen-green.log`.

GoLand inspected the native harness with errors and warnings included:
zero findings, complete result. The two production/test files changed for
fullscreen restoration were also inspected as described above.

The final `make verify` completes with exit 2. Formatting, TUF/Qodana metadata,
vet/build and all three UI race partitions pass (395.736s / 407.971s /
415.739s). The only failed cases are the existing local amd64 container
isolation failures:

- `internal/similarity`: `TestLinuxWorkerIsolation` reports
  `offline worker seccomp: invalid argument`.
- `scripts/explorereval`: `TestAssetInstall/worker_reaches_asset_check_and_exits`
  reports the same error; its parent `TestAssetInstall` consequently fails.

All other package results pass, including the new fullscreen regression in
the Linux Spiral race suite. Raw final artifacts:
`.scratch/race-runs/20260912T132125Z-zbZIio/`; the extracted package/test
dispositions are in `gate-results.json`. The earlier run
`20260912T131458Z-wxL8xc` was deliberately interrupted to fix the native
finding and is not a final verification result.

`make build`, `GOOS=windows GOARCH=amd64 go vet ./internal/...`, changed-file
GoLand inspections and `git diff --check` pass. The current native binary is
`bin/picfetch`. There are no package moves or new test files/runnables, so the
architecture map, Qodana exclusions and UI shard manifest require no changes.

Ticket 05 remains open for a native recording with sufficient frame rate to
assess brief stalls, smooth acceleration and complete exits. Native
one/two/all-failed-source demonstrations, exact GPU allocation/alpha
measurement and other GL/GLES backends were not qualified here; existing
behavior tests retain their narrower coverage. No release-ready or clean
complete local-gate claim is made. All work stayed with the lead; no commit
was created.

## Live-window and soft-entry follow-up

Ronin requested live-window observation with the FPS overlay during the next
continuation, then requested fully transparent image entrances and launches
closer to the centre. The implementation and current verification are tracked
in [the soft-entry plan](../finished_refactorings/2026-09-12-spiral-soft-entry.md).

The new entrance multiplies the normal, live-configured opacity by a
0.75-second smoothstep fade. Travel and GIF playback keep their existing
admission clock. The protected disc is now 3% of the shorter physical
viewport dimension, down from 8%; the default square card's launch radius in
an 800x600 viewport decreases from 94.026 to 64.026 pixels. Whole-card
clearance and full exits remain covered across the existing size/aspect
matrix. These values supersede the earlier entrance defaults in this record.

The pre-change native trial used e6416ff and retained its separate binary,
source harness, per-second samples and live-window observations under
`.scratch/hypno-spiral-tunnel/motion-qualification-20260912/`. Its normal
exit was observed after 992.121 seconds. The 24 window snapshots span
55.267 seconds between capture midpoints; median spacing is 1.767 seconds,
with a 19.966-second gap between observation batches. They show upright
photos, a clear centre, translucent image layers and edge departures. This
is sampled visual evidence, not continuous high-frame-rate recording.

The FPS overlay was read directly from retained native screenshots:

| Observation | FPS readings |
|---|---|
| Initial Ripple | 63, 62, 63, 60, 62, 63 |
| Nautilus after switching | 15, 10, 66, 30, 30, 30 |
| Later Nautilus | 12, 60, 15, 30, 15, 60 |
| Return to Ripple | 30, 60, 59, 32, 12, 29 |
| Ripple with five-second sampling pauses | 8, 58, 7, 12, 62 |

Both presets showed variable later readings. There is no established
preset-specific, decode-related or snapshot-related cause. `updateFPS`
displays the reciprocal interval of the acknowledged UI-update loop, while
the shader animation uses Fyne's separate render clock. These readings must
not be described as GPU presentation measurements or as a sustained 60 FPS
pass. The trial's warm live heap remained bounded; expanding preloading has
no demonstrated performance benefit here.

The rebuilt soft-entry trial uses the same seven fixtures and a new isolated
app identity, `io.github.frathe.picfetch.spiralsoftentry20260912`. Its
artifacts are under `.scratch/hypno-spiral-tunnel/soft-entry-20260912/`.
The live window at 3840x1600 shows the closer launches with the centre still
clear; initial FPS readings were 63, 63, 63, 63, 60, 63. This run overlaps
Docker verification and is not a controlled before/after performance test.
The screenshot cadence does not establish the exact 0.75-second rendered
alpha curve. The deterministic flight regression covers that clock/opacity
behavior; native inspection verifies that the revised GLSL compiles and
renders, with no observed new layout defect.
The trial exits normally after 571.844 seconds. Its 571 telemetry samples
retain at most three texture slots; live heap after the first minute ranges
from 21.45 to 37.95 MiB (median 32.57 MiB). The final direct overlay reading
is 63 FPS. `native-summary.json` records these measurements and the rebuilt
binary hash beside the retained screenshots and raw samples.

The guards independently failed for immediate 15% opacity and the old
94.026-pixel radius, then passed after their respective changes. Focused
flight/clock/resize checks pass, the complete native Spiral race package
passes in 2.749 seconds, and individual GoLand inspections of all three
changed code files report no errors or warnings. `make build` passes and
refreshes `bin/picfetch`. The final `make verify` exits 2 with only the same
two local amd64 seccomp cases described above. Formatting, metadata checks,
vet/build and all three UI race partitions pass (390.438s / 397.205s /
405.428s). Raw race events are in
`.scratch/race-runs/20260912T140914Z-sNvm4r/`; the extracted package/test
results are in the soft-entry artifact directory's `gate-results.json`.
`git diff --check` passes. The requested entrance changes are implemented;
the broader motion/backend qualification limits remain open. No commit was
created.

One prepared preview and three active slots remain. Preparing three previews
instead would increase the retained animated-preview budget from four to six
16 MiB previews, without addressing shader/UI rendering delays. No decoder
starvation was established in this trial, so that change was not made.
