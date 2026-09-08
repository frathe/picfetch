# Release review follow-ups on d7d2f8e

Standard route: fix the two new PR #17 P2 findings and update the existing
release assessment. The lead owns diagnosis, implementation, review and the
final gate. No commits, pushes, reviewer comments or release/Store operations.

## Acceptance criteria

1. Exporting to an unrelated destination preserves favorite-preview admission,
   grid thumbnails and duplicate facts. Writes aliasing any loaded source still
   invalidate derived state, including when the loaded set changes before UI
   delivery. Verify: `go test ./internal/ui -run 'TestExport|TestFileMutation'`.
2. Changing sensitivity after closing an unfinished Hide Duplicates pass hashes
   missing files and updates navigation without reopening the grid. Pending
   browsing remains active until facts finish; Stop stays terminal. Verify:
   `go test ./internal/ui/grid -run 'TestDuplicateDistance|TestGridWork_Stop|TestSetDuplicateDistance'`.
3. Integrated formatting, guards, vet, build and Linux race suite pass:
   `make verify`. Remote checks are attributed only to their actual SHA.

## Tasks

T1 (lead): filework.go, exportwork_test.go and UI shard manifest. Pin unrelated
exports and loaded aliases with regressions, then gate derived invalidation on
an off-UI identity check of the current immutable file set. No new public API.
Verify AC1. Budget: no implementation spawns, two review rounds, no full suite.

T2 (lead, independent of T1): grid/dupes.go and grid/thumbs_test.go. Pin cancelled
hash resumption, pending browse and terminal Stop; resume missing facts when an
active duplicate mode needs them. Verify AC2. Same budget as T1.

T3 (lead, after T1/T2): update todos.md and release-readiness/assessment.md;
verify AC3, review the integrated diff and archive this plan. One full gate.

One reused read-only Scout locates grid cancellation/harness examples while the
lead diagnoses export. G1: bounded question; G2: cited methods checked against
source; G3: no writes; G4: harness search spread across tests; G5: lead holds
export context, not that test harness. No implementation or review delegated.

## Evidence and ledger

Both reported regressions failed before fixes. Controlled negative checks also
failed for noncurrent aliases, both loaded-set changes and the existing
unrelated-commit reconciliation retry. Native focused race suites passed (root
UI 104.123s, grid 2.345s; adjusted reconciliation guard 3.994s). The first
full gate recorded a Docker OOM: comparison printed PASS, then was killed.
The original raw streams and Docker event are retained. A second complete
gate was justified by this resource failure. It passed with `GOFLAGS=-p=1`
injected into the unchanged Ubuntu 24.04 image: package parallelism was limited
to one while all four race partitions ran. All 53 non-UI packages passed;
ui-1/ui-2/ui-3 passed in 307.744/310.008/296.533s. Exact UI inventory: 678.
Both attempts retain complete raw JSON under `.scratch/release-review-followups/`
(original files and `retry/`). The assessment records the remote d7d2f8e checks
and both local gate outcomes. No code changed after the successful gate.

Honest limits: native Windows packaged
startup/WACK and live Store permissions retain their existing deferred status;
remote checks on d7d2f8e do not validate these new local edits.

| Task | Spawns budget / actual | Reviews | Full suite |
| --- | --- | --- | --- |
| Recon | 1 / 1 reused Scout | source verification | no |
| T1 | 0 / 0 | 2 | no |
| T2 | 0 / 0 | 1 | no |
| T3 | 0 / 0 | 1 | Budget 1 / actual 2: first OOM, bounded retry passed |

Completed locally without commit or push. Suggested commit: `fix(ui): preserve export previews and resume duplicate hashing`.
