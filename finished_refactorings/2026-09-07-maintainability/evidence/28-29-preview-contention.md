# Preview contention and bounded admission

Tickets 28/29, 2026-09-07. Baseline: 524fcf8 with the new benchmark and the
original four-worker Sync policy. After: identical benchmark with a per-pass
limit of `min(4, max(1, runtime.GOMAXPROCS(0)-1))`. Foreground ownership stays
independent. The limit reduces competing work; it is not an OS CPU reservation.
On one processor, prewarm still progresses with one worker.

Environment: Apple M5 Max, 18 CPUs, 48 GiB RAM, macOS 26.6.2 (25G83), Go 1.27.1
darwin/arm64; race instrumentation off for measurements. Each leaf explicitly
sets/restores GOMAXPROCS to 2 or 18 and reports the actual count. Go resets this
setting between sub-benchmarks, so an initial parent-only setup was rejected
before the policy decision; its mislabeled P2 results are not used here.

`BenchmarkPreviewForegroundContention` creates 64 preview sources plus eight
foreground sources: distinct temporary paths containing the same deterministic
3072x2048 JPEG (quality 85, gradient plus seeded LCG texture). Exact generation
is in `internal/favthumbs/contention_test.go`. Cold means no application disk
previews and an empty 1 MiB memory sink; it does not claim a purged OS page cache.
Disk-warm primes the disk previews outside timing, with a new empty memory sink
per iteration. Foreground always reads/decodes eight uncached thumbnails through
`imaging.LoadThumbnailContext`, the same decode boundary as visible grid work.
This measures read/decode contention, not a GPU frame or full GUI input latency.

The real Sync walk starts before foreground work. Its first sink lookup supplies
an admission barrier; subsequent sources use the normal worker scheduling.
All eight foreground requests overlap the cold pass. Only the first overlaps
the short disk-warm pass. Each cold pass reads exactly 64 sources; warm passes
read zero sources. Every one of the 64 current disk previews is decoded again
outside timing to verify convergence. The memory sink stops offering at its
budget, and the cache is checked not to exceed it. Fixture setup, warmup and
verification are excluded from timings and allocations. B/op and allocs/op
cover the entire combined iteration, not one thumbnail or peak live memory.

Commands (three independent samples per case):

```sh
go test ./internal/favthumbs -run '^$' -bench '^BenchmarkPreviewForegroundContention$' -benchmem -count=3 -v
```

[Baseline raw output](28-preview-before.txt): PASS, 72.529s.
[After raw output](29-preview-after.txt): PASS, 97.615s.
Medians across the three samples:

| CPUs / policy / cache | Foreground p50 ms | Foreground p95 ms | Foreground/s | Complete previews ms | Previews/s | B/op | allocs/op |
|---|---:|---:|---:|---:|---:|---:|---:|
| 2 / baseline / alone | 55.30 | 55.81 | 18.10 | — | — | 114,255,957 | 385 |
| 2 / baseline / cold | 152.50 | 204.50 | 6.64 | 2,176 | 29.41 | 1,029,072,000 | 9,275 |
| 2 / baseline / disk-warm | 56.71 | 60.78 | 17.57 | 14.33 | 4,414 | 118,810,461 | 4,391 |
| 2 / bounded / alone | 58.11 | 60.18 | 17.18 | — | — | 114,255,957 | 385 |
| 2 / bounded / cold | 61.60 | 63.16 | 16.27 | 4,084 | 15.67 | 1,029,086,024 | 9,418 |
| 2 / bounded / disk-warm | 57.63 | 58.38 | 17.35 | 16.67 | 3,838 | 118,810,157 | 4,392 |
| 18 / baseline / alone | 57.15 | 59.70 | 17.43 | — | — | 114,255,994 | 385 |
| 18 / baseline / cold | 68.12 | 70.60 | 14.94 | 1,148 | 55.72 | 1,029,145,088 | 9,507 |
| 18 / baseline / disk-warm | 58.67 | 61.94 | 17.03 | 5.33 | 11,897 | 118,814,592 | 4,422 |
| 18 / bounded / alone | 58.69 | 61.62 | 17.06 | — | — | 114,256,000 | 385 |
| 18 / bounded / cold | 67.03 | 70.26 | 14.87 | 1,163 | 55.00 | 1,029,149,336 | 9,496 |
| 18 / bounded / disk-warm | 59.86 | 60.96 | 16.71 | 5.00 | 12,009 | 118,813,962 | 4,420 |

Decision: activate and implement ticket 29. The corrected preliminary measurement
(P2 p95 175.5 ms versus 56.91 ms alone) established the target before the change:
P2 cold p95 <= 1.5x its paired alone case, foreground throughput >= 80% of alone,
and preview convergence <= 4,440 ms (twice the preliminary 2,220 ms baseline).
The final measurements meet all three: 1.05x, 94.7%, and 4,084 ms. The tradeoff is
roughly halving cold preview throughput on P2. P18 keeps four workers and its
results are essentially unchanged. These are bounded synthetic-workload results,
not promises for every codec, storage device or physical two-core machine.

Cancellation is a separate contract. An already-running decoder/sampler or
blocking storage Read must return before its next context check; reducing the
worker count does not make those calls interruptible. The existing held-reader
regression rejects obsolete subsequent reads, publishes no cancelled preview,
leaves unvisited previews unswept and verifies the next complete pass converges.
The imaging cancelled-decoder guard verifies discarded pixels after a
non-interruptible decode. No cancellation timing is inferred from this benchmark.

`TestSyncLeavesForegroundCapacityAndConverges` was [red](29-capacity-red.txt):
P1/P2/P3 admitted 4 readers instead of 1/1/2. The [full favthumbs race suite](29-capacity-green.txt)
passes (1.888s), including actual admission on P1/P2/P3/P8, independent foreground
decode while previews are blocked, joined cancellation and idle disk convergence.
The [negative overlay run](phase6-negative.txt) rejects restoring four workers,
running no preview work and omitting preview writes. Inventory is retained in
[phase6-inventory.txt](phase6-inventory.txt).

Affected-package and common-gate results are recorded in the Phase 6 plan ledger. Windows native validation remains the user's deferred checklist.

Final common gate: [make verify PASS](phase6-verify.txt), including all 667 root UI tests, canonical Linux/amd64 goldens and the full race suite. The native-only Copy Selection golden mismatch remains documented in [the macOS package output](phase6-native-packages.txt).
