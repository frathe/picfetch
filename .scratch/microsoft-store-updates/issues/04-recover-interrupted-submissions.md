# 04: Recover interrupted submissions without creating duplicates

**What to build:** A fresh run can safely continue a partially completed Store update after an upload, API or runner failure. It discovers completed remote effects before retrying and reports conflicts instead of overwriting someone else's work.

**Blocked by:** [03: Submit one validated update and report its Store state](03-submit-validated-update.md)

**Status:** ready-for-agent

**Parent:** [Automatic Microsoft Store updates](../spec.md).

**Coverage:** User stories US15, US16, US17, US18, US19, US21; AC4, AC6 (retry and authentication failures).

## Acceptance criteria

- [ ] Restart from durable evidence after failures before/after create, upload and commit, including a lost response and failure after a remote effect but before local recording. Resume identified work, observe processing submissions and treat the exact already-published release as a no-op. Tests must prove no duplicate submission across fresh invocations.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishRecovery$'`.

- [ ] Reject a changed artifact or notes snapshot for the same release, moved tags, obsolete versions and stale build evidence on retry. Preserve unknown/manual drafts. If an ambiguous create/commit cannot be associated with recorded intent through safe reads, report a conflict and make no guessed mutation.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishRecovery$'`.

- [ ] Handle transient API/upload failures and throttling with bounded backoff and Retry-After behavior. Refresh expiring access tokens as needed. Treat permanent permission failures, certification rejection and unknown states distinctly; never resubmit rejected identical bits indefinitely. Redact sensitive diagnostics on every failure path.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishStatus`.`.

- [ ] Concurrent or stale command invocations cannot mutate this product simultaneously, including after a process restart. A new release does not cancel an in-flight mutation; a stale owner must not apply changes after losing its claim. Verify through contending command invocations and external service effects.
  Verification: `go test -race ./scripts/storepublish -run '^TestStorePublishRecovery$'`.

## Implementation notes

Use the same command seam, durable receipt and per-invocation adapters as ticket 03. Tests control time and interruption explicitly; do not use sleeps to guess completion. Record recovery behavior in the Store operations documentation. Automatic deletion of conflicting drafts and rollback to lower package versions remain out of scope.

The spec's proposed defaults remain defaults, not newly confirmed user decisions.
Use the command boundary and fake-service testing strategy from the spec.
Verification commands referring to publisher tests or commands describe future
implementation requirements; they have not been executed as part of ticket creation.

## Comments

2026-09-07: Created from the implementation spec. No implementation or live
verification has been performed for this ticket.
