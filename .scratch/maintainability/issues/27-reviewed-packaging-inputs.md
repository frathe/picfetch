# 27: Pin and validate reviewed packaging inputs

**What to build:** Local, release and Store builds use reviewed packaging tool/image inputs and produce inspectable, smoke-tested artifacts without publication.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: claimed
Scope: required
Source: [Maintainability specification](../spec.md), AC20 / MA-020; plan work package S.
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Centralize reviewed tool versions and image references across packaging routes, replacing floating tool selection and recording versioned/pinned image resolution.
- [x] Build logs identify actual tool versions and resolved images; deliberate upgrades have maintained packaging contract coverage.
- [ ] Build the changed macOS, Windows, Store and Linux routes in a disposable checkout; inspect architecture/distribution metadata and native startup.
- [ ] The Store bundle path executes existing package-validation/WACK steps on Windows without submission; retain actual artifact and smoke commands/results.
- [x] Preserve application identity, distribution policy and existing release controls; do not promise byte-identical timestamped or signed artifacts.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./scripts/msixstage ./scripts/plistdoctypes -count=1`
2. `make install-tools`
3. `make package-mac`
4. `make package-windows package-windows-store package-linux`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Use macOS for native packaging, Docker for cross-builds, and each artifact's target OS for startup evidence. Make dry runs and cross-compilation alone are insufficient.

## Completion boundary

This ticket closes only its named part of MA-020. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

Resumed 2026-09-07. Reviewed packaging inputs and all installation/workflow routes are implemented, with executed cross-route failure/provenance guards and 17 rejected negative mutations. Actual macOS plus Windows/Store/Linux amd64+arm64 builds passed in the disposable checkout; architecture/distribution/identity inspection and actual packaged macOS fixture rendering/clean quit passed. See [retained evidence](../evidence/27-reviewed-packaging-smoke.md). The first common gate found the Make help filename-prefix regression; its existing guard reproduced it and all affected script suites pass after the fix. The complete common gate passed with GOGC=25 in the unchanged Docker race suite after a default-GC run hit the 8 GB VM OOM limit; all 665 UI tests and remaining packages passed.

Windows native tests now pass: eight full accepted package runs and all 14 required guards executed without skips. The complete common gate also passes after the XDG parser correction those tests exposed. All seven package artifacts were refreshed from that verified source, and metadata inspection passes. The user handles VM keyboard input because automated input drops keystrokes. Z:\smoke.cmd is prepared to inspect the four refreshed Windows package variants from the Desktop beside the existing libraries, with isolated test settings. Windows/Linux native graphical startup and actual Windows SDK/WACK validation remain open. The user ran Z:\pack.cmd and confirmed its missing-SDK prerequisite failure before any package/trust changes. Ubuntu's local smoke ISO is mounted, but no application launch succeeded; the VM was shut down.
