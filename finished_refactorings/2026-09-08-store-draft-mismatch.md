# Diagnose and fix Store draft metadata mismatch

Route: Standard. Owner: lead, inline. Budget: zero spawns, one final full suite
per reviewable change; focused publisher tests while iterating.

## Problem and evidence

Publisher run 34222625099, job 102049260504, failed with `pending submission has
changes outside the recorded update`. Receipt 6326959682 records phase `created`,
submission 1152921505701835817, base submission 1152921505701823150, and the original
v1.0.3 artifact. Partner Center shows the draft's old release notes and unchanged
packages/listings. The error records neither the mismatched fields nor whether
package matching failed, so the exact API discrepancy is not yet known.

## Decisions

- Preserve the current draft, receipt, artifact and approval requirements.
- Keep existing metadata/package validation unchanged until the actual difference
  is captured. Browser-visible fields cannot establish equality of API resources.
- Extend the existing read-only `check` command to report bounded metadata field
  paths and match booleans, never field values, credentials or upload URLs.
- Compare with the receipt's original base submission and frozen notes, even if
  the current published submission has changed.

## Acceptance criteria

- AC1: `check` explains the recorded pending draft's differences without a Store
  write or receipt mutation. `go test ./scripts/storepublish -run
  TestStorePublishPendingDiagnostic -count=1`.
- AC2: field paths are deterministic and bounded; values and ignored upload/status
  fields do not appear. Same command, including redaction and size guards.
- AC3: existing publisher lifecycle and tampering guards remain green.
  `go test -race ./scripts/storepublish -count=1`.
- AC4: capture the live diagnostic report through an approved GitHub `check` run,
  then reproduce the actual discrepancy in the command harness and fix it.
- Final gate: `make verify`, `git diff --check`.

## Tasks

### 1. Add read-only pending-draft diagnostics
Owner: lead inline. Files: `scripts/storepublish/{notes.go,lifecycle.go,main_test.go}`,
`docs/microsoft-store.md`, `todos.md`. Depends: none.
Contract: `check` optionally adds `pending_validation` for a pending submission
matching the saved receipt; its field differences omit values and volatile fields.
Test: recorded draft drift is explained, secrets are absent, no mutations occur;
unrelated drafts and nonpending submissions retain the existing check behavior.
Verify: AC1-AC3, final gate. Budget: zero spawns, one review round, full suite once.

### 2. Fix the evidenced discrepancy and validate recovery
Owner: lead inline. Depends: task 1 plus the approved live check report.
Files: `scripts/storepublish/{notes.go,lifecycle.go,main_test.go}`,
`docs/microsoft-store.md`, `todos.md`.
Contract: `metadataMatches(object, string) (bool, error)` admits the exact recorded
hash or one differing only in the documented read-only boolean
`pricing.isAdvancedPricingModel` (including its omission). Keep `metadataDigest`
and durable receipt hashes unchanged: test the finite boolean representations
against the original hash instead of migrating the receipt or dropping pricing.
Use this match at each existing update/upload/commit guard and in check's match
booleans; diagnostic paths continue to show raw differences.
Test: reproduce the successful PUT followed by the observed metadata error;
new submissions and existing `created`/`updated` receipts recover while retaining
their original hash. Editable pricing, listing, note and package changes fail.
Verify: reproduce the exact response shape, red/green command tests, final gate.
Budget: zero spawns; no speculative normalization or receipt migration.

Completion: the user committed the fix and the approved live recovery succeeded.
Upload and submission acceptance are verified. Final certification and Store
availability remain tracked in `todos.md`; successful submission does not establish
publication. Microsoft credentials remain in the reviewer-protected environment.

## Evidence

- Initial investigation: API receipt and read-only Partner Center inspection above.
- Task 1: the new command test first failed because `check` omitted the report.
  The full publisher race suite passes. Temporary compiler overlays confirmed
  failures when disabling the size bound, field-name redaction, missing/null
  distinction, original-base lookup, receipt/state scoping, or value omission;
  an injected Store update also triggered the read-only guard.
  Existing metadata and package acceptance rules are unchanged.
- Task 1 final gate: `make verify` passed (format/TUF/Qodana checks, host vet and
  build, complete Linux/amd64 Docker race suite); `git diff --check` passes.
  Log: `/tmp/picfetch-store-draft-diagnostics-verify.log`.
- Task 2 live evidence: approved check run 34226413927 succeeded. Its only
  `changes_from_prepared` entry is `/pricing/isAdvancedPricingModel`;
  `packages_match_recorded` and `prepared_base_matches_recorded` are true. Unlike
  the earlier browser view, the API proves the updated notes and package are
  already saved. Microsoft's pricing-resource reference documents this field as
  read-only account capability metadata, distinct from editable prices.
- Task 2 red/green: the command regression failed with the production error
  `pending submission has changes outside the recorded update` on both create
  and update responses. Existing-receipt checks rejected the sole pricing flag
  difference. All publisher race tests pass after the fix, including every
  optional boolean transition and existing `created`/`updated` recovery.
- The pre-fix fixture hash is pinned literally to prove receipt compatibility.
  Temporary compiler overlays caught regressions in flag acceptance, editable
  metadata checks, package validation, original hash format, input ownership and
  recovery without a repeated PUT.
- Task 2 full gate: `make verify` passed (format/TUF/Qodana checks, host vet and
  build, full Linux/amd64 Docker race suite); `git diff --check` passed. Log:
  `/tmp/picfetch-store-pricing-fix-verify.log`.
- Live recovery: fix commit `5533991` is on main. Approved reconcile run
  34228812095 succeeded using the original v1.0.3 artifact and submission
  1152921505701835817. Microsoft returned `CommitStarted` at 12:57:22 UTC on
  2026-09-08. Receipt 6328101033 records phase `observing` and the original metadata
  hash. A refreshed Partner Center overview at 13:02 UTC confirmed Submission
  complete and Pre-processing active (step 2 of 4), with automatic publishing
  after certification. The earlier "In draft" browser page was stale.

## Cost ledger

- Zero delegates; diagnosis, implementation and review stayed inline.
- Focused command tests during iteration; one full repository gate for task 1.
- Task 2 likewise uses focused publisher tests and one full repository gate.
- Two test-fixture corrections: restrict the alternate published-submission route
  to avoid intercepting existing draft reads, and keep the fixture's package at
  the recognized bootstrap version while changing its published submission ID.
- No Store updates, uploads, commits or receipt writes during diagnosis.
