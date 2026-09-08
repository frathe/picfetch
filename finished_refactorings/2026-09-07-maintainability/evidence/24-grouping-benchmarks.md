# Ticket 24 grouping measurements

2026-09-07; Apple M5 Max, 18 logical CPUs, 48 GiB RAM. Go 1.27.1, darwin/arm64, native process, no race instrumentation. Default grouping distance 6. The grid race suite overlapped the beginning of the run for about two seconds; the complete benchmark command took 44.990 seconds. These are local measurements, not CI latency guarantees.

Command: `go test ./internal/dupes -run '^$' -bench '^BenchmarkGrouping$' -benchmem -count=3` — PASS. [Complete output](24-grouping-benchmarks.txt).

The fixture publishes an immutable file snapshot and fills hash/native facts before timing. Unrelated inputs use xorshift 13/7/17, seed 1. Dense inputs XOR one of 64 bits into nonzero base `0xf0f0f0f0f0f0f0f0`, so all pair distances are at most 2. Native sizes cycle from 1x1 through 64x64; the dense representative is index 63. Cold times include input snapshot arrays, grouping and representative selection. Reuse times measure an already accepted unchanged snapshot.

| Distribution | Files | Cold latency, three-run range | Bytes/op | Allocations/op |
|---|---:|---:|---:|---:|
| Unrelated | 10,000 | 2.302–2.348 ms | 4,118,322–4,118,335 | 73 |
| Unrelated | 50,000 | 23.567–24.542 ms | 11,697,716–11,698,300 | 267 |
| Unrelated | 200,000 | 310.150–319.935 ms | 40,565,680 | 1,037 |
| Dense | 10,000 | 0.270–0.273 ms | 822,232–822,233 | 39 |
| Dense | 50,000 | 1.417–1.443 ms | 4,064,217–4,064,224 | 45 |
| Dense | 200,000 | 6.896–6.986 ms | 16,647,128–16,647,163 | 51 |

Every reuse case was 0 B/op and 0 allocations/op, 7.962–8.238 ns/op. Reuse does not scan files or rebuild grouping.

Index selection followed the reuse change and a bounded single-iteration baseline: at 50k files the original cold grouping took 1.152 seconds for unrelated inputs and 0.602 seconds for dense inputs. The indexed single-iteration repeat took 29.735 ms and 1.567 ms respectively. No quadratic 200k baseline was run. Candidate storage is computation-local and O(n); the extra storage buys lower cold latency. Distances outside the projection index's 1–7 range retain the exact cancellable scan fallback. Randomized/adversarial oracle tests cover membership, complete linkage, first-seen assignment and representative ties separately from these benchmarks.
