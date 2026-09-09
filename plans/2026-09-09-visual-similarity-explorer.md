# Visual similarity explorer implementation

Date: 2026-09-09
Status: real smoke experiment implemented; semantic review pending; repository verification passed
Route: Deep — new local analysis subsystem and cross-feature UI behavior
Request: `/implement use tdd and sdd`
Spec: [Visual similarity explorer](../.scratch/visual-similarity-explorer/spec.md)
Tickets: [Execution sequence](../.scratch/visual-similarity-explorer/ticket-breakdown.md)

## Deliverable and accepted contract

Deliver the real local content-similarity explorer through the seven-ticket
sequence. Ticket 01 is complete. Ticket 02 now has a measured real pipeline and
local cohort report. Its semantic quality verdict is pending; production
integration is not complete or marked ready merely because inference ran.

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
Status: technical experiment complete; semantic acceptance pending; final gate passed

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
claimed. The user's semantic question remains pending after verification.

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
The required remaining input is the user's semantic verdict on the concrete
cohort report. No application explorer completion is claimed.
