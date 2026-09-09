# Visual similarity explorer implementation

Date: 2026-09-09
Status: native completed-map increment verified and open for local trial; later increments remain active
Route: Deep — new local analysis subsystem and cross-feature UI behavior
Request: `/implement use tdd and sdd`
Spec: [Visual similarity explorer](../.scratch/visual-similarity-explorer/spec.md)
Tickets: [Execution sequence](../.scratch/visual-similarity-explorer/ticket-breakdown.md)

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
