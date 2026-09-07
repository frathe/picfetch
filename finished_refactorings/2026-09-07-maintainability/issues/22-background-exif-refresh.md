# 22: Load EXIF panel data without blocking UI

**What to build:** Open or refresh the EXIF panel without blocking input while a slow source is read, and show only metadata for the current displayed request.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC14 / MA-014; plan work package N (EXIF reads).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Capture the displayed source and use a cancellable panel-owned read/probe request; gate it while UI interaction continues.
- [x] Navigate, close, supersede, or complete a source mutation during a held read: obsolete metadata, GPS state and action availability cannot install.
- [x] Expose completion, cancel owned work on close, and add it to harness drain without conflating it with map-warm completion.
- [x] Apply each result/error once on UI and avoid redundant successful-removal rereads while preserving the metadata-removal notification contract.
- [x] Keep metadata removal as an original-file mutation; this ticket does not change metadata omission on exported copies.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/ui/exifwin ./internal/ui ./internal/imaging -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Ticket 19's tile ownership is separate and does not gate metadata reads. Ticket 23 must preserve this request contract if it lands later, and vice versa.

## Completion boundary

This ticket closes only its named part of MA-014. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

## Implementation evidence

The held-source Right-key tracer reproduced UI blocking (0.943s). Post-removal reads reproduced a duplicate successful refresh (1.064s). Viewer navigation/reset reproduced obsolete GPS/text/actions (2.763s), and loading admission incorrectly exposed the next selected source before it loaded (2.875s). Panel-owned reads and Invalidate/Refresh now pass the full EXIF race suite (6.223s) and affected viewer selection (15.728s). Named Metadata tests and TestExifNavigationCancelsMetadataBeforeTheNextImageLoads cover source cancellation, queued results, same-URI replacement, close/reopen/no-source/Stop, all-read completion and single notification/refresh. Sixteen deliberate violations were rejected; `/private/tmp/picfetch-maintainability-22-negative.log`. NewTestWindow and the viewer harness drain the EXIF queue and all metadata/map workers.

Shared `make verify` passed formatting/TUF/Qodana, native vet/build and all Linux race partitions: 653 UI tests (212/213/228), shard times 332.664s/287.301s/255.054s. Log: `/private/tmp/picfetch-maintainability-21-22-verify.log`. No commit was created. MA-014 remains open for ticket 23.
