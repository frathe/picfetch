## PR 18 active review loop (2026-09-11)

- [x] Validate and fix cached/live file-size limits, keyboard retry, Favorite
  directory identity, completed-map image evidence, evaluator timing/provenance,
  modal shortcuts and Intel macOS version admission using red/green regressions.
- [x] Inspect all ten post-suppression Qodana findings. Correct three mechanical
  issues; narrowly suppress seven build-tag/proper-name false positives.
  CodeQL ZIP alert 6 is dismissed with exact allowlist/test evidence.
- [x] Push 724a221 and reply to all ten Codex findings; no unresolved threads.
  Follow-up Qodana ownership warning is disproved by a negatively verified
  directory-handle closure guard and narrowly suppressed.
- [ ] Obtain fresh clean code/security reviews and passing CI on the latest commit.
  See [the SDD/TDD record](plans/2026-09-11-pr18-review-limits-evidence.md).

## Intel macOS Explorer (2026-09-11)

- [x] Add Microsoft's pinned Intel ONNX Runtime 1.23.2, with its compatible
  API 23 Go binding isolated to darwin/amd64; retain 1.29.0 elsewhere.
  Extend verified downloads, worker control, native evaluation and documentation.
  Intel Explorer requires macOS 13.4+. See [the implementation plan](plans/2026-09-11-explorer-macos-x64.md).
- [x] Add Intel macOS CI coverage for native guards, worker exit and actual
  asset installation/offline inference.
- [ ] Observe Intel CI and hardware acceptance: model inference, cancellation,
  sandbox enforcement, Explorer UI and packaged app. Cross-compilation alone
  does not qualify native behavior; the legacy Intel runtime has no current
  upstream binary updates.

## Windows Explorer setup (2026-09-11)

- [x] Add the pinned Windows x64 runtime ZIP, bounded extraction of required
  DLLs/notices, native setup command, and download-only qualification. Actual
  451.5 MB download and reuse without HTTP pass on Windows 11.
- [x] Add ONNX Runtime's explicit telemetry API opt-out before session creation,
  with initialization cleanup on failure. Clarify in the privacy policy that
  Explorer uses local process pipes and opens no localhost/model-server port.
  Native Windows synthetic-image inference passed with the explicit opt-out.
- [x] Enable the normal Windows worker following the user's choice of telemetry
  opt-out without firewall/AppContainer isolation, clarified after the local-port
  misunderstanding. Add the Windows first-use notice and privacy/manual updates;
  preserve macOS/Linux isolation and honest `OfflineVerified` evidence.
  Do not repeat the AppContainer/BFS probes after the reported host crashes.
  See [the active plan](plans/2026-09-11-explorer-windows-setup.md).
- [x] Qualify native inference, first-use setup, configured file-size limits,
  cancellation, and worker/UI restart with small serial Windows tests. Inspect
  the setup notice and action button at a small window size.
- [x] Build `bin/picfetch.exe` (Windows x64 GUI, 53.6 MB) for local testing;
  verify one synthetic image through that exact EXE and clean worker exit.
  Native vet/build, translations/manual guards, formatting and exclusions pass.
- [x] User accepted the local Windows x64 EXE on 2026-09-11: all tested behavior
  worked well.
- [ ] CI's complete Linux/race gate remains outstanding. The earlier host crash cause is unresolved; do not
  repeat the isolation probes or broad Docker/race run on this host.
- [x] Bundle architecture-pinned ONNX Runtime DLLs and upstream notices in Store
  packages; download only 372 MB model data. Verify that cache overrides cannot
  replace bundled DLLs. Declare the Store-managed C++ framework dependency and
  update privacy, notices, manuals and Store listing/certification notes.
- [x] Implement native Windows ARM64 runtime setup and cross-build support using
  the verified portable LLVM-MinGW toolchain, without Docker or system changes.
- [ ] User to test Explorer on ARM64 hardware. Cross-compilation and archive/DLL
  integrity checks do not qualify native ARM64 execution. MSIX/WACK stays in CI.
