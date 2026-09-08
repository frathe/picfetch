# 15: Cancel capture-date sorting inside source reads

**What to build:** Cancel a capture-date sort during an ancillary source read so obsolete work releases the sort request promptly where the reader supports it.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC10 / MA-010; plan work package J (capture-date sorting).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Manual-test follow-up: cancellation restores the menu/title mode belonging to the retained order, and the cancelled menu action can be retried.

- [x] Carry the existing sort request context through key extraction and CaptureDate into the bounded read/probe path.
- [x] A controlled reader paused between chunks proves cancellation stops further reads and no obsolete ordering installs.
- [x] Cancellation is not silently treated as missing metadata followed by ordinary modification-time fallback; valid missing-metadata fallback remains intact.
- [x] Expose completion of the cancelled sort and describe the limit of an already-blocked underlying read.
- [x] Keep existing callers buildable throughout any API migration; add a context-bearing form alongside a compatibility form if needed.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/imaging ./internal/filesort ./internal/ui -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Completion boundary

This ticket closes only its named part of MA-010. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

## Implementation progress

Context now crosses capture-date extraction and bounded source reads. Controlled reads, distinct cancellation/fallback policy and viewer completion pass, with five negative mutations rejected. Exact tests, commands and observed red/green results are in the [active plan](../../../plans/2026-09-06-maintainability-plan.md). Lead review is complete. The common `make verify` gate shared with tickets 17–18 remains pending; this ticket is not yet resolved.

Common gate passed: `make verify` (641 Linux/amd64 UI tests, remaining race packages, native vet/build, formatting/TUF/Qodana). Evidence: `/private/tmp/picfetch-maintainability-15-18-verify.log`. Lead review and negative guards are recorded in the active plan. No commit was created.

Shared verification passed: `make verify`, 644 Linux/amd64 UI tests across 212/204/228-test shards (325.882s/278.304s/260.193s), all remaining race packages, native vet/build and formatting/TUF/Qodana checks. Log: `/private/tmp/picfetch-maintainability-15-19-20-verify.log`. Lead review and negative guards are recorded in the active plan.

The manual-test follow-up is fixed: cancellation restores the retained order’s mode in the menu/title, and selecting the cancelled mode again starts a fresh sort. Two maintained viewer regressions cover retry and superseded completion; six negative violations are rejected. User confirmed on 2026-09-07: "sorting cancel works." The manual menu/order cancellation check now passes.
