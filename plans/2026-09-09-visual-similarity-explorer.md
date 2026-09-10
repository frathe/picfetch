# Visual similarity explorer implementation

Date: 2026-09-09
Status: semantic tags, recovery, trial controls, conditional large-map zoom, viewport preview retention and production throughput profiling implemented and verified; saved presets and full-library qualification remain open
Route: Deep — new local analysis subsystem and cross-feature UI behavior
Request: `/implement use tdd and sdd`
Spec: [Visual similarity explorer](../.scratch/visual-similarity-explorer/spec.md)
Tickets: [Execution sequence](../.scratch/visual-similarity-explorer/ticket-breakdown.md)
Current increment: [Production throughput measurement](#production-throughput-measurement--resumed) — complete and verified; real worker stage/count instrumentation, bounded cold/warm command and 446-image evidence retained.

## Deliverable and accepted contract

Deliver the real local content-similarity explorer through the seven-ticket
sequence. Ticket 01 is complete. Ticket 02 now has a measured real pipeline and
local cohort report. The user accepted its quality (“it looks promising. yes
continue”), authorizing the completed-map native integration now implemented.

The user explicitly accepted the proposed defaults and test boundaries and
supplied `/Users/ronin/Projects/picfetch/.scratch/visual-similarity-explorer/demo`.
This answer supersedes all earlier pending preparation questions:

- Main-window map with existing Grid View over it and preserved map return.
- Go application code with native local inference for this Mac trial.
- A separate Unassigned collection, accessible in Grid View.
- Persistent image representations, reused only for unchanged sources with
  matching model/preprocessing. Ticket 05 is retained.
- Up to fifteen distinct sampled members per cohort pile, stable while
  membership is unchanged; all members for smaller cohorts.
- Separate successful, failed/skipped, and grouping/layout progress;
  completion includes final map delivery.

Q1-Q6 remain settled. D1-D6 are lead-owned decisions in the
[accepted contract](../.scratch/visual-similarity-explorer/decisions.md), not
another approval round. The [protocol](../.scratch/visual-similarity-explorer/evaluation-protocol.md)
uses all 446 supplied demo images for smoke evaluation. This corpus does not
establish the intended roughly 50,000-image qualification. Geolocation,
text-directed grouping, expanding piles at high zoom and broader packaging
remain outside the first trial.

## Confirmed test boundaries

1. Production-aligned viewer interactions, ordinary inputs, displayed file
   identities and visible surfaces. Controlled analysis results enter only
   through an instance-owned provider boundary.
2. The actual local engine's input/output, using real representations,
   grouping and projection with effective outbound denial.

The command boundary exercises the second seam. Normal tests require no
model assets; the explicit `explorertrial` tests require real pinned assets,
fail rather than skip on missing prerequisites, and run inside OS denial.

## Task 01 — Resolve the trial contract

Owner: T0 inline
Files: local decisions/protocol/tickets, this plan, todos.md
Depends: explicit user answers (received)
Contract: Q7-Q12, D1-D6, test boundaries and corpus recorded with provenance
Test: twelve accepted rows and supporting user/lead provenance
Verify: `rg -n '^\| (Q(7|8|9|10|11|12)|D[1-6]) \| accepted \|' .scratch/visual-similarity-explorer/decisions.md`
Budget: no additional scout, one review, no suite for documentary acceptance
Status: complete

## Task 02 — Real local evaluation command

Owner: T0 inline; one read-only T3 scout for concrete Go algorithm APIs
Files: scripts/explorereval/{main,assets,offline,files,encoder,evaluate,grouping,
report,memory_darwin,memory_other}.go; main_test.go; trial_test.go; review.html;
setup.sh; evaluate.sh; assets.sha256; README.md; Makefile; go.mod/go.sum;
qodana.yaml; ARCHITECTURE.md; this plan; todos.md; local tracker/evidence
Depends: accepted ticket 01
Contract: `make explorer-evaluate TRIAL=smoke` uses pinned local assets,
processes at most 512 local images under inherited OS network denial, and
writes initial/final representations, exact source accounting, cohorts,
positions, source-linked review and measured evidence in a fresh directory.
The CLI subprocess can be canceled and observed through its exit status.
No production explorer interface is fixed by the experiment.
Test: cancellation before admission and after actual progress; missing/tampered
assets; real vectors preserve identical inputs and distinguish different
pixels; unreadable inputs remain accounted for; finite normalized outputs,
initial publication, full regrouping, timings, native RSS and local review.
Verify: `go test ./scripts/explorereval -count=1`; `make explorer-test`;
`make explorer-evaluate TRIAL=smoke`; explicit denied/undenied probe checks;
`make verify`; separate user semantic verdict on the generated review.
Budget: at most one scout, two lead review rounds, one complete final race suite
Status: complete; user accepted the real report; experiment verification passed

## Task graph and later planning

`01 -> 02 -> 03 -> {04, 05, 06} -> 07`

Ticket 03 owns the completed real map/Grid View/return path. Tickets 04-06
extend that path with progressive updates, valid reuse and recovery. Ticket
07 owns full-library/native interaction evidence. The current quadratic
HDBSCAN and batch layout are experimental candidates, not selected
full-library implementations. Plan exact production files and interfaces
after the measured pipeline is accepted and those limits are addressed.

## Delegation gate

One read-only scout inspected pinned UMAP/HDBSCAN APIs and synthetic edge cases
while the lead implemented native inference and the CLI. G1: bounded API
question; G2: module revisions, source locations and synthetic command output
verified by the lead; G3: no repository writes; G4: algorithms independent of
native inference; G5: lead had not inspected implementations. S/W: adaptive
source reading, no supplied implementation. The harness lacks the literal
T3 Explore agent, so it used the inherited model in a read-only role. Spec,
architecture, all review and all fixes remained with the lead.

## Evidence

- Public model revision `ba1f3b0843f24bc5417d38e19c37b287d719b2f4`, float32
  SigLIP 2 base patch16 224 vision export; model/archive checksums matched
  their published digests. Go binding v1.36.0 and native ONNX Runtime 1.29.0.
- Scout findings verified against downloaded sources: UMAP Transform has no
  retained training data and panics; use fresh fits. HDBSCAN numeric labels
  enumerate a map; use membership-derived IDs. Neither API accepts context.
- TDD red/green evidence: initial cancellation returned nil before its guard;
  missing assets and a corrupted model were initially accepted; real pipeline
  initially produced no result; incremental evidence lacked first-map/memory
  outputs. Each failed for the intended behavior and then passed.
- The native test additionally exposed a missing driverless file repository
  and the model's actual `pooler_output` name. Those integration errors were
  fixed before any smoke-run success was claimed.
- Deliberately disabling evaluation cancellation made the progress-cancel
  test fail with nil error. Restoration passed. The offline probe rejected
  unprotected execution (TCP timeout is not permission denial) and passed
  under explicit `sandbox-exec` denial. No image/path payload is sent by probes.
- `make explorer-test`: three ordinary command guards and all three real-model
  tests passed under explicit OS denial. No skips.
- Smoke run `smoke-TZppbI`: 446/446 represented, zero failures, 36 non-noise
  cohorts, 90 unassigned (20.2%). Initial 223-input map at 49.756 s; measured
  processing time 90.638 s; worker peak 1,143,521,280 bytes (1090.5 MiB).
  No claimed GPU/ANE execution; this run used CPU with six intra-op threads.
- Lead evidence checks confirmed all 446 source identities/hashes unchanged,
  finite unit 768D vectors and finite 2D positions, all thumbnails present,
  and no external report resources. Generated JavaScript passed `node --check`.
  The report was opened locally in Safari for the user's quality assessment;
  the agent has not viewed or uploaded library images.
- Source formatting, focused tests and vet passed. The initial `make verify`
  stopped at formatting an agent-created scratch inspector; that inspector
  was moved outside the repo and the gate restarted before any race suite ran.
  Final Linux/amd64 race verification passed (all four streams retained under
  `.scratch/race-runs/20260909T123102Z-y8OnfU`).

The [local report](../.scratch/visual-similarity-explorer/evidence/smoke-TZppbI/result/review.html)
and [measurement record](../.scratch/visual-similarity-explorer/evidence/smoke-TZppbI/result/pipeline-evaluation.md)
are concrete review artifacts. No content-similarity quality, native viewer
integration, stable progressive layout, persistence reuse, or 50k result is
claimed. The user accepted the semantic result in the following turn.

## Cost ledger and handoff

Earlier documentary preparation is recorded in Git history. This execution:

| Task | Spawns budget/actual | Lead review rounds | Full suite | Evidence |
| --- | --- | --- | --- | --- |
| 01 | 0 / 0 | 1 | no | Explicit user acceptance and twelve resolved rows |
| 02 | 1 / 1 | 2 | no | Real model tests, negative guards and 446-image trial |
| Final gate | 0 / 0 | 1 | passed, one race run | make verify; refreshed build and focused checks after termination fix |

Keep the plan active until accepted completion. No git commit is authorized;
end with a suggested commit message. The unrelated untracked `:memory:.ses`
file appeared during the session and was left untouched.


Post-review termination guard: a synthetic real worker initially exited by
SIGTERM without using the cancellation path. The observed red failure was
`signal: terminated`; after SIGTERM joined `signal.NotifyContext`, it returned
`context canceled`, released native resources and published no result. All
six explicit-denial checks then passed, followed by `make verify-build`.
The first invocation of that new native test outside OS denial correctly
failed its prerequisite; that was not counted as its behavioral red proof.

Final closeout: affected-package race tests passed after the termination fix,
and `make explorer-setup` reverified local assets without downloading.
Hardware evidence confirms Apple M5 Max, 18 CPUs, 48 GiB, macOS 26.6.2 and
Go 1.27.1. All 55 local links across 16 documents and whitespace checks passed.
The semantic verdict was received in the following turn. Native integration
and its own evidence are recorded below.

## Native integration acceptance and task 03

The user reviewed the real corpus report and replied “it looks promising. yes
continue”. Semantic quality is accepted for the small native integration; the
446-image result does not qualify 50k behavior. Earlier pending-verdict text is
historical and superseded by this acceptance.

Owner: T0 inline. One read-only T3 scout traces existing grid/viewer return paths
while the lead fixes the engine interface. Files: internal/similarity (reusable
local engine/process protocol), internal/ui/explorer (map surface), root UI
composition/input/navigation, grid cohort filtering, menus/translations/manuals,
main worker dispatch, experiment adapters, tests/manifests and architecture.
Contract: analysis accepts the exact opened source identities and emits immutable
progress/results; a cancellable child process contains native inference and
non-cancellable batch algorithms. Map UI owns its camera, sampled pile previews
and completion delivery; root UI owns map/grid/image transitions. Cohort grids
filter by source identity without replacing the opened file set.
Test: ticket 03 completed map, full cohort membership, camera return and opened
inputs at the approved UI seam; actual local engine under OS network denial.
Verify: focused TestVisualSimilarityExplorer subtests; explicit native offline
trial; make verify. Budget: one scout, two lead reviews, one final race suite.

Scout gate: G1 bounded trace of grid/image return and navigation; G2 source
locations verified with rg/sed; G3 zero writes; G4 isolated cross-file input flow;
G5 lead has only read grid filtering and activation. S/W: adaptive call tracing,
no mechanical transform. Literal Explore is unavailable; inherited model is
read-only. Review and fixes remain with the lead.

### Task 03 implementation and verification record

Public engine contract: `similarity.Client{Assets}.Analyze(context.Context,
[]string, func(similarity.Event)) error`; every callback is immutable and serial,
and return observes child exit. `WorkerMain` runs before app startup and verifies
TCP/UDP denial plus pinned assets before reading sources. The bounded experiment
uses the same encoder/grouping implementation. No new public launch flag.

Map contract: `explorer.Map` has a two-method Host for opening captured source
identities and leaving the map. It owns no goroutines. Root `explorerWork` owns
requestLifecycle, tracked workers and a drainable UI queue. Grid `OpenSubset`
preserves host indexes; image navigation wraps within frozen cohort identities.

Observed red/green: missing menu; End escaped the cohort; actual canvas drag
missed a zero-sized map; real native provider was disconnected; leaving the map
did not cancel analysis; covered image actions remained enabled. All corrected
and focused tests passed. Negative verification additionally narrowed inputs,
removed staleness guards, removed the sample cap, and reset the return camera: all
four targeted guards failed as intended, then source was restored.

`make explorer-test` passed all six shared-engine command/native guards. The
three real UI-worker scenarios passed: offline map/cohort/image return, a missing
source separately accounted, and worker exit after cancellation on actual
progress. Focused UI race checks passed. Synthetic rendered QA exposed a
transparent toolbar; an opaque backdrop fixed it and the next render was
inspected. User images have not been viewed by the agent or transmitted.

Native QA found very large projection units could produce tiny piles. A new
projection_scale UI guard failed at a 7.2px pile width, then normalized display
spacing passed. The user additionally requested default maximization; entering
the explorer now uses the same native work-area maximize as Grid View. Cohort
navigation retains that size; later ordinary-image resizing clears the native
maximized state. The native screenshot before the change showed the 520px window.

Verification budget update: the first complete gate found an existing menu
inventory assertion missing the added item. Native QA and the user's maximize
request also changed the final UI. Fixes remain lead-owned; the final gate will
be rerun after native QA.


Native interaction evidence (synthetic pixels only): the refreshed application
enters a 1226 x 768 work-area window and displays two readable 12-image piles
from 24 actual local model results. A pile opens its 12 Grid View members;
Enter opens an image, End reaches that cohort's last image, and Right wraps
to its first. Escape returns to the cohort grid; Back to map restores the
map. Native drag and zoom visibly alter the camera, and a subsequent
cohort/grid return preserves that geometry. No library pixels were inspected.

Final boundary review found two gaps and closed them with observed red/green:
merging a repeated path produced four samples from three distinct sources;
preview selection now deduplicates sources while Grid View retains every
opened occurrence. An uncached cohort navigation shrank the window from
1100 x 700 to 520 x 340 at the early header probe; that probe now honors the
same explorer sizing rule as final image delivery. Removing its guard made
the adjusted cold-navigation test fail again, then restoration passed.

The initial cold-load test activated an uncached cell inside Fyne's selection
callback. Fyne's inline test driver allowed load completion to race its deferred
UnselectAll; the native driver serializes those callbacks. The boundary test
now opens the initial cached member, then performs uncached End navigation,
which exercises the sizing bug without overlapping that selection callback.
Focused explorer/grid sizing race tests pass after this adjustment.

Verification bookkeeping: the second complete gate compiled that earlier
cold-load test and reported its race. A final run of the affected ui-1 shard
uses the corrected test; unaffected completed partitions are retained from
that gate. Formatting, vet, build and focused native race checks were rerun
after the final source fixes. This exceeds the planned two review rounds
because native inspection, user-requested sizing and boundary tests exposed
additional concrete findings; no fixes or reviews were delegated.


### Task 03 closeout

- `make verify-build` passed after the final production fixes; `make build`
  refreshed the trial binary. Formatting and the exact 680-runnable shard
  inventory passed after both additional subtests were added.
- All Linux race partitions pass using the retained second-gate non-ui,
  ui-2 and ui-3 outputs plus the corrected ui-1 rerun. The literal second
  `make verify` exited 2 on the earlier test-driver race; it is not reported
  as a successful invocation. The affected shard was rerun with the same
  Makefile target and Linux/amd64 environment, and exited 0 with 223 top-level
  passes. Evidence: `.scratch/race-runs/20260909T133105Z-HBMsUI` and
  `.scratch/race-runs/explorer-ui-1-final`.
- `make explorer-ui-test` passed again on the final code: offline real-engine
  round trip, missing-source accounting and canceled-worker exit, plus every
  controlled UI scenario. No required native tests skipped.
- Focused native race tests covered all explorer cases and both existing grid
  maximization cases. Both new regression guards were observed failing before
  their fixes. All 58 local documentation links and whitespace checks pass.
- The refreshed trial app is open with the supplied 446-image demo; its local
  analysis child has exited. Only synthetic pixels were inspected by the lead.
  Native behavior evidence is in
  [native-completed-map.md](../.scratch/visual-similarity-explorer/evidence/native-completed-map.md).

| Task | Spawns budget/actual | Lead review rounds | Full suite | Evidence |
| --- | --- | --- | --- | --- |
| 03 | 1 / 1 | 4 (budget 2) | two attempts; affected shard rerun | Native QA, real offline tests, negative guards, all race partitions green |

The extra review rounds and verification were caused by observed native scale
and cold-load issues, the user's maximize request, and the test-driver race
exposed by the added boundary scenario. Tickets 04-07 and the full plan remain
open; no progressive/persistence/full-library result is claimed. No commit was
made, and both unrelated `:memory:.ses` files remain untouched.


## Task 04 — Native feedback and progressive map

The user accepted the general native feel and requested separated piles,
thinner image borders, wider scattering, partial results every 10-30 images,
fewer unassigned images when meaningful, and Shift-to-pan.
Owner: T0 inline; one read-only scout for grouping parameters and aggregate
nearest-neighbor evidence from existing representations (no source pixels).
Files: internal/ui/explorer/{map,layout}.go; internal/ui/explorer.go;
internal/similarity/{analyze,grouping}.go and experiment adapters as required;
root explorer UI/native tests, manuals/translations, architecture and tracker.
Contract: immutable partial snapshots at a default 30-image cadence and final
completion; frozen open cohorts; preserved camera and continuing pile positions;
minimum pile separation; stable wider-scattered samples with thin fitted frames;
Shift-scroll pans via the existing instance-owned modifier seam. Adjust grouping
only against measured vector evidence; retain a meaningful Unassigned collection.
Tests: production-aligned surface geometry/input/frozen-cohort/progress scenarios;
actual offline engine publishes before completion and accounts for all sources.
Verify: focused explorer tests; make explorer-ui-test; existing-representation
comparison and real smoke timings; synthetic native QA; final make verify.
Budget: one scout, two lead review rounds, one full final suite.
Delegation gate: G1 bounded API/statistical question; G2 pinned source references
and reproducible aggregate script output; G3 no repo writes; G4 independent of
map/progress implementation; G5 lead has not inspected density implementation or
nearest-neighbor distributions. S/W: adaptive source investigation with computed
numbers, not a scripted repository transform. Review and fixes stay with lead.

### Task 04 lead review and evidence

The controlled UI and actual offline engine boundaries pass on the restored
production code (`make explorer-ui-test`, ui package 8.633s). Observed failures
before implementation pin pile spacing, fitted preview aspect/scattering,
Shift-scroll pan, partial delivery, continuing positions, and a useful
four-image real cohort. Repeated-source density was negatively verified at
the actual engine boundary. Deliberate violations also fail frozen-cohort,
current non-overlap and window-fit guards. No new test files or top-level
runnables were added; the existing Qodana paths and ui-1 assignment still cover
all added subtests. All 64 checked local documentation links resolve.

Native synthetic inspection confirmed the final separated stacks and thin
portrait/landscape frames. A four-member pile opened four Grid View members,
Enter opened an image, and two Escapes restored the fitted map and maximized
window. A deliberately held synthetic source explained the user's apparent
30-image stop; release resumed that fixture. A subsequent all-regular-source
run completed 60/60 without failures. The updated build then reopened the demo.

The production offline worker published 14 partial maps and a final map for
446 sources: first map 9.840s, completion 101.088s, 0 failures, 46 cohorts,
55 unassigned. All source hashes match the earlier smoke run. This is the same
seeded input order that previously had 90 unassigned. A lexical-order check
produced 89 versus 72, so input-order sensitivity remains an explicit limit;
counts do not establish semantic accuracy. The lead independently reran the
scout's vector-only sweep. No source pixels were viewed or transmitted.

Native fit inspection added an orientation guard. Its first absolute 250px
cutoff was too prescriptive about packing; review adopted a one-fifth-window
minimum for six cohorts, retained nearest-position packing, and verified that
removing orientation still fails. It now measures 234.6px in a 1100px window,
versus 141.9px before. More complex lattice placement was not retained.

Full evidence and the reproducible measurement script are linked from
[progressive-exploration.md](../.scratch/visual-similarity-explorer/evidence/progressive-exploration.md).
The user's saved grouping preset idea is recorded separately with its two
pending design choices. Persistence, extended recovery and the intended
50,000-image qualification remain later tickets; this work makes no claim
that those are complete.

The final `make verify` passed with exit 0, including formatting/TUF/Qodana
exclusions, vet, build, exact shard validation and the full Linux/amd64 race
suite. Race evidence: `.scratch/race-runs/20260909T144231Z-hbc3Ej` (container
exit 0, no OOM). All three UI partitions passed (223, 226 and 230 top-level
tests), as did the non-UI partition (1,864 top-level tests). The explorer's
complete controlled UI scenario passed under Linux race detection in 37.100s.
The separate real offline UI/engine suite passed in 8.633s. The demo worker
has exited; its updated native map remains open for the user.

| Task | Spawns budget/actual | Lead review rounds | Full suite | Evidence |
| --- | --- | --- | --- | --- |
| 04 | 1 / 1 | 2 | one successful invocation | Real partial publications, native synthetic inspection, negative guards, full CI gate |

Only tracker/plan closeout changed after the gate. No commit was made; both
unrelated `:memory:.ses` files remain untouched. Ticket 04 awaits the user's
updated visual verdict; the full implementation plan stays active for the
remaining persistence, recovery and qualification tickets.

### Task 04 follow-up — growing viewport and duplicate representatives

The user accepted the improved spacing and now requests automatic zoom-out as
new stacks arrive, plus scanning only the highest-quality member of duplicate
groups when Grid View duplicate filtering is active. This supersedes the earlier
fixed-camera rule during publication; cohort round trips still preserve the
resulting camera. The two dog piles prompted a separate question about desired
grouping granularity; no new subject-label behavior is assumed.

Owner: T0 inline; one read-only duplicate-path scout.
Files: explorer map/root integration, existing grid consumer seam if needed,
existing root explorer UI tests and manuals/tracker/architecture.
Contract: publication includes every pile in the viewport, zooming out only
when necessary and never zooming back in automatically. Use Grid's existing
quality representative when hide-duplicates is enabled; ordinary search and
selection still do not narrow analysis. Capture source identities before Grid
close retires its session. Pending duplicate work must not silently analyze
redundant sources or lose sources.
Tests: held partial results, camera expansion/no automatic zoom-in and preserved
cohort return; production Grid duplicate filtering and provider input identities,
quality election and pending work through existing UI inputs/queues.
Verify: focused explorer UI boundary, real offline boundary if engine inputs
change, native synthetic QA, final make verify after implementation and review.
Budget: one scout, two lead review rounds, one new final suite because this is
new requested behavior after the preceding successful gate.
Scout gate: G1 bounded duplicate API/lifecycle question; G2 exact source/test
references verified by lead; G3 read-only; G4 separate Grid duplicate subsystem;
G5 lead has not built this context. S/W: lifecycle investigation across files,
not a mechanical transformation. All design/review/fixes remain lead-owned.

#### Live trial interruption — 2026-09-09 15:01 UTC

The user started 50,655 files in the already running native build and requested
KPI monitoring. Builds/tests are paused to avoid competing with this run.
Read-only 15-second resource capture is running for viewer PID 27312 and worker
PID 31913; evidence and a continuously updated status file are under
`.scratch/visual-similarity-explorer/evidence/full-library-live-20260909T150213Z`.
It stops when the worker exits. Physical footprint, CPU, elapsed time, I/O,
memory-free percentage and shared-system swap are recorded. No library pixels
were inspected. Fyne exposes no toolbar counts to accessibility, so the user
was asked for ready/failed counts before throughput/ETA can be estimated.

Implementation state at this interruption: `discovery_expands_view` was observed
failing for offscreen piles, then passed with ExpandToFit plus navigation,
cohort-return and progressive-exploration checks. Camera publication now expands
the existing visible area and never zooms back in automatically. The new
`duplicate_representatives` boundary test is intentionally RED: the current
implementation still supplies all three fixture sources instead of the larger
duplicate representative and unique image. Do not claim this current tree
passes the full suite or deploy it as a completed follow-up yet.

The duplicate scout found existing `dupes.Visibility` filtering and native-pixel
quality election. Missing hashes remain visible, so pending work must finish
before snapshotting input; Grid.Close currently cancels that work. The planned
lead-owned continuation is a small Grid preparation/readiness API using its
existing tracked hash/group workers and duplicate-state callback, joined to
the explorer lifecycle while its waiting map remains cancelable. Root input
selection will retain full-set order and ignore ordinary search/selection.
Add pending/cancel/source-replacement boundary cases after the warm quality
slice passes, then update manuals and run the new final gate once the live
trial has ended. No duplicate preparation code has been implemented yet.

#### Manual rebuild control — updated request, 2026-09-09

The user now requests manual map rebuilds by default, with automatic updates
every 30 images retained as an opt-in. This replaces mandatory 30-image
publications. The Update map button rebuilds from already represented data;
it must not restart scanning or reread/encode earlier images. One final map
remains automatic when scanning completes. Existing maps remain browsable
while a requested rebuild is pending. Repeated requests are coalesced, and
switching automatic updates off suppresses subsequent periodic rebuilds.

The same accepted provider/actual-engine test seams apply. The provider gains
a receive-only channel of `similarity.Control` (latest Automatic flag plus
an Update request); the private offline worker receives controls through its
existing input pipe. Its scan loop owns retained items and consumes controls
at safe image boundaries. UI controls show pending rebuild state, with manual
updates disabled until there are new successful representations. Worker/client
control readers and writers must terminate observably with the analysis session.

Tests: actual manual default and explicit/automatic control, cancellation of
both directions, no rereading represented inputs; visible update button and
checkbox driven through ordinary UI interactions; frozen cohort and viewport
expansion remain required. Commands: focused TestVisualSimilarityExplorer and
TestVisualSimilarityExplorerLocal cases, then make explorer-ui-test and the
final make verify. No new test seam or broad subject-label decision is assumed.

Focused tests resume to implement this explicit request. Their intervals are
recorded in the live KPI evidence so measurements can distinguish development
load. The running 50k app is not rebuilt or restarted. Final build deployment
will wait until the user ends that run; its resource observer remains active.

#### Favorite analysis reuse and settings — accepted 2026-09-09

Persist successful per-source representations for existing favorite members in
<favorite>/analysis beside file-list.json and thumbs. Reuse on future analysis
only when source identity/version and model/preprocessing version match. Keep
cohort membership/layout out of the cache because both depend on the current
input set. Invalid, missing or corrupt entries fall back to analysis. Persist
incrementally so cancellation retains completed work; never recreate a removed
favorite. Ordinary non-favorite inputs remain transient. Existing favorite
replacement/removal owns membership and folder lifetime.

Settings agreed by user: save analysis for favorites ON; automatic updates every
30 images OFF; fit newly discovered stacks into view ON. All three persist through
existing preferences. Map checkbox and Settings must agree. Cache toggle applies
to newly started scans; current scan settings are captured at admission.

Lead owns implementation/review. Scout favorite_cache_routes was read-only and
independent of active controls work (G1 bounded provenance/storage/settings
question, G2 references verified by lead shell, G3 no writes, G4 separate storage
subsystem, G5 cold context; no mechanical transform). Budget 1/1 scout; no further
implementation delegation. Existing provider UI and real offline engine seams
cover defaults/controls/persistence, cache reuse/freshness/corruption, favorite
membership and cancellation. Verify focused Explorer subtests, full local engine
suite and final make verify. Settings controls also checked in rendered tree.

#### Implementation evidence, manual controls / favorites / settings

- Manual control real-engine RED: old automatic publication at 30; GREEN 4.290s
  after fixing pollable worker-input shutdown. UI controls RED missing button;
  GREEN with navigation 1.351s.
- Duplicate boundary fixture corrected to actual D key events (typed runes are
  search input); representative test GREEN 0.933s. Cold preparation, cancellation
  and source replacement GREEN 1.048s. Negative mutation bypassing preparation
  and visibility made all these guards fail for the expected input/start reasons.
- Settings UI missing-controls RED, defaults/persistence GREEN 0.953s. Auto-fit-off
  guard rejects an unconditional camera expansion.
- Favorite actual-engine RED: unreadable cached source failed and reused=0;
  GREEN 3.434s across hit/change/corruption/membership/off. Cancellation/moved
  favorite/model-version/UI-wiring suite GREEN 5.664s before negative review.
  Negative review found the model-version fixture rounded mtime through float64;
  changed it to json.Decoder.UseNumber so only Version is altered.
- All-tag-active initial state is explicitly accepted for the forthcoming tag
  overlay. It replaces suggested single cohort naming, not similarity grouping.

No native bundle replacement while the live 50k scan runs. No commits created.
Full verification and final visual checks follow the focused boundary gate.

#### Release map resources on exit — user-reported retention

The user stopped the large scan at roughly 6.2k images, then reported memory
remaining high after leaving the Explorer. A single read-only observation found
viewer RSS 16,122,208 KiB after aggregation had stopped. No background KPI
collection was restarted. New acceptance: exiting Explorer releases its map
resources and cancels unfinished analysis; cohort/image/back-to-map transitions
retain the active map. Favorite analysis stays on disk. Verify via the existing
production UI/provider seam with an observed heap-allocation delta for large
synthetic map resources, plus existing cohort return/cancellation scenarios.
A read-only scout is locating Fyne renderer/resource lifetimes while the lead
constructs the reproduction. This is an independent cold dependency search;
lead owns diagnosis, fixes and final verification. Full gate postponed until
this newly requested fix is checked.

Memory diagnosis evidence: initial boundary baseline 21.0 MiB / map 69.0 MiB /
exit 69.0 MiB. Clearing scene/pile lists alone still retained 69.1 MiB after
exit. Two consecutive publications retained 117.1 MiB against 21.0 MiB baseline.
Fyne 2.8 renderer caches retain removed widgets; parent refresh deliberately
skips image textures. Clearing each old canvas.Image source and explicitly
refreshing it releases pixels/texture references. `SetResult` now does that after
anchoring against the previous map. Full exit invokes closeExplorer and clears
result/completion state; cohort/image transitions still only hide/preserve it.
Focused memory, cohort-return, progress, cancellation and replacement tests pass
(1.419s). This proves Go heap ownership; OS RSS can lag garbage collection.
No global caches were purged and no forced production GC was introduced.

#### Final gate — manual controls, favorites, settings and memory release

- `make explorer-ui-test`: PASS 18.559s, no skips. Heap reproduction: 37.2 MiB
  baseline, 85.2 MiB current map after two publications, 37.2 MiB after exit.
- Adjacent preferences/settings/Grid/Favorites/help package suites: PASS.
- `make verify`: PASS, exit 0. Formatting/TUF/Qodana checks, vet, native build,
  shard manifest and every Linux/amd64 Docker race partition passed. Artifacts:
  `.scratch/race-runs/20260909T155214Z-fb0HxS`; Explorer race test 39.650s.
- `make build`: PASS; updated executable `bin/picfetch`. Running old trial app
  untouched. Synthetic map screenshot inspected: toolbar readable, manual update
  and unchecked automatic option visible; spaced thin-framed piles preserved.
- `git diff --check` and 28 local documentation links: clean.
- Preserved user staging of layout.go and unrelated :memory:.ses files. No commit.

Evidence bundle: `.scratch/visual-similarity-explorer/evidence/manual-cache-release-20260909`.
KPI aggregation stopped; 167 samples and the user's roughly 6.2k scanned count
are retained. No completed 50k qualification or new-build RSS measurement is
claimed. The tag overlay and saved grouping presets remain separate next slices;
all tag checkboxes starting active is an explicit accepted requirement.

Cost ledger for these follow-ups: lead-owned design, implementation and fixes;
read-only scouts for duplicate inputs, favorite persistence routing and Fyne
resource lifetime (one per independent question). No implementation/review
subagents; one final full verify for this increment. Focused repeated tests were
limited to new behavior, fixture corrections, negative guards and the subsequent
user-reported memory retention.

## Semantic tags — resumed implementation, 2026-09-09

Deliver the next ready backlog slice: local semantic labels and a checkbox
overlay filtering existing cohorts. Route: Deep (engine/UI integration).
Existing accepted provider/UI and actual offline-engine test seams apply.
The user has been offered an optional vocabulary preference; common subjects
and scenes are the default. Saved presets, custom trait discovery, source
recovery ticket 06 and full-library qualification remain separate work.

### Tag task 1 — Cohort filtering through the viewer
Owner: T0 inline
Files: internal/similarity/similarity.go; internal/ui/explorer/{map,tags}.go;
internal/ui/explorer_test.go; translations/{en,de}.json
Depends: accepted overlay interaction
Contract: Item.Tags carries stable semantic IDs. Rows count unique successful
mapped source paths across the whole map. OR filters show a cohort when any
member has an active tag; Untagged includes members with no recognized label.
Unassigned is a cohort for filtering and retains its complete captured members.
All controls start checked; All tags/Clear tags avoid toggling 31 rows to
isolate one subject. Filtering preserves memberships, pile placement,
camera, and counts. Publications preserve choices (even temporarily absent
tags), default new tags on; full exit resets choices. All-off shows no cohorts.
Test: ordinary checkbox/pile/Grid/back interactions and progressive provider
delivery verify counts, overlapping labels, membership, camera, and reset.
Verify: `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/^tags' -count=1 -v`
Budget: 0 implementation spawns; 1 lead review; focused suite only

### Tag task 2 — Real offline labels
Owner: T0 inline; T3 read-only asset-contract scout
Files: internal/similarity label implementation and generation assets/tooling;
internal/ui/explorer_local_test.go; scripts/explorereval evaluation integration
Depends: verified pinned text/vision model compatibility and task 1 contract
Contract: label each successful fresh or reused representation locally. Multiple
labels may qualify; low-confidence/unrecognized inputs remain untagged. Cache
stores image representations, with labels recomputed against the current label
catalogue. No image uploads or additional per-image model execution.
Test: actual offline worker processes known-content fixtures and blank input;
reused representations get identical labels without rereading pixels.
Verify: `make explorer-ui-test` and `make explorer-test`
Budget: 1/1 scout; 1 lead review; focused actual-engine suite

### Tag task 3 — Handoff
Owner: T0 inline
Files: ARCHITECTURE.md; manuals; todos.md; local tag spec/evidence; this plan
Depends: tasks 1 and 2
Test: localization, native/synthetic visible layout, existing Explorer regressions
Verify: `make verify`; `make build`; `git diff --check`
Budget: no spawns; one final full race gate

Graph: contract scout in parallel with UI recon; 1 -> 2 -> 3.
Scout gate: G1 bounded text-asset question; G2 source locations/metadata commands;
G3 no writes; G4 cold inference tooling context; G5 lead owns separate UI recon.
S/W: source/API investigation, no mechanical transform or implementation supplied.
Literal T3 Explore unavailable: inherited model read-only, as earlier increments.
Lead owns all product decisions, implementation, review and fixes.

### Semantic tag implementation and final evidence

- Completed 31 local subject/scene tags, localized checkbox panel, unique-image
  counts, OR filtering, Untagged, All tags/Clear tags, unchanged cohort members
  and camera, preserved choices across publications/visits, and fresh-session
  reset. New controls release focus; background publication preserves Grid focus.
- Existing image vectors feed fixed embedded text prototypes, regenerated from
  the matching pinned SigLIP 2 text tower. Runtime needs no extra model or Python.
  Favorite representations remain reusable; derived labels are recomputed.
- Initial absolute cutoffs were rejected by coverage and expanded known-content
  tests. Final policy compares normalized catalogue score shares: strongest
  share >= 0.35 and raw score >= 0.00001; every share >= 0.15 qualifies.
  Cat, Person+Portrait and Food positives, three blank negatives, actual offline
  operation, cached pixels without read permission and visible UI delivery pass.
  Existing smoke vectors produce 377 tagged, 69 untagged and 171 multi-tag images
  out of 446. These counts establish coverage, not semantic accuracy.
- TDD REDs: missing rows; missing clear action; captured checkbox focus; missing
  labels; rejected known portrait. Corresponding GREENs recorded. Deliberate
  mutations broke choice preservation, cached labeling and ambiguity rejection;
  all failed for the intended reason and were restored.
- Vector regeneration is byte-identical: 95,232 bytes, SHA256
  `3fa76594e8e59d21ae534343f802953aa51efbb1cb0746748aac7460e8c712ec`.
- Synthetic UI capture inspected. Isolated native app with public cat verified
  filtering, Unassigned/Grid return, reset and physical Escape after controls;
  it was closed normally. Production app identity/settings were untouched.
- `make explorer-test`: PASS. Final `make explorer-ui-test`: PASS, 20.053s,
  no skips. `make verify`: PASS, exit 0; all Linux/amd64 race partitions passed
  in `.scratch/race-runs/20260909T162132Z-URO0Wg`. While that race gate ran,
  known-content evidence refined only the worker's label selection policy and
  its native-tagged tests. The complete native suite, `make verify-build` and
  `make build` were rerun on the final policy and passed; UI filtering code
  remained unchanged. No second full race run was needed for this worker-only,
  non-concurrent scoring change. Updated executable: `bin/picfetch`.
- Existing test files/subtests retained their Qodana exclusions and shard rows.
  Documentation and catalogue/UI-ID checks pass. No commits; pre-existing
  `:memory:.ses` files remain untouched. The broad plan stays active for trial,
  saved presets and tickets 06-07.

Full [tag evidence](../.scratch/visual-similarity-explorer/evidence/tag-overlay/README.md).
Cost: one read-only scout (budget 1/actual 1); all implementation and review
inline. Review extended to a second pass because native focus and aggregate
coverage exposed concrete defects. One full race gate; focused/native repeats
were limited to those findings and negative verification.

## Recovery — resumed implementation, 2026-09-09

Deliver ticket 06: coherent exit/restart, source mutation and failure recovery
through the existing viewer and offline engine. Route: Deep. Accepted D2/D5
and the confirmed UI/provider and actual-engine seams apply. Preset product
choices and full-library qualification remain outside this slice. External
file watching is not introduced; reconciliation follows ordinary viewer actions
and engine source-version checks.

### Recovery task 1 — Lifetime and retry
Owner: T0 inline
Files: internal/ui/explorer.go; internal/ui/explorer_test.go
Depends: accepted D5
Contract: exit/shutdown invalidate queued delivery; shutdown refuses admission;
retry starts fresh; failed or incomplete analysis cannot be retained as complete.
Test: outstanding publication and termination callbacks across exit, shutdown,
restart; setup/analysis failure with normal input focus and a successful retry.
Verify: `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/^(lifecycle|recovery)$' -count=1 -v`
Budget: 0 spawns; up to 2 lead reviews; focused suite only

### Recovery task 2 — Source identities and committed effects
Owner: T0 inline; read-only scout for existing mutation/reconciliation routes
Files: internal/ui/explorer.go; filework.go; viewer removal glue as identified;
internal/ui/load.go; internal/ui/explorer_test.go; internal/ui/explorer_local_test.go
Depends: accepted D2; task 1 lifetime contract
Contract: reorder preserves identity; removal reconciles actionable cohort
members; writes invalidate affected analysis even when the initiating request
is stale. New exports outside the input set leave analysis intact. Rejected
late results cannot restore invalidated content. Existing file-work workers
resolve aliases; no filesystem reads on UI and no new worker family.
Test: actual UI sort/delete/write/navigation paths plus actual offline worker
missing-source, cancellation/restart and source-version rejection.
Verify: `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/^source_changes$' -count=1 -v`; `make explorer-ui-test`
Budget: 1 scout; up to 2 lead reviews; focused suite only

### Recovery task 3 — Verify and record
Owner: T0 inline
Files: ticket 06, evidence/recovery.md, todos.md, this plan, manuals/architecture
only where behavior or locators change
Depends: tasks 1 and 2
Contract: observed red/green and negative guards; exact native/smoke results;
honest remaining limits and canonical gate evidence.
Verify: `make explorer-test`; `make explorer-evaluate TRIAL=smoke`; `make verify`; `make build`; `git diff --check`
Budget: 0 spawns; one final full race gate

Graph: source-route scout alongside task 1; 1 -> 2 -> 3.
Scout gate: G1 bounded mutation-route question; G2 source locations verified by
lead shell; G3 no writes; G4 separate cross-feature routing context; G5 lead has
read only explorer and filework, not mutation callers/tests. S/W: adaptive
call tracing, no mechanical transform. Literal Explore unavailable; inherited
model in read-only role. Lead owns all design, review and fixes.

### Recovery implementation evidence

Observed RED -> GREEN: final publication followed by worker failure prevented
retry; an incomplete successful return showed no failure; stale committed source
exports (including a loaded symlink) retained old map content; removal followed
by late publication restored old content; a missing cohort member sent image
navigation into an unrelated cohort. Each behavior now passes its boundary test.
The initial alias fixture exported *over* a symlink (replacing the link), which
correctly did not change the source; it was corrected to a loaded symlink whose
actual target is overwritten. No production alias behavior was changed.

Deliberately removing the queued-publication token check made the lifecycle
restart test fail with the old source replacing the fresh cohort. Source was
restored. The test delays the old queue until the new map has been delivered.
Shutdown and exit use the same accepted UI/provider seam and observe worker exit.

Chosen source-change behavior: retire the entire grouping snapshot and cancel
its worker, retaining the remaining frozen cohort identities for browsing.
Retry explicitly rebuilds grouping and reuses valid favorite representations.
This avoids silently continuing a grouping whose input changed; incremental
per-source regrouping and external filesystem watching are not introduced.
An unrelated exported copy leaves the map intact. Missing-file load retries
stay within a surviving cohort; ordinary viewer behavior applies if none remain.

`make explorer-ui-test`: PASS, 22.215s, no skips. Actual native failure/retry,
cancel/restart, missing-source and favorite freshness cases ran under verified
network denial. `make explorer-test`: PASS. Initial attempts within the agent
sandbox could not install macOS network denial; the authorized tests were rerun
outside that sandbox, retaining the worker's own outbound denial. Focused race,
446-image smoke and final canonical gate evidence follow below.

### Recovery final gate and handoff

- Focused Explorer race suite: PASS, 58.687s. All native/actual-engine tests
  passed without required skips. Synthetic source-change render inspected at
  1280 x 800; message fully readable, stale piles cleared, Update map disabled.
- `make explorer-evaluate TRIAL=smoke`: PASS; all 446 images represented,
  zero failed, 93.189s processing, 1,106.1 MiB worker peak RSS. Run overlapped
  verification load; not a controlled timing comparison or 50k qualification.
- `make verify`: PASS, exit 0. Formatting/TUF/Qodana checks, vet, host build,
  exact shard inventory and all four Linux/amd64 Docker race partitions passed.
  Explorer itself passed in 74.280s under Linux race detection. Retained run:
  `.scratch/race-runs/20260909T165106Z-Mz79ox`.
- `make build`: PASS; refreshed `bin/picfetch`. No app bundle replacement,
  running-app restart, or commit. Pre-existing untracked files are untouched.
- Evidence: [recovery.md](../.scratch/visual-similarity-explorer/evidence/recovery.md).
  Ticket 06 is ready for human trial; the broader plan remains active.

| Task | Spawns budget/actual | Lead review rounds | Full suite | Notes |
| --- | --- | --- | --- | --- |
| Recovery 1 | 0 / 0 | 1 | no | Failed/incomplete retry and queued lifecycle delivery |
| Recovery 2 | 1 / 1 | 2 | no | Read-only route scout; source writes/removals and missing-load regression |
| Recovery 3 | 0 / 0 | 1 | one, passed | Native suites, smoke, synthetic render, canonical gate and build |

## Explorer controls and large-drop performance — 2026-09-09

Deliver the five requested trial refinements: collapsible tags, tag-only grids,
adjustable grouping granularity, highlighted keyboard stack navigation, and a
measured reduction in growing analysis/publication cost. Route: Deep. Existing
accepted production UI/provider and actual-engine input/output seams apply.
The user's running client/worker stay intact; local resource observation continues.
Saved presets and full-library semantic acceptance remain separate.

### Decisions and acceptance criteria

- AC1: Tags start expanded; a toolbar toggle collapses the entire sidebar and
  restores it without changing filters, camera or browsing state.
  Verify: `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/^tags_collapse$' -count=1 -v`.
- AC2: A separate clickable count opens the distinct images with that tag across
  all cohorts, including Unassigned. Counts ignore current checkbox filters.
  The grid freezes membership until reopened; return preserves map/filter state.
  Verify: `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/^tags' -count=1 -v`.
- AC3: Top-right Granularity slider defaults to the existing finest cohorts.
  Broader settings merge related cohorts using a worker-computed hierarchy;
  finer settings restore them. No source decoding/inference or full grouping
  runs when the slider moves. Unassigned remains separate; counts, source
  identities and open grid membership remain correct across publications.
  Verify: `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/^granularity$' -count=1 -v`; `make explorer-ui-test`.
- AC4: +/= and - zoom. Arrow keys select a visible stack in that direction,
  with visible highlighting and camera reveal; Enter opens it. Selection is
  retained across a grid visit and reconciled after filters/publications.
  Verify: `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/^keyboard$' -count=1 -v`.
- AC5: Establish a growing-work replay/profile before changing performance
  behavior. Record baseline, hypotheses, focused red/green regression and
  the same post-change measurement. Preserve exact source accounting and
  cancellation. No unmeasured end-to-end speedup or 50k qualification claim.
  Verify: measured reproduction command recorded below; focused regression,
  `make explorer-test`, and `make explorer-ui-test`.

### Tasks and graph

1. Tags (T0): `internal/ui/explorer/{map,tags}.go`, existing UI/native tests,
   translations. Contract: existing OpenSimilarityCohort receives captured
   tag-member paths. Tests AC1/AC2. Budget: 0 spawns, 2 reviews, no full suite.
2. Keyboard (T0): new `internal/ui/explorer/keys.go`, map renderer and root
   explorer dispatcher, existing UI tests. Contract: `Map.HandleKey(fyne.KeyName)`.
   Test AC4. Budget: 0 spawns, 2 reviews, no full suite.
3. Granularity (T0): similarity grouping/protocol, map/root delivery, existing
   UI/native tests. Contract: immutable `Event.Merges []CohortMerge` from the
   actual grouping representation; slider cuts that hierarchy locally. Test
   AC3. Budget: 0 spawns, 2 reviews, no full suite.
4. Performance (T0; read-only scout): files fixed after the reproduction.
   Test/command AC5. Budget: 1 scout, 2 reviews, no full suite.
5. Handoff (T0): architecture, English/German manuals/catalogues, todos, this
   plan and local evidence. Verify: `make verify`, `make build`, diff checks,
   synthetic rendered QA. Budget: 0 spawns, 1 final complete race gate.

Graph: performance scout alongside 1; 1 -> 2 -> 3; 4 follows measured evidence;
all -> 5. All implementation, review and fixes stay with the lead. Scout gate:
G1 bounded growing-cost question; G2 executable reproduction and source locations;
G3 no repository writes; G4 cold performance replay context; G5 lead owns UI
controls. S/W: adaptive profiling, no mechanical transformation or supplied fix.
Inherited model used read-only because this harness has no literal T3 Explore.

### Controls implementation and measured evidence

- All five criteria have passing focused evidence. Tags collapse without
  losing choices; separate counts open exact distinct members, including
  Unassigned, independently of active filters. Open grids retain their frozen
  identities while later publications arrive.
- The top-right slider cuts a deterministic centroid spanning tree from the
  existing grouping representation. Finest restores base cohorts; broadest
  joins assigned cohorts and retains Unassigned separately. Slider changes
  require no source reads, inference, or UMAP/HDBSCAN run. The hierarchy uses
  quadratic time in base cohort count and linear storage.
- Arrow navigation selects and visibly outlines a directional neighbor,
  revealing it with the existing camera. +/= and - zoom; Enter opens the
  selected stack. Selection follows source identity across map publications,
  filters, and grid return.
- Observed RED -> GREEN: absent collapse button, absent count links, missing
  keyboard zoom, missing slider, missing hierarchy from actual grouping, and
  redundant embeddings in display events. Deliberate mutations to tag paths,
  hierarchy delivery, selection outline, and adjacent-cell collision checks
  each failed their guard; all changes were restored.
- Before/after replay of 1,600 stacks: local packing budget failed at
  240.789ms, then passed at 30.018ms after nearby-cell collision lookup and
  perimeter-only placement search. Approximately 8x for this case; the 50ms
  target is a local optimization probe, not a CI or full-library threshold.
  Behavioral geometry guards cover coincident piles and cell boundaries.
- For 10,000 items with previews omitted on both sides, removing unused
  embeddings reduced display JSON from 97,436,643 to 1,217,090 bytes. Encoding
  fell from 0.292726s to 0.002500s; decoding from 0.595924s to 0.005401s.
  Only worker-to-display snapshots omit vectors; cache and evaluator retain
  them. Native publication, exact accounting and reuse regressions pass.
- Tag scoring and growing cache-directory probes showed no clear per-item
  slowdown within their measured bounds. One live sample showed ONNX MatMul
  active; it cannot explain the whole scan. Full-prefix batch grouping and
  full-library encoding throughput remain unqualified. The original client
  and worker continue with read-only resource observation, which exposes no
  completed-image counts. No end-to-end scan speedup is claimed.
- `make explorer-test`: PASS; all required actual-model tests ran under OS
  network denial. `make explorer-ui-test`: PASS, 25.300s; actual engine 20.01s
  and ordinary UI 4.67s, no required skips. Existing test files and root test
  names preserve their Qodana exclusions and exact 680-test shard assignment.
- English and German offscreen renders inspected with synthetic pixels only.
  Manuals, catalogues, architecture and todos are updated. `make build`:
  PASS; refreshed `bin/picfetch` without restarting the existing window.
- Full control/performance evidence and replay command:
  [controls-20260909](../.scratch/visual-similarity-explorer/evidence/controls-20260909/README.md).
- `make verify`: PASS, exit 0. Formatting/TUF/Qodana checks, vet, host build,
  exact shard inventory and all four Linux/amd64 Docker race partitions passed.
  Explorer passed under race detection in 84.520s. Retained final run:
  `.scratch/race-runs/20260909T181231Z-lsYLza`. No source changes followed
  this gate; only this evidence record was completed. No commit or restart.
  The broader plan remains active for saved presets and full-library trial.

| Task | Spawns budget/actual | Lead review rounds | Full suite | Notes |
| --- | --- | --- | --- | --- |
| Controls 1 | 0 / 0 | 1 | no | Collapse, independent counts and frozen grids |
| Controls 2 | 0 / 0 | 2 | no | Directional selection, outline, camera and focus |
| Controls 3 | 0 / 0 | 2 | no | Actual hierarchy plus local granularity cuts |
| Controls 4 | 1 / 1 | 2 | no | Read-only scout; lead replayed and verified optimizations |
| Controls 5 | 0 / 0 | 1 | one, passed | Native suites, synthetic renders, canonical gate and build |

## Original 50k trial: render transition and interaction observation

The user requested profiling as the original client approached completion,
reporting approximately 48k indexed images. The lead attached native stack
sampling and one-second resource observation to the exact existing viewer and
worker PIDs, with no app relaunch or GUI lookup. Two read-only scouts traced
existing phase signals and bounded symbolization options; the lead retained
interpretation and the evidence record. No application code changed and no
application test/build run was needed for this observational step.

The user confirmed the map appeared, remained running without a reported crash,
and lagged badly zoomed out while behaving relatively fluently zoomed in. The
worker exited; native samples captured the subsequent main-thread construction
and OpenGL texture uploads. Viewer footprint peaked at 9.532 GiB and settled
near 5 GiB. The user then closed the client; its PID disappeared at 20:51:10 CEST.

Both old and current map code visit all piles on camera changes and preserve
all sampled image objects at distant zoom. Fyne retains decoded image pixels
on resize but invalidates textures; its smooth painter performs software
rescaling. Reduced overview detail and texture reuse are recorded as follow-up
candidates in todos.md. No per-frame latency or exact per-image success counts
were exposed, and raw Go stacks could not be authenticated against the replaced
binary. This establishes an observed interaction limit, not full-library semantic
acceptance or an end-to-end speedup. The user called the load an extreme case.

Evidence and reproducible resource aggregation:
[render-transition-20260909](../.scratch/visual-similarity-explorer/evidence/render-transition-20260909/README.md).
The capture is complete; the observers and original app processes have ended.

## Resume on 2026-09-10

The user asked to preserve the following next steps for tomorrow. The next
milestone is **smooth overview navigation with the existing grouping and
browsing features**. The user considers the roughly 50k-image run an extreme
use case: prioritize excellent normal-sized browsing and a usable large-map
overview, rather than adopting an unmeasured universal frame-rate target.

### Start here

- Read this resume section, ARCHITECTURE.md, and the working agreement in
  `.agents/skills/improved_sdd_tdd_cycle.md` before implementation. Continue
  SDD/TDD using the confirmed production UI/provider and actual-engine seams
  recorded near the top of this plan.
- The collapsible tags, clickable tag-count grids, granularity slider,
  keyboard navigation, leaner display messages and faster stack placement
  are implemented and verified. Their final gate is recorded above. The
  user's 50k window ran the older executable, so it did not test those new
  controls or establish their effect on the stress case.
- The user confirmed a visible, working map with severe lag zoomed out,
  relatively fluent interaction zoomed in, and no observed crash. Viewer
  footprint peaked at 9.532 GiB, then settled near 5 GiB. Counts, semantic
  quality and frame latency were not fully instrumented.
- Read the [profile findings](../.scratch/visual-similarity-explorer/evidence/render-transition-20260909/README.md)
  and its `timeline.csv`/`summary.json`. Fyne already retains decoded image
  pixels on resize; repeated JPEG decoding is not the established cause.
  Full-detail piles, smooth software rescaling and texture invalidation are
  the concrete rendering paths to investigate.
- The user closed the original client; its worker and all observers ended.
  Historical PIDs are evidence, not reusable targets. Inspect current state
  before starting or attaching to a client. App-name GUI lookup previously
  opened the installed app instead of attaching to the trial executable;
  avoid repeating that. Keep library images local and do not commit changes.

### Priority order

1. **Briefly trial the current build on a smaller collection.** Check tag
   collapse, exact tag-count grid membership and return, granularity behavior,
   keyboard highlighting/navigation, and overall map usability. Reuse the
   existing demo or a user-selected modest collection; another 50k scan is
   unnecessary for this first check.
2. **Establish a repeatable profiling baseline before changing rendering.**
   Retain the exact executable and matching symbols. Record phase durations,
   completed/failed counts, map construction, first paint, and interaction
   timing. Use consistent zoom/pan gestures and collection sizes so the
   before/after comparison is meaningful. Keep instrumentation local and
   record metadata rather than image contents.
3. **Implement the smallest supported overview optimization with SDD/TDD.**
   Candidates: compact representative/count displays at distant zoom,
   progressive thumbnail detail as the user zooms in, texture reuse across
   camera changes, and less off-screen layout work. First turn the chosen
   change into acceptance criteria with executable checks. Preserve cohort
   membership, deterministic samples, selection, tag filtering, camera
   anchoring and the existing grid-return behavior. Rendering changes must
   not trigger image inference or regrouping.
4. **Validate progressively.** Iterate with small and medium collections and
   focused tests, then exercise the real native UI and ordinary controls.
   Repeat the 50k stress case once the change is ready and profiling is in
   place. Compare latency, allocation/texture work, memory peaks and visible
   behavior; distinguish user observations from measured guarantees. Finish
   application changes with the relevant native suites, `make verify` and
   `make build`; retain the usual localization/shard/Qodana checks when their
   scope changes.
5. **After responsiveness, revisit scan throughput and saved presets.** Use
   stage/count measurements to separate encoding from batch UMAP/HDBSCAN
   cost, and reuse valid saved representations where available. Saved group
   presets remain a later feature with unresolved trait/future-map choices;
   they should not displace the responsiveness milestone.

Keep the broad plan active until its remaining trial and product decisions
are resolved. Record actual results and the next unresolved step here at the
end of the next session.

## Minimum map zoom — 2026-09-09 resumed

Deliverable: enforce a readable minimum zoom on large maps while keeping every
sampled thumbnail. The user explicitly rejected reducing thumbnail detail and
chose a minimum zoom factor instead. This supersedes the earlier overview-detail
candidates. Standard increment inside the active Deep plan; confirmed UI/provider
seams remain in force. The provisional cutoff is more than 100 cohort piles,
with a 0.5x floor (190px-wide piles). Maps of 100 piles or fewer keep 0.03x,
even in a small window. The user clarified "but only when the map gets too big";
the numeric cutoff was offered as an optional preference and is lead-assumed
pending a different choice.
No inference, grouping, sample-count, renderer, persistence or preset changes.

### Acceptance criteria

- AC1: wheel/trackpad, keyboard and toolbar zoom-out stop at the large-map floor without camera
  drift; all fifteen samples remain, zoom-in works, and cohort/Grid return keeps
  the camera and exact full membership.
  `go test ./internal/ui -run '^TestVisualSimilarityExplorer/minimum_zoom_inputs$' -count=1`
- AC2: initial/manual Fit map and automatic expansion respect the same conditional floor.
  Automatic discovery at the floor preserves the user's camera; keyboard and
  pan can reach off-screen piles. Smaller maps continue to fit and zoom out normally; growing and shrinking maps
  re-evaluate whether the floor applies; resizing does not bypass it.
  `go test ./internal/ui -run '^TestVisualSimilarityExplorer/minimum_zoom_fit$' -count=1`
- AC3: existing controls, deterministic samples, selection, granularity,
  source accounting and map/Grid return remain intact.
  `go test ./internal/ui -run '^TestVisualSimilarityExplorer$' -count=1`
- AC4: retain matching native replay source, unstripped binaries, symbol/build
  identity, phase and repeatable interaction timings before/after, using only
  synthetic pixels. Run small and medium cases, then a 50k-source render replay.
  Commands and measured limits will be recorded in the evidence README.
- AC5: `make explorer-test`, `make explorer-ui-test`, `make verify`, and
  `make build` pass. Native render QA covers the actual GL painter; the full
  library's semantic quality and encoding throughput remain unqualified.

### Tasks and ownership

1. Baseline (T0): existing UI controls tests; retained local native replay
   harness and before measurements. One read-only T3 scout finds reusable
   native/profiling artifacts; all interpretations and review stay with T0.
2. Input floor (T0): `internal/ui/explorer/map.go`, existing `explorer_test.go`;
   AC1 red/green. One private conditional zoom-floor helper; no interface changes.
3. Fit floor (T0): same files; AC2 red/green, then AC3. Preserve existing small-map fitting assertions.
4. Gate (T0): native replay/QA, guard mutations, AC5, manuals/todos and evidence.

Graph: baseline -> input floor -> fit floor -> gate. Budget: one scout, two lead
review rounds, one complete final race suite; record necessary overruns.
Scout G1: bounded artifact search; G2: verify returned paths/commands; G3: no
writes; G4: separate historical harness context; G5: not previously inspected
by lead. S/W: adaptive search, not a scripted transform or supplied code.
Literal Explore is unavailable, so the inherited agent is read-only. The
available TDD and diagnosing-bugs skills cover the referenced unavailable
superpowers workflows. No review or fix is delegated.

Honest limit: the zoom floor deliberately prevents fitting every pile of a
large map on screen at once. A synthetic render replay isolates rendering and
does not qualify the private 50k library's inference or semantic result.

### Minimum-zoom implementation evidence

- Observed TDD red: large-map input zoom reached 11.40px and initial Fit reached
  41.18px. Both now stop at 190px. All ordinary explorer UI tests pass (5.244s),
  including the existing small-map fitting/controls tests. New subtests keep
  the same top-level runnable and existing Qodana exclusion.
- Six isolated negative mutations each failed the intended guard: input, Fit,
  map growth, at-floor camera stability, small-map exemption, and exactly 100
  piles. Production source was unchanged by the overlay-based negative checks.
- `make explorer-test explorer-ui-test`: PASS, exit 0; required actual-model
  tests ran under OS outbound denial and the viewer suite passed in 23.914s.
  `make build`: PASS; `bin/picfetch` refreshed without launching a library scan.
- Retained native replay: 30 synthetic sources / 2 piles behaved similarly
  before/after. For 12,000 sources / 800 piles, observed overview-pan median
  changed from 66.43ms to 16.47ms; first map from 8.958s to 4.439s. The changed
  overview scale is intentional, and frame observations include readback.
- The 50,000-source / 3,334-pile replay completed all 43 input/frame steps and
  exited 0: first map 18.429s, overview pan median 34.25ms. Sampled Go heap still
  reached 7.433 GB; no memory ceiling or construction fix is claimed. All
  samples remain, including in the inspected synthetic native screenshots.
- Evidence, reproduction commands, source, unstripped binaries, build identity,
  symbol lists, negative checks and timings:
  [minimum-zoom evidence](../.scratch/visual-similarity-explorer/evidence/overview-20260909/README.md).
- The first preflight found Docker stopped; the CLI started it. A host-only
  shard check included a Darwin-only runnable outside the Linux manifest;
  canonical inventory verification passed inside Docker. `make verify`: PASS,
  exit 0. Formatting/TUF/Qodana checks, vet, build, all 680 Linux UI runnable
  assignments and all four race partitions passed. Explorer passed in 83.740s
  under race detection; final ui-1 package completed in 385.200s. Evidence:
  `.scratch/race-runs/20260909T210256Z-7ZGVuz` and the retained `verify.log`.
  Only documentation closeout followed the successful gate.

| Task | Spawns budget/actual | Lead review rounds | Full suite | Notes |
| --- | --- | --- | --- | --- |
| Baseline | 1 / 1 | 1 | no | Read-only artifact scout; native harness built by lead |
| Input and fitting | 0 / 0 | 2 | no | User-chosen direction; all samples retained; red/green and negative guards |
| Final gate | 0 / 0 | 1 | one, passed | Native suites, build and canonical Linux gate passed |

The wider explorer plan remains active for the user's cutoff/usability trial,
full-library qualification, construction/memory costs, scan throughput and saved
presets. No commit was made; unrelated `:memory:.ses` files remain untouched.

## Map construction and memory — resumed

Status: implemented and verified. The user identified this active plan for
`/implement use tdd and sdd`; the zoom increment is complete, not the wider plan.
Deliverable: reproduce the remaining construction/preview memory cost and make
the smallest measured improvement without reducing sampled thumbnails.
Standard investigation/implementation increment inside this Deep plan. Keep the
confirmed production UI/provider seam; native synthetic replay covers GL costs.
No preset decisions, private-library semantic verdict, inference/grouping changes,
new zoom policy, or full-library qualification are assumed by this increment.

### Acceptance and task sequence

1. T0 baseline: retain a reproducible synthetic replay and allocation/heap
   profiles, executable identity, input counts, and construction/interaction
   measurements. Start with the existing overview replay; record the exact
   adapted command in the new evidence README before any production change.
   One T3 read-only scout traces engine preview dimensions/ownership while T0
   builds the reproduction. No implementation hypothesis is accepted without
   a measured probe. Budget: one scout, one review, no full suite.
2. T0 red/green: turn the measured finding into an explicit behavioral/resource
   criterion at the confirmed UI/provider seam before implementation. Preserve
   complete sampled membership, camera, tags, granularity and grid return.
   Files: existing `internal/ui/explorer_test.go`, and the smallest affected
   explorer production file set established by the profile. Command:
   `go test ./internal/ui -run '^TestVisualSimilarityExplorer$' -count=1`.
   Budget: zero spawns, two reviews, no full suite.
3. T0 verification: repeat the same synthetic measurements and native visual QA;
   negatively verify the new guard, run `make explorer-test`,
   `make explorer-ui-test`, `make verify`, and `make build`. Record actual limits
   and update todos/this plan. Budget: zero spawns, one final full race gate.

Graph: baseline -> measured criterion -> red/green -> verification.
Scout gate: G1 bounded preview lifecycle search; G2 lead checks returned source
locations and reported dimensions; G3 read-only, no shared writes; G4 independent
engine/cache context; G5 lead has only renderer/replay context. S/W: adaptive
cross-file ownership tracing, not a mechanical transformation or supplied fix.
Literal T3 Explore is unavailable; inherited model is used strictly read-only.
All spec, design, review and fixes stay with the lead.

### Scope clarification and first measured slice

The user clarified that 50k images is an edge case and optimization must not
decrease usability or looks. Normal-sized maps are the priority; keep all
samples, rendering quality and controls. No viewport culling or reduced detail
is selected. The old replay used 256x192 Q85 previews, whereas the engine makes
JPEG Q80 previews at a maximum edge of 160. The new baseline uses 160x120 Q80
previews with independently owned encoded buffers. The old heap figure is not
a production-equivalent budget.

The initial retained native profile (3,000 sources) attributes 92% of live heap
to decoded JPEG buffers. Investigate separately: unchanged-preview decodes on
selection; temporary construction allocations; complete pile reconstruction
on publications. The first candidate is ordinary two-pile keyboard navigation.

- AC1: forty alternating selections on a completed 30-image/two-pile map
  allocate under 8 MiB after warmup, preserving all 15 samples per pile and
  the rendered image when selection returns to its starting state. This is
  a broad allocation regression bound, not a frame-rate requirement.
  `go test ./internal/ui -run '^TestVisualSimilarityExplorer/navigation_preview_reuse$' -count=1 -v`
- AC2: unchanged controls, samples, source replacement, tags, granularity,
  highlighting, exit resource release and grid return pass the complete
  `TestVisualSimilarityExplorer` suite. Native before/after screenshots must
  match for identical states; compare small and medium maps using the corrected
  replay, and retain real allocation/timing evidence without extrapolating to 50k.
- Baseline replay command (before production edits):
  `.scratch/visual-similarity-explorer/evidence/map-memory-20260909/before-production-previews -items 30 -out .scratch/visual-similarity-explorer/evidence/map-memory-20260909/before-small`

The user subsequently invited viewport-only memory retention if it is elegant
and does not visibly stutter. This supersedes the earlier exclusion of viewport
culling for this investigation. The candidate keeps every compressed preview
and sample identity, admits decoded pixels one pile-width beyond the viewport,
and releases them beyond two pile-widths to avoid boundary churn. It adds no
background worker or placeholder state: newly exposed samples must be ready
before the corresponding frame is drawn. Implement only if native replay keeps
appearance identical and supports smooth ordinary boundary crossings.

- AC3: panning away from a normal 300-image map releases at least 4 MiB of
  decoded pixels; returning restores every sample immediately with identical
  rendered output and complete Grid View membership. Zoom, resize, selection
  and tag controls must still reveal complete previews. Test through existing
  provider/input/canvas boundaries:
  `go test ./internal/ui -run '^TestVisualSimilarityExplorer/viewport_previews$' -count=1 -v`
- The minimum-zoom input guard will first pan its target pile into view before
  asserting its 15 drawn thumbnails; offscreen render-tree residency is no
  longer the feature contract. Sampling and the conditional zoom floor remain
  unchanged. All production changes stay within explorer/map.go and tags.go;
  existing root UI subtests retain the same shard and Qodana assignments.

### Implementation and evidence

- RED: 40 selections on a normal two-pile map allocated 79.91 MiB for unchanged
  previews; GREEN: under 0.01 MiB. Pile selection redraws no longer call the
  source-decoding `canvas.Image.Refresh` for unchanged images.
- RED: panning away from a 300-image map retained all decoded pixels (31.63 MiB
  before and after). GREEN: the full focused suite measured 56.43 -> 47.02 MiB,
  releasing 9.41 MiB. Returning restores every sample synchronously, with an
  identical captured map and complete cohort browsing.
- The renderer retains encoded sources and complete sample identities, admits
  pixels one pile-width outside the viewport, and evicts beyond two pile-widths.
  Distant renderers expose no children, so Fyne's native minimum-size traversal
  cannot decode them again. This adds one per-pile state bit and no workers,
  queues, placeholder state, resampling, quality reduction, or source I/O.
- Four overlay mutations failed their intended guards: redundant selection
  decode, disabled pixel eviction, disabled same-size pixel restoration, and
  removal of the wider release margin (15.84 MiB of churn during tiny reversals).
  Production files were not modified by negative checks.
- Native replays use 160x120 Q80 previews with independent encoded buffers,
  matching the production preview contract. Across 30/300/3,000 sources, all
  90 before/after PNG comparisons matched exactly; each of six runs completed
  all 109 input/frame measurements. Synthetic screenshots were inspected.
- The 3,000-source construction frame changed from 603.42 to 215.07 ms, and
  sampled retained construction heap from 123.55 to 34.98 MiB. Pixel refills
  add UI work: boundary-crossing maximum 4.68 ms at 300 sources and 3.69 ms at
  3,000. Observed frame timings include readback and scheduling, not pure FPS.
  Normal 30/300-image construction stayed roughly comparable; no universal
  speedup or stutter-free guarantee is claimed. The user prioritizes normal
  usability; a new 50k trial is not required for this increment.
- Evidence, replay source, matching unstripped binaries/overlays, hashes, heap
  profiles, frame comparisons, negative checks and limits:
  [map-memory evidence](../.scratch/visual-similarity-explorer/evidence/map-memory-20260909/README.md).

All ordinary explorer subtests pass (6.083s). Required native suites and the
canonical gate passed as recorded below. No new root test name,
test file, user-visible string, interface or package was added; existing shard,
Qodana and locale assignments remain applicable.

| Task | Spawns budget/actual | Lead review rounds | Full suite | Notes |
| --- | --- | --- | --- | --- |
| Baseline | 1 / 1 | 1 | no | Scout preview contract verified; corrected old synthetic assumptions |
| Red/green | 0 / 0 | 2 | no | Normal selection plus user-authorized viewport retention; all fixes inline |
| Verification | 0 / 0 | 1 | one, passed | Native visual/performance replay, four negative mutations, final gate |

Final gate: `make explorer-test explorer-ui-test` passed (23.986s UI suite),
with all required real-model tests under OS outbound denial. `make verify`
passed, exit 0: formatting/TUF/Qodana checks, vet, build, exact 680-runnable
Linux shard inventory and all four Docker race partitions. Explorer passed
under race detection in 99.690s; the final UI partition completed in 405.835s.
The raw race stream explicitly records passing `navigation_preview_reuse`,
`viewport_previews`, and `viewport_margin` subtests. Retained gate evidence:
`.scratch/race-runs/20260909T213521Z-znXD0M`. `make build` passed and refreshed
`bin/picfetch`. Only documentation changed after these successful gates.

This increment is complete. Keep the broader plan active for the user's native
usability/cutoff verdict, scan-throughput measurement, full-library qualification,
and saved presets with their unresolved product choices. Further construction
or rendering work should follow a measured ordinary-use problem; the 50k case
alone does not justify reduced quality or more complicated browsing behavior.
No commit was made. Unrelated `:memory:.ses` files remain untouched.

## Production throughput measurement — resumed

Status: complete and verified. Resumed the next independently actionable item after the
completed viewport increment: measure production scan stages on the supplied
modest demo. This increment does not reopen rendering or preset decisions and
cannot qualify the private 50k library or infer the user's native verdict.
Route: Standard increment within the Deep plan. The accepted actual-engine
command seam remains the test boundary; no additional test seam is introduced.

### Tasks and acceptance criteria

1. T0: expose immutable cumulative measurements on production worker events:
   elapsed/setup, source read/decode, inference, previews, cache I/O and grouping
   stages, with attempted-inference and completed-map counts. Snapshot counts
   distinguish successfully encoded, reused and failed sources. Test through
   the real subprocess using pinned assets and explicit network denial.
   Files: internal/similarity/{similarity,analyze}.go and existing native tests.
   Verify: `make explorer-ui-test`. Budget: zero implementation spawns, two
   lead review rounds, no full suite during iteration.
2. T0: add a bounded production profiling command to scripts/explorereval,
   retaining metadata-only JSONL events and a completed summary in a fresh
   evidence directory. Report timing definitions and exact source accounting;
   separate source analysis from every UMAP/HDBSCAN/layout publication. Exercise
   cold and repeated analysis with an isolated favorite cache through the same
   production client. Missing prerequisites, cancellation or incomplete runs
   must not yield a completed summary. Files: command main/profile and existing
   command/native tests, Makefile, README, ARCHITECTURE.md.
   Verify: `make explorer-test`; `make explorer-profile` on the supplied demo.
   Budget: one read-only scout, two lead reviews, no full suite during iteration.
3. T0: inspect actual aggregate evidence, negatively verify guards, run the
   canonical `make verify` once and `make build`, update this plan and todos.
   Keep full-library qualification and unresolved product choices open.

Graph: event measurements -> production profiler -> measured demo and final gate.
Scout gate: G1 bounded cache/trial-fixture search; G2 returned file:line facts
verified by the lead; G3 read-only; G4 independent cache fixture context;
G5 lead has read only timing flow and event contract. S/W: cross-file tracing,
not a mechanical transformation or supplied implementation. Literal Explore
is unavailable, so the inherited model serves only as a read-only scout.
All design, review and fixes stay with the lead; no git commit.

### Throughput implementation and evidence — 2026-09-10

- RED: the real six-image cold/cache test received zero stage/count measurements.
  GREEN: cold processing reports actual setup/decode/inference/preview/cache/tag/
  grouping time; fully cached processing reports zero model/decode/inference/
  preview time. Existing semantic and cache-identity assertions still pass.
- RED: the production profiling command refused `-trial throughput`. GREEN:
  it runs the actual Client/WorkerMain, serializes aggregate event snapshots,
  observes cold and warm worker exits and removes its isolated favorite cache
  before atomically publishing a completed summary. Existing evidence is never
  overwritten. Source changes, cache warnings, output failure and cancellation
  refuse completion. Events/report exclude paths, images, previews and vectors.
- A real 513-file command scenario processes exactly 512; broken sources remain
  failures with zero inference attempts. Cancellation after actual progress
  preserves its trace without a completed summary. Profiling tests run outside
  the older evaluator's parent sandbox because the production client creates
  its own explicit outbound-denied child. All native prerequisites are required.
- Negative Go overlays reset publication counts, inserted a synthetic source
  name into the trace, and raised the sample cap to 513. Each targeted guard
  failed for the intended reason; production files were not mutated. The cold
  measurements guard was already observed failing before instrumentation.
- `make explorer-test explorer-ui-test` passed on final code: profiling command
  2.32s (including its bounded scenario), ordinary command/native evaluator
  guards, and every controlled/real viewer scenario (UI package 24.058s).
- `make explorer-profile` processed all 446 supplied demo inputs: cold 89.484s,
  warm 1.317s; zero failures, 446 warm reuses, 46 cohorts and 55 Unassigned in
  each pass. Cold stage times: decode 33.768s, inference 27.320s, previews
  26.810s, grouping 1.073s. Temporary cache occupied 7,613,236 bytes.
  Both passes used CPU and final-only publication. All 896 event rows had
  monotonic elapsed/cumulative work and exact counts; the retained binary's
  digest matches its report. The trace's 100-image inference windows range
  from 13.65 to 18.36 images per encode-second; this mixed-size/format sample
  does not establish an order-dependent slowdown.
- Evidence: [production throughput](../.scratch/visual-similarity-explorer/evidence/throughput-gSB1MB/README.md).
  The exact executable and report are retained. The final mechanical scan
  consolidation is covered by native tests; it changes no measured inference,
  preview, grouping or sampling behavior. Existing evaluation/profiling commands
  now share directory validation/scanning rather than duplicate it. The shell
  evaluator recognizes throughput artifacts when passed that trial explicitly.

No renderer/UI flow, source sampling limit, algorithm, quality, model asset or
platform support changed. This supplies the missing modest-corpus stage/count
evidence; native responsiveness, per-stage RSS, 50k scaling and semantic/user
acceptance remain unqualified. Decode and full-quality preview generation are
measured targets for a future improvement; do not assume inference is the sole
cost. Preset scope/application choices were presented again asynchronously;
no answer is assumed, and no preset implementation starts without resolution.

No new test file or top-level root UI test was introduced, so existing exact
Qodana exclusions and shard rows apply. ARCHITECTURE.md and the command README
locate the new production profiler and define nested versus additive timings.

| Task | Spawns budget/actual | Lead review rounds | Full suite | Notes |
| --- | --- | --- | --- | --- |
| Worker measurements | 0 / 0 | 1 | no | Real cold/warm plus cumulative-publication guards |
| Production command | 1 / 1 | 2 | no | Read-only cache scout; bounded/failed/canceled command evidence |
| Final gate | 0 / 0 | 1 | one, passed | Native suites, make verify and make build all passed |

Final gate: `make verify` exited 0, including formatting/TUF/Qodana checks,
vet/build, the exact 680-runnable Linux shard inventory and all four race
partitions. The explorer scenario passed under race detection in 99.350s;
the final UI partition completed in 398.947s. Raw artifacts:
`.scratch/race-runs/20260909T220239Z-N9Wixf`. `make build` passed and refreshed
`bin/picfetch`. Native logs, the three failing mutation outputs, source digests
and final build/gate logs are retained in
`.scratch/visual-similarity-explorer/evidence/throughput-checks/`.
All 38 local links across the edited plan/todos/command README and whitespace
checks pass. Lead standards/spec review found no outstanding findings; shared
scan consolidation and all fixes were performed inline. Only documentation
changed after these verification commands.

This increment is complete; the broader plan stays active. The next technical
candidate is a measured investigation of source decoding and preview generation
with identical quality, followed by the existing native usability/large-library
qualification work. Saved presets still need the two explicitly pending product
choices. No git commit was made. Both unrelated `:memory:.ses` files are untouched.
