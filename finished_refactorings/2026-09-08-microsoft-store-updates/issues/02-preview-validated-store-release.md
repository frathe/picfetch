# 02: Preview a validated release with generated Store notes

**What to build:** Given a validated build and a captured published Store snapshot, the maintainer can run one preview command and inspect the exact candidate, generated notes and intended metadata changes. The same input produces the same result and makes no Store mutation.

**Blocked by:** None (can start immediately).

**Status:** implemented; local verification passed

**Parent:** [Approved Microsoft Store updates](../spec.md).

**Coverage:** User stories US2, US3, US4, US5, US6, US7, US8, US9, US10, US14, US18, US20; AC1, AC2, AC7 (preview), AC9 (focused regression).

## Acceptance criteria

- [x] Admit trusted canonical stable tags only when source commit, application version, contained x64/ARM64 package versions, notes and successful CI/WACK evidence agree. Bind the original bundle's digest and producing run. Reject branch/fork/prerelease, moved-tag, older-version, stale-note, missing/expired artifact and mismatched-evidence fixtures before any mutation.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishAdmission$'`.

- [x] Generate plain-text notes from captured release notes, including intervening releases since the last published Store version. Omit Internal and explicitly Linux/macOS-only entries; retain shared, mixed and unmarked entries. Preserve the spec's ordering and complete-entry fitting rules, version/range link, 1,500-character and UTF-16 bounds, and neutral empty/oversized fallback. Include Unicode, stale/missing source and skipped-version fixtures.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishNotes$'`.

- [x] Preview the selected release and all intended field changes against a supplied published snapshot. Apply the spec's English-note default to the snapshot's existing locales, including German and note overrides, while preserving unrelated localized metadata and the locale set. Freeze the candidate, note and artifact identities for later submission; no creation, upload or commit request is possible in preview.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishReadOnly$'`.

- [x] The command works end to end with temporary release/artifact fixtures and fake service data, with inputs supplied per invocation. Existing release-note generation and packaging behavior remain covered. Add the new tooling package to the architecture map and each new test file to Qodana exclusions.
  Verification: `go test ./scripts/releasenotes ./scripts/msixstage ./scripts/storepublish` and `make check-qodana-test-exclusions`.

## Implementation notes

Do not wait for live credentials. The captured snapshot is a documented preview input; live retrieval is connected in ticket 03. Prefer the existing command and workflow-test patterns named in the spec. Record the command/input/output contract in the implementation plan so submission can consume this result without rebuilding or reinterpreting the release. This ticket introduces no automatic publishing trigger.

The final user amendment requires frathe approval of each specific validated release.
Use the command boundary and fake-service testing strategy from the spec.
Verification commands define the existing command/fake-service acceptance seam.

## Comments

2026-09-07: Created from the implementation spec. No implementation or live
verification has been performed for this ticket.

2026-09-08: Implemented through the `scripts/storepublish` command and fake
GitHub/Microsoft/blob services. `go test ./scripts/releasenotes ./scripts/msixstage ./scripts/storepublish` and `go test -race ./scripts/storepublish` passed.
Admission, metadata, artifact-digest and serialization guards were deliberately
broken, observed failing and restored; green focused reruns passed. No live Store
submission was made. Account setup and live publication belong to 01/06.

2026-09-08 approval amendment (supersedes earlier unattended/setup comments):
The final user requirement is approval of every specific release via GitHub Required
reviewers. GitHub metadata now verifies main-only, frathe reviewer, self-review
allowed, no bypass/timer/custom rules, and all three secret names. The user confirmed
the linked Developer application and replacement key. Credentials remain exclusively
in GitHub Secrets. The publisher is not yet on main, so the approved CI read check,
actual submission permission and first ordinary approved publication remain open.
Preparation freezes artifact/notes/base/receipt before the protected job; submit
uses only that selection and reconcile never admits a newer release. Cron is removed;
status/recovery require manual approved dispatch. Local approval and workflow tests
cover the amendment; see the plan for verification. No Store update was submitted.
