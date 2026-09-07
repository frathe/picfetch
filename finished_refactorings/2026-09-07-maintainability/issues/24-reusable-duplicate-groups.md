# 24: Reuse cancellable duplicate-group snapshots

**What to build:** Searching a warm Grid result reuses unchanged duplicate groups, while changed grouping computes away from UI and cannot install obsolete results.

**Blocked by:** [18: Admit thumbnail facts only to their source generation](18-generation-safe-facts.md)

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC08 / MA-008; plan work package O.
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Key accepted groups by file generation, hash/native-fact revision and distance; plain search edits filter the accepted snapshot without recomputing grouping.
- [x] Run changed computations off UI with cancellation/coalescing and current-snapshot installation; cover source replacement, fact changes and distance changes.
- [x] Compare membership and representatives to the current greedy complete-linkage algorithm on bounded randomized and adversarial cases, including non-transitive chains and pixel-count/index ties.
- [x] Add the specified BenchmarkGrouping cases for unrelated and dense 10k/50k/200k inputs; record hardware, race status, inputs, latency and allocations.
- [x] First eliminate recomputation; choose an index only if measurements require it and retain equivalent grouping. Never run the quadratic 200k baseline on UI.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/imaging ./internal/dupes ./internal/ui/grid ./internal/ui -count=1`
2. `go test ./internal/dupes -run '^$' -bench '^BenchmarkGrouping$' -benchmem -count=3`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Ticket 18 establishes the generation-correct facts this accepted snapshot consumes. This ticket includes its UI integration and benchmark evidence rather than leaving an unconsumed model API.

## Completion boundary

This ticket closes only its named part of MA-008. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

Resolution (2026-09-07): generation/fact/reset/distance keys now reuse accepted groups; changed grid grouping is cancellable, coalesced worker work with guarded installation. The indexed complete-linkage implementation preserves earliest compatible membership and native-size/index representatives. All 27 negative mutations were rejected. The full 10k/50k/200k unrelated/dense benchmark matrix ran three times; [measurement evidence](../evidence/24-grouping-benchmarks.md) retains inputs, timing, allocations and limitations. Shared `make verify` passed native checks and every Linux race partition, including all 665 UI tests (212/225/228). Log: `/private/tmp/picfetch-maintainability-24-25-verify.log`; UI shard times 319.878s/307.214s/244.663s. The canonical Linux gate passed the platform-sensitive golden test; no golden was regenerated.


2026-09-07 follow-up: the user reported lost progressive updates. Reopened for
[the reproduced partial-pass regression and fix](../evidence/24-progressive-hide.md).
Independent grouping restores incremental hiding; cancellation, stale-source
and worker-delivery guards pass. The shared Phase 6 common gate is running.

Phase 6 completion: [evidence](../evidence/24-progressive-hide.md) and the [full common gate](../evidence/phase6-verify.txt) pass. Lead review and eight negative overlays are complete. Windows native package/WACK validation remains deferred in ticket 27.
