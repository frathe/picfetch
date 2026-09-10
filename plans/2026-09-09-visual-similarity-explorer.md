# Visual similarity explorer implementation

## Approved continuation — reusable presets and native trial tooling, 2026-09-10

Status: approved continuation implemented and verified. Ronin approved both increments, presets first,
using the existing UI/provider and actual offline-engine test seams.

Decisions: global local preset library; existing visual tags plus file type,
oriented dimensions/orientation, camera make/model and capture-date ranges;
AND matching, including metadata-only rules. Explicit preview/application to
Unassigned only. Editing reviews the linked current cohort too; removed members
return to persistent Unassigned. Deleting a rule preserves and detaches its cohort.
New arrivals require another explicit review; pending saved members survive.
Other favorites retain reviewed memberships until explicit application.

Tasks (T0 inline, sequential; two review rounds each, extras recorded):

| Task | Contract/files | Acceptance command |
|---|---|---|
| P1 | similarity Item facts and cache backfill; real local UI suite | `go test -tags explorertrial ./internal/ui -run '^TestVisualSimilarityExplorerLocal$/^presets' -count=1 -v` |
| P2 | explorerpresets rule/store; explorer UI browser/editor and root composition | `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/^presets' -count=1 -v` |
| P3 | favorite cohort links/Unassigned overrides, legacy migration and failures | P2 plus `go test ./internal/favstore -run '^TestCohorts' -count=1 -v` |
| N1 | explorertrial operational recorder; UI receipt/application/exit/visits | `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/^trial_recording' -count=1 -v` |
| N2 | launch flag, isolated native library runner and process ownership | `go test ./internal/launch ./scripts/explorereval -count=1` |
| Gate | docs/locales/manifests, negative guards, native public-fixture trial | `make explorer-test`; `make explorer-ui-test`; `make verify`; `make build` |

Graph: P1 -> P2 -> P3 -> N1 -> N2 -> gate. One behavior red/green at a
time. No implementation delegates; all reviews/fixes remain lead-owned.
Canonical full suite once after both increments. Native trial storage includes
isolated presets as well as app identity, Favorites and updates. OS denial
precedes source reads. Collection is separate from qualification; no private
library run is authorized by this increment. Structured traces omit image
metadata as well as paths/pixels/vectors. UI application timing is not paint.

Handoff must list open tasks in this milestone, reflecting actual evidence.
Keep the broad plan active for library/resource/native usability qualification.
Do not commit. Existing unrelated `:memory:.ses` files remain untouched.

### Continuation implementation evidence (in progress)

P1 real offline red: facts were absent. Green: oriented RAW dimensions, camera
and calendar date. Legacy-cache red: reused representation had no facts; green
backfilled facts with zero inference attempts. The outer tool sandbox initially
refused sandbox-exec; focused native tests ran with approved host execution and
the worker's actual network denial intact.

P2/P3 UI reds: missing Presets, image-property/model fields, linked-edit review,
Favorite restoration, Delete preset and Analyze's Save as preset. Each passed
after its corresponding implementation. Current presets use a virtual match
list and a separately tracked preset waitgroup on the same Explorer UI queue,
so they remain operable during a live analysis.

Focused preset race suite passed (8 scenarios, 15.49s test time). It covers
frozen preview arrivals, protected cohorts, stale targets, linked removals and
restart persistence. Existing create-cohort regression scenarios also passed.
Legacy Favorite array migration and cancellation/replacement/removal storage
guards passed. One incorrectly constructed combined test regex matched no root
UI tests; this was identified and rerun with correct anchored filters. It was
not counted as evidence.

P2/P3 review added explicit incompatible-rule guidance, preserved unsupported
conditions until review, validated known file types and absent dimensions, and
covered pending Favorite members plus failed membership writes. Negative guards
were observed failing with rollback/pending preservation removed, restored, and
passed in the complete focused race filter (`internal/ui` 43.153s,
`internal/favstore` 1.502s).

N1 recorder reds/greens distinguish receipt from UI application, observe canceled
worker exit and reject completed events with mismatched source identities.
N2 parser/startup reds/greens cover isolated storage, disabled updates, automatic
ordinary scan-to-map entry and truncation refusal. Direct native launch without
OS denial exited 1 before creating evidence or reading source files. The native subprocess runner
passed retention/no-overwrite/cancellation tests under race (4.035s).

Native validation: the unmodified retained app completed 3 and 600 public sources
with zero failures; both recorded application/worker exit and finalized collection
with exit 0 and Qualified=false. No private source library was opened. Computer
Use transport failed before app connection, including after a Node reset. A
retained native Fyne/GL replay overlay captured six frames after a visible paint
marker; editor, three-match preview, named pile and exact cohort grid were
visually inspected. Direct desktop dragging and production paint latency remain
unverified. See [evidence](../.scratch/visual-similarity-explorer/evidence/presets-native-20260910/README.md).

