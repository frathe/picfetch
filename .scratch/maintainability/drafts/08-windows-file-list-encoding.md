# 08: Decode Windows copied-file lists explicitly as UTF-8

**What to build:** Copy Windows file references containing accented, CJK and emoji names without changing their paths at the PowerShell boundary.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: draft — awaiting breakdown approval
Scope: required
Source: [Maintainability specification](../spec.md), AC12 / MA-012; plan work package F (clipboard encoding).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] The BOM-less UTF-8 list is explicitly decoded at the Windows PowerShell consumer.
- [ ] Execute the generated decoding boundary on Windows with accented, CJK and emoji temporary paths and assert exact decoded names; stub clipboard mutation.
- [ ] Exercise an ANSI-default configuration or otherwise establish explicit decoding independently of the host's convenient UTF-8 settings.
- [ ] Retain list cleanup, ordering and failure behavior; actual native test execution is required, not only script-string inspection.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/clipboard -count=1 -v`

These commands are acceptance requirements for future implementation; they have not been run as proof that this draft is implemented. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Requires native Windows PowerShell. Cross-compilation and skipped tests do not close this ticket.

## Completion boundary

This ticket closes only its named part of MA-012. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Draft prepared for breakdown approval; no implementation or publication has occurred.
