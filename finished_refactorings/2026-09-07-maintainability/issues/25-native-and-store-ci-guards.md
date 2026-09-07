# 25: Execute omitted native and Store regression guards

**What to build:** Release validation actually executes the existing Windows, macOS and Store-specific guards with visible evidence of test selection.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC17 / MA-017; plan work package R (automated native coverage).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Native Windows executes wallpaper target/Unicode validation guards as well as the existing updater checks, with desktop mutations stubbed.
- [x] Native macOS runs main's actual delegate-graft guard; testing only the open-with package does not substitute for the Cocoa-linked binary.
- [x] Explicit microsoftstore selection executes the Store-managed distribution guard and preserves ordinary-build policy.
- [x] CI logs enumerate the required tests and reject missing/skipped required guards; cross-vet/build remains useful but is not execution evidence.
- [x] Preserve Linux reference/golden validation and existing release gates without publishing an artifact.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/wallpaper ./internal/clipboard ./internal/filepicker ./internal/update ./internal/ui/autoupdate -count=1 -v`
2. `go test . ./internal/openwith ./internal/displays ./internal/winpos -count=1 -v`
3. `go test -tags=microsoftstore ./internal/distribution ./internal/ui/autoupdate -count=1 -v`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Run the first command on Windows and the second on macOS; the third needs an appropriate native C/GL runner. The real-GL portion of MA-017 remains open until ticket 26 is also complete.

## Completion boundary

This ticket closes only its named part of MA-017. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

Progress (2026-09-07): `scripts/nativeguards` enforces native suite selection, inventories required tests and rejects missing/skipped/failed guards while retaining raw JSON events. The CI changes use this runner and retain existing Linux/release gates. Twelve negative mutations were rejected; shared `make verify` passed all partitions and 665 UI tests. Native macOS (including Cocoa-linked main) and explicit Store suites both executed successfully without skips; [retained evidence and Windows commands](../evidence/25-native-guards.md). Actual Windows execution remains required, so this ticket and MA-017 remain open.

Resolved 2026-09-07: actual Windows 11 ARM64 execution now supplies the missing native evidence. Eight complete accepted package runs passed, 254 top-level executions and no skipped tests; the production validator accepted all 14 required Windows/Store guards. The wallpaper XDG parser correction was red on Windows before the fix, then green on Windows and macOS. The updater binary passed unchanged after moving its working directory from WebDAV to a local test folder. The follow-up `make verify` gate passed all 665 UI tests and remaining race packages; Windows/amd64 cross-vet also passed. See [native results](../evidence/25-native-guards.md#offline-windows-execution) and [common gate](../evidence/25-windows-followup-verify.txt). MA-017 remains open for ticket 26's outstanding native renderer checks.
