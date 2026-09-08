# Progressive duplicate hiding follow-up

User report, 2026-09-07: detection works, but the view updates only after the
entire list is analyzed; it previously updated successively.

The [regression](24-progressive-hide-red.txt) completes two matching sources in
an eight-file list while the remaining six stay blocked in controlled reads.
Draining ready UI work leaves eight visible files instead of seven. The test
uses the real hash/grouping path, a drainable UI queue and synctest barriers;
it does not wait for the full list or sleep to guess completion.

Cause: ticket 24 moved grouping off UI onto the shared decode pool. A cold
hash pass occupies every slot and queues the rest of the source list before
its grouping job. That makes the progress needed to update the view wait for
the work it is supposed to summarize.

Fix: `groupWork` owns one independent, tracked worker. Existing pending-request
coalescing, cancellation, strict snapshot freshness and UI publication remain.
`Overview.Settle` waits both worker owners, drains their completions and repeats.
Close/Stop still cancel the common session; no obsolete grouping can publish.
The full root harness already drains through Grid.Settle, so it covers the new
worker. The architecture and agent guide now record this invariant.

[Three race-instrumented repetitions](24-progressive-hide-grouping.txt) pass
(1.559s), including stale snapshots, changed requests and close/stop cancellation.
The [complete grid race suite](24-progressive-hide-grid.txt) passes (2.811s).
A dedicated [completion guard](24-progressive-hide-barrier.txt) also passes:
Settle cannot return while the grouping worker or its delivery is outstanding.
[Negative verification](phase6-negative.txt) rejects both sharing the decode
queue again and removing grouping from the completion barrier.

The new regression observes partial progress while reads remain blocked; it
catches the gap that older tests missed by settling the entire pass first.
Common-gate acceptance is recorded in the Phase 6 plan ledger.

Final common gate: [make verify PASS](phase6-verify.txt), including all 667 root UI tests, canonical Linux/amd64 goldens and the full race suite. The native-only Copy Selection golden mismatch remains documented in [the macOS package output](phase6-native-packages.txt).
