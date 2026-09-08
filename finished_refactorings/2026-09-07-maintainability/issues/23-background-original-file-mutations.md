# 23: Run original-file mutations in serialized background work

**What to build:** Save Changes and metadata removal leave the UI responsive and cannot overlap unsafely on the same original source.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: resolved
Scope: required
Source: [Maintainability specification](../spec.md), AC14 / MA-014; plan work package N (Save Changes and metadata removal).
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [x] Gate Save Changes and metadata removal independently while input/redraw continue; capture source identity and intended pixels before background work.
- [x] Serialize conflicting operations across their complete read-transform-write transaction using the resolved source identity, including symlink aliases; atomic rename alone is not transaction serialization.
- [x] Ensure existing write routes participate when they can target that same resolved file, including an Export destination aliasing the source; preserve independent-file concurrency and avoid a universal task registry.
- [x] Preserve permissions, atomic replacement, correct existing image-record/dimension policy, and error behavior; a failed save retains its rotation state.
- [x] Cover cancellation before write and completion after replacement committed; navigation/close cannot retarget writes or install old UI state and must not pretend to undo a completed disk mutation.
- [x] Refresh/invalidate the appropriate caches and current metadata/info once after success; expose all workers and causal UI completion to shutdown/harness drain.
- [x] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/ui ./internal/ui/exifwin ./internal/imaging -count=1`

These commands are acceptance requirements; an unchecked ticket has not yet supplied implementation evidence. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

These two mutation actions stay together because making either asynchronous creates a real shared-source race. Tickets 04/05/12-17/22 are useful precedents and their established contracts must be preserved; they do not supply a required new interface that blocks starting this slice.

## Completion boundary

This ticket closes only its named part of MA-014. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Published after the user requested `/implement sdd tdd` on the parent specification. Conditional and watch activation rules remain in force.

## Implementation evidence

Held Save encoding and independently held metadata removal reproduced blocked input; both now run in owned cancellable workers with queued results. Complete resolved-path transactions serialize Save/Strip/Export across symlink aliases while unrelated destinations proceed. Context cancellation leaves original bytes intact before rename; WriteResult retains committed replacement identity after navigation/close. Export and mosaic participate, with separate admission and stale presentation guards. Save preserves failed/later rotations and installs its own frame slice, avoiding mutation of shared decoded records.

Committed writes invalidate cache revisions, thumbnails and duplicate facts. Captured cache writers reject old producers; current aliases refresh via tracked background path/info reconciliation. An unrelated intervening commit cannot discard a required current-source refresh. Comparison reload retains transforms/layout/link/order. Favorite previews carry source versions captured before decode, preventing both held old reads and memory hits before UI invalidation from being persisted under a replacement's newer name. EXIF removal reports one notification/refresh and keeps unrelated/current rotated presentation intact.

Named regressions are in imaging/mutations_test.go, UI savework_test.go/exportwork_test.go/exif_test.go/favthumbs_test.go, grid/thumbs_test.go, EXIF stripwork_test.go, comparison compare_test.go, and mosaic export_test.go. Forty-three distinct deliberate violations were rejected by temporary overlays; `/private/tmp/picfetch-maintainability-23-negative.log`. Focused red/green commands and timings are recorded in the active plan. Source versions retain the existing path/mtime/size policy, and cache invalidation is conservative after explicit writes.

`make verify` passed formatting/TUF/Qodana, native vet/build, exact Linux shard inventory and all Linux race partitions: **665 UI tests (212/225/228)**, shard times **307.645s/297.999s/246.786s**. Log: `/private/tmp/picfetch-maintainability-23-verify.log`. Ticket 23 completes MA-014 together with tickets 21/22. No commit was created.
