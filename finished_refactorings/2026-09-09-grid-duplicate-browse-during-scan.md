# Grid duplicate browsing during analysis

Status: complete; all acceptance criteria and canonical verification pass
Route: Standard
Spec: [.scratch/grid-duplicate-browse-during-scan/spec.md](../.scratch/grid-duplicate-browse-during-scan/spec.md)
Ticket: [01](../.scratch/grid-duplicate-browse-during-scan/issues/01-browse-known-groups-during-analysis.md)

Deliverable: Shift+D presents the highlighted source's accepted duplicate group
immediately during analysis and follows that source's accepted updates.
Matching, representatives, unknown-group requests, and platform behavior retain
their existing contracts. The unknown-group pending presentation remains an
explicit limitation. The spec already approves the viewer keyboard/result seam;
no additional seam approval is needed.

## Decisions

| Decision | Contract |
| --- | --- |
| Entry | Accepted membership, not global hash completion, permits known-group browsing. |
| Updates | Accepted groups update the original URI identity; pending delivery preserves the established view. |
| Unknown source | Preserve the existing wait through analysis completion. |
| Ownership | Existing grid lifecycle, duplicate model, workers, and UIQueue; no new production interface. |
| Tests | One root `TestGridBrowseDuringAnalysis` with the five specified behavioral subtests, controlled ReaderURI reads and queued delivery. |
| Review | Lead owns design-bearing tests, implementation, review, fixes, and final gate. |

## Tasks

### Task 1 — Reproduce and correct partial browsing
Owner: T0 inline
Files: modify `internal/ui/grid/{dupes.go,search.go,groupwork.go}` and private
state in `grid.go` only if required; extend `internal/ui/grid_test.go`; update
`.github/testshards/internal-ui.tsv` with the new runnable immediately.
Depends: none
Contract: existing `SetBrowsingDuplicates(bool)`, `BrowseReady() bool`,
`ResultIndexes() []int`, and `FilesChanged()` retain their interfaces. Source
identity, accepted membership, and cancellation govern the browse lifecycle.
Test: AC1–AC6 at the approved viewer seam, in vertical red/green slices;
negative verification of each regression guard, plus existing compatibility.
Verify: `go test ./internal/ui -run '^TestGridBrowseDuringAnalysis$' -count=1 -v`
and every AC1–AC6 command in the spec.
Budget: <= 1 read-only scout, 0 implementation spawns; <= 2 lead review rounds;
full suite: no.

### Task 2 — Documentation and handoff
Owner: T0 inline
Files: modify `internal/ui/help/{manual.md,manual_de.md}`, `todos.md`, this
plan, and local ticket/spec status and evidence.
Depends: Task 1
Contract: manuals describe known groups opening during continuing analysis,
while unknown groups retain the existing pending behavior; no new UI strings.
Test: AC7 manual rendering guards and semantic diff; AC8 shard and final gate.
Verify: AC7 commands, `make check-test-shards`, `make verify` once at handoff.
Budget: 0 spawns; 1 lead review round; full suite: yes, final gate only.

Graph: Task 1 -> Task 2. Read-only fixture reconnaissance runs independently
of the lead's production lifecycle analysis within Task 1.

## Delegation gate

One bounded fixture scout follows shell searches across the existing harness,
synctest, partial-hide, and variant tests. G1: a short read-only question;
G2: returned file:line findings checked with targeted reads; G3: zero edits;
G4: fixture breadth is separate from production context; G5: no fixture trace
has yet been built locally. S: tracing scheduling/lifecycle is not a text
transform. W: no implementation is supplied or delegated. This expands the
ticket's optional reconnaissance budget by one scout, with zero implementation
delegation. Scout output is evidence to inspect, never delegated review.

## Evidence and cost ledger

The original regression failed with all eight URI identities visible instead
of the highlighted source's pair, with three reads still held, both with hide
off and on (`/tmp/grid-browse-red.log`). The next vertical slice failed because
a completed matching source did not join the pair
(`/tmp/grid-browse-updates-red.log`). Both now pass.

Accepted member URI keys live in private `Overview.browseGroup`; a nil group
keeps the existing unknown-source request path. The existing source setter
captures initial membership, accepted grouping deliveries update it, and the
existing source clear removes it. Pending updates filter by retained identities,
including during file reordering. Existing workers and public interfaces remain.

Every new behavioral guard has failed on its intended deliberate violation:
partial hash callbacks suppressed, cancellation disabled, source remapping
omitted, partial dissolution suppressed, and inspect admission omitted. Each
mutation was restored in a `finally` block. Logs are
`/tmp/grid-browse-negative-{hash_progress,cancel,identity,dissolution,inspect}.log`.
The inspect guard was rerun after the fixture synchronization adjustment.

Review caught an existing lower-level fixture using the private source setter;
capturing initial membership there preserves that invariant without altering
the fixture. Full `go test ./internal/ui/grid -count=1` now passes. A focused
race run caught the Fyne test driver's inline load completion overlapping
GridWrap's deferred unselect. The viewer test now visits the source and returns
to the first image, so its normal neighbor preload supplies the known copy.
The full `TestGridBrowseDuringAnalysis` then passes under `-race` (15.500s).
These were fixture/lifecycle verification findings, not broader production
changes. Both initial failure logs remain in `/tmp`.

Both manual diffs agree on immediate known-group entry, continuing updates,
and the unchanged unknown-group wait. Embedding/table/Unicode-arrow guards pass.
`make check-test-shards` passes: 679 runnables, three shards, with the new root
test assigned to ui-1. No new test file or Qodana exclusion is needed.

Final AC command output: `/tmp/grid-browse-final-ac.log`.
Canonical `make verify` completed successfully in one run, using the default
four concurrent Linux/amd64 race partitions. Formatting, TUF/Qodana checks, vet,
build and all race packages pass. The regression itself passed in 18.980s on
Linux. The attempted capture-path override was not forwarded into Docker's
inner Make invocation: its four raw JSON streams used container-local `/tmp`
and were removed with the container. The full console log is retained and
records all test/package outcomes; raw JSON is unavailable for this run.
No golden was regenerated, and the gate was not repeated solely for logging.

The [verification record](../.scratch/grid-duplicate-browse-during-scan/evidence/verification.md)
links the retained red/green, mutation, compatibility, shard and canonical logs.
`todos.md` and the implementation ticket are complete. No commit was made.

| Task | Spawns budget/actual | Review rounds | Full suite | Notes |
| --- | --- | --- | --- | --- |
| 1 | 1 / 1 | 2 | no | Read-only fixture reconnaissance; all fixes inline. |
| 2 | 0 / 0 | 1 | yes, once | Manual review and canonical gate pass. |
