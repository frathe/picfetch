# 16: Cancel favorite thumbnail reads through imaging

**What to build:** Cancel a favorite preview pass through thumbnail read/probe/decode boundaries while retaining previews the pass never visited.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC10 / MA-010; plan work package J (favorite previews).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Introduce context-bearing thumbnail APIs and thread the favorite-pass context through memory/disk/decode work; retain compatibility for unmigrated grid callers.
- [x] A controlled chunked source read stops on cancellation, and a cancelled slot waiter never starts decoding.
- [x] Check context before and after non-interruptible decoder work and before accepting its result; the cancelled pass exposes completion.
- [x] Do not sweep unvisited previews on cancellation; preserve normal preview cache hits, writes and idle convergence.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/imaging ./internal/favthumbs ./internal/ui -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

This is the expand step with a complete favorite-preview caller migrated. Ticket 17 migrates the grid to the established context-bearing thumbnail contract.

## Completion boundary

This ticket closes only its named part of MA-010. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

## Implementation progress

Context now crosses thumbnail generation, memory/disk preview paths and atomic preview writes. Favorite shutdown cancels and stops admission; all passes remain tracked. Controlled read, decode, cache, encode, idle convergence and lifecycle regressions pass, with twelve negative mutations rejected. Exact tests, commands and observed red/green results are in the [active plan](../../../plans/2026-09-06-maintainability-plan.md). Lead review is complete. The common `make verify` gate shared with tickets 17–18 remains pending; this ticket is not yet resolved.

Common gate passed: `make verify` (641 Linux/amd64 UI tests, remaining race packages, native vet/build, formatting/TUF/Qodana). Evidence: `/private/tmp/picfetch-maintainability-15-18-verify.log`. Lead review and negative guards are recorded in the active plan. No commit was created.
