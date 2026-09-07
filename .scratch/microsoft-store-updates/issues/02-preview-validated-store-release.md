# 02: Preview a validated release with generated Store notes

**What to build:** Given a validated build and a captured published Store snapshot, the maintainer can run one preview command and inspect the exact candidate, generated notes and intended metadata changes. The same input produces the same result and makes no Store mutation.

**Blocked by:** None (can start immediately).

**Status:** ready-for-agent

**Parent:** [Automatic Microsoft Store updates](../spec.md).

**Coverage:** User stories US2, US3, US4, US5, US6, US7, US8, US9, US10, US14, US18, US20; AC1, AC2, AC7 (preview), AC9 (focused regression).

## Acceptance criteria

- [ ] Admit trusted canonical stable tags only when source commit, application version, contained x64/ARM64 package versions, notes and successful CI/WACK evidence agree. Bind the original bundle's digest and producing run. Reject branch/fork/prerelease, moved-tag, older-version, stale-note, missing/expired artifact and mismatched-evidence fixtures before any mutation.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishAdmission$'`.

- [ ] Generate plain-text notes from captured release notes, including intervening releases since the last published Store version. Omit Internal and explicitly Linux/macOS-only entries; retain shared, mixed and unmarked entries. Preserve the spec's ordering and complete-entry fitting rules, version/range link, 1,500-character and UTF-16 bounds, and neutral empty/oversized fallback. Include Unicode, stale/missing source and skipped-version fixtures.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishNotes$'`.

- [ ] Preview the selected release and all intended field changes against a supplied published snapshot. Apply the spec's English-note default to the snapshot's existing locales, including German and note overrides, while preserving unrelated localized metadata and the locale set. Freeze the candidate, note and artifact identities for later submission; no creation, upload or commit request is possible in preview.
  Verification: `go test ./scripts/storepublish -run '^TestStorePublishReadOnly`.`.

- [ ] The command works end to end with temporary release/artifact fixtures and fake service data, with inputs supplied per invocation. Existing release-note generation and packaging behavior remain covered. Add the new tooling package to the architecture map and each new test file to Qodana exclusions.
  Verification: `go test ./scripts/releasenotes ./scripts/msixstage ./scripts/storepublish` and `make check-qodana-test-exclusions`.

## Implementation notes

Do not wait for live credentials. The captured snapshot is a documented preview input; live retrieval is connected in ticket 03. Prefer the existing command and workflow-test patterns named in the spec. Record the command/input/output contract in the implementation plan so submission can consume this result without rebuilding or reinterpreting the release. This ticket introduces no automatic publishing trigger.

The spec's proposed defaults remain defaults, not newly confirmed user decisions.
Use the command boundary and fake-service testing strategy from the spec.
Verification commands referring to publisher tests or commands describe future
implementation requirements; they have not been executed as part of ticket creation.

## Comments

2026-09-07: Created from the implementation spec. No implementation or live
verification has been performed for this ticket.
