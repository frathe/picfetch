# PicFetch — TODOs

## Done

### What's Changed

#### New Features

- Spiral tunnel pictures now play bounded GIF previews with independent timing.
  A live image-transparency slider shifts the range, preserves the 85% visibility
  ceiling, and retains its setting across reopening in the current process.
  Image size is adjustable from 0.5x to 2x for new arrivals, with centre clearance
  preserved. Ronin confirmed all sliders work as intended in the native trial.
  See the [follow-up evidence](plans/2026-09-12-spiral-help-and-gif-playback.md).

#### Bugfix

- H toggles Spiral's local help overlay; F1 retains the main manual binding.
- PR #20 fixes route F1 from the Spiral canvas to the manual, keep blocked
  preview reads from delaying process exit, restore arrivals after shrinking
  an off-screen centre, and prevent random-cycle boundary repeats when a URI
  occurs more than once. Regression evidence is in the
  [review record](plans/2026-09-12-spiral-help-and-gif-playback.md#pr-20-review-loop).

#### Internal

- Rebalanced Linux UI race shards using Release v1.1.0 test timings: 27 whole-test
  moves preserve all 681 assignments and project a 37.9% lower maximum test load.
  See [evidence and validation](plans/2026-09-11-ui-shard-rebalance.md).

## TODO

### Hypno Spiral tunnel image stream

Implement the accepted infinite image-tunnel enhancement for the Hypno Spiral
easter egg: a frozen duplicate-aware source snapshot, bounded three-texture GPU
renderer, main/random cyclic ordering, animated GIF previews, controls, and
lifecycle coverage. Refined motion calls for gentle acceleration, soft edges,
subtle batch variation, translucent depth overlap, and native visual tuning.
The original implementation is in c725cae; the help/GIF/control follow-up is in
24dd11f. The [PR #20 review loop](https://github.com/frathe/picfetch/pull/20)
tracks current-commit Codex code/security reviews, Qodana, CodeQL and CI;
its evidence is in the [follow-up plan](plans/2026-09-12-spiral-help-and-gif-playback.md#pr-20-review-loop).
Focused race checks and GoLand inspections
pass, and Ronin found the native defaults very calming. Final qualification
remains open: all UI partitions passed, while the full gate failed only on the two existing
local amd64 seccomp cases;
a native moving capture and measured resource plateau are still unverified.
See the [ticket evidence](.scratch/hypno-spiral-tunnel/issues/README.md).
See the [specification](.scratch/hypno-spiral-tunnel/spec.md) and
[implementation plan](plans/2026-09-11-hypno-spiral-tunnel.md).

### Confirm UI shard balance in CI

Run CI after the September 11 manifest rebalance and compare the three raw test
artifacts. Projected test sums are 470.520s / 470.530s / 504.870s, based on one
completed release attempt; runner/setup variance and the realized improvement
remain unmeasured. All three local UI race shards pass; the full local gate still
has the separate seccomp failures below. See the
[rebalance record](plans/2026-09-11-ui-shard-rebalance.md).

### Linux worker isolation in the local amd64 test container

The September 11 `make verify` run on the Apple Silicon host reports
`offline worker seccomp: invalid argument` in `TestLinuxWorkerIsolation` and
`TestAssetInstall/worker_reaches_asset_check_and_exits`. Both checks pass with
the race detector in a native Linux ARM64 container. Investigate the local amd64
execution environment before claiming a clean complete local gate; retain
native Linux amd64 CI coverage of sandbox enforcement. This is separate from
the confirmed Windows map fix. See the
[verification record](plans/2026-09-11-explorer-windows-setup.md).

### Antivirus verdicts on unreleased builds

The September 11 local builds have likely false positives: Microsoft flags both
Linux architectures as `Trojan:Script/Wacatac.C!ml`; Trapmine alone flags Windows
AMD64 as `Malicious.high.ml.score`. Microsoft reports both Windows builds as
undetected. Windows ARM64 is 0/67, but Trapmine cannot process that file type.
All 99 distinct cached dependency directories and archives match the build
metadata and `go.sum`. Vendor review remains pending; no samples have been
submitted by the agent. Record final vendor determinations and rescan the final
release artifacts. See [the investigation](docs/antivirus-triage-2026-09-11.md).

### Comparison test deadline under build contention

The Ubuntu full race run on 2026-09-10 timed out in
`TestCompareSettle_DrainsVectorReplacementBeforeWaitingForObsoleteTiles`
while another distro build ran. Its one-second Settle deadline expired; an isolated native and exact Ubuntu Docker race
reruns passed. Review the fixture's deadline/load sensitivity; the Explorer change does not modify comparison code.

### Explorer on Linux ARM64

Linux ARM64 now has the pinned native runtime, shared Linux seccomp worker, pollable controls and setup support. The
runtime requires glibc 2.28+; the local test viewer is built against Debian 12 with required symbols up to GLIBC_2.34.
`bin/picfetch-linux-arm64.tar.gz` includes the executable and notices. The user will qualify native analysis, kernel
enforcement and GUI behavior on ARM64 hardware. Pure BPF decisions and cross-compilation do not claim that hardware
qualification. See the
[Linux ARM64 plan](plans/2026-09-11-explorer-linux-arm64.md). Intel macOS Explorer and 32-bit ARM remain unsupported;
Windows ARM64 map rendering was confirmed in the
[Windows setup plan](plans/2026-09-11-explorer-windows-setup.md).

### Similarity-map zoom trial

The current increment enforces a 50% zoom floor only above 100 cohort piles, including Fit map and automatic discovery,
while keeping every sampled thumbnail. Smaller maps retain the existing zoom range. The active plan records SDD/TDD and
verification evidence. Trial the cutoff on normal and large maps; full-library scan throughput remains later work.

### Similarity Explorer proof of concept

**Complete: tickets 01–07.** The final 50,655-source trial, fully cached reopen, recovery, memory and human acceptance
checks pass. No open tasks remain in this milestone. See
the [final qualification report](.scratch/visual-similarity-explorer/evidence/full-library-qualification-20260910/completion.md).
The development observations below are historical; this status supersedes their then-pending qualification work.

Implement a local, offline content-similarity map to trial with roughly 50,000 library images, with sampled cohort piles
opening in Grid View. The [specification](.scratch/visual-similarity-explorer/spec.md) captures the settled behavior and
proposed verification, including the accepted first round: real grouping on this Mac, opened-file-set inputs,
non-overlapping cohorts, and fixed membership while browsing. The Mac trial uses SigLIP 2 representations, HDBSCAN
grouping and UMAP projections. The user accepted the remaining defaults and test boundaries and supplied
`.scratch/visual-similarity-explorer/demo` (446 images). Ticket 01 is complete;
[ticket 02](.scratch/visual-similarity-explorer/issues/02-evaluate-local-pipeline.md)
now contains a measured real Go/native experiment under OS network denial. All 446 demo images processed successfully in
90.638 seconds, producing 36 cohorts and 90 unassigned images. Real-model tests and `make verify` pass.
The [local report](.scratch/visual-similarity-explorer/evidence/smoke-TZppbI/result/review.html)
was accepted by the user (“it looks promising. yes continue”). The
[execution plan](finished_refactorings/2026-09-09-visual-similarity-explorer.md) records TDD slices and measured
evidence. The [trial contract](.scratch/visual-similarity-explorer/decisions.md)
and [protocol](.scratch/visual-similarity-explorer/evaluation-protocol.md) retain accepted choices. Persistent analysis
is included in the later ticket sequence.

The native completed-map/Grid View/return path is implemented, with tested cohort navigation, cancellation, staleness,
command isolation and real worker network denial. Entry maximizes the window, including uncached cohort image
navigation. Native synthetic QA, actual offline tests, build/vet and all Linux race partitions pass (the affected shard
was rerun after a test-driver fix). Progressive delivery now publishes every 30 processed images. Continuing piles
retain their positions, open cohorts remain frozen, and Shift-scroll pans. Stack spacing, preview scattering and fitted
borders address the native feedback. The same 446-image benchmark completed in 101.088 seconds (first map: 9.840s), with
46 cohorts and 55 unassigned after the density adjustment, versus 90 before. The final `make verify` and real offline
`make explorer-ui-test` pass. Counts depend on input order and do not establish semantic accuracy. Favorite
representation persistence and source/failure recovery have been implemented; full-library qualification remains ticket
07. The supplied demo does not qualify the intended roughly 50,000-image trial. The current batch HDBSCAN/UMAP
experiment has cancellation and scaling limits; these must be resolved before selecting the full-library engine.

The user has now started a 50,655-file run and requested live KPI monitoring.
[Resource status](.scratch/visual-similarity-explorer/evidence/full-library-live-20260909T150213Z/status.md)
contains 167 samples at 15-second intervals. Aggregation has stopped with the worker; the user reported approximately
6,200 scanned images. No successful full-library qualification is claimed. Focused native tests resumed for the
subsequent requests;
`development-intervals.log` records their timing and annotation gaps. The running app is not replaced. Manual map
updates (optional periodic updates), automatic zoom-out, duplicate representative preparation, favorite representation
reuse, three persisted settings, and explicit release of map image resources now pass focused UI/real-engine boundaries.
The complete Explorer native suite (18.559s), adjacent package tests, `make verify`
(including Linux race tests) and `make build` pass for this increment. Updated executable: `bin/picfetch`; the running
old trial bundle has not been replaced.

### Similarity tag overlay and filtering

Implemented; updated native library-quality verdict remains open. The
[tag contract](.scratch/visual-similarity-explorer/tag-overlay.md) and
[evidence](.scratch/visual-similarity-explorer/evidence/tag-overlay/README.md)
record the provider/UI and real offline-engine coverage. All tags start active; counts remain stable while filtering,
unknown content stays accessible, and open cohorts retain their complete membership. Labeling uses small embedded SigLIP
2 text prototypes against existing image representations. It uses no per-image text inference or new runtime assets.
Broad/mixed content can be missed or mislabeled; 31 tags and a provisional score threshold do not establish full-library
semantic accuracy. Custom traits remain separate; reusable presets are implemented below. Final real offline/native
suites, `make verify`, refreshed build checks and
`make build` pass. The updated executable is `bin/picfetch`.

Semantic tag vectors are now stored as readable, tag-keyed JSON and decoded when the tagger starts. Every float32 bit
and the original numeric digest are preserved; the generator reproduces the JSON exactly. Ordinary contributor builds
require no vector-generation step. See
[regeneration instructions](scripts/explorertags/README.md).

### Similarity explorer recovery

Ronin accepted native interruption/source-change recovery on 2026-09-10:
“Hey Pico, the recovery behavior is fine.” Refreshed evidence covers native failure/retry, cancellation/restart,
committed source-save recovery with frozen cohort identities and in-flight file-set replacement. Six worker exits, ten
framebuffer observations and normal app exit are retained. Strengthened feedback and actual-worker UI cancellation
guards were negatively verified; production recovery code is unchanged. Real-engine/viewer suites, focused native race
and
`make build` pass. Refreshed `make verify` exited 0 with all Docker race partitions passing; ticket 06 is complete.
[Task 06 evidence](.scratch/visual-similarity-explorer/evidence/task06-finalization-20260910/README.md)
records the trial and preserves the distinction from ticket 07's full-library recovery qualification.

### Native Explorer qualification after reusable presets

Reusable presets are implemented: a global local searchable library, AND rules for existing visual tags, file type,
oriented dimensions/orientation, camera make/model and EXIF calendar dates; metadata-only rules are supported. Explicit
previews protect existing cohorts and freeze reviewed candidates. Linked edits return removed members to persistent
Unassigned, preserve pending members, and keep other Favorites unchanged. Deleting a definition keeps its cohort.
Favorite memberships and rule definitions persist separately, including legacy migration.

`make explorer-evaluate TRIAL=library` now collects isolated native evidence with network denial before source reads, a
retained executable, numbered causal events, observed process exits and sampled process-group RSS. Collection is
separate from qualification.
See [current evidence](.scratch/visual-similarity-explorer/evidence/presets-native-20260910/README.md).

Native trial records now include source-free foreground map geometry and a camera snapshot before cohort departure. A
generated-source native replay crosses 100/101 piles, verifies the 50% zoom floor and retains a 16-member grid across
growth to 129 piles with an unchanged return camera. This closes the bounded geometry-evidence gap; representative
usability remains to be tried.
`make explorer-ui-test`, the full `make verify` gate and `make build` pass.
See [large-map trace evidence](.scratch/visual-similarity-explorer/evidence/large-map-trace-20260910/README.md).

Task 04 finalization fixed automatic fitting moving the map camera while a cohort or image was open. A new TDD guard
covers partial/final publications, frozen membership and current grouping on reopen. The real native replay passes with
fitting enabled: 600 inputs, zero failures, 20 observed publications and a preserved departure camera. Ronin accepted
native during-analysis usability on 2026-09-10 and authorized closure; task 04 is done.
See [finalization evidence](.scratch/visual-similarity-explorer/evidence/task04-finalization-20260910/README.md). Final
`make explorer-test`, `make explorer-ui-test`, `make verify` and
`make build` pass, including both new camera cases under Linux race detection.

Ronin's telemetry report identified the pinned ONNX Runtime's default-enabled Microsoft collector. Image analysis and
the tag-generation tool now set the full opt-out before native library loading, in addition to existing OS network
denial. A real native environment observer guards initialization order; the tag tool reproduces identical vectors. The
exact reported process and whether its connection succeeded remain unestablished.

Task 05's refreshed automated native trial passes: 36/36 reused on warm reopen and full app restart with zero inference;
one changed/corrupt/incompatible entry reanalyzes once; cache-off reanalyzes all sources. The isolated representation
cache is 630.8 KiB. Fresh-viewer and nanosecond-invalidation guards were negatively
verified. [Task 05 evidence](.scratch/visual-similarity-explorer/evidence/task05-finalization-20260910/README.md)
keeps public-fixture observations separate from library-scale qualification. The supplied 446-image demo also reused all
records without warm inference (79.072s cold, 1.412s warm; 7.31 MiB temporary cache, subsequently removed). These
measurements overlapped verification. Final `make explorer-ui-test`,
`make verify` and `make build` all pass; production cache code is unchanged. Ronin confirmed his native testing and
explicitly authorized closure on 2026-09-10; task 05 is complete.

Ticket 07 is now complete. The actual final build accounts for 50,655 sources, with 50,569 successes and 86 failures.
Unchanged Favorite reopen reuses every successful representation with zero inference; the warm pass takes 322.015s,
including 302.144s grouping. Cached five-member grid/image/map return preserves membership and camera.
Progressive/frozen browsing, large-map zoom-floor use, granularity rearrangement, invalid-cache repair, cancellation and
source-change recovery all have representative native evidence and recorded human acceptance.

Live Go heap falls from 1,008.14 MiB on the loaded/browsed map to 45.84 MiB after list closure and scavenging, against
21.12 MiB startup. Residual widget/metadata and native memory remain documented; RSS alone is not treated as live
retention. The original Favorite's 50,569 valid analysis records occupy 790.30 MiB logically. The app and observer exit
normally with complete collection and no forced stop. Real-engine UI/evaluator tests, the canonical `make verify` race
gate and normal
`make build` pass.
The [final report](.scratch/visual-similarity-explorer/evidence/full-library-qualification-20260910/completion.md)
retains timings, exact settings, resource/storage accounting, privacy boundaries, replay checks and platform/measurement
limits. **Milestone tasks 01–07 are done; open tasks: none.** The implementation plan is archived; no commit was
created.

### Full-library similarity throughput

`make explorer-profile` now measures the production worker on up to 512 sources, using an isolated temporary favorite
for cold and cached passes. Its metadata-only trace records completed/failed/reused counts, inference attempts,
publications, decode/inference/preview/cache/tag time and UMAP/HDBSCAN/hierarchy stages. The 446-image demo completed in
89.484s cold and 1.317s cached, with zero failures and complete reuse. Decode (33.768s) and preview generation (26.810s)
are material alongside inference (27.320s); grouping was 1.073s. Preserve image quality when investigating those costs.
See the [measurement record](.scratch/visual-similarity-explorer/evidence/throughput-gSB1MB/README.md).

The
subsequent [source-throughput increment](.scratch/visual-similarity-explorer/evidence/source-throughput-20260910/README.md)
removed color-interface allocations from canonical EXIF correction. Its matched before/after demo runs completed in
88.727s and 82.153s (7.4% less total time), with the decode stage falling from 32.556s to 24.863s (23.6% less time). All
representations/tags, previews and groups/positions matched across cold and warm runs. The follow-up preview resampling
increment retains the CatmullRom kernel and sixteen-bit YCbCr colors, converts each source row once, and keeps only the
filtered rows needed by the current output row. The real-engine 3072x4096 JPEG guard fell from 74.6 MB to 51.4 MB of
allocated Go memory. Exact pixel comparisons and matched 446-source fingerprints pass. The demo completed in 74.608s
versus 82.113s before (9.1% less time); preview generation took 20.750s versus 27.053s (23.3% less). These are single
local observations. Native suites, the combined `make verify` race gate and `make build` passed; production measurements
and verification are recorded in the
[preview resampling evidence](.scratch/visual-similarity-explorer/evidence/preview-resampling-20260910/README.md).

This bounded command does not measure native paint/interaction or qualify 50k. The later 50,655-input native run
supplies completed-map stage/count timings; progressive and cached full-library timings still need qualification. A
declining per-image encoding rate has not been established; mixed source sizes/formats make simple windows insufficient
evidence of an order-dependent slowdown.

### Large similarity-map interaction

The original roughly 50k-image trial reached a visible map without a reported crash, but the user observed severe lag
zoomed out and much smoother interaction zoomed in. The live capture recorded a 9.532 GiB viewer peak during
publication, settling near 5 GiB, plus main-thread CPU bursts and repeated texture uploads during interaction. Map
layout visits every pile; distant piles retain all sampled thumbnails, and smooth image rescaling recreates textures on
zoom. The user chose a minimum zoom for large maps and rejected reducing thumbnail counts. The current increment
preserves all samples, clamps maps above 100 piles to 50% zoom, and keeps the camera stable when automatic expansion
reaches that floor. Native replay results are recorded in the active plan. The later 50,655-input run supplies source
accounting/stage timings and accepted completed-map responsiveness; progressive/cache/recovery qualification stays open.
Viewport preview retention now releases distant decoded pixels, keeps a warm margin, and preserves all samples and
native rendered appearance. Corrected 30/300/3,000-image synthetic replays use the actual 160px JPEG preview contract;
the earlier 256px replay overstated per-preview pixel cost. Completed-map usability is accepted; progressive interaction
at scale and explicit coverage above 100 piles remain open. The 50k edge case is not a normal-use target.
See [render-transition evidence](.scratch/visual-similarity-explorer/evidence/render-transition-20260909/README.md).

## LATER

### Revisit HEIC and verifier dependencies at their next upgrades

[MA-023](needs_refactoring.md#ma-023) tracks retiring the HEIC fork when an approved official release includes its leak
fix. [MA-024](needs_refactoring.md#ma-024)
tracks measuring verifier dependency cost at its next major upgrade. Both retain their separate upgrade triggers;
neither starts immediate work.

### Retire the GitHub-hosted Intel macOS runner before August 2027

GitHub plans to retire `macos-15-intel`, its final hosted x86_64 macOS runner, in August 2027. Before then, decide
whether PicFetch will stop shipping an Intel macOS archive or retain it through another build path. If Intel support
remains, replace the `macos-15-intel` release job with a tested alternative; otherwise remove the x86_64 artifact and
update the release and installation documentation. The native Apple-silicon build is not affected by Rosetta's
retirement.

## not deemed worth implementing (edge cases)

- **Retained decoded map tiles (MA-025):** Accepted by the user on 2026-09-09. The map loads only when opened, and
  checking the geolocation of thousands of images is outside expected use. The upstream decoded-tile cache remains
  unbounded; its long-session impact is unmeasured. No further measurement or implementation work is planned.
  See [MA-025](needs_refactoring.md#ma-025).

- Windows releases are not Authenticode-signed. Controlled Folder Access and SmartScreen both judge by signature and
  reputation as well as by which program is writing, so an unsigned `picfetch.exe` can still be blocked even with the
  in-process swap (see Done → Bugfix above, where the block would now name `picfetch.exe` instead of `cmd.exe`). The
  real remaining fix is signing the Windows release build — Azure Trusted Signing or a purchased certificate — in
  `.github/workflows/release.yml`, which runs no
  `signtool` today.

- There is a bug in the Windows Version: WHen in Gridview, multiselect via the space key works, but when trying it with
  mouse and Ctrl key, it does not. Holding the Ctrl key down and clicking on an image does not select it but instead
  opens it. Observation, when pushing the Ctrl key at exactly the same time as clicking on the image, it actually works,
  and the image is selected. (this seems to be a bug in fyne, created an issue, sorry Windows users)

### Qodana drops detected duplicates during serialisation (upstream)

At `210fee5` (run `33270269940`), the IDE reports 71 `DuplicatedCode`
fragments and the CI SARIF reports 63, with CI's 63 a strict subset of the IDE's 71. The 8 fragments CI is missing are 7
in
`internal/imaging/loader_test.go` and 1 at
`internal/update/tufroot_test.go:173`. That run's own `log/idea.log` carries exactly 3
`#o.j.q.s.i.r.g.DuplicatesProblem` "Can't find duplicate problem in db" warnings, naming exactly those two files and no
others, emitted immediately after the line `The Project analysis stage completed in 41s` — so Qodana's own log shows
detection succeeded and serialisation into the report/SARIF failed afterwards. This is an upstream defect, not a
picfetch config problem: nothing here suppresses or excludes those two files, and the drop happens before any
project-side filtering runs.

`qodana.yaml`'s new `_test.go` exclusion (see Done → Internal above) makes this defect invisible going forward in this
repository, because every dropped fragment happens to live in a test file that the exclusion now removes from the
inspection entirely — recorded here so the defect is not lost along with the rule that used to surface it. Of the
12-fragment CSV-to-SARIF gap at `210fee5`, these 8 serialisation losses are one part; the other 4 are the
source-suppressed production fragments in the orientation pixel loops recorded above, so nothing about that gap is left
open — only the underlying serialisation defect itself is. See
`finished_refactorings/2026-08-29-qodana-evidence.md` for the decoded byte offsets and anchoring detail, and
`plans/2026-08-29-qodana-serialisation-bug-report.md`, Task 8's draft of the upstream report text — as of this writing
not yet submitted to JetBrains; check that file for whether it has been sent since.
