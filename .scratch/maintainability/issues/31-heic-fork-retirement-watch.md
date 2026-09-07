# 31: Reassess the HEIC fork at its next dependency update

**What to build:** At a future authorized HEIC update, determine whether an official release can replace the fork while preserving its leak mitigation.

**Blocked by:** [11: Handle declining RSS in the optional HEIC check](11-safe-rss-growth.md)

Status: ready-for-agent
Scope: watch
Source: [Maintainability specification](../spec.md), AC23 / MA-023; plan work package Accepted watch.
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] Activation requires a separate HEIC dependency-update task; creating this reminder does not start an upgrade or recurring monitor.
- [ ] Verify and link evidence that the chosen official tag actually includes the leak fix before replacing the fork.
- [ ] If inclusion cannot be established, retain the fork and record the watch outcome without claiming retirement.
- [ ] Any replacement passes imaging regressions and the optional Linux RSS check after ticket 11's arithmetic fix; record the actual platform and decoder path.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/imaging -count=1`
2. `PICFETCH_HEIC_LEAK_TEST=1 go test -tags=heicleak ./internal/imaging -run '^TestHEICDecode_DoesNotGrowRSSUnbounded$' -count=1 -v`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Activation: next separately authorized HEIC update. This accepted watch is excluded from the ordinary work frontier even when ticket 11 is complete.

## Completion boundary

This ticket closes only its named part of MA-023. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.
