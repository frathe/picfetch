# 29: Protect foreground capacity if prewarm measurements justify it

**What to build:** Apply the smallest measured prewarm share/yield policy that preserves interactive capacity and still completes disk previews when idle.

**Blocked by:** [28: Measure foreground contention during favorite prewarming](28-measure-preview-contention.md)

Status: draft — awaiting breakdown approval
Scope: conditional
Source: [Maintainability specification](../spec.md), AC21 / MA-021; plan work package P (conditional policy).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] Start only when ticket 28 demonstrates contention and records a concrete foreground-latency target and chosen bounded policy.
- [ ] Foreground work retains capacity under a cold favorite; compare actual before/after latency as well as throughput with the same workload.
- [ ] Disk-preview generation converges when idle, cancellation does not sweep unvisited valid previews, and full-memory-cache behavior stays bounded.
- [ ] Preserve separate foreground/background ownership; avoid a shared semaphore that lets previews starve visible thumbnails.
- [ ] If measurement supports retaining the policy, record this ticket as intentionally unactivated rather than pretending a scheduler change was delivered.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/favthumbs -run '^$' -bench '^BenchmarkPreviewForegroundContention$' -benchmem -count=3`
2. `go test ./internal/favthumbs ./internal/ui/grid ./internal/ui -count=1`

These commands are acceptance requirements for future implementation; they have not been run as proof that this draft is implemented. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Activation: measured need plus an accepted bounded policy from ticket 28. It is not on the implementation frontier merely because its numbered blocker finished.

## Completion boundary

This ticket closes only its named part of MA-021. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Draft prepared for breakdown approval; no implementation or publication has occurred.