`make explorer-test` and `make explorer-ui-test` passed (the latter 28.074s).
`make verify` completed with exit 0, including formatting/TUF/exclusions,
vet/build, exact shard checks and the complete Linux/amd64 Docker race suite.
UI shards passed in 468.156s, 298.360s and 286.199s; the full Explorer scenario
passed in 157.120s. Artifacts: `.scratch/race-runs/20260910T104425Z-ZK5ltM`.
The final `make build` also passed. Both native library runs and the replay
exited 0; direct unprotected trial launch was correctly refused with exit 1. Manuals, locale parity, architecture and local tracker
were updated. No new test files or top-level root UI tests were added, so exact
Qodana exclusions and shard assignments remain unchanged.

Ledger: one read-only scout reused for bundle/bridge reconnaissance; zero
implementation/review delegates. T0 kept all implementation and review fixes.
P1/N1 used two review rounds; P2/P3/N2 used three (the extra rounds closed
compatibility, independent-save and native process/evidence findings). One full
canonical suite, at the final gate; focused suites during the slices.


Date: 2026-09-09
Status: reusable presets and isolated native collection tooling implemented and verified, alongside cohort persistence, semantic tags, recovery, map refinement and throughput work; full-library/native qualification remains open
Route: Deep — new local analysis subsystem and cross-feature UI behavior
Request: `/implement use tdd and sdd`
Spec: [Visual similarity explorer](../.scratch/visual-similarity-explorer/spec.md)
Tickets: [Execution sequence](../.scratch/visual-similarity-explorer/ticket-breakdown.md)
Current increment: [Reusable presets and native trial tooling](#approved-continuation--reusable-presets-and-native-trial-tooling-2026-09-10), complete. The broader milestone remains active.

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

## Granularity release — 2026-09-10

Status: complete and verified.
Request: only rebuild the map when the Granularity slider is released, or debounce.
Route: Standard increment (two code/test files plus existing manuals and tracking).
Owner: T0 inline; no spawns, one lead review, one final race gate.
Seam: previously accepted viewer/provider boundary, real Slider drag/release,
click and keyboard input, map/cohort contents and progressive event delivery.

Acceptance: dragging changes the thumb without changing map grouping; release
applies the chosen grouping once. Publications during the gesture retain the
last applied grouping. Clicks/keyboard still apply, Unassigned stays separate,
open cohorts keep their captured members, and reopening resets to finest.
Verify: `go test ./internal/ui -run '^TestVisualSimilarityExplorer$' -count=1 -v`;
`make explorer-ui-test`; `make verify`; `make build`.
Files: internal/ui/explorer/map.go; existing explorer_test.go; English/German
manuals; command README; this plan; todos.md. No new test file/root runnable,
translation key, worker, timer, package or public interface.

RED: `dragging granularity rebuilt the map before release` (0.18s). The cause
was the synchronous OnChanged rebuild. The pinned Fyne slider's OnChangeEnded
fires on release, taps, SetValue and keyboard input, so debounce is unnecessary.
The focused guard passed after changing the callback (0.22s).

Second RED: `analysis publication applied unfinished granularity or moved the
thumb` (0.20s). Retain the applied granularity separately from the live thumb
and use it for every map rebuild; commit it at change end and reset it on exit.
The focused scenario passed (0.23s), including click/keyboard and cohort return.
The feedback loop directly reproduced the requested behavior; broad hypothesis
fan-out, instrumentation and bisection were unnecessary for this callback fault.

`make explorer-ui-test` passed on the final code (24.049s), including the
release/publication/click/keyboard/reopen scenario and both existing minimum-zoom
scenarios. A Go overlay deliberately omitted the applied-value reset; the same
scenario failed with `reopening the explorer retained the previous applied
granularity`. The overlay did not modify production files. Native build passed
and refreshed `bin/picfetch`. `make verify` exited 0: formatting/TUF/Qodana,
vet/build, exact 680-runnable inventory and all four Linux Docker race partitions
passed. The full explorer scenario passed under race detection in 103.400s.
Raw artifacts: `.scratch/race-runs/20260910T052304Z-2HepFT`.

Native/gate/build logs, source digests and the negative overlay are retained in
`.scratch/visual-similarity-explorer/evidence/granularity-release/`.
All 38 local documentation links and whitespace checks passed. Lead review
found no outstanding findings. Cost: zero spawns, one lead review, one full race
gate. No manual native GUI trial is claimed; the agreed production UI/provider
seam exercises actual slider input methods and rendered cohort membership.
Only plan/evidence closeout changed after verification. No git commit was made;
the unrelated `:memory:.ses` files remain untouched.


## Source throughput investigation — resumed 2026-09-10

Status: complete and verified. Profiled source decoding and preview generation,
then removed per-pixel color allocations from canonical EXIF correction while
preserving source pixels and all demo outputs. Route: Standard
increment within the Deep plan. Existing accepted engine/command and UI
boundaries apply; all design, review and fixes stay with T0.
Saved presets still await trait/application answers (presented asynchronously);
full-library/native semantic qualification requires the user's trial verdict.

### Acceptance and task sequence

1. T0 recon: capture CPU profiles from the real outbound-denied production
   worker on the unchanged 446-source demo, retaining binary, overlay and
   aggregate stage/count output. Identify a measured hot path before editing.
   Verify: inspect `go tool pprof -top` and completed `profile.json`.
2. T0 red/green: add an executable resource-regression criterion at the already
   accepted actual-engine command seam for the selected bottleneck. Preserve
   canonical oriented source pixels, model/preprocessing, preview kernel/size/
   quality, source accounting and cache validity. Derive concrete criteria
   from the profile before writing the test. Verify: focused native command
   tests, actual failure output followed by green, and independent pixel parity.
3. T0 gate: compare the same demo using the production profiler, run affected
   native suites and canonical `make verify`, refresh `make build`; record
   actual improvements and honest limits here and in todos.md.

Graph: profile -> bounded resource guard -> minimal optimization -> parity,
production measurement and final gate. Budget: one read-only scout (actual 1),
two lead review rounds, one complete race suite after iteration.
Scout gate G1-G5: bounded canonical resampling/orientation trace; source-line
and local command evidence; zero writes; isolated imaging/dependency context;
lead had only traced worker stage accounting. S/W: adaptive source tracing,
no deterministic edit or supplied implementation. Literal Explore is absent;
the inherited model serves read-only. No review is delegated.


### Selected slice and executable criteria

CPU evidence: canonical ApplyOrientation accounts for 4.74 sampled CPU seconds,
with 2.88 seconds under color-interface boxing; rotate90CW contributes 4.44s.
Resampling is larger (35.79s combined kernel work) but changing its kernel or
8-bit-preconverting YCbCr would change output precision. Optimize the existing
orientation loop's color access first; leave full-quality resampling intact.

AC1: complete real offline evaluation of a 1024x768 orientation-6 JPEG uses at
most 100,000 Go allocations, leaving room for inference/layout while excluding
one or more boxed objects per pixel. Test: TestRealEvaluationOrientationAllocations
in existing scripts/explorereval/trial_test.go; verify with `make explorer-test`.
AC2: all eight orientations preserve canonical RGBA pixels, nonzero bounds,
alpha, source immutability and generic image compatibility. Verify existing
public imaging orientation/decode/export guards, plus independent pre-change
versus post-change fixtures and actual-engine source outputs.
AC3: unchanged demo input/output and cold/warm accounting; compare production
profiles and retained result fingerprints. Verify `make explorer-profile` and
`make explorer-ui-test`; finish with `make verify` and `make build`.
Files: internal/imaging/orientation.go; existing native command test; plan/todos.
No new package, model/version, background work, UI string or feature interface.


### Implementation and observed TDD evidence

The real offline command first failed at 1,574,859 allocations for one 1024x768
orientation-6 JPEG; after direct premultiplied RGBA64 access it passed at 1,994.
Expanding the same command criterion to orientations 2-8 failed for each unfixed
transform (1.57-2.36 million allocations); after the corresponding loop changes
all passed at 1,362-1,999 allocations. No pixel-coordinate formula changed.
Standard images avoid color.Color boxing; uncommon image.Image implementations
retain compatibility through a local adapter with the same channel truncation.

The existing public imaging regression boundary additionally covers eight
pixel formats, all eight orientations, nonzero subimage bounds/stride, fractional
alpha, 16-bit channels and generic-only access. Literal source-position tables
provide the mapping oracle. This broadens shared-path regression coverage in
existing orientation_test.go; no new file or feature seam was introduced.
A negative Go overlay forcing opaque alpha failed the new pixel guard for the
RGBA, NRGBA, RGBA64, NRGBA64 and paletted cases. Production files were unchanged
by the mutation; the final native/race gates use the restored source.


### Source-throughput final evidence and handoff

- Matched retained production runs on all 446 demo sources: 88.727s before,
  82.153s after (7.4% less total time); decode/read/hash/orientation 32.556s to
  24.863s (23.6% less). Warm exits were 1.354s and 1.383s, with all 446 reused.
  Both cold/warm passes in both versions produced the same representation/tag,
  preview-byte and cohort/position fingerprints. All completed with zero failures,
  46 cohorts and 55 Unassigned. This is one local observation; brief native test
  work overlapped the before pass, so it is not a statistical speed guarantee.
- `make explorer-test explorer-ui-test` exited 0. The actual engine allocation
  guard ran all seven non-identity EXIF cases; the real worker/cache/cancellation
  and controlled viewer scenarios passed, with no required skips. UI: 24.967s.
- `make explorer-profile` on final source without either measurement overlay
  exited 0 and verified the same 446-source identity, zero failures and complete
  warm reuse. That command overlapped race verification; its timings are not
  used for the before/after comparison.
- `make verify` exited 0: formatting, TUF/Qodana checks, vet/build, exact
  680-runnable UI inventory, and all four Linux race partitions passed.
  Imaging: 35.014s; explorer scenario: 101.430s; final ui-1 partition: 410.653s.
  Raw artifacts: `.scratch/race-runs/20260910T071503Z-tyi366`.
- `make build` exited 0 and refreshed `bin/picfetch`. No production/test code
  changed after verification. Existing exact Qodana exclusions still apply;
  no root UI runnable, package, user-visible string or model version was added.
  Lead standards/spec review found no outstanding findings. All source fixes
  were performed inline. Documentation links and whitespace checks passed.

[Detailed source-throughput evidence](../.scratch/visual-similarity-explorer/evidence/source-throughput-20260910/README.md)
retains binaries, CPU/negative overlays, failing and passing commands, fingerprints,
stage/count reports and final logs. The source library was processed locally
under outbound denial; its pixels were not viewed or uploaded. No native GUI
trial, peak-RSS reduction, 50k scaling or semantic acceptance is inferred.

| Task | Spawns budget/actual | Lead review rounds | Full suite | Notes |
| --- | --- | --- | --- | --- |
| Profile and EXIF allocation reduction | 1 / 1 | 2 | no | Read-only scout; two observed red/green slices; exact pixel/output parity |
| Final gate | 0 / 0 | 1 | one, passed | Native suites, normal profiling command, make verify and make build |

This increment is complete. The wider plan stays active for native usability
and full-library qualification, and for saved presets whose trait/application
choices remain unanswered. Further source work should target the measured
full-quality resampling cost only with pixel-equivalent evidence. No commit was
made; both unrelated `:memory:.ses` files were left untouched.

## Full-quality preview resampling — 2026-09-10

Status: complete and verified. Standard increment within the accepted Deep plan. Deliver
lower-workspace, faster JPEG previews with identical CatmullRom output. Existing
actual-engine command and canonical imaging regression boundaries apply. The
shared export path must preserve its size, pixel, source-immutability and format
contracts. Inference preprocessing, native UI behavior and saved presets are
outside this measured slice; preset choices have been asked asynchronously.

Evidence selecting the slice: the retained production CPU profile attributes
29.57 sampled CPU seconds to YCbCr 420/422 horizontal filtering. The pinned
scaler converts source colors for every filter contribution and retains
32 * destination width * source height bytes of intermediate pixels.

AC1: the real offline evaluation of one 3072x4096 ordinary JPEG allocates at
most 64 MiB of Go memory, including decode, inference, preview and grouping.
Verify: `make explorer-test`, including TestRealEvaluationPreviewWorkspace.
AC2: public ScaleForExport returns byte-identical pixels to the pinned
CatmullRom scaler for JPEG subsampling modes, nonzero bounds, thin images and
different downscale ratios, and leaves input unchanged. Other formats keep
their current path. Verify: focused TestScaleForExport tests and a deliberate
precision-loss mutation that the parity guard rejects.
AC3: all 446 demo representations/tags, preview bytes and cohort positions
remain identical in matched production cold/warm runs; every source is
accounted for and warm reuse is complete. Retain timings and exact executables.
Verify: production profiling command with metadata-only fingerprint overlay.
AC4: `make explorer-test explorer-ui-test`, `make verify`, `make build` pass.

Tasks: (1) T0 baseline and AC1 red; (2) T0 minimal row-based CatmullRom path,
AC1 green and AC2 parity; (3) T0 production comparison, negative guard, review
and final gate. Files: internal/imaging/thumbnail.go and a private resampling
file, existing thumbnail_test.go and scripts/explorereval/trial_test.go,
ARCHITECTURE.md, this plan and todos.md. No new dependency or background work.
Graph: baseline -> red -> green -> parity -> production comparison -> gate.
Budget: one read-only scout, two lead review rounds, one full race suite.
Scout G1-G5: bounded pinned scaler trace; source locations/commands as oracle;
zero writes; dependency implementation isolated from lead's production/test
context; no previously held dependency context. S/W: adaptive source reading,
not a scripted edit. Literal Explore is unavailable; inherited model is read-only.
All design, tests, implementation, review and fixes remain with T0.

### Preview resampling implementation evidence

- Frame/spec/recon/plan/delegation gate complete: one Standard performance
  increment, existing accepted boundaries, one read-only scaler scout.
- AC1 red: 74,622,424 allocated bytes in the full real offline command;
  green: 51,364,280 bytes. The final guard requires a successful 768-value
  representation and preview delivery as well as the 64 MiB resource limit.
  Restoring the old call site through a Go overlay still fails the final guard
  at 74,621,240 bytes. These are Go allocations, not a peak-RSS measurement.
- AC2: the retained public scaler comparison covers six YCbCr subsampling
  modes, varied reduction ratios, thin dimensions and strided subimages with
  zero, positive and negative origins. Every output byte matches the pinned
  CatmullRom implementation and sources remain unchanged. An early mismatch
  exposed reused rows skipped by zero-weight taps; explicit row identities
  fixed it. Removing low color bits via overlay fails the guard in every
  optimized mode. Uncommon 411/410 layouts retain the original generic path.
- The first production comparison completed every one of 446 sources, with
  zero failures and complete warm reuse. All four cold/warm representation/tag,
  preview and group/position fingerprints match. Final reviewed source is
  measured separately because its uncommon-layout fallback was added in review.
- `make explorer-test explorer-ui-test` passed, including the final resource
  guard at 51,313,160 bytes and native UI scenarios in 25.741s. Focused imaging
  and evaluation packages passed in 2.526s and 0.850s. Formatting passed.
- Lead review closed its source-format and skipped-work findings inline.
  No new test file, root UI runnable, dependency, interface or goroutine was
  introduced; existing exact Qodana exclusions and UI shard counts apply.

Final production measurements, canonical race gate and build are recorded below.
[Detailed evidence](../.scratch/visual-similarity-explorer/evidence/preview-resampling-20260910/README.md)
retains executables, source overlays, failures, successes and aggregate results.

### Readable tag-vector source — user steering during verification

The user requested a human-readable checked-in representation of
`tag-vectors.bin` that is converted for use. Store a JSON object keyed by tag
identity, with decimal float32 arrays, and decode at tagger startup. No generated
binary or additional contributor build prerequisite is needed. Preserve exact
float32 bits and the existing canonical little-endian numeric checksum, so no
tag/model/cache version changes. Update the regeneration command and its docs.

Owner: T0 inline; no delegation (lead holds the complete small loader/generator
context). Files: internal/similarity/{tags.go,tag-vectors.json}, remove the old
binary after exact conversion, scripts/explorertags/{main.go,README.md},
ARCHITECTURE.md, this plan, todos.md. Test boundary remains the real offline
engine command and production UI/provider path; existing tag/ambiguity/reuse
tests provide behavior coverage. Red: embed the converted JSON while the loader
still expects binary; the real command must fail invalid semantic assets.
Green: decode/validate keyed float32 vectors and keep canonical checksum and
normalization validation. Verify exact old/new numeric bits, real offline tests,
generator reproduction where local assets exist, then the single combined
`make verify` and `make build` gate. No new test seam or test file is needed for
this representation-preserving change. Budget: zero spawns, one lead review,
reuse the pending final full suite for both changes.

### Combined verification record

The reviewed resampler's 446-source run completed in 74.608s, versus 82.113s
before (9.1% less time); previews took 20.750s versus 27.053s (23.3% less).
Warm completion was 1.366s with all 446 representations reused. All six
before/candidate/final cold/warm representation/tag, preview and group/position
fingerprints match, with zero failures. These are local observations; the
baseline briefly overlapped focused compilation and one resource test.

Readable-vector TDD: embedding JSON before updating the loader caused the real
command to fail with `invalid semantic tag assets`. After decoding keyed
float32 arrays, `make explorer-test explorer-ui-test` passed (UI 24.588s).
All 23,808 values are bit-identical to the old binary; the existing numeric
digest is unchanged. The updated generator ran the pinned local text model
under outbound denial and reproduced the checked-in JSON byte-for-byte.
A one-bit canonical encoding mutation failed the same real command with
`semantic tag vector checksum mismatch`. The old binary is removed; JSON is
421,820 bytes and requires no build-time conversion or generator dependency.

`make build` refreshed bin/picfetch successfully. Formatting, TUF/Qodana checks,
vet/build and shard admission passed. `make verify` exited 0; all four Linux
race partitions passed, including TestVisualSimilarityExplorer in 99.040s and
the last UI partition in 399.042s. Raw race artifacts are retained at
`.scratch/race-runs/20260910T074825Z-4hZNNa`.
[Readable-vector evidence](../.scratch/visual-similarity-explorer/evidence/readable-tags-20260910/README.md)
holds reproduction, failures, native success, source hashes and gate logs.
No production/test source changed after this final gate started.

| Task | Spawns budget/actual | Lead review rounds | Full suite | Notes |
| --- | --- | --- | --- | --- |
| Preview resampling | 1 / 1 | 2 | no | Read-only source scout; real allocation red/green; exact pixel and production parity |
| Readable tag vectors | 0 / 0 | 1 | no | Real-command format red/green; exact numeric identity and pinned regeneration |
| Combined final gate | 0 / 0 | 1 | one, passed | Native suites, canonical race partitions and refreshed build all passed |

The wider plan remains active: native/library qualification and saved-preset
trait/application choices remain open. No git commit was made and the two
pre-existing unrelated `:memory:.ses` files remain untouched.

## Native qualification readiness — 2026-09-10

Resumed with `/implement use tdd and sdd`. All preceding implementation slices
are complete at commit `2e38e31`; the remaining product work is saved presets
and native/library qualification. Preset trait scope and future-map application
were requested asynchronously again; elapsed time does not resolve them.
The native trial verdict/current collection was requested alongside them.

Deliverable: current-build native rendering and integration evidence, followed
by the next accepted feature slice when its recorded choices are answered.
Use the existing confirmed viewer/provider and actual-engine boundaries. No
new behavior is specified merely to create another implementation increment.
The existing synthetic native replay avoids private-pixel inspection and does
not qualify real-library semantics or complete ticket 07.

Tasks (T0): (1) inspect current processes and retain a current replay executable;
(2) run the existing 300-source/20-pile native replay and current native viewer
suite; (3) record actual outputs and the next unresolved decision. Verify with
the replay's complete phase JSONL/exit status and `make explorer-ui-test`.
No production edit or new test is planned for already implemented behavior;
any observed regression gets its own red/green slice before a fix.

Budget: one read-only scout, one lead review, no repeat full race suite unless
production/test code changes. Scout G1-G5: bounded existing instrumentation
search; source locations and commands as oracle; zero writes; evidence-script
context independent of the lead's command/preset trace; context not previously
held. S/W: adaptive reconnaissance rather than a deterministic transform.
Review and fixes stay with T0. A newly started PicFetch process was found and
is left untouched; historical monitor PIDs are not reused.

### Qualification readiness evidence

`make explorer-ui-test` passed in 28.303s with both required suites and no
skips. The initial outer-sandbox attempt failed its native `sandbox_apply`
prerequisite; the successful retry preserved the worker's explicit outbound
denial. The retained native replay exited 0 with all 43 expected phases for
300 synthetic images/20 piles. Its construction screenshot was inspected;
no private images were inspected. Median observed pans were about 17 ms;
overview zoom was 192 ms, including capture/readback and polling overhead.
These measurements do not qualify real-load latency or library semantics.

[Evidence and limits](../.scratch/visual-similarity-explorer/evidence/native-qualification-20260910/README.md)
include commands, executable digest, raw logs and aggregate timings. One scout
and one lead review were used. No production/test edit, new TDD cycle or repeat
full race run was needed. The plan remains active: saved presets need the two
recorded user choices, and ticket 07 still needs the native trial/evidence flow,
the selected full collection and the user's native quality verdict. No answer
was inferred from the asynchronous questions. No commit was made.

After Ronin handed over the desktop, the current source was exercised in an
isolated, network-denied native app with public fixture copies. The 60-source
run verified sidebar collapse, exact 20-image tag grids, bounded image navigation,
map return, granularity clicks and keyboard selection/zoom restoration. A
600-source run showed a real partial map and completed 600/600 without failures;
its final Cat grid showed 200 matches. Native drag automation did not move either
slider or map and remains unverified. The grid opened after completion, so native
frozen browsing during updates is still unqualified. The repeated-image fixture
produced 480 Unassigned and provides no semantic quality verdict. The owned app
and worker exited; public screenshots and precise limits are in the evidence.
Only documentation changed. Saved-preset choices and real-library acceptance
remain pending; desktop availability did not answer those product questions.

## Qualification runner continuation — 2026-09-10

Resumed by `/implement use tdd and sdd`. Inspect the missing ticket 07 native
launch/evidence path while awaiting the recorded preset choices and current
collection/verdict. Existing test boundaries remain accepted. All design,
implementation, review and fixes stay with T0.

Recon budget: one read-only scout, zero implementation delegates. G1: locate
existing native launch and measurement hooks; G2: verify returned source
locations and commands; G3: no writes; G4: native measurement script breadth
is independent of the lead's evaluator/launch trace; G5: that script context
has not been read by the lead. S/W: adaptive source search, no mechanical
transform or supplied implementation. The literal T3 model is unavailable;
use the harness's inherited model in a read-only scout role.

Recon findings: `scripts/explorereval/main.go` still rejects `TRIAL=library`.
The production profiler records worker receipts and observed child exit;
`internal/ui/explorer.go` has no native map-delivery or input/paint recorder.
The existing synthetic replay's no-op Host cannot qualify live Grid return
while analysis continues. A complete native runner therefore remains real
implementation work, not another invocation of an existing acceptance command.

Isolation detail for that runner: the earlier overlay changed the Fyne app ID,
which isolates preferences/session only. `favstore.DefaultDir()` and
`autoupdate.DefaultDir()` use independent `picfetch/favorites` and
`picfetch/updates` directories; the runner must explicitly isolate those too.
The lead verified these source definitions and the profiler's final accounting
checks. No native app, library scan, production edit or new test ran in this
reconnaissance. Existing passing test records were not presented as fresh runs.

The preset trait/application questions and current trial collection/verdict
were presented together and remain unanswered. No preset choice or native
acceptance is inferred. Actual cost: one read-only scout, one lead recon check,
zero implementation delegates and zero full-suite repetitions. Documentation
validation: `git diff --check`. Keep this plan active.

## Native trial recording — 2026-09-10

Deliver an opt-in native trial launch with isolated storage and metadata-only
evidence from the production viewer. This implements independently actionable
ticket 07 infrastructure. It does not choose saved-preset behavior or infer
the user's library/semantic verdict. Existing viewer/provider and command
boundaries remain the confirmed test seams. Route: Deep.

Decisions: `--explorer-trial DIR` is an explicit development launch option;
DIR must be new. Trial launches require actual OS network denial before opening
sources, use a unique Fyne app ID, isolate Favorites/update paths, and disable
update checks. Ordinary launches retain their standing behavior. Records omit
paths, pixels, previews and embeddings. Worker receipt, UI application and
worker exit are separate facts; UI application duration is not paint latency.
Human observations and full-library acceptance remain separate from collection.

### Task NT1 — Record a production viewer session
Owner: T0 inline
Files: internal/explorertrial/session.go; internal/ui/{explorer.go,explorer_test.go}
Contract: per-session recorder, numbered analysis runs, serialized metadata
events; Begin/Received/Applied/Exited plus cohort-open/map-return/exit records;
Close waits on no UI work and rejects incomplete/error evidence.
Test: viewer inputs and controlled provider deliveries produce ordered map,
frozen-cohort and exit evidence without retaining source content; cancellation
cannot become successful completion. One red/green behavior at a time.
Verify: `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/^trial_recording' -count=1 -v`
Budget: 0 spawns; 2 lead reviews; full suite at final gate only.

### Task NT2 — Isolate and launch the native trial
Owner: T0 inline
Files: internal/launch/{launch.go,launch_test.go}; main.go;
internal/ui/{run.go,launchoptions.go,explorer_test.go};
scripts/explorereval/{main.go,main_test.go,library.go,evaluate.sh,README.md}; Makefile
Depends: NT1
Contract: `make explorer-evaluate TRIAL=library` launches a retained native
executable under network denial, retains console/exit/session evidence and
reports collection separately from qualification. Cancellation terminates and
joins the owned app/analysis tree. Trial storage never uses standing Favorites
or update directories. Existing evidence is never overwritten.
Test: command rejects absent native executable and invalid/existing output;
viewer trial configuration isolates runtime directories; native fixture launch
records a completed map and observed worker exit. Canceled runs stay incomplete.
Verify: targeted launch/evaluator tests; `make explorer-ui-test`; actual denied
native fixture run with retained logs and process-exit evidence.
Budget: 0 implementation spawns; 2 lead reviews; full suite at final gate only.

### Task NT3 — Verify and document
Owner: T0 inline
Files: ARCHITECTURE.md, todos.md, this plan; shard/Qodana manifests if needed
Depends: NT1, NT2
Test/verify: negative guards, focused native suites, `make verify`, `make build`;
record limits and leave the broader plan open for human qualification/presets.
Budget: 0 spawns; 1 lead review; one full suite.

Graph: NT1 -> NT2 -> NT3. No implementation delegation: contracts and runtime
wiring are cross-package lead work. One independent read-only command-lifecycle
scout is permitted while T0 writes NT1. G1: bounded process ownership question;
G2: source locations/commands; G3: zero writes; G4/G5: shell lifecycle breadth
independent of session-recorder implementation; S/W: adaptive search, no transform.

### User quality verdict

Ronin reports: "I am quite happy with the quality of matching tags to images
and the forming of cohorts also looks good!" This accepts tag matching and
cohort quality in his tested experience. It supersedes the unanswered semantic
verdict for that experience; it does not establish a collection size, measured
latency or the saved-preset trait/application choices.

### Native recorder deferred by user steering

Ronin explicitly prioritized creating cohorts from selected Unassigned images.
The recorder's receipt/application/exit and cancellation tests reached green;
the next cohort-visit record test was red when work switched. In-progress
recorder source and its viewer patch are retained under
`.scratch/visual-similarity-explorer/deferred-native-recorder/` and were removed
from the active application diff. No incomplete recorder is being shipped.

## Create cohorts from Unassigned — 2026-09-10

User contract: an Analyze button in the Unassigned Grid View, enabled only
when more than one image is selected; show similarities between those images
and offer to create a cohort. This is the priority implementation increment.
Use existing local semantic tags as named visual similarities. No new inference
or metadata discovery is needed. The concrete flow creates current-map groups and persists explicit memberships
for favorite-based collections (see UC4). Reusable rules for unrelated future
maps remain the separately documented follow-up.
The existing production UI/provider seam is accepted; no new seam is requested.

### Task UC1 — Contextual selection action
Owner: T0 inline
Files: internal/ui/{explorer.go,explorer_test.go}, internal/ui/grid/{grid.go,subset.go,search.go}, internal/ui/explorer/map.go
Contract: explicit Unassigned activation distinct from normal/tag cohorts;
grid-owned Analyze action enabled for at least two explicit selected sources.
Test: button is in the Unassigned surface, disabled for zero/one, enabled for
two; absent from ordinary and cohort grids; image/map return preserves context.
Verify: `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/^create_cohort' -count=1 -v`
Budget: 0 spawns; 2 lead reviews; no full suite.

### Task UC2 — Review shared traits and create a named cohort
Owner: T0 inline
Files: internal/ui/explorer/cohorts.go, internal/ui/explorer/map.go,
internal/ui/explorercohorts.go, internal/ui/explorer_test.go, translations/{en,de}.json
Depends: UC1
Contract: show common recognized visual tags, chosen trait matches and a name;
creation updates the current map without overlap or new analysis. User-created
membership survives incoming publications; already opened cohorts remain frozen.
No shared traits, invalid name and stale source selections cannot create a cohort.
The review offers an explicit inclusion checkbox, initially on: include other
matching Unassigned sources, or restrict creation to the selected sources.
Test: real viewer selection/dialog/create path, exact resulting membership,
Unassigned reduction, cancellation/stale publication and preservation of existing
cohorts. Use controlled analysis only at the agreed provider boundary.
Verify: focused create-cohort suite plus real `make explorer-ui-test`.
Budget: 0 spawns; 2 lead reviews; full suite at final gate only.

### Task UC3 — Handoff and verification
Owner: T0 inline
Files: ARCHITECTURE.md, todos.md, this plan, manuals, catalogue/shard/Qodana guards
Depends: UC2
Verify: `make verify`, `make build`, native UI review of public fixtures.
Budget: 0 spawns; 1 lead review; one canonical full suite.
Graph: UC1 -> UC2 -> UC3. Cross-feature composition, user strings and all review
stay with the lead; no implementation delegation.

### Favorite persistence amendment

Ronin requires user-defined cohorts to be saved for favorite-based collections.
Persist each favorite's named groups and exact reviewed source membership beside
its file list. Restore them on reopening that favorite, including a fresh viewer;
manual membership takes precedence over fresh automatic grouping. This is saved
collection state, independent of the optional analysis cache. Ordinary file sets
and mixed merge-mode collections remain temporary; favorites with identical
files keep separate groups. Replacing/removing a favorite invalidates pending
writes. Surviving members only appear in the current analyzed source set.

UC4 (T0): favorite-owned atomic cohort storage, explicit favorite open identity,
tracked load/save on existing Explorer worker/queue, persistence failure feedback.
Red/green command: `go test ./internal/ui -run '^TestVisualSimilarityExplorer$/^create_cohort_favorite' -count=1 -v`.
Storage guard command: `go test ./internal/favstore -run '^TestCohorts' -count=1 -v`.
No new worker pool or test seam. Final UC3 gate follows UC4. One additional
read-only scout checks Host/reset call-site breadth while the lead owns design
and implementation; zero implementation delegates, two lead review rounds.

### UC implementation and lead review

UC1/UC2 followed red-green slices: missing contextual Analyze, absent trait
review, missing inclusion choice, map-filter reveal, stale dialog and empty
state all failed for their specified UI behavior before their fixes. The
real-model native suite created a cohort from shared Cat tags without new
inference. Native screenshots from public Chelsea fixtures show the dialog,
“Public cats (2)” on the map and its exact two-image grid; the owned app exited.

UC4's first production UI test failed with “reopening the favorite lost its
named cohort.” It now reopens a new viewer, restores the exact named members
over fresh automatic grouping with analysis caching disabled, and keeps a
second identical favorite independent. A removed favorite produces a visible
save error and rolls back the proposed group. Favorite identity enters through
OpenFavorite and the common drop pipeline; mixed outside sources do not gain
favorite persistence. Storage binds writes to the observed file-list version,
checks cancellation before atomic replacement, and never recreates a removed
favorite. Existing Explorer workers and UI queue own all load/save completion.

The read-only favorite-contract scout reported call-site/reset breadth; the
lead verified these locations before changing the Host and common open path.
No implementation or review was delegated. New tests are subtests of the
existing Explorer suite or additions to the existing favstore test file, so
there is no new Qodana test-file exclusion or root UI shard assignment.
The final verification below covers the favorite persistence amendment; earlier
passing runs were not used as its final gate.

Native favorite amendment check: the final isolated build saved a public two-cat
favorite through its menu, created “Saved cats (2)”, fully exited and restarted,
and restored that exact named cohort and two-member grid after reopening the
favorite. Screenshots are retained in the [cohort creation evidence](../.scratch/visual-similarity-explorer/evidence/create-cohorts-20260910/README.md).
The native app was quit and its exact executable path had no remaining process.

### UC final verification and ledger

All final commands exited zero on September 10:

- `make explorer-ui-test`: `ok github.com/frathe/picfetch/internal/ui 26.249s`;
  real offline model/worker tests and the production UI suite.
- `make verify`: format, offline TUF root, exact Qodana exclusions, vet and build;
  canonical Linux/amd64 Docker race partitions all passed. UI partitions took
  428.401s, 298.714s and 290.968s. `TestVisualSimilarityExplorer` passed under
  race detection in 119.890s. Race artifacts: `.scratch/race-runs/20260910T092637Z-UWjA4H`.
- `make build`: produced `bin/picfetch` with the Makefile release flags.
- Native visual creation plus favorite reopen after a complete process restart:
  named groups and exact grid membership observed in the final isolated build.
- `git diff --check`: clean. No golden images were regenerated.

Six deliberate negative mutations produced the expected behavioral failures
and were restored: selection threshold, existing-cohort protection, incoming
publication retention, favorite restoration, favorite replacement and canceled
writes. Error-path UI coverage also confirms failed saves leave no proposed
cohort and present a visible explanation. The final gate ran once after all
implementation changes; no verification failures required another full run.

| Task | Spawns budget/actual | Lead review rounds | Full suite | Notes |
|---|---|---|---|---|
| UC1 | 0 / 0 | 1 | no | Contextual selection path |
| UC2 | 0 / 0 | 3 | no | Third review justified by native empty-state/filename observations |
| UC4 | 1 scout / 1 scout | 2 | no | Explicit favorite identity, atomic storage, failure rollback |
| UC3 | 0 / 0 | 1 | once | Final offline/native/CI gates and documentation |

The requested creation/persistence increment is complete. Keep the broader
plan active: reusable rules/preset browser and full-library qualification still
have outstanding scope. No git commit was made. Suggested commit message:
`Add cohort creation and favorite persistence`.
