# 18: Admit thumbnail facts only to their source generation

**What to build:** Keep old hashes, failure markers and native dimensions out of a replacement/reset file set, including same-URI replacement.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC09 / MA-009; plan work package K.
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Pause old reads across source replacement/reset and release both success and failure; neither hash nor thumbnail paths change new-generation facts.
- [x] Check the captured generation atomically with every model mutation, before a later UI installation guard can become relevant.
- [x] Cover reorder/adoption separately so valid established facts can survive without adopting facts from replacement sources.
- [x] Exercise native-dimension backfill and failure retry behavior, with deterministic gated workers and observable completion.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/dupes ./internal/ui/grid ./internal/ui -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Cancellation tickets 15-17 are useful precedents, not blockers: correctness of conditional publication is independently testable.

## Completion boundary

This ticket closes only its named part of MA-009. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

Implemented and lead-reviewed: captured FactWriter guards cover replacement/reset, atomic admission, native backfill, retry and adoption. Full dupes/grid race suites pass (1.320s/2.549s); fourteen negative mutations are rejected. Shared 15–18 common gate remains pending.

Common gate passed: `make verify` (641 Linux/amd64 UI tests, remaining race packages, native vet/build, formatting/TUF/Qodana). Evidence: `/private/tmp/picfetch-maintainability-15-18-verify.log`. Lead review and negative guards are recorded in the active plan. No commit was created.
