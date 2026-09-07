# 28: Measure foreground contention during favorite prewarming

**What to build:** Determine whether current favorite prewarming materially delays visible image work using a repeatable cold-favorite workload.

**Blocked by:** [15: Cancel capture-date sorting inside source reads](15-capture-sort-cancellation.md); [16: Cancel favorite thumbnail reads through imaging](16-favorite-thumbnail-cancellation.md); [17: Cancel grid thumbnails and native-size backfill](17-grid-read-cancellation.md)

Status: ready-for-agent
Scope: conditional
Source: [Maintainability specification](../spec.md), AC21 / MA-021; plan work package P (measurement).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] After ancillary cancellation is complete, add the specified BenchmarkPreviewForegroundContention harness and compare interaction with and without competing preview work.
- [ ] Record hardware, source set, cold/warm conditions, foreground latency, throughput, allocations and preview convergence; distinguish non-interruptible work from obsolete reads.
- [ ] Exercise the real scheduling boundary through controlled instrumentation without public services or desktop mutations.
- [ ] Record an explicit evidence-backed decision to retain the current policy or activate ticket 29 with the measured acceptance target; no fixed latency claim is invented before measurement.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/favthumbs -run '^$' -bench '^BenchmarkPreviewForegroundContention$' -benchmem -count=3`
2. `go test ./internal/favthumbs ./internal/ui/grid ./internal/ui -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Activation: only when MA-021 measurement is selected. Required work may finish without this ticket; MA-021 stays open while unselected.

## Completion boundary

This ticket closes only its named part of MA-021. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.