- [x] Replace the GPL clustering dependency as explicitly authorized: the pinned,
  independently MIT-licensed HDBSCAN subset now lives in `internal/hdbscan`, with
  its license/provenance and synthetic tests. Preserve the two-other-neighbor
  density setting and four-point minimum. Automatic group membership can change;
  source-versioned embedding caches and explicit saved cohorts remain usable.
  See [replacement evidence](plans/2026-09-11-permissive-clustering.md).
- [x] Include license/third-party notices and source-access information in
  standalone release packaging, retain them through Windows signing/repacking,
  and copy them into the macOS bundle. Store staging already includes them.
  Actual release archives and certification remain CI verification.

## PR 18 review fixes (2026-09-11)

- [x] Follow-up review: preserve Unassigned Analyze on Window-menu grid return,
  refuse Explorer admission during replacement scan/sort, and document Linux
  support in the README. See [implementation evidence](finished_refactorings/2026-09-11-pr18-explorer-admission.md).
  Fresh reviews and full CI results are recorded on PR 18.

- [x] Follow-up review: cohort Viewer access, duplicate-distance invalidation,
  repeated-source reuse, and consistent Linux requirements in both manuals.
  See [follow-up evidence](finished_refactorings/2026-09-11-pr18-cohort-settings-reuse.md).

- [x] Follow-up review: restore the viewer after rejecting a truncated trial,
  and carry the configured encoded-file limit into each analysis subprocess.
  See [follow-up evidence](finished_refactorings/2026-09-11-pr18-scan-worker-limits.md).

