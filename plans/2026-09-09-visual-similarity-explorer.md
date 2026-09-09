# Visual similarity explorer implementation

Date: 2026-09-09
Status: preparation active; product answers and trial inputs pending
Route: Deep — new local analysis subsystem and cross-feature UI behavior
Request: `/implement use tdd and sdd`
Spec: [Visual similarity explorer](../.scratch/visual-similarity-explorer/spec.md)
Tickets: [Execution sequence](../.scratch/visual-similarity-explorer/ticket-breakdown.md)

## Deliverable and current boundary

Deliver the real local content-similarity explorer through the published
seven-ticket sequence, including native trial evidence on the intended Mac.
The implementation request authorizes beginning the sequence and supersedes
the prior hold on publishing local tickets. Q1-Q6 remain settled. Q7-Q12,
the trial inputs and confirmation of proposed test boundaries are pending;
no answer is inferred from a preselected option or elapsed time.

The current task is ticket 01. Ticket 02 must establish a measured real
pipeline before production interfaces and the file map for ticket 03 are
fixed. A deterministic provider fixture cannot substitute for that gate.
Geolocation, text-directed grouping, expanding piles at high zoom and broader
platform packaging remain outside this first trial.

## Proposed test boundaries

1. Viewer interactions through the production-aligned UI harness, ordinary
   input events, displayed file identities and visible surfaces. Controlled
   results enter only through an instance-owned analysis provider boundary.
2. The actual local analysis engine's input/output boundary, with real image
   representations, grouping and projection under effective network denial.

These are the spec's proposed boundaries. Confirmation was requested because
the TDD skill requires agreement before writing tests. No tests have been
written. Once confirmed, implement one behavioral test and minimal passing
slice at a time, observing the intended red failure before green.

## Task 01 — Resolve and record the trial contract

Owner: T0 inline; one read-only scout for local runtime inventory
Files: published issues, ticket-breakdown.md, this plan, todos.md; draft
decisions.md and evaluation-protocol.md, finalize after the relevant answers
Depends: Q7-Q12 product answers, test boundaries and trial-library location
Contract: preserve Q1-Q6; record product provenance separately from lead-owned
technical decisions; no production interface is fixed during preparation
Test: twelve decision rows have supporting provenance; the protocol identifies
inputs, actual offline enforcement, observations and the user's quality verdict
Verify: ticket 01's documented acceptance commands and `git diff --check`
Budget: at most 1 scout; at most 2 lead review rounds; full suite: no

Technical decisions D1-D6 belong to the lead where they follow accepted
product choices. Do not turn them into another blanket approval ceremony.
Ask only for missing user preferences or trial input that affects the result.

## Task graph and later planning

`01 -> 02 -> 03 -> {04, 05 if retained, 06} -> 07`

Ticket 02 owns the real engine experiment. Its exact files, commands and
input/output contract will be recorded after ticket 01 resolves. Ticket 03
owns the completed small-map/Grid View/return path. Tickets 04-06 extend that
path; their potential parallelism depends on disjoint files and fixed
interfaces. Ticket 07 owns the full-library trial and final `make verify`.
Do not invent production contracts before the feasibility evidence exists.

## Delegation gate

One independent read-only scout inventories existing runtimes, candidate
model caches and offline-denial tools while the lead prepares local ticket
records. It does not choose a pipeline or process images.

| Gate | Evidence |
| --- | --- |
| G1 | Bounded capability inventory described in fewer than 25 lines. |
| G2 | Each capability must include the command or file location proving it. |
| G3 | Read-only; zero files modified, no shared mutation. |
| G4 | Environment discovery is independent of product/spec preparation. |
| G5 | The lead has not inspected the runtime or model caches. |
| S/W | Discovery needs adaptive command selection; no implementation is supplied. |

The higher-priority session delegation instruction permits independent
read-only preparation while product questions are pending. No ambiguous
implementation task or review is delegated. The harness exposes no literal
Explore agent; the scout uses the inherited model in the T3 read-only role.

## Evidence and cost ledger

Initial working tree was clean. Latest commits contain the explorer spec and
draft tickets; it is the sole active feature under `todos.md`'s TODO section.
The source specification is preserved during ticket publication.

Lead-verified local prerequisites:

- PATH Python is 3.14.7; `importlib.util.find_spec` finds no `torch`,
  `transformers`, `hdbscan` or `umap`. `uv` is absent from PATH and the repo
  has no `.venv/bin/python`. No SigLIP-named directory exists in the default
  Hugging Face hub cache. This bounded inventory does not exclude custom
  environments elsewhere; no packages or models were installed.
