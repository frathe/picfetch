# Spiral native qualification, September 12, 2026

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
