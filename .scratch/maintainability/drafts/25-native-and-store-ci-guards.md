# 25: Execute omitted native and Store regression guards

**What to build:** Release validation actually executes the existing Windows, macOS and Store-specific guards with visible evidence of test selection.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: draft — awaiting breakdown approval
Scope: required
Source: [Maintainability specification](../spec.md), AC17 / MA-017; plan work package R (automated native coverage).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] Native Windows executes wallpaper target/Unicode validation guards as well as the existing updater checks, with desktop mutations stubbed.
- [ ] Native macOS runs main's actual delegate-graft guard; testing only the open-with package does not substitute for the Cocoa-linked binary.
- [ ] Explicit microsoftstore selection executes the Store-managed distribution guard and preserves ordinary-build policy.
- [ ] CI logs enumerate the required tests and reject missing/skipped required guards; cross-vet/build remains useful but is not execution evidence.
- [ ] Preserve Linux reference/golden validation and existing release gates without publishing an artifact.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/wallpaper ./internal/clipboard ./internal/filepicker ./internal/update ./internal/ui/autoupdate -count=1 -v`
2. `go test . ./internal/openwith ./internal/displays ./internal/winpos -count=1 -v`
3. `go test -tags=microsoftstore ./internal/distribution ./internal/ui/autoupdate -count=1 -v`

These commands are acceptance requirements for future implementation; they have not been run as proof that this draft is implemented. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Run the first command on Windows and the second on macOS; the third needs an appropriate native C/GL runner. The real-GL portion of MA-017 remains open until ticket 26 is also complete.

## Completion boundary

This ticket closes only its named part of MA-017. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Draft prepared for breakdown approval; no implementation or publication has occurred.
