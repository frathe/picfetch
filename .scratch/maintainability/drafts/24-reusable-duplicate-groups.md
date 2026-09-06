# 24: Reuse cancellable duplicate-group snapshots

**What to build:** Searching a warm Grid result reuses unchanged duplicate groups, while changed grouping computes away from UI and cannot install obsolete results.

**Blocked by:** [18: Admit thumbnail facts only to their source generation](18-generation-safe-facts.md)

Status: draft — awaiting breakdown approval
Scope: required
Source: [Maintainability specification](../spec.md), AC08 / MA-008; plan work package O.
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] Key accepted groups by file generation, hash/native-fact revision and distance; plain search edits filter the accepted snapshot without recomputing grouping.
- [ ] Run changed computations off UI with cancellation/coalescing and current-snapshot installation; cover source replacement, fact changes and distance changes.
- [ ] Compare membership and representatives to the current greedy complete-linkage algorithm on bounded randomized and adversarial cases, including non-transitive chains and pixel-count/index ties.
- [ ] Add the specified BenchmarkGrouping cases for unrelated and dense 10k/50k/200k inputs; record hardware, race status, inputs, latency and allocations.
- [ ] First eliminate recomputation; choose an index only if measurements require it and retain equivalent grouping. Never run the quadratic 200k baseline on UI.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/imaging ./internal/dupes ./internal/ui/grid ./internal/ui -count=1`
2. `go test ./internal/dupes -run '^$' -bench '^BenchmarkGrouping$' -benchmem -count=3`

These commands are acceptance requirements for future implementation; they have not been run as proof that this draft is implemented. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Ticket 18 establishes the generation-correct facts this accepted snapshot consumes. This ticket includes its UI integration and benchmark evidence rather than leaving an unconsumed model API.

## Completion boundary

This ticket closes only its named part of MA-008. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Draft prepared for breakdown approval; no implementation or publication has occurred.
