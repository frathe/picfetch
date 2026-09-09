# PicFetch — TODOs

## Done

### What's Changed

#### New Features

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
  No additional model download is needed to use tags. Label quality remains
  provisional pending library trial; saved grouping presets remain separate.

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
full-library semantic accuracy. Custom traits and saved presets remain separate.
Final real offline/native suites, `make verify`, refreshed build checks and
`make build` pass. The updated executable is `bin/picfetch`.

### Similarity explorer recovery

Source/failure recovery (ticket 06) is implemented and verified: focused/native
suites, the 446-image smoke run, `make verify` and `make build` passed.
[Recovery evidence](.scratch/visual-similarity-explorer/evidence/recovery.md)
records the source-change policy and regression coverage. Full-library
qualification and the saved-preset choices below remain open.

### Saved similarity group presets

Requested during the native explorer trial: select several Unassigned images,
use Link to inspect shared traits, select the desired traits, name the group,
and add it to the map. Retain named presets for future maps and provide a compact
preset browser instead of a nested menu tree. Trait scope (visual/metadata) and
future-map application behavior are being clarified with the user; preserve
current non-overlapping cohorts and the active map refinement work.

### Full-library similarity throughput

The controls increment removes measured display-transfer and stack-packing
costs. The live process sample was inside ONNX encoding; the current observer
does not expose per-stage completed-image counts, so a decreasing per-image
encoding rate has not been isolated. Record stage/count throughput during the
next native trial and qualify batch UMAP/HDBSCAN before claiming 50k scaling.
See the active plan's controls/performance evidence.

### Large similarity-map interaction

The original roughly 50k-image trial reached a visible map without a reported
crash, but the user observed severe lag zoomed out and much smoother interaction
zoomed in. The live capture recorded a 9.532 GiB viewer peak during publication,
settling near 5 GiB, plus main-thread CPU bursts and repeated texture uploads
during interaction. Map layout visits every pile; distant piles retain all
sampled thumbnails, and smooth image rescaling recreates textures on zoom.
Evaluate reduced detail at overview zoom and texture reuse before expanding
the renderer. Treat this as the observed extreme-case limit, with no measured
FPS guarantee. Exact source accounting and semantic qualification remain open.
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
