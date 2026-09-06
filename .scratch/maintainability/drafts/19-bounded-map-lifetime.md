# 19: Bound aggregate map requests and failure state

**What to build:** Map navigation and window closure keep aggregate fetch concurrency and failure metadata bounded while retaining useful deduplication and cache behavior.

**Blocked by:** None (can start after publication and any activation condition is satisfied).

Status: draft — awaiting breakdown approval
Scope: required
Source: [Maintainability specification](../spec.md), AC15 / MA-015; plan work package L.
Owner: lead for contract decisions, review and final gate; implementation routing follows the current project working agreement.

## Acceptance criteria

- [ ] Concurrent warm passes and foreground requests share one declared per-fetcher limit rather than per-pass independent caps.
- [ ] Bound queued work as well as active requests so per-URL goroutines or cancelled waiters cannot accumulate without limit; preserve useful request deduplication under admission pressure.
- [ ] Superseded warm work and window closure cancel obsolete queued/in-flight work with observable worker completion.
- [ ] Failure metadata has a tested capacity or expiry bound; retries and deduplication remain correct.
- [ ] Use a controlled local tile server to verify bounds, cancellation, retries and callback behavior; never load public tile services.
- [ ] Wait for onChange's own observable effect rather than a pending counter cleared before that callback.
- [ ] Run the verification commands below, retain named non-skipped regression/native/benchmark evidence for this slice, and satisfy the spec's AC25 common gate with `make verify` for a mergeable implementation change.

## Verification

1. `go test ./internal/ui/exifwin -count=1`

These commands are acceptance requirements for future implementation; they have not been run as proof that this draft is implemented. The named fuzz/benchmark/RSS targets specified by the parent may still need to be added. Before using a focused filter, check the same build-selected inventory with `go test`'s `-list` option; an absent or skipped required test is not a pass.

Record the expected failing regression for a reproduced defect, then the successful result after the fix. Use observable queues/channels and instance-owned seams where needed, with OS mutations stubbed; maintain exact test manifests and worker cleanup. The parent spec's Testing Decisions and AC25 supply the common completion contract.

## Scope and sequencing

The queue patterns from tickets 12-14 are nonblocking precedents. Keep this lifecycle separate from metadata-refresh/removal work.

## Completion boundary

This ticket closes only its named part of MA-015. If that MA item is split across other tickets, retain it as open until all required parts and evidence are complete. Preserve the parent specification, historical audit evidence and unrelated working-tree changes. Populate concrete files/contracts/tests and routing budget in the active implementation plan before coding; no git commit is authorized by this ticket.

## Comments

Draft prepared for breakdown approval; no implementation or publication has occurred.
