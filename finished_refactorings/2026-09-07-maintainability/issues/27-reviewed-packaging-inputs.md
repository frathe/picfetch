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

Windows native tests now pass: eight full accepted package runs and all 14 required guards executed without skips. The complete common gate also passes after the XDG parser correction those tests exposed. All seven package artifacts were refreshed from that verified source, and metadata inspection passes. The user handles VM keyboard input because automated input drops keystrokes. Z:\smoke.cmd completed all four refreshed Windows variants with isolated test settings. Ordinary ARM64 renders Alpha/Beta and comparison correctly; Store ARM64 renders Alpha; both report window DPI 96, empty stdout/stderr and exit 0. Both amd64 variants fail before a visible window with WGL OpenGL context-unavailable errors followed by a Fyne nil-window panic. Raw logs correct the earlier visual-order attribution: the empty drop area belonged to the still-running ordinary ARM64 process, not a new package failing to load images. A read-only DLL/driver/monitor probe is prepared. x64 graphical startup, Linux native startup and actual Windows SDK/WACK validation remain open. The user ran Z:\pack.cmd and confirmed its missing-SDK prerequisite failure before any package/trust changes. Ubuntu's local smoke ISO is mounted, but no application launch succeeded; the VM was shut down.

The x64 runtime retry overcame the WebDAV size limit through verified ZIP transfer. Both variants then loaded the matching app-local DLL and created a DPI-96 window, but crashed on the first Fyne rectangle draw with illegal instruction 0xc000001d (exit 2). These remain failed rendering attempts. The next isolated probe restricts Mesa CPU capabilities to SSE2 via child environment only, using the unchanged executables/runtime. Raw logs and hypotheses are retained in the packaging evidence.

User deferred further Windows work and will test later on native x64 systems. The [Windows test checklist](../windows-test-todo.md) records exact artifacts, native smoke steps, WACK prerequisites and existing evidence. The SSE2 VM experiment is prepared but not executed. Continue macOS/Linux checks; do not resume Windows testing automatically.

Non-Windows follow-up complete: refreshed macOS build 450 rendered and quit 0 with empty log and original state restored. Both refreshed Linux executables rendered Alpha/Beta and quit 0 with empty logs in isolated Debian 12 / Mesa llvmpipe desktops; ARM64 additionally passed comparison/swipe and quit while comparison was open. ARM64 execution is native to the Linux VM; amd64 uses Docker CPU emulation. [Linux evidence and limitations](../evidence/27-linux-packaged-smoke.md). Ticket remains open only for the deferred native x64 Windows startup and Windows SDK/WACK acceptance. Production source is unchanged; the existing common gate remains valid.

Phase 6 refresh: all four ordinary/Store Windows executables were rebuilt from
the final application source, inspected and copied with fixtures/SHA256SUMS to
`bin/maintainability-validation/`. [Build provenance](../evidence/phase6-windows-build.txt)
and [hashes/build metadata](../evidence/phase6-windows-artifacts.txt) are retained.
The updated Windows checklist points to these artifacts. No Windows execution
or SDK/trust changes were performed; native x64 and WACK acceptance stay open.
