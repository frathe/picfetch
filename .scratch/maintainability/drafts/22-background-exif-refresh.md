# 22: Load EXIF panel data without blocking UI

**What to build:** Open or refresh the EXIF panel without blocking input while a slow source is read, and show only metadata for the current displayed request.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: draft — awaiting breakdown approval
Scope: required
Source: [Maintainability specification](../spec.md), AC14 / MA-014; plan work package N (EXIF reads).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] Capture the displayed source and use a cancellable panel-owned read/probe request; gate it while UI interaction continues.
- [ ] Navigate, close, supersede, or complete a source mutation during a held read: obsolete metadata, GPS state and action availability cannot install.
- [ ] Expose completion, cancel owned work on close, and add it to harness drain without conflating it with map-warm completion.
- [ ] Apply each result/error once on UI and avoid redundant successful-removal rereads while preserving the metadata-removal notification contract.
- [ ] Keep metadata removal as an original-file mutation; this ticket does not change metadata omission on exported copies.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/ui/exifwin ./internal/ui ./internal/imaging -count=1`

These commands are acceptance requirements for future implementation; they have not been run as proof that this draft is implemented. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

Ticket 19's tile ownership is separate and does not gate metadata reads. Ticket 23 must preserve this request contract if it lands later, and vice versa.

## Completion boundary

This ticket closes only its named part of MA-014. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Draft prepared for breakdown approval; no implementation or publication has occurred.
