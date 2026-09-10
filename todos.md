# PicFetch — TODOs

## Done

### What's Changed

#### New Features

- **Explorer and Mosaic access.** Shift+S opens the Visual Similarity Explorer;
  Shift+M opens Mosaic from the loaded collection, or the current selection/
  filtered result in Grid View. Generate Image Mosaic now lives in Window.
  Plain M/S retain merge/sort, and search/dialogs keep keyboard ownership.
  Explorer Presets is beside Unassigned on the right. Native public-fixture
  captures verify the toolbar and loaded-source Mosaic window; the full-library
  qualification remains ticket 07. Final `make verify`, real offline UI tests,
  native evidence checks and `make build` pass. See [continuation evidence](.scratch/visual-similarity-explorer/evidence/task07-finalization-20260910/README.md).

- **Reusable similarity presets.** Global local tag/metadata rules, explicit
  previews, linked cohort edits, persistent Unassigned returns and Favorite
  membership links, independent of the analysis cache.
- **Isolated native Explorer collection.** `TRIAL=library` retains the app,
  records receipt/application/exit boundaries and process-group RSS, and keeps
  technical collection separate from full-library qualification.

- **Create cohorts from Unassigned images.** Select at least two images and use
  Analyze to review their shared visual tags, matching filenames and a group
  name. Include other matching Unassigned images or keep only the selection.
  Favorite-based collections save their named cohorts and exact members for
  reopening; ordinary/mixed collections keep them for the current map.
  Automatic updates and granularity preserve user-defined groups. Verified with
  `make verify`, `make build`, offline integration tests and native restart QA;
  see the [implementation record](plans/2026-09-09-visual-similarity-explorer.md#create-cohorts-from-unassigned--2026-09-10).

- **Refine similarity maps without rescanning.** Collapse the tag sidebar,
  click a tag's count to browse only matching images, and use the top-right
  Granularity slider to combine related stacks or restore the original groups.
  Arrow keys highlight stacks in the chosen direction; Enter opens the active
  stack and +/- zoom. Open grids keep their captured members during updates.

- **Filter similarity maps by semantic tags.** A local 31-tag catalogue labels
  subjects and scenes from fresh or saved image representations. The tag panel
  starts fully selected, includes Untagged, and provides All tags/Clear tags.
  Counts show unique images across the complete map; OR filtering preserves
  cohort membership, camera, and choices across updates and Grid View visits.
  No additional model download is needed to use tags. Ronin accepted the tag
  and cohort quality in his tested experience on September 10. Full-library
  measurements and reusable grouping rules remain separate.

### Keep Random mosaics varied and add Shelf

Mosaic keeps the original seeded **Random** arrangement as the default and adds
an ordered **Shelf** arrangement in the Advanced settings. Both modes use the
same chosen source pool, frame, size, overlap, and shadow settings. Random
retains rotation through 90 degrees; Shelf is deliberately axis-aligned and
hides its rotation control. Missing or unknown saved layout values safely
restore Random.

Shelf carries the strict final-image visibility and overlap-response checks,
including every retained photo occurrence across frames, shadows, and
source-pool sizes. It remains axis-aligned, but Overlap stays available and has
a measured effect. Random retains its original primary-card visibility guard,
varied placement, and rotation checks through 90 degrees. Its gap-repair cards
can still be fully covered, so the strict every-occurrence condition
intentionally does not apply to Random. See the [implementation
record](finished_refactorings/2026-09-08-mosaic-rotation-overlap.md) for the selected contract
and the remaining limitation.

The user approved the current visual result. Earlier apparent gap-filling or
maze-like output is historical behavior, not a regression introduced here.

#### Bugfix

- **Keep the similarity-map view during browsing.** New analysis results keep
  the departure camera while a cohort or image is open, even with automatic
  fitting enabled. Returning and reopening still uses current grouping.
- **Disable native model telemetry.** Image analysis and tag generation opt out
  before ONNX Runtime loads, preventing its telemetry initialization. Existing
  OS network denial remains in place.

- **Release browsed images on Close Files.** Clear thumbnail-cache and recycled
  grid-cell image references, dismiss an open cohort grid, and keep reopening
  usable. Covered by measured viewer regressions and a generated-source native
  replay; the original full-library resource retest remains open.

- **Reduce EXIF correction overhead during similarity analysis.** Direct pixel
  access removes per-pixel color allocations while preserving the exact oriented
  image. On the 446-image demo, cold analysis fell from 88.7s to 82.2s; source
  decoding/correction fell from 32.6s to 24.9s. Representations, tags, previews
  and grouping matched exactly before/after. These are bounded local timings.

- **Apply map Granularity when the slider is released.** Dragging moves the
  thumb without repeatedly rebuilding the map. Incoming results retain the
  last applied grouping until release; clicks and keyboard adjustments still
  apply immediately.

- **Retain decoded similarity previews near the viewport.** Keep every sample
  and its compressed source, prepare nearby piles before they enter the screen,
  and release distant decoded pixels/textures. A wider release margin avoids
  repeatedly decoding piles during small reversals. Selection also reuses
  unchanged pixels. Native synthetic comparisons preserve identical appearance;
  the 3,000-image replay reduced retained construction heap from about 124 MiB
  to 35 MiB. Refills add a few milliseconds in the measured runs; normal-sized
  browsing and visual quality remain the priority, with no 50k guarantee.

- **Keep large similarity maps at a usable zoom.** Maps above 100 cohort piles
  stop at 50% zoom, including manual and automatic fitting. Every sampled
  thumbnail remains, smaller maps retain their zoom range, and new discoveries
  stop moving the camera at the floor. Pan or use direction keys to browse
  beyond the viewport.

- **Reduce large similarity-map update overhead.** Display messages omit the
  inference vectors retained by the worker/cache, and stack placement checks
  nearby cells and ring perimeters. The local 1,600-stack packing replay fell
  from roughly 241 ms to 30 ms. This measures publication overhead, not the
  full-library encoding rate; batch regrouping still grows with library size.

- **Recover similarity exploration after file changes or analysis failures.**
  Failed and incomplete runs can be retried. Source writes and removals retire
  stale maps while preserving surviving cohort members; missing-file navigation
  stays within that cohort. Late results cannot overwrite a restarted map.

Grid View now keeps the ring visible while scrolling with the mouse wheel or
trackpad, and live duplicate merges preserve the viewport. Explicit selection
and the displayed image stay unchanged. Native scrollbar movement keeps its
existing behavior; the next duplicate reflow reconciles the ring.

Shift+D now opens the highlighted shot's known duplicate group immediately
while analysis continues, including hidden copies. Live updates follow that
same source, and pending updates preserve the group. Navigation, opening a
copy, exits, reordering, and sensitivity changes have regression coverage.
Both manuals are updated; all SDD/TDD acceptance checks and `make verify` pass.
See the [implementation record](finished_refactorings/2026-09-09-grid-duplicate-browse-during-scan.md).

#### Internal

Native Explorer trial records now link worker receipts and map application by
event identity, record foreground map/grid/image/comparison state, and retain
only hashes/counts of the actual grid results. Both keyboard and button map
returns use the same evidence path. This allows a frozen cohort to be checked
across live publications without retaining filenames or image content.

Local race verification now keeps each attempt's four raw streams, console,
exit status, and available Docker/cgroup memory diagnostics in a unique host
directory under `.scratch/race-runs/`. Failed and interrupted runs retain their
evidence and failure status; diagnostic collection precedes container cleanup.
The existing 16 GiB mitigation and subsequent passing gates close the memory
investigation. The original September 7 package-only failure remains unexplained
in the [implementation and evidence record](finished_refactorings/2026-09-09-local-race-evidence.md).
Focused race tests, deliberate regression checks, and one default `make verify`
pass; its four complete streams and final memory counters survive cleanup.

### Complete the September 6 maintainability audit

The audit's required and selected conditional work (tickets 01–30) is complete.
The retained implementation records cover input safety, file identity, cache
consistency, background lifetimes, duplicate grouping, preview contention,
command admission and native/package validation, with their common gates.
On September 9 the user reported successful Windows 11 ARM and x64 testing
and accepted the remaining detailed Windows checks as edge cases, closing
MA-020. WACK certification is waived for this audit closeout; no new WACK pass
is claimed. The export-window checkbox also has user-reported x64 validation.

See the [implementation record](finished_refactorings/2026-09-06-maintainability-plan.md),
[specification](finished_refactorings/2026-09-07-maintainability/spec.md),
[tickets](finished_refactorings/2026-09-07-maintainability/ticket-breakdown.md),
and [Windows acceptance](finished_refactorings/2026-09-07-maintainability/windows-test-todo.md).
The native macOS Copy Selection golden mismatch remains documented; the
canonical Linux golden gate passed. MA-025 is an accepted edge case below.

## TODO

### Similarity-map zoom trial

The current increment enforces a 50% zoom floor only above 100 cohort piles,
including Fit map and automatic discovery, while keeping every sampled
thumbnail. Smaller maps retain the existing zoom range. The active plan records
SDD/TDD and verification evidence. Trial the cutoff on normal and large maps;
full-library scan throughput remains later work.

### Visual similarity explorer proof of concept

Implement a local, offline content-similarity map to trial with
roughly 50,000 library images, with sampled cohort piles opening in Grid View.
The [specification](.scratch/visual-similarity-explorer/spec.md) captures the
settled behavior and proposed verification, including the accepted first
round: real grouping on this Mac, opened-file-set inputs, non-overlapping
cohorts, and fixed membership while browsing. The Mac trial uses SigLIP 2
representations, HDBSCAN grouping and UMAP projections. The user accepted the
remaining defaults and test boundaries and supplied
`.scratch/visual-similarity-explorer/demo` (446 images). Ticket 01 is complete;
[ticket 02](.scratch/visual-similarity-explorer/issues/02-evaluate-local-pipeline.md)
now contains a measured real Go/native experiment under OS network denial.
All 446 demo images processed successfully in 90.638 seconds, producing
36 cohorts and 90 unassigned images. Real-model tests and `make verify` pass.
The [local report](.scratch/visual-similarity-explorer/evidence/smoke-TZppbI/result/review.html)
was accepted by the user (“it looks promising. yes continue”). The
[execution plan](plans/2026-09-09-visual-similarity-explorer.md) records TDD
slices and measured evidence. The [trial contract](.scratch/visual-similarity-explorer/decisions.md)
and [protocol](.scratch/visual-similarity-explorer/evaluation-protocol.md) retain
accepted choices. Persistent analysis is included in the later ticket sequence.

The native completed-map/Grid View/return path is implemented, with tested
cohort navigation, cancellation, staleness, command isolation and real worker
network denial. Entry maximizes the window, including uncached cohort image
navigation. Native synthetic QA, actual offline tests, build/vet and all Linux
race partitions pass (the affected shard was rerun after a test-driver fix).
Progressive delivery now publishes every 30 processed images. Continuing piles
retain their positions, open cohorts remain frozen, and Shift-scroll pans.
Stack spacing, preview scattering and fitted borders address the native feedback.
The same 446-image benchmark completed in 101.088 seconds (first map: 9.840s),
with 46 cohorts and 55 unassigned after the density adjustment, versus 90 before.
The final `make verify` and real offline `make explorer-ui-test` pass. Counts
depend on input order and do not establish semantic accuracy. Favorite representation persistence and source/failure recovery have been implemented;
full-library qualification remains ticket 07.
The supplied demo does not qualify the intended roughly 50,000-image trial.
The current batch HDBSCAN/UMAP experiment has cancellation and scaling limits;
these must be resolved before selecting the full-library engine.

The user has now started a 50,655-file run and requested live KPI monitoring.
[Resource status](.scratch/visual-similarity-explorer/evidence/full-library-live-20260909T150213Z/status.md)
contains 167 samples at 15-second intervals. Aggregation has stopped with the
worker; the user reported approximately 6,200 scanned images. No successful
full-library qualification is claimed. Focused native tests resumed for the subsequent requests;
`development-intervals.log` records their timing and annotation gaps. The running
app is not replaced. Manual map updates (optional periodic updates), automatic
zoom-out, duplicate representative preparation, favorite representation reuse,
three persisted settings, and explicit release of map image resources now pass
focused UI/real-engine boundaries.
The complete Explorer native suite (18.559s), adjacent package tests, `make verify`
(including Linux race tests) and `make build` pass for this increment. Updated
executable: `bin/picfetch`; the running old trial bundle has not been replaced.

### Similarity tag overlay and filtering

Implemented; updated native library-quality verdict remains open. The
[tag contract](.scratch/visual-similarity-explorer/tag-overlay.md) and
[evidence](.scratch/visual-similarity-explorer/evidence/tag-overlay/README.md)
record the provider/UI and real offline-engine coverage. All tags start active;
counts remain stable while filtering, unknown content stays accessible, and
open cohorts retain their complete membership. Labeling uses small embedded
SigLIP 2 text prototypes against existing image representations. It uses no
per-image text inference or new runtime assets. Broad/mixed content can be
missed or mislabeled; 31 tags and a provisional score threshold do not establish
full-library semantic accuracy. Custom traits remain separate; reusable presets are implemented below.
Final real offline/native suites, `make verify`, refreshed build checks and
`make build` pass. The updated executable is `bin/picfetch`.

Semantic tag vectors are now stored as readable, tag-keyed JSON and decoded
when the tagger starts. Every float32 bit and the original numeric digest are
preserved; the generator reproduces the JSON exactly. Ordinary contributor
builds require no vector-generation step. See
[regeneration instructions](scripts/explorertags/README.md).

### Similarity explorer recovery

Ronin accepted native interruption/source-change recovery on 2026-09-10:
“Hey Pico, the recovery behavior is fine.” Refreshed evidence covers native
failure/retry, cancellation/restart, committed source-save recovery with frozen
cohort identities and in-flight file-set replacement. Six worker exits, ten
framebuffer observations and normal app exit are retained. Strengthened feedback
and actual-worker UI cancellation guards were negatively verified; production
recovery code is unchanged. Real-engine/viewer suites, focused native race and
`make build` pass. Refreshed `make verify` exited 0 with all Docker race
partitions passing; ticket 06 is complete.
[Task 06 evidence](.scratch/visual-similarity-explorer/evidence/task06-finalization-20260910/README.md)
records the trial and preserves the distinction from ticket 07's full-library
recovery qualification.

### Native Explorer qualification after reusable presets

Reusable presets are implemented: a global local searchable library, AND rules
for existing visual tags, file type, oriented dimensions/orientation, camera
make/model and EXIF calendar dates; metadata-only rules are supported. Explicit
previews protect existing cohorts and freeze reviewed candidates. Linked edits
return removed members to persistent Unassigned, preserve pending members, and
keep other Favorites unchanged. Deleting a definition keeps its cohort. Favorite
memberships and rule definitions persist separately, including legacy migration.

`make explorer-evaluate TRIAL=library` now collects isolated native evidence with
network denial before source reads, a retained executable, numbered causal events,
observed process exits and sampled process-group RSS. Collection is separate from
qualification. See [current evidence](.scratch/visual-similarity-explorer/evidence/presets-native-20260910/README.md).

Native trial records now include source-free foreground map geometry and a
camera snapshot before cohort departure. A generated-source native replay
crosses 100/101 piles, verifies the 50% zoom floor and retains a 16-member grid
across growth to 129 piles with an unchanged return camera. This closes the
bounded geometry-evidence gap; representative usability remains to be tried.
`make explorer-ui-test`, the full `make verify` gate and `make build` pass.
See [large-map trace evidence](.scratch/visual-similarity-explorer/evidence/large-map-trace-20260910/README.md).

Task 04 finalization fixed automatic fitting moving the map camera while a
cohort or image was open. A new TDD guard covers partial/final publications,
frozen membership and current grouping on reopen. The real native replay
passes with fitting enabled: 600 inputs, zero failures, 20 observed publications
and a preserved departure camera. Ronin accepted native during-analysis
usability on 2026-09-10 and authorized closure; task 04 is done.
See [finalization evidence](.scratch/visual-similarity-explorer/evidence/task04-finalization-20260910/README.md).
Final `make explorer-test`, `make explorer-ui-test`, `make verify` and
`make build` pass, including both new camera cases under Linux race detection.

Ronin's telemetry report identified the pinned ONNX Runtime's default-enabled
Microsoft collector. Image analysis and the tag-generation tool now set the
full opt-out before native library loading, in addition to existing OS network
denial. A real native environment observer guards initialization order; the
tag tool reproduces identical vectors. The exact reported process and whether
its connection succeeded remain unestablished.

Task 05's refreshed automated native trial passes: 36/36 reused on warm reopen
and full app restart with zero inference; one changed/corrupt/incompatible entry
reanalyzes once; cache-off reanalyzes all sources. The isolated representation
cache is 630.8 KiB. Fresh-viewer and nanosecond-invalidation guards were negatively
verified. [Task 05 evidence](.scratch/visual-similarity-explorer/evidence/task05-finalization-20260910/README.md)
keeps public-fixture observations separate from library-scale qualification.
The supplied 446-image demo also reused all records without warm inference
(79.072s cold, 1.412s warm; 7.31 MiB temporary cache, subsequently removed).
These measurements overlapped verification. Final `make explorer-ui-test`,
`make verify` and `make build` all pass; production cache code is unchanged.
Ronin confirmed his native testing and explicitly authorized closure on
2026-09-10; task 05 is complete.

Tickets 01-06 are complete. The only open milestone ticket is 07's integrated
full-library qualification. It includes
live progressive/frozen browsing and
recovery/cache trials at library scale; representative usability of the 50% zoom
floor above 100 piles; representative event-to-visible paint measurements; final
full-library verification of post-close memory reclamation and the remaining
integrated qualification. Ronin has
accepted completed-map responsiveness on the 50,655-input run. The direct
desktop-control bridge failed, so precise physical-input timing remains
unmeasured. Public-fixture replay evidence remains distinct from his trial.
Ronin accepted tag matching and cohort quality in his tested experience on
September 10; this acceptance remains recorded.

The September 10 finalization request revalidated the retained 50,655-input
evidence and completed the requested shortcut/toolbar/Mosaic-menu amendments.
The full-library root or another manual drop is still needed for a fresh scale
trial; the current source-free records do not contain that location. Individual
fixture tools cannot substitute for the outstanding integrated trial. No new
full-library acceptance is inferred from the public-fixture run.

The subsequent [native progressive replay](.scratch/visual-similarity-explorer/evidence/progressive-native-20260910/README.md)
processed 600 public fixture copies with zero failures and observed all 20
publications. A ten-image partial grid retained its exact source digest across
a later publication; image/grid/map return preserved the camera. The first
partial map was observed at 2.244s and visible publications at median 41.794ms
after receipt, including capture/readback/polling overhead. Callback pan/zoom
and these repeated-image results provide a bounded baseline; direct input and
representative full-library qualification remain open.

This increment passed `make explorer-ui-test`, `make verify` (complete Docker
race suite), `make build` and the native trial-runner subprocess test. Ronin
authorized another same-library comparison; the isolated profiling client and
numeric resource observer completed successfully. His manual drop admitted
50,655 inputs: 50,569 succeeded, 86 failed, none reused. The worker completed
in 73m 24.702s; grouping/layout took 297.784s and UI construction 425.429ms.
Ronin reported the rendered view was “very responsive.” Observed viewer peak
physical footprint through 13:00:39 UTC was 2.013 GiB versus the baseline's
9.532 GiB (78.9% lower). This supplies exact stage/count/resource evidence;
the remaining scale trials stay open. Normal app exit subsequently passed,
with collected evidence, exit code 0 and no worker left behind. Ronin confirmed
the view looks much better than before. The final observed viewer footprint
peak after browsing was 2.474 GiB, 74.0% below the baseline. See
[full-library report](.scratch/visual-similarity-explorer/evidence/full-library-trial.md).

The user then closed the list while keeping the app running. Five cohort/grid
round trips were recorded, including a 16,389-member cohort. No worker remains;
at 13:05 UTC the app retained 1.634 GiB physical footprint (2.779 GiB resident).
The later sampled viewer peak was 2.474 GiB. The controlled post-close check
now identifies and fixes retained thumbnail-cache pixels and recycled grid-cell
images; Close Files also dismisses an open cohort grid. A 512-source native
replay returns live Go heap to 23.0–23.9 MiB after two browse/close cycles
(21.1 MiB baseline), with zero cached source thumbnails. Native process footprint
remains above startup, and the original full-library observation has not been
repeated. Keep that scale/resource qualification open. The user previously quit
the original app normally; process exit did not prove its in-app close released
live objects. See [controlled memory evidence](.scratch/visual-similarity-explorer/evidence/list-close-memory-20260910/README.md).

### Full-library similarity throughput

`make explorer-profile` now measures the production worker on up to 512 sources,
using an isolated temporary favorite for cold and cached passes. Its metadata-only
trace records completed/failed/reused counts, inference attempts, publications,
decode/inference/preview/cache/tag time and UMAP/HDBSCAN/hierarchy stages. The
446-image demo completed in 89.484s cold and 1.317s cached, with zero failures
and complete reuse. Decode (33.768s) and preview generation (26.810s) are material
alongside inference (27.320s); grouping was 1.073s. Preserve image quality when
investigating those costs. See the [measurement record](.scratch/visual-similarity-explorer/evidence/throughput-gSB1MB/README.md).

The subsequent [source-throughput increment](.scratch/visual-similarity-explorer/evidence/source-throughput-20260910/README.md)
removed color-interface allocations from canonical EXIF correction. Its matched
before/after demo runs completed in 88.727s and 82.153s (7.4% less total time),
with the decode stage falling from 32.556s to 24.863s (23.6% less time). All
representations/tags, previews and groups/positions matched across cold and
warm runs. The follow-up preview resampling increment retains the CatmullRom
kernel and sixteen-bit YCbCr colors, converts each source row once, and keeps
only the filtered rows needed by the current output row. The real-engine
3072x4096 JPEG guard fell from 74.6 MB to 51.4 MB of allocated Go memory.
Exact pixel comparisons and matched 446-source fingerprints pass. The demo
completed in 74.608s versus 82.113s before (9.1% less time); preview generation
took 20.750s versus 27.053s (23.3% less). These are single local observations.
Native suites, the combined `make verify` race gate and `make build` passed;
production measurements and verification are recorded in the
[preview resampling evidence](.scratch/visual-similarity-explorer/evidence/preview-resampling-20260910/README.md).

This bounded command does not measure native paint/interaction or qualify 50k.
The later 50,655-input native run supplies completed-map stage/count timings;
progressive and cached full-library timings still need qualification. A declining
per-image encoding rate has not been established; mixed source sizes/formats make simple windows
insufficient evidence of an order-dependent slowdown.

### Large similarity-map interaction

The original roughly 50k-image trial reached a visible map without a reported
crash, but the user observed severe lag zoomed out and much smoother interaction
zoomed in. The live capture recorded a 9.532 GiB viewer peak during publication,
settling near 5 GiB, plus main-thread CPU bursts and repeated texture uploads
during interaction. Map layout visits every pile; distant piles retain all
sampled thumbnails, and smooth image rescaling recreates textures on zoom.
The user chose a minimum zoom for large maps and rejected reducing thumbnail
counts. The current increment preserves all samples, clamps maps above 100 piles
to 50% zoom, and keeps the camera stable when automatic expansion reaches that
floor. Native replay results are recorded in the active plan. The later
50,655-input run supplies source accounting/stage timings and accepted
completed-map responsiveness; progressive/cache/recovery qualification stays open.
Viewport preview retention now releases distant decoded pixels, keeps a warm
margin, and preserves all samples and native rendered appearance. Corrected
30/300/3,000-image synthetic replays use the actual 160px JPEG preview contract;
the earlier 256px replay overstated per-preview pixel cost. Completed-map
usability is accepted; progressive interaction at scale and explicit coverage
above 100 piles remain open. The 50k edge case is not a normal-use target.
See [render-transition evidence](.scratch/visual-similarity-explorer/evidence/render-transition-20260909/README.md).

## LATER

### Revisit HEIC and verifier dependencies at their next upgrades

[MA-023](needs_refactoring.md#ma-023) tracks retiring the HEIC fork when an
approved official release includes its leak fix. [MA-024](needs_refactoring.md#ma-024)
tracks measuring verifier dependency cost at its next major upgrade. Both
retain their separate upgrade triggers; neither starts immediate work.

### Retire the GitHub-hosted Intel macOS runner before August 2027

GitHub plans to retire `macos-15-intel`, its final hosted x86_64 macOS runner, in August 2027. Before then, decide
whether PicFetch will stop shipping an Intel macOS archive or retain it through another build path. If Intel support
remains, replace the `macos-15-intel` release job with a tested alternative; otherwise remove the x86_64 artifact and
update the release and installation documentation. The native Apple-silicon build is not affected by Rosetta's
retirement.

## not deemed worth implementing (edge cases)

- **Retained decoded map tiles (MA-025):** Accepted by the user on 2026-09-09.
  The map loads only when opened, and checking the geolocation of thousands
  of images is outside expected use. The upstream decoded-tile cache remains
  unbounded; its long-session impact is unmeasured. No further measurement or
  implementation work is planned. See [MA-025](needs_refactoring.md#ma-025).

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