- The nested tool sandbox initially rejected `sandbox-exec`. Running the
  harmless probe outside that sandbox succeeded. With the profile
  `(version 1) (allow default) (deny network*)`, `/usr/bin/python3` TCP connect
  and empty UDP send attempts to documentation address `192.0.2.1:443` both
  failed with OS `EPERM` (1). No images or user data were involved. This
  establishes availability of the denial mechanism, not offline acceptance
  of an analysis runtime that has not been selected or run.
- `.scratch/` is intentionally ignored by Git; tickets remain local tracker
  artifacts. The active plan and `todos.md` are reviewable repository changes.
- Publication checks pass: seven numbered tickets retain `needs-info`, all
  33 local links resolve, and whitespace checks plus `git diff --check` pass.
  The link check exposed a pre-existing moved mosaic-plan link in `todos.md`;
  it now points to the existing `finished_refactorings` record. No application
  tests were run because no application code or tests changed.

| Task | Spawns budget/actual | Review rounds | Full suite | Evidence |
| --- | --- | --- | --- | --- |
| Preparation | 1 / 1 | 2 | no | Seven local tickets published; lead-verified runtime and denial probes |
| 01 | 0 / 0 additional | pending | no | Questions pending; no acceptance claimed |
| Final gate | 0 / 0 | pending | pending | No application implementation yet |

Keep this plan active until accepted completion. No git commit is authorized.

## Resumed preparation

The repeated implementation request resumes ticket 01. The six recorded
recommendations, proposed test boundaries and trial-library path have been
presented together. Their answers remain pending until actually received.

Prepare reviewable `decisions.md` and `evaluation-protocol.md` drafts without
claiming acceptance or processing images. One additional read-only scout may
trace existing grid subset/return behavior while the lead writes the protocol.
This is new integration reconnaissance, not a repeat of the runtime inventory.

Delegation gate: G1 is a bounded grid-flow question; G2 is verification of
returned symbols with `rg -n` and targeted reads; G3 is zero writes; G4 is an
independent code sweep while the lead prepares evaluation requirements; G5
holds because the lead has not traced these flows. S/W: cross-file control
flow needs comprehension, and no implementation is specified. Budget: one
additional scout, one lead review, no full suite for documentary preparation.
Production contracts remain ticket 02's output after real evaluation.

Drafts now exist and distinguish pending product answers from lead-owned
technical proposals. The protocol defines a proposed bounded 512-image smoke
run, effective offline enforcement, real output checks, local semantic
inspection, process-scope memory measurements and the full-library trial.
Input paths and runtime assets remain unresolved; no evaluation command is
claimed to exist or pass.

Lead-verified reconnaissance: `grid/search.go:119` has only search/duplicate
filters; `grid/selection.go:35` returns results but cannot supply a subset;
`grid/grid.go:404` resolves an activated index before closing, and `:621`
clears transient grid state; `grid/nav.go:176` defines Escape precedence.
Comparison keeps the grid open (`internal/ui/compare.go:35`). These are
integration constraints for ticket 03, not new production contracts.

| Task | Spawns budget/actual | Review rounds | Full suite | Evidence |
| --- | --- | --- | --- | --- |
| Resumed preparation | 1 / 1 additional | 1 | no | Draft contract/protocol; grid-flow facts verified by lead reads |

No tests were written, dependencies installed or library files processed.
The implementation remains at ticket 01 pending the requested answers.
Document verification passed: 13 documents, 48 local links, no trailing
whitespace, exactly twelve pending/draft decision rows and seven `needs-info`
tickets. `git diff --check` passed. The tracked diff contains this plan and
`todos.md`; the new protocol/decision records remain in the ignored local
tracker. Application verification is deferred until application changes exist.

## Runtime question: Go feasibility

The user asked why Python was proposed and whether Go can implement the
feature. Investigate this as Q8 clarification; the question itself does not
accept the earlier Python recommendation or resolve the other trial inputs.
The [candidate notes](../docs/2026-09-09-visual-similarity-explorer-candidates.md)
record a Go/native-inference route and the remaining measurement requirements.
The lead now recommends evaluating that route before a Python helper.

One bounded research scout checks Go clustering/projection libraries against
primary sources while the lead checks model inference. Its sole writable
file is `.scratch/visual-similarity-explorer/go-clustering-research.md`.
G1: one bounded question; G2: source-linked findings verified by lead reads;
G3: one disjoint note; G4: independent algorithm research; G5: no prior lead
context on those Go libraries. S/W: adaptive source investigation, no supplied
implementation. No spec decision or review is delegated. Budget: one scout,
one lead review, documentation checks only; no installs or image processing.

Actual: one scout, one lead review. The lead verified the Go ONNX binding,
published vision exports, CoreML requirements, and concrete limitations in
the inspected Go HDBSCAN/UMAP documentation. No application tests or trial
benchmarks ran. This answers feasibility at source level and leaves measured
pipeline acceptance outstanding.