- [x] Address the initial 11 Codex findings and Qodana warnings with regression
  tests and commit-linked thread replies. Fixes cover map exit/retry, Favorites
  membership and cancellation, cohort Escape, shortcut parity, asset cancellation,
  all-failed analyses, Linux throughput, and startup/queue ownership.
  The reusable **GitHub Cortex review loop** is documented in `AGENTS.md`.
  Changed tests run locally; full CI and successive reports are checked on
  [PR 18](https://github.com/frathe/picfetch/pull/18).
  See [implementation evidence](finished_refactorings/2026-09-11-pr18-review-cycle.md).

# PicFetch — TODOs

## Done

### What's Changed

#### New Features

- **Explorer setup on Linux.** The shared installer now selects a pinned Linux
  x86-64 runtime, reports the correct 383 MB download, and retains its licenses.
  Worker-only seccomp denial needs no root privileges or distro helper. Real
  installation, offline inference and UI lifecycle tests pass on Ubuntu;
  offline inference also passes on Debian 12 as an unprivileged user without
  networking. A Debian-built executable has a GLIBC_2.34 baseline. See the
  [implementation record](finished_refactorings/2026-09-10-explorer-ubuntu-setup.md).

- **First-use Explorer setup.** Trane introduces local similarity analysis.
  An explicit Download action installs and verifies the model/runtime, with
  progress, cancellation and retry. Pictures stay local; there is no analytics
  or feedback collection. Help, About and setup link to GitHub Discussions,
  and the privacy policy explains the public asset downloads. Explorer currently
  supports Apple Silicon Macs and x86-64 Linux. The introduction fills the window and uses
  verified transparent Trane artwork. Setup, offline inference, packaging and
  the canonical verification gate pass; see the
  [completed release setup plan](finished_refactorings/2026-09-10-explorer-release-setup.md).

- **Explorer and Mosaic access.** Shift+S opens the Visual Similarity Explorer;
  Shift+M opens Mosaic from the loaded collection, or the current selection/
  filtered result in Grid View. Generate Image Mosaic now lives in Window.
  Plain M/S retain merge/sort, and search/dialogs keep keyboard ownership.
  Explorer Presets is beside Unassigned on the right. Native public-fixture
  captures verify the toolbar and loaded-source Mosaic window; full-library
  qualification is complete. Final `make verify`, real offline UI tests,
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
  see the [implementation record](finished_refactorings/2026-09-09-visual-similarity-explorer.md#create-cohorts-from-unassigned--2026-09-10).

- **Refine similarity maps without rescanning.** Collapse the tag sidebar,
  click a tag's count to browse only matching images, and use the top-right
  Granularity slider to combine related stacks or restore the original groups.
  Arrow keys highlight stacks in the chosen direction; Enter opens the active
  stack and +/- zoom. Open grids keep their captured members during updates.

- **More useful map refinements.** Completed granularity changes rearrange
  cohorts while preserving zoom; background publications retain their anchors.
  Cohort titles summarize up to two subjects shared by at least half the images,
  with explicit names taking precedence. The expanded 75-tag catalogue includes
  Costume, Train, festivals, activities, landmarks, objects and food/drink.
  Actual-model cold/warm tests and the canonical verification gate pass.

- **Filter similarity maps by semantic tags.** A local 75-tag catalogue labels
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

- **PR #18 inspection cleanup.** Correct exported comments, a redundant slice bound,
  a shadowed builtin name and error capitalization. Scope exclusions to test
  conversions and the intentional JPEG rounding/fallback and partial-cache
  contracts; retain the underlying behavior. Runtime extraction selects fixed
  trusted output names to remove CodeQL's archive-path data flow. Focused tests,
  real model installation/offline analysis and `make verify` pass. See the
  [inspection cleanup record](finished_refactorings/2026-09-10-pr18-qodana.md).

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

### Comparison test deadline under build contention

The Ubuntu full race run on 2026-09-10 timed out in
`TestCompareSettle_DrainsVectorReplacementBeforeWaitingForObsoleteTiles`
while another distro build ran. Its one-second Settle deadline expired; an
isolated native and exact Ubuntu Docker race reruns passed. Review the fixture's deadline/load sensitivity;
the Explorer change does not modify comparison code.

### Explorer on Linux ARM64

Linux ARM64 now has the pinned native runtime, shared Linux seccomp worker,
pollable controls and setup support. The runtime requires glibc 2.28+; the local
test viewer is built against Debian 12 with required symbols up to GLIBC_2.34.
`bin/picfetch-linux-arm64.tar.gz` includes the executable and notices.
The user will qualify native analysis,
kernel enforcement and GUI behavior on ARM64 hardware. Pure BPF decisions and
cross-compilation do not claim that hardware qualification. See the
[Linux ARM64 plan](plans/2026-09-11-explorer-linux-arm64.md). Intel macOS Explorer
and 32-bit ARM remain unsupported; Windows ARM64 testing is separate above.

### Similarity-map zoom trial

The current increment enforces a 50% zoom floor only above 100 cohort piles,
including Fit map and automatic discovery, while keeping every sampled
thumbnail. Smaller maps retain the existing zoom range. The active plan records
SDD/TDD and verification evidence. Trial the cutoff on normal and large maps;
full-library scan throughput remains later work.

### Visual similarity explorer proof of concept

**Complete: tickets 01–07.** The final 50,655-source trial, fully cached reopen,
recovery, memory and human acceptance checks pass. No open tasks remain in this
milestone. See the [final qualification report](.scratch/visual-similarity-explorer/evidence/full-library-qualification-20260910/completion.md).
The development observations below are historical; this status supersedes their
then-pending qualification work.

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
[execution plan](finished_refactorings/2026-09-09-visual-similarity-explorer.md) records TDD
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

Ticket 07 is now complete. The actual final build accounts for 50,655 sources,
with 50,569 successes and 86 failures. Unchanged Favorite reopen reuses every
successful representation with zero inference; the warm pass takes 322.015s,
including 302.144s grouping. Cached five-member grid/image/map return preserves
membership and camera. Progressive/frozen browsing, large-map zoom-floor use,
granularity rearrangement, invalid-cache repair, cancellation and source-change
recovery all have representative native evidence and recorded human acceptance.

Live Go heap falls from 1,008.14 MiB on the loaded/browsed map to 45.84 MiB after
list closure and scavenging, against 21.12 MiB startup. Residual widget/metadata
and native memory remain documented; RSS alone is not treated as live retention.
The original Favorite's 50,569 valid analysis records occupy 790.30 MiB logically.
The app and observer exit normally with complete collection and no forced stop.
Real-engine UI/evaluator tests, the canonical `make verify` race gate and normal
`make build` pass. The [final report](.scratch/visual-similarity-explorer/evidence/full-library-qualification-20260910/completion.md)
retains timings, exact settings, resource/storage accounting, privacy boundaries,
replay checks and platform/measurement limits. **Milestone tasks 01–07 are done;
open tasks: none.** The implementation plan is archived; no commit was created.

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
